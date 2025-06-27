package appbuilder

import (
	"fmt"

	"slices"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type ContainerDefinition struct {
	container *ecs.TaskDefinition_ContainerDefinition

	Name string

	appName           string
	mountDockerSocket bool
	source            application_pb.IsContainer_Source
	envVars           []EnvVar
	exposePorts       []int

	Databases  []*DatabaseReference
	Buckets    []*BucketReference
	Secrets    []*SecretReference
	Parameters []*ParameterReference

	AdapterEndpoint *AdapterEndpoint
}

type ParameterReference struct {
	def   awsdeployer_pb.IsParameterSourceTypeWrappedType
	Name  string
	Value *valuePromise
}

type DatabaseReference struct {
	Name     string
	Database DatabaseRef
	envVars  []*DatabaseVariable
}

type DatabaseVariable struct {
	Name  string
	def   *application_pb.DatabaseEnvVar
	Value *valuePromise
}

type SecretReference struct {
	Name   string
	def    *application_pb.SecretEnvVar
	secret SecretRef
}

type BucketReference struct {
	Bucket  BucketRef
	Name    string
	envVars []*BucketVariable
}

type BucketVariable struct {
	def   *application_pb.EnvironmentVariable_Blobstore
	Value *valuePromise
}

type AdapterEndpoint struct {
	Value *valuePromise
}

type EnvVar struct {
	Name  string
	Value *valuePromise
}

type valuePromise struct {
	value      *string
	secretFrom *string
}

func (vp *valuePromise) SetValue(value string) {
	vp.value = &value
}

func (vp *valuePromise) SetRef(ref cflib.TemplateRef) {
	vp.value = ref.RefPtr()
}

func (vp *valuePromise) SetSecretFrom(ref cflib.TemplateRef) {
	vp.secretFrom = ref.RefPtr()
}

func buildContainer(globals Globals, def *application_pb.Container) (*ContainerDefinition, error) {
	container := &ecs.TaskDefinition_ContainerDefinition{
		Name:      def.Name,
		Essential: cflib.Bool(true),

		// TODO: Make these a Parameter for each size, for the environment to
		// set
		Cpu:               cflib.Int(128),
		MemoryReservation: cflib.Int(256),
		MountPoints: []ecs.TaskDefinition_MountPoint{{
			ContainerPath: cflib.String("/sockets"),
			SourceVolume:  cflib.String("sockets"),
		}},
	}
	if len(def.Command) > 0 {
		container.Command = def.Command
	}
	cd := &ContainerDefinition{
		container: container,

		Name:              def.Name,
		appName:           globals.AppName(),
		mountDockerSocket: def.MountDockerSocket,
		source:            def.Source,
	}
	for _, envVar := range def.EnvVars {
		err := cd.addDefEnvVar(globals, envVar)
		if err != nil {
			return nil, err
		}
	}

	return cd, nil
}

func (cd *ContainerDefinition) bucket(globals Globals, bucketName string) (*BucketReference, error) {
	for _, bucket := range cd.Buckets {
		if bucket.Name == bucketName {
			return bucket, nil
		}
	}
	bucket, ok := globals.Bucket(bucketName)
	if !ok {
		return nil, fmt.Errorf("unknown bucket %q", bucketName)
	}
	bucketRef := &BucketReference{
		Bucket: bucket,
		Name:   bucketName,
	}
	cd.Buckets = append(cd.Buckets, bucketRef)
	return bucketRef, nil
}

func (cd *ContainerDefinition) database(globals Globals, dbName string) (*DatabaseReference, error) {
	for _, db := range cd.Databases {
		if db.Name == dbName {
			return db, nil
		}
	}
	db, ok := globals.Database(dbName)
	if !ok {
		return nil, fmt.Errorf("unknown database %q", dbName)
	}
	dbRef := &DatabaseReference{
		Name:     dbName,
		Database: db,
		envVars:  []*DatabaseVariable{},
	}
	cd.Databases = append(cd.Databases, dbRef)
	return dbRef, nil
}

func (cd *ContainerDefinition) addDefEnvVar(globals Globals, envVar *application_pb.EnvironmentVariable) error {
	switch varType := envVar.Spec.(type) {
	case *application_pb.EnvironmentVariable_Value:
		cd.envVars = append(cd.envVars, EnvVar{
			Name: envVar.Name,
			Value: &valuePromise{
				value: cloudformation.String(varType.Value),
			},
		})

	case *application_pb.EnvironmentVariable_Blobstore:
		bucket, err := cd.bucket(globals, varType.Blobstore.Name)
		if err != nil {
			return err
		}

		if !varType.Blobstore.GetS3Direct() {
			return fmt.Errorf("only S3Direct is supported")
		}

		value := &valuePromise{}
		bucketValue := &BucketVariable{
			def:   varType,
			Value: value,
		}
		bucket.envVars = append(bucket.envVars, bucketValue)
		cd.envVars = append(cd.envVars, EnvVar{
			Name:  envVar.Name,
			Value: value,
		})

		return nil

	case *application_pb.EnvironmentVariable_Secret:
		secretName := varType.Secret.SecretName
		secretDef, ok := globals.Secret(secretName)
		if !ok {
			return fmt.Errorf("unknown secret: %s", secretName)
		}

		cd.Secrets = append(cd.Secrets, &SecretReference{
			Name:   envVar.Name,
			secret: secretDef,
			def:    varType.Secret,
		})

		return nil

	case *application_pb.EnvironmentVariable_Database:
		dbDef, err := cd.database(globals, varType.Database.DatabaseName)
		if err != nil {
			return err
		}

		value := &valuePromise{}
		variable := &DatabaseVariable{
			def:   varType.Database,
			Value: value,
		}
		dbDef.envVars = append(dbDef.envVars, variable)

		cd.envVars = append(cd.envVars, EnvVar{
			Name:  envVar.Name,
			Value: value,
		})

		return nil
	case *application_pb.EnvironmentVariable_EnvMap:
		return fmt.Errorf("EnvMap not implemented")

	case *application_pb.EnvironmentVariable_FromEnv:
		varName := varType.FromEnv.Name
		value := &valuePromise{}
		param := &ParameterReference{
			Name: varName,
			def: &awsdeployer_pb.ParameterSourceType_EnvVar{
				Name: varType.FromEnv.Name,
			},
			Value: value,
		}
		cd.Parameters = append(cd.Parameters, param)

		cd.envVars = append(cd.envVars, EnvVar{
			Name:  envVar.Name,
			Value: value,
		})

		return nil

	case *application_pb.EnvironmentVariable_O5:
		value := &valuePromise{}
		envVar := EnvVar{
			Name:  envVar.Name,
			Value: value,
		}
		switch varType.O5 {
		case application_pb.O5Var_ADAPTER_ENDPOINT:
			cd.AdapterEndpoint = &AdapterEndpoint{
				Value: value,
			}

		default:
			return fmt.Errorf("unknown O5 var: %s", varType.O5)
		}
		cd.envVars = append(cd.envVars, envVar)

	default:
		return fmt.Errorf("unknown env var type: %T", varType)

	}

	return nil
}

func (cd *ContainerDefinition) ExposePort(port int) {
	if slices.Contains(cd.exposePorts, port) {
		return
	}
	cd.exposePorts = append(cd.exposePorts, port)
}

func (cd *ContainerDefinition) ToCloudformation() (*ecs.TaskDefinition_ContainerDefinition, error) {

	container := cd.container

	for _, ref := range cd.Secrets {
		container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
			Name:      ref.Name,
			ValueFrom: string(ref.secret.SecretValueFrom(ref.def.JsonKey)),
		})
	}

	addLogsToContainer(container, cd.appName)

	ensureEnvVar(&container.Environment, "AWS_REGION", cloudformation.RefPtr("AWS::Region"))

	switch src := cd.source.(type) {
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

	for _, envVar := range cd.envVars {
		if envVar.Value.secretFrom != nil {
			container.Secrets = append(container.Secrets, ecs.TaskDefinition_Secret{
				Name:      envVar.Name,
				ValueFrom: string(*envVar.Value.secretFrom),
			})
		} else {
			container.Environment = append(container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String(envVar.Name),
				Value: envVar.Value.value,
			})
		}
	}
	for _, port := range cd.exposePorts {
		container.PortMappings = append(container.PortMappings, ecs.TaskDefinition_PortMapping{
			ContainerPort: cflib.Int(port),
		})
	}

	if cd.mountDockerSocket {
		container.MountPoints = append(container.MountPoints, ecs.TaskDefinition_MountPoint{
			ContainerPath: cflib.String("/var/run/docker.sock"),
			SourceVolume:  cflib.String("docker-socket"),
		})
	}
	return container, nil
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

func addLogsToContainer(container *ecs.TaskDefinition_ContainerDefinition, appName string) {
	container.LogConfiguration = &ecs.TaskDefinition_LogConfiguration{
		LogDriver: "awslogs",
		Options: map[string]string{
			"awslogs-group": cloudformation.Join("/", []string{
				"ecs",
				cloudformation.Ref(EnvNameParameter),
				appName,
				container.Name,
			}),
			"awslogs-create-group":  "true",
			"awslogs-region":        cloudformation.Ref("AWS::Region"),
			"awslogs-stream-prefix": container.Name,
		},
	}
}
