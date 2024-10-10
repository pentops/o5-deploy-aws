package appbuilder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/iancoleman/strcase"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"golang.org/x/exp/maps"
)

type SidecarBuilder struct {
	container *ecs.TaskDefinition_ContainerDefinition

	isRequired bool

	serviceEndpoints map[string]struct{}
	links            map[string]struct{}
	proxyDBs         map[string]struct{}
	outboxDBs        map[string]struct{}

	dbEndpoints map[string]DatabaseRef
}

func NewSidecarBuilder(appName string) *SidecarBuilder {
	runtimeSidecar := &ecs.TaskDefinition_ContainerDefinition{
		Name:      O5SidecarContainerName,
		Essential: cflib.Bool(true),
		Image:     cloudformation.Ref(O5SidecarImageParameter),
		Cpu:       cflib.Int(128),
		Memory:    cflib.Int(128),
		PortMappings: []ecs.TaskDefinition_PortMapping{{
			ContainerPort: cflib.Int(8080),
		}},
		Environment: []ecs.TaskDefinition_KeyValuePair{{
			Name:  cflib.String("APP_NAME"),
			Value: cflib.String(appName),
		}, {
			Name:  cflib.String("ENVIRONMENT_NAME"),
			Value: cflib.String(cloudformation.Ref(EnvNameParameter)),
		}, {
			Name:  cflib.String("AWS_REGION"),
			Value: cflib.String(cloudformation.Ref(AWSRegionParameter)),
		}},
	}

	sb := &SidecarBuilder{
		container: runtimeSidecar,

		serviceEndpoints: make(map[string]struct{}),
		links:            make(map[string]struct{}),
		proxyDBs:         make(map[string]struct{}),
		outboxDBs:        make(map[string]struct{}),

		dbEndpoints: make(map[string]DatabaseRef),
	}

	return sb
}

func (sb *SidecarBuilder) IsRequired() bool {
	return sb.isRequired
}

func (sb *SidecarBuilder) Build() (*ecs.TaskDefinition_ContainerDefinition, error) {

	// Links adds explicit network links for each referenced container. While
	// all documentation points to this not being required... it totally is
	{
		links := maps.Keys(sb.links)
		sort.Strings(links)
		linkPairs := make([]string, 0, len(links))
		for _, link := range links {
			linkPairs = append(linkPairs, fmt.Sprintf("%s:%s", link, link))
		}
		sb.container.Links = linkPairs
	}

	if err := sb.setEnvValFromMapKeys("POSTGRES_OUTBOX", sb.outboxDBs); err != nil {
		return nil, err
	}
	if err := sb.setEnvValFromMapKeys("POSTGRES_IAM_PROXY", sb.proxyDBs); err != nil {
		return nil, err
	}
	if err := sb.setEnvValFromMapKeys("SERVICE_ENDPOINT", sb.serviceEndpoints); err != nil {
		return nil, err
	}

	for envVarSuffix, db := range sb.dbEndpoints {
		envVarName := "DB_CREDS_" + envVarSuffix
		if endpointParam, ok := db.EndpointParameter(); ok {
			sb.container.Environment = append(sb.container.Environment, ecs.TaskDefinition_KeyValuePair{
				Name:  cflib.String(envVarName),
				Value: endpointParam.RefPtr(),
			})
		} else if secretVal, ok := db.SecretValueFrom(); ok {
			sb.container.Secrets = append(sb.container.Secrets, ecs.TaskDefinition_Secret{
				Name:      envVarName,
				ValueFrom: secretVal.Ref(),
			})
		} else {
			return nil, fmt.Errorf("outbox database %s is not a proxy and has no secret", db.Name())
		}
	}

	return sb.container, nil
}

func (sb *SidecarBuilder) setEnvValFromMapKeys(envName string, m map[string]struct{}) error {
	if len(m) == 0 {
		return nil
	}
	keys := maps.Keys(m)
	sort.Strings(keys)
	return sb.setEnv(envName, strings.Join(keys, ","))
}

func (sb *SidecarBuilder) mustSetEnv(name, value string) {
	if err := sb.setEnv(name, value); err != nil {
		panic(err)
	}
}

func (sb *SidecarBuilder) setEnv(name, value string) error {
	for _, env := range sb.container.Environment {
		if *env.Name == name {
			existingValue := *env.Value
			if existingValue == value {
				return nil
			}
			return fmt.Errorf("env var %s already set to %s, cannot set to %s", name, existingValue, value)
		}
	}

	sb.container.Environment = append(sb.container.Environment, ecs.TaskDefinition_KeyValuePair{
		Name:  cflib.String(name),
		Value: cflib.String(value),
	})

	return nil
}

func (sb *SidecarBuilder) SetWorkerConfig(cfg *application_pb.WorkerConfig) error {
	if cfg.DeadletterChance > 0 {
		if err := sb.setEnv("DEADLETTER_CHANCE", fmt.Sprintf("%v", cfg.DeadletterChance)); err != nil {
			return err
		}

	}
	if cfg.ReplayChance > 0 {
		if err := sb.setEnv("RESEND_CHANCE", fmt.Sprintf("%v", cfg.ReplayChance)); err != nil {
			return err
		}
	}
	if cfg.NoDeadletters {
		if err := sb.setEnv("NO_DEADLETTERS", "true"); err != nil {
			return err
		}
	}

	return nil
}

func (sb *SidecarBuilder) PublishToEventBridge() {
	sb.mustSetEnv("EVENTBRIDGE_ARN", cloudformation.Ref(EventBusARNParameter))
}

func (sb *SidecarBuilder) SubscribeSQS(urlRef string) error {
	sb.isRequired = true
	return sb.setEnv("SQS_URL", urlRef)

}

func (sb *SidecarBuilder) ServePublic() {
	sb.isRequired = true
	sb.mustSetEnv("PUBLIC_ADDR", ":8080")
	sb.mustSetEnv("JWKS", cloudformation.Ref(JWKSParameter))
	sb.mustSetEnv("CORS_ORIGINS", cloudformation.Ref(CORSOriginParameter))
	sb.container.PortMappings = append(sb.container.PortMappings, ecs.TaskDefinition_PortMapping{
		ContainerPort: cflib.Int(8080),
	})
}

func (sb *SidecarBuilder) AddAppEndpoint(containerName string, port int64) {
	sb.isRequired = true
	sb.links[containerName] = struct{}{}
	addr := fmt.Sprintf("%s:%d", containerName, port)
	sb.serviceEndpoints[addr] = struct{}{}
}

func (sb *SidecarBuilder) ServeAdapter() {
	sb.isRequired = true
	sb.mustSetEnv("ADAPTER_ADDR", fmt.Sprintf(":%d", O5SidecarInternalPort))
}

func (sb *SidecarBuilder) ProxyDB(db DatabaseRef) string {
	sb.isRequired = true
	envVarValue := buildDBProxyEnvVal(db.Name(), O5SidecarContainerName)
	envVarName := strcase.ToScreamingSnake(db.Name())
	sb.proxyDBs[envVarName] = struct{}{}
	sb.dbEndpoints[envVarName] = db
	return envVarValue
}

func (sb *SidecarBuilder) RunOutbox(db DatabaseRef) error {
	sb.isRequired = true
	envVarName := strcase.ToScreamingSnake(db.Name())
	sb.outboxDBs[envVarName] = struct{}{}
	sb.dbEndpoints[envVarName] = db
	return nil
}

func buildDBProxyEnvVal(dbName string, proxyHost string) string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s", dbName, "proxy", proxyHost, dbName)
}
