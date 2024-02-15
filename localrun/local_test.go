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

func (m *MockInfra) Send(msg deployer_pb.DeploymentPSMEvent) {
	fmt.Printf("Send %s\n", msg.PSMEventKey())
	m.outgoing <- msg
	fmt.Printf("Sent %s\n", msg.PSMEventKey())
}

func (m *MockInfra) Pop(ctx context.Context) proto.Message {
	fmt.Printf("Pop\n")
	select {
	case <-ctx.Done():
		return nil
	case in := <-m.incoming:
		fmt.Printf("Popped %s\n", in.ProtoReflect().Descriptor().FullName())
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
					},
				},
			}},
		},
		EnvConfig: &environment_pb.Environment{
			FullName: "env",
			Provider: &environment_pb.Environment_Aws{
				Aws: &environment_pb.AWS{
					ListenerArn: "arn:listener",
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
		_, ok := infra.Pop(ctx).(*deployer_tpb.StabalizeStackMessage)
		if !ok {
			t.Fatalf("expected StabalizeStackMessage")
		}

		infra.Send(&deployer_pb.DeploymentEventType_StackStatus{
			Lifecycle: deployer_pb.StackLifecycle_MISSING,
		})
	})

	ss.StepC("CreateNewStack", func(ctx context.Context, t flowtest.Asserter) {
		_, ok := infra.Pop(ctx).(*deployer_tpb.CreateNewStackMessage)
		if !ok {
			t.Fatalf("expected CreateNewStackMessage")
		}

		infra.Send(&deployer_pb.DeploymentEventType_StackStatus{
			Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
		})
	})

	ss.StepC("MigratePostgresDatabase", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(ctx).(*deployer_tpb.MigratePostgresDatabaseMessage)
		if !ok {
			t.Fatalf("expected MigratePostgresDatabaseMessage")
		}

		t.Equal("foo", msg.MigrationSpec.Database.Database.Name)

		reqState := &deployer_pb.DeploymentMigrationContext{}
		err := proto.Unmarshal(msg.Request.Context, reqState)
		t.NoError(err)

		infra.Send(&deployer_pb.DeploymentEventType_DBMigrateStatus{
			MigrationId: reqState.MigrationId,
			Status:      deployer_pb.DatabaseMigrationStatus_COMPLETED,
		})
	})

	ss.StepC("ScaleUp", func(ctx context.Context, t flowtest.Asserter) {
		msg, ok := infra.Pop(ctx).(*deployer_tpb.ScaleStackMessage)
		if !ok {
			t.Fatalf("expected ScaleStackMessage")
		}

		t.Equal(int32(1), msg.DesiredCount)

		infra.Send(&deployer_pb.DeploymentEventType_StackStatus{
			Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
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
