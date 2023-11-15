package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/localrun"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"golang.org/x/sync/errgroup"
)

type flagConfig struct {
	envFilename   string
	appFilename   string
	version       string
	dryRun        bool
	rotateSecrets bool
	cancelUpdate  bool
}

func main() {
	cfg := flagConfig{}
	flag.StringVar(&cfg.envFilename, "env", "", "environment file")
	flag.StringVar(&cfg.appFilename, "app", "", "application file")
	flag.StringVar(&cfg.version, "version", "", "version tag")
	flag.BoolVar(&cfg.dryRun, "dry", false, "dry run - print template and exit")
	flag.BoolVar(&cfg.rotateSecrets, "rotate-secrets", false, "rotate secrets - rotate any existing secrets (e.g. db creds)")
	flag.BoolVar(&cfg.cancelUpdate, "cancel-update", false, "cancel update - cancel any ongoing update prior to deployment")

	flag.Parse()

	if cfg.appFilename == "" {
		fmt.Fprintln(os.Stderr, "missing application file (-app)")
		os.Exit(1)
	}

	if !cfg.dryRun {
		if cfg.envFilename == "" {
			fmt.Fprintln(os.Stderr, "missing environment file (-env)")
			os.Exit(1)
		}

		if cfg.version == "" {
			fmt.Fprintln(os.Stderr, "missing version (-version)")
			os.Exit(1)
		}
	}

	ctx := context.Background()
	if err := do(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func do(ctx context.Context, flagConfig flagConfig) error {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)

	appConfig := &application_pb.Application{}
	if err := protoread.PullAndParse(ctx, s3Client, flagConfig.appFilename, appConfig); err != nil {
		return err
	}

	app, err := app.BuildApplication(appConfig, flagConfig.version)
	if err != nil {
		return err
	}
	built := app.Build()

	if flagConfig.dryRun {
		tpl := built.Template
		yaml, err := tpl.YAML()
		if err != nil {
			return err
		}
		fmt.Println(string(yaml))
		return nil
	}

	env := &environment_pb.Environment{}
	if err := protoread.PullAndParse(ctx, s3Client, flagConfig.envFilename, env); err != nil {
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

	eventLoop := localrun.NewLocalEventLoop()

	stateStore := localrun.NewLocalStateStore(eventLoop)

	if dbURL := os.Getenv("POSTGRES_URL"); dbURL != "" {
		db, err := service.OpenDatabase(ctx)
		if err != nil {
			return err
		}

		pgStore, err := deployer.NewPostgresStateStore(db, []*environment_pb.Environment{env})
		if err != nil {
			return err
		}

		stateStore.StoreCallback = pgStore.StoreDeploymentEvent
	}

	if err := stateStore.AddEnvironment(env); err != nil {
		return err
	}

	deploymentManager, err := deployer.NewDeployer(stateStore, awsTarget.ScratchBucket, s3Client)

	if err != nil {
		return err
	}

	deploymentManager.RotateSecrets = flagConfig.rotateSecrets
	deploymentManager.CancelUpdates = flagConfig.cancelUpdate

	if err := localrun.RegisterDeployerHandlers(eventLoop, deploymentManager); err != nil {
		return err
	}

	awsRunner := awsinfra.NewRunner(clientSet, eventLoop)

	if err := localrun.RegisterLocalHandlers(eventLoop, awsRunner); err != nil {
		return err
	}

	stackName := fmt.Sprintf("%s-%s", env.FullName, app.AppName())

	poller, err := awsRunner.PollStack(ctx, stackName)
	if err != nil {
		return err
	}

	if err := deploymentManager.BeginDeployment(ctx, built, env.FullName); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return poller.Wait(ctx)
	})
	eg.Go(func() error {
		return eventLoop.Wait(ctx)
	})
	eg.Go(func() error {
		defer cancel()
		// Cancel should run even if the Wait function returns no error, this
		// stops the poller and event loop
		return stateStore.Wait(ctx)
	})

	return eg.Wait()
}
