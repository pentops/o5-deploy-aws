package states

import (
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func NewClusterEventer() (*awsdeployer_pb.ClusterPSM, error) {
	sm, err := awsdeployer_pb.ClusterPSMBuilder().
		SystemActor(psm.MustSystemActor("D777D42C-3FE6-4A0D-9F70-5BC1348516F5")).BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(
		awsdeployer_pb.ClusterStatus_UNSPECIFIED,
		awsdeployer_pb.ClusterStatus_ACTIVE,
	).
		OnEvent(awsdeployer_pb.ClusterPSMEventConfigured).
		SetStatus(awsdeployer_pb.ClusterStatus_ACTIVE).
		Mutate(awsdeployer_pb.ClusterPSMMutation(func(
			state *awsdeployer_pb.ClusterStateData,
			event *awsdeployer_pb.ClusterEventType_Configured,
		) error {
			var err error
			state.BaseConfig = event.Config
			state.Config, err = mergeClusterConfig(state.BaseConfig, state.Overrides)
			if err != nil {
				return fmt.Errorf("failed to merge cluster config: %w", err)
			}
			return nil
		}))

	sm.From(awsdeployer_pb.ClusterStatus_ACTIVE).
		OnEvent(awsdeployer_pb.ClusterPSMEventOverride).
		Mutate(awsdeployer_pb.ClusterPSMMutation(func(
			state *awsdeployer_pb.ClusterStateData,
			event *awsdeployer_pb.ClusterEventType_Override,
		) error {
			var err error
			state.Overrides = applyOverrides(state.Overrides, event.Overrides)
			state.Config, err = mergeClusterConfig(state.BaseConfig, state.Overrides)
			if err != nil {
				return fmt.Errorf("failed to merge cluster config: %w", err)
			}
			return nil
		}))

	return sm, nil
}

func sliceToMap[T any](s []*T, keyFn func(*T) string) map[string]*T {
	m := make(map[string]*T, len(s))
	for _, item := range s {
		m[keyFn(item)] = item
	}
	return m
}

func applyOverrides(base, in []*awsdeployer_pb.ParameterOverride) []*awsdeployer_pb.ParameterOverride {
	overrideMap := sliceToMap(in, func(p *awsdeployer_pb.ParameterOverride) string { return p.Key })
	out := make([]*awsdeployer_pb.ParameterOverride, 0, len(base))
	for _, p := range base {
		override, ok := overrideMap[p.Key]
		if !ok {
			out = append(out, p)
		}
		if override.Value != nil {
			out = append(out, override)
		}
		delete(overrideMap, p.Key)
	}

	for _, p := range overrideMap {
		out = append(out, p)
	}
	return out
}

var clusterOverrides = map[string]func(*environment_pb.AWSCluster, string){
	"sidecarImageVersion": func(c *environment_pb.AWSCluster, v string) {
		c.O5Sidecar.ImageVersion = v
	},
}

func ValidateClusterOverrides(overrides []*awsdeployer_pb.ParameterOverride) error {
	for _, override := range overrides {
		if override.Key == "" {
			return status.Error(codes.InvalidArgument, "key is empty")
		}
		_, ok := clusterOverrides[override.Key]
		if !ok {
			return status.Errorf(codes.InvalidArgument, "override not found: %s", override.Key)
		}
	}
	return nil
}

func mergeClusterConfig(base *environment_pb.Cluster, overrides []*awsdeployer_pb.ParameterOverride) (*environment_pb.Cluster, error) {
	if len(overrides) == 0 {
		return base, nil
	}

	merged := proto.Clone(base).(*environment_pb.Cluster)
	provider := merged.GetAws()
	if provider == nil {
		return nil, fmt.Errorf("aws provider not set")
	}
	for _, override := range overrides {
		fn, ok := clusterOverrides[override.Key]
		if !ok {
			return nil, fmt.Errorf("override not found: %s", override.Key)
		}
		fn(provider, *override.Value)

	}
	return merged, nil
}
