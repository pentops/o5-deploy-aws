package awsinfra

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type deferredParameterResolver struct {
	clients         ClientBuilder
	desiredCount    int32
	listenerARN     string
	takenPriorities map[int]bool
}

func newDeferredParameterResolver(clients ClientBuilder, listenerARN string, desiredCount int32) (*deferredParameterResolver, error) {
	return &deferredParameterResolver{
		clients:      clients,
		listenerARN:  listenerARN,
		desiredCount: desiredCount,
	}, nil
}

func (dpr *deferredParameterResolver) getTakenPriorities(ctx context.Context) (map[int]bool, error) {
	if dpr.takenPriorities != nil {
		return dpr.takenPriorities, nil
	}

	if dpr.listenerARN == "" {
		return nil, fmt.Errorf("missing listener ARN - The Stack must have a parameter named %s to automatically resolve rule priorities", app.ListenerARNParameter)
	}

	clients, err := dpr.clients.Clients(ctx)
	if err != nil {
		return nil, err
	}

	takenPriorities := make(map[int]bool)
	var marker *string
	for {
		rules, err := clients.ELB.DescribeRules(ctx, &elbv2.DescribeRulesInput{
			ListenerArn: aws.String(dpr.listenerARN),
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

	dpr.takenPriorities = takenPriorities
	return takenPriorities, nil
}

func (dpr *deferredParameterResolver) nextAvailableListenerRulePriority(ctx context.Context, group application_pb.RouteGroup) (int, error) {
	takenPriorities, err := dpr.getTakenPriorities(ctx)
	if err != nil {
		return 0, err
	}

	a, b, err := groupRangeForPriority(group)
	if err != nil {
		return 0, err
	}
	for i := a; i < b; i++ {
		if _, ok := takenPriorities[i]; !ok {
			takenPriorities[i] = true
			return i, nil
		}
	}

	return 0, errors.New("no available listener rule priorities")
}

func groupRangeForPriority(group application_pb.RouteGroup) (int, int, error) {
	switch group {
	case application_pb.RouteGroup_ROUTE_GROUP_NORMAL,
		application_pb.RouteGroup_ROUTE_GROUP_UNSPECIFIED:
		return 1000, 2000, nil
	case application_pb.RouteGroup_ROUTE_GROUP_FIRST:
		return 1, 1000, nil
	case application_pb.RouteGroup_ROUTE_GROUP_FALLBACK:
		return 2000, 3000, nil
	default:
		return 0, 0, fmt.Errorf("unknown route group %s", group)
	}
}

func validPreviousPriority(group application_pb.RouteGroup, previous *types.Parameter) (string, bool) {
	if previous == nil || previous.ParameterValue == nil {
		return "", false
	}

	previousPriority, err := strconv.Atoi(*previous.ParameterValue)
	if err != nil {
		return "", false // Who knows how, but let's just pretend it didn't happen
	}

	a, b, err := groupRangeForPriority(group)
	if err != nil {
		return "", false
	}

	if previousPriority >= a && previousPriority < b {
		return *previous.ParameterValue, true
	}

	return "", false
}

func (dpr *deferredParameterResolver) Resolve(ctx context.Context, input *deployer_pb.CloudFormationStackParameterType, previous *types.Parameter) (string, error) {
	switch pt := input.Type.(type) {
	case *deployer_pb.CloudFormationStackParameterType_RulePriority_:
		group := pt.RulePriority.RouteGroup
		previous, ok := validPreviousPriority(group, previous)
		if ok {
			return previous, nil
		}

		priority, err := dpr.nextAvailableListenerRulePriority(ctx, group)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d", priority), nil

	case *deployer_pb.CloudFormationStackParameterType_DesiredCount_:
		return fmt.Sprintf("%d", dpr.desiredCount), nil

	default:
		return "", fmt.Errorf("unknown parameter type: %T", pt)

	}

}
