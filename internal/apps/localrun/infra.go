package localrun

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/aws_cf"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/aws_postgres"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsapi"
	"github.com/pentops/o5-deploy-aws/internal/apps/service"
	"google.golang.org/protobuf/proto"
)

type InfraAdapter struct {
	cfClient  *aws_cf.CFClient
	dbClient  *aws_postgres.DBMigrator
	ecsClient *ecsRunner
}

func NewInfraAdapter(ctx context.Context, cl *awsapi.DeployerClients) (*InfraAdapter, error) {
	cfClient := aws_cf.NewCFAdapter(cl)
	dbMigrator := aws_postgres.NewDBMigrator(cl.SecretsManager, cl.RDSAuthProvider)
	ecsClient := &ecsRunner{
		ecsClient:        cl.ECS,
		cloudwatchClient: cl.CloudWatchLogs,
	}

	return &InfraAdapter{
		cfClient:  cfClient,
		dbClient:  dbMigrator,
		ecsClient: ecsClient,
	}, nil
}
func newToken() string {
	return fmt.Sprintf("local-%d", time.Now().UnixNano())
}

func stackEvent(msg *awsinfra_tpb.StackStatusChangedMessage, err error) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, fmt.Errorf("stackEvent: msg and error are nil")
	}
	if msg.Request == nil {
		return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
	}
	event, err := service.StackStatusToEvent(msg)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func ecsEvent(msg *awsinfra_tpb.ECSTaskStatusMessage, err error) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("ecsEvent: msg and error are nil")
	}

	if msg.Request == nil {
		return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
	}

	event, err := service.ECSTaskStatusToEvent(msg)
	if err != nil {
		return nil, err
	}
	return event, nil

}

func dbEvent(msg *awsinfra_tpb.PostgresDatabaseStatusMessage, err error) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	if err != nil {
		return nil, err
	}
	if msg.Request == nil {
		return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
	}
	if msg.Request.Context == nil {
		return nil, fmt.Errorf("missing request.Context in %s", msg.ProtoReflect().Descriptor().FullName())
	}
	event, err := service.PostgresMigrationToEvent(msg)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func changesetEvent(msg *awsinfra_tpb.ChangeSetStatusChangedMessage, err error) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	if err != nil {
		return nil, err
	}

	if msg == nil {
		return nil, fmt.Errorf("changesetEvent: msg and error are nil")
	}

	if msg.Request == nil {
		return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
	}

	event, err := service.ChangeSetStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	return event, nil
}

func (cf *InfraAdapter) HandleMessage(ctx context.Context, msg proto.Message) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	log.WithField(ctx, "infraReq", msg.ProtoReflect().Descriptor().FullName()).Debug("InfraHandleMessage")
	switch msg := msg.(type) {
	case *awsinfra_tpb.UpdateStackMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return stackEvent(cf.UpdateStack(ctx, msg))

	case *awsinfra_tpb.CreateNewStackMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return stackEvent(cf.CreateNewStack(ctx, msg))

	case *awsinfra_tpb.CreateChangeSetMessage:
		return changesetEvent(cf.CreateChangeSet(ctx, msg))

	case *awsinfra_tpb.ScaleStackMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return stackEvent(cf.ScaleStack(ctx, msg))

	case *awsinfra_tpb.StabalizeStackMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return stackEvent(cf.StabalizeStack(ctx, msg))

	case *awsinfra_tpb.UpsertPostgresDatabaseMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return dbEvent(cf.UpsertPostgresDatabase(ctx, msg))

	case *awsinfra_tpb.RunECSTaskMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return ecsEvent(cf.ecsClient.RunECSTask(ctx, msg))

	case *awsinfra_tpb.CleanupPostgresDatabaseMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return dbEvent(cf.CleanupPostgresDatabase(ctx, msg))

	case *awsinfra_tpb.DestroyPostgresDatabaseMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return dbEvent(cf.DestroyPostgresDatabase(ctx, msg))

	}

	return nil, fmt.Errorf("unknown side effect message type: %T", msg)
}

func (cf *InfraAdapter) CreateNewStack(ctx context.Context, msg *awsinfra_tpb.CreateNewStackMessage) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	reqToken := newToken()

	err := cf.cfClient.CreateNewStack(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, msg.Spec.StackName, reqToken, msg.Request)
}

