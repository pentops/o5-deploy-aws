package localrun

import (
	"context"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type Spec struct {
	Version       string
	AppConfig     *application_pb.Application
	EnvConfig     *environment_pb.Environment
	RotateSecrets bool
	CancelUpdates bool
	QuickMode     bool
	ScratchBucket string
}

func RunLocalDeploy(ctx context.Context, templateStore deployer.TemplateStore, infraAdapter IInfra, spec Spec) error {

	deploymentManager, err := deployer.NewSpecBuilder(templateStore)
	if err != nil {
		return err
	}
	deploymentManager.RotateSecrets = spec.RotateSecrets
	deploymentManager.CancelUpdates = spec.CancelUpdates
	deploymentManager.QuickMode = spec.QuickMode

	stateStore := NewStateStore()
	eventLoop := NewEventLoop(infraAdapter, stateStore, deploymentManager)

	var environmentID = uuid.NewString()

	trigger := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId:  uuid.NewString(),
		Application:   spec.AppConfig,
		Version:       spec.Version,
		EnvironmentId: environmentID,
	}

	return eventLoop.Run(ctx, trigger, spec.EnvConfig)

}
