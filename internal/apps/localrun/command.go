package localrun

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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
	var environmentID = uuid.NewString()

	trigger := &awsdeployer_tpb.RequestDeploymentMessage{
		DeploymentId:  uuid.NewString(),
		Application:   spec.AppConfig,
		Version:       spec.Version,
		EnvironmentId: environmentID,
		Flags:         spec.Flags,
	}

	if spec.EnvConfig == nil {
		return fmt.Errorf("environment config is required")
	}
	if spec.ClusterConfig == nil {
		return fmt.Errorf("cluster config is required")
	}

	rr := &Runner{
		awsRunner:   infraAdapter,
		specBuilder: deploymentManager,
		confirmPlan: spec.ConfirmPlan,
	}

	deploymentSpec, err := deploymentManager.BuildSpec(ctx, trigger, spec.ClusterConfig, spec.EnvConfig)
	if err != nil {
		return err
	}

	return rr.RunDeployment(ctx, &awsdeployer_pb.DeploymentStateData{
		Spec: deploymentSpec,
	})
}
