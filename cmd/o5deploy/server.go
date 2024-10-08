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
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/awsinfra"
	"github.com/pentops/o5-deploy-aws/internal/cf/app"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
	"github.com/pentops/o5-deploy-aws/internal/github"
	"github.com/pentops/o5-deploy-aws/internal/localrun"
	"github.com/pentops/o5-deploy-aws/internal/protoread"
	"github.com/pentops/o5-deploy-aws/internal/service"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_tpb"
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
	cmdGroup.Add("template", commander.NewCommand(runTemplate))

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

	templateStore, err := deployer.NewS3TemplateStore(ctx, s3Client, cfg.CFTemplates)
	if err != nil {
		return err
	}

	infraStore, err := awsinfra.NewStorage(db)
	if err != nil {
		return err
	}
	cfAdapter, err := awsinfra.NewCFAdapterFromConfig(ctx, awsConfig, []string{cfg.CallbackARN})
	if err != nil {
		return err
	}

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

	githubApps := []github.AppConfig{}
	if err := json.Unmarshal([]byte(cfg.GithubAppsJSON), &githubApps); err != nil {
		return fmt.Errorf("GITHUB_APPS env var: %w", err)
	}
	githubClient, err := github.NewMultiOrgClientFromConfigs(githubApps...)
	if err != nil {
		return err
	}

	serviceApp, err := service.NewApp(service.AppDeps{
		DB:           sqrlx.NewPostgres(db),
		GithubClient: githubClient,
		SpecBuilder:  specBuilder,
	})

	if err != nil {
		return err
	}

	middleware := service.GRPCMiddleware()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(middleware...)),
	)

	serviceApp.RegisterGRPC(grpcServer)

	messaging_tpb.RegisterRawMessageTopicServer(grpcServer, rawWorker)
	awsinfra_tpb.RegisterCloudFormationRequestTopicServer(grpcServer, awsInfraRunner)
	awsinfra_tpb.RegisterPostgresRequestTopicServer(grpcServer, pgMigrateRunner)
	awsinfra_tpb.RegisterECSReplyTopicServer(grpcServer, pgMigrateRunner)
	awsinfra_tpb.RegisterECSRequestTopicServer(grpcServer, ecsWorker)

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

func runTemplate(ctx context.Context, cfg struct {
	AppFilename string `flag:"app" description:"application file"`
	Version     string `flag:"version" default:"VERSION" description:"version tag"`
}) error {

	if cfg.AppFilename == "" {
		return fmt.Errorf("missing application file (-app)")
	}

	appConfig := &application_pb.Application{}
	if err := protoread.PullAndParse(ctx, cfg.AppFilename, appConfig); err != nil {
		return err
	}

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

func getCluster(ctx context.Context, clusterFilename string, envName string) (*environment_pb.Cluster, *environment_pb.Environment, error) {
	clusterFile := &environment_pb.CombinedConfig{}
	if err := protoread.PullAndParse(ctx, clusterFilename, clusterFile); err != nil {
		return nil, nil, err
	}

	cluster := &environment_pb.Cluster{
		Name: clusterFile.Name,
	}
	switch et := clusterFile.Provider.(type) {
	case *environment_pb.CombinedConfig_EcsCluster:
		cluster.Provider = &environment_pb.Cluster_EcsCluster{
			EcsCluster: et.EcsCluster,
		}
	default:
		return nil, nil, fmt.Errorf("unsupported provider %T", clusterFile.Provider)
	}

	var env *environment_pb.Environment

	for _, e := range clusterFile.Environments {
		if e.FullName == envName {
			env = e
			break
		}
	}

	if env == nil {
		return nil, nil, fmt.Errorf("environment %s not found in cluster", envName)
	}

	return cluster, env, nil
}

func runLocalDeploy(ctx context.Context, cfg struct {
	ClusterFilename string `flag:"cluster" description:"cluster file"`
	EnvName         string `flag:"envname" description:"environment name"`
	AppFilename     string `flag:"app" description:"application file"`
	Version         string `flag:"version" description:"version tag"`
	SidecarVersion  string `flag:"sidecar-version" required:"false" description:"sidecar version tag - defaults to the cluster config"`
	ScratchBucket   string `flag:"scratch-bucket" required:"false" description:"An S3 bucket name to upload templates"`

	Auto bool `flag:"auto" description:"Automatically approve plan"`

	// Stratedy Flags
	RotateSecrets   bool `flag:"rotate-secrets" description:"rotate secrets - rotate any existing secrets (e.g. db creds)"`
	CancelUpdate    bool `flag:"cancel-update" description:"cancel update - cancel any ongoing update prior to deployment"`
	SlowMode        bool `flag:"slow" description:"Default is to run the quick flag"`
	InfraOnly       bool `flag:"infra-only" description:"Deploy with scale at 0"`
	DBOnly          bool `flag:"db-only" description:"Only migrate database"`
	ImportResources bool `flag:"import-resources" description:"Import resources, implies infra-only"`
}) error {

	if cfg.AppFilename == "" {
		return fmt.Errorf("missing application file (-app)")
	}

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	appConfig := &application_pb.Application{}
	if err := protoread.PullAndParse(ctx, cfg.AppFilename, appConfig); err != nil {
		return err
	}

	if appConfig.DeploymentConfig == nil {
		appConfig.DeploymentConfig = &application_pb.DeploymentConfig{}
	}

	cluster, env, err := getCluster(ctx, cfg.ClusterFilename, cfg.EnvName)
	if err != nil {
		return err
	}

	ecsCluster := cluster.GetEcsCluster()
	if cfg.SidecarVersion != "" {
		ecsCluster.SidecarImageVersion = cfg.SidecarVersion
	}

	awsTarget := env.GetAws()
	if awsTarget == nil {
		return fmt.Errorf("AWS Deployer requires the type of environment provider to be AWS")
	}

	if cfg.ScratchBucket == "" {
		cfg.ScratchBucket = fmt.Sprintf("%s.o5-deployer.%s.%s", cluster.Name, ecsCluster.AwsRegion, ecsCluster.GlobalNamespace)
	}

	templateStore, err := deployer.NewS3TemplateStore(ctx, s3Client, cfg.ScratchBucket)
	if err != nil {
		return err
	}

	infra, err := localrun.NewInfraAdapterFromConfig(ctx, awsConfig)
	if err != nil {
		return err
	}

	return localrun.RunLocalDeploy(ctx, templateStore, infra, localrun.Spec{
		Version:       cfg.Version,
		AppConfig:     appConfig,
		EnvConfig:     env,
		ClusterConfig: cluster,
		ConfirmPlan:   !cfg.Auto,
		Flags: &awsdeployer_pb.DeploymentFlags{
			RotateCredentials: cfg.RotateSecrets,
			QuickMode:         !cfg.SlowMode,
			InfraOnly:         cfg.InfraOnly,
			CancelUpdates:     cfg.CancelUpdate,
			DbOnly:            cfg.DBOnly,
			ImportResources:   cfg.ImportResources,
		},
	})

}
