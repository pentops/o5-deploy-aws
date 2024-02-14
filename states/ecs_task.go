package states

import (
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

func NewEcsTaskEventer() (*deployer_pb.EcsTaskPSM, error) {
	config := deployer_pb.DefaultEcsTaskPSMConfig()
	sm, err := deployer_pb.NewEcsTaskPSM(config)
	if err != nil {
		return nil, err
	}

	return sm, nil
}
