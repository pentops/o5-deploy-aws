// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package awsdeployer_pb

import (
	context "context"
	fmt "fmt"
	psm_j5pb "github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	psm "github.com/pentops/protostate/psm"
	sqrlx "github.com/pentops/sqrlx.go/sqrlx"
)

// PSM DeploymentPSM

type DeploymentPSM = psm.StateMachine[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
]

type DeploymentPSMDB = psm.DBStateMachine[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
]

type DeploymentPSMEventSpec = psm.EventSpec[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
]

type DeploymentPSMEventKey = string

const (
	DeploymentPSMEventNil              DeploymentPSMEventKey = "<nil>"
	DeploymentPSMEventCreated          DeploymentPSMEventKey = "created"
	DeploymentPSMEventTriggered        DeploymentPSMEventKey = "triggered"
	DeploymentPSMEventStackWait        DeploymentPSMEventKey = "stack_wait"
	DeploymentPSMEventStackWaitFailure DeploymentPSMEventKey = "stack_wait_failure"
	DeploymentPSMEventStackAvailable   DeploymentPSMEventKey = "stack_available"
	DeploymentPSMEventRunSteps         DeploymentPSMEventKey = "run_steps"
	DeploymentPSMEventStepResult       DeploymentPSMEventKey = "step_result"
	DeploymentPSMEventRunStep          DeploymentPSMEventKey = "run_step"
	DeploymentPSMEventError            DeploymentPSMEventKey = "error"
	DeploymentPSMEventDone             DeploymentPSMEventKey = "done"
	DeploymentPSMEventTerminated       DeploymentPSMEventKey = "terminated"
)

// EXTEND DeploymentKeys with the psm.IKeyset interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentKeys) PSMIsSet() bool {
	return msg != nil
}

// PSMFullName returns the full name of state machine with package prefix
func (msg *DeploymentKeys) PSMFullName() string {
	return "o5.aws.deployer.v1.deployment"
}
func (msg *DeploymentKeys) PSMKeyValues() (map[string]string, error) {
	keyset := map[string]string{
		"deployment_id": msg.DeploymentId,
	}
	if msg.StackId != "" {
		keyset["stack_id"] = msg.StackId
	}
	if msg.EnvironmentId != "" {
		keyset["environment_id"] = msg.EnvironmentId
	}
	if msg.ClusterId != "" {
		keyset["cluster_id"] = msg.ClusterId
	}
	return keyset, nil
}

// EXTEND DeploymentState with the psm.IState interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentState) PSMIsSet() bool {
	return msg != nil
}

func (msg *DeploymentState) PSMMetadata() *psm_j5pb.StateMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_j5pb.StateMetadata{}
	}
	return msg.Metadata
}

func (msg *DeploymentState) PSMKeys() *DeploymentKeys {
	return msg.Keys
}

func (msg *DeploymentState) SetStatus(status DeploymentStatus) {
	msg.Status = status
}

func (msg *DeploymentState) SetPSMKeys(inner *DeploymentKeys) {
	msg.Keys = inner
}

func (msg *DeploymentState) PSMData() *DeploymentStateData {
	if msg.Data == nil {
		msg.Data = &DeploymentStateData{}
	}
	return msg.Data
}

// EXTEND DeploymentStateData with the psm.IStateData interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentStateData) PSMIsSet() bool {
	return msg != nil
}

// EXTEND DeploymentEvent with the psm.IEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEvent) PSMIsSet() bool {
	return msg != nil
}

func (msg *DeploymentEvent) PSMMetadata() *psm_j5pb.EventMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_j5pb.EventMetadata{}
	}
	return msg.Metadata
}

func (msg *DeploymentEvent) PSMKeys() *DeploymentKeys {
	return msg.Keys
}

func (msg *DeploymentEvent) SetPSMKeys(inner *DeploymentKeys) {
	msg.Keys = inner
}

// PSMEventKey returns the DeploymentPSMEventPSMEventKey for the event, implementing psm.IEvent
func (msg *DeploymentEvent) PSMEventKey() DeploymentPSMEventKey {
	tt := msg.UnwrapPSMEvent()
	if tt == nil {
		return DeploymentPSMEventNil
	}
	return tt.PSMEventKey()
}

