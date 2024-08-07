// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/aws/deployer/v1/cluster.proto

package awsdeployer_pb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	_ "github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	_ "github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	psm_pb "github.com/pentops/j5/gen/psm/state/v1/psm_pb"
	environment_pb "github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ClusterStatus int32

const (
	ClusterStatus_CLUSTER_STATUS_UNSPECIFIED ClusterStatus = 0
	ClusterStatus_CLUSTER_STATUS_ACTIVE      ClusterStatus = 1
)

// Enum value maps for ClusterStatus.
var (
	ClusterStatus_name = map[int32]string{
		0: "CLUSTER_STATUS_UNSPECIFIED",
		1: "CLUSTER_STATUS_ACTIVE",
	}
	ClusterStatus_value = map[string]int32{
		"CLUSTER_STATUS_UNSPECIFIED": 0,
		"CLUSTER_STATUS_ACTIVE":      1,
	}
)

func (x ClusterStatus) Enum() *ClusterStatus {
	p := new(ClusterStatus)
	*p = x
	return p
}

func (x ClusterStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ClusterStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_o5_aws_deployer_v1_cluster_proto_enumTypes[0].Descriptor()
}

func (ClusterStatus) Type() protoreflect.EnumType {
	return &file_o5_aws_deployer_v1_cluster_proto_enumTypes[0]
}

func (x ClusterStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ClusterStatus.Descriptor instead.
func (ClusterStatus) EnumDescriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{0}
}

type ClusterKeys struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClusterId string `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty"`
}

func (x *ClusterKeys) Reset() {
	*x = ClusterKeys{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClusterKeys) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterKeys) ProtoMessage() {}

func (x *ClusterKeys) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterKeys.ProtoReflect.Descriptor instead.
func (*ClusterKeys) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{0}
}

func (x *ClusterKeys) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

type ClusterState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *psm_pb.StateMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Keys     *ClusterKeys          `protobuf:"bytes,2,opt,name=keys,proto3" json:"keys,omitempty"`
	Status   ClusterStatus         `protobuf:"varint,3,opt,name=status,proto3,enum=o5.aws.deployer.v1.ClusterStatus" json:"status,omitempty"`
	Data     *ClusterStateData     `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ClusterState) Reset() {
	*x = ClusterState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClusterState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterState) ProtoMessage() {}

func (x *ClusterState) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterState.ProtoReflect.Descriptor instead.
func (*ClusterState) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{1}
}

func (x *ClusterState) GetMetadata() *psm_pb.StateMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *ClusterState) GetKeys() *ClusterKeys {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *ClusterState) GetStatus() ClusterStatus {
	if x != nil {
		return x.Status
	}
	return ClusterStatus_CLUSTER_STATUS_UNSPECIFIED
}

func (x *ClusterState) GetData() *ClusterStateData {
	if x != nil {
		return x.Data
	}
	return nil
}

type ClusterStateData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config *environment_pb.Cluster `protobuf:"bytes,4,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *ClusterStateData) Reset() {
	*x = ClusterStateData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClusterStateData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterStateData) ProtoMessage() {}

func (x *ClusterStateData) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterStateData.ProtoReflect.Descriptor instead.
func (*ClusterStateData) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{2}
}

func (x *ClusterStateData) GetConfig() *environment_pb.Cluster {
	if x != nil {
		return x.Config
	}
	return nil
}

type ClusterEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *psm_pb.EventMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Keys     *ClusterKeys          `protobuf:"bytes,2,opt,name=keys,proto3" json:"keys,omitempty"`
	Event    *ClusterEventType     `protobuf:"bytes,3,opt,name=event,proto3" json:"event,omitempty"`
}

func (x *ClusterEvent) Reset() {
	*x = ClusterEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClusterEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterEvent) ProtoMessage() {}

func (x *ClusterEvent) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterEvent.ProtoReflect.Descriptor instead.
func (*ClusterEvent) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{3}
}

func (x *ClusterEvent) GetMetadata() *psm_pb.EventMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *ClusterEvent) GetKeys() *ClusterKeys {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *ClusterEvent) GetEvent() *ClusterEventType {
	if x != nil {
		return x.Event
	}
	return nil
}

type ClusterEventType struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//
	//	*ClusterEventType_Configured_
	Type isClusterEventType_Type `protobuf_oneof:"type"`
}

func (x *ClusterEventType) Reset() {
	*x = ClusterEventType{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClusterEventType) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterEventType) ProtoMessage() {}

func (x *ClusterEventType) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterEventType.ProtoReflect.Descriptor instead.
func (*ClusterEventType) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{4}
}

func (m *ClusterEventType) GetType() isClusterEventType_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *ClusterEventType) GetConfigured() *ClusterEventType_Configured {
	if x, ok := x.GetType().(*ClusterEventType_Configured_); ok {
		return x.Configured
	}
	return nil
}

