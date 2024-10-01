package states

import (
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/protostate/psm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func ValidateClusterOverrides(overrides []*awsdeployer_pb.ParameterOverride) error {
	descFields := (&environment_pb.ECSCluster{}).ProtoReflect().Descriptor().Fields()
	for _, override := range overrides {
		if override.Key == "" {
			return status.Error(codes.InvalidArgument, "key is empty")
		}
		field := descFields.ByJSONName(override.Key)
		if field == nil {
			return status.Errorf(codes.InvalidArgument, "field not found: %s", override.Key)
		}
		if field.IsMap() || field.IsList() || field.Kind() != protoreflect.StringKind {
			return status.Errorf(codes.InvalidArgument, "field not string: %s", override.Key)
		}
	}
	return nil
}

func mergeClusterConfig(base *environment_pb.Cluster, overrides []*awsdeployer_pb.ParameterOverride) (*environment_pb.Cluster, error) {
	if len(overrides) == 0 {
		return base, nil
	}

	merged := proto.Clone(base).(*environment_pb.Cluster)
	provider := merged.GetEcsCluster()
	if provider == nil {
		return nil, fmt.Errorf("cluster provider not set")
	}
	refl := provider.ProtoReflect()
	descFields := refl.Descriptor().Fields()
	for _, override := range overrides {
		field := descFields.ByJSONName(override.Key)
		if field == nil {
			return nil, fmt.Errorf("field not found: %s", override.Key)
		}
		if field.IsMap() || field.IsList() || field.Kind() != protoreflect.StringKind {
			return nil, fmt.Errorf("field not string: %s", override.Key)
		}
		refl.Set(field, protoreflect.ValueOfString(*override.Value))
	}
	return merged, nil
}
