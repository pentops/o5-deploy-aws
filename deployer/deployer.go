package deployer

import (
	"bytes"
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
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.daemonl.com/sqrlx"
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

type Deployer struct {
	Environment   *environment_pb.Environment
	AWS           *environment_pb.AWS
	RotateSecrets bool
	clientCache   *DeployerClients
	awsConfig     aws.Config
	eg            errgroup.Group

	db *sqrlx.Wrapper

	deployments map[string]*deployer_pb.DeploymentState
}

func NewDeployer(conn sqrlx.Connection, environment *environment_pb.Environment, awsConfig aws.Config) (*Deployer, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

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
		eg:          errgroup.Group{},
		db:          db,
	}, nil
}

func (d *Deployer) getDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error) {
	if deployment, ok := d.deployments[id]; ok {
		return deployment, nil
	}
	return nil, fmt.Errorf("no deployment found with id %s", id)
}

func newEvent(d *deployer_pb.DeploymentState, event deployer_pb.IsDeploymentEventType_Type) *deployer_pb.DeploymentEvent {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: d.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: event,
		},
	}
}

func (d *Deployer) AsyncEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) {
	d.eg.Go(func() error {
		return d.RegisterEvent(ctx, event)
	})
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

func (d *Deployer) Deploy(ctx context.Context, app *app.BuiltApplication, cancelUpdates bool) error {
	stackName := fmt.Sprintf("%s-%s", d.Environment.FullName, app.Name)
	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackName":   stackName,
		"environment": d.Environment.FullName,
	})

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return err
	}

	templateKey := fmt.Sprintf("%s/%s/%s.json", d.Environment.FullName, stackName, deploymentID)
	_, err = clients.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(d.AWS.ScratchBucket),
		Key:    aws.String(templateKey),
		Body:   bytes.NewReader(templateJSON),
	})

	if err != nil {
		return err
	}

	templateURL := fmt.Sprintf("https://s3.us-east-1.amazonaws.com/%s/%s", d.AWS.ScratchBucket, templateKey)

	deployment := &deployer_pb.DeploymentState{
		DeploymentId: deploymentID,
		Status:       deployer_pb.DeploymentStatus_LOCKED,
		StackName:    stackName,
		Spec: &deployer_pb.DeploymentSpec{
			AppName:         app.Name,
			Version:         app.Version,
			EnvironmentName: d.Environment.FullName,
			TemplateUrl:     templateURL,
			Databases:       app.PostgresDatabases(),
			Parameters:      app.Parameters(),
			CancelUpdates:   cancelUpdates,
			SnsTopics:       app.SNSTopics,
		},
	}

	d.deployments[deploymentID] = deployment

	if err := d.RegisterEvent(ctx, newEvent(deployment, &deployer_pb.DeploymentEventType_Triggered_{
		Triggered: &deployer_pb.DeploymentEventType_Triggered{
			Spec: &deployer_pb.DeploymentSpec{
				AppName:         app.Name,
				Version:         app.Version,
				EnvironmentName: d.Environment.FullName,
			},
		},
	})); err != nil {
		return err
	}

	// TODO: Wait for lock.

	d.eg.Go(func() error {
		return d.RegisterEvent(ctx, newEvent(deployment, &deployer_pb.DeploymentEventType_GotLock_{
			GotLock: &deployer_pb.DeploymentEventType_GotLock{},
		}))
	})

	return d.eg.Wait()
}

