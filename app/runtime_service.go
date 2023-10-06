package app

import (
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

type RuntimeService struct {
	Prefix string
	Name   string

	TaskDefinition *Resource[*ecs.TaskDefinition]
	Containers     []*ecs.TaskDefinition_ContainerDefinition
	Service        *Resource[*ecs.Service]

	TargetGroups map[string]*Resource[*elbv2.TargetGroup]

	Policy *PolicyBuilder

	IngressContainer *ecs.TaskDefinition_ContainerDefinition

	ingressEndpoints map[string]struct{}

	spec *application_pb.Runtime
}

func NewRuntimeService(globals globalData, runtime *application_pb.Runtime) (*RuntimeService, error) {

	defs := []*ecs.TaskDefinition_ContainerDefinition{}
	serviceLinks := []string{}

	for _, def := range runtime.Containers {

		serviceLinks = append(serviceLinks, fmt.Sprintf("%s:%s", def.Name, def.Name))

		container, err := buildContainer(globals, def)
		if err != nil {
			return nil, err
		}

		addLogs(container, globals.appName)
		defs = append(defs, container)
	}

	runtimeSidecar := &ecs.TaskDefinition_ContainerDefinition{
		Name:      O5SidecarContainerName,
		Essential: Bool(true),
		Image:     O5SidecarImageName,
		Cpu:       Int(128),
		Memory:    Int(128),
		PortMappings: []ecs.TaskDefinition_PortMapping{{
			ContainerPort: Int(8080),
		}},
		Links: serviceLinks,
	}

	addLogs(runtimeSidecar, globals.appName)

	defs = append(defs, runtimeSidecar)

	taskDefinition := NewResource(runtime.Name, &ecs.TaskDefinition{
		Family:                  String(fmt.Sprintf("%s_%s", globals.appName, runtime.Name)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
		//TaskRoleArn:  Set on Apply
	})

	service := NewResource(runtime.Name, &ecs.Service{
		Cluster:        String(cloudformation.Ref(ECSClusterParameter)),
		TaskDefinition: String(taskDefinition.Ref()),
		//	DesiredCount:  Set on Apply
	})

	policy := NewPolicyBuilder()

	for _, bucket := range globals.buckets {
		policy.AddBucketReadWrite(bucket.GetAtt("Arn"))
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
		IngressContainer: runtimeSidecar,
	}, nil
}

func addLogs(def *ecs.TaskDefinition_ContainerDefinition, rsPrefix string) {
	def.LogConfiguration = &ecs.TaskDefinition_LogConfiguration{
		LogDriver: "awslogs",
		Options: map[string]string{
			// TODO: Include an Environment prefix using parameters
			"awslogs-group":         fmt.Sprintf("ecs/%s/%s", "TODO", rsPrefix),
			"awslogs-create-group":  "true",
			"awslogs-region":        cloudformation.Ref("AWS::Region"),
			"awslogs-stream-prefix": def.Name,
		},
	}
}

func buildContainer(globals globalData, def *application_pb.Container) (*ecs.TaskDefinition_ContainerDefinition, error) {
	container := &ecs.TaskDefinition_ContainerDefinition{
		Name:      def.Name,
		Essential: Bool(true),

		// TODO: Make these a Parameter for each size, for the environment to
		// set
		Cpu:    Int(128),
		Memory: Int(128),
	}

	if len(def.Command) > 0 {
		container.Command = def.Command
	}

	switch src := def.Source.(type) {
	case *application_pb.Container_Image_:
		var tag string
		if src.Image.Tag == nil {
			tag = cloudformation.Ref(VersionTagParameter)
		} else {
			tag = *src.Image.Tag
		}

		registry := cloudformation.Ref(ECSRepoParameter)
		if src.Image.Registry != nil {
			registry = *src.Image.Registry
		}
		container.Image = cloudformation.Join("", []string{
			registry,
			"/",
			src.Image.Name,
			":",
			tag,
		})

	case *application_pb.Container_ImageUrl:
		container.Image = src.ImageUrl

	default:
		return nil, fmt.Errorf("unknown source type: %T", src)
	}

	for _, envVar := range def.EnvVars {
		switch varType := envVar.Spec.(type) {
		case *application_pb.EnvironmentVariable_Value:
			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  String(envVar.Name),
				Value: String(varType.Value),
			})

		case *application_pb.EnvironmentVariable_Blobstore:
			bucketName := varType.Blobstore.Name
			bucketResource, ok := globals.buckets[bucketName]
			if !ok {
				return nil, fmt.Errorf("unknown blobstore: %s", bucketName)
			}

			if !varType.Blobstore.GetS3Direct() {
				return nil, fmt.Errorf("only S3Direct is supported")
			}

			var value *string
			if varType.Blobstore.SubPath == nil {
				value = cloudformation.JoinPtr("", []string{
					"s3://",
					*bucketResource.Resource.BucketName, // This is NOT a real string
				})
			} else {
				value = cloudformation.JoinPtr("", []string{
					"s3://",
					*bucketResource.Resource.BucketName, // This is NOT a real string
					"/",
					*varType.Blobstore.SubPath,
				})
			}

			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  String(envVar.Name),
				Value: value,
			})
			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  String("AWS_REGION"),
				Value: cloudformation.RefPtr("AWS::Region"),
			})

		case *application_pb.EnvironmentVariable_Secret:
			secretName := varType.Secret.SecretName
			secretDef, ok := globals.secrets[secretName]
			if !ok {
				return nil, fmt.Errorf("unknown secret: %s", secretName)
			}

			jsonKey := varType.Secret.JsonKey
			versionStage := ""
			versionID := ""

			container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
				Name: envVar.Name,
				ValueFrom: cloudformation.Join(":", []string{
					secretDef.Ref(),
					jsonKey,
					versionStage,
					versionID,
				}),
			})

		case *application_pb.EnvironmentVariable_Database:
			dbName := varType.Database.DatabaseName
			dbDef, ok := globals.databases[dbName]
			if !ok {
				return nil, fmt.Errorf("unknown database: %s", dbName)
			}
			jsonKey := "dburl"
			versionStage := ""
			versionID := ""
			container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
				Name: envVar.Name,
				ValueFrom: cloudformation.Join(":", []string{
					dbDef.SecretResource.Ref(),
					jsonKey,
					versionStage,
					versionID,
				}),
			})

			continue
		case *application_pb.EnvironmentVariable_EnvMap:
			return nil, fmt.Errorf("EnvMap not implemented")

		case *application_pb.EnvironmentVariable_FromEnv:
			return nil, fmt.Errorf("FromEnv not implemented")

		default:
			return nil, fmt.Errorf("unknown env var type: %T", varType)
		}

	}
	return container, nil
}

