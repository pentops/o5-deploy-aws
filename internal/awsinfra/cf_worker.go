package awsinfra

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	"github.com/pentops/o5-messaging/o5msg"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DBLite interface {
	PublishEvent(context.Context, o5msg.Message) error
	RequestToClientToken(context.Context, *messaging_pb.RequestMetadata) (string, error)
	ClientTokenToRequest(context.Context, string) (*messaging_pb.RequestMetadata, error)
}

var RequestTokenNotFound = errors.New("request token not found")

type InfraWorker struct {
	awsinfra_tpb.UnimplementedCloudFormationRequestTopicServer

	db DBLite
	*CFClient
}

func NewInfraWorker(db DBLite, adapter *CFClient) *InfraWorker {
	return &InfraWorker{
		CFClient: adapter,
		db:       db,
	}
}

func (cf *InfraWorker) eventOut(ctx context.Context, msg o5msg.Message) error {
	return cf.db.PublishEvent(ctx, msg)
}

func (cf *InfraWorker) HandleCloudFormationEvent(ctx context.Context, fields map[string]string) error {

	eventID, ok := fields["EventId"]
	if !ok {
		return fmt.Errorf("missing EventId")
	}

	/*
		timestamp, ok := fields["Timestamp"]
		if !ok {
			return fmt.Errorf("missing Timestamp")
		}*/

	resourceType, ok := fields["ResourceType"]
	if !ok {
		return fmt.Errorf("missing ResourceType")
	}

	if resourceType != "AWS::CloudFormation::Stack" {
		return nil
	}

	stackName, ok := fields["StackName"]
	if !ok {
		return fmt.Errorf("missing StackName")
	}

	clientToken, ok := fields["ClientRequestToken"]
	if !ok {
		return fmt.Errorf("missing ClientRequestToken")
	}

	resourceStatus, ok := fields["ResourceStatus"]
	if !ok {
		return fmt.Errorf("missing ResourceStatus")
	}

	lifecycle, err := stackLifecycle(types.StackStatus(resourceStatus))
	if err != nil {
		return err
	}

	var outputs []*awsdeployer_pb.KeyValue

	if lifecycle == awsdeployer_pb.CFLifecycle_COMPLETE {

		stack, err := cf.getOneStack(ctx, stackName)
		if err != nil {
			return err
		}

		if stack == nil {
			return fmt.Errorf("missing stack %s", stackName)
		}

		outputs = mapOutputs(stack.Outputs)
	}

	requestMetadata, err := cf.db.ClientTokenToRequest(ctx, clientToken)
	if errors.Is(err, RequestTokenNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	if err := cf.eventOut(ctx, &awsinfra_tpb.StackStatusChangedMessage{
		Request:   requestMetadata,
		EventId:   eventID,
		StackName: stackName,
		Status:    resourceStatus,
		Outputs:   outputs,
		Lifecycle: lifecycle,
	}); err != nil {
		return err
	}

	return nil
}

func (cf *InfraWorker) CreateNewStack(ctx context.Context, msg *awsinfra_tpb.CreateNewStackMessage) (*emptypb.Empty, error) {

	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	err = cf.CFClient.CreateNewStack(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) UpdateStack(ctx context.Context, msg *awsinfra_tpb.UpdateStackMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	err = cf.CFClient.UpdateStack(ctx, reqToken, msg)
	if err != nil {
		if !IsNoUpdatesError(err) {
			return nil, fmt.Errorf("UpdateStack: %w", err)
		}

		if err := cf.noUpdatesToBePerformed(ctx, msg.Spec.StackName, msg.Request, reqToken); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) CreateChangeSet(ctx context.Context, msg *awsinfra_tpb.CreateChangeSetMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	err = cf.CFClient.CreateChangeSet(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) CancelStackUpdate(ctx context.Context, msg *awsinfra_tpb.CancelStackUpdateMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}
	err = cf.CFClient.CancelStackUpdate(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) DeleteStack(ctx context.Context, msg *awsinfra_tpb.DeleteStackMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}
	err = cf.CFClient.DeleteStack(ctx, reqToken, msg)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) ScaleStack(ctx context.Context, msg *awsinfra_tpb.ScaleStackMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}
	err = cf.CFClient.ScaleStack(ctx, reqToken, msg)
	if err != nil {
		if !IsNoUpdatesError(err) {
			return nil, fmt.Errorf("ScaleStack: %w", err)
		}

		if err := cf.noUpdatesToBePerformed(ctx, msg.StackName, msg.Request, reqToken); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) StabalizeStack(ctx context.Context, msg *awsinfra_tpb.StabalizeStackMessage) (*emptypb.Empty, error) {

	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	remoteStack, err := cf.getOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		err := cf.eventOut(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			Request:   msg.Request,
			EventId:   reqToken,
			StackName: msg.StackName,
			Status:    "MISSING",
			Lifecycle: awsdeployer_pb.CFLifecycle_MISSING,
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
		err := cf.eventOut(ctx, &awsinfra_tpb.DeleteStackMessage{
			Request:   msg.Request,
			StackName: msg.StackName,
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}

	needsCancel := msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress
	if needsCancel {
		err := cf.eventOut(ctx, &awsinfra_tpb.CancelStackUpdateMessage{
			Request:   msg.Request,
			StackName: msg.StackName,
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
		lifecycle = awsdeployer_pb.CFLifecycle_COMPLETE
		err = cf.eventOut(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			Request:   msg.Request,
			EventId:   reqToken,
			StackName: msg.StackName,
			Status:    string(remoteStack.StackStatus),
			Lifecycle: lifecycle,
			Outputs:   mapOutputs(remoteStack.Outputs),
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
		"stackName":   msg.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	err = cf.eventOut(ctx, &awsinfra_tpb.StackStatusChangedMessage{
		Request:   msg.Request,
		EventId:   reqToken,
		StackName: msg.StackName,
		Status:    string(remoteStack.StackStatus),
		Lifecycle: lifecycle,
		Outputs:   mapOutputs(remoteStack.Outputs),
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// sends a fake status update message back to the deployer so that it thinks
// something has happened and continues the event chain
func (cf *InfraWorker) noUpdatesToBePerformed(ctx context.Context, stackName string, request *messaging_pb.RequestMetadata, eventID string) error {

	remoteStack, err := cf.getOneStack(ctx, stackName)
	if err != nil {
		return err
	}

	summary, err := summarizeStackStatus(remoteStack)
	if err != nil {
		return err
	}

	if err := cf.eventOut(ctx, &awsinfra_tpb.StackStatusChangedMessage{
		Request:   request,
		EventId:   eventID,
		StackName: stackName,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   summary.Outputs,
		Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
	}); err != nil {
		return err
	}

	return nil
}
