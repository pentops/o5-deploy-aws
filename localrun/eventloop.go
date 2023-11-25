package localrun

import (
	"context"
	"errors"
	"fmt"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

	outerEvent, err := deployer.TranslateTrigger(trigger)
	if err != nil {
		return err
	}

	tx := lel.storage

	environment, err := tx.GetEnvironment(ctx, trigger.Spec.EnvironmentName)
	if err != nil {
		return err
	}

	deployerResolver, err := deployer.BuildParameterResolver(ctx, environment)
	if err != nil {
		return err
	}

	eventQueue := make([]*deployer_pb.DeploymentEvent, 0)
	eventQueue = append(eventQueue, outerEvent)

	for len(eventQueue) > 0 {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		deployment, err := tx.GetDeployment(ctx, outerEvent.DeploymentId)
		if errors.Is(err, deployer.DeploymentNotFoundError) {
			deployment = &deployer_pb.DeploymentState{
				DeploymentId: outerEvent.DeploymentId,
			}
		} else if err != nil {
			return err
		}

		baton := &deployer.TransitionData{
			ParameterResolver: deployerResolver,
		}

		typeKey, _ := outerEvent.Event.TypeKey()
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
			eventQueue = append(eventQueue, baton.ChainEvents...)
			continue
		}

		for _, sideEffect := range baton.SideEffects {
			result, err := lel.handleSideEffect(ctx, sideEffect)
			if err != nil {
				return err
			}
			mapped, err := mapSideEffectResult(result)
			if err != nil {
				return err
			}
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
	log.WithField(ctx, "inputMessage", msg.ProtoReflect().Descriptor().FullName()).Debug("Side Effect")

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
