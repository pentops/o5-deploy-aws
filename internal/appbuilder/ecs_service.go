package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v7/cloudformation/sqs"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type RuntimeService struct {
	Prefix string
	Name   string
	spec   *application_pb.Runtime

	subscriptionPlan *subscriptionPlan

	TargetGroups map[string]*targetGroup

	TaskDefinition *ECSTaskDefinition
}

type targetGroup struct {
	attachment loadBalancerAttachment
	resource   *cflib.Resource[*elbv2.TargetGroup]
	rules      []*cflib.Resource[*elbv2.ListenerRule]
}

type loadBalancerAttachment struct {
	ContainerName string
	ContainerPort int
	TargetGroup   *cflib.Resource[*elbv2.TargetGroup]
}

func NewRuntimeService(globals Globals, spec *application_pb.Runtime) (*RuntimeService, error) {
	taskDef := NewECSTaskDefinition(globals, spec.Name)

	rs := &RuntimeService{
		spec:           spec,
		Prefix:         globals.AppName(),
		Name:           spec.Name,
		TaskDefinition: taskDef,
		TargetGroups:   map[string]*targetGroup{},
	}

	for _, def := range spec.Containers {
		err := taskDef.BuildRuntimeContainer(def)
		if err != nil {
			return nil, fmt.Errorf("building runtime container %s: %w", def.Name, err)
		}
	}

	if spec.WorkerConfig != nil {
		cfg := spec.WorkerConfig
		if err := taskDef.Sidecar.SetWorkerConfig(cfg); err != nil {
			return nil, fmt.Errorf("setting sidecar worker config: %w", err)
		}
	}

	rs.TaskDefinition.AddNamedPolicies(rs.spec.NamedEnvPolicies)

	if len(spec.Subscriptions) > 0 {
		for _, sub := range spec.Subscriptions {

			if sub.TargetContainer == "" {
				sub.TargetContainer = spec.Containers[0].Name
			}
			if sub.Port == 0 {
				sub.Port = 8080
			}

			taskDef.Sidecar.AddAppEndpoint(sub.TargetContainer, sub.Port)
		}

		subscriptionPlan, err := buildSubscriptionPlan(rs.Prefix, spec)
		if err != nil {
			return nil, err
		}
		rs.subscriptionPlan = subscriptionPlan

	}
	return rs, nil
}

func (rs *RuntimeService) AddTemplateResources(template *cflib.TemplateBuilder) error {
	var err error

	if rs.subscriptionPlan != nil {
		queueResource := cflib.NewResource(rs.Name, &sqs.Queue{
			QueueName: cloudformation.JoinPtr("-", []string{
				cloudformation.Ref("AWS::StackName"),
				rs.Name,
			}),
			SqsManagedSseEnabled: cflib.Bool(true),
			Tags:                 sourceTags(),
		})

		template.AddResource(queueResource)

		err = rs.TaskDefinition.Sidecar.SubscribeSQS(queueResource.Ref(), queueResource.GetAtt("Arn"))
		if err != nil {
			return err
		}

		err = rs.subscriptionPlan.AddToTemplate(template, queueResource)
		if err != nil {
			return err
		}
	}

	taskDefinition, err := rs.TaskDefinition.AddToTemplate(template)
	if err != nil {
		return err
	}

	service := cflib.NewResource(cflib.CleanParameterName(rs.Name), &ecs.Service{
		Cluster:        cflib.String(cloudformation.Ref(ECSClusterParameter)),
		TaskDefinition: cflib.String(taskDefinition.Ref()),
		//	DesiredCount:  Set on Apply
		DeploymentConfiguration: &ecs.Service_DeploymentConfiguration{
			DeploymentCircuitBreaker: &ecs.Service_DeploymentCircuitBreaker{
				Enable:   true,
				Rollback: true,
			},
			MinimumHealthyPercent: cflib.Int(0),
		},
		PropagateTags: cflib.String("TASK_DEFINITION"),

		NetworkConfiguration: &ecs.Service_NetworkConfiguration{
			AwsvpcConfiguration: &ecs.Service_AwsVpcConfiguration{},
		},
	})

	service.Override("NetworkConfiguration.AwsvpcConfiguration.SecurityGroups", cloudformation.Split(",", cloudformation.Ref(SecurityGroupParameter)))
	service.Override("NetworkConfiguration.AwsvpcConfiguration.Subnets", cloudformation.Split(",", cloudformation.Ref(SubnetIDsParameter)))

	for _, target := range rs.TargetGroups {
		lb := target.attachment
		service.Resource.LoadBalancers = append(service.Resource.LoadBalancers, ecs.Service_LoadBalancer{
			ContainerName:  cflib.String(lb.ContainerName),
			ContainerPort:  cflib.Int(lb.ContainerPort),
			TargetGroupArn: lb.TargetGroup.Ref().RefPtr(),
		})
		template.AddResource(target.resource)
		for _, rule := range target.rules {
			template.AddResource(rule)
			service.DependsOn(rule)
		}
	}

	desiredCountParameter := fmt.Sprintf("DesiredCount%s", rs.Name)
	template.AddParameter(&awsdeployer_pb.Parameter{
		Name: desiredCountParameter,
		Type: "Number",
		Source: &awsdeployer_pb.ParameterSourceType{
			Type: &awsdeployer_pb.ParameterSourceType_DesiredCount_{
				DesiredCount: &awsdeployer_pb.ParameterSourceType_DesiredCount{},
			},
		},
	})

	service.Override("DesiredCount", cloudformation.Ref(desiredCountParameter))

	template.AddResource(service)

	return nil
}

