package deployer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeployerWorker struct {
	*deployer_tpb.UnimplementedDeployerTopicServer
	*deployer_tpb.UnimplementedCloudFormationReplyTopicServer
	*deployer_tpb.UnimplementedPostgresMigrationReplyTopicServer

	db *sqrlx.Wrapper

	specBuilder       *SpecBuilder
	stackEventer      *deployer_pb.StackPSM
	deploymentEventer *deployer_pb.DeploymentPSM
}

func NewDeployerWorker(conn sqrlx.Connection, specBuilder *SpecBuilder, states *states.StateMachines) (*DeployerWorker, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &DeployerWorker{
		db:                db,
		specBuilder:       specBuilder,
		stackEventer:      states.Stack,
		deploymentEventer: states.Deployment,
	}, nil
}

func (dw *DeployerWorker) doDeploymentEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {
	_, err := dw.deploymentEventer.Transition(ctx, dw.db, event)
	return err

	/*

		if err := dw.db.Transact(ctx, psm.TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
			deployment, err := getDeployment(ctx, tx, event.DeploymentId)
			if errors.Is(err, DeploymentNotFoundError) {
				trigger := event.Event.GetCreated()
				if trigger == nil {
					return fmt.Errorf("deployment %s not found, and the event is not an initiating event", event.DeploymentId)
				}

				deployment = &deployer_pb.DeploymentState{
					DeploymentId: event.DeploymentId,
					Spec:         trigger.Spec,
				}
			} else if err != nil {
				return fmt.Errorf("GetDeployment: %w", err)
			}

			return dw.deploymentEventer.handleEvent(ctx, tx, deployment, event)
		}); err != nil {
			return fmt.Errorf("doDeploymentEvent: %w", err)
		}
		return nil*/
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

func (dw *DeployerWorker) getEnvironment(ctx context.Context, environmentId string) (*environment_pb.Environment, error) {
	var envJSON []byte

	err := dw.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		return tx.SelectRow(ctx, sq.Select("state").From("environment").Where("id = ?", environmentId)).Scan(&envJSON)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("environment %q not found", environmentId)
	} else if err != nil {
		return nil, err
	}
	env := &deployer_pb.EnvironmentState{}
	if err := protojson.Unmarshal(envJSON, env); err != nil {
		return nil, fmt.Errorf("unmarshal environment: %w", err)
	}
	return env.Config, nil
}

func (dw *DeployerWorker) RequestDeployment(ctx context.Context, msg *deployer_tpb.RequestDeploymentMessage) (*emptypb.Empty, error) {

	environment, err := dw.getEnvironment(ctx, msg.EnvironmentId)
	if err != nil {
		return nil, err
	}

	spec, err := dw.specBuilder.BuildSpec(ctx, msg, environment)
	if err != nil {
		return nil, err
	}

	createDeploymentEvent := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Created_{
				Created: &deployer_pb.DeploymentEventType_Created{
					Spec: spec,
				},
			},
		},
	}

	if _, err := dw.deploymentEventer.Transition(ctx, dw.db, createDeploymentEvent); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func TranslateStackStatusChanged(msg *deployer_tpb.StackStatusChangedMessage) (*deployer_pb.DeploymentEvent, error) {
	context := &deployer_pb.DeploymentStackContext{}
	if err := proto.Unmarshal(msg.Request.Context, context); err != nil {
		return nil, err
	}

	return &deployer_pb.DeploymentEvent{
		DeploymentId: context.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_StackStatus_{
				StackStatus: &deployer_pb.DeploymentEventType_StackStatus{
					Lifecycle:       msg.Lifecycle,
					FullStatus:      msg.Status,
					StackOutput:     msg.Outputs,
					DeploymentPhase: context.DeploymentPhase,
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
	stackStatus := event.Event.GetStackStatus()

	if err := dw.db.Transact(ctx, psm.TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
		deployment, err := getDeployment(ctx, tx, event.DeploymentId)
		if err != nil {
			if errors.Is(err, DeploymentNotFoundError) {
				log.WithError(ctx, err).Error("Deployment not found")
				return nil
			}
			return err
		}

		if deployment.WaitingOnRemotePhase == nil || *deployment.WaitingOnRemotePhase != stackStatus.DeploymentPhase {
			waiting := ""
			if deployment.WaitingOnRemotePhase != nil {
				waiting = *deployment.WaitingOnRemotePhase
			}

			ctx := log.WithFields(ctx, map[string]interface{}{
				"cfReturnPhase":        stackStatus.DeploymentPhase,
				"waitingOnRemotePhase": waiting,
			})
			if msg.Lifecycle == deployer_pb.StackLifecycle_COMPLETE || msg.Lifecycle == deployer_pb.StackLifecycle_PROGRESS {
				log.Error(ctx, "Deployment phase mismatch, Ignoring")
				return nil
			} else {
				log.Error(ctx, "Deployment phase mismatch")
				return fmt.Errorf("deployment phase mismatch")
			}
		}

		if _, err := dw.deploymentEventer.TransitionInTx(ctx, tx, event); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) MigrationStatusChanged(ctx context.Context, msg *deployer_tpb.PostgresMigrationEventMessage) (*emptypb.Empty, error) {

	context := &deployer_pb.DeploymentMigrationContext{}
	if err := proto.Unmarshal(msg.Request.Context, context); err != nil {
		return nil, err
	}

	migrationStatus := &deployer_pb.DeploymentEventType_DbMigrateStatus{
		DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
			MigrationId: context.MigrationId,
		},
	}

	switch inner := msg.Event.UnwrapPSMEvent().(type) {
	case *deployer_pb.PostgresMigrationEventType_Prepare:
		migrationStatus.DbMigrateStatus.Status = deployer_pb.DatabaseMigrationStatus_PENDING
	case *deployer_pb.PostgresMigrationEventType_Done:
		migrationStatus.DbMigrateStatus.Status = deployer_pb.DatabaseMigrationStatus_COMPLETED
	case *deployer_pb.PostgresMigrationEventType_Error:
		migrationStatus.DbMigrateStatus.Status = deployer_pb.DatabaseMigrationStatus_FAILED
		migrationStatus.DbMigrateStatus.Error = &inner.Error
	default:
		return &emptypb.Empty{}, nil // ignore
	}

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: context.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},

		Event: &deployer_pb.DeploymentEventType{
			Type: migrationStatus,
		},
	}

	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)

}
