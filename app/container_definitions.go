package app

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

type ContainerDefinition struct {
	Container  *ecs.TaskDefinition_ContainerDefinition
	Parameters map[string]*Parameter
}

func buildContainer(globals globalData, def *application_pb.Container) (*ContainerDefinition, error) {
	container := &ecs.TaskDefinition_ContainerDefinition{
		Name:      def.Name,
		Essential: Bool(true),

		// TODO: Make these a Parameter for each size, for the environment to
		// set
		Cpu:               Int(128),
		MemoryReservation: Int(256),
	}

	if len(def.Command) > 0 {
		container.Command = def.Command
	}

	containerDef := &ContainerDefinition{
		Container:  container,
		Parameters: map[string]*Parameter{},
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
			varName := varType.FromEnv.Name
			paramName := fmt.Sprintf("EnvVar%s", CleanParameterName(varName))
			containerDef.Parameters[paramName] = &Parameter{
				Name:   paramName,
				Type:   "String",
				Source: ParameterSourceEnvVar,
				Args:   []interface{}{varType.FromEnv.Name},
			}

			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  String(envVar.Name),
				Value: cloudformation.RefPtr(paramName),
			})

			continue

		default:
			return nil, fmt.Errorf("unknown env var type: %T", varType)
		}

	}
	return containerDef, nil
}
