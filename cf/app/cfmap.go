package app

import (

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"

	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

const (
	ECSClusterParameter           = "ECSCluster"
	ECSRepoParameter              = "ECSRepo"
	ECSTaskExecutionRoleParameter = "ECSTaskExecutionRole"
	VersionTagParameter           = "VersionTag"
	ListenerARNParameter          = "ListenerARN"
	HostHeaderParameter           = "HostHeader"
	EnvNameParameter              = "EnvName"
	ClusterNameParameter          = "ClusterName"
	VPCParameter                  = "VPCID"
	MetaDeployAssumeRoleParameter = "MetaDeployAssumeRoleArns"
	JWKSParameter                 = "JWKS"
	AWSRegionParameter            = "AWS::Region"
	CORSOriginParameter           = "CORSOrigin"
	SNSPrefixParameter            = "SNSPrefix"
	S3BucketNamespaceParameter    = "S3BucketNamespace"
	O5SidecarImageParameter       = "O5SidecarImage"
	SESConditionsParameter        = "SESConditions"
	SourceTagParameter            = "SourceTag"
	EventBusARNParameter          = "EventBusARN"

	AWSAccountIDParameter = "AWS::AccountId"

	O5SidecarContainerName = "o5_runtime"
	O5SidecarInternalPort  = 8081

	DeadLetterTargetName = "dead-letter"
	O5MonitorTargetName  = "o5-monitor"
)

type globalData struct {
	appName string

	databases map[string]DatabaseReference
	secrets   map[string]*cf.Resource[*secretsmanager.Secret]
	buckets   map[string]*bucketInfo
}

type bucketInfo struct {
	name  *string
	arn   string
	read  bool
	write bool
}

// DatabaseReference is used to look up parameters ECS Task Definitions
type DatabaseReference struct {
	Definition     *application_pb.Database
	SecretResource *cf.Resource[*secretsmanager.Secret]
}

func (dbDef DatabaseReference) SecretValueFrom() string {
	jsonKey := "dburl"
	versionStage := ""
	versionID := ""
	return cloudformation.Join(":", []string{
		dbDef.SecretResource.Ref(),
		jsonKey,
		versionStage,
		versionID,
	})
}

func sourceTags(extra ...tags.Tag) []tags.Tag {
	return append(extra, tags.Tag{
		Key:   "o5-source",
		Value: cloudformation.Ref(SourceTagParameter),
	})
}

func BuildApplication(app *application_pb.Application, versionTag string) (*BuiltApplication, error) {

	stackTemplate := NewApplication(app.Name, versionTag)

	if app.DeploymentConfig != nil {
		if app.DeploymentConfig.QuickMode {
			stackTemplate.quickMode = true
		}
	}

	for _, key := range []string{
		ECSClusterParameter,
		ECSRepoParameter,
		ECSTaskExecutionRoleParameter,
		ListenerARNParameter,
		HostHeaderParameter,
		EnvNameParameter,
		ClusterNameParameter,
		VPCParameter,
		VersionTagParameter,
		MetaDeployAssumeRoleParameter,
		JWKSParameter,
		CORSOriginParameter,
		SNSPrefixParameter,
		S3BucketNamespaceParameter,
		O5SidecarImageParameter,
		SESConditionsParameter,
		EventBusARNParameter,
		SourceTagParameter,
	} {
		parameter := &awsdeployer_pb.Parameter{
			Name: key,
			Type: "String",
			Source: &awsdeployer_pb.ParameterSourceType{
				Type: &awsdeployer_pb.ParameterSourceType_WellKnown_{
					WellKnown: &awsdeployer_pb.ParameterSourceType_WellKnown{},
				},
			},
		}
		switch key {
		case VersionTagParameter:
			parameter.Source.Type = &awsdeployer_pb.ParameterSourceType_Static_{
				Static: &awsdeployer_pb.ParameterSourceType_Static{
					Value: versionTag,
				},
			}
		case VPCParameter:
			parameter.Type = "AWS::EC2::VPC::Id"
		case SourceTagParameter:
			parameter.Source.Type = &awsdeployer_pb.ParameterSourceType_Static_{
				Static: &awsdeployer_pb.ParameterSourceType_Static{
					Value: fmt.Sprintf("o5/%s/%s", app.Name, versionTag),
				},
			}
		}
		stackTemplate.AddParameter(parameter)
	}

	global, err := mapResources(app, stackTemplate)
	if err != nil {
		return nil, err
	}

	runtimes := map[string]*RuntimeService{}

	for _, database := range app.Databases {
		switch dbType := database.Engine.(type) {
		case *application_pb.Database_Postgres_:

			secret := cf.NewResource(cf.CleanParameterName("Database", database.Name), &secretsmanager.Secret{
				AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
				Name: cloudformation.JoinPtr("/", []string{
					"", // Leading /
					cloudformation.Ref(EnvNameParameter),
					app.Name,
					"postgres",
					database.Name,
				}),
				Description: cf.Stringf("Secret for Postgres database %s in app %s", database.Name, app.Name),
			})

			stackTemplate.AddResource(secret)

			ref := DatabaseReference{
				SecretResource: secret,
				Definition:     database,
			}
			global.databases[database.Name] = ref

			secretName := fmt.Sprintf("DatabaseSecret%s", cf.CleanParameterName(database.Name))
			def := &awsdeployer_pb.PostgresDatabaseResource{
				DbName:           database.Name,
				ServerGroup:      dbType.Postgres.ServerGroup,
				SecretOutputName: secretName,
				DbExtensions:     dbType.Postgres.DbExtensions,
			}

			if dbType.Postgres.DbName != "" {
				def.DbName = dbType.Postgres.DbName
			}

			stackTemplate.AddOutput(&cf.Output{
				Name:  secretName,
				Value: secret.Ref(),
			})

			if dbType.Postgres.MigrateContainer != nil {
				if dbType.Postgres.MigrateContainer.Name == "" {
					dbType.Postgres.MigrateContainer.Name = "migrate"
				}

				// TODO: This takes the global var, which is added to by other
				// databases, so this whole step should be deferred until all
				// other databases (and likely other resources) are created.
				// Not likely to be a problem any time soon so long as THIS
				// database is added early which it is.
				migrationContainer, err := buildContainer(*global, dbType.Postgres.MigrateContainer)
				if err != nil {
					return nil, fmt.Errorf("building migration container for %s: %w", database.Name, err)
				}
				addLogs(migrationContainer.Container, fmt.Sprintf("%s/migrate", global.appName))
				name := fmt.Sprintf("MigrationTaskDefinition%s", cf.CleanParameterName(database.Name))

				migrationTaskDefinition := cf.NewResource(name, &ecs.TaskDefinition{
					ContainerDefinitions: []ecs.TaskDefinition_ContainerDefinition{
						*migrationContainer.Container,
					},
					Family:                  cf.String(fmt.Sprintf("%s_migrate_%s", global.appName, database.Name)),
					ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
					RequiresCompatibilities: []string{"EC2"},
				})
				stackTemplate.AddResource(migrationTaskDefinition)
				def.MigrationTaskOutputName = cf.String(name)
				stackTemplate.AddOutput(&cf.Output{
					Name:  name,
					Value: migrationTaskDefinition.Ref(),
				})
			}

			stackTemplate.postgresDatabases = append(stackTemplate.postgresDatabases, def)

		default:
			return nil, fmt.Errorf("unknown database type %T", dbType)
		}
	}

	{ // SNS Topics
		needsDeadLetter := false
		if len(app.Targets) > 0 {
			needsDeadLetter = true
		} else {
			for _, runtime := range app.Runtimes {
				if len(runtime.Subscriptions) > 0 {
					needsDeadLetter = true
				}
			}
		}

		app.Targets = append(app.Targets, &application_pb.Target{
			Name: O5MonitorTargetName,
		})
		if needsDeadLetter {
			app.Targets = append(app.Targets, &application_pb.Target{
				Name: DeadLetterTargetName,
			})
		}

	}

	listener := NewListenerRuleSet()

	for _, runtime := range app.Runtimes {
		runtimeStack, err := NewRuntimeService(*global, runtime)
		if err != nil {
			return nil, err
		}
		runtimes[runtime.Name] = runtimeStack

		if err := runtimeStack.AddRoutes(listener); err != nil {
			return nil, fmt.Errorf("adding routes to %s: %w", runtime.Name, err)
		}

		for _, target := range app.Targets {
			runtimeStack.Policy.AddEventBridgePublish(target.Name)

		}

		if app.AwsConfig != nil {
			if app.AwsConfig.Ses != nil {
				runtimeStack.Policy.AddSES(app.AwsConfig.Ses)
			}
			if app.AwsConfig.SqsPublishAnywhere != nil && app.AwsConfig.SqsPublishAnywhere.SendAnywhere {
				runtimeStack.Policy.AddSQSPublish(cloudformation.Join("", []string{
					"arn:aws:sqs:",
					"*",
					":",
					cloudformation.Ref(AWSAccountIDParameter),
					":",
					"*",
				}))
			}
		}

		if err := runtimeStack.Apply(stackTemplate); err != nil {
			return nil, fmt.Errorf("adding %s: %w", runtime.Name, err)
		}
		stackTemplate.runtimes[runtime.Name] = runtimeStack
	}

	for _, listenerRule := range listener.Rules {
		stackTemplate.AddResource(listenerRule)
	}

	return stackTemplate.Build(), nil
}
