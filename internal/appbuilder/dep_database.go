package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
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

	ServerGroup() DatabaseServerGroup
	AuroraProxy() (*AuroraDatabaseRef, bool)
	SecretValueFrom() (cflib.TemplateRef, bool)
}

type DatabaseServerGroup struct {
	GroupName           string
	ClientSecurityGroup cflib.TemplateRef
}

// DatabaseReference is used to look up parameters ECS Task Definitions

type AuroraDatabaseRef struct {
	refName            string
	group              DatabaseServerGroup
	endpointParamRef   cflib.TemplateRef
	identifierParamRef cflib.TemplateRef
	dbNameParamRef     cflib.TemplateRef
}

func (dbDef AuroraDatabaseRef) Name() string {
	return dbDef.refName
}

func (dbDef AuroraDatabaseRef) IsProxy() bool {
	return true
}

func (dbDef AuroraDatabaseRef) SecretValueFrom() (cflib.TemplateRef, bool) {
	return "", false
}

func (dbDef AuroraDatabaseRef) ServerGroup() DatabaseServerGroup {
	return dbDef.group
}

func (dbDef AuroraDatabaseRef) AuroraProxy() (*AuroraDatabaseRef, bool) {
	return &dbDef, true
}

func (dbDef AuroraDatabaseRef) AuroraEndpoint() cflib.TemplateRef {
	return dbDef.endpointParamRef
}

func (dbDef AuroraDatabaseRef) AuroraConnectARN() cflib.TemplateRef {
	joined := cloudformation.Join(":", []string{
		"arn",
		"aws",
		"rds-db",
		cloudformation.Ref("AWS::Region"),
		cloudformation.Ref("AWS::AccountId"),
		"dbuser",
		cloudformation.Join("/", []string{
			string(dbDef.identifierParamRef),
			string(dbDef.dbNameParamRef),
		}),
	})

	return cflib.TemplateRef(joined)
}

func (dbDef AuroraDatabaseRef) DSNToProxy(host string) string {
	return fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable", host, dbDef.refName, dbDef.refName)
}

type secretsDatabaseRef struct {
	refName        string
	group          DatabaseServerGroup
	secretResource *cflib.Resource[*secretsmanager.Secret]
}

func (dbDef secretsDatabaseRef) Name() string {
	return dbDef.refName
}

func (dbDef secretsDatabaseRef) IsProxy() bool {
	return false
}

func (dbDef secretsDatabaseRef) ServerGroup() DatabaseServerGroup {
	return dbDef.group
}

func (dbDef secretsDatabaseRef) AuroraProxy() (*AuroraDatabaseRef, bool) {
	return nil, false
}

func (dbDef secretsDatabaseRef) SecretValueFrom() (cflib.TemplateRef, bool) {
	jsonKey := "dburl"
	versionStage := ""
	versionID := ""
	return cflib.Join(":",
		dbDef.secretResource.Ref(),
		jsonKey,
		versionStage,
		versionID,
	), true
}

