package plan

import (
	"context"
	"fmt"

	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/states/plan/planbuild"
	"github.com/pentops/o5-messaging/o5msg"
	"google.golang.org/protobuf/proto"
)

type DeploymentInput struct {
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

func DeploymentSteps(ctx context.Context, input DeploymentInput) ([]*awsdeployer_pb.DeploymentStep, error) {
	plan := &planbuild.DeploymentPlan{
		StackStatus: input.StackStatus,
		Deployment:  input.Deployment,
	}

	if err := buildPlan(ctx, plan); err != nil {
		return nil, err
	}
	return plan.Steps, nil
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

	if plan.Deployment.Flags.QuickMode {
		if plan.StackStatus != nil {
			infraMigrate := plan.CFUpdate(1)
			plan.MigrateDatabases(ctx, infraMigrate)
		} else {
			update := plan.CFCreateEmpty().
				Then(plan.CFUpdate(0))
			plan.MigrateDatabases(ctx, update)
		}
		return nil
	}

	if plan.Deployment.Flags.DbOnly {
		if plan.StackStatus == nil {
			return fmt.Errorf("cannot migrate databases without a stack")
		}
		discovery := plan.NOPDiscovery()
		plan.MigrateDatabases(ctx, discovery)
		return nil
	}

	// Full blown slow deployment with DB.

	if plan.StackStatus != nil {
		infraReady := plan.ScaleDown().
			Then(plan.CFUpdate(0))
		dbSteps := plan.MigrateDatabases(ctx, infraReady)
		plan.ScaleUp().
			DependsOn(infraReady).
			DependsOn(dbSteps...)
	} else {
		infraReady := plan.CFCreateEmpty().
			Then(plan.CFUpdate(0))
		dbSteps := plan.MigrateDatabases(ctx, infraReady)
		plan.ScaleUp().
			DependsOn(infraReady).
			DependsOn(dbSteps...)
	}

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

	log.WithFields(ctx, map[string]interface{}{
		"stepId":   thisStep.Id,
		"stepName": thisStep.Name,
	}).Info("Run Step")

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

		outputs, err := getStackOutputs(dependencies, st.PgUpsert.InfraOutputStepId)
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
			secretARN, ok := outputs.Find(conn.AppSecretOutputName)
			if !ok {
				return nil, fmt.Errorf("stack output missing %s for database %s", conn.AppSecretOutputName, src.DbName)
			}

			appSpec.Set(&awsinfra_pb.RDSAppSpecType_SecretsManager{
				AppSecretArn:      secretARN,
				RotateCredentials: st.PgUpsert.RotateCredentials,
			})

		default:
			return nil, fmt.Errorf("unknown RDS spec type: %T", src.AppConnection.Get())
		}

		return &awsinfra_tpb.UpsertPostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			AdminHost:   src.AdminConnection,
			AppAccess:   appSpec,
			Spec: &awsinfra_pb.RDSCreateSpec{
				DbExtensions: src.DbExtensions,
				DbName:       src.DbName,
			},
		}, nil

	case *awsdeployer_pb.StepRequestType_PgMigrate:
		src := st.PgMigrate.Spec
		msg := &awsinfra_tpb.MigratePostgresDatabaseMessage{
			Request:        requestMetadata,
			MigrationId:    step.Id,
			EcsClusterName: st.PgMigrate.EcsClusterName,

			// Explicitly Default
			MigrationTaskArn: "",
			SecretArn:        "",
		}

		outputs, err := getStackOutputs(dependencies, st.PgMigrate.InfraOutputStepId)
		if err != nil {
			return nil, err
		}

		if src.MigrationTaskOutputName != nil {
			migrationTaskARN, ok := outputs.Find(*src.MigrationTaskOutputName)
			if !ok {
				return nil, fmt.Errorf("stack output missing %s for database %s", *src.MigrationTaskOutputName, src.DbName)
			}
			msg.MigrationTaskArn = migrationTaskARN
		}

		if smSpec := src.AppConnection.GetSecretsManager(); smSpec != nil {
			secretARN, ok := outputs.Find(smSpec.AppSecretOutputName)
			if !ok {
				return nil, fmt.Errorf("stack output missing %s for database %s", smSpec.AppSecretOutputName, src.DbName)
			}
			msg.SecretArn = secretARN
		}

		return msg, nil

	case *awsdeployer_pb.StepRequestType_PgCleanup:
		src := st.PgCleanup.Spec

		return &awsinfra_tpb.CleanupPostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			DbName:      src.DbName,
			AdminHost:   src.AdminConnection,
		}, nil

	default:
		return nil, fmt.Errorf("unknown step type: %T", step.Request.Type)
	}
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

func getStackOutputs(dependencies map[string]*awsdeployer_pb.DeploymentStep, id string) (StackOutputs, error) {

	infraOutputStep, ok := dependencies[id]
	if !ok {
		return nil, fmt.Errorf("PG Migrate step depends on %s for CF output, not passed in", id)
	}
	stackOutput, ok := infraOutputStep.Output.Type.(*awsdeployer_pb.StepOutputType_CfStatus)
	if !ok {
		return nil, fmt.Errorf("unexpected step type for CF Output: %T", infraOutputStep.Output)
	}

	return StackOutputs(stackOutput.CfStatus.Output.Outputs), nil
}
