package appbuilder

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type BucketRef interface {
	Name() cflib.TemplateRef
	S3URL(subPathPtr *string) cflib.TemplateRef
	GetPermissions() RWPermission
	ARN() cflib.TemplateRef
}

type bucketInfo struct {
	specName string
	name     cflib.TemplateRef
	arn      cflib.TemplateRef
	read     bool
	write    bool
}

// Name returns a reference to the 'bucket name'
func (bi bucketInfo) Name() cflib.TemplateRef {
	return bi.name
}

func (bi bucketInfo) ARN() cflib.TemplateRef {
	return bi.arn
}

func (bi bucketInfo) S3URL(subPathPtr *string) cflib.TemplateRef {
	if subPathPtr == nil {
		return cflib.Join("", []string{
			"s3://",
			string(bi.name),
		})
	}
	subPath := *subPathPtr
	if !strings.HasPrefix(subPath, "/") {
		subPath = "/" + subPath
	}
	return cflib.Join("",
		"s3://",
		string(bi.name),
		subPath,
	)
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
		bucketName := cflib.Join(".",
			blobstoreDef.Name,
			appName,
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		)
		bucket := cflib.NewResource(blobstoreDef.Name, &s3.Bucket{
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
			BucketName:                      bucketName.RefPtr(),
		})
		bb.Template.AddResource(bucket)

		bucketInfo := &bucketInfo{
			specName: blobstoreDef.Name,
			name:     bucketName,
			arn:      bucket.GetAtt("Arn"),
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
		bucketName := cflib.Join(".",
			blobstoreDef.Name,
			st.Application,
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		)

		return &bucketInfo{
			specName: blobstoreDef.Name,
			name:     bucketName,
			arn: cflib.Join(":",
				"arn:aws:s3",
				"", //cloudformation.Ref(AWSRegionParameter),
				"", //cloudformation.Ref(AWSAccountIDParameter),
				bucketName),
			read:  readPermission,
			write: writePermission,
		}, nil

	case *application_pb.BlobstoreRef_BucketName:
		bucketName := cflib.TemplateRef(st.BucketName)
		return &bucketInfo{
			specName: blobstoreDef.Name,
			name:     bucketName,
			arn: cflib.Join(":",
				"arn:aws:s3",
				cloudformation.Ref(AWSRegionParameter),
				cloudformation.Ref(AWSAccountIDParameter),
				bucketName,
			),
			read:  readPermission,
			write: writePermission,
		}, nil

	default:
		return nil, fmt.Errorf("unknown blobstore source type %T", st)
	}
}
