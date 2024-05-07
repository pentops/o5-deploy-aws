// Code generated by protoc-gen-go-sugar. DO NOT EDIT.

package deployer_pb

import (
	driver "database/sql/driver"
	fmt "fmt"
)

// CodeSourceType is a oneof wrapper
type CodeSourceTypeKey string

const (
	CodeSource_Github CodeSourceTypeKey = "github"
)

func (x *CodeSourceType) TypeKey() (CodeSourceTypeKey, bool) {
	switch x.Type.(type) {
	case *CodeSourceType_Github_:
		return CodeSource_Github, true
	default:
		return "", false
	}
}

type IsCodeSourceTypeWrappedType interface {
	TypeKey() CodeSourceTypeKey
}

func (x *CodeSourceType) Set(val IsCodeSourceTypeWrappedType) {
	switch v := val.(type) {
	case *CodeSourceType_Github:
		x.Type = &CodeSourceType_Github_{Github: v}
	}
}
func (x *CodeSourceType) Get() IsCodeSourceTypeWrappedType {
	switch v := x.Type.(type) {
	case *CodeSourceType_Github_:
		return v.Github
	default:
		return nil
	}
}
func (x *CodeSourceType_Github) TypeKey() CodeSourceTypeKey {
	return CodeSource_Github
}

type IsCodeSourceType_Type = isCodeSourceType_Type

// DeploymentEventType is a oneof wrapper
type DeploymentEventTypeKey string

const (
	DeploymentEvent_Created          DeploymentEventTypeKey = "created"
	DeploymentEvent_Triggered        DeploymentEventTypeKey = "triggered"
	DeploymentEvent_StackWait        DeploymentEventTypeKey = "stackWait"
	DeploymentEvent_StackWaitFailure DeploymentEventTypeKey = "stackWaitFailure"
	DeploymentEvent_StackAvailable   DeploymentEventTypeKey = "stackAvailable"
	DeploymentEvent_RunSteps         DeploymentEventTypeKey = "runSteps"
	DeploymentEvent_StepResult       DeploymentEventTypeKey = "stepResult"
	DeploymentEvent_Error            DeploymentEventTypeKey = "error"
	DeploymentEvent_Done             DeploymentEventTypeKey = "done"
	DeploymentEvent_Terminated       DeploymentEventTypeKey = "terminated"
)

func (x *DeploymentEventType) TypeKey() (DeploymentEventTypeKey, bool) {
	switch x.Type.(type) {
	case *DeploymentEventType_Created_:
		return DeploymentEvent_Created, true
	case *DeploymentEventType_Triggered_:
		return DeploymentEvent_Triggered, true
	case *DeploymentEventType_StackWait_:
		return DeploymentEvent_StackWait, true
	case *DeploymentEventType_StackWaitFailure_:
		return DeploymentEvent_StackWaitFailure, true
	case *DeploymentEventType_StackAvailable_:
		return DeploymentEvent_StackAvailable, true
	case *DeploymentEventType_RunSteps_:
		return DeploymentEvent_RunSteps, true
	case *DeploymentEventType_StepResult_:
		return DeploymentEvent_StepResult, true
	case *DeploymentEventType_Error_:
		return DeploymentEvent_Error, true
	case *DeploymentEventType_Done_:
		return DeploymentEvent_Done, true
	case *DeploymentEventType_Terminated_:
		return DeploymentEvent_Terminated, true
	default:
		return "", false
	}
}

type IsDeploymentEventTypeWrappedType interface {
	TypeKey() DeploymentEventTypeKey
}

