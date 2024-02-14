package awsinfra

import (
	"context"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
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
		Request: msg.Request,
		State: &deployer_pb.PostgresMigrationState{
			Status: deployer_pb.PostgresMigrationStatus_UPSERTING,
		},
		Event: &deployer_pb.PostgresMigrationEvent{
			Event: &deployer_pb.PostgresMigrationEventType{
				Type: &deployer_pb.PostgresMigrationEventType_Prepare_{
					Prepare: &deployer_pb.PostgresMigrationEventType_Prepare{
						Spec: msg.MigrationSpec,
					},
				},
			},
		},
	}); err != nil {
		return nil, err
	}

	migrator := &DBMigrator{
		Clients: d.Clients,
	}

	migrateErr := migrator.MigratePostgresDatabase(ctx, msg)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresMigrationEventMessage{
			Request: msg.Request,
			State: &deployer_pb.PostgresMigrationState{
				Status: deployer_pb.PostgresMigrationStatus_FAILED,
			},
			Event: &deployer_pb.PostgresMigrationEvent{
				Event: &deployer_pb.PostgresMigrationEventType{
					Type: &deployer_pb.PostgresMigrationEventType_Error_{
						Error: &deployer_pb.PostgresMigrationEventType_Error{
							Error: migrateErr.Error(),
						},
					},
				},
			},
		}); err != nil {
			return nil, err
		}
		return &emptypb.Empty{}, nil
	}

	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresMigrationEventMessage{
		Request: msg.Request,
		State: &deployer_pb.PostgresMigrationState{
			Status: deployer_pb.PostgresMigrationStatus_DONE,
		},
		Event: &deployer_pb.PostgresMigrationEvent{
			Event: &deployer_pb.PostgresMigrationEventType{
				Type: &deployer_pb.PostgresMigrationEventType_Done_{
					Done: &deployer_pb.PostgresMigrationEventType_Done{},
				},
			},
		},
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
