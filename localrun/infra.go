package localrun

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
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

func (cf *InfraAdapter) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {
	err := cf.cfClient.CreateNewStack(ctx, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, msg.StackId)
}

func (cf *InfraAdapter) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, msg.StackId.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		return &deployer_tpb.StackStatusChangedMessage{
			StackId:   msg.StackId,
			Status:    "MISSING",
			Lifecycle: deployer_pb.StackLifecycle_MISSING,
		}, nil
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err = cf.cfClient.DeleteStack(ctx, &deployer_tpb.DeleteStackMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	needsCancel := msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress
	if needsCancel {
		err = cf.cfClient.CancelStackUpdate(ctx, &deployer_tpb.CancelStackUpdateMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}
		return cf.pollStack(ctx, msg.StackId)
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
			StackId:   msg.StackId,
			Status:    string(remoteStack.StackStatus),
			Lifecycle: lifecycle,
			Outputs:   remoteStack.Outputs,
		}, nil

	case types.StackStatusRollbackInProgress:
		// Short exit: Further events will be emitted during the rollback
		return cf.pollStack(ctx, msg.StackId)
	}

	log.WithFields(ctx, map[string]interface{}{
		"stackName":   msg.StackId.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	return cf.pollStack(ctx, msg.StackId)
}

func (cf *InfraAdapter) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	err := cf.cfClient.UpdateStack(ctx, msg)
	if err != nil {
		if awsinfra.IsNoUpdatesError(err) {
			return cf.noUpdatesToBePerformed(ctx, msg.StackId)
		}

		return nil, err
	}

	return cf.pollStack(ctx, msg.StackId)
}

func (cf *InfraAdapter) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	err := cf.cfClient.ScaleStack(ctx, msg)
	if err != nil {
		if awsinfra.IsNoUpdatesError(err) {
			return cf.noUpdatesToBePerformed(ctx, msg.StackId)
		}
		return nil, err
	}

	return cf.pollStack(ctx, msg.StackId)
}

func (cf *InfraAdapter) RunDatabaseMigration(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) (*deployer_tpb.MigrationStatusChangedMessage, error) {

	migrateErr := cf.dbClient.RunDatabaseMigration(ctx, msg)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("DB Migration Failed")
		return &deployer_tpb.MigrationStatusChangedMessage{
			MigrationId:  msg.MigrationId,
			DeploymentId: msg.DeploymentId,
			Status:       deployer_pb.DatabaseMigrationStatus_FAILED,
			Error:        proto.String(migrateErr.Error()),
		}, nil
	}

	return &deployer_tpb.MigrationStatusChangedMessage{
		DeploymentId: msg.DeploymentId,
		MigrationId:  msg.MigrationId,
		Status:       deployer_pb.DatabaseMigrationStatus_COMPLETED,
	}, nil

}

func (cf *InfraAdapter) noUpdatesToBePerformed(ctx context.Context, stackID *deployer_tpb.StackID) (*deployer_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, stackID.StackName)
	if err != nil {
		return nil, err
	}

	return &deployer_tpb.StackStatusChangedMessage{
		StackId:   stackID,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   remoteStack.Outputs,
		Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
	}, nil
}

func (cf *InfraAdapter) pollStack(
	ctx context.Context,
	stackID *deployer_tpb.StackID,
) (*deployer_tpb.StackStatusChangedMessage, error) {

	stackName := stackID.StackName

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
		}).Info("PollStack Final Result")

		return &deployer_tpb.StackStatusChangedMessage{
			StackId:   stackID,
			Status:    string(remoteStack.StackStatus),
			Outputs:   remoteStack.Outputs,
			Lifecycle: remoteStack.SummaryType,
		}, nil
	}

}