func (x *DeploymentEventType) Set(val IsDeploymentEventTypeWrappedType) {
	switch v := val.(type) {
	case *DeploymentEventType_Created:
		x.Type = &DeploymentEventType_Created_{Created: v}
	case *DeploymentEventType_Triggered:
		x.Type = &DeploymentEventType_Triggered_{Triggered: v}
	case *DeploymentEventType_StackWait:
		x.Type = &DeploymentEventType_StackWait_{StackWait: v}
	case *DeploymentEventType_StackWaitFailure:
		x.Type = &DeploymentEventType_StackWaitFailure_{StackWaitFailure: v}
	case *DeploymentEventType_StackAvailable:
		x.Type = &DeploymentEventType_StackAvailable_{StackAvailable: v}
	case *DeploymentEventType_RunSteps:
		x.Type = &DeploymentEventType_RunSteps_{RunSteps: v}
	case *DeploymentEventType_StepResult:
		x.Type = &DeploymentEventType_StepResult_{StepResult: v}
	case *DeploymentEventType_Error:
		x.Type = &DeploymentEventType_Error_{Error: v}
	case *DeploymentEventType_Done:
		x.Type = &DeploymentEventType_Done_{Done: v}
	case *DeploymentEventType_Terminated:
		x.Type = &DeploymentEventType_Terminated_{Terminated: v}
	}
}
func (x *DeploymentEventType) Get() IsDeploymentEventTypeWrappedType {
	switch v := x.Type.(type) {
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
func (x *DeploymentEventType_Created) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_Created
}
func (x *DeploymentEventType_Triggered) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_Triggered
}
func (x *DeploymentEventType_StackWait) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_StackWait
}
func (x *DeploymentEventType_StackWaitFailure) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_StackWaitFailure
}
func (x *DeploymentEventType_StackAvailable) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_StackAvailable
}
func (x *DeploymentEventType_RunSteps) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_RunSteps
}
func (x *DeploymentEventType_StepResult) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_StepResult
}
func (x *DeploymentEventType_Error) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_Error
}
func (x *DeploymentEventType_Done) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_Done
}
func (x *DeploymentEventType_Terminated) TypeKey() DeploymentEventTypeKey {
	return DeploymentEvent_Terminated
}

type IsDeploymentEventType_Type = isDeploymentEventType_Type

// DeploymentStatus
const (
	DeploymentStatus_UNSPECIFIED DeploymentStatus = 0
	DeploymentStatus_QUEUED      DeploymentStatus = 1
	DeploymentStatus_TRIGGERED   DeploymentStatus = 2
	DeploymentStatus_WAITING     DeploymentStatus = 3
	DeploymentStatus_AVAILABLE   DeploymentStatus = 4
	DeploymentStatus_RUNNING     DeploymentStatus = 5
	DeploymentStatus_DONE        DeploymentStatus = 100
	DeploymentStatus_FAILED      DeploymentStatus = 101
	DeploymentStatus_TERMINATED  DeploymentStatus = 102
)

var (
	DeploymentStatus_name_short = map[int32]string{
		0:   "UNSPECIFIED",
		1:   "QUEUED",
		2:   "TRIGGERED",
		3:   "WAITING",
		4:   "AVAILABLE",
		5:   "RUNNING",
		100: "DONE",
		101: "FAILED",
		102: "TERMINATED",
	}
	DeploymentStatus_value_short = map[string]int32{
		"UNSPECIFIED": 0,
		"QUEUED":      1,
		"TRIGGERED":   2,
		"WAITING":     3,
		"AVAILABLE":   4,
		"RUNNING":     5,
		"DONE":        100,
		"FAILED":      101,
		"TERMINATED":  102,
	}
	DeploymentStatus_value_either = map[string]int32{
		"UNSPECIFIED":                   0,
		"DEPLOYMENT_STATUS_UNSPECIFIED": 0,
		"QUEUED":                        1,
		"DEPLOYMENT_STATUS_QUEUED":      1,
		"TRIGGERED":                     2,
		"DEPLOYMENT_STATUS_TRIGGERED":   2,
		"WAITING":                       3,
		"DEPLOYMENT_STATUS_WAITING":     3,
		"AVAILABLE":                     4,
		"DEPLOYMENT_STATUS_AVAILABLE":   4,
		"RUNNING":                       5,
		"DEPLOYMENT_STATUS_RUNNING":     5,
		"DONE":                          100,
		"DEPLOYMENT_STATUS_DONE":        100,
		"FAILED":                        101,
		"DEPLOYMENT_STATUS_FAILED":      101,
		"TERMINATED":                    102,
		"DEPLOYMENT_STATUS_TERMINATED":  102,
	}
)

// ShortString returns the un-prefixed string representation of the enum value
func (x DeploymentStatus) ShortString() string {
	return DeploymentStatus_name_short[int32(x)]
}
func (x DeploymentStatus) Value() (driver.Value, error) {
	return []uint8(x.ShortString()), nil
}
func (x *DeploymentStatus) Scan(value interface{}) error {
	var strVal string
	switch vt := value.(type) {
	case []uint8:
		strVal = string(vt)
	case string:
		strVal = vt
	default:
		return fmt.Errorf("invalid type %T", value)
	}
	val := DeploymentStatus_value_either[strVal]
	*x = DeploymentStatus(val)
	return nil
}
