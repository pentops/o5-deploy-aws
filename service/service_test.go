package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/integration/mocks"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/outbox.pg.go/outboxtest"
	"github.com/pentops/pgtest.go/pgtest"
)

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

	ds, err := NewDeployerService(conn, githubMock, stateMachines)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	environmentID := uuid.NewString()

	_, err = ds.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
		Config: &environment_pb.Environment{
			FullName: "test",
		},
		EnvironmentId: environmentID,
	})
	if err != nil {
		t.Fatal(err)
	}

	githubMock.Configs["owner/repo/commit"] = []*application_pb.Application{{
		Name: "app",
	}}

	_, err = ds.TriggerDeployment(ctx, &deployer_spb.TriggerDeploymentRequest{
		DeploymentId:    uuid.NewString(),
		EnvironmentName: "test",
		Source: &deployer_spb.TriggerDeploymentRequest_Github{
			Github: &deployer_spb.TriggerDeploymentRequest_GithubSource{
				Owner:  "owner",
				Repo:   "repo",
				Commit: "commit",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	reqMsg := &deployer_tpb.RequestDeploymentMessage{}
	outbox.PopMessage(t, reqMsg)

	if reqMsg.EnvironmentId != environmentID {
		t.Errorf("expected environment id %s, got %s", environmentID, reqMsg.EnvironmentId)
	}
	if reqMsg.Application.Name != "app" {
		t.Errorf("expected application name app, got %s", reqMsg.Application.Name)
	}

}
