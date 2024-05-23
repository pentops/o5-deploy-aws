// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package deployer_pb

import (
	context "context"
	fmt "fmt"
	psm_pb "github.com/pentops/protostate/gen/state/v1/psm_pb"
	psm "github.com/pentops/protostate/psm"
	sqrlx "github.com/pentops/sqrlx.go/sqrlx"
)

// PSM ClusterPSM

type ClusterPSM = psm.StateMachine[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]

type ClusterPSMDB = psm.DBStateMachine[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]

type ClusterPSMEventer = psm.Eventer[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]

type ClusterPSMEventSpec = psm.EventSpec[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]

type ClusterPSMEventKey = string

const (
	ClusterPSMEventNil        ClusterPSMEventKey = "<nil>"
	ClusterPSMEventConfigured ClusterPSMEventKey = "configured"
)

// EXTEND ClusterKeys with the psm.IKeyset interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *ClusterKeys) PSMIsSet() bool {
	return msg != nil
}

// PSMFullName returns the full name of state machine with package prefix
func (msg *ClusterKeys) PSMFullName() string {
	return "o5.deployer.v1.cluster"
}

// EXTEND ClusterState with the psm.IState interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *ClusterState) PSMIsSet() bool {
	return msg != nil
}

func (msg *ClusterState) PSMMetadata() *psm_pb.StateMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_pb.StateMetadata{}
	}
	return msg.Metadata
}

func (msg *ClusterState) PSMKeys() *ClusterKeys {
	return msg.Keys
}

func (msg *ClusterState) SetStatus(status ClusterStatus) {
	msg.Status = status
}

func (msg *ClusterState) SetPSMKeys(inner *ClusterKeys) {
	msg.Keys = inner
}

func (msg *ClusterState) PSMData() *ClusterStateData {
	if msg.Data == nil {
		msg.Data = &ClusterStateData{}
	}
	return msg.Data
}

// EXTEND ClusterStateData with the psm.IStateData interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *ClusterStateData) PSMIsSet() bool {
	return msg != nil
}

// EXTEND ClusterEvent with the psm.IEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *ClusterEvent) PSMIsSet() bool {
	return msg != nil
}

func (msg *ClusterEvent) PSMMetadata() *psm_pb.EventMetadata {
	if msg.Metadata == nil {
		msg.Metadata = &psm_pb.EventMetadata{}
	}
	return msg.Metadata
}

func (msg *ClusterEvent) PSMKeys() *ClusterKeys {
	return msg.Keys
}

func (msg *ClusterEvent) SetPSMKeys(inner *ClusterKeys) {
	msg.Keys = inner
}

// PSMEventKey returns the ClusterPSMEventPSMEventKey for the event, implementing psm.IEvent
func (msg *ClusterEvent) PSMEventKey() ClusterPSMEventKey {
	tt := msg.UnwrapPSMEvent()
	if tt == nil {
		return ClusterPSMEventNil
	}
	return tt.PSMEventKey()
}

// UnwrapPSMEvent implements psm.IEvent, returning the inner event message
func (msg *ClusterEvent) UnwrapPSMEvent() ClusterPSMEvent {
	if msg == nil {
		return nil
	}
	if msg.Event == nil {
		return nil
	}
	switch v := msg.Event.Type.(type) {
	case *ClusterEventType_Configured_:
		return v.Configured
	default:
		return nil
	}
}

// SetPSMEvent sets the inner event message from a concrete type, implementing psm.IEvent
func (msg *ClusterEvent) SetPSMEvent(inner ClusterPSMEvent) error {
	if msg.Event == nil {
		msg.Event = &ClusterEventType{}
	}
	switch v := inner.(type) {
	case *ClusterEventType_Configured:
		msg.Event.Type = &ClusterEventType_Configured_{Configured: v}
	default:
		return fmt.Errorf("invalid type %T for ClusterEventType", v)
	}
	return nil
}

type ClusterPSMEvent interface {
	psm.IInnerEvent
	PSMEventKey() ClusterPSMEventKey
}

// EXTEND ClusterEventType_Configured with the ClusterPSMEvent interface

// PSMIsSet is a helper for != nil, which does not work with generic parameters
func (msg *ClusterEventType_Configured) PSMIsSet() bool {
	return msg != nil
}

func (*ClusterEventType_Configured) PSMEventKey() ClusterPSMEventKey {
	return ClusterPSMEventConfigured
}

type ClusterPSMTableSpec = psm.PSMTableSpec[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]

