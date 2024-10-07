package states

import (
	"context"
	"fmt"
	"strings"

	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan"
	"github.com/pentops/protostate/psm"
	"google.golang.org/protobuf/proto"
)

func buildRequestMetadata(contextMessage proto.Message) (*messaging_j5pb.RequestMetadata, error) {
	contextBytes, err := proto.Marshal(contextMessage)
	if err != nil {
		return nil, err
	}

	req := &messaging_j5pb.RequestMetadata{
		ReplyTo: "o5-deployer",
		Context: contextBytes,
	}
	return req, nil
}

func NewDeploymentEventer() (*awsdeployer_pb.DeploymentPSM, error) {
	sm, err := awsdeployer_pb.DeploymentPSMBuilder().
		SystemActor(psm.MustSystemActor("9C88DF5B-6ED0-46DF-A389-474F27A7395F")).
		BuildStateMachine()
	if err != nil {
		return nil, err
	}

	// [*] --> QUEUED : Created
	sm.From(awsdeployer_pb.DeploymentStatus_UNSPECIFIED).
		OnEvent(awsdeployer_pb.DeploymentPSMEventCreated).
		SetStatus(awsdeployer_pb.DeploymentStatus_QUEUED).
		Mutate(awsdeployer_pb.DeploymentPSMMutation(func(
			deployment *awsdeployer_pb.DeploymentStateData,
			event *awsdeployer_pb.DeploymentEventType_Created,
		) error {
			deployment.Spec = event.Spec
			deployment.Request = event.Request

			// No follow on, the stack state will trigger

			return nil
		})).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Created,
		) error {
			if deployment.Data.Request != nil {
				tb.SideEffect(&awsdeployer_tpb.DeploymentStatusMessage{
					Request:      deployment.Data.Request,
					DeploymentId: deployment.Keys.DeploymentId,
					Status:       awsdeployer_tpb.DeploymentStatus_PENDING,
				})
			}
			return nil
		}))

	// QUEUED --> TRIGGERED : Trigger
	sm.From(awsdeployer_pb.DeploymentStatus_QUEUED).
		OnEvent(awsdeployer_pb.DeploymentPSMEventTriggered).
		SetStatus(awsdeployer_pb.DeploymentStatus_TRIGGERED).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Triggered,
		) error {

			tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_StackWait{})

			if deployment.Data.Request != nil {
				tb.SideEffect(&awsdeployer_tpb.DeploymentStatusMessage{
					Request:      deployment.Data.Request,
					DeploymentId: deployment.Keys.DeploymentId,
					Status:       awsdeployer_tpb.DeploymentStatus_IN_PROGRESS,
				})
			}
			return nil
		}))

	// TRIGGERED --> WAITING : StackWait
	sm.From(awsdeployer_pb.DeploymentStatus_TRIGGERED).
		OnEvent(awsdeployer_pb.DeploymentPSMEventStackWait).
		SetStatus(awsdeployer_pb.DeploymentStatus_WAITING).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_StackWait,
		) error {
			requestMetadata, err := buildRequestMetadata(&awsdeployer_pb.StepContext{
				Phase:        awsdeployer_pb.StepPhase_WAIT,
				DeploymentId: deployment.Keys.DeploymentId,
			})
			if err != nil {
				return err
			}

			tb.SideEffect(&awsinfra_tpb.StabalizeStackMessage{
				Request:      requestMetadata,
				StackName:    deployment.Data.Spec.CfStackName,
				CancelUpdate: deployment.Data.Spec.Flags.CancelUpdates,
			})

			return nil
		}))

	// WAITIHG --> FAILED : StackWaitFailure
	sm.From(awsdeployer_pb.DeploymentStatus_WAITING).
		OnEvent(awsdeployer_pb.DeploymentPSMEventStackWaitFailure).
		SetStatus(awsdeployer_pb.DeploymentStatus_FAILED)
		// REFACTOR NOTE: This used to return error.

	// WAITING --> AVAILABLE : StackAvailable
	sm.From(awsdeployer_pb.DeploymentStatus_WAITING).
		OnEvent(awsdeployer_pb.DeploymentPSMEventStackAvailable).
		SetStatus(awsdeployer_pb.DeploymentStatus_AVAILABLE).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_StackAvailable,
		) error {

			plan, err := plan.DeploymentSteps(ctx, plan.DeploymentInput{
				Deployment:  deployment.Data.Spec,
				StackStatus: event.StackOutput,
			})
			if err != nil {
				return err
			}

			tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_RunSteps{
				Steps: plan,
			})

			return nil
		}))

	// AVAILABLE --> RUNNING : RunSteps
	sm.From(awsdeployer_pb.DeploymentStatus_AVAILABLE).
		OnEvent(awsdeployer_pb.DeploymentPSMEventRunSteps).
		SetStatus(awsdeployer_pb.DeploymentStatus_RUNNING).
		Mutate(awsdeployer_pb.DeploymentPSMMutation(func(
			deployment *awsdeployer_pb.DeploymentStateData,
			event *awsdeployer_pb.DeploymentEventType_RunSteps,
		) error {

			deployment.Steps = event.Steps
			return plan.UpdateStepDependencies(deployment.Steps)
		})).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_RunSteps,
		) error {

			if deployment.Data.Request != nil {
				msg := make([]string, 0)
				for _, step := range event.Steps {
					msg = append(msg, step.Name)
				}
				tb.SideEffect(&awsdeployer_tpb.DeploymentStatusMessage{
					Request:      deployment.Data.Request,
					DeploymentId: deployment.Keys.DeploymentId,
					Status:       awsdeployer_tpb.DeploymentStatus_IN_PROGRESS,
					Message:      fmt.Sprintf("Running %d steps\n%s", len(event.Steps), strings.Join(msg, "\n")),
				})
			}
			return plan.StepNext(ctx, tb, deployment.Data.Steps)
		}))

	// RUNNING --> RUNNING : StepResult
	sm.From(awsdeployer_pb.DeploymentStatus_RUNNING).
		Mutate(awsdeployer_pb.DeploymentPSMMutation(func(
			deployment *awsdeployer_pb.DeploymentStateData,
			event *awsdeployer_pb.DeploymentEventType_StepResult,
		) error {
			return plan.UpdateDeploymentStep(deployment.Steps, event)
		})).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_StepResult,
		) error {
			return plan.StepNext(ctx, tb, deployment.Data.Steps)
		}))

	// RUNNING --> RUNNING : RunStep
	sm.From(awsdeployer_pb.DeploymentStatus_RUNNING).
		OnEvent(awsdeployer_pb.DeploymentPSMEventRunStep).
		Mutate(awsdeployer_pb.DeploymentPSMMutation(func(
			deployment *awsdeployer_pb.DeploymentStateData,
			event *awsdeployer_pb.DeploymentEventType_RunStep,
		) error {
			return plan.ActivateDeploymentStep(deployment.Steps, event)
		})).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_RunStep,
		) error {
			sideEffect, err := plan.RunStep(ctx, deployment.Keys, deployment.Data.Steps, event)
			if err != nil {
				return err
			}
			if sideEffect != nil {
				tb.SideEffect(sideEffect)
			}
			return nil
		}))

	// RUNNING --> DONE : Done
	sm.From(awsdeployer_pb.DeploymentStatus_RUNNING).
		OnEvent(awsdeployer_pb.DeploymentPSMEventDone).
		SetStatus(awsdeployer_pb.DeploymentStatus_DONE).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Done,
		) error {
			if deployment.Data.Request != nil {
				tb.SideEffect(&awsdeployer_tpb.DeploymentStatusMessage{
					Request:      deployment.Data.Request,
					DeploymentId: deployment.Keys.DeploymentId,
					Status:       awsdeployer_tpb.DeploymentStatus_SUCCESS,
				})
			}
			return nil
		}))

	// * --> FAILED : Error
	sm.From().OnEvent(awsdeployer_pb.DeploymentPSMEventError).
		SetStatus(awsdeployer_pb.DeploymentStatus_FAILED).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Error,
		) error {
			if deployment.Data.Request != nil {
				tb.SideEffect(&awsdeployer_tpb.DeploymentStatusMessage{
					Request:      deployment.Data.Request,
					DeploymentId: deployment.Keys.DeploymentId,
					Status:       awsdeployer_tpb.DeploymentStatus_FAILED,
					Message:      event.Error,
				})
			}
			return nil
		}))

	// * --> TERMINATED : Terminated
	sm.From().OnEvent(awsdeployer_pb.DeploymentPSMEventTerminated).
		SetStatus(awsdeployer_pb.DeploymentStatus_TERMINATED).
		LogicHook(awsdeployer_pb.DeploymentPSMLogicHook(func(
			ctx context.Context,
			tb awsdeployer_pb.DeploymentPSMHookBaton,
			deployment *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Terminated,
		) error {
			if deployment.Data.Request != nil {
				tb.SideEffect(&awsdeployer_tpb.DeploymentStatusMessage{
					Request:      deployment.Data.Request,
					DeploymentId: deployment.Keys.DeploymentId,
					Status:       awsdeployer_tpb.DeploymentStatus_FAILED,
					Message:      "Deployment Terminated",
				})
			}
			return nil
		}))

	// Discard Triggered
	sm.From(
		awsdeployer_pb.DeploymentStatus_FAILED,
		awsdeployer_pb.DeploymentStatus_TERMINATED,
	).
		OnEvent(awsdeployer_pb.DeploymentPSMEventTriggered).
		Noop()

	// Discard Step Results
	sm.From(awsdeployer_pb.DeploymentStatus_TERMINATED).
		OnEvent(awsdeployer_pb.DeploymentPSMEventStepResult).
		Mutate(awsdeployer_pb.DeploymentPSMMutation(func(
			deployment *awsdeployer_pb.DeploymentStateData,
			event *awsdeployer_pb.DeploymentEventType_StepResult,
		) error {
			return plan.UpdateDeploymentStep(deployment.Steps, event)
		}))

	return sm, nil
}
