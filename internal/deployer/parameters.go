package deployer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
)

const (
	DefaultO5SidecarImageName = "ghcr.io/pentops/o5-runtime-sidecar"
)

type ParameterResolver interface {
	ResolveParameter(param *awsdeployer_pb.Parameter) (*awsdeployer_pb.CloudFormationStackParameter, error)
}

func BuildParameterResolver(ctx context.Context, cluster *environment_pb.Cluster, environment *environment_pb.Environment, pgSpecs []*awsdeployer_pb.PostgresSpec) (*deployerResolver, error) {

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	ecsCluster := cluster.GetAws()

	hostHeader := "*.*"
	if awsEnv.HostHeader != nil {
		hostHeader = *awsEnv.HostHeader
	}

	crossEnvs := map[string]*environment_pb.AWSLink{}

	for _, envLink := range awsEnv.EnvironmentLinks {
		crossEnvs[envLink.LookupName] = envLink
	}

	sidecarImageName := DefaultO5SidecarImageName
	if ecsCluster.O5Sidecar == nil {
		return nil, errors.New("O5Sidecar is required for AWS Deployer")
	}
	if ecsCluster.O5Sidecar.ImageRepo != nil {
		sidecarImageName = *ecsCluster.O5Sidecar.ImageRepo
	}
	sidecarImageVersion := ecsCluster.O5Sidecar.ImageVersion

	sidecarFullImage := fmt.Sprintf("%s:%s", sidecarImageName, sidecarImageVersion)

	namedPolicies := map[string]string{}
	for _, policy := range awsEnv.IamPolicies {
		namedPolicies[policy.Name] = policy.PolicyArn
	}

	auroraHosts := map[string]*awsinfra_pb.AuroraConnection{}
	for _, rdsHost := range pgSpecs {
		aurora := rdsHost.AppConnection.GetAurora()
		if aurora != nil {
			auroraHosts[rdsHost.DbName] = aurora.Conn
		}
	}

	dr := &deployerResolver{
		wellKnown: map[string]string{
			appbuilder.ListenerARNParameter:          ecsCluster.AlbIngress.ListenerArn,
			appbuilder.HostHeaderParameter:           hostHeader,
			appbuilder.ECSClusterParameter:           ecsCluster.EcsCluster.ClusterName,
			appbuilder.ECSRepoParameter:              ecsCluster.EcsCluster.DefaultRegistry,
			appbuilder.ECSTaskExecutionRoleParameter: ecsCluster.EcsCluster.TaskExecutionRole,
			appbuilder.EnvNameParameter:              environment.FullName,
			appbuilder.ClusterNameParameter:          cluster.Name,
			appbuilder.VPCParameter:                  ecsCluster.VpcId,
			appbuilder.MetaDeployAssumeRoleParameter: ecsCluster.O5Deployer.AssumeRoleArn,
			appbuilder.JWKSParameter:                 strings.Join(environment.TrustJwks, ","),
			appbuilder.SNSPrefixParameter:            fmt.Sprintf("arn:aws:sns:%s:%s:%s", ecsCluster.AwsRegion, ecsCluster.AwsAccount, environment.FullName),
			appbuilder.S3BucketNamespaceParameter:    ecsCluster.GlobalNamespace,
			appbuilder.O5SidecarImageParameter:       sidecarFullImage,
			appbuilder.CORSOriginParameter:           strings.Join(environment.CorsOrigins, ","),
			appbuilder.EventBusARNParameter:          ecsCluster.EventBridge.EventBusArn,
		},
		custom:        environment.Vars,
		crossEnvs:     crossEnvs,
		namedPolicies: namedPolicies,

		auroraHosts: auroraHosts,
	}
	return dr, nil
}

type deployerResolver struct {
	wellKnown     map[string]string
	namedPolicies map[string]string
	custom        []*environment_pb.CustomVariable
	crossEnvs     map[string]*environment_pb.AWSLink
	auroraHosts   map[string]*awsinfra_pb.AuroraConnection
}

