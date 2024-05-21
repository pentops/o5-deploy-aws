package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeployerWorker struct {
	*deployer_tpb.UnimplementedDeployerInputTopicServer
	*deployer_tpb.UnimplementedCloudFormationReplyTopicServer
	*deployer_tpb.UnimplementedPostgresReplyTopicServer

	db *sqrlx.Wrapper

	specBuilder       *deployer.SpecBuilder
	stackEventer      *deployer_pb.StackPSM
	deploymentEventer *deployer_pb.DeploymentPSM
}

func NewDeployerWorker(conn sqrlx.Connection, specBuilder *deployer.SpecBuilder, states *states.StateMachines) (*DeployerWorker, error) {
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

func (dw *DeployerWorker) doDeploymentEvent(ctx context.Context, event *deployer_pb.DeploymentPSMEventSpec) error {
	_, err := dw.deploymentEventer.Transition(ctx, dw.db, event)
	return err
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
	return env.Data.Config, nil
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

	createDeploymentEvent := &deployer_pb.DeploymentPSMEventSpec{
		Keys: &deployer_pb.DeploymentKeys{
			DeploymentId: msg.DeploymentId,
		},
		EventID:   msg.DeploymentId,
		Timestamp: time.Now(),
		Event: &deployer_pb.DeploymentEventType_Created{
			Spec: spec,
		},
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "deployer",
					EventName:  "RequestDeployment",
				},
			},
		},
	}

	if _, err := dw.deploymentEventer.Transition(ctx, dw.db, createDeploymentEvent); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

var cfEventNamespace = uuid.MustParse("6BE12207-A62C-4D9E-8A94-48A091DDFB53")

func StackStatusToEvent(msg *deployer_tpb.StackStatusChangedMessage) (*deployer_pb.DeploymentPSMEventSpec, error) {

	stepContext := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &deployer_pb.DeploymentPSMEventSpec{
		Keys: &deployer_pb.DeploymentKeys{
			DeploymentId: stepContext.DeploymentId,
		},
		EventID:   uuid.NewSHA1(cfEventNamespace, []byte(msg.EventId)).String(),
		Timestamp: time.Now(),
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "deployer-cf",
					EventName:  "StackStatusChanged",
					ExternalId: &msg.EventId,
				},
			},
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

		event.Event = stepResult

	case deployer_pb.StepPhase_WAIT:
		switch msg.Lifecycle {
		case deployer_pb.CFLifecycle_PROGRESS,
			deployer_pb.CFLifecycle_ROLLING_BACK:
			return nil, nil

		case deployer_pb.CFLifecycle_COMPLETE,
			deployer_pb.CFLifecycle_ROLLED_BACK:

			event.Event = &deployer_pb.DeploymentEventType_StackAvailable{
				StackOutput: cfOutput,
			}

		case deployer_pb.CFLifecycle_MISSING:
			event.Event = &deployer_pb.DeploymentEventType_StackAvailable{
				StackOutput: nil,
			}

		default:
			event.Event = &deployer_pb.DeploymentEventType_StackWaitFailure{
				Error: fmt.Sprintf("Stack Status: %s", msg.Status),
			}
		}

	}

	return event, nil
}

func ChangeSetStatusToEvent(msg *deployer_tpb.ChangeSetStatusChangedMessage) (*deployer_pb.DeploymentPSMEventSpec, error) {
	stepContext := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &deployer_pb.DeploymentPSMEventSpec{
		Keys: &deployer_pb.DeploymentKeys{
			DeploymentId: stepContext.DeploymentId,
		},
		EventID:   msg.EventId,
		Timestamp: time.Now(),
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "deployer",
					EventName:  "PlanStatusChanged",
					ExternalId: &msg.EventId,
				},
			},
		},
	}

	if stepContext.Phase != deployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("Plan context expects STEPS and an ID")
	}

	changesetStatus := &deployer_pb.CFChangesetOutput{
		Lifecycle: msg.Lifecycle,
	}
	stepStatus := &deployer_pb.DeploymentEventType_StepResult{
		StepId: *stepContext.StepId,
		Output: &deployer_pb.StepOutputType{
			Type: &deployer_pb.StepOutputType_CfPlanStatus{
				CfPlanStatus: &deployer_pb.StepOutputType_CFPlanStatus{
					Output: changesetStatus,
				},
			},
		},
	}
	event.Event = stepStatus

	return event, nil
}

func (dw *DeployerWorker) ChangeSetStatusChanged(ctx context.Context, msg *deployer_tpb.ChangeSetStatusChangedMessage) (*emptypb.Empty, error) {
	event, err := ChangeSetStatusToEvent(msg)
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

func PostgresMigrationToEvent(msg *deployer_tpb.PostgresDatabaseStatusMessage) (*deployer_pb.DeploymentPSMEventSpec, error) {

	stepContext := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &deployer_pb.DeploymentPSMEventSpec{
		Keys: &deployer_pb.DeploymentKeys{
			DeploymentId: stepContext.DeploymentId,
		},
		EventID:   msg.EventId,
		Timestamp: time.Now(),
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "deployer-postgres",
					EventName:  "PostgresDatabaseStatus",
					ExternalId: &msg.EventId,
				},
			},
		},
	}

	if stepContext.Phase != deployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("DB Migration context expects STEPS and an ID")
	}

	stepStatus := &deployer_pb.DeploymentEventType_StepResult{
		StepId: *stepContext.StepId,
	}
	event.Event = stepStatus

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
