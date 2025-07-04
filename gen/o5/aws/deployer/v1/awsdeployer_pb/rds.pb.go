// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/aws/deployer/v1/rds.proto

package awsdeployer_pb

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type RDSCreateSpec struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	DbExtensions []string `protobuf:"bytes,2,rep,name=db_extensions,json=dbExtensions,proto3" json:"db_extensions,omitempty"`
	DbName       string   `protobuf:"bytes,3,opt,name=db_name,json=dbName,proto3" json:"db_name,omitempty"`
}

func (x *RDSCreateSpec) Reset() {
	*x = RDSCreateSpec{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSCreateSpec) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSCreateSpec) ProtoMessage() {}

func (x *RDSCreateSpec) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSCreateSpec.ProtoReflect.Descriptor instead.
func (*RDSCreateSpec) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{0}
}

func (x *RDSCreateSpec) GetDbExtensions() []string {
	if x != nil {
		return x.DbExtensions
	}
	return nil
}

func (x *RDSCreateSpec) GetDbName() string {
	if x != nil {
		return x.DbName
	}
	return ""
}

type RDSAppSpecType struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//
	//	*RDSAppSpecType_AppSecret
	//	*RDSAppSpecType_AppConn
	Type isRDSAppSpecType_Type `protobuf_oneof:"type"`
}

func (x *RDSAppSpecType) Reset() {
	*x = RDSAppSpecType{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSAppSpecType) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSAppSpecType) ProtoMessage() {}

func (x *RDSAppSpecType) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSAppSpecType.ProtoReflect.Descriptor instead.
func (*RDSAppSpecType) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{1}
}

func (m *RDSAppSpecType) GetType() isRDSAppSpecType_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *RDSAppSpecType) GetAppSecret() *RDSAppSpecType_SecretsManager {
	if x, ok := x.GetType().(*RDSAppSpecType_AppSecret); ok {
		return x.AppSecret
	}
	return nil
}

func (x *RDSAppSpecType) GetAppConn() *RDSAppSpecType_Aurora {
	if x, ok := x.GetType().(*RDSAppSpecType_AppConn); ok {
		return x.AppConn
	}
	return nil
}

type isRDSAppSpecType_Type interface {
	isRDSAppSpecType_Type()
}

type RDSAppSpecType_AppSecret struct {
	AppSecret *RDSAppSpecType_SecretsManager `protobuf:"bytes,3,opt,name=app_secret,json=appSecret,proto3,oneof"`
}

type RDSAppSpecType_AppConn struct {
	AppConn *RDSAppSpecType_Aurora `protobuf:"bytes,4,opt,name=app_conn,json=appConn,proto3,oneof"`
}

func (*RDSAppSpecType_AppSecret) isRDSAppSpecType_Type() {}

func (*RDSAppSpecType_AppConn) isRDSAppSpecType_Type() {}

type RDSHostType struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Type:
	//
	//	*RDSHostType_Aurora_
	//	*RDSHostType_SecretsManager_
	Type isRDSHostType_Type `protobuf_oneof:"type"`
}

func (x *RDSHostType) Reset() {
	*x = RDSHostType{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSHostType) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSHostType) ProtoMessage() {}

func (x *RDSHostType) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSHostType.ProtoReflect.Descriptor instead.
func (*RDSHostType) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{2}
}

func (m *RDSHostType) GetType() isRDSHostType_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (x *RDSHostType) GetAurora() *RDSHostType_Aurora {
	if x, ok := x.GetType().(*RDSHostType_Aurora_); ok {
		return x.Aurora
	}
	return nil
}

func (x *RDSHostType) GetSecretsManager() *RDSHostType_SecretsManager {
	if x, ok := x.GetType().(*RDSHostType_SecretsManager_); ok {
		return x.SecretsManager
	}
	return nil
}

type isRDSHostType_Type interface {
	isRDSHostType_Type()
}

