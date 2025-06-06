package localrun

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/flowtest/prototest"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"google.golang.org/protobuf/proto"
)

type MockTemplateStore struct {
	Templates map[string]string
}

func (m *MockTemplateStore) PutTemplate(ctx context.Context, envName, appName, deploymentID string, template []byte) (*awsdeployer_pb.S3Template, error) {
	key := envName + appName + deploymentID
	m.Templates[key] = string(template)
	return &awsdeployer_pb.S3Template{
		Key:    key,
		Bucket: "foo",
		Region: "us-east-1",
	}, nil
}

type MockInfra struct {
	incoming        chan proto.Message
	outgoing        chan awsdeployer_pb.DeploymentPSMEvent
	stabalizeResult *awsinfra_tpb.StackStatusChangedMessage
}

func (m *MockInfra) StabalizeStack(ctx context.Context, msg *awsinfra_tpb.StabalizeStackMessage) (*awsinfra_tpb.StackStatusChangedMessage, error) {
	return m.stabalizeResult, nil
}

func (m *MockInfra) HandleMessage(ctx context.Context, msg proto.Message) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	fmt.Printf("MSG: %s\n", msg.ProtoReflect().Descriptor().FullName())
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case m.incoming <- msg:
		fmt.Printf("Released\n")
		msg := <-m.outgoing
		return &awsdeployer_pb.DeploymentPSMEventSpec{
			Keys:    &awsdeployer_pb.DeploymentKeys{},
			EventID: uuid.NewString(),
			Cause:   &psm_j5pb.Cause{},
			Event:   msg,
		}, nil
	}
}

func (m *MockInfra) CFResult(md *messaging_j5pb.RequestMetadata, status drss_pb.StepStatus, result *awsdeployer_pb.CFStackOutput) {
	m.StepResult(md, &awsdeployer_pb.DeploymentEventType_StepResult{
		Result: &drss_pb.StepResult{
			Status: status,
		},
		Output: &awsdeployer_pb.StepOutputType{
			Type: &awsdeployer_pb.StepOutputType_CfStackStatus{
				CfStackStatus: &awsdeployer_pb.StepOutputType_CFStackStatus{
					Lifecycle: result.Lifecycle,
					Outputs:   result.Outputs,
				},
			},
		},
	})
}

func (m *MockInfra) StepResult(md *messaging_j5pb.RequestMetadata, result *awsdeployer_pb.DeploymentEventType_StepResult) {
	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(md.Context, stepContext); err != nil {
		panic(err)
	}

	result.Result.StepId = *stepContext.StepId

	m.Send(result)
}

func (m *MockInfra) Send(msg awsdeployer_pb.DeploymentPSMEvent) {
	m.outgoing <- msg
}

func (m *MockInfra) Pop(t flowtest.TB, ctx context.Context) proto.Message {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	t.Helper()
	t.Logf("Pop Request")
	select {
	case <-ctx.Done():
		t.Logf("Pop Timeout\n")
		return nil
	case in := <-m.incoming:
		t.Logf("Popped %s\n", in.ProtoReflect().Descriptor().FullName())
		return in
	}
}

