package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"

	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/awsinspect"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/localrun"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/runner"
	"github.com/pentops/runner/commander"
)

var Version string

func main() {

	cmdGroup := commander.NewCommandSet()

	cmdGroup.Add("local-deploy", commander.NewCommand(runLocalDeploy))
	cmdGroup.Add("watch-events", commander.NewCommand(runWatchEvents))

	remoteGroup := commander.NewCommandSet()
	cmdGroup.Add("remote", remoteGroup)
	remoteGroup.Add("trigger", commander.NewCommand(runTrigger))

	awsGroup := commander.NewCommandSet()
	cmdGroup.Add("aws", awsGroup)
	awsGroup.Add("logs", commander.NewCommand(runAWSLogs))
	awsGroup.Add("redeploy", commander.NewCommand(runRedeploy))

	cmdGroup.RunMain("o5-deploy-aws", Version)

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

type SNSEvent struct {
	Type     string `json:"Type"`
	Message  string `json:"Message"`
	TopicArn string `json:"TopicArn"`
}

func runWatchEvents(ctx context.Context, cfg struct {
	QueueURL string `flag:"queue-url"`
}) error {

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	sqsClient := sqs.NewFromConfig(awsConfig)

	for {
		req, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(cfg.QueueURL),
			MaxNumberOfMessages: 10,
		})
		if err != nil {
			return err
		}

		for _, msg := range req.Messages {
			body := []byte(*msg.Body)
			snsEvent := &SNSEvent{}
			if err := json.Unmarshal(body, snsEvent); err == nil && snsEvent.Type == "Notification" {
				body = []byte(snsEvent.Message)
			}

			if snsEvent.TopicArn == "" {
				snsEvent.TopicArn = "fake-o5-cloudwatch-events"
			}

			if strings.HasSuffix(snsEvent.TopicArn, "-o5-cloudwatch-events") {
				infraEvent := &awsinfra.InfraEvent{}
				if err := json.Unmarshal(body, infraEvent); err != nil {
					return err
				}

				if err := awsinfra.HandleInfraEvent(ctx, infraEvent); err != nil {
					return fmt.Errorf("handle infra event: %w", err)
				}
			} else {
				bb := &bytes.Buffer{}
				if err := json.Indent(bb, body, "  ", "  "); err != nil {
					return err
				}

				fmt.Printf("Unhandled Message on %s\n  %s\n", snsEvent.TopicArn, bb.String())
				continue
			}

			if _, err := sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(cfg.QueueURL),
				ReceiptHandle: msg.ReceiptHandle,
			}); err != nil {
				return err
			}
		}
	}
}

func runTrigger(ctx context.Context, cfg struct {
	AppName string `env:"APP_NAME" flag:"repo"`
	Org     string `env:"GITHUB_ORG" flag:"org"`
	EnvName string `env:"ENV_NAME" flag:"env"`
	Version string `env:"VERSION" flag:"version"`
	API     string `env:"O5_API" flag:"api"`
}) error {
	deploymentID := uuid.NewString()
	triggerBody, _ := json.MarshalIndent(map[string]interface{}{
		"environmentName": cfg.EnvName,
		"source": map[string]interface{}{
			"github": map[string]interface{}{
				"owner":  cfg.Org,
				"repo":   cfg.AppName,
				"commit": cfg.Version,
			},
		},
	}, "", "  ")

	fmt.Printf("REQ: %s\n", triggerBody)
	req, err := http.NewRequest("POST", cfg.API+"/deployer/v1/c/deployments/"+deploymentID, bytes.NewReader(triggerBody))
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	io.Copy(os.Stdout, res.Body) // nolint:errcheck
	fmt.Println("")
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	fmt.Printf("DeploymentID: %s\n", deploymentID)
	return nil
}

func runRedeploy(ctx context.Context, cfg struct {
	ClusterName string `flag:"cluster-name"`
}) error {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	ecsClient := ecs.NewFromConfig(awsConfig)

	listRes, err := ecsClient.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: aws.String(cfg.ClusterName),
	})
	if err != nil {
		return err
	}

	for _, arn := range listRes.ServiceArns {
		fmt.Printf("Service: %s\n", arn)

		_, err := ecsClient.UpdateService(ctx, &ecs.UpdateServiceInput{
			ForceNewDeployment: true,
			Service:            aws.String(arn),
			Cluster:            aws.String(cfg.ClusterName),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func runAWSLogs(ctx context.Context, cfg struct {
	StackName string `flag:"stack-name"`
}) error {
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	formationClient := cloudformation.NewFromConfig(awsConfig)

	serviceSummary, err := awsinspect.GetStackServices(ctx, formationClient, cfg.StackName)
	if err != nil {
		return err
	}

	ecsClient := ecs.NewFromConfig(awsConfig)
	logStreams, err := awsinspect.GetAllLogStreams(ctx, ecsClient, serviceSummary)
	if err != nil {
		return err
	}

	logClient := cloudwatchlogs.NewFromConfig(awsConfig)

	fromTime := time.Now()

	rg := runner.NewGroup()
	for _, logStream := range logStreams {
		logStream := logStream
		rg.Add(logStream.Container, func(ctx context.Context) error {
			return awsinspect.TailLogStream(ctx, logClient, logStream, fromTime)
		})
	}

	return rg.Run(ctx)

}