type RDSHostType_Aurora_ struct {
	Aurora *RDSHostType_Aurora `protobuf:"bytes,1,opt,name=aurora,proto3,oneof"`
}

type RDSHostType_SecretsManager_ struct {
	SecretsManager *RDSHostType_SecretsManager `protobuf:"bytes,2,opt,name=secrets_manager,json=secretsManager,proto3,oneof"`
}

func (*RDSHostType_Aurora_) isRDSHostType_Type() {}

func (*RDSHostType_SecretsManager_) isRDSHostType_Type() {}

type AuroraConnection struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// AWS API: 'Endpoint' - hostname without port.
	// Used for connection and authentication tokens.
	Endpoint string `protobuf:"bytes,1,opt,name=endpoint,proto3" json:"endpoint,omitempty"`
	// AWS API: 'Port'
	// The port that the database engine is listening on.
	Port int32 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	// A username to use getting the token, which then also matches the user
	// implementation of the engine.
	// For root access, this could be the master_user but it only needs to be a
	// user with the correct permissions.
	DbUser string `protobuf:"bytes,3,opt,name=db_user,json=dbUser,proto3" json:"db_user,omitempty"`
	// defaults to the user's name. Used when accessing, not required for the
	// token.
	DbName     string `protobuf:"bytes,4,opt,name=db_name,json=dbName,proto3" json:"db_name,omitempty"`
	Identifier string `protobuf:"bytes,5,opt,name=identifier,proto3" json:"identifier,omitempty"`
}

func (x *AuroraConnection) Reset() {
	*x = AuroraConnection{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AuroraConnection) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuroraConnection) ProtoMessage() {}

func (x *AuroraConnection) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuroraConnection.ProtoReflect.Descriptor instead.
func (*AuroraConnection) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{3}
}

func (x *AuroraConnection) GetEndpoint() string {
	if x != nil {
		return x.Endpoint
	}
	return ""
}

func (x *AuroraConnection) GetPort() int32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *AuroraConnection) GetDbUser() string {
	if x != nil {
		return x.DbUser
	}
	return ""
}

func (x *AuroraConnection) GetDbName() string {
	if x != nil {
		return x.DbName
	}
	return ""
}

func (x *AuroraConnection) GetIdentifier() string {
	if x != nil {
		return x.Identifier
	}
	return ""
}

type RDSAppSpecType_SecretsManager struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AppSecretName     string `protobuf:"bytes,2,opt,name=app_secret_name,json=appSecretName,proto3" json:"app_secret_name,omitempty"`
	RotateCredentials bool   `protobuf:"varint,5,opt,name=rotate_credentials,json=rotateCredentials,proto3" json:"rotate_credentials,omitempty"`
}

func (x *RDSAppSpecType_SecretsManager) Reset() {
	*x = RDSAppSpecType_SecretsManager{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSAppSpecType_SecretsManager) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSAppSpecType_SecretsManager) ProtoMessage() {}

func (x *RDSAppSpecType_SecretsManager) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSAppSpecType_SecretsManager.ProtoReflect.Descriptor instead.
func (*RDSAppSpecType_SecretsManager) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{1, 0}
}

func (x *RDSAppSpecType_SecretsManager) GetAppSecretName() string {
	if x != nil {
		return x.AppSecretName
	}
	return ""
}

func (x *RDSAppSpecType_SecretsManager) GetRotateCredentials() bool {
	if x != nil {
		return x.RotateCredentials
	}
	return false
}

type RDSAppSpecType_Aurora struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Conn *AuroraConnection `protobuf:"bytes,1,opt,name=conn,proto3" json:"conn,omitempty"`
}

func (x *RDSAppSpecType_Aurora) Reset() {
	*x = RDSAppSpecType_Aurora{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSAppSpecType_Aurora) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSAppSpecType_Aurora) ProtoMessage() {}

func (x *RDSAppSpecType_Aurora) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSAppSpecType_Aurora.ProtoReflect.Descriptor instead.
func (*RDSAppSpecType_Aurora) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{1, 1}
}

