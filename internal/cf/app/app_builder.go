package app

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/cf"
)

func BuildApplication(app *application_pb.Application, versionTag string) (*BuiltApplication, error) {

	bb, resourceBuilder, err := NewBuilder(app, versionTag)
	if err != nil {
		return nil, err
	}

	addGlobalParameters(bb, versionTag)

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

	built *awsdeployer_pb.BuiltApplication
}

func NewBuilder(spec *application_pb.Application, versionTag string) (*Builder, resourceBuilder, error) {

	template := cf.NewTemplateBuilder()

	builtApp := &awsdeployer_pb.BuiltApplication{
		Name:    spec.Name,
		Version: versionTag,
	}

	if spec.DeploymentConfig != nil {
		if spec.DeploymentConfig.QuickMode {
			builtApp.QuickMode = true
		}
	}

	global := &globals{
		spec: spec,

		databases: map[string]DatabaseReference{},
		secrets:   map[string]*secretInfo{},
		buckets:   map[string]*bucketInfo{},
	}

	return &Builder{
		Globals:  global,
		Template: template,
		built:    builtApp,
	}, global, nil
}

func (bb *Builder) AddPostgresResource(pg *awsdeployer_pb.PostgresDatabaseResource) {
	bb.built.PostgresDatabases = append(bb.built.PostgresDatabases, pg)
}

type BuiltApplication struct {
	Template *cloudformation.Template
	*awsdeployer_pb.BuiltApplication
}

func (bb *Builder) Export() *BuiltApplication {
	template := bb.Template.Build()
	built := bb.built
	built.Parameters = template.Parameters
	return &BuiltApplication{
		Template:         template.Template,
		BuiltApplication: built,
	}
}
