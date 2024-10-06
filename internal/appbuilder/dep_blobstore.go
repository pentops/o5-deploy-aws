package appbuilder

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
)

type BucketRef interface {
	Name() TemplateRef
	S3URL(subPathPtr *string) TemplateRef
	GetPermissions() RWPermission
	ARN() TemplateRef
}

type bucketInfo struct {
	specName string
	name     TemplateRef
	arn      TemplateRef
	read     bool
	write    bool
}

// Name returns a reference to the 'bucket name'
func (bi bucketInfo) Name() TemplateRef {
	return bi.name
}

func (bi bucketInfo) ARN() TemplateRef {
	return bi.arn
}

func (bi bucketInfo) S3URL(subPathPtr *string) TemplateRef {
	if subPathPtr == nil {
		return TemplateRef(cloudformation.Join("", []string{
			"s3://",
			string(bi.name),
		}))
	}
	subPath := *subPathPtr
	if !strings.HasPrefix(subPath, "/") {
		subPath = "/" + subPath
	}
	return TemplateRef(cloudformation.Join("", []string{
		"s3://",
		string(bi.name),
		subPath,
	}))
}

type RWPermission int

const (
	ReadOnly RWPermission = iota
	WriteOnly
	ReadWrite
)

func (bi bucketInfo) GetPermissions() RWPermission {
	if !bi.read && !bi.write {
		return ReadOnly
	}
	if bi.read && bi.write {
		return ReadWrite
	}
	if bi.read {
		return ReadOnly
	}
	return WriteOnly
}

func mapBlobstore(bb *Builder, blobstoreDef *application_pb.Blobstore) (*bucketInfo, error) {

	appName := bb.Globals.AppName()

	if blobstoreDef.Ref == nil {
		bucketName := cloudformation.Join(".", []string{
			blobstoreDef.Name,
			appName,
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		})
		bucket := cf.NewResource(blobstoreDef.Name, &s3.Bucket{
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
			BucketName:                      cloudformation.String(bucketName),
		})
		bb.Template.AddResource(bucket)

		bucketInfo := &bucketInfo{
			specName: blobstoreDef.Name,
			name:     TemplateRef(bucketName),
			arn:      TemplateRef(bucket.GetAtt("Arn")),
			read:     true,
			write:    true,
		}
		return bucketInfo, nil
	}

	readPermission := blobstoreDef.Ref.ReadPermission
	writePermission := blobstoreDef.Ref.WritePermission
	if !readPermission && !writePermission {
		// sand default is read-only
		readPermission = true
	}

	switch st := blobstoreDef.Ref.Source.(type) {
	case *application_pb.BlobstoreRef_Application:
		bucketName := cloudformation.Join(".", []string{
			blobstoreDef.Name,
			st.Application,
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		})

		return &bucketInfo{
			specName: blobstoreDef.Name,
			name:     TemplateRef(bucketName),
			arn: TemplateRef(cloudformation.Join(":", []string{
				"arn:aws:s3",
				"", //cloudformation.Ref(AWSRegionParameter),
				"", //cloudformation.Ref(AWSAccountIDParameter),
				bucketName})),
			read:  readPermission,
			write: writePermission,
		}, nil

	case *application_pb.BlobstoreRef_BucketName:
		bucketName := st.BucketName
		return &bucketInfo{
			specName: blobstoreDef.Name,
			name:     TemplateRef(bucketName),
			arn: TemplateRef(
				cloudformation.Join(":", []string{
					"arn:aws:s3",
					cloudformation.Ref(AWSRegionParameter),
					cloudformation.Ref(AWSAccountIDParameter),
					bucketName,
				}),
			),
			read:  readPermission,
			write: writePermission,
		}, nil

	default:
		return nil, fmt.Errorf("unknown blobstore source type %T", st)
	}
}
