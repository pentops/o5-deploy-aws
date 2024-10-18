package integration

import (
	"context"
	"testing"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/realms/authtest"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

func TestConfigFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	ss.Step("Configure Base", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		_, err := t.DeployerCommand.UpsertCluster(ctx, &awsdeployer_spb.UpsertClusterRequest{
			ClusterId: "cluster",
			Src: &awsdeployer_spb.UpsertClusterRequest_Config{

				Config: &environment_pb.CombinedConfig{
					Name: "cluster",
					Provider: &environment_pb.CombinedConfig_Aws{
						Aws: &environment_pb.AWSCluster{
							EcsCluster: &environment_pb.ECSCluster{
								ClusterName: "cluster",
							},
							GlobalNamespace: "g1",
							O5Sidecar: &environment_pb.O5Sidecar{
								ImageVersion: "version1",
							},
						},
					},
				},
			},
		})
		t.NoError(err)
	})

	ss.Step("Get Cluster", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		resp, err := t.ClusterQuery.GetCluster(ctx, &awsdeployer_spb.GetClusterRequest{
			ClusterId: "cluster",
		})
		t.NoError(err)
		t.Equal("cluster", resp.State.Data.Config.Name)
		t.Equal("g1", resp.State.Data.Config.GetAws().GlobalNamespace)
		t.Equal("version1", resp.State.Data.Config.GetAws().GetO5Sidecar().ImageVersion)
	})

	ss.Step("Configure Override", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		_, err := t.DeployerCommand.SetClusterOverride(ctx, &awsdeployer_spb.SetClusterOverrideRequest{
			ClusterId: "cluster",
			Overrides: []*awsdeployer_pb.ParameterOverride{{
				Key:   "sidecarImageVersion",
				Value: proto.String("version2"),
			}},
		})
		t.NoError(err)
	})

	ss.Step("Get Cluster", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		resp, err := t.ClusterQuery.GetCluster(ctx, &awsdeployer_spb.GetClusterRequest{
			ClusterId: "cluster",
		})
		t.NoError(err)
		t.Equal("cluster", resp.State.Data.Config.Name)
		t.Equal("g1", resp.State.Data.Config.GetAws().GlobalNamespace)
		t.Equal("version2", resp.State.Data.Config.GetAws().GetO5Sidecar().ImageVersion)
	})

	ss.Step("Remove Override", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		_, err := t.DeployerCommand.SetClusterOverride(ctx, &awsdeployer_spb.SetClusterOverrideRequest{
			ClusterId: "cluster",
			Overrides: []*awsdeployer_pb.ParameterOverride{{
				Key:   "sidecarImageVersion",
				Value: nil,
			}},
		})
		t.NoError(err)
	})

	ss.Step("Get Cluster", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		resp, err := t.ClusterQuery.GetCluster(ctx, &awsdeployer_spb.GetClusterRequest{
			ClusterId: "cluster",
		})
		t.NoError(err)
		t.Equal("cluster", resp.State.Data.Config.Name)
		t.Equal("g1", resp.State.Data.Config.GetAws().GlobalNamespace)
		t.Equal("version1", resp.State.Data.Config.GetAws().GetO5Sidecar().ImageVersion)
	})

	ss.Step("Invalid Override", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		_, err := t.DeployerCommand.SetClusterOverride(ctx, &awsdeployer_spb.SetClusterOverrideRequest{
			ClusterId: "cluster",
			Overrides: []*awsdeployer_pb.ParameterOverride{{
				Key:   "asdf",
				Value: proto.String("version2"),
			}},
		})
		t.CodeError(err, codes.InvalidArgument)
	})
}
