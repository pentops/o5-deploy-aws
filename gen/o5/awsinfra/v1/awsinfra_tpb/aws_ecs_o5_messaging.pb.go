// Code generated by Bprotoc-gen-go-o5-messaging . DO NOT EDIT.
// versions:
// - protoc-gen-go-o5-messaging 0.0.0
// source: o5/awsinfra/v1/topic/aws_ecs.proto

package awsinfra_tpb

import (
	context "context"
	messaging_pb "github.com/pentops/o5-go/messaging/v1/messaging_pb"
	o5msg "github.com/pentops/o5-messaging.go/o5msg"
)

// Service: ECSRequestTopic
type ECSRequestTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewECSRequestTopicTxSender[C any](sender o5msg.TxSender[C]) *ECSRequestTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.ECSRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "RunECSTask",
				Message: (*RunECSTaskMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSRequestTopicTxSender[C]{sender: sender}
}

type ECSRequestTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewECSRequestTopicCollector[C any](collector o5msg.Collector[C]) *ECSRequestTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.ECSRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "RunECSTask",
				Message: (*RunECSTaskMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSRequestTopicCollector[C]{collector: collector}
}

type ECSRequestTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewECSRequestTopicPublisher(publisher o5msg.Publisher) *ECSRequestTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.ECSRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "RunECSTask",
				Message: (*RunECSTaskMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSRequestTopicPublisher{publisher: publisher}
}

// Method: RunECSTask

func (msg *RunECSTaskMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.awsinfra.v1.topic.ECSRequestTopic",
		GrpcMethod:       "RunECSTask",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send ECSRequestTopicTxSender[C]) RunECSTask(ctx context.Context, sendContext C, msg *RunECSTaskMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect ECSRequestTopicCollector[C]) RunECSTask(sendContext C, msg *RunECSTaskMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish ECSRequestTopicPublisher) RunECSTask(ctx context.Context, msg *RunECSTaskMessage) {
	publish.publisher.Publish(ctx, msg)
}

// Service: ECSReplyTopic
type ECSReplyTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewECSReplyTopicTxSender[C any](sender o5msg.TxSender[C]) *ECSReplyTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.ECSReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "ECSTaskStatus",
				Message: (*ECSTaskStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSReplyTopicTxSender[C]{sender: sender}
}

type ECSReplyTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewECSReplyTopicCollector[C any](collector o5msg.Collector[C]) *ECSReplyTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.ECSReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "ECSTaskStatus",
				Message: (*ECSTaskStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSReplyTopicCollector[C]{collector: collector}
}

type ECSReplyTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewECSReplyTopicPublisher(publisher o5msg.Publisher) *ECSReplyTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "o5.awsinfra.v1.topic.ECSReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "ECSTaskStatus",
				Message: (*ECSTaskStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSReplyTopicPublisher{publisher: publisher}
}

// Method: ECSTaskStatus

func (msg *ECSTaskStatusMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.awsinfra.v1.topic.ECSReplyTopic",
		GrpcMethod:       "ECSTaskStatus",
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

func (send ECSReplyTopicTxSender[C]) ECSTaskStatus(ctx context.Context, sendContext C, msg *ECSTaskStatusMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect ECSReplyTopicCollector[C]) ECSTaskStatus(sendContext C, msg *ECSTaskStatusMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish ECSReplyTopicPublisher) ECSTaskStatus(ctx context.Context, msg *ECSTaskStatusMessage) {
	publish.publisher.Publish(ctx, msg)
}
