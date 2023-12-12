package app

import (
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

type PolicyBuilder struct {
	s3ReadWrite    []string
	s3ReadWriteAcl []string
	s3ReadOnly     []string
	sqsSubscribe   []string
	sqsPublish     []string
	snsPublish     []string
	ses            *application_pb.AWSConfig_SES

	metaDeployPermissions bool
}

func NewPolicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{}
}

func (pb *PolicyBuilder) AddBucketReadOnly(arn string) {
	pb.s3ReadOnly = append(pb.s3ReadOnly, arn)
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

func (pb *PolicyBuilder) AddSES(policy *application_pb.AWSConfig_SES) {
	pb.ses = policy
}

func (pb *PolicyBuilder) AddMetaDeployPermissions() {
	pb.metaDeployPermissions = true
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
		PolicyDocument: map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []interface{}{
				map[string]interface{}{
					"Effect": "Allow",
					"Action": []interface{}{
						"secretsmanager:GetSecretValue",
					},
					"Resource": cloudformation.Join("", []string{
						"arn:aws:secretsmanager:us-east-1:",
						accountIDRef,
						":secret:/",
						envNameRef,
						"/",
						appName,
						"/*",
					},
					),
				},
			},
		},
	})

	if pb.ses != nil {
		if pb.ses.SendEmail {
			rolePolicies = append(rolePolicies, iam.Role_Policy{
				PolicyName:     uniqueName("ses"),
				PolicyDocument: cloudformation.Ref(SESConditionsParameter),
				/* TODO: Partially build the policy document here with a ref to
				* the conditions. Unclear why the below did not work, the error
				* given is "Resource handler returned message: "Syntax errors in policy.
				* providing no use at all...

				* When fixing, also make sure the parameter is only the
				* condition subet, currently the parameter is set to the whole
				* policy which is not ideal

					map[string]interface{}{
						"Version": "2012-10-17",
						"Statement": []interface{}{
							map[string]interface{}{
								"Effect":   "Allow",
								"Resource": []interface{}{"*"}, // constrained by conditions
								"Action": []interface{}{
									"ses:SendEmail",
								},
								"Condition": cloudformation.Ref(SESConditionsParameter),
							},
						},
					},
				*/
			})
		}
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
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   "s3:ListBucket",
						"Resource": pb.s3ReadWriteAcl,
					},
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"s3:GetObject",
							"s3:PutObject",
							"s3:PutObjectAcl",
						},
						"Resource": addS3TrailingSlash(pb.s3ReadWriteAcl),
					},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3ReadWrite) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("s3-readwrite"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   "s3:ListBucket",
						"Resource": pb.s3ReadWrite,
					},
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"s3:GetObject",
							"s3:PutObject",
						},
						"Resource": addS3TrailingSlash(pb.s3ReadWrite),
					},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if len(pb.s3ReadOnly) > 0 {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("s3-readonly"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect":   "Allow",
						"Action":   "s3:ListBucket",
						"Resource": pb.s3ReadOnly,
					},
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"s3:GetObject",
						},
						"Resource": addS3TrailingSlash(pb.s3ReadOnly),
					},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)
	}

	if pb.metaDeployPermissions {
		policy := iam.Role_Policy{
			PolicyName: uniqueName("meta-deploy-permissions"),
			PolicyDocument: map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []interface{}{
					map[string]interface{}{
						"Effect": "Allow",
						"Action": []interface{}{
							"sts:AssumeRole",
						},
						"Resource": cloudformation.Split(",", cloudformation.Ref(MetaDeployAssumeRoleParameter)),
					},
				},
			},
		}

		rolePolicies = append(rolePolicies, policy)

	}

	return rolePolicies

}

func addS3TrailingSlash(in []string) []interface{} {

	subResources := make([]interface{}, 0, len(in)*2)

	for i := range in {
		//This represents all of the objects inside of the s3 buckets. Receiver of Get and Put permissions.
		subResources = append(subResources,
			in[i],
			cloudformation.Join("", []interface{}{in[i], "/*"}),
		)
	}
	return subResources
}
