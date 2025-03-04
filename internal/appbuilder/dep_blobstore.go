package appbuilder

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/awslabs/goformation/v7/cloudformation/transfer"
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

func addSFTP(bb *Builder, blobstoreDef *application_pb.Blobstore) error {
	n := blobstoreDef.Name + "sftp"

	s := transfer.Server{
		Protocols:    []string{"SFTP"},
		EndpointType: aws.String("VPC"),
		EndpointDetails: &transfer.Server_EndpointDetails{
			SecurityGroupIds: []string{},
			// not sure if this one will work as I hope:
			SubnetIds: []string{cloudformation.Split(",", cloudformation.Ref(SubnetIDsParameter))},
			VpcId:     cloudformation.RefPtr(VPCParameter),
		},
		Tags: []tags.Tag{{Key: "Name", Value: n}},
	}

	sftp := cflib.NewResource(n, &s)
	bb.Template.AddResource(sftp)

	// need the s3 bucket read/write policy for this role:
	for _, u := range blobstoreDef.SftpSettings.Users {
		u1 := transfer.User{
			Role:          "", // TBD: IAM role ARN: it's for S3 access
			ServerId:      string(sftp.GetAtt("ServerId")),
			UserName:      u.Username,
			SshPublicKeys: []string{u.PublicSshKey},
			Policy:        &awsExamplePolicy, // IAM policy for this user on this bucket
			HomeDirectory: aws.String("/" + blobstoreDef.Name + "/" + u.Username),
		}
		user := cflib.NewResource(blobstoreDef.Name+u1.UserName, &u1)
		bb.Template.AddResource(user)
	}

	return nil
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

		cfg := &s3.Bucket{
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
			BucketName:                      bucketName.RefPtr(),
			NotificationConfiguration: &s3.Bucket_NotificationConfiguration{
				EventBridgeConfiguration: &s3.Bucket_EventBridgeConfiguration{
					EventBridgeEnabled: blobstoreDef.EmitEvents,
				},
			},
		}

		bucket := cflib.NewResource(blobstoreDef.Name, cfg)

		bb.Template.AddResource(bucket)

		bucketInfo := &bucketInfo{
			specName: blobstoreDef.Name,
			name:     bucketName,
			arn:      bucket.GetAtt("Arn"),
			read:     true,
			write:    true,
		}

		if blobstoreDef.SftpSettings != nil && len(blobstoreDef.SftpSettings.Users) > 0 {
			err := addSFTP(bb, blobstoreDef)
			if err != nil {
				return nil, fmt.Errorf("error setting sftp access: %w", err)
			}
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

// https://docs.aws.amazon.com/transfer/latest/userguide/users-policies-session.html
var awsExamplePolicy = `
{
  "Version": "2012-10-17",
  "Statement": [
      {
          "Sid": "AllowListingOfUserFolder",
          "Action": [
              "s3:ListBucket"
          ],
          "Effect": "Allow",
          "Resource": [
              "arn:aws:s3:::${transfer:HomeBucket}"
          ],
          "Condition": {
              "StringLike": {
                  "s3:prefix": [
                      "${transfer:HomeFolder}/*",
                      "${transfer:HomeFolder}"
                  ]
              }
          }
      },
      {
          "Sid": "HomeDirObjectAccess",
          "Effect": "Allow",
          "Action": [
              "s3:PutObject",
              "s3:GetObject",
              "s3:DeleteObjectVersion",
              "s3:DeleteObject",
              "s3:GetObjectVersion",
              "s3:GetObjectACL",
              "s3:PutObjectACL"
          ],
          "Resource": "arn:aws:s3:::${transfer:HomeDirectory}/*"
       }
  ]
}
`
