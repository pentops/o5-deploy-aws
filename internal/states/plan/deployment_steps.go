package plan

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-messaging/o5msg"
	"google.golang.org/protobuf/proto"
)

type DeploymentInput struct {
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

type DeploymentPlan struct {
	Steps       []*awsdeployer_pb.DeploymentStep
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

func (db *DeploymentPlan) AddStep(step *awsdeployer_pb.DeploymentStep) {
	db.Steps = append(db.Steps, step)
}

func (plan *DeploymentPlan) stackInput(desiredCount int32) *awsdeployer_pb.CFStackInput {
	return &awsdeployer_pb.CFStackInput{
		StackName:    plan.Deployment.CfStackName,
		DesiredCount: desiredCount,
		Template: &awsdeployer_pb.CFStackInput_S3Template{
			S3Template: plan.Deployment.Template,
		},
		Parameters: plan.Deployment.Parameters,
		SnsTopics:  plan.Deployment.SnsTopics,
	}
}

func DeploymentSteps(ctx context.Context, input DeploymentInput) ([]*awsdeployer_pb.DeploymentStep, error) {
	plan := &DeploymentPlan{
		StackStatus: input.StackStatus,
		Deployment:  input.Deployment,
	}

	if err := plan.build(ctx); err != nil {
		return nil, err
	}
	return plan.Steps, nil
}

func (plan *DeploymentPlan) build(ctx context.Context) error {

	if plan.Deployment.Flags.ImportResources {
		return plan.ImportResources(ctx)
	}

	if plan.Deployment.Flags.InfraOnly {
		if plan.StackStatus != nil {
			plan.InfraMigrate(0)
		} else {
			plan.InfraCreate(0)
		}
		return nil
	}

	if plan.Deployment.Flags.QuickMode {
		if plan.StackStatus != nil {
			infraMigrate := plan.InfraMigrate(1)
			plan.MigrateDatabases(ctx, infraMigrate.Id)
		} else {
			infraCreate := plan.InfraCreate(0)
			plan.MigrateDatabases(ctx, infraCreate.Id)
		}
		return nil
	}

	if plan.Deployment.Flags.DbOnly {
		if plan.StackStatus == nil {
			return fmt.Errorf("cannot migrate databases without a stack")
		}
		discovery := plan.NopDiscovery()
		plan.MigrateDatabases(ctx, discovery.Id)
		return nil
	}

	// Full blown slow deployment with DB.

	var infraReadyStepID string
	if plan.StackStatus != nil {
		scaleDown := plan.ScaleDown()
		infraMigrate := plan.InfraMigrate(0, scaleDown.Id)
		infraReadyStepID = infraMigrate.Id
		finalDbIDs := plan.MigrateDatabases(ctx, infraReadyStepID)
		plan.ScaleUp(append(finalDbIDs, infraReadyStepID)...)
	} else {
		infraCreate := plan.InfraCreate(1)
		infraReadyStepID = infraCreate.Id
		finalDbIDs := plan.MigrateDatabases(ctx, infraReadyStepID)
		plan.ScaleUp(append(finalDbIDs, infraReadyStepID)...)
	}

	return nil
}

func (plan *DeploymentPlan) ScaleDown(dependsOn ...string) *awsdeployer_pb.DeploymentStep {
	return plan.Scale("ScaleDown", 0, dependsOn...)
}

func (plan *DeploymentPlan) ScaleUp(dependsOn ...string) *awsdeployer_pb.DeploymentStep {
	return plan.Scale("ScaleUp", 1, dependsOn...)
}

func (plan *DeploymentPlan) Scale(name string, desiredCount int32, dependsOn ...string) *awsdeployer_pb.DeploymentStep {
	scaleDown := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   name,
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_CfScale{
				CfScale: &awsdeployer_pb.StepRequestType_CFScale{
					DesiredCount: desiredCount,
					StackName:    plan.Deployment.CfStackName,
				},
			},
		},
		DependsOn: dependsOn,
	}

	plan.AddStep(scaleDown)
	return scaleDown
}

func (plan *DeploymentPlan) InfraMigrate(scale int32, dependsOn ...string) *awsdeployer_pb.DeploymentStep {
	infraMigrate := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   "InfraMigrate",
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_CfUpdate{
				CfUpdate: &awsdeployer_pb.StepRequestType_CFUpdate{
					Spec: plan.stackInput(scale),
				},
			},
		},
		DependsOn: dependsOn,
	}
	plan.AddStep(infraMigrate)
	return infraMigrate
}

