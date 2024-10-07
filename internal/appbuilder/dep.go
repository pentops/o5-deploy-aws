package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Globals interface {
	AppName() string
	FindRDSHost(string) (*RDSHost, bool)

	Bucket(string) (BucketRef, bool)
	Secret(string) (SecretRef, bool)
	Database(string) (DatabaseRef, bool)
}

type resourceBuilder interface {
	addBucket(*bucketInfo)
	addSecret(*secretInfo)
	addDatabase(DatabaseRef)
}

type globals struct {
	// input
	spec     *application_pb.Application
	rdsHosts RDSHostLookup

	// built
	databases map[string]DatabaseRef
	secrets   map[string]*secretInfo
	buckets   map[string]*bucketInfo
}

func (gg globals) FindRDSHost(serverGroup string) (*RDSHost, bool) {
	return gg.rdsHosts.FindRDSHost(serverGroup)
}

func (gg globals) AppName() string {
	return gg.spec.Name
}

func (gg globals) Bucket(name string) (BucketRef, bool) {
	bucket, ok := gg.buckets[name]
	if !ok {
		return nil, false
	}
	return bucket, true
}

func (gg globals) Secret(name string) (SecretRef, bool) {
	secret, ok := gg.secrets[name]
	if !ok {
		return nil, false
	}
	return secret, true
}

func (gg globals) Database(name string) (DatabaseRef, bool) {
	db, ok := gg.databases[name]
	if !ok {
		return nil, false
	}
	return db, true
}

func (gg globals) addBucket(summary *bucketInfo) {
	gg.buckets[summary.specName] = summary
}

func (gg globals) addSecret(summary *secretInfo) {
	gg.secrets[summary.refName] = summary
}

func (gg globals) addDatabase(summary DatabaseRef) {
	gg.databases[summary.Name()] = summary
}

type SecretRef interface {
	ARN() TemplateRef
	SecretValueFrom(jsonKey string) TemplateRef
}

type secretInfo struct {
	refName       string
	parameterName string
}

func (si secretInfo) ARN() TemplateRef {
	return TemplateRef(si.parameterName)
}

func (si secretInfo) SecretValueFrom(jsonKey string) TemplateRef {

	versionStage := ""
	versionID := ""
	return TemplateRef(cloudformation.Join(":", []string{
		si.parameterName,
		jsonKey,
		versionStage,
		versionID,
	}))
}

func mapResources(bb *Builder, resources resourceBuilder, app *application_pb.Application) error {

	for _, blobstoreDef := range app.Blobstores {
		ref, err := mapBlobstore(bb, blobstoreDef)
		if err != nil {
			return fmt.Errorf("bucket %s: %w", blobstoreDef.Name, err)
		}
		resources.addBucket(ref)
	}

	for _, secretDef := range app.Secrets {
		parameterName := fmt.Sprintf("AppSecret%s", cases.Title(language.English).String(secretDef.Name))
		secret := cflib.NewResource(parameterName, &secretsmanager.Secret{
			Name: cloudformation.JoinPtr("/", []string{
				"", // Leading /
				cloudformation.Ref(EnvNameParameter),
				app.Name,
				secretDef.Name,
			}),
			Description:                     cflib.Stringf("Application Level Secret for %s:%s - value must be set manually", app.Name, secretDef.Name),
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
		})
		bb.Template.AddResource(secret)
		resources.addSecret(&secretInfo{
			parameterName: parameterName,
			refName:       secretDef.Name,
		})
	}

	for _, databaseDef := range app.Databases {
		switch dbType := databaseDef.Engine.(type) {
		case *application_pb.Database_Postgres_:
			ref, def, err := mapPostgresDatabase(bb, databaseDef)
			if err != nil {
				return fmt.Errorf("mapping postgres database %s: %w", databaseDef.Name, err)
			}
			resources.addDatabase(ref)
			bb.AddPostgresResource(def)

			if dbType.Postgres.MigrateContainer != nil {
				err := mapPostgresMigration(bb, def, dbType.Postgres.MigrateContainer)
				if err != nil {
					return fmt.Errorf("mapping postgres migration for %s: %w", databaseDef.Name, err)
				}
			}

		default:
			return fmt.Errorf("unknown database engine type %T", databaseDef.Engine)
		}
	}

	return nil

}
