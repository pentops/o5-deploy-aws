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
	if err != nil {
		return nil, err
	}

	dbSpecs := make([]*deployer_pb.PostgresSpec, len(app.PostgresDatabases))
	for idx, db := range app.PostgresDatabases {
		dbSpec := &deployer_pb.PostgresSpec{
			DbName:                  db.DbName,
			DbExtensions:            db.DbExtensions,
			MigrationTaskOutputName: db.MigrationTaskOutputName,
			SecretOutputName:        db.SecretOutputName,
		}
		for _, host := range awsEnv.RdsHosts {
			if host.ServerGroup == db.ServerGroup {
				dbSpec.RootSecretName = host.SecretName
				break
			}
		}
		if dbSpec.RootSecretName == "" {
			return nil, fmt.Errorf("no RDS host found for database %s", db.DbName)
		}

		dbSpecs[idx] = dbSpec
	}

	deployerResolver, err := BuildParameterResolver(ctx, environment)
	if err != nil {
		return nil, err
	}

	parameters := make([]*deployer_pb.CloudFormationStackParameter, len(app.Parameters))
	for idx, param := range app.Parameters {
		parameter, err := deployerResolver.ResolveParameter(param)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s': %w", param.Name, err)
		}
		parameters[idx] = parameter
	}

	snsTopics := make([]string, len(app.SnsTopics))
	for idx, topic := range app.SnsTopics {
		snsTopics[idx] = fmt.Sprintf("%s-%s", environment.FullName, topic)
	}

	if trigger.Flags == nil {
		trigger.Flags = &deployer_pb.DeploymentFlags{}
	}
	if app.QuickMode {
		trigger.Flags.QuickMode = true
	}
	spec := &deployer_pb.DeploymentSpec{
		AppName:         app.Name,
		Version:         app.Version,
		EnvironmentName: environment.FullName,
		EnvironmentId:   trigger.EnvironmentId,
		TemplateUrl:     templateURL,
		Databases:       dbSpecs,
		Parameters:      parameters,
		SnsTopics:       snsTopics,
		Flags:           trigger.Flags,

		EcsCluster: awsEnv.EcsClusterName,
	}

	return spec, nil
}
