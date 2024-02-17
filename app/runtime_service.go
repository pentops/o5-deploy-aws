package app

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/sns"
	"github.com/awslabs/goformation/v7/cloudformation/sqs"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type RuntimeService struct {
	Prefix string
	Name   string

	TaskDefinition *Resource[*ecs.TaskDefinition]
	Containers     []*ContainerDefinition
	Service        *Resource[*ecs.Service]

	TargetGroups map[string]*Resource[*elbv2.TargetGroup]

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
		Essential: Bool(true),
		Image:     cloudformation.Ref(O5SidecarImageParameter),
		Cpu:       Int(128),
		Memory:    Int(128),
		PortMappings: []ecs.TaskDefinition_PortMapping{{
			ContainerPort: Int(8080),
		}},
		Links: serviceLinks,
		Environment: []ecs.TaskDefinition_KeyValuePair{{
			Name:  String("SNS_PREFIX"),
			Value: String(cloudformation.Ref(SNSPrefixParameter)),
		}, {
			Name:  String("AWS_REGION"),
			Value: String(cloudformation.Ref(AWSRegionParameter)),
		}, {
			Name:  String("CORS_ORIGINS"),
			Value: String(cloudformation.Ref(CORSOriginParameter)),
		}},
	}

	addLogs(runtimeSidecar, globals.appName)

	taskDefinition := NewResource(runtime.Name, &ecs.TaskDefinition{
		Family:                  String(fmt.Sprintf("%s_%s", globals.appName, runtime.Name)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
		//TaskRoleArn:  Set on Apply
		Tags: sourceTags(tags.Tag{
			Key:   "o5-deployment-version",
			Value: cloudformation.Ref(VersionTagParameter),
		}),
	})

	service := NewResource(CleanParameterName(runtime.Name), &ecs.Service{
		Cluster:        String(cloudformation.Ref(ECSClusterParameter)),
		TaskDefinition: String(taskDefinition.Ref()),
		//	DesiredCount:  Set on Apply
		DeploymentConfiguration: &ecs.Service_DeploymentConfiguration{
			DeploymentCircuitBreaker: &ecs.Service_DeploymentCircuitBreaker{
				Enable:   true,
				Rollback: true,
			},
		},
		PropagateTags: String("TASK_DEFINITION"),
	})

	policy := NewPolicyBuilder()

	if needsDockerVolume {
		taskDefinition.Resource.Volumes = []ecs.TaskDefinition_Volume{{
			Name: String("docker-socket"),
			Host: &ecs.TaskDefinition_HostVolumeProperties{
				SourcePath: String("/var/run/docker.sock"),
			},
		}}
		policy.AddECRPull()
	}

	for _, bucket := range globals.buckets {
		policy.AddBucketReadWrite(bucket.GetAtt("Arn"))
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
		TargetGroups:     map[string]*Resource[*elbv2.TargetGroup]{},
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
				def.Name,
			}),
			"awslogs-create-group":  "true",
			"awslogs-region":        cloudformation.Ref("AWS::Region"),
			"awslogs-stream-prefix": def.Name,
		},
	}
}

