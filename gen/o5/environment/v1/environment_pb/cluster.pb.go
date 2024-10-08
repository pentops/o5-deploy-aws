// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/environment/v1/cluster.proto

package environment_pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Cluster struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Types that are assignable to Provider:
	//
	//	*Cluster_EcsCluster
	Provider isCluster_Provider `protobuf_oneof:"provider"`
}

func (x *Cluster) Reset() {
	*x = Cluster{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_environment_v1_cluster_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cluster) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cluster) ProtoMessage() {}

func (x *Cluster) ProtoReflect() protoreflect.Message {
	mi := &file_o5_environment_v1_cluster_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cluster.ProtoReflect.Descriptor instead.
func (*Cluster) Descriptor() ([]byte, []int) {
	return file_o5_environment_v1_cluster_proto_rawDescGZIP(), []int{0}
}

func (x *Cluster) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (m *Cluster) GetProvider() isCluster_Provider {
	if m != nil {
		return m.Provider
	}
	return nil
}

func (x *Cluster) GetEcsCluster() *ECSCluster {
	if x, ok := x.GetProvider().(*Cluster_EcsCluster); ok {
		return x.EcsCluster
	}
	return nil
}

type isCluster_Provider interface {
	isCluster_Provider()
}

type Cluster_EcsCluster struct {
	EcsCluster *ECSCluster `protobuf:"bytes,10,opt,name=ecs_cluster,json=ecsCluster,proto3,oneof"`
}

func (*Cluster_EcsCluster) isCluster_Provider() {}

type CombinedConfig struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Types that are assignable to Provider:
	//
	//	*CombinedConfig_EcsCluster
	Provider     isCombinedConfig_Provider `protobuf_oneof:"provider"`
	Environments []*Environment            `protobuf:"bytes,2,rep,name=environments,proto3" json:"environments,omitempty"`
}

