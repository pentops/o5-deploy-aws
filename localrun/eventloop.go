package localrun

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventLoop emulates AWS infrastructure by running all handlers in the
// same process. Used when running as a standalone tool, e.g. when
// bootstrapping a new environment.
type EventLoop struct {
	validator *protovalidate.Validator
	storage   *StateStore
	awsRunner *InfraAdapter
}

func NewEventLoop(awsRunner *InfraAdapter, stateStore *StateStore) *EventLoop {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}
	return &EventLoop{
		awsRunner: awsRunner,
		storage:   stateStore,
		validator: validator,
	}
}

func (lel *EventLoop) Run(ctx context.Context, trigger *deployer_tpb.TriggerDeploymentMessage) error {
	if err := lel.validator.Validate(trigger); err != nil {
		return err
	}

	deploymentId := trigger.DeploymentId

	tx := lel.storage

	environment, err := tx.GetEnvironment(ctx, trigger.Spec.EnvironmentName)
	if err != nil {
		return err
	}

	deployerResolver, err := deployer.BuildParameterResolver(ctx, environment)
	if err != nil {
		return err
	}

	eventQueue := []*deployer_pb.DeploymentEvent{{
		DeploymentId: trigger.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Created_{
				Created: &deployer_pb.DeploymentEventType_Created{
					Spec: trigger.Spec,
				},
			},
		},
	}, {
		DeploymentId: trigger.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Triggered_{
				Triggered: &deployer_pb.DeploymentEventType_Triggered{},
			},
		},
	}}

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		deployment, err := tx.GetDeployment(ctx, deploymentId)
		if errors.Is(err, deployer.DeploymentNotFoundError) {
			deployment = &deployer_pb.DeploymentState{
				DeploymentId: deploymentId,
			}
		} else if err != nil {
			return err
		}

		baton := &deployer.TransitionData{
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

		transition, err := deployer.FindTransition(ctx, deployment, innerEvent)
		if err != nil {
			return err
		}
		if err := transition.RunTransition(ctx, baton, deployment, innerEvent); err != nil {
			return err
		}

		ctx = log.WithFields(ctx, map[string]interface{}{
			"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, deployment.Status.ShortString(), typeKey),
		})
		log.Info(ctx, "End Deployment Event")

		if err := tx.StoreDeploymentEvent(ctx, deployment, innerEvent); err != nil {
			return err
		}

		// Each transiton will produce either one chain event, or one side
		// effect, until the deployment is terminal.
		// Each of the side effect handlers will take an action then return a
		// single status result which triggers the next message.
		// This is not true of state machines always, but in this case the logic
		// of the transitions and effects is constrainted to make it possible
		// to run locally without tracking a complex event loop (and figuring
		// out when to exit)

		if len(baton.ChainEvents) > 0 && len(baton.SideEffects) > 0 {
			return fmt.Errorf("cannot have both side effects and chained events in local run mode")
		}

		if len(baton.ChainEvents) > 0 {
			if len(baton.ChainEvents) > 1 {
				for _, evt := range baton.ChainEvents {
					log.WithField(ctx, "chainEvent", protojson.Format(evt)).Debug("Chain Event")
				}
				return fmt.Errorf("cannot have more than one chain event in local run mode")
			}
			evt := baton.ChainEvents[0]
			log.WithField(ctx, "chainEvent", protojson.Format(evt)).Debug("Chain Event")
			eventQueue = append(eventQueue, evt)
			continue
		}

		for _, sideEffect := range baton.SideEffects {
			ctx = log.WithField(ctx, "inputMessage", sideEffect.ProtoReflect().Descriptor().FullName())
			log.Debug(ctx, "Side Effect")
			result, err := lel.handleSideEffect(ctx, sideEffect)
			if err != nil {
				log.WithError(ctx, err).Error("Side Effect Error")
				return err
			}
			mapped, err := mapSideEffectResult(result)
			if err != nil {
				return err
			}
			log.WithField(ctx, "nextEvent", protojson.Format(mapped)).Debug("Side Effect Result")
			eventQueue = append(eventQueue, mapped)
		}

	}

	return nil
}

func mapSideEffectResult(result proto.Message) (*deployer_pb.DeploymentEvent, error) {

	switch result := result.(type) {
	case *deployer_tpb.StackStatusChangedMessage:
		return deployer.TranslateStackStatusChanged(result)

	case *deployer_tpb.MigrationStatusChangedMessage:
		return deployer.TranslateMigrationStatusChanged(result)

	default:
		return nil, fmt.Errorf("unknown side effect result type: %T", result)
	}

}

func (lel *EventLoop) handleSideEffect(ctx context.Context, msg proto.Message) (proto.Message, error) {

	switch msg := msg.(type) {
	case *deployer_tpb.UpdateStackMessage:
		return lel.awsRunner.UpdateStack(ctx, msg)

	case *deployer_tpb.CreateNewStackMessage:
		return lel.awsRunner.CreateNewStack(ctx, msg)

	case *deployer_tpb.ScaleStackMessage:
		return lel.awsRunner.ScaleStack(ctx, msg)

	case *deployer_tpb.StabalizeStackMessage:
		return lel.awsRunner.StabalizeStack(ctx, msg)

	case *deployer_tpb.RunDatabaseMigrationMessage:
		return lel.awsRunner.RunDatabaseMigration(ctx, msg)
	}

	return nil, fmt.Errorf("unknown side effect message type: %T", msg)
}
