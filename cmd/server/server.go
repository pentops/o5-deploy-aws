package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pressly/goose"
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

func runServe(ctx context.Context) error {
	type envConfig struct {
		ConfigFile string `env:"CONFIG_FILE"`
		WorkerPort int    `env:"WORKER_PORT" default:"8081"`
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

	configFile := &github_pb.DeployerConfig{}
	if err := protoread.PullAndParse(ctx, s3Client, cfg.ConfigFile, configFile); err != nil {
		return err
	}

	environmentDeployers := map[string]github.IDeployer{}

	db, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	for _, envConfigFile := range configFile.TargetEnvironments {
		env := &environment_pb.Environment{}
		if err := protoread.PullAndParse(ctx, s3Client, envConfigFile, env); err != nil {
			return err
		}

		envDeployer, err := deployer.NewDeployer(db, env, awsConfig)
		if err != nil {
			return err
		}

		environmentDeployers[env.FullName] = envDeployer
	}

	githubClient, err := github.NewEnvClient(ctx)
	if err != nil {
		return err
	}

	refLookup := RefLookup(configFile.Refs)

	githubWorker, err := github.NewWebhookWorker(
		githubClient,
		environmentDeployers,
		refLookup,
	)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	github_pb.RegisterWebhookTopicServer(grpcServer, githubWorker)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.WorkerPort))
	if err != nil {
		return err
	}
	log.WithField(ctx, "port", cfg.WorkerPort).Info("Begin Worker Server")
	closeOnContextCancel(ctx, grpcServer)

	return grpcServer.Serve(lis)
}

func closeOnContextCancel(ctx context.Context, srv *grpc.Server) {
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()
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
