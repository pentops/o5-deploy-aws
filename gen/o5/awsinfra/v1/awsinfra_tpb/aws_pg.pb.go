// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: o5/awsinfra/v1/topic/aws_pg.proto

package awsinfra_tpb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	awsdeployer_pb "github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	messaging_pb "github.com/pentops/o5-go/messaging/v1/messaging_pb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PostgresStatus int32

const (
	PostgresStatus_POSTGRES_STATUS_UNSPECIFIED PostgresStatus = 0
	PostgresStatus_POSTGRES_STATUS_STARTED     PostgresStatus = 1
	PostgresStatus_POSTGRES_STATUS_DONE        PostgresStatus = 2
	PostgresStatus_POSTGRES_STATUS_ERROR       PostgresStatus = 3
)

// Enum value maps for PostgresStatus.
var (
	PostgresStatus_name = map[int32]string{
		0: "POSTGRES_STATUS_UNSPECIFIED",
		1: "POSTGRES_STATUS_STARTED",
		2: "POSTGRES_STATUS_DONE",
		3: "POSTGRES_STATUS_ERROR",
	}
	PostgresStatus_value = map[string]int32{
		"POSTGRES_STATUS_UNSPECIFIED": 0,
		"POSTGRES_STATUS_STARTED":     1,
		"POSTGRES_STATUS_DONE":        2,
		"POSTGRES_STATUS_ERROR":       3,
	}
)

func (x PostgresStatus) Enum() *PostgresStatus {
	p := new(PostgresStatus)
	*p = x
	return p
}

