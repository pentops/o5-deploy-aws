package appbuilder

import (
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type PolicyBuilder struct {
	s3ReadWrite    []string
	s3ReadWriteAcl []string
	s3ReadOnly     []string
	s3WriteOnly    []string

	sqsSubscribe       []string
	sqsPublish         []string
	snsPublish         []string
	eventBridgePublish []string
	rdsConnectARNs     []string
	readSecretARNs     []string

	managedPolicyARNs []string

	ecrPull bool
}

func NewPolicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{}
}

func (pb *PolicyBuilder) AddBucketReadOnly(arn cflib.TemplateRef) {
	pb.s3ReadOnly = append(pb.s3ReadOnly, arn.Ref())
}

func (pb *PolicyBuilder) AddBucketWriteOnly(arn cflib.TemplateRef) {
	pb.s3WriteOnly = append(pb.s3WriteOnly, arn.Ref())
}

func (pb *PolicyBuilder) AddBucketReadWrite(arn cflib.TemplateRef) {
	pb.s3ReadWrite = append(pb.s3ReadWrite, arn.Ref())
}

func (pb *PolicyBuilder) AddBucketReadWriteAcl(arn cflib.TemplateRef) {
	pb.s3ReadWriteAcl = append(pb.s3ReadWriteAcl, arn.Ref())
}

func (pb *PolicyBuilder) AddSQSSubscribe(arn cflib.TemplateRef) {
	pb.sqsSubscribe = append(pb.sqsSubscribe, arn.Ref())
}

func (pb *PolicyBuilder) AddSQSPublish(arn string) {
	pb.sqsPublish = append(pb.sqsPublish, arn)
}

func (pb *PolicyBuilder) AddSNSPublish(arn string) {
	pb.snsPublish = append(pb.snsPublish, arn)
}

func (pb *PolicyBuilder) AddEventBridgePublish(topicName string) {
	pb.eventBridgePublish = append(pb.eventBridgePublish, topicName)
}

func (pb *PolicyBuilder) AddRDSConnect(arn string) {
	pb.rdsConnectARNs = append(pb.rdsConnectARNs, arn)
}

func (pb *PolicyBuilder) AddReadSecret(arn string) {
	pb.readSecretARNs = append(pb.readSecretARNs, arn)
}

func (pb *PolicyBuilder) AddManagedPolicyARN(arn string) {
	pb.managedPolicyARNs = append(pb.managedPolicyARNs, arn)
}

func (pb *PolicyBuilder) AddECRPull() {
	pb.ecrPull = true
}

type PolicyDocument struct {
	// Version of the policy document
	Version string `json:"Version"`
	// Statement is the list of statements in the policy document
	Statement []StatementEntry `json:"Statement"`
}

type StatementEntry struct {
	// Effect is the effect of the statement
	Effect string `json:"Effect"`
	// Action is the list of actions allowed or denied by the statement
	Action []string `json:"Action"`

	// Resource is the list of resources the statement applies to
	Resource []string `json:"Resource,omitempty"`

	Principal map[string]string `json:"Principal,omitempty"`
}

func (pb *PolicyBuilder) BuildRole(relativeName, familyName string) *iam.Role {
	rolePolicies := pb.Build(familyName)
	role := &iam.Role{
		AssumeRolePolicyDocument: PolicyDocument{
			Version: policyVersion,
			Statement: []StatementEntry{{
				Effect: "Allow",
				Principal: map[string]string{
					"Service": "ecs-tasks.amazonaws.com",
				},
				Action: []string{"sts:AssumeRole"},
			}},
		},
		Description:       cflib.Stringf("Execution role for ecs in %s", familyName),
		ManagedPolicyArns: pb.managedPolicyARNs,
		Policies:          rolePolicies,
		RoleName: cloudformation.JoinPtr("-", []string{
			cloudformation.Ref("AWS::StackName"),
			relativeName,
		}),
	}
	return role
}

const (
	PolicyNameReadSecrets        = "read-secrets"
	PolicyNameECRPull            = "ecr-pull"
	PolicyNameEventbridgePublish = "eventbridge-publish"
	PolicyNameSNSPublish         = "sns-publish"
	PolicyNameSQSPublish         = "sqs-publish"
	PolicyNameSQSSubscribe       = "sqs-subscribe"
	PolicyNameS3ReadwriteACL     = "s3-readwrite-acl"
	PolicyNameS3ReadWrite        = "s3-readwrite"
	PolicyNameS3ReadOnly         = "s3-read-only"
	PolicyNameS3WriteOnly        = "s3-write-only"
	PolicyNameRDSConnect         = "rds-connect"
)

