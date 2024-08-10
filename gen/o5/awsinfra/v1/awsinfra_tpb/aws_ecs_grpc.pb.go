// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: o5/aws/infra/v1/topic/aws_ecs.proto

package awsinfra_tpb

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	ECSRequestTopic_RunECSTask_FullMethodName = "/o5.aws.infra.v1.topic.ECSRequestTopic/RunECSTask"
)

// ECSRequestTopicClient is the client API for ECSRequestTopic service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ECSRequestTopicClient interface {
	RunECSTask(ctx context.Context, in *RunECSTaskMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type eCSRequestTopicClient struct {
	cc grpc.ClientConnInterface
}

func NewECSRequestTopicClient(cc grpc.ClientConnInterface) ECSRequestTopicClient {
	return &eCSRequestTopicClient{cc}
}

func (c *eCSRequestTopicClient) RunECSTask(ctx context.Context, in *RunECSTaskMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, ECSRequestTopic_RunECSTask_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ECSRequestTopicServer is the server API for ECSRequestTopic service.
// All implementations must embed UnimplementedECSRequestTopicServer
// for forward compatibility
type ECSRequestTopicServer interface {
	RunECSTask(context.Context, *RunECSTaskMessage) (*emptypb.Empty, error)
	mustEmbedUnimplementedECSRequestTopicServer()
}

// UnimplementedECSRequestTopicServer must be embedded to have forward compatible implementations.
type UnimplementedECSRequestTopicServer struct {
}

func (UnimplementedECSRequestTopicServer) RunECSTask(context.Context, *RunECSTaskMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RunECSTask not implemented")
}
func (UnimplementedECSRequestTopicServer) mustEmbedUnimplementedECSRequestTopicServer() {}

// UnsafeECSRequestTopicServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ECSRequestTopicServer will
// result in compilation errors.
type UnsafeECSRequestTopicServer interface {
	mustEmbedUnimplementedECSRequestTopicServer()
}

func RegisterECSRequestTopicServer(s grpc.ServiceRegistrar, srv ECSRequestTopicServer) {
	s.RegisterService(&ECSRequestTopic_ServiceDesc, srv)
}

func _ECSRequestTopic_RunECSTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RunECSTaskMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ECSRequestTopicServer).RunECSTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ECSRequestTopic_RunECSTask_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ECSRequestTopicServer).RunECSTask(ctx, req.(*RunECSTaskMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// ECSRequestTopic_ServiceDesc is the grpc.ServiceDesc for ECSRequestTopic service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ECSRequestTopic_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.infra.v1.topic.ECSRequestTopic",
	HandlerType: (*ECSRequestTopicServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RunECSTask",
			Handler:    _ECSRequestTopic_RunECSTask_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/infra/v1/topic/aws_ecs.proto",
}

const (
	ECSReplyTopic_ECSTaskStatus_FullMethodName = "/o5.aws.infra.v1.topic.ECSReplyTopic/ECSTaskStatus"
)

// ECSReplyTopicClient is the client API for ECSReplyTopic service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ECSReplyTopicClient interface {
	ECSTaskStatus(ctx context.Context, in *ECSTaskStatusMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type eCSReplyTopicClient struct {
	cc grpc.ClientConnInterface
}

func NewECSReplyTopicClient(cc grpc.ClientConnInterface) ECSReplyTopicClient {
	return &eCSReplyTopicClient{cc}
}

func (c *eCSReplyTopicClient) ECSTaskStatus(ctx context.Context, in *ECSTaskStatusMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, ECSReplyTopic_ECSTaskStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ECSReplyTopicServer is the server API for ECSReplyTopic service.
// All implementations must embed UnimplementedECSReplyTopicServer
// for forward compatibility
type ECSReplyTopicServer interface {
	ECSTaskStatus(context.Context, *ECSTaskStatusMessage) (*emptypb.Empty, error)
	mustEmbedUnimplementedECSReplyTopicServer()
}

// UnimplementedECSReplyTopicServer must be embedded to have forward compatible implementations.
type UnimplementedECSReplyTopicServer struct {
}

func (UnimplementedECSReplyTopicServer) ECSTaskStatus(context.Context, *ECSTaskStatusMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ECSTaskStatus not implemented")
}
func (UnimplementedECSReplyTopicServer) mustEmbedUnimplementedECSReplyTopicServer() {}

// UnsafeECSReplyTopicServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ECSReplyTopicServer will
// result in compilation errors.
type UnsafeECSReplyTopicServer interface {
	mustEmbedUnimplementedECSReplyTopicServer()
}

func RegisterECSReplyTopicServer(s grpc.ServiceRegistrar, srv ECSReplyTopicServer) {
	s.RegisterService(&ECSReplyTopic_ServiceDesc, srv)
}

func _ECSReplyTopic_ECSTaskStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ECSTaskStatusMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ECSReplyTopicServer).ECSTaskStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ECSReplyTopic_ECSTaskStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ECSReplyTopicServer).ECSTaskStatus(ctx, req.(*ECSTaskStatusMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// ECSReplyTopic_ServiceDesc is the grpc.ServiceDesc for ECSReplyTopic service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ECSReplyTopic_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.infra.v1.topic.ECSReplyTopic",
	HandlerType: (*ECSReplyTopicServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ECSTaskStatus",
			Handler:    _ECSReplyTopic_ECSTaskStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/infra/v1/topic/aws_ecs.proto",
}
