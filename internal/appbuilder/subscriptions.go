package appbuilder

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/sns"
	"github.com/awslabs/goformation/v7/cloudformation/sqs"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type subscriptionPlan struct {
	targetContainers map[string]struct{}
	namePrefix       string

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
	globals       []string
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

// patterns to match o5.messaging.v1.Message
type singlePattern struct {
	DestinationTopic []string `json:"destinationTopic,omitempty"`
	GRPCService      []string `json:"grpcService,omitempty"`
	GRPCMethod       []string `json:"grpcMethod,omitempty"`
	SourceEnv        []string `json:"sourceEnv,omitempty"`

	// Message Extensions
	Reply   any `json:"reply,omitempty"`
	Request any `json:"request,omitempty"`
	Upsert  any `json:"upsert,omitempty"`
	Event   any `json:"event,omitempty"`
}

func buildSubscriptionPlan(appName string, spec *application_pb.Runtime) (*subscriptionPlan, error) {
	plan := &subscriptionPlan{
		namePrefix:       fmt.Sprintf("%s_%s", appName, spec.Name),
		targetContainers: make(map[string]struct{}),
	}

	rulesByEnv := map[string]*eventBusRules{}
	localEnvRules := &eventBusRules{
		sourceEnvName: "local",

		// The name of the environment to subscribe to, matches
		sourceEnvRef: []string{cloudformation.Ref(EnvNameParameter)},

		// The name of the environment to subscribe to, matches
	}
	rulesByEnv[""] = localEnvRules

	for _, sub := range spec.Subscriptions {
		if strings.HasPrefix(sub.Name, "o5-infra/") {
			topicName := sub.Name[len("o5-infra/"):]
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
				envParamName := cflib.CleanParameterName(*sub.EnvName, "FullName")
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
		switch {
		case len(topicParts) == 1:
			// {topic-name}
			ruleSet.topics = append(ruleSet.topics, topicParts[0])

		case len(topicParts) == 2 && topicParts[0] == "global":
			// global/{type}
			ruleSet.globals = append(ruleSet.globals, topicParts[1])

		case len(topicParts) == 2 && topicParts[0] == "":
			// /{package}.{topic}
			ruleSet.services = append(ruleSet.services, topicParts[1])

		case len(topicParts) == 3 && topicParts[0] == "":
			// /{package}.{topic}/{method}
			ruleSet.methods = append(ruleSet.methods, serviceAndMethod{
				service: topicParts[1],
				method:  topicParts[2],
			})

		default:
			return nil, fmt.Errorf("invalid topic name %s", sub.Name)
		}

	}

	fullNameRef := cloudformation.Join("/", []string{
		cloudformation.Ref(EnvNameParameter),
		appName,
	})

	replyTo := map[string]any{
		"replyTo": []any{
			map[string]any{"exists": false},
			fullNameRef,
		},
	}

	for envGroup, rules := range rulesByEnv {
		rulePatterns := make([]singlePattern, 0)

		if len(rules.globals) > 0 {
			for _, global := range rules.globals {
				switch global {
				case "upsert":
					rulePatterns = append(rulePatterns, singlePattern{
						SourceEnv: rules.sourceEnvRef,
						Upsert: map[string]any{
							"entityName": []any{map[string]any{
								"exists": true,
							}},
						},
					})
				case "event":
					rulePatterns = append(rulePatterns, singlePattern{
						SourceEnv: rules.sourceEnvRef,
						Event: map[string]any{
							"entityName": []any{map[string]any{
								"exists": true,
							}},
						},
					})
				default:
					return nil, fmt.Errorf("invalid global topic %s", global)

				}
			}
		}

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
				Reply:       replyTo,
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
						Reply:       replyTo,
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
			pattern := &rulePatterns[0]
			detailPattern.singlePattern = pattern
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

func (sp *subscriptionPlan) AddToTemplate(template *cflib.TemplateBuilder, queueResource *cflib.Resource[*sqs.Queue]) error {
	namePrefix := sp.namePrefix

	for _, param := range sp.parameters {
		template.AddParameter(param)
	}

	topicARNs := []string{}
	for _, sub := range sp.snsSubscriptions {
		subscription := cflib.NewResource(cflib.CleanParameterName(namePrefix, sub.name), &sns.Subscription{
			TopicArn:           sub.topicARN,
			Protocol:           "sqs",
			RawMessageDelivery: cflib.Bool(false), // Always include the SNS header info for infra events.
			Endpoint:           queueResource.GetAtt("Arn").RefPtr(),
		})

		// The topic is not added to the stack, it should already exist
		// in this case.
		template.AddResource(subscription)
		topicARNs = append(topicARNs, sub.topicARN)
	}

	for _, sub := range sp.eventBusSubscriptions {
		eventBusSubscription := &events.Rule{
			Description:  cflib.String(fmt.Sprintf("Subscription for app %s %s", namePrefix, sub.name)),
			EventBusName: cloudformation.RefPtr(EventBusARNParameter),
			Targets: []events.Rule_Target{{
				Arn: queueResource.GetAtt("Arn").Ref(),
				Id:  "SQS",
			}},
			EventPattern: sub.eventPattern,
		}
		template.AddResource(cflib.NewResource(cflib.CleanParameterName(namePrefix, "subscription", sub.name), eventBusSubscription))
	}

	queuePolicyStatement := []any{
		map[string]any{
			"Effect": "Allow",
			"Principal": map[string]any{
				"Service": "events.amazonaws.com",
			},
			"Action":   "sqs:SendMessage",
			"Resource": queueResource.GetAtt("Arn"),
		}}

	if len(topicARNs) > 0 {
		queuePolicyStatement = append(queuePolicyStatement, map[string]any{
			"Effect":    "Allow",
			"Principal": "*",
			"Action":    "sqs:SendMessage",
			"Resource":  queueResource.GetAtt("Arn"),
			"Condition": map[string]any{
				"ArnEquals": map[string]any{
					"aws:SourceArn": topicARNs,
				},
			},
		})
	}

	// Allow SNS and EventBridge to publish to SQS...
	// (The ARN distinguishes the source)
	template.AddResource(cflib.NewResource(cflib.CleanParameterName(namePrefix), &sqs.QueuePolicy{
		Queues: []string{queueResource.Ref().Ref()},
		PolicyDocument: map[string]any{
			"Version":   "2012-10-17",
			"Statement": queuePolicyStatement,
		},
	}))

	return nil
}
