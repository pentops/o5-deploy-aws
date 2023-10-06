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
)

func (d *Deployer) applyInitialParameters(ctx context.Context, stack stackParameters) ([]types.Parameter, error) {

	mappedPreviousParameters := make(map[string]string, len(stack.previousParameters))
	for _, param := range stack.previousParameters {
		mappedPreviousParameters[*param.ParameterKey] = *param.ParameterValue
	}

	stackParameters := stack.template.Template().Parameters
	parameters := make([]types.Parameter, 0, len(stackParameters))

	for key, param := range stackParameters {
		if param.Default != nil {
			continue
		}
		parameter := types.Parameter{
			ParameterKey: aws.String(key),
		}
		switch {
		case key == app.ListenerARNParameter:
			parameter.ParameterValue = aws.String(d.AWS.ListenerArn)

		case key == app.HostHeaderParameter:
			hh := "*.*"
			if d.AWS.HostHeader != nil {
				hh = *d.AWS.HostHeader
			}
			parameter.ParameterValue = aws.String(hh)

		case key == app.ECSClusterParameter:
			parameter.ParameterValue = aws.String(d.AWS.EcsClusterName)

		case key == app.ECSRepoParameter:
			parameter.ParameterValue = aws.String(d.AWS.EcsRepo)

		case key == app.ECSTaskExecutionRoleParameter:
			parameter.ParameterValue = aws.String(d.AWS.EcsTaskExecutionRole)

		case key == app.EnvNameParameter:
			parameter.ParameterValue = aws.String(d.Environment.FullName)

		case key == app.VPCParameter:
			parameter.ParameterValue = aws.String(d.AWS.VpcId)

		case strings.HasPrefix(key, "ListenerRulePriority"):
			group := application_pb.RouteGroup_ROUTE_GROUP_NORMAL
			if strings.HasPrefix(key, "ListenerRulePriorityFirst") {
				group = application_pb.RouteGroup_ROUTE_GROUP_FIRST
			} else if strings.HasPrefix(key, "ListenerRulePriorityFallback") {
				group = application_pb.RouteGroup_ROUTE_GROUP_FALLBACK
			}
			existingPriority, ok := mappedPreviousParameters[key]
			if ok {
				parameter.ParameterValue = aws.String(existingPriority)
			} else {
				priority, err := d.nextAvailableListenerRulePriority(ctx, group)
				if err != nil {
					return nil, err
				}

				parameter.ParameterValue = aws.String(fmt.Sprintf("%d", priority))
			}

		case strings.HasPrefix(key, "DesiredCount"):
			parameter.ParameterValue = app.Stringf("%d", stack.scale)

		default:
			return nil, fmt.Errorf("unknown parameter %s", key)
		}
		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

func (d *Deployer) nextAvailableListenerRulePriority(ctx context.Context, group application_pb.RouteGroup) (int, error) {
	if d.takenPriorities == nil {
		d.takenPriorities = make(map[int]bool)
		var marker *string
		for {
			rules, err := d.ELB.DescribeRules(ctx, &elbv2.DescribeRulesInput{
				ListenerArn: aws.String(d.AWS.ListenerArn),
				Marker:      marker,
			})
			if err != nil {
				return 0, err
			}

			for _, rule := range rules.Rules {
				if *rule.Priority == "default" {
					continue
				}
				priority, err := strconv.Atoi(*rule.Priority)
				if err != nil {
					return 0, err
				}

				d.takenPriorities[priority] = true
			}

			marker = rules.NextMarker
			if marker == nil {
				break
			}
		}
	}

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
		if _, ok := d.takenPriorities[i]; !ok {
			d.takenPriorities[i] = true
			return i, nil
		}
	}

	return 0, errors.New("no available listener rule priorities")
}
