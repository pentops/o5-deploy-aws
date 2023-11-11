package service

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/genericstate"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
)

type StateManager struct {
	deploymentStateMachine *genericstate.StateMachine[*deployer_pb.DeploymentEvent, *deployer_pb.DeploymentState]
}

func NewStateManager(conn *sql.DB) (*StateManager, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	deploymentStateMachine, err := genericstate.NewStateMachine(
		db,
		DeploymentStateSpec,
	)
	if err != nil {
		return nil, err
	}

	return &StateManager{
		deploymentStateMachine: deploymentStateMachine,
	}, nil
}

var DeploymentStateSpec = genericstate.StateSpec[*deployer_pb.DeploymentEvent, *deployer_pb.DeploymentState]{

	EmptyState: func(event *deployer_pb.DeploymentEvent) *deployer_pb.DeploymentState {
		return &deployer_pb.DeploymentState{
			DeploymentId: event.DeploymentId,
		}
	},
	EventData: func(event *deployer_pb.DeploymentEvent) proto.Message {
		return event.Event
	},
	EventMetadata: func(event *deployer_pb.DeploymentEvent) *genericstate.Metadata {
		return &genericstate.Metadata{
			Actor:     event.Metadata.Actor,
			Timestamp: event.Metadata.Timestamp.AsTime(),
			EventID:   event.Metadata.EventId,
		}
	},
	StateTable: "deployment",
	EventTable: "deployment_event",
	PrimaryKey: func(event *deployer_pb.DeploymentEvent) map[string]interface{} {
		return map[string]interface{}{
			"id": event.DeploymentId,
		}
	},
	EventForeignKey: func(event *deployer_pb.DeploymentEvent) map[string]interface{} {
		return map[string]interface{}{
			"deployment_id": event.DeploymentId,
		}
	},
	Transition: func(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
		switch et := event.Event.Type.(type) {
		case *deployer_pb.DeploymentEventType_Triggered_:
			state.Status = deployer_pb.DeploymentStatus_QUEUED
			state.Spec = et.Triggered.Spec

		default:
			return fmt.Errorf("unknown event type: %T", event.Event.Type)
		}
		return nil
	},
}