func (x PostgresStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (PostgresStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_o5_awsinfra_v1_topic_aws_pg_proto_enumTypes[0].Descriptor()
}

func (PostgresStatus) Type() protoreflect.EnumType {
	return &file_o5_awsinfra_v1_topic_aws_pg_proto_enumTypes[0]
}

func (x PostgresStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use PostgresStatus.Descriptor instead.
func (PostgresStatus) EnumDescriptor() ([]byte, []int) {
	return file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescGZIP(), []int{0}
}

type UpsertPostgresDatabaseMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request     *messaging_pb.RequestMetadata        `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	MigrationId string                               `protobuf:"bytes,2,opt,name=migration_id,json=migrationId,proto3" json:"migration_id,omitempty"`
	Spec        *awsdeployer_pb.PostgresCreationSpec `protobuf:"bytes,3,opt,name=spec,proto3" json:"spec,omitempty"`
}

func (x *UpsertPostgresDatabaseMessage) Reset() {
	*x = UpsertPostgresDatabaseMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *UpsertPostgresDatabaseMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*UpsertPostgresDatabaseMessage) ProtoMessage() {}

func (x *UpsertPostgresDatabaseMessage) ProtoReflect() protoreflect.Message {
	mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use UpsertPostgresDatabaseMessage.ProtoReflect.Descriptor instead.
func (*UpsertPostgresDatabaseMessage) Descriptor() ([]byte, []int) {
	return file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescGZIP(), []int{0}
}

func (x *UpsertPostgresDatabaseMessage) GetRequest() *messaging_pb.RequestMetadata {
	if x != nil {
		return x.Request
	}
	return nil
}

func (x *UpsertPostgresDatabaseMessage) GetMigrationId() string {
	if x != nil {
		return x.MigrationId
	}
	return ""
}

func (x *UpsertPostgresDatabaseMessage) GetSpec() *awsdeployer_pb.PostgresCreationSpec {
	if x != nil {
		return x.Spec
	}
	return nil
}

type CleanupPostgresDatabaseMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request     *messaging_pb.RequestMetadata       `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	MigrationId string                              `protobuf:"bytes,2,opt,name=migration_id,json=migrationId,proto3" json:"migration_id,omitempty"`
	Spec        *awsdeployer_pb.PostgresCleanupSpec `protobuf:"bytes,3,opt,name=spec,proto3" json:"spec,omitempty"`
}

func (x *CleanupPostgresDatabaseMessage) Reset() {
	*x = CleanupPostgresDatabaseMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CleanupPostgresDatabaseMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CleanupPostgresDatabaseMessage) ProtoMessage() {}

func (x *CleanupPostgresDatabaseMessage) ProtoReflect() protoreflect.Message {
	mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CleanupPostgresDatabaseMessage.ProtoReflect.Descriptor instead.
func (*CleanupPostgresDatabaseMessage) Descriptor() ([]byte, []int) {
	return file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescGZIP(), []int{1}
}

func (x *CleanupPostgresDatabaseMessage) GetRequest() *messaging_pb.RequestMetadata {
	if x != nil {
		return x.Request
	}
	return nil
}

func (x *CleanupPostgresDatabaseMessage) GetMigrationId() string {
	if x != nil {
		return x.MigrationId
	}
	return ""
}

func (x *CleanupPostgresDatabaseMessage) GetSpec() *awsdeployer_pb.PostgresCleanupSpec {
	if x != nil {
		return x.Spec
	}
	return nil
}

type MigratePostgresDatabaseMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request     *messaging_pb.RequestMetadata         `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	MigrationId string                                `protobuf:"bytes,2,opt,name=migration_id,json=migrationId,proto3" json:"migration_id,omitempty"`
	Spec        *awsdeployer_pb.PostgresMigrationSpec `protobuf:"bytes,3,opt,name=spec,proto3" json:"spec,omitempty"`
}

func (x *MigratePostgresDatabaseMessage) Reset() {
	*x = MigratePostgresDatabaseMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MigratePostgresDatabaseMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MigratePostgresDatabaseMessage) ProtoMessage() {}

func (x *MigratePostgresDatabaseMessage) ProtoReflect() protoreflect.Message {
	mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MigratePostgresDatabaseMessage.ProtoReflect.Descriptor instead.
func (*MigratePostgresDatabaseMessage) Descriptor() ([]byte, []int) {
	return file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescGZIP(), []int{2}
}

func (x *MigratePostgresDatabaseMessage) GetRequest() *messaging_pb.RequestMetadata {
	if x != nil {
		return x.Request
	}
	return nil
}

func (x *MigratePostgresDatabaseMessage) GetMigrationId() string {
	if x != nil {
		return x.MigrationId
	}
	return ""
}

func (x *MigratePostgresDatabaseMessage) GetSpec() *awsdeployer_pb.PostgresMigrationSpec {
	if x != nil {
		return x.Spec
	}
	return nil
}

type PostgresDatabaseStatusMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Request     *messaging_pb.RequestMetadata `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	MigrationId string                        `protobuf:"bytes,2,opt,name=migration_id,json=migrationId,proto3" json:"migration_id,omitempty"`
	EventId     string                        `protobuf:"bytes,3,opt,name=event_id,json=eventId,proto3" json:"event_id,omitempty"`
	Status      PostgresStatus                `protobuf:"varint,4,opt,name=status,proto3,enum=o5.awsinfra.v1.topic.PostgresStatus" json:"status,omitempty"`
	Error       *string                       `protobuf:"bytes,5,opt,name=error,proto3,oneof" json:"error,omitempty"`
}

func (x *PostgresDatabaseStatusMessage) Reset() {
	*x = PostgresDatabaseStatusMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostgresDatabaseStatusMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostgresDatabaseStatusMessage) ProtoMessage() {}

func (x *PostgresDatabaseStatusMessage) ProtoReflect() protoreflect.Message {
	mi := &file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostgresDatabaseStatusMessage.ProtoReflect.Descriptor instead.
func (*PostgresDatabaseStatusMessage) Descriptor() ([]byte, []int) {
	return file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescGZIP(), []int{3}
}

func (x *PostgresDatabaseStatusMessage) GetRequest() *messaging_pb.RequestMetadata {
	if x != nil {
		return x.Request
	}
	return nil
}

func (x *PostgresDatabaseStatusMessage) GetMigrationId() string {
	if x != nil {
		return x.MigrationId
	}
	return ""
}

func (x *PostgresDatabaseStatusMessage) GetEventId() string {
	if x != nil {
		return x.EventId
	}
	return ""
}

func (x *PostgresDatabaseStatusMessage) GetStatus() PostgresStatus {
	if x != nil {
		return x.Status
	}
	return PostgresStatus_POSTGRES_STATUS_UNSPECIFIED
}

func (x *PostgresDatabaseStatusMessage) GetError() string {
	if x != nil && x.Error != nil {
		return *x.Error
	}
	return ""
}

var File_o5_awsinfra_v1_topic_aws_pg_proto protoreflect.FileDescriptor

var file_o5_awsinfra_v1_topic_aws_pg_proto_rawDesc = []byte{
	0x0a, 0x21, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2f, 0x76, 0x31,
	0x2f, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x2f, 0x61, 0x77, 0x73, 0x5f, 0x70, 0x67, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x14, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61,
	0x2e, 0x76, 0x31, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76,
	0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x67, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x21, 0x6f, 0x35, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x6f, 0x35, 0x2f, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x71, 0x72, 0x65,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xc5, 0x01, 0x0a, 0x1d, 0x55, 0x70, 0x73, 0x65,
	0x72, 0x74, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61,
	0x73, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x3a, 0x0a, 0x07, 0x72, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35, 0x2e,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x07, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2b, 0x0a, 0x0c, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69,
	0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05,
	0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x0b, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x49, 0x64, 0x12, 0x3b, 0x0a, 0x04, 0x73, 0x70, 0x65, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x27, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x43, 0x72, 0x65,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x52, 0x04, 0x73, 0x70, 0x65, 0x63, 0x22,
	0xc5, 0x01, 0x0a, 0x1e, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x50, 0x6f, 0x73, 0x74, 0x67,
	0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x3a, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x69,
	0x6e, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2b,
	0x0a, 0x0c, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x0b,
	0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x3a, 0x0a, 0x04, 0x73,
	0x70, 0x65, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x6f, 0x35, 0x2e, 0x61,
	0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x6f,
	0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70, 0x53, 0x70, 0x65,
	0x63, 0x52, 0x04, 0x73, 0x70, 0x65, 0x63, 0x22, 0xc7, 0x01, 0x0a, 0x1e, 0x4d, 0x69, 0x67, 0x72,
	0x61, 0x74, 0x65, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62,
	0x61, 0x73, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x3a, 0x0a, 0x07, 0x72, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35,
	0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x07, 0x72,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2b, 0x0a, 0x0c, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48,
	0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x0b, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x49, 0x64, 0x12, 0x3c, 0x0a, 0x04, 0x73, 0x70, 0x65, 0x63, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x28, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x4d, 0x69,
	0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x52, 0x04, 0x73, 0x70, 0x65,
	0x63, 0x22, 0x90, 0x02, 0x0a, 0x1d, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61,
	0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x12, 0x3a, 0x0a, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x69, 0x6e, 0x67, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x4d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x52, 0x07, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x2b, 0x0a, 0x0c, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52,
	0x0b, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x23, 0x0a, 0x08,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08,
	0xba, 0x48, 0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x07, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x49,
	0x64, 0x12, 0x3c, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x0e, 0x32, 0x24, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2e,
	0x76, 0x31, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x2e, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65,
	0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12,
	0x19, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x88, 0x01, 0x01, 0x42, 0x08, 0x0a, 0x06, 0x5f, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x2a, 0x83, 0x01, 0x0a, 0x0e, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65,
	0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1f, 0x0a, 0x1b, 0x50, 0x4f, 0x53, 0x54, 0x47,
	0x52, 0x45, 0x53, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45,
	0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x1b, 0x0a, 0x17, 0x50, 0x4f, 0x53, 0x54,
	0x47, 0x52, 0x45, 0x53, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x53, 0x54, 0x41, 0x52,
	0x54, 0x45, 0x44, 0x10, 0x01, 0x12, 0x18, 0x0a, 0x14, 0x50, 0x4f, 0x53, 0x54, 0x47, 0x52, 0x45,
	0x53, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x44, 0x4f, 0x4e, 0x45, 0x10, 0x02, 0x12,
	0x19, 0x0a, 0x15, 0x50, 0x4f, 0x53, 0x54, 0x47, 0x52, 0x45, 0x53, 0x5f, 0x53, 0x54, 0x41, 0x54,
	0x55, 0x53, 0x5f, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10, 0x03, 0x32, 0xef, 0x02, 0x0a, 0x14, 0x50,
	0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x54, 0x6f,
	0x70, 0x69, 0x63, 0x12, 0x67, 0x0a, 0x16, 0x55, 0x70, 0x73, 0x65, 0x72, 0x74, 0x50, 0x6f, 0x73,
	0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x12, 0x33, 0x2e,
	0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x74,
	0x6f, 0x70, 0x69, 0x63, 0x2e, 0x55, 0x70, 0x73, 0x65, 0x72, 0x74, 0x50, 0x6f, 0x73, 0x74, 0x67,
	0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x69, 0x0a, 0x17,
	0x4d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x65, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44,
	0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x12, 0x34, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73,
	0x69, 0x6e, 0x66, 0x72, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x2e, 0x4d,
	0x69, 0x67, 0x72, 0x61, 0x74, 0x65, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61,
	0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x69, 0x0a, 0x17, 0x43, 0x6c, 0x65, 0x61, 0x6e,
	0x75, 0x70, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61,
	0x73, 0x65, 0x12, 0x34, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61,
	0x2e, 0x76, 0x31, 0x2e, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x2e, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x75,
	0x70, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61, 0x74, 0x61, 0x62, 0x61, 0x73,
	0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x22, 0x00, 0x1a, 0x18, 0xd2, 0xa2, 0xf5, 0xe4, 0x02, 0x12, 0x1a, 0x10, 0x0a, 0x0e, 0x6f, 0x35,
	0x2d, 0x61, 0x77, 0x73, 0x2d, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x32, 0x97, 0x01, 0x0a,
	0x12, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x54, 0x6f,
	0x70, 0x69, 0x63, 0x12, 0x67, 0x0a, 0x16, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44,
	0x61, 0x74, 0x61, 0x62, 0x61, 0x73, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x33, 0x2e,
	0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2e, 0x76, 0x31, 0x2e, 0x74,
	0x6f, 0x70, 0x69, 0x63, 0x2e, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x44, 0x61, 0x74,
	0x61, 0x62, 0x61, 0x73, 0x65, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x1a, 0x18, 0xd2, 0xa2,
	0xf5, 0xe4, 0x02, 0x12, 0x22, 0x10, 0x0a, 0x0e, 0x6f, 0x35, 0x2d, 0x61, 0x77, 0x73, 0x2d, 0x63,
	0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x42, 0x42, 0x5a, 0x40, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x6f, 0x35, 0x2d,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6f,
	0x35, 0x2f, 0x61, 0x77, 0x73, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x77,
	0x73, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x5f, 0x74, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescOnce sync.Once
	file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescData = file_o5_awsinfra_v1_topic_aws_pg_proto_rawDesc
)

func file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescGZIP() []byte {
	file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescOnce.Do(func() {
		file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescData)
	})
	return file_o5_awsinfra_v1_topic_aws_pg_proto_rawDescData
}

var file_o5_awsinfra_v1_topic_aws_pg_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_o5_awsinfra_v1_topic_aws_pg_proto_goTypes = []interface{}{
	(PostgresStatus)(0),                          // 0: o5.awsinfra.v1.topic.PostgresStatus
	(*UpsertPostgresDatabaseMessage)(nil),        // 1: o5.awsinfra.v1.topic.UpsertPostgresDatabaseMessage
	(*CleanupPostgresDatabaseMessage)(nil),       // 2: o5.awsinfra.v1.topic.CleanupPostgresDatabaseMessage
	(*MigratePostgresDatabaseMessage)(nil),       // 3: o5.awsinfra.v1.topic.MigratePostgresDatabaseMessage
	(*PostgresDatabaseStatusMessage)(nil),        // 4: o5.awsinfra.v1.topic.PostgresDatabaseStatusMessage
	(*messaging_pb.RequestMetadata)(nil),         // 5: o5.messaging.v1.RequestMetadata
	(*awsdeployer_pb.PostgresCreationSpec)(nil),  // 6: o5.awsdeployer.v1.PostgresCreationSpec
	(*awsdeployer_pb.PostgresCleanupSpec)(nil),   // 7: o5.awsdeployer.v1.PostgresCleanupSpec
	(*awsdeployer_pb.PostgresMigrationSpec)(nil), // 8: o5.awsdeployer.v1.PostgresMigrationSpec
	(*emptypb.Empty)(nil),                        // 9: google.protobuf.Empty
}
var file_o5_awsinfra_v1_topic_aws_pg_proto_depIdxs = []int32{
	5,  // 0: o5.awsinfra.v1.topic.UpsertPostgresDatabaseMessage.request:type_name -> o5.messaging.v1.RequestMetadata
	6,  // 1: o5.awsinfra.v1.topic.UpsertPostgresDatabaseMessage.spec:type_name -> o5.awsdeployer.v1.PostgresCreationSpec
	5,  // 2: o5.awsinfra.v1.topic.CleanupPostgresDatabaseMessage.request:type_name -> o5.messaging.v1.RequestMetadata
	7,  // 3: o5.awsinfra.v1.topic.CleanupPostgresDatabaseMessage.spec:type_name -> o5.awsdeployer.v1.PostgresCleanupSpec
	5,  // 4: o5.awsinfra.v1.topic.MigratePostgresDatabaseMessage.request:type_name -> o5.messaging.v1.RequestMetadata
	8,  // 5: o5.awsinfra.v1.topic.MigratePostgresDatabaseMessage.spec:type_name -> o5.awsdeployer.v1.PostgresMigrationSpec
	5,  // 6: o5.awsinfra.v1.topic.PostgresDatabaseStatusMessage.request:type_name -> o5.messaging.v1.RequestMetadata
	0,  // 7: o5.awsinfra.v1.topic.PostgresDatabaseStatusMessage.status:type_name -> o5.awsinfra.v1.topic.PostgresStatus
	1,  // 8: o5.awsinfra.v1.topic.PostgresRequestTopic.UpsertPostgresDatabase:input_type -> o5.awsinfra.v1.topic.UpsertPostgresDatabaseMessage
	3,  // 9: o5.awsinfra.v1.topic.PostgresRequestTopic.MigratePostgresDatabase:input_type -> o5.awsinfra.v1.topic.MigratePostgresDatabaseMessage
	2,  // 10: o5.awsinfra.v1.topic.PostgresRequestTopic.CleanupPostgresDatabase:input_type -> o5.awsinfra.v1.topic.CleanupPostgresDatabaseMessage
	4,  // 11: o5.awsinfra.v1.topic.PostgresReplyTopic.PostgresDatabaseStatus:input_type -> o5.awsinfra.v1.topic.PostgresDatabaseStatusMessage
	9,  // 12: o5.awsinfra.v1.topic.PostgresRequestTopic.UpsertPostgresDatabase:output_type -> google.protobuf.Empty
	9,  // 13: o5.awsinfra.v1.topic.PostgresRequestTopic.MigratePostgresDatabase:output_type -> google.protobuf.Empty
	9,  // 14: o5.awsinfra.v1.topic.PostgresRequestTopic.CleanupPostgresDatabase:output_type -> google.protobuf.Empty
	9,  // 15: o5.awsinfra.v1.topic.PostgresReplyTopic.PostgresDatabaseStatus:output_type -> google.protobuf.Empty
	12, // [12:16] is the sub-list for method output_type
	8,  // [8:12] is the sub-list for method input_type
	8,  // [8:8] is the sub-list for extension type_name
	8,  // [8:8] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_o5_awsinfra_v1_topic_aws_pg_proto_init() }
func file_o5_awsinfra_v1_topic_aws_pg_proto_init() {
	if File_o5_awsinfra_v1_topic_aws_pg_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*UpsertPostgresDatabaseMessage); i {
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
		file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CleanupPostgresDatabaseMessage); i {
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
		file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MigratePostgresDatabaseMessage); i {
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
		file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostgresDatabaseStatusMessage); i {
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
	file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes[3].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_awsinfra_v1_topic_aws_pg_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_o5_awsinfra_v1_topic_aws_pg_proto_goTypes,
		DependencyIndexes: file_o5_awsinfra_v1_topic_aws_pg_proto_depIdxs,
		EnumInfos:         file_o5_awsinfra_v1_topic_aws_pg_proto_enumTypes,
		MessageInfos:      file_o5_awsinfra_v1_topic_aws_pg_proto_msgTypes,
	}.Build()
	File_o5_awsinfra_v1_topic_aws_pg_proto = out.File
	file_o5_awsinfra_v1_topic_aws_pg_proto_rawDesc = nil
	file_o5_awsinfra_v1_topic_aws_pg_proto_goTypes = nil
	file_o5_awsinfra_v1_topic_aws_pg_proto_depIdxs = nil
}