package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/localrun"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/runner"
	"github.com/pentops/runner/commander"
	"github.com/pentops/sqrlx.go/sqrlx"
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
	ConfigFile string `env:"CONFIG_FILE"`
	GRPCPort   int    `env:"GRPC_PORT" default:"8081"`

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

	configFile := &github_pb.DeployerConfig{}
	if err := protoread.PullAndParse(ctx, s3Client, cfg.ConfigFile, configFile); err != nil {
		return err
	}

	environments := deployer.EnvList([]*environment_pb.Environment{})

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

	clientSet := &awsinfra.ClientSet{
		AssumeRoleARN: cfg.DeployerAssumeRole,
		AWSConfig:     awsConfig,
	}

	outbox, err := newOutboxSender(db)
	if err != nil {
		return err
	}
	awsInfraRunner := awsinfra.NewInfraWorker(clientSet, outbox)
	awsInfraRunner.CallbackARNs = []string{cfg.CallbackARN}

	specBuilder, err := deployer.NewSpecBuilder(environments, cfg.CFTemplates, s3Client)
	if err != nil {
		return err
	}

	stateMachines, err := deployer.NewStateMachines()
	if err != nil {
		return err
	}

	deploymentWorker, err := deployer.NewDeployerWorker(db, specBuilder, stateMachines)
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
		db,
		githubClient,
		refLookup,
	)
	if err != nil {
		return err
	}

	service, err := service.NewDeployerService(db, githubClient)
	if err != nil {
		return err
	}

	runGroup := runner.NewGroup(runner.WithName("main"), runner.WithCancelOnSignals())

	runGroup.Add("grpcServer", func(ctx context.Context) error {
		grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
			grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
		))
		github_pb.RegisterWebhookTopicServer(grpcServer, githubWorker)
		deployer_spb.RegisterDeploymentQueryServiceServer(grpcServer, service)
		deployer_spb.RegisterDeploymentCommandServiceServer(grpcServer, service)
		deployer_tpb.RegisterAWSCommandTopicServer(grpcServer, awsInfraRunner)
		deployer_tpb.RegisterDeployerTopicServer(grpcServer, deploymentWorker)
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
	})

	return runGroup.Run(ctx)
}

type outboxSender struct {
	db *sqrlx.Wrapper
}

func newOutboxSender(conn sqrlx.Connection) (*outboxSender, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &outboxSender{
		db: db,
	}, nil
}

func (s *outboxSender) PublishEvent(ctx context.Context, msg outbox.OutboxMessage) error {
	return s.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return outbox.Send(ctx, tx, msg)
	})
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

	app, err := app.BuildApplication(appConfig, cfg.Version)
	if err != nil {
		return err
	}
	built := app.Build()

	if cfg.DryRun {
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

	awsRunner := localrun.NewInfraAdapter(clientSet)
	stateStore := localrun.NewStateStore()
	deploymentManager, err := deployer.NewSpecBuilder(stateStore, cfg.ScratchBucket, s3Client)
	if err != nil {
		return err
	}
	deploymentManager.RotateSecrets = cfg.RotateSecrets
	deploymentManager.CancelUpdates = cfg.CancelUpdate
	deploymentManager.QuickMode = cfg.QuickMode

	eventLoop := localrun.NewEventLoop(awsRunner, stateStore, deploymentManager)

	if err := stateStore.AddEnvironment(env); err != nil {
		return err
	}

	trigger := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId:    uuid.NewString(),
		Application:     appConfig,
		Version:         cfg.Version,
		EnvironmentName: env.FullName,
	}

	return eventLoop.Run(ctx, trigger)

}
