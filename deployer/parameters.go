package deployer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type ParameterResolver interface {
	ResolveParameter(param *deployer_pb.Parameter) (*deployer_pb.CloudFormationStackParameter, error)
}

func BuildParameterResolver(ctx context.Context, environment *environment_pb.Environment) (*deployerResolver, error) {

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	hostHeader := "*.*"
	if awsEnv.HostHeader != nil {
		hostHeader = *awsEnv.HostHeader
	}

	crossEnvSNS := map[string]string{}

	for _, envLink := range awsEnv.EnvironmentLinks {
		crossEnvSNS[envLink.FullName] = envLink.SnsPrefix
	}

	dr := &deployerResolver{
		wellKnown: map[string]string{
			app.ListenerARNParameter:          awsEnv.ListenerArn,
			app.HostHeaderParameter:           hostHeader,
			app.ECSClusterParameter:           awsEnv.EcsClusterName,
			app.ECSRepoParameter:              awsEnv.EcsRepo,
			app.ECSTaskExecutionRoleParameter: awsEnv.EcsTaskExecutionRole,
			app.EnvNameParameter:              environment.FullName,
			app.VPCParameter:                  awsEnv.VpcId,
			app.MetaDeployAssumeRoleParameter: strings.Join(awsEnv.O5DeployerGrantRoles, ","),
			app.JWKSParameter:                 strings.Join(environment.TrustJwks, ","),
			app.SNSPrefixParameter:            awsEnv.SnsPrefix,
		},
		custom:              environment.Vars,
		crossEnvSNSPrefixes: crossEnvSNS,
	}
	return dr, nil
}

func (rr *deployerResolver) ResolveParameter(param *deployer_pb.Parameter) (*deployer_pb.CloudFormationStackParameter, error) {

	switch ps := param.Source.Type.(type) {
	case *deployer_pb.ParameterSourceType_RulePriority_:

		return &deployer_pb.CloudFormationStackParameter{
			Name: param.Name,
			Source: &deployer_pb.CloudFormationStackParameter_Resolve{
				Resolve: &deployer_pb.CloudFormationStackParameterType{
					Type: &deployer_pb.CloudFormationStackParameterType_RulePriority_{
						RulePriority: &deployer_pb.CloudFormationStackParameterType_RulePriority{
							RouteGroup: ps.RulePriority.RouteGroup,
						},
					},
				},
			},
		}, nil

	case *deployer_pb.ParameterSourceType_DesiredCount_:
		return &deployer_pb.CloudFormationStackParameter{
			Name: param.Name,
			Source: &deployer_pb.CloudFormationStackParameter_Resolve{
				Resolve: &deployer_pb.CloudFormationStackParameterType{
					Type: &deployer_pb.CloudFormationStackParameterType_DesiredCount_{
						DesiredCount: &deployer_pb.CloudFormationStackParameterType_DesiredCount{},
					},
				},
			},
		}, nil
	}

	var value string
	switch ps := param.Source.Type.(type) {
	case *deployer_pb.ParameterSourceType_Static_:
		value = ps.Static.Value

	case *deployer_pb.ParameterSourceType_WellKnown_:
		val, ok := rr.WellKnownParameter(param.Name)
		if !ok {
			return nil, fmt.Errorf("unknown well known parameter: %s", param.Name)
		}
		value = val

	case *deployer_pb.ParameterSourceType_CrossEnvSns_:
		envName := ps.CrossEnvSns.EnvName
		topicPrefix, err := rr.CrossEnvSNSPrefix(envName)
		if err != nil {
			return nil, err
		}

		value = topicPrefix

	case *deployer_pb.ParameterSourceType_EnvVar_:
		key := ps.EnvVar.Name
		val, ok := rr.CustomEnvVar(key)
		if !ok {
			return nil, fmt.Errorf("unknown env var: %s", key)
		}
		value = val

	default:
		return nil, fmt.Errorf("unknown parameter source (%v) %s", param.Source, param.Name)
	}

	return &deployer_pb.CloudFormationStackParameter{
		Name: param.Name,
		Source: &deployer_pb.CloudFormationStackParameter_Value{
			Value: value,
		},
	}, nil

}

type deployerResolver struct {
	wellKnown           map[string]string
	custom              []*environment_pb.CustomVariable
	crossEnvSNSPrefixes map[string]string
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

func (dr *deployerResolver) CrossEnvSNSPrefix(envName string) (string, error) {
	if prefix, ok := dr.crossEnvSNSPrefixes[envName]; ok {
		return prefix, nil
	}
	return "", fmt.Errorf("unknown env for SNS prefix: %s", envName)
}

func (dr *deployerResolver) WellKnownParameter(name string) (string, bool) {
	val, ok := dr.wellKnown[name]
	return val, ok
}
