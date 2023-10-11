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
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	protovalidate "github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
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

type DeployerClients struct {
	CloudFormation CloudFormationAPI
	SNS            SNSAPI
	ELB            ELBV2API
	SecretsManager SecretsManagerAPI
	ECS            ECSAPI
}

type RoleAssumer struct {
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
	}, nil
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

	return &DeployerClients{
		CloudFormation: cloudformation.NewFromConfig(assumeRoleConfig),
		SNS:            sns.NewFromConfig(assumeRoleConfig),
		ELB:            elasticloadbalancingv2.NewFromConfig(assumeRoleConfig),
		SecretsManager: secretsmanager.NewFromConfig(assumeRoleConfig),
		ECS:            ecs.NewFromConfig(assumeRoleConfig),
	}, nil

}

func (d *Deployer) Deploy(ctx context.Context, app *app.BuiltApplication, cancelUpdates bool) error {
	stackName := fmt.Sprintf("%s-%s", d.Environment.FullName, app.Name)
	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackName":   stackName,
		"environment": d.Environment.FullName,
	})

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
		return d.createNewDeployment(ctx, stackName, app)
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

	return d.updateDeployment(ctx, stackName, app, remoteStackStable.Parameters)
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

func (d *Deployer) createNewDeployment(ctx context.Context, stackName string, app *app.BuiltApplication) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		name:     stackName,
		template: app,
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
		Template:   app,
		Name:       stackName,
	}); err != nil {
		return err
	}

	lastState, err := cf.waitForSuccess(ctx, stackName, "Stack Create")
	if err != nil {
		return err
	}

	// Migrate
	log.Info(ctx, "Migrate Database")
	if err := d.migrateData(ctx, lastState.Outputs, app, true); err != nil {
		return err
	}

	// Scale Up
	log.Info(ctx, "Scale Up")
	if err := cf.setScale(ctx, stackName, 1); err != nil {
		return err
	}
	if err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, stackName, "Scale Up"); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) updateDeployment(ctx context.Context, stackName string, app *app.BuiltApplication, previous []types.Parameter) error {

	clients, err := d.Clients(ctx)
	if err != nil {
		return err
	}

	cf := &CFWrapper{
		client: clients.CloudFormation,
	}

	// Scale Down
	log.Info(ctx, "Scale Down")
	if err := cf.setScale(ctx, stackName, 0); err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, stackName, "Scale Down"); err != nil {
		return err
	}

	// Update, Keep Scale 0
	log.Info(ctx, "Update Pre Migrate")
	initialParameters, err := d.applyInitialParameters(ctx, stackParameters{
		previousParameters: previous,
		name:               stackName,
		template:           app,
		scale:              0,
	})
	if err != nil {
		return err
	}

	if err := cf.updateCloudformationStack(ctx, StackArgs{
		Template:   app,
		Parameters: initialParameters,
		Name:       stackName,
	}); err != nil {
		return err
	}

	lastState, err := cf.waitForSuccess(ctx, stackName, "Update")
	if err != nil {
		return err
	}

	// Migrate
	log.Info(ctx, "Data Migrate")
	if err := d.migrateData(ctx, lastState.Outputs, app, d.RotateSecrets); err != nil {
		return err
	}

	// Scale Up
	log.Info(ctx, "Scale Up")
	if err := cf.setScale(ctx, stackName, 1); err != nil {
		return err
	}

	if err := cf.WaitForSuccess(ctx, stackName, "Scale Up"); err != nil {
		return err
	}

	return nil
}
