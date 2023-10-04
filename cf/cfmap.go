package cf

import (

	// "github.com/aws/aws-cdk-go/awscdk/v2/awssqs"

	"crypto/sha1"
	"fmt"
	"regexp"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	elbv2 "github.com/awslabs/goformation/v7/cloudformation/elasticloadbalancingv2"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

const (
	ECSClusterParameter           = "ECSCluster"
	ECSRepoParameter              = "ECSRepo"
	ECSTaskExecutionRoleParameter = "ECSTaskExecutionRole"
	VersionTagParameter           = "VersionTag"
	ListenerARNParameter          = "ListenerARN"
	EnvNameParameter              = "EnvName"
	VPCParameter                  = "VPCID"

	O5SidecarContainerName = "o5_runtime"
	O5SidecarImageName     = "ghcr.io/pentops/o5-runtime-sidecar:latest"
)

type globalData struct {
	appName string

	databases map[string]DatabaseReference
}

// DatabaseReference is used to look up parameters ECS Task Definitions
type DatabaseReference struct {
	Definition     *application_pb.Database
	SecretResource *Resource[*secretsmanager.Secret]
}

func BuildCloudformation(app *application_pb.Application, env *environment_pb.Environment) (*Template, error) {

	stackTemplate := NewTemplate()

	global := globalData{
		appName:   app.Name,
		databases: map[string]DatabaseReference{},
	}

	runtimes := map[string]*RuntimeService{}

	for _, database := range app.Databases {

		switch dbType := database.Engine.(type) {
		case *application_pb.Database_Postgres_:

			parameterName := fmt.Sprintf("DatabaseSecret%s", strings.Title(database.Name))

			secret := NewResource(parameterName, &secretsmanager.Secret{
				Name:        Stringf("/%s/%s/%s", env.FullName, app.Name, database.Name),
				Description: Stringf("Secret for Postgres database %s in app %s environment %s", database.Name, app.Name, env.FullName),
			})

			addStackResource(stackTemplate.template, secret)

			ref := DatabaseReference{
				SecretResource: secret,
				Definition:     database,
			}
			global.databases[database.Name] = ref

			def := &PostgresDefinition{
				Databse:  database,
				Postgres: dbType.Postgres,
				Secret:   secret,
			}

			if dbType.Postgres.MigrateContainer != nil {
				if dbType.Postgres.MigrateContainer.Name == "" {
					dbType.Postgres.MigrateContainer.Name = "migrate"
				}

				// TODO: This takes the global var, which is added to by other
				// databases, so this whole step should be deferred until all
				// other databases (and likely other resources) are created.
				// Not likely to be a problem any time soon so long as THIS
				// database is added early which it is.
				migrationContainer, err := buildContainer(global, dbType.Postgres.MigrateContainer)
				if err != nil {
					return nil, err
				}
				addLogs(migrationContainer, fmt.Sprintf("%s/migrate", global.appName))
				name := fmt.Sprintf("MigrationTaskDefinition%s", strings.Title(database.Name))

				migrationTaskDefinition := NewResource(name, &ecs.TaskDefinition{
					ContainerDefinitions: []ecs.TaskDefinition_ContainerDefinition{
						*migrationContainer,
					},
					Family:                  String(fmt.Sprintf("%s_migrate_%s", global.appName, database.Name)),
					ExecutionRoleArn:        cloudformation.RefPtr(ECSTaskExecutionRoleParameter),
					RequiresCompatibilities: []string{"EC2"},
				})
				addStackResource(stackTemplate.template, migrationTaskDefinition)
				def.MigrationTaskOutputName = String(name)
				stackTemplate.template.Outputs[name] = cloudformation.Output{
					Value: migrationTaskDefinition.Ref(),
				}
			}

			stackTemplate.postgresDatabases = append(stackTemplate.postgresDatabases, def)

		default:
			return nil, fmt.Errorf("unknown database type %T", dbType)
		}
	}

	for _, runtime := range app.Runtimes {
		runtimeStack, err := NewRuntimeService(global, runtime)
		if err != nil {
			return nil, err
		}
		runtimes[runtime.Name] = runtimeStack
	}

	listener := NewListenerRuleSet()

	for _, ingress := range app.Ingress {
		for _, route := range ingress.HttpRoutes {
			runtime, ok := runtimes[route.TargetRuntime]
			if !ok {
				return nil, fmt.Errorf("runtime %s not found for route %s", route.TargetRuntime, ingress.Name)
			}

			targetGroup := runtime.LazyHTTPTargetGroup()
			listener.AddRoute(targetGroup, route.Prefix)
		}

		for _, route := range ingress.GrpcRoutes {
			runtime, ok := runtimes[route.TargetRuntime]
			if !ok {
				return nil, fmt.Errorf("runtime %s not found for route %s", route.TargetRuntime, ingress.Name)
			}

			targetGroup := runtime.LazyGRPCTargetGroup()
			listener.AddRoute(targetGroup, route.Prefix)
		}
	}

	for _, runtime := range runtimes {
		runtime.Apply(stackTemplate.template)
	}

	listener.Apply(stackTemplate.template)

	return stackTemplate, nil
}

type ListenerRuleSet struct {
	Rules []*Resource[*elbv2.ListenerRule]
}

func NewListenerRuleSet() *ListenerRuleSet {
	return &ListenerRuleSet{
		Rules: []*Resource[*elbv2.ListenerRule]{},
	}
}

var reUnsafe = regexp.MustCompile("[^a-zA-Z0-9]+")

func (ll *ListenerRuleSet) AddRoute(targetGroup *Resource[*elbv2.TargetGroup], prefix string) {
	hash := sha1.New()
	hash.Write([]byte(prefix))
	pathClean := fmt.Sprintf("%x", hash.Sum(nil))

	name := fmt.Sprintf("ListenerRule%s", pathClean)
	priority := fmt.Sprintf("ListenerRulePriority%s", pathClean)
	rule := &elbv2.ListenerRule{
		//Priority:    cloudformation.Ref(priority),
		ListenerArn: cloudformation.RefPtr(ListenerARNParameter),
		Actions: []elbv2.ListenerRule_Action{{
			Type: "forward",
			ForwardConfig: &elbv2.ListenerRule_ForwardConfig{
				TargetGroups: []elbv2.ListenerRule_TargetGroupTuple{{
					TargetGroupArn: targetGroup.Ref(),
				}},
			},
		}},
		Conditions: []elbv2.ListenerRule_RuleCondition{{
			Field:  String("path-pattern"),
			Values: []string{fmt.Sprintf("%s*", prefix)},
		}},
	}

	resource := &Resource[*elbv2.ListenerRule]{
		Name:     name,
		Resource: rule,
		Overrides: map[string]string{
			"Priority": cloudformation.Ref(priority),
		},
		Parameters: []Parameter{{
			Name:        priority,
			Type:        "Number",
			Description: fmt.Sprintf(":o5:priority:%s", prefix),
		}},
	}

	ll.Rules = append(ll.Rules, resource)
}

func (ll *ListenerRuleSet) Apply(template *cloudformation.Template) {
	for _, rule := range ll.Rules {
		addStackResource(template, rule)
	}
}

func addStackResource[T cloudformation.Resource](template *cloudformation.Template, resource *Resource[T]) {
	template.Resources[resource.Name] = resource
	for _, param := range resource.Parameters {
		template.Parameters[param.Name] = cloudformation.Parameter{
			Description: String(param.Description),
			Type:        param.Type,
			Default:     param.Default,
		}
	}
}