var DefaultClusterPSMTableSpec = ClusterPSMTableSpec{
	State: psm.TableSpec[*ClusterState]{
		TableName:  "cluster",
		DataColumn: "state",
		StoreExtraColumns: func(state *ClusterState) (map[string]interface{}, error) {
			return map[string]interface{}{}, nil
		},
		PKFieldPaths: []string{
			"keys.cluster_id",
		},
	},
	Event: psm.TableSpec[*ClusterEvent]{
		TableName:  "cluster_event",
		DataColumn: "data",
		StoreExtraColumns: func(event *ClusterEvent) (map[string]interface{}, error) {
			metadata := event.Metadata
			return map[string]interface{}{
				"id":         metadata.EventId,
				"timestamp":  metadata.Timestamp,
				"cause":      metadata.Cause,
				"sequence":   metadata.Sequence,
				"cluster_id": event.Keys.ClusterId,
			}, nil
		},
		PKFieldPaths: []string{
			"metadata.EventId",
		},
	},
	EventPrimaryKey: func(id string, keys *ClusterKeys) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id": id,
		}, nil
	},
	PrimaryKey: func(keys *ClusterKeys) (map[string]interface{}, error) {
		return map[string]interface{}{
			"id": keys.ClusterId,
		}, nil
	},
}

func DefaultClusterPSMConfig() *psm.StateMachineConfig[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.NewStateMachineConfig[
		*ClusterKeys,      // implements psm.IKeyset
		*ClusterState,     // implements psm.IState
		ClusterStatus,     // implements psm.IStatusEnum
		*ClusterStateData, // implements psm.IStateData
		*ClusterEvent,     // implements psm.IEvent
		ClusterPSMEvent,   // implements psm.IInnerEvent
	](DefaultClusterPSMTableSpec)
}

func NewClusterPSM(config *psm.StateMachineConfig[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]) (*ClusterPSM, error) {
	return psm.NewStateMachine[
		*ClusterKeys,      // implements psm.IKeyset
		*ClusterState,     // implements psm.IState
		ClusterStatus,     // implements psm.IStatusEnum
		*ClusterStateData, // implements psm.IStateData
		*ClusterEvent,     // implements psm.IEvent
		ClusterPSMEvent,   // implements psm.IInnerEvent
	](config)
}

func ClusterPSMMutation[SE ClusterPSMEvent](cb func(*ClusterStateData, SE) error) psm.PSMMutationFunc[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
	SE,                // Specific event type for the transition
] {
	return psm.PSMMutationFunc[
		*ClusterKeys,      // implements psm.IKeyset
		*ClusterState,     // implements psm.IState
		ClusterStatus,     // implements psm.IStatusEnum
		*ClusterStateData, // implements psm.IStateData
		*ClusterEvent,     // implements psm.IEvent
		ClusterPSMEvent,   // implements psm.IInnerEvent
		SE,                // Specific event type for the transition
	](cb)
}

type ClusterPSMHookBaton = psm.HookBaton[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
]

func ClusterPSMHook[SE ClusterPSMEvent](cb func(context.Context, sqrlx.Transaction, ClusterPSMHookBaton, *ClusterState, SE) error) psm.PSMHookFunc[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
	SE,                // Specific event type for the transition
] {
	return psm.PSMHookFunc[
		*ClusterKeys,      // implements psm.IKeyset
		*ClusterState,     // implements psm.IState
		ClusterStatus,     // implements psm.IStatusEnum
		*ClusterStateData, // implements psm.IStateData
		*ClusterEvent,     // implements psm.IEvent
		ClusterPSMEvent,   // implements psm.IInnerEvent
		SE,                // Specific event type for the transition
	](cb)
}
func ClusterPSMGeneralHook(cb func(context.Context, sqrlx.Transaction, ClusterPSMHookBaton, *ClusterState, *ClusterEvent) error) psm.GeneralStateHook[
	*ClusterKeys,      // implements psm.IKeyset
	*ClusterState,     // implements psm.IState
	ClusterStatus,     // implements psm.IStatusEnum
	*ClusterStateData, // implements psm.IStateData
	*ClusterEvent,     // implements psm.IEvent
	ClusterPSMEvent,   // implements psm.IInnerEvent
] {
	return psm.GeneralStateHook[
		*ClusterKeys,      // implements psm.IKeyset
		*ClusterState,     // implements psm.IState
		ClusterStatus,     // implements psm.IStatusEnum
		*ClusterStateData, // implements psm.IStateData
		*ClusterEvent,     // implements psm.IEvent
		ClusterPSMEvent,   // implements psm.IInnerEvent
	](cb)
}