type isClusterEventType_Type interface {
	isClusterEventType_Type()
}

type ClusterEventType_Configured_ struct {
	Configured *ClusterEventType_Configured `protobuf:"bytes,1,opt,name=configured,proto3,oneof"`
}

func (*ClusterEventType_Configured_) isClusterEventType_Type() {}

type ClusterEventType_Configured struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config *environment_pb.Cluster `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *ClusterEventType_Configured) Reset() {
	*x = ClusterEventType_Configured{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ClusterEventType_Configured) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterEventType_Configured) ProtoMessage() {}

func (x *ClusterEventType_Configured) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_cluster_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterEventType_Configured.ProtoReflect.Descriptor instead.
func (*ClusterEventType_Configured) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP(), []int{4, 0}
}

func (x *ClusterEventType_Configured) GetConfig() *environment_pb.Cluster {
	if x != nil {
		return x.Config
	}
	return nil
}

var File_o5_aws_deployer_v1_cluster_proto protoreflect.FileDescriptor

var file_o5_aws_deployer_v1_cluster_proto_rawDesc = []byte{
	0x0a, 0x20, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x12, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69,
	0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6a, 0x35, 0x2f, 0x65, 0x78, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x61,
	0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1c, 0x6a, 0x35, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e,
	0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f,
	0x6f, 0x35, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2f, 0x76,
	0x31, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x1b, 0x70, 0x73, 0x6d, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x4d, 0x0a, 0x0b,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x73, 0x12, 0x2e, 0x0a, 0x0a, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42,
	0x0f, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0xea, 0x85, 0x8f, 0x02, 0x02, 0x08, 0x01,
	0x52, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64, 0x3a, 0x0e, 0xea, 0x85, 0x8f,
	0x02, 0x09, 0x0a, 0x07, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x22, 0xb0, 0x02, 0x0a, 0x0c,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3f, 0x0a, 0x08,
	0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b,
	0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x42, 0x06, 0xba, 0x48, 0x03,
	0xc8, 0x01, 0x01, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x44, 0x0a,
	0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6f, 0x35,
	0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x73, 0x42, 0x0f, 0xba, 0x48,
	0x03, 0xc8, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x0a, 0x02, 0x08, 0x01, 0x52, 0x04, 0x6b,
	0x65, 0x79, 0x73, 0x12, 0x5f, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x21, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x42, 0x24, 0x8a, 0xf7, 0x98, 0xc6, 0x02, 0x1e, 0xa2, 0x01,
	0x1b, 0x52, 0x19, 0x08, 0x01, 0x12, 0x15, 0x43, 0x4c, 0x55, 0x53, 0x54, 0x45, 0x52, 0x5f, 0x53,
	0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x56, 0x45, 0x52, 0x06, 0x73, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x38, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x24, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c,
	0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53,
	0x74, 0x61, 0x74, 0x65, 0x44, 0x61, 0x74, 0x61, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x46,
	0x0a, 0x10, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x44, 0x61,
	0x74, 0x61, 0x12, 0x32, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x06,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0xd9, 0x01, 0x0a, 0x0c, 0x43, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x3f, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64,
	0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e, 0x70, 0x73, 0x6d, 0x2e,
	0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x08,
	0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x44, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1f, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x4b, 0x65, 0x79, 0x73, 0x42, 0x0f, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0xc2,
	0xff, 0x8e, 0x02, 0x04, 0x0a, 0x02, 0x08, 0x01, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x12, 0x42,
	0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e,
	0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54,
	0x79, 0x70, 0x65, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x05, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x22, 0xbb, 0x01, 0x0a, 0x10, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76,
	0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x51, 0x0a, 0x0a, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x75, 0x72, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2f, 0x2e, 0x6f, 0x35,
	0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70,
	0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x65, 0x64, 0x48, 0x00, 0x52, 0x0a,
	0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x65, 0x64, 0x1a, 0x40, 0x0a, 0x0a, 0x43, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x65, 0x64, 0x12, 0x32, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6f, 0x35, 0x2e, 0x65, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x42, 0x12, 0x0a, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x12, 0x0a, 0x92, 0xf7, 0x98, 0xc6, 0x02, 0x04, 0x52, 0x02, 0x08, 0x01,
	0x2a, 0x4a, 0x0a, 0x0d, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x1e, 0x0a, 0x1a, 0x43, 0x4c, 0x55, 0x53, 0x54, 0x45, 0x52, 0x5f, 0x53, 0x54, 0x41,
	0x54, 0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10,
	0x00, 0x12, 0x19, 0x0a, 0x15, 0x43, 0x4c, 0x55, 0x53, 0x54, 0x45, 0x52, 0x5f, 0x53, 0x54, 0x41,
	0x54, 0x55, 0x53, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x56, 0x45, 0x10, 0x01, 0x42, 0x48, 0x5a, 0x46,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f,
	0x70, 0x73, 0x2f, 0x6f, 0x35, 0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73,
	0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c,
	0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x65, 0x72, 0x5f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_o5_aws_deployer_v1_cluster_proto_rawDescOnce sync.Once
	file_o5_aws_deployer_v1_cluster_proto_rawDescData = file_o5_aws_deployer_v1_cluster_proto_rawDesc
)

func file_o5_aws_deployer_v1_cluster_proto_rawDescGZIP() []byte {
	file_o5_aws_deployer_v1_cluster_proto_rawDescOnce.Do(func() {
		file_o5_aws_deployer_v1_cluster_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_aws_deployer_v1_cluster_proto_rawDescData)
	})
	return file_o5_aws_deployer_v1_cluster_proto_rawDescData
}

var file_o5_aws_deployer_v1_cluster_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_o5_aws_deployer_v1_cluster_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_o5_aws_deployer_v1_cluster_proto_goTypes = []interface{}{
	(ClusterStatus)(0),                  // 0: o5.aws.deployer.v1.ClusterStatus
	(*ClusterKeys)(nil),                 // 1: o5.aws.deployer.v1.ClusterKeys
	(*ClusterState)(nil),                // 2: o5.aws.deployer.v1.ClusterState
	(*ClusterStateData)(nil),            // 3: o5.aws.deployer.v1.ClusterStateData
	(*ClusterEvent)(nil),                // 4: o5.aws.deployer.v1.ClusterEvent
	(*ClusterEventType)(nil),            // 5: o5.aws.deployer.v1.ClusterEventType
	(*ClusterEventType_Configured)(nil), // 6: o5.aws.deployer.v1.ClusterEventType.Configured
	(*psm_pb.StateMetadata)(nil),        // 7: psm.state.v1.StateMetadata
	(*environment_pb.Cluster)(nil),      // 8: o5.environment.v1.Cluster
	(*psm_pb.EventMetadata)(nil),        // 9: psm.state.v1.EventMetadata
}
var file_o5_aws_deployer_v1_cluster_proto_depIdxs = []int32{
	7,  // 0: o5.aws.deployer.v1.ClusterState.metadata:type_name -> psm.state.v1.StateMetadata
	1,  // 1: o5.aws.deployer.v1.ClusterState.keys:type_name -> o5.aws.deployer.v1.ClusterKeys
	0,  // 2: o5.aws.deployer.v1.ClusterState.status:type_name -> o5.aws.deployer.v1.ClusterStatus
	3,  // 3: o5.aws.deployer.v1.ClusterState.data:type_name -> o5.aws.deployer.v1.ClusterStateData
	8,  // 4: o5.aws.deployer.v1.ClusterStateData.config:type_name -> o5.environment.v1.Cluster
	9,  // 5: o5.aws.deployer.v1.ClusterEvent.metadata:type_name -> psm.state.v1.EventMetadata
	1,  // 6: o5.aws.deployer.v1.ClusterEvent.keys:type_name -> o5.aws.deployer.v1.ClusterKeys
	5,  // 7: o5.aws.deployer.v1.ClusterEvent.event:type_name -> o5.aws.deployer.v1.ClusterEventType
	6,  // 8: o5.aws.deployer.v1.ClusterEventType.configured:type_name -> o5.aws.deployer.v1.ClusterEventType.Configured
	8,  // 9: o5.aws.deployer.v1.ClusterEventType.Configured.config:type_name -> o5.environment.v1.Cluster
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_o5_aws_deployer_v1_cluster_proto_init() }
func file_o5_aws_deployer_v1_cluster_proto_init() {
	if File_o5_aws_deployer_v1_cluster_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_aws_deployer_v1_cluster_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClusterKeys); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_aws_deployer_v1_cluster_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClusterState); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_aws_deployer_v1_cluster_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClusterStateData); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_aws_deployer_v1_cluster_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClusterEvent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_aws_deployer_v1_cluster_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClusterEventType); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_aws_deployer_v1_cluster_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ClusterEventType_Configured); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_o5_aws_deployer_v1_cluster_proto_msgTypes[4].OneofWrappers = []interface{}{
		(*ClusterEventType_Configured_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_aws_deployer_v1_cluster_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_o5_aws_deployer_v1_cluster_proto_goTypes,
		DependencyIndexes: file_o5_aws_deployer_v1_cluster_proto_depIdxs,
		EnumInfos:         file_o5_aws_deployer_v1_cluster_proto_enumTypes,
		MessageInfos:      file_o5_aws_deployer_v1_cluster_proto_msgTypes,
	}.Build()
	File_o5_aws_deployer_v1_cluster_proto = out.File
	file_o5_aws_deployer_v1_cluster_proto_rawDesc = nil
	file_o5_aws_deployer_v1_cluster_proto_goTypes = nil
	file_o5_aws_deployer_v1_cluster_proto_depIdxs = nil
}
