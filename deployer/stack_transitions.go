package deployer

import (
	"context"
	"fmt"

	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
)

type StackTransition[Event deployer_pb.IsStackEventTypeWrappedType] struct {
	FromStatus  []deployer_pb.StackStatus
	EventFilter func(Event) bool
	Transition  func(context.Context, StackTransitionBaton, *deployer_pb.StackState, Event) error
}

func (ts StackTransition[Event]) RunTransition(
	ctx context.Context,
	tb TransitionBaton[deployer_pb.IsStackEventTypeWrappedType],
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

func findStackTransition(stack *deployer_pb.StackState, event deployer_pb.IsStackEventTypeWrappedType) (ITransitionSpec[*deployer_pb.StackState, deployer_pb.IsStackEventTypeWrappedType], error) {
	for _, search := range stackTransitions {
		if search.Matches(stack, event) {
			return search, nil
		}
	}
	typeKey := event.TypeKey()
	return nil, fmt.Errorf("no transition found for %s -> %s", stack.Status.String(), typeKey)
}

type StackTransitionBaton TransitionBaton[deployer_pb.IsStackEventTypeWrappedType]

var stackTransitions = []ITransitionSpec[*deployer_pb.StackState, deployer_pb.IsStackEventTypeWrappedType]{
	// [*] --> CREATING : Triggered
	StackTransition[*deployer_pb.StackEventType_Triggered]{
		FromStatus: []deployer_pb.StackStatus{
			deployer_pb.StackStatus_UNSPECIFIED,
		},
		Transition: func(
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
	},

	// CREATING --> STABLE : DeploymentCompleted
	// MIGRATING --> STABLE : DeploymentCompleted
	StackTransition[*deployer_pb.StackEventType_DeploymentCompleted]{
		FromStatus: []deployer_pb.StackStatus{
			deployer_pb.StackStatus_CREATING,
			deployer_pb.StackStatus_MIGRATING,
		},
		Transition: func(
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
	},

	// STABLE --> MIGRATING : Triggered (Int)
	StackTransition[*deployer_pb.StackEventType_Triggered]{
		FromStatus: []deployer_pb.StackStatus{
			deployer_pb.StackStatus_STABLE,
		},
		Transition: func(
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
	},

	// AVAILABLE --> MIGRATING : Triggered (Ext)
	StackTransition[*deployer_pb.StackEventType_Triggered]{
		FromStatus: []deployer_pb.StackStatus{
			deployer_pb.StackStatus_AVAILABLE,
		},
		Transition: func(
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
	},

	// STABLE -> AVAILABLE : Available
	StackTransition[*deployer_pb.StackEventType_Available]{
		FromStatus: []deployer_pb.StackStatus{
			deployer_pb.StackStatus_STABLE,
		},
		Transition: func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Available,
		) error {
			state.Status = deployer_pb.StackStatus_AVAILABLE
			return nil
		},
	},

	// BROKEN --> BROKEN : Triggered
	// CREATING --> CREATING : Triggered
	// MIGRATING --> MIGRATING : Triggered
	StackTransition[*deployer_pb.StackEventType_Triggered]{
		FromStatus: []deployer_pb.StackStatus{
			deployer_pb.StackStatus_BROKEN,
			deployer_pb.StackStatus_CREATING,
			deployer_pb.StackStatus_MIGRATING,
		},
		Transition: func(
			ctx context.Context,
			tb StackTransitionBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Triggered,
		) error {
			// No state change.
			state.QueuedDeployments = append(state.QueuedDeployments, event.Deployment)
			return nil
		},
	},
}
