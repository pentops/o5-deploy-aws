package service

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/apps/service/internal/states"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc"
)

type App struct {
	DeployerWorker *DeployerWorker
	CommandService *CommandService
	QueryService   *QueryService
}

type AppDeps struct {
	SpecBuilder  *deployer.SpecBuilder
	GithubClient GithubClient
	DB           sqrlx.Transactor
}

func NewApp(deps AppDeps) (*App, error) {

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		return nil, err
	}

	deploymentWorker, err := NewDeployerWorker(deps.DB, deps.SpecBuilder, stateMachines)
	if err != nil {
		return nil, err
	}

	commandService, err := NewCommandService(deps.DB, deps.GithubClient, stateMachines)
	if err != nil {
		return nil, err
	}

	queryService, err := NewQueryService(deps.DB, stateMachines)
	if err != nil {
		return nil, err
	}

	return &App{
		DeployerWorker: deploymentWorker,
		CommandService: commandService,
		QueryService:   queryService,
	}, nil

}

func (app *App) RegisterGRPC(server *grpc.Server) {
	app.QueryService.RegisterGRPC(server)

	awsdeployer_spb.RegisterDeploymentCommandServiceServer(server, app.CommandService)
	awsdeployer_tpb.RegisterDeploymentRequestTopicServer(server, app.DeployerWorker)
	awsinfra_tpb.RegisterCloudFormationReplyTopicServer(server, app.DeployerWorker)
	awsinfra_tpb.RegisterPostgresReplyTopicServer(server, app.DeployerWorker)
}
