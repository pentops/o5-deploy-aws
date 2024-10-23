package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/internal/apps/service/internal/states"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc"
)

type QueryService struct {
	deploymentQuery *awsdeployer_spb.DeploymentPSMQuerySet
	*awsdeployer_spb.UnimplementedDeploymentQueryServiceServer

	stackQuery *awsdeployer_spb.StackPSMQuerySet
	*awsdeployer_spb.UnimplementedStackQueryServiceServer

	environmentQuery *awsdeployer_spb.EnvironmentPSMQuerySet
	*awsdeployer_spb.UnimplementedEnvironmentQueryServiceServer

	clusterQuery *awsdeployer_spb.ClusterPSMQuerySet
	*awsdeployer_spb.UnimplementedClusterQueryServiceServer

	lookup *LookupProvider

	db sqrlx.Transactor
}

func NewQueryService(db sqrlx.Transactor, stateMachines *states.StateMachines) (*QueryService, error) {

	lookup, err := NewLookupProvider(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create lookup provider: %w", err)
	}

	deploymentQuery, err := awsdeployer_spb.NewDeploymentPSMQuerySet(
		awsdeployer_spb.DefaultDeploymentPSMQuerySpec(stateMachines.Deployment.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build deployment query: %w", err)
	}

	stackQuery, err := awsdeployer_spb.NewStackPSMQuerySet(
		awsdeployer_spb.DefaultStackPSMQuerySpec(stateMachines.Stack.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build stack query: %w", err)
	}

	environmentQuery, err := awsdeployer_spb.NewEnvironmentPSMQuerySet(
		awsdeployer_spb.DefaultEnvironmentPSMQuerySpec(stateMachines.Environment.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build environment query: %w", err)
	}

	clusterQuery, err := awsdeployer_spb.NewClusterPSMQuerySet(
		awsdeployer_spb.DefaultClusterPSMQuerySpec(stateMachines.Cluster.StateTableSpec()),
		psm.StateQueryOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build cluster query: %w", err)
	}

	return &QueryService{
		db:     db,
		lookup: lookup,

		deploymentQuery:  deploymentQuery,
		stackQuery:       stackQuery,
		environmentQuery: environmentQuery,
		clusterQuery:     clusterQuery,
	}, nil
}

func (ds *QueryService) RegisterGRPC(s *grpc.Server) {
	awsdeployer_spb.RegisterDeploymentQueryServiceServer(s, ds)
	awsdeployer_spb.RegisterStackQueryServiceServer(s, ds)
	awsdeployer_spb.RegisterEnvironmentQueryServiceServer(s, ds)
	awsdeployer_spb.RegisterClusterQueryServiceServer(s, ds)
}

func (ds *QueryService) GetDeployment(ctx context.Context, req *awsdeployer_spb.GetDeploymentRequest) (*awsdeployer_spb.GetDeploymentResponse, error) {
	res := &awsdeployer_spb.GetDeploymentResponse{}
	return res, ds.deploymentQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListDeployments(ctx context.Context, req *awsdeployer_spb.ListDeploymentsRequest) (*awsdeployer_spb.ListDeploymentsResponse, error) {
	res := &awsdeployer_spb.ListDeploymentsResponse{}
	return res, ds.deploymentQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListDeploymentEvents(ctx context.Context, req *awsdeployer_spb.ListDeploymentEventsRequest) (*awsdeployer_spb.ListDeploymentEventsResponse, error) {
	res := &awsdeployer_spb.ListDeploymentEventsResponse{}
	return res, ds.deploymentQuery.ListEvents(ctx, ds.db, req, res)
}

func (ds *QueryService) GetStack(ctx context.Context, req *awsdeployer_spb.GetStackRequest) (*awsdeployer_spb.GetStackResponse, error) {
	res := &awsdeployer_spb.GetStackResponse{}
	return res, ds.stackQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListStacks(ctx context.Context, req *awsdeployer_spb.ListStacksRequest) (*awsdeployer_spb.ListStacksResponse, error) {
	res := &awsdeployer_spb.ListStacksResponse{}
	return res, ds.stackQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListStackEvents(ctx context.Context, req *awsdeployer_spb.ListStackEventsRequest) (*awsdeployer_spb.ListStackEventsResponse, error) {
	res := &awsdeployer_spb.ListStackEventsResponse{}
	return res, ds.stackQuery.ListEvents(ctx, ds.db, req, res)
}

func (ds *QueryService) GetEnvironment(ctx context.Context, req *awsdeployer_spb.GetEnvironmentRequest) (*awsdeployer_spb.GetEnvironmentResponse, error) {
	res := &awsdeployer_spb.GetEnvironmentResponse{}
	return res, ds.environmentQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListEnvironments(ctx context.Context, req *awsdeployer_spb.ListEnvironmentsRequest) (*awsdeployer_spb.ListEnvironmentsResponse, error) {
	res := &awsdeployer_spb.ListEnvironmentsResponse{}
	return res, ds.environmentQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListEnvironmentEvents(ctx context.Context, req *awsdeployer_spb.ListEnvironmentEventsRequest) (*awsdeployer_spb.ListEnvironmentEventsResponse, error) {
	res := &awsdeployer_spb.ListEnvironmentEventsResponse{}
	return res, ds.environmentQuery.ListEvents(ctx, ds.db, req, res)
}

func (ds *QueryService) GetCluster(ctx context.Context, req *awsdeployer_spb.GetClusterRequest) (*awsdeployer_spb.GetClusterResponse, error) {
	if _, err := uuid.Parse(req.ClusterId); err != nil {
		cluster, err := ds.lookup.lookupCluster(ctx, req.ClusterId)
		if err != nil {
			return nil, err
		}
		req.ClusterId = cluster.clusterID
	}
	res := &awsdeployer_spb.GetClusterResponse{}
	return res, ds.clusterQuery.Get(ctx, ds.db, req, res)
}

func (ds *QueryService) ListClusters(ctx context.Context, req *awsdeployer_spb.ListClustersRequest) (*awsdeployer_spb.ListClustersResponse, error) {
	res := &awsdeployer_spb.ListClustersResponse{}
	return res, ds.clusterQuery.List(ctx, ds.db, req, res)
}

func (ds *QueryService) ListClusterEvents(ctx context.Context, req *awsdeployer_spb.ListClusterEventsRequest) (*awsdeployer_spb.ListClusterEventsResponse, error) {
	if _, err := uuid.Parse(req.ClusterId); err != nil {
		cluster, err := ds.lookup.lookupCluster(ctx, req.ClusterId)
		if err != nil {
			return nil, err
		}
		req.ClusterId = cluster.clusterID
	}
	res := &awsdeployer_spb.ListClusterEventsResponse{}
	return res, ds.clusterQuery.ListEvents(ctx, ds.db, req, res)
}