func (cf *InfraAdapter) StabalizeStack(ctx context.Context, msg *awsinfra_tpb.StabalizeStackMessage) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		return &awsinfra_tpb.StackStatusChangedMessage{
			Request:   msg.Request,
			Status:    "MISSING",
			Lifecycle: awsdeployer_pb.CFLifecycle_MISSING,
		}, nil
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err = cf.cfClient.DeleteStack(ctx, newToken(), msg.StackName)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	needsCancel := msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress
	if needsCancel {
		reqToken := newToken()
		err = cf.cfClient.CancelStackUpdate(ctx, reqToken, &awsinfra_tpb.CancelStackUpdateMessage{
			Request:   msg.Request,
			StackName: msg.StackName,
		})
		if err != nil {
			return nil, err
		}
		return cf.pollStack(ctx, msg.StackName, reqToken, msg.Request)
	}

	lifecycle := remoteStack.SummaryType

	// Special cases for Stabalize only
	switch remoteStack.StackStatus {
	case types.StackStatusUpdateRollbackComplete:
		// When a previous attempt has failed, the stack will be in this state
		// In the Stabalize handler ONLY, this counts as a success, as the stack
		// is stable and ready for another attempt
		lifecycle = awsdeployer_pb.CFLifecycle_COMPLETE
		return &awsinfra_tpb.StackStatusChangedMessage{
			Request:   msg.Request,
			Status:    string(remoteStack.StackStatus),
			Lifecycle: lifecycle,
			Outputs:   remoteStack.Outputs,
		}, nil

	case types.StackStatusRollbackInProgress:
		// Short exit: Further events will be emitted during the rollback
		return cf.pollStack(ctx, msg.StackName, "", msg.Request)
	}

	log.WithFields(ctx, map[string]any{
		"stackName":   msg.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	return cf.pollStack(ctx, msg.StackName, "", msg.Request)
}

func (cf *InfraAdapter) UpdateStack(ctx context.Context, msg *awsinfra_tpb.UpdateStackMessage) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	reqToken := newToken()

	err := cf.cfClient.UpdateStack(ctx, reqToken, msg)
	if err != nil {
		if aws_cf.IsNoUpdatesError(err) {
			return cf.noUpdatesToBePerformed(ctx, msg.Spec.StackName, msg.Request)
		}

		return nil, err
	}

	return cf.pollStack(ctx, msg.Spec.StackName, reqToken, msg.Request)
}

func (cf *InfraAdapter) ScaleStack(ctx context.Context, msg *awsinfra_tpb.ScaleStackMessage) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	reqToken := newToken()

	err := cf.cfClient.ScaleStack(ctx, reqToken, msg)
	if err != nil {
		if aws_cf.IsNoUpdatesError(err) {
			return cf.noUpdatesToBePerformed(ctx, msg.StackName, msg.Request)
		}
		return nil, err
	}

	return cf.pollStack(ctx, msg.StackName, reqToken, msg.Request)
}

func (cf *InfraAdapter) CreateChangeSet(ctx context.Context, msg *awsinfra_tpb.CreateChangeSetMessage) (*awsinfra_tpb.ChangeSetStatusChangedMessage, error) {

	reqToken := newToken()

	err := cf.cfClient.CreateChangeSet(ctx, reqToken, msg)
	if err != nil {
		if aws_cf.IsNoUpdatesError(err) {
			return &awsinfra_tpb.ChangeSetStatusChangedMessage{
				Request: msg.Request,
				Status:  "NO UPDATES TO BE PERFORMED",
			}, nil
		}
		return nil, err
	}

	return cf.pollChangeSet(ctx, msg.Spec.StackName, reqToken, msg.Request)
}

type pgRequest interface {
	GetRequest() *messaging_j5pb.RequestMetadata
	GetMigrationId() string
}

func (cf *InfraAdapter) runPostgresCallback(ctx context.Context, msg pgRequest, cb func(context.Context) error) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {

	migrateErr := cb(ctx)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		errMsg := migrateErr.Error()
		return &awsinfra_tpb.PostgresDatabaseStatusMessage{
			Request:     msg.GetRequest(),
			MigrationId: msg.GetMigrationId(),
			Status:      awsinfra_tpb.PostgresStatus_ERROR,
			Error:       &errMsg,
		}, nil
	}

	return &awsinfra_tpb.PostgresDatabaseStatusMessage{
		Request:     msg.GetRequest(),
		MigrationId: msg.GetMigrationId(),
		Status:      awsinfra_tpb.PostgresStatus_DONE,
	}, nil
}
func (cf *InfraAdapter) UpsertPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.UpsertPostgresDatabaseMessage) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {
	return cf.runPostgresCallback(ctx, msg, func(ctx context.Context) error {
		return cf.dbClient.UpsertPostgresDatabase(ctx, msg.MigrationId, msg)
	})
}

