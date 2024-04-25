package awsinfra

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
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

type ClientSet struct {
	clientCache   *DeployerClients
	AWSConfig     aws.Config
	AssumeRoleARN string
}

func NewClientSet(awsConfig aws.Config, assumeRoleARN string) *ClientSet {
	return &ClientSet{
		AWSConfig:     awsConfig,
		AssumeRoleARN: assumeRoleARN,
	}
}

func (cs *ClientSet) Clients(ctx context.Context) (*DeployerClients, error) {
	if cs.clientCache != nil {
		// TODO: Check Expiry
		return cs.clientCache, nil
	}

	assumeRoleConfig := cs.AWSConfig
	if cs.AssumeRoleARN != "" {
		assumer := sts.NewFromConfig(cs.AWSConfig)

		tempCredentials, err := assumer.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         aws.String(cs.AssumeRoleARN),
			DurationSeconds: aws.Int32(900),
			RoleSessionName: aws.String(fmt.Sprintf("o5-deploy-aws-%d", time.Now().Unix())),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to assume role '%s': %w", cs.AssumeRoleARN, err)
		}

		assumeRoleConfig, err = config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				*tempCredentials.Credentials.AccessKeyId,
				*tempCredentials.Credentials.SecretAccessKey,
				*tempCredentials.Credentials.SessionToken),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
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
