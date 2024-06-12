package app

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
)

type subscriptionPlan struct {
	ingressEndpoints map[string]struct{}

	parameters []*awsdeployer_pb.Parameter

	eventBusSubscriptions []eventBusSubscriptionPlan
	snsSubscriptions      []snsSubscriptionPlan
}

type eventBusSubscriptionPlan struct {
	eventPattern busPattern
	name         string
}

type eventBusRules struct {
	topics        []string
	services      []string
	methods       []serviceAndMethod
	sourceEnvRef  []string
	sourceEnvName string
}

type serviceAndMethod struct {
	service string
	method  string
}

type snsSubscriptionPlan struct {
	topicARN string
	name     string
}

type busPattern struct {
	Detail busDetailPattern `json:"detail"`
}

type busDetailPattern struct {
	// one of:
	Or []singlePattern `json:"$or,omitempty"`
	*singlePattern
}

type singlePattern struct {
	DestinationTopic []string `json:"destinationTopic,omitempty"`
	GRPCService      []string `json:"grpcService,omitempty"`
	GRPCMethod       []string `json:"grpcMethod,omitempty"`
	SourceEnv        []string `json:"sourceEnv,omitempty"`
}

func buildSubscriptionPlan(spec *application_pb.Runtime) (*subscriptionPlan, error) {

	plan := &subscriptionPlan{
		ingressEndpoints: make(map[string]struct{}),
	}

	rulesByEnv := map[string]*eventBusRules{}
	localEnvRules := &eventBusRules{
		sourceEnvName: "local",
		sourceEnvRef:  []string{cloudformation.Ref(EnvNameParameter)},
	}
	rulesByEnv[""] = localEnvRules

	for _, sub := range spec.Subscriptions {

		if sub.TargetContainer == "" {
			sub.TargetContainer = spec.Containers[0].Name
		}
		if sub.Port == 0 {
			sub.Port = 8080
		}

		plan.ingressEndpoints[fmt.Sprintf("%s:%d", sub.TargetContainer, sub.Port)] = struct{}{}

		if strings.HasPrefix(sub.Name, "o5-infra/") {
			topicName := sub.Name[len("o5-infra/"):]
			if !spec.GrantMetaDeployPermissions {
				return nil, fmt.Errorf("o5-infra subscription requires meta deploy permissions")
			}
			snsTopicARN := cloudformation.Join("", []string{
				"arn:aws:sns:",
				cloudformation.Ref("AWS::Region"),
				":",
				cloudformation.Ref("AWS::AccountId"),
				":",
				// Note this is the Cluster name, not the env name, one o5 listener per cluster.
				cloudformation.Ref(ClusterNameParameter),
				"-",
				topicName,
			})
			plan.snsSubscriptions = append(plan.snsSubscriptions, snsSubscriptionPlan{
				topicARN: snsTopicARN,
				name:     topicName,
			})
			continue
		}

		var ruleSet *eventBusRules
		if sub.EnvName == nil {
			ruleSet = localEnvRules
		} else {
			var ok bool
			ruleSet, ok = rulesByEnv[*sub.EnvName]
			if !ok {
				ruleSet = &eventBusRules{}
				rulesByEnv[*sub.EnvName] = ruleSet
			}

			ruleSet.sourceEnvName = *sub.EnvName
			if *sub.EnvName == "*" {
				ruleSet.sourceEnvRef = nil // Don't filter the env.
			} else {
				envParamName := cf.CleanParameterName(*sub.EnvName, "FullName")
				ruleSet.sourceEnvRef = []string{cloudformation.Ref(envParamName)}

				plan.parameters = append(plan.parameters, &awsdeployer_pb.Parameter{
					Name: envParamName,
					Type: "String",
					Source: &awsdeployer_pb.ParameterSourceType{
						Type: &awsdeployer_pb.ParameterSourceType_CrossEnvAttr_{
							CrossEnvAttr: &awsdeployer_pb.ParameterSourceType_CrossEnvAttr{
								EnvName: *sub.EnvName,
								Attr:    awsdeployer_pb.EnvAttr_FULL_NAME,
							},
						},
					},
				})
			}
		}

		topicParts := strings.Split(sub.Name, "/")

		if len(topicParts) == 1 {
			// topic-name
			ruleSet.topics = append(ruleSet.topics, topicParts[0])
		} else if len(topicParts) == 2 {
			// /foo.bar.v1.FooTopic
			ruleSet.services = append(ruleSet.services, topicParts[1])
		} else if len(topicParts) == 3 {
			// /foo.bar.v1.FooTopic/FooMethod
			ruleSet.methods = append(ruleSet.methods, serviceAndMethod{
				service: topicParts[1],
				method:  topicParts[2],
			})
		} else {
			return nil, fmt.Errorf("invalid topic name %s", sub.Name)
		}

	}

	for envGroup, rules := range rulesByEnv {

		rulePatterns := make([]singlePattern, 0)

		if len(rules.topics) > 0 {
			rulePatterns = append(rulePatterns, singlePattern{
				DestinationTopic: rules.topics,
				SourceEnv:        rules.sourceEnvRef,
			})
		}

		if len(rules.services) > 0 {
			rulePatterns = append(rulePatterns, singlePattern{
				GRPCService: rules.services,
				SourceEnv:   rules.sourceEnvRef,
			})
		}

		if len(rules.methods) > 0 {
			methodMatchers := make(map[string]singlePattern, 0)
			for _, method := range rules.methods {
				mm, ok := methodMatchers[method.service]
				if !ok {
					mm = singlePattern{
						GRPCService: []string{method.service},
						GRPCMethod:  []string{},
						SourceEnv:   rules.sourceEnvRef,
					}
				}
				mm.GRPCMethod = append(mm.GRPCMethod, method.method)
				methodMatchers[method.service] = mm
			}
			for _, mm := range methodMatchers {
				rulePatterns = append(rulePatterns, mm)
			}
		}

		if len(rulePatterns) == 0 {
			continue
		}

		var detailPattern busDetailPattern
		if len(rulePatterns) == 1 {
			detailPattern.singlePattern = &rulePatterns[0]
		} else {
			detailPattern.Or = rulePatterns
		}

		plan.eventBusSubscriptions = append(plan.eventBusSubscriptions, eventBusSubscriptionPlan{
			name: envGroup,
			eventPattern: busPattern{
				Detail: detailPattern,
			},
		})

	}

	return plan, nil
}
