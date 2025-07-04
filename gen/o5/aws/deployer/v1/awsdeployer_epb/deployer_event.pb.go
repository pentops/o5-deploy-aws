// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/aws/deployer/v1/events/deployer_event.proto

package awsdeployer_epb

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	awsdeployer_pb "github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type DeploymentEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Event  *awsdeployer_pb.DeploymentEvent     `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
	Status awsdeployer_pb.DeploymentStatus     `protobuf:"varint,2,opt,name=status,proto3,enum=o5.aws.deployer.v1.DeploymentStatus" json:"status,omitempty"`
	State  *awsdeployer_pb.DeploymentStateData `protobuf:"bytes,3,opt,name=state,proto3" json:"state,omitempty"`
}

func (x *DeploymentEvent) Reset() {
	*x = DeploymentEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeploymentEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeploymentEvent) ProtoMessage() {}

func (x *DeploymentEvent) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeploymentEvent.ProtoReflect.Descriptor instead.
func (*DeploymentEvent) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescGZIP(), []int{0}
}

func (x *DeploymentEvent) GetEvent() *awsdeployer_pb.DeploymentEvent {
	if x != nil {
		return x.Event
	}
	return nil
}

func (x *DeploymentEvent) GetStatus() awsdeployer_pb.DeploymentStatus {
	if x != nil {
		return x.Status
	}
	return awsdeployer_pb.DeploymentStatus(0)
}

func (x *DeploymentEvent) GetState() *awsdeployer_pb.DeploymentStateData {
	if x != nil {
		return x.State
	}
	return nil
}

type StackEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Event  *awsdeployer_pb.StackEvent     `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
	Status awsdeployer_pb.StackStatus     `protobuf:"varint,2,opt,name=status,proto3,enum=o5.aws.deployer.v1.StackStatus" json:"status,omitempty"`
	State  *awsdeployer_pb.StackStateData `protobuf:"bytes,3,opt,name=state,proto3" json:"state,omitempty"`
}

func (x *StackEvent) Reset() {
	*x = StackEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StackEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StackEvent) ProtoMessage() {}

func (x *StackEvent) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StackEvent.ProtoReflect.Descriptor instead.
func (*StackEvent) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescGZIP(), []int{1}
}

func (x *StackEvent) GetEvent() *awsdeployer_pb.StackEvent {
	if x != nil {
		return x.Event
	}
	return nil
}

func (x *StackEvent) GetStatus() awsdeployer_pb.StackStatus {
	if x != nil {
		return x.Status
	}
	return awsdeployer_pb.StackStatus(0)
}

func (x *StackEvent) GetState() *awsdeployer_pb.StackStateData {
	if x != nil {
		return x.State
	}
	return nil
}

type EnvironmentEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Event  *awsdeployer_pb.EnvironmentEvent     `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
	Status awsdeployer_pb.EnvironmentStatus     `protobuf:"varint,2,opt,name=status,proto3,enum=o5.aws.deployer.v1.EnvironmentStatus" json:"status,omitempty"`
	State  *awsdeployer_pb.EnvironmentStateData `protobuf:"bytes,3,opt,name=state,proto3" json:"state,omitempty"`
}

func (x *EnvironmentEvent) Reset() {
	*x = EnvironmentEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EnvironmentEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EnvironmentEvent) ProtoMessage() {}

func (x *EnvironmentEvent) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[2]
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
	return file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescGZIP(), []int{2}
}

func (x *EnvironmentEvent) GetEvent() *awsdeployer_pb.EnvironmentEvent {
	if x != nil {
		return x.Event
	}
	return nil
}

func (x *EnvironmentEvent) GetStatus() awsdeployer_pb.EnvironmentStatus {
	if x != nil {
		return x.Status
	}
	return awsdeployer_pb.EnvironmentStatus(0)
}

func (x *EnvironmentEvent) GetState() *awsdeployer_pb.EnvironmentStateData {
	if x != nil {
		return x.State
	}
	return nil
}

var File_o5_aws_deployer_v1_events_deployer_event_proto protoreflect.FileDescriptor