func (rr *deployerResolver) ResolveParameter(param *awsdeployer_pb.Parameter) (*awsdeployer_pb.CloudFormationStackParameter, error) {

	switch ps := param.Source.Type.(type) {
	case *awsdeployer_pb.ParameterSourceType_RulePriority_:

		return &awsdeployer_pb.CloudFormationStackParameter{
			Name: param.Name,
			Source: &awsdeployer_pb.CloudFormationStackParameter_Resolve{
				Resolve: &awsdeployer_pb.CloudFormationStackParameterType{
					Type: &awsdeployer_pb.CloudFormationStackParameterType_RulePriority_{
						RulePriority: &awsdeployer_pb.CloudFormationStackParameterType_RulePriority{
							RouteGroup: ps.RulePriority.RouteGroup,
						},
					},
				},
			},
		}, nil

	case *awsdeployer_pb.ParameterSourceType_DesiredCount_:
		return &awsdeployer_pb.CloudFormationStackParameter{
			Name: param.Name,
			Source: &awsdeployer_pb.CloudFormationStackParameter_Resolve{
				Resolve: &awsdeployer_pb.CloudFormationStackParameterType{
					Type: &awsdeployer_pb.CloudFormationStackParameterType_DesiredCount_{
						DesiredCount: &awsdeployer_pb.CloudFormationStackParameterType_DesiredCount{},
					},
				},
			},
		}, nil
	}

	var value string
	switch ps := param.Source.Type.(type) {
	case *awsdeployer_pb.ParameterSourceType_Static_:
		value = ps.Static.Value

	case *awsdeployer_pb.ParameterSourceType_WellKnown_:
		val, ok := rr.WellKnownParameter(param.Name)
		if !ok {
			return nil, fmt.Errorf("unknown well known parameter: %s", param.Name)
		}
		value = val

	case *awsdeployer_pb.ParameterSourceType_CrossEnvAttr_:
		val, err := rr.CrossEnvAttr(ps.CrossEnvAttr.EnvName, ps.CrossEnvAttr.Attr)
		if err != nil {
			return nil, err
		}

		value = val

	case *awsdeployer_pb.ParameterSourceType_EnvVar_:
		key := ps.EnvVar.Name
		val, ok := rr.CustomEnvVar(key)
		if !ok {
			return nil, fmt.Errorf("unknown env var: %s", key)
		}
		value = val

	case *awsdeployer_pb.ParameterSourceType_NamedIamPolicy:
		name := ps.NamedIamPolicy.Name
		arn, ok := rr.namedPolicies[name]
		if !ok {
			return nil, fmt.Errorf("unknown named policy: %s", name)
		}
		value = arn

	case *awsdeployer_pb.ParameterSourceType_AuroraEndpoint_:
		host, ok := rr.auroraHosts[ps.AuroraEndpoint.DbName]
		if !ok {
			return nil, fmt.Errorf("unknown aurora server server group: %s", ps.AuroraEndpoint.ServerGroup)
		}

		if host.Port == 0 {
			host.Port = 5432
		}
		jsonValue, err := json.Marshal(&AuroraIAMParameterValue{
			Endpoint: fmt.Sprintf("%s:%d", host.Endpoint, host.Port),
			DbName:   host.DbName,
			DbUser:   host.DbUser,
		})
		if err != nil {
			return nil, err
		}

		value = string(jsonValue)

	default:
		return nil, fmt.Errorf("unknown parameter source (%v) %s", param.Source, param.Name)
	}

	return &awsdeployer_pb.CloudFormationStackParameter{
		Name: param.Name,
		Source: &awsdeployer_pb.CloudFormationStackParameter_Value{
			Value: value,
		},
	}, nil

}

type AuroraIAMParameterValue struct {
	Endpoint string `json:"endpoint"` // Address and Port
	DbName   string `json:"dbName"`
	DbUser   string `json:"dbUser"`
}

func (dr *deployerResolver) CustomEnvVar(name string) (string, bool) {
	for _, envVar := range dr.custom {
		if envVar.Name == name {
			var val string
			switch sourceType := envVar.Src.(type) {
			case *environment_pb.CustomVariable_Value:
				val = sourceType.Value
			case *environment_pb.CustomVariable_Join_:
				val = strings.Join(sourceType.Join.Values, sourceType.Join.Delimiter)
			}
			return val, true
		}
	}
	return "", false
}

func (dr *deployerResolver) CrossEnvAttr(envName string, attr awsdeployer_pb.EnvAttr) (string, error) {
	env, ok := dr.crossEnvs[envName]
	if !ok {
		return "", fmt.Errorf("unknown env: %s", envName)
	}

	switch attr {
	case awsdeployer_pb.EnvAttr_SNS_PREFIX:
		return env.SnsPrefix, nil
	case awsdeployer_pb.EnvAttr_FULL_NAME:
		return env.FullName, nil
	default:
		return "", fmt.Errorf("unknown attr: %v", attr)

	}
}

func (dr *deployerResolver) WellKnownParameter(name string) (string, bool) {
	val, ok := dr.wellKnown[name]
	return val, ok
}
