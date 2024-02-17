package awsinfra

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ECSTaskRunner interface {
	RunTask(ctx context.Context, req *deployer_tpb.RunECSTaskMessage) error
}
type PostgresMigrateWorker struct {
	deployer_tpb.UnimplementedPostgresRequestTopicServer
	deployer_tpb.UnimplementedECSReplyTopicServer

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

var migrationNamespace = uuid.MustParse("0C99B6B3-826C-4428-940A-62492DE5BA8F")

func (d *PostgresMigrateWorker) runCallback(ctx context.Context, msg pgRequest, phase string, cb func(context.Context, *DBMigrator) error) error {
	request := msg.GetRequest()
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	migrationId := msg.GetMigrationId()

	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		MigrationId: msg.GetMigrationId(),
		Status:      deployer_tpb.PostgresStatus_STARTED,
		EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("%s-%s-started", phase, migrationId))).String(),
	}); err != nil {
		return err
	}

	migrator := NewDBMigrator(d.Clients)

	migrateErr := cb(ctx, migrator)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		errMsg := migrateErr.Error()
		if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresDatabaseStatusMessage{
			Request:     request,
			EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("%s-%s-error", phase, migrationId))).String(),
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
		EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("%s-%s-done", phase, migrationId))).String(),
		MigrationId: msg.GetMigrationId(),
		Status:      deployer_tpb.PostgresStatus_DONE,
	}); err != nil {
		return err
	}

	return nil
}

func (d *PostgresMigrateWorker) UpsertPostgresDatabase(ctx context.Context, msg *deployer_tpb.UpsertPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, d.runCallback(ctx, msg, "upsert", func(ctx context.Context, migrator *DBMigrator) error {
		return migrator.UpsertPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (d *PostgresMigrateWorker) CleanupPostgresDatabase(ctx context.Context, msg *deployer_tpb.CleanupPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, d.runCallback(ctx, msg, "cleanup", func(ctx context.Context, migrator *DBMigrator) error {
		return migrator.CleanupPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (d *PostgresMigrateWorker) MigratePostgresDatabase(ctx context.Context, msg *deployer_tpb.MigratePostgresDatabaseMessage) (*emptypb.Empty, error) {

	request := msg.GetRequest()
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if err := d.db.PublishEvent(ctx, &deployer_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		MigrationId: msg.GetMigrationId(),
		Status:      deployer_tpb.PostgresStatus_STARTED,
	}); err != nil {
		return nil, err
	}

	taskContext := &deployer_pb.MigrationTaskContext{
		Upstream:    msg.Request,
		MigrationId: msg.MigrationId,
	}
	taskContextBytes, err := proto.Marshal(taskContext)
	if err != nil {
		return nil, err
	}

	if err := d.db.PublishEvent(ctx, &deployer_tpb.RunECSTaskMessage{
		Request: &messaging_pb.RequestMetadata{
			Context: taskContextBytes,
			ReplyTo: "deployer",
		},
		TaskDefinition: msg.Spec.MigrationTaskArn,
		Cluster:        msg.Spec.EcsClusterName,
		Count:          1,
	}); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (d *PostgresMigrateWorker) ECSTaskStatus(ctx context.Context, msg *deployer_tpb.ECSTaskStatusMessage) (*emptypb.Empty, error) {

	taskContext := &deployer_pb.MigrationTaskContext{}
	if err := proto.Unmarshal(msg.Request.Context, taskContext); err != nil {
		return nil, err
	}

	replyMessage := &deployer_tpb.PostgresDatabaseStatusMessage{
		Request:     taskContext.Upstream,
		EventId:     msg.EventId,
		MigrationId: taskContext.MigrationId,
	}

	switch et := msg.Event.Type.(type) {
	case *deployer_tpb.ECSTaskEventType_Pending_, *deployer_tpb.ECSTaskEventType_Running_:
		// Ignore
		return &emptypb.Empty{}, nil

	case *deployer_tpb.ECSTaskEventType_Exited_:
		if et.Exited.ExitCode == 0 {
			replyMessage.Status = deployer_tpb.PostgresStatus_DONE
		} else {
			replyMessage.Status = deployer_tpb.PostgresStatus_ERROR
			replyMessage.Error = aws.String(fmt.Sprintf("task exited with code %d", et.Exited.ExitCode))
		}

	case *deployer_tpb.ECSTaskEventType_Failed_:
		replyMessage.Status = deployer_tpb.PostgresStatus_ERROR
		replyMessage.Error = aws.String(et.Failed.Reason)
		if et.Failed.ContainerName != nil {
			replyMessage.Error = aws.String(fmt.Sprintf("%s: %s", *et.Failed.ContainerName, et.Failed.Reason))
		}

	case *deployer_tpb.ECSTaskEventType_Stopped_:
		replyMessage.Status = deployer_tpb.PostgresStatus_ERROR
		replyMessage.Error = aws.String(fmt.Sprintf("ECS Task Stopped: %s", et.Stopped.Reason))

	default:
		return nil, fmt.Errorf("unknown ECS Status Event type: %T", et)
	}

	if err := d.db.PublishEvent(ctx, replyMessage); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
