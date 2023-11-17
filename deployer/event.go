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
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/encoding/protojson"
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
	SideEffect(outbox.OutboxMessage)

	ResolveParameters([]*deployer_pb.Parameter) ([]*deployer_pb.CloudFormationStackParameter, error)

	buildMigrationRequest(ctx context.Context, deployment *deployer_pb.DeploymentState, migration *deployer_pb.DatabaseMigrationState) (*deployer_tpb.RunDatabaseMigrationMessage, error)
}

type transitionData struct {
	sideEffects []outbox.OutboxMessage
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

	if db.MigrationTaskOutputName != nil {
		if migrationTaskARN == "" {
			return nil, fmt.Errorf("no migration task found for database %s", db.Database.Name)
		}
		if secretARN == "" {
			return nil, fmt.Errorf("no migration secret found for database %s", db.Database.Name)
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

func (td *transitionData) ResolveParameters(stackParameters []*deployer_pb.Parameter) ([]*deployer_pb.CloudFormationStackParameter, error) {

	parameters := make([]*deployer_pb.CloudFormationStackParameter, 0, len(stackParameters))

	for _, param := range stackParameters {
		parameter, err := td.parameterResolver.ResolveParameter(param)
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

func (td *transitionData) SideEffect(msg outbox.OutboxMessage) {
	td.sideEffects = append(td.sideEffects, msg)
}

func (d *DeploymentManager) findTransition(ctx context.Context, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) ITransitionSpec {
	for _, search := range transitions {
		if search.Matches(deployment, event) {
			return search
		}
	}
	return nil
}

func (dd *DeploymentManager) RegisterEvent(ctx context.Context, outerEvent *deployer_pb.DeploymentEvent) error {

	runTransition := func(ctx context.Context, tx TransitionTransaction, transition TransitionBaton, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {

		spec := dd.findTransition(ctx, deployment, event)
		if spec == nil {
			typeKey, ok := event.Event.TypeKey()
			if !ok {
				return fmt.Errorf("unknown event type: %T", event.Event)
			}
			// TODO: This by generation and annotation
			if stackStatus := event.Event.GetStackStatus(); stackStatus != nil {
				typeKey = deployer_pb.DeploymentEventTypeKey(fmt.Sprintf("%s.%s", typeKey, stackStatus.Lifecycle.ShortString()))
			}
			return fmt.Errorf("no transition found for status %s -> %s",
				deployment.Status.ShortString(),
				typeKey,
			)
		}

		if err := spec.RunTransition(ctx, transition, deployment, event); err != nil {
			return err
		}

		if err := tx.StoreDeploymentEvent(ctx, deployment, event); err != nil {
			return err
		}

		return nil
	}

	return dd.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		deployment, err := tx.GetDeployment(ctx, outerEvent.DeploymentId)
		if errors.Is(err, DeploymentNotFoundError) {
			deployment = &deployer_pb.DeploymentState{
				DeploymentId: outerEvent.DeploymentId,
			}
		} else if err != nil {
			return err
		}

		spec := deployment.Spec
		if spec == nil {
			trigger := outerEvent.Event.GetTriggered()
			if trigger == nil {
				return fmt.Errorf("no spec found for deployment %s, and the event is not an initiating event", outerEvent.DeploymentId)
			}
			spec = trigger.Spec
		}

		environment, err := tx.GetEnvironment(ctx, spec.EnvironmentName)
		if err != nil {
			return err
		}

		deployerResolver, err := BuildParameterResolver(ctx, environment)
		if err != nil {
			return err
		}

		// Recursive function, so that the event and all chained events are
		// handled atomically.
		// Deployment is modified in place by the transition, and stored at the
		// end with the event. We then pass the same deployment pointer through
		// to the next event as-is, which should match the stored state
		var runOne func(event *deployer_pb.DeploymentEvent) error
		runOne = func(innerEvent *deployer_pb.DeploymentEvent) error {
			baton := &transitionData{
				parameterResolver: deployerResolver,
				env:               environment,
			}
			typeKey, _ := innerEvent.Event.TypeKey()

			ctx = log.WithFields(ctx, map[string]interface{}{
				"deploymentId": innerEvent.DeploymentId,
				"eventType":    typeKey,
				"stateBefore":  deployment.Status.ShortString(),
			})
			log.WithField(ctx, "event", protojson.Format(innerEvent.Event)).Debug("Begin Deployment Event")
			if err := runTransition(ctx, tx, baton, deployment, innerEvent); err != nil {
				log.WithError(ctx, err).Error("Running Deployment Tarnsition")
				return err
			}

			log.WithFields(ctx, map[string]interface{}{
				"stateAfter": deployment.Status.ShortString(),
			}).Info("Deployment Event Handled")

			for _, se := range baton.sideEffects {
				if err := tx.PublishEvent(ctx, se); err != nil {
					return err
				}
			}

			for _, nextEvent := range baton.chainEvents {
				if err := runOne(nextEvent); err != nil {
					return err
				}
			}
			return nil
		}

		return runOne(outerEvent)

	})
}
