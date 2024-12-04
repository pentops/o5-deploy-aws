// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package awsdeployer_pb

import (
	context "context"
	fmt "fmt"
	psm_j5pb "github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	psm "github.com/pentops/protostate/psm"
	sqrlx "github.com/pentops/sqrlx.go/sqrlx"
)

// PSM StackPSM

type StackPSM = psm.StateMachine[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
]

type StackPSMDB = psm.DBStateMachine[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
]

type StackPSMEventSpec = psm.EventSpec[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
]

type StackPSMEventKey = string

const (
	StackPSMEventNil                 StackPSMEventKey = "<nil>"
	StackPSMEventConfigured          StackPSMEventKey = "configured"
	StackPSMEventDeploymentRequested StackPSMEventKey = "deployment_requested"
	StackPSMEventDeploymentCompleted StackPSMEventKey = "deployment_completed"
	StackPSMEventDeploymentFailed    StackPSMEventKey = "deployment_failed"
	StackPSMEventRunDeployment       StackPSMEventKey = "run_deployment"
)

// EXTEND StackKeys with the psm.IKeyset interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackKeys) PSMIsSet() bool {
	return msg != nil
}

// PSMFullName returns the full name of state machine with package prefix
func (msg *StackKeys) PSMFullName() string {
	return "o5.aws.deployer.v1.stack"
}
func (msg *StackKeys) PSMKeyValues() (map[string]string, error) {
	keyset := map[string]string{
		"stack_id": msg.StackId,
	}
	if msg.EnvironmentId != "" {
		keyset["environment_id"] = msg.EnvironmentId
	}
	if msg.ClusterId != "" {
		keyset["cluster_id"] = msg.ClusterId
	}
	return keyset, nil
}

// EXTEND StackState with the psm.IState interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackState) PSMIsSet() bool {
	return msg != nil
}

func (msg *StackState) PSMMetadata() *psm_j5pb.StateMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_j5pb.StateMetadata{}
	}
	return msg.Metadata
}

func (msg *StackState) PSMKeys() *StackKeys {
	return msg.Keys
}

func (msg *StackState) SetStatus(status StackStatus) {
	msg.Status = status
}

func (msg *StackState) SetPSMKeys(inner *StackKeys) {
	msg.Keys = inner
}

func (msg *StackState) PSMData() *StackStateData {
	if msg.Data == nil {
		msg.Data = &StackStateData{}
	}
	return msg.Data
}

// EXTEND StackStateData with the psm.IStateData interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackStateData) PSMIsSet() bool {
	return msg != nil
}

// EXTEND StackEvent with the psm.IEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackEvent) PSMIsSet() bool {
	return msg != nil
}

func (msg *StackEvent) PSMMetadata() *psm_j5pb.EventMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_j5pb.EventMetadata{}
	}
	return msg.Metadata
}

func (msg *StackEvent) PSMKeys() *StackKeys {
	return msg.Keys
}

func (msg *StackEvent) SetPSMKeys(inner *StackKeys) {
	msg.Keys = inner
}

// PSMEventKey returns the StackPSMEventPSMEventKey for the event, implementing psm.IEvent
func (msg *StackEvent) PSMEventKey() StackPSMEventKey {
	tt := msg.UnwrapPSMEvent()
	if tt == nil {
		return StackPSMEventNil
	}
	return tt.PSMEventKey()
}

// UnwrapPSMEvent implements psm.IEvent, returning the inner event message
func (msg *StackEvent) UnwrapPSMEvent() StackPSMEvent {
	if msg == nil {
		return nil
	}
	if msg.Event == nil {
		return nil
	}
	switch v := msg.Event.Type.(type) {
	case *StackEventType_Configured_:
		return v.Configured
	case *StackEventType_DeploymentRequested_:
		return v.DeploymentRequested
	case *StackEventType_DeploymentCompleted_:
		return v.DeploymentCompleted
	case *StackEventType_DeploymentFailed_:
		return v.DeploymentFailed
	case *StackEventType_RunDeployment_:
		return v.RunDeployment
	default:
		return nil
	}
}

// SetPSMEvent sets the inner event message from a concrete type, implementing psm.IEvent
func (msg *StackEvent) SetPSMEvent(inner StackPSMEvent) error {
	if msg.Event == nil {
		msg.Event = &StackEventType{}
	}
	switch v := inner.(type) {
	case *StackEventType_Configured:
		msg.Event.Type = &StackEventType_Configured_{Configured: v}
	case *StackEventType_DeploymentRequested:
		msg.Event.Type = &StackEventType_DeploymentRequested_{DeploymentRequested: v}
	case *StackEventType_DeploymentCompleted:
		msg.Event.Type = &StackEventType_DeploymentCompleted_{DeploymentCompleted: v}
	case *StackEventType_DeploymentFailed:
		msg.Event.Type = &StackEventType_DeploymentFailed_{DeploymentFailed: v}
	case *StackEventType_RunDeployment:
		msg.Event.Type = &StackEventType_RunDeployment_{RunDeployment: v}
	default:
		return fmt.Errorf("invalid type %T for StackEventType", v)
	}
	return nil
}

