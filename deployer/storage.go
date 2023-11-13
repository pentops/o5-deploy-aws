package deployer

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/sqrlx"
)

var DeploymentNotFoundError = fmt.Errorf("deployment not found")

type DeployerStateStore interface {
	StoreDeploymentEvent(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error
	GetDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error)
	GetDeploymentForStack(ctx context.Context, stackName string) (*deployer_pb.DeploymentState, error)

	PublishEvent(ctx context.Context, msg proto.Message) error
}

type PostgresStateStore struct {
	db *sqrlx.Wrapper
}

func NewPostgresStateStore(conn sqrlx.Connection) (*PostgresStateStore, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}
	return &PostgresStateStore{db: db}, nil
}

func (pgs *PostgresStateStore) GetDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error) {
	return nil, fmt.Errorf("not implemented")
}

func (pgs *PostgresStateStore) StoreDeploymentEvent(ctx context.Context, deployment *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {
	deploymentJSON, err := protojson.Marshal(deployment)
	if err != nil {
		return err
	}

	upsertState := sqrlx.Upsert("deployment").
		Key("id", deployment.DeploymentId).
		Set("state", deploymentJSON)

	eventJSON, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	insertEvent := sq.Insert("deployment_event").SetMap(map[string]interface{}{
		"deployment_id": deployment.DeploymentId,
		"id":            event.Metadata.EventId,
		"event":         eventJSON,
		"timestamp":     event.Metadata.Timestamp.AsTime(),
	})
	return pgs.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		Retryable: true,
		ReadOnly:  false,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		_, err := tx.Insert(ctx, upsertState)
		if err != nil {
			return err
		}

		_, err = tx.Insert(ctx, insertEvent)
		if err != nil {
			return err
		}

		return nil

	})
}