func (x *RDSAppSpecType_Aurora) GetConn() *AuroraConnection {
	if x != nil {
		return x.Conn
	}
	return nil
}

type RDSHostType_Aurora struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Conn *AuroraConnection `protobuf:"bytes,1,opt,name=conn,proto3" json:"conn,omitempty"`
}

func (x *RDSHostType_Aurora) Reset() {
	*x = RDSHostType_Aurora{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSHostType_Aurora) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSHostType_Aurora) ProtoMessage() {}

func (x *RDSHostType_Aurora) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSHostType_Aurora.ProtoReflect.Descriptor instead.
func (*RDSHostType_Aurora) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{2, 0}
}

func (x *RDSHostType_Aurora) GetConn() *AuroraConnection {
	if x != nil {
		return x.Conn
	}
	return nil
}

type RDSHostType_SecretsManager struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SecretName string `protobuf:"bytes,1,opt,name=secret_name,json=secretName,proto3" json:"secret_name,omitempty"`
}

func (x *RDSHostType_SecretsManager) Reset() {
	*x = RDSHostType_SecretsManager{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSHostType_SecretsManager) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSHostType_SecretsManager) ProtoMessage() {}

func (x *RDSHostType_SecretsManager) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_rds_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSHostType_SecretsManager.ProtoReflect.Descriptor instead.
func (*RDSHostType_SecretsManager) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_rds_proto_rawDescGZIP(), []int{2, 1}
}

func (x *RDSHostType_SecretsManager) GetSecretName() string {
	if x != nil {
		return x.SecretName
	}
	return ""
}

var File_o5_aws_deployer_v1_rds_proto protoreflect.FileDescriptor

