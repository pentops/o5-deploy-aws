// Code generated by protoc-gen-go-sugar. DO NOT EDIT.

package deployer_tpb

// ECSTaskEventType is a oneof wrapper
type ECSTaskEventTypeKey string

const (
	ECSTaskEvent_Pending ECSTaskEventTypeKey = "pending"
	ECSTaskEvent_Running ECSTaskEventTypeKey = "running"
	ECSTaskEvent_Failed  ECSTaskEventTypeKey = "failed"
	ECSTaskEvent_Exited  ECSTaskEventTypeKey = "exited"
	ECSTaskEvent_Stopped ECSTaskEventTypeKey = "stopped"
)

func (x *ECSTaskEventType) TypeKey() (ECSTaskEventTypeKey, bool) {
	switch x.Type.(type) {
	case *ECSTaskEventType_Pending_:
		return ECSTaskEvent_Pending, true
	case *ECSTaskEventType_Running_:
		return ECSTaskEvent_Running, true
	case *ECSTaskEventType_Failed_:
		return ECSTaskEvent_Failed, true
	case *ECSTaskEventType_Exited_:
		return ECSTaskEvent_Exited, true
	case *ECSTaskEventType_Stopped_:
		return ECSTaskEvent_Stopped, true
	default:
		return "", false
	}
}

type IsECSTaskEventTypeWrappedType interface {
	TypeKey() ECSTaskEventTypeKey
}

func (x *ECSTaskEventType) Set(val IsECSTaskEventTypeWrappedType) {
	switch v := val.(type) {
	case *ECSTaskEventType_Pending:
		x.Type = &ECSTaskEventType_Pending_{Pending: v}
	case *ECSTaskEventType_Running:
		x.Type = &ECSTaskEventType_Running_{Running: v}
	case *ECSTaskEventType_Failed:
		x.Type = &ECSTaskEventType_Failed_{Failed: v}
	case *ECSTaskEventType_Exited:
		x.Type = &ECSTaskEventType_Exited_{Exited: v}
	case *ECSTaskEventType_Stopped:
		x.Type = &ECSTaskEventType_Stopped_{Stopped: v}
	}
}
func (x *ECSTaskEventType) Get() IsECSTaskEventTypeWrappedType {
	switch v := x.Type.(type) {
	case *ECSTaskEventType_Pending_:
		return v.Pending
	case *ECSTaskEventType_Running_:
		return v.Running
	case *ECSTaskEventType_Failed_:
		return v.Failed
	case *ECSTaskEventType_Exited_:
		return v.Exited
	case *ECSTaskEventType_Stopped_:
		return v.Stopped
	default:
		return nil
	}
}
func (x *ECSTaskEventType_Pending) TypeKey() ECSTaskEventTypeKey {
	return ECSTaskEvent_Pending
}
func (x *ECSTaskEventType_Running) TypeKey() ECSTaskEventTypeKey {
	return ECSTaskEvent_Running
}
func (x *ECSTaskEventType_Failed) TypeKey() ECSTaskEventTypeKey {
	return ECSTaskEvent_Failed
}
func (x *ECSTaskEventType_Exited) TypeKey() ECSTaskEventTypeKey {
	return ECSTaskEvent_Exited
}
func (x *ECSTaskEventType_Stopped) TypeKey() ECSTaskEventTypeKey {
	return ECSTaskEvent_Stopped
}

type IsECSTaskEventType_Type = isECSTaskEventType_Type