// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: o5/aws/deployer/v1/service/query.proto

package awsdeployer_spb

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

const (
	DeploymentQueryService_GetDeployment_FullMethodName        = "/o5.aws.deployer.v1.service.DeploymentQueryService/GetDeployment"
	DeploymentQueryService_ListDeploymentEvents_FullMethodName = "/o5.aws.deployer.v1.service.DeploymentQueryService/ListDeploymentEvents"
	DeploymentQueryService_ListDeployments_FullMethodName      = "/o5.aws.deployer.v1.service.DeploymentQueryService/ListDeployments"
)

// DeploymentQueryServiceClient is the client API for DeploymentQueryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DeploymentQueryServiceClient interface {
	GetDeployment(ctx context.Context, in *GetDeploymentRequest, opts ...grpc.CallOption) (*GetDeploymentResponse, error)
	ListDeploymentEvents(ctx context.Context, in *ListDeploymentEventsRequest, opts ...grpc.CallOption) (*ListDeploymentEventsResponse, error)
	ListDeployments(ctx context.Context, in *ListDeploymentsRequest, opts ...grpc.CallOption) (*ListDeploymentsResponse, error)
}

type deploymentQueryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDeploymentQueryServiceClient(cc grpc.ClientConnInterface) DeploymentQueryServiceClient {
	return &deploymentQueryServiceClient{cc}
}

