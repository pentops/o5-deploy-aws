// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/aws/deployer/v1/pgmigrate.proto

package awsdeployer_pb

import (
	messaging_j5pb "github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
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

type PostgresSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DbName                  string   `protobuf:"bytes,1,opt,name=db_name,json=dbName,proto3" json:"db_name,omitempty"`
	DbExtensions            []string `protobuf:"bytes,3,rep,name=db_extensions,json=dbExtensions,proto3" json:"db_extensions,omitempty"`
	RootSecretName          string   `protobuf:"bytes,5,opt,name=root_secret_name,json=rootSecretName,proto3" json:"root_secret_name,omitempty"`
	MigrationTaskOutputName *string  `protobuf:"bytes,7,opt,name=migration_task_output_name,json=migrationTaskOutputName,proto3,oneof" json:"migration_task_output_name,omitempty"`
	SecretOutputName        string   `protobuf:"bytes,9,opt,name=secret_output_name,json=secretOutputName,proto3" json:"secret_output_name,omitempty"`
}

func (x *PostgresSpec) Reset() {
	*x = PostgresSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostgresSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostgresSpec) ProtoMessage() {}

func (x *PostgresSpec) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostgresSpec.ProtoReflect.Descriptor instead.
func (*PostgresSpec) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_pgmigrate_proto_rawDescGZIP(), []int{0}
}

func (x *PostgresSpec) GetDbName() string {
	if x != nil {
		return x.DbName
	}
	return ""
}

func (x *PostgresSpec) GetDbExtensions() []string {
	if x != nil {
		return x.DbExtensions
	}
	return nil
}

func (x *PostgresSpec) GetRootSecretName() string {
	if x != nil {
		return x.RootSecretName
	}
	return ""
}

func (x *PostgresSpec) GetMigrationTaskOutputName() string {
	if x != nil && x.MigrationTaskOutputName != nil {
		return *x.MigrationTaskOutputName
	}
	return ""
}

func (x *PostgresSpec) GetSecretOutputName() string {
	if x != nil {
		return x.SecretOutputName
	}
	return ""
}

type PostgresCreationSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DbName            string   `protobuf:"bytes,1,opt,name=db_name,json=dbName,proto3" json:"db_name,omitempty"`
	RootSecretName    string   `protobuf:"bytes,2,opt,name=root_secret_name,json=rootSecretName,proto3" json:"root_secret_name,omitempty"`
	DbExtensions      []string `protobuf:"bytes,3,rep,name=db_extensions,json=dbExtensions,proto3" json:"db_extensions,omitempty"`
	SecretArn         string   `protobuf:"bytes,4,opt,name=secret_arn,json=secretArn,proto3" json:"secret_arn,omitempty"`
	RotateCredentials bool     `protobuf:"varint,5,opt,name=rotate_credentials,json=rotateCredentials,proto3" json:"rotate_credentials,omitempty"`
}

func (x *PostgresCreationSpec) Reset() {
	*x = PostgresCreationSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostgresCreationSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostgresCreationSpec) ProtoMessage() {}

func (x *PostgresCreationSpec) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostgresCreationSpec.ProtoReflect.Descriptor instead.
func (*PostgresCreationSpec) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_pgmigrate_proto_rawDescGZIP(), []int{1}
}

func (x *PostgresCreationSpec) GetDbName() string {
	if x != nil {
		return x.DbName
	}
	return ""
}

func (x *PostgresCreationSpec) GetRootSecretName() string {
	if x != nil {
		return x.RootSecretName
	}
	return ""
}

func (x *PostgresCreationSpec) GetDbExtensions() []string {
	if x != nil {
		return x.DbExtensions
	}
	return nil
}

func (x *PostgresCreationSpec) GetSecretArn() string {
	if x != nil {
		return x.SecretArn
	}
	return ""
}

func (x *PostgresCreationSpec) GetRotateCredentials() bool {
	if x != nil {
		return x.RotateCredentials
	}
	return false
}

type PostgresCleanupSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DbName         string `protobuf:"bytes,1,opt,name=db_name,json=dbName,proto3" json:"db_name,omitempty"`
	RootSecretName string `protobuf:"bytes,2,opt,name=root_secret_name,json=rootSecretName,proto3" json:"root_secret_name,omitempty"`
}

func (x *PostgresCleanupSpec) Reset() {
	*x = PostgresCleanupSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostgresCleanupSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostgresCleanupSpec) ProtoMessage() {}

func (x *PostgresCleanupSpec) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostgresCleanupSpec.ProtoReflect.Descriptor instead.
func (*PostgresCleanupSpec) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_pgmigrate_proto_rawDescGZIP(), []int{2}
}

func (x *PostgresCleanupSpec) GetDbName() string {
	if x != nil {
		return x.DbName
	}
	return ""
}

