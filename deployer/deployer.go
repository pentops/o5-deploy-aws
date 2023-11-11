package deployer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	protovalidate "github.com/bufbuild/protovalidate-go"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CloudFormationAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
	UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
	CancelUpdateStack(ctx context.Context, params *cloudformation.CancelUpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CancelUpdateStackOutput, error)
}

type ELBV2API interface {
	DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
}

type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
}

type ECSAPI interface {
	RunTask(ctx context.Context, params *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)

	// used by the TasksStoppedWaiter
	ecs.DescribeTasksAPIClient
}

type SNSAPI interface {
	CreateTopic(ctx context.Context, params *sns.CreateTopicInput, optsFns ...func(*sns.Options)) (*sns.CreateTopicOutput, error)
}

type S3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type DeployerClients struct {
	CloudFormation CloudFormationAPI
	SNS            SNSAPI
	ELB            ELBV2API
	SecretsManager SecretsManagerAPI
	ECS            ECSAPI
	S3             S3API
}

type ClientBuilder interface {
	GetConfig(ctx context.Context, assumeRole string) (*DeployerClients, error)
}

type EventCallback func(ctx context.Context, event *deployer_pb.DeploymentEvent) error

type Deployer struct {
	Environment   *environment_pb.Environment
	AWS           *environment_pb.AWS
	RotateSecrets bool
	clientCache   *DeployerClients
	awsConfig     aws.Config
	EventCallback EventCallback

	deployments map[string]*deployer_pb.DeploymentState
}

func NewDeployer(environment *environment_pb.Environment, awsConfig aws.Config) (*Deployer, error) {

	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}

	if err := validator.Validate(environment); err != nil {
		return nil, err
	}

	awsTarget := environment.GetAws()
	if awsTarget == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	return &Deployer{
		Environment: environment,
		AWS:         awsTarget,
		awsConfig:   awsConfig,
		deployments: map[string]*deployer_pb.DeploymentState{},
	}, nil
}

func (d *Deployer) getDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error) {
	if deployment, ok := d.deployments[id]; ok {
		return deployment, nil
	}
	return nil, fmt.Errorf("no deployment found with id %s", id)
}

func (d *Deployer) RegisterEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {
	if d.EventCallback != nil {
		return d.EventCallback(ctx, event)
	}
	return nil
}

func (d *Deployer) Clients(ctx context.Context) (*DeployerClients, error) {
	if d.clientCache != nil {
		// TODO: Check Expiry
		return d.clientCache, nil
	}

	assumer := sts.NewFromConfig(d.awsConfig)

	tempCredentials, err := assumer.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(d.AWS.O5DeployerAssumeRole),
		DurationSeconds: aws.Int32(900),
		RoleSessionName: aws.String(fmt.Sprintf("o5-deploy-aws-%d", time.Now().Unix())),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assume role '%s': %w", d.AWS.O5DeployerAssumeRole, err)
	}

	assumeRoleConfig, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			*tempCredentials.Credentials.AccessKeyId,
			*tempCredentials.Credentials.SecretAccessKey,
			*tempCredentials.Credentials.SessionToken),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &DeployerClients{
		CloudFormation: cloudformation.NewFromConfig(assumeRoleConfig),
		SNS:            sns.NewFromConfig(assumeRoleConfig),
		ELB:            elasticloadbalancingv2.NewFromConfig(assumeRoleConfig),
		SecretsManager: secretsmanager.NewFromConfig(assumeRoleConfig),
		ECS:            ecs.NewFromConfig(assumeRoleConfig),
		S3:             s3.NewFromConfig(assumeRoleConfig),
	}, nil

}

type Deployment struct {
	ID        string
	StackName string
	App       Application //*app.BuiltApplication
}

type Application interface {
	TemplateJSON() ([]byte, error)
	PostgresDatabases() []*deployer_pb.PostgresDatabase
	Parameters() []*deployer_pb.Parameter
}

func (d *Deployer) Deploy(ctx context.Context, app *app.BuiltApplication, cancelUpdates bool) error {
	stackName := fmt.Sprintf("%s-%s", d.Environment.FullName, app.Name)
	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackName":   stackName,
		"environment": d.Environment.FullName,
	})

	deployment := &Deployment{
		ID:        uuid.NewString(),
		StackName: stackName,
		App:       app,
	}

	d.deployments[deployment.ID] = &deployer_pb.DeploymentState{
		DeploymentId: deployment.ID,
		Status:       deployer_pb.DeploymentStatus_LOCKED,
		Spec: &deployer_pb.DeploymentSpec{
			AppName:         app.Name,
			Version:         app.Version,
			EnvironmentName: d.Environment.FullName,
		},
	}

	if err := d.RegisterEvent(ctx, &deployer_pb.DeploymentEvent{
		DeploymentId: deployment.ID,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Triggered_{
				Triggered: &deployer_pb.DeploymentEventType_Triggered{
					Spec: &deployer_pb.DeploymentSpec{
						AppName:         app.Name,
						Version:         app.Version,
						EnvironmentName: d.Environment.FullName,
					},
				},
			},
		},
	}); err != nil {
		return err
	}

	if err := d.upsertSNSTopics(ctx, app.SNSTopics); err != nil {
		return err
	}

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	remoteStack, err := cf.getOneStack(ctx, stackName)
	if err != nil {
		return err
	}

	if remoteStack == nil {
		return d.createNewDeployment(ctx, deployment)
	}

	if cancelUpdates && remoteStack.StackStatus == types.StackStatusUpdateInProgress {
		if err := cf.cancelUpdate(ctx, stackName); err != nil {
			return err
		}
	}

	remoteStackStable, err := cf.waitForStack(ctx, stackName)
	if err != nil {
		return err
	}

	return d.updateDeployment(ctx, deployment, remoteStackStable.Parameters)
}

