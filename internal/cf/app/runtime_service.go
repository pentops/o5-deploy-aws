package app

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
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
)

type RuntimeService struct {
	Prefix string
	Name   string

	TaskDefinition *cf.Resource[*ecs.TaskDefinition]
	Containers     []*ContainerDefinition
	Service        *cf.Resource[*ecs.Service]

	TargetGroups map[string]*cf.Resource[*elbv2.TargetGroup]

	Policy *PolicyBuilder

	AdapterContainer *ecs.TaskDefinition_ContainerDefinition

	ingressEndpoints map[string]struct{}

	spec *application_pb.Runtime

	outboxDatabases []DatabaseReference
}

func NewRuntimeService(globals globalData, runtime *application_pb.Runtime) (*RuntimeService, error) {
	defs := []*ContainerDefinition{}
	serviceLinks := []string{}

	needsDockerVolume := false
	for _, def := range runtime.Containers {

		serviceLinks = append(serviceLinks, fmt.Sprintf("%s:%s", def.Name, def.Name))

		container, err := buildContainer(globals, def)
		if err != nil {
			return nil, fmt.Errorf("building service container %s: %w", def.Name, err)
		}

		needsDockerVolume = needsDockerVolume || def.MountDockerSocket

		addLogs(container.Container, globals.appName)
		defs = append(defs, container)
	}

	runtimeSidecar := &ecs.TaskDefinition_ContainerDefinition{
		Name:      O5SidecarContainerName,
		Essential: cf.Bool(true),
		Image:     cloudformation.Ref(O5SidecarImageParameter),
		Cpu:       cf.Int(128),
		Memory:    cf.Int(128),
		PortMappings: []ecs.TaskDefinition_PortMapping{{
			ContainerPort: cf.Int(8080),
		}},
		Links: serviceLinks,
		Environment: []ecs.TaskDefinition_KeyValuePair{{
			Name:  cf.String("EVENTBRIDGE_ARN"),
			Value: cf.String(cloudformation.Ref(EventBusARNParameter)),
		}, {
			Name:  cf.String("APP_NAME"),
			Value: cf.String(globals.appName),
		}, {
			Name:  cf.String("ENVIRONMENT_NAME"),
			Value: cf.String(cloudformation.Ref(EnvNameParameter)),
		}, {
			Name:  cf.String("AWS_REGION"),
			Value: cf.String(cloudformation.Ref(AWSRegionParameter)),
		}, {
			Name:  cf.String("CORS_ORIGINS"),
			Value: cf.String(cloudformation.Ref(CORSOriginParameter)),
		}},
	}

	if runtime.WorkerConfig != nil {
		cfg := runtime.WorkerConfig
		if cfg.DeadletterChance > 0 {
			runtimeSidecar.Environment = append(runtimeSidecar.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String("DEADLETTER_CHANCE"),
				Value: cf.String(fmt.Sprintf("%v", cfg.DeadletterChance)),
			})
		}
		if cfg.ReplayChance > 0 {
			runtimeSidecar.Environment = append(runtimeSidecar.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String("RESEND_CHANCE"),
				Value: cf.String(fmt.Sprintf("%v", cfg.ReplayChance)),
			})
		}
		if cfg.NoDeadletters {
			runtimeSidecar.Environment = append(runtimeSidecar.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String("NO_DEADLETTERS"),
				Value: cf.String("true"),
			})
		}
	}

	addLogs(runtimeSidecar, globals.appName)

	taskDefinition := cf.NewResource(runtime.Name, &ecs.TaskDefinition{
		Family:                  cf.String(fmt.Sprintf("%s_%s", globals.appName, runtime.Name)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
		//TaskRoleArn:  Set on Apply
		Tags: sourceTags(tags.Tag{
			Key:   "o5-deployment-version",
			Value: cloudformation.Ref(VersionTagParameter),
		}),
	})

	service := cf.NewResource(cf.CleanParameterName(runtime.Name), &ecs.Service{
		Cluster:        cf.String(cloudformation.Ref(ECSClusterParameter)),
		TaskDefinition: cf.String(taskDefinition.Ref()),
		//	DesiredCount:  Set on Apply
		DeploymentConfiguration: &ecs.Service_DeploymentConfiguration{
			DeploymentCircuitBreaker: &ecs.Service_DeploymentCircuitBreaker{
				Enable:   true,
				Rollback: true,
			},
		},
		PropagateTags: cf.String("TASK_DEFINITION"),
	})

	policy := NewPolicyBuilder()

	if needsDockerVolume {
		taskDefinition.Resource.Volumes = []ecs.TaskDefinition_Volume{{
			Name: cf.String("docker-socket"),
			Host: &ecs.TaskDefinition_HostVolumeProperties{
				SourcePath: cf.String("/var/run/docker.sock"),
			},
		}}
		policy.AddECRPull()
	}

	for _, bucket := range globals.buckets {
		if !bucket.write && !bucket.read {
			bucket.read = true
		}
		if bucket.read && bucket.write {
			policy.AddBucketReadWrite(bucket.arn)
		} else if bucket.read {
			policy.AddBucketReadOnly(bucket.arn)
		} else if bucket.write {
			policy.AddBucketWriteOnly(bucket.arn)
		}
	}

	if runtime.GrantMetaDeployPermissions {
		policy.AddMetaDeployPermissions()
	}

	outboxDatabases := []DatabaseReference{}
	for _, db := range globals.databases {
		postgres := db.Definition.GetPostgres()
		if postgres == nil || !postgres.RunOutbox {
			continue
		}
		outboxDatabases = append(outboxDatabases, db)
	}

	return &RuntimeService{
		spec:             runtime,
		Prefix:           globals.appName,
		Name:             runtime.Name,
		Containers:       defs,
		TaskDefinition:   taskDefinition,
		Service:          service,
		Policy:           policy,
		TargetGroups:     map[string]*cf.Resource[*elbv2.TargetGroup]{},
		ingressEndpoints: map[string]struct{}{},
		AdapterContainer: runtimeSidecar,
		outboxDatabases:  outboxDatabases,
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

func (rs *RuntimeService) Apply(template *Application) error {

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

	// capture beofre running subscriptions as that adds to this set
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

		queueResource := cf.NewResource(rs.Name, &sqs.Queue{
			QueueName: cloudformation.JoinPtr("-", []string{
				cloudformation.Ref("AWS::StackName"),
				rs.Name,
			}),
			SqsManagedSseEnabled: cf.Bool(true),
			Tags:                 sourceTags(),
		})

		template.AddResource(queueResource)

		rs.Policy.AddSQSSubscribe(queueResource.GetAtt("Arn"))

		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  cf.String("SQS_URL"),
			Value: cf.String(queueResource.Ref()),
		})

		for _, param := range subscriptionPlan.parameters {
			template.AddParameter(param)
		}

		topicARNs := []string{}
		for _, sub := range subscriptionPlan.snsSubscriptions {
			subscription := cf.NewResource(cf.CleanParameterName(rs.Name, sub.name), &sns.Subscription{
				TopicArn:           sub.topicARN,
				Protocol:           "sqs",
				RawMessageDelivery: cf.Bool(false), // Always include the SNS header info for infra events.
				Endpoint:           cf.String(queueResource.GetAtt("Arn")),
			})

			// The topic is not added to the stack, it should already exist
			// in this case.
			template.AddResource(subscription)
			topicARNs = append(topicARNs, sub.topicARN)
		}

		for _, sub := range subscriptionPlan.eventBusSubscriptions {
			eventBusSubscription := &events.Rule{
				Description:  cf.String(fmt.Sprintf("Subscription for app %s %s", rs.Name, sub.name)),
				EventBusName: cloudformation.RefPtr(EventBusARNParameter),
				Targets: []events.Rule_Target{{
					Arn: queueResource.GetAtt("Arn"),
					Id:  "SQS",
				}},
				EventPattern: sub.eventPattern,
			}
			template.AddResource(cf.NewResource(cf.CleanParameterName(rs.Name, "subscription", sub.name), eventBusSubscription))
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
		template.AddResource(cf.NewResource(cf.CleanParameterName(rs.Name), &sqs.QueuePolicy{
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
			Name:  cf.String("SERVICE_ENDPOINT"),
			Value: cf.String(strings.Join(ingressEndpoints, ",")),
		}, ecs.TaskDefinition_KeyValuePair{
			Name:  cf.String("JWKS"),
			Value: cf.String(cloudformation.Ref(JWKSParameter)),
		})
		if ingressNeedsPublicPort {
			rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String("PUBLIC_ADDR"),
				Value: cf.String(":8080"),
			})
		}
	}

	if len(rs.outboxDatabases) > 1 {
		return fmt.Errorf("only one outbox DB supported")
	}

	for _, db := range rs.outboxDatabases {
		needsAdapterSidecar = true
		rs.AdapterContainer.Secrets = append(rs.AdapterContainer.Secrets, ecs.TaskDefinition_Secret{
			Name:      "POSTGRES_OUTBOX",
			ValueFrom: db.SecretValueFrom(),
		})
	}

	needsAdapterPort := false

	// If the app has the endpoint of the adapter, we still need ingress
	for _, container := range rs.spec.Containers {
		for _, envVar := range container.EnvVars {
			o5Type, ok := envVar.Spec.(*application_pb.EnvironmentVariable_O5)
			if !ok {
				continue
			}
			if o5Type.O5 == application_pb.O5Var_ADAPTER_ENDPOINT {
				needsAdapterPort = true
				break
			}
		}
	}

	if needsAdapterPort {
		needsAdapterSidecar = true
		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  cf.String("ADAPTER_ADDR"),
			Value: cf.String(fmt.Sprintf(":%d", O5SidecarInternalPort)),
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
	role := cf.NewResource(fmt.Sprintf("%sAssume", rs.Name), &iam.Role{
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
		Description:       cf.Stringf("Execution role for ecs in %s - %s", rs.Prefix, rs.Name),
		ManagedPolicyArns: []string{},
		Policies:          rolePolicies,
		RoleName: cloudformation.JoinPtr("-", []string{
			cloudformation.Ref("AWS::StackName"),
			rs.Name,
			"assume-role",
		}),
	})

	rs.TaskDefinition.Resource.TaskRoleArn = cf.String(role.GetAtt("Arn"))

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

func (rs *RuntimeService) LazyTargetGroup(protocol application_pb.RouteProtocol, targetContainer string, port int) (*cf.Resource[*elbv2.TargetGroup], error) {
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
			VpcId:                      cf.String(cloudformation.Ref(VPCParameter)),
			Port:                       cf.Int(8080),
			Protocol:                   cf.String("HTTP"),
			HealthCheckEnabled:         cf.Bool(true),
			HealthCheckPort:            cf.String("traffic-port"),
			HealthCheckPath:            cf.String("/healthz"),
			HealthyThresholdCount:      cf.Int(2),
			HealthCheckIntervalSeconds: cf.Int(15),
			Matcher: &elbv2.TargetGroup_Matcher{
				HttpCode: cf.String("200,401,404"),
			},
			Tags: sourceTags(),
		}

	case application_pb.RouteProtocol_ROUTE_PROTOCOL_GRPC:
		targetGroupDefinition = &elbv2.TargetGroup{
			VpcId:                      cf.String(cloudformation.Ref(VPCParameter)),
			Port:                       cf.Int(8080),
			Protocol:                   cf.String("HTTP"),
			ProtocolVersion:            cf.String("GRPC"),
			HealthCheckEnabled:         cf.Bool(true),
			HealthCheckPort:            cf.String("traffic-port"),
			HealthCheckPath:            cf.String("/AWS.ALB/healthcheck"),
			HealthyThresholdCount:      cf.Int(2),
			HealthCheckIntervalSeconds: cf.Int(15),
			Matcher: &elbv2.TargetGroup_Matcher{
				GrpcCode: cf.String("0-99"),
			},
			Tags: sourceTags(),
		}
	default:
		return nil, fmt.Errorf("unsupported protocol %s", protocol)
	}
	// faster deregistration
	a := elbv2.TargetGroup_TargetGroupAttribute{
		Key:   cf.String("deregistration_delay.timeout_seconds"),
		Value: cf.String("30"),
	}
	targetGroupDefinition.TargetGroupAttributes = []elbv2.TargetGroup_TargetGroupAttribute{a}

	targetGroupResource := cf.NewResource(lookupKey, targetGroupDefinition)
	rs.TargetGroups[lookupKey] = targetGroupResource

	rs.Service.Resource.LoadBalancers = append(rs.Service.Resource.LoadBalancers, ecs.Service_LoadBalancer{
		ContainerName:  cf.String(targetContainer),
		ContainerPort:  cf.Int(port),
		TargetGroupArn: cf.String(targetGroupResource.Ref()),
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
			ContainerPort: cf.Int(port),
		})
	}

	return targetGroupResource, nil
}
