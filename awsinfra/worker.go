package awsinfra

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MessageHandler interface {
	PublishEvent(context.Context, outbox.OutboxMessage) error
}

type InfraWorker struct {
	*deployer_tpb.UnimplementedAWSCommandTopicServer
	*messaging_tpb.UnimplementedRawMessageTopicServer

	messageHandler MessageHandler
	*CFClient
}

func NewInfraWorker(clients ClientBuilder, messageHandler MessageHandler) *InfraWorker {
	cfClient := &CFClient{
		Clients: clients,
	}
	return &InfraWorker{
		CFClient:       cfClient,
		messageHandler: messageHandler,
	}
}

func (cf *InfraWorker) eventOut(ctx context.Context, msg outbox.OutboxMessage) error {
	fmt.Printf("eventOut: %s\n", protojson.Format(msg))
	return cf.messageHandler.PublishEvent(ctx, msg)
}

func parseAWSRawMessage(raw []byte) (map[string]string, error) {

	lines := strings.Split(string(raw), "\n")
	fields := map[string]string{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		s := parts[1]
		if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
			s = s[1 : len(s)-1]
		}
		fields[parts[0]] = s
	}
	return fields, nil
}

func (cf *InfraWorker) Raw(ctx context.Context, msg *messaging_tpb.RawMessage) (*emptypb.Empty, error) {

	fields, err := parseAWSRawMessage(msg.Payload)
	if err != nil {
		return nil, err
	}

	resourceType, ok := fields["ResourceType"]
	if !ok {
		return nil, fmt.Errorf("missing ResourceType")
	}

	if resourceType != "AWS::CloudFormation::Stack" {
		return &emptypb.Empty{}, nil
	}

	stackName, ok := fields["StackName"]
	if !ok {
		return nil, fmt.Errorf("missing StackName")
	}

	clientToken, ok := fields["ClientRequestToken"]
	if !ok {
		return nil, fmt.Errorf("missing ClientRequestToken")
	}

	resourceStatus, ok := fields["ResourceStatus"]
	if !ok {
		return nil, fmt.Errorf("missing ResourceStatus")
	}

	lifecycle, err := stackLifecycle(types.StackStatus(resourceStatus))
	if err != nil {
		return nil, err
	}

	var outputs []*deployer_pb.KeyValue

	if lifecycle == deployer_pb.StackLifecycle_COMPLETE {

		stack, err := cf.getOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}

		if stack == nil {
			return nil, fmt.Errorf("missing stack %s", stackName)
		}

		outputs = mapOutputs(stack.Outputs)
	}

	stackID, err := parseStackID(stackName, clientToken)
	if err != nil {
		return nil, err
	}

	if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackId:   stackID,
		Status:    resourceStatus,
		Outputs:   outputs,
		Lifecycle: lifecycle,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*emptypb.Empty, error) {

	err := cf.CFClient.CreateNewStack(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*emptypb.Empty, error) {

	err := cf.CFClient.UpdateStack(ctx, msg)
	if err != nil && !IsNoUpdatesError(err) {
		return nil, fmt.Errorf("UpdateStack: %w", err)
	}

	if err := cf.noUpdatesToBePerformed(ctx, msg.StackId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) CancelStackUpdate(ctx context.Context, msg *deployer_tpb.CancelStackUpdateMessage) (*emptypb.Empty, error) {
	err := cf.CFClient.CancelStackUpdate(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) DeleteStack(ctx context.Context, msg *deployer_tpb.DeleteStackMessage) (*emptypb.Empty, error) {
	err := cf.CFClient.DeleteStack(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*emptypb.Empty, error) {
	err := cf.CFClient.ScaleStack(ctx, msg)
	if err != nil && !IsNoUpdatesError(err) {
		return nil, fmt.Errorf("ScaleStack: %w", err)
	}

	if err := cf.noUpdatesToBePerformed(ctx, msg.StackId); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*emptypb.Empty, error) {

	remoteStack, err := cf.getOneStack(ctx, msg.StackId.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   msg.StackId,
			Status:    "MISSING",
			Lifecycle: deployer_pb.StackLifecycle_MISSING,
		})
		if err != nil {
			return nil, fmt.Errorf("eventOut: %w", err)
		}

		return &emptypb.Empty{}, nil
	}

	lifecycle, err := stackLifecycle(remoteStack.StackStatus)
	if err != nil {
		return nil, err
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err := cf.eventOut(ctx, &deployer_tpb.DeleteStackMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}

	needsCancel := msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress
	if needsCancel {
		err := cf.eventOut(ctx, &deployer_tpb.CancelStackUpdateMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}
	}

	// Special cases for Stabalize only
	switch remoteStack.StackStatus {
	case types.StackStatusUpdateRollbackComplete:
		// When a previous attempt has failed, the stack will be in this state
		// In the Stabalize handler ONLY, this counts as a success, as the stack
		// is stable and ready for another attempt
		lifecycle = deployer_pb.StackLifecycle_COMPLETE
		err = cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   msg.StackId,
			Status:    string(remoteStack.StackStatus),
			Lifecycle: lifecycle,
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil

	case types.StackStatusRollbackInProgress:
		// Short exit: Further events will be emitted during the rollback
		return &emptypb.Empty{}, nil
	}

	log.WithFields(ctx, map[string]interface{}{
		"stackName":   msg.StackId.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	// This only runs if we aren't running the poller, as the poller will do
	// the same thing
	err = cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackId:   msg.StackId,
		Status:    string(remoteStack.StackStatus),
		Lifecycle: lifecycle,
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// sends a fake status update message back to the deployer so that it thinks
// something has happened and continues the event chain
func (cf *InfraWorker) noUpdatesToBePerformed(ctx context.Context, stackID *deployer_tpb.StackID) error {

	remoteStack, err := cf.getOneStack(ctx, stackID.StackName)
	if err != nil {
		return err
	}

	summary, err := summarizeStackStatus(remoteStack)
	if err != nil {
		return err
	}

	if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackId:   stackID,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   summary.Outputs,
		Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
	}); err != nil {
		return err
	}

	return nil
}

func (d *InfraWorker) RunDatabaseMigration(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) (*emptypb.Empty, error) {
	if err := d.messageHandler.PublishEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
		MigrationId:  msg.MigrationId,
		DeploymentId: msg.DeploymentId,
		Status:       deployer_pb.DatabaseMigrationStatus_PENDING,
	}); err != nil {
		return nil, err
	}

	migrator := &DBMigrator{
		Clients: d.Clients,
	}
	migrateErr := migrator.RunDatabaseMigration(ctx, msg)

	if migrateErr != nil {
		log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
		if err := d.messageHandler.PublishEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
			MigrationId:  msg.MigrationId,
			DeploymentId: msg.DeploymentId,
			Status:       deployer_pb.DatabaseMigrationStatus_FAILED,
			Error:        proto.String(migrateErr.Error()),
		}); err != nil {
			return nil, err
		}
		return &empty.Empty{}, nil
	}

	if err := d.messageHandler.PublishEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
		DeploymentId: msg.DeploymentId,
		MigrationId:  msg.MigrationId,
		Status:       deployer_pb.DatabaseMigrationStatus_COMPLETED,
	}); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}
