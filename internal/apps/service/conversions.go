package service

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"google.golang.org/protobuf/proto"
)

func PostgresMigrationToEvent(msg *awsinfra_tpb.PostgresDatabaseStatusMessage) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {

	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	if stepContext.Phase != awsdeployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("DB Migration context expects STEPS and an ID")
	}

	event := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
			DeploymentId: stepContext.DeploymentId,
		},
		EventID:   msg.EventId,
		Timestamp: time.Now(),
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
					SystemName: "deployer-postgres",
					EventName:  "PostgresDatabaseStatus",
					ExternalId: &msg.EventId,
				},
			},
		},
	}

	stepStatus := &awsdeployer_pb.DeploymentEventType_StepResult{
		Result: &drss_pb.StepResult{
			StepId: *stepContext.StepId,
		},
	}
	event.Event = stepStatus

	switch msg.Status {
	case awsinfra_tpb.PostgresStatus_DONE:
		stepStatus.Result.Status = drss_pb.StepStatus_DONE
	case awsinfra_tpb.PostgresStatus_ERROR:
		stepStatus.Result.Status = drss_pb.StepStatus_FAILED
		stepStatus.Result.Error = msg.Error

	case awsinfra_tpb.PostgresStatus_STARTED:
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown status: %s", msg.Status)
	}

	return event, nil
}

func ECSTaskStatusToEvent(msg *awsinfra_tpb.ECSTaskStatusMessage) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {

	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, stepContext); err != nil {
		return nil, err
	}

	if stepContext.Phase != awsdeployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("DB Migration context expects STEPS and an ID")
	}

	event := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
			DeploymentId: stepContext.DeploymentId,
		},
		EventID:   msg.EventId,
		Timestamp: time.Now(),
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
					SystemName: "deployer-postgres",
					EventName:  "PostgresDatabaseStatus",
					ExternalId: &msg.EventId,
				},
			},
		},
	}

	stepStatus := &awsdeployer_pb.DeploymentEventType_StepResult{
		Result: &drss_pb.StepResult{
			StepId: *stepContext.StepId,
		},
	}
	event.Event = stepStatus

	switch et := msg.Event.Type.(type) {
	case *awsinfra_tpb.ECSTaskEventType_Pending_, *awsinfra_tpb.ECSTaskEventType_Running_:
		// Ignore
		return nil, nil

	case *awsinfra_tpb.ECSTaskEventType_Exited_:
		if et.Exited.ExitCode == 0 {
			stepStatus.Result.Status = drss_pb.StepStatus_DONE
		} else {
			stepStatus.Result.Status = drss_pb.StepStatus_FAILED
			stepStatus.Result.Error = aws.String(fmt.Sprintf("task exited with code %d", et.Exited.ExitCode))
		}

	case *awsinfra_tpb.ECSTaskEventType_Failed_:
		stepStatus.Result.Status = drss_pb.StepStatus_FAILED
		stepStatus.Result.Error = aws.String(et.Failed.Reason)
		if et.Failed.ContainerName != nil {
			stepStatus.Result.Error = aws.String(fmt.Sprintf("%s: %s", *et.Failed.ContainerName, et.Failed.Reason))
		}

	case *awsinfra_tpb.ECSTaskEventType_Stopped_:
		stepStatus.Result.Status = drss_pb.StepStatus_FAILED
		stepStatus.Result.Error = aws.String(fmt.Sprintf("ECS Task Stopped: %s", et.Stopped.Reason))

	default:
		return nil, fmt.Errorf("unknown ECS Status Event type: %T", et)
	}
	return event, nil
}

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
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
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
			Result: &drss_pb.StepResult{
				StepId: *stepContext.StepId,
			},

			Output: &awsdeployer_pb.StepOutputType{
				Type: &awsdeployer_pb.StepOutputType_CfStackStatus{
					CfStackStatus: &awsdeployer_pb.StepOutputType_CFStackStatus{
						Lifecycle: msg.Lifecycle,
						Outputs:   msg.Outputs,
					},
				},
			},
		}

		switch msg.Lifecycle {
		case awsdeployer_pb.CFLifecycle_PROGRESS,
			awsdeployer_pb.CFLifecycle_ROLLING_BACK:
			return nil, nil

		case awsdeployer_pb.CFLifecycle_COMPLETE:
			stepResult.Result.Status = drss_pb.StepStatus_DONE
		case awsdeployer_pb.CFLifecycle_CREATE_FAILED,
			awsdeployer_pb.CFLifecycle_ROLLED_BACK,
			awsdeployer_pb.CFLifecycle_MISSING:
			stepResult.Result.Status = drss_pb.StepStatus_FAILED
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
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
					SystemName: "deployer",
					EventName:  "PlanStatusChanged",
					ExternalId: &msg.EventId,
				},
			},
		},
	}

	if stepContext.Phase != awsdeployer_pb.StepPhase_STEPS || stepContext.StepId == nil {
		return nil, fmt.Errorf("plan context expects STEPS and an ID")
	}

	status := drss_pb.StepStatus_DONE
	if msg.Lifecycle != awsdeployer_pb.CFChangesetLifecycle_AVAILABLE {
		status = drss_pb.StepStatus_FAILED
	}
	stepStatus := &awsdeployer_pb.DeploymentEventType_StepResult{
		Result: &drss_pb.StepResult{
			StepId: *stepContext.StepId,
			Status: status,
		},
		Output: &awsdeployer_pb.StepOutputType{
			Type: &awsdeployer_pb.StepOutputType_CfChangesetStatus{
				CfChangesetStatus: &awsdeployer_pb.StepOutputType_CFChangesetStatus{
					Lifecycle: msg.Lifecycle,
				},
			},
		},
	}
	event.Event = stepStatus

	return event, nil
}

func ECSDeploymentStatusToEvent(msg *awsinfra_tpb.ECSDeploymentStatusMessage) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {

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
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
					SystemName: "deployer",
					EventName:  "DeploymentStatusChanged",
					ExternalId: &msg.EventId,
				},
			},
		},
	}

	_ = event
	return nil, fmt.Errorf("not implemented")

}