func (d *Deployer) eventGotLock(ctx context.Context, deployment *deployer_pb.DeploymentState) (*deployer_pb.DeploymentEvent, error) {

	clients, err := d.Clients(ctx)
	if err != nil {
		return nil, err
	}

	if err := d.upsertSNSTopics(ctx, deployment.Spec.SnsTopics); err != nil {
		return nil, err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	remoteStack, err := cf.getOneStack(ctx, deployment.StackName)
	if err != nil {
		return nil, err
	}

	if remoteStack == nil {
		return newEvent(deployment, &deployer_pb.DeploymentEventType_StackCreate_{
			StackCreate: &deployer_pb.DeploymentEventType_StackCreate{},
		}), nil
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		if err := cf.deleteStack(ctx, deployment.StackName); err != nil {
			return nil, err
		}
		return newEvent(deployment, &deployer_pb.DeploymentEventType_StackCreate_{
			StackCreate: &deployer_pb.DeploymentEventType_StackCreate{},
		}), nil
	}

	// TODO: This is not an event
	if deployment.Spec.CancelUpdates && remoteStack.StackStatus == types.StackStatusUpdateInProgress {
		if err := cf.cancelUpdate(ctx, deployment.StackName); err != nil {
			return nil, err
		}
	}

	return newEvent(deployment, &deployer_pb.DeploymentEventType_StackWait_{
		StackWait: &deployer_pb.DeploymentEventType_StackWait{},
	}), nil
}

func (d *Deployer) runStackWait(ctx context.Context, deployment *deployer_pb.DeploymentState) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	_, err = cf.waitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment))
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployer) upsertSNSTopics(ctx context.Context, topics []*deployer_pb.SNSTopic) error {
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

func (d *Deployer) waitStatusCallback(deployment *deployer_pb.DeploymentState) waiterCallback {
	return func(ctx context.Context, stackStatus StackStatus) error {

		d.AsyncEvent(ctx, &deployer_pb.DeploymentEvent{
			DeploymentId: deployment.DeploymentId,
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
		return nil
	}
}

func (d *Deployer) createNewDeployment(ctx context.Context, deployment *deployer_pb.DeploymentState) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		parameters: deployment.Spec.Parameters,
		scale:      0,
	})
	if err != nil {
		return err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	// Create, scale 0
	log.Info(ctx, "Create with scale 0")
	if err := cf.createCloudformationStack(ctx, deployment, initialParameters); err != nil {
		return err
	}

	_, err = cf.waitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment))
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployer) eventStackScale(ctx context.Context, deployment *deployer_pb.DeploymentState, scale int) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}
	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	if err := cf.setScale(ctx, deployment.StackName, scale); err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment)); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) migrateInfra(ctx context.Context, deployment *deployer_pb.DeploymentState) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}
	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	stack, err := cf.getOneStack(ctx, deployment.StackName)
	if err != nil {
		return err
	}

	stackStatus, err := summarizeStackStatus(stack)
	if err != nil {
		return err
	}

	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		previousParameters: stackStatus.Parameters,
		parameters:         deployment.Spec.Parameters,
		scale:              0,
	})
	if err != nil {
		return err
	}

	if err := cf.updateCloudformationStack(ctx, deployment, initialParameters); err != nil {
		return err
	}

	_, err = cf.waitForSuccess(ctx, deployment.StackName, d.waitStatusCallback(deployment))
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployer) runMigration(ctx context.Context, deployment *deployer_pb.DeploymentState, migration *deployer_pb.DatabaseMigrationState) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}
	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	stack, err := cf.getOneStack(ctx, deployment.StackName)
	if err != nil {
		return err
	}

	d.AsyncEvent(ctx, newEvent(deployment, &deployer_pb.DeploymentEventType_DbMigrateStatus{
		DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
			DbName:      migration.DbName,
			MigrationId: migration.MigrationId,
			Status:      deployer_pb.DatabaseMigrationStatus_PENDING,
		},
	}))

	// Migrate
	if err := d.migrateData(ctx, stack.Outputs, deployment, migration); err != nil {
		d.AsyncEvent(ctx, newEvent(deployment, &deployer_pb.DeploymentEventType_DbMigrateStatus{
			DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
				DbName:      migration.DbName,
				MigrationId: migration.MigrationId,
				Status:      deployer_pb.DatabaseMigrationStatus_FAILED,
				Error:       proto.String(err.Error()),
			},
		}))
		return err
	}

	d.AsyncEvent(ctx, newEvent(deployment, &deployer_pb.DeploymentEventType_DbMigrateStatus{
		DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
			DbName:      migration.DbName,
			MigrationId: migration.MigrationId,
			Status:      deployer_pb.DatabaseMigrationStatus_COMPLETED,
		},
	}))

	return nil
}
