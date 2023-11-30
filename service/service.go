package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/genericstate"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"google.golang.org/protobuf/reflect/protoreflect"
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
	DeploymentQuery *genericstate.StateQuerySet
	StackQuery      *genericstate.StateQuerySet

	db *sqrlx.Wrapper
	*deployer_spb.UnimplementedDeploymentQueryServiceServer
}

func NewDeployerService(conn sqrlx.Connection) (*DeployerService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	deploymentQuery, err := genericstate.NewStateQuery(genericstate.StateQuerySpec{
		TableName:              "deployment",
		DataColumn:             "state",
		PrimaryKeyColumn:       "id",
		PrimaryKeyRequestField: protoreflect.Name("deployment_id"),
		Events: &genericstate.GetJoinSpec{
			TableName:        "deployment_event",
			DataColumn:       "event",
			FieldInParent:    "events",
			ForeignKeyColumn: "deployment_id",
		},

		Get: &genericstate.MethodDescriptor{
			Request:  (&deployer_spb.GetDeploymentRequest{}).ProtoReflect().Descriptor(),
			Response: (&deployer_spb.GetDeploymentResponse{}).ProtoReflect().Descriptor(),
		},

		List: &genericstate.MethodDescriptor{
			Request:  (&deployer_spb.ListDeploymentsRequest{}).ProtoReflect().Descriptor(),
			Response: (&deployer_spb.ListDeploymentsResponse{}).ProtoReflect().Descriptor(),
		},

		ListEvents: &genericstate.MethodDescriptor{
			Request:  (&deployer_spb.ListDeploymentEventsRequest{}).ProtoReflect().Descriptor(),
			Response: (&deployer_spb.ListDeploymentEventsResponse{}).ProtoReflect().Descriptor(),
		},
	})
	if err != nil {
		return nil, err
	}

	stackQuery, err := genericstate.NewStateQuery(genericstate.StateQuerySpec{
		TableName:              "stack",
		DataColumn:             "state",
		PrimaryKeyColumn:       "id",
		PrimaryKeyRequestField: protoreflect.Name("stack_id"),
		Events: &genericstate.GetJoinSpec{
			TableName:        "stack_event",
			DataColumn:       "event",
			FieldInParent:    "events",
			ForeignKeyColumn: "stack_id",
		},

		Get: &genericstate.MethodDescriptor{
			Request:  (&deployer_spb.GetStackRequest{}).ProtoReflect().Descriptor(),
			Response: (&deployer_spb.GetStackResponse{}).ProtoReflect().Descriptor(),
		},

		List: &genericstate.MethodDescriptor{
			Request:  (&deployer_spb.ListStacksRequest{}).ProtoReflect().Descriptor(),
			Response: (&deployer_spb.ListStacksResponse{}).ProtoReflect().Descriptor(),
		},

		ListEvents: &genericstate.MethodDescriptor{
			Request:  (&deployer_spb.ListStackEventsRequest{}).ProtoReflect().Descriptor(),
			Response: (&deployer_spb.ListStackEventsResponse{}).ProtoReflect().Descriptor(),
		},
	})
	if err != nil {
		return nil, err
	}

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

func (ds *DeployerService) GetStack(ctx context.Context, req *deployer_spb.GetStackRequest) (*deployer_spb.GetStackResponse, error) {
	res := &deployer_spb.GetStackResponse{}
	return res, ds.StackQuery.Get(ctx, ds.db, req, res)
}
