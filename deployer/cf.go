package deployer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type stackParameters struct {
	parameters         []*deployer_pb.Parameter
	scale              int
	previousParameters []*deployer_pb.AWSParameter
}

type AWSRunner struct {
	Clients ClientBuilder

	*deployer_tpb.UnimplementedAWSCommandTopicServer
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

func (cf *AWSRunner) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*emptypb.Empty, error) {

	//deployment *deployer_pb.DeploymentState, parameters []types.Parameter) error {
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
	Parameters  []*deployer_pb.AWSParameter
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

	parameters := make([]*deployer_pb.AWSParameter, len(stack.Parameters))
	for idx, param := range stack.Parameters {
		parameters[idx] = &deployer_pb.AWSParameter{
			Name:  *param.ParameterKey,
			Value: *param.ParameterValue,
		}
	}

	out := StackStatus{
		StatusName: stack.StackStatus,
		Parameters: parameters,
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
