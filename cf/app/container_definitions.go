package app

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

type ContainerDefinition struct {
	Container  *ecs.TaskDefinition_ContainerDefinition
	Parameters map[string]*awsdeployer_pb.Parameter
}

func ensureEnvVar(envVars *[]ecs.TaskDefinition_KeyValuePair, name string, value *string) {
	for _, envVar := range *envVars {
		if *envVar.Name == name {
			return
		}
	}
	*envVars = append(*envVars, ecs.TaskDefinition_KeyValuePair{
		Name:  &name,
		Value: value,
	})
}

func buildContainer(globals globalData, def *application_pb.Container) (*ContainerDefinition, error) {
	container := &ecs.TaskDefinition_ContainerDefinition{
		Name:      def.Name,
		Essential: cf.Bool(true),

		// TODO: Make these a Parameter for each size, for the environment to
		// set
		Cpu:               cf.Int(128),
		MemoryReservation: cf.Int(256),
	}
	ensureEnvVar(&container.Environment, "AWS_REGION", cloudformation.RefPtr("AWS::Region"))

	if len(def.Command) > 0 {
		container.Command = def.Command
	}

	containerDef := &ContainerDefinition{
		Container:  container,
		Parameters: map[string]*awsdeployer_pb.Parameter{},
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
		return nil, fmt.Errorf("unknown container source type: %T", src)
	}

	for _, envVar := range def.EnvVars {
		switch varType := envVar.Spec.(type) {
		case *application_pb.EnvironmentVariable_Value:
			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String(envVar.Name),
				Value: cf.String(varType.Value),
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
					*bucketResource.name, // This is NOT a real string
				})
			} else {
				value = cloudformation.JoinPtr("", []string{
					"s3://",
					*bucketResource.name, // This is NOT a real string
					"/",
					*varType.Blobstore.SubPath,
				})
			}

			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String(envVar.Name),
				Value: value,
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
			container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
				Name:      envVar.Name,
				ValueFrom: dbDef.SecretValueFrom(),
			})

			continue
		case *application_pb.EnvironmentVariable_EnvMap:
			return nil, fmt.Errorf("EnvMap not implemented")

		case *application_pb.EnvironmentVariable_FromEnv:
			varName := varType.FromEnv.Name
			paramName := fmt.Sprintf("EnvVar%s", cf.CleanParameterName(varName))
			containerDef.Parameters[paramName] = &awsdeployer_pb.Parameter{
				Name: paramName,
				Type: "String",
				Source: &awsdeployer_pb.ParameterSourceType{
					Type: &awsdeployer_pb.ParameterSourceType_EnvVar_{
						EnvVar: &awsdeployer_pb.ParameterSourceType_EnvVar{
							Name: varType.FromEnv.Name,
						},
					},
				},
			}

			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String(envVar.Name),
				Value: cloudformation.RefPtr(paramName),
			})

			continue

		case *application_pb.EnvironmentVariable_O5:
			var value *string
			switch varType.O5 {
			case application_pb.O5Var_ADAPTER_ENDPOINT:
				value = cf.String(fmt.Sprintf("http://%s:%d", O5SidecarContainerName, O5SidecarInternalPort))
			default:
				return nil, fmt.Errorf("unknown O5 var: %s", varType.O5)
			}

			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cf.String(envVar.Name),
				Value: value,
			})

		default:
			return nil, fmt.Errorf("unknown env var type: %T", varType)
		}

	}

	if def.MountDockerSocket {
		containerDef.Container.MountPoints = append(containerDef.Container.MountPoints, ecs.TaskDefinition_MountPoint{
			ContainerPath: cf.String("/var/run/docker.sock"),
			SourceVolume:  cf.String("docker-socket"),
		})
	}
	return containerDef, nil
}
