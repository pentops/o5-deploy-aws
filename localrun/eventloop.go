package localrun

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bufbuild/protovalidate-go"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventLoop emulates AWS infrastructure by running all handlers in the
// same process. Used when running as a standalone tool, e.g. when
// bootstrapping a new environment.
type EventLoop struct {
	validator   *protovalidate.Validator
	storage     *StateStore
	awsRunner   IInfra
	specBuilder *deployer.SpecBuilder
	confirmPlan bool
}

type IInfra interface {
	HandleMessage(ctx context.Context, msg proto.Message) (*deployer_pb.DeploymentPSMEventSpec, error)
}

func NewEventLoop(awsRunner IInfra, stateStore *StateStore, specBuilder *deployer.SpecBuilder) *EventLoop {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}
	return &EventLoop{
		awsRunner:   awsRunner,
		storage:     stateStore,
		specBuilder: specBuilder,
		validator:   validator,
	}
}

type TransitionData struct {
	CausedBy    *deployer_pb.DeploymentEvent
	SideEffects []outbox.OutboxMessage
	ChainEvents []deployer_pb.DeploymentPSMEvent
}

func (td *TransitionData) ChainEvent(event deployer_pb.DeploymentPSMEvent) {
	td.ChainEvents = append(td.ChainEvents, event)
}

func (td *TransitionData) SideEffect(msg outbox.OutboxMessage) {
	td.SideEffects = append(td.SideEffects, msg)
}

func (td *TransitionData) AsCause() *psm_pb.Cause {
	return &psm_pb.Cause{
		Type: &psm_pb.Cause_PsmEvent{
			PsmEvent: &psm_pb.PSMEventCause{
				EventId:      td.CausedBy.Metadata.EventId,
				StateMachine: td.CausedBy.Keys.PSMFullName(),
			},
		},
	}

}

func (td *TransitionData) FullCause() *deployer_pb.DeploymentEvent {
	return td.CausedBy
}

