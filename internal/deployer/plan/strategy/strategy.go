package strategy

import (
	"fmt"

	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-messaging/o5msg"
)

type SideEffect interface {
	GetRequest() *messaging_j5pb.RequestMetadata
	SetJ5RequestMetadata(*messaging_j5pb.RequestMetadata)
	o5msg.Message
}

type Step[T any, O any] interface {
	GetMeta() *drss_pb.StepMeta
	GetOutput() O
	SetOutput(O)
	GetStep() T
}

type StepSet[I any, O any] map[string]Step[I, O]

func (ss StepSet[I, O]) BuildBaton(stepID string) (Step[I, O], *StepBatonImpl[I, O], error) {
	thisStep, ok := ss[stepID]
	if !ok {
		return nil, nil, fmt.Errorf("step not found: %s", stepID)
	}

	depMap := map[string]O{}

	for _, dependencyId := range thisStep.GetMeta().DependsOn {
		foundStep, ok := ss[dependencyId]
		if !ok {
			return nil, nil, fmt.Errorf("dependency not found: %s", dependencyId)
		}
		output := foundStep.GetOutput()
		depMap[dependencyId] = output
	}

	return thisStep, &StepBatonImpl[I, O]{
		step:         thisStep.GetMeta(),
		dependencies: depMap,
	}, nil
}

type StepBatonImpl[I any, O any] struct {
	step         *drss_pb.StepMeta
	dependencies map[string]O
	sideEffects  []SideEffect
}

func NewStepBaton[I any, O any](step Step[I, O], deps map[string]Step[I, O]) *StepBatonImpl[I, O] {
	return &StepBatonImpl[I, O]{
		step:         step.GetMeta(),
		dependencies: map[string]O{},
	}
}

func (ds *StepBatonImpl[I, O]) GetDependency(id string) (dep O, ok bool) {
	if dep, ok := ds.dependencies[id]; ok {
		return dep, true
	}
	return
}

func (ds *StepBatonImpl[I, O]) GetID() string {
	return ds.step.StepId
}

func (ds *StepBatonImpl[I, O]) SideEffect(effect SideEffect) {
	ds.sideEffects = append(ds.sideEffects, effect)
}

func (ds *StepBatonImpl[I, O]) Done() {
}

type Outcome[O any] interface {
	isOutcome()
}

type RequestOutcome struct {
	Request SideEffect
}

var _ Outcome[*awsdeployer_pb.StepOutputType] = RequestOutcome{}
var _ Outcome[DoneOutcome[*awsdeployer_pb.StepOutputType]] = DoneOutcome[awsdeployer_pb.StepOutputType]{}

func (RequestOutcome) isOutcome() {}

type DoneOutcome[O any] struct {
	Output O
}

func (DoneOutcome[O]) isOutcome() {}

type EventOutput[O any] interface {
	GetOutput() O
	GetStepId() string
	GetStatus() drss_pb.StepStatus
	GetError() *string
}

func UpdateDeploymentStep[I any, O any](steps StepSet[I, O], result *drss_pb.StepResult, outputData O) error {
	for _, step := range steps {
		sm := step.GetMeta()
		if sm.StepId == result.StepId {
			sm.Status = result.Status
			sm.Error = result.Error
			step.SetOutput(outputData)

			if sm.Status == drss_pb.StepStatus_DONE {
				// If the step is done, we can update the dependencies
				return UpdateStepDependencies(steps)
			}
			return nil
		}
	}

	return fmt.Errorf("step %s not found", result.StepId)
}

func UpdateStepDependencies[I any, O any](stepMap StepSet[I, O]) error {

	for _, step := range stepMap {
		if step.GetMeta().Status != drss_pb.StepStatus_BLOCKED && step.GetMeta().Status != drss_pb.StepStatus_UNSPECIFIED {
			continue
		}

		isBlocked := false
		for _, dep := range step.GetMeta().DependsOn {
			depStep, ok := stepMap[dep]
			if !ok {
				return fmt.Errorf("step %s depends on %s, but it does not exist", step.GetMeta().StepId, dep)
			}
			if depStep.GetMeta().Status != drss_pb.StepStatus_DONE {
				isBlocked = true
				break
			}
		}

		if isBlocked {
			step.GetMeta().Status = drss_pb.StepStatus_BLOCKED
		} else {
			step.GetMeta().Status = drss_pb.StepStatus_READY
		}
	}

	return nil
}

type StepNextOutcome int

const (
	StepNextWait StepNextOutcome = iota
	StepNextFail
	StepNextDone
	StepNextDeadlock
)

func StepNext[I any, O any](stepMap StepSet[I, O], start func(string)) (StepNextOutcome, error) {

	anyOpen := false
	anyRunning := false
	anyFailed := false

	readySteps := make([]string, 0)

	for _, step := range stepMap {
		sm := step.GetMeta()
		switch sm.Status {
		case drss_pb.StepStatus_BLOCKED:
			// Do nothing

		case drss_pb.StepStatus_READY:
			readySteps = append(readySteps, sm.StepId)

		case drss_pb.StepStatus_ACTIVE:
			anyRunning = true

		case drss_pb.StepStatus_DONE:
			// Do Nothing

		case drss_pb.StepStatus_FAILED:
			anyFailed = true

		default:
			return 0, fmt.Errorf("unexpected step status: %s", sm.Status)
		}

	}

	if anyFailed {
		if anyRunning {
			// Wait for cleanup
			return StepNextWait, nil
		}

		// Nothing still running, we don't need to trigger any further
		// tasks.
		return StepNextFail, nil
	}

	if len(readySteps) > 0 {
		for _, step := range readySteps {
			start(step)
		}
		anyRunning = true
	}

	if anyRunning {
		return StepNextWait, nil
	}

	if anyOpen {
		return StepNextDeadlock, nil
	}

	return StepNextDone, nil
}
