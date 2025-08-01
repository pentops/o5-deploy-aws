package states

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
)

func NewEnvironmentEventer() (*awsdeployer_pb.EnvironmentPSM, error) {
	sm, err := awsdeployer_pb.EnvironmentPSMBuilder().
		BuildStateMachine()
	if err != nil {
		return nil, err
	}

	sm.From(
		awsdeployer_pb.EnvironmentStatus_UNSPECIFIED,
		awsdeployer_pb.EnvironmentStatus_ACTIVE,
	).
		OnEvent(awsdeployer_pb.EnvironmentPSMEventConfigured).
		SetStatus(awsdeployer_pb.EnvironmentStatus_ACTIVE).
		Mutate(awsdeployer_pb.EnvironmentPSMMutation(func(
			state *awsdeployer_pb.EnvironmentStateData,
			event *awsdeployer_pb.EnvironmentEventType_Configured,
		) error {
			state.Config = event.Config
			return nil
		}))

	return sm, nil
}
