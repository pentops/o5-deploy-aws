package states

import (
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

func NewPostgresMigrateEventer() (*deployer_pb.PostgresMigrationPSM, error) {
	config := deployer_pb.DefaultPostgresMigrationPSMConfig()
	sm, err := deployer_pb.NewPostgresMigrationPSM(config)
	if err != nil {
		return nil, err
	}

	return sm, nil
}