func (d *Deployer) upsertSNSTopics(ctx context.Context, topics []*app.SNSTopic) error {
	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	snsClient := clients.SNS

	for _, topic := range topics {
		_, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: aws.String(fmt.Sprintf("%s-%s", d.Environment.FullName, topic.Name)),
		})
		if err != nil {
			return fmt.Errorf("creating sns topic %s: %w", topic.Name, err)
		}
	}
	return nil
}

func (d *Deployer) createNewDeployment(ctx context.Context, deployment *Deployment) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		name:     deployment.StackName,
		template: deployment.App,
		scale:    0,
	})
	if err != nil {
		return err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	// Create, scale 0
	log.Info(ctx, "Create with scale 0")
	if err := cf.createCloudformationStack(ctx, StackArgs{
		Parameters: initialParameters,
		Template:   deployment.App,
		Name:       deployment.StackName,
	}); err != nil {
		return err
	}

	lastState, err := cf.waitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment, "stack create"))
	if err != nil {
		return err
	}

	// Migrate
	log.Info(ctx, "Migrate Database")
	if err := d.migrateData(ctx, lastState.Outputs, deployment.App, true); err != nil {
		return err
	}

	// Scale Up
	log.Info(ctx, "Scale Up")
	if err := cf.setScale(ctx, deployment.StackName, 1); err != nil {
		return err
	}
	if err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment, "scale up")); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) waitStatusCallback(deployment *Deployment, phaseLabel string) waiterCallback {
	return func(ctx context.Context, stackStatus StackStatus) error {
		return d.RegisterEvent(ctx, &deployer_pb.DeploymentEvent{
			DeploymentId: deployment.ID,
			Metadata: &deployer_pb.EventMetadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
			},
			Event: &deployer_pb.DeploymentEventType{
				Type: &deployer_pb.DeploymentEventType_StackStatus_{
					StackStatus: &deployer_pb.DeploymentEventType_StackStatus{
						Lifecycle:  stackStatus.SummaryType,
						FullStatus: string(stackStatus.StatusName),
					},
				},
			},
		})
	}
}

func (d *Deployer) updateDeployment(ctx context.Context, deployment *Deployment, previous []types.Parameter) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	// Scale Down
	log.Info(ctx, "Trigger: Scale Down")
	if err := d.RegisterEvent(ctx, &deployer_pb.DeploymentEvent{
		DeploymentId: deployment.ID,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_StackTrigger_{
				StackTrigger: &deployer_pb.DeploymentEventType_StackTrigger{
					Phase: "scale down",
				},
			},
		},
	}); err != nil {
		return err
	}

	if err := cf.setScale(ctx, deployment.StackName, 0); err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment, "scale down")); err != nil {
		return err
	}

	// Update, Keep Scale 0
	if err := d.RegisterEvent(ctx, &deployer_pb.DeploymentEvent{
		DeploymentId: deployment.ID,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_StackTrigger_{
				StackTrigger: &deployer_pb.DeploymentEventType_StackTrigger{
					Phase: "update",
				},
			},
		},
	}); err != nil {
		return err
	}

	log.Info(ctx, "Update Pre Migrate")
	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		previousParameters: previous,
		name:               deployment.StackName,
		template:           deployment.App,
		scale:              0,
	})
	if err != nil {
		return err
	}

	if err := cf.updateCloudformationStack(ctx, StackArgs{
		Template:   deployment.App,
		Parameters: initialParameters,
		Name:       deployment.StackName,
	}); err != nil {
		return err
	}

	lastState, err := cf.waitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment, "update"))
	if err != nil {
		return err
	}

	// Migrate
	log.Info(ctx, "Data Migrate")
	if err := d.migrateData(ctx, lastState.Outputs, deployment.App, d.RotateSecrets); err != nil {
		return err
	}

	// Scale Up
	log.Info(ctx, "Scale Up")
	if err := cf.setScale(ctx, deployment.StackName, 1); err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment, "scale up")); err != nil {
		return err
	}

	return nil
}
