package awsinfra

import (
	"context"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PostgresMigrateWorker struct {
	*deployer_tpb.UnimplementedPostgresMigrationTopicServer

	db DBLite
	*CFClient
}

func NewPostgresMigrateWorker(clients ClientBuilder, db DBLite) *PostgresMigrateWorker {
	cfClient := &CFClient{
		Clients: clients,
	}
	return &PostgresMigrateWorker{
		db:       db,
		CFClient: cfClient,
	}
}

func (d *PostgresMigrateWorker) MigratePostgresDatabase(ctx context.Context, msg *deployer_tpb.MigratePostgresDatabaseMessage) (*emptypb.Empty, error) {
	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresMigrationEventMessage{
		Request:     msg.Request,
		MigrationId: msg.MigrationId,
		Status:      deployer_tpb.PostgresStatus_CREATING,
	}); err != nil {
		return nil, err
	}

	migrator := &DBMigrator{
		Clients: d.Clients,
	}

	migrateErr := migrator.MigratePostgresDatabase(ctx, msg)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		errMsg := migrateErr.Error()
		if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresMigrationEventMessage{
			Request:     msg.Request,
			MigrationId: msg.MigrationId,
			Status:      deployer_tpb.PostgresStatus_ERROR,
			Error:       &errMsg,
		}); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresMigrationEventMessage{
		Request:     msg.Request,
		MigrationId: msg.MigrationId,
		Status:      deployer_tpb.PostgresStatus_AVAILABLE,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