type StackPSMEvent interface {
	psm.IInnerEvent
	PSMEventKey() StackPSMEventKey
}

// EXTEND StackEventType_Configured with the StackPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackEventType_Configured) PSMIsSet() bool {
	return msg != nil
}

func (*StackEventType_Configured) PSMEventKey() StackPSMEventKey {
	return StackPSMEventConfigured
}

// EXTEND StackEventType_DeploymentRequested with the StackPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackEventType_DeploymentRequested) PSMIsSet() bool {
	return msg != nil
}

func (*StackEventType_DeploymentRequested) PSMEventKey() StackPSMEventKey {
	return StackPSMEventDeploymentRequested
}

// EXTEND StackEventType_DeploymentCompleted with the StackPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackEventType_DeploymentCompleted) PSMIsSet() bool {
	return msg != nil
}

func (*StackEventType_DeploymentCompleted) PSMEventKey() StackPSMEventKey {
	return StackPSMEventDeploymentCompleted
}

// EXTEND StackEventType_DeploymentFailed with the StackPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackEventType_DeploymentFailed) PSMIsSet() bool {
	return msg != nil
}

func (*StackEventType_DeploymentFailed) PSMEventKey() StackPSMEventKey {
	return StackPSMEventDeploymentFailed
}

// EXTEND StackEventType_RunDeployment with the StackPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *StackEventType_RunDeployment) PSMIsSet() bool {
	return msg != nil
}

func (*StackEventType_RunDeployment) PSMEventKey() StackPSMEventKey {
	return StackPSMEventRunDeployment
}

func StackPSMBuilder() *psm.StateMachineConfig[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
] {
	return &psm.StateMachineConfig[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
	]{}
}

func StackPSMMutation[SE StackPSMEvent](cb func(*StackStateData, SE) error) psm.TransitionMutation[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
	SE,              // Specific event type for the transition
] {
	return psm.TransitionMutation[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
		SE,              // Specific event type for the transition
	](cb)
}

type StackPSMHookBaton = psm.HookBaton[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
]

func StackPSMLogicHook[SE StackPSMEvent](cb func(context.Context, StackPSMHookBaton, *StackState, SE) error) psm.TransitionLogicHook[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
	SE,              // Specific event type for the transition
] {
	return psm.TransitionLogicHook[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
		SE,              // Specific event type for the transition
	](cb)
}
func StackPSMDataHook[SE StackPSMEvent](cb func(context.Context, sqrlx.Transaction, *StackState, SE) error) psm.TransitionDataHook[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
	SE,              // Specific event type for the transition
] {
	return psm.TransitionDataHook[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
		SE,              // Specific event type for the transition
	](cb)
}
func StackPSMLinkHook[SE StackPSMEvent, DK psm.IKeyset, DIE psm.IInnerEvent](
	linkDestination psm.LinkDestination[DK, DIE],
	cb func(context.Context, *StackState, SE, func(DK, DIE)) error,
) psm.LinkHook[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
	SE,              // Specific event type for the transition
	DK,              // Destination Keys
	DIE,             // Destination Inner Event
] {
	return psm.LinkHook[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
		SE,              // Specific event type for the transition
		DK,              // Destination Keys
		DIE,             // Destination Inner Event
	]{
		Derive:      cb,
		Destination: linkDestination,
	}
}
func StackPSMGeneralLogicHook(cb func(context.Context, StackPSMHookBaton, *StackState, *StackEvent) error) psm.GeneralLogicHook[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralLogicHook[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
func StackPSMGeneralStateDataHook(cb func(context.Context, sqrlx.Transaction, *StackState) error) psm.GeneralStateDataHook[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralStateDataHook[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
func StackPSMGeneralEventDataHook(cb func(context.Context, sqrlx.Transaction, *StackState, *StackEvent) error) psm.GeneralEventDataHook[
	*StackKeys,      // implements psm.IKeyset
	*StackState,     // implements psm.IState
	StackStatus,     // implements psm.IStatusEnum
	*StackStateData, // implements psm.IStateData
	*StackEvent,     // implements psm.IEvent
	StackPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralEventDataHook[
		*StackKeys,      // implements psm.IKeyset
		*StackState,     // implements psm.IState
		StackStatus,     // implements psm.IStatusEnum
		*StackStateData, // implements psm.IStateData
		*StackEvent,     // implements psm.IEvent
		StackPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
