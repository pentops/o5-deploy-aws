// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/aws/deployer/v1/environment.proto

package awsdeployer_pb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	_ "github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	_ "github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	psm_j5pb "github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
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

type EnvironmentStatus int32

const (
	EnvironmentStatus_ENVIRONMENT_STATUS_UNSPECIFIED EnvironmentStatus = 0
	EnvironmentStatus_ENVIRONMENT_STATUS_ACTIVE      EnvironmentStatus = 1
)

// Enum value maps for EnvironmentStatus.
var (
	EnvironmentStatus_name = map[int32]string{
		0: "ENVIRONMENT_STATUS_UNSPECIFIED",
		1: "ENVIRONMENT_STATUS_ACTIVE",
	}
	EnvironmentStatus_value = map[string]int32{
		"ENVIRONMENT_STATUS_UNSPECIFIED": 0,
		"ENVIRONMENT_STATUS_ACTIVE":      1,
	}
)

func (x EnvironmentStatus) Enum() *EnvironmentStatus {
	p := new(EnvironmentStatus)
	*p = x
	return p
}

func (x EnvironmentStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (EnvironmentStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_o5_aws_deployer_v1_environment_proto_enumTypes[0].Descriptor()
}

func (EnvironmentStatus) Type() protoreflect.EnumType {
	return &file_o5_aws_deployer_v1_environment_proto_enumTypes[0]
}

func (x EnvironmentStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use EnvironmentStatus.Descriptor instead.
func (EnvironmentStatus) EnumDescriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{0}
}

type EnvironmentKeys struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	EnvironmentId string `protobuf:"bytes,1,opt,name=environment_id,json=environmentId,proto3" json:"environment_id,omitempty"`
	ClusterId     string `protobuf:"bytes,2,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty"`
}

func (x *EnvironmentKeys) Reset() {
	*x = EnvironmentKeys{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentKeys) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentKeys) ProtoMessage() {}

func (x *EnvironmentKeys) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentKeys.ProtoReflect.Descriptor instead.
func (*EnvironmentKeys) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{0}
}

func (x *EnvironmentKeys) GetEnvironmentId() string {
	if x != nil {
		return x.EnvironmentId
	}
	return ""
}

func (x *EnvironmentKeys) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

type EnvironmentState struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *psm_j5pb.StateMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Keys     *EnvironmentKeys        `protobuf:"bytes,2,opt,name=keys,proto3" json:"keys,omitempty"`
	Status   EnvironmentStatus       `protobuf:"varint,3,opt,name=status,proto3,enum=o5.aws.deployer.v1.EnvironmentStatus" json:"status,omitempty"`
	Data     *EnvironmentStateData   `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *EnvironmentState) Reset() {
	*x = EnvironmentState{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentState) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentState) ProtoMessage() {}

func (x *EnvironmentState) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentState.ProtoReflect.Descriptor instead.
func (*EnvironmentState) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{1}
}

func (x *EnvironmentState) GetMetadata() *psm_j5pb.StateMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *EnvironmentState) GetKeys() *EnvironmentKeys {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *EnvironmentState) GetStatus() EnvironmentStatus {
	if x != nil {
		return x.Status
	}
	return EnvironmentStatus_ENVIRONMENT_STATUS_UNSPECIFIED
}

func (x *EnvironmentState) GetData() *EnvironmentStateData {
	if x != nil {
		return x.Data
	}
	return nil
}

type EnvironmentStateData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config *environment_pb.Environment `protobuf:"bytes,4,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *EnvironmentStateData) Reset() {
	*x = EnvironmentStateData{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentStateData) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentStateData) ProtoMessage() {}

func (x *EnvironmentStateData) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentStateData.ProtoReflect.Descriptor instead.
func (*EnvironmentStateData) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{2}
}

func (x *EnvironmentStateData) GetConfig() *environment_pb.Environment {
	if x != nil {
		return x.Config
	}
	return nil
}

type EnvironmentEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Metadata *psm_j5pb.EventMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	Keys     *EnvironmentKeys        `protobuf:"bytes,2,opt,name=keys,proto3" json:"keys,omitempty"`
	Event    *EnvironmentEventType   `protobuf:"bytes,3,opt,name=event,proto3" json:"event,omitempty"`
}

func (x *EnvironmentEvent) Reset() {
	*x = EnvironmentEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentEvent) ProtoMessage() {}

func (x *EnvironmentEvent) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentEvent.ProtoReflect.Descriptor instead.
func (*EnvironmentEvent) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{3}
}

func (x *EnvironmentEvent) GetMetadata() *psm_j5pb.EventMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *EnvironmentEvent) GetKeys() *EnvironmentKeys {
	if x != nil {
		return x.Keys
	}
	return nil
}

func (x *EnvironmentEvent) GetEvent() *EnvironmentEventType {
	if x != nil {
		return x.Event
	}
	return nil
}

type EnvironmentEventType struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//
	//	*EnvironmentEventType_Configured_
	Type isEnvironmentEventType_Type `protobuf_oneof:"type"`
}

func (x *EnvironmentEventType) Reset() {
	*x = EnvironmentEventType{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentEventType) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentEventType) ProtoMessage() {}

func (x *EnvironmentEventType) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentEventType.ProtoReflect.Descriptor instead.
func (*EnvironmentEventType) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{4}
}

func (m *EnvironmentEventType) GetType() isEnvironmentEventType_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *EnvironmentEventType) GetConfigured() *EnvironmentEventType_Configured {
	if x, ok := x.GetType().(*EnvironmentEventType_Configured_); ok {
		return x.Configured
	}
	return nil
}

type isEnvironmentEventType_Type interface {
	isEnvironmentEventType_Type()
}

type EnvironmentEventType_Configured_ struct {
	Configured *EnvironmentEventType_Configured `protobuf:"bytes,1,opt,name=configured,proto3,oneof"`
}

func (*EnvironmentEventType_Configured_) isEnvironmentEventType_Type() {}

