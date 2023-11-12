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
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
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
	Clients(ctx context.Context) (*DeployerClients, error)
}

type Deployer struct {
	Environment   *environment_pb.Environment
	AWS           *environment_pb.AWS
	RotateSecrets bool
	Clients       ClientBuilder
	storage       DeployerStateStore
}

func NewDeployer(storage DeployerStateStore, environment *environment_pb.Environment, clientSet ClientBuilder) (*Deployer, error) {

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
		Clients:     clientSet,
		storage:     storage,
	}, nil
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

type ClientSet struct {
	clientCache   *DeployerClients
	AWSConfig     aws.Config
	AssumeRoleARN string
}

func (cs *ClientSet) Clients(ctx context.Context) (*DeployerClients, error) {
	if cs.clientCache != nil {
		// TODO: Check Expiry
		return cs.clientCache, nil
	}

	assumer := sts.NewFromConfig(cs.AWSConfig)

	tempCredentials, err := assumer.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         aws.String(cs.AssumeRoleARN),
		DurationSeconds: aws.Int32(900),
		RoleSessionName: aws.String(fmt.Sprintf("o5-deploy-aws-%d", time.Now().Unix())),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to assume role '%s': %w", cs.AssumeRoleARN, err)
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
	ctx = log.WithFields(ctx, map[string]interface{}{
		"appName":     app.Name,
		"environment": d.Environment.FullName,
	})

	clients, err := d.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return err
	}

	templateKey := fmt.Sprintf("%s/%s/%s.json", d.Environment.FullName, app.Name, deploymentID)
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
	}

	if err := d.RegisterEvent(ctx, newEvent(deployment, &deployer_pb.DeploymentEventType_Triggered_{
		Triggered: &deployer_pb.DeploymentEventType_Triggered{
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
		},
	})); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) eventGotLock(ctx context.Context, deployment *deployer_pb.DeploymentState) (*deployer_pb.DeploymentEvent, error) {

	cf := &AWSRunner{
		Clients: d.Clients,
	}
	topicNames := make([]string, len(deployment.Spec.SnsTopics))
	for i, topic := range deployment.Spec.SnsTopics {
		topicNames[i] = topic.Name
	}

	if _, err := cf.UpsertSNSTopics(ctx, &deployer_tpb.UpsertSNSTopicsMessage{
		EnvironmentName: d.Environment.FullName,
		TopicNames:      topicNames,
	}); err != nil {
		return nil, err
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
		// TODO: This is not async
		if _, err := cf.DeleteStack(ctx, &deployer_tpb.DeleteStackMessage{
			StackName: deployment.StackName,
		}); err != nil {
			return nil, err
		}

		return newEvent(deployment, &deployer_pb.DeploymentEventType_StackCreate_{
			StackCreate: &deployer_pb.DeploymentEventType_StackCreate{},
		}), nil
	}

	needsCancel := deployment.Spec.CancelUpdates && remoteStack.StackStatus == types.StackStatusUpdateInProgress

	return newEvent(deployment, &deployer_pb.DeploymentEventType_StackWait_{
		StackWait: &deployer_pb.DeploymentEventType_StackWait{
			CancelUpdates: needsCancel,
		},
	}), nil
}

func (d *Deployer) createNewDeployment(ctx context.Context, deployment *deployer_pb.DeploymentState) (*deployer_tpb.CreateNewStackMessage, error) {

	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		parameters: deployment.Spec.Parameters,
		scale:      0,
	})
	if err != nil {
		return nil, err
	}

	// Create, scale 0
	return &deployer_tpb.CreateNewStackMessage{
		StackName:   deployment.StackName,
		Parameters:  initialParameters,
		TemplateUrl: deployment.Spec.TemplateUrl,
	}, nil
}

func (d *Deployer) buildUpdateStackRequest(ctx context.Context, deployment *deployer_pb.DeploymentState) (*deployer_tpb.UpdateStackMessage, error) {

	cf := &AWSRunner{
		Clients: d.Clients,
	}

	stack, err := cf.getOneStack(ctx, deployment.StackName)
	if err != nil {
		return nil, err
	}

	stackStatus, err := summarizeStackStatus(stack)
	if err != nil {
		return nil, err
	}

	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		previousParameters: stackStatus.Parameters,
		parameters:         deployment.Spec.Parameters,
		scale:              0,
	})
	if err != nil {
		return nil, err
	}

	return &deployer_tpb.UpdateStackMessage{
		StackName:   deployment.StackName,
		Parameters:  initialParameters,
		TemplateUrl: deployment.Spec.TemplateUrl,
	}, nil

}

func (d *Deployer) buildMigrationRequest(ctx context.Context, deployment *deployer_pb.DeploymentState, migration *deployer_pb.DatabaseMigrationState) (*deployer_tpb.RunDatabaseMigrationMessage, error) {

	cf := &AWSRunner{
		Clients: d.Clients,
	}

	stack, err := cf.getOneStack(ctx, deployment.StackName)
	if err != nil {
		return nil, err
	}

	// Migrate

	var db *deployer_pb.PostgresDatabase
	for _, search := range deployment.Spec.Databases {
		if search.Database.Name == migration.DbName {
			db = search
			break
		}
	}

	if db == nil {
		return nil, fmt.Errorf("no database found with name %s", migration.DbName)
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"database":    db.Database.Name,
		"serverGroup": db.Database.GetPostgres().ServerGroup,
	})
	log.Debug(ctx, "Upsert Database")
	var migrationTaskARN string
	var secretARN string
	for _, output := range stack.Outputs {
		if *db.MigrationTaskOutputName == *output.OutputKey {
			migrationTaskARN = *output.OutputValue
		}
		if *db.SecretOutputName == *output.OutputKey {
			secretARN = *output.OutputValue
		}
	}

	var secretName string
	for _, host := range d.AWS.RdsHosts {
		if host.ServerGroup == db.Database.GetPostgres().ServerGroup {
			secretName = host.SecretName
			break
		}
	}
	if secretName == "" {
		return nil, fmt.Errorf("no host found for server group %q", db.Database.GetPostgres().ServerGroup)
	}

	return &deployer_tpb.RunDatabaseMigrationMessage{
		MigrationId:       migration.MigrationId,
		DeploymentId:      deployment.DeploymentId,
		MigrationTaskArn:  migrationTaskARN,
		SecretArn:         secretARN,
		Database:          db,
		RotateCredentials: migration.RotateCredentials,
		EcsClusterName:    d.AWS.EcsClusterName,
		RootSecretName:    secretName,
	}, nil

}
