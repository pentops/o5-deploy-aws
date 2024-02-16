package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
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
	md := tb.FullCause().Metadata
	de := &deployer_pb.DeploymentEvent{
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor:     md.Actor,
		},
		DeploymentId: tb.FullCause().DeploymentId,
		Event:        &deployer_pb.DeploymentEventType{},
	}
	de.Event.Set(event)
	return de
}

func DeploymentTableSpec() deployer_pb.DeploymentPSMTableSpec {
	tableSpec := deployer_pb.DefaultDeploymentPSMTableSpec
	tableSpec.EventDataColumn = "event"
	tableSpec.StateDataColumn = "state"
	tableSpec.StateColumns = func(s *deployer_pb.DeploymentState) (map[string]interface{}, error) {
		return map[string]interface{}{
			"stack_id": s.StackId,
		}, nil
	}

	tableSpec.EventColumns = func(e *deployer_pb.DeploymentEvent) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id":            e.Metadata.EventId,
			"deployment_id": e.DeploymentId,
			"event":         e,
			"timestamp":     e.Metadata.Timestamp,
		}, nil
	}

	return tableSpec
}

type deployerConversions struct {
	deployer_pb.DeploymentPSMConverter
}

func (c deployerConversions) EventLabel(event deployer_pb.DeploymentPSMEvent) string {
	typeKey := string(event.PSMEventKey())

	// TODO: This by generation and annotation
	if stackStatus, ok := event.(*deployer_pb.DeploymentEventType_StepResult); ok {
		typeKey = fmt.Sprintf("%s.%s", typeKey, stackStatus.Status.ShortString())
	}
	return typeKey
}

func NewDeploymentEventer() (*deployer_pb.DeploymentPSM, error) {

	config := deployer_pb.DefaultDeploymentPSMConfig().WithTableSpec(DeploymentTableSpec()).WithEventTypeConverter(deployerConversions{})
	sm, err := deployer_pb.NewDeploymentPSM(config)
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
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
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
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_Triggered,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_TRIGGERED

			tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_StackWait{}))

			return nil
		}))

	// TRIGGERED --> WAITING : StackWait
	sm.From(deployer_pb.DeploymentStatus_TRIGGERED).
		Do(deployer_pb.DeploymentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.DeploymentPSMTransitionBaton,
			deployment *deployer_pb.DeploymentState,
			event *deployer_pb.DeploymentEventType_StackWait,
		) error {
			deployment.Status = deployer_pb.DeploymentStatus_WAITING
			//deployment.WaitingOnRemotePhase = proto.String("wait")

			requestMetadata, err := buildRequestMetadata(&deployer_pb.StepContext{
				Phase:        deployer_pb.StepPhase_WAIT,
				DeploymentId: deployment.DeploymentId,
			})
			if err != nil {
				return err
			}

			tb.SideEffect(&deployer_tpb.StabalizeStackMessage{
				Request:      requestMetadata,
				StackName:    deployment.StackName,
				CancelUpdate: deployment.Spec.CancelUpdates,
			})

			return nil
		}))

	// WAITIHG --> FAILED : StackWaitFailure
	sm.From(
		deployer_pb.DeploymentStatus_WAITING,
	).Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_StackWaitFailure,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_FAILED
		return fmt.Errorf("stack failed: %s", event.Error)
	}))

	// WAITING --> AVAILABLE : StackAvailable
	sm.From(
		deployer_pb.DeploymentStatus_WAITING,
	).Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_StackAvailable,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_AVAILABLE

		plan, err := planDeploymentSteps(ctx, deployment, planInput{
			stackExists: event.StackExists,
			quickMode:   deployment.Spec.QuickMode,
		})
		if err != nil {
			return err
		}
		deployment.Steps = plan

		tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_RunSteps{}))

		return nil
	}))

	// AVAILABLE --> RUNNING : RunSteps
	sm.From(
		deployer_pb.DeploymentStatus_AVAILABLE,
	).Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_RunSteps,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_RUNNING
		return stepNext(ctx, tb, deployment)
	}))

	// RUNNING --> RUNNING : StepResult
	sm.From(
		deployer_pb.DeploymentStatus_RUNNING,
	).Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_StepResult,
	) error {

		for _, step := range deployment.Steps {
			if step.Id == event.StepId {
				step.Status = event.Status
				step.Output = event.Output
				step.Error = event.Error
				break
			}
		}

		return stepNext(ctx, tb, deployment)
	}))

	// RUNNING --> DONE : Done
	sm.From(
		deployer_pb.DeploymentStatus_RUNNING,
	).Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Done,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_DONE
		return nil
	}))

	// * --> FAILED : Error
	sm.From().Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Error,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_FAILED
		return nil
	}))

	// * --> TERMINATED : Terminated
	sm.From().Do(deployer_pb.DeploymentPSMFunc(func(
		ctx context.Context,
		tb deployer_pb.DeploymentPSMTransitionBaton,
		deployment *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Terminated,
	) error {
		deployment.Status = deployer_pb.DeploymentStatus_TERMINATED
		return nil
	}))

	return sm, nil
}