const policyVersion = "2012-10-17"

func (pb *PolicyBuilder) Build(familyName string) []iam.Role_Policy {

	//accountIDRef := cloudformation.Ref("AWS::AccountId")
	envNameRef := cloudformation.Ref("EnvName")
	uniqueName := func(a string) string {
		return cloudformation.Join("-", []string{
			envNameRef,
			familyName,
			a,
		})
	}

	rolePolicies := make([]iam.Role_Policy, 0, 2)

	if len(pb.readSecretARNs) > 0 {

		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameReadSecrets),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"secretsmanager:GetSecretValue",
					},
					Resource: pb.readSecretARNs,
				}},
			},
		})
	}

	if pb.ecrPull {
		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameECRPull),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"ecr:GetAuthorizationToken",
						"ecr:BatchCheckLayerAvailability",
						"ecr:GetDownloadUrlForLayer",
						"ecr:GetRepositoryPolicy",
						"ecr:DescribeRepositories",
						"ecr:ListImages",
						"ecr:DescribeImages",
						"ecr:BatchGetImage",
					},
					Resource: []string{"*"},
				}},
			},
		})
	}

	if len(pb.eventBridgePublish) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameEventbridgePublish),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"events:PutEvents",
					},
					Resource: []string{cloudformation.Ref(EventBusARNParameter)},
				}},
			},
		}
		// TODO: Filter this down.
		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.snsPublish) > 0 {
		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameSNSPublish),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"SNS:Publish",
					},
					Resource: pb.snsPublish,
				},
				},
			},
		})
	}

	if len(pb.sqsPublish) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameSQSPublish),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"sqs:SendMessage",
						"sqs:GetQueueAttributes",
					},
					Resource: pb.sqsPublish,
				}, {
					Effect: "Allow",
					Action: []string{
						"sqs:ListQueues",
						"sqs:ListQueueTags",
					},
					Resource: []string{"*"},
				}},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}
	if len(pb.sqsSubscribe) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameSQSSubscribe),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"sqs:SendMessage",
						"sqs:ReceiveMessage",
						"sqs:DeleteMessage",
						"sqs:GetQueueAttributes",
						"sqs:ChangeMessageVisibility",
					},
					Resource: pb.sqsSubscribe,
				},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3ReadWriteAcl) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameS3ReadwriteACL),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
						"s3:GetBucketLocation",
					},
					Resource: pb.s3ReadWriteAcl,
				}, {
					Effect: "Allow",
					Action: []string{
						"s3:GetObject",
						"s3:PutObject",
						"s3:PutObjectAcl",
					},
					Resource: addS3TrailingSlash(pb.s3ReadWriteAcl),
				}},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3ReadWrite) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameS3ReadWrite),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
						"s3:GetBucketLocation",
					},
					Resource: pb.s3ReadWrite,
				}, {
					Effect: "Allow",
					Action: []string{
						"s3:GetObject",
						"s3:PutObject",
					},
					Resource: addS3TrailingSlash(pb.s3ReadWrite),
				}},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3ReadOnly) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameS3ReadOnly),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
						"s3:GetBucketLocation",
					},
					Resource: pb.s3ReadOnly,
				}, {
					Effect: "Allow",
					Action: []string{
						"s3:GetObject",
					},
					Resource: addS3TrailingSlash(pb.s3ReadOnly),
				}},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3WriteOnly) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameS3WriteOnly),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
						"s3:GetBucketLocation",
					},
					Resource: pb.s3WriteOnly,
				}, {
					Effect: "Allow",
					Action: []string{
						"s3:PutObject",
					},
					Resource: addS3TrailingSlash(pb.s3WriteOnly),
				}},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.rdsConnectARNs) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName(PolicyNameRDSConnect),
			PolicyDocument: PolicyDocument{
				Version: policyVersion,
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"rds-db:connect",
					},
					Resource: pb.rdsConnectARNs,
				}},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	return rolePolicies

}

func addS3TrailingSlash(in []string) []string {

	subResources := make([]string, 0, len(in)*2)

	for i := range in {
		//This represents all of the objects inside of the s3 buckets. Receiver of Get and Put permissions.
		subResources = append(subResources,
			in[i],
			cloudformation.Join("", []any{in[i], "/*"}),
		)
	}
	return subResources
}
