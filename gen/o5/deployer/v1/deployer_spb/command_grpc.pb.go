// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: o5/deployer/v1/service/command.proto

package deployer_spb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DeploymentCommandServiceClient is the client API for DeploymentCommandService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeploymentCommandServiceClient interface {
	TriggerDeployment(ctx context.Context, in *TriggerDeploymentRequest, opts ...grpc.CallOption) (*TriggerDeploymentResponse, error)
	TerminateDeployment(ctx context.Context, in *TerminateDeploymentRequest, opts ...grpc.CallOption) (*TerminateDeploymentResponse, error)
	UpsertCluster(ctx context.Context, in *UpsertClusterRequest, opts ...grpc.CallOption) (*UpsertClusterResponse, error)
	UpsertEnvironment(ctx context.Context, in *UpsertEnvironmentRequest, opts ...grpc.CallOption) (*UpsertEnvironmentResponse, error)
	UpsertStack(ctx context.Context, in *UpsertStackRequest, opts ...grpc.CallOption) (*UpsertStackResponse, error)
}

type deploymentCommandServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDeploymentCommandServiceClient(cc grpc.ClientConnInterface) DeploymentCommandServiceClient {
	return &deploymentCommandServiceClient{cc}
}

func (c *deploymentCommandServiceClient) TriggerDeployment(ctx context.Context, in *TriggerDeploymentRequest, opts ...grpc.CallOption) (*TriggerDeploymentResponse, error) {
	out := new(TriggerDeploymentResponse)
	err := c.cc.Invoke(ctx, "/o5.deployer.v1.service.DeploymentCommandService/TriggerDeployment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deploymentCommandServiceClient) TerminateDeployment(ctx context.Context, in *TerminateDeploymentRequest, opts ...grpc.CallOption) (*TerminateDeploymentResponse, error) {
	out := new(TerminateDeploymentResponse)
	err := c.cc.Invoke(ctx, "/o5.deployer.v1.service.DeploymentCommandService/TerminateDeployment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deploymentCommandServiceClient) UpsertCluster(ctx context.Context, in *UpsertClusterRequest, opts ...grpc.CallOption) (*UpsertClusterResponse, error) {
	out := new(UpsertClusterResponse)
	err := c.cc.Invoke(ctx, "/o5.deployer.v1.service.DeploymentCommandService/UpsertCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deploymentCommandServiceClient) UpsertEnvironment(ctx context.Context, in *UpsertEnvironmentRequest, opts ...grpc.CallOption) (*UpsertEnvironmentResponse, error) {
	out := new(UpsertEnvironmentResponse)
	err := c.cc.Invoke(ctx, "/o5.deployer.v1.service.DeploymentCommandService/UpsertEnvironment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deploymentCommandServiceClient) UpsertStack(ctx context.Context, in *UpsertStackRequest, opts ...grpc.CallOption) (*UpsertStackResponse, error) {
	out := new(UpsertStackResponse)
	err := c.cc.Invoke(ctx, "/o5.deployer.v1.service.DeploymentCommandService/UpsertStack", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeploymentCommandServiceServer is the server API for DeploymentCommandService service.
// All implementations must embed UnimplementedDeploymentCommandServiceServer
// for forward compatibility
type DeploymentCommandServiceServer interface {
	TriggerDeployment(context.Context, *TriggerDeploymentRequest) (*TriggerDeploymentResponse, error)
	TerminateDeployment(context.Context, *TerminateDeploymentRequest) (*TerminateDeploymentResponse, error)
	UpsertCluster(context.Context, *UpsertClusterRequest) (*UpsertClusterResponse, error)
	UpsertEnvironment(context.Context, *UpsertEnvironmentRequest) (*UpsertEnvironmentResponse, error)
	UpsertStack(context.Context, *UpsertStackRequest) (*UpsertStackResponse, error)
	mustEmbedUnimplementedDeploymentCommandServiceServer()
}

// UnimplementedDeploymentCommandServiceServer must be embedded to have forward compatible implementations.
type UnimplementedDeploymentCommandServiceServer struct {
}

func (UnimplementedDeploymentCommandServiceServer) TriggerDeployment(context.Context, *TriggerDeploymentRequest) (*TriggerDeploymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TriggerDeployment not implemented")
}
func (UnimplementedDeploymentCommandServiceServer) TerminateDeployment(context.Context, *TerminateDeploymentRequest) (*TerminateDeploymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TerminateDeployment not implemented")
}
func (UnimplementedDeploymentCommandServiceServer) UpsertCluster(context.Context, *UpsertClusterRequest) (*UpsertClusterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpsertCluster not implemented")
}
func (UnimplementedDeploymentCommandServiceServer) UpsertEnvironment(context.Context, *UpsertEnvironmentRequest) (*UpsertEnvironmentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpsertEnvironment not implemented")
}
func (UnimplementedDeploymentCommandServiceServer) UpsertStack(context.Context, *UpsertStackRequest) (*UpsertStackResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpsertStack not implemented")
}
func (UnimplementedDeploymentCommandServiceServer) mustEmbedUnimplementedDeploymentCommandServiceServer() {
}

// UnsafeDeploymentCommandServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeploymentCommandServiceServer will
// result in compilation errors.
type UnsafeDeploymentCommandServiceServer interface {
	mustEmbedUnimplementedDeploymentCommandServiceServer()
}

func RegisterDeploymentCommandServiceServer(s grpc.ServiceRegistrar, srv DeploymentCommandServiceServer) {
	s.RegisterService(&DeploymentCommandService_ServiceDesc, srv)
}

func _DeploymentCommandService_TriggerDeployment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TriggerDeploymentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentCommandServiceServer).TriggerDeployment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.deployer.v1.service.DeploymentCommandService/TriggerDeployment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentCommandServiceServer).TriggerDeployment(ctx, req.(*TriggerDeploymentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeploymentCommandService_TerminateDeployment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TerminateDeploymentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentCommandServiceServer).TerminateDeployment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.deployer.v1.service.DeploymentCommandService/TerminateDeployment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentCommandServiceServer).TerminateDeployment(ctx, req.(*TerminateDeploymentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeploymentCommandService_UpsertCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpsertClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentCommandServiceServer).UpsertCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.deployer.v1.service.DeploymentCommandService/UpsertCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentCommandServiceServer).UpsertCluster(ctx, req.(*UpsertClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeploymentCommandService_UpsertEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpsertEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentCommandServiceServer).UpsertEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.deployer.v1.service.DeploymentCommandService/UpsertEnvironment",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentCommandServiceServer).UpsertEnvironment(ctx, req.(*UpsertEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeploymentCommandService_UpsertStack_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpsertStackRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentCommandServiceServer).UpsertStack(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/o5.deployer.v1.service.DeploymentCommandService/UpsertStack",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentCommandServiceServer).UpsertStack(ctx, req.(*UpsertStackRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DeploymentCommandService_ServiceDesc is the grpc.ServiceDesc for DeploymentCommandService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeploymentCommandService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.deployer.v1.service.DeploymentCommandService",
	HandlerType: (*DeploymentCommandServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "TriggerDeployment",
			Handler:    _DeploymentCommandService_TriggerDeployment_Handler,
		},
		{
			MethodName: "TerminateDeployment",
			Handler:    _DeploymentCommandService_TerminateDeployment_Handler,
		},
		{
			MethodName: "UpsertCluster",
			Handler:    _DeploymentCommandService_UpsertCluster_Handler,
		},
		{
			MethodName: "UpsertEnvironment",
			Handler:    _DeploymentCommandService_UpsertEnvironment_Handler,
		},
		{
			MethodName: "UpsertStack",
			Handler:    _DeploymentCommandService_UpsertStack_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/deployer/v1/service/command.proto",
}