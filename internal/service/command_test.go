package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/o5-auth/authtest"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/integration/mocks"
	"github.com/pentops/o5-deploy-aws/internal/states"
	"github.com/pentops/o5-messaging/outbox/outboxtest"
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
		pgtest.WithDir("../../ext/db"),
		pgtest.WithSchemaName("testservice"),
	)
	//outbox := outboxtest.NewOutboxAsserter(t, conn)

	ds, err := NewCommandService(conn, githubMock, stateMachines)
	if err != nil {
		t.Fatal(err)
	}

	ctx := authtest.ActionContext(context.Background())

	{ // Attempt to upsert stack before the environment exists.
		_, err := ds.UpsertStack(ctx, &awsdeployer_spb.UpsertStackRequest{
			StackId: "test-app",
		})
		assertCodeError(t, err, codes.NotFound)
	}

	{ // Attempt to upsert the environment before the cluster exists
		_, err = ds.UpsertEnvironment(ctx, &awsdeployer_spb.UpsertEnvironmentRequest{
			EnvironmentId: "test",
			ClusterId:     "cluster",
			Src: &awsdeployer_spb.UpsertEnvironmentRequest_Config{
				Config: &environment_pb.Environment{
					FullName: "test",
				},
			},
		})
		assertCodeError(t, err, codes.NotFound)
	}

	_, err = ds.UpsertCluster(ctx, &awsdeployer_spb.UpsertClusterRequest{
		ClusterId: "cluster",
		Src: &awsdeployer_spb.UpsertClusterRequest_Config{
			Config: &environment_pb.CombinedConfig{
				Name: "cluster",
				Provider: &environment_pb.CombinedConfig_EcsCluster{
					EcsCluster: &environment_pb.ECSCluster{},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = ds.UpsertEnvironment(ctx, &awsdeployer_spb.UpsertEnvironmentRequest{
		EnvironmentId: "test",
		ClusterId:     "cluster",
		Src: &awsdeployer_spb.UpsertEnvironmentRequest_Config{
			Config: &environment_pb.Environment{
				FullName: "test",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := ds.UpsertStack(ctx, &awsdeployer_spb.UpsertStackRequest{
		StackId: "test-app",
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.State.Status != awsdeployer_pb.StackStatus_AVAILABLE {
		t.Errorf("expected status %s, got %s", awsdeployer_pb.StackStatus_AVAILABLE, res.State.Status)
	}

}

func TestTriggerDeployment(t *testing.T) {

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		t.Fatal(err)
	}

	githubMock := mocks.NewGithub()

	conn := pgtest.GetTestDB(t,
		pgtest.WithDir("../../ext/db"),
		pgtest.WithSchemaName("testservice"),
	)
	outbox := outboxtest.NewOutboxAsserter(t, conn)

	ds, err := NewCommandService(conn, githubMock, stateMachines)
	if err != nil {
		t.Fatal(err)
	}

	ctx := authtest.ActionContext(context.Background())

	_, err = ds.UpsertCluster(ctx, &awsdeployer_spb.UpsertClusterRequest{
		ClusterId: "cluster",
		Src: &awsdeployer_spb.UpsertClusterRequest_Config{
			Config: &environment_pb.CombinedConfig{
				Name: "cluster",
				Provider: &environment_pb.CombinedConfig_EcsCluster{
					EcsCluster: &environment_pb.ECSCluster{},
				},
				Environments: []*environment_pb.Environment{{
					FullName: "test",
				}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	githubMock.Configs["owner/repo/commit"] = []*application_pb.Application{{
		Name: "app",
	}}

	triggerRes, err := ds.TriggerDeployment(ctx, &awsdeployer_spb.TriggerDeploymentRequest{
		DeploymentId: uuid.NewString(),
		Environment:  "test",
		Source: &awsdeployer_spb.TriggerSource{
			Type: &awsdeployer_spb.TriggerSource_Github{
				Github: &awsdeployer_spb.TriggerSource_GithubSource{
					Owner: "owner",
					Repo:  "repo",
					Ref: &awsdeployer_spb.TriggerSource_GithubSource_Commit{
						Commit: "commit",
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	reqMsg := &awsdeployer_tpb.RequestDeploymentMessage{}
	outbox.PopMessage(t, reqMsg)

	if reqMsg.EnvironmentId != triggerRes.EnvironmentId {
		t.Errorf("expected environment id %s, got %s", triggerRes.EnvironmentId, reqMsg.EnvironmentId)
	}
	if reqMsg.Application.Name != "app" {
		t.Errorf("expected application name app, got %s", reqMsg.Application.Name)
	}

}
