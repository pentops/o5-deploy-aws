package deployer

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/awsinfra"
	"github.com/pentops/o5-deploy-aws/internal/cf/app"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type TemplateStore interface {
	PutTemplate(ctx context.Context, envName, appName, deploymentID string, template []byte) (*awsdeployer_pb.S3Template, error)
}

type S3TemplateStore struct {
	s3Client         awsinfra.S3API
	region           string
	cfTemplateBucket string
}

func NewS3TemplateStore(ctx context.Context, s3Client awsinfra.S3API, cfTemplateBucket string) (*S3TemplateStore, error) {
	cfTemplateBucket = strings.TrimPrefix(cfTemplateBucket, "s3://")

	regionRes, err := s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(cfTemplateBucket),
	})
	if err != nil {
		return nil, err
	}

	region := string(regionRes.LocationConstraint)
	return &S3TemplateStore{
		s3Client:         s3Client,
		cfTemplateBucket: cfTemplateBucket,
		region:           region,
	}, nil
}

func (s3ts *S3TemplateStore) PutTemplate(ctx context.Context, envName string, appName string, deploymentID string, templateJSON []byte) (*awsdeployer_pb.S3Template, error) {

	templateKey := fmt.Sprintf("%s/%s/%s.json", envName, appName, deploymentID)
	_, err := s3ts.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3ts.cfTemplateBucket),
		Key:    aws.String(templateKey),
		Body:   bytes.NewReader(templateJSON),
	})
	if err != nil {
		return nil, err
	}

	return &awsdeployer_pb.S3Template{
		Bucket: s3ts.cfTemplateBucket,
		Key:    templateKey,
		Region: s3ts.region,
	}, nil
}

type SpecBuilder struct {
	templateStore TemplateStore
}

func NewSpecBuilder(templateStore TemplateStore) (*SpecBuilder, error) {
	return &SpecBuilder{
		templateStore: templateStore,
	}, nil
}

func safeDBName(dbName string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return '_'
	}, dbName)
}

func (dd *SpecBuilder) BuildSpec(ctx context.Context, trigger *deployer_tpb.RequestDeploymentMessage, cluster *environment_pb.Cluster, environment *environment_pb.Environment) (*awsdeployer_pb.DeploymentSpec, error) {
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

	ecsCluster := cluster.GetEcsCluster()
	if ecsCluster == nil {
		return nil, fmt.Errorf("cluster %s is not an ECS cluster", cluster.Name)
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return nil, err
	}
	templateLocation, err := dd.templateStore.PutTemplate(ctx, environment.FullName, app.Name, deploymentID, templateJSON)
	if err != nil {
		return nil, err
	}

	dbSpecs := make([]*awsdeployer_pb.PostgresSpec, len(app.PostgresDatabases))
	for idx, db := range app.PostgresDatabases {
		fullName := safeDBName(fmt.Sprintf("%s_%s_%s", environment.FullName, app.Name, db.DbName))
		dbSpec := &awsdeployer_pb.PostgresSpec{
			DbName:                  fullName,
			DbExtensions:            db.DbExtensions,
			MigrationTaskOutputName: db.MigrationTaskOutputName,
			SecretOutputName:        db.SecretOutputName,
		}
		for _, host := range ecsCluster.RdsHosts {
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

	deployerResolver, err := BuildParameterResolver(ctx, cluster, environment)
	if err != nil {
		return nil, err
	}

	parameters := make([]*awsdeployer_pb.CloudFormationStackParameter, len(app.Parameters))
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

	spec := &awsdeployer_pb.DeploymentSpec{
		AppName:         app.Name,
		Version:         app.Version,
		EnvironmentName: environment.FullName,
		EnvironmentId:   trigger.EnvironmentId,
		Template:        templateLocation,
		Databases:       dbSpecs,
		Parameters:      parameters,
		SnsTopics:       snsTopics,
		Flags:           trigger.Flags,

		EcsCluster: ecsCluster.EcsClusterName,
	}

	return spec, nil
}