func (x *CombinedConfig) Reset() {
	*x = CombinedConfig{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_environment_v1_cluster_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CombinedConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CombinedConfig) ProtoMessage() {}

func (x *CombinedConfig) ProtoReflect() protoreflect.Message {
	mi := &file_o5_environment_v1_cluster_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CombinedConfig.ProtoReflect.Descriptor instead.
func (*CombinedConfig) Descriptor() ([]byte, []int) {
	return file_o5_environment_v1_cluster_proto_rawDescGZIP(), []int{1}
}

func (x *CombinedConfig) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (m *CombinedConfig) GetProvider() isCombinedConfig_Provider {
	if m != nil {
		return m.Provider
	}
	return nil
}

func (x *CombinedConfig) GetEcsCluster() *ECSCluster {
	if x, ok := x.GetProvider().(*CombinedConfig_EcsCluster); ok {
		return x.EcsCluster
	}
	return nil
}

func (x *CombinedConfig) GetEnvironments() []*Environment {
	if x != nil {
		return x.Environments
	}
	return nil
}

type isCombinedConfig_Provider interface {
	isCombinedConfig_Provider()
}

type CombinedConfig_EcsCluster struct {
	EcsCluster *ECSCluster `protobuf:"bytes,10,opt,name=ecs_cluster,json=ecsCluster,proto3,oneof"`
}

func (*CombinedConfig_EcsCluster) isCombinedConfig_Provider() {}

type ECSCluster struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ListenerArn          string `protobuf:"bytes,1,opt,name=listener_arn,json=listenerArn,proto3" json:"listener_arn,omitempty"`
	EcsClusterName       string `protobuf:"bytes,2,opt,name=ecs_cluster_name,json=ecsClusterName,proto3" json:"ecs_cluster_name,omitempty"`
	EcsRepo              string `protobuf:"bytes,3,opt,name=ecs_repo,json=ecsRepo,proto3" json:"ecs_repo,omitempty"`
	EcsTaskExecutionRole string `protobuf:"bytes,4,opt,name=ecs_task_execution_role,json=ecsTaskExecutionRole,proto3" json:"ecs_task_execution_role,omitempty"`
	VpcId                string `protobuf:"bytes,5,opt,name=vpc_id,json=vpcId,proto3" json:"vpc_id,omitempty"`
	AwsAccount           string `protobuf:"bytes,6,opt,name=aws_account,json=awsAccount,proto3" json:"aws_account,omitempty"`
	AwsRegion            string `protobuf:"bytes,7,opt,name=aws_region,json=awsRegion,proto3" json:"aws_region,omitempty"`
	EventBusArn          string `protobuf:"bytes,16,opt,name=event_bus_arn,json=eventBusArn,proto3" json:"event_bus_arn,omitempty"`
	// The role to assume when deploying to this environment
	O5DeployerAssumeRole string `protobuf:"bytes,8,opt,name=o5_deployer_assume_role,json=o5DeployerAssumeRole,proto3" json:"o5_deployer_assume_role,omitempty"`
	// The roles which a deployer service (e.g. o5 itself) should be allowed to
	// assume, when grant_meta_deploy_permissions is set in the application config
	O5DeployerGrantRoles []string   `protobuf:"bytes,9,rep,name=o5_deployer_grant_roles,json=o5DeployerGrantRoles,proto3" json:"o5_deployer_grant_roles,omitempty"`
	RdsHosts             []*RDSHost `protobuf:"bytes,11,rep,name=rds_hosts,json=rdsHosts,proto3" json:"rds_hosts,omitempty"`
	SidecarImageVersion  string     `protobuf:"bytes,13,opt,name=sidecar_image_version,json=sidecarImageVersion,proto3" json:"sidecar_image_version,omitempty"` // Must be set, no default.
	SidecarImageRepo     *string    `protobuf:"bytes,14,opt,name=sidecar_image_repo,json=sidecarImageRepo,proto3,oneof" json:"sidecar_image_repo,omitempty"`    // defaults to ghcr.io/pentops/o5-runtime-sidecar
	// S3 buckets (and others?) must be globally unique for all AWS accounts and regions,
	// buckets are named {name}-{app}-{environment}-{region}-{global_namespace}
	GlobalNamespace string `protobuf:"bytes,15,opt,name=global_namespace,json=globalNamespace,proto3" json:"global_namespace,omitempty"`
}

func (x *ECSCluster) Reset() {
	*x = ECSCluster{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_environment_v1_cluster_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ECSCluster) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ECSCluster) ProtoMessage() {}

func (x *ECSCluster) ProtoReflect() protoreflect.Message {
	mi := &file_o5_environment_v1_cluster_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ECSCluster.ProtoReflect.Descriptor instead.
func (*ECSCluster) Descriptor() ([]byte, []int) {
	return file_o5_environment_v1_cluster_proto_rawDescGZIP(), []int{2}
}

func (x *ECSCluster) GetListenerArn() string {
	if x != nil {
		return x.ListenerArn
	}
	return ""
}

func (x *ECSCluster) GetEcsClusterName() string {
	if x != nil {
		return x.EcsClusterName
	}
	return ""
}

func (x *ECSCluster) GetEcsRepo() string {
	if x != nil {
		return x.EcsRepo
	}
	return ""
}

func (x *ECSCluster) GetEcsTaskExecutionRole() string {
	if x != nil {
		return x.EcsTaskExecutionRole
	}
	return ""
}

func (x *ECSCluster) GetVpcId() string {
	if x != nil {
		return x.VpcId
	}
	return ""
}

func (x *ECSCluster) GetAwsAccount() string {
	if x != nil {
		return x.AwsAccount
	}
	return ""
}

func (x *ECSCluster) GetAwsRegion() string {
	if x != nil {
		return x.AwsRegion
	}
	return ""
}

func (x *ECSCluster) GetEventBusArn() string {
	if x != nil {
		return x.EventBusArn
	}
	return ""
}

func (x *ECSCluster) GetO5DeployerAssumeRole() string {
	if x != nil {
		return x.O5DeployerAssumeRole
	}
	return ""
}

func (x *ECSCluster) GetO5DeployerGrantRoles() []string {
	if x != nil {
		return x.O5DeployerGrantRoles
	}
	return nil
}

func (x *ECSCluster) GetRdsHosts() []*RDSHost {
	if x != nil {
		return x.RdsHosts
	}
	return nil
}

func (x *ECSCluster) GetSidecarImageVersion() string {
	if x != nil {
		return x.SidecarImageVersion
	}
	return ""
}

func (x *ECSCluster) GetSidecarImageRepo() string {
	if x != nil && x.SidecarImageRepo != nil {
		return *x.SidecarImageRepo
	}
	return ""
}

func (x *ECSCluster) GetGlobalNamespace() string {
	if x != nil {
		return x.GlobalNamespace
	}
	return ""
}

type RDSHost struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ServerGroup string `protobuf:"bytes,1,opt,name=server_group,json=serverGroup,proto3" json:"server_group,omitempty"`
	SecretName  string `protobuf:"bytes,2,opt,name=secret_name,json=secretName,proto3" json:"secret_name,omitempty"`
}

func (x *RDSHost) Reset() {
	*x = RDSHost{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_environment_v1_cluster_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RDSHost) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RDSHost) ProtoMessage() {}

func (x *RDSHost) ProtoReflect() protoreflect.Message {
	mi := &file_o5_environment_v1_cluster_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RDSHost.ProtoReflect.Descriptor instead.
func (*RDSHost) Descriptor() ([]byte, []int) {
	return file_o5_environment_v1_cluster_proto_rawDescGZIP(), []int{3}
}

func (x *RDSHost) GetServerGroup() string {
	if x != nil {
		return x.ServerGroup
	}
	return ""
}

func (x *RDSHost) GetSecretName() string {
	if x != nil {
		return x.SecretName
	}
	return ""
}

var File_o5_environment_v1_cluster_proto protoreflect.FileDescriptor

var file_o5_environment_v1_cluster_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x6f, 0x35, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74,
	0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x11, 0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e,
	0x74, 0x2e, 0x76, 0x31, 0x1a, 0x23, 0x6f, 0x35, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e,
	0x6d, 0x65, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x6b, 0x0a, 0x07, 0x43, 0x6c, 0x75,
	0x73, 0x74, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x40, 0x0a, 0x0b, 0x65, 0x63, 0x73, 0x5f,
	0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1d, 0x2e,
	0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x45, 0x43, 0x53, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x48, 0x00, 0x52, 0x0a,
	0x65, 0x63, 0x73, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x42, 0x0a, 0x0a, 0x08, 0x70, 0x72,
	0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x22, 0xb6, 0x01, 0x0a, 0x0e, 0x43, 0x6f, 0x6d, 0x62, 0x69,
	0x6e, 0x65, 0x64, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x40, 0x0a,
	0x0b, 0x65, 0x63, 0x73, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x18, 0x0a, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d,
	0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x43, 0x53, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65,
	0x72, 0x48, 0x00, 0x52, 0x0a, 0x65, 0x63, 0x73, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12,
	0x42, 0x0a, 0x0c, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x18,
	0x02, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x6f, 0x35, 0x2e, 0x65, 0x6e, 0x76, 0x69, 0x72,
	0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6e, 0x76, 0x69, 0x72, 0x6f,
	0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x52, 0x0c, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65,
	0x6e, 0x74, 0x73, 0x42, 0x0a, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72, 0x22,
	0xf6, 0x04, 0x0a, 0x0a, 0x45, 0x43, 0x53, 0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x12, 0x21,
	0x0a, 0x0c, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x5f, 0x61, 0x72, 0x6e, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x6c, 0x69, 0x73, 0x74, 0x65, 0x6e, 0x65, 0x72, 0x41, 0x72,
	0x6e, 0x12, 0x28, 0x0a, 0x10, 0x65, 0x63, 0x73, 0x5f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0e, 0x65, 0x63, 0x73,
	0x43, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x19, 0x0a, 0x08, 0x65,
	0x63, 0x73, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x65,
	0x63, 0x73, 0x52, 0x65, 0x70, 0x6f, 0x12, 0x35, 0x0a, 0x17, 0x65, 0x63, 0x73, 0x5f, 0x74, 0x61,
	0x73, 0x6b, 0x5f, 0x65, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x72, 0x6f, 0x6c,
	0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x65, 0x63, 0x73, 0x54, 0x61, 0x73, 0x6b,
	0x45, 0x78, 0x65, 0x63, 0x75, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x6f, 0x6c, 0x65, 0x12, 0x15, 0x0a,
	0x06, 0x76, 0x70, 0x63, 0x5f, 0x69, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76,
	0x70, 0x63, 0x49, 0x64, 0x12, 0x1f, 0x0a, 0x0b, 0x61, 0x77, 0x73, 0x5f, 0x61, 0x63, 0x63, 0x6f,
	0x75, 0x6e, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x61, 0x77, 0x73, 0x41, 0x63,
	0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x61, 0x77, 0x73, 0x5f, 0x72, 0x65, 0x67,
	0x69, 0x6f, 0x6e, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x61, 0x77, 0x73, 0x52, 0x65,
	0x67, 0x69, 0x6f, 0x6e, 0x12, 0x22, 0x0a, 0x0d, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x5f, 0x62, 0x75,
	0x73, 0x5f, 0x61, 0x72, 0x6e, 0x18, 0x10, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x65, 0x76, 0x65,
	0x6e, 0x74, 0x42, 0x75, 0x73, 0x41, 0x72, 0x6e, 0x12, 0x35, 0x0a, 0x17, 0x6f, 0x35, 0x5f, 0x64,
	0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f, 0x61, 0x73, 0x73, 0x75, 0x6d, 0x65, 0x5f, 0x72,
	0x6f, 0x6c, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x14, 0x6f, 0x35, 0x44, 0x65, 0x70,
	0x6c, 0x6f, 0x79, 0x65, 0x72, 0x41, 0x73, 0x73, 0x75, 0x6d, 0x65, 0x52, 0x6f, 0x6c, 0x65, 0x12,
	0x35, 0x0a, 0x17, 0x6f, 0x35, 0x5f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x5f, 0x67,
	0x72, 0x61, 0x6e, 0x74, 0x5f, 0x72, 0x6f, 0x6c, 0x65, 0x73, 0x18, 0x09, 0x20, 0x03, 0x28, 0x09,
	0x52, 0x14, 0x6f, 0x35, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x65, 0x72, 0x47, 0x72, 0x61, 0x6e,
	0x74, 0x52, 0x6f, 0x6c, 0x65, 0x73, 0x12, 0x37, 0x0a, 0x09, 0x72, 0x64, 0x73, 0x5f, 0x68, 0x6f,
	0x73, 0x74, 0x73, 0x18, 0x0b, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x6f, 0x35, 0x2e, 0x65,
	0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x44,
	0x53, 0x48, 0x6f, 0x73, 0x74, 0x52, 0x08, 0x72, 0x64, 0x73, 0x48, 0x6f, 0x73, 0x74, 0x73, 0x12,
	0x32, 0x0a, 0x15, 0x73, 0x69, 0x64, 0x65, 0x63, 0x61, 0x72, 0x5f, 0x69, 0x6d, 0x61, 0x67, 0x65,
	0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x09, 0x52, 0x13,
	0x73, 0x69, 0x64, 0x65, 0x63, 0x61, 0x72, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x56, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x12, 0x31, 0x0a, 0x12, 0x73, 0x69, 0x64, 0x65, 0x63, 0x61, 0x72, 0x5f, 0x69,
	0x6d, 0x61, 0x67, 0x65, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x09, 0x48,
	0x00, 0x52, 0x10, 0x73, 0x69, 0x64, 0x65, 0x63, 0x61, 0x72, 0x49, 0x6d, 0x61, 0x67, 0x65, 0x52,
	0x65, 0x70, 0x6f, 0x88, 0x01, 0x01, 0x12, 0x29, 0x0a, 0x10, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0f, 0x67, 0x6c, 0x6f, 0x62, 0x61, 0x6c, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63,
	0x65, 0x42, 0x15, 0x0a, 0x13, 0x5f, 0x73, 0x69, 0x64, 0x65, 0x63, 0x61, 0x72, 0x5f, 0x69, 0x6d,
	0x61, 0x67, 0x65, 0x5f, 0x72, 0x65, 0x70, 0x6f, 0x22, 0x4d, 0x0a, 0x07, 0x52, 0x44, 0x53, 0x48,
	0x6f, 0x73, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x67, 0x72,
	0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x72, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x65, 0x63, 0x72, 0x65, 0x74,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x65, 0x63,
	0x72, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x42, 0x47, 0x5a, 0x45, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x6f, 0x35,
	0x2d, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x2d, 0x61, 0x77, 0x73, 0x2f, 0x67, 0x65, 0x6e, 0x2f,
	0x6f, 0x35, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x2f, 0x76,
	0x31, 0x2f, 0x65, 0x6e, 0x76, 0x69, 0x72, 0x6f, 0x6e, 0x6d, 0x65, 0x6e, 0x74, 0x5f, 0x70, 0x62,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_o5_environment_v1_cluster_proto_rawDescOnce sync.Once
	file_o5_environment_v1_cluster_proto_rawDescData = file_o5_environment_v1_cluster_proto_rawDesc
)

func file_o5_environment_v1_cluster_proto_rawDescGZIP() []byte {
	file_o5_environment_v1_cluster_proto_rawDescOnce.Do(func() {
		file_o5_environment_v1_cluster_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_environment_v1_cluster_proto_rawDescData)
	})
	return file_o5_environment_v1_cluster_proto_rawDescData
}

