package localrun

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan"
	"github.com/pentops/o5-messaging/o5msg"
	"google.golang.org/protobuf/proto"
)

type IInfra interface {
	HandleMessage(ctx context.Context, msg proto.Message) (*awsdeployer_pb.DeploymentPSMEventSpec, error)
	StabalizeStack(ctx context.Context, msg *awsinfra_tpb.StabalizeStackMessage) (*awsinfra_tpb.StackStatusChangedMessage, error)
}

type TransitionData struct {
	CausedBy    *awsdeployer_pb.DeploymentEvent
	SideEffects []o5msg.Message
	ChainEvents []awsdeployer_pb.DeploymentPSMEvent
}

func (td *TransitionData) ChainEvent(event awsdeployer_pb.DeploymentPSMEvent) {
	td.ChainEvents = append(td.ChainEvents, event)
}

func (td *TransitionData) SideEffect(msg o5msg.Message) {
	td.SideEffects = append(td.SideEffects, msg)
}

func (td *TransitionData) AsCause() *psm_j5pb.Cause {
	return &psm_j5pb.Cause{
		Type: &psm_j5pb.Cause_PsmEvent{
			PsmEvent: &psm_j5pb.PSMEventCause{
				EventId:      td.CausedBy.Metadata.EventId,
				StateMachine: td.CausedBy.Keys.PSMFullName(),
			},
		},
	}

}

func (td *TransitionData) FullCause() *awsdeployer_pb.DeploymentEvent {
	return td.CausedBy
}

type Runner struct {
	awsRunner   IInfra
	specBuilder *deployer.SpecBuilder
	confirmPlan bool
}

func (rr *Runner) RunDeployment(ctx context.Context, deployment *awsdeployer_pb.DeploymentStateData) error {

	// Wait For Stack

	stackStatus, err := rr.awsRunner.StabalizeStack(ctx, &awsinfra_tpb.StabalizeStackMessage{
		StackName:    deployment.Spec.CfStackName,
		CancelUpdate: deployment.Spec.Flags.CancelUpdates,
	})
	if err != nil {
		return err
	}

	deploymentInput := plan.DeploymentInput{
		Deployment: deployment.Spec,
	}

	if stackStatus != nil {
		fmt.Printf("Existing Stack: %s\n", stackStatus.StackName)
		fmt.Printf("  Status: %s\n", stackStatus.Status)
		fmt.Printf("  Lifecycle: %s\n", stackStatus.Lifecycle.ShortString())
		switch stackStatus.Lifecycle {
		case awsdeployer_pb.CFLifecycle_COMPLETE,
			awsdeployer_pb.CFLifecycle_ROLLED_BACK:

			deploymentInput.StackStatus = &awsdeployer_pb.CFStackOutput{
				Lifecycle: stackStatus.Lifecycle,
				Outputs:   stackStatus.Outputs,
			}

		case awsdeployer_pb.CFLifecycle_MISSING:
			// leave nil

		default:
			return fmt.Errorf("Stack Status: %s", stackStatus.Status)
		}
	}

	steps, err := plan.PlanDeploymentSteps(ctx, deploymentInput)
	if err != nil {
		return err
	}

	if rr.confirmPlan {
		if !confirmPlan(steps) {
			return nil
		}

	}

	err = rr.runSteps(ctx, steps)
	if err != nil {
		log.WithError(ctx, err).Error("runSteps Error")
		return err
	}
	log.Info(ctx, "runSteps Completed with no error")
	return nil
}

func (rr *Runner) runSteps(ctx context.Context, steps []*awsdeployer_pb.DeploymentStep) error {

	for {
		if err := plan.UpdateStepDependencies(steps); err != nil {
			return err
		}
		tb := &TransitionData{}
		if err := plan.StepNext(ctx, tb, steps); err != nil {
			return err
		}

		if len(tb.ChainEvents) == 0 {
			return fmt.Errorf("no chain events")
		}

		for _, chainEvent := range tb.ChainEvents {
			fields := map[string]interface{}{
				"event":     chainEvent,
				"eventType": chainEvent.PSMEventKey(),
			}
			log.WithFields(ctx, fields).Info("ChainEvent")

			switch evt := chainEvent.(type) {
			case *awsdeployer_pb.DeploymentEventType_RunStep:
				var step *awsdeployer_pb.DeploymentStep
				for _, search := range steps {
					if search.Meta.StepId == evt.StepId {
						search.Meta.Status = drss_pb.StepStatus_ACTIVE
						step = search
						break
					}
				}
				if step == nil {
					return fmt.Errorf("step not found: %s", evt.StepId)
				}
				ctx := log.WithField(ctx, "step", step.Meta.Name)
				log.Info(ctx, "Running Step")

				tb := &TransitionData{}
				err := plan.RunStep(ctx, tb, &awsdeployer_pb.DeploymentKeys{}, steps, evt.StepId)
				if err != nil {
					return err
				}
				if len(tb.SideEffects) != 1 {
					return fmt.Errorf("expected 1 side effect")
				}

				result, err := rr.awsRunner.HandleMessage(ctx, tb.SideEffects[0])
				if err != nil {
					return err
				}

				resultStepEvent, ok := result.Event.(*awsdeployer_pb.DeploymentEventType_StepResult)
				if !ok {
					return fmt.Errorf("unexpected result type: %T", result)
				}

				if err := plan.UpdateDeploymentStep(steps, resultStepEvent); err != nil {
					return err
				}

				continue

			case *awsdeployer_pb.DeploymentEventType_Error:
				return fmt.Errorf("step error: %s", evt.Error)

			case *awsdeployer_pb.DeploymentEventType_Done:
				return nil

			default:
				return fmt.Errorf("unknown event type: %T", evt)
			}
		}
	}

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

func confirmPlan(steps []*awsdeployer_pb.DeploymentStep) bool {
	fmt.Printf("CONFIRM STEPS\n")
	stepMap := make(map[string]*awsdeployer_pb.DeploymentStep)
	for _, step := range steps {
		stepMap[step.Meta.StepId] = step
	}
	for _, step := range steps {
		typeKey := step.Step.Get().TypeKey()
		fmt.Printf("- %s (%s)\n", step.Meta.Name, typeKey)
		for _, dep := range step.Meta.DependsOn {
			fmt.Printf("   <- %s\n", stepMap[dep].Meta.Name)
		}
	}
	return AskBool("Continue?")
}
