package deployer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeployerWorker struct {
	*deployer_tpb.UnimplementedDeployerTopicServer
	storage DeployerStorage
}

func NewDeployerWorker(store DeployerStorage) (*DeployerWorker, error) {
	return &DeployerWorker{
		storage: store,
	}, nil
}

func (dw *DeployerWorker) doDeploymentEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {
	if err := dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return doDeploymentEvent(ctx, tx, event)
	}); err != nil {
		return err
	}
	return nil
}

type Eventer[WrappedEvent proto.Message, Event any, State proto.Message] struct {
	wrapEvent      func(Event) WrappedEvent
	unwrapEvent    func(WrappedEvent) Event
	runEvent       func(context.Context, TransitionTransaction, TransitionBaton[Event], Event) error
	findTransition func(State, Event) (ITransitionSpec[State, Event], error)
	storeEvent     func(context.Context, TransitionTransaction, State, WrappedEvent) error
	stateLabel     func(State) string
	eventLabel     func(Event) string
}

func (ee Eventer[WrappedEvent, Event, State]) Run(ctx context.Context, tx TransitionTransaction, state State, outerEvent WrappedEvent) error {

	eventQueue := []WrappedEvent{outerEvent}

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		baton := &TransitionData[Event]{}

		unwrapped := ee.unwrapEvent(innerEvent)

		transition, err := ee.findTransition(state, unwrapped)
		if err != nil {
			return err
		}

		typeKey := ee.eventLabel(unwrapped)
		stateBefore := ee.stateLabel(state)

		ctx = log.WithFields(ctx, map[string]interface{}{
			"eventType":  typeKey,
			"transition": fmt.Sprintf("%s -> ? : %s", stateBefore, typeKey),
		})

		log.Debug(ctx, "Begin Event")

		if err := transition.RunTransition(ctx, baton, state, unwrapped); err != nil {
			log.WithError(ctx, err).Error("Running Transition")
			return err
		}

		ctx = log.WithFields(ctx, map[string]interface{}{
			"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, ee.stateLabel(state), typeKey),
		})

		log.Info(ctx, "Event Handled")

		if err := ee.storeEvent(ctx, tx, state, innerEvent); err != nil {
			return err
		}

		for _, se := range baton.SideEffects {
			if err := tx.PublishEvent(ctx, se); err != nil {
				return fmt.Errorf("publishEvent: %w", err)
			}
		}

		for _, event := range baton.ChainEvents {
			wrappedEvent := ee.wrapEvent(event)
			eventQueue = append(eventQueue, wrappedEvent)
		}
	}

	return nil
}

func doDeploymentEvent(ctx context.Context, tx TransitionTransaction, outerEvent *deployer_pb.DeploymentEvent) error {

	deployment, err := tx.GetDeployment(ctx, outerEvent.DeploymentId)
	if errors.Is(err, DeploymentNotFoundError) {
		trigger := outerEvent.Event.GetCreated()
		if trigger == nil {
			return fmt.Errorf("deployment %s not found, and the event is not an initiating event", outerEvent.DeploymentId)
		}

		deployment = &deployer_pb.DeploymentState{
			DeploymentId: outerEvent.DeploymentId,
			Spec:         trigger.Spec,
		}
	} else if err != nil {
		return err
	}

	ctx = log.WithFields(ctx, map[string]interface{}{
		"deploymentId": deployment.DeploymentId,
	})

	ee := &Eventer[*deployer_pb.DeploymentEvent, deployer_pb.IsDeploymentEventTypeWrappedType, *deployer_pb.DeploymentState]{
		wrapEvent: func(event deployer_pb.IsDeploymentEventTypeWrappedType) *deployer_pb.DeploymentEvent {
			wrappedEvent := &deployer_pb.DeploymentEvent{
				DeploymentId: deployment.DeploymentId,
				Metadata: &deployer_pb.EventMetadata{
					EventId:   uuid.NewString(),
					Timestamp: timestamppb.Now(),
				},
				Event: &deployer_pb.DeploymentEventType{},
			}
			wrappedEvent.Event.Set(event)
			return wrappedEvent
		},
		unwrapEvent: func(event *deployer_pb.DeploymentEvent) deployer_pb.IsDeploymentEventTypeWrappedType {
			return event.Event.Get()
		},
		findTransition: findTransition,
		storeEvent: func(ctx context.Context, tx TransitionTransaction, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
			return tx.StoreDeploymentEvent(ctx, deployment, event)
		},
		stateLabel: func(state *deployer_pb.DeploymentState) string {
			return state.Status.ShortString()
		},
		eventLabel: func(event deployer_pb.IsDeploymentEventTypeWrappedType) string {
			return string(event.TypeKey())
		},
	}

	return ee.Run(ctx, tx, deployment, outerEvent)
}