func (rs *RuntimeService) Apply(template *Application) {
	ingressEndpoints := make([]string, 0, len(rs.ingressEndpoints))
	for endpoint := range rs.ingressEndpoints {
		ingressEndpoints = append(ingressEndpoints, endpoint)
	}
	rs.IngressContainer.Environment = append(rs.IngressContainer.Environment, ecs.TaskDefinition_KeyValuePair{
		Name:  String("SERVICE_ENDPOINT"),
		Value: String(strings.Join(ingressEndpoints, ",")),
	})

	desiredCountParameter := fmt.Sprintf("DesiredCount%s", rs.Name)
	template.AddParameter(desiredCountParameter, cloudformation.Parameter{
		Type: "Number",
	})

	if rs.Service.Overrides == nil {
		rs.Service.Overrides = map[string]string{}
	}

	rs.Service.Overrides["DesiredCount"] = cloudformation.Ref(desiredCountParameter)

	// Not sure who thought it would be a good idea to not use pointers here...
	defs := make([]ecs.TaskDefinition_ContainerDefinition, len(rs.Containers))
	for i, def := range rs.Containers {
		defs[i] = *def
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

	for _, search := range rs.Containers {
		if search.Name == targetContainer {
			container = search
			break
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
				HttpCode: String("0-99"),
			},
		}
	}
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
