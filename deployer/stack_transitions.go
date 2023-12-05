package deployer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.daemonl.com/sqrlx"
)

type StackTransitionBaton = psm.TransitionBaton[deployer_pb.IsStackEventTypeWrappedType]

type StackTransition[Event deployer_pb.IsStackEventTypeWrappedType] struct {
	FromStatus  []deployer_pb.StackStatus
	EventFilter func(Event) bool
	Transition  func(context.Context, StackTransitionBaton, *deployer_pb.StackState, Event) error
}

func (ts StackTransition[Event]) RunTransition(
	ctx context.Context,
	tb psm.TransitionBaton[deployer_pb.IsStackEventTypeWrappedType],
	state *deployer_pb.StackState,
	event deployer_pb.IsStackEventTypeWrappedType) error {
	asType, ok := event.(Event)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	return ts.Transition(ctx, tb, state, asType)
}

func (ts StackTransition[Event]) Matches(deployment *deployer_pb.StackState, event deployer_pb.IsStackEventTypeWrappedType) bool {
	asType, ok := event.(Event)
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

type StackEventer struct {
	psm.Eventer[
		*deployer_pb.StackState,
		deployer_pb.StackStatus,
		*deployer_pb.StackEvent,
		deployer_pb.IsStackEventTypeWrappedType,
	]
}

func (se StackEventer) handleEvent(ctx context.Context, tx sqrlx.Transaction, stack *deployer_pb.StackState, outerEvent *deployer_pb.StackEvent) error {
	ctx = log.WithFields(ctx, map[string]interface{}{
		"entityType": "Stack",
		"stackId":    stack.StackId,
	})
	wrapped := psm.NewSqrlxTransaction[*deployer_pb.StackState, *deployer_pb.StackEvent](tx, storeStackEvent)
	return se.Eventer.Run(ctx, wrapped, stack, outerEvent)
}

func NewStackEventer() (*StackEventer, error) {

	stackEventer := &StackEventer{
		Eventer: psm.Eventer[
			*deployer_pb.StackState,
			deployer_pb.StackStatus,
			*deployer_pb.StackEvent,
			deployer_pb.IsStackEventTypeWrappedType,
		]{
			WrapEvent: func(
				ctx context.Context,
				state *deployer_pb.StackState,
				event deployer_pb.IsStackEventTypeWrappedType,
			) *deployer_pb.StackEvent {
				wrappedEvent := &deployer_pb.StackEvent{
					StackId: state.StackId,
					Metadata: &deployer_pb.EventMetadata{
						EventId:   uuid.NewString(),
						Timestamp: timestamppb.Now(),
					},
					Event: &deployer_pb.StackEventType{},
				}
				wrappedEvent.Event.Set(event)
				return wrappedEvent
			},
			UnwrapEvent: func(event *deployer_pb.StackEvent) deployer_pb.IsStackEventTypeWrappedType {
				return event.Event.Get()
			},
			StateLabel: func(state *deployer_pb.StackState) string {
				return state.Status.String()
			},
			EventLabel: func(event deployer_pb.IsStackEventTypeWrappedType) string {
				return string(event.TypeKey())
			},
		},
	}

	// [*] --> CREATING : Triggered
	stackEventer.Register(psm.NewTransition([]deployer_pb.StackStatus{
		deployer_pb.StackStatus_UNSPECIFIED,
	},
		func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Triggered,
		) error {
			state.Status = deployer_pb.StackStatus_CREATING
			state.CurrentDeployment = event.Deployment

			tb.SideEffect(&deployer_tpb.TriggerDeploymentMessage{
				DeploymentId:    state.CurrentDeployment.DeploymentId,
				StackId:         state.StackId,
				Version:         state.CurrentDeployment.Version,
				EnvironmentName: state.EnvironmentName,
				ApplicationName: state.ApplicationName,
			})
			return nil

		},
	))

	// CREATING --> STABLE : DeploymentCompleted
	// MIGRATING --> STABLE : DeploymentCompleted
	stackEventer.Register(psm.NewTransition([]deployer_pb.StackStatus{
		deployer_pb.StackStatus_CREATING,
		deployer_pb.StackStatus_MIGRATING,
	},
		func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_DeploymentCompleted,
		) error {
			state.Status = deployer_pb.StackStatus_STABLE

			if len(state.QueuedDeployments) == 0 {
				// CHAIN NEXT
				tb.ChainEvent(&deployer_pb.StackEventType_Available{})
				return nil
			}

			tb.ChainEvent(&deployer_pb.StackEventType_Triggered{
				Deployment: state.QueuedDeployments[0],
			})
			return nil
		},
	))

	// STABLE --> MIGRATING : Triggered (Int)
	stackEventer.Register(psm.NewTransition([]deployer_pb.StackStatus{
		deployer_pb.StackStatus_STABLE,
	},
		func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Triggered,
		) error {
			state.Status = deployer_pb.StackStatus_MIGRATING

			state.CurrentDeployment = state.QueuedDeployments[0]
			state.QueuedDeployments = state.QueuedDeployments[1:]

			tb.SideEffect(&deployer_tpb.TriggerDeploymentMessage{
				DeploymentId:    state.CurrentDeployment.DeploymentId,
				StackId:         state.StackId,
				Version:         state.CurrentDeployment.Version,
				EnvironmentName: state.EnvironmentName,
				ApplicationName: state.ApplicationName,
			})
			return nil
		},
	))

	// AVAILABLE --> MIGRATING : Triggered (Ext)
	stackEventer.Register(psm.NewTransition([]deployer_pb.StackStatus{
		deployer_pb.StackStatus_AVAILABLE,
	},
		func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Triggered,
		) error {
			state.Status = deployer_pb.StackStatus_MIGRATING

			state.CurrentDeployment = event.Deployment

			tb.SideEffect(&deployer_tpb.TriggerDeploymentMessage{
				DeploymentId:    state.CurrentDeployment.DeploymentId,
				StackId:         state.StackId,
				Version:         state.CurrentDeployment.Version,
				EnvironmentName: state.EnvironmentName,
				ApplicationName: state.ApplicationName,
			})
			return nil
		},
	))

	// STABLE -> AVAILABLE : Available
	stackEventer.Register(psm.NewTransition([]deployer_pb.StackStatus{
		deployer_pb.StackStatus_STABLE,
	},
		func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Available,
		) error {
			state.Status = deployer_pb.StackStatus_AVAILABLE
			return nil
		},
	))

	// BROKEN --> BROKEN : Triggered
	// CREATING --> CREATING : Triggered
	// MIGRATING --> MIGRATING : Triggered
	stackEventer.Register(psm.NewTransition([]deployer_pb.StackStatus{
		deployer_pb.StackStatus_BROKEN,
		deployer_pb.StackStatus_CREATING,
		deployer_pb.StackStatus_MIGRATING,
	},
		func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Triggered,
		) error {
			// No state change.
			state.QueuedDeployments = append(state.QueuedDeployments, event.Deployment)
			return nil
		},
	))
	return stackEventer, nil
}
