package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/cf/app"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/localrun"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/runner/commander"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/pressly/goose"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
)

var Version string

func main() {

	cmdGroup := commander.NewCommandSet()

	cmdGroup.Add("serve", commander.NewCommand(runServe))
	cmdGroup.Add("migrate", commander.NewCommand(runMigrate))
	cmdGroup.Add("data-pass", commander.NewCommand(runDataPass))
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

func runDataPass(ctx context.Context, config struct{}) error {
	conn, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return err
	}

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		return err
	}

	{
		stackQuery, err := deployer_spb.NewStackPSMQuerySet(
			deployer_spb.DefaultStackPSMQuerySpec(stateMachines.Stack.StateTableSpec()),
			psm.StateQueryOptions{},
		)
		if err != nil {
			return fmt.Errorf("failed to build stack query: %w", err)
		}

		page := &psml_pb.PageRequest{}
		for {
			res := &deployer_spb.ListStacksResponse{}
			if err := stackQuery.List(ctx, db, &deployer_spb.ListStacksRequest{
				Page: page,
			}, res); err != nil {
				return err
			}
			if res.Page != nil && res.Page.NextToken != nil {
				page.Token = res.Page.NextToken
			} else {
				break
			}

			for _, stack := range res.Stacks {
				fmt.Printf("stack %s\n", protojson.Format(stack))
				eventPage := &psml_pb.PageRequest{}
				for {
					res := &deployer_spb.ListStackEventsResponse{}
					if err := stackQuery.ListEvents(ctx, db, &deployer_spb.ListStackEventsRequest{
						StackId: stack.StackId,
						Page:    eventPage,
					}, res); err != nil {
						return err
					}

					if res.Page != nil && res.Page.NextToken != nil {
						eventPage.Token = res.Page.NextToken
					} else {
						break
					}
				}

			}
		}
	}

	{
		deploymentQuery, err := deployer_spb.NewDeploymentPSMQuerySet(
			deployer_spb.DefaultDeploymentPSMQuerySpec(stateMachines.Deployment.StateTableSpec()),
			psm.StateQueryOptions{},
		)
		if err != nil {
			return fmt.Errorf("failed to build deployment query: %w", err)
		}
		page := &psml_pb.PageRequest{}
		for {
			res := &deployer_spb.ListDeploymentsResponse{}
			if err := deploymentQuery.List(ctx, db, &deployer_spb.ListDeploymentsRequest{
				Page: page,
			}, res); err != nil {
				return err
			}
			if res.Page != nil && res.Page.NextToken != nil {
				page.Token = res.Page.NextToken
			} else {
				break
			}

			for _, deployment := range res.Deployments {
				fmt.Printf("deployment %s\n", protojson.Format(deployment))
				eventPage := &psml_pb.PageRequest{}
				for {
					res := &deployer_spb.ListDeploymentEventsResponse{}
					if err := deploymentQuery.ListEvents(ctx, db, &deployer_spb.ListDeploymentEventsRequest{
						DeploymentId: deployment.DeploymentId,
						Page:         eventPage,
					}, res); err != nil {
						return err
					}

					if res.Page != nil && res.Page.NextToken != nil {
						eventPage.Token = res.Page.NextToken
					} else {
						break
					}
				}

			}
		}

	}
	return nil
}

