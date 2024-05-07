package states

import (
	"context"

	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
)

func NewEnvironmentEventer() (*deployer_pb.EnvironmentPSM, error) {
	config := deployer_pb.DefaultEnvironmentPSMConfig().
		StoreExtraEventColumns(func(e *deployer_pb.EnvironmentEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":             e.Metadata.EventId,
				"environment_id": e.EnvironmentId,
				"timestamp":      e.Metadata.Timestamp,
			}, nil
		})

	sm, err := deployer_pb.NewEnvironmentPSM(config)
	if err != nil {
		return nil, err
	}

	sm.From(
		deployer_pb.EnvironmentStatus_UNSPECIFIED,
		deployer_pb.EnvironmentStatus_ACTIVE,
	).
		Do(deployer_pb.EnvironmentPSMFunc(func(
			ctx context.Context,
			tb deployer_pb.EnvironmentPSMTransitionBaton,
			state *deployer_pb.EnvironmentState,
			event *deployer_pb.EnvironmentEventType_Configured,
		) error {
			state.Config = event.Config
			state.Status = deployer_pb.EnvironmentStatus_ACTIVE
			return nil
		}))

	return sm, nil
}
