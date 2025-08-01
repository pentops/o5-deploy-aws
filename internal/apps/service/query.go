package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/j5/lib/psm"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/internal/apps/service/internal/states"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc"
)

type QueryService struct {
	deploymentQuery  *awsdeployer_spb.DeploymentQueryServiceImpl
	stackQuery       *awsdeployer_spb.StackQueryServiceImpl
	environmentQuery *awsdeployer_spb.EnvironmentQueryServiceImpl

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

		deploymentQuery:  awsdeployer_spb.NewDeploymentQueryServiceImpl(db, deploymentQuery),
		environmentQuery: awsdeployer_spb.NewEnvironmentQueryServiceImpl(db, environmentQuery),
		stackQuery:       awsdeployer_spb.NewStackQueryServiceImpl(db, stackQuery),

		clusterQuery: clusterQuery,
	}, nil
}

func (ds *QueryService) RegisterGRPC(s *grpc.Server) {
	awsdeployer_spb.RegisterDeploymentQueryServiceServer(s, ds.deploymentQuery)
	awsdeployer_spb.RegisterStackQueryServiceServer(s, ds.stackQuery)
	awsdeployer_spb.RegisterEnvironmentQueryServiceServer(s, ds.environmentQuery)

	awsdeployer_spb.RegisterClusterQueryServiceServer(s, ds)
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
	return res, ds.clusterQuery.Get(ctx, ds.db, req.J5Object(), res.J5Object())
}

func (ds *QueryService) ListClusters(ctx context.Context, req *awsdeployer_spb.ListClustersRequest) (*awsdeployer_spb.ListClustersResponse, error) {
	res := &awsdeployer_spb.ListClustersResponse{}
	return res, ds.clusterQuery.List(ctx, ds.db, req.J5Object(), res.J5Object())
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
	return res, ds.clusterQuery.ListEvents(ctx, ds.db, req.J5Object(), res.J5Object())
}
