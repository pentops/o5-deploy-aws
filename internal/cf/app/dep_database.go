package app

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
)

type DatabaseRef interface {
	SecretValueFrom() TemplateRef
}

// DatabaseReference is used to look up parameters ECS Task Definitions
type DatabaseReference struct {
	refName        string
	Definition     *application_pb.Database
	SecretResource *cf.Resource[*secretsmanager.Secret]
}

func (dbDef DatabaseReference) SecretValueFrom() TemplateRef {
	jsonKey := "dburl"
	versionStage := ""
	versionID := ""
	return TemplateRef(cloudformation.Join(":", []string{
		dbDef.SecretResource.Ref(),
		jsonKey,
		versionStage,
		versionID,
	}))
}

func mapPostgresDatabase(builder *Builder, database *application_pb.Database) (*DatabaseReference, *awsdeployer_pb.PostgresDatabaseResource, error) {

	dbType := database.GetPostgres()

	secret := cf.NewResource(cf.CleanParameterName("Database", database.Name), &secretsmanager.Secret{
		AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
		Name: cloudformation.JoinPtr("/", []string{
			"", // Leading /
			cloudformation.Ref(EnvNameParameter),
			builder.AppName(),
			"postgres",
			database.Name,
		}),
		Description: cf.Stringf("Secret for Postgres database %s in app %s", database.Name, builder.AppName()),
	})

	builder.Template.AddResource(secret)

	ref := DatabaseReference{
		refName:        database.Name,
		SecretResource: secret,
		Definition:     database,
	}

	secretName := fmt.Sprintf("DatabaseSecret%s", cf.CleanParameterName(database.Name))

	def := &awsdeployer_pb.PostgresDatabaseResource{
		DbName:           database.Name,
		ServerGroup:      dbType.ServerGroup,
		SecretOutputName: secretName,
		DbExtensions:     dbType.DbExtensions,
	}

	if dbType.DbName != "" {
		def.DbName = dbType.DbName
	}

	builder.Template.AddOutput(&cf.Output{
		Name:  secretName,
		Value: secret.Ref(),
	})

	return &ref, def, nil
}

func mapPostgresMigration(builder *Builder, resource *awsdeployer_pb.PostgresDatabaseResource, spec *application_pb.Container) error {

	if spec.Name == "" {
		spec.Name = "migrate"
	}

	migrationContainer, err := buildContainer(builder, nil, spec)
	if err != nil {
		return fmt.Errorf("building migration container for %s: %w", resource.DbName, err)
	}
	addLogs(migrationContainer.Container, fmt.Sprintf("%s/migrate", builder.AppName()))
	name := fmt.Sprintf("MigrationTaskDefinition%s", cf.CleanParameterName(resource.DbName))

	migrationTaskDefinition := cf.NewResource(name, &ecs.TaskDefinition{
		ContainerDefinitions: []ecs.TaskDefinition_ContainerDefinition{
			*migrationContainer.Container,
		},
		Family:                  cf.String(fmt.Sprintf("%s_migrate_%s", builder.AppName(), resource.DbName)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
	})

	builder.Template.AddResource(migrationTaskDefinition)

	resource.MigrationTaskOutputName = cf.String(name)

	builder.Template.AddOutput(&cf.Output{
		Name:  name,
		Value: migrationTaskDefinition.Ref(),
	})

	return nil

}