func (c *deploymentQueryServiceClient) GetDeployment(ctx context.Context, in *GetDeploymentRequest, opts ...grpc.CallOption) (*GetDeploymentResponse, error) {
	out := new(GetDeploymentResponse)
	err := c.cc.Invoke(ctx, DeploymentQueryService_GetDeployment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deploymentQueryServiceClient) ListDeploymentEvents(ctx context.Context, in *ListDeploymentEventsRequest, opts ...grpc.CallOption) (*ListDeploymentEventsResponse, error) {
	out := new(ListDeploymentEventsResponse)
	err := c.cc.Invoke(ctx, DeploymentQueryService_ListDeploymentEvents_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *deploymentQueryServiceClient) ListDeployments(ctx context.Context, in *ListDeploymentsRequest, opts ...grpc.CallOption) (*ListDeploymentsResponse, error) {
	out := new(ListDeploymentsResponse)
	err := c.cc.Invoke(ctx, DeploymentQueryService_ListDeployments_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DeploymentQueryServiceServer is the server API for DeploymentQueryService service.
// All implementations must embed UnimplementedDeploymentQueryServiceServer
// for forward compatibility
type DeploymentQueryServiceServer interface {
	GetDeployment(context.Context, *GetDeploymentRequest) (*GetDeploymentResponse, error)
	ListDeploymentEvents(context.Context, *ListDeploymentEventsRequest) (*ListDeploymentEventsResponse, error)
	ListDeployments(context.Context, *ListDeploymentsRequest) (*ListDeploymentsResponse, error)
	mustEmbedUnimplementedDeploymentQueryServiceServer()
}

// UnimplementedDeploymentQueryServiceServer must be embedded to have forward compatible implementations.
type UnimplementedDeploymentQueryServiceServer struct {
}

func (UnimplementedDeploymentQueryServiceServer) GetDeployment(context.Context, *GetDeploymentRequest) (*GetDeploymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDeployment not implemented")
}
func (UnimplementedDeploymentQueryServiceServer) ListDeploymentEvents(context.Context, *ListDeploymentEventsRequest) (*ListDeploymentEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListDeploymentEvents not implemented")
}
func (UnimplementedDeploymentQueryServiceServer) ListDeployments(context.Context, *ListDeploymentsRequest) (*ListDeploymentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListDeployments not implemented")
}
func (UnimplementedDeploymentQueryServiceServer) mustEmbedUnimplementedDeploymentQueryServiceServer() {
}

// UnsafeDeploymentQueryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DeploymentQueryServiceServer will
// result in compilation errors.
type UnsafeDeploymentQueryServiceServer interface {
	mustEmbedUnimplementedDeploymentQueryServiceServer()
}

func RegisterDeploymentQueryServiceServer(s grpc.ServiceRegistrar, srv DeploymentQueryServiceServer) {
	s.RegisterService(&DeploymentQueryService_ServiceDesc, srv)
}

func _DeploymentQueryService_GetDeployment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDeploymentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentQueryServiceServer).GetDeployment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeploymentQueryService_GetDeployment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentQueryServiceServer).GetDeployment(ctx, req.(*GetDeploymentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeploymentQueryService_ListDeploymentEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListDeploymentEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentQueryServiceServer).ListDeploymentEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeploymentQueryService_ListDeploymentEvents_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentQueryServiceServer).ListDeploymentEvents(ctx, req.(*ListDeploymentEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DeploymentQueryService_ListDeployments_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListDeploymentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DeploymentQueryServiceServer).ListDeployments(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DeploymentQueryService_ListDeployments_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DeploymentQueryServiceServer).ListDeployments(ctx, req.(*ListDeploymentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DeploymentQueryService_ServiceDesc is the grpc.ServiceDesc for DeploymentQueryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DeploymentQueryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.deployer.v1.service.DeploymentQueryService",
	HandlerType: (*DeploymentQueryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetDeployment",
			Handler:    _DeploymentQueryService_GetDeployment_Handler,
		},
		{
			MethodName: "ListDeploymentEvents",
			Handler:    _DeploymentQueryService_ListDeploymentEvents_Handler,
		},
		{
			MethodName: "ListDeployments",
			Handler:    _DeploymentQueryService_ListDeployments_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/deployer/v1/service/query.proto",
}

const (
	StackQueryService_GetStack_FullMethodName        = "/o5.aws.deployer.v1.service.StackQueryService/GetStack"
	StackQueryService_ListStacks_FullMethodName      = "/o5.aws.deployer.v1.service.StackQueryService/ListStacks"
	StackQueryService_ListStackEvents_FullMethodName = "/o5.aws.deployer.v1.service.StackQueryService/ListStackEvents"
)

// StackQueryServiceClient is the client API for StackQueryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StackQueryServiceClient interface {
	GetStack(ctx context.Context, in *GetStackRequest, opts ...grpc.CallOption) (*GetStackResponse, error)
	ListStacks(ctx context.Context, in *ListStacksRequest, opts ...grpc.CallOption) (*ListStacksResponse, error)
	ListStackEvents(ctx context.Context, in *ListStackEventsRequest, opts ...grpc.CallOption) (*ListStackEventsResponse, error)
}

type stackQueryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewStackQueryServiceClient(cc grpc.ClientConnInterface) StackQueryServiceClient {
	return &stackQueryServiceClient{cc}
}

func (c *stackQueryServiceClient) GetStack(ctx context.Context, in *GetStackRequest, opts ...grpc.CallOption) (*GetStackResponse, error) {
	out := new(GetStackResponse)
	err := c.cc.Invoke(ctx, StackQueryService_GetStack_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *stackQueryServiceClient) ListStacks(ctx context.Context, in *ListStacksRequest, opts ...grpc.CallOption) (*ListStacksResponse, error) {
	out := new(ListStacksResponse)
	err := c.cc.Invoke(ctx, StackQueryService_ListStacks_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *stackQueryServiceClient) ListStackEvents(ctx context.Context, in *ListStackEventsRequest, opts ...grpc.CallOption) (*ListStackEventsResponse, error) {
	out := new(ListStackEventsResponse)
	err := c.cc.Invoke(ctx, StackQueryService_ListStackEvents_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// StackQueryServiceServer is the server API for StackQueryService service.
// All implementations must embed UnimplementedStackQueryServiceServer
// for forward compatibility
type StackQueryServiceServer interface {
	GetStack(context.Context, *GetStackRequest) (*GetStackResponse, error)
	ListStacks(context.Context, *ListStacksRequest) (*ListStacksResponse, error)
	ListStackEvents(context.Context, *ListStackEventsRequest) (*ListStackEventsResponse, error)
	mustEmbedUnimplementedStackQueryServiceServer()
}

// UnimplementedStackQueryServiceServer must be embedded to have forward compatible implementations.
type UnimplementedStackQueryServiceServer struct {
}

func (UnimplementedStackQueryServiceServer) GetStack(context.Context, *GetStackRequest) (*GetStackResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStack not implemented")
}
func (UnimplementedStackQueryServiceServer) ListStacks(context.Context, *ListStacksRequest) (*ListStacksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStacks not implemented")
}
func (UnimplementedStackQueryServiceServer) ListStackEvents(context.Context, *ListStackEventsRequest) (*ListStackEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListStackEvents not implemented")
}
func (UnimplementedStackQueryServiceServer) mustEmbedUnimplementedStackQueryServiceServer() {}

// UnsafeStackQueryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StackQueryServiceServer will
// result in compilation errors.
type UnsafeStackQueryServiceServer interface {
	mustEmbedUnimplementedStackQueryServiceServer()
}

func RegisterStackQueryServiceServer(s grpc.ServiceRegistrar, srv StackQueryServiceServer) {
	s.RegisterService(&StackQueryService_ServiceDesc, srv)
}

func _StackQueryService_GetStack_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStackRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StackQueryServiceServer).GetStack(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StackQueryService_GetStack_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StackQueryServiceServer).GetStack(ctx, req.(*GetStackRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StackQueryService_ListStacks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListStacksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StackQueryServiceServer).ListStacks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StackQueryService_ListStacks_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StackQueryServiceServer).ListStacks(ctx, req.(*ListStacksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _StackQueryService_ListStackEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListStackEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StackQueryServiceServer).ListStackEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: StackQueryService_ListStackEvents_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StackQueryServiceServer).ListStackEvents(ctx, req.(*ListStackEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// StackQueryService_ServiceDesc is the grpc.ServiceDesc for StackQueryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var StackQueryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.deployer.v1.service.StackQueryService",
	HandlerType: (*StackQueryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetStack",
			Handler:    _StackQueryService_GetStack_Handler,
		},
		{
			MethodName: "ListStacks",
			Handler:    _StackQueryService_ListStacks_Handler,
		},
		{
			MethodName: "ListStackEvents",
			Handler:    _StackQueryService_ListStackEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/deployer/v1/service/query.proto",
}

const (
	EnvironmentQueryService_ListEnvironments_FullMethodName      = "/o5.aws.deployer.v1.service.EnvironmentQueryService/ListEnvironments"
	EnvironmentQueryService_GetEnvironment_FullMethodName        = "/o5.aws.deployer.v1.service.EnvironmentQueryService/GetEnvironment"
	EnvironmentQueryService_ListEnvironmentEvents_FullMethodName = "/o5.aws.deployer.v1.service.EnvironmentQueryService/ListEnvironmentEvents"
)

// EnvironmentQueryServiceClient is the client API for EnvironmentQueryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type EnvironmentQueryServiceClient interface {
	ListEnvironments(ctx context.Context, in *ListEnvironmentsRequest, opts ...grpc.CallOption) (*ListEnvironmentsResponse, error)
	GetEnvironment(ctx context.Context, in *GetEnvironmentRequest, opts ...grpc.CallOption) (*GetEnvironmentResponse, error)
	ListEnvironmentEvents(ctx context.Context, in *ListEnvironmentEventsRequest, opts ...grpc.CallOption) (*ListEnvironmentEventsResponse, error)
}

type environmentQueryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewEnvironmentQueryServiceClient(cc grpc.ClientConnInterface) EnvironmentQueryServiceClient {
	return &environmentQueryServiceClient{cc}
}

func (c *environmentQueryServiceClient) ListEnvironments(ctx context.Context, in *ListEnvironmentsRequest, opts ...grpc.CallOption) (*ListEnvironmentsResponse, error) {
	out := new(ListEnvironmentsResponse)
	err := c.cc.Invoke(ctx, EnvironmentQueryService_ListEnvironments_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentQueryServiceClient) GetEnvironment(ctx context.Context, in *GetEnvironmentRequest, opts ...grpc.CallOption) (*GetEnvironmentResponse, error) {
	out := new(GetEnvironmentResponse)
	err := c.cc.Invoke(ctx, EnvironmentQueryService_GetEnvironment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *environmentQueryServiceClient) ListEnvironmentEvents(ctx context.Context, in *ListEnvironmentEventsRequest, opts ...grpc.CallOption) (*ListEnvironmentEventsResponse, error) {
	out := new(ListEnvironmentEventsResponse)
	err := c.cc.Invoke(ctx, EnvironmentQueryService_ListEnvironmentEvents_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EnvironmentQueryServiceServer is the server API for EnvironmentQueryService service.
// All implementations must embed UnimplementedEnvironmentQueryServiceServer
// for forward compatibility
type EnvironmentQueryServiceServer interface {
	ListEnvironments(context.Context, *ListEnvironmentsRequest) (*ListEnvironmentsResponse, error)
	GetEnvironment(context.Context, *GetEnvironmentRequest) (*GetEnvironmentResponse, error)
	ListEnvironmentEvents(context.Context, *ListEnvironmentEventsRequest) (*ListEnvironmentEventsResponse, error)
	mustEmbedUnimplementedEnvironmentQueryServiceServer()
}

// UnimplementedEnvironmentQueryServiceServer must be embedded to have forward compatible implementations.
type UnimplementedEnvironmentQueryServiceServer struct {
}

func (UnimplementedEnvironmentQueryServiceServer) ListEnvironments(context.Context, *ListEnvironmentsRequest) (*ListEnvironmentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListEnvironments not implemented")
}
func (UnimplementedEnvironmentQueryServiceServer) GetEnvironment(context.Context, *GetEnvironmentRequest) (*GetEnvironmentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetEnvironment not implemented")
}
func (UnimplementedEnvironmentQueryServiceServer) ListEnvironmentEvents(context.Context, *ListEnvironmentEventsRequest) (*ListEnvironmentEventsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListEnvironmentEvents not implemented")
}
func (UnimplementedEnvironmentQueryServiceServer) mustEmbedUnimplementedEnvironmentQueryServiceServer() {
}

// UnsafeEnvironmentQueryServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to EnvironmentQueryServiceServer will
// result in compilation errors.
type UnsafeEnvironmentQueryServiceServer interface {
	mustEmbedUnimplementedEnvironmentQueryServiceServer()
}

func RegisterEnvironmentQueryServiceServer(s grpc.ServiceRegistrar, srv EnvironmentQueryServiceServer) {
	s.RegisterService(&EnvironmentQueryService_ServiceDesc, srv)
}

func _EnvironmentQueryService_ListEnvironments_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListEnvironmentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentQueryServiceServer).ListEnvironments(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentQueryService_ListEnvironments_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentQueryServiceServer).ListEnvironments(ctx, req.(*ListEnvironmentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentQueryService_GetEnvironment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetEnvironmentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentQueryServiceServer).GetEnvironment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentQueryService_GetEnvironment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentQueryServiceServer).GetEnvironment(ctx, req.(*GetEnvironmentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EnvironmentQueryService_ListEnvironmentEvents_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListEnvironmentEventsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EnvironmentQueryServiceServer).ListEnvironmentEvents(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: EnvironmentQueryService_ListEnvironmentEvents_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EnvironmentQueryServiceServer).ListEnvironmentEvents(ctx, req.(*ListEnvironmentEventsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// EnvironmentQueryService_ServiceDesc is the grpc.ServiceDesc for EnvironmentQueryService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var EnvironmentQueryService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.deployer.v1.service.EnvironmentQueryService",
	HandlerType: (*EnvironmentQueryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListEnvironments",
			Handler:    _EnvironmentQueryService_ListEnvironments_Handler,
		},
		{
			MethodName: "GetEnvironment",
			Handler:    _EnvironmentQueryService_GetEnvironment_Handler,
		},
		{
			MethodName: "ListEnvironmentEvents",
			Handler:    _EnvironmentQueryService_ListEnvironmentEvents_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/deployer/v1/service/query.proto",
}
