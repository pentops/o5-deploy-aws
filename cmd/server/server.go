package main

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"google.golang.org/grpc"
	"gopkg.daemonl.com/envconf"
)

type envConfig struct {
	EnvConfig  string `env:"ENV_CONFIG"`
	WorkerPort int    `env:"WORKER_PORT" default:"8081"`
}

var Version string

func main() {
	ctx := context.Background()
	ctx = log.WithFields(ctx, map[string]interface{}{
		"application": "userauth",
		"version":     Version,
	})

	config := envConfig{}
	if err := envconf.Parse(&config); err != nil {
		log.WithError(ctx, err).Error("Failed to load config")
		os.Exit(1)
	}

	if err := do(ctx, config); err != nil {
		log.WithError(ctx, err).Error("Failed to serve")
		os.Exit(1)
	}
}

func do(ctx context.Context, cfg envConfig) error {

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)
	deployerClients := deployer.NewDeployerClientsFromConfig(awsConfig)

	env := &environment_pb.Environment{}
	if err := protoread.PullAndParse(ctx, s3Client, cfg.EnvConfig, env); err != nil {
		return err
	}

	envDeployer, err := deployer.NewDeployer(env, deployerClients)
	if err != nil {
		return err
	}

	githubClient, err := github.NewEnvClient(ctx)
	if err != nil {
		return err
	}

	asyncDeployer := &AsyncDeployer{
		IDeployer: envDeployer,
	}

	githubWorker, err := github.NewWebhookWorker(
		githubClient,
		asyncDeployer,
	)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	github_pb.RegisterWebhookTopicServer(grpcServer, githubWorker)

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

type AsyncDeployer struct {
	github.IDeployer
}

func (d *AsyncDeployer) Deploy(ctx context.Context, appStack *app.BuiltApplication, cancelUpdate bool) error {
	// TODO: Env Deployer should run asynchronously, in multiple stages, inside
	// the environment.
	// In this version, the deployment must run in the same environment as the webhook server.
	// This version is only appropriate for single env dev deployments and will
	// require quite a lot of work, e.g. listening for cloudformation and ECS
	// events.
	go func() {
		ctx := context.Background()
		err := d.IDeployer.Deploy(ctx, appStack, cancelUpdate)
		if err != nil {
			log.WithError(ctx, err).Error("Failed to deploy")
		} else {
			log.Info(ctx, "Deployed")
		}

		// TODO: Write outcome back to Github, or otherwise notify the outcome
	}()
	return nil
}
