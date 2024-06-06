package app

import (
	"crypto/sha1"
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

type ListenerRuleSet struct {
	Rules []*cf.Resource[*elbv2.ListenerRule]
}

func NewListenerRuleSet() *ListenerRuleSet {
	return &ListenerRuleSet{
		Rules: []*cf.Resource[*elbv2.ListenerRule]{},
	}
}

func (ll *ListenerRuleSet) AddRoute(targetGroup *cf.Resource[*elbv2.TargetGroup], route *application_pb.Route) (*cf.Resource[*elbv2.ListenerRule], error) {
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
					TargetGroupArn: cf.String(targetGroup.Ref()),
				}},
			},
		}},
		Conditions: []elbv2.ListenerRule_RuleCondition{{
			Field:  cf.String("path-pattern"),
			Values: []string{fmt.Sprintf("%s*", route.Prefix)},
		}},
	}

	if len(route.Subdomains) > 0 {
		// TODO: Split into groups of 3 - ALB can only handle 3 hosts per
		// condition.
		if len(route.Subdomains) > 3 {
			return nil, fmt.Errorf("too many subdomains for route %s, max 3", route.Prefix)
		}
		condition := elbv2.ListenerRule_RuleCondition{
			Field:  cf.String("host-header"),
			Values: []string{},
		}
		for _, subdomain := range route.Subdomains {
			condition.Values = append(condition.Values, cloudformation.Join(".", []string{subdomain, cloudformation.Ref(HostHeaderParameter)}))
		}
		rule.Conditions = append(rule.Conditions, condition)

	} else {
		rule.Conditions = append(rule.Conditions, elbv2.ListenerRule_RuleCondition{
			Field:  cf.String("host-header"),
			Values: []string{cloudformation.Ref(HostHeaderParameter)},
		})
	}

	resource := cf.NewResource(cf.QualifiedName(name), rule)
	resource.Override("Priority", cloudformation.Ref(priority))
	resource.AddParameter(&awsdeployer_pb.Parameter{
		Name:        priority,
		Type:        "Number",
		Description: fmt.Sprintf(":o5:priority:%s", route.Prefix),
		Source: &awsdeployer_pb.ParameterSourceType{
			Type: &awsdeployer_pb.ParameterSourceType_RulePriority_{
				RulePriority: &awsdeployer_pb.ParameterSourceType_RulePriority{
					RouteGroup: route.RouteGroup,
				},
			},
		},
	})

	ll.Rules = append(ll.Rules, resource)
	return resource, nil
}
