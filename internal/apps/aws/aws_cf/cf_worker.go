package aws_cf

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/tokenstore"
	"github.com/pentops/o5-messaging/o5msg"
	"google.golang.org/protobuf/types/known/emptypb"
)

type InfraWorker struct {
	awsinfra_tpb.UnimplementedCloudFormationRequestTopicServer

	db tokenstore.DBLite
	*CFClient
}

func NewInfraWorker(db tokenstore.DBLite, adapter *CFClient) *InfraWorker {
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
	if errors.Is(err, tokenstore.RequestTokenNotFound) {
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

// StabalizeStack is sent to check that the stack is ready for a new deployment.
// It will emit a 'fake' status change event if the stack is in a stable state,
// so that the deployer can catch up with remote status.
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

	log.WithFields(ctx, map[string]interface{}{
		"stackName":   msg.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	switch lifecycle {
	case awsdeployer_pb.CFLifecycle_PROGRESS,
		awsdeployer_pb.CFLifecycle_ROLLING_BACK:
		// Keep Waiting, further events will be emitted

		if msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress {
			err := cf.eventOut(ctx, &awsinfra_tpb.CancelStackUpdateMessage{
				Request:   msg.Request,
				StackName: msg.StackName,
			})
			if err != nil {
				return nil, err
			}
			// Further events will be emitted as the cancel progresses
		}
		return &emptypb.Empty{}, nil

	case awsdeployer_pb.CFLifecycle_TERMINAL,
		awsdeployer_pb.CFLifecycle_ROLLED_BACK,
		awsdeployer_pb.CFLifecycle_CREATE_FAILED:
		// Roll on, no further events will be emitted, this is a failure for the
		// waiting deployment.

	case awsdeployer_pb.CFLifecycle_COMPLETE,
		awsdeployer_pb.CFLifecycle_MISSING:
		// Roll on, no further events will be emitted, this is a success for the
		// waiting deployment.

	default:
		return nil, fmt.Errorf("unexpected lifecycle %s", lifecycle)
	}

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
func (cf *InfraWorker) noUpdatesToBePerformed(ctx context.Context, stackName string, request *messaging_j5pb.RequestMetadata, eventID string) error {

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
