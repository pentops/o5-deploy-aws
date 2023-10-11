package app

import (

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"

	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-go/application/v1/application_pb"
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

	O5SidecarContainerName = "o5_runtime"
	O5SidecarImageName     = "ghcr.io/pentops/o5-runtime-sidecar:latest"
)

type globalData struct {
	appName string

	databases map[string]DatabaseReference
	secrets   map[string]*Resource[*secretsmanager.Secret]
	buckets   map[string]*Resource[*s3.Bucket]
}

// DatabaseReference is used to look up parameters ECS Task Definitions
type DatabaseReference struct {
	Definition     *application_pb.Database
	SecretResource *Resource[*secretsmanager.Secret]
}

func BuildApplication(app *application_pb.Application, versionTag string) (*Application, error) {

	stackTemplate := NewApplication(app.Name)

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
	} {
		parameter := &Parameter{
			Name:   key,
			Type:   "String",
			Source: ParameterSourceWellKnown,
		}
		switch key {
		case VersionTagParameter:
			parameter.Default = String(versionTag)
			parameter.Source = ParameterSourceDefault
		case VPCParameter:
			parameter.Type = "AWS::EC2::VPC::Id"
		}
		stackTemplate.AddParameter(parameter)
	}

	global := globalData{
		appName:   app.Name,
		databases: map[string]DatabaseReference{},
		secrets:   map[string]*Resource[*secretsmanager.Secret]{},
		buckets:   map[string]*Resource[*s3.Bucket]{},
	}

	for _, blobstoreDef := range app.Blobstores {
		bucket := NewResource(blobstoreDef.Name, &s3.Bucket{
			BucketName: cloudformation.JoinPtr("-", []string{
				cloudformation.Ref(EnvNameParameter),
				app.Name,
				blobstoreDef.Name,
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

			def := &PostgresDefinition{
				Databse:  database,
				Postgres: dbType.Postgres,
				Secret:   secret,
			}

			secretName := fmt.Sprintf("DatabaseSecret%s", CleanParameterName(database.Name))
			def.SecretOutputName = String(secretName)
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
					return nil, err
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

		runtimeStack.Apply(stackTemplate)
	}

	for _, listenerRule := range listener.Rules {
		stackTemplate.AddResource(listenerRule)
	}

	return stackTemplate, nil
}
