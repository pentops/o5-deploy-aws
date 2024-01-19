package deployer

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeployerWorker struct {
	*deployer_tpb.UnimplementedDeployerTopicServer
	db *sqrlx.Wrapper

	specBuilder       *SpecBuilder
	stackEventer      *deployer_pb.StackPSM
	deploymentEventer *deployer_pb.DeploymentPSM
}

func NewDeployerWorker(conn sqrlx.Connection, specBuilder *SpecBuilder) (*DeployerWorker, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	stackEventer, err := NewStackEventer(db)
	if err != nil {
		return nil, fmt.Errorf("NewStackEventer: %w", err)
	}

	deploymentEventer, err := NewDeploymentEventer(db)
	if err != nil {
		return nil, fmt.Errorf("NewDeploymentEventer: %w", err)
	}

	return &DeployerWorker{
		db:                db,
		specBuilder:       specBuilder,
		stackEventer:      stackEventer,
		deploymentEventer: deploymentEventer,
	}, nil
}

func (dw *DeployerWorker) doDeploymentEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {
	_, err := dw.deploymentEventer.Transition(ctx, event)
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

func (dw *DeployerWorker) RequestDeployment(ctx context.Context, msg *deployer_tpb.RequestDeploymentMessage) (*emptypb.Empty, error) {

	spec, err := dw.specBuilder.BuildSpec(ctx, msg)
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

	evt := &deployer_pb.StackEvent{
		StackId: StackID(spec.EnvironmentName, spec.AppName),
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.StackEventType{
			Type: &deployer_pb.StackEventType_Triggered_{
				Triggered: &deployer_pb.StackEventType_Triggered{
					Deployment: &deployer_pb.StackDeployment{
						DeploymentId: msg.DeploymentId,
						Version:      spec.Version,
					},
					EnvironmentName: spec.EnvironmentName,
					ApplicationName: spec.AppName,
				},
			},
		},
	}

	if err := dw.db.Transact(ctx, psm.TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
		if _, err := dw.stackEventer.TransitionInTx(ctx, tx, evt); err != nil {
			return err
		}

		if _, err := dw.deploymentEventer.TransitionInTx(ctx, tx, createDeploymentEvent); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) DeploymentComplete(ctx context.Context, msg *deployer_tpb.DeploymentCompleteMessage) (*emptypb.Empty, error) {

	stackEvent := &deployer_pb.StackEvent{
		StackId: StackID(msg.EnvironmentName, msg.ApplicationName),
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
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

	_, err := dw.stackEventer.Transition(ctx, stackEvent)
	return &emptypb.Empty{}, err
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
					Lifecycle:       msg.Lifecycle,
					FullStatus:      msg.Status,
					StackOutput:     msg.Outputs,
					DeploymentPhase: msg.StackId.DeploymentPhase,
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

	if err := dw.db.Transact(ctx, psm.TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
		deployment, err := getDeployment(ctx, tx, event.DeploymentId)
		if err != nil {
			if errors.Is(err, DeploymentNotFoundError) {
				log.WithError(ctx, err).Error("Deployment not found")
				return nil
			}
			return err
		}

		if deployment.WaitingOnRemotePhase == nil || *deployment.WaitingOnRemotePhase != msg.StackId.DeploymentPhase {
			waiting := ""
			if deployment.WaitingOnRemotePhase != nil {
				waiting = *deployment.WaitingOnRemotePhase
			}

			ctx := log.WithFields(ctx, map[string]interface{}{
				"cfReturnPhase":        msg.StackId.DeploymentPhase,
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
