// Code generated by protoc-gen-go-o5-messaging. DO NOT EDIT.
// versions:
// - protoc-gen-go-o5-messaging 0.0.0
// source: o5/aws/infra/v1/topic/aws_cf.proto

package awsinfra_tpb

import (
	context "context"
	messaging_j5pb "github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	messaging_pb "github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	o5msg "github.com/pentops/o5-messaging/o5msg"
)

// Service: CloudFormationRequestTopic
// Expose Request Metadata
func (msg *CreateNewStackMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *CreateNewStackMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *UpdateStackMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *UpdateStackMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *CreateChangeSetMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *CreateChangeSetMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *ApplyChangeSetMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *ApplyChangeSetMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *DeleteStackMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *DeleteStackMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *ScaleStackMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *ScaleStackMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *CancelStackUpdateMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *CancelStackUpdateMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *StabalizeStackMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *StabalizeStackMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

type CloudFormationRequestTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewCloudFormationRequestTopicTxSender[C any](sender o5msg.TxSender[C]) *CloudFormationRequestTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "CreateNewStack",
				Message: (*CreateNewStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "UpdateStack",
				Message: (*UpdateStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CreateChangeSet",
				Message: (*CreateChangeSetMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ApplyChangeSet",
				Message: (*ApplyChangeSetMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "DeleteStack",
				Message: (*DeleteStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ScaleStack",
				Message: (*ScaleStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CancelStackUpdate",
				Message: (*CancelStackUpdateMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "StabalizeStack",
				Message: (*StabalizeStackMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &CloudFormationRequestTopicTxSender[C]{sender: sender}
}

type CloudFormationRequestTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewCloudFormationRequestTopicCollector[C any](collector o5msg.Collector[C]) *CloudFormationRequestTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "CreateNewStack",
				Message: (*CreateNewStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "UpdateStack",
				Message: (*UpdateStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CreateChangeSet",
				Message: (*CreateChangeSetMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ApplyChangeSet",
				Message: (*ApplyChangeSetMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "DeleteStack",
				Message: (*DeleteStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ScaleStack",
				Message: (*ScaleStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CancelStackUpdate",
				Message: (*CancelStackUpdateMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "StabalizeStack",
				Message: (*StabalizeStackMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &CloudFormationRequestTopicCollector[C]{collector: collector}
}

type CloudFormationRequestTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewCloudFormationRequestTopicPublisher(publisher o5msg.Publisher) *CloudFormationRequestTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "CreateNewStack",
				Message: (*CreateNewStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "UpdateStack",
				Message: (*UpdateStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CreateChangeSet",
				Message: (*CreateChangeSetMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ApplyChangeSet",
				Message: (*ApplyChangeSetMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "DeleteStack",
				Message: (*DeleteStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ScaleStack",
				Message: (*ScaleStackMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "CancelStackUpdate",
				Message: (*CancelStackUpdateMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "StabalizeStack",
				Message: (*StabalizeStackMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &CloudFormationRequestTopicPublisher{publisher: publisher}
}

// Method: CreateNewStack

func (msg *CreateNewStackMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "CreateNewStack",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) CreateNewStack(ctx context.Context, sendContext C, msg *CreateNewStackMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) CreateNewStack(sendContext C, msg *CreateNewStackMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) CreateNewStack(ctx context.Context, msg *CreateNewStackMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: UpdateStack

func (msg *UpdateStackMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "UpdateStack",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) UpdateStack(ctx context.Context, sendContext C, msg *UpdateStackMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) UpdateStack(sendContext C, msg *UpdateStackMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) UpdateStack(ctx context.Context, msg *UpdateStackMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: CreateChangeSet

func (msg *CreateChangeSetMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "CreateChangeSet",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) CreateChangeSet(ctx context.Context, sendContext C, msg *CreateChangeSetMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) CreateChangeSet(sendContext C, msg *CreateChangeSetMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) CreateChangeSet(ctx context.Context, msg *CreateChangeSetMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: ApplyChangeSet

func (msg *ApplyChangeSetMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "ApplyChangeSet",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) ApplyChangeSet(ctx context.Context, sendContext C, msg *ApplyChangeSetMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) ApplyChangeSet(sendContext C, msg *ApplyChangeSetMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) ApplyChangeSet(ctx context.Context, msg *ApplyChangeSetMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: DeleteStack

func (msg *DeleteStackMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "DeleteStack",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) DeleteStack(ctx context.Context, sendContext C, msg *DeleteStackMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) DeleteStack(sendContext C, msg *DeleteStackMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) DeleteStack(ctx context.Context, msg *DeleteStackMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: ScaleStack

func (msg *ScaleStackMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "ScaleStack",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) ScaleStack(ctx context.Context, sendContext C, msg *ScaleStackMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) ScaleStack(sendContext C, msg *ScaleStackMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) ScaleStack(ctx context.Context, msg *ScaleStackMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: CancelStackUpdate

func (msg *CancelStackUpdateMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "CancelStackUpdate",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) CancelStackUpdate(ctx context.Context, sendContext C, msg *CancelStackUpdateMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) CancelStackUpdate(sendContext C, msg *CancelStackUpdateMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) CancelStackUpdate(ctx context.Context, msg *CancelStackUpdateMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: StabalizeStack

func (msg *StabalizeStackMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationRequestTopic",
		GrpcMethod:       "StabalizeStack",
		Headers:          map[string]string{},
		DestinationTopic: "o5-aws-command_request",
	}
	return header
}

func (send CloudFormationRequestTopicTxSender[C]) StabalizeStack(ctx context.Context, sendContext C, msg *StabalizeStackMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationRequestTopicCollector[C]) StabalizeStack(sendContext C, msg *StabalizeStackMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationRequestTopicPublisher) StabalizeStack(ctx context.Context, msg *StabalizeStackMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Service: CloudFormationReplyTopic
// Expose Request Metadata
func (msg *StackStatusChangedMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *StackStatusChangedMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

// Expose Request Metadata
func (msg *ChangeSetStatusChangedMessage) SetJ5RequestMetadata(md *messaging_j5pb.RequestMetadata) {
	msg.Request = md
}
func (msg *ChangeSetStatusChangedMessage) GetJ5RequestMetadata() *messaging_j5pb.RequestMetadata {
	return msg.Request
}

type CloudFormationReplyTopicTxSender[C any] struct {
	sender o5msg.TxSender[C]
}

func NewCloudFormationReplyTopicTxSender[C any](sender o5msg.TxSender[C]) *CloudFormationReplyTopicTxSender[C] {
	sender.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.CloudFormationReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "StackStatusChanged",
				Message: (*StackStatusChangedMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ChangeSetStatusChanged",
				Message: (*ChangeSetStatusChangedMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &CloudFormationReplyTopicTxSender[C]{sender: sender}
}

type CloudFormationReplyTopicCollector[C any] struct {
	collector o5msg.Collector[C]
}

func NewCloudFormationReplyTopicCollector[C any](collector o5msg.Collector[C]) *CloudFormationReplyTopicCollector[C] {
	collector.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.CloudFormationReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "StackStatusChanged",
				Message: (*StackStatusChangedMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ChangeSetStatusChanged",
				Message: (*ChangeSetStatusChangedMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &CloudFormationReplyTopicCollector[C]{collector: collector}
}

type CloudFormationReplyTopicPublisher struct {
	publisher o5msg.Publisher
}

func NewCloudFormationReplyTopicPublisher(publisher o5msg.Publisher) *CloudFormationReplyTopicPublisher {
	publisher.Register(o5msg.TopicDescriptor{
		Service: "o5.aws.infra.v1.topic.CloudFormationReplyTopic",
		Methods: []o5msg.MethodDescriptor{
			{
				Name:    "StackStatusChanged",
				Message: (*StackStatusChangedMessage).ProtoReflect(nil).Descriptor(),
			},
			{
				Name:    "ChangeSetStatusChanged",
				Message: (*ChangeSetStatusChangedMessage).ProtoReflect(nil).Descriptor(),
			},
		},
	})
	return &CloudFormationReplyTopicPublisher{publisher: publisher}
}

// Method: StackStatusChanged

func (msg *StackStatusChangedMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationReplyTopic",
		GrpcMethod:       "StackStatusChanged",
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

func (send CloudFormationReplyTopicTxSender[C]) StackStatusChanged(ctx context.Context, sendContext C, msg *StackStatusChangedMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationReplyTopicCollector[C]) StackStatusChanged(sendContext C, msg *StackStatusChangedMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationReplyTopicPublisher) StackStatusChanged(ctx context.Context, msg *StackStatusChangedMessage) error {
	return publish.publisher.Publish(ctx, msg)
}

// Method: ChangeSetStatusChanged

func (msg *ChangeSetStatusChangedMessage) O5MessageHeader() o5msg.Header {
	header := o5msg.Header{
		GrpcService:      "o5.aws.infra.v1.topic.CloudFormationReplyTopic",
		GrpcMethod:       "ChangeSetStatusChanged",
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

func (send CloudFormationReplyTopicTxSender[C]) ChangeSetStatusChanged(ctx context.Context, sendContext C, msg *ChangeSetStatusChangedMessage) error {
	return send.sender.Send(ctx, sendContext, msg)
}

func (collect CloudFormationReplyTopicCollector[C]) ChangeSetStatusChanged(sendContext C, msg *ChangeSetStatusChangedMessage) {
	collect.collector.Collect(sendContext, msg)
}

func (publish CloudFormationReplyTopicPublisher) ChangeSetStatusChanged(ctx context.Context, msg *ChangeSetStatusChangedMessage) error {
	return publish.publisher.Publish(ctx, msg)
}
