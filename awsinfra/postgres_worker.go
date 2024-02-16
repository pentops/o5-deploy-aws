package awsinfra

import (
	"context"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PostgresMigrateWorker struct {
	*deployer_tpb.UnimplementedPostgresRequestTopicServer

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

type pgRequest interface {
	GetRequest() *messaging_pb.RequestMetadata
	GetMigrationId() string
}

func (d *PostgresMigrateWorker) runCallback(ctx context.Context, msg pgRequest, cb func(context.Context, *DBMigrator) error) error {
	request := msg.GetRequest()
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		MigrationId: msg.GetMigrationId(),
		Status:      deployer_tpb.PostgresStatus_STARTED,
	}); err != nil {
		return err
	}

	migrator := &DBMigrator{
		Clients: d.Clients,
	}

	migrateErr := cb(ctx, migrator)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		errMsg := migrateErr.Error()
		if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresDatabaseStatusMessage{
			Request:     request,
			MigrationId: msg.GetMigrationId(),
			Status:      deployer_tpb.PostgresStatus_ERROR,
			Error:       &errMsg,
		}); err != nil {
			return err
		}
		return nil
	}

	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		MigrationId: msg.GetMigrationId(),
		Status:      deployer_tpb.PostgresStatus_DONE,
	}); err != nil {
		return err
	}

	return nil
}

func (d *PostgresMigrateWorker) MigratePostgresDatabase(ctx context.Context, msg *deployer_tpb.MigratePostgresDatabaseMessage) (*emptypb.Empty, error) {

	return &emptypb.Empty{}, d.runCallback(ctx, msg, func(ctx context.Context, migrator *DBMigrator) error {
		return migrator.MigratePostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (d *PostgresMigrateWorker) UpsertPostgresDatabase(ctx context.Context, msg *deployer_tpb.UpsertPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, d.runCallback(ctx, msg, func(ctx context.Context, migrator *DBMigrator) error {
		return migrator.UpsertPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (d *PostgresMigrateWorker) CleanupPostgresDatabase(ctx context.Context, msg *deployer_tpb.CleanupPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, d.runCallback(ctx, msg, func(ctx context.Context, migrator *DBMigrator) error {
		return migrator.CleanupPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}
