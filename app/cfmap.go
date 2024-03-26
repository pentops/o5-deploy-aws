package app

import (

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"

	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	ECSClusterParameter           = "ECSCluster"
	ECSRepoParameter              = "ECSRepo"
	ECSTaskExecutionRoleParameter = "ECSTaskExecutionRole"
	VersionTagParameter           = "VersionTag"
	ListenerARNParameter          = "ListenerARN"
	HostHeaderParameter           = "HostHeader"
	EnvNameParameter              = "EnvName"
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

	AWSAccountIDParameter = "AWS::AccountId"

	O5SidecarContainerName = "o5_runtime"
	O5SidecarInternalPort  = 8081

	DeadLetterTargetName = "dead-letter"
	O5MonitorTargetName  = "o5-monitor"
)

type globalData struct {
	appName string

	databases map[string]DatabaseReference
	secrets   map[string]*Resource[*secretsmanager.Secret]
	buckets   map[string]*Resource[*s3.Bucket]

	replayChance     int64
	deadletterChance int64
}

// DatabaseReference is used to look up parameters ECS Task Definitions
type DatabaseReference struct {
	Definition     *application_pb.Database
	SecretResource *Resource[*secretsmanager.Secret]
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
		VPCParameter,
		VersionTagParameter,
		MetaDeployAssumeRoleParameter,
		JWKSParameter,
		CORSOriginParameter,
		SNSPrefixParameter,
		S3BucketNamespaceParameter,
		O5SidecarImageParameter,
		SESConditionsParameter,
		SourceTagParameter,
	} {
		parameter := &deployer_pb.Parameter{
			Name: key,
			Type: "String",
			Source: &deployer_pb.ParameterSourceType{
				Type: &deployer_pb.ParameterSourceType_WellKnown_{
					WellKnown: &deployer_pb.ParameterSourceType_WellKnown{},
				},
			},
		}
		switch key {
		case VersionTagParameter:
			parameter.Source.Type = &deployer_pb.ParameterSourceType_Static_{
				Static: &deployer_pb.ParameterSourceType_Static{
					Value: versionTag,
				},
			}
		case VPCParameter:
			parameter.Type = "AWS::EC2::VPC::Id"
		case SourceTagParameter:
			parameter.Source.Type = &deployer_pb.ParameterSourceType_Static_{
				Static: &deployer_pb.ParameterSourceType_Static{
					Value: fmt.Sprintf("o5/%s/%s", app.Name, versionTag),
				},
			}
		}
		stackTemplate.AddParameter(parameter)
	}

	global := globalData{
		appName:          app.Name,
		databases:        map[string]DatabaseReference{},
		secrets:          map[string]*Resource[*secretsmanager.Secret]{},
		buckets:          map[string]*Resource[*s3.Bucket]{},
		replayChance:     0,
		deadletterChance: 0,
	}

	if app.SidecarConfig != nil && app.SidecarConfig.DeadletterChance > 0 {
		global.deadletterChance = app.SidecarConfig.DeadletterChance
	}
	if app.SidecarConfig != nil && app.SidecarConfig.ReplayChance > 0 {
		global.replayChance = app.SidecarConfig.ReplayChance
	}

	for _, blobstoreDef := range app.Blobstores {
		bucket := NewResource(blobstoreDef.Name, &s3.Bucket{
			BucketName: cloudformation.JoinPtr(".", []string{
				blobstoreDef.Name,
				app.Name,
				cloudformation.Ref(EnvNameParameter),
				cloudformation.Ref(AWSRegionParameter),
				cloudformation.Ref(S3BucketNamespaceParameter),
			}),
		})
		global.buckets[blobstoreDef.Name] = bucket
		stackTemplate.AddResource(bucket)
	}

	for _, secretDef := range app.Secrets {
		parameterName := fmt.Sprintf("AppSecret%s", cases.Title(language.English).String(secretDef.Name))
		secret := NewResource(parameterName, &secretsmanager.Secret{
			Name: cloudformation.JoinPtr("/", []string{
				"", // Leading /
				cloudformation.Ref(EnvNameParameter),
				app.Name,
				secretDef.Name,
			}),
			Description: Stringf("Application Level Secret for %s:%s - value must be set manually", app.Name, secretDef.Name),
		})
		global.secrets[secretDef.Name] = secret
		stackTemplate.AddResource(secret)
	}

	runtimes := map[string]*RuntimeService{}

	for _, database := range app.Databases {
		switch dbType := database.Engine.(type) {
		case *application_pb.Database_Postgres_:

			parameterName := fmt.Sprintf("DatabaseSecret%s", cases.Title(language.English).String(database.Name))

			secret := NewResource(parameterName, &secretsmanager.Secret{
				Name: cloudformation.JoinPtr("/", []string{
					"", // Leading /
					cloudformation.Ref(EnvNameParameter),
					app.Name,
					"postgres",
					database.Name,
				}),
				Description: Stringf("Secret for Postgres database %s in app %s", database.Name, app.Name),
			})

			stackTemplate.AddResource(secret)

			ref := DatabaseReference{
				SecretResource: secret,
				Definition:     database,
			}
			global.databases[database.Name] = ref

			secretName := fmt.Sprintf("DatabaseSecret%s", CleanParameterName(database.Name))
			def := &deployer_pb.PostgresDatabaseResource{
				DbName:           database.Name,
				ServerGroup:      dbType.Postgres.ServerGroup,
				SecretOutputName: secretName,
				DbExtensions:     dbType.Postgres.DbExtensions,
			}
			if dbType.Postgres.DbName != "" {
				def.DbName = dbType.Postgres.DbName
			}

			stackTemplate.AddOutput(&Output{
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
				migrationContainer, err := buildContainer(global, dbType.Postgres.MigrateContainer)
				if err != nil {
					return nil, fmt.Errorf("building migration container for %s: %w", database.Name, err)
				}
				addLogs(migrationContainer.Container, fmt.Sprintf("%s/migrate", global.appName))
				name := fmt.Sprintf("MigrationTaskDefinition%s", CleanParameterName(database.Name))

				migrationTaskDefinition := NewResource(name, &ecs.TaskDefinition{
					ContainerDefinitions: []ecs.TaskDefinition_ContainerDefinition{
						*migrationContainer.Container,
					},
					Family:                  String(fmt.Sprintf("%s_migrate_%s", global.appName, database.Name)),
					ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
					RequiresCompatibilities: []string{"EC2"},
				})
				stackTemplate.AddResource(migrationTaskDefinition)
				def.MigrationTaskOutputName = String(name)
				stackTemplate.AddOutput(&Output{
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

		for _, target := range app.Targets {
			stackTemplate.AddSNSTopic(target.Name)
		}
	}

	listener := NewListenerRuleSet()

	for _, runtime := range app.Runtimes {
		runtimeStack, err := NewRuntimeService(global, runtime)
		if err != nil {
			return nil, err
		}
		runtimes[runtime.Name] = runtimeStack

		if err := runtimeStack.AddRoutes(listener); err != nil {
			return nil, fmt.Errorf("adding routes to %s: %w", runtime.Name, err)
		}

		for _, target := range app.Targets {
			snsTopicARN := cloudformation.Join("", []string{
				"arn:aws:sns:",
				cloudformation.Ref(AWSRegionParameter),
				":",
				cloudformation.Ref(AWSAccountIDParameter),
				":",
				cloudformation.Ref(EnvNameParameter),
				"-",
				target.Name,
			})
			runtimeStack.Policy.AddSNSPublish(snsTopicARN)
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
