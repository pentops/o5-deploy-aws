package deployer

import (
	"context"
	"fmt"

	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/proto"
)

type ITransitionSpec[State proto.Message, Event any] interface {
	Matches(State, Event) bool
	RunTransition(context.Context, TransitionBaton[Event], State, Event) error
}

type TransitionBaton[Event any] interface {
	SideEffect(outbox.OutboxMessage)
	ChainEvent(Event)
}

type TransitionData[Event any] struct {
	SideEffects []outbox.OutboxMessage
	ChainEvents []Event
}

func (td *TransitionData[Event]) ChainEvent(event Event) {
	td.ChainEvents = append(td.ChainEvents, event)
}

func (td *TransitionData[Event]) SideEffect(msg outbox.OutboxMessage) {
	td.SideEffects = append(td.SideEffects, msg)
}

func FindTransition(ctx context.Context, deployment *deployer_pb.DeploymentState, event deployer_pb.IsDeploymentEventTypeWrappedType) (ITransitionSpec[*deployer_pb.DeploymentState, deployer_pb.IsDeploymentEventTypeWrappedType], error) {
	return findTransition(deployment, event)
}

func findTransition(deployment *deployer_pb.DeploymentState, event deployer_pb.IsDeploymentEventTypeWrappedType) (ITransitionSpec[*deployer_pb.DeploymentState, deployer_pb.IsDeploymentEventTypeWrappedType], error) {
	for _, search := range deploymentTransitions {
		if search.Matches(deployment, event) {
			return search, nil
		}
	}
	typeKey := event.TypeKey()

	// TODO: This by generation and annotation
	if stackStatus, ok := event.(*deployer_pb.DeploymentEventType_StackStatus); ok {
		typeKey = deployer_pb.DeploymentEventTypeKey(fmt.Sprintf("%s.%s", typeKey, stackStatus.Lifecycle.ShortString()))
	}
	return nil, fmt.Errorf("no transition found for status %s -> %s",
		deployment.Status.ShortString(),
		typeKey,
	)
}
