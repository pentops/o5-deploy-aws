package states

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/protostate/psm"
)

func NewClusterEventer() (*awsdeployer_pb.ClusterPSM, error) {
	config := awsdeployer_pb.DefaultClusterPSMConfig().
		SystemActor(psm.MustSystemActor("D777D42C-3FE6-4A0D-9F70-5BC1348516F5"))

	sm, err := awsdeployer_pb.NewClusterPSM(config)
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
			state.Config = event.Config
			return nil
		}))

	return sm, nil
}