var file_o5_aws_deployer_v1_events_deployer_event_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c,
	0x6f, 0x79, 0x65, 0x72, 0x5f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x19, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x1a, 0x1b, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70,
	0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x23, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73,
	0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x24, 0x6f,
	0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76,
	0x31, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c,
	0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x6a, 0x35, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x69, 0x6e,
	0x67, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc9, 0x01, 0x0a, 0x0f, 0x44, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x39, 0x0a, 0x05, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x23, 0x2e, 0x6f, 0x35, 0x2e, 0x61,
	0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x44,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x05,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x3c, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x24, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61,
	0x74, 0x75, 0x73, 0x12, 0x3d, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x27, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c,
	0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65,
	0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x44, 0x61, 0x74, 0x61, 0x52, 0x05, 0x73, 0x74, 0x61,
	0x74, 0x65, 0x22, 0xb5, 0x01, 0x0a, 0x0a, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x12, 0x34, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1e, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x37, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1f, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73,
	0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61,
	0x63, 0x6b, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x12, 0x38, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x22, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x53, 0x74, 0x61, 0x74, 0x65, 0x44,
	0x61, 0x74, 0x61, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x22, 0xcd, 0x01, 0x0a, 0x10, 0x45,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x12,
	0x3a, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24,
	0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x45,
	0x76, 0x65, 0x6e, 0x74, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x12, 0x3d, 0x0a, 0x06, 0x73,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x25, 0x2e, 0x6f, 0x35,
	0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x3e, 0x0a, 0x05, 0x73, 0x74,
	0x61, 0x74, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x28, 0x2e, 0x6f, 0x35, 0x2e, 0x61,
	0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x45,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x53, 0x74, 0x61, 0x74, 0x65, 0x44,
	0x61, 0x74, 0x61, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x32, 0xa2, 0x02, 0x0a, 0x0e, 0x44,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x52, 0x0a,
	0x0a, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x2a, 0x2e, 0x6f, 0x35,
	0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x2e, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65,
	0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22,
	0x00, 0x12, 0x48, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x12, 0x25, 0x2e, 0x6f, 0x35, 0x2e,
	0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x2e, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x54, 0x0a, 0x0b, 0x45,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x2b, 0x2e, 0x6f, 0x35, 0x2e,
	0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65,
	0x6e, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22,
	0x00, 0x1a, 0x1c, 0xd2, 0xa2, 0xf5, 0xe4, 0x02, 0x16, 0x12, 0x14, 0x0a, 0x12, 0x6f, 0x35, 0x2d,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2d, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x42,
	0x49, 0x5a, 0x47, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65,
	0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x6f, 0x35, 0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d,
	0x61, 0x77, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x77, 0x73, 0x64, 0x65,
	0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f, 0x65, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescOnce sync.Once
	file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescData = file_o5_aws_deployer_v1_events_deployer_event_proto_rawDesc
)

func file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescGZIP() []byte {
	file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescOnce.Do(func() {
		file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescData)
	})
	return file_o5_aws_deployer_v1_events_deployer_event_proto_rawDescData
}

