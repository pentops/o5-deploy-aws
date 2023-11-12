package deployer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransitionSpec[Event deployer_pb.IsDeploymentEventTypeWrappedType] struct {
	FromStatus  []deployer_pb.DeploymentStatus
	EventFilter func(Event) bool
	Transition  func(context.Context, TransitionBaton, *deployer_pb.DeploymentState, Event) error
}

func (ts TransitionSpec[Event]) RunTransition(ctx context.Context, tb TransitionBaton, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {

	asType, ok := event.Event.Get().(Event)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event.Event.Get())
	}

	return ts.Transition(ctx, tb, deployment, asType)
}

func (ts TransitionSpec[Event]) Matches(deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) bool {
	got := event.Event.Get()
	if got == nil {
		return false
	}
	asType, ok := got.(Event)
	if !ok {
		return false
	}
	didMatch := false
	for _, fromStatus := range ts.FromStatus {
		if fromStatus == deployment.Status {
			didMatch = true
			break
		}
	}
	if !didMatch {
		return false
	}

	if ts.EventFilter != nil && !ts.EventFilter(asType) {
		return false
	}
	return true
}

type ITransitionSpec interface {
	Matches(*deployer_pb.DeploymentState, *deployer_pb.DeploymentEvent) bool
	RunTransition(context.Context, TransitionBaton, *deployer_pb.DeploymentState, *deployer_pb.DeploymentEvent) error
}

type TransitionBaton interface {
	ChainEvent(*deployer_pb.DeploymentEvent)
	SideEffect(proto.Message)

	ResolveParameters([]*deployer_pb.Parameter, VariableParameters) ([]*deployer_pb.KeyValue, error)

	buildMigrationRequest(ctx context.Context, deployment *deployer_pb.DeploymentState, migration *deployer_pb.DatabaseMigrationState) (*deployer_tpb.RunDatabaseMigrationMessage, error)
}

type transitionData struct {
	sideEffects []proto.Message
	chainEvents []*deployer_pb.DeploymentEvent

	parameterResolver ParameterResolver
	env               *environment_pb.Environment
}

func newEvent(d *deployer_pb.DeploymentState, event deployer_pb.IsDeploymentEventType_Type) *deployer_pb.DeploymentEvent {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: d.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: event,
		},
	}
}

func (td *transitionData) buildMigrationRequest(ctx context.Context, deployment *deployer_pb.DeploymentState, migration *deployer_pb.DatabaseMigrationState) (*deployer_tpb.RunDatabaseMigrationMessage, error) {

	var db *deployer_pb.PostgresDatabase
	for _, search := range deployment.Spec.Databases {
		if search.Database.Name == migration.DbName {
			db = search
			break
		}
	}

	if db == nil {
		return nil, fmt.Errorf("no database found with name %s", migration.DbName)
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"database":    db.Database.Name,
		"serverGroup": db.Database.GetPostgres().ServerGroup,
	})
	log.Debug(ctx, "Upsert Database")
	var migrationTaskARN string
	var secretARN string
	for _, output := range deployment.StackOutput {
		if *db.MigrationTaskOutputName == output.Name {
			migrationTaskARN = output.Value
		}
		if *db.SecretOutputName == output.Name {
			secretARN = output.Value
		}
	}

	awsEnv := td.env.GetAws()
	var secretName string
	for _, host := range awsEnv.RdsHosts {
		if host.ServerGroup == db.Database.GetPostgres().ServerGroup {
			secretName = host.SecretName
			break
		}
	}
	if secretName == "" {
		return nil, fmt.Errorf("no host found for server group %q", db.Database.GetPostgres().ServerGroup)
	}

	return &deployer_tpb.RunDatabaseMigrationMessage{
		MigrationId:       migration.MigrationId,
		DeploymentId:      deployment.DeploymentId,
		MigrationTaskArn:  migrationTaskARN,
		SecretArn:         secretARN,
		Database:          db,
		RotateCredentials: migration.RotateCredentials,
		EcsClusterName:    awsEnv.EcsClusterName,
		RootSecretName:    secretName,
	}, nil

}

func (td *transitionData) ResolveParameters(stackParameters []*deployer_pb.Parameter, variables VariableParameters) ([]*deployer_pb.KeyValue, error) {

	parameters := make([]*deployer_pb.KeyValue, 0, len(stackParameters))

	for _, param := range stackParameters {
		parameter, err := td.parameterResolver.ResolveParameter(param, variables)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s': %w", param.Name, err)
		}
		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

func (td *transitionData) ChainEvent(event *deployer_pb.DeploymentEvent) {
	td.chainEvents = append(td.chainEvents, event)
}

func (td *transitionData) SideEffect(msg proto.Message) {
	td.sideEffects = append(td.sideEffects, msg)
}

func (d *Deployer) findTransition(ctx context.Context, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) ITransitionSpec {
	for _, search := range transitions {
		if search.Matches(deployment, event) {
			return search
		}
	}
	return nil
}

func (d *Deployer) RegisterEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {

	deployment, err := d.storage.GetDeployment(ctx, event.DeploymentId)
	if errors.Is(err, DeploymentNotFoundError) {
		deployment = &deployer_pb.DeploymentState{
			DeploymentId: event.DeploymentId,
		}
	} else if err != nil {
		return err
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"deploymentId": event.DeploymentId,
		"event":        protojson.Format(event.Event),
		"stateBefore":  deployment.Status.ShortString(),
	})
	log.Debug(ctx, "Beign Deployment Event")

	// TODO: This is a heavy operation, but also needs to run
	// just-in-time, so we should only run it when the event
	// requires it.
	deployerResolver, err := d.BuildParameterResolver(ctx)
	if err != nil {
		return err
	}

	transition := &transitionData{
		parameterResolver: deployerResolver,
		env:               d.Environment,
	}

	typeKey, ok := event.Event.TypeKey()
	if !ok {
		return fmt.Errorf("unknown event type: %T", event.Event)
	}

	spec := d.findTransition(ctx, deployment, event)
	if spec == nil {
		return fmt.Errorf("no transition found for status %s -> %s",
			deployment.Status.ShortString(),
			typeKey,
		)
	}

	if err := spec.RunTransition(ctx, transition, deployment, event); err != nil {
		return err
	}

	if err := d.storage.StoreDeploymentEvent(ctx, deployment, event); err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"stateAfter": deployment.Status.ShortString(),
	}).Info("Deployment Event Handled")

	for _, se := range transition.sideEffects {
		if err := d.storage.QueueSideEffect(ctx, se); err != nil {
			return err
		}
	}

	for _, nextEvent := range transition.chainEvents {
		if err := d.storage.ChainNextEvent(ctx, nextEvent); err != nil {
			return err
		}
	}

	return nil
}