type EnvironmentEventType_Configured struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config *environment_pb.Environment `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *EnvironmentEventType_Configured) Reset() {
	*x = EnvironmentEventType_Configured{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentEventType_Configured) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentEventType_Configured) ProtoMessage() {}

func (x *EnvironmentEventType_Configured) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_environment_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EnvironmentEventType_Configured.ProtoReflect.Descriptor instead.
func (*EnvironmentEventType_Configured) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_environment_proto_rawDescGZIP(), []int{4, 0}
}

func (x *EnvironmentEventType_Configured) GetConfig() *environment_pb.Environment {
	if x != nil {
		return x.Config
	}
	return nil
}

var File_o5_aws_deployer_v1_environment_proto protoreflect.FileDescriptor

var file_o5_aws_deployer_v1_environment_proto_rawDesc = []byte{
	0x0a, 0x24, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f,
	0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6a, 0x35, 0x2f, 0x65, 0x78, 0x74, 0x2f,
	0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x6a, 0x35, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31,
	0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x23, 0x6f, 0x35, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65,
	0x6e, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1a, 0x6a, 0x35, 0x2f, 0x73, 0x74, 0x61, 0x74,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x8e, 0x01, 0x0a, 0x0f, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x4b, 0x65, 0x79, 0x73, 0x12, 0x3e, 0x0a, 0x0e, 0x65, 0x6e, 0x76, 0x69, 0x72,
	0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42,
	0x17, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x03, 0xb2, 0x02,
	0x00, 0xea, 0x85, 0x8f, 0x02, 0x02, 0x08, 0x01, 0x52, 0x0d, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f,
	0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x27, 0x0a, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05,
	0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64,
	0x3a, 0x12, 0xea, 0x85, 0x8f, 0x02, 0x0d, 0x0a, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x22, 0xc3, 0x02, 0x0a, 0x10, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x12, 0x3e, 0x0a, 0x08, 0x6d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6a, 0x35,
	0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x74, 0x65, 0x4d,
	0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52,
	0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x48, 0x0a, 0x04, 0x6b, 0x65, 0x79,
	0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73,
	0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76,
	0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x4b, 0x65, 0x79, 0x73, 0x42, 0x0f, 0xba, 0x48,
	0x03, 0xc8, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x0a, 0x02, 0x08, 0x01, 0x52, 0x04, 0x6b,
	0x65, 0x79, 0x73, 0x12, 0x67, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x25, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x42, 0x28, 0x8a, 0xf7, 0x98, 0xc6,
	0x02, 0x22, 0xa2, 0x01, 0x1f, 0x52, 0x1d, 0x08, 0x01, 0x12, 0x19, 0x45, 0x4e, 0x56, 0x49, 0x52,
	0x4f, 0x4e, 0x4d, 0x45, 0x4e, 0x54, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x41, 0x43,
	0x54, 0x49, 0x56, 0x45, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x3c, 0x0a, 0x04,
	0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x6f, 0x35, 0x2e,
	0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65,
	0x44, 0x61, 0x74, 0x61, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x4e, 0x0a, 0x14, 0x45, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x44, 0x61,
	0x74, 0x61, 0x12, 0x36, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x04, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65,
	0x6e, 0x74, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x22, 0xe4, 0x01, 0x0a, 0x10, 0x45,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12,
	0x3e, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1a, 0x2e, 0x6a, 0x35, 0x2e, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x45, 0x76, 0x65, 0x6e, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x42, 0x06, 0xba,
	0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12,
	0x48, 0x0a, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e,
	0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x4b, 0x65,
	0x79, 0x73, 0x42, 0x0f, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x0a,
	0x02, 0x08, 0x01, 0x52, 0x04, 0x6b, 0x65, 0x79, 0x73, 0x12, 0x46, 0x0a, 0x05, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77,
	0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e,
	0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79,
	0x70, 0x65, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x22, 0xc7, 0x01, 0x0a, 0x14, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x55, 0x0a, 0x0a, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x33,
	0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x54, 0x79, 0x70, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75,
	0x72, 0x65, 0x64, 0x48, 0x00, 0x52, 0x0a, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x65,
	0x64, 0x1a, 0x44, 0x0a, 0x0a, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x75, 0x72, 0x65, 0x64, 0x12,
	0x36, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1e, 0x2e, 0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74,
	0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x52,
	0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x42, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12,
	0x0a, 0x92, 0xf7, 0x98, 0xc6, 0x02, 0x04, 0x52, 0x02, 0x08, 0x01, 0x2a, 0x56, 0x0a, 0x11, 0x45,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x12, 0x22, 0x0a, 0x1e, 0x45, 0x4e, 0x56, 0x49, 0x52, 0x4f, 0x4e, 0x4d, 0x45, 0x4e, 0x54, 0x5f,
	0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49,
	0x45, 0x44, 0x10, 0x00, 0x12, 0x1d, 0x0a, 0x19, 0x45, 0x4e, 0x56, 0x49, 0x52, 0x4f, 0x4e, 0x4d,
	0x45, 0x4e, 0x54, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x56,
	0x45, 0x10, 0x01, 0x42, 0x48, 0x5a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f,
	0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x6f, 0x35, 0x2d, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x61,
	0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x61,
	0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_o5_aws_deployer_v1_environment_proto_rawDescOnce sync.Once
	file_o5_aws_deployer_v1_environment_proto_rawDescData = file_o5_aws_deployer_v1_environment_proto_rawDesc
)

func file_o5_aws_deployer_v1_environment_proto_rawDescGZIP() []byte {
	file_o5_aws_deployer_v1_environment_proto_rawDescOnce.Do(func() {
		file_o5_aws_deployer_v1_environment_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_aws_deployer_v1_environment_proto_rawDescData)
	})
	return file_o5_aws_deployer_v1_environment_proto_rawDescData
}

var file_o5_aws_deployer_v1_environment_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_o5_aws_deployer_v1_environment_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_o5_aws_deployer_v1_environment_proto_goTypes = []interface{}{
	(EnvironmentStatus)(0),                  // 0: o5.aws.deployer.v1.EnvironmentStatus
	(*EnvironmentKeys)(nil),                 // 1: o5.aws.deployer.v1.EnvironmentKeys
	(*EnvironmentState)(nil),                // 2: o5.aws.deployer.v1.EnvironmentState
	(*EnvironmentStateData)(nil),            // 3: o5.aws.deployer.v1.EnvironmentStateData
	(*EnvironmentEvent)(nil),                // 4: o5.aws.deployer.v1.EnvironmentEvent
	(*EnvironmentEventType)(nil),            // 5: o5.aws.deployer.v1.EnvironmentEventType
	(*EnvironmentEventType_Configured)(nil), // 6: o5.aws.deployer.v1.EnvironmentEventType.Configured
	(*psm_j5pb.StateMetadata)(nil),          // 7: j5.state.v1.StateMetadata
	(*environment_pb.Environment)(nil),      // 8: o5.environment.v1.Environment
	(*psm_j5pb.EventMetadata)(nil),          // 9: j5.state.v1.EventMetadata
}
var file_o5_aws_deployer_v1_environment_proto_depIdxs = []int32{
	7,  // 0: o5.aws.deployer.v1.EnvironmentState.metadata:type_name -> j5.state.v1.StateMetadata
	1,  // 1: o5.aws.deployer.v1.EnvironmentState.keys:type_name -> o5.aws.deployer.v1.EnvironmentKeys
	0,  // 2: o5.aws.deployer.v1.EnvironmentState.status:type_name -> o5.aws.deployer.v1.EnvironmentStatus
	3,  // 3: o5.aws.deployer.v1.EnvironmentState.data:type_name -> o5.aws.deployer.v1.EnvironmentStateData
	8,  // 4: o5.aws.deployer.v1.EnvironmentStateData.config:type_name -> o5.environment.v1.Environment
	9,  // 5: o5.aws.deployer.v1.EnvironmentEvent.metadata:type_name -> j5.state.v1.EventMetadata
	1,  // 6: o5.aws.deployer.v1.EnvironmentEvent.keys:type_name -> o5.aws.deployer.v1.EnvironmentKeys
	5,  // 7: o5.aws.deployer.v1.EnvironmentEvent.event:type_name -> o5.aws.deployer.v1.EnvironmentEventType
	6,  // 8: o5.aws.deployer.v1.EnvironmentEventType.configured:type_name -> o5.aws.deployer.v1.EnvironmentEventType.Configured
	8,  // 9: o5.aws.deployer.v1.EnvironmentEventType.Configured.config:type_name -> o5.environment.v1.Environment
	10, // [10:10] is the sub-list for method output_type
	10, // [10:10] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_o5_aws_deployer_v1_environment_proto_init() }
func file_o5_aws_deployer_v1_environment_proto_init() {
	if File_o5_aws_deployer_v1_environment_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_aws_deployer_v1_environment_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentKeys); i {
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
		file_o5_aws_deployer_v1_environment_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentState); i {
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
		file_o5_aws_deployer_v1_environment_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentStateData); i {
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
		file_o5_aws_deployer_v1_environment_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentEvent); i {
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
		file_o5_aws_deployer_v1_environment_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentEventType); i {
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
		file_o5_aws_deployer_v1_environment_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EnvironmentEventType_Configured); i {
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
	file_o5_aws_deployer_v1_environment_proto_msgTypes[4].OneofWrappers = []interface{}{
		(*EnvironmentEventType_Configured_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_aws_deployer_v1_environment_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_o5_aws_deployer_v1_environment_proto_goTypes,
		DependencyIndexes: file_o5_aws_deployer_v1_environment_proto_depIdxs,
		EnumInfos:         file_o5_aws_deployer_v1_environment_proto_enumTypes,
		MessageInfos:      file_o5_aws_deployer_v1_environment_proto_msgTypes,
	}.Build()
	File_o5_aws_deployer_v1_environment_proto = out.File
	file_o5_aws_deployer_v1_environment_proto_rawDesc = nil
	file_o5_aws_deployer_v1_environment_proto_goTypes = nil
	file_o5_aws_deployer_v1_environment_proto_depIdxs = nil
}
