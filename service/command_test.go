package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/integration/mocks"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/outbox.pg.go/outboxtest"
	"github.com/pentops/pgtest.go/pgtest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func assertCodeError(t *testing.T, err error, code codes.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", code)
	}
	codeErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected error %s, got %v", code, err)
	}
	if codeErr.Code() != code {
		t.Fatalf("expected error %s, got %s", code, codeErr.Code())
	}
}

func TestConfiguration(t *testing.T) {

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		t.Fatal(err)
	}

	githubMock := mocks.NewGithub()

	conn := pgtest.GetTestDB(t,
		pgtest.WithDir("../ext/db"),
		pgtest.WithSchemaName("testservice"),
	)
	//outbox := outboxtest.NewOutboxAsserter(t, conn)

	ds, err := NewCommandService(conn, githubMock, stateMachines)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	{ // Attempt to upsert stack before the environment exists.
		_, err := ds.UpsertStack(ctx, &deployer_spb.UpsertStackRequest{
			StackId: "test-app",
			Config:  &deployer_pb.StackConfig{},
		})
		assertCodeError(t, err, codes.NotFound)
	}

	_, err = ds.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
		Src: &deployer_spb.UpsertEnvironmentRequest_Config{
			Config: &environment_pb.Environment{
				FullName: "test",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := ds.UpsertStack(ctx, &deployer_spb.UpsertStackRequest{
		StackId: "test-app",
		Config:  &deployer_pb.StackConfig{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.State.Status != deployer_pb.StackStatus_AVAILABLE {
		t.Errorf("expected status %s, got %s", deployer_pb.StackStatus_AVAILABLE, res.State.Status)
	}

}

func TestTriggerDeployment(t *testing.T) {

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		t.Fatal(err)
	}

	githubMock := mocks.NewGithub()

	conn := pgtest.GetTestDB(t,
		pgtest.WithDir("../ext/db"),
		pgtest.WithSchemaName("testservice"),
	)
	outbox := outboxtest.NewOutboxAsserter(t, conn)

	ds, err := NewCommandService(conn, githubMock, stateMachines)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	_, err = ds.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
		Src: &deployer_spb.UpsertEnvironmentRequest_Config{
			Config: &environment_pb.Environment{
				FullName: "test",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	githubMock.Configs["owner/repo/commit"] = []*application_pb.Application{{
		Name: "app",
	}}

	triggerRes, err := ds.TriggerDeployment(ctx, &deployer_spb.TriggerDeploymentRequest{
		DeploymentId: uuid.NewString(),
		Environment:  "test",
		Source: &deployer_spb.TriggerSource{
			Type: &deployer_spb.TriggerSource_Github{
				Github: &deployer_spb.TriggerSource_GithubSource{
					Owner: "owner",
					Repo:  "repo",
					Ref: &deployer_spb.TriggerSource_GithubSource_Commit{
						Commit: "commit",
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	reqMsg := &deployer_tpb.RequestDeploymentMessage{}
	outbox.PopMessage(t, reqMsg)

	if reqMsg.EnvironmentId != triggerRes.EnvironmentId {
		t.Errorf("expected environment id %s, got %s", triggerRes.EnvironmentId, reqMsg.EnvironmentId)
	}
	if reqMsg.Application.Name != "app" {
		t.Errorf("expected application name app, got %s", reqMsg.Application.Name)
	}

}