func (x *PostgresCleanupSpec) GetRootSecretName() string {
	if x != nil {
		return x.RootSecretName
	}
	return ""
}

type PostgresMigrationSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MigrationTaskArn string `protobuf:"bytes,1,opt,name=migration_task_arn,json=migrationTaskArn,proto3" json:"migration_task_arn,omitempty"`
	SecretArn        string `protobuf:"bytes,2,opt,name=secret_arn,json=secretArn,proto3" json:"secret_arn,omitempty"`
	EcsClusterName   string `protobuf:"bytes,3,opt,name=ecs_cluster_name,json=ecsClusterName,proto3" json:"ecs_cluster_name,omitempty"`
}

func (x *PostgresMigrationSpec) Reset() {
	*x = PostgresMigrationSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PostgresMigrationSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostgresMigrationSpec) ProtoMessage() {}

func (x *PostgresMigrationSpec) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PostgresMigrationSpec.ProtoReflect.Descriptor instead.
func (*PostgresMigrationSpec) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_pgmigrate_proto_rawDescGZIP(), []int{3}
}

func (x *PostgresMigrationSpec) GetMigrationTaskArn() string {
	if x != nil {
		return x.MigrationTaskArn
	}
	return ""
}

func (x *PostgresMigrationSpec) GetSecretArn() string {
	if x != nil {
		return x.SecretArn
	}
	return ""
}

func (x *PostgresMigrationSpec) GetEcsClusterName() string {
	if x != nil {
		return x.EcsClusterName
	}
	return ""
}

type MigrationTaskContext struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Upstream    *messaging_j5pb.RequestMetadata `protobuf:"bytes,1,opt,name=upstream,proto3" json:"upstream,omitempty"`
	MigrationId string                          `protobuf:"bytes,2,opt,name=migration_id,json=migrationId,proto3" json:"migration_id,omitempty"`
}

func (x *MigrationTaskContext) Reset() {
	*x = MigrationTaskContext{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MigrationTaskContext) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MigrationTaskContext) ProtoMessage() {}

func (x *MigrationTaskContext) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MigrationTaskContext.ProtoReflect.Descriptor instead.
func (*MigrationTaskContext) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_pgmigrate_proto_rawDescGZIP(), []int{4}
}

func (x *MigrationTaskContext) GetUpstream() *messaging_j5pb.RequestMetadata {
	if x != nil {
		return x.Upstream
	}
	return nil
}

func (x *MigrationTaskContext) GetMigrationId() string {
	if x != nil {
		return x.MigrationId
	}
	return ""
}

var File_o5_aws_deployer_v1_pgmigrate_proto protoreflect.FileDescriptor

var file_o5_aws_deployer_v1_pgmigrate_proto_rawDesc = []byte{
	0x0a, 0x22, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x67, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x1a, 0x1c, 0x6a, 0x35, 0x2f, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x65, 0x71, 0x72, 0x65, 0x73,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x85, 0x02, 0x0a, 0x0c, 0x50, 0x6f, 0x73, 0x74, 0x67,
	0x72, 0x65, 0x73, 0x53, 0x70, 0x65, 0x63, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x62, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x62, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x23, 0x0a, 0x0d, 0x64, 0x62, 0x5f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e,
	0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0c, 0x64, 0x62, 0x45, 0x78, 0x74, 0x65, 0x6e,
	0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x28, 0x0a, 0x10, 0x72, 0x6f, 0x6f, 0x74, 0x5f, 0x73, 0x65,
	0x63, 0x72, 0x65, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x0e, 0x72, 0x6f, 0x6f, 0x74, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12,
	0x40, 0x0a, 0x1a, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x61, 0x73,
	0x6b, 0x5f, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x17, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x54, 0x61, 0x73, 0x6b, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x88, 0x01,
	0x01, 0x12, 0x2c, 0x0a, 0x12, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f, 0x6f, 0x75, 0x74, 0x70,
	0x75, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x73,
	0x65, 0x63, 0x72, 0x65, 0x74, 0x4f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x42,
	0x1d, 0x0a, 0x1b, 0x5f, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74, 0x61,
	0x73, 0x6b, 0x5f, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0xcc,
	0x01, 0x0a, 0x14, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65, 0x63, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x62, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x62, 0x4e, 0x61, 0x6d, 0x65,
	0x12, 0x28, 0x0a, 0x10, 0x72, 0x6f, 0x6f, 0x74, 0x5f, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f,
	0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x6f, 0x6f, 0x74,
	0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x64, 0x62,
	0x5f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x0c, 0x64, 0x62, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12,
	0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f, 0x61, 0x72, 0x6e, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x41, 0x72, 0x6e, 0x12, 0x2d,
	0x0a, 0x12, 0x72, 0x6f, 0x74, 0x61, 0x74, 0x65, 0x5f, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74,
	0x69, 0x61, 0x6c, 0x73, 0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x11, 0x72, 0x6f, 0x74, 0x61,
	0x74, 0x65, 0x43, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x22, 0x58, 0x0a,
	0x13, 0x50, 0x6f, 0x73, 0x74, 0x67, 0x72, 0x65, 0x73, 0x43, 0x6c, 0x65, 0x61, 0x6e, 0x75, 0x70,
	0x53, 0x70, 0x65, 0x63, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x62, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x62, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x28, 0x0a,
	0x10, 0x72, 0x6f, 0x6f, 0x74, 0x5f, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x6f, 0x6f, 0x74, 0x53, 0x65, 0x63,
	0x72, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x8e, 0x01, 0x0a, 0x15, 0x50, 0x6f, 0x73, 0x74,
	0x67, 0x72, 0x65, 0x73, 0x4d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x70, 0x65,
	0x63, 0x12, 0x2c, 0x0a, 0x12, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x74,
	0x61, 0x73, 0x6b, 0x5f, 0x61, 0x72, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x6d,
	0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73, 0x6b, 0x41, 0x72, 0x6e, 0x12,
	0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x5f, 0x61, 0x72, 0x6e, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x41, 0x72, 0x6e, 0x12, 0x28,
	0x0a, 0x10, 0x65, 0x63, 0x73, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x65, 0x63, 0x73, 0x43, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x77, 0x0a, 0x14, 0x4d, 0x69, 0x67, 0x72,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x54, 0x61, 0x73, 0x6b, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x78, 0x74,
	0x12, 0x3c, 0x0a, 0x08, 0x75, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6a, 0x35, 0x2e, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x69, 0x6e,
	0x67, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x74, 0x61,
	0x64, 0x61, 0x74, 0x61, 0x52, 0x08, 0x75, 0x70, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x12, 0x21,
	0x0a, 0x0c, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6d, 0x69, 0x67, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49,
	0x64, 0x42, 0x48, 0x5a, 0x46, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x6f, 0x35, 0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73,
	0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x77, 0x73,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_o5_aws_deployer_v1_pgmigrate_proto_rawDescOnce sync.Once
	file_o5_aws_deployer_v1_pgmigrate_proto_rawDescData = file_o5_aws_deployer_v1_pgmigrate_proto_rawDesc
)

