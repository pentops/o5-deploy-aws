package states

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/protostate/psm"
)

func NewClusterEventer() (*deployer_pb.ClusterPSM, error) {
	config := deployer_pb.DefaultClusterPSMConfig().
		SystemActor(psm.MustSystemActor("D777D42C-3FE6-4A0D-9F70-5BC1348516F5"))

	sm, err := deployer_pb.NewClusterPSM(config)
	if err != nil {
		return nil, err
	}

	sm.From(
		deployer_pb.ClusterStatus_UNSPECIFIED,
		deployer_pb.ClusterStatus_ACTIVE,
	).
		OnEvent(deployer_pb.ClusterPSMEventConfigured).
		SetStatus(deployer_pb.ClusterStatus_ACTIVE).
		Mutate(deployer_pb.ClusterPSMMutation(func(
			state *deployer_pb.ClusterStateData,
			event *deployer_pb.ClusterEventType_Configured,
		) error {
			state.Config = event.Config
			return nil
		}))

	return sm, nil
}