func (rs *RuntimeService) Apply(template *Application) error {

	desiredCountParameter := fmt.Sprintf("DesiredCount%s", rs.Name)
	template.AddParameter(&deployer_pb.Parameter{
		Name: desiredCountParameter,
		Type: "Number",
		Source: &deployer_pb.ParameterSourceType{
			Type: &deployer_pb.ParameterSourceType_DesiredCount_{
				DesiredCount: &deployer_pb.ParameterSourceType_DesiredCount{},
			},
		},
	})

	if rs.Service.Overrides == nil {
		rs.Service.Overrides = map[string]string{}
	}

	rs.Service.Overrides["DesiredCount"] = cloudformation.Ref(desiredCountParameter)

	// capture beofre running subscriptions as that adds to this set
	ingressNeedsPublicPort := len(rs.ingressEndpoints) > 0

	if len(rs.spec.Subscriptions) > 0 {
		for _, sub := range rs.spec.Subscriptions {
			if sub.TargetContainer == "" {
				sub.TargetContainer = rs.spec.Containers[0].Name
			}
			if sub.Port == 0 {
				sub.Port = 8080
			}

			// TODO: This registers the whole endpoint for proto reflection, but isn't hard-linked to
			// the queue or subscriptions.
			// If multiple containers both subscribe to the same topic via the
			// proto reflection, it's random which one will receive the message.
			rs.ingressEndpoints[fmt.Sprintf("%s:%d", rs.spec.Containers[0].Name, sub.Port)] = struct{}{}
		}

		queueResource := NewResource(rs.Name, &sqs.Queue{
			QueueName: cloudformation.JoinPtr("-", []string{
				cloudformation.Ref("AWS::StackName"),
				rs.Name,
			}),
			SqsManagedSseEnabled: Bool(true),
			Tags:                 sourceTags(),
		})
		template.AddResource(queueResource)
		rs.Policy.AddSQSSubscribe(queueResource.GetAtt("Arn"))

		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  String("SQS_URL"),
			Value: String(queueResource.Ref()),
		})

		topicARNs := []string{}
		for _, sub := range rs.spec.Subscriptions {

			snsTopicARN := cloudformation.Join("", []string{
				"arn:aws:sns:",
				cloudformation.Ref("AWS::Region"),
				":",
				cloudformation.Ref("AWS::AccountId"),
				":",
				cloudformation.Ref(EnvNameParameter),
				"-",
				sub.Name,
			})
			if sub.EnvName != nil {
				envSNSParamName := CleanParameterName(*sub.EnvName, "SNSPrefix")
				template.AddParameter(&deployer_pb.Parameter{
					Name: envSNSParamName,
					Type: "String",
					Source: &deployer_pb.ParameterSourceType{
						Type: &deployer_pb.ParameterSourceType_CrossEnvSns_{
							CrossEnvSns: &deployer_pb.ParameterSourceType_CrossEnvSns{
								EnvName: *sub.EnvName,
							},
						},
					},
				})
				snsTopicARN = cloudformation.Join("", []string{
					cloudformation.Ref(envSNSParamName),
					sub.Name,
				})
			}
			topicARNs = append(topicARNs, snsTopicARN)
			subscription := &sns.Subscription{
				TopicArn:           snsTopicARN,
				Protocol:           "sqs",
				RawMessageDelivery: Bool(true),
				Endpoint:           String(queueResource.GetAtt("Arn")),
			}

			if sub.RawMessage {
				// 'RawMessage' in this context means the subscription is a
				// o5.messaging.topic.RawMessage.
				// The sidecar will use the SNS wrapper to pull out the
				// original topic.
				subscription.RawMessageDelivery = Bool(false)
			}

			template.AddSNSTopic(sub.Name)
			template.AddResource(NewResource(CleanParameterName(rs.Name, sub.Name), subscription))
		}

		// Allow SNS to publish to SQS...
		template.AddResource(NewResource(CleanParameterName(rs.Name), &sqs.QueuePolicy{
			Queues: []string{queueResource.Ref()},
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":    "Allow",
						"Principal": "*",
						"Action":    "sqs:SendMessage",
						"Resource":  queueResource.GetAtt("Arn"),
						"Condition": map[string]interface{}{
							"ArnEquals": map[string]interface{}{
								"aws:SourceArn": topicARNs,
							},
						},
					},
				},
			},
		}))
	}

	needsAdapterSidecar := false
	if len(rs.ingressEndpoints) > 0 {
		needsAdapterSidecar = true
		ingressEndpoints := make([]string, 0, len(rs.ingressEndpoints))
		for endpoint := range rs.ingressEndpoints {
			ingressEndpoints = append(ingressEndpoints, endpoint)
		}
		rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
			Name:  String("SERVICE_ENDPOINT"),
			Value: String(strings.Join(ingressEndpoints, ",")),
		}, ecs.TaskDefinition_KeyValuePair{
			Name:  String("JWKS"),
			Value: String(cloudformation.Ref(JWKSParameter)),
		})
		if ingressNeedsPublicPort {
			rs.AdapterContainer.Environment = append(rs.AdapterContainer.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  String("PUBLIC_ADDR"),
				Value: String(":8080"),
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
			Name:  String("ADAPTER_ADDR"),
			Value: String(fmt.Sprintf(":%d", O5SidecarInternalPort)),
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
			template.parameters[param.Name] = param
		}
	}

	if rs.AdapterContainer != nil {
		defs = append(defs, *rs.AdapterContainer)
	}

	rs.TaskDefinition.Resource.ContainerDefinitions = defs

	template.AddResource(rs.TaskDefinition)
	template.AddResource(rs.Service)

	rolePolicies := rs.Policy.Build(rs.Prefix, rs.Name)
	role := NewResource(fmt.Sprintf("%sAssume", rs.Name), &iam.Role{
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
		Description:       Stringf("Execution role for ecs in %s - %s", rs.Prefix, rs.Name),
		ManagedPolicyArns: []string{},
		Policies:          rolePolicies,
		RoleName: cloudformation.JoinPtr("-", []string{
			cloudformation.Ref("AWS::StackName"),
			rs.Name,
			"assume-role",
		}),
	})

	rs.TaskDefinition.Resource.TaskRoleArn = String(role.GetAtt("Arn"))

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

