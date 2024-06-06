package deployer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pentops/o5-deploy-aws/cf/app"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

const (
	DefaultO5SidecarImageName = "ghcr.io/pentops/o5-runtime-sidecar"
	DefaultO5SidecarVersion   = "c675f65802d988e345e8fad098335b0bfed6fd70"
)

type ParameterResolver interface {
	ResolveParameter(param *awsdeployer_pb.Parameter) (*awsdeployer_pb.CloudFormationStackParameter, error)
}

func BuildParameterResolver(ctx context.Context, cluster *environment_pb.Cluster, environment *environment_pb.Environment) (*deployerResolver, error) {

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	ecsCluster := cluster.GetEcsCluster()

	hostHeader := "*.*"
	if awsEnv.HostHeader != nil {
		hostHeader = *awsEnv.HostHeader
	}

	crossEnvs := map[string]*environment_pb.AWSLink{}

	for _, envLink := range awsEnv.EnvironmentLinks {
		crossEnvs[envLink.LookupName] = envLink
	}

	sidecarImageName := DefaultO5SidecarImageName
	if ecsCluster.SidecarImageRepo != nil {
		sidecarImageName = *ecsCluster.SidecarImageRepo
	}

	sidecarImageVersion := DefaultO5SidecarVersion
	if ecsCluster.SidecarImageVersion != nil {
		sidecarImageVersion = *ecsCluster.SidecarImageVersion
	}

	sesIdentityCondition := ""
	if awsEnv.SesIdentity != nil {
		// awsEnv.SesConditionsJSON,
		stringLikeCondition := map[string]interface{}{}
		nullCondition := map[string]interface{}{}

		if len(awsEnv.SesIdentity.Recipients) != 1 || awsEnv.SesIdentity.Recipients[0] != "*" {
			stringLikeCondition["ses:Recipients"] = awsEnv.SesIdentity.Recipients
			nullCondition["ses:Recipients"] = false
		}

		if len(awsEnv.SesIdentity.Senders) != 1 || awsEnv.SesIdentity.Senders[0] != "*" {
			stringLikeCondition["ses:FromAddress"] = awsEnv.SesIdentity.Senders
			nullCondition["ses:FromAddress"] = false
		}

		if len(stringLikeCondition) != 0 {
			policyDocument := map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Resource": []interface{}{"*"}, // constrained by conditions
						"Action": []interface{}{
							"ses:SendEmail",
						},
						"Condition": map[string]interface{}{
							"ForAllValues:StringLike": stringLikeCondition,
							"Null":                    nullCondition,
							// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference_policies_condition-single-vs-multi-valued-context-keys.html#reference_policies_condition-multi-valued-context-keys#reference_policies_condition-multi-valued-context-keys
							// "Use caution if you use ForAllValues with an Allow effect
							// because it can be overly permissive if the presence of
							// missing context keys or context keys with empty values in the
							// request context is unexpected. You can include the Null
							// condition operator in your policy with a false value to check
							// if the context key exists and its value is not null. For an
							// example, see Controlling access based on tag keys."
							//
							// I have no idea if this actually applies to SES - DW
						},
					},
				},
			}
			sesIdentityConditionBytes, err := json.Marshal(policyDocument)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal SES conditions: %w", err)
			}
			sesIdentityCondition = string(sesIdentityConditionBytes)
		}
	}

	dr := &deployerResolver{
		wellKnown: map[string]string{
			app.ListenerARNParameter:          ecsCluster.ListenerArn,
			app.HostHeaderParameter:           hostHeader,
			app.ECSClusterParameter:           ecsCluster.EcsClusterName,
			app.ECSRepoParameter:              ecsCluster.EcsRepo,
			app.ECSTaskExecutionRoleParameter: ecsCluster.EcsTaskExecutionRole,
			app.EnvNameParameter:              environment.FullName,
			app.ClusterNameParameter:          cluster.Name,
			app.VPCParameter:                  ecsCluster.VpcId,
			app.MetaDeployAssumeRoleParameter: strings.Join(ecsCluster.O5DeployerGrantRoles, ","),
			app.JWKSParameter:                 strings.Join(environment.TrustJwks, ","),
			app.SNSPrefixParameter:            fmt.Sprintf("arn:aws:sns:%s:%s:%s", ecsCluster.AwsRegion, ecsCluster.AwsAccount, environment.FullName),
			app.S3BucketNamespaceParameter:    ecsCluster.GlobalNamespace,
			app.O5SidecarImageParameter:       fmt.Sprintf("%s:%s", sidecarImageName, sidecarImageVersion),
			app.SESConditionsParameter:        sesIdentityCondition,
			app.CORSOriginParameter:           strings.Join(environment.CorsOrigins, ","),
			app.EventBusARNParameter:          ecsCluster.EventBusArn,
		},
		custom:    environment.Vars,
		crossEnvs: crossEnvs,
	}
	return dr, nil
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

type deployerResolver struct {
	wellKnown map[string]string
	custom    []*environment_pb.CustomVariable
	crossEnvs map[string]*environment_pb.AWSLink
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
