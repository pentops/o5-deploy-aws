package deployer

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type TemplateStore interface {
	PutTemplate(ctx context.Context, envName, appName, deploymentID string, template []byte) (string, error)
}

type S3TemplateStore struct {
	s3Client         awsinfra.S3API
	cfTemplateBucket string
}

func (s3ts *S3TemplateStore) PutTemplate(ctx context.Context, envName string, appName string, deploymentID string, templateJSON []byte) (string, error) {

	templateKey := fmt.Sprintf("%s/%s/%s.json", envName, appName, deploymentID)
	_, err := s3ts.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3ts.cfTemplateBucket),
		Key:    aws.String(templateKey),
		Body:   bytes.NewReader(templateJSON),
	})
	if err != nil {
		return "", err
	}

	templateURL := fmt.Sprintf("https://s3.us-east-1.amazonaws.com/%s/%s", s3ts.cfTemplateBucket, templateKey)
	return templateURL, nil
}

func NewS3TemplateStore(s3Client awsinfra.S3API, cfTemplateBucket string) *S3TemplateStore {
	cfTemplateBucket = strings.TrimPrefix(cfTemplateBucket, "s3://")
	return &S3TemplateStore{
		s3Client:         s3Client,
		cfTemplateBucket: cfTemplateBucket,
	}
}

type SpecBuilder struct {
	RotateSecrets bool
	CancelUpdates bool
	QuickMode     bool

	templateStore TemplateStore
}

func NewSpecBuilder(templateStore TemplateStore) (*SpecBuilder, error) {
	return &SpecBuilder{
		templateStore: templateStore,
	}, nil
}

func (dd *SpecBuilder) BuildSpec(ctx context.Context, trigger *deployer_tpb.RequestDeploymentMessage, environment *environment_pb.Environment) (*deployer_pb.DeploymentSpec, error) {

	app, err := app.BuildApplication(trigger.Application, trigger.Version)
	if err != nil {
		return nil, err
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"appName":     app.Name,
		"environment": environment.FullName,
	})

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, fmt.Errorf("environment %s is not an AWS environment", environment.FullName)
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return nil, err
	}
	templateURL, err := dd.templateStore.PutTemplate(ctx, environment.FullName, app.Name, deploymentID, templateJSON)

	postgresDatabases := app.PostgresDatabases()

	for _, db := range postgresDatabases {
		pgdb := db.Database.GetPostgres()
		for _, host := range awsEnv.RdsHosts {
			if host.ServerGroup == pgdb.ServerGroup {
				db.RdsHost = host
				break
			}
		}
		if db.RdsHost == nil {
			return nil, fmt.Errorf("no RDS host found for database %s", db.Database.Name)
		}

	}

	deployerResolver, err := BuildParameterResolver(ctx, environment)
	if err != nil {
		return nil, err
	}

	appParameters := app.Parameters()
	parameters := make([]*deployer_pb.CloudFormationStackParameter, 0, len(appParameters))
	for _, param := range appParameters {
		parameter, err := deployerResolver.ResolveParameter(param)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s': %w", param.Name, err)
		}
		parameters = append(parameters, parameter)
	}

	spec := &deployer_pb.DeploymentSpec{
		AppName:         app.Name,
		Version:         app.Version,
		EnvironmentName: environment.FullName,
		EnvironmentId:   trigger.EnvironmentId,
		TemplateUrl:     templateURL,
		Databases:       postgresDatabases,
		Parameters:      parameters,
		SnsTopics:       app.SNSTopics,

		CancelUpdates:     dd.CancelUpdates,
		RotateCredentials: dd.RotateSecrets,
		QuickMode:         dd.QuickMode || app.QuickMode,

		EcsCluster: awsEnv.EcsClusterName,
	}

	return spec, nil
}