func TestLocalRun(t *testing.T) {
	runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	name := t.Name()
	ss := flowtest.NewStepper[*testing.T](name)

	spec := Spec{
		Version: "v1",
		AppConfig: &application_pb.Application{
			Name: "app",
			DeploymentConfig: &application_pb.DeploymentConfig{
				QuickMode: false,
			},
			Runtimes: []*application_pb.Runtime{{
				Name: "foo",
				Containers: []*application_pb.Container{{
					Name: "main",
					Source: &application_pb.Container_Image_{
						Image: &application_pb.Container_Image{
							Name: "foo",
						},
					},
				}},
			}},
			Databases: []*application_pb.Database{{
				Name: "foo",
				Engine: &application_pb.Database_Postgres_{
					Postgres: &application_pb.Database_Postgres{
						ServerGroup: "default",
						MigrateContainer: &application_pb.Container{
							Name: "foo",
							Source: &application_pb.Container_Image_{
								Image: &application_pb.Container_Image{
									Name: "foo",
								},
							},
						},
					},
				},
			}},
		},
		ClusterConfig: &environment_pb.Cluster{
			Name: "cluster",
			Provider: &environment_pb.Cluster_Aws{
				Aws: &environment_pb.AWSCluster{
					O5Sidecar: &environment_pb.O5Sidecar{
						ImageVersion: "version",
					},
					EventBridge: &environment_pb.EventBridge{
						EventBusArn: "arn:bus",
					},
					EcsCluster: &environment_pb.ECSCluster{
						ClusterName: "ecs-cluster",
					},
					O5Deployer: &environment_pb.O5Deployer{
						AssumeRoleArn: "arn:role",
					},
					AlbIngress: &environment_pb.ALBIngress{
						ListenerArn: "arn:listener",
					},
					RdsHosts: []*environment_pb.RDSHost{{
						ServerGroupName: "default",
						Endpoint:        "endpoint",
						Port:            5432,
						Auth: &environment_pb.RDSAuthType{
							Type: &environment_pb.RDSAuthType_SecretsManager_{
								SecretsManager: &environment_pb.RDSAuthType_SecretsManager{
									SecretName: "/env/root/secret",
								},
							},
						},
					}},
				},
			},
		},
		EnvConfig: &environment_pb.Environment{
			FullName: "env",
			Provider: &environment_pb.Environment_Aws{
				Aws: &environment_pb.AWSEnvironment{
					HostHeader: aws.String("host"),
				},
			},
		},
		ScratchBucket: "bucket",
	}

	templateStore := &MockTemplateStore{
		Templates: map[string]string{},
	}

	infra := &MockInfra{
		incoming: make(chan proto.Message),
		outgoing: make(chan awsdeployer_pb.DeploymentPSMEvent),
		stabalizeResult: &awsinfra_tpb.StackStatusChangedMessage{
			Lifecycle: awsdeployer_pb.CFLifecycle_MISSING,
		},
	}

	runErr := make(chan error, 1)
	ss.Setup(func(ctx context.Context, t flowtest.Asserter) error {
		go func() {
			err := RunLocalDeploy(runCtx, templateStore, infra, spec)
			if err != nil {
				fmt.Printf("RunLocalDeploy: %s\n", err)
			} else {
				fmt.Printf("RunLocalDeploy: success\n")
			}
			cancel()
			runErr <- err
		}()
		return nil
	})
	ss.Step("CreateNewStack", func(ctx context.Context, t flowtest.Asserter) {
		req, ok := infra.Pop(t, ctx).(*awsinfra_tpb.CreateNewStackMessage)
		if !ok {
			cancel()
			t.Fatalf("expected CreateNewStackMessage")
		}

		infra.CFResult(req.Request, drss_pb.StepStatus_DONE, &awsdeployer_pb.CFStackOutput{
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Outputs: []*awsdeployer_pb.KeyValue{{
				Name:  "MigrationTaskDefinitionFoo",
				Value: "arn:taskdef",
			}, {
				Name:  "DatabaseSecretFoo",
				Value: "/env/app/secret",
			}},
		})
	})

	ss.Step("UpdateStack", func(ctx context.Context, t flowtest.Asserter) {
		req, ok := infra.Pop(t, ctx).(*awsinfra_tpb.UpdateStackMessage)
		if !ok {
			cancel()
			t.Fatalf("expected UpsertPostgresDatabaseMessage")
		}

		infra.CFResult(req.Request, drss_pb.StepStatus_DONE, &awsdeployer_pb.CFStackOutput{
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Outputs: []*awsdeployer_pb.KeyValue{{
				Name:  "MigrationTaskDefinitionFoo",
				Value: "arn:taskdef",
			}, {
				Name:  "DatabaseSecretFoo",
				Value: "/env/app/secret",
			}},
		})
	})

	ss.Step("UpsertPostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.UpsertPostgresDatabaseMessage)
		if !ok {
			cancel()
			t.Fatalf("expected UpsertPostgresDatabaseMessage")
		}

		prototest.AssertEqualProto(t, msg.AppAccess, &awsdeployer_pb.RDSAppSpecType{
			Type: &awsdeployer_pb.RDSAppSpecType_AppSecret{
				AppSecret: &awsdeployer_pb.RDSAppSpecType_SecretsManager{
					AppSecretName: "/env/app/secret",
				},
			},
		})

		prototest.AssertEqualProto(t, msg.AdminHost, &awsdeployer_pb.RDSHostType{
			Type: &awsdeployer_pb.RDSHostType_SecretsManager_{
				SecretsManager: &awsdeployer_pb.RDSHostType_SecretsManager{
					SecretName: "/env/root/secret",
				},
			},
		})

		infra.StepResult(msg.Request, &awsdeployer_pb.DeploymentEventType_StepResult{
			Result: &drss_pb.StepResult{
				Status: drss_pb.StepStatus_DONE,
			},
		})
	})

	ss.Step("MigratePostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.RunECSTaskMessage)
		if !ok {
			cancel()
			t.Fatalf("expected MigratePostgresDatabaseMessage")
		}

		t.Equal("ecs-cluster", msg.Context.Cluster)

		infra.StepResult(msg.Request, &awsdeployer_pb.DeploymentEventType_StepResult{
			Result: &drss_pb.StepResult{
				Status: drss_pb.StepStatus_DONE,
			},
		})
	})

	ss.Step("CleanupPostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.CleanupPostgresDatabaseMessage)
		if !ok {
			cancel()
			t.Fatalf("expected CleanupPostgresDatabaseMessage")
		}

		infra.StepResult(msg.Request, &awsdeployer_pb.DeploymentEventType_StepResult{
			Result: &drss_pb.StepResult{
				Status: drss_pb.StepStatus_DONE,
			},
		})
	})

	ss.Step("ScaleUp", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.ScaleStackMessage)
		if !ok {
			cancel()
			t.Fatalf("expected ScaleStackMessage")
		}

		t.Equal(int32(1), msg.DesiredCount)

		infra.CFResult(msg.Request, drss_pb.StepStatus_DONE, &awsdeployer_pb.CFStackOutput{
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Outputs:   []*awsdeployer_pb.KeyValue{},
		})
	})

	ss.RunSteps(t)

	ss.Step("Final Wait", func(ctx context.Context, t flowtest.Asserter) {
		if err := <-runErr; err != nil {
			t.Fatal(err.Error())
		}
	})
}
