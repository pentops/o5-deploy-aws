package awsinfra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type AWSRunner struct {
	Clients ClientBuilder
	*deployer_tpb.UnimplementedAWSCommandTopicServer
	MessageHandler interface {
		PublishEvent(context.Context, proto.Message) error
	}
}

func (cf *AWSRunner) eventOut(ctx context.Context, msg proto.Message) error {
	return cf.MessageHandler.PublishEvent(ctx, msg)
}

func (cf *AWSRunner) getClient(ctx context.Context) (CloudFormationAPI, error) {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}
	return clients.CloudFormation, nil
}

func (cf *AWSRunner) getOneStack(ctx context.Context, stackName string) (*types.Stack, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	res, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) {
			if awsErr.ErrorCode() == "ValidationError" && awsErr.ErrorMessage() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
				return nil, nil
			}
		}

		return nil, err
	}

	if len(res.Stacks) == 0 {
		return nil, nil
	}

	if len(res.Stacks) > 1 {
		return nil, fmt.Errorf("found more than one stack with name %s", stackName)
	}

	remoteStack := res.Stacks[0]
	return &remoteStack, nil
}

func (cf *AWSRunner) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*emptypb.Empty, error) {
	remoteStack, err := cf.getOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, err
	}

	if remoteStack == nil {
		err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackName: msg.StackName,
			Status:    "MISSING",
			Lifecycle: deployer_pb.StackLifecycle_MISSING,
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}

	lifecycle, err := stackLifecycle(remoteStack.StackStatus)
	if err != nil {
		return nil, err
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err := cf.eventOut(ctx, &deployer_tpb.DeleteStackMessage{
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
			StackName: msg.StackName,
		})
		if err != nil {
			return nil, err
		}
	}

	err = cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackName: msg.StackName,
		Status:    string(remoteStack.StackStatus),
		Lifecycle: lifecycle,
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil

}

func (cf *AWSRunner) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*emptypb.Empty, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	parameters := make([]types.Parameter, len(msg.Parameters))
	for idx, param := range msg.Parameters {
		parameters[idx] = types.Parameter{
			ParameterKey:   aws.String(param.Name),
			ParameterValue: aws.String(param.Value),
		}
	}

	_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:   aws.String(msg.StackName),
		TemplateURL: aws.String(msg.TemplateUrl),
		Parameters:  parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *AWSRunner) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*emptypb.Empty, error) {

	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	parameters := make([]types.Parameter, len(msg.Parameters))
	for idx, param := range msg.Parameters {
		parameters[idx] = types.Parameter{
			ParameterKey:   aws.String(param.Name),
			ParameterValue: aws.String(param.Value),
		}
	}

	_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:   aws.String(msg.StackName),
		TemplateURL: aws.String(msg.TemplateUrl),
		Parameters:  parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		// TODO: This will throw of the waiter, as no event is being performed
		if !isNoUpdatesError(err) {
			return nil, fmt.Errorf("updateCFStack: %w", err)
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *AWSRunner) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*emptypb.Empty, error) {

	current, err := cf.getOneStack(ctx, msg.StackName)
	if err != nil {
		return nil, err
	}

	parameters := make([]types.Parameter, len(current.Parameters))
	for idx, param := range current.Parameters {
		if strings.HasPrefix(*param.ParameterKey, "DesiredCount") {
			parameters[idx] = types.Parameter{
				ParameterKey:   param.ParameterKey,
				ParameterValue: aws.String(fmt.Sprintf("%d", msg.DesiredCount)),
			}

		} else {
			parameters[idx] = types.Parameter{
				UsePreviousValue: aws.Bool(true),
				ParameterKey:     param.ParameterKey,
			}
		}

	}

	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:           aws.String(msg.StackName),
		UsePreviousTemplate: aws.Bool(true),
		Parameters:          parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		// TODO: This will throw of the waiter, as no event is being performed
		if !isNoUpdatesError(err) {
			return nil, fmt.Errorf("setScale: %w", err)
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *AWSRunner) CancelStackUpdate(ctx context.Context, evt *deployer_tpb.CancelStackUpdateMessage) (*emptypb.Empty, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.CancelUpdateStack(ctx, &cloudformation.CancelUpdateStackInput{
		StackName: aws.String(evt.StackName),
	})
	return &emptypb.Empty{}, err
}

func (cf *AWSRunner) DeleteStack(ctx context.Context, evt *deployer_tpb.DeleteStackMessage) (*emptypb.Empty, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(evt.StackName),
	})

	return nil, err
}

func (cf *AWSRunner) UpsertSNSTopics(ctx context.Context, evt *deployer_tpb.UpsertSNSTopicsMessage) (*emptypb.Empty, error) {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}

	snsClient := clients.SNS

	for _, topic := range evt.TopicNames {
		_, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: aws.String(fmt.Sprintf("%s-%s", evt.EnvironmentName, topic)),
		})
		if err != nil {
			return nil, fmt.Errorf("creating sns topic %s: %w", topic, err)
		}
	}
	return &emptypb.Empty{}, nil
}

var stackStatusProgress = []types.StackStatus{
	types.StackStatusCreateInProgress,
	types.StackStatusUpdateInProgress,
	types.StackStatusReviewInProgress,
	types.StackStatusImportInProgress,
	types.StackStatusDeleteInProgress,
	types.StackStatusUpdateCompleteCleanupInProgress,
}

