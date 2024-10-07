package appbuilder

import (
	"encoding/json"
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
	IsProxy() bool
	SecretValueFrom() (TemplateRef, bool)
	ProxyEnvVal(proxyHost string) (*string, error)
	Name() string
}

// DatabaseReference is used to look up parameters ECS Task Definitions
type DatabaseReference struct {
	refName        string
	Definition     *application_pb.Database
	SecretResource *cflib.Resource[*secretsmanager.Secret]
}

func (dbDef DatabaseReference) Name() string {
	return dbDef.refName
}

func (dbDef DatabaseReference) SecretValueFrom() (TemplateRef, bool) {
	if dbDef.SecretResource == nil {
		return "", false
	}
	jsonKey := "dburl"
	versionStage := ""
	versionID := ""
	return TemplateRef(cloudformation.Join(":", []string{
		dbDef.SecretResource.Ref(),
		jsonKey,
		versionStage,
		versionID,
	})), true
}

type DBEnvVar struct {
	Username string `json:"dbuser"`
	Password string `json:"dbpass"`
	Hostname string `json:"dbhost"`
	DBName   string `json:"dbname"`
	URL      string `json:"dburl"`
}

func (dbDef DatabaseReference) ProxyEnvVal(proxyHost string) (*string, error) {
	if dbDef.SecretResource != nil {
		panic("SecretResource is not nil calling ProxyEnvVal")
	}

	data := DBEnvVar{
		Username: dbDef.Definition.Name,
		Password: "proxy",
		Hostname: proxyHost,
		DBName:   dbDef.Definition.Name,
	}
	data.URL = fmt.Sprintf("postgres://%s:%s@%s/%s", data.Username, data.Password, data.Hostname, data.DBName)

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling DBEnvVar: %w", err)
	}

	str := string(jsonBytes)
	return &str, nil
}

func (dbDef DatabaseReference) IsProxy() bool {
	return dbDef.SecretResource == nil
}

func mapPostgresDatabase(builder *Builder, database *application_pb.Database) (*DatabaseReference, *awsdeployer_pb.PostgresDatabaseResource, error) {

	dbType := database.GetPostgres()

	dbHost, ok := builder.Globals.FindRDSHost(dbType.ServerGroup)
	if !ok {
		return nil, nil, fmt.Errorf("no RDS host %q for database %s", dbType.ServerGroup, database.Name)
	}
	ref := DatabaseReference{
		refName:    database.Name,
		Definition: database,
	}

	def := &awsdeployer_pb.PostgresDatabaseResource{
		DbName:       database.Name,
		DbExtensions: dbType.DbExtensions,
		ServerGroup:  dbType.ServerGroup,
	}

	if dbType.DbName != "" {
		def.DbName = dbType.DbName
	}

	if dbHost.AuthType == environment_pb.RDSAuth_SecretsManager {
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

		ref.SecretResource = secret
		def.Connection = &awsdeployer_pb.PostgresDatabaseResource_SecretOutputName{
			SecretOutputName: secretName,
		}

	} else {
		paramName := fmt.Sprintf("DatabaseParam%s", cflib.CleanParameterName(database.Name))
		builder.Template.AddParameter(&awsdeployer_pb.Parameter{
			Name:        paramName,
			Type:        "String",
			Description: fmt.Sprintf("Parameter for IAM Postgres database %s in app %s", database.Name, builder.AppName()),
		})
		def.Connection = &awsdeployer_pb.PostgresDatabaseResource_ParameterName{
			ParameterName: paramName,
		}
	}

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
	name := fmt.Sprintf("MigrationTaskDefinition%s", cflib.CleanParameterName(resource.DbName))

	migrationTaskDefinition := cflib.NewResource(name, &ecs.TaskDefinition{
		ContainerDefinitions: []ecs.TaskDefinition_ContainerDefinition{
			*migrationContainer.Container,
		},
		Family:                  cflib.String(fmt.Sprintf("%s_migrate_%s", builder.AppName(), resource.DbName)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
	})

	builder.Template.AddResource(migrationTaskDefinition)

	resource.MigrationTaskOutputName = cflib.String(name)

	builder.Template.AddOutput(&cflib.Output{
		Name:  name,
		Value: migrationTaskDefinition.Ref(),
	})

	return nil

}
