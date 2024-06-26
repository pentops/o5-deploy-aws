package plan

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	"github.com/pentops/o5-messaging/o5msg"
	"google.golang.org/protobuf/proto"
)

type DeploymentInput struct {
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

func DeploymentSteps(ctx context.Context, input DeploymentInput) ([]*awsdeployer_pb.DeploymentStep, error) {

	deployment := input.Deployment

	plan := make([]*awsdeployer_pb.DeploymentStep, 0)

	stackInput := func(desiredCount int32) *awsdeployer_pb.CFStackInput {
		return &awsdeployer_pb.CFStackInput{
			StackName:    deployment.CfStackName,
			DesiredCount: desiredCount,
			Template: &awsdeployer_pb.CFStackInput_S3Template{
				S3Template: deployment.Template,
			},
			Parameters: deployment.Parameters,
			SnsTopics:  deployment.SnsTopics,
		}
	}

	if input.Deployment.Flags.ImportResources {

		if input.StackStatus != nil {
			return nil, fmt.Errorf("cannot import resources into an existing stack")
		}
		spec := stackInput(0)
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

		plan = append(plan, createEmptyStack)

		createChangeset := &awsdeployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   "CFPlan",
			Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
			Request: &awsdeployer_pb.StepRequestType{
				Type: &awsdeployer_pb.StepRequestType_CfPlan{
					CfPlan: &awsdeployer_pb.StepRequestType_CFPlan{
						Spec:            stackInput(0),
						ImportResources: true,
					},
				},
			},
		}

		plan = append(plan, createChangeset)

		return plan, nil

	}

	if input.Deployment.Flags.QuickMode || input.Deployment.Flags.InfraOnly {
		scale := int32(1)
		if input.Deployment.Flags.InfraOnly {
			scale = 0
		}
		if input.StackStatus != nil {
			infraMigrate := &awsdeployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "CFUpdate",
				Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
				Request: &awsdeployer_pb.StepRequestType{
					Type: &awsdeployer_pb.StepRequestType_CfUpdate{
						CfUpdate: &awsdeployer_pb.StepRequestType_CFUpdate{
							Spec: stackInput(scale),
						},
					},
				},
			}
			plan = append(plan, infraMigrate)
		} else {
			infraCreate := &awsdeployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "CFCreate",
				Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
				Request: &awsdeployer_pb.StepRequestType{
					Type: &awsdeployer_pb.StepRequestType_CfCreate{
						CfCreate: &awsdeployer_pb.StepRequestType_CFCreate{
							Spec: stackInput(scale),
						},
					},
				},
			}
			plan = append(plan, infraCreate)
		}
		// Databases are ignored.
		return plan, nil
	}

	var infraReadyStepID string

	if input.Deployment.Flags.DbOnly && input.StackStatus == nil {
		return nil, fmt.Errorf("cannot migrate databases without a stack")
	}

	var scaleUp *awsdeployer_pb.DeploymentStep

	if !input.Deployment.Flags.DbOnly {
		if input.StackStatus != nil {
			scaleDown := &awsdeployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "ScaleDown",
				Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
				Request: &awsdeployer_pb.StepRequestType{
					Type: &awsdeployer_pb.StepRequestType_CfScale{
						CfScale: &awsdeployer_pb.StepRequestType_CFScale{
							DesiredCount: 0,
							StackName:    deployment.CfStackName,
						},
					},
				},
			}
			plan = append(plan, scaleDown)

			infraMigrate := &awsdeployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "InfraMigrate",
				Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
				Request: &awsdeployer_pb.StepRequestType{
					Type: &awsdeployer_pb.StepRequestType_CfUpdate{
						CfUpdate: &awsdeployer_pb.StepRequestType_CFUpdate{
							Spec: stackInput(0),
						},
					},
				},
				DependsOn: []string{scaleDown.Id},
			}
			plan = append(plan, infraMigrate)
			infraReadyStepID = infraMigrate.Id
		} else {
			infraCreate := &awsdeployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "InfraCreate",
				Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
				Request: &awsdeployer_pb.StepRequestType{
					Type: &awsdeployer_pb.StepRequestType_CfCreate{
						CfCreate: &awsdeployer_pb.StepRequestType_CFCreate{
							Spec: stackInput(0),
						},
					},
				},
			}
			plan = append(plan, infraCreate)
			infraReadyStepID = infraCreate.Id
		}

		scaleUp = &awsdeployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   "ScaleUp",
			Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
			Request: &awsdeployer_pb.StepRequestType{
				Type: &awsdeployer_pb.StepRequestType_CfScale{
					CfScale: &awsdeployer_pb.StepRequestType_CFScale{
						DesiredCount: 1,
						StackName:    deployment.CfStackName,
					},
				},
			},
			DependsOn: []string{infraReadyStepID},
		}
		plan = append(plan, scaleUp)
	} else {
		// add a fake discovery step which is already completed.
		discoveryStep := &awsdeployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   "Discovery",
			Status: awsdeployer_pb.StepStatus_DONE,
			Output: &awsdeployer_pb.StepOutputType{
				Type: &awsdeployer_pb.StepOutputType_CfStatus{
					CfStatus: &awsdeployer_pb.StepOutputType_CFStatus{
						Output: input.StackStatus,
					},
				},
			},
		}
		plan = append(plan, discoveryStep)
		infraReadyStepID = discoveryStep.Id
	}

	for _, db := range deployment.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.DbName,
			"root":     db.RootSecretName,
		})
		log.Debug(ctx, "Upsert Database")

		upsertStep := &awsdeployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   fmt.Sprintf("PgUpsert-%s", db.DbName),
			Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
			Request: &awsdeployer_pb.StepRequestType{
				Type: &awsdeployer_pb.StepRequestType_PgUpsert{
					PgUpsert: &awsdeployer_pb.StepRequestType_PGUpsert{
						Spec:              db,
						InfraOutputStepId: infraReadyStepID,
						RotateCredentials: deployment.Flags.RotateCredentials,
					},
				},
			},
			DependsOn: []string{infraReadyStepID},
		}

		migrateStep := &awsdeployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   fmt.Sprintf("PgMigrate-%s", db.DbName),
			Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
			Request: &awsdeployer_pb.StepRequestType{
				Type: &awsdeployer_pb.StepRequestType_PgMigrate{
					PgMigrate: &awsdeployer_pb.StepRequestType_PGMigrate{
						Spec:              db,
						InfraOutputStepId: infraReadyStepID,
						EcsClusterName:    deployment.EcsCluster,
					},
				},
			},
			// depends on infraReady even though upsert also depends on it, so
			// that the output of the infraStep is still injected
			DependsOn: []string{infraReadyStepID, upsertStep.Id},
		}

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

		plan = append(plan, upsertStep, migrateStep, cleanupStep)
		if scaleUp != nil {
			scaleUp.DependsOn = append(scaleUp.DependsOn, cleanupStep.Id)
		}
	}

	return plan, nil
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

func buildRequestMetadata(deploymentID string, stepID string) (*messaging_pb.RequestMetadata, error) {
	contextMessage := &awsdeployer_pb.StepContext{
		StepId:       &stepID,
		Phase:        awsdeployer_pb.StepPhase_STEPS,
		DeploymentId: deploymentID,
	}

	contextBytes, err := proto.Marshal(contextMessage)
	if err != nil {
		return nil, err
	}

	req := &messaging_pb.RequestMetadata{
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
