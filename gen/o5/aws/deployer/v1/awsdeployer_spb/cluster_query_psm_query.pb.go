// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package awsdeployer_spb

import (
	context "context"
	psm "github.com/pentops/protostate/psm"
	sqrlx "github.com/pentops/sqrlx.go/sqrlx"
)

// State Query Service for %sCluster
// QuerySet is the query set for the Cluster service.

type ClusterPSMQuerySet = psm.StateQuerySet[
	*GetClusterRequest,
	*GetClusterResponse,
	*ListClustersRequest,
	*ListClustersResponse,
	*ListClusterEventsRequest,
	*ListClusterEventsResponse,
]

func NewClusterPSMQuerySet(
	smSpec psm.QuerySpec[
		*GetClusterRequest,
		*GetClusterResponse,
		*ListClustersRequest,
		*ListClustersResponse,
		*ListClusterEventsRequest,
		*ListClusterEventsResponse,
	],
	options psm.StateQueryOptions,
) (*ClusterPSMQuerySet, error) {
	return psm.BuildStateQuerySet[
		*GetClusterRequest,
		*GetClusterResponse,
		*ListClustersRequest,
		*ListClustersResponse,
		*ListClusterEventsRequest,
		*ListClusterEventsResponse,
	](smSpec, options)
}

type ClusterPSMQuerySpec = psm.QuerySpec[
	*GetClusterRequest,
	*GetClusterResponse,
	*ListClustersRequest,
	*ListClustersResponse,
	*ListClusterEventsRequest,
	*ListClusterEventsResponse,
]

func DefaultClusterPSMQuerySpec(tableSpec psm.QueryTableSpec) ClusterPSMQuerySpec {
	return psm.QuerySpec[
		*GetClusterRequest,
		*GetClusterResponse,
		*ListClustersRequest,
		*ListClustersResponse,
		*ListClusterEventsRequest,
		*ListClusterEventsResponse,
	]{
		QueryTableSpec: tableSpec,
		ListRequestFilter: func(req *ListClustersRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			return filter, nil
		},
		ListEventsRequestFilter: func(req *ListClusterEventsRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			filter["cluster_id"] = req.ClusterId
			return filter, nil
		},
	}
}

type ClusterQueryServiceImpl struct {
	db       sqrlx.Transactor
	querySet *ClusterPSMQuerySet
	UnsafeClusterQueryServiceServer
}

var _ ClusterQueryServiceServer = &ClusterQueryServiceImpl{}

func NewClusterQueryServiceImpl(db sqrlx.Transactor, querySet *ClusterPSMQuerySet) *ClusterQueryServiceImpl {
	return &ClusterQueryServiceImpl{
		db:       db,
		querySet: querySet,
	}
}

func (s *ClusterQueryServiceImpl) GetCluster(ctx context.Context, req *GetClusterRequest) (*GetClusterResponse, error) {
	resObject := &GetClusterResponse{}
	err := s.querySet.Get(ctx, s.db, req, resObject)
	if err != nil {
		return nil, err
	}
	return resObject, nil
}

func (s *ClusterQueryServiceImpl) ListClusters(ctx context.Context, req *ListClustersRequest) (*ListClustersResponse, error) {
	resObject := &ListClustersResponse{}
	err := s.querySet.List(ctx, s.db, req, resObject)
	if err != nil {
		return nil, err
	}
	return resObject, nil
}

func (s *ClusterQueryServiceImpl) ListClusterEvents(ctx context.Context, req *ListClusterEventsRequest) (*ListClusterEventsResponse, error) {
	resObject := &ListClusterEventsResponse{}
	err := s.querySet.ListEvents(ctx, s.db, req, resObject)
	if err != nil {
		return nil, err
	}
	return resObject, nil
}
