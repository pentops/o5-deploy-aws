package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/protostate/psm"
	"gopkg.daemonl.com/envconf"
	"gopkg.daemonl.com/sqrlx"
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
	DeploymentQuery *deployer_spb.DeploymentPSMStateQuerySet
	StackQuery      *deployer_spb.StackPSMStateQuerySet

	db *sqrlx.Wrapper
	*deployer_spb.UnimplementedDeploymentQueryServiceServer
}

func NewDeployerService(conn sqrlx.Connection) (*DeployerService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	deploymentQuery, err := psm.BuildStateQuerySet(
		deployer.DeploymentTableSpec().QuerySpec(),
		deployer_spb.DeploymentPSMStateQuerySpec{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build deployment query: %w", err)
	}

	stackQuery, err := psm.BuildStateQuerySet(
		deployer.StackTableSpec().QuerySpec(),
		deployer_spb.StackPSMStateQuerySpec{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build stack query: %w", err)
	}

	/*

		deploymentQuery, err := pquery.NewStateQuery(pquery.StateQuerySpec{
			TableName:              "deployment",
			DataColumn:             "state",
			PrimaryKeyColumn:       "id",
			PrimaryKeyRequestField: protoreflect.Name("deployment_id"),
			Events: &pquery.GetJoinSpec{
				TableName:        "deployment_event",
				DataColumn:       "event",
				FieldInParent:    "events",
				ForeignKeyColumn: "deployment_id",
			},

			Get: &pquery.MethodDescriptor{
				Request:  (&deployer_spb.GetDeploymentRequest{}).ProtoReflect().Descriptor(),
				Response: (&deployer_spb.GetDeploymentResponse{}).ProtoReflect().Descriptor(),
			},

			List: &pquery.MethodDescriptor{
				Request:  (&deployer_spb.ListDeploymentsRequest{}).ProtoReflect().Descriptor(),
				Response: (&deployer_spb.ListDeploymentsResponse{}).ProtoReflect().Descriptor(),
			},

			ListEvents: &pquery.MethodDescriptor{
				Request:  (&deployer_spb.ListDeploymentEventsRequest{}).ProtoReflect().Descriptor(),
				Response: (&deployer_spb.ListDeploymentEventsResponse{}).ProtoReflect().Descriptor(),
			},
		})
		if err != nil {
			return nil, err
		}

		stackQuery, err := pquery.NewStateQuery(pquery.StateQuerySpec{
			TableName:              "stack",
			DataColumn:             "state",
			PrimaryKeyColumn:       "id",
			PrimaryKeyRequestField: protoreflect.Name("stack_id"),
			Events: &pquery.GetJoinSpec{
				TableName:        "stack_event",
				DataColumn:       "event",
				FieldInParent:    "events",
				ForeignKeyColumn: "stack_id",
			},

			Get: &pquery.MethodDescriptor{
				Request:  (&deployer_spb.GetStackRequest{}).ProtoReflect().Descriptor(),
				Response: (&deployer_spb.GetStackResponse{}).ProtoReflect().Descriptor(),
			},

			List: &pquery.MethodDescriptor{
				Request:  (&deployer_spb.ListStacksRequest{}).ProtoReflect().Descriptor(),
				Response: (&deployer_spb.ListStacksResponse{}).ProtoReflect().Descriptor(),
			},

			ListEvents: &pquery.MethodDescriptor{
				Request:  (&deployer_spb.ListStackEventsRequest{}).ProtoReflect().Descriptor(),
				Response: (&deployer_spb.ListStackEventsResponse{}).ProtoReflect().Descriptor(),
			},
		})
		if err != nil {
			return nil, err
		}
	*/

	return &DeployerService{
		db:              db,
		DeploymentQuery: deploymentQuery,
		StackQuery:      stackQuery,
	}, nil
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
