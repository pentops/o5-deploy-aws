package planbuild

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan/awsdeployer_step_pb"
)

type StepBuild struct{}

const (
	StepScaleDown     = "ScaleDown"
	StepScaleUp       = "ScaleUp"
	StepNOPDiscovery  = "NOPDiscovery"
	StepCFUpdate      = "CF.Update"
	StepCFImportPlan  = "CF.ImportPlan"
	StepCFCreateEmpty = "CF.CreateEmpty"
	StepPgUpsert      = "DB.Upsert"
	StepPgMigrate     = "DB.Migrate"
	StepPgCleanup     = "DB.Cleanup"
	StepPgDestroy     = "DB.Destroy"
)

func QualifiedStep(parts ...string) string {
	return strings.Join(parts, ".")
}

type DeploymentPlan struct {
	Steps       []*PlanStep
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

func (plan *DeploymentPlan) GetSteps() []*awsdeployer_pb.DeploymentStep {
	steps := make([]*awsdeployer_pb.DeploymentStep, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		impl := &awsdeployer_pb.DeploymentStepType{}
		impl.Set(step.step)
		steps = append(steps, &awsdeployer_pb.DeploymentStep{
			Step: impl,
			Meta: step.meta,
		})
	}
	return steps
}

type PlanStep struct {
	step awsdeployer_pb.IsDeploymentStepTypeWrappedType
	meta *drss_pb.StepMeta
}

func (ps *PlanStep) DependsOn(depStep ...*PlanStep) *PlanStep {
	for _, step := range depStep {
		ps.meta.DependsOn = append(ps.meta.DependsOn, step.meta.StepId)
	}
	return ps
}

func (ps *PlanStep) Then(next *PlanStep) *PlanStep {
	next.DependsOn(ps)
	return next
}

func (db *DeploymentPlan) addStep(name string, step awsdeployer_pb.IsDeploymentStepTypeWrappedType) *PlanStep {
	ps := &PlanStep{
		step: step,
		meta: &drss_pb.StepMeta{
			StepId: uuid.NewString(),
			Name:   name,
		},
	}
	db.Steps = append(db.Steps, ps)
	return ps
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

func (plan *DeploymentPlan) ScaleDown() *PlanStep {
	return plan.Scale(StepScaleDown, 0)
}

func (plan *DeploymentPlan) ScaleUp() *PlanStep {
	return plan.Scale(StepScaleUp, 1)
}

func (plan *DeploymentPlan) Scale(name string, desiredCount int32) *PlanStep {
	return plan.addStep(name, &awsdeployer_pb.DeploymentStepType_CFScale{
		DesiredCount: desiredCount,
		StackName:    plan.Deployment.CfStackName,
	})
}

func (ds *StepBuild) RunCFScale(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_CFScale) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Request(&awsinfra_tpb.ScaleStackMessage{
		StackName:    req.StackName,
		DesiredCount: req.DesiredCount,
	})
}

// NopDiscovery add a fake discovery step which is already completed.
func (plan *DeploymentPlan) NOPDiscovery() *PlanStep {
	evalJoin := &awsdeployer_pb.DeploymentStepType_EvalJoin{
		StackOutput: plan.StackStatus,
	}
	return plan.addStep(StepNOPDiscovery, evalJoin)
}

func (ds *StepBuild) RunEvalJoin(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_EvalJoin) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Done(&awsdeployer_pb.StepOutputType_CFStackStatus{
		Lifecycle: req.StackOutput.Lifecycle,
		Outputs:   req.StackOutput.Outputs,
	})
}

func (plan *DeploymentPlan) CFUpdate(scale int32) *PlanStep {
	return plan.addStep(StepCFUpdate, &awsdeployer_pb.DeploymentStepType_CFUpdate{
		Spec: plan.stackInput(scale),
	})
}

func (ds *StepBuild) RunCFUpdate(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_CFUpdate) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Request(&awsinfra_tpb.UpdateStackMessage{
		Spec: req.Spec,
	})
}

func (plan *DeploymentPlan) CFCreateEmpty() *PlanStep {
	spec := plan.stackInput(0)
	spec.Template = &awsdeployer_pb.CFStackInput_EmptyStack{
		EmptyStack: true,
	}
	spec.Parameters = nil

	return plan.addStep(StepCFCreateEmpty, &awsdeployer_pb.DeploymentStepType_CFCreate{
		Spec:       spec,
		EmptyStack: true,
	})
}