var file_o5_environment_v1_cluster_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_o5_environment_v1_cluster_proto_goTypes = []interface{}{
	(*Cluster)(nil),        // 0: o5.environment.v1.Cluster
	(*CombinedConfig)(nil), // 1: o5.environment.v1.CombinedConfig
	(*ECSCluster)(nil),     // 2: o5.environment.v1.ECSCluster
	(*RDSHost)(nil),        // 3: o5.environment.v1.RDSHost
	(*Environment)(nil),    // 4: o5.environment.v1.Environment
}
var file_o5_environment_v1_cluster_proto_depIdxs = []int32{
	2, // 0: o5.environment.v1.Cluster.ecs_cluster:type_name -> o5.environment.v1.ECSCluster
	2, // 1: o5.environment.v1.CombinedConfig.ecs_cluster:type_name -> o5.environment.v1.ECSCluster
	4, // 2: o5.environment.v1.CombinedConfig.environments:type_name -> o5.environment.v1.Environment
	3, // 3: o5.environment.v1.ECSCluster.rds_hosts:type_name -> o5.environment.v1.RDSHost
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_o5_environment_v1_cluster_proto_init() }
func file_o5_environment_v1_cluster_proto_init() {
	if File_o5_environment_v1_cluster_proto != nil {
		return
	}
	file_o5_environment_v1_environment_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_o5_environment_v1_cluster_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cluster); i {
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
		file_o5_environment_v1_cluster_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CombinedConfig); i {
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
		file_o5_environment_v1_cluster_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ECSCluster); i {
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
		file_o5_environment_v1_cluster_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RDSHost); i {
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
	file_o5_environment_v1_cluster_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Cluster_EcsCluster)(nil),
	}
	file_o5_environment_v1_cluster_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*CombinedConfig_EcsCluster)(nil),
	}
	file_o5_environment_v1_cluster_proto_msgTypes[2].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_environment_v1_cluster_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_o5_environment_v1_cluster_proto_goTypes,
		DependencyIndexes: file_o5_environment_v1_cluster_proto_depIdxs,
		MessageInfos:      file_o5_environment_v1_cluster_proto_msgTypes,
	}.Build()
	File_o5_environment_v1_cluster_proto = out.File
	file_o5_environment_v1_cluster_proto_rawDesc = nil
	file_o5_environment_v1_cluster_proto_goTypes = nil
	file_o5_environment_v1_cluster_proto_depIdxs = nil
}
