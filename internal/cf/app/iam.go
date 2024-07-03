package app

import (
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
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
	ecrPull            bool
}

func NewPolicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{}
}

func (pb *PolicyBuilder) AddBucketReadOnly(arn string) {
	pb.s3ReadOnly = append(pb.s3ReadOnly, arn)
}

func (pb *PolicyBuilder) AddBucketWriteOnly(arn string) {
	pb.s3WriteOnly = append(pb.s3WriteOnly, arn)
}

func (pb *PolicyBuilder) AddBucketReadWrite(arn string) {
	pb.s3ReadWrite = append(pb.s3ReadWrite, arn)
}

func (pb *PolicyBuilder) AddBucketReadWriteAcl(arn string) {
	pb.s3ReadWriteAcl = append(pb.s3ReadWriteAcl, arn)
}

func (pb *PolicyBuilder) AddSQSSubscribe(arn string) {
	pb.sqsSubscribe = append(pb.sqsSubscribe, arn)
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
	Resource []string `json:"Resource"`
}

func (pb *PolicyBuilder) Build(appName string, runtimeName string) []iam.Role_Policy {

	accountIDRef := cloudformation.Ref("AWS::AccountId")
	envNameRef := cloudformation.Ref("EnvName")
	uniqueName := func(a string) string {
		return cloudformation.Join("-", []string{
			envNameRef,
			appName,
			runtimeName,
			a,
		})
	}

	rolePolicies := make([]iam.Role_Policy, 0, 2)

	rolePolicies = append(rolePolicies, iam.Role_Policy{
		PolicyName: uniqueName("secrets"),
		PolicyDocument: PolicyDocument{
			Version: "2012-10-17",
			Statement: []StatementEntry{{
				Effect: "Allow",
				Action: []string{
					"secretsmanager:GetSecretValue",
				},
				Resource: []string{cloudformation.Join("", []string{
					"arn:aws:secretsmanager:us-east-1:",
					accountIDRef,
					":secret:/",
					envNameRef,
					"/",
					appName,
					"/*",
				})},
			}},
		},
	})

	if pb.ecrPull {
		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName: uniqueName("ecr-pull"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"ecr:GetAuthorizationToken",
							"ecr:BatchCheckLayerAvailability",
							"ecr:GetDownloadUrlForLayer",
							"ecr:GetRepositoryPolicy",
							"ecr:DescribeRepositories",
							"ecr:ListImages",
							"ecr:DescribeImages",
							"ecr:BatchGetImage",
						},
						"Resource": "*",
					},
				},
			},
		})
	}

	if len(pb.eventBridgePublish) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("eventbridge-publish"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"events:PutEvents",
						},
						"Resource": []string{cloudformation.Ref(EventBusARNParameter)},
					},
				},
			},
		}
		// TODO: Filter this down.
		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.snsPublish) > 0 {
		rolePolicies = append(rolePolicies, iam.Role_Policy{
			PolicyName: uniqueName("sns"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"SNS:Publish",
						},
						"Resource": pb.snsPublish,
					},
				},
			},
		})
	}

	if len(pb.sqsPublish) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("sqs-publish"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"sqs:SendMessage",
							"sqs:GetQueueAttributes",
						},
						"Resource": pb.sqsPublish,
					},
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"sqs:ListQueues",
							"sqs:ListQueueTags",
						},
						"Resource": "*",
					},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}
	if len(pb.sqsSubscribe) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("sqs-subscribe"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"sqs:SendMessage",
							"sqs:ReceiveMessage",
							"sqs:DeleteMessage",
							"sqs:GetQueueAttributes",
							"sqs:ChangeMessageVisibility",
						},
						"Resource": pb.sqsSubscribe,
					},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3ReadWriteAcl) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("s3-readwrite-acl"),
			PolicyDocument: PolicyDocument{
				Version: "2012-10-17",
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
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
			PolicyName: uniqueName("s3-readwrite"),
			PolicyDocument: PolicyDocument{
				Version: "2012-10-17",
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
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
			PolicyName: uniqueName("s3-readonly"),
			PolicyDocument: PolicyDocument{
				Version: "2012-10-17",
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
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
			PolicyName: uniqueName("s3-writeonly"),
			PolicyDocument: PolicyDocument{
				Version: "2012-10-17",
				Statement: []StatementEntry{{
					Effect: "Allow",
					Action: []string{
						"s3:ListBucket",
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

	return rolePolicies

}

func addS3TrailingSlash(in []string) []string {

	subResources := make([]string, 0, len(in)*2)

	for i := range in {
		//This represents all of the objects inside of the s3 buckets. Receiver of Get and Put permissions.
		subResources = append(subResources,
			in[i],
			cloudformation.Join("", []interface{}{in[i], "/*"}),
		)
	}
	return subResources
}
