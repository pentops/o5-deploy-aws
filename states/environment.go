package states

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/protostate/psm"
)

func NewEnvironmentEventer() (*deployer_pb.EnvironmentPSM, error) {
	config := deployer_pb.DefaultEnvironmentPSMConfig().
		StoreExtraEventColumns(func(e *deployer_pb.EnvironmentEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":             e.Metadata.EventId,
				"environment_id": e.Keys.EnvironmentId,
				"timestamp":      e.Metadata.Timestamp,
			}, nil
		}).
		SystemActor(psm.MustSystemActor("216B6C2E-D996-492C-B80C-9AAD0CCFEEC4"))

	sm, err := deployer_pb.NewEnvironmentPSM(config)
	if err != nil {
		return nil, err
	}

	sm.From(
		deployer_pb.EnvironmentStatus_UNSPECIFIED,
		deployer_pb.EnvironmentStatus_ACTIVE,
	).
		OnEvent(deployer_pb.EnvironmentPSMEventConfigured).
		SetStatus(deployer_pb.EnvironmentStatus_ACTIVE).
		Mutate(deployer_pb.EnvironmentPSMMutation(func(
			state *deployer_pb.EnvironmentStateData,
			event *deployer_pb.EnvironmentEventType_Configured,
		) error {
			state.Config = event.Config
			return nil
		}))

	return sm, nil
}