// UnwrapPSMEvent implements psm.IEvent, returning the inner event message
func (msg *DeploymentEvent) UnwrapPSMEvent() DeploymentPSMEvent {
	if msg == nil {
		return nil
	}
	if msg.Event == nil {
		return nil
	}
	switch v := msg.Event.Type.(type) {
	case *DeploymentEventType_Created_:
		return v.Created
	case *DeploymentEventType_Triggered_:
		return v.Triggered
	case *DeploymentEventType_StackWait_:
		return v.StackWait
	case *DeploymentEventType_StackWaitFailure_:
		return v.StackWaitFailure
	case *DeploymentEventType_StackAvailable_:
		return v.StackAvailable
	case *DeploymentEventType_RunSteps_:
		return v.RunSteps
	case *DeploymentEventType_StepResult_:
		return v.StepResult
	case *DeploymentEventType_RunStep_:
		return v.RunStep
	case *DeploymentEventType_Error_:
		return v.Error
	case *DeploymentEventType_Done_:
		return v.Done
	case *DeploymentEventType_Terminated_:
		return v.Terminated
	default:
		return nil
	}
}

// SetPSMEvent sets the inner event message from a concrete type, implementing psm.IEvent
func (msg *DeploymentEvent) SetPSMEvent(inner DeploymentPSMEvent) error {
	if msg.Event == nil {
		msg.Event = &DeploymentEventType{}
	}
	switch v := inner.(type) {
	case *DeploymentEventType_Created:
		msg.Event.Type = &DeploymentEventType_Created_{Created: v}
	case *DeploymentEventType_Triggered:
		msg.Event.Type = &DeploymentEventType_Triggered_{Triggered: v}
	case *DeploymentEventType_StackWait:
		msg.Event.Type = &DeploymentEventType_StackWait_{StackWait: v}
	case *DeploymentEventType_StackWaitFailure:
		msg.Event.Type = &DeploymentEventType_StackWaitFailure_{StackWaitFailure: v}
	case *DeploymentEventType_StackAvailable:
		msg.Event.Type = &DeploymentEventType_StackAvailable_{StackAvailable: v}
	case *DeploymentEventType_RunSteps:
		msg.Event.Type = &DeploymentEventType_RunSteps_{RunSteps: v}
	case *DeploymentEventType_StepResult:
		msg.Event.Type = &DeploymentEventType_StepResult_{StepResult: v}
	case *DeploymentEventType_RunStep:
		msg.Event.Type = &DeploymentEventType_RunStep_{RunStep: v}
	case *DeploymentEventType_Error:
		msg.Event.Type = &DeploymentEventType_Error_{Error: v}
	case *DeploymentEventType_Done:
		msg.Event.Type = &DeploymentEventType_Done_{Done: v}
	case *DeploymentEventType_Terminated:
		msg.Event.Type = &DeploymentEventType_Terminated_{Terminated: v}
	default:
		return fmt.Errorf("invalid type %T for DeploymentEventType", v)
	}
	return nil
}

type DeploymentPSMEvent interface {
	psm.IInnerEvent
	PSMEventKey() DeploymentPSMEventKey
}

// EXTEND DeploymentEventType_Created with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_Created) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_Created) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventCreated
}

// EXTEND DeploymentEventType_Triggered with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_Triggered) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_Triggered) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventTriggered
}

// EXTEND DeploymentEventType_StackWait with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_StackWait) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_StackWait) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventStackWait
}

// EXTEND DeploymentEventType_StackWaitFailure with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_StackWaitFailure) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_StackWaitFailure) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventStackWaitFailure
}

// EXTEND DeploymentEventType_StackAvailable with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_StackAvailable) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_StackAvailable) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventStackAvailable
}

// EXTEND DeploymentEventType_RunSteps with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_RunSteps) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_RunSteps) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventRunSteps
}

// EXTEND DeploymentEventType_StepResult with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_StepResult) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_StepResult) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventStepResult
}

// EXTEND DeploymentEventType_RunStep with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_RunStep) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_RunStep) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventRunStep
}

// EXTEND DeploymentEventType_Error with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_Error) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_Error) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventError
}

// EXTEND DeploymentEventType_Done with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_Done) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_Done) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventDone
}

