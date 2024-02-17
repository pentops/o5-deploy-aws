package deployer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
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
	*deployer_tpb.UnimplementedPostgresReplyTopicServer

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

func StackStatusToEvent(msg *deployer_tpb.StackStatusChangedMessage) (*deployer_pb.DeploymentEvent, error) {

	stepContext := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: stepContext.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	cfOutput := &deployer_pb.CFStackOutput{
		Lifecycle: msg.Lifecycle,
		Outputs:   msg.Outputs,
	}
	switch stepContext.Phase {
	case deployer_pb.StepPhase_STEPS:
		if stepContext.StepId == nil {
			return nil, fmt.Errorf("StepId is required when Phase is STEPS")
		}

		stepResult := &deployer_pb.DeploymentEventType_StepResult{
			StepId: *stepContext.StepId,
			Output: &deployer_pb.StepOutputType{
				Type: &deployer_pb.StepOutputType_CfStatus{
					CfStatus: &deployer_pb.StepOutputType_CFStatus{
						Output: cfOutput,
					},
				},
			},
		}

		switch msg.Lifecycle {
		case deployer_pb.CFLifecycle_PROGRESS,
			deployer_pb.CFLifecycle_ROLLING_BACK:
			return nil, nil

		case deployer_pb.CFLifecycle_COMPLETE:
			stepResult.Status = deployer_pb.StepStatus_DONE
		case deployer_pb.CFLifecycle_CREATE_FAILED,
			deployer_pb.CFLifecycle_ROLLED_BACK,
			deployer_pb.CFLifecycle_MISSING:
			stepResult.Status = deployer_pb.StepStatus_FAILED
		default:
			return nil, fmt.Errorf("unknown lifecycle: %s", msg.Lifecycle)
		}

		event.SetPSMEvent(stepResult)

	case deployer_pb.StepPhase_WAIT:
		switch msg.Lifecycle {
		case deployer_pb.CFLifecycle_PROGRESS,
			deployer_pb.CFLifecycle_ROLLING_BACK:
			return nil, nil

		case deployer_pb.CFLifecycle_COMPLETE,
			deployer_pb.CFLifecycle_ROLLED_BACK:

			event.SetPSMEvent(&deployer_pb.DeploymentEventType_StackAvailable{
				StackOutput: cfOutput,
			})

		case deployer_pb.CFLifecycle_MISSING:
			event.SetPSMEvent(&deployer_pb.DeploymentEventType_StackAvailable{
				StackOutput: nil,
			})

		default:
			event.SetPSMEvent(&deployer_pb.DeploymentEventType_StackWaitFailure{
				Error: fmt.Sprintf("Stack Status: %s", msg.Status),
			})
		}

	}

	return event, nil
}

func (dw *DeployerWorker) StackStatusChanged(ctx context.Context, msg *deployer_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := StackStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &emptypb.Empty{}, nil
	}

	if err := dw.db.Transact(ctx, psm.TxOptions, func(ctx context.Context, tx sqrlx.Transaction) error {
		if _, err := dw.deploymentEventer.TransitionInTx(ctx, tx, event); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func PostgresMigrationToEvent(msg *deployer_tpb.PostgresDatabaseStatusMessage) (*deployer_pb.DeploymentEvent, error) {

	stepContext := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: stepContext.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	if stepContext.Phase != deployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("DB Migration context expects STEPS and an ID")
	}

	stepStatus := &deployer_pb.DeploymentEventType_StepResult{
		StepId: *stepContext.StepId,
	}
	event.SetPSMEvent(stepStatus)

	switch msg.Status {
	case deployer_tpb.PostgresStatus_DONE:
		stepStatus.Status = deployer_pb.StepStatus_DONE
	case deployer_tpb.PostgresStatus_ERROR:
		stepStatus.Status = deployer_pb.StepStatus_FAILED
		stepStatus.Error = msg.Error

	case deployer_tpb.PostgresStatus_STARTED:
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown status: %s", msg.Status)
	}

	return event, nil
}

func (dw *DeployerWorker) PostgresDatabaseStatus(ctx context.Context, msg *deployer_tpb.PostgresDatabaseStatusMessage) (*emptypb.Empty, error) {

	event, err := PostgresMigrationToEvent(msg)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)

}
