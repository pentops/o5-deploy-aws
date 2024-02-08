package states

import (
	"fmt"

	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type StateMachines struct {
	Deployment *deployer_pb.DeploymentPSM
	Stack      *deployer_pb.StackPSM
}

func NewStateMachines() (*StateMachines, error) {
	deployment, err := NewDeploymentEventer()
	if err != nil {
		return nil, fmt.Errorf("NewDeploymentEventer: %w", err)
	}

	stack, err := NewStackEventer()
	if err != nil {
		return nil, fmt.Errorf("NewStackEventer: %w", err)
	}

	return &StateMachines{
		Deployment: deployment,
		Stack:      stack,
	}, nil
}
