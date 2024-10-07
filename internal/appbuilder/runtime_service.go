package appbuilder

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v7/cloudformation/events"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
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

	AdapterContainer *ecs.TaskDefinition_ContainerDefinition

	ingressEndpoints map[string]struct{}

	spec *application_pb.Runtime

	outboxDatabases []DatabaseRef
}

func NewRuntimeService(globals Globals, runtime *application_pb.Runtime) (*RuntimeService, error) {

	policy := NewPolicyBuilder()

	defs := []*ContainerDefinition{}
	serviceLinks := []string{}

	needsDockerVolume := false
	for _, def := range runtime.Containers {

		serviceLinks = append(serviceLinks, fmt.Sprintf("%s:%s", def.Name, def.Name))

		container, err := buildContainer(globals, policy, def)
		if err != nil {
			return nil, fmt.Errorf("building service container %s: %w", def.Name, err)
		}

		needsDockerVolume = needsDockerVolume || def.MountDockerSocket

		addLogs(container.Container, globals.AppName())
		defs = append(defs, container)
	}

	runtimeSidecar := &ecs.TaskDefinition_ContainerDefinition{
		Name:      O5SidecarContainerName,
		Essential: cflib.Bool(true),
		Image:     cloudformation.Ref(O5SidecarImageParameter),
		Cpu:       cflib.Int(128),
		Memory:    cflib.Int(128),
		PortMappings: []ecs.TaskDefinition_PortMapping{{
			ContainerPort: cflib.Int(8080),
		}},
		Links: serviceLinks,
		Environment: []ecs.TaskDefinition_KeyValuePair{{
			Name:  cflib.String("EVENTBRIDGE_ARN"),
			Value: cflib.String(cloudformation.Ref(EventBusARNParameter)),
		}, {
			Name:  cflib.String("APP_NAME"),
			Value: cflib.String(globals.AppName()),
		}, {
			Name:  cflib.String("ENVIRONMENT_NAME"),
			Value: cflib.String(cloudformation.Ref(EnvNameParameter)),
		}, {
			Name:  cflib.String("AWS_REGION"),
			Value: cflib.String(cloudformation.Ref(AWSRegionParameter)),
		}, {
			Name:  cflib.String("CORS_ORIGINS"),
			Value: cflib.String(cloudformation.Ref(CORSOriginParameter)),
		}},
	}

	if runtime.WorkerConfig != nil {
		cfg := runtime.WorkerConfig
		if cfg.DeadletterChance > 0 {
			runtimeSidecar.Environment = append(runtimeSidecar.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String("DEADLETTER_CHANCE"),
				Value: cflib.String(fmt.Sprintf("%v", cfg.DeadletterChance)),
			})
		}
		if cfg.ReplayChance > 0 {
			runtimeSidecar.Environment = append(runtimeSidecar.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String("RESEND_CHANCE"),
				Value: cflib.String(fmt.Sprintf("%v", cfg.ReplayChance)),
			})
		}
		if cfg.NoDeadletters {
			runtimeSidecar.Environment = append(runtimeSidecar.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String("NO_DEADLETTERS"),
				Value: cflib.String("true"),
			})
		}
	}

	addLogs(runtimeSidecar, globals.AppName())

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
		spec:             runtime,
		Prefix:           globals.AppName(),
		Name:             runtime.Name,
		Containers:       defs,
		TaskDefinition:   taskDefinition,
		Service:          service,
		Policy:           policy,
		TargetGroups:     map[string]*cflib.Resource[*elbv2.TargetGroup]{},
		ingressEndpoints: map[string]struct{}{},
		AdapterContainer: runtimeSidecar,
	}, nil
}

