package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"github.com/pkg/errors"
)

type AppInput struct {
	Application *application_pb.Application

	RDSHosts   RDSHostLookup
	VersionTag string
}

func (ai AppInput) Validate() error {
	if ai.Application == nil {
		return fmt.Errorf("application is required")
	}
	if ai.RDSHosts == nil {
		return fmt.Errorf("RDSHosts is required")
	}
	if ai.VersionTag == "" {
		return fmt.Errorf("VersionTag is required")
	}
	return nil
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

func updateApp(app *application_pb.Application) {
	for _, db := range app.Databases {
		pg := db.GetPostgres()
		if pg == nil {
			continue
		}

		if pg.DbName != "" && pg.DbNameSuffix == "" {
			pg.DbNameSuffix = pg.DbName
		}
		pg.DbName = ""
	}
}

func BuildApplication(spec AppInput) (*BuiltApplication, error) {
	updateApp(spec.Application)
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

		runtimeStack.TaskDefinition.AddEventBridgeTargets(app.Targets)

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

		dbRef, ok := bb.Database(dbSpec.Name)
		if !ok {
			// panic because this is a logic issue in the code, not a user error
			panic(fmt.Sprintf("database %s not found for outbox, but was defined in spec", dbSpec.Name))
		}

		// TODO: Assign to the first runtime which references the DB.
		if err := firstRuntime.TaskDefinition.Sidecar.RunOutbox(dbRef, pg.OutboxDelayable); err != nil {
			return nil, err
		}
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
	Template *cflib.TemplateBuilder

	dbs []*awsdeployer_pb.PostgresDatabaseResource

	input AppInput
}

func NewBuilder(input AppInput) (*Builder, resourceBuilder, error) {
	err := input.Validate()
	if err != nil {
		return nil, nil, errors.WithStack(fmt.Errorf("AppInput invalid: %w", err))
	}

	template := cflib.NewTemplateBuilder()

	global := &globals{
		spec:     input.Application,
		rdsHosts: input.RDSHosts,

		databases: map[string]DatabaseRef{},
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
