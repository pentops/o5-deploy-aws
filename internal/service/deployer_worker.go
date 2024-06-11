package service

import (
	"context"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
	"github.com/pentops/o5-deploy-aws/internal/states"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeployerWorker struct {
	*deployer_tpb.UnimplementedDeploymentRequestTopicServer
	*awsinfra_tpb.UnimplementedCloudFormationReplyTopicServer
	*awsinfra_tpb.UnimplementedPostgresReplyTopicServer

	lookup *LookupProvider

	specBuilder       *deployer.SpecBuilder
	stackEventer      *awsdeployer_pb.StackPSM
	deploymentEventer *awsdeployer_pb.DeploymentPSMDB
}

func NewDeployerWorker(conn sqrlx.Connection, specBuilder *deployer.SpecBuilder, states *states.StateMachines) (*DeployerWorker, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	lookupProvider, err := NewLookupProvider(conn)
	if err != nil {
		return nil, err
	}

	return &DeployerWorker{
		specBuilder:       specBuilder,
		stackEventer:      states.Stack,
		deploymentEventer: states.Deployment.WithDB(db),
		lookup:            lookupProvider,
	}, nil
}

func (dw *DeployerWorker) doDeploymentEvent(ctx context.Context, event *awsdeployer_pb.DeploymentPSMEventSpec) error {
	_, err := dw.deploymentEventer.Transition(ctx, event)
	return err
}

type appStack struct {
	environment   *environment_pb.Environment
	cluster       *environment_pb.Cluster
	environmentID string
	clusterID     string
	stackID       string
}

func (dw *DeployerWorker) RequestDeployment(ctx context.Context, msg *deployer_tpb.RequestDeploymentMessage) (*emptypb.Empty, error) {
	appID, err := dw.lookup.lookupAppStack(ctx, msg.EnvironmentId, msg.Application.Name)
	if err != nil {
		return nil, err
	}

	spec, err := dw.specBuilder.BuildSpec(ctx, msg, appID.cluster, appID.environment)
	if err != nil {
		return nil, err
	}

	createDeploymentEvent := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
			DeploymentId:  msg.DeploymentId,
			EnvironmentId: appID.environmentID,
			StackId:       appID.stackID,
			ClusterId:     appID.clusterID,
		},
		EventID:   msg.DeploymentId,
		Timestamp: time.Now(),
		Event: &awsdeployer_pb.DeploymentEventType_Created{
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

	if _, err := dw.deploymentEventer.Transition(ctx, createDeploymentEvent); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

var cfEventNamespace = uuid.MustParse("6BE12207-A62C-4D9E-8A94-48A091DDFB53")

func StackStatusToEvent(msg *awsinfra_tpb.StackStatusChangedMessage) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {

	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
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

	cfOutput := &awsdeployer_pb.CFStackOutput{
		Lifecycle: msg.Lifecycle,
		Outputs:   msg.Outputs,
	}
	switch stepContext.Phase {
	case awsdeployer_pb.StepPhase_STEPS:
		if stepContext.StepId == nil {
			return nil, fmt.Errorf("StepId is required when Phase is STEPS")
		}

		stepResult := &awsdeployer_pb.DeploymentEventType_StepResult{
			StepId: *stepContext.StepId,
			Output: &awsdeployer_pb.StepOutputType{
				Type: &awsdeployer_pb.StepOutputType_CfStatus{
					CfStatus: &awsdeployer_pb.StepOutputType_CFStatus{
						Output: cfOutput,
					},
				},
			},
		}

		switch msg.Lifecycle {
		case awsdeployer_pb.CFLifecycle_PROGRESS,
			awsdeployer_pb.CFLifecycle_ROLLING_BACK:
			return nil, nil

		case awsdeployer_pb.CFLifecycle_COMPLETE:
			stepResult.Status = awsdeployer_pb.StepStatus_DONE
		case awsdeployer_pb.CFLifecycle_CREATE_FAILED,
			awsdeployer_pb.CFLifecycle_ROLLED_BACK,
			awsdeployer_pb.CFLifecycle_MISSING:
			stepResult.Status = awsdeployer_pb.StepStatus_FAILED
		default:
			return nil, fmt.Errorf("unknown lifecycle: %s", msg.Lifecycle)
		}

		event.Event = stepResult

	case awsdeployer_pb.StepPhase_WAIT:
		switch msg.Lifecycle {
		case awsdeployer_pb.CFLifecycle_PROGRESS,
			awsdeployer_pb.CFLifecycle_ROLLING_BACK:
			return nil, nil

		case awsdeployer_pb.CFLifecycle_COMPLETE,
			awsdeployer_pb.CFLifecycle_ROLLED_BACK:

			event.Event = &awsdeployer_pb.DeploymentEventType_StackAvailable{
				StackOutput: cfOutput,
			}

		case awsdeployer_pb.CFLifecycle_MISSING:
			event.Event = &awsdeployer_pb.DeploymentEventType_StackAvailable{
				StackOutput: nil,
			}

		default:
			event.Event = &awsdeployer_pb.DeploymentEventType_StackWaitFailure{
				Error: fmt.Sprintf("Stack Status: %s", msg.Status),
			}
		}

	}

	return event, nil
}

func ChangeSetStatusToEvent(msg *awsinfra_tpb.ChangeSetStatusChangedMessage) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
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

	if stepContext.Phase != awsdeployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("Plan context expects STEPS and an ID")
	}

	status := awsdeployer_pb.StepStatus_DONE
	if msg.Lifecycle != awsdeployer_pb.CFChangesetLifecycle_AVAILABLE {
		status = awsdeployer_pb.StepStatus_FAILED
	}
	changesetStatus := &awsdeployer_pb.CFChangesetOutput{
		Lifecycle: msg.Lifecycle,
	}
	stepStatus := &awsdeployer_pb.DeploymentEventType_StepResult{
		StepId: *stepContext.StepId,
		Status: status,
		Output: &awsdeployer_pb.StepOutputType{
			Type: &awsdeployer_pb.StepOutputType_CfPlanStatus{
				CfPlanStatus: &awsdeployer_pb.StepOutputType_CFPlanStatus{
					Output: changesetStatus,
				},
			},
		},
	}
	event.Event = stepStatus

	return event, nil
}

