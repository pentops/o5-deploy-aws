package deployer

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type ParameterResolver interface {
	WellKnownParameter(name string) (string, bool)
	CustomEnvVar(name string) (string, bool)
	NextAvailableListenerRulePriority(group application_pb.RouteGroup) (int, error)
	CrossEnvSNSPrefix(envName string) (string, error)
	DesiredCount() int
}

func resolveParameter(ctx context.Context, param *app.Parameter, resolver ParameterResolver) (*types.Parameter, error) {

	parameter := &types.Parameter{
		ParameterKey: aws.String(param.Name),
	}
	switch param.Source {
	case app.ParameterSourceDefault:
		if param.Default == nil {
			return nil, errors.New("default parameter source requires a default value")
		}
		parameter.ParameterValue = aws.String(*param.Default)

	case app.ParameterSourceWellKnown:
		value, ok := resolver.WellKnownParameter(param.Name)
		if !ok {
			return nil, fmt.Errorf("unknown well known parameter: %s", param.Name)
		}
		parameter.ParameterValue = aws.String(value)

	case app.ParameterSourceRulePriority:
		if len(param.Args) != 1 {
			return nil, errors.New("invalid parameter source args")
		}

		group, ok := param.Args[0].(application_pb.RouteGroup)
		if !ok {
			return nil, errors.New("invalid parameter source args")
		}
		priority, err := resolver.NextAvailableListenerRulePriority(group)
		if err != nil {
			return nil, err
		}

		parameter.ParameterValue = aws.String(fmt.Sprintf("%d", priority))

	case app.ParameterSourceDesiredCount:
		parameter.ParameterValue = app.Stringf("%d", resolver.DesiredCount())

	case app.ParameterSourceCrossEnvSNS:
		if len(param.Args) != 1 {
			return nil, errors.New("invalid parameter source args")
		}
		envName, ok := param.Args[0].(string)
		if !ok {
			return nil, errors.New("invalid parameter source args")
		}

		topicPrefix, err := resolver.CrossEnvSNSPrefix(envName)
		if err != nil {
			return nil, err
		}

		parameter.ParameterValue = aws.String(topicPrefix)

	case app.ParameterSourceEnvVar:
		key, ok := param.Args[0].(string)
		if !ok {
			return nil, errors.New("invalid parameter source args")
		}
		val, ok := resolver.CustomEnvVar(key)
		if !ok {
			return nil, fmt.Errorf("unknown env var: %s", key)
		}
		parameter.ParameterValue = aws.String(val)

	default:
		return nil, fmt.Errorf("unknown parameter source (%v) %s", param.Source, param.Name)
	}
	return parameter, nil
}

type deployerResolver struct {
	takenPriorities     map[int]bool
	wellKnown           map[string]string
	custom              []*environment_pb.CustomVariable
	desiredCount        int
	crossEnvSNSPrefixes map[string]string
}

func (dr *deployerResolver) DesiredCount() int {
	return dr.desiredCount
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

func (dr *deployerResolver) NextAvailableListenerRulePriority(group application_pb.RouteGroup) (int, error) {
	var groupRange [2]int

	switch group {
	case application_pb.RouteGroup_ROUTE_GROUP_NORMAL,
		application_pb.RouteGroup_ROUTE_GROUP_UNSPECIFIED:
		groupRange = [2]int{1000, 2000}
	case application_pb.RouteGroup_ROUTE_GROUP_FIRST:
		groupRange = [2]int{1, 1000}
	case application_pb.RouteGroup_ROUTE_GROUP_FALLBACK:
		groupRange = [2]int{2000, 3000}
	default:
		return 0, fmt.Errorf("unknown route group %s", group)
	}

	for i := groupRange[0]; i < groupRange[1]; i++ {
		if _, ok := dr.takenPriorities[i]; !ok {
			dr.takenPriorities[i] = true
			return i, nil
		}
	}

	return 0, errors.New("no available listener rule priorities")
}

func (dr *deployerResolver) WellKnownParameter(name string) (string, bool) {
	val, ok := dr.wellKnown[name]
	return val, ok
}

func (d *Deployer) applyInitialParameters(ctx context.Context, stack stackParameters) ([]types.Parameter, error) {

	mappedPreviousParameters := make(map[string]string, len(stack.previousParameters))
	for _, param := range stack.previousParameters {
		mappedPreviousParameters[*param.ParameterKey] = *param.ParameterValue
	}

	stackParameters := stack.template.Parameters
	parameters := make([]types.Parameter, 0, len(stackParameters))

	takenPriorities, err := d.loadTakenPriorities(ctx)
	if err != nil {
		return nil, err
	}

	hostHeader := "*.*"
	if d.AWS.HostHeader != nil {
		hostHeader = *d.AWS.HostHeader
	}

	crossEnvSNS := map[string]string{}

	for _, envLink := range d.AWS.EnvironmentLinks {
		crossEnvSNS[envLink.FullName] = envLink.SnsPrefix
	}

	dr := &deployerResolver{
		takenPriorities: takenPriorities,
		wellKnown: map[string]string{
			app.ListenerARNParameter:          d.AWS.ListenerArn,
			app.HostHeaderParameter:           hostHeader,
			app.ECSClusterParameter:           d.AWS.EcsClusterName,
			app.ECSRepoParameter:              d.AWS.EcsRepo,
			app.ECSTaskExecutionRoleParameter: d.AWS.EcsTaskExecutionRole,
			app.EnvNameParameter:              d.Environment.FullName,
			app.VPCParameter:                  d.AWS.VpcId,
			app.MetaDeployAssumeRoleParameter: strings.Join(d.AWS.O5DeployerGrantRoles, ","),
		},
		custom:              d.Environment.Vars,
		desiredCount:        stack.scale,
		crossEnvSNSPrefixes: crossEnvSNS,
	}

	for _, param := range stackParameters {
		parameter, err := resolveParameter(ctx, param, dr)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s': %w", param.Name, err)
		}
		parameters = append(parameters, *parameter)
	}

	return parameters, nil
}

func (d *Deployer) loadTakenPriorities(ctx context.Context) (map[int]bool, error) {
	clients, err := d.Clients(ctx)
	if err != nil {
		return nil, err
	}

	elbClient := clients.ELB

	takenPriorities := make(map[int]bool)
	var marker *string
	for {
		rules, err := elbClient.DescribeRules(ctx, &elbv2.DescribeRulesInput{
			ListenerArn: aws.String(d.AWS.ListenerArn),
			Marker:      marker,
		})
		if err != nil {
			return nil, err
		}

		for _, rule := range rules.Rules {
			if *rule.Priority == "default" {
				continue
			}
			priority, err := strconv.Atoi(*rule.Priority)
			if err != nil {
				return nil, err
			}

			takenPriorities[priority] = true
		}

		marker = rules.NextMarker
		if marker == nil {
			break
		}
	}
	return takenPriorities, nil
}
