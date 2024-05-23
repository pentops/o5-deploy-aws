// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: o5/deployer/v1/config/stack_config.proto

package deployer_cpb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	deployer_pb "github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
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

type StackConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Stacks []*Stack `protobuf:"bytes,1,rep,name=stacks,proto3" json:"stacks,omitempty"`
}

func (x *StackConfig) Reset() {
	*x = StackConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_deployer_v1_config_stack_config_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StackConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StackConfig) ProtoMessage() {}

func (x *StackConfig) ProtoReflect() protoreflect.Message {
	mi := &file_o5_deployer_v1_config_stack_config_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StackConfig.ProtoReflect.Descriptor instead.
func (*StackConfig) Descriptor() ([]byte, []int) {
	return file_o5_deployer_v1_config_stack_config_proto_rawDescGZIP(), []int{0}
}

func (x *StackConfig) GetStacks() []*Stack {
	if x != nil {
		return x.Stacks
	}
	return nil
}

type Stack struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Application string                   `protobuf:"bytes,1,opt,name=application,proto3" json:"application,omitempty"`
	Environment string                   `protobuf:"bytes,2,opt,name=environment,proto3" json:"environment,omitempty"`
	Config      *deployer_pb.StackConfig `protobuf:"bytes,3,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *Stack) Reset() {
	*x = Stack{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_deployer_v1_config_stack_config_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Stack) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Stack) ProtoMessage() {}

func (x *Stack) ProtoReflect() protoreflect.Message {
	mi := &file_o5_deployer_v1_config_stack_config_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Stack.ProtoReflect.Descriptor instead.
func (*Stack) Descriptor() ([]byte, []int) {
	return file_o5_deployer_v1_config_stack_config_proto_rawDescGZIP(), []int{1}
}

func (x *Stack) GetApplication() string {
	if x != nil {
		return x.Application
	}
	return ""
}

func (x *Stack) GetEnvironment() string {
	if x != nil {
		return x.Environment
	}
	return ""
}

func (x *Stack) GetConfig() *deployer_pb.StackConfig {
	if x != nil {
		return x.Config
	}
	return nil
}

var File_o5_deployer_v1_config_stack_config_proto protoreflect.FileDescriptor

var file_o5_deployer_v1_config_stack_config_proto_rawDesc = []byte{
	0x0a, 0x28, 0x6f, 0x35, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31,
	0x2f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x2f, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x5f, 0x63, 0x6f,
	0x6e, 0x66, 0x69, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x15, 0x6f, 0x35, 0x2e, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f,
	0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1a,
	0x6f, 0x35, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x73,
	0x74, 0x61, 0x63, 0x6b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x43, 0x0a, 0x0b, 0x53, 0x74,
	0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x34, 0x0a, 0x06, 0x73, 0x74, 0x61,
	0x63, 0x6b, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x6f, 0x35, 0x2e, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x63, 0x6f, 0x6e, 0x66, 0x69,
	0x67, 0x2e, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x52, 0x06, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x73, 0x22,
	0x98, 0x01, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x63, 0x6b, 0x12, 0x28, 0x0a, 0x0b, 0x61, 0x70, 0x70,
	0x6c, 0x69, 0x63, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x06,
	0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x0b, 0x61, 0x70, 0x70, 0x6c, 0x69, 0x63, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x28, 0x0a, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65,
	0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8, 0x01, 0x01,
	0x52, 0x0b, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x12, 0x3b, 0x0a,
	0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1b, 0x2e,
	0x6f, 0x35, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x53,
	0x74, 0x61, 0x63, 0x6b, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x42, 0x06, 0xba, 0x48, 0x03, 0xc8,
	0x01, 0x01, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x42, 0x42, 0x5a, 0x40, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73,
	0x2f, 0x6f, 0x35, 0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67,
	0x65, 0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76,
	0x31, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f, 0x63, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_o5_deployer_v1_config_stack_config_proto_rawDescOnce sync.Once
	file_o5_deployer_v1_config_stack_config_proto_rawDescData = file_o5_deployer_v1_config_stack_config_proto_rawDesc
)

func file_o5_deployer_v1_config_stack_config_proto_rawDescGZIP() []byte {
	file_o5_deployer_v1_config_stack_config_proto_rawDescOnce.Do(func() {
		file_o5_deployer_v1_config_stack_config_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_deployer_v1_config_stack_config_proto_rawDescData)
	})
	return file_o5_deployer_v1_config_stack_config_proto_rawDescData
}

var file_o5_deployer_v1_config_stack_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_o5_deployer_v1_config_stack_config_proto_goTypes = []interface{}{
	(*StackConfig)(nil),             // 0: o5.deployer.v1.config.StackConfig
	(*Stack)(nil),                   // 1: o5.deployer.v1.config.Stack
	(*deployer_pb.StackConfig)(nil), // 2: o5.deployer.v1.StackConfig
}
var file_o5_deployer_v1_config_stack_config_proto_depIdxs = []int32{
	1, // 0: o5.deployer.v1.config.StackConfig.stacks:type_name -> o5.deployer.v1.config.Stack
	2, // 1: o5.deployer.v1.config.Stack.config:type_name -> o5.deployer.v1.StackConfig
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_o5_deployer_v1_config_stack_config_proto_init() }
func file_o5_deployer_v1_config_stack_config_proto_init() {
	if File_o5_deployer_v1_config_stack_config_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_deployer_v1_config_stack_config_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StackConfig); i {
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
		file_o5_deployer_v1_config_stack_config_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Stack); i {
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
			RawDescriptor: file_o5_deployer_v1_config_stack_config_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_o5_deployer_v1_config_stack_config_proto_goTypes,
		DependencyIndexes: file_o5_deployer_v1_config_stack_config_proto_depIdxs,
		MessageInfos:      file_o5_deployer_v1_config_stack_config_proto_msgTypes,
	}.Build()
	File_o5_deployer_v1_config_stack_config_proto = out.File
	file_o5_deployer_v1_config_stack_config_proto_rawDesc = nil
	file_o5_deployer_v1_config_stack_config_proto_goTypes = nil
	file_o5_deployer_v1_config_stack_config_proto_depIdxs = nil
}