var file_o5_aws_deployer_v1_rds_proto_rawDesc = []byte{
	0x0a, 0x1c, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x64, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x12,
	0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x22, 0x4d, 0x0a, 0x0d, 0x52, 0x44, 0x53, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x53,
	0x70, 0x65, 0x63, 0x12, 0x23, 0x0a, 0x0d, 0x64, 0x62, 0x5f, 0x65, 0x78, 0x74, 0x65, 0x6e, 0x73,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0c, 0x64, 0x62, 0x45, 0x78,
	0x74, 0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x62, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x62, 0x4e, 0x61, 0x6d,
	0x65, 0x22, 0xe1, 0x02, 0x0a, 0x0e, 0x52, 0x44, 0x53, 0x41, 0x70, 0x70, 0x53, 0x70, 0x65, 0x63,
	0x54, 0x79, 0x70, 0x65, 0x12, 0x52, 0x0a, 0x0a, 0x61, 0x70, 0x70, 0x5f, 0x73, 0x65, 0x63, 0x72,
	0x65, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x31, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77,
	0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x44,
	0x53, 0x41, 0x70, 0x70, 0x53, 0x70, 0x65, 0x63, 0x54, 0x79, 0x70, 0x65, 0x2e, 0x53, 0x65, 0x63,
	0x72, 0x65, 0x74, 0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x48, 0x00, 0x52, 0x09, 0x61,
	0x70, 0x70, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x12, 0x46, 0x0a, 0x08, 0x61, 0x70, 0x70, 0x5f,
	0x63, 0x6f, 0x6e, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x6f, 0x35, 0x2e,
	0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x52, 0x44, 0x53, 0x41, 0x70, 0x70, 0x53, 0x70, 0x65, 0x63, 0x54, 0x79, 0x70, 0x65, 0x2e, 0x41,
	0x75, 0x72, 0x6f, 0x72, 0x61, 0x48, 0x00, 0x52, 0x07, 0x61, 0x70, 0x70, 0x43, 0x6f, 0x6e, 0x6e,
	0x1a, 0x67, 0x0a, 0x0e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67,
	0x65, 0x72, 0x12, 0x26, 0x0a, 0x0f, 0x61, 0x70, 0x70, 0x5f, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0d, 0x61, 0x70, 0x70,
	0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x2d, 0x0a, 0x12, 0x72, 0x6f,
	0x74, 0x61, 0x74, 0x65, 0x5f, 0x63, 0x72, 0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73,
	0x18, 0x05, 0x20, 0x01, 0x28, 0x08, 0x52, 0x11, 0x72, 0x6f, 0x74, 0x61, 0x74, 0x65, 0x43, 0x72,
	0x65, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x61, 0x6c, 0x73, 0x1a, 0x42, 0x0a, 0x06, 0x41, 0x75, 0x72,
	0x6f, 0x72, 0x61, 0x12, 0x38, 0x0a, 0x04, 0x63, 0x6f, 0x6e, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x24, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f,
	0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x43, 0x6f, 0x6e,
	0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x04, 0x63, 0x6f, 0x6e, 0x6e, 0x42, 0x06, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0xa9, 0x02, 0x0a, 0x0b, 0x52, 0x44, 0x53, 0x48, 0x6f, 0x73,
	0x74, 0x54, 0x79, 0x70, 0x65, 0x12, 0x40, 0x0a, 0x06, 0x61, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x26, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x44, 0x53, 0x48, 0x6f,
	0x73, 0x74, 0x54, 0x79, 0x70, 0x65, 0x2e, 0x41, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x48, 0x00, 0x52,
	0x06, 0x61, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x12, 0x59, 0x0a, 0x0f, 0x73, 0x65, 0x63, 0x72, 0x65,
	0x74, 0x73, 0x5f, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x2e, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x44, 0x53, 0x48, 0x6f, 0x73, 0x74, 0x54, 0x79, 0x70,
	0x65, 0x2e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74, 0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72,
	0x48, 0x00, 0x52, 0x0e, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74, 0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67,
	0x65, 0x72, 0x1a, 0x42, 0x0a, 0x06, 0x41, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x12, 0x38, 0x0a, 0x04,
	0x63, 0x6f, 0x6e, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x24, 0x2e, 0x6f, 0x35, 0x2e,
	0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x41, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x43, 0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e,
	0x52, 0x04, 0x63, 0x6f, 0x6e, 0x6e, 0x1a, 0x31, 0x0a, 0x0e, 0x53, 0x65, 0x63, 0x72, 0x65, 0x74,
	0x73, 0x4d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x65, 0x63, 0x72,
	0x65, 0x74, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73,
	0x65, 0x63, 0x72, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x42, 0x06, 0x0a, 0x04, 0x74, 0x79, 0x70,
	0x65, 0x22, 0x94, 0x01, 0x0a, 0x10, 0x41, 0x75, 0x72, 0x6f, 0x72, 0x61, 0x43, 0x6f, 0x6e, 0x6e,
	0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69,
	0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x65, 0x6e, 0x64, 0x70, 0x6f, 0x69,
	0x6e, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x64, 0x62, 0x5f, 0x75, 0x73, 0x65,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x64, 0x62, 0x55, 0x73, 0x65, 0x72, 0x12,
	0x17, 0x0a, 0x07, 0x64, 0x62, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x64, 0x62, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1e, 0x0a, 0x0a, 0x69, 0x64, 0x65, 0x6e,
	0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x64,
	0x65, 0x6e, 0x74, 0x69, 0x66, 0x69, 0x65, 0x72, 0x42, 0x48, 0x5a, 0x46, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x6f,
	0x35, 0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67, 0x65, 0x6e,
	0x2f, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72,
	0x2f, 0x76, 0x31, 0x2f, 0x61, 0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f,
	0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_o5_aws_deployer_v1_rds_proto_rawDescOnce sync.Once
	file_o5_aws_deployer_v1_rds_proto_rawDescData = file_o5_aws_deployer_v1_rds_proto_rawDesc
)

