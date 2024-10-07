package deployer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
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

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, fmt.Errorf("environment %s is not an AWS environment", environment.FullName)
	}

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

	dbSpecs := make([]*awsdeployer_pb.PostgresSpec, len(app.Databases))
	for idx, db := range app.Databases {
		fullName := safeDBName(fmt.Sprintf("%s_%s_%s", environment.FullName, app.Name, db.DbName))

		var host *environment_pb.RDSHost
		for _, search := range awsCluster.RdsHosts {
			if search.ServerGroupName == db.ServerGroup {
				host = search
				break
			}
		}
		if host == nil {
			return nil, fmt.Errorf("no RDS host found for database %s", db.DbName)
		}

		appConn := &awsdeployer_pb.PostgresConnectionType{}
		adminConn := &awsinfra_pb.RDSHostType{}

		switch hostType := host.Auth.Get().(type) {
		case *environment_pb.RDSAuthType_SecretsManager:
			secret := db.GetSecretOutputName()
			if secret == "" {
				panic(fmt.Sprintf("no secret name found for database %s", db.DbName))
			}

			appConn.Set(&awsdeployer_pb.PostgresConnectionType_SecretsManager{
				AppSecretOutputName: secret,
			})

			adminConn.Set(&awsinfra_pb.RDSHostType_SecretsManager{
				SecretName: hostType.SecretName,
			})

		case *environment_pb.RDSAuthType_IAM:
			appConn.Set(&awsdeployer_pb.PostgresConnectionType_Aurora{
				Conn: &awsinfra_pb.AuroraConnection{
					Endpoint: host.Endpoint,
					Port:     host.Port,
					DbUser:   fullName,
					DbName:   fullName,
				},
			})

			adminConn.Set(&awsinfra_pb.RDSHostType_Aurora{
				Conn: &awsinfra_pb.AuroraConnection{
					Endpoint: host.Endpoint,
					Port:     host.Port,
					DbUser:   hostType.DbUser,
					DbName:   coalesce(hostType.DbUser, hostType.DbName),
				},
			})

			paramName := db.GetParameterName()
			if paramName == "" {
				panic(fmt.Sprintf("no parameter name found for database %s", db.DbName))
			}

			var param *awsdeployer_pb.Parameter
			for _, p := range app.Parameters {
				if p.Name == paramName {
					param = p
					break
				}
			}
			if param == nil {
				panic(fmt.Sprintf("no parameter found for database %s", db.DbName))
			}
			jsonValue, err := json.Marshal(&DBSecret{
				Username: fullName,
				Password: paramName,
				Hostname: host.Endpoint,
				DBName:   fullName,
				URL:      fmt.Sprintf("postgres://%s:%s@%s:%d/%s", fullName, paramName, host.Endpoint, host.Port, fullName),
			})
			if err != nil {
				return nil, err
			}
			param.Source = &awsdeployer_pb.ParameterSourceType{
				Type: &awsdeployer_pb.ParameterSourceType_Static_{
					Static: &awsdeployer_pb.ParameterSourceType_Static{
						Value: string(jsonValue),
					},
				},
			}

		}

		dbSpec := &awsdeployer_pb.PostgresSpec{
			DbName:                  fullName,
			DbExtensions:            db.DbExtensions,
			MigrationTaskOutputName: db.MigrationTaskOutputName,
			AppConnection:           appConn,
			AdminConnection:         adminConn,
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

type DBSecret struct {
	Username string `json:"dbuser"`
	Password string `json:"dbpass"`
	Hostname string `json:"dbhost"`
	DBName   string `json:"dbname"`
	URL      string `json:"dburl"`
}