func addLogs(def *ecs.TaskDefinition_ContainerDefinition, rsPrefix string) {
	def.LogConfiguration = &ecs.TaskDefinition_LogConfiguration{
		LogDriver: "awslogs",
		Options: map[string]string{
			"awslogs-group": cloudformation.Join("/", []string{
				"ecs",
				cloudformation.Ref(EnvNameParameter),
				rsPrefix,
			}),
			"awslogs-create-group":  "true",
			"awslogs-region":        cloudformation.Ref("AWS::Region"),
			"awslogs-stream-prefix": def.Name,
		},
	}

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

	// capture beofore running subscriptions as that adds to this set
	ingressNeedsPublicPort := len(rs.ingressEndpoints) > 0

	needsAdapterSidecar := false
	if len(rs.spec.Subscriptions) > 0 {

		needsAdapterSidecar = true

		subscriptionPlan, err := buildSubscriptionPlan(rs.spec)
		if err != nil {
			return err
		}

		for endpoint := range subscriptionPlan.ingressEndpoints {
			// TODO: This registers the whole endpoint for proto reflection, but isn't hard-linked to
			// the queue or subscriptions.
			// If multiple containers both subscribe to the same topic via the
			// proto reflection, it's random which one will receive the message.
			rs.ingressEndpoints[endpoint] = struct{}{}
		}

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

		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  cflib.String("SQS_URL"),
			Value: cflib.String(queueResource.Ref()),
		})

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
	}

	if len(rs.ingressEndpoints) > 0 {
		needsAdapterSidecar = true
		ingressEndpoints := make([]string, 0, len(rs.ingressEndpoints))
		for endpoint := range rs.ingressEndpoints {
			ingressEndpoints = append(ingressEndpoints, endpoint)
		}
		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  cflib.String("SERVICE_ENDPOINT"),
			Value: cflib.String(strings.Join(ingressEndpoints, ",")),
		}, ecs.TaskDefinition_KeyValuePair{
			Name:  cflib.String("JWKS"),
			Value: cflib.String(cloudformation.Ref(JWKSParameter)),
		})
		if ingressNeedsPublicPort {
			rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String("PUBLIC_ADDR"),
				Value: cflib.String(":8080"),
			})
		}
	}

	if len(rs.outboxDatabases) > 1 {
		return fmt.Errorf("only one outbox DB supported")
	}

	for _, db := range rs.outboxDatabases {
		needsAdapterSidecar = true
		if db.IsProxy() {
		} else {
			secretVal, ok := db.SecretValueFrom()
			if !ok {
				return fmt.Errorf("outbox database %s is not a proxy and has no secret", db)
			}
			rs.AdapterContainer.Secrets = append(rs.AdapterContainer.Secrets, ecs.TaskDefinition_Secret{
				Name:      "POSTGRES_OUTBOX",
				ValueFrom: secretVal.Ref(),
			})
		}
	}

	needsAdapterPort := false

	proxyDBs := map[string]DatabaseRef{}

	// If the app has the endpoint of the adapter, we still need ingress
	for _, container := range rs.Containers {
		if container.AdapterEndpoint != nil {
			needsAdapterPort = true
			value := cflib.String(fmt.Sprintf("http://%s:%d", O5SidecarContainerName, O5SidecarInternalPort))
			container.AdapterEndpoint.EnvVar.Value = value
		}

		for _, ref := range container.ProxyDBs {
			envVarValue, err := ref.Database.ProxyEnvVal(O5SidecarContainerName)
			if err != nil {
				return err
			}

			ref.EnvVar.Value = envVarValue
			proxyDBs[ref.Database.Name()] = ref.Database
		}
	}

	if needsAdapterPort {
		needsAdapterSidecar = true
		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  cflib.String("ADAPTER_ADDR"),
			Value: cflib.String(fmt.Sprintf(":%d", O5SidecarInternalPort)),
		})
	}

	if !needsAdapterSidecar {
		rs.AdapterContainer = nil
	}

	// Not sure who thought it would be a good idea to not use pointers here...
	defs := make([]ecs.TaskDefinition_ContainerDefinition, len(rs.Containers))
	for i, def := range rs.Containers {
		defs[i] = *def.Container
		for _, param := range def.Parameters {
			rs.Service.AddParameter(param)
		}
	}

	if rs.AdapterContainer != nil {
		defs = append(defs, *rs.AdapterContainer)
	}

	rs.TaskDefinition.Resource.ContainerDefinitions = defs

	template.AddResource(rs.TaskDefinition)
	template.AddResource(rs.Service)

	rolePolicies := rs.Policy.Build(rs.Prefix, rs.Name)
	role := cflib.NewResource(fmt.Sprintf("%sAssume", rs.Name), &iam.Role{
		AssumeRolePolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": "ecs-tasks.amazonaws.com",
					},
					"Action": "sts:AssumeRole",
				},
			},
		},
		Description:       cflib.Stringf("Execution role for ecs in %s - %s", rs.Prefix, rs.Name),
		ManagedPolicyArns: []string{},
		Policies:          rolePolicies,
		RoleName: cloudformation.JoinPtr("-", []string{
			cloudformation.Ref("AWS::StackName"),
			rs.Name,
			"assume-role",
		}),
	})

	for _, policy := range rs.spec.NamedEnvPolicies {
		policyARN := cflib.CleanParameterName("Named IAM Policy", rs.Name, policy)
		role.AddParameter(&awsdeployer_pb.Parameter{
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
		role.Resource.ManagedPolicyArns = append(role.Resource.ManagedPolicyArns, cloudformation.Ref(policyARN))
	}

	rs.TaskDefinition.Resource.TaskRoleArn = cflib.String(role.GetAtt("Arn"))

	template.AddResource(role)

	for _, targetGroup := range rs.TargetGroups {
		template.AddResource(targetGroup)
	}

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

		if route.BypassIngress {
			targetContainer = route.TargetContainer
		} else {
			rs.ingressEndpoints[fmt.Sprintf("%s:%d", route.TargetContainer, port)] = struct{}{}
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

	var container *ecs.TaskDefinition_ContainerDefinition

	if targetContainer == O5SidecarContainerName {
		container = rs.AdapterContainer
	} else {

		for _, search := range rs.Containers {
			if search.Container.Name == targetContainer {
				container = search.Container
				break
			}
		}
	}

	if container == nil {
		return nil, fmt.Errorf("container %s not found in service %s", targetContainer, rs.Name)
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

	found := false
	for _, portMap := range container.PortMappings {
		if *portMap.ContainerPort == port {
			found = true
			break
		}
	}
	if !found {
		container.PortMappings = append(container.PortMappings, ecs.TaskDefinition_PortMapping{
			ContainerPort: cflib.Int(port),
		})
	}

	return targetGroupResource, nil
}
