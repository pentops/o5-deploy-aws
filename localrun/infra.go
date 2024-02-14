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
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
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

func (cf *InfraAdapter) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*deployer_pb.DeploymentEventType_StackStatus, error) {

	reqToken := newToken()

	err := cf.cfClient.CreateNewStack(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return cf.pollStack(ctx, msg.StackName, reqToken, msg.Request)
}

func (cf *InfraAdapter) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*deployer_pb.DeploymentEventType_StackStatus, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		return &deployer_pb.DeploymentEventType_StackStatus{
			FullStatus: "MISSING",
			Lifecycle:  deployer_pb.StackLifecycle_MISSING,
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
		return &deployer_pb.DeploymentEventType_StackStatus{
			FullStatus:  string(remoteStack.StackStatus),
			Lifecycle:   lifecycle,
			StackOutput: remoteStack.Outputs,
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

func (cf *InfraAdapter) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*deployer_pb.DeploymentEventType_StackStatus, error) {

	reqToken := newToken()

	err := cf.cfClient.UpdateStack(ctx, reqToken, msg)
	if err != nil {
		if awsinfra.IsNoUpdatesError(err) {
			return cf.noUpdatesToBePerformed(ctx, msg.StackName, msg.Request)
		}

		return nil, err
	}

	return cf.pollStack(ctx, msg.StackName, reqToken, msg.Request)
}

func (cf *InfraAdapter) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*deployer_pb.DeploymentEventType_StackStatus, error) {

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

func (cf *InfraAdapter) MigratePostgresDatabase(ctx context.Context, msg *deployer_tpb.MigratePostgresDatabaseMessage) (*deployer_pb.DeploymentEventType_DBMigrateStatus, error) {

	migrateError := cf.dbClient.MigratePostgresDatabase(ctx, msg)

	if migrateError != nil {
		errMsg := migrateError.Error()
		return &deployer_pb.DeploymentEventType_DBMigrateStatus{
			Status: deployer_pb.DatabaseMigrationStatus_FAILED,
			Error:  &errMsg,
		}, nil
	}
	return &deployer_pb.DeploymentEventType_DBMigrateStatus{
		Status: deployer_pb.DatabaseMigrationStatus_COMPLETED,
	}, nil

}

func (cf *InfraAdapter) noUpdatesToBePerformed(ctx context.Context, stackName string, request *messaging_pb.RequestMetadata) (*deployer_pb.DeploymentEventType_StackStatus, error) {

	remoteStack, err := cf.cfClient.GetOneStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	return &deployer_pb.DeploymentEventType_StackStatus{
		FullStatus:  "NO UPDATES TO BE PERFORMED",
		StackOutput: remoteStack.Outputs,
		Lifecycle:   deployer_pb.StackLifecycle_COMPLETE,
	}, nil
}

func (cf *InfraAdapter) pollStack(
	ctx context.Context,
	stackName string,
	reqToken string,
	request *messaging_pb.RequestMetadata,
) (*deployer_pb.DeploymentEventType_StackStatus, error) {

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
		return &deployer_pb.DeploymentEventType_StackStatus{
			Lifecycle:   remoteStack.SummaryType,
			FullStatus:  string(remoteStack.StackStatus),
			StackOutput: remoteStack.Outputs,
		}, nil

	}

}
