// Code generated by protoc-gen-go-o5-messaging. DO NOT EDIT.
// versions:
// - protoc-gen-go-o5-messaging 0.0.0
// source: o5/aws/infra/v1/topic/aws_ecs.proto

package awsinfra_tpb

import (
	context "context"
	messaging_j5pb "github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	messaging_pb "github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	o5msg "github.com/pentops/o5-messaging/o5msg"
)

// Service: ECSRequestTopic
// Expose Request Metadata
func (msg *RunECSTaskMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *RunECSTaskMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *SetECSScaleMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *SetECSScaleMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

type ECSRequestTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewECSRequestTopicTxSender[C any](sender o5msg.TxSender[C]) *ECSRequestTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.ECSRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "RunECSTask",
				Message: (*RunECSTaskMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "SetECSScale",
				Message: (*SetECSScaleMessage).ProtoReflect(nil).Descriptor(),
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
		Service: "o5.aws.infra.v1.topic.ECSRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "RunECSTask",
				Message: (*RunECSTaskMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "SetECSScale",
				Message: (*SetECSScaleMessage).ProtoReflect(nil).Descriptor(),
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
		Service: "o5.aws.infra.v1.topic.ECSRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "RunECSTask",
				Message: (*RunECSTaskMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "SetECSScale",
				Message: (*SetECSScaleMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSRequestTopicPublisher{publisher: publisher}
}

// Method: RunECSTask

func (msg *RunECSTaskMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.ECSRequestTopic",
		GrpcMethod:       "RunECSTask",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	if msg.Request != nil {
		header.Extension = &messaging_pb.Message_Request_{
			Request: &messaging_pb.Message_Request{
				ReplyTo: msg.Request.ReplyTo,
			},
		}
	} else {
		header.Extension = &messaging_pb.Message_Request_{
			Request: &messaging_pb.Message_Request{
				ReplyTo: "",
			},
		}
	}
	return header
}

func (send ECSRequestTopicTxSender[C]) RunECSTask(ctx context.Context, sendContext C, msg *RunECSTaskMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect ECSRequestTopicCollector[C]) RunECSTask(sendContext C, msg *RunECSTaskMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish ECSRequestTopicPublisher) RunECSTask(ctx context.Context, msg *RunECSTaskMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: SetECSScale

func (msg *SetECSScaleMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.ECSRequestTopic",
		GrpcMethod:       "SetECSScale",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	if msg.Request != nil {
		header.Extension = &messaging_pb.Message_Request_{
			Request: &messaging_pb.Message_Request{
				ReplyTo: msg.Request.ReplyTo,
			},
		}
	} else {
		header.Extension = &messaging_pb.Message_Request_{
			Request: &messaging_pb.Message_Request{
				ReplyTo: "",
			},
		}
	}
	return header
}

func (send ECSRequestTopicTxSender[C]) SetECSScale(ctx context.Context, sendContext C, msg *SetECSScaleMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect ECSRequestTopicCollector[C]) SetECSScale(sendContext C, msg *SetECSScaleMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish ECSRequestTopicPublisher) SetECSScale(ctx context.Context, msg *SetECSScaleMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Service: ECSReplyTopic
// Expose Request Metadata
func (msg *ECSTaskStatusMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *ECSTaskStatusMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *ECSDeploymentStatusMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *ECSDeploymentStatusMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

type ECSReplyTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewECSReplyTopicTxSender[C any](sender o5msg.TxSender[C]) *ECSReplyTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.ECSReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "ECSTaskStatus",
				Message: (*ECSTaskStatusMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ECSDeploymentStatus",
				Message: (*ECSDeploymentStatusMessage).ProtoReflect(nil).Descriptor(),
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
		Service: "o5.aws.infra.v1.topic.ECSReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "ECSTaskStatus",
				Message: (*ECSTaskStatusMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ECSDeploymentStatus",
				Message: (*ECSDeploymentStatusMessage).ProtoReflect(nil).Descriptor(),
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
		Service: "o5.aws.infra.v1.topic.ECSReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "ECSTaskStatus",
				Message: (*ECSTaskStatusMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ECSDeploymentStatus",
				Message: (*ECSDeploymentStatusMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &ECSReplyTopicPublisher{publisher: publisher}
}

// Method: ECSTaskStatus

func (msg *ECSTaskStatusMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.ECSReplyTopic",
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

func (publish ECSReplyTopicPublisher) ECSTaskStatus(ctx context.Context, msg *ECSTaskStatusMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: ECSDeploymentStatus

func (msg *ECSDeploymentStatusMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.ECSReplyTopic",
		GrpcMethod:       "ECSDeploymentStatus",
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

func (send ECSReplyTopicTxSender[C]) ECSDeploymentStatus(ctx context.Context, sendContext C, msg *ECSDeploymentStatusMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect ECSReplyTopicCollector[C]) ECSDeploymentStatus(sendContext C, msg *ECSDeploymentStatusMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish ECSReplyTopicPublisher) ECSDeploymentStatus(ctx context.Context, msg *ECSDeploymentStatusMessage) error {
	return publish.publisher.Publish(ctx, msg)
}