func (plan *DeploymentPlan) InfraCreate(scale int32) *awsdeployer_pb.DeploymentStep {
	infraCreate := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   "InfraCreate",
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_CfCreate{
				CfCreate: &awsdeployer_pb.StepRequestType_CFCreate{
					Spec: plan.stackInput(scale),
				},
			},
		},
	}
	plan.AddStep(infraCreate)
	return infraCreate
}

// NopDiscovery add a fake discovery step which is already completed.
func (plan *DeploymentPlan) NopDiscovery() *awsdeployer_pb.DeploymentStep {
	discoveryStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   "Discovery",
		Status: awsdeployer_pb.StepStatus_DONE,
		Output: &awsdeployer_pb.StepOutputType{
			Type: &awsdeployer_pb.StepOutputType_CfStatus{
				CfStatus: &awsdeployer_pb.StepOutputType_CFStatus{
					Output: plan.StackStatus,
				},
			},
		},
	}
	plan.AddStep(discoveryStep)
	return discoveryStep

}

func (plan *DeploymentPlan) MigrateDatabases(ctx context.Context, infraReadyStepID string) []string {
	finalSteps := make([]string, 0)
	for _, db := range plan.Deployment.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.DbName,
			"root":     db.RootSecretName,
		})
		log.Debug(ctx, "Upsert Database")

		lastStep := plan.UpsertDatabase(db, infraReadyStepID)

		finalSteps = append(finalSteps, lastStep.Id)
	}
	return finalSteps
}

func (plan *DeploymentPlan) UpsertDatabase(db *awsdeployer_pb.PostgresSpec, infraReadyStepID string) *awsdeployer_pb.DeploymentStep {

	upsertStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   fmt.Sprintf("PgUpsert-%s", db.DbName),
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_PgUpsert{
				PgUpsert: &awsdeployer_pb.StepRequestType_PGUpsert{
					Spec:              db,
					InfraOutputStepId: infraReadyStepID,
					RotateCredentials: plan.Deployment.Flags.RotateCredentials,
				},
			},
		},
		DependsOn: []string{infraReadyStepID},
	}
	plan.AddStep(upsertStep)

	migrateStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   fmt.Sprintf("PgMigrate-%s", db.DbName),
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_PgMigrate{
				PgMigrate: &awsdeployer_pb.StepRequestType_PGMigrate{
					Spec:              db,
					InfraOutputStepId: infraReadyStepID,
					EcsClusterName:    plan.Deployment.EcsCluster,
				},
			},
		},
		// depends on infraReady even though upsert also depends on it, so
		// that the output of the infraStep is still injected
		DependsOn: []string{infraReadyStepID, upsertStep.Id},
	}
	plan.AddStep(migrateStep)

	cleanupStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   fmt.Sprintf("PgCleanup-%s", db.DbName),
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_PgCleanup{
				PgCleanup: &awsdeployer_pb.StepRequestType_PGCleanup{
					Spec: db,
				},
			},
		},
		DependsOn: []string{migrateStep.Id},
	}
	plan.AddStep(cleanupStep)

	return cleanupStep
}

func (plan *DeploymentPlan) ImportResources(ctx context.Context) error {
	spec := plan.stackInput(0)
	spec.Template = &awsdeployer_pb.CFStackInput_EmptyStack{
		EmptyStack: true,
	}
	spec.Parameters = nil

	createEmptyStack := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   "CFCreate",
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_CfCreate{
				CfCreate: &awsdeployer_pb.StepRequestType_CFCreate{
					Spec:       spec,
					EmptyStack: true,
				},
			},
		},
	}

	plan.AddStep(createEmptyStack)

	createChangeset := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   "CFPlan",
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_CfPlan{
				CfPlan: &awsdeployer_pb.StepRequestType_CFPlan{
					Spec:            plan.stackInput(0),
					ImportResources: true,
				},
			},
		},
	}

	plan.AddStep(createChangeset)

	return nil
}

func ActivateDeploymentStep(steps []*awsdeployer_pb.DeploymentStep, event *awsdeployer_pb.DeploymentEventType_RunStep) error {
	for _, step := range steps {
		if step.Id == event.StepId {
			step.Status = awsdeployer_pb.StepStatus_ACTIVE
		}
	}
	return nil
}

