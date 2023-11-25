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

func (dw *DeployerWorker) RegisterEvent(ctx context.Context, outerEvent *deployer_pb.DeploymentEvent) error {

	return dw.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		deployment, err := tx.GetDeployment(ctx, outerEvent.DeploymentId)
		if errors.Is(err, DeploymentNotFoundError) {
			deployment = &deployer_pb.DeploymentState{
				DeploymentId: outerEvent.DeploymentId,
			}
		} else if err != nil {
			return err
		}

		var environmentName string
		if deployment.Spec != nil {
			environmentName = deployment.Spec.EnvironmentName
		} else if trigger := outerEvent.Event.GetTriggered(); trigger != nil {
			environmentName = trigger.Spec.EnvironmentName
		} else {
			return fmt.Errorf("no spec found for deployment %s, and the event is not an initiating event", outerEvent.DeploymentId)
		}

		environment, err := tx.GetEnvironment(ctx, environmentName)
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
	})
}

func TranslateTrigger(msg *deployer_tpb.TriggerDeploymentMessage) (*deployer_pb.DeploymentEvent, error) {
	return &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Triggered_{
				Triggered: &deployer_pb.DeploymentEventType_Triggered{
					Spec: msg.Spec,
				},
			},
		},
	}, nil
}

func (dw *DeployerWorker) TriggerDeployment(ctx context.Context, msg *deployer_tpb.TriggerDeploymentMessage) (*emptypb.Empty, error) {

	deploymentEvent, err := TranslateTrigger(msg)
	if err != nil {
		return nil, err
	}

	if err := dw.RegisterEvent(ctx, deploymentEvent); err != nil {
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
	return &emptypb.Empty{}, dw.RegisterEvent(ctx, event)
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

	return &emptypb.Empty{}, dw.RegisterEvent(ctx, event)

}