func mapPostgresDatabase(builder *Builder, database *application_pb.Database) (DatabaseRef, *awsdeployer_pb.PostgresDatabaseResource, error) {

	dbType := database.GetPostgres()

	dbHost, ok := builder.FindRDSHost(dbType.ServerGroup)
	if !ok {
		return nil, nil, fmt.Errorf("no RDS host %q for database %s", dbType.ServerGroup, database.Name)
	}

	serverGroupClientSGParamName := fmt.Sprintf("DatabaseGroup%sClientSecurityGroup", cflib.CleanParameterName(dbType.ServerGroup))
	builder.Template.AddParameter(&awsdeployer_pb.Parameter{
		Name:        serverGroupClientSGParamName,
		Type:        "String",
		Description: fmt.Sprintf("Client security group for Postgres database %s in app %s", database.Name, builder.AppName()),
		Source: &awsdeployer_pb.ParameterSourceType{
			Type: &awsdeployer_pb.ParameterSourceType_DatabaseServer_{
				DatabaseServer: &awsdeployer_pb.ParameterSourceType_DatabaseServer{
					ServerGroup: dbType.ServerGroup,
					Part:        awsdeployer_pb.ParameterSourceType_DatabaseServer_PART_CLIENT_SECURITY_GROUP,
				},
			},
		},
	})

	group := DatabaseServerGroup{
		GroupName:           dbType.ServerGroup,
		ClientSecurityGroup: cflib.TemplateRef(cloudformation.Ref(serverGroupClientSGParamName)),
	}

	def := &awsdeployer_pb.PostgresDatabaseResource{
		AppKey:       database.Name,
		DbNameSuffix: dbType.DbNameSuffix,
		DbExtensions: dbType.DbExtensions,
		ServerGroup:  dbType.ServerGroup,
	}

	if dbType.DbNameSuffix == "" {
		def.DbNameSuffix = database.Name
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
			group:          group,
		}

	case environment_pb.RDSAuth_Iam:
		endpointParamName := fmt.Sprintf("DatabaseParam%sJSON", cflib.CleanParameterName(database.Name))
		identifierParamName := fmt.Sprintf("DatabaseParam%sIdentifier", cflib.CleanParameterName(database.Name))
		dbNameParamName := fmt.Sprintf("DatabaseParam%sDbName", cflib.CleanParameterName(database.Name))
		for param, part := range map[string]awsdeployer_pb.ParameterSourceType_Aurora_Part{
			endpointParamName:   awsdeployer_pb.ParameterSourceType_Aurora_PART_JSON,
			identifierParamName: awsdeployer_pb.ParameterSourceType_Aurora_PART_IDENTIFIER,
			dbNameParamName:     awsdeployer_pb.ParameterSourceType_Aurora_PART_DBNAME,
		} {
			builder.Template.AddParameter(&awsdeployer_pb.Parameter{
				Name:        param,
				Type:        "String",
				Description: fmt.Sprintf("IAM Postgres database %s in app %s : %s", database.Name, builder.AppName(), part.String()),
				Source: &awsdeployer_pb.ParameterSourceType{
					Type: &awsdeployer_pb.ParameterSourceType_Aurora_{
						Aurora: &awsdeployer_pb.ParameterSourceType_Aurora{
							ServerGroup: dbType.ServerGroup,
							AppKey:      def.AppKey,
							Part:        part,
						},
					},
				},
			})
		}

		def.Connection = &awsdeployer_pb.PostgresDatabaseResource_ParameterName{
			ParameterName: endpointParamName,
		}
		ref = &AuroraDatabaseRef{
			refName:            database.Name, // Matching the passed in reference name for lookup
			endpointParamRef:   cflib.TemplateRef(cloudformation.Ref(endpointParamName)),
			identifierParamRef: cflib.TemplateRef(cloudformation.Ref(identifierParamName)),
			dbNameParamRef:     cflib.TemplateRef(cloudformation.Ref(dbNameParamName)),
			group:              group,
		}
	default:
		return nil, nil, fmt.Errorf("unknown auth type %q for database %s", dbHost.AuthType, database.Name)
	}

	return ref, def, nil
}

func mapPostgresMigration(builder *Builder, resource *awsdeployer_pb.PostgresDatabaseResource, spec *application_pb.Container) error {

	if spec.Name == "" {
		spec.Name = "migrate"
	}

	taskDefinition := NewECSTaskDefinition(builder.Globals, fmt.Sprintf("migrate_%s", resource.AppKey))
	err := taskDefinition.BuildRuntimeContainer(spec)
	if err != nil {
		return fmt.Errorf("building migration container: %w", err)
	}

	taskDef, err := taskDefinition.AddToTemplate(builder.Template)
	if err != nil {
		return fmt.Errorf("adding task definition for migration: %w", err)
	}
	outputName := fmt.Sprintf("MigrationTaskDefinition%s", cflib.CleanParameterName(resource.AppKey))
	builder.Template.AddOutput(&cflib.Output{
		Name:  outputName,
		Value: taskDef,
	})
	resource.MigrationTaskOutputName = cflib.String(outputName)

	return nil

}
