package aws

import (
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/aws_cf"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/aws_ecs"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/aws_postgres"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsapi"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsraw"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/tokenstore"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_tpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc"
)

type App struct {
	CFRunner      *aws_cf.InfraWorker
	MigrateRunner *aws_postgres.PostgresMigrateWorker
	ECSWorker     *aws_ecs.ECSWorker
	RawWorker     *awsraw.RawMessageWorker
}

func NewApp(db sqrlx.Transactor, awsClients *awsapi.DeployerClients) (*App, error) {

	infraStore, err := tokenstore.NewStorage(db)
	if err != nil {
		return nil, err
	}
	cfAdapter := aws_cf.NewCFAdapter(awsClients)

	awsInfraRunner := aws_cf.NewInfraWorker(infraStore, cfAdapter)

	ecsWorker, err := aws_ecs.NewECSWorker(infraStore, awsClients.ECS)
	if err != nil {
		return nil, err
	}

	rawWorker := awsraw.NewRawMessageWorker(awsInfraRunner, ecsWorker)

	dbMigrator := aws_postgres.NewDBMigrator(
		awsClients.SecretsManager,
		awsClients.RDSAuthProvider,
	)

	pgMigrateRunner := aws_postgres.NewPostgresMigrateWorker(infraStore, dbMigrator)

	return &App{
		CFRunner:      awsInfraRunner,
		MigrateRunner: pgMigrateRunner,
		RawWorker:     rawWorker,
		ECSWorker:     ecsWorker,
	}, nil

}

func (app *App) RegisterGRPC(grpcServer *grpc.Server) {

	messaging_tpb.RegisterRawMessageTopicServer(grpcServer, app.RawWorker)
	awsinfra_tpb.RegisterCloudFormationRequestTopicServer(grpcServer, app.CFRunner)
	awsinfra_tpb.RegisterPostgresRequestTopicServer(grpcServer, app.MigrateRunner)
	awsinfra_tpb.RegisterECSRequestTopicServer(grpcServer, app.ECSWorker)

}