func RunStep(ctx context.Context, keys *awsdeployer_pb.DeploymentKeys, steps []*awsdeployer_pb.DeploymentStep, event *awsdeployer_pb.DeploymentEventType_RunStep) (o5msg.Message, error) {

	stepMap := map[string]*awsdeployer_pb.DeploymentStep{}
	for _, search := range steps {
		stepMap[search.Id] = search
	}

	thisStep, ok := stepMap[event.StepId]
	if !ok {
		return nil, fmt.Errorf("step not found: %s", event.StepId)
	}

	depMap := map[string]*awsdeployer_pb.DeploymentStep{}

	for _, dep := range thisStep.DependsOn {
		depMap[dep], ok = stepMap[dep]
		if !ok {
			return nil, fmt.Errorf("dependency not found: %s", dep)
		}
	}

	return stepToSideEffect(thisStep, keys, depMap)
}

func UpdateDeploymentStep(steps []*awsdeployer_pb.DeploymentStep, event *awsdeployer_pb.DeploymentEventType_StepResult) error {

	for _, step := range steps {
		if step.Id == event.StepId {
			step.Status = event.Status
			step.Output = event.Output
			step.Error = event.Error

			if event.Status == awsdeployer_pb.StepStatus_DONE {
				// If the step is done, we can update the dependencies
				return UpdateStepDependencies(steps)
			}
			return nil
		}
	}

	return fmt.Errorf("step %s not found", event.StepId)
}

func UpdateStepDependencies(steps []*awsdeployer_pb.DeploymentStep) error {

	stepMap := make(map[string]*awsdeployer_pb.DeploymentStep)

	for _, step := range steps {
		stepMap[step.Id] = step
	}

	for _, step := range steps {
		if step.Status != awsdeployer_pb.StepStatus_BLOCKED && step.Status != awsdeployer_pb.StepStatus_UNSPECIFIED {
			continue
		}

		isBlocked := false
		for _, dep := range step.DependsOn {
			depStep, ok := stepMap[dep]
			if !ok {
				return fmt.Errorf("step %s depends on %s, but it does not exist", step.Id, dep)
			}
			if depStep.Status != awsdeployer_pb.StepStatus_DONE {
				isBlocked = true
				break
			}
		}

		if isBlocked {
			step.Status = awsdeployer_pb.StepStatus_BLOCKED
		} else {
			step.Status = awsdeployer_pb.StepStatus_READY
		}
	}

	return nil
}

type Chainer interface {
	ChainEvent(event awsdeployer_pb.DeploymentPSMEvent)
}

func StepNext(ctx context.Context, tb Chainer, steps []*awsdeployer_pb.DeploymentStep) error {
	stepMap := make(map[string]*awsdeployer_pb.DeploymentStep)
	for _, step := range steps {
		stepMap[step.Id] = step
	}

	anyOpen := false
	anyRunning := false
	anyFailed := false

	readySteps := make([]*awsdeployer_pb.DeploymentStep, 0)

	for _, step := range steps {
		switch step.Status {
		case awsdeployer_pb.StepStatus_BLOCKED:
			// Do nothing

		case awsdeployer_pb.StepStatus_READY:
			readySteps = append(readySteps, step)

		case awsdeployer_pb.StepStatus_ACTIVE:
			anyRunning = true

		case awsdeployer_pb.StepStatus_DONE:
			// Do Nothing

		case awsdeployer_pb.StepStatus_FAILED:
			anyFailed = true

		default:
			return fmt.Errorf("unexpected step status: %s", step.Status)
		}

	}

	if anyFailed {
		if anyRunning {
			// Wait for cleanup
			return nil
		}

		// Nothing still running, we don't need to trigger any further
		// tasks.
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_Error{
			Error: "deployment steps failed",
		})
		return nil
	}

	// allows steps to pull outputs from previous steps.

	for _, step := range readySteps {
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_RunStep{
			StepId: step.Id,
		})

		anyRunning = true
	}

	if anyRunning {
		return nil
	}

	if anyOpen {
		tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_Error{
			Error: "deployment steps deadlock",
		})
		return nil
	}

	// All steps are done
	tb.ChainEvent(&awsdeployer_pb.DeploymentEventType_Done{})

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