func (ds *StepBuild) RunCFCreate(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_CFCreate) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Request(&awsinfra_tpb.CreateNewStackMessage{
		Spec:       req.Spec,
		EmptyStack: req.EmptyStack,
	})
}

func (plan *DeploymentPlan) ImportResources(ctx context.Context) *PlanStep {
	return plan.addStep(StepCFImportPlan, &awsdeployer_pb.DeploymentStepType_CFPlan{
		Spec:            plan.stackInput(0),
		ImportResources: true,
	})
}

func (ds *StepBuild) RunCFPlan(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_CFPlan) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Request(&awsinfra_tpb.CreateChangeSetMessage{
		Spec:            req.Spec,
		ImportResources: req.ImportResources,
	})
}

func (plan *DeploymentPlan) MigrateDatabases(ctx context.Context, infraReadyStep *PlanStep) ([]*PlanStep, error) {
	finalSteps := make([]*PlanStep, 0)
	for _, db := range plan.Deployment.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.AppKey,
		})
		log.Debug(ctx, "Upsert Database")

		lastStep, err := plan.UpsertDatabase(db, infraReadyStep)
		if err != nil {
			return nil, err
		}

		finalSteps = append(finalSteps, lastStep)
	}
	return finalSteps, nil
}

func (plan *DeploymentPlan) DestroyDatabases(ctx context.Context, infraReadyStep *PlanStep) ([]*PlanStep, error) {
	finalSteps := make([]*PlanStep, 0)
	for _, db := range plan.Deployment.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.AppKey,
		})
		log.Debug(ctx, "Destroy Database")

		destroyStep := plan.addStep(QualifiedStep(StepPgDestroy, db.AppKey),
			&awsdeployer_pb.DeploymentStepType_PGDestroy{
				Spec: db,
			},
		).DependsOn(infraReadyStep)

		finalSteps = append(finalSteps, destroyStep)
	}
	return finalSteps, nil
}

func (plan *DeploymentPlan) RecreateDatabases(ctx context.Context, infraReadyStep *PlanStep) ([]*PlanStep, error) {
	finalSteps := make([]*PlanStep, 0)
	for _, db := range plan.Deployment.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.AppKey,
		})
		log.Debug(ctx, "Recreate Database")

		destroyStep := plan.addStep(QualifiedStep(StepPgDestroy, db.AppKey),
			&awsdeployer_pb.DeploymentStepType_PGDestroy{
				Spec: db,
			},
		).DependsOn(infraReadyStep)

		upsertStep, err := plan.UpsertDatabase(db, infraReadyStep, destroyStep)
		if err != nil {
			return nil, err
		}

		finalSteps = append(finalSteps, upsertStep)
	}

	return finalSteps, nil
}

func (plan *DeploymentPlan) UpsertDatabase(db *awsdeployer_pb.PostgresSpec, infraReadyStep *PlanStep, extraDepends ...*PlanStep) (*PlanStep, error) {

	upsertStep := plan.addStep(QualifiedStep(StepPgUpsert, db.AppKey),
		&awsdeployer_pb.DeploymentStepType_PGUpsert{
			Spec:              db,
			InfraOutputStepId: infraReadyStep.meta.StepId,
			RotateCredentials: plan.Deployment.Flags.RotateCredentials,
		},
	).DependsOn(infraReadyStep)

	for _, dep := range extraDepends {
		upsertStep.DependsOn(dep)
	}

	if db.Migrate == nil {
		return upsertStep, nil
	}

	ecsMigrate := db.Migrate.GetEcs()
	if ecsMigrate == nil {
		return nil, fmt.Errorf("missing ECS spec for database %s", db.AppKey)
	}

	migrateStep := plan.addStep(QualifiedStep(StepPgMigrate, db.AppKey),
		&awsdeployer_pb.DeploymentStepType_PGMigrate{
			Spec:              db,
			InfraOutputStepId: infraReadyStep.meta.StepId,
			EcsContext:        ecsMigrate.TaskContext,
		},
	).DependsOn(infraReadyStep, upsertStep)
	// depends on infraReady even though upsert also depends on it, so
	// that the output of the infraStep is still injected

	cleanupStep := plan.addStep(QualifiedStep(StepPgCleanup, db.AppKey),
		&awsdeployer_pb.DeploymentStepType_PGCleanup{
			Spec: db,
		},
	).DependsOn(migrateStep)

	return cleanupStep, nil
}