func file_o5_aws_deployer_v1_rds_proto_rawDescGZIP() []byte {
	file_o5_aws_deployer_v1_rds_proto_rawDescOnce.Do(func() {
		file_o5_aws_deployer_v1_rds_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_aws_deployer_v1_rds_proto_rawDescData)
	})
	return file_o5_aws_deployer_v1_rds_proto_rawDescData
}

var file_o5_aws_deployer_v1_rds_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_o5_aws_deployer_v1_rds_proto_goTypes = []interface{}{
	(*RDSCreateSpec)(nil),                 // 0: o5.aws.deployer.v1.RDSCreateSpec
	(*RDSAppSpecType)(nil),                // 1: o5.aws.deployer.v1.RDSAppSpecType
	(*RDSHostType)(nil),                   // 2: o5.aws.deployer.v1.RDSHostType
	(*AuroraConnection)(nil),              // 3: o5.aws.deployer.v1.AuroraConnection
	(*RDSAppSpecType_SecretsManager)(nil), // 4: o5.aws.deployer.v1.RDSAppSpecType.SecretsManager
	(*RDSAppSpecType_Aurora)(nil),         // 5: o5.aws.deployer.v1.RDSAppSpecType.Aurora
	(*RDSHostType_Aurora)(nil),            // 6: o5.aws.deployer.v1.RDSHostType.Aurora
	(*RDSHostType_SecretsManager)(nil),    // 7: o5.aws.deployer.v1.RDSHostType.SecretsManager
}
var file_o5_aws_deployer_v1_rds_proto_depIdxs = []int32{
	4, // 0: o5.aws.deployer.v1.RDSAppSpecType.app_secret:type_name -> o5.aws.deployer.v1.RDSAppSpecType.SecretsManager
	5, // 1: o5.aws.deployer.v1.RDSAppSpecType.app_conn:type_name -> o5.aws.deployer.v1.RDSAppSpecType.Aurora
	6, // 2: o5.aws.deployer.v1.RDSHostType.aurora:type_name -> o5.aws.deployer.v1.RDSHostType.Aurora
	7, // 3: o5.aws.deployer.v1.RDSHostType.secrets_manager:type_name -> o5.aws.deployer.v1.RDSHostType.SecretsManager
	3, // 4: o5.aws.deployer.v1.RDSAppSpecType.Aurora.conn:type_name -> o5.aws.deployer.v1.AuroraConnection
	3, // 5: o5.aws.deployer.v1.RDSHostType.Aurora.conn:type_name -> o5.aws.deployer.v1.AuroraConnection
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_o5_aws_deployer_v1_rds_proto_init() }
func file_o5_aws_deployer_v1_rds_proto_init() {
	if File_o5_aws_deployer_v1_rds_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_aws_deployer_v1_rds_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSCreateSpec); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSAppSpecType); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSHostType); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AuroraConnection); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSAppSpecType_SecretsManager); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSAppSpecType_Aurora); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSHostType_Aurora); i {
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
		file_o5_aws_deployer_v1_rds_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSHostType_SecretsManager); i {
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
	file_o5_aws_deployer_v1_rds_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*RDSAppSpecType_AppSecret)(nil),
		(*RDSAppSpecType_AppConn)(nil),
	}
	file_o5_aws_deployer_v1_rds_proto_msgTypes[2].OneofWrappers = []interface{}{
		(*RDSHostType_Aurora_)(nil),
		(*RDSHostType_SecretsManager_)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_aws_deployer_v1_rds_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_o5_aws_deployer_v1_rds_proto_goTypes,
		DependencyIndexes: file_o5_aws_deployer_v1_rds_proto_depIdxs,
		MessageInfos:      file_o5_aws_deployer_v1_rds_proto_msgTypes,
	}.Build()
	File_o5_aws_deployer_v1_rds_proto = out.File
	file_o5_aws_deployer_v1_rds_proto_rawDesc = nil
	file_o5_aws_deployer_v1_rds_proto_goTypes = nil
	file_o5_aws_deployer_v1_rds_proto_depIdxs = nil
}
