package deployer

import (
	"context"

	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_epb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func chainStackEvent(tb deployer_pb.StackPSMTransitionBaton, event deployer_pb.IsStackEventTypeWrappedType) *deployer_pb.StackEvent {
	md := tb.FullCause().Metadata
	de := &deployer_pb.StackEvent{
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor:     md.Actor,
		},
		StackId: tb.FullCause().StackId,
		Event:   &deployer_pb.StackEventType{},
	}
	de.Event.Set(event)
	return de
}

func StackTableSpec() deployer_pb.StackPSMTableSpec {

	tableSpec := deployer_pb.DefaultStackPSMTableSpec
	tableSpec.StateDataColumn = "state"
	tableSpec.ExtraStateColumns = func(s *deployer_pb.StackState) (map[string]interface{}, error) {
		return map[string]interface{}{
			"env_name": s.EnvironmentName,
			"app_name": s.ApplicationName,
		}, nil
	}
	tableSpec.EventDataColumn = "event"
	tableSpec.EventColumns = func(e *deployer_pb.StackEvent) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id":        e.Metadata.EventId,
			"stack_id":  e.StackId,
			"event":     e,
			"timestamp": e.Metadata.Timestamp,
		}, nil
	}

	return tableSpec

}

func NewStackEventer(db psm.Transactor) (*deployer_pb.StackPSM, error) {

	sm, err := deployer_pb.NewStackPSM(db, psm.WithTableSpec(StackTableSpec()))
	if err != nil {
		return nil, err
	}

	sm.AddHook(func(ctx context.Context, tx sqrlx.Transaction, state *deployer_pb.StackState, event *deployer_pb.StackEvent) error {
		evt := &deployer_epb.StackEventMessage{
			Metadata: event.Metadata,
			Event:    event.Event,
			State:    state,
		}
		return outbox.Send(ctx, tx, evt)
	})

	// [*] --> CREATING : Triggered
	sm.From(
		deployer_pb.StackStatus_UNSPECIFIED,
	).Do(deployer_pb.StackPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.StackPSMTransitionBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {
		state.Status = deployer_pb.StackStatus_CREATING
		state.CurrentDeployment = event.Deployment
		state.ApplicationName = event.ApplicationName
		state.EnvironmentName = event.EnvironmentName
		state.QueuedDeployments = []*deployer_pb.StackDeployment{}

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
	sm.From(
		deployer_pb.StackStatus_CREATING,
		deployer_pb.StackStatus_MIGRATING,
	).Do(deployer_pb.StackPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.StackPSMTransitionBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_DeploymentCompleted,
	) error {
		state.Status = deployer_pb.StackStatus_STABLE

		if len(state.QueuedDeployments) == 0 {
			// CHAIN NEXT
			tb.ChainEvent(chainStackEvent(tb, &deployer_pb.StackEventType_Available{}))
			return nil
		}

		tb.ChainEvent(chainStackEvent(tb, &deployer_pb.StackEventType_Triggered{
			Deployment: state.QueuedDeployments[0],
		}))
		return nil
	},
	))

	// STABLE --> MIGRATING : Triggered (Int)
	sm.From(
		deployer_pb.StackStatus_STABLE,
	).Do(deployer_pb.StackPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.StackPSMTransitionBaton,
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
	sm.From(
		deployer_pb.StackStatus_AVAILABLE,
	).Do(deployer_pb.StackPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.StackPSMTransitionBaton,
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
	sm.From(
		deployer_pb.StackStatus_STABLE,
	).Do(deployer_pb.StackPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.StackPSMTransitionBaton,
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
	sm.From(
		deployer_pb.StackStatus_BROKEN,
		deployer_pb.StackStatus_CREATING,
		deployer_pb.StackStatus_MIGRATING,
	).Do(deployer_pb.StackPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.StackPSMTransitionBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {
		// No state change.
		state.QueuedDeployments = append(state.QueuedDeployments, event.Deployment)
		return nil
	},
	))
	return sm, nil
}
