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
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
)

type TemplateStore interface {
	PutTemplate(ctx context.Context, envName, appName, deploymentID string, template []byte) (*awsdeployer_pb.S3Template, error)
}

type S3API interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error)
}

type S3TemplateStore struct {
	s3Client         S3API
	region           string
	cfTemplateBucket string
}

func NewS3TemplateStore(ctx context.Context, s3Client S3API, cfTemplateBucket string) (*S3TemplateStore, error) {
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

func (dd *SpecBuilder) BuildSpec(ctx context.Context, trigger *awsdeployer_tpb.RequestDeploymentMessage, cluster *environment_pb.Cluster, environment *environment_pb.Environment) (*awsdeployer_pb.DeploymentSpec, error) {
	ctx = log.WithFields(ctx, map[string]interface{}{
		"appName":     trigger.Application.Name,
		"environment": environment.FullName,
	})

	awsCluster := cluster.GetAws()
	if awsCluster == nil {
		return nil, fmt.Errorf("cluster %s is not an ECS cluster", cluster.Name)
	}

	rdsHosts := appbuilder.RDSHostMap{}
	for _, host := range awsCluster.RdsHosts {
		rdsHosts[host.ServerGroupName] = &appbuilder.RDSHost{
			AuthType: host.Auth.Get().TypeKey(),
		}
	}

	app, err := appbuilder.BuildApplication(appbuilder.AppInput{
		Application: trigger.Application,
		VersionTag:  trigger.Version,
		RDSHosts:    rdsHosts,
	})
	if err != nil {
		return nil, err
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.Template.JSON()
	if err != nil {
		return nil, err
	}
	templateLocation, err := dd.templateStore.PutTemplate(ctx, environment.FullName, app.Name, deploymentID, templateJSON)
	if err != nil {
		return nil, err
	}

	dbDeps, err := buildDBSpecs(ctx, app.Databases, awsCluster)
	if err != nil {
		return nil, err
	}

	deployerResolver, err := buildParameterResolver(ctx, parameterInput{
		cluster:     cluster,
		environment: environment,
		auroraHosts: auroraHosts,
	})
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

	if trigger.Flags == nil {
		trigger.Flags = &awsdeployer_pb.DeploymentFlags{}
	}

	if trigger.Application.DeploymentConfig != nil {
		if trigger.Application.DeploymentConfig.QuickMode {
			trigger.Flags.QuickMode = true
		}
	}

	spec := &awsdeployer_pb.DeploymentSpec{
		AppName:         app.Name,
		Version:         app.Version,
		EnvironmentName: environment.FullName,
		EnvironmentId:   trigger.EnvironmentId,
		Template:        templateLocation,
		Databases:       dbSpecs,
		Parameters:      parameters,
		Flags:           trigger.Flags,
		CfStackName:     fmt.Sprintf("%s-%s", environment.FullName, app.Name),
		SnsTopics:       []string{},

		EcsCluster: awsCluster.EcsCluster.ClusterName,
	}

	return spec, nil
}

func coalesce[A any](backup A, from ...*A) A {
	for _, val := range from {
		if val != nil {
			return *val
		}
	}
	return backup
}
