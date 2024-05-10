package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func chainDeploymentEvent(tb deployer_pb.DeploymentPSMTransitionBaton, event deployer_pb.IsDeploymentEventTypeWrappedType) *deployer_pb.DeploymentEvent {
	md := tb.FullCause()
	de := &deployer_pb.DeploymentEvent{
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Keys:  proto.Clone(md.Keys).(*deployer_pb.DeploymentKeys),
		Event: &deployer_pb.DeploymentEventType{},
	}
	de.Event.Set(event)
	return de
}

func NewDeploymentEventer() (*deployer_pb.DeploymentPSM, error) {
	config := deployer_pb.DefaultDeploymentPSMConfig().
		StoreExtraStateColumns(func(s *deployer_pb.DeploymentState) (map[string]interface{}, error) {
			return map[string]interface{}{
				"stack_id": s.StackId,
			}, nil
		}).
		StoreExtraEventColumns(func(e *deployer_pb.DeploymentEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":            e.Metadata.EventId,
				"deployment_id": e.Keys.DeploymentId,
				"timestamp":     e.Metadata.Timestamp,
			}, nil
		})

	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	/*
		TODO: Future hook
		sm.AddHook(func(ctx context.Context, tx sqrlx.Transaction, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
			evt := &deployer_epb.DeploymentEventMessage{
				Metadata: event.Metadata,
				Event:    event.Event,
				State:    state,
			}
			return outbox.Send(ctx, tx, evt)
		})
	*/

	// [*] --> QUEUED : Created
	sm.From(deployer_pb.DeploymentStatus_UNSPECIFIED).
		Transition(deployer_pb.DeploymentPSMTransition(func(
			ctx context.Context,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Created,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_QUEUED
			deployment.Spec = event.Spec
			deployment.StackName = fmt.Sprintf("%s-%s", event.Spec.EnvironmentName, event.Spec.AppName)
			deployment.StackId = StackID(event.Spec.EnvironmentName, event.Spec.AppName)

			// No follow on, the stack state will trigger

			return nil
		}))

	// QUEUED --> TRIGGERED : Trigger
	sm.From(deployer_pb.DeploymentStatus_QUEUED).
		Transition(deployer_pb.DeploymentPSMTransition(func(
			ctx context.Context,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Triggered,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_TRIGGERED
			return nil
		})).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Triggered,
		) error {

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackWait{}))

			return nil
		}))

	// TRIGGERED --> WAITING : StackWait
	sm.From(deployer_pb.DeploymentStatus_TRIGGERED).
		Transition(deployer_pb.DeploymentPSMTransition(func(
			ctx context.Context,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackWait,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_WAITING
			return nil

		})).
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
				StackName:    deployment.StackName,
				CancelUpdate: deployment.Spec.Flags.CancelUpdates,
			})

			return nil
		}))

	// WAITIHG --> FAILED : StackWaitFailure
	sm.From(
		deployer_pb.DeploymentStatus_WAITING,
	).Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_StackWaitFailure,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_FAILED
		return fmt.Errorf("stack failed: %s", event.Error)
	}))

	// WAITING --> AVAILABLE : StackAvailable
	sm.From(
		deployer_pb.DeploymentStatus_WAITING,
	).Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_StackAvailable,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_AVAILABLE

		// TODO: The plan should be generated in a side effect and stored as a
		// new event.

		plan, err := planDeploymentSteps(ctx, deployment, planInput{
			stackStatus: event.StackOutput,
			flags:       deployment.Spec.Flags,
		})
		if err != nil {
			return err
		}
		deployment.Steps = plan
		return nil
	})).
		Hook(deployer_pb.DeploymentPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.DeploymentPSMHookBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackAvailable,
		) error {

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_RunSteps{}))

			return nil
		}))

	// AVAILABLE --> RUNNING : RunSteps
	sm.From(deployer_pb.DeploymentStatus_AVAILABLE).
		Transition(deployer_pb.DeploymentPSMTransition(func(
			ctx context.Context,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_RunSteps,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_RUNNING
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
		Transition(deployer_pb.DeploymentPSMTransition(func(
			ctx context.Context,
			deployment *deployer_pb.DeploymentState,
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
	sm.From(
		deployer_pb.DeploymentStatus_RUNNING,
	).Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Done,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_DONE
		return nil
	}))

	// * --> FAILED : Error
	sm.From().Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Error,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_FAILED
		return nil
	}))

	// * --> TERMINATED : Terminated
	sm.From().Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Terminated,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_TERMINATED
		return nil
	}))

	// Discard Triggered
	sm.From(
		deployer_pb.DeploymentStatus_FAILED,
		deployer_pb.DeploymentStatus_TERMINATED,
	).Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Triggered,
	) error {
		return nil
	}))

	// Discard Step Results
	sm.From(
		deployer_pb.DeploymentStatus_TERMINATED,
	).Transition(deployer_pb.DeploymentPSMTransition(func(
		ctx context.Context,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_StepResult,
	) error {
		updateDeploymentStep(deployment, event)
		return nil
	}))

	return sm, nil
}

func updateDeploymentStep(deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEventType_StepResult) {
	for _, step := range deployment.Steps {
		if step.Id == event.StepId {
			step.Status = event.Status
			step.Output = event.Output
			step.Error = event.Error
			return
		}
	}
}
