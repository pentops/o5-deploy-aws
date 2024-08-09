package localrun

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/awsinfra"
	"github.com/pentops/o5-deploy-aws/internal/service"
	"google.golang.org/protobuf/proto"
)

type InfraAdapter struct {
	cfClient  *awsinfra.CFClient
	dbClient  *awsinfra.DBMigrator
	ecsClient *ecsRunner
}

func NewInfraAdapter(ctx context.Context, cl *awsinfra.DeployerClients) (*InfraAdapter, error) {
	cfClient := awsinfra.NewCFAdapter(cl, []string{})
	dbMigrator := awsinfra.NewDBMigrator(cl.SecretsManager)
	ecsClient := &ecsRunner{
		ecsClient: cl.ECS,
	}

	return &InfraAdapter{
		cfClient:  cfClient,
		dbClient:  dbMigrator,
		ecsClient: ecsClient,
	}, nil
}

func NewInfraAdapterFromConfig(ctx context.Context, config aws.Config) (*InfraAdapter, error) {
	cfClient, err := awsinfra.NewCFAdapterFromConfig(ctx, config, []string{})
	if err != nil {
		return nil, err
	}

	dbMigrator := awsinfra.NewDBMigrator(secretsmanager.NewFromConfig(config))
	ecsClient := &ecsRunner{
		ecsClient: ecs.NewFromConfig(config),
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

func dbEvent(msg *awsinfra_tpb.PostgresDatabaseStatusMessage, err error) (*awsdeployer_pb.DeploymentPSMEventSpec, error) {
	if err != nil {
		return nil, err
	}
	if msg.Request == nil {
		return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
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

	case *awsinfra_tpb.MigratePostgresDatabaseMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return dbEvent(cf.MigratePostgresDatabase(ctx, msg))

	case *awsinfra_tpb.CleanupPostgresDatabaseMessage:
		if msg.Request == nil {
			return nil, fmt.Errorf("missing request in %s", msg.ProtoReflect().Descriptor().FullName())
		}
		return dbEvent(cf.CleanupPostgresDatabase(ctx, msg))

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
		err = cf.cfClient.DeleteStack(ctx, newToken(), &awsinfra_tpb.DeleteStackMessage{
			Request:   msg.Request,
			StackName: msg.StackName,
		})
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

	log.WithFields(ctx, map[string]interface{}{
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
		if awsinfra.IsNoUpdatesError(err) {
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
		if awsinfra.IsNoUpdatesError(err) {
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
		if awsinfra.IsNoUpdatesError(err) {
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

func (cf *InfraAdapter) MigratePostgresDatabase(ctx context.Context, msg *awsinfra_tpb.MigratePostgresDatabaseMessage) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {

	return cf.runPostgresCallback(ctx, msg, func(ctx context.Context) error {
		return cf.ecsClient.runMigrationTask(ctx, msg.MigrationId, msg.Spec)
	})
}

func (cf *InfraAdapter) UpsertPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.UpsertPostgresDatabaseMessage) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {
	return cf.runPostgresCallback(ctx, msg, func(ctx context.Context) error {
		return cf.dbClient.UpsertPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
	})
}

func (cf *InfraAdapter) CleanupPostgresDatabase(ctx context.Context, msg *awsinfra_tpb.CleanupPostgresDatabaseMessage) (*awsinfra_tpb.PostgresDatabaseStatusMessage, error) {
	return cf.runPostgresCallback(ctx, msg, func(ctx context.Context) error {
		return cf.dbClient.CleanupPostgresDatabase(ctx, msg.MigrationId, msg.Spec)
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

func (cf *InfraAdapter) PollStack(
	ctx context.Context,
	stackName string,
) (*awsinfra_tpb.StackStatusChangedMessage, error) {
	return cf.pollStack(ctx, stackName, "", nil)
}

func (cf *InfraAdapter) pollStack(
	ctx context.Context,
	stackName string,
	reqToken string,
	request *messaging_j5pb.RequestMetadata,
) (*awsinfra_tpb.StackStatusChangedMessage, error) {

	beginTime := time.Now()

	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackName": stackName,
	})

	log.Debug(ctx, "PollStack Begin")

	for {
		remoteStack, err := cf.cfClient.GetOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		if remoteStack == nil {
			return nil, fmt.Errorf("missing stack %s", stackName)
		}

		if !remoteStack.Stable {
			log.WithFields(ctx, map[string]interface{}{
				"lifecycle":   remoteStack.SummaryType.ShortString(),
				"stackStatus": remoteStack.StackStatus,
			}).Debug("PollStack Intermediate Result")
			time.Sleep(1 * time.Second)
			continue
		}

		log.WithFields(ctx, map[string]interface{}{
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

	ctx = log.WithFields(ctx, map[string]interface{}{
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

		log.WithFields(ctx, map[string]interface{}{
			"lifecycle": remoteChangeSet.Lifecycle.ShortString(),
			"status":    remoteChangeSet.Status,
		}).Debug("PollChangeSet Intermediate Result")
		time.Sleep(1 * time.Second)

	}
}
