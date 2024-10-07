package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/sns"
	"github.com/awslabs/goformation/v7/cloudformation/sqs"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type RuntimeService struct {
	Prefix string
	Name   string

	TaskDefinition *cflib.Resource[*ecs.TaskDefinition]
	Containers     []*ContainerDefinition
	Service        *cflib.Resource[*ecs.Service]

	TargetGroups map[string]*cflib.Resource[*elbv2.TargetGroup]

	Policy *PolicyBuilder

	Sidecar *SidecarBuilder

	spec *application_pb.Runtime
}

func NewRuntimeService(globals Globals, runtime *application_pb.Runtime) (*RuntimeService, error) {

	policy := NewPolicyBuilder()

	defs := []*ContainerDefinition{}

	needsDockerVolume := false
	for _, def := range runtime.Containers {
		container, err := buildContainer(globals, policy, def)
		if err != nil {
			return nil, fmt.Errorf("building service container %s: %w", def.Name, err)
		}

		needsDockerVolume = needsDockerVolume || def.MountDockerSocket

		addLogs(container.Container, globals.AppName())
		defs = append(defs, container)
	}

	sidecar := NewSidecarBuilder(globals.AppName())

	if runtime.WorkerConfig != nil {
		cfg := runtime.WorkerConfig
		if err := sidecar.SetWorkerConfig(cfg); err != nil {
			return nil, fmt.Errorf("setting sidecar worker config: %w", err)
		}
	}

	taskDefinition := cflib.NewResource(runtime.Name, &ecs.TaskDefinition{
		Family:                  cflib.String(fmt.Sprintf("%s_%s", globals.AppName(), runtime.Name)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
		//TaskRoleArn:  Set on Apply
		Tags: sourceTags(tags.Tag{
			Key:   "o5-deployment-version",
			Value: cloudformation.Ref(VersionTagParameter),
		}),
	})

	service := cflib.NewResource(cflib.CleanParameterName(runtime.Name), &ecs.Service{
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
	})

	if needsDockerVolume {
		taskDefinition.Resource.Volumes = []ecs.TaskDefinition_Volume{{
			Name: cflib.String("docker-socket"),
			Host: &ecs.TaskDefinition_HostVolumeProperties{
				SourcePath: cflib.String("/var/run/docker.sock"),
			},
		}}
		policy.AddECRPull()
	}

	return &RuntimeService{
		spec:           runtime,
		Prefix:         globals.AppName(),
		Name:           runtime.Name,
		Containers:     defs,
		TaskDefinition: taskDefinition,
		Service:        service,
		Policy:         policy,
		TargetGroups:   map[string]*cflib.Resource[*elbv2.TargetGroup]{},
		Sidecar:        sidecar,
	}, nil
}

func (rs *RuntimeService) AddTemplateResources(template *cflib.TemplateBuilder) error {

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

	rs.Service.Override("DesiredCount", cloudformation.Ref(desiredCountParameter))

	if len(rs.spec.Subscriptions) > 0 {
		for _, sub := range rs.spec.Subscriptions {

			if sub.TargetContainer == "" {
				sub.TargetContainer = rs.spec.Containers[0].Name
			}
			if sub.Port == 0 {
				sub.Port = 8080
			}

			rs.Sidecar.AddAppEndpoint(sub.TargetContainer, sub.Port)
		}

		subscriptionPlan, err := buildSubscriptionPlan(rs.spec)
		if err != nil {
			return err
		}

		if err := rs.applySubscriptionPlan(template, subscriptionPlan); err != nil {
			return err
		}
	}

	for _, container := range rs.Containers {
		if container.AdapterEndpoint != nil {
			value := cflib.String(fmt.Sprintf("http://%s:%d", O5SidecarContainerName, O5SidecarInternalPort))
			container.AdapterEndpoint.EnvVar.Value = value
			rs.Sidecar.ServeAdapter()
		}

		for _, ref := range container.ProxyDBs {
			envVarValue, err := rs.Sidecar.ProxyDB(ref.Database)
			if err != nil {
				return fmt.Errorf("building proxy var for database %s: %w", ref.Database.Name(), err)
			}
			*ref.EnvVarVal = envVarValue
		}
	}

	// Not sure who thought it would be a good idea to not use pointers here...
	defs := make([]ecs.TaskDefinition_ContainerDefinition, len(rs.Containers))
	for i, def := range rs.Containers {
		defs[i] = *def.Container
		for _, param := range def.Parameters {
			rs.Service.AddParameter(param)
		}
	}

	if rs.Sidecar.IsRequired() {
		def, err := rs.Sidecar.Build()
		if err != nil {
			return err
		}
		defs = append(defs, *def)
	}

	rs.TaskDefinition.Resource.ContainerDefinitions = defs

	roleARN := rs.applyRole(template)
	rs.TaskDefinition.Resource.TaskRoleArn = roleARN.RefPtr()

	template.AddResource(rs.TaskDefinition)
	template.AddResource(rs.Service)

	for _, targetGroup := range rs.TargetGroups {
		template.AddResource(targetGroup)
	}

	return nil
}

func (rs *RuntimeService) applySubscriptionPlan(template *cflib.TemplateBuilder, subscriptionPlan *subscriptionPlan) error {
	queueResource := cflib.NewResource(rs.Name, &sqs.Queue{
		QueueName: cloudformation.JoinPtr("-", []string{
			cloudformation.Ref("AWS::StackName"),
			rs.Name,
		}),
		SqsManagedSseEnabled: cflib.Bool(true),
		Tags:                 sourceTags(),
	})

	template.AddResource(queueResource)

	rs.Policy.AddSQSSubscribe(queueResource.GetAtt("Arn"))
	if err := rs.Sidecar.SubscribeSQS(queueResource.Ref()); err != nil {
		return err
	}

	for _, param := range subscriptionPlan.parameters {
		template.AddParameter(param)
	}

	topicARNs := []string{}
	for _, sub := range subscriptionPlan.snsSubscriptions {
		subscription := cflib.NewResource(cflib.CleanParameterName(rs.Name, sub.name), &sns.Subscription{
			TopicArn:           sub.topicARN,
			Protocol:           "sqs",
			RawMessageDelivery: cflib.Bool(false), // Always include the SNS header info for infra events.
			Endpoint:           cflib.String(queueResource.GetAtt("Arn")),
		})

		// The topic is not added to the stack, it should already exist
		// in this case.
		template.AddResource(subscription)
		topicARNs = append(topicARNs, sub.topicARN)
	}

	for _, sub := range subscriptionPlan.eventBusSubscriptions {
		eventBusSubscription := &events.Rule{
			Description:  cflib.String(fmt.Sprintf("Subscription for app %s %s", rs.Name, sub.name)),
			EventBusName: cloudformation.RefPtr(EventBusARNParameter),
			Targets: []events.Rule_Target{{
				Arn: queueResource.GetAtt("Arn"),
				Id:  "SQS",
			}},
			EventPattern: sub.eventPattern,
		}
		template.AddResource(cflib.NewResource(cflib.CleanParameterName(rs.Name, "subscription", sub.name), eventBusSubscription))
	}

	queuePolicyStatement := []interface{}{
		map[string]interface{}{
			"Effect": "Allow",
			"Principal": map[string]interface{}{
				"Service": "events.amazonaws.com",
			},
			"Action":   "sqs:SendMessage",
			"Resource": queueResource.GetAtt("Arn"),
		}}

	if len(topicARNs) > 0 {
		queuePolicyStatement = append(queuePolicyStatement, map[string]interface{}{
			"Effect":    "Allow",
			"Principal": "*",
			"Action":    "sqs:SendMessage",
			"Resource":  queueResource.GetAtt("Arn"),
			"Condition": map[string]interface{}{
				"ArnEquals": map[string]interface{}{
					"aws:SourceArn": topicARNs,
				},
			},
		})
	}

	// Allow SNS and EventBridge to publish to SQS...
	// (The ARN distinguishes the source)
	template.AddResource(cflib.NewResource(cflib.CleanParameterName(rs.Name), &sqs.QueuePolicy{
		Queues: []string{queueResource.Ref()},
		PolicyDocument: map[string]interface{}{
			"Version":   "2012-10-17",
			"Statement": queuePolicyStatement,
		},
	}))

	return nil
}

func (rs *RuntimeService) applyRole(template *cflib.TemplateBuilder) TemplateRef {
	builtRole := rs.Policy.BuildRole(rs.Prefix, rs.Name)

	roleResource := cflib.NewResource(fmt.Sprintf("%sAssume", rs.Name), builtRole)
	for _, policy := range rs.spec.NamedEnvPolicies {
		policyARN := cflib.CleanParameterName("Named IAM Policy", rs.Name, policy)
		roleResource.AddParameter(&awsdeployer_pb.Parameter{
			Name:        policyARN,
			Type:        "String",
			Description: fmt.Sprintf("ARN of the env-named IAM policy %s", policy),
			Source: &awsdeployer_pb.ParameterSourceType{
				Type: &awsdeployer_pb.ParameterSourceType_NamedIamPolicy{
					NamedIamPolicy: &awsdeployer_pb.ParameterSourceType_NamedIAMPolicy{
						Name: policy,
					},
				},
			},
		})
		roleResource.Resource.ManagedPolicyArns = append(roleResource.Resource.ManagedPolicyArns, cloudformation.Ref(policyARN))
	}
	template.AddResource(roleResource)
	return TemplateRef(roleResource.GetAtt("Arn"))
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

		if route.BypassIngress {
			targetContainer = route.TargetContainer
			if err := rs.exposeContainerPort(targetContainer, int(port)); err != nil {
				return err
			}
		} else {
			rs.Sidecar.AddAppEndpoint(route.TargetContainer, port)
		}
		targetGroup, err := rs.LazyTargetGroup(route.Protocol, targetContainer, int(port))
		if err != nil {
			return err
		}

		routeRule, err := ingress.AddRoute(targetGroup, route)
		if err != nil {
			return err
		}

		rs.Service.DependsOn(routeRule)
	}

	return nil
}

