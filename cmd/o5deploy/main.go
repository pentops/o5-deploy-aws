package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsapi"
	"github.com/pentops/o5-deploy-aws/internal/apps/localrun"
	"github.com/pentops/o5-deploy-aws/internal/apps/service"
	"github.com/pentops/o5-deploy-aws/internal/apps/service/github"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
	"github.com/pentops/o5-deploy-aws/internal/protoread"
	"github.com/pentops/runner/commander"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/pressly/goose"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var Version string

func main() {

	cmdGroup := commander.NewCommandSet()

	cmdGroup.Add("validate", commander.NewCommand(runValidate))
	cmdGroup.Add("serve", commander.NewCommand(runServe))
	cmdGroup.Add("migrate", commander.NewCommand(runMigrate))
	cmdGroup.Add("local-deploy", commander.NewCommand(runLocalDeploy))
	cmdGroup.Add("template", commander.NewCommand(runTemplate))

	cmdGroup.RunMain("o5-deploy-aws", Version)
}

func runMigrate(ctx context.Context, config struct {
	MigrationsDir string `env:"MIGRATIONS_DIR" default:"./ext/db"`
	DBConfig
}) error {

	db, err := config.OpenDatabase(ctx)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info(ctx, "Running migrations")
	err = goose.Up(db, "/migrations")
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info(ctx, "Migrations complete")
	return nil
}

func runServe(ctx context.Context, cfg struct {
	GRPCPort int `env:"GRPC_PORT" default:"8081"`

	DeployerAssumeRole string `env:"DEPLOYER_ASSUME_ROLE"`
	CFTemplates        string `env:"CF_TEMPLATES"`
	GithubAppsJSON     string `env:"GITHUB_APPS"`

	DBConfig
}) error {

	log.WithField(ctx, "PORT", cfg.GRPCPort).Info("Boot")

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// NOTE: This config does not participate in 'assume role' (if any).
	// The O5 deployer *service* should have write access to the bucket on its own behalf.
	// The AWS adaptor should be able to read the configs with the assumed role
	s3Client := s3.NewFromConfig(awsConfig)

	dbConn, err := cfg.OpenDatabase(ctx)
	if err != nil {
		return err
	}

	db := sqrlx.NewPostgres(dbConn)

	middleware := service.GRPCMiddleware()
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(middleware...),
	)

	clients, err := awsapi.LoadFromConfig(ctx, awsConfig, awsapi.WithAssumeRole(cfg.DeployerAssumeRole))
	if err != nil {
		return err
	}

	infraApp, err := aws.NewApp(db, clients)
	if err != nil {
		return err
	}
	infraApp.RegisterGRPC(grpcServer)

	templateStore, err := deployer.NewS3TemplateStore(ctx, s3Client, cfg.CFTemplates)
	if err != nil {
		return err
	}

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
		DB:           db,
		GithubClient: githubClient,
		SpecBuilder:  specBuilder,
	})
	if err != nil {
		return err
	}

	serviceApp.RegisterGRPC(grpcServer)
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
	RDSIAM      bool   `flag:"rds-iam" description:"make all RDS hosts Aurora-IAM style"`
	JSON        bool   `flag:"json" description:"output as JSON"`
}) error {

	if cfg.AppFilename == "" {
		return fmt.Errorf("missing application file (-app)")
	}

	appConfig := &application_pb.Application{}
	if err := protoread.PullAndParse(ctx, cfg.AppFilename, appConfig); err != nil {
		return err
	}

	hostType := environment_pb.RDSAuth_SecretsManager
	if cfg.RDSIAM {
		hostType = environment_pb.RDSAuth_Iam
	}
	built, err := appbuilder.BuildApplication(appbuilder.AppInput{
		Application: appConfig,
		VersionTag:  cfg.Version,
		RDSHosts:    fakeHosts(hostType),
	})
	if err != nil {
		return err
	}

	tpl := built.Template
	if cfg.JSON {
		json, err := tpl.JSON()
		if err != nil {
			return err
		}
		fmt.Println(string(json))

		return nil
	}
	yaml, err := tpl.YAML()
	if err != nil {
		return err
	}
	fmt.Println(string(yaml))

	return nil
}

func runValidate(ctx context.Context, cfg struct {
	AppFilename string `flag:"app" required:"false" description:"application file"`
	ClusterFile string `flag:"cluster" required:"false" description:"cluster file"`
}) error {
	if cfg.AppFilename == "" && cfg.ClusterFile == "" {
		return fmt.Errorf("requires application file (-app) and/or cluster file (-cluster)")
	}

	pv, err := protovalidate.New()
	if err != nil {
		return err
	}

	if cfg.AppFilename != "" {
		appConfig := &application_pb.Application{}
		if err := protoread.PullAndParse(ctx, cfg.AppFilename, appConfig); err != nil {
			return err
		}
		if err := pv.Validate(appConfig); err != nil {
			return err
		}
	}

	if cfg.ClusterFile != "" {
		clusterFile := &environment_pb.CombinedConfig{}
		if err := protoread.PullAndParse(ctx, cfg.ClusterFile, clusterFile); err != nil {
			return err
		}
		if err := pv.Validate(clusterFile); err != nil {
			return err
		}
	}
	return nil
}

type fakeHosts environment_pb.RDSAuthTypeKey

func (hh fakeHosts) FindRDSHost(string) (*appbuilder.RDSHost, bool) {
	return &appbuilder.RDSHost{
		AuthType: environment_pb.RDSAuthTypeKey(hh),
	}, true
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
	case *environment_pb.CombinedConfig_Aws:
		cluster.Provider = &environment_pb.Cluster_Aws{
			Aws: et.Aws,
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
	DBRecreate      bool `flag:"db-recreate" description:"Destroy and Recreate database"`
	DBDestroy       bool `flag:"db-destroy" description:"Destroy database"`
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

	ecsCluster := cluster.GetAws()
	if cfg.SidecarVersion != "" {
		ecsCluster.O5Sidecar.ImageVersion = cfg.SidecarVersion
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

	clients, err := awsapi.LoadFromConfig(ctx, awsConfig)
	if err != nil {
		return err
	}

	infra, err := localrun.NewInfraAdapter(ctx, clients)
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
			DestroyDatabase:   cfg.DBDestroy,
			RecreateDatabase:  cfg.DBRecreate,
			ImportResources:   cfg.ImportResources,
		},
	})

}
