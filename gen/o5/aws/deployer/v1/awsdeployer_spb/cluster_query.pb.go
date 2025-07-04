// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/aws/deployer/v1/service/cluster_query.proto

package awsdeployer_spb

import (
	reflect "reflect"
	sync "sync"

	_ "github.com/pentops/j5/gen/j5/ext/v1/ext_j5pb"
	list_j5pb "github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	awsdeployer_pb "github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetClusterRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClusterId string `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty"`
}

func (x *GetClusterRequest) Reset() {
	*x = GetClusterRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetClusterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetClusterRequest) ProtoMessage() {}

func (x *GetClusterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetClusterRequest.ProtoReflect.Descriptor instead.
func (*GetClusterRequest) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP(), []int{0}
}

func (x *GetClusterRequest) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

type GetClusterResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	State  *awsdeployer_pb.ClusterState   `protobuf:"bytes,1,opt,name=state,proto3" json:"state,omitempty"`
	Events []*awsdeployer_pb.ClusterEvent `protobuf:"bytes,2,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *GetClusterResponse) Reset() {
	*x = GetClusterResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetClusterResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetClusterResponse) ProtoMessage() {}

func (x *GetClusterResponse) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetClusterResponse.ProtoReflect.Descriptor instead.
func (*GetClusterResponse) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP(), []int{1}
}

func (x *GetClusterResponse) GetState() *awsdeployer_pb.ClusterState {
	if x != nil {
		return x.State
	}
	return nil
}

func (x *GetClusterResponse) GetEvents() []*awsdeployer_pb.ClusterEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

type ListClusterEventsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ClusterId string                  `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty"`
	Page      *list_j5pb.PageRequest  `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
	Query     *list_j5pb.QueryRequest `protobuf:"bytes,101,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *ListClusterEventsRequest) Reset() {
	*x = ListClusterEventsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListClusterEventsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListClusterEventsRequest) ProtoMessage() {}

func (x *ListClusterEventsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListClusterEventsRequest.ProtoReflect.Descriptor instead.
func (*ListClusterEventsRequest) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP(), []int{2}
}

func (x *ListClusterEventsRequest) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

func (x *ListClusterEventsRequest) GetPage() *list_j5pb.PageRequest {
	if x != nil {
		return x.Page
	}
	return nil
}

func (x *ListClusterEventsRequest) GetQuery() *list_j5pb.QueryRequest {
	if x != nil {
		return x.Query
	}
	return nil
}

type ListClusterEventsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events []*awsdeployer_pb.ClusterEvent `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	Page   *list_j5pb.PageResponse        `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
}

func (x *ListClusterEventsResponse) Reset() {
	*x = ListClusterEventsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListClusterEventsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListClusterEventsResponse) ProtoMessage() {}

func (x *ListClusterEventsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListClusterEventsResponse.ProtoReflect.Descriptor instead.
func (*ListClusterEventsResponse) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP(), []int{3}
}

func (x *ListClusterEventsResponse) GetEvents() []*awsdeployer_pb.ClusterEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

func (x *ListClusterEventsResponse) GetPage() *list_j5pb.PageResponse {
	if x != nil {
		return x.Page
	}
	return nil
}

type ListClustersRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Page  *list_j5pb.PageRequest  `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
	Query *list_j5pb.QueryRequest `protobuf:"bytes,101,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *ListClustersRequest) Reset() {
	*x = ListClustersRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListClustersRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListClustersRequest) ProtoMessage() {}

func (x *ListClustersRequest) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListClustersRequest.ProtoReflect.Descriptor instead.
func (*ListClustersRequest) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP(), []int{4}
}

func (x *ListClustersRequest) GetPage() *list_j5pb.PageRequest {
	if x != nil {
		return x.Page
	}
	return nil
}

func (x *ListClustersRequest) GetQuery() *list_j5pb.QueryRequest {
	if x != nil {
		return x.Query
	}
	return nil
}

type ListClustersResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Clusters []*awsdeployer_pb.ClusterState `protobuf:"bytes,1,rep,name=clusters,proto3" json:"clusters,omitempty"`
	Page     *list_j5pb.PageResponse        `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
}

func (x *ListClustersResponse) Reset() {
	*x = ListClustersResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListClustersResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListClustersResponse) ProtoMessage() {}

func (x *ListClustersResponse) ProtoReflect() protoreflect.Message {
	mi := &file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListClustersResponse.ProtoReflect.Descriptor instead.
func (*ListClustersResponse) Descriptor() ([]byte, []int) {
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP(), []int{5}
}

func (x *ListClustersResponse) GetClusters() []*awsdeployer_pb.ClusterState {
	if x != nil {
		return x.Clusters
	}
	return nil
}

func (x *ListClustersResponse) GetPage() *list_j5pb.PageResponse {
	if x != nil {
		return x.Page
	}
	return nil
}

var File_o5_aws_deployer_v1_service_cluster_query_proto protoreflect.FileDescriptor

var file_o5_aws_deployer_v1_service_cluster_query_proto_rawDesc = []byte{
	0x0a, 0x2e, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x1a, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x1c, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6a, 0x35, 0x2f, 0x65,
	0x78, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x6a, 0x35, 0x2f, 0x6c, 0x69, 0x73, 0x74,
	0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x15, 0x6a, 0x35, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76,
	0x31, 0x2f, 0x70, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16, 0x6a, 0x35,
	0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x32, 0x0a, 0x11, 0x47, 0x65, 0x74, 0x43, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64, 0x22, 0x86, 0x01, 0x0a, 0x12, 0x47,
	0x65, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x36, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x20, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61,
	0x74, 0x65, 0x52, 0x05, 0x73, 0x74, 0x61, 0x74, 0x65, 0x12, 0x38, 0x0a, 0x06, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35, 0x2e, 0x61,
	0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x43,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x73, 0x22, 0x96, 0x01, 0x0a, 0x18, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x49, 0x64, 0x12,
	0x2b, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e,
	0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x12, 0x2e, 0x0a, 0x05,
	0x71, 0x75, 0x65, 0x72, 0x79, 0x18, 0x65, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35,
	0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x22, 0x83, 0x01, 0x0a,
	0x19, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x38, 0x0a, 0x06, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35, 0x2e,
	0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x06, 0x65, 0x76,
	0x65, 0x6e, 0x74, 0x73, 0x12, 0x2c, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18, 0x64, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e,
	0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52, 0x04, 0x70, 0x61,
	0x67, 0x65, 0x22, 0x8b, 0x01, 0x0a, 0x13, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2b, 0x0a, 0x04, 0x70, 0x61,
	0x67, 0x65, 0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69,
	0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x12, 0x2e, 0x0a, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x18, 0x65, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74,
	0x2e, 0x76, 0x31, 0x2e, 0x51, 0x75, 0x65, 0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x3a, 0x17, 0x8a, 0xf7, 0x98, 0xc6, 0x02, 0x11, 0x12,
	0x0f, 0x6b, 0x65, 0x79, 0x73, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64,
	0x22, 0x82, 0x01, 0x0a, 0x14, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3c, 0x0a, 0x08, 0x63, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x20, 0x2e, 0x6f, 0x35,
	0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x08, 0x63,
	0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x12, 0x2c, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18,
	0x64, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x18, 0x2e, 0x6a, 0x35, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e,
	0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52,
	0x04, 0x70, 0x61, 0x67, 0x65, 0x32, 0xb1, 0x04, 0x0a, 0x13, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x9e, 0x01,
	0x0a, 0x0c, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x12, 0x2f,
	0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x30, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73,
	0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x2b, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52, 0x02, 0x10, 0x01, 0x82, 0xd3, 0xe4, 0x93,
	0x02, 0x1c, 0x3a, 0x01, 0x2a, 0x22, 0x17, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72,
	0x2f, 0x76, 0x31, 0x2f, 0x71, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x12, 0xa2,
	0x01, 0x0a, 0x0a, 0x47, 0x65, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12, 0x2d, 0x2e,
	0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e,
	0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x6c,
	0x75, 0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2e, 0x2e, 0x6f,
	0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76,
	0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x43, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x35, 0xc2, 0xff,
	0x8e, 0x02, 0x04, 0x52, 0x02, 0x08, 0x01, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x26, 0x12, 0x24, 0x2f,
	0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x2f, 0x63, 0x6c,
	0x75, 0x73, 0x74, 0x65, 0x72, 0x73, 0x2f, 0x7b, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f,
	0x69, 0x64, 0x7d, 0x12, 0xc1, 0x01, 0x0a, 0x11, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73,
	0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x34, 0x2e, 0x6f, 0x35, 0x2e, 0x61,
	0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x43, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x35, 0x2e, 0x6f, 0x35, 0x2e, 0x61, 0x77, 0x73, 0x2e, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73,
	0x74, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x3f, 0xc2, 0xff, 0x8e, 0x02, 0x04, 0x52, 0x02, 0x18,
	0x01, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x30, 0x3a, 0x01, 0x2a, 0x22, 0x2b, 0x2f, 0x64, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x65, 0x72, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x73, 0x2f, 0x7b, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x7d,
	0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x1a, 0x10, 0xea, 0x85, 0x8f, 0x02, 0x0b, 0x0a, 0x09,
	0x0a, 0x07, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x42, 0x49, 0x5a, 0x47, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f,
	0x6f, 0x35, 0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67, 0x65,
	0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x61, 0x77, 0x73, 0x2f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65,
	0x72, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x77, 0x73, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72,
	0x5f, 0x73, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescOnce sync.Once
	file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescData = file_o5_aws_deployer_v1_service_cluster_query_proto_rawDesc
)

func file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescGZIP() []byte {
	file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescOnce.Do(func() {
		file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescData)
	})
	return file_o5_aws_deployer_v1_service_cluster_query_proto_rawDescData
}

var file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_o5_aws_deployer_v1_service_cluster_query_proto_goTypes = []interface{}{
	(*GetClusterRequest)(nil),           // 0: o5.aws.deployer.v1.service.GetClusterRequest
	(*GetClusterResponse)(nil),          // 1: o5.aws.deployer.v1.service.GetClusterResponse
	(*ListClusterEventsRequest)(nil),    // 2: o5.aws.deployer.v1.service.ListClusterEventsRequest
	(*ListClusterEventsResponse)(nil),   // 3: o5.aws.deployer.v1.service.ListClusterEventsResponse
	(*ListClustersRequest)(nil),         // 4: o5.aws.deployer.v1.service.ListClustersRequest
	(*ListClustersResponse)(nil),        // 5: o5.aws.deployer.v1.service.ListClustersResponse
	(*awsdeployer_pb.ClusterState)(nil), // 6: o5.aws.deployer.v1.ClusterState
	(*awsdeployer_pb.ClusterEvent)(nil), // 7: o5.aws.deployer.v1.ClusterEvent
	(*list_j5pb.PageRequest)(nil),       // 8: j5.list.v1.PageRequest
	(*list_j5pb.QueryRequest)(nil),      // 9: j5.list.v1.QueryRequest
	(*list_j5pb.PageResponse)(nil),      // 10: j5.list.v1.PageResponse
}
var file_o5_aws_deployer_v1_service_cluster_query_proto_depIdxs = []int32{
	6,  // 0: o5.aws.deployer.v1.service.GetClusterResponse.state:type_name -> o5.aws.deployer.v1.ClusterState
	7,  // 1: o5.aws.deployer.v1.service.GetClusterResponse.events:type_name -> o5.aws.deployer.v1.ClusterEvent
	8,  // 2: o5.aws.deployer.v1.service.ListClusterEventsRequest.page:type_name -> j5.list.v1.PageRequest
	9,  // 3: o5.aws.deployer.v1.service.ListClusterEventsRequest.query:type_name -> j5.list.v1.QueryRequest
	7,  // 4: o5.aws.deployer.v1.service.ListClusterEventsResponse.events:type_name -> o5.aws.deployer.v1.ClusterEvent
	10, // 5: o5.aws.deployer.v1.service.ListClusterEventsResponse.page:type_name -> j5.list.v1.PageResponse
	8,  // 6: o5.aws.deployer.v1.service.ListClustersRequest.page:type_name -> j5.list.v1.PageRequest
	9,  // 7: o5.aws.deployer.v1.service.ListClustersRequest.query:type_name -> j5.list.v1.QueryRequest
	6,  // 8: o5.aws.deployer.v1.service.ListClustersResponse.clusters:type_name -> o5.aws.deployer.v1.ClusterState
	10, // 9: o5.aws.deployer.v1.service.ListClustersResponse.page:type_name -> j5.list.v1.PageResponse
	4,  // 10: o5.aws.deployer.v1.service.ClusterQueryService.ListClusters:input_type -> o5.aws.deployer.v1.service.ListClustersRequest
	0,  // 11: o5.aws.deployer.v1.service.ClusterQueryService.GetCluster:input_type -> o5.aws.deployer.v1.service.GetClusterRequest
	2,  // 12: o5.aws.deployer.v1.service.ClusterQueryService.ListClusterEvents:input_type -> o5.aws.deployer.v1.service.ListClusterEventsRequest
	5,  // 13: o5.aws.deployer.v1.service.ClusterQueryService.ListClusters:output_type -> o5.aws.deployer.v1.service.ListClustersResponse
	1,  // 14: o5.aws.deployer.v1.service.ClusterQueryService.GetCluster:output_type -> o5.aws.deployer.v1.service.GetClusterResponse
	3,  // 15: o5.aws.deployer.v1.service.ClusterQueryService.ListClusterEvents:output_type -> o5.aws.deployer.v1.service.ListClusterEventsResponse
	13, // [13:16] is the sub-list for method output_type
	10, // [10:13] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_o5_aws_deployer_v1_service_cluster_query_proto_init() }
func file_o5_aws_deployer_v1_service_cluster_query_proto_init() {
	if File_o5_aws_deployer_v1_service_cluster_query_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetClusterRequest); i {
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
		file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetClusterResponse); i {
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
		file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListClusterEventsRequest); i {
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
		file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListClusterEventsResponse); i {
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
		file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListClustersRequest); i {
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
		file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListClustersResponse); i {
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
			RawDescriptor: file_o5_aws_deployer_v1_service_cluster_query_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_o5_aws_deployer_v1_service_cluster_query_proto_goTypes,
		DependencyIndexes: file_o5_aws_deployer_v1_service_cluster_query_proto_depIdxs,
		MessageInfos:      file_o5_aws_deployer_v1_service_cluster_query_proto_msgTypes,
	}.Build()
	File_o5_aws_deployer_v1_service_cluster_query_proto = out.File
	file_o5_aws_deployer_v1_service_cluster_query_proto_rawDesc = nil
	file_o5_aws_deployer_v1_service_cluster_query_proto_goTypes = nil
	file_o5_aws_deployer_v1_service_cluster_query_proto_depIdxs = nil
}
