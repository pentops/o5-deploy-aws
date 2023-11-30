package deployer

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DeployerWorker struct {
	*deployer_tpb.UnimplementedDeployerTopicServer
	storage DeployerStorage
}

func NewDeployerWorker(store DeployerStorage) (*DeployerWorker, error) {
	return &DeployerWorker{
		storage: store,
	}, nil
}

func (dw *DeployerWorker) registerEvent(ctx context.Context, event *deployer_pb.DeploymentEvent) error {
	if err := dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return registerEvent(ctx, tx, event)
	}); err != nil {
		return err
	}
	return nil
}

func registerEvent(ctx context.Context, tx TransitionTransaction, outerEvent *deployer_pb.DeploymentEvent) error {

	deployment, err := tx.GetDeployment(ctx, outerEvent.DeploymentId)
	if errors.Is(err, DeploymentNotFoundError) {
		trigger := outerEvent.Event.GetCreated()
		if trigger == nil {
			return fmt.Errorf("deployment %s not found, and the event is not an initiating event", outerEvent.DeploymentId)
		}

		deployment = &deployer_pb.DeploymentState{
			DeploymentId: outerEvent.DeploymentId,
			Spec:         trigger.Spec,
		}
	} else if err != nil {
		return err
	}

	environment, err := tx.GetEnvironment(ctx, deployment.Spec.EnvironmentName)
	if err != nil {
		return err
	}

	deployerResolver, err := BuildParameterResolver(ctx, environment)
	if err != nil {
		return err
	}

	eventsToProcess := []*deployer_pb.DeploymentEvent{outerEvent}

	for len(eventsToProcess) > 0 {
		innerEvent := eventsToProcess[0]
		eventsToProcess = eventsToProcess[1:]

		baton := &TransitionData{
			ParameterResolver: deployerResolver,
		}
		typeKey, _ := innerEvent.Event.TypeKey()
		stateBefore := deployment.Status.ShortString()

		ctx = log.WithFields(ctx, map[string]interface{}{
			"deploymentId": innerEvent.DeploymentId,
			"eventType":    typeKey,
			"transition":   fmt.Sprintf("%s -> ? : %s", stateBefore, typeKey),
		})
		log.WithField(ctx, "event", protojson.Format(innerEvent.Event)).Debug("Begin Deployment Event")
		transition, err := FindTransition(ctx, deployment, innerEvent)

		if err != nil {
			return err
		}

		if err := transition.RunTransition(ctx, baton, deployment, innerEvent); err != nil {
			log.WithError(ctx, err).Error("Running Deployment Transition")
			return err
		}

		if err := tx.StoreDeploymentEvent(ctx, deployment, innerEvent); err != nil {
			return err
		}

		ctx = log.WithFields(ctx, map[string]interface{}{
			"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, deployment.Status.ShortString(), typeKey),
		})

		log.Info(ctx, "Deployment Event Handled")

		for _, se := range baton.SideEffects {
			if err := tx.PublishEvent(ctx, se); err != nil {
				return fmt.Errorf("publishEvent: %w", err)
			}
		}

		eventsToProcess = append(eventsToProcess, baton.ChainEvents...)
	}
	return nil
}

