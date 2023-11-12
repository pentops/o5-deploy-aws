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
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
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

	stateStore := deployer.NewLocalStateStore(clientSet)
	if err != nil {
		return err
	}

	deployer, err := deployer.NewDeployer(stateStore, env, clientSet)
	if err != nil {
		return err
	}

	stateStore.DeployerEvent = func(ctx context.Context, event *deployer_pb.DeploymentEvent) error {
		return deployer.RegisterEvent(ctx, event)
	}

	deployer.RotateSecrets = flagConfig.rotateSecrets

	if err := deployer.Deploy(ctx, built, flagConfig.cancelUpdate); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return stateStore.Wait()
}
