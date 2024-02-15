package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
)

type planInput struct {
	stackExists bool
	quickMode   bool
}

func planDeploymentSteps(ctx context.Context, deployment *deployer_pb.DeploymentState, input planInput) ([]*deployer_pb.DeploymentStep, error) {

	plan := make([]*deployer_pb.DeploymentStep, 0)

	if input.quickMode {
		if input.stackExists {
			infraMigrate := &deployer_pb.DeploymentStep{
				Id:     uuid.NewString(),
				Name:   "CFUpdate",
				Status: deployer_pb.StepStatus_UNSPECIFIED,
				Request: &deployer_pb.StepRequestType{
					Type: &deployer_pb.StepRequestType_CfUpdate{
						CfUpdate: &deployer_pb.StepRequestType_CFUpdate{
							Spec: &deployer_pb.CFStackInput{
								StackName:    deployment.StackName,
								DesiredCount: 0,
								TemplateUrl:  deployment.Spec.TemplateUrl,
								Parameters:   deployment.Spec.Parameters,
							},
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
							Spec: &deployer_pb.CFStackInput{
								StackName:    deployment.StackName,
								DesiredCount: 0,
								TemplateUrl:  deployment.Spec.TemplateUrl,
								Parameters:   deployment.Spec.Parameters,
							},
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
	if input.stackExists {
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
						Spec: &deployer_pb.CFStackInput{
							StackName:    deployment.StackName,
							DesiredCount: 0,
							TemplateUrl:  deployment.Spec.TemplateUrl,
							Parameters:   deployment.Spec.Parameters,
						},
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
						Spec: &deployer_pb.CFStackInput{
							StackName:    deployment.StackName,
							DesiredCount: 0,
							TemplateUrl:  deployment.Spec.TemplateUrl,
							Parameters:   deployment.Spec.Parameters,
						},
					},
				},
			},
		}
		plan = append(plan, infraCreate)
		infraReadyStepID = infraCreate.Id
	}

	scaleUp := &deployer_pb.DeploymentStep{
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

	for _, db := range deployment.Spec.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database":    db.Database.Name,
			"serverGroup": db.Database.GetPostgres().ServerGroup,
		})
		log.Debug(ctx, "Upsert Database")

		secretName := db.RdsHost.SecretName
		if secretName == "" {
			return nil, fmt.Errorf("no host found for server group %q", db.Database.GetPostgres().ServerGroup)
		}

		migrateReq := &deployer_pb.StepRequestType_PGMigrate{
			Spec: &deployer_pb.PostgresMigrationSpec{
				//MigrationTaskArn:  migrationTaskARN,
				//SecretArn:         secretARN,
				Database:          db,
				RotateCredentials: deployment.Spec.RotateCredentials,
				EcsClusterName:    deployment.Spec.EcsCluster,
				RootSecretName:    secretName,
			},
		}

		if db.MigrationTaskOutputName != nil {
			// pull outputs from InfraMigrate for secret and migration task
			// ARNs
			migrateReq.InfraOutputStepId = &infraReadyStepID
		}

		pgMigrate := &deployer_pb.DeploymentStep{
			Id:     uuid.NewString(),
			Name:   "PgMigrate",
			Status: deployer_pb.StepStatus_UNSPECIFIED,
			Request: &deployer_pb.StepRequestType{
				Type: &deployer_pb.StepRequestType_PgMigrate{
					PgMigrate: migrateReq,
				},
			},
			DependsOn: []string{infraReadyStepID},
		}

		plan = append(plan, pgMigrate)
		scaleUp.DependsOn = append(scaleUp.DependsOn, pgMigrate.Id)
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

	startSteps := make([]outbox.OutboxMessage, 0)

steps:
	for _, step := range deployment.Steps {
		switch step.Status {
		case deployer_pb.StepStatus_UNSPECIFIED:
		case deployer_pb.StepStatus_ACTIVE:
			anyRunning = true
			continue steps
		case deployer_pb.StepStatus_DONE:
			continue steps
		case deployer_pb.StepStatus_FAILED:
			anyFailed = true
			continue steps
		}

		// allows steps to pull outputs from previous steps.
		depMap := make(map[string]*deployer_pb.DeploymentStep)
		for _, dep := range step.DependsOn {
			depStep, ok := stepMap[dep]
			if !ok {
				return fmt.Errorf("step %s depends on %s, but it does not exist", step.Id, dep)
			}
			if depStep.Status != deployer_pb.StepStatus_DONE {
				anyOpen = true
				continue steps
			}
			depMap[dep] = depStep
		}

		sideEffect, err := stepToSideEffect(step, deployment, depMap)
		if err != nil {
			return err
		}
		startSteps = append(startSteps, sideEffect)
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

	// Only start steps if no other step has failed
	for _, step := range startSteps {
		anyRunning = true
		tb.SideEffect(step)
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

	case *deployer_pb.StepRequestType_PgMigrate:
		spec := st.PgMigrate.Spec
		if st.PgMigrate.InfraOutputStepId != nil {
			infraOutputStep, ok := dependencies[*st.PgMigrate.InfraOutputStepId]
			if !ok {
				return nil, fmt.Errorf("PG Migrate step %s depends on %s for CF output, not passed in", step.Id, *st.PgMigrate.InfraOutputStepId)
			}
			stackOutput, ok := infraOutputStep.Output.Type.(*deployer_pb.StepOutputType_CfStatus)
			if !ok {
				return nil, fmt.Errorf("unexpected step type for CF Output: %T", infraOutputStep.Output)
			}

			db := st.PgMigrate.Spec.Database

			for _, output := range stackOutput.CfStatus.Output.Outputs {
				if *db.MigrationTaskOutputName == output.Name {
					spec.MigrationTaskArn = output.Value
				}
				if *db.SecretOutputName == output.Name {
					spec.SecretArn = output.Value
				}
			}
			if spec.MigrationTaskArn == "" {
				return nil, fmt.Errorf("stack output missing %s for database %s", *db.MigrationTaskOutputName, db.Database.Name)
			}
			if spec.SecretArn == "" {
				return nil, fmt.Errorf("stack output missing %s for database %s", *db.SecretOutputName, db.Database.Name)
			}
		}
		return &deployer_tpb.MigratePostgresDatabaseMessage{
			Request:       requestMetadata,
			MigrationSpec: spec,
		}, nil

	default:
		return nil, fmt.Errorf("unknown step type: %T", step.Request.Type)
	}
}
