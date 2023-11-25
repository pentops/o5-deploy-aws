package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/japi"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pressly/goose"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.daemonl.com/envconf"
)

var Version string

func main() {
	ctx := context.Background()
	ctx = log.WithFields(ctx, map[string]interface{}{
		"application": "o5-deploy-aws",
		"version":     Version,
	})

	args := os.Args[1:]
	if len(args) == 0 {
		args = append(args, "serve")
	}

	switch args[0] {
	case "serve":
		if err := runServe(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to serve")
			os.Exit(1)
		}

	case "registry":
		if err := runRegistry(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to serve")
			os.Exit(1)
		}

	case "migrate":
		if err := runMigrate(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to migrate")
			os.Exit(1)
		}

	default:
		log.WithField(ctx, "command", args[0]).Error("Unknown command")
		os.Exit(1)
	}
}

func runMigrate(ctx context.Context) error {
	var config = struct {
		MigrationsDir string `env:"MIGRATIONS_DIR" default:"./ext/db"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	return goose.Up(db, "/migrations")
}

func runRegistry(ctx context.Context) error {
	type envConfig struct {
		RegistryPort   int    `env:"REGISTRY_PORT" default:""`
		RegistryBucket string `env:"REGISTRY_BUCKET" default:""`
	}
	cfg := envConfig{}
	if err := envconf.Parse(&cfg); err != nil {
		return err
	}

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)
	handler, err := japi.NewRegistry(ctx, s3Client, cfg.RegistryBucket)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.RegistryPort),
		Handler: handler,
	}
	log.WithField(ctx, "port", cfg.RegistryPort).Info("Begin Registry Server")

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(ctx) // nolint:errcheck
	}()

	return httpServer.ListenAndServe()

}

func runServe(ctx context.Context) error {
	type envConfig struct {
		ConfigFile string `env:"CONFIG_FILE"`
		WorkerPort int    `env:"WORKER_PORT" default:"8081"`

		RegistryPort   int    `env:"REGISTRY_PORT" default:""`
		RegistryBucket string `env:"REGISTRY_BUCKET" default:""`

		DeployerAssumeRole string `env:"DEPLOYER_ASSUME_ROLE"`
		CFTemplates        string `env:"CF_TEMPLATES"`
		CallbackARN        string `env:"CALLBACK_ARN"`
	}
	cfg := envConfig{}
	if err := envconf.Parse(&cfg); err != nil {
		return err
	}

	log.WithField(ctx, "PORT", cfg.WorkerPort).Info("Boot")

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	configFile := &github_pb.DeployerConfig{}
	if err := protoread.PullAndParse(ctx, s3Client, cfg.ConfigFile, configFile); err != nil {
		return err
	}

	environments := []*environment_pb.Environment{}

	for _, envConfigFile := range configFile.TargetEnvironments {
		env := &environment_pb.Environment{}
		if err := protoread.PullAndParse(ctx, s3Client, envConfigFile, env); err != nil {
			return err
		}

		environments = append(environments, env)
	}

	db, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	pgStore, err := deployer.NewPostgresStateStore(db, environments)
	if err != nil {
		return err
	}

	clientSet := &awsinfra.ClientSet{
		AssumeRoleARN: cfg.DeployerAssumeRole,
		AWSConfig:     awsConfig,
	}

	awsInfraRunner := awsinfra.NewRunner(clientSet, pgStore)
	awsInfraRunner.CallbackARNs = []string{cfg.CallbackARN}

	deploymentWorker, err := deployer.NewDeployerWorker(pgStore)
	if err != nil {
		return err
	}

	deploymentManager, err := deployer.NewDeployer(pgStore, cfg.CFTemplates, s3Client)
	if err != nil {
		return err
	}

	log.Debug(ctx, "Got DB")

	githubClient, err := github.NewEnvClient(ctx)
	if err != nil {
		return err
	}

	refLookup := RefLookup(configFile.Refs)

	githubWorker, err := github.NewWebhookWorker(
		githubClient,
		deploymentManager,
		refLookup,
	)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		grpcServer := grpc.NewServer()
		github_pb.RegisterWebhookTopicServer(grpcServer, githubWorker)
		deployer_tpb.RegisterAWSCommandTopicServer(grpcServer, awsInfraRunner)
		deployer_tpb.RegisterDeployerTopicServer(grpcServer, deploymentWorker)
		messaging_tpb.RegisterRawMessageTopicServer(grpcServer, awsInfraRunner)

		reflection.Register(grpcServer)

		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.WorkerPort))
		if err != nil {
			return err
		}
		log.WithField(ctx, "port", cfg.WorkerPort).Info("Begin Worker Server")
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop() // nolint:errcheck
		}()

		return grpcServer.Serve(lis)
	})

	if cfg.RegistryPort != 0 {
		eg.Go(func() error {
			handler, err := japi.NewRegistry(ctx, s3Client, cfg.RegistryBucket)
			if err != nil {
				return err
			}

			httpServer := &http.Server{
				Addr:    fmt.Sprintf(":%d", cfg.RegistryPort),
				Handler: handler,
			}
			log.WithField(ctx, "port", cfg.RegistryPort).Info("Begin Registry Server")

			go func() {
				<-ctx.Done()
				httpServer.Shutdown(ctx) // nolint:errcheck
			}()

			return httpServer.ListenAndServe()
		})
	}

	return eg.Wait()
}

type RefLookup []*github_pb.RefLink

func (rl RefLookup) PushTargets(push *github_pb.PushMessage) []string {
	environments := []string{}
	for _, r := range rl {
		if r.Owner != push.Owner {
			continue
		}

		if r.Repo != push.Repo {
			continue
		}

		if r.RefMatch != push.Ref {
			continue
		}

		environments = append(environments, r.Targets...)
	}
	return environments
}
