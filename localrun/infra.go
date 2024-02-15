package localrun

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"google.golang.org/protobuf/proto"
)

type InfraAdapter struct {
	cfClient *awsinfra.CFClient
	dbClient *awsinfra.DBMigrator
}

func NewInfraAdapter(clients awsinfra.ClientBuilder) *InfraAdapter {
	cfClient := &awsinfra.CFClient{
		Clients: clients,
	}
	dbMigrator := &awsinfra.DBMigrator{
		Clients: clients,
	}
	return &InfraAdapter{
		cfClient: cfClient,
		dbClient: dbMigrator,
	}
}

func newToken() string {
	return fmt.Sprintf("local-%d", time.Now().UnixNano())
}

func stackEvent(msg *deployer_tpb.StackStatusChangedMessage, err error) (deployer_pb.DeploymentPSMEvent, error) {
	if err != nil {
		return nil, err
	}
	event, err := deployer.StackStatusToEvent(msg)
	if err != nil {
		return nil, err
	}
	return event.UnwrapPSMEvent(), nil
}

func dbEvent(msg *deployer_tpb.PostgresMigrationEventMessage, err error) (deployer_pb.DeploymentPSMEvent, error) {
	if err != nil {
		return nil, err
	}
	event, err := deployer.PostgresMigrationToEvent(msg)
	if err != nil {
		return nil, err
	}
	return event.UnwrapPSMEvent(), nil
}

func (cf *InfraAdapter) HandleMessage(ctx context.Context, msg proto.Message) (deployer_pb.DeploymentPSMEvent, error) {
	switch msg := msg.(type) {
	case *deployer_tpb.UpdateStackMessage:
		return stackEvent(cf.UpdateStack(ctx, msg))

	case *deployer_tpb.CreateNewStackMessage:
		return stackEvent(cf.CreateNewStack(ctx, msg))

	case *deployer_tpb.ScaleStackMessage:
		return stackEvent(cf.ScaleStack(ctx, msg))

	case *deployer_tpb.StabalizeStackMessage:
		return stackEvent(cf.StabalizeStack(ctx, msg))

	case *deployer_tpb.MigratePostgresDatabaseMessage:
		return dbEvent(cf.MigratePostgresDatabase(ctx, msg))

	case *deployer_tpb.DeploymentCompleteMessage:
		return nil, nil
	}

	return nil, fmt.Errorf("unknown side effect message type: %T", msg)
}

func (cf *InfraAdapter) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	reqToken := newToken()

	err := cf.cfClient.CreateNewStack(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, msg.Spec.StackName, reqToken, msg.Request)
}

func (cf *InfraAdapter) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		return &deployer_tpb.StackStatusChangedMessage{
			Status:    "MISSING",
			Lifecycle: deployer_pb.StackLifecycle_MISSING,
		}, nil
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err = cf.cfClient.DeleteStack(ctx, newToken(), &deployer_tpb.DeleteStackMessage{
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
		err = cf.cfClient.CancelStackUpdate(ctx, reqToken, &deployer_tpb.CancelStackUpdateMessage{
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
		lifecycle = deployer_pb.StackLifecycle_COMPLETE
		return &deployer_tpb.StackStatusChangedMessage{
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

func (cf *InfraAdapter) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

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

func (cf *InfraAdapter) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

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

func (cf *InfraAdapter) MigratePostgresDatabase(ctx context.Context, msg *deployer_tpb.MigratePostgresDatabaseMessage) (*deployer_tpb.PostgresMigrationEventMessage, error) {

	replyMsg := &deployer_tpb.PostgresMigrationEventMessage{
		Request:     msg.Request,
		MigrationId: msg.MigrationId,
	}

	migrateError := cf.dbClient.MigratePostgresDatabase(ctx, msg)

	reqState := &deployer_pb.StepContext{}
	if err := proto.Unmarshal(msg.Request.Context, reqState); err != nil {
		return nil, err
	}

	if migrateError != nil {
		errMsg := migrateError.Error()
		replyMsg.Status = deployer_tpb.PostgresStatus_ERROR
		replyMsg.Error = &errMsg
	} else {
		replyMsg.Status = deployer_tpb.PostgresStatus_AVAILABLE
	}
	return replyMsg, nil
}

func (cf *InfraAdapter) noUpdatesToBePerformed(ctx context.Context, stackName string, request *messaging_pb.RequestMetadata) (*deployer_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	return &deployer_tpb.StackStatusChangedMessage{
		Request:   request,
		StackName: stackName,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   remoteStack.Outputs,
		Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
	}, nil
}

func (cf *InfraAdapter) pollStack(
	ctx context.Context,
	stackName string,
	reqToken string,
	request *messaging_pb.RequestMetadata,
) (*deployer_tpb.StackStatusChangedMessage, error) {

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
		return &deployer_tpb.StackStatusChangedMessage{
			Request:   request,
			StackName: stackName,
			Status:    string(remoteStack.StackStatus),
			Lifecycle: remoteStack.SummaryType,
			Outputs:   remoteStack.Outputs,
		}, nil

	}

}
