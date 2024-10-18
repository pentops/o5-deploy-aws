package awsapi

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
)

type RDSAuthProvider interface {
	BuildAuthToken(ctx context.Context, dbEndpoint, dbUser string) (string, error)
}

type rdsAuthProvider struct {
	region string
	creds  aws.CredentialsProvider
}

func (rds rdsAuthProvider) BuildAuthToken(ctx context.Context, dbEndpoint, dbUser string) (string, error) {
	authenticationToken, err := auth.BuildAuthToken(
		ctx, dbEndpoint, rds.region, dbUser, rds.creds)
	if err != nil {
		return "", fmt.Errorf("failed to create authentication token: %w", err)
	}
	return authenticationToken, nil
}

func NewRDSAuthProviderFromConfig(config aws.Config) RDSAuthProvider {
	return rdsAuthProvider{
		region: config.Region,
		creds:  config.Credentials,
	}
}
