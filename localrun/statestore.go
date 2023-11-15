package localrun

import (
	"context"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/outbox.pg.go/outbox"
)

type Publisher interface {
	PublishEvent(ctx context.Context, msg outbox.OutboxMessage) error
}

// LocalStateStore wires back the events to the deployer, rather than relying on
// an event bus and database
type LocalStateStore struct {
	stackMap map[string]string // stackName -> deploymentId

	deployments  map[string]*deployer_pb.DeploymentState
	environments map[string]*environment_pb.Environment

	eventLoop Publisher

	validator *protovalidate.Validator

	waitChan chan error

	StoreCallback func(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error
}

func NewLocalStateStore(eventLoop Publisher) *LocalStateStore {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}

	lss := &LocalStateStore{
		stackMap:     map[string]string{},
		environments: map[string]*environment_pb.Environment{},
		deployments:  map[string]*deployer_pb.DeploymentState{},
		eventLoop:    eventLoop,
		validator:    validator,
		waitChan:     make(chan error),
	}
	return lss
}

func (lss *LocalStateStore) Wait(ctx context.Context) error {
	return <-lss.waitChan
}

func (lss *LocalStateStore) AddEnvironment(environment *environment_pb.Environment) error {
	if err := lss.validator.Validate(environment); err != nil {
		return err
	}

	lss.environments[environment.FullName] = environment
	return nil
}

func (lss *LocalStateStore) Transact(ctx context.Context, fn func(context.Context, deployer.TransitionTransaction) error) error {
	// Local state doesn't need a transaction as it doesn't have a database
	return fn(ctx, lss)
}

func (lss *LocalStateStore) GetEnvironment(ctx context.Context, environmentName string) (*environment_pb.Environment, error) {
	if env, ok := lss.environments[environmentName]; ok {
		return env, nil
	}
	return nil, fmt.Errorf("missing environment %s", environmentName)
}

func (lss *LocalStateStore) PublishEvent(ctx context.Context, msg outbox.OutboxMessage) error {
	return lss.eventLoop.PublishEvent(ctx, msg)
}

func (lss *LocalStateStore) StoreDeploymentEvent(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
	if err := lss.validator.Validate(event); err != nil {
		return fmt.Errorf("validating event: %s", err)
	}

	if err := lss.validator.Validate(state); err != nil {
		return fmt.Errorf("validating state: %s", err)
	}

	lss.deployments[state.DeploymentId] = state

	if lss.StoreCallback != nil {
		if err := lss.StoreCallback(ctx, state, event); err != nil {
			return err
		}
	}

	// Special case handlers for local only, registers pending callbacks
	// to the deployment which triggered them.

	switch event.Event.Type.(type) {
	case *deployer_pb.DeploymentEventType_GotLock_:
		// Registers THIS deployment as THE deployment for the stack
		lss.stackMap[state.StackName] = state.DeploymentId

	case *deployer_pb.DeploymentEventType_Error_:
		lss.waitChan <- fmt.Errorf("Failed: %s", event.Event.GetError().Error)
	}

	if state.Status == deployer_pb.DeploymentStatus_DONE {
		close(lss.waitChan)
	}

	return nil
}

func (lss *LocalStateStore) GetDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error) {
	if deployment, ok := lss.deployments[id]; ok {
		return deployment, nil
	}
	return nil, deployer.DeploymentNotFoundError
}

func (lss *LocalStateStore) GetDeploymentForStack(ctx context.Context, stackName string) (*deployer_pb.DeploymentState, error) {

	deploymentId, ok := lss.stackMap[stackName]
	if !ok {
		return nil, fmt.Errorf("missing deploymentId for stack %s", stackName)
	}

	return lss.GetDeployment(ctx, deploymentId)
}