func (ds *StepBuild) RunPGUpsert(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_PGUpsert) (awsdeployer_step_pb.Outcome, error) {
	src := req.Spec

	outputs, err := getStackOutputs(sb, req.InfraOutputStepId)
	if err != nil {
		return nil, err
	}
	appSpec := &awsinfra_pb.RDSAppSpecType{}
	switch conn := src.AppConnection.Get().(type) {
	case *awsdeployer_pb.PostgresConnectionType_Aurora:
		appSpec.Set(&awsinfra_pb.RDSAppSpecType_Aurora{
			Conn: conn.Conn,
		})

	case *awsdeployer_pb.PostgresConnectionType_SecretsManager:
		secret, ok := outputs.Find(conn.AppSecretOutputName)
		if !ok {
			return nil, fmt.Errorf("stack output missing %s for database %s", conn.AppSecretOutputName, src.AppKey)
		}

		appSpec.Set(&awsinfra_pb.RDSAppSpecType_SecretsManager{
			AppSecretName:     secret,
			RotateCredentials: req.RotateCredentials,
		})

	default:
		return nil, fmt.Errorf("unknown RDS spec type: %T", src.AppConnection.Get())
	}

	return awsdeployer_step_pb.Request(&awsinfra_tpb.UpsertPostgresDatabaseMessage{
		MigrationId: sb.GetID(),
		AdminHost:   src.AdminConnection,
		AppAccess:   appSpec,
		Spec: &awsinfra_pb.RDSCreateSpec{
			DbExtensions: src.DbExtensions,
			DbName:       src.FullDbName,
		},
	})
}

func (ds *StepBuild) RunPGMigrate(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_PGMigrate) (awsdeployer_step_pb.Outcome, error) {
	src := req.Spec
	ecsMigrate := req.Spec.Migrate.GetEcs()
	if ecsMigrate == nil {
		return nil, fmt.Errorf("missing ECS spec for database %s", src.AppKey)
	}

	msg := &awsinfra_tpb.RunECSTaskMessage{
		TaskDefinition: "",
		Count:          1,
		Context:        req.EcsContext,
	}

	outputs, err := getStackOutputs(sb, req.InfraOutputStepId)
	if err != nil {
		return nil, err
	}

	migrationTaskARN, ok := outputs.Find(ecsMigrate.TaskOutputName)
	if !ok {
		return nil, fmt.Errorf("stack output missing %s for database %s", ecsMigrate.TaskOutputName, src.AppKey)
	}
	msg.TaskDefinition = migrationTaskARN

	return awsdeployer_step_pb.Request(msg)
}

func (ds *StepBuild) RunPGCleanup(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_PGCleanup) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Request(&awsinfra_tpb.CleanupPostgresDatabaseMessage{
		MigrationId: sb.GetID(),
		DbName:      req.Spec.FullDbName,
		AdminHost:   req.Spec.AdminConnection,
	})
}

func (ds *StepBuild) RunPGDestroy(sb awsdeployer_step_pb.StepBaton, req *awsdeployer_pb.DeploymentStepType_PGDestroy) (awsdeployer_step_pb.Outcome, error) {
	return awsdeployer_step_pb.Request(&awsinfra_tpb.DestroyPostgresDatabaseMessage{
		MigrationId: sb.GetID(),
		DbName:      req.Spec.FullDbName,
		AdminHost:   req.Spec.AdminConnection,
	})
}

type StackOutputs []*awsdeployer_pb.KeyValue

func (s StackOutputs) Find(name string) (string, bool) {
	for _, kv := range s {
		if kv.Name == name {
			return kv.Value, true
		}
	}
	return "", false
}

func getStackOutputs(sb awsdeployer_step_pb.StepBaton, depID string) (StackOutputs, error) {

	infraOutputStep, ok := sb.GetDependency(depID)
	if !ok {
		return nil, fmt.Errorf("PG Migrate step depends on %s for CF output, not passed in", depID)
	}
	stackOutput, ok := infraOutputStep.(*awsdeployer_pb.StepOutputType_CFStackStatus)
	if !ok {
		return nil, fmt.Errorf("unexpected step type for CF Output: %T", infraOutputStep)
	}

	return StackOutputs(stackOutput.Outputs), nil
}
