package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type flagConfig struct {
	envFilename string
	appFilename string
	version     string
	wait        bool
}

func main() {
	cfg := flagConfig{}
	flag.StringVar(&cfg.envFilename, "env", "", "environment file")
	flag.StringVar(&cfg.appFilename, "app", "", "application file")
	flag.StringVar(&cfg.version, "version", "", "version tag")

	flag.Parse()

	if cfg.envFilename == "" {
		fmt.Fprintln(os.Stderr, "missing environment file (-env)")
		os.Exit(1)
	}

	if cfg.appFilename == "" {
		fmt.Fprintln(os.Stderr, "missing application file (-app)")
		os.Exit(1)
	}

	if cfg.version == "" {
		fmt.Fprintln(os.Stderr, "missing version (-version)")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := do(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

}

func do(ctx context.Context, flagConfig flagConfig) error {
	app := &application_pb.Application{}
	if err := protoread.ParseFile(flagConfig.appFilename, app); err != nil {
		return err
	}

	env := &environment_pb.Environment{}
	if err := protoread.ParseFile(flagConfig.envFilename, env); err != nil {
		return err
	}

	built, err := cf.BuildCloudformation(app, env)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	clients := cf.NewDeployerClientsFromConfig(awsConfig)

	deployer, err := cf.NewDeployer(env, flagConfig.version, clients)
	if err != nil {
		return err
	}

	stackName := fmt.Sprintf("%s-%s", env.FullName, app.Name)

	if err := deployer.Deploy(ctx, stackName, built); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	return nil
}
