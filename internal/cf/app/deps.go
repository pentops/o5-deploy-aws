package app

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func mapResources(app *application_pb.Application, stackTemplate *Application) (*globalData, error) {

	global := globalData{

		appName:   app.Name,
		databases: map[string]DatabaseReference{},
		secrets:   map[string]*cf.Resource[*secretsmanager.Secret]{},
		buckets:   map[string]*bucketInfo{},
	}

	for _, blobstoreDef := range app.Blobstores {
		summary, resource, err := mapBlobstore(blobstoreDef, app.Name)
		if err != nil {
			return nil, fmt.Errorf("bucket %s: %w", blobstoreDef.Name, err)
		}

		if resource != nil {
			stackTemplate.AddResource(resource)
		}

		global.buckets[blobstoreDef.Name] = summary

	}

	for _, secretDef := range app.Secrets {
		parameterName := fmt.Sprintf("AppSecret%s", cases.Title(language.English).String(secretDef.Name))
		secret := cf.NewResource(parameterName, &secretsmanager.Secret{
			Name: cloudformation.JoinPtr("/", []string{
				"", // Leading /
				cloudformation.Ref(EnvNameParameter),
				app.Name,
				secretDef.Name,
			}),
			Description:                     cf.Stringf("Application Level Secret for %s:%s - value must be set manually", app.Name, secretDef.Name),
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
		})
		global.secrets[secretDef.Name] = secret
		stackTemplate.AddResource(secret)
	}

	return &global, nil

}

func mapBlobstore(blobstoreDef *application_pb.Blobstore, appName string) (*bucketInfo, *cf.Resource[*s3.Bucket], error) {
	if blobstoreDef.Ref == nil {
		bucketName := cloudformation.JoinPtr(".", []string{
			blobstoreDef.Name,
			appName,
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		})
		bucket := cf.NewResource(blobstoreDef.Name, &s3.Bucket{
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
			BucketName:                      bucketName,
		})
		return &bucketInfo{
			name:  bucketName,
			arn:   bucket.GetAtt("Arn"),
			read:  true,
			write: true,
		}, bucket, nil
	}

	readPermission := blobstoreDef.Ref.ReadPermission
	writePermission := blobstoreDef.Ref.WritePermission
	if !readPermission && !writePermission {
		// sand default is read-only
		readPermission = true
	}

	switch st := blobstoreDef.Ref.Source.(type) {
	case *application_pb.BlobstoreRef_Application:
		bucketName := cloudformation.JoinPtr(".", []string{
			blobstoreDef.Name,
			st.Application,
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		})

		return &bucketInfo{
			name: bucketName,
			arn: cloudformation.Join(":", []string{
				"arn:aws:s3",
				"", //cloudformation.Ref(AWSRegionParameter),
				"", //cloudformation.Ref(AWSAccountIDParameter),
				*bucketName}),
			read:  readPermission,
			write: writePermission,
		}, nil, nil

	case *application_pb.BlobstoreRef_BucketName:
		bucketName := st.BucketName
		return &bucketInfo{
			name:  &bucketName,
			arn:   cloudformation.Join(":", []string{"arn:aws:s3", cloudformation.Ref(AWSRegionParameter), cloudformation.Ref(AWSAccountIDParameter), bucketName}),
			read:  readPermission,
			write: writePermission,
		}, nil, nil

	default:
		return nil, nil, fmt.Errorf("unknown blobstore source type %T", st)
	}

}
