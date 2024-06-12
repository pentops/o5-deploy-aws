package localrun

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
)

// StateStore wires back the events to the deployer, rather than relying on
// an event bus and database
type StateStore struct {
	deployments map[string]*awsdeployer_pb.DeploymentState

	validator *protovalidate.Validator

	StoreCallback func(ctx context.Context, state *awsdeployer_pb.DeploymentState, event *awsdeployer_pb.DeploymentEvent) error
}

func NewStateStore() *StateStore {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}

	lss := &StateStore{
		deployments: map[string]*awsdeployer_pb.DeploymentState{},
		validator:   validator,
	}
	return lss
}

func (lss *StateStore) StoreDeploymentEvent(ctx context.Context, state *awsdeployer_pb.DeploymentState, event *awsdeployer_pb.DeploymentEvent) error {
	if err := lss.validator.Validate(event); err != nil {
		return fmt.Errorf("validating event: %s", err)
	}

	if err := lss.validator.Validate(state); err != nil {
		return fmt.Errorf("validating state: %s", err)
	}

	lss.deployments[state.Keys.DeploymentId] = state

	if lss.StoreCallback != nil {
		if err := lss.StoreCallback(ctx, state, event); err != nil {
			return err
		}
	}

	return nil
}

func (lss *StateStore) GetDeployment(ctx context.Context, id string) (*awsdeployer_pb.DeploymentState, error) {
	if deployment, ok := lss.deployments[id]; ok {
		return deployment, nil
	}
	return nil, deployer.DeploymentNotFoundError
}