func doStackEvent(ctx context.Context, tx TransitionTransaction, stack *deployer_pb.StackState, outerEvent *deployer_pb.StackEvent) error {
	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackId": stack.StackId,
	})

	ee := &Eventer[*deployer_pb.StackEvent, deployer_pb.IsStackEventTypeWrappedType, *deployer_pb.StackState]{
		wrapEvent: func(event deployer_pb.IsStackEventTypeWrappedType) *deployer_pb.StackEvent {
			wrappedEvent := &deployer_pb.StackEvent{
				StackId: stack.StackId,
				Metadata: &deployer_pb.EventMetadata{
					EventId:   uuid.NewString(),
					Timestamp: timestamppb.Now(),
				},
				Event: &deployer_pb.StackEventType{},
			}
			wrappedEvent.Event.Set(event)
			return wrappedEvent
		},
		unwrapEvent: func(event *deployer_pb.StackEvent) deployer_pb.IsStackEventTypeWrappedType {
			return event.Event.Get()
		},
		findTransition: findStackTransition,
		storeEvent: func(ctx context.Context, tx TransitionTransaction, stack *deployer_pb.StackState, event *deployer_pb.StackEvent) error {
			return tx.StoreStackEvent(ctx, stack, event)
		},
		stateLabel: func(state *deployer_pb.StackState) string {
			return state.Status.String()
		},
		eventLabel: func(event deployer_pb.IsStackEventTypeWrappedType) string {
			return string(event.TypeKey())
		},
	}

	return ee.Run(ctx, tx, stack, outerEvent)
}

func (dw *DeployerWorker) TriggerDeployment(ctx context.Context, msg *deployer_tpb.TriggerDeploymentMessage) (*emptypb.Empty, error) {

	// SIDE EFFECT
	triggerDeploymentEvent := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Triggered_{
				Triggered: &deployer_pb.DeploymentEventType_Triggered{},
			},
		},
	}

	if err := dw.doDeploymentEvent(ctx, triggerDeploymentEvent); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) RequestDeployment(ctx context.Context, msg *deployer_tpb.RequestDeploymentMessage) (*emptypb.Empty, error) {
	createDeploymentEvent := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Created_{
				Created: &deployer_pb.DeploymentEventType_Created{
					Spec: msg.Spec,
				},
			},
		},
	}

	evt := &deployer_pb.StackEvent{
		StackId: StackID(msg.Spec.EnvironmentName, msg.Spec.AppName),
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.StackEventType{
			Type: &deployer_pb.StackEventType_Triggered_{
				Triggered: &deployer_pb.StackEventType_Triggered{
					Deployment: &deployer_pb.StackDeployment{
						DeploymentId: msg.DeploymentId,
						Version:      msg.Spec.Version,
					},
				},
			},
		},
	}

	if err := dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		stack, err := tx.GetStack(ctx, evt.StackId)
		if errors.Is(err, StackNotFoundError) {
			stack = &deployer_pb.StackState{
				StackId:           evt.StackId,
				Status:            deployer_pb.StackStatus_UNSPECIFIED, // New
				CurrentDeployment: nil,
				ApplicationName:   msg.Spec.AppName,
				EnvironmentName:   msg.Spec.EnvironmentName,
				QueuedDeployments: []*deployer_pb.StackDeployment{},
			}

		} else if err != nil {
			return err
		}

		if err := doStackEvent(ctx, tx, stack, evt); err != nil {
			return err
		}

		return doDeploymentEvent(ctx, tx, createDeploymentEvent)
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) DeploymentComplete(ctx context.Context, msg *deployer_tpb.DeploymentCompleteMessage) (*emptypb.Empty, error) {

	stackEvent := &deployer_pb.StackEvent{
		StackId:  StackID(msg.EnvironmentName, msg.ApplicationName),
		Metadata: &deployer_pb.EventMetadata{},
		Event: &deployer_pb.StackEventType{
			Type: &deployer_pb.StackEventType_DeploymentCompleted_{
				DeploymentCompleted: &deployer_pb.StackEventType_DeploymentCompleted{
					Deployment: &deployer_pb.StackDeployment{
						DeploymentId: msg.DeploymentId,
						Version:      msg.Version,
					},
				},
			},
		},
	}

	if err := dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		stack, err := tx.GetStack(ctx, stackEvent.StackId)
		if err != nil {
			return err
		}
		return doStackEvent(ctx, tx, stack, stackEvent)
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func TranslateStackStatusChanged(msg *deployer_tpb.StackStatusChangedMessage) (*deployer_pb.DeploymentEvent, error) {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: msg.StackId.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_StackStatus_{
				StackStatus: &deployer_pb.DeploymentEventType_StackStatus{
					Lifecycle:   msg.Lifecycle,
					FullStatus:  msg.Status,
					StackOutput: msg.Outputs,
				},
			},
		},
	}, nil
}

func (dw *DeployerWorker) StackStatusChanged(ctx context.Context, msg *deployer_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := TranslateStackStatusChanged(msg)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)
}

func TranslateMigrationStatusChanged(msg *deployer_tpb.MigrationStatusChangedMessage) (*deployer_pb.DeploymentEvent, error) {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},

		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_DbMigrateStatus{
				DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
					MigrationId: msg.MigrationId,
					Status:      msg.Status,
				},
			},
		},
	}, nil
}

func (dw *DeployerWorker) MigrationStatusChanged(ctx context.Context, msg *deployer_tpb.MigrationStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := TranslateMigrationStatusChanged(msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)

}