func file_o5_aws_deployer_v1_pgmigrate_proto_rawDescGZIP() []byte {
	file_o5_aws_deployer_v1_pgmigrate_proto_rawDescOnce.Do(func() {
		file_o5_aws_deployer_v1_pgmigrate_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_aws_deployer_v1_pgmigrate_proto_rawDescData)
	})
	return file_o5_aws_deployer_v1_pgmigrate_proto_rawDescData
}

var file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_o5_aws_deployer_v1_pgmigrate_proto_goTypes = []interface{}{
	(*PostgresSpec)(nil),                   // 0: o5.aws.deployer.v1.PostgresSpec
	(*PostgresCreationSpec)(nil),           // 1: o5.aws.deployer.v1.PostgresCreationSpec
	(*PostgresCleanupSpec)(nil),            // 2: o5.aws.deployer.v1.PostgresCleanupSpec
	(*PostgresMigrationSpec)(nil),          // 3: o5.aws.deployer.v1.PostgresMigrationSpec
	(*MigrationTaskContext)(nil),           // 4: o5.aws.deployer.v1.MigrationTaskContext
	(*messaging_j5pb.RequestMetadata)(nil), // 5: j5.messaging.v1.RequestMetadata
}
var file_o5_aws_deployer_v1_pgmigrate_proto_depIdxs = []int32{
	5, // 0: o5.aws.deployer.v1.MigrationTaskContext.upstream:type_name -> j5.messaging.v1.RequestMetadata
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_o5_aws_deployer_v1_pgmigrate_proto_init() }
func file_o5_aws_deployer_v1_pgmigrate_proto_init() {
	if File_o5_aws_deployer_v1_pgmigrate_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostgresSpec); i {
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
		file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostgresCreationSpec); i {
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
		file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostgresCleanupSpec); i {
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
		file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PostgresMigrationSpec); i {
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
		file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MigrationTaskContext); i {
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
	file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes[0].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_aws_deployer_v1_pgmigrate_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_o5_aws_deployer_v1_pgmigrate_proto_goTypes,
		DependencyIndexes: file_o5_aws_deployer_v1_pgmigrate_proto_depIdxs,
		MessageInfos:      file_o5_aws_deployer_v1_pgmigrate_proto_msgTypes,
	}.Build()
	File_o5_aws_deployer_v1_pgmigrate_proto = out.File
	file_o5_aws_deployer_v1_pgmigrate_proto_rawDesc = nil
	file_o5_aws_deployer_v1_pgmigrate_proto_goTypes = nil
	file_o5_aws_deployer_v1_pgmigrate_proto_depIdxs = nil
}