func (rs *RuntimeService) LazyTargetGroup(protocol application_pb.RouteProtocol, targetContainer string, port int) (*cflib.Resource[*elbv2.TargetGroup], error) {
	lookupKey := fmt.Sprintf("%s%s", protocol, targetContainer)
	existing, ok := rs.TargetGroups[lookupKey]
	if ok {
		return existing, nil
	}

	var targetGroupDefinition *elbv2.TargetGroup

	switch protocol {
	case application_pb.RouteProtocol_ROUTE_PROTOCOL_HTTP:
		targetGroupDefinition = &elbv2.TargetGroup{
			VpcId:                      cflib.String(cloudformation.Ref(VPCParameter)),
			Port:                       cflib.Int(8080),
			Protocol:                   cflib.String("HTTP"),
			HealthCheckEnabled:         cflib.Bool(true),
			HealthCheckPort:            cflib.String("traffic-port"),
			HealthCheckPath:            cflib.String("/healthz"),
			HealthyThresholdCount:      cflib.Int(2),
			HealthCheckIntervalSeconds: cflib.Int(15),
			Matcher: &elbv2.TargetGroup_Matcher{
				HttpCode: cflib.String("200,401,404"),
			},
			Tags: sourceTags(),
		}

	case application_pb.RouteProtocol_ROUTE_PROTOCOL_GRPC:
		targetGroupDefinition = &elbv2.TargetGroup{
			VpcId:                      cflib.String(cloudformation.Ref(VPCParameter)),
			Port:                       cflib.Int(8080),
			Protocol:                   cflib.String("HTTP"),
			ProtocolVersion:            cflib.String("GRPC"),
			HealthCheckEnabled:         cflib.Bool(true),
			HealthCheckPort:            cflib.String("traffic-port"),
			HealthCheckPath:            cflib.String("/AWS.ALB/healthcheck"),
			HealthyThresholdCount:      cflib.Int(2),
			HealthCheckIntervalSeconds: cflib.Int(15),
			Matcher: &elbv2.TargetGroup_Matcher{
				GrpcCode: cflib.String("0-99"),
			},
			Tags: sourceTags(),
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
	rs.TargetGroups[lookupKey] = targetGroupResource

	rs.Service.Resource.LoadBalancers = append(rs.Service.Resource.LoadBalancers, ecs.Service_LoadBalancer{
		ContainerName:  cflib.String(targetContainer),
		ContainerPort:  cflib.Int(port),
		TargetGroupArn: cflib.String(targetGroupResource.Ref()),
	})

	return targetGroupResource, nil
}

func (rs *RuntimeService) exposeContainerPort(containerName string, port int) error {
	for _, container := range rs.Containers {
		if container.Container.Name != containerName {
			continue
		}
		found := false
		for _, portMap := range container.Container.PortMappings {
			if *portMap.ContainerPort == port {
				found = true
				break
			}
		}
		if !found {
			container.Container.PortMappings = append(container.Container.PortMappings, ecs.TaskDefinition_PortMapping{
				ContainerPort: cflib.Int(port),
			})
		}
		return nil
	}
	return fmt.Errorf("container %s not found in service %s", containerName, rs.Name)
}
