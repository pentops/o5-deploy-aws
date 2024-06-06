package localrun

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
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
	incoming chan proto.Message
	outgoing chan awsdeployer_pb.DeploymentPSMEvent
}

func (m *MockInfra) HandleMessage(ctx context.Context, msg proto.Message) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
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
			Cause:   &psm_pb.Cause{},
			Event:   msg,
		}, nil
	}
}

func (m *MockInfra) CFResult(md *messaging_pb.RequestMetadata, status awsdeployer_pb.StepStatus, result *awsdeployer_pb.CFStackOutput) {
	m.StepResult(md, &awsdeployer_pb.DeploymentEventType_StepResult{
		Status: status,
		Output: &awsdeployer_pb.StepOutputType{
			Type: &awsdeployer_pb.StepOutputType_CfStatus{
				CfStatus: &awsdeployer_pb.StepOutputType_CFStatus{
					Output: result,
				},
			},
		},
	})
}

func (m *MockInfra) StepResult(md *messaging_pb.RequestMetadata, result *awsdeployer_pb.DeploymentEventType_StepResult) {
	stepContext := &awsdeployer_pb.StepContext{}
	if err := proto.Unmarshal(md.Context, stepContext); err != nil {
		panic(err)
	}

	result.StepId = *stepContext.StepId

	m.Send(result)
}

func (m *MockInfra) Send(msg awsdeployer_pb.DeploymentPSMEvent) {
	m.outgoing <- msg
}

func (m *MockInfra) Pop(t flowtest.TB, ctx context.Context) proto.Message {
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

	name := t.Name()
	ss := flowtest.NewStepper[*testing.T](name)

	spec := Spec{
		Version: "v1",
		AppConfig: &application_pb.Application{
			Name: "app",
			DeploymentConfig: &application_pb.DeploymentConfig{
				QuickMode: false,
			},
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
			Provider: &environment_pb.Cluster_EcsCluster{
				EcsCluster: &environment_pb.ECSCluster{
					EcsClusterName: "ecs-cluster",
					ListenerArn:    "arn:listener",
					RdsHosts: []*environment_pb.RDSHost{{
						ServerGroup: "default",
						SecretName:  "secret",
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
	}

	ss.Step("Stabalize", func(ctx context.Context, t flowtest.Asserter) {
		_, ok := infra.Pop(t, ctx).(*awsinfra_tpb.StabalizeStackMessage)
		if !ok {
			t.Fatalf("expected StabalizeStackMessage")
		}

		infra.Send(&awsdeployer_pb.DeploymentEventType_StackAvailable{
			StackOutput: nil,
		})
	})

	ss.Step("CreateNewStack", func(ctx context.Context, t flowtest.Asserter) {
		req, ok := infra.Pop(t, ctx).(*awsinfra_tpb.CreateNewStackMessage)
		if !ok {
			t.Fatalf("expected CreateNewStackMessage")
		}

		infra.CFResult(req.Request, awsdeployer_pb.StepStatus_DONE, &awsdeployer_pb.CFStackOutput{
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Outputs: []*awsdeployer_pb.KeyValue{{
				Name:  "MigrationTaskDefinitionFoo",
				Value: "arn:taskdef",
			}, {
				Name:  "DatabaseSecretFoo",
				Value: "arn:secret",
			}},
		})
	})

	ss.Step("UpsertPostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.UpsertPostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected UpsertPostgresDatabaseMessage")
		}

		t.Equal("arn:secret", msg.Spec.SecretArn)

		infra.StepResult(msg.Request, &awsdeployer_pb.DeploymentEventType_StepResult{
			Status: awsdeployer_pb.StepStatus_DONE,
		})
	})

	ss.Step("MigratePostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.MigratePostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected MigratePostgresDatabaseMessage")
		}

		t.Equal("ecs-cluster", msg.Spec.EcsClusterName)
		t.Equal("arn:secret", msg.Spec.SecretArn)
		t.Equal("arn:taskdef", msg.Spec.MigrationTaskArn)

		infra.StepResult(msg.Request, &awsdeployer_pb.DeploymentEventType_StepResult{
			Status: awsdeployer_pb.StepStatus_DONE,
		})
	})

	ss.Step("CleanupPostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.CleanupPostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected CleanupPostgresDatabaseMessage")
		}

		infra.StepResult(msg.Request, &awsdeployer_pb.DeploymentEventType_StepResult{
			Status: awsdeployer_pb.StepStatus_DONE,
		})
	})

	ss.Step("ScaleUp", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*awsinfra_tpb.ScaleStackMessage)
		if !ok {
			t.Fatalf("expected ScaleStackMessage")
		}

		t.Equal(int32(1), msg.DesiredCount)

		infra.CFResult(msg.Request, awsdeployer_pb.StepStatus_DONE, &awsdeployer_pb.CFStackOutput{
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Outputs:   []*awsdeployer_pb.KeyValue{},
		})
	})

	runCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runErr := make(chan error, 1)
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

	ss.RunSteps(t)

	if err := <-runErr; err != nil {
		t.Fatal(err.Error())
	}
}
