package appbuilder

import (

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"

	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
)

const (
	ECSClusterParameter           = "ECSCluster"
	ECSRepoParameter              = "ECSRepo"
	ECSTaskExecutionRoleParameter = "ECSTaskExecutionRole"
	VersionTagParameter           = "VersionTag"
	ListenerARNParameter          = "ListenerARN"
	HostHeaderParameter           = "HostHeader"
	EnvNameParameter              = "EnvName"
	ClusterNameParameter          = "ClusterName"
	VPCParameter                  = "VPCID"
	MetaDeployAssumeRoleParameter = "MetaDeployAssumeRoleArns"
	JWKSParameter                 = "JWKS"
	AWSRegionParameter            = "AWS::Region"
	CORSOriginParameter           = "CORSOrigin"
	SNSPrefixParameter            = "SNSPrefix"
	S3BucketNamespaceParameter    = "S3BucketNamespace"
	O5SidecarImageParameter       = "O5SidecarImage"
	SourceTagParameter            = "SourceTag"
	EventBusARNParameter          = "EventBusARN"
	SecurityGroupParameter        = "SecurityGroup"
	SubnetIDsParameter            = "SubnetIDs"

	AWSAccountIDParameter = "AWS::AccountId"

	O5SidecarContainerName = "o5_runtime"
	O5SidecarInternalPort  = 8081

	DeadLetterTargetName = "dead-letter"
	O5MonitorTargetName  = "o5-monitor"
)

func sourceTags(extra ...tags.Tag) []tags.Tag {
	return append(extra, tags.Tag{
		Key:   "o5-source",
		Value: cloudformation.Ref(SourceTagParameter),
	})
}

func addGlobalParameters(bb *Builder, versionTag string) {
	for _, key := range []string{
		ECSClusterParameter,
		ECSRepoParameter,
		ECSTaskExecutionRoleParameter,
		ListenerARNParameter,
		HostHeaderParameter,
		EnvNameParameter,
		ClusterNameParameter,
		VPCParameter,
		VersionTagParameter,
		MetaDeployAssumeRoleParameter,
		JWKSParameter,
		CORSOriginParameter,
		SNSPrefixParameter,
		S3BucketNamespaceParameter,
		O5SidecarImageParameter,
		EventBusARNParameter,
		SourceTagParameter,
		SecurityGroupParameter,
		SubnetIDsParameter,
	} {
		parameter := &awsdeployer_pb.Parameter{
			Name: key,
			Type: "String",
			Source: &awsdeployer_pb.ParameterSourceType{
				Type: &awsdeployer_pb.ParameterSourceType_WellKnown_{
					WellKnown: &awsdeployer_pb.ParameterSourceType_WellKnown{},
				},
			},
		}
		switch key {
		case VersionTagParameter:
			parameter.Source.Type = &awsdeployer_pb.ParameterSourceType_Static_{
				Static: &awsdeployer_pb.ParameterSourceType_Static{
					Value: versionTag,
				},
			}
		case VPCParameter:
			parameter.Type = "AWS::EC2::VPC::Id"
		case SourceTagParameter:
			parameter.Source.Type = &awsdeployer_pb.ParameterSourceType_Static_{
				Static: &awsdeployer_pb.ParameterSourceType_Static{
					Value: fmt.Sprintf("o5/%s/%s", bb.AppName(), versionTag),
				},
			}
		}
		bb.Template.AddParameter(parameter)
	}
}
