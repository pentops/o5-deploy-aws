package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
)

type AppInput struct {
	Application *application_pb.Application

	RDSHosts   RDSHostLookup
	VersionTag string
}

type RDSHostMap map[string]*RDSHost

type RDSHostLookup interface {
	FindRDSHost(string) (*RDSHost, bool)
}

func (r RDSHostMap) FindRDSHost(serverGroup string) (*RDSHost, bool) {
	host, ok := r[serverGroup]
	return host, ok
}

type RDSAuthType int

const (
	RDSAuthTypeIAM = iota
	RDSAuthTypeSecretsManager
)

type RDSHost struct {
	AuthType environment_pb.RDSAuthTypeKey
}

func BuildApplication(spec AppInput) (*BuiltApplication, error) {
	app := spec.Application

	bb, resourceBuilder, err := NewBuilder(spec)
	if err != nil {
		return nil, err
	}

	err = mapResources(bb, resourceBuilder, app)
	if err != nil {
		return nil, err
	}

	{ // SNS Topics
		needsDeadLetter := false
		if len(app.Targets) > 0 {
			needsDeadLetter = true
		} else {
			for _, runtime := range app.Runtimes {
				if len(runtime.Subscriptions) > 0 {
					needsDeadLetter = true
				}
			}
		}

		app.Targets = append(app.Targets, &application_pb.Target{
			Name: O5MonitorTargetName,
		})
		if needsDeadLetter {
			app.Targets = append(app.Targets, &application_pb.Target{
				Name: DeadLetterTargetName,
			})
		}

	}

	listener := NewListenerRuleSet()

	ecsServices := make([]*RuntimeService, 0, len(app.Runtimes))
	for _, runtime := range app.Runtimes {
		runtimeStack, err := NewRuntimeService(bb, runtime)
		if err != nil {
			return nil, err
		}
		ecsServices = append(ecsServices, runtimeStack)

		if err := runtimeStack.AddRoutes(listener); err != nil {
			return nil, fmt.Errorf("adding routes to %s: %w", runtime.Name, err)
		}

		for _, target := range app.Targets {
			runtimeStack.Policy.AddEventBridgePublish(target.Name)
		}

	}

	if len(ecsServices) < 1 {
		return nil, fmt.Errorf("no runtimes defined")
	}
	firstRuntime := ecsServices[0]
	for _, dbSpec := range app.Databases {
		pg := dbSpec.GetPostgres()
		if pg == nil || !pg.RunOutbox {
			continue
		}
		dbRef, ok := bb.Globals.Database(dbSpec.Name)
		if !ok {
			// panic because this is a logic issue in the code, not a user error
			panic(fmt.Sprintf("database %s not found for outbox, but was defined in spec", dbSpec.Name))
		}
		firstRuntime.outboxDatabases = append(firstRuntime.outboxDatabases, dbRef)
	}

	for _, runtime := range ecsServices {
		if err := runtime.AddTemplateResources(bb.Template); err != nil {
			return nil, fmt.Errorf("adding %s: %w", runtime.Name, err)
		}
	}

	listener.AddTemplateResources(bb.Template)

	return bb.Export(), nil
}

type Builder struct {
	Globals
	Template *cf.TemplateBuilder

	dbs []*awsdeployer_pb.PostgresDatabaseResource

	input AppInput
}

func NewBuilder(input AppInput) (*Builder, resourceBuilder, error) {

	template := cf.NewTemplateBuilder()

	global := &globals{
		spec:     input.Application,
		rdsHosts: input.RDSHosts,

		databases: map[string]DatabaseReference{},
		secrets:   map[string]*secretInfo{},
		buckets:   map[string]*bucketInfo{},
	}

	bb := &Builder{
		Globals:  global,
		Template: template,
		input:    input,
	}

	addGlobalParameters(bb, input.VersionTag)

	return bb, global, nil
}

func (bb *Builder) AddPostgresResource(pg *awsdeployer_pb.PostgresDatabaseResource) {
	bb.dbs = append(bb.dbs, pg)
}

type BuiltApplication struct {
	Template   *cloudformation.Template
	Parameters []*awsdeployer_pb.Parameter

	Databases []*awsdeployer_pb.PostgresDatabaseResource
	Name      string
	Version   string
}

func (bb *Builder) Export() *BuiltApplication {
	template := bb.Template.Build()
	return &BuiltApplication{
		Template:   template.Template,
		Parameters: template.Parameters,
		Databases:  bb.dbs,

		Name:    bb.input.Application.Name,
		Version: bb.input.VersionTag,
	}
}
