package states

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/protostate/psm"
)

func NewEnvironmentEventer() (*awsdeployer_pb.EnvironmentPSM, error) {
	sm, err := awsdeployer_pb.EnvironmentPSMBuilder().
		SystemActor(psm.MustSystemActor("216B6C2E-D996-492C-B80C-9AAD0CCFEEC4")).
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
