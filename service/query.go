package service

import (
	"context"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_spb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type QueryService struct {
	deploymentQuery  *deployer_spb.DeploymentPSMQuerySet
	stackQuery       *deployer_spb.StackPSMQuerySet
	environmentQuery *deployer_spb.EnvironmentPSMQuerySet

	db *sqrlx.Wrapper
	*deployer_spb.UnimplementedDeploymentQueryServiceServer
}

func NewQueryService(conn sqrlx.Connection, stateMachines *states.StateMachines) (*QueryService, error) {

	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	deploymentQuery, err := deployer_spb.NewDeploymentPSMQuerySet(
		deployer_spb.DefaultDeploymentPSMQuerySpec(stateMachines.Deployment.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build deployment query: %w", err)
	}

	stackQuery, err := deployer_spb.NewStackPSMQuerySet(
		deployer_spb.DefaultStackPSMQuerySpec(stateMachines.Stack.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build stack query: %w", err)
	}

	environmentQuery, err := deployer_spb.NewEnvironmentPSMQuerySet(
		deployer_spb.DefaultEnvironmentPSMQuerySpec(stateMachines.Environment.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment query: %w", err)
	}

	return &QueryService{
		db: db,

		deploymentQuery:  deploymentQuery,
		stackQuery:       stackQuery,
		environmentQuery: environmentQuery,
	}, nil
}

func (ds *QueryService) GetDeployment(ctx context.Context, req *deployer_spb.GetDeploymentRequest) (*deployer_spb.GetDeploymentResponse, error) {
	res := &deployer_spb.GetDeploymentResponse{}
	return res, ds.deploymentQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListDeployments(ctx context.Context, req *deployer_spb.ListDeploymentsRequest) (*deployer_spb.ListDeploymentsResponse, error) {
	res := &deployer_spb.ListDeploymentsResponse{}
	return res, ds.deploymentQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListDeploymentEvents(ctx context.Context, req *deployer_spb.ListDeploymentEventsRequest) (*deployer_spb.ListDeploymentEventsResponse, error) {
	res := &deployer_spb.ListDeploymentEventsResponse{}
	return res, ds.deploymentQuery.ListEvents(ctx, ds.db, req, res)
}

func (ds *QueryService) GetStack(ctx context.Context, req *deployer_spb.GetStackRequest) (*deployer_spb.GetStackResponse, error) {
	res := &deployer_spb.GetStackResponse{}
	return res, ds.stackQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListStacks(ctx context.Context, req *deployer_spb.ListStacksRequest) (*deployer_spb.ListStacksResponse, error) {
	res := &deployer_spb.ListStacksResponse{}
	return res, ds.stackQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListStackEvents(ctx context.Context, req *deployer_spb.ListStackEventsRequest) (*deployer_spb.ListStackEventsResponse, error) {
	res := &deployer_spb.ListStackEventsResponse{}
	return res, ds.stackQuery.ListEvents(ctx, ds.db, req, res)
}

func (ds *QueryService) GetEnvironment(ctx context.Context, req *deployer_spb.GetEnvironmentRequest) (*deployer_spb.GetEnvironmentResponse, error) {
	res := &deployer_spb.GetEnvironmentResponse{}
	return res, ds.environmentQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListEnvironments(ctx context.Context, req *deployer_spb.ListEnvironmentsRequest) (*deployer_spb.ListEnvironmentsResponse, error) {
	res := &deployer_spb.ListEnvironmentsResponse{}
	return res, ds.environmentQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListEnvironmentEvents(ctx context.Context, req *deployer_spb.ListEnvironmentEventsRequest) (*deployer_spb.ListEnvironmentEventsResponse, error) {
	res := &deployer_spb.ListEnvironmentEventsResponse{}
	return res, ds.environmentQuery.ListEvents(ctx, ds.db, req, res)
}
