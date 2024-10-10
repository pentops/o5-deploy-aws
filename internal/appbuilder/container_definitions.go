package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type ContainerDefinition struct {
	Container  *ecs.TaskDefinition_ContainerDefinition
	Parameters map[string]*awsdeployer_pb.Parameter

	ProxyDBs        []ProxyDB
	AdapterEndpoint *AdapterEndpoint
}

type ProxyDB struct {
	Database  DatabaseRef
	EnvVarVal *string
}

type AdapterEndpoint struct {
	EnvVar *ecs.TaskDefinition_KeyValuePair
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

func buildContainer(globals Globals, iamPolicy *PolicyBuilder, def *application_pb.Container) (*ContainerDefinition, error) {
	container := &ecs.TaskDefinition_ContainerDefinition{
		Name:      def.Name,
		Essential: cflib.Bool(true),

		// TODO: Make these a Parameter for each size, for the environment to
		// set
		Cpu:               cflib.Int(128),
		MemoryReservation: cflib.Int(256),
		LogConfiguration: &ecs.TaskDefinition_LogConfiguration{
			LogDriver: "awslogs",
			Options: map[string]string{
				"awslogs-group": cloudformation.Join("/", []string{
					"ecs",
					cloudformation.Ref(EnvNameParameter),
					globals.AppName(),
				}),
				"awslogs-create-group":  "true",
				"awslogs-region":        cloudformation.Ref("AWS::Region"),
				"awslogs-stream-prefix": def.Name,
			},
		},
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
				Name:  cflib.String(envVar.Name),
				Value: cflib.String(varType.Value),
			})

		case *application_pb.EnvironmentVariable_Blobstore:
			bucketName := varType.Blobstore.Name
			bucket, ok := globals.Bucket(bucketName)
			if !ok {
				return nil, fmt.Errorf("unknown blobstore: %s", bucketName)
			}

			if !varType.Blobstore.GetS3Direct() {
				return nil, fmt.Errorf("only S3Direct is supported")
			}

			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String(envVar.Name),
				Value: bucket.S3URL(varType.Blobstore.SubPath).RefPtr(),
			})

			if iamPolicy == nil {
				return nil, fmt.Errorf("iamPolicy is required for blobstore env vars")
			}
			switch bucket.GetPermissions() {
			case ReadWrite:
				iamPolicy.AddBucketReadWrite(bucket.ARN())
			case ReadOnly:
				iamPolicy.AddBucketReadOnly(bucket.ARN())
			case WriteOnly:
				iamPolicy.AddBucketWriteOnly(bucket.ARN())
			}

		case *application_pb.EnvironmentVariable_Secret:
			secretName := varType.Secret.SecretName
			secretDef, ok := globals.Secret(secretName)
			if !ok {
				return nil, fmt.Errorf("unknown secret: %s", secretName)
			}

			container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
				Name:      envVar.Name,
				ValueFrom: string(secretDef.SecretValueFrom(varType.Secret.JsonKey)),
			})

		case *application_pb.EnvironmentVariable_Database:
			dbName := varType.Database.DatabaseName
			dbDef, ok := globals.Database(dbName)
			if !ok {
				return nil, fmt.Errorf("unknown database: %s", dbName)
			}

			isProxy := dbDef.IsProxy()
			if isProxy {
				envVarStr := ""
				envVarVal := &envVarStr
				envVar := ecs.TaskDefinition_KeyValuePair{
					Name:  cflib.String(envVar.Name),
					Value: envVarVal, // This pointer gets filled from the sidecar var
				}
				container.Environment = append(container.Environment, envVar)
				containerDef.ProxyDBs = append(containerDef.ProxyDBs, ProxyDB{
					Database:  dbDef,
					EnvVarVal: envVarVal,
				})

			} else {
				secretRef, ok := dbDef.SecretValueFrom()
				if !ok {
					return nil, fmt.Errorf("database %s is a proxy, but has no secret", dbName)
				}
				container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
					Name:      envVar.Name,
					ValueFrom: string(secretRef),
				})
			}

			continue
		case *application_pb.EnvironmentVariable_EnvMap:
			return nil, fmt.Errorf("EnvMap not implemented")

		case *application_pb.EnvironmentVariable_FromEnv:
			varName := varType.FromEnv.Name
			paramName := fmt.Sprintf("EnvVar%s", cflib.CleanParameterName(varName))
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
				Name:  cflib.String(envVar.Name),
				Value: cloudformation.RefPtr(paramName),
			})

			continue

		case *application_pb.EnvironmentVariable_O5:
			envVar := ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String(envVar.Name),
				Value: nil,
			}
			container.Environment = append(container.Environment, envVar)
			switch varType.O5 {
			case application_pb.O5Var_ADAPTER_ENDPOINT:
				containerDef.AdapterEndpoint = &AdapterEndpoint{
					EnvVar: &envVar,
				}

			default:
				return nil, fmt.Errorf("unknown O5 var: %s", varType.O5)
			}

		default:
			return nil, fmt.Errorf("unknown env var type: %T", varType)
		}

	}

	if def.MountDockerSocket {
		containerDef.Container.MountPoints = append(containerDef.Container.MountPoints, ecs.TaskDefinition_MountPoint{
			ContainerPath: cflib.String("/var/run/docker.sock"),
			SourceVolume:  cflib.String("docker-socket"),
		})
	}
	return containerDef, nil
}