var file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_o5_aws_deployer_v1_events_deployer_event_proto_goTypes = []interface{}{
	(*DeploymentEvent)(nil),                     // 0: o5.aws.deployer.v1.events.DeploymentEvent
	(*StackEvent)(nil),                          // 1: o5.aws.deployer.v1.events.StackEvent
	(*EnvironmentEvent)(nil),                    // 2: o5.aws.deployer.v1.events.EnvironmentEvent
	(*awsdeployer_pb.DeploymentEvent)(nil),      // 3: o5.aws.deployer.v1.DeploymentEvent
	(awsdeployer_pb.DeploymentStatus)(0),        // 4: o5.aws.deployer.v1.DeploymentStatus
	(*awsdeployer_pb.DeploymentStateData)(nil),  // 5: o5.aws.deployer.v1.DeploymentStateData
	(*awsdeployer_pb.StackEvent)(nil),           // 6: o5.aws.deployer.v1.StackEvent
	(awsdeployer_pb.StackStatus)(0),             // 7: o5.aws.deployer.v1.StackStatus
	(*awsdeployer_pb.StackStateData)(nil),       // 8: o5.aws.deployer.v1.StackStateData
	(*awsdeployer_pb.EnvironmentEvent)(nil),     // 9: o5.aws.deployer.v1.EnvironmentEvent
	(awsdeployer_pb.EnvironmentStatus)(0),       // 10: o5.aws.deployer.v1.EnvironmentStatus
	(*awsdeployer_pb.EnvironmentStateData)(nil), // 11: o5.aws.deployer.v1.EnvironmentStateData
	(*emptypb.Empty)(nil),                       // 12: google.protobuf.Empty
}
var file_o5_aws_deployer_v1_events_deployer_event_proto_depIdxs = []int32{
	3,  // 0: o5.aws.deployer.v1.events.DeploymentEvent.event:type_name -> o5.aws.deployer.v1.DeploymentEvent
	4,  // 1: o5.aws.deployer.v1.events.DeploymentEvent.status:type_name -> o5.aws.deployer.v1.DeploymentStatus
	5,  // 2: o5.aws.deployer.v1.events.DeploymentEvent.state:type_name -> o5.aws.deployer.v1.DeploymentStateData
	6,  // 3: o5.aws.deployer.v1.events.StackEvent.event:type_name -> o5.aws.deployer.v1.StackEvent
	7,  // 4: o5.aws.deployer.v1.events.StackEvent.status:type_name -> o5.aws.deployer.v1.StackStatus
	8,  // 5: o5.aws.deployer.v1.events.StackEvent.state:type_name -> o5.aws.deployer.v1.StackStateData
	9,  // 6: o5.aws.deployer.v1.events.EnvironmentEvent.event:type_name -> o5.aws.deployer.v1.EnvironmentEvent
	10, // 7: o5.aws.deployer.v1.events.EnvironmentEvent.status:type_name -> o5.aws.deployer.v1.EnvironmentStatus
	11, // 8: o5.aws.deployer.v1.events.EnvironmentEvent.state:type_name -> o5.aws.deployer.v1.EnvironmentStateData
	0,  // 9: o5.aws.deployer.v1.events.DeployerEvents.Deployment:input_type -> o5.aws.deployer.v1.events.DeploymentEvent
	1,  // 10: o5.aws.deployer.v1.events.DeployerEvents.Stack:input_type -> o5.aws.deployer.v1.events.StackEvent
	2,  // 11: o5.aws.deployer.v1.events.DeployerEvents.Environment:input_type -> o5.aws.deployer.v1.events.EnvironmentEvent
	12, // 12: o5.aws.deployer.v1.events.DeployerEvents.Deployment:output_type -> google.protobuf.Empty
	12, // 13: o5.aws.deployer.v1.events.DeployerEvents.Stack:output_type -> google.protobuf.Empty
	12, // 14: o5.aws.deployer.v1.events.DeployerEvents.Environment:output_type -> google.protobuf.Empty
	12, // [12:15] is the sub-list for method output_type
	9,  // [9:12] is the sub-list for method input_type
	9,  // [9:9] is the sub-list for extension type_name
	9,  // [9:9] is the sub-list for extension extendee
	0,  // [0:9] is the sub-list for field type_name
}

func init() { file_o5_aws_deployer_v1_events_deployer_event_proto_init() }
func file_o5_aws_deployer_v1_events_deployer_event_proto_init() {
	if File_o5_aws_deployer_v1_events_deployer_event_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeploymentEvent); i {
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
		file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StackEvent); i {
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
		file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_aws_deployer_v1_events_deployer_event_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_o5_aws_deployer_v1_events_deployer_event_proto_goTypes,
		DependencyIndexes: file_o5_aws_deployer_v1_events_deployer_event_proto_depIdxs,
		MessageInfos:      file_o5_aws_deployer_v1_events_deployer_event_proto_msgTypes,
	}.Build()
	File_o5_aws_deployer_v1_events_deployer_event_proto = out.File
	file_o5_aws_deployer_v1_events_deployer_event_proto_rawDesc = nil
	file_o5_aws_deployer_v1_events_deployer_event_proto_goTypes = nil
	file_o5_aws_deployer_v1_events_deployer_event_proto_depIdxs = nil
}
