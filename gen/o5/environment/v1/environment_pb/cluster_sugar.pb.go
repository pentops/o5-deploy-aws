// Code generated by protoc-gen-go-sugar. DO NOT EDIT.

package environment_pb

import (
	proto "google.golang.org/protobuf/proto"
)

type IsCluster_Provider = isCluster_Provider
type IsCombinedConfig_Provider = isCombinedConfig_Provider

// RDSAuthType is a oneof wrapper
type RDSAuthTypeKey string

const (
	RDSAuth_SecretsManager RDSAuthTypeKey = "secretsManager"
	RDSAuth_Iam            RDSAuthTypeKey = "iam"
)

func (x *RDSAuthType) TypeKey() (RDSAuthTypeKey, bool) {
	switch x.Type.(type) {
	case *RDSAuthType_SecretsManager_:
		return RDSAuth_SecretsManager, true
	case *RDSAuthType_Iam:
		return RDSAuth_Iam, true
	default:
		return "", false
	}
}

type IsRDSAuthTypeWrappedType interface {
	TypeKey() RDSAuthTypeKey
	proto.Message
}

func (x *RDSAuthType) Set(val IsRDSAuthTypeWrappedType) {
	switch v := val.(type) {
	case *RDSAuthType_SecretsManager:
		x.Type = &RDSAuthType_SecretsManager_{SecretsManager: v}
	case *RDSAuthType_IAM:
		x.Type = &RDSAuthType_Iam{Iam: v}
	}
}
func (x *RDSAuthType) Get() IsRDSAuthTypeWrappedType {
	switch v := x.Type.(type) {
	case *RDSAuthType_SecretsManager_:
		return v.SecretsManager
	case *RDSAuthType_Iam:
		return v.Iam
	default:
		return nil
	}
}
func (x *RDSAuthType_SecretsManager) TypeKey() RDSAuthTypeKey {
	return RDSAuth_SecretsManager
}
func (x *RDSAuthType_IAM) TypeKey() RDSAuthTypeKey {
	return RDSAuth_Iam
}

type IsRDSAuthType_Type = isRDSAuthType_Type