func (lel *EventLoop) Run(ctx context.Context, trigger *deployer_tpb.RequestDeploymentMessage, cluster *environment_pb.Cluster, environment *environment_pb.Environment) error {
	if err := lel.validator.Validate(trigger); err != nil {
		return err
	}

	spec, err := lel.specBuilder.BuildSpec(ctx, trigger, cluster, environment)
	if err != nil {
		return err
	}

	deploymentId := trigger.DeploymentId

	deploymentKeys := &deployer_pb.DeploymentKeys{
		DeploymentId:  trigger.DeploymentId,
		EnvironmentId: trigger.EnvironmentId,
		// These don't mean anything locally.
		StackId:   uuid.NewString(),
		ClusterId: uuid.NewString(),
	}

	tx := lel.storage

	eventQueue := []*deployer_pb.DeploymentEvent{{
		Keys: deploymentKeys,
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Created_{
				Created: &deployer_pb.DeploymentEventType_Created{
					Spec: spec,
				},
			},
		},
	}, {
		Keys: deploymentKeys,
		Metadata: &psm_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Triggered_{
				Triggered: &deployer_pb.DeploymentEventType_Triggered{},
			},
		},
	}}

	stateMachine, err := states.NewDeploymentEventer()
	if err != nil {
		return err
	}

	stopped := false
	go func() {
		<-ctx.Done()
		stopped = true
	}()

	for len(eventQueue) > 0 && !stopped {
		innerEvent := eventQueue[0]
		eventQueue = eventQueue[1:]

		deployment, err := tx.GetDeployment(ctx, deploymentId)
		if errors.Is(err, deployer.DeploymentNotFoundError) {
			deployment = &deployer_pb.DeploymentState{
				Metadata: &psm_pb.StateMetadata{},
				Keys:     deploymentKeys,
			}
		} else if err != nil {
			return err
		}
		innerEvent.Keys = deploymentKeys

		baton := &TransitionData{
			CausedBy: innerEvent,
		}

		typeKey, _ := innerEvent.Event.TypeKey()
		stateBefore := deployment.Status.ShortString()

		ctx = log.WithFields(ctx, map[string]interface{}{
			"deploymentId": innerEvent.Keys.DeploymentId,
			"eventType":    typeKey,
			"transition":   fmt.Sprintf("%s -> ? : %s", stateBefore, typeKey),
		})
		log.WithField(ctx, "event", protojson.Format(innerEvent.Event)).Debug("Begin Deployment Event")

		statusBefore := deployment.Status
		transition, err := stateMachine.FindTransition(statusBefore, innerEvent)
		if err != nil {
			return err
		}
		if err := transition.RunTransition(ctx, deployment, innerEvent); err != nil {
			return fmt.Errorf("run transisiotn: %w", err)
		}

		ctx = log.WithFields(ctx, map[string]interface{}{
			"transition": fmt.Sprintf("%s -> %s : %s", stateBefore, deployment.Status.ShortString(), typeKey),
		})
		log.Info(ctx, "End Deployment Event")

		if err := tx.StoreDeploymentEvent(ctx, deployment, innerEvent); err != nil {
			return err
		}

		hooks := stateMachine.FindHooks(statusBefore, innerEvent)
		for _, hook := range hooks {
			// nil TX means no hooks can use the database. This method is
			// getting bad.
			if err := hook.RunStateHook(ctx, nil, baton, deployment, innerEvent); err != nil {
				return err
			}
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
				return fmt.Errorf("cannot have more than one chain event in local run mode")
			}
			evt := baton.ChainEvents[0]

			if lel.confirmPlan {
				if evt.PSMEventKey() == deployer_pb.DeploymentPSMEventRunSteps {
					if !confirmPlan(evt.(*deployer_pb.DeploymentEventType_RunSteps)) {
						return nil
					}

				}
			}
			log.WithField(ctx, "chainEvent", protojson.Format(evt)).Debug("Chain Event")
			wrapped := &deployer_pb.DeploymentEvent{
				Keys: innerEvent.Keys,
				Metadata: &psm_pb.EventMetadata{
					EventId:   uuid.NewString(),
					Timestamp: timestamppb.Now(),
				},
			}
			if err := wrapped.SetPSMEvent(evt); err != nil {
				return err
			}

			eventQueue = append(eventQueue, wrapped)
			continue
		}

		for _, sideEffect := range baton.SideEffects {
			ctx = log.WithField(ctx, "inputMessage", sideEffect.ProtoReflect().Descriptor().FullName())
			log.Debug(ctx, "Side Effect")
			result, err := lel.awsRunner.HandleMessage(ctx, sideEffect)
			if err != nil {
				log.WithError(ctx, err).Error("Side Effect Error")
				return err
			}

			if result != nil {
				mapped := &deployer_pb.DeploymentEvent{
					Keys: deploymentKeys,
					Metadata: &psm_pb.EventMetadata{
						EventId:   uuid.NewString(),
						Timestamp: timestamppb.Now(),
					},
				}
				if err := mapped.SetPSMEvent(result.Event); err != nil {
					return err
				}
				log.WithField(ctx, "nextEvent", protojson.Format(mapped)).Debug("Side Effect Result")
				eventQueue = append(eventQueue, mapped)
			}
		}

	}

	return nil
}

var inputReader *bufio.Reader

func Ask(prompt string) string {
	if inputReader == nil {
		inputReader = bufio.NewReader(os.Stdin)
	}
	fmt.Printf("%s: \n", prompt)
	answer, err := inputReader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return answer
}

func AskBool(prompt string) bool {
	answer := Ask(prompt + " [y/n]")
	return strings.HasPrefix(strings.ToLower(answer), "y")
}

func confirmPlan(deployment *deployer_pb.DeploymentEventType_RunSteps) bool {
	fmt.Printf("CONFIRM STEPS\n")
	stepMap := make(map[string]*deployer_pb.DeploymentStep)
	for _, step := range deployment.Steps {
		stepMap[step.Id] = step
	}
	for _, step := range deployment.Steps {
		typeKey, _ := step.Request.TypeKey()
		fmt.Printf("- %s (%s)\n", step.Name, typeKey)
		for _, dep := range step.DependsOn {
			fmt.Printf("   <- %s\n", stepMap[dep].Name)
		}
	}
	return AskBool("Continue?")
}