func (dw *DeployerWorker) ChangeSetStatusChanged(ctx context.Context, msg *awsinfra_tpb.ChangeSetStatusChangedMessage) (*emptypb.Empty, error) {
	event, err := ChangeSetStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &emptypb.Empty{}, nil
	}

	if _, err := dw.deploymentEventer.Transition(ctx, event); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) StackStatusChanged(ctx context.Context, msg *awsinfra_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := StackStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &emptypb.Empty{}, nil
	}

	if _, err := dw.deploymentEventer.Transition(ctx, event); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func PostgresMigrationToEvent(msg *awsinfra_tpb.PostgresDatabaseStatusMessage) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {

	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	event := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
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

	if stepContext.Phase != awsdeployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("DB Migration context expects STEPS and an ID")
	}

	stepStatus := &awsdeployer_pb.DeploymentEventType_StepResult{
		StepId: *stepContext.StepId,
	}
	event.Event = stepStatus

	switch msg.Status {
	case awsinfra_tpb.PostgresStatus_DONE:
		stepStatus.Status = awsdeployer_pb.StepStatus_DONE
	case awsinfra_tpb.PostgresStatus_ERROR:
		stepStatus.Status = awsdeployer_pb.StepStatus_FAILED
		stepStatus.Error = msg.Error

	case awsinfra_tpb.PostgresStatus_STARTED:
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown status: %s", msg.Status)
	}

	return event, nil
}

func (dw *DeployerWorker) PostgresDatabaseStatus(ctx context.Context, msg *awsinfra_tpb.PostgresDatabaseStatusMessage) (*emptypb.Empty, error) {

	event, err := PostgresMigrationToEvent(msg)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)

}
