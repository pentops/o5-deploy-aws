package service

import (
	"context"
	"testing"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/apps/service/internal/states"
	"github.com/pentops/o5-deploy-aws/internal/integration/mocks"
	"github.com/pentops/pgtest.go/pgtest"
	"github.com/pentops/realms/authtest"
	"github.com/pentops/sqrlx.go/sqrlx"
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
	// TODO: Merge this with integration.TestConfigFlow

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		t.Fatal(err)
	}

	githubMock := mocks.NewGithub()

	conn := pgtest.GetTestDB(t,
		pgtest.WithDir("../../../ext/db"),
		pgtest.WithSchemaName("testservice"),
	)
	//outbox := outboxtest.NewOutboxAsserter(t, conn)

	ds, err := NewCommandService(sqrlx.NewPostgres(conn), nil, githubMock, stateMachines)
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
				Provider: &environment_pb.CombinedConfig_Aws{
					Aws: &environment_pb.AWSCluster{},
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
