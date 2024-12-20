package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/tags"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type ECSTaskDefinition struct {
	globals           Globals
	containers        []*ContainerDefinition
	family            string
	relativeName      string
	needsDockerVolume bool
	networkMode       string
	policy            *PolicyBuilder

	namedIAMPolicies []string

	Sidecar *SidecarBuilder
}

func NewECSTaskDefinition(globals Globals, runtimeName string) *ECSTaskDefinition {
	family := fmt.Sprintf("%s_%s", globals.AppName(), runtimeName)
	policy := NewPolicyBuilder()
	sidecar := NewSidecarBuilder(globals.AppName(), policy)
	return &ECSTaskDefinition{
		globals:      globals,
		policy:       policy,
		family:       family,
		relativeName: runtimeName,
		networkMode:  "bridge",
		Sidecar:      sidecar,
	}
}

func (td *ECSTaskDefinition) BuildRuntimeContainer(def *application_pb.Container) error {
	container, err := buildContainer(td.globals, def)
	if err != nil {
		return fmt.Errorf("building service container %s: %w", def.Name, err)
	}

	td.needsDockerVolume = td.needsDockerVolume || def.MountDockerSocket
	td.containers = append(td.containers, container)

	return nil
}

func (td *ECSTaskDefinition) ListDatabases() []*DatabaseReference {
	databases := make([]*DatabaseReference, 0, len(td.containers))
	for _, container := range td.containers {
		for _, db := range container.Databases {
			found := false
			for _, existing := range databases {
				if existing.Name == db.Name {
					found = true
					break
				}
			}
			if !found {
				databases = append(databases, db)
			}
		}
	}
	return databases
}

func (td *ECSTaskDefinition) resolveContainerResources(cd *ContainerDefinition) ([]*awsdeployer_pb.Parameter, error) {
	params := make([]*awsdeployer_pb.Parameter, 0)

	if cd.AdapterEndpoint != nil {
		value := fmt.Sprintf("http://%s:%d", O5SidecarContainerName, O5SidecarInternalPort)
		cd.AdapterEndpoint.Value.SetValue(value)
		td.Sidecar.ServeAdapter()
	}

	for _, ref := range cd.Buckets {
		bucket := ref.Bucket
		for _, bucketVar := range ref.envVars {
			bucketVar.Value.SetRef(bucket.S3URL(bucketVar.def.Blobstore.SubPath))
		}
		if td.policy == nil {
			return nil, fmt.Errorf("iamPolicy is required for blobstore env vars")
		}
		switch bucket.GetPermissions() {
		case ReadWrite:
			td.policy.AddBucketReadWrite(bucket.ARN())
		case ReadOnly:
			td.policy.AddBucketReadOnly(bucket.ARN())
		case WriteOnly:
			td.policy.AddBucketWriteOnly(bucket.ARN())
		}
	}

	for _, ref := range cd.Parameters {
		paramName := fmt.Sprintf("EnvVar%s", cflib.CleanParameterName(ref.Name))
		param := &awsdeployer_pb.Parameter{
			Name:   paramName,
			Type:   "String",
			Source: &awsdeployer_pb.ParameterSourceType{},
		}
		param.Source.Set(ref.def)
		ref.Value.SetRef(cflib.TemplateRef(cloudformation.Ref(paramName)))
		params = append(params, param)
	}

	for _, ref := range cd.Databases {
		if aurora, ok := ref.Database.AuroraProxy(); ok {
			dsnToProxy := td.Sidecar.ProxyDB(ref.Database)
			for _, envVar := range ref.envVars {
				envVar.Value.SetValue(dsnToProxy)
			}

			td.policy.AddRDSConnect(aurora.AuroraConnectARN().Ref())
		} else if secretRef, ok := ref.Database.SecretValueFrom(); ok {
			if !ok {
				return nil, fmt.Errorf("database %s is a proxy, but has no secret", ref.Name)
			}
			for _, envVar := range ref.envVars {
				envVar.Value.SetSecretFrom(secretRef)
			}
		} else {
			return nil, fmt.Errorf("database %s is not a proxy and has no secret", ref.Name)
		}
	}

	return params, nil
}

func (td *ECSTaskDefinition) ExposeContainerPort(containerName string, port int) error {
	for _, container := range td.containers {
		if container.Name != containerName {
			continue
		}
		container.ExposePort(port)
		return nil
	}
	return fmt.Errorf("container %s not found in task %s", containerName, td.family)
}

func (td *ECSTaskDefinition) AddEventBridgeTargets(targets []*application_pb.Target) {
	for _, target := range targets {
		td.policy.AddEventBridgePublish(target.Name)
	}
}

func (ts *ECSTaskDefinition) AddNamedPolicies(policyNames []string) {
	ts.namedIAMPolicies = append(ts.namedIAMPolicies, policyNames...)
}

func (td *ECSTaskDefinition) applyRole(template *cflib.TemplateBuilder) cflib.TemplateRef {

	params := []*awsdeployer_pb.Parameter{}
	for _, policy := range td.namedIAMPolicies {
		policyARN := cflib.CleanParameterName("Named IAM Policy", td.family, policy)
		params = append(params, &awsdeployer_pb.Parameter{
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
		td.policy.AddManagedPolicyARN(cloudformation.Ref(policyARN))
	}

	builtRole := td.policy.BuildRole(td.family)
	roleResource := cflib.NewResource(fmt.Sprintf("%sAssume", td.relativeName), builtRole)
	for _, param := range params {
		roleResource.AddParameter(param)
	}
	template.AddResource(roleResource)
	return roleResource.GetAtt("Arn")
}

func (td *ECSTaskDefinition) AddToTemplate(template *cflib.TemplateBuilder) (cflib.TemplateRef, error) {

	params := []*awsdeployer_pb.Parameter{}

	// Not sure who thought it would be a good idea to not use pointers here...
	defs := make([]ecs.TaskDefinition_ContainerDefinition, 0, len(td.containers))
	for _, def := range td.containers {
		defParams, err := td.resolveContainerResources(def)
		if err != nil {
			return "", err
		}
		params = append(params, defParams...)

		cfContainer, err := def.ToCloudformation()
		if err != nil {
			return "", err
		}
		defs = append(defs, *cfContainer)
	}

	if td.Sidecar.IsRequired() {
		def, err := td.Sidecar.Build()
		if err != nil {
			return "", err
		}
		defs = append(defs, *def)
	}

	taskDefinition := &ecs.TaskDefinition{
		Family:                  cflib.String(td.family),
		ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
		RequiresCompatibilities: []string{"EC2"},
		ContainerDefinitions:    defs,
		NetworkMode:             cflib.String(td.networkMode),
		Tags: sourceTags(tags.Tag{
			Key:   "o5-deployment-version",
			Value: cloudformation.Ref(VersionTagParameter),
		}),
		Volumes: []ecs.TaskDefinition_Volume{{
			Name: cflib.String("sockets"),
		}},
	}

	if td.needsDockerVolume {
		td.policy.AddECRPull()
		taskDefinition.Volumes = append(taskDefinition.Volumes, ecs.TaskDefinition_Volume{
			Name: cflib.String("docker-socket"),
			Host: &ecs.TaskDefinition_HostVolumeProperties{
				SourcePath: cflib.String("/var/run/docker.sock"),
			},
		})
	}
	taskDefinition.TaskRoleArn = td.applyRole(template).RefPtr()

	tdResource := cflib.NewResource(td.relativeName, taskDefinition)
	for _, param := range params {
		tdResource.AddParameter(param)
	}
	template.AddResource(tdResource)
	return tdResource.Ref(), nil
}
