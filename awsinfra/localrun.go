package awsinfra

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/proto"
)

type LocalRunner struct {
	*CFClient
}

func NewLocalRunner(clients ClientBuilder) *LocalRunner {
	cfClient := &CFClient{
		Clients: clients,
	}
	return &LocalRunner{
		CFClient: cfClient,
	}
}

func (cf *LocalRunner) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {
	err := cf.CFClient.CreateNewStack(ctx, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, "", msg.StackId)
}

func (cf *LocalRunner) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	remoteStack, err := cf.getOneStack(ctx, msg.StackId.StackName)
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

	lifecycle, err := stackLifecycle(remoteStack.StackStatus)
	if err != nil {
		return nil, err
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err = cf.CFClient.DeleteStack(ctx, &deployer_tpb.DeleteStackMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	needsCancel := msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress
	if needsCancel {
		err = cf.CFClient.CancelStackUpdate(ctx, &deployer_tpb.CancelStackUpdateMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}
		return cf.pollStack(ctx, "", msg.StackId)
	}

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
		}, nil

	case types.StackStatusRollbackInProgress:
		// Short exit: Further events will be emitted during the rollback
		return cf.pollStack(ctx, "", msg.StackId)
	}

	log.WithFields(ctx, map[string]interface{}{
		"stackName":   msg.StackId.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	return cf.pollStack(ctx, "", msg.StackId)
}

func (cf *LocalRunner) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	err := cf.CFClient.UpdateStack(ctx, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, "", msg.StackId)
}

func (cf *LocalRunner) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*deployer_tpb.StackStatusChangedMessage, error) {

	err := cf.CFClient.ScaleStack(ctx, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, "", msg.StackId)
}

func (cf *LocalRunner) RunDatabaseMigration(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) (*deployer_tpb.MigrationStatusChangedMessage, error) {

	migrator := &DBMigrator{
		Clients: cf.Clients,
	}
	migrateErr := migrator.runDatabaseMigration(ctx, msg)

	if migrateErr != nil {
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
