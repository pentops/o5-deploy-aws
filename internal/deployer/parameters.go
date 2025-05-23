package deployer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
)

const (
	DefaultO5SidecarImageName = "ghcr.io/pentops/o5-runtime-sidecar"
)

type ParameterResolver interface {
	ResolveParameter(param *awsdeployer_pb.Parameter) (*awsdeployer_pb.CloudFormationStackParameter, error)
}

type parameterInput struct {
	cluster     *environment_pb.Cluster
	dbDeps      []*awsdeployer_pb.PostgresSpec
	environment *environment_pb.Environment
}

func buildParameterResolver(input parameterInput) (*deployerResolver, error) {

	awsEnv := input.environment.GetAws()
	if awsEnv == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	ecsCluster := input.cluster.GetAws()
	if ecsCluster == nil {
		return nil, errors.New("AWS Deployer requires the type of cluster provider to be AWS")
	}

	namedPolicies := map[string]string{}
	for _, policy := range awsEnv.IamPolicies {
		namedPolicies[policy.Name] = policy.PolicyArn
	}

	crossEnvs := map[string]*environment_pb.AWSLink{}
	for _, envLink := range awsEnv.EnvironmentLinks {
		crossEnvs[envLink.LookupName] = envLink
	}

	databaseHosts := map[string]*environment_pb.RDSHost{}
	for _, host := range ecsCluster.RdsHosts {
		databaseHosts[host.ServerGroupName] = host
	}

	postgresSpecs := map[string]*awsdeployer_pb.PostgresSpec{}
	for _, dbDep := range input.dbDeps {
		postgresSpecs[dbDep.AppKey] = dbDep
	}

	wellKnown, err := buildWellKnown(input.cluster, input.environment)
	if err != nil {
		return nil, err
	}

	dr := &deployerResolver{
		wellKnown:     wellKnown,
		custom:        input.environment.Vars,
		crossEnvs:     crossEnvs,
		namedPolicies: namedPolicies,
		databaseHosts: databaseHosts,
		postgresSpecs: postgresSpecs,
	}
	return dr, nil
}

func buildWellKnown(cluster *environment_pb.Cluster, environment *environment_pb.Environment) (map[string]string, error) {

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	ecsCluster := cluster.GetAws()
	if ecsCluster == nil {
		return nil, errors.New("AWS Deployer requires the type of cluster provider to be AWS")
	}

	hostHeader := "*.*"
	if awsEnv.HostHeader != nil {
		hostHeader = *awsEnv.HostHeader
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

	wellKnown := map[string]string{
		appbuilder.ListenerARNParameter:          ecsCluster.AlbIngress.ListenerArn,
		appbuilder.HostHeaderParameter:           hostHeader,
		appbuilder.ECSClusterParameter:           ecsCluster.EcsCluster.ClusterName,
		appbuilder.ECSRepoParameter:              ecsCluster.EcsCluster.DefaultRegistry,
		appbuilder.ECSTaskExecutionRoleParameter: ecsCluster.EcsCluster.TaskExecutionRole,
		appbuilder.VPCParameter:                  ecsCluster.VpcId,
		appbuilder.MetaDeployAssumeRoleParameter: ecsCluster.O5Deployer.AssumeRoleArn,
		appbuilder.SNSPrefixParameter:            fmt.Sprintf("arn:aws:sns:%s:%s:%s", ecsCluster.AwsRegion, ecsCluster.AwsAccount, environment.FullName),
		appbuilder.S3BucketNamespaceParameter:    ecsCluster.GlobalNamespace,
		appbuilder.O5SidecarImageParameter:       sidecarFullImage,

		appbuilder.LoadBalancerSecurityGroup: ecsCluster.AlbIngress.SecurityGroupId,
		appbuilder.SubnetIDsParameter:        strings.Join(ecsCluster.EcsCluster.SubnetIds, ","),
		appbuilder.EventBusARNParameter:      ecsCluster.EventBridge.EventBusArn,
		appbuilder.JWKSParameter:             strings.Join(environment.TrustJwks, ","),
		appbuilder.CORSOriginParameter:       strings.Join(environment.CorsOrigins, ","),
		appbuilder.ClusterNameParameter:      cluster.Name,
		appbuilder.EnvNameParameter:          environment.FullName,
	}
	return wellKnown, nil

}

type deployerResolver struct {
	wellKnown     map[string]string
	namedPolicies map[string]string
	custom        []*environment_pb.CustomVariable
	crossEnvs     map[string]*environment_pb.AWSLink
	postgresSpecs map[string]*awsdeployer_pb.PostgresSpec
	databaseHosts map[string]*environment_pb.RDSHost
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

	case *awsdeployer_pb.ParameterSourceType_Aurora_:
		spec, ok := rr.postgresSpecs[ps.Aurora.AppKey]
		if !ok {
			return nil, fmt.Errorf("unknown postgres DB (aurora): %s", ps.Aurora.AppKey)
		}
		aurora := spec.AppConnection.GetAurora()
		if aurora == nil {
			return nil, fmt.Errorf("database %s is not an aurora database but is used in parameter %s", ps.Aurora.AppKey, param.Name)
		}

		host := aurora.Conn
		switch ps.Aurora.Part {
		case awsdeployer_pb.ParameterSourceType_Aurora_PART_JSON:

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
		case awsdeployer_pb.ParameterSourceType_Aurora_PART_ENDPOINT:
			value = host.Endpoint

		case awsdeployer_pb.ParameterSourceType_Aurora_PART_IDENTIFIER:
			value = host.Identifier

		case awsdeployer_pb.ParameterSourceType_Aurora_PART_DBNAME:
			value = host.DbName

		default:
			return nil, fmt.Errorf("unknown aurora part: %v", ps.Aurora.Part)
		}

	case *awsdeployer_pb.ParameterSourceType_DatabaseServer_:
		host, ok := rr.databaseHosts[ps.DatabaseServer.ServerGroup]
		if !ok {
			return nil, fmt.Errorf("unknown aurora server group: %s", ps.DatabaseServer.ServerGroup)
		}

		switch ps.DatabaseServer.Part {
		case awsdeployer_pb.ParameterSourceType_DatabaseServer_PART_CLIENT_SECURITY_GROUP:
			value = host.ClientSecurityGroupId
		default:
			return nil, fmt.Errorf("unknown database part: %v", ps.DatabaseServer.Part)
		}

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