func (cf *InfraAdapter) CleanupPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.CleanupPostgresDatabaseMessage) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {
	return cf.runPostgresCallback(ctx, msg, func(ctx context.Context) error {
		return cf.dbClient.CleanupPostgresDatabase(ctx, msg.MigrationId, msg)
	})
}

func (cf *InfraAdapter) DestroyPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.DestroyPostgresDatabaseMessage) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {
	return cf.runPostgresCallback(ctx, msg, func(ctx context.Context) error {
		return cf.dbClient.DestroyPostgresDatabase(ctx, msg.MigrationId, msg)
	})
}

func (cf *InfraAdapter) noUpdatesToBePerformed(ctx context.Context, stackName string, request *messaging_j5pb.RequestMetadata) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	return &awsinfra_tpb.StackStatusChangedMessage{
		Request:   request,
		StackName: stackName,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   remoteStack.Outputs,
		Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
	}, nil
}

func (cf *InfraAdapter) pollStack(
	ctx context.Context,
	stackName string,
	_ string,
	request *messaging_j5pb.RequestMetadata,
) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	beginTime := time.Now()

	ctx = log.WithFields(ctx, map[string]any{
		"stackName": stackName,
	})

	log.Debug(ctx, "PollStack Begin")

	lastEvent := time.Now().Add(time.Second * -1)

	for {
		remoteStack, err := cf.cfClient.GetOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		if remoteStack == nil {
			return nil, fmt.Errorf("missing stack %s", stackName)
		}

		events, err := cf.cfClient.Logs(ctx, stackName)
		if err == nil {
			for _, event := range events {
				if event.Timestamp.After(lastEvent) {
					lastEvent = event.Timestamp
					args := map[string]any{
						"timestamp": event.Timestamp,
						"resource":  event.Resource,
						"status":    event.Status,
					}

					if event.Detail != "" {
						args["details"] = event.Detail
					}

					if event.IsFailure {
						log.WithFields(ctx, args).Error("CF Stack Event")
					} else {
						log.WithFields(ctx, args).Info("CF Stack Event")
					}
				}
			}
		}

		if !remoteStack.Stable {
			log.WithFields(ctx, map[string]any{
				"lifecycle":   remoteStack.SummaryType.ShortString(),
				"stackStatus": remoteStack.StackStatus,
			}).Debug("PollStack Intermediate Result")
			time.Sleep(1 * time.Second)
			continue
		}

		log.WithFields(ctx, map[string]any{
			"lifecycle":   remoteStack.SummaryType.ShortString(),
			"stackStatus": remoteStack.StackStatus,
			"outputs":     remoteStack.Outputs,
			"duration":    time.Since(beginTime).String(),
		}).Info("PollStack Final Result")
		return &awsinfra_tpb.StackStatusChangedMessage{
			Request:   request,
			StackName: stackName,
			Status:    string(remoteStack.StackStatus),
			Lifecycle: remoteStack.SummaryType,
			Outputs:   remoteStack.Outputs,
		}, nil

	}
}

func (cf *InfraAdapter) pollChangeSet(
	ctx context.Context,
	stackName string,
	reqToken string,
	request *messaging_j5pb.RequestMetadata,
) (*awsinfra_tpb.ChangeSetStatusChangedMessage, error) {

	changeSetID := fmt.Sprintf("%s-%s", stackName, reqToken)

	ctx = log.WithFields(ctx, map[string]any{
		"stackName":   stackName,
		"changeSetID": changeSetID,
	})
	log.Debug(ctx, "PollChangeSet Begin")

	for {
		remoteChangeSet, err := cf.cfClient.GetChangeSet(ctx, stackName, changeSetID)
		if err != nil {
			return nil, err
		}

		if remoteChangeSet == nil {
			return nil, fmt.Errorf("missing changeset %s", changeSetID)
		}

		if remoteChangeSet.Lifecycle == awsdeployer_pb.CFChangesetLifecycle_AVAILABLE {
			return &awsinfra_tpb.ChangeSetStatusChangedMessage{
				Request:       request,
				StackName:     stackName,
				Status:        remoteChangeSet.Status,
				Lifecycle:     remoteChangeSet.Lifecycle,
				EventId:       uuid.New().String(),
				ChangeSetName: changeSetID,
			}, nil
		}

		log.WithFields(ctx, map[string]any{
			"lifecycle": remoteChangeSet.Lifecycle.ShortString(),
			"status":    remoteChangeSet.Status,
		}).Debug("PollChangeSet Intermediate Result")
		time.Sleep(1 * time.Second)

	}
}
