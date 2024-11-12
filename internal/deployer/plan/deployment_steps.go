package plan

import (
	"context"
	"fmt"

	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan/awsdeployer_step_pb"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan/planbuild"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan/strategy"
	"google.golang.org/protobuf/proto"
)

type DeploymentInput struct {
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

func PlanDeploymentSteps(ctx context.Context, input DeploymentInput) ([]*awsdeployer_pb.DeploymentStep, error) {
	plan := &planbuild.DeploymentPlan{
		StackStatus: input.StackStatus,
		Deployment:  input.Deployment,
	}

	if err := buildPlan(ctx, plan); err != nil {
		return nil, err
	}
	return plan.GetSteps(), nil
}

func buildPlan(ctx context.Context, plan *planbuild.DeploymentPlan) error {

	if plan.Deployment.Flags.ImportResources {
		if plan.StackStatus != nil {
			return fmt.Errorf("cannot import resources with existing stack")
		}
		plan.CFCreateEmpty().
			Then(plan.ImportResources(ctx))
		return nil
	}

	if plan.Deployment.Flags.InfraOnly {
		if plan.StackStatus == nil {
			plan.CFUpdate(0)
		} else {
			plan.CFCreateEmpty().
				Then(plan.CFUpdate(0))
		}
		return nil
	}

	if plan.Deployment.Flags.DbOnly {
		if plan.StackStatus == nil {
			return fmt.Errorf("cannot migrate databases without a stack")
		}
		discovery := plan.NOPDiscovery()
		_, err := plan.MigrateDatabases(ctx, discovery)
		return err
	}

	if plan.Deployment.Flags.QuickMode {
		if plan.StackStatus != nil {
			infraMigrate := plan.CFUpdate(1)
			_, err := plan.MigrateDatabases(ctx, infraMigrate)
			if err != nil {
				return err
			}

		} else {
			update := plan.CFCreateEmpty().
				Then(plan.CFUpdate(0))
			_, err := plan.MigrateDatabases(ctx, update)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Full blown slow deployment with DB.

	if plan.StackStatus != nil {
		infraReady := plan.ScaleDown().
			Then(plan.CFUpdate(0))
		dbSteps, err := plan.MigrateDatabases(ctx, infraReady)
		if err != nil {
			return err
		}
		plan.ScaleUp().
			DependsOn(infraReady).
			DependsOn(dbSteps...)
	} else {
		infraReady := plan.CFCreateEmpty().
			Then(plan.CFUpdate(0))
		dbSteps, err := plan.MigrateDatabases(ctx, infraReady)
		if err != nil {
			return err
		}
		plan.ScaleUp().
			DependsOn(infraReady).
			DependsOn(dbSteps...)
	}

	return nil
}

type StepSet[T awsdeployer_step_pb.Step] []T

func ActivateDeploymentStep(steps []*awsdeployer_pb.DeploymentStep, event *awsdeployer_pb.DeploymentEventType_RunStep) error {
	for _, step := range steps {
		meta := step.GetMeta()
		if meta.StepId == event.StepId {
			meta.Status = drss_pb.StepStatus_ACTIVE
		}
	}
	return nil
}

func RunStep(ctx context.Context,
	tb awsdeployer_pb.DeploymentPSMHookBaton,
	keys *awsdeployer_pb.DeploymentKeys,
	steps []*awsdeployer_pb.DeploymentStep,
	stepID string,
) error {
	wrapped := awsdeployer_step_pb.WrapSteps(steps)

	input, baton, err := wrapped.BuildBaton(stepID)
	if err != nil {
		return err
	}

	outcome, err := awsdeployer_step_pb.RunDeploymentSteps(&planbuild.StepBuild{}, input, baton)
	if err != nil {
		return err
	}

	switch outcome := outcome.(type) {
	case *strategy.RequestOutcome:
		requestMetadata, err := buildRequestMetadata(keys.DeploymentId, input.GetMeta().StepId)
		if err != nil {
			return err
		}
		sideEffect := outcome.Request
		sideEffect.SetJ5RequestMetadata(requestMetadata)
		tb.SideEffect(sideEffect)
		return nil

	case *strategy.DoneOutcome[*awsdeployer_pb.StepOutputType]:
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_StepResult{
			Result: &drss_pb.StepResult{
				StepId: input.GetMeta().StepId,
				Status: drss_pb.StepStatus_DONE,
			},
			Output: outcome.Output,
		})
		return nil

	default:
		return fmt.Errorf("unexpected outcome: %T", outcome)

	}
}

func UpdateDeploymentStep(steps []*awsdeployer_pb.DeploymentStep, event *awsdeployer_pb.DeploymentEventType_StepResult) error {
	stepMap := awsdeployer_step_pb.WrapSteps(steps)
	var output awsdeployer_pb.IsStepOutputTypeWrappedType
	if event.Output != nil {
		output = event.Output.Get()
	}
	return strategy.UpdateDeploymentStep(stepMap, event.Result, output)
}

func UpdateStepDependencies(steps []*awsdeployer_pb.DeploymentStep) error {
	stepMap := awsdeployer_step_pb.WrapSteps(steps)
	return strategy.UpdateStepDependencies(stepMap)
}

type Chainer interface {
	ChainEvent(event awsdeployer_pb.DeploymentPSMEvent)
}

func StepNext(ctx context.Context, tb Chainer, steps []*awsdeployer_pb.DeploymentStep) error {
	stepMap := awsdeployer_step_pb.WrapSteps(steps)
	outcome, err := strategy.StepNext(stepMap, func(stepID string) {
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_RunStep{
			StepId: stepID,
		})
	})

	if err != nil {
		return err
	}
	switch outcome {
	case strategy.StepNextFail:
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_Error{
			Error: "Deployment steps failed",
		})

	case strategy.StepNextDone:
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_Done{})

	case strategy.StepNextDeadlock:
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_Error{
			Error: "Deployment steps deadlock",
		})

	case strategy.StepNextWait:
		return nil
	}

	return nil
}

func buildRequestMetadata(deploymentID string, stepID string) (*messaging_j5pb.RequestMetadata, error) {
	contextMessage := &awsdeployer_pb.StepContext{
		StepId:       &stepID,
		Phase:        awsdeployer_pb.StepPhase_STEPS,
		DeploymentId: deploymentID,
	}

	contextBytes, err := proto.Marshal(contextMessage)
	if err != nil {
		return nil, err
	}

	req := &messaging_j5pb.RequestMetadata{
		ReplyTo: "o5-deployer",
		Context: contextBytes,
	}
	return req, nil
}
