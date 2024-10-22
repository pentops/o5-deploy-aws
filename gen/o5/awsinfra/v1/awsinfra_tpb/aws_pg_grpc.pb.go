// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: o5/aws/infra/v1/topic/aws_pg.proto

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
	PostgresRequestTopic_UpsertPostgresDatabase_FullMethodName  = "/o5.aws.infra.v1.topic.PostgresRequestTopic/UpsertPostgresDatabase"
	PostgresRequestTopic_CleanupPostgresDatabase_FullMethodName = "/o5.aws.infra.v1.topic.PostgresRequestTopic/CleanupPostgresDatabase"
)

// PostgresRequestTopicClient is the client API for PostgresRequestTopic service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PostgresRequestTopicClient interface {
	UpsertPostgresDatabase(ctx context.Context, in *UpsertPostgresDatabaseMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
	// rpc MigratePostgresDatabase(MigratePostgresDatabaseMessage) returns (google.protobuf.Empty) {}
	CleanupPostgresDatabase(ctx context.Context, in *CleanupPostgresDatabaseMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type postgresRequestTopicClient struct {
	cc grpc.ClientConnInterface
}

func NewPostgresRequestTopicClient(cc grpc.ClientConnInterface) PostgresRequestTopicClient {
	return &postgresRequestTopicClient{cc}
}

func (c *postgresRequestTopicClient) UpsertPostgresDatabase(ctx context.Context, in *UpsertPostgresDatabaseMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, PostgresRequestTopic_UpsertPostgresDatabase_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *postgresRequestTopicClient) CleanupPostgresDatabase(ctx context.Context, in *CleanupPostgresDatabaseMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, PostgresRequestTopic_CleanupPostgresDatabase_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PostgresRequestTopicServer is the server API for PostgresRequestTopic service.
// All implementations must embed UnimplementedPostgresRequestTopicServer
// for forward compatibility
type PostgresRequestTopicServer interface {
	UpsertPostgresDatabase(context.Context, *UpsertPostgresDatabaseMessage) (*emptypb.Empty, error)
	// rpc MigratePostgresDatabase(MigratePostgresDatabaseMessage) returns (google.protobuf.Empty) {}
	CleanupPostgresDatabase(context.Context, *CleanupPostgresDatabaseMessage) (*emptypb.Empty, error)
	mustEmbedUnimplementedPostgresRequestTopicServer()
}

// UnimplementedPostgresRequestTopicServer must be embedded to have forward compatible implementations.
type UnimplementedPostgresRequestTopicServer struct {
}

func (UnimplementedPostgresRequestTopicServer) UpsertPostgresDatabase(context.Context, *UpsertPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpsertPostgresDatabase not implemented")
}
func (UnimplementedPostgresRequestTopicServer) CleanupPostgresDatabase(context.Context, *CleanupPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CleanupPostgresDatabase not implemented")
}
func (UnimplementedPostgresRequestTopicServer) mustEmbedUnimplementedPostgresRequestTopicServer() {}

// UnsafePostgresRequestTopicServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PostgresRequestTopicServer will
// result in compilation errors.
type UnsafePostgresRequestTopicServer interface {
	mustEmbedUnimplementedPostgresRequestTopicServer()
}

func RegisterPostgresRequestTopicServer(s grpc.ServiceRegistrar, srv PostgresRequestTopicServer) {
	s.RegisterService(&PostgresRequestTopic_ServiceDesc, srv)
}

func _PostgresRequestTopic_UpsertPostgresDatabase_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpsertPostgresDatabaseMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostgresRequestTopicServer).UpsertPostgresDatabase(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostgresRequestTopic_UpsertPostgresDatabase_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostgresRequestTopicServer).UpsertPostgresDatabase(ctx, req.(*UpsertPostgresDatabaseMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _PostgresRequestTopic_CleanupPostgresDatabase_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CleanupPostgresDatabaseMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostgresRequestTopicServer).CleanupPostgresDatabase(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostgresRequestTopic_CleanupPostgresDatabase_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostgresRequestTopicServer).CleanupPostgresDatabase(ctx, req.(*CleanupPostgresDatabaseMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// PostgresRequestTopic_ServiceDesc is the grpc.ServiceDesc for PostgresRequestTopic service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PostgresRequestTopic_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.infra.v1.topic.PostgresRequestTopic",
	HandlerType: (*PostgresRequestTopicServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpsertPostgresDatabase",
			Handler:    _PostgresRequestTopic_UpsertPostgresDatabase_Handler,
		},
		{
			MethodName: "CleanupPostgresDatabase",
			Handler:    _PostgresRequestTopic_CleanupPostgresDatabase_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/infra/v1/topic/aws_pg.proto",
}

const (
	PostgresReplyTopic_PostgresDatabaseStatus_FullMethodName = "/o5.aws.infra.v1.topic.PostgresReplyTopic/PostgresDatabaseStatus"
)

// PostgresReplyTopicClient is the client API for PostgresReplyTopic service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PostgresReplyTopicClient interface {
	PostgresDatabaseStatus(ctx context.Context, in *PostgresDatabaseStatusMessage, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type postgresReplyTopicClient struct {
	cc grpc.ClientConnInterface
}

func NewPostgresReplyTopicClient(cc grpc.ClientConnInterface) PostgresReplyTopicClient {
	return &postgresReplyTopicClient{cc}
}

func (c *postgresReplyTopicClient) PostgresDatabaseStatus(ctx context.Context, in *PostgresDatabaseStatusMessage, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, PostgresReplyTopic_PostgresDatabaseStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PostgresReplyTopicServer is the server API for PostgresReplyTopic service.
// All implementations must embed UnimplementedPostgresReplyTopicServer
// for forward compatibility
type PostgresReplyTopicServer interface {
	PostgresDatabaseStatus(context.Context, *PostgresDatabaseStatusMessage) (*emptypb.Empty, error)
	mustEmbedUnimplementedPostgresReplyTopicServer()
}

// UnimplementedPostgresReplyTopicServer must be embedded to have forward compatible implementations.
type UnimplementedPostgresReplyTopicServer struct {
}

func (UnimplementedPostgresReplyTopicServer) PostgresDatabaseStatus(context.Context, *PostgresDatabaseStatusMessage) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostgresDatabaseStatus not implemented")
}
func (UnimplementedPostgresReplyTopicServer) mustEmbedUnimplementedPostgresReplyTopicServer() {}

// UnsafePostgresReplyTopicServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PostgresReplyTopicServer will
// result in compilation errors.
type UnsafePostgresReplyTopicServer interface {
	mustEmbedUnimplementedPostgresReplyTopicServer()
}

func RegisterPostgresReplyTopicServer(s grpc.ServiceRegistrar, srv PostgresReplyTopicServer) {
	s.RegisterService(&PostgresReplyTopic_ServiceDesc, srv)
}

func _PostgresReplyTopic_PostgresDatabaseStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PostgresDatabaseStatusMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PostgresReplyTopicServer).PostgresDatabaseStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PostgresReplyTopic_PostgresDatabaseStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PostgresReplyTopicServer).PostgresDatabaseStatus(ctx, req.(*PostgresDatabaseStatusMessage))
	}
	return interceptor(ctx, in, info, handler)
}

// PostgresReplyTopic_ServiceDesc is the grpc.ServiceDesc for PostgresReplyTopic service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PostgresReplyTopic_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "o5.aws.infra.v1.topic.PostgresReplyTopic",
	HandlerType: (*PostgresReplyTopicServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PostgresDatabaseStatus",
			Handler:    _PostgresReplyTopic_PostgresDatabaseStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "o5/aws/infra/v1/topic/aws_pg.proto",
}