var stackStatusRollingBack = []types.StackStatus{
	types.StackStatusRollbackInProgress,
	types.StackStatusRollbackComplete,
	types.StackStatusUpdateRollbackFailed,
	types.StackStatusUpdateRollbackCompleteCleanupInProgress,
}

var stackStatusCreateFailed = []types.StackStatus{
	types.StackStatusRollbackComplete,
}

var stackStatusComplete = []types.StackStatus{
	types.StackStatusCreateComplete,
	types.StackStatusImportComplete,
	types.StackStatusUpdateComplete,
	types.StackStatusDeleteComplete,
}

var stackStatusesTerminal = []types.StackStatus{
	types.StackStatusCreateFailed,
	types.StackStatusRollbackFailed,
	types.StackStatusDeleteFailed,
	types.StackStatusUpdateFailed,
	types.StackStatusUpdateRollbackInProgress,
	types.StackStatusImportRollbackInProgress,
	types.StackStatusImportRollbackFailed,
	types.StackStatusImportRollbackComplete,
	types.StackStatusUpdateRollbackComplete,
}

type StackStatus struct {
	StatusName  types.StackStatus
	SummaryType deployer_pb.StackLifecycle
	IsOK        bool
	Stable      bool
	Parameters  []*deployer_pb.KeyValue
}

func stackLifecycle(remoteStatus types.StackStatus) (deployer_pb.StackLifecycle, error) {
	for _, status := range stackStatusesTerminal {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_TERMINAL, nil
		}
	}

	for _, status := range stackStatusComplete {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_COMPLETE, nil
		}
	}

	for _, status := range stackStatusCreateFailed {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_CREATE_FAILED, nil
		}
	}

	for _, status := range stackStatusRollingBack {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_ROLLING_BACK, nil
		}
	}

	for _, status := range stackStatusProgress {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_PROGRESS, nil
		}
	}

	return deployer_pb.StackLifecycle_UNSPECIFIED, fmt.Errorf("unknown stack status %s", remoteStatus)

}
func summarizeStackStatus(stack *types.Stack) (StackStatus, error) {

	lifecycle, err := stackLifecycle(stack.StackStatus)
	if err != nil {
		return StackStatus{}, err
	}

	parameters := make([]*deployer_pb.KeyValue, len(stack.Parameters))
	for idx, param := range stack.Parameters {
		parameters[idx] = &deployer_pb.KeyValue{
			Name:  *param.ParameterKey,
			Value: *param.ParameterValue,
		}
	}

	out := StackStatus{
		StatusName:  stack.StackStatus,
		Parameters:  parameters,
		SummaryType: lifecycle,
	}

	switch lifecycle {

	case deployer_pb.StackLifecycle_COMPLETE:
		out.IsOK = true
		out.Stable = true

	case deployer_pb.StackLifecycle_TERMINAL:
		out.IsOK = false
		out.Stable = true

	case deployer_pb.StackLifecycle_CREATE_FAILED:
		out.IsOK = false
		out.Stable = true

	case deployer_pb.StackLifecycle_ROLLING_BACK:
		out.IsOK = false
		out.Stable = false

	case deployer_pb.StackLifecycle_PROGRESS:
		out.IsOK = true
		out.Stable = false

	default:
		return StackStatus{}, fmt.Errorf("unknown stack lifecycle: %s", lifecycle)
	}

	return out, nil
}

type StackArgs struct {
	Template   *deployer_pb.DeploymentState
	Parameters []types.Parameter
}

func isNoUpdatesError(err error) bool {
	var opError smithy.APIError
	if !errors.As(err, &opError) {
		return false
	}

	return opError.ErrorCode() == "ValidationError" && opError.ErrorMessage() == "No updates are to be performed."
}

func (cf *AWSRunner) PollStack(ctx context.Context, stackName string) error {

	ctx = log.WithField(ctx, "stackName", stackName)

	log.Debug(ctx, "PollStack Begin")

	var lastStatus types.StackStatus
	for {
		remoteStack, err := cf.getOneStack(ctx, stackName)
		if err != nil {
			return err
		}
		if remoteStack == nil {
			return fmt.Errorf("missing stack %s", stackName)
		}

		if lastStatus == remoteStack.StackStatus {
			time.Sleep(5 * time.Second)
			continue
		}

		lastStatus = remoteStack.StackStatus

		outputs := make([]*deployer_pb.KeyValue, len(remoteStack.Outputs))
		for i, output := range remoteStack.Outputs {
			outputs[i] = &deployer_pb.KeyValue{
				Name:  *output.OutputKey,
				Value: *output.OutputValue,
			}
		}

		summary, err := summarizeStackStatus(remoteStack)
		if err != nil {
			return err
		}

		log.WithFields(ctx, map[string]interface{}{
			"lifecycle":   summary.SummaryType.ShortString(),
			"stackStatus": remoteStack.StackStatus,
		}).Debug("PollStack Result")

		if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackName: *remoteStack.StackName,
			Status:    string(remoteStack.StackStatus),
			Outputs:   outputs,
			Lifecycle: summary.SummaryType,
		}); err != nil {
			return err
		}

		if summary.Stable {
			break
		}
	}

	return nil
}
