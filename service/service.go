package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	deploymentStateMachine  *deployer_pb.DeploymentPSM
	environmentStateMachine *deployer_pb.EnvironmentPSM
	stackStateMachine       *deployer_pb.StackPSM

	db     *sqrlx.Wrapper
	github github.IClient

	*deployer_spb.UnimplementedDeploymentCommandServiceServer
}

func NewDeployerService(conn sqrlx.Connection, github github.IClient, stateMachines *states.StateMachines) (*DeployerService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &DeployerService{
		deploymentStateMachine:  stateMachines.Deployment,
		environmentStateMachine: stateMachines.Environment,
		stackStateMachine:       stateMachines.Stack,

		db:     db,
		github: github,
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

	var environmentID string

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, sq.Select("id").From("environment").Where("state->'config'->>'fullName' = ?", req.EnvironmentName)).Scan(&environmentID)
	}); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "environment not found")
		}
		return nil, err
	}

	requestMessage := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId:  req.DeploymentId,
		Application:   apps[0],
		Version:       gh.Commit,
		EnvironmentId: environmentID,
		Flags:         req.Flags,
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

func (ds *DeployerService) TerminateDeployment(ctx context.Context, req *deployer_spb.TerminateDeploymentRequest) (*deployer_spb.TerminateDeploymentResponse, error) {

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: req.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	event.SetPSMEvent(&deployer_pb.DeploymentEventType_Terminated{})

	_, err := ds.deploymentStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &deployer_spb.TerminateDeploymentResponse{}, nil
}

func (ds *DeployerService) UpsertEnvironment(ctx context.Context, req *deployer_spb.UpsertEnvironmentRequest) (*deployer_spb.UpsertEnvironmentResponse, error) {

	event := &deployer_pb.EnvironmentEvent{
		EnvironmentId: req.EnvironmentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	event.SetPSMEvent(&deployer_pb.EnvironmentEventType_Configured{
		Config: req.Config,
	})

	_, err := ds.environmentStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &deployer_spb.UpsertEnvironmentResponse{}, nil
}

func (ds *DeployerService) UpsertStack(ctx context.Context, req *deployer_spb.UpsertStackRequest) (*deployer_spb.UpsertStackResponse, error) {

	event := &deployer_pb.StackEvent{
		StackId: req.StackId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	event.SetPSMEvent(&deployer_pb.StackEventType_Configured{
		Config:          req.Config,
		EnvironmentId:   req.EnvironmentId,
		ApplicationName: req.ApplicationName,
	})

	_, err := ds.stackStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &deployer_spb.UpsertStackResponse{}, nil
}
