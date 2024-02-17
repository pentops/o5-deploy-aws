package localrun

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pentops/flowtest"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"google.golang.org/protobuf/proto"
)

type MockTemplateStore struct {
	Templates map[string]string
}

func (m *MockTemplateStore) PutTemplate(ctx context.Context, envName, appName, deploymentID string, template []byte) (string, error) {
	key := envName + appName + deploymentID
	m.Templates[key] = string(template)
	return key, nil
}

type MockInfra struct {
	incoming chan proto.Message
	outgoing chan deployer_pb.DeploymentPSMEvent
}

func (m *MockInfra) HandleMessage(ctx context.Context, msg proto.Message) (deployer_pb.DeploymentPSMEvent, error) {
	fmt.Printf("MSG: %s\n", msg.ProtoReflect().Descriptor().FullName())
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case m.incoming <- msg:
		fmt.Printf("Released\n")
		return <-m.outgoing, nil
	}
}

func (m *MockInfra) CFResult(md *messaging_pb.RequestMetadata, status deployer_pb.StepStatus, result *deployer_pb.CFStackOutput) {
	m.StepResult(md, &deployer_pb.DeploymentEventType_StepResult{
		Status: status,
		Output: &deployer_pb.StepOutputType{
			Type: &deployer_pb.StepOutputType_CfStatus{
				CfStatus: &deployer_pb.StepOutputType_CFStatus{
					Output: result,
				},
			},
		},
	})
}

func (m *MockInfra) StepResult(md *messaging_pb.RequestMetadata, result *deployer_pb.DeploymentEventType_StepResult) {
	stepContext := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(md.Context, stepContext); err != nil {
		panic(err)
	}

	result.StepId = *stepContext.StepId

	m.Send(result)
}

func (m *MockInfra) Send(msg deployer_pb.DeploymentPSMEvent) {
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
		EnvConfig: &environment_pb.Environment{
			FullName: "env",
			Provider: &environment_pb.Environment_Aws{
				Aws: &environment_pb.AWS{
					ListenerArn:    "arn:listener",
					EcsClusterName: "ecs-cluster",
					RdsHosts: []*environment_pb.RDSHost{{
						ServerGroup: "default",
						SecretName:  "secret",
					}},
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
		outgoing: make(chan deployer_pb.DeploymentPSMEvent),
	}

	ss.StepC("Stabalize", func(ctx context.Context, t flowtest.Asserter) {
		_, ok := infra.Pop(t, ctx).(*deployer_tpb.StabalizeStackMessage)
		if !ok {
			t.Fatalf("expected StabalizeStackMessage")
		}

		infra.Send(&deployer_pb.DeploymentEventType_StackAvailable{
			StackOutput: nil,
		})
	})

	ss.StepC("CreateNewStack", func(ctx context.Context, t flowtest.Asserter) {
		req, ok := infra.Pop(t, ctx).(*deployer_tpb.CreateNewStackMessage)
		if !ok {
			t.Fatalf("expected CreateNewStackMessage")
		}

		infra.CFResult(req.Request, deployer_pb.StepStatus_DONE, &deployer_pb.CFStackOutput{
			Lifecycle: deployer_pb.CFLifecycle_COMPLETE,
			Outputs: []*deployer_pb.KeyValue{{
				Name:  "MigrationTaskDefinitionFoo",
				Value: "arn:taskdef",
			}, {
				Name:  "DatabaseSecretFoo",
				Value: "arn:secret",
			}},
		})
	})

	ss.StepC("UpsertPostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*deployer_tpb.UpsertPostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected UpsertPostgresDatabaseMessage")
		}

		t.Equal("arn:secret", msg.Spec.SecretArn)

		infra.StepResult(msg.Request, &deployer_pb.DeploymentEventType_StepResult{
			Status: deployer_pb.StepStatus_DONE,
		})
	})

	ss.StepC("MigratePostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*deployer_tpb.MigratePostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected MigratePostgresDatabaseMessage")
		}

		t.Equal("ecs-cluster", msg.Spec.EcsClusterName)
		t.Equal("arn:secret", msg.Spec.SecretArn)
		t.Equal("arn:taskdef", msg.Spec.MigrationTaskArn)

		infra.StepResult(msg.Request, &deployer_pb.DeploymentEventType_StepResult{
			Status: deployer_pb.StepStatus_DONE,
		})
	})

	ss.StepC("CleanupPostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*deployer_tpb.CleanupPostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected CleanupPostgresDatabaseMessage")
		}

		infra.StepResult(msg.Request, &deployer_pb.DeploymentEventType_StepResult{
			Status: deployer_pb.StepStatus_DONE,
		})
	})

	ss.StepC("ScaleUp", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(t, ctx).(*deployer_tpb.ScaleStackMessage)
		if !ok {
			t.Fatalf("expected ScaleStackMessage")
		}

		t.Equal(int32(1), msg.DesiredCount)

		infra.CFResult(msg.Request, deployer_pb.StepStatus_DONE, &deployer_pb.CFStackOutput{
			Lifecycle: deployer_pb.CFLifecycle_COMPLETE,
			Outputs:   []*deployer_pb.KeyValue{},
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

	ss.RunStepsC(runCtx, t)

	if err := <-runErr; err != nil {
		t.Fatal(err.Error())
	}
}
