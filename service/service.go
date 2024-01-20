package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.daemonl.com/envconf"
)

func OpenDatabase(ctx context.Context) (*sql.DB, error) {
	var config = struct {
		PostgresURL string `env:"POSTGRES_URL"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, err
	}

	for {
		if err := db.Ping(); err != nil {
			log.WithError(ctx, err).Error("pinging PG")
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return db, nil
}

type DeployerService struct {
	DeploymentQuery *deployer_spb.DeploymentPSMQuerySet
	StackQuery      *deployer_spb.StackPSMQuerySet
	github          github.IClient

	db *sqrlx.Wrapper
	*deployer_spb.UnimplementedDeploymentQueryServiceServer
	*deployer_spb.UnimplementedDeploymentCommandServiceServer
}

func NewDeployerService(conn sqrlx.Connection, github github.IClient) (*DeployerService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	deploymentQuery, err := deployer_spb.NewDeploymentPSMQuerySet(
		deployer_spb.DefaultDeploymentPSMQuerySpec(deployer.DeploymentTableSpec().StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build deployment query: %w", err)
	}

	stackQuery, err := deployer_spb.NewStackPSMQuerySet(
		deployer_spb.DefaultStackPSMQuerySpec(deployer.StackTableSpec().StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build stack query: %w", err)
	}

	return &DeployerService{
		db:              db,
		github:          github,
		DeploymentQuery: deploymentQuery,
		StackQuery:      stackQuery,
	}, nil
}

func (ds *DeployerService) TriggerDeployment(ctx context.Context, req *deployer_spb.TriggerDeploymentRequest) (*deployer_spb.TriggerDeploymentResponse, error) {

	gh := req.GetGithub()
	if gh == nil {
		return nil, status.Error(codes.Unimplemented, "only github source is supported")
	}

	apps, err := ds.github.PullO5Configs(ctx, gh.Owner, gh.Repo, gh.Commit)
	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found in push event")
	}

	if len(apps) > 1 {
		return nil, fmt.Errorf("multiple applications found in push event, not yet supported")
	}

	requestMessage := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId:    req.DeploymentId,
		Application:     apps[0],
		Version:         gh.Commit,
		EnvironmentName: req.EnvironmentName,
	}
	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return outbox.Send(ctx, tx, requestMessage)
	}); err != nil {
		return nil, err
	}

	return &deployer_spb.TriggerDeploymentResponse{}, nil
}

func (ds *DeployerService) GetDeployment(ctx context.Context, req *deployer_spb.GetDeploymentRequest) (*deployer_spb.GetDeploymentResponse, error) {
	res := &deployer_spb.GetDeploymentResponse{}
	return res, ds.DeploymentQuery.Get(ctx, ds.db, req, res)
}

func (ds *DeployerService) ListDeployments(ctx context.Context, req *deployer_spb.ListDeploymentsRequest) (*deployer_spb.ListDeploymentsResponse, error) {
	res := &deployer_spb.ListDeploymentsResponse{}
	return res, ds.DeploymentQuery.List(ctx, ds.db, req, res)
}

func (ds *DeployerService) ListDeploymentEvents(ctx context.Context, req *deployer_spb.ListDeploymentEventsRequest) (*deployer_spb.ListDeploymentEventsResponse, error) {
	res := &deployer_spb.ListDeploymentEventsResponse{}
	return res, ds.DeploymentQuery.ListEvents(ctx, ds.db, req, res)
}

func (ds *DeployerService) GetStack(ctx context.Context, req *deployer_spb.GetStackRequest) (*deployer_spb.GetStackResponse, error) {
	res := &deployer_spb.GetStackResponse{}
	return res, ds.StackQuery.Get(ctx, ds.db, req, res)
}

func (ds *DeployerService) ListStacks(ctx context.Context, req *deployer_spb.ListStacksRequest) (*deployer_spb.ListStacksResponse, error) {
	res := &deployer_spb.ListStacksResponse{}
	return res, ds.StackQuery.List(ctx, ds.db, req, res)
}

func (ds *DeployerService) ListStackEvents(ctx context.Context, req *deployer_spb.ListStackEventsRequest) (*deployer_spb.ListStackEventsResponse, error) {
	res := &deployer_spb.ListStackEventsResponse{}
	return res, ds.StackQuery.ListEvents(ctx, ds.db, req, res)
}
