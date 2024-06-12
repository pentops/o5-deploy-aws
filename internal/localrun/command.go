package localrun

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
)

type Spec struct {
	Version       string
	AppConfig     *application_pb.Application
	ClusterConfig *environment_pb.Cluster
	EnvConfig     *environment_pb.Environment
	ScratchBucket string
	Flags         *awsdeployer_pb.DeploymentFlags
	ConfirmPlan   bool
}

func RunLocalDeploy(ctx context.Context, templateStore deployer.TemplateStore, infraAdapter IInfra, spec Spec) error {

	deploymentManager, err := deployer.NewSpecBuilder(templateStore)
	if err != nil {
		return err
	}
	stateStore := NewStateStore()
	eventLoop := NewEventLoop(infraAdapter, stateStore, deploymentManager)

	eventLoop.confirmPlan = spec.ConfirmPlan
	var environmentID = uuid.NewString()

	trigger := &awsdeployer_tpb.RequestDeploymentMessage{
		DeploymentId:  uuid.NewString(),
		Application:   spec.AppConfig,
		Version:       spec.Version,
		EnvironmentId: environmentID,
		Flags:         spec.Flags,
	}

	if spec.EnvConfig == nil {
		return fmt.Errorf("Environment config is required")
	}
	if spec.ClusterConfig == nil {
		return fmt.Errorf("Cluster config is required")
	}

	err = eventLoop.Run(ctx, trigger, spec.ClusterConfig, spec.EnvConfig)
	if err != nil {
		return fmt.Errorf("Error *running* event loop (errors are not expected): %w", err)
	}

	endState, err := stateStore.GetDeployment(ctx, trigger.DeploymentId)
	if err != nil {
		return fmt.Errorf("Error getting deployment state: %w", err)
	}

	log.WithField(ctx, "End Status", endState.Status.ShortString()).Info("Deployment completed")

	if endState.Status != awsdeployer_pb.DeploymentStatus_DONE {
		return fmt.Errorf("Deployment did not complete successfully: %s", endState.Status)
	}

	return err
}