func (rs *RuntimeService) LazyTargetGroup(protocol application_pb.RouteProtocol, targetContainer string, port int) (*Resource[*elbv2.TargetGroup], error) {
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
			VpcId:                      String(cloudformation.Ref(VPCParameter)),
			Port:                       Int(8080),
			Protocol:                   String("HTTP"),
			HealthCheckEnabled:         Bool(true),
			HealthCheckPort:            String("traffic-port"),
			HealthCheckPath:            String("/healthz"),
			HealthyThresholdCount:      Int(2),
			HealthCheckIntervalSeconds: Int(15),
			Matcher: &elbv2.TargetGroup_Matcher{
				HttpCode: String("200,401,404"),
			},
			Tags: sourceTags(),
		}

	case application_pb.RouteProtocol_ROUTE_PROTOCOL_GRPC:
		targetGroupDefinition = &elbv2.TargetGroup{
			VpcId:                      String(cloudformation.Ref(VPCParameter)),
			Port:                       Int(8080),
			Protocol:                   String("HTTP"),
			ProtocolVersion:            String("GRPC"),
			HealthCheckEnabled:         Bool(true),
			HealthCheckPort:            String("traffic-port"),
			HealthCheckPath:            String("/AWS.ALB/healthcheck"),
			HealthyThresholdCount:      Int(2),
			HealthCheckIntervalSeconds: Int(15),
			Matcher: &elbv2.TargetGroup_Matcher{
				GrpcCode: String("0-99"),
			},
			Tags: sourceTags(),
		}
	default:
		return nil, fmt.Errorf("unsupported protocol %s", protocol)
	}
	// faster deregistration
	a := elbv2.TargetGroup_TargetGroupAttribute{
		Key:   String("deregistration_delay.timeout_seconds"),
		Value: String("30"),
	}
	targetGroupDefinition.TargetGroupAttributes = []elbv2.TargetGroup_TargetGroupAttribute{a}

	targetGroupResource := NewResource(lookupKey, targetGroupDefinition)
	rs.TargetGroups[lookupKey] = targetGroupResource

	rs.Service.Resource.LoadBalancers = append(rs.Service.Resource.LoadBalancers, ecs.Service_LoadBalancer{
		ContainerName:  String(targetContainer),
		ContainerPort:  Int(port),
		TargetGroupArn: String(targetGroupResource.Ref()),
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
			ContainerPort: Int(port),
		})
	}

	return targetGroupResource, nil
}
