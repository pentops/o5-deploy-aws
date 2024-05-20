package states

import (
	"context"
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
)

func buildRequestMetadata(contextMessage proto.Message) (*messaging_pb.RequestMetadata, error) {
	contextBytes, err := proto.Marshal(contextMessage)
	if err != nil {
		return nil, err
	}

	req := &messaging_pb.RequestMetadata{
		ReplyTo: "o5-deployer",
		Context: contextBytes,
	}
	return req, nil
}

func NewDeploymentEventer() (*deployer_pb.DeploymentPSM, error) {
	config := deployer_pb.DefaultDeploymentPSMConfig().
		StoreExtraStateColumns(func(s *deployer_pb.DeploymentState) (map[string]interface{}, error) {
			return map[string]interface{}{
				"stack_id": s.Data.StackId,
			}, nil
		}).
		StoreExtraEventColumns(func(e *deployer_pb.DeploymentEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":            e.Metadata.EventId,
				"deployment_id": e.Keys.DeploymentId,
				"timestamp":     e.Metadata.Timestamp,
			}, nil
		}).
		SystemActor(psm.MustSystemActor("9C88DF5B-6ED0-46DF-A389-474F27A7395F"))

	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	// [*] --> QUEUED : Created
	sm.From(deployer_pb.DeploymentStatus_UNSPECIFIED).
		OnEvent(deployer_pb.DeploymentPSMEventCreated).
		SetStatus(deployer_pb.DeploymentStatus_QUEUED).
		Mutate(deployer_pb.DeploymentPSMMutation(func(
			deployment *deployer_pb.DeploymentStateData,
			event *deployer_pb.DeploymentEventType_Created,
		) error {
			deployment.Spec = event.Spec
			deployment.StackName = fmt.Sprintf("%s-%s", event.Spec.EnvironmentName, event.Spec.AppName)
			deployment.StackId = StackID(event.Spec.EnvironmentName, event.Spec.AppName)

			// No follow on, the stack state will trigger

			return nil
		}))

	// QUEUED --> TRIGGERED : Trigger
	sm.From(deployer_pb.DeploymentStatus_QUEUED).
		OnEvent(deployer_pb.DeploymentPSMEventTriggered).
		SetStatus(deployer_pb.DeploymentStatus_TRIGGERED).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Triggered,
		) error {

			tb.ChainEvent(&deployer_pb.DeploymentEventType_StackWait{})

			return nil
		}))

	// TRIGGERED --> WAITING : StackWait
	sm.From(deployer_pb.DeploymentStatus_TRIGGERED).
		OnEvent(deployer_pb.DeploymentPSMEventStackWait).
		SetStatus(deployer_pb.DeploymentStatus_WAITING).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackWait,
		) error {
			requestMetadata, err := buildRequestMetadata(&deployer_pb.StepContext{
				Phase:        deployer_pb.StepPhase_WAIT,
				DeploymentId: deployment.Keys.DeploymentId,
			})
			if err != nil {
				return err
			}

			tb.SideEffect(&deployer_tpb.StabalizeStackMessage{
				Request:      requestMetadata,
				StackName:    deployment.Data.StackName,
				CancelUpdate: deployment.Data.Spec.Flags.CancelUpdates,
			})

			return nil
		}))

	// WAITIHG --> FAILED : StackWaitFailure
	sm.From(deployer_pb.DeploymentStatus_WAITING).
		OnEvent(deployer_pb.DeploymentPSMEventStackWaitFailure).
		SetStatus(deployer_pb.DeploymentStatus_FAILED)
		// REFACTOR NOTE: This used to return error.

	// WAITING --> AVAILABLE : StackAvailable
	sm.From(deployer_pb.DeploymentStatus_WAITING).
		OnEvent(deployer_pb.DeploymentPSMEventStackAvailable).
		SetStatus(deployer_pb.DeploymentStatus_AVAILABLE).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackAvailable,
		) error {

			// TODO: The plan should be generated in a side effect and stored as a
			// new event.

			plan, err := planDeploymentSteps(ctx, deployment.Data, planInput{
				stackStatus: event.StackOutput,
				flags:       deployment.Data.Spec.Flags,
			})
			if err != nil {
				return err
			}

			tb.ChainEvent(&deployer_pb.DeploymentEventType_RunSteps{
				Steps: plan,
			})

			return nil
		}))

	// AVAILABLE --> RUNNING : RunSteps
	sm.From(deployer_pb.DeploymentStatus_AVAILABLE).
		OnEvent(deployer_pb.DeploymentPSMEventRunSteps).
		SetStatus(deployer_pb.DeploymentStatus_RUNNING).
		Mutate(deployer_pb.DeploymentPSMMutation(func(
			deployment *deployer_pb.DeploymentStateData,
			event *deployer_pb.DeploymentEventType_RunSteps,
		) error {

			deployment.Steps = event.Steps
			return nil
		})).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_RunSteps,
		) error {
			return stepNext(ctx, tb, deployment)
		}))

	// RUNNING --> RUNNING : StepResult
	sm.From(deployer_pb.DeploymentStatus_RUNNING).
		Mutate(deployer_pb.DeploymentPSMMutation(func(
			deployment *deployer_pb.DeploymentStateData,
			event *deployer_pb.DeploymentEventType_StepResult,
		) error {
			updateDeploymentStep(deployment, event)
			return nil
		})).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StepResult,
		) error {
			return stepNext(ctx, tb, deployment)
		}))

	// RUNNING --> DONE : Done
	sm.From(deployer_pb.DeploymentStatus_RUNNING).
		OnEvent(deployer_pb.DeploymentPSMEventDone).
		SetStatus(deployer_pb.DeploymentStatus_DONE)

	// * --> FAILED : Error
	sm.From().OnEvent(deployer_pb.DeploymentPSMEventError).
		SetStatus(deployer_pb.DeploymentStatus_FAILED)

	// * --> TERMINATED : Terminated
	sm.From().OnEvent(deployer_pb.DeploymentPSMEventTerminated).
		SetStatus(deployer_pb.DeploymentStatus_TERMINATED)

	// Discard Triggered
	sm.From(
		deployer_pb.DeploymentStatus_FAILED,
		deployer_pb.DeploymentStatus_TERMINATED,
	).
		OnEvent(deployer_pb.DeploymentPSMEventTriggered).
		Noop()

	// Discard Step Results
	sm.From(deployer_pb.DeploymentStatus_TERMINATED).
		OnEvent(deployer_pb.DeploymentPSMEventStepResult).
		Mutate(deployer_pb.DeploymentPSMMutation(func(
			deployment *deployer_pb.DeploymentStateData,
			event *deployer_pb.DeploymentEventType_StepResult,
		) error {
			updateDeploymentStep(deployment, event)
			return nil
		}))

	return sm, nil
}

func updateDeploymentStep(deployment *deployer_pb.DeploymentStateData, event *deployer_pb.DeploymentEventType_StepResult) {
	for _, step := range deployment.Steps {
		if step.Id == event.StepId {
			step.Status = event.Status
			step.Output = event.Output
			step.Error = event.Error
			return
		}
	}
}
