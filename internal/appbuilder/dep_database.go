package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type DatabaseRef interface {
	// Name as specified in the application o5 file
	Name() string

	IsProxy() bool
	SecretValueFrom() (TemplateRef, bool)
	EndpointParameter() (TemplateRef, bool)
}

// DatabaseReference is used to look up parameters ECS Task Definitions

type iamDatabaseRef struct {
	refName  string
	paramRef TemplateRef
}

func (dbDef iamDatabaseRef) Name() string {
	return dbDef.refName
}

func (dbDef iamDatabaseRef) EndpointParameter() (TemplateRef, bool) {
	return dbDef.paramRef, true
}

func (dbDef iamDatabaseRef) SecretValueFrom() (TemplateRef, bool) {
	return "", false
}

func (dbDef iamDatabaseRef) IsProxy() bool {
	return true
}

type secretsDatabaseRef struct {
	refName        string
	secretResource *cflib.Resource[*secretsmanager.Secret]
}

func (dbDef secretsDatabaseRef) Name() string {
	return dbDef.refName
}

func (dbDef secretsDatabaseRef) IsProxy() bool {
	return false
}

func (dbDef secretsDatabaseRef) EndpointParameter() (TemplateRef, bool) {
	return "", false
}

func (dbDef secretsDatabaseRef) SecretValueFrom() (TemplateRef, bool) {
	jsonKey := "dburl"
	versionStage := ""
	versionID := ""
	return TemplateRef(cloudformation.Join(":", []string{
		dbDef.secretResource.Ref(),
		jsonKey,
		versionStage,
		versionID,
	})), true
}

func mapPostgresDatabase(builder *Builder, database *application_pb.Database) (DatabaseRef, *awsdeployer_pb.PostgresDatabaseResource, error) {

	dbType := database.GetPostgres()

	dbHost, ok := builder.Globals.FindRDSHost(dbType.ServerGroup)
	if !ok {
		return nil, nil, fmt.Errorf("no RDS host %q for database %s", dbType.ServerGroup, database.Name)
	}

	def := &awsdeployer_pb.PostgresDatabaseResource{
		DbName:       database.Name,
		DbExtensions: dbType.DbExtensions,
		ServerGroup:  dbType.ServerGroup,
	}

	if dbType.DbName != "" {
		def.DbName = dbType.DbName
	}

	var ref DatabaseRef
	switch dbHost.AuthType {
	case environment_pb.RDSAuth_SecretsManager:
		secret := cflib.NewResource(cflib.CleanParameterName("Database", database.Name), &secretsmanager.Secret{
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
			Name: cloudformation.JoinPtr("/", []string{
				"", // Leading /
				cloudformation.Ref(EnvNameParameter),
				builder.AppName(),
				"postgres",
				database.Name,
			}),
			Description: cflib.Stringf("Secret for Postgres database %s in app %s", database.Name, builder.AppName()),
		})

		builder.Template.AddResource(secret)
		secretName := fmt.Sprintf("DatabaseSecret%s", cflib.CleanParameterName(database.Name))
		builder.Template.AddOutput(&cflib.Output{
			Name:  secretName,
			Value: secret.Ref(),
		})

		def.Connection = &awsdeployer_pb.PostgresDatabaseResource_SecretOutputName{
			SecretOutputName: secretName,
		}
		ref = &secretsDatabaseRef{
			refName:        database.Name,
			secretResource: secret,
		}

	case environment_pb.RDSAuth_Iam:
		paramName := fmt.Sprintf("DatabaseParam%s", cflib.CleanParameterName(database.Name))
		builder.Template.AddParameter(&awsdeployer_pb.Parameter{
			Name:        paramName,
			Type:        "String",
			Description: fmt.Sprintf("Parameter for IAM Postgres database %s in app %s", database.Name, builder.AppName()),
			Source: &awsdeployer_pb.ParameterSourceType{
				Type: &awsdeployer_pb.ParameterSourceType_AuroraEndpoint_{
					AuroraEndpoint: &awsdeployer_pb.ParameterSourceType_AuroraEndpoint{
						ServerGroup: dbType.ServerGroup,
						DbName:      def.DbName, // Matching the resource def
					},
				},
			},
		})
		def.Connection = &awsdeployer_pb.PostgresDatabaseResource_ParameterName{
			ParameterName: paramName,
		}
		ref = &iamDatabaseRef{
			refName:  database.Name, // Matching the passed in reference name for lookup
			paramRef: TemplateRef(cloudformation.Ref(paramName)),
		}
	default:
		return nil, nil, fmt.Errorf("unknown auth type %q for database %s", dbHost.AuthType, database.Name)
	}

	if dbType.MigrateContainer != nil {
		err := mapPostgresMigration(builder, def, dbType.MigrateContainer)
		if err != nil {
			return nil, nil, fmt.Errorf("mapping postgres migration for %s: %w", database.Name, err)
		}
	}

	return ref, def, nil
}

func mapPostgresMigration(builder *Builder, resource *awsdeployer_pb.PostgresDatabaseResource, spec *application_pb.Container) error {

	if spec.Name == "" {
		spec.Name = "migrate"
	}

	migrationContainer, err := buildContainer(builder, nil, spec)
	if err != nil {
		return fmt.Errorf("building migration container for %s: %w", resource.DbName, err)
	}

	name := fmt.Sprintf("MigrationTaskDefinition%s", cflib.CleanParameterName(resource.DbName))

	taskDef := cflib.NewResource(name, &ecs.TaskDefinition{
		ContainerDefinitions: []ecs.TaskDefinition_ContainerDefinition{
			*migrationContainer.Container,
		},
		Family:                  cflib.String(fmt.Sprintf("%s_migrate_%s", builder.AppName(), resource.DbName)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
	})

	if len(migrationContainer.ProxyDBs) > 0 {
		// Migration requires sidecar
		sidecar := NewSidecarBuilder(builder.Globals.AppName())
		for _, ref := range migrationContainer.ProxyDBs {
			envVarValue := sidecar.ProxyDB(ref.Database)
			*ref.EnvVarVal = envVarValue
		}
		container, err := sidecar.Build()
		if err != nil {
			return fmt.Errorf("building migration sidecar for %s: %w", resource.DbName, err)
		}

		taskDef.Resource.ContainerDefinitions = append(taskDef.Resource.ContainerDefinitions, *container)
	}

	builder.Template.AddResource(taskDef)

	resource.MigrationTaskOutputName = cflib.String(name)

	builder.Template.AddOutput(&cflib.Output{
		Name:  name,
		Value: taskDef.Ref(),
	})

	return nil

}
