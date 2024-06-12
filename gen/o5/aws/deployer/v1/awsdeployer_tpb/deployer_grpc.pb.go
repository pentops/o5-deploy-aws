// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: o5/aws/deployer/v1/topic/deployer.proto

package awsdeployer_tpb

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

// DeploymentRequestTopicClient is the client API for DeploymentRequestTopic service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeploymentRequestTopicClient interface {
	RequestDeployment(ctx context.Context, in *RequestDeploymentMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type deploymentRequestTopicClient struct {
	cc grpc.ClientConnInterface
}

func NewDeploymentRequestTopicClient(cc grpc.ClientConnInterface) DeploymentRequestTopicClient {
	return &deploymentRequestTopicClient{cc}
}

func (c *deploymentRequestTopicClient) RequestDeployment(ctx context.Context, in *RequestDeploymentMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/o5.aws.deployer.v1.topic.DeploymentRequestTopic/RequestDeployment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeploymentRequestTopicServer is the server API for DeploymentRequestTopic service.
// All implementations must embed UnimplementedDeploymentRequestTopicServer
// for forward compatibility
type DeploymentRequestTopicServer interface {
	RequestDeployment(context.Context, *RequestDeploymentMessage) (*emptypb.Empty, error)
	mustEmbedUnimplementedDeploymentRequestTopicServer()
}

// UnimplementedDeploymentRequestTopicServer must be embedded to have forward compatible implementations.
type UnimplementedDeploymentRequestTopicServer struct {
}

func (UnimplementedDeploymentRequestTopicServer) RequestDeployment(context.Context, *RequestDeploymentMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RequestDeployment not implemented")
}
func (UnimplementedDeploymentRequestTopicServer) mustEmbedUnimplementedDeploymentRequestTopicServer() {
}

// UnsafeDeploymentRequestTopicServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeploymentRequestTopicServer will
// result in compilation errors.
type UnsafeDeploymentRequestTopicServer interface {
	mustEmbedUnimplementedDeploymentRequestTopicServer()
}

func RegisterDeploymentRequestTopicServer(s grpc.ServiceRegistrar, srv DeploymentRequestTopicServer) {
	s.RegisterService(&DeploymentRequestTopic_ServiceDesc, srv)
}

func _DeploymentRequestTopic_RequestDeployment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestDeploymentMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentRequestTopicServer).RequestDeployment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.aws.deployer.v1.topic.DeploymentRequestTopic/RequestDeployment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentRequestTopicServer).RequestDeployment(ctx, req.(*RequestDeploymentMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// DeploymentRequestTopic_ServiceDesc is the grpc.ServiceDesc for DeploymentRequestTopic service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeploymentRequestTopic_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.deployer.v1.topic.DeploymentRequestTopic",
	HandlerType: (*DeploymentRequestTopicServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RequestDeployment",
			Handler:    _DeploymentRequestTopic_RequestDeployment_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/deployer/v1/topic/deployer.proto",
}

// DeploymentReplyTopicClient is the client API for DeploymentReplyTopic service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeploymentReplyTopicClient interface {
	DeploymentStatus(ctx context.Context, in *DeploymentStatusMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type deploymentReplyTopicClient struct {
	cc grpc.ClientConnInterface
}

func NewDeploymentReplyTopicClient(cc grpc.ClientConnInterface) DeploymentReplyTopicClient {
	return &deploymentReplyTopicClient{cc}
}

func (c *deploymentReplyTopicClient) DeploymentStatus(ctx context.Context, in *DeploymentStatusMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, "/o5.aws.deployer.v1.topic.DeploymentReplyTopic/DeploymentStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeploymentReplyTopicServer is the server API for DeploymentReplyTopic service.
// All implementations must embed UnimplementedDeploymentReplyTopicServer
// for forward compatibility
type DeploymentReplyTopicServer interface {
	DeploymentStatus(context.Context, *DeploymentStatusMessage) (*emptypb.Empty, error)
	mustEmbedUnimplementedDeploymentReplyTopicServer()
}

// UnimplementedDeploymentReplyTopicServer must be embedded to have forward compatible implementations.
type UnimplementedDeploymentReplyTopicServer struct {
}

func (UnimplementedDeploymentReplyTopicServer) DeploymentStatus(context.Context, *DeploymentStatusMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeploymentStatus not implemented")
}
func (UnimplementedDeploymentReplyTopicServer) mustEmbedUnimplementedDeploymentReplyTopicServer() {}

// UnsafeDeploymentReplyTopicServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeploymentReplyTopicServer will
// result in compilation errors.
type UnsafeDeploymentReplyTopicServer interface {
	mustEmbedUnimplementedDeploymentReplyTopicServer()
}

func RegisterDeploymentReplyTopicServer(s grpc.ServiceRegistrar, srv DeploymentReplyTopicServer) {
	s.RegisterService(&DeploymentReplyTopic_ServiceDesc, srv)
}

func _DeploymentReplyTopic_DeploymentStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeploymentStatusMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentReplyTopicServer).DeploymentStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.aws.deployer.v1.topic.DeploymentReplyTopic/DeploymentStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentReplyTopicServer).DeploymentStatus(ctx, req.(*DeploymentStatusMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// DeploymentReplyTopic_ServiceDesc is the grpc.ServiceDesc for DeploymentReplyTopic service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeploymentReplyTopic_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.deployer.v1.topic.DeploymentReplyTopic",
	HandlerType: (*DeploymentReplyTopicServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "DeploymentStatus",
			Handler:    _DeploymentReplyTopic_DeploymentStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/deployer/v1/topic/deployer.proto",
}