// EXTEND DeploymentEventType_Terminated with the DeploymentPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *DeploymentEventType_Terminated) PSMIsSet() bool {
	return msg != nil
}

func (*DeploymentEventType_Terminated) PSMEventKey() DeploymentPSMEventKey {
	return DeploymentPSMEventTerminated
}

func DeploymentPSMBuilder() *psm.StateMachineConfig[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
] {
	return &psm.StateMachineConfig[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
	]{}
}

func DeploymentPSMMutation[SE DeploymentPSMEvent](cb func(*DeploymentStateData, SE) error) psm.TransitionMutation[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
	SE,                   // Specific event type for the transition
] {
	return psm.TransitionMutation[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
		SE,                   // Specific event type for the transition
	](cb)
}

type DeploymentPSMHookBaton = psm.HookBaton[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
]

func DeploymentPSMLogicHook[SE DeploymentPSMEvent](cb func(context.Context, DeploymentPSMHookBaton, *DeploymentState, SE) error) psm.TransitionLogicHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
	SE,                   // Specific event type for the transition
] {
	return psm.TransitionLogicHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
		SE,                   // Specific event type for the transition
	](cb)
}
func DeploymentPSMDataHook[SE DeploymentPSMEvent](cb func(context.Context, sqrlx.Transaction, *DeploymentState, SE) error) psm.TransitionDataHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
	SE,                   // Specific event type for the transition
] {
	return psm.TransitionDataHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
		SE,                   // Specific event type for the transition
	](cb)
}
func DeploymentPSMLinkHook[SE DeploymentPSMEvent, DK psm.IKeyset, DIE psm.IInnerEvent](
	linkDestination psm.LinkDestination[DK, DIE],
	cb func(context.Context, *DeploymentState, SE, func(DK, DIE)) error,
) psm.LinkHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
	SE,                   // Specific event type for the transition
	DK,                   // Destination Keys
	DIE,                  // Destination Inner Event
] {
	return psm.LinkHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
		SE,                   // Specific event type for the transition
		DK,                   // Destination Keys
		DIE,                  // Destination Inner Event
	]{
		Derive:      cb,
		Destination: linkDestination,
	}
}
func DeploymentPSMGeneralLogicHook(cb func(context.Context, DeploymentPSMHookBaton, *DeploymentState, *DeploymentEvent) error) psm.GeneralLogicHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralLogicHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
func DeploymentPSMGeneralStateDataHook(cb func(context.Context, sqrlx.Transaction, *DeploymentState) error) psm.GeneralStateDataHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralStateDataHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
func DeploymentPSMGeneralEventDataHook(cb func(context.Context, sqrlx.Transaction, *DeploymentState, *DeploymentEvent) error) psm.GeneralEventDataHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralEventDataHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
func DeploymentPSMEventPublishHook(cb func(context.Context, psm.Publisher, *DeploymentState, *DeploymentEvent) error) psm.EventPublishHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
	*DeploymentEvent,     // implements psm.IEvent
	DeploymentPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.EventPublishHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
		*DeploymentEvent,     // implements psm.IEvent
		DeploymentPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
func DeploymentPSMUpsertPublishHook(cb func(context.Context, psm.Publisher, *DeploymentState) error) psm.UpsertPublishHook[
	*DeploymentKeys,      // implements psm.IKeyset
	*DeploymentState,     // implements psm.IState
	DeploymentStatus,     // implements psm.IStatusEnum
	*DeploymentStateData, // implements psm.IStateData
] {
	return psm.UpsertPublishHook[
		*DeploymentKeys,      // implements psm.IKeyset
		*DeploymentState,     // implements psm.IState
		DeploymentStatus,     // implements psm.IStatusEnum
		*DeploymentStateData, // implements psm.IStateData
	](cb)
}

func (event *DeploymentEvent) EventPublishMetadata() *psm_j5pb.EventPublishMetadata {
	tenantKeys := make([]*psm_j5pb.EventTenant, 0)
	return &psm_j5pb.EventPublishMetadata{
		EventId:   event.Metadata.EventId,
		Sequence:  event.Metadata.Sequence,
		Timestamp: event.Metadata.Timestamp,
		Cause:     event.Metadata.Cause,
		Auth: &psm_j5pb.PublishAuth{
			TenantKeys: tenantKeys,
		},
	}
}
