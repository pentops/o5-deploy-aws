package awsdeployer_step_pb

import (
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan/strategy"
)

type Outcome = strategy.Outcome[*awsdeployer_pb.StepOutputType]

func Done(output awsdeployer_pb.IsStepOutputTypeWrappedType) (Outcome, error) {
	var result *awsdeployer_pb.StepOutputType
	if output != nil {
		result = &awsdeployer_pb.StepOutputType{}
		result.Set(output)
	}
	return &strategy.DoneOutcome[*awsdeployer_pb.StepOutputType]{
		Output: result,
	}, nil
}

func Request(req strategy.SideEffect) (Outcome, error) {
	return &strategy.RequestOutcome{
		Request: req,
	}, nil
}

type DeploymentStep interface {
	SetOutput(awsdeployer_pb.IsStepOutputTypeWrappedType)
	GetOutput() *awsdeployer_pb.StepOutputType
}

type DeploymentSteps interface {
	RunEvalJoin(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_EvalJoin) (Outcome, error)
	RunCFScale(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_CFScale) (Outcome, error)
	RunCFUpdate(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_CFUpdate) (Outcome, error)
	RunCFCreate(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_CFCreate) (Outcome, error)
	RunCFPlan(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_CFPlan) (Outcome, error)
	RunPGUpsert(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_PGUpsert) (Outcome, error)
	RunPGMigrate(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_PGMigrate) (Outcome, error)
	RunPGCleanup(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_PGCleanup) (Outcome, error)
	RunPGDestroy(deps StepBaton, req *awsdeployer_pb.DeploymentStepType_PGDestroy) (Outcome, error)
}

type StepBaton interface {
	GetDependency(id string) (awsdeployer_pb.IsStepOutputTypeWrappedType, bool)
	GetID() string
}

func RunDeploymentSteps(steps DeploymentSteps, thisStep strategy.Step[
	awsdeployer_pb.IsDeploymentStepTypeWrappedType,
	awsdeployer_pb.IsStepOutputTypeWrappedType,
], deps StepBaton) (Outcome, error) {
	switch st := thisStep.GetStep().(type) {
	case *awsdeployer_pb.DeploymentStepType_EvalJoin:
		return steps.RunEvalJoin(deps, st)
	case *awsdeployer_pb.DeploymentStepType_CFScale:
		return steps.RunCFScale(deps, st)
	case *awsdeployer_pb.DeploymentStepType_CFUpdate:
		return steps.RunCFUpdate(deps, st)
	case *awsdeployer_pb.DeploymentStepType_CFCreate:
		return steps.RunCFCreate(deps, st)
	case *awsdeployer_pb.DeploymentStepType_CFPlan:
		return steps.RunCFPlan(deps, st)
	case *awsdeployer_pb.DeploymentStepType_PGUpsert:
		return steps.RunPGUpsert(deps, st)
	case *awsdeployer_pb.DeploymentStepType_PGMigrate:
		return steps.RunPGMigrate(deps, st)
	case *awsdeployer_pb.DeploymentStepType_PGCleanup:
		return steps.RunPGCleanup(deps, st)
	case *awsdeployer_pb.DeploymentStepType_PGDestroy:
		return steps.RunPGDestroy(deps, st)
	default:
		return nil, fmt.Errorf("unexpected step type: %T", st)
	}
}

type Step struct {
	step *awsdeployer_pb.DeploymentStep
}

func (s *Step) SetOutput(output awsdeployer_pb.IsStepOutputTypeWrappedType) {
	if s.step.Output == nil {
		s.step.Output = &awsdeployer_pb.StepOutputType{}
	}
	s.step.Output.Set(output)
}

func (s *Step) GetOutput() awsdeployer_pb.IsStepOutputTypeWrappedType {
	if s.step.Output == nil {
		return nil
	}
	return s.step.Output.Get()
}

func (s *Step) GetMeta() *drss_pb.StepMeta {
	return s.step.Meta
}

func (s *Step) GetStep() awsdeployer_pb.IsDeploymentStepTypeWrappedType {
	return s.step.Step.Get()
}

func WrapSteps(steps []*awsdeployer_pb.DeploymentStep) strategy.StepSet[
	awsdeployer_pb.IsDeploymentStepTypeWrappedType,
	awsdeployer_pb.IsStepOutputTypeWrappedType,
] {
	wrapped := make(strategy.StepSet[
		awsdeployer_pb.IsDeploymentStepTypeWrappedType,
		awsdeployer_pb.IsStepOutputTypeWrappedType,
	], len(steps))
	for _, step := range steps {
		wrapped[step.Meta.StepId] = &Step{step: step}
	}
	return wrapped
}
