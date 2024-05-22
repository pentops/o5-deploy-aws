package awsinfra

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type CloudFormationAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
	UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
	CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error)
	ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error)
	DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error)
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
	CancelUpdateStack(ctx context.Context, params *cloudformation.CancelUpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CancelUpdateStackOutput, error)
}

type ELBV2API interface {
	DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
}

type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
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
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
}

type DeployerClients struct {
	CloudFormation CloudFormationAPI
	SNS            SNSAPI
	ELB            ELBV2API
	SecretsManager SecretsManagerAPI
	ECS            ECSAPI
	S3             S3API
	Region         string
	AccountID      string
}

func NewDeployerClientsFromConfig(ctx context.Context, config aws.Config) (*DeployerClients, error) {

	region := config.Region

	stsClient := sts.NewFromConfig(config)
	callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	return &DeployerClients{
		CloudFormation: cloudformation.NewFromConfig(config),
		SNS:            sns.NewFromConfig(config),
		ELB:            elbv2.NewFromConfig(config),
		SecretsManager: secretsmanager.NewFromConfig(config),
		ECS:            ecs.NewFromConfig(config),
		S3:             s3.NewFromConfig(config),
		Region:         region,
		AccountID:      *callerIdentity.Account,
	}, nil

}