func runServe(ctx context.Context, cfg struct {
	GRPCPort int `env:"GRPC_PORT" default:"8081"`

	DeployerAssumeRole string `env:"DEPLOYER_ASSUME_ROLE"`
	CFTemplates        string `env:"CF_TEMPLATES"`
	CallbackARN        string `env:"CALLBACK_ARN"`
	GithubAppsJSON     string `env:"GITHUB_APPS"`
}) error {

	log.WithField(ctx, "PORT", cfg.GRPCPort).Info("Boot")

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if cfg.DeployerAssumeRole != "" {
		stsClient := sts.NewFromConfig(awsConfig)
		provider := stscreds.NewAssumeRoleProvider(stsClient, cfg.DeployerAssumeRole)
		creds := aws.NewCredentialsCache(provider)

		assumeRoleConfig, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(creds),
		)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		awsConfig = assumeRoleConfig
	}

	{
		stsClient := sts.NewFromConfig(awsConfig)
		identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
		if err != nil {
			return fmt.Errorf("failed to get caller identity: %w", err)
		}
		log.WithFields(ctx, map[string]interface{}{
			"account":     aws.ToString(identity.Account),
			"arn":         aws.ToString(identity.Arn),
			"user":        aws.ToString(identity.UserId),
			"assumedRole": cfg.DeployerAssumeRole,
		}).Info("Running With AWS Identity")
	}

	s3Client := s3.NewFromConfig(awsConfig)

	db, err := service.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	templateStore := deployer.NewS3TemplateStore(s3Client, cfg.CFTemplates)

	infraStore, err := awsinfra.NewStorage(db)
	if err != nil {
		return err
	}
	cfAdapter := awsinfra.NewCFAdapterFromConfig(awsConfig, []string{cfg.CallbackARN})
	awsInfraRunner := awsinfra.NewInfraWorker(infraStore, cfAdapter)

	ecsWorker, err := awsinfra.NewECSWorker(infraStore, ecs.NewFromConfig(awsConfig))
	if err != nil {
		return err
	}

	rawWorker := awsinfra.NewRawMessageWorker(awsInfraRunner, ecsWorker)

	dbMigrator := awsinfra.NewDBMigrator(secretsmanager.NewFromConfig(awsConfig))

	pgMigrateRunner := awsinfra.NewPostgresMigrateWorker(infraStore, dbMigrator)

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

	githubApps := []github.AppConfig{}
	if err := json.Unmarshal([]byte(cfg.GithubAppsJSON), &githubApps); err != nil {
		return fmt.Errorf("GITHUB_APPS env var: %w", err)
	}
	githubClient, err := github.NewMultiOrgClientFromConfigs(githubApps...)
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

	commandService, err := service.NewCommandService(db, githubClient, stateMachines)
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

	deployer_tpb.RegisterDeployerTopicServer(grpcServer, deploymentWorker)

	messaging_tpb.RegisterRawMessageTopicServer(grpcServer, rawWorker)

	deployer_tpb.RegisterCloudFormationRequestTopicServer(grpcServer, awsInfraRunner)
	deployer_tpb.RegisterCloudFormationReplyTopicServer(grpcServer, deploymentWorker)

	deployer_tpb.RegisterPostgresRequestTopicServer(grpcServer, pgMigrateRunner)
	deployer_tpb.RegisterPostgresReplyTopicServer(grpcServer, deploymentWorker)

	deployer_tpb.RegisterECSReplyTopicServer(grpcServer, pgMigrateRunner)
	deployer_tpb.RegisterECSRequestTopicServer(grpcServer, ecsWorker)

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
	InfraOnly     bool   `flag:"infra-only" description:"Deploy with scale at 0"`
	DBOnly        bool   `flag:"db-only" description:"Only migrate database"`
	Auto          bool   `flag:"auto" description:"Automatically approve plan"`
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

		for _, target := range built.SnsTopics {
			fmt.Printf("SNS Topic: %s\n", target)
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

	templateStore := deployer.NewS3TemplateStore(s3Client, cfg.ScratchBucket)

	infra, err := localrun.NewInfraAdapterFromConfig(ctx, awsConfig)
	if err != nil {
		return err
	}

	return localrun.RunLocalDeploy(ctx, templateStore, infra, localrun.Spec{
		Version:     cfg.Version,
		AppConfig:   appConfig,
		EnvConfig:   env,
		ConfirmPlan: !cfg.Auto,
		Flags: &deployer_pb.DeploymentFlags{
			RotateCredentials: cfg.RotateSecrets,
			CancelUpdates:     cfg.CancelUpdate,
			QuickMode:         cfg.QuickMode,
			InfraOnly:         cfg.InfraOnly,
			DbOnly:            cfg.DBOnly,
		},
	})

}
