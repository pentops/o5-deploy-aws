package main

import (
	"context"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/localrun"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/runner/commander"
	"github.com/pressly/goose"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version string

func main() {

	cmdGroup := commander.NewCommandSet()

	cmdGroup.Add("serve", commander.NewCommand(runServe))
	cmdGroup.Add("migrate", commander.NewCommand(runMigrate))
	cmdGroup.Add("local-deploy", commander.NewCommand(runLocalDeploy))

	cmdGroup.RunMain("o5-deploy-aws", Version)
}

func runMigrate(ctx context.Context, config struct {
	MigrationsDir string `env:"MIGRATIONS_DIR" default:"./ext/db"`
}) error {

	db, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	return goose.Up(db, "/migrations")
}

func runServe(ctx context.Context, cfg struct {
	GRPCPort int `env:"GRPC_PORT" default:"8081"`

	DeployerAssumeRole string `env:"DEPLOYER_ASSUME_ROLE"`
	CFTemplates        string `env:"CF_TEMPLATES"`
	CallbackARN        string `env:"CALLBACK_ARN"`
}) error {

	log.WithField(ctx, "PORT", cfg.GRPCPort).Info("Boot")

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	db, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	templateStore := deployer.NewS3TemplateStore(s3Client, cfg.CFTemplates)

	clientSet := &awsinfra.ClientSet{
		AssumeRoleARN: cfg.DeployerAssumeRole,
		AWSConfig:     awsConfig,
	}

	infraStore, err := awsinfra.NewStorage(db)
	if err != nil {
		return err
	}
	awsInfraRunner := awsinfra.NewInfraWorker(clientSet, infraStore)
	awsInfraRunner.CallbackARNs = []string{cfg.CallbackARN}

	pgMigrateRunner := awsinfra.NewPostgresMigrateWorker(clientSet, infraStore)

	specBuilder, err := deployer.NewSpecBuilder(templateStore)
	if err != nil {
		return err
	}

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		return err
	}

	deploymentWorker, err := deployer.NewDeployerWorker(db, specBuilder, stateMachines)
	if err != nil {
		return err
	}

	githubClient, err := github.NewEnvClient(ctx)
	if err != nil {
		return err
	}

	refLookup, err := github.NewRefStore(db)
	if err != nil {
		return err
	}

	githubWorker, err := github.NewWebhookWorker(
		db,
		githubClient,
		refLookup,
	)
	if err != nil {
		return err
	}

	commandService, err := service.NewDeployerService(db, githubClient, stateMachines)
	if err != nil {
		return err
	}

	queryService, err := service.NewQueryService(db, stateMachines)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
	))
	github_pb.RegisterWebhookTopicServer(grpcServer, githubWorker)
	deployer_spb.RegisterDeploymentQueryServiceServer(grpcServer, queryService)
	deployer_spb.RegisterDeploymentCommandServiceServer(grpcServer, commandService)
	deployer_tpb.RegisterCloudFormationRequestTopicServer(grpcServer, awsInfraRunner)
	deployer_tpb.RegisterPostgresMigrationTopicServer(grpcServer, pgMigrateRunner)
	deployer_tpb.RegisterDeployerTopicServer(grpcServer, deploymentWorker)
	deployer_tpb.RegisterCloudFormationReplyTopicServer(grpcServer, deploymentWorker)
	deployer_tpb.RegisterPostgresMigrationReplyTopicServer(grpcServer, deploymentWorker)
	messaging_tpb.RegisterRawMessageTopicServer(grpcServer, awsInfraRunner)

	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		return err
	}
	log.WithField(ctx, "port", cfg.GRPCPort).Info("Begin Worker Server")
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop() // nolint:errcheck
	}()

	return grpcServer.Serve(lis)
}

func runLocalDeploy(ctx context.Context, cfg struct {
	EnvFilename   string `flag:"env" description:"environment file"`
	AppFilename   string `flag:"app" description:"application file"`
	Version       string `flag:"version" description:"version tag"`
	DryRun        bool   `flag:"dry" description:"dry run - print template and exit"`
	RotateSecrets bool   `flag:"rotate-secrets" description:"rotate secrets - rotate any existing secrets (e.g. db creds)"`
	CancelUpdate  bool   `flag:"cancel-update" description:"cancel update - cancel any ongoing update prior to deployment"`
	ScratchBucket string `flag:"scratch-bucket" env:"O5_DEPLOYER_SCRATCH_BUCKET" description:"An S3 bucket name to upload templates"`
	QuickMode     bool   `flag:"quick" description:"Skips scale down/up, calls stack update with all changes once, and skips DB migration"`
}) error {

	if cfg.AppFilename == "" {
		return fmt.Errorf("missing application file (-app)")
	}

	if cfg.ScratchBucket == "" {
		return fmt.Errorf("missing scratch bucket (-scratch-bucket)")
	}

	if !cfg.DryRun {
		if cfg.EnvFilename == "" {
			return fmt.Errorf("missing environment file (-env)")
		}

		if cfg.Version == "" {
			return fmt.Errorf("missing version (-version)")
		}
	}

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	appConfig := &application_pb.Application{}
	if err := protoread.PullAndParse(ctx, s3Client, cfg.AppFilename, appConfig); err != nil {
		return err
	}

	if appConfig.DeploymentConfig == nil {
		appConfig.DeploymentConfig = &application_pb.DeploymentConfig{}
	}

	if cfg.DryRun {
		built, err := app.BuildApplication(appConfig, cfg.Version)
		if err != nil {
			return err
		}
		tpl := built.Template
		yaml, err := tpl.YAML()
		if err != nil {
			return err
		}
		fmt.Println(string(yaml))

		fmt.Println("-----")

		for _, target := range built.SNSTopics {
			fmt.Printf("SNS Topic: %s\n", target.Name)
		}
		return nil
	}

	env := &environment_pb.Environment{}
	if err := protoread.PullAndParse(ctx, s3Client, cfg.EnvFilename, env); err != nil {
		return err
	}

	awsTarget := env.GetAws()
	if awsTarget == nil {
		return fmt.Errorf("AWS Deployer requires the type of environment provider to be AWS")
	}

	clientSet := &awsinfra.ClientSet{
		AssumeRoleARN: awsTarget.O5DeployerAssumeRole,
		AWSConfig:     awsConfig,
	}

	templateStore := deployer.NewS3TemplateStore(s3Client, cfg.ScratchBucket)

	infra := localrun.NewInfraAdapter(clientSet)

	return localrun.RunLocalDeploy(ctx, templateStore, infra, localrun.Spec{
		Version:   cfg.Version,
		AppConfig: appConfig,
		EnvConfig: env,
	})

}
