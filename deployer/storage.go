package deployer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.daemonl.com/sqrlx"

	"github.com/pentops/outbox.pg.go/outbox"
)

var DeploymentNotFoundError = fmt.Errorf("deployment not found")
var StackNotFoundError = fmt.Errorf("stack not found")

var namespaceStackID = uuid.MustParse("C27983FD-BC4B-493F-A056-CC8C869A1999")

func StackID(envName, appName string) string {
	return uuid.NewMD5(namespaceStackID, []byte(fmt.Sprintf("%s-%s", envName, appName))).String()
}

type DeployerStorage interface {
	Transact(context.Context, func(context.Context, TransitionTransaction) error) error
}

type TransitionTransaction interface {
	StoreDeploymentEvent(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error
	GetDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error)
	GetEnvironment(ctx context.Context, environmentName string) (*environment_pb.Environment, error)

	PublishEvent(ctx context.Context, msg outbox.OutboxMessage) error

	GetStack(ctx context.Context, stackID string) (*deployer_pb.StackState, error)
	StoreStackEvent(ctx context.Context, stack *deployer_pb.StackState, event *deployer_pb.StackEvent) error
}

type PostgresStateStore struct {
	db           *sqrlx.Wrapper
	environments map[string]*environment_pb.Environment
}

func NewPostgresStateStore(conn sqrlx.Connection, environments []*environment_pb.Environment) (*PostgresStateStore, error) {
	envMap := map[string]*environment_pb.Environment{}
	for _, env := range environments {
		envMap[env.FullName] = env
	}

	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}
	return &PostgresStateStore{db: db, environments: envMap}, nil
}

func (pgs *PostgresStateStore) Transact(ctx context.Context, callback func(context.Context, TransitionTransaction) error) error {
	return pgs.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		Retryable: true,
		ReadOnly:  false,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return callback(ctx, &postgresTxWrapper{tx: tx, environments: pgs.environments})
	})
}

func (pgs *PostgresStateStore) PublishEvent(ctx context.Context, evt outbox.OutboxMessage) error {
	return pgs.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return tx.PublishEvent(ctx, evt)
	})
}

func (pgs *PostgresStateStore) StoreDeploymentEvent(ctx context.Context, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
	return pgs.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return tx.StoreDeploymentEvent(ctx, deployment, event)
	})
}

func (pgs *PostgresStateStore) GetEnvironment(ctx context.Context, environmentName string) (*environment_pb.Environment, error) {
	var env *environment_pb.Environment

	return env, pgs.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		var err error
		env, err = tx.GetEnvironment(ctx, environmentName)
		return err
	})
}

type postgresTxWrapper struct {
	tx           sqrlx.Transaction
	environments map[string]*environment_pb.Environment
}

func (ptw *postgresTxWrapper) GetEnvironment(ctx context.Context, environmentName string) (*environment_pb.Environment, error) {
	env, ok := ptw.environments[environmentName]
	if ok {
		return env, nil
	}

	return nil, status.Errorf(codes.NotFound, "environment '%s' not found", environmentName)
}

func (ptw *postgresTxWrapper) PublishEvent(ctx context.Context, evt outbox.OutboxMessage) error {
	return outbox.Send(ctx, ptw.tx, evt)
}

func (ptw *postgresTxWrapper) GetDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error) {
	var deploymentJSON []byte
	err := ptw.tx.SelectRow(ctx, sq.Select("state").From("deployment").Where("id = ?", id)).Scan(&deploymentJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, DeploymentNotFoundError
	} else if err != nil {
		return nil, err
	}
	var deployment deployer_pb.DeploymentState
	if err := protojson.Unmarshal(deploymentJSON, &deployment); err != nil {
		return nil, fmt.Errorf("unmarshal deployment: %w", err)
	}
	return &deployment, nil
}

func (ptw *postgresTxWrapper) GetStack(ctx context.Context, stackID string) (*deployer_pb.StackState, error) {
	var stackJSON []byte
	err := ptw.tx.SelectRow(ctx, sq.Select("state").From("stack").Where("id = ?", stackID)).Scan(&stackJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, StackNotFoundError
	}
	if err != nil {
		return nil, err
	}
	var stack deployer_pb.StackState
	if err := protojson.Unmarshal(stackJSON, &stack); err != nil {
		return nil, err
	}
	return &stack, nil
}

func (ptw *postgresTxWrapper) StoreStackEvent(ctx context.Context, stack *deployer_pb.StackState, event *deployer_pb.StackEvent) error {
	stackJSON, err := protojson.Marshal(stack)
	if err != nil {
		return err
	}

	upsertStack := sqrlx.Upsert("stack").
		Key("id", stack.StackId).SetMap(map[string]interface{}{
		"env_name": stack.EnvironmentName,
		"app_name": stack.ApplicationName,
		"state":    stackJSON,
	})

	_, err = ptw.tx.Insert(ctx, upsertStack)
	if err != nil {
		return err
	}

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	insertEvent := sq.Insert("stack_event").SetMap(map[string]interface{}{
		"stack_id":  stack.StackId,
		"id":        event.Metadata.EventId,
		"event":     eventJSON,
		"timestamp": event.Metadata.Timestamp.AsTime(),
	})

	_, err = ptw.tx.Insert(ctx, insertEvent)
	if err != nil {
		return err
	}

	return nil
}

func (ptw *postgresTxWrapper) StoreDeploymentEvent(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
	deploymentJSON, err := protojson.Marshal(state)
	if err != nil {
		return err
	}

	upsertState := sqrlx.Upsert("deployment").
		Key("id", state.DeploymentId).
		Set("state", deploymentJSON).
		Set("stack_id", state.StackId)

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	_, err = ptw.tx.Insert(ctx, upsertState)
	if err != nil {
		return err
	}

	insertEvent := sq.Insert("deployment_event").SetMap(map[string]interface{}{
		"deployment_id": state.DeploymentId,
		"id":            event.Metadata.EventId,
		"event":         eventJSON,
		"timestamp":     event.Metadata.Timestamp.AsTime(),
	})

	_, err = ptw.tx.Insert(ctx, insertEvent)
	if err != nil {
		return err
	}

	return nil

}
