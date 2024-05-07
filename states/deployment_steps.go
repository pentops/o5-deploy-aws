package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
)

type planInput struct {
	stackStatus *deployer_pb.CFStackOutput
	flags       *deployer_pb.DeploymentFlags
}

func planDeploymentSteps(ctx context.Context, deployment *deployer_pb.DeploymentState, input planInput) ([]*deployer_pb.DeploymentStep, error) {

	plan := make([]*deployer_pb.DeploymentStep, 0)

	stackInput := func(desiredCount int32) *deployer_pb.CFStackInput {
		return &deployer_pb.CFStackInput{
			StackName:    deployment.StackName,
			DesiredCount: desiredCount,
			TemplateUrl:  deployment.Spec.TemplateUrl,
			Parameters:   deployment.Spec.Parameters,
			SnsTopics:    deployment.Spec.SnsTopics,
		}
	}

	if input.flags.QuickMode || input.flags.InfraOnly {
		scale := int32(1)
		if input.flags.InfraOnly {
			scale = 0
		}
		if input.stackStatus != nil {
			infraMigrate := &deployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "CFUpdate",
				Status: deployer_pb.StepStatus_UNSPECIFIED,
				Request: &deployer_pb.StepRequestType{
					Type: &deployer_pb.StepRequestType_CfUpdate{
						CfUpdate: &deployer_pb.StepRequestType_CFUpdate{
							Spec: stackInput(scale),
						},
					},
				},
			}
			plan = append(plan, infraMigrate)
		} else {
			infraCreate := &deployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "CFCreate",
				Status: deployer_pb.StepStatus_UNSPECIFIED,
				Request: &deployer_pb.StepRequestType{
					Type: &deployer_pb.StepRequestType_CfCreate{
						CfCreate: &deployer_pb.StepRequestType_CFCreate{
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

	if input.flags.DbOnly && input.stackStatus == nil {
		return nil, fmt.Errorf("cannot migrate databases without a stack")
	}

	var scaleUp *deployer_pb.DeploymentStep

	if !input.flags.DbOnly {
		if input.stackStatus != nil {
			scaleDown := &deployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "ScaleDown",
				Status: deployer_pb.StepStatus_UNSPECIFIED,
				Request: &deployer_pb.StepRequestType{
					Type: &deployer_pb.StepRequestType_CfScale{
						CfScale: &deployer_pb.StepRequestType_CFScale{
							DesiredCount: 0,
							StackName:    deployment.StackName,
						},
					},
				},
			}
			plan = append(plan, scaleDown)

			infraMigrate := &deployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "InfraMigrate",
				Status: deployer_pb.StepStatus_UNSPECIFIED,
				Request: &deployer_pb.StepRequestType{
					Type: &deployer_pb.StepRequestType_CfUpdate{
						CfUpdate: &deployer_pb.StepRequestType_CFUpdate{
							Spec: stackInput(0),
						},
					},
				},
				DependsOn: []string{scaleDown.Id},
			}
			plan = append(plan, infraMigrate)
			infraReadyStepID = infraMigrate.Id
		} else {
			infraCreate := &deployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "InfraCreate",
				Status: deployer_pb.StepStatus_UNSPECIFIED,
				Request: &deployer_pb.StepRequestType{
					Type: &deployer_pb.StepRequestType_CfCreate{
						CfCreate: &deployer_pb.StepRequestType_CFCreate{
							Spec: stackInput(0),
						},
					},
				},
			}
			plan = append(plan, infraCreate)
			infraReadyStepID = infraCreate.Id
		}

		scaleUp = &deployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   "ScaleUp",
			Status: deployer_pb.StepStatus_UNSPECIFIED,
			Request: &deployer_pb.StepRequestType{
				Type: &deployer_pb.StepRequestType_CfScale{
					CfScale: &deployer_pb.StepRequestType_CFScale{
						DesiredCount: 1,
						StackName:    deployment.StackName,
					},
				},
			},
			DependsOn: []string{infraReadyStepID},
		}
		plan = append(plan, scaleUp)
	} else {
		// add a fake discovery step which is already completed.
		discoveryStep := &deployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   "Discovery",
			Status: deployer_pb.StepStatus_DONE,
			Output: &deployer_pb.StepOutputType{
				Type: &deployer_pb.StepOutputType_CfStatus{
					CfStatus: &deployer_pb.StepOutputType_CFStatus{
						Output: input.stackStatus,
					},
				},
			},
		}
		plan = append(plan, discoveryStep)
		infraReadyStepID = discoveryStep.Id
	}

	for _, db := range deployment.Spec.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.DbName,
			"root":     db.RootSecretName,
		})
		log.Debug(ctx, "Upsert Database")

		upsertStep := &deployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   fmt.Sprintf("PgUpsert-%s", db.DbName),
			Status: deployer_pb.StepStatus_UNSPECIFIED,
			Request: &deployer_pb.StepRequestType{
				Type: &deployer_pb.StepRequestType_PgUpsert{
					PgUpsert: &deployer_pb.StepRequestType_PGUpsert{
						Spec:              db,
						InfraOutputStepId: infraReadyStepID,
					},
				},
			},
			DependsOn: []string{infraReadyStepID},
		}

		migrateStep := &deployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   fmt.Sprintf("PgMigrate-%s", db.DbName),
			Status: deployer_pb.StepStatus_UNSPECIFIED,
			Request: &deployer_pb.StepRequestType{
				Type: &deployer_pb.StepRequestType_PgMigrate{
					PgMigrate: &deployer_pb.StepRequestType_PGMigrate{
						Spec:              db,
						InfraOutputStepId: infraReadyStepID,
					},
				},
			},
			// depends on infraReady even though upsert also depends on it, so
			// that the output of the infraStep is still injected
			DependsOn: []string{infraReadyStepID, upsertStep.Id},
		}

		cleanupStep := &deployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   fmt.Sprintf("PgCleanup-%s", db.DbName),
			Status: deployer_pb.StepStatus_UNSPECIFIED,
			Request: &deployer_pb.StepRequestType{
				Type: &deployer_pb.StepRequestType_PgCleanup{
					PgCleanup: &deployer_pb.StepRequestType_PGCleanup{
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

func stepNext(ctx context.Context, tb deployer_pb.DeploymentPSMTransitionBaton, deployment *deployer_pb.DeploymentState) error {
	stepMap := make(map[string]*deployer_pb.DeploymentStep)
	for _, step := range deployment.Steps {
		stepMap[step.Id] = step
	}

	anyOpen := false
	anyRunning := false
	anyFailed := false

	waitingSteps := make([]*deployer_pb.DeploymentStep, 0)

	for _, step := range deployment.Steps {
		switch step.Status {
		case deployer_pb.StepStatus_UNSPECIFIED, deployer_pb.StepStatus_BLOCKED:
			waitingSteps = append(waitingSteps, step)
		case deployer_pb.StepStatus_ACTIVE:
			anyRunning = true
		case deployer_pb.StepStatus_DONE:
		case deployer_pb.StepStatus_FAILED:
			anyFailed = true
		}
	}

	if anyFailed {
		if anyRunning {
			// Wait for cleanup
			return nil
		}

		// Nothing still running, we don't need to trigger any further
		// tasks.
		tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_Error{
			Error: "deployment steps failed",
		}))
		return nil
	}

	// allows steps to pull outputs from previous steps.

steps:
	for _, step := range waitingSteps {

		depMap := make(map[string]*deployer_pb.DeploymentStep)
		for _, dep := range step.DependsOn {
			depStep, ok := stepMap[dep]
			if !ok {
				return fmt.Errorf("step %s depends on %s, but it does not exist", step.Id, dep)
			}
			if depStep.Status != deployer_pb.StepStatus_DONE {
				anyOpen = true
				step.Status = deployer_pb.StepStatus_BLOCKED
				continue steps
			}
			depMap[dep] = depStep
		}

		sideEffect, err := stepToSideEffect(step, deployment, depMap)
		if err != nil {
			return err
		}
		anyRunning = true
		tb.SideEffect(sideEffect)
		step.Status = deployer_pb.StepStatus_ACTIVE
	}

	if anyRunning {
		return nil
	}

	if anyOpen {
		tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_Error{
			Error: "deployment steps deadlock",
		}))
		return nil
	}

	// All steps are done
	tb.ChainEvent(chainDeploymentEvent(tb, &deployer_pb.DeploymentEventType_Done{}))

	return nil
}
func stepToSideEffect(step *deployer_pb.DeploymentStep, deployment *deployer_pb.DeploymentState, dependencies map[string]*deployer_pb.DeploymentStep) (outbox.OutboxMessage, error) {
	requestMetadata, err := buildRequestMetadata(&deployer_pb.StepContext{
		StepId:       &step.Id,
		Phase:        deployer_pb.StepPhase_STEPS,
		DeploymentId: deployment.DeploymentId,
	})
	if err != nil {
		return nil, err
	}
	switch st := step.Request.Type.(type) {
	case *deployer_pb.StepRequestType_CfScale:
		return &deployer_tpb.ScaleStackMessage{
			Request:      requestMetadata,
			StackName:    st.CfScale.StackName,
			DesiredCount: st.CfScale.DesiredCount,
		}, nil

	case *deployer_pb.StepRequestType_CfUpdate:
		return &deployer_tpb.UpdateStackMessage{
			Request: requestMetadata,
			Spec:    st.CfUpdate.Spec,
		}, nil

	case *deployer_pb.StepRequestType_CfCreate:
		return &deployer_tpb.CreateNewStackMessage{
			Request: requestMetadata,
			Spec:    st.CfCreate.Spec,
		}, nil

	case *deployer_pb.StepRequestType_PgUpsert:
		src := st.PgUpsert.Spec
		spec := &deployer_pb.PostgresCreationSpec{
			DbName:            src.DbName,
			RootSecretName:    src.RootSecretName,
			DbExtensions:      src.DbExtensions,
			RotateCredentials: deployment.Spec.Flags.RotateCredentials,
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

		return &deployer_tpb.UpsertPostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			Spec:        spec,
		}, nil

	case *deployer_pb.StepRequestType_PgMigrate:
		src := st.PgMigrate.Spec
		spec := &deployer_pb.PostgresMigrationSpec{
			EcsClusterName: deployment.Spec.EcsCluster,
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

		return &deployer_tpb.MigratePostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			Spec:        spec,
		}, nil

	case *deployer_pb.StepRequestType_PgCleanup:
		src := st.PgCleanup.Spec
		spec := &deployer_pb.PostgresCleanupSpec{
			DbName:         src.DbName,
			RootSecretName: src.RootSecretName,
		}
		return &deployer_tpb.CleanupPostgresDatabaseMessage{
			Request:     requestMetadata,
			MigrationId: step.Id,
			Spec:        spec,
		}, nil

	default:
		return nil, fmt.Errorf("unknown step type: %T", step.Request.Type)
	}
}

func getStackOutputs(dependencies map[string]*deployer_pb.DeploymentStep, id string) ([]*deployer_pb.KeyValue, error) {

	infraOutputStep, ok := dependencies[id]
	if !ok {
		return nil, fmt.Errorf("PG Migrate step depends on %s for CF output, not passed in", id)
	}
	stackOutput, ok := infraOutputStep.Output.Type.(*deployer_pb.StepOutputType_CfStatus)
	if !ok {
		return nil, fmt.Errorf("unexpected step type for CF Output: %T", infraOutputStep.Output)
	}

	return stackOutput.CfStatus.Output.Outputs, nil
}