func (rs *RuntimeService) AddRoutes(ingress *ListenerRuleSet) error {
	for _, route := range rs.spec.Routes {
		targetContainer := O5SidecarContainerName
		port := route.Port
		if port == 0 {
			port = 8080
		}
		if route.TargetContainer == "" {
			route.TargetContainer = rs.spec.Containers[0].Name
		}

		var targetPort int
		if route.BypassIngress {
			targetPort = int(port)
			targetContainer = route.TargetContainer
			if err := rs.TaskDefinition.ExposeContainerPort(targetContainer, int(port)); err != nil {
				return err
			}
		} else {
			targetPort = 8888
			rs.TaskDefinition.Sidecar.AddAppEndpoint(route.TargetContainer, port)
			rs.TaskDefinition.Sidecar.ServePublic()
		}
		targetGroup, err := rs.LazyTargetGroup(route.Protocol, targetContainer, targetPort)
		if err != nil {
			return err
		}

		ruleResource, err := ingress.AddRoute(targetGroup.resource, route)
		if err != nil {
			return err
		}

		targetGroup.rules = append(targetGroup.rules, ruleResource)

	}

	return nil
}

func (rs *RuntimeService) LazyTargetGroup(protocol application_pb.RouteProtocol, targetContainer string, port int) (*targetGroup, error) {
	lookupKey := fmt.Sprintf("%s%s", protocol, targetContainer)
	existing, ok := rs.TargetGroups[lookupKey]
	if ok {
		return existing, nil
	}

	targetGroupDefinition := &elbv2.TargetGroup{
		VpcId:                      cflib.String(cloudformation.Ref(VPCParameter)),
		Port:                       cflib.Int(8080),
		Protocol:                   cflib.String("HTTP"),
		HealthCheckEnabled:         cflib.Bool(true),
		HealthCheckPort:            cflib.String("traffic-port"),
		HealthyThresholdCount:      cflib.Int(2),
		HealthCheckIntervalSeconds: cflib.Int(15),
		Matcher: &elbv2.TargetGroup_Matcher{
			HttpCode: cflib.String("200,401,404"),
		},
		Tags:       sourceTags(),
		TargetType: cflib.String("ip"),
	}
	switch protocol {
	case application_pb.RouteProtocol_ROUTE_PROTOCOL_HTTP:
		targetGroupDefinition.HealthCheckPath = cflib.String("/healthz")
		targetGroupDefinition.Matcher = &elbv2.TargetGroup_Matcher{
			HttpCode: cflib.String("200,401,404"),
		}

	case application_pb.RouteProtocol_ROUTE_PROTOCOL_GRPC:
		targetGroupDefinition.ProtocolVersion = cflib.String("GRPC")
		targetGroupDefinition.HealthCheckPath = cflib.String("/AWS.ALB/healthcheck")
		targetGroupDefinition.Matcher = &elbv2.TargetGroup_Matcher{
			GrpcCode: cflib.String("0-99"),
		}
	default:
		return nil, fmt.Errorf("unsupported protocol %s", protocol)
	}
	// faster deregistration
	a := elbv2.TargetGroup_TargetGroupAttribute{
		Key:   cflib.String("deregistration_delay.timeout_seconds"),
		Value: cflib.String("30"),
	}
	targetGroupDefinition.TargetGroupAttributes = []elbv2.TargetGroup_TargetGroupAttribute{a}

	targetGroupResource := cflib.NewResource(lookupKey, targetGroupDefinition)
	attachment := loadBalancerAttachment{
		ContainerName: targetContainer,
		ContainerPort: port,
		TargetGroup:   targetGroupResource,
	}
	tg := &targetGroup{
		resource:   targetGroupResource,
		attachment: attachment,
	}
	rs.TargetGroups[lookupKey] = tg

	return tg, nil
}
