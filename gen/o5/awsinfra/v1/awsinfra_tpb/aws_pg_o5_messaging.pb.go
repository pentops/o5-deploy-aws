// Code generated by protoc-gen-go-o5-messaging. DO NOT EDIT.
// versions:
// - protoc-gen-go-o5-messaging 0.0.0
// source: o5/awsinfra/v1/topic/aws_pg.proto

package awsinfra_tpb

import (
	context "context"
	messaging_pb "github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	o5msg "github.com/pentops/o5-messaging/o5msg"
)

// Service: PostgresRequestTopic
type PostgresRequestTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewPostgresRequestTopicTxSender[C any](sender o5msg.TxSender[C]) *PostgresRequestTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.PostgresRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "UpsertPostgresDatabase",
				Message: (*UpsertPostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "MigratePostgresDatabase",
				Message: (*MigratePostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CleanupPostgresDatabase",
				Message: (*CleanupPostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &PostgresRequestTopicTxSender[C]{sender: sender}
}

type PostgresRequestTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewPostgresRequestTopicCollector[C any](collector o5msg.Collector[C]) *PostgresRequestTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.PostgresRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "UpsertPostgresDatabase",
				Message: (*UpsertPostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "MigratePostgresDatabase",
				Message: (*MigratePostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CleanupPostgresDatabase",
				Message: (*CleanupPostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &PostgresRequestTopicCollector[C]{collector: collector}
}

type PostgresRequestTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewPostgresRequestTopicPublisher(publisher o5msg.Publisher) *PostgresRequestTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.PostgresRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "UpsertPostgresDatabase",
				Message: (*UpsertPostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "MigratePostgresDatabase",
				Message: (*MigratePostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CleanupPostgresDatabase",
				Message: (*CleanupPostgresDatabaseMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &PostgresRequestTopicPublisher{publisher: publisher}
}

// Method: UpsertPostgresDatabase

func (msg *UpsertPostgresDatabaseMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.awsinfra.v1.topic.PostgresRequestTopic",
		GrpcMethod:       "UpsertPostgresDatabase",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send PostgresRequestTopicTxSender[C]) UpsertPostgresDatabase(ctx context.Context, sendContext C, msg *UpsertPostgresDatabaseMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect PostgresRequestTopicCollector[C]) UpsertPostgresDatabase(sendContext C, msg *UpsertPostgresDatabaseMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish PostgresRequestTopicPublisher) UpsertPostgresDatabase(ctx context.Context, msg *UpsertPostgresDatabaseMessage) {
	publish.publisher.Publish(ctx, msg)
}

// Method: MigratePostgresDatabase

func (msg *MigratePostgresDatabaseMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.awsinfra.v1.topic.PostgresRequestTopic",
		GrpcMethod:       "MigratePostgresDatabase",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send PostgresRequestTopicTxSender[C]) MigratePostgresDatabase(ctx context.Context, sendContext C, msg *MigratePostgresDatabaseMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect PostgresRequestTopicCollector[C]) MigratePostgresDatabase(sendContext C, msg *MigratePostgresDatabaseMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish PostgresRequestTopicPublisher) MigratePostgresDatabase(ctx context.Context, msg *MigratePostgresDatabaseMessage) {
	publish.publisher.Publish(ctx, msg)
}

// Method: CleanupPostgresDatabase

func (msg *CleanupPostgresDatabaseMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.awsinfra.v1.topic.PostgresRequestTopic",
		GrpcMethod:       "CleanupPostgresDatabase",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send PostgresRequestTopicTxSender[C]) CleanupPostgresDatabase(ctx context.Context, sendContext C, msg *CleanupPostgresDatabaseMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect PostgresRequestTopicCollector[C]) CleanupPostgresDatabase(sendContext C, msg *CleanupPostgresDatabaseMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish PostgresRequestTopicPublisher) CleanupPostgresDatabase(ctx context.Context, msg *CleanupPostgresDatabaseMessage) {
	publish.publisher.Publish(ctx, msg)
}

// Service: PostgresReplyTopic
type PostgresReplyTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewPostgresReplyTopicTxSender[C any](sender o5msg.TxSender[C]) *PostgresReplyTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.PostgresReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "PostgresDatabaseStatus",
				Message: (*PostgresDatabaseStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &PostgresReplyTopicTxSender[C]{sender: sender}
}

type PostgresReplyTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewPostgresReplyTopicCollector[C any](collector o5msg.Collector[C]) *PostgresReplyTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.PostgresReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "PostgresDatabaseStatus",
				Message: (*PostgresDatabaseStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &PostgresReplyTopicCollector[C]{collector: collector}
}

type PostgresReplyTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewPostgresReplyTopicPublisher(publisher o5msg.Publisher) *PostgresReplyTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.PostgresReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "PostgresDatabaseStatus",
				Message: (*PostgresDatabaseStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &PostgresReplyTopicPublisher{publisher: publisher}
}

// Method: PostgresDatabaseStatus

func (msg *PostgresDatabaseStatusMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.awsinfra.v1.topic.PostgresReplyTopic",
		GrpcMethod:       "PostgresDatabaseStatus",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_reply",
	}
	if msg.Request != nil {
		header.Extension = &messaging_pb.Message_Reply_{
			Reply: &messaging_pb.Message_Reply{
				ReplyTo: msg.Request.ReplyTo,
			},
		}
	}
	return header
}

func (send PostgresReplyTopicTxSender[C]) PostgresDatabaseStatus(ctx context.Context, sendContext C, msg *PostgresDatabaseStatusMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect PostgresReplyTopicCollector[C]) PostgresDatabaseStatus(sendContext C, msg *PostgresDatabaseStatusMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish PostgresReplyTopicPublisher) PostgresDatabaseStatus(ctx context.Context, msg *PostgresDatabaseStatusMessage) {
	publish.publisher.Publish(ctx, msg)
}
