package app

import (
	"crypto/sha1"
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type ListenerRuleSet struct {
	Rules []*Resource[*elbv2.ListenerRule]
}

func NewListenerRuleSet() *ListenerRuleSet {
	return &ListenerRuleSet{
		Rules: []*Resource[*elbv2.ListenerRule]{},
	}
}

func (ll *ListenerRuleSet) AddRoute(targetGroup *Resource[*elbv2.TargetGroup], route *application_pb.Route) (*Resource[*elbv2.ListenerRule], error) {
	hash := sha1.New()
	hash.Write([]byte(route.Prefix))
	pathClean := fmt.Sprintf("%x", hash.Sum(nil))

	name := fmt.Sprintf("ListenerRule%s", pathClean)
	groupString := ""
	switch route.RouteGroup {
	case application_pb.RouteGroup_ROUTE_GROUP_NORMAL,
		application_pb.RouteGroup_ROUTE_GROUP_UNSPECIFIED:
		groupString = ""
	case application_pb.RouteGroup_ROUTE_GROUP_FIRST:
		groupString = "First"
	case application_pb.RouteGroup_ROUTE_GROUP_FALLBACK:
		groupString = "Fallback"
	default:
		return nil, fmt.Errorf("unknown route group %v", route.RouteGroup)
	}

	priority := fmt.Sprintf("ListenerRulePriority%s%s", groupString, pathClean)
	rule := &elbv2.ListenerRule{
		//Priority:    cloudformation.Ref(priority),
		ListenerArn: cloudformation.RefPtr(ListenerARNParameter),
		Actions: []elbv2.ListenerRule_Action{{
			Type: "forward",
			ForwardConfig: &elbv2.ListenerRule_ForwardConfig{
				TargetGroups: []elbv2.ListenerRule_TargetGroupTuple{{
					TargetGroupArn: String(targetGroup.Ref()),
				}},
			},
		}},
		Conditions: []elbv2.ListenerRule_RuleCondition{{
			Field:  String("path-pattern"),
			Values: []string{fmt.Sprintf("%s*", route.Prefix)},
		}, {
			Field:  String("host-header"),
			Values: []string{cloudformation.Ref(HostHeaderParameter)},
		}},
	}

	resource := &Resource[*elbv2.ListenerRule]{
		// TODO: This should use the constructor
		name:     name,
		Resource: rule,
		Overrides: map[string]string{
			"Priority": cloudformation.Ref(priority),
		},
		parameters: []*deployer_pb.Parameter{{
			Name:        priority,
			Type:        "Number",
			Description: fmt.Sprintf(":o5:priority:%s", route.Prefix),
			Source: &deployer_pb.ParameterSourceType{
				Type: &deployer_pb.ParameterSourceType_RulePriority_{
					RulePriority: &deployer_pb.ParameterSourceType_RulePriority{
						RouteGroup: route.RouteGroup,
					},
				},
			},
		}},
	}

	ll.Rules = append(ll.Rules, resource)
	return resource, nil
}
