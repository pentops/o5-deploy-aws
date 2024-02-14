package awsinfra

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DBLite interface {
	PublishEvent(context.Context, outbox.OutboxMessage) error
	RequestToClientToken(context.Context, *messaging_pb.RequestMetadata) (string, error)
	ClientTokenToRequest(context.Context, string) (*messaging_pb.RequestMetadata, error)
}

var RequestTokenNotFound = errors.New("request token not found")

type InfraWorker struct {
	*deployer_tpb.UnimplementedCloudFormationRequestTopicServer
	*messaging_tpb.UnimplementedRawMessageTopicServer

	db DBLite
	*CFClient
}

func NewInfraWorker(clients ClientBuilder, db DBLite) *InfraWorker {
	cfClient := &CFClient{
		Clients: clients,
	}
	return &InfraWorker{
		CFClient: cfClient,
		db:       db,
	}
}

func (cf *InfraWorker) eventOut(ctx context.Context, msg outbox.OutboxMessage) error {
	fmt.Printf("eventOut: %s\n", protojson.Format(msg))
	return cf.db.PublishEvent(ctx, msg)
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

	requestMetadata, err := cf.db.ClientTokenToRequest(ctx, clientToken)
	if errors.Is(err, RequestTokenNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		Request:   requestMetadata,
		StackName: stackName,
		Status:    resourceStatus,
		Outputs:   outputs,
		Lifecycle: lifecycle,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*emptypb.Empty, error) {

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

func (cf *InfraWorker) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	err = cf.CFClient.UpdateStack(ctx, reqToken, msg)
	if err != nil {
		if !IsNoUpdatesError(err) {
			return nil, fmt.Errorf("UpdateStack: %w", err)
		}

		if err := cf.noUpdatesToBePerformed(ctx, msg.StackName, msg.Request); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) CancelStackUpdate(ctx context.Context, msg *deployer_tpb.CancelStackUpdateMessage) (*emptypb.Empty, error) {
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

func (cf *InfraWorker) DeleteStack(ctx context.Context, msg *deployer_tpb.DeleteStackMessage) (*emptypb.Empty, error) {
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

func (cf *InfraWorker) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*emptypb.Empty, error) {
	reqToken, err := cf.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}
	err = cf.CFClient.ScaleStack(ctx, reqToken, msg)
	if err != nil {
		if !IsNoUpdatesError(err) {
			return nil, fmt.Errorf("ScaleStack: %w", err)
		}

		if err := cf.noUpdatesToBePerformed(ctx, msg.StackName, msg.Request); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *InfraWorker) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*emptypb.Empty, error) {

	remoteStack, err := cf.getOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, fmt.Errorf("getOneStack: %w", err)
	}

	if remoteStack == nil {
		err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			Request:   msg.Request,
			StackName: msg.StackName,
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
		err := cf.eventOut(ctx, &deployer_tpb.CancelStackUpdateMessage{
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
		lifecycle = deployer_pb.StackLifecycle_COMPLETE
		err = cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			Request:   msg.Request,
			StackName: msg.StackName,
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
		"stackName":   msg.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	// This only runs if we aren't running the poller, as the poller will do
	// the same thing
	err = cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		Request:   msg.Request,
		StackName: msg.StackName,
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
func (cf *InfraWorker) noUpdatesToBePerformed(ctx context.Context, stackName string, request *messaging_pb.RequestMetadata) error {

	remoteStack, err := cf.getOneStack(ctx, stackName)
	if err != nil {
		return err
	}

	summary, err := summarizeStackStatus(remoteStack)
	if err != nil {
		return err
	}

	if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		Request:   request,
		StackName: stackName,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   summary.Outputs,
		Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
	}); err != nil {
		return err
	}

	return nil
}