func (dw *DeployerWorker) TriggerDeployment(ctx context.Context, msg *deployer_tpb.TriggerDeploymentMessage) (*emptypb.Empty, error) {

	createDeploymentEvent := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Created_{
				Created: &deployer_pb.DeploymentEventType_Created{
					Spec: msg.Spec,
				},
			},
		},
	}

	if err := dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {

		deploymentEvents := make([]*deployer_pb.DeploymentEvent, 1, 2)

		deploymentEvents[0] = createDeploymentEvent

		stackID := StackID(msg.Spec.EnvironmentName, msg.Spec.AppName)
		stack, err := tx.GetStack(ctx, stackID)
		if errors.Is(err, StackNotFoundError) {
			stack = &deployer_pb.StackState{
				StackId:           stackID,
				Status:            deployer_pb.StackStatus_UNSPECIFIED, // New
				CurrentDeployment: nil,
				ApplicationName:   msg.Spec.AppName,
				EnvironmentName:   msg.Spec.EnvironmentName,
				QueuedDeployments: []*deployer_pb.StackDeployment{},
			}
		} else if err != nil {
			return err
		}

		evt := &deployer_pb.StackEvent{
			StackId:  stackID,
			Metadata: &deployer_pb.EventMetadata{},
			Event: &deployer_pb.StackEventType{
				Type: &deployer_pb.StackEventType_Triggered_{
					Triggered: &deployer_pb.StackEventType_Triggered{
						Deployment: &deployer_pb.StackDeployment{
							DeploymentId: msg.DeploymentId,
							Version:      msg.Spec.Version,
						},
					},
				},
			},
		}

		shouldTrigger := false
		switch stack.Status {
		case deployer_pb.StackStatus_UNSPECIFIED:
			stack.Status = deployer_pb.StackStatus_CREATING
			shouldTrigger = true

		case deployer_pb.StackStatus_STABLE:
			stack.Status = deployer_pb.StackStatus_MIGRATING
			shouldTrigger = true

		case deployer_pb.StackStatus_BROKEN,
			deployer_pb.StackStatus_CREATING,
			deployer_pb.StackStatus_MIGRATING:

			stack.QueuedDeployments = append(stack.QueuedDeployments, &deployer_pb.StackDeployment{
				DeploymentId: msg.DeploymentId,
				Version:      msg.Spec.Version,
			})

			if err := tx.StoreStackEvent(ctx, stack, evt); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unexpected stack status: %s", stack.Status.String())

		}

		if err := tx.StoreStackEvent(ctx, stack, evt); err != nil {
			return err
		}

		if err := registerEvent(ctx, tx, createDeploymentEvent); err != nil {
			return err
		}

		if !shouldTrigger {
			return nil
		}

		triggerDeploymentEvent := &deployer_pb.DeploymentEvent{
			DeploymentId: msg.DeploymentId,
			Metadata: &deployer_pb.EventMetadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
			},
			Event: &deployer_pb.DeploymentEventType{
				Type: &deployer_pb.DeploymentEventType_Triggered_{
					Triggered: &deployer_pb.DeploymentEventType_Triggered{},
				},
			},
		}

		if err := registerEvent(ctx, tx, triggerDeploymentEvent); err != nil {
			return err
		}

		return nil

	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) DeploymentComplete(ctx context.Context, msg *deployer_tpb.DeploymentCompleteMessage) (*emptypb.Empty, error) {

	stackEvent := &deployer_pb.StackEvent{
		StackId:  StackID(msg.EnvironmentName, msg.ApplicationName),
		Metadata: &deployer_pb.EventMetadata{},
		Event: &deployer_pb.StackEventType{
			Type: &deployer_pb.StackEventType_DeploymentCompleted_{
				DeploymentCompleted: &deployer_pb.StackEventType_DeploymentCompleted{
					Deployment: &deployer_pb.StackDeployment{
						DeploymentId: msg.DeploymentId,
						Version:      msg.Version,
					},
				},
			},
		},
	}

	if err := dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		stack, err := tx.GetStack(ctx, stackEvent.StackId)
		if err != nil {
			return err
		}

		switch stack.Status {
		case deployer_pb.StackStatus_CREATING, deployer_pb.StackStatus_MIGRATING:
			stack.Status = deployer_pb.StackStatus_STABLE

			if len(stack.QueuedDeployments) > 0 {
				stack.CurrentDeployment = stack.QueuedDeployments[0]
				stack.QueuedDeployments = stack.QueuedDeployments[1:]

				triggerDeploymentEvent := &deployer_pb.DeploymentEvent{
					DeploymentId: stack.CurrentDeployment.DeploymentId,
					Metadata: &deployer_pb.EventMetadata{
						EventId:   uuid.NewString(),
						Timestamp: timestamppb.Now(),
					},
					Event: &deployer_pb.DeploymentEventType{
						Type: &deployer_pb.DeploymentEventType_Triggered_{
							Triggered: &deployer_pb.DeploymentEventType_Triggered{},
						},
					},
				}

				if err := registerEvent(ctx, tx, triggerDeploymentEvent); err != nil {
					return err
				}
			}

			if err := tx.StoreStackEvent(ctx, stack, stackEvent); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unexpected stack status: %s", stack.Status.String())
		}

		return nil

	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func TranslateStackStatusChanged(msg *deployer_tpb.StackStatusChangedMessage) (*deployer_pb.DeploymentEvent, error) {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: msg.StackId.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_StackStatus_{
				StackStatus: &deployer_pb.DeploymentEventType_StackStatus{
					Lifecycle:   msg.Lifecycle,
					FullStatus:  msg.Status,
					StackOutput: msg.Outputs,
				},
			},
		},
	}, nil
}

func (dw *DeployerWorker) StackStatusChanged(ctx context.Context, msg *deployer_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := TranslateStackStatusChanged(msg)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, dw.registerEvent(ctx, event)
}

func TranslateMigrationStatusChanged(msg *deployer_tpb.MigrationStatusChangedMessage) (*deployer_pb.DeploymentEvent, error) {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},

		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_DbMigrateStatus{
				DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
					MigrationId: msg.MigrationId,
					Status:      msg.Status,
				},
			},
		},
	}, nil
}

func (dw *DeployerWorker) MigrationStatusChanged(ctx context.Context, msg *deployer_tpb.MigrationStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := TranslateMigrationStatusChanged(msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, dw.registerEvent(ctx, event)

}
