package awsinfra

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type PostgresMigrateWorker struct {
	awsinfra_tpb.UnimplementedPostgresRequestTopicServer
	awsinfra_tpb.UnimplementedECSReplyTopicServer

	db       DBLite
	migrator IDBMigrator
}

func NewPostgresMigrateWorker(db DBLite, migrator IDBMigrator) *PostgresMigrateWorker {
	return &PostgresMigrateWorker{
		db:       db,
		migrator: migrator,
	}
}

type IDBMigrator interface {
	UpsertPostgresDatabase(ctx context.Context, migrationID string, msg *awsdeployer_pb.PostgresCreationSpec) error
	CleanupPostgresDatabase(ctx context.Context, migrationID string, msg *awsdeployer_pb.PostgresCleanupSpec) error
}

type pgRequest interface {
	GetRequest() *messaging_j5pb.RequestMetadata
	GetMigrationId() string
}

var migrationNamespace = uuid.MustParse("0C99B6B3-826C-4428-940A-62492DE5BA8F")

func (d *PostgresMigrateWorker) runCallback(ctx context.Context, msg pgRequest, phase string, cb func(context.Context) error) error {
	request := msg.GetRequest()
	if request == nil {
		return fmt.Errorf("request is nil")
	}

	migrationId := msg.GetMigrationId()

	if err := d.db.PublishEvent(ctx, &awsinfra_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		MigrationId: msg.GetMigrationId(),
		Status:      awsinfra_tpb.PostgresStatus_STARTED,
		EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("%s-%s-started", phase, migrationId))).String(),
	}); err != nil {
		return err
	}

	migrateErr := cb(ctx)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		errMsg := migrateErr.Error()
		if err := d.db.PublishEvent(ctx, &awsinfra_tpb.PostgresDatabaseStatusMessage{
			Request:     request,
			EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("%s-%s-error", phase, migrationId))).String(),
			MigrationId: msg.GetMigrationId(),
			Status:      awsinfra_tpb.PostgresStatus_ERROR,
			Error:       &errMsg,
		}); err != nil {
			return err
		}
		return nil
	}

	if err := d.db.PublishEvent(ctx, &awsinfra_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("%s-%s-done", phase, migrationId))).String(),
		MigrationId: msg.GetMigrationId(),
		Status:      awsinfra_tpb.PostgresStatus_DONE,
	}); err != nil {
		return err
	}

	return nil
}

func (d *PostgresMigrateWorker) UpsertPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.UpsertPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, d.runCallback(ctx, msg, "upsert", func(ctx context.Context) error {
		return d.migrator.UpsertPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (d *PostgresMigrateWorker) CleanupPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.CleanupPostgresDatabaseMessage) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, d.runCallback(ctx, msg, "cleanup", func(ctx context.Context) error {
		return d.migrator.CleanupPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (d *PostgresMigrateWorker) MigratePostgresDatabase(ctx context.Context, msg *awsinfra_tpb.MigratePostgresDatabaseMessage) (*emptypb.Empty, error) {

	request := msg.GetRequest()
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if err := d.db.PublishEvent(ctx, &awsinfra_tpb.PostgresDatabaseStatusMessage{
		Request:     request,
		MigrationId: msg.GetMigrationId(),
		Status:      awsinfra_tpb.PostgresStatus_STARTED,
		EventId:     uuid.NewSHA1(migrationNamespace, []byte(fmt.Sprintf("migrate-%s-started", msg.MigrationId))).String(),
	}); err != nil {
		return nil, err
	}

	taskContext := &awsdeployer_pb.MigrationTaskContext{
		Upstream:    msg.Request,
		MigrationId: msg.MigrationId,
	}
	taskContextBytes, err := proto.Marshal(taskContext)
	if err != nil {
		return nil, err
	}

	if err := d.db.PublishEvent(ctx, &awsinfra_tpb.RunECSTaskMessage{
		Request: &messaging_j5pb.RequestMetadata{
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

func (d *PostgresMigrateWorker) ECSTaskStatus(ctx context.Context, msg *awsinfra_tpb.ECSTaskStatusMessage) (*emptypb.Empty, error) {

	taskContext := &awsdeployer_pb.MigrationTaskContext{}
	if err := proto.Unmarshal(msg.Request.Context, taskContext); err != nil {
		return nil, err
	}

	replyMessage := &awsinfra_tpb.PostgresDatabaseStatusMessage{
		Request:     taskContext.Upstream,
		EventId:     msg.EventId,
		MigrationId: taskContext.MigrationId,
	}

	switch et := msg.Event.Type.(type) {
	case *awsinfra_tpb.ECSTaskEventType_Pending_, *awsinfra_tpb.ECSTaskEventType_Running_:
		// Ignore
		return &emptypb.Empty{}, nil

	case *awsinfra_tpb.ECSTaskEventType_Exited_:
		if et.Exited.ExitCode == 0 {
			replyMessage.Status = awsinfra_tpb.PostgresStatus_DONE
		} else {
			replyMessage.Status = awsinfra_tpb.PostgresStatus_ERROR
			replyMessage.Error = aws.String(fmt.Sprintf("task exited with code %d", et.Exited.ExitCode))
		}

	case *awsinfra_tpb.ECSTaskEventType_Failed_:
		replyMessage.Status = awsinfra_tpb.PostgresStatus_ERROR
		replyMessage.Error = aws.String(et.Failed.Reason)
		if et.Failed.ContainerName != nil {
			replyMessage.Error = aws.String(fmt.Sprintf("%s: %s", *et.Failed.ContainerName, et.Failed.Reason))
		}

	case *awsinfra_tpb.ECSTaskEventType_Stopped_:
		replyMessage.Status = awsinfra_tpb.PostgresStatus_ERROR
		replyMessage.Error = aws.String(fmt.Sprintf("ECS Task Stopped: %s", et.Stopped.Reason))

	default:
		return nil, fmt.Errorf("unknown ECS Status Event type: %T", et)
	}

	if err := d.db.PublishEvent(ctx, replyMessage); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
