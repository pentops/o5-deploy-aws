package cf

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

	HTTPTargetGroup *Resource[*elbv2.TargetGroup]
	GRPCTargetGroup *Resource[*elbv2.TargetGroup]

	Policy *PolicyBuilder
}

func NewRuntimeService(globals globalData, runtime *application_pb.Runtime) (*RuntimeService, error) {

	defs := []*ecs.TaskDefinition_ContainerDefinition{}
	serviceEndpoints := []string{}
	serviceLinks := []string{}

	for _, def := range runtime.Containers {
		container := &ecs.TaskDefinition_ContainerDefinition{
			Name:      def.Name,
			Essential: Bool(true),

			// TODO: Make these a Parameter,
			// then Map from App * Env
			// ... does CF do math?
			Cpu:    Int(128),
			Memory: Int(128),
		}

		// TODO: This should probably be configured
		serviceEndpoints = append(serviceEndpoints, fmt.Sprintf("%s:8080", def.Name))
		serviceLinks = append(serviceLinks, fmt.Sprintf("%s:%s", def.Name, def.Name))

		switch src := def.Source.(type) {
		case *application_pb.Container_Image_:
			tag := src.Image.Tag
			if tag == "" {
				tag = cloudformation.Ref(VersionTagParameter)
			}
			container.Image = cloudformation.Join("", []string{
				cloudformation.Ref(ECSRepoParameter),
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
				return nil, fmt.Errorf("BlobStore not implemented")
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
						*dbDef.SecretResource.Ref(),
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
		Environment: []ecs.TaskDefinition_KeyValuePair{{
			Name:  String("SERVICE_ENDPOINT"),
			Value: String(strings.Join(serviceEndpoints, ",")),
		}},
		Links: serviceLinks,
	}

	defs = append(defs, runtimeSidecar)

	taskDefinition := NewResource(runtime.Name, &ecs.TaskDefinition{
		Family:                  String(fmt.Sprintf("%s_%s", globals.uniquePrefix, runtime.Name)),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
		//TaskRoleArn:  Set on Apply
	})

	service := NewResource(runtime.Name, &ecs.Service{
		Cluster:        String(cloudformation.Ref(ECSClusterParameter)),
		TaskDefinition: taskDefinition.Ref(),
		//	DesiredCount:  Set on Apply
	})

	policy := NewPolicyBuilder()

	return &RuntimeService{
		Prefix:         globals.uniquePrefix,
		Name:           runtime.Name,
		Containers:     defs,
		TaskDefinition: taskDefinition,
		Service:        service,
		Policy:         policy,
	}, nil
}

func (rs *RuntimeService) Apply(template *cloudformation.Template) {
	template.Parameters[ECSClusterParameter] = cloudformation.Parameter{
		Type: "String",
	}
	template.Parameters[ECSRepoParameter] = cloudformation.Parameter{
		Type: "String",
	}
	template.Parameters[VersionTagParameter] = cloudformation.Parameter{
		Type: "String",
	}
	template.Parameters[ECSTaskExecutionRoleParameter] = cloudformation.Parameter{
		Type: "String",
	}
	template.Parameters[ListenerARNParameter] = cloudformation.Parameter{
		Type: "String",
	}
	template.Parameters[EnvNameParameter] = cloudformation.Parameter{
		Type: "String",
	}
	template.Parameters[VPCParameter] = cloudformation.Parameter{
		Type: "AWS::EC2::VPC::Id",
	}

	desiredCountParameter := fmt.Sprintf("DesiredCount%s", rs.Name)
	template.Parameters[desiredCountParameter] = cloudformation.Parameter{
		Type: "Number",
	}

	if rs.Service.Overrides == nil {
		rs.Service.Overrides = map[string]string{}
	}

	rs.Service.Overrides["DesiredCount"] = cloudformation.Ref(desiredCountParameter)

	// Not sure who thought it would be a good idea to not use pointers here...
	defs := make([]ecs.TaskDefinition_ContainerDefinition, len(rs.Containers))
	for i, def := range rs.Containers {
		def.LogConfiguration = &ecs.TaskDefinition_LogConfiguration{
			LogDriver: "awslogs",
			Options: map[string]string{
				"awslogs-group":         fmt.Sprintf("ecs/%s/%s/%s", rs.Prefix, rs.Name, def.Name),
				"awslogs-create-group":  "true",
				"awslogs-region":        cloudformation.Ref("AWS::Region"),
				"awslogs-stream-prefix": fmt.Sprintf("%s-%s", rs.Prefix, rs.Name),
			},
		}
		defs[i] = *def
	}
	rs.TaskDefinition.Resource.ContainerDefinitions = defs

	template.Resources[rs.TaskDefinition.Name] = rs.TaskDefinition
	template.Resources[rs.Service.Name] = rs.Service

	rolePolicies := rs.Policy.Build(rs.Prefix, rs.Name)
	role := &iam.Role{
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
			rs.Prefix,
			rs.Name,
			"role",
		}),
	}

	taskRoleName := fmt.Sprintf("%sTaskRole", rs.Service.Name)
	rs.TaskDefinition.Resource.TaskRoleArn = cloudformation.GetAttPtr(taskRoleName, "Arn")
	template.Resources[taskRoleName] = role

	if rs.HTTPTargetGroup != nil {
		addStackResource(template, rs.HTTPTargetGroup)
	}
	if rs.GRPCTargetGroup != nil {
		addStackResource(template, rs.GRPCTargetGroup)
	}

}

func (rs *RuntimeService) LazyHTTPTargetGroup() *Resource[*elbv2.TargetGroup] {
	if rs.HTTPTargetGroup == nil {
		rs.HTTPTargetGroup = NewResource("HTTP", &elbv2.TargetGroup{
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
		})
		rs.Service.Resource.LoadBalancers = append(rs.Service.Resource.LoadBalancers, ecs.Service_LoadBalancer{
			ContainerName:  String(O5SidecarContainerName),
			ContainerPort:  Int(8080),
			TargetGroupArn: rs.HTTPTargetGroup.Ref(),
		})
	}
	return rs.HTTPTargetGroup

}

func (rs *RuntimeService) LazyGRPCTargetGroup() *Resource[*elbv2.TargetGroup] {
	if rs.GRPCTargetGroup == nil {
		rs.GRPCTargetGroup = NewResource("GRPC", &elbv2.TargetGroup{
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
		})
		rs.Service.Resource.LoadBalancers = append(rs.Service.Resource.LoadBalancers, ecs.Service_LoadBalancer{
			ContainerName:  String(O5SidecarContainerName),
			ContainerPort:  Int(8080),
			TargetGroupArn: rs.GRPCTargetGroup.Ref(),
		})
	}
	return rs.GRPCTargetGroup
}
