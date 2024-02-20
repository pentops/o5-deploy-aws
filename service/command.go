package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
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

type CommandService struct {
	deploymentStateMachine  *deployer_pb.DeploymentPSM
	environmentStateMachine *deployer_pb.EnvironmentPSM
	stackStateMachine       *deployer_pb.StackPSM

	db     *sqrlx.Wrapper
	github github.IClient

	*deployer_spb.UnimplementedDeploymentCommandServiceServer
}

func NewCommandService(conn sqrlx.Connection, github github.IClient, stateMachines *states.StateMachines) (*CommandService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &CommandService{
		deploymentStateMachine:  stateMachines.Deployment,
		environmentStateMachine: stateMachines.Environment,
		stackStateMachine:       stateMachines.Stack,

		db:     db,
		github: github,
	}, nil
}

type environmentIdentifiers struct {
	fullName string
	id       string
}

type stackIdentifiers struct {
	environment environmentIdentifiers
	appName     string
	stackID     string
}

var stackIDNamespace = uuid.MustParse("87742CC6-9D66-4547-A744-647B1C0D7F59")

func (ds *CommandService) lookupStack(ctx context.Context, presented string) (stackIdentifiers, error) {

	query := sq.
		Select(
			"id",
			"state->>'applicationName'",
			"state->>'environmentId'",
			"state->>'environmentName'",
		).From("stack")

	fallbackAppName := ""
	fallbackEnvName := ""

	parts := strings.Split(presented, "-")
	if len(parts) == 2 {
		query.Where("state->>'environmentName' = ?", parts[0])
		query.Where("state->>'applicationName' = ?", parts[1])
		fallbackEnvName = parts[0]
		fallbackAppName = parts[1]
	} else if _, err := uuid.Parse(presented); err == nil {
		query = query.Where("id = ?", presented)
	} else {
		return stackIdentifiers{}, status.Error(codes.InvalidArgument, "invalid stack id")
	}

	res := stackIdentifiers{}

	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		err := tx.SelectRow(ctx, query).Scan(
			&res.stackID,
			&res.appName,
			&res.environment.id,
			&res.environment.fullName,
		)
		if err == nil { // HAPPY SAD FLIP
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if fallbackAppName == "" {
			return status.Error(codes.NotFound, "stack not found")
		}

		res.appName = fallbackAppName
		res.environment.fullName = fallbackEnvName

		err = tx.SelectRow(ctx, sq.Select("id").
			From("environment").Where("state->'config'->>'fullName' = ?", fallbackEnvName)).
			Scan(&res.environment.id)

		if errors.Is(err, sql.ErrNoRows) {
			// Will fill in a generated ID later
			return nil
		} else if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return res, err
	}

	// Build up any missing information from the fallbacks.
	if res.environment.id == "" {
		res.environment.id = uuid.NewSHA1(environmentIDNamespace, []byte(fallbackEnvName)).String()
		log.WithFields(ctx, map[string]interface{}{
			"environment_id": res.environment.id,
			"environment":    fallbackEnvName,
		}).Debug("derived environment id")
	}
	if res.stackID == "" {
		res.stackID = uuid.NewSHA1(stackIDNamespace, []byte(fallbackAppName+"-"+fallbackEnvName)).String()
		log.WithFields(ctx, map[string]interface{}{
			"stack_id":    res.stackID,
			"environment": fallbackEnvName,
			"application": fallbackAppName,
		}).Debug("derived stack id")
	}

	return res, nil

}

var environmentIDNamespace = uuid.MustParse("0D783718-F8FD-4543-AE3D-6382AB0B8178")

func (ds *CommandService) lookupEnvironment(ctx context.Context, presented string) (environmentIdentifiers, error) {
	query := sq.
		Select("id", "state->'config'->>'fullName'").
		From("environment")

	fallbackToName := ""
	if _, err := uuid.Parse(presented); err == nil {
		query = query.Where("id = ?", presented)
	} else {
		query = query.Where("state->'config'->>'fullName' = ?", presented)
		fallbackToName = presented
	}

	res := environmentIdentifiers{}
	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, query).Scan(&res.id, &res.fullName)
	})
	if errors.Is(err, sql.ErrNoRows) {
		if fallbackToName == "" {
			return res, status.Error(codes.NotFound, "environment not found")
		}
		id := uuid.NewSHA1(environmentIDNamespace, []byte(fallbackToName)).String()
		return environmentIdentifiers{
			fullName: fallbackToName,
			id:       id,
		}, nil

	} else if err != nil {
		return res, err
	}
	return res, nil
}

func (ds *CommandService) TriggerDeployment(ctx context.Context, req *deployer_spb.TriggerDeploymentRequest) (*deployer_spb.TriggerDeploymentResponse, error) {

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

	environmentID, err := ds.lookupEnvironment(ctx, req.Environment)
	if err != nil {
		return nil, err
	}

	requestMessage := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId:  req.DeploymentId,
		Application:   apps[0],
		Version:       gh.Commit,
		EnvironmentId: environmentID.id,
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

	return &deployer_spb.TriggerDeploymentResponse{
		DeploymentId:    req.DeploymentId,
		EnvironmentId:   environmentID.id,
		EnvironmentName: environmentID.fullName,
	}, nil
}

func (ds *CommandService) TerminateDeployment(ctx context.Context, req *deployer_spb.TerminateDeploymentRequest) (*deployer_spb.TerminateDeploymentResponse, error) {

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

func (ds *CommandService) UpsertEnvironment(ctx context.Context, req *deployer_spb.UpsertEnvironmentRequest) (*deployer_spb.UpsertEnvironmentResponse, error) {

	var config *environment_pb.Environment
	switch src := req.Src.(type) {
	case *deployer_spb.UpsertEnvironmentRequest_Config:
		config = src.Config

	case *deployer_spb.UpsertEnvironmentRequest_ConfigYaml:
		config = &environment_pb.Environment{}
		if err := protoread.Parse("env.yaml", src.ConfigYaml, config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

	case *deployer_spb.UpsertEnvironmentRequest_ConfigJson:
		config = &environment_pb.Environment{}
		if err := protoread.Parse("env.json", src.ConfigJson, config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid config type")
	}

	if req.EnvironmentId == "" || req.EnvironmentId == "-" {
		req.EnvironmentId = config.FullName
	}
	identifiers, err := ds.lookupEnvironment(ctx, req.EnvironmentId)
	if err != nil {
		return nil, err
	}

	event := &deployer_pb.EnvironmentEvent{
		EnvironmentId: identifiers.id,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	event.SetPSMEvent(&deployer_pb.EnvironmentEventType_Configured{
		Config: config,
	})

	state, err := ds.environmentStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		log.WithError(ctx, err).Error("failed to transition environment")
		return nil, err
	}

	return &deployer_spb.UpsertEnvironmentResponse{
		State: state,
	}, nil
}

func (ds *CommandService) UpsertStack(ctx context.Context, req *deployer_spb.UpsertStackRequest) (*deployer_spb.UpsertStackResponse, error) {

	identifiers, err := ds.lookupStack(ctx, req.StackId)
	if err != nil {
		return nil, err
	}

	event := &deployer_pb.StackEvent{
		StackId: identifiers.stackID,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
	}

	event.SetPSMEvent(&deployer_pb.StackEventType_Configured{
		Config:          req.Config,
		EnvironmentId:   identifiers.environment.id,
		ApplicationName: identifiers.appName,
		EnvironmentName: identifiers.environment.fullName,
	})

	_, err = ds.stackStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &deployer_spb.UpsertStackResponse{}, nil
}
