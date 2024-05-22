// Code generated by protoc-gen-go-sugar. DO NOT EDIT.

package deployer_pb

import (
	driver "database/sql/driver"
	fmt "fmt"
)

// ClusterEventType is a oneof wrapper
type ClusterEventTypeKey string

const (
	ClusterEvent_Configured ClusterEventTypeKey = "configured"
)

func (x *ClusterEventType) TypeKey() (ClusterEventTypeKey, bool) {
	switch x.Type.(type) {
	case *ClusterEventType_Configured_:
		return ClusterEvent_Configured, true
	default:
		return "", false
	}
}

type IsClusterEventTypeWrappedType interface {
	TypeKey() ClusterEventTypeKey
}

func (x *ClusterEventType) Set(val IsClusterEventTypeWrappedType) {
	switch v := val.(type) {
	case *ClusterEventType_Configured:
		x.Type = &ClusterEventType_Configured_{Configured: v}
	}
}
func (x *ClusterEventType) Get() IsClusterEventTypeWrappedType {
	switch v := x.Type.(type) {
	case *ClusterEventType_Configured_:
		return v.Configured
	default:
		return nil
	}
}
func (x *ClusterEventType_Configured) TypeKey() ClusterEventTypeKey {
	return ClusterEvent_Configured
}

type IsClusterEventType_Type = isClusterEventType_Type

// ClusterStatus
const (
	ClusterStatus_UNSPECIFIED ClusterStatus = 0
	ClusterStatus_ACTIVE      ClusterStatus = 1
)

var (
	ClusterStatus_name_short = map[int32]string{
		0: "UNSPECIFIED",
		1: "ACTIVE",
	}
	ClusterStatus_value_short = map[string]int32{
		"UNSPECIFIED": 0,
		"ACTIVE":      1,
	}
	ClusterStatus_value_either = map[string]int32{
		"UNSPECIFIED":                0,
		"CLUSTER_STATUS_UNSPECIFIED": 0,
		"ACTIVE":                     1,
		"CLUSTER_STATUS_ACTIVE":      1,
	}
)

// ShortString returns the un-prefixed string representation of the enum value
func (x ClusterStatus) ShortString() string {
	return ClusterStatus_name_short[int32(x)]
}
func (x ClusterStatus) Value() (driver.Value, error) {
	return []uint8(x.ShortString()), nil
}
func (x *ClusterStatus) Scan(value interface{}) error {
	var strVal string
	switch vt := value.(type) {
	case []uint8:
		strVal = string(vt)
	case string:
		strVal = vt
	default:
		return fmt.Errorf("invalid type %T", value)
	}
	val := ClusterStatus_value_either[strVal]
	*x = ClusterStatus(val)
	return nil
}
