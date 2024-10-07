package planbuild

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
)

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
)

func QualifiedStep(parts ...string) string {
	return strings.Join(parts, ".")
}

type DeploymentPlan struct {
	Steps       []*awsdeployer_pb.DeploymentStep
	StackStatus *awsdeployer_pb.CFStackOutput
	Deployment  *awsdeployer_pb.DeploymentSpec
}

type PlanStep struct {
	step *awsdeployer_pb.DeploymentStep
}

func (ps *PlanStep) DependsOn(depStep ...*PlanStep) *PlanStep {
	for _, step := range depStep {
		ps.step.DependsOn = append(ps.step.DependsOn, step.step.Id)
	}
	return ps
}

func (ps *PlanStep) Then(next *PlanStep) *PlanStep {
	next.DependsOn(ps)
	return next
}

func (db *DeploymentPlan) addStep(step *awsdeployer_pb.DeploymentStep) *PlanStep {
	db.Steps = append(db.Steps, step)
	return &PlanStep{
		step: step,
	}
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
	}

	return plan.addStep(scaleDown)
}

// NopDiscovery add a fake discovery step which is already completed.
func (plan *DeploymentPlan) NOPDiscovery() *PlanStep {
	discoveryStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   StepNOPDiscovery,
		Status: awsdeployer_pb.StepStatus_DONE,
		Output: &awsdeployer_pb.StepOutputType{
			Type: &awsdeployer_pb.StepOutputType_CfStatus{
				CfStatus: &awsdeployer_pb.StepOutputType_CFStatus{
					Output: plan.StackStatus,
				},
			},
		},
	}
	return plan.addStep(discoveryStep)
}

func (plan *DeploymentPlan) CFUpdate(scale int32) *PlanStep {
	infraMigrate := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   StepCFUpdate,
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_CfUpdate{
				CfUpdate: &awsdeployer_pb.StepRequestType_CFUpdate{
					Spec: plan.stackInput(scale),
				},
			},
		},
	}
	return plan.addStep(infraMigrate)
}

func (plan *DeploymentPlan) CFCreateEmpty() *PlanStep {
	spec := plan.stackInput(0)
	spec.Template = &awsdeployer_pb.CFStackInput_EmptyStack{
		EmptyStack: true,
	}
	spec.Parameters = nil

	createEmptyStack := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   StepCFCreateEmpty,
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
	return plan.addStep(createEmptyStack)
}

func (plan *DeploymentPlan) ImportResources(ctx context.Context) *PlanStep {

	createChangeset := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   StepCFImportPlan,
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
	return plan.addStep(createChangeset)

}

func (plan *DeploymentPlan) MigrateDatabases(ctx context.Context, infraReadyStep *PlanStep) []*PlanStep {
	finalSteps := make([]*PlanStep, 0)
	for _, db := range plan.Deployment.Databases {

		ctx = log.WithFields(ctx, map[string]interface{}{
			"database": db.DbName,
		})
		log.Debug(ctx, "Upsert Database")

		lastStep := plan.UpsertDatabase(db, infraReadyStep)

		finalSteps = append(finalSteps, lastStep)
	}
	return finalSteps

}

func (plan *DeploymentPlan) UpsertDatabase(db *awsdeployer_pb.PostgresSpec, infraReadyStep *PlanStep) *PlanStep {

	upsertStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   QualifiedStep(StepPgUpsert, db.DbName),
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_PgUpsert{
				PgUpsert: &awsdeployer_pb.StepRequestType_PGUpsert{
					Spec:              db,
					InfraOutputStepId: infraReadyStep.step.Id,
					RotateCredentials: plan.Deployment.Flags.RotateCredentials,
				},
			},
		},
		DependsOn: []string{infraReadyStep.step.Id},
	}
	plan.addStep(upsertStep)

	migrateStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   QualifiedStep(StepPgMigrate, db.DbName),
		Status: awsdeployer_pb.StepStatus_UNSPECIFIED,
		Request: &awsdeployer_pb.StepRequestType{
			Type: &awsdeployer_pb.StepRequestType_PgMigrate{
				PgMigrate: &awsdeployer_pb.StepRequestType_PGMigrate{
					Spec:              db,
					InfraOutputStepId: infraReadyStep.step.Id,
					EcsClusterName:    plan.Deployment.EcsCluster,
				},
			},
		},
		// depends on infraReady even though upsert also depends on it, so
		// that the output of the infraStep is still injected
		DependsOn: []string{infraReadyStep.step.Id, upsertStep.Id},
	}
	plan.addStep(migrateStep)

	cleanupStep := &awsdeployer_pb.DeploymentStep{
		Id:     uuid.NewString(),
		Name:   QualifiedStep(StepPgCleanup, db.DbName),
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
	return plan.addStep(cleanupStep)
}
