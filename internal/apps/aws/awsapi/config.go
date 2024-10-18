package awsapi

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pentops/log.go/log"
)

type loaderOptions struct {
	assumeRole string
	region     string
}

type Option func(*loaderOptions)

func WithAssumeRole(assumeRole string) Option {
	return func(cfg *loaderOptions) {
		cfg.assumeRole = assumeRole
	}
}

func LoadFromConfig(ctx context.Context, awsConfig aws.Config, opts ...Option) (*DeployerClients, error) {

	cfg := loaderOptions{
		region: awsConfig.Region,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.assumeRole != "" {
		var err error
		awsConfig, err = assumeRole(ctx, awsConfig, cfg.assumeRole)
		if err != nil {
			return nil, err
		}
	}

	stsClient := sts.NewFromConfig(awsConfig)
	callerIdentity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, err
	}

	log.WithFields(ctx, map[string]interface{}{
		"account":     aws.ToString(callerIdentity.Account),
		"arn":         aws.ToString(callerIdentity.Arn),
		"user":        aws.ToString(callerIdentity.UserId),
		"assumedRole": cfg.assumeRole,
	}).Info("Infra Adapter AWS Identity")

	return &DeployerClients{
		CloudFormation: cloudformation.NewFromConfig(awsConfig),
		CloudWatchLogs: cloudwatchlogs.NewFromConfig(awsConfig),
		SNS:            sns.NewFromConfig(awsConfig),
		ELB:            elasticloadbalancingv2.NewFromConfig(awsConfig),
		SecretsManager: secretsmanager.NewFromConfig(awsConfig),
		ECS:            ecs.NewFromConfig(awsConfig),
		S3:             s3.NewFromConfig(awsConfig),
		Region:         cfg.region,
		AccountID:      *callerIdentity.Account,
		RDSAuthProvider: rdsAuthProvider{
			region: cfg.region,
			creds:  awsConfig.Credentials,
		},
	}, nil

}

func assumeRole(ctx context.Context, awsConfig aws.Config, roleARN string) (aws.Config, error) {

	stsClient := sts.NewFromConfig(awsConfig)
	provider := stscreds.NewAssumeRoleProvider(stsClient, roleARN)
	creds := aws.NewCredentialsCache(provider)

	assumeRoleConfig, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return awsConfig, fmt.Errorf("failed to assume role: %w", err)
	}

	return assumeRoleConfig, nil
}