func stepToSideEffect(step *awsdeployer_pb.DeploymentStep, keys *awsdeployer_pb.DeploymentKeys, dependencies map[string]*awsdeployer_pb.DeploymentStep) (o5msg.Message, error) {

	requestMetadata, err := buildRequestMetadata(keys.DeploymentId, step.Id)
	if err != nil {
		return nil, err
	}
	switch st := step.Request.Type.(type) {
	case *awsdeployer_pb.StepRequestType_CfScale:
		return &awsinfra_tpb.ScaleStackMessage{
			Request:      requestMetadata,
			StackName:    st.CfScale.StackName,
			DesiredCount: st.CfScale.DesiredCount,
		}, nil

	case *awsdeployer_pb.StepRequestType_CfUpdate:
		return &awsinfra_tpb.UpdateStackMessage{
			Request: requestMetadata,
			Spec:    st.CfUpdate.Spec,
		}, nil

	case *awsdeployer_pb.StepRequestType_CfCreate:
		return &awsinfra_tpb.CreateNewStackMessage{
			Request:    requestMetadata,
			Spec:       st.CfCreate.Spec,
			EmptyStack: st.CfCreate.EmptyStack,
		}, nil

	case *awsdeployer_pb.StepRequestType_CfPlan:
		return &awsinfra_tpb.CreateChangeSetMessage{
			Request:         requestMetadata,
			Spec:            st.CfPlan.Spec,
			ImportResources: st.CfPlan.ImportResources,
		}, nil

	case *awsdeployer_pb.StepRequestType_PgUpsert:
		src := st.PgUpsert.Spec
		spec := &awsdeployer_pb.PostgresCreationSpec{
			DbName:            src.DbName,
			RootSecretName:    src.RootSecretName,
			DbExtensions:      src.DbExtensions,
			RotateCredentials: st.PgUpsert.RotateCredentials,
		}
		outputs, err := getStackOutputs(dependencies, st.PgUpsert.InfraOutputStepId)
		if err != nil {
			return nil, err
		}

		for _, output := range outputs {
			if src.SecretOutputName == output.Name {
				spec.SecretArn = output.Value
			}
		}
		if spec.SecretArn == "" {
			return nil, fmt.Errorf("stack output missing %s for database %s", src.SecretOutputName, src.DbName)
		}

		return &awsinfra_tpb.UpsertPostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			Spec:        spec,
		}, nil

	case *awsdeployer_pb.StepRequestType_PgMigrate:
		src := st.PgMigrate.Spec
		spec := &awsdeployer_pb.PostgresMigrationSpec{
			EcsClusterName: st.PgMigrate.EcsClusterName,
		}
		outputs, err := getStackOutputs(dependencies, st.PgMigrate.InfraOutputStepId)
		if err != nil {
			return nil, err
		}

		for _, output := range outputs {
			if *src.MigrationTaskOutputName == output.Name {
				spec.MigrationTaskArn = output.Value
			}
			if src.SecretOutputName == output.Name {
				spec.SecretArn = output.Value
			}
		}
		if spec.MigrationTaskArn == "" {
			return nil, fmt.Errorf("stack output missing %s for database %s", *src.MigrationTaskOutputName, src.DbName)
		}
		if spec.SecretArn == "" {
			return nil, fmt.Errorf("stack output missing %s for database %s", src.SecretOutputName, src.DbName)
		}

		return &awsinfra_tpb.MigratePostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			Spec:        spec,
		}, nil

	case *awsdeployer_pb.StepRequestType_PgCleanup:
		src := st.PgCleanup.Spec
		spec := &awsdeployer_pb.PostgresCleanupSpec{
			DbName:         src.DbName,
			RootSecretName: src.RootSecretName,
		}
		return &awsinfra_tpb.CleanupPostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			Spec:        spec,
		}, nil

	default:
		return nil, fmt.Errorf("unknown step type: %T", step.Request.Type)
	}
}

func getStackOutputs(dependencies map[string]*awsdeployer_pb.DeploymentStep, id string) ([]*awsdeployer_pb.KeyValue, error) {

	infraOutputStep, ok := dependencies[id]
	if !ok {
		return nil, fmt.Errorf("PG Migrate step depends on %s for CF output, not passed in", id)
	}
	stackOutput, ok := infraOutputStep.Output.Type.(*awsdeployer_pb.StepOutputType_CfStatus)
	if !ok {
		return nil, fmt.Errorf("unexpected step type for CF Output: %T", infraOutputStep.Output)
	}

	return stackOutput.CfStatus.Output.Outputs, nil
}
