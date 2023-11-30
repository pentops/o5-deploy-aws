package deployer

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TransitionSpec[Event deployer_pb.IsDeploymentEventTypeWrappedType] struct {
	FromStatus  []deployer_pb.DeploymentStatus
	EventFilter func(Event) bool
	Transition  func(context.Context, DeploymentTransitionBaton, *deployer_pb.DeploymentState, Event) error
}

func (ts TransitionSpec[Event]) RunTransition(ctx context.Context, tb TransitionBaton[*deployer_pb.DeploymentEvent], deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {

	asType, ok := event.Event.Get().(Event)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event.Event.Get())
	}

	return ts.Transition(ctx, tb, deployment, asType)
}

func (ts TransitionSpec[Event]) Matches(deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) bool {
	got := event.Event.Get()
	if got == nil {
		return false
	}
	asType, ok := got.(Event)
	if !ok {
		return false
	}
	didMatch := false
	for _, fromStatus := range ts.FromStatus {
		if fromStatus == deployment.Status {
			didMatch = true
			break
		}
	}
	if !didMatch {
		return false
	}

	if ts.EventFilter != nil && !ts.EventFilter(asType) {
		return false
	}
	return true
}

type ITransitionSpec[State proto.Message, Event proto.Message] interface {
	Matches(State, Event) bool
	RunTransition(context.Context, TransitionBaton[Event], State, Event) error
}

type TransitionBaton[Event proto.Message] interface {
	ChainEvent(Event)
	SideEffect(outbox.OutboxMessage)

	ResolveParameters([]*deployer_pb.Parameter) ([]*deployer_pb.CloudFormationStackParameter, error)
}

func newEvent(d *deployer_pb.DeploymentState, event deployer_pb.IsDeploymentEventType_Type) *deployer_pb.DeploymentEvent {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: d.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: event,
		},
	}
}

type TransitionData[Event proto.Message] struct {
	SideEffects       []outbox.OutboxMessage
	ChainEvents       []Event
	ParameterResolver ParameterResolver
}

func (td *TransitionData[Event]) ResolveParameters(stackParameters []*deployer_pb.Parameter) ([]*deployer_pb.CloudFormationStackParameter, error) {

	parameters := make([]*deployer_pb.CloudFormationStackParameter, 0, len(stackParameters))

	for _, param := range stackParameters {
		parameter, err := td.ParameterResolver.ResolveParameter(param)
		if err != nil {
			return nil, fmt.Errorf("parameter '%s': %w", param.Name, err)
		}
		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

func (td *TransitionData[Event]) ChainEvent(event Event) {
	td.ChainEvents = append(td.ChainEvents, event)
}

func (td *TransitionData[Event]) SideEffect(msg outbox.OutboxMessage) {
	td.SideEffects = append(td.SideEffects, msg)
}

func FindTransition(ctx context.Context, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) (ITransitionSpec[*deployer_pb.DeploymentState, *deployer_pb.DeploymentEvent], error) {
	for _, search := range transitions {
		if search.Matches(deployment, event) {
			return search, nil
		}
	}
	typeKey, ok := event.Event.TypeKey()
	if !ok {
		return nil, fmt.Errorf("unknown event type: %T", event.Event)
	}
	// TODO: This by generation and annotation
	if stackStatus := event.Event.GetStackStatus(); stackStatus != nil {
		typeKey = deployer_pb.DeploymentEventTypeKey(fmt.Sprintf("%s.%s", typeKey, stackStatus.Lifecycle.ShortString()))
	}
	return nil, fmt.Errorf("no transition found for status %s -> %s",
		deployment.Status.ShortString(),
		typeKey,
	)
}
