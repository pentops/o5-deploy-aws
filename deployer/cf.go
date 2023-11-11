package deployer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type stackParameters struct {
	parameters         []*deployer_pb.Parameter
	scale              int
	previousParameters []*deployer_pb.AWSParameter
}

type CFWrapper struct {
	client CloudFormationAPI
}

func (cf *CFWrapper) getOneStack(ctx context.Context, stackName string) (*types.Stack, error) {

	res, err := cf.client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
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

func (cf *CFWrapper) cancelUpdate(ctx context.Context, stackName string) error {
	log.Info(ctx, "Cancel Update")
	_, err := cf.client.CancelUpdateStack(ctx, &cloudformation.CancelUpdateStackInput{
		StackName: aws.String(stackName),
	})
	return err
}

func (cf *CFWrapper) waitForStack(ctx context.Context, stackName string) (*types.Stack, error) {
	for {
		remoteStack, err := cf.getOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		if remoteStack == nil {
			return nil, nil
		}

		log.WithFields(ctx, map[string]interface{}{
			"currentStatus": remoteStack.StackStatus,
		}).Debug("Waiting for stack to be stable")

		switch remoteStack.StackStatus {
		case types.StackStatusCreateComplete, // Initial
			types.StackStatusUpdateComplete,         // Update
			types.StackStatusUpdateRollbackComplete: // Ready to try again.
			log.WithFields(ctx, map[string]interface{}{
				"currentStatus": remoteStack.StackStatus,
			}).Debug("Stack is stable")
			return remoteStack, nil

		case types.StackStatusRollbackComplete:
			log.WithFields(ctx, map[string]interface{}{
				"currentStatus": remoteStack.StackStatus,
			}).Debug("Deleting Rolled Back Stack")
			if err := cf.deleteStack(ctx, stackName); err != nil {
				return nil, err
			}

			// then continue waiting

		case types.StackStatusCreateFailed, types.StackStatusRollbackFailed, types.StackStatusDeleteComplete, types.StackStatusDeleteFailed:
			return nil, fmt.Errorf("stack %s is in status %s", stackName, remoteStack.StackStatus)

		default:
		}

		time.Sleep(time.Second)
	}
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

type waiterCallback func(ctx context.Context, stackStatus StackStatus) error

type StackStatus struct {
	StatusName  types.StackStatus
	SummaryType deployer_pb.StackLifecycle
	IsOK        bool
	Stable      bool
	Parameters  []*deployer_pb.AWSParameter
}

func (cf *CFWrapper) WaitForSuccess(ctx context.Context, stackName string, callback waiterCallback) error {
	_, err := cf.waitForSuccess(ctx, stackName, callback)
	return err
}

func (cf *CFWrapper) waitForSuccess(ctx context.Context, stackName string, callback waiterCallback) (*types.Stack, error) {
	var remoteStack *types.Stack
	var err error
	var lastStatus types.StackStatus
	for {
		remoteStack, err = cf.getOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		if remoteStack == nil {
			return nil, fmt.Errorf("missing stack %s", stackName)
		}

		if lastStatus != remoteStack.StackStatus {
			summary, err := summarizeStackStatus(remoteStack)
			if err != nil {
				return nil, err
			}

			if err := callback(ctx, summary); err != nil {
				return nil, err
			}

			if !summary.IsOK {
				return nil, fmt.Errorf("stack %s is NOT OK in status %s", stackName, remoteStack.StackStatus)
			}

			if summary.Stable {
				if summary.SummaryType == deployer_pb.StackLifecycle_CREATE_FAILED {
					return nil, nil
				}
				return remoteStack, nil
			}
		}

		lastStatus = remoteStack.StackStatus
		time.Sleep(time.Second)

	}
}

func summarizeStackStatus(stack *types.Stack) (StackStatus, error) {

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

	for _, status := range stackStatusesTerminal {
		if stack.StackStatus == status {
			out.SummaryType = deployer_pb.StackLifecycle_TERMINAL
			out.IsOK = false
			out.Stable = true
			return out, nil
		}
	}

	for _, status := range stackStatusComplete {
		if stack.StackStatus == status {
			out.SummaryType = deployer_pb.StackLifecycle_COMPLETE
			out.IsOK = true
			out.Stable = true
			return out, nil
		}
	}

	for _, status := range stackStatusCreateFailed {
		if stack.StackStatus == status {
			out.SummaryType = deployer_pb.StackLifecycle_CREATE_FAILED
			out.IsOK = false
			out.Stable = true
			return out, nil
		}
	}

	for _, status := range stackStatusRollingBack {
		if stack.StackStatus == status {
			out.SummaryType = deployer_pb.StackLifecycle_ROLLING_BACK
			out.IsOK = false
			out.Stable = false
			return out, nil
		}
	}

	for _, status := range stackStatusProgress {
		if stack.StackStatus == status {
			out.SummaryType = deployer_pb.StackLifecycle_PROGRESS
			out.IsOK = true
			out.Stable = false
			return out, nil
		}
	}

	return StackStatus{}, fmt.Errorf("unknown stack status %s", stack.StackStatus)
}

func (cf *CFWrapper) deleteStack(ctx context.Context, name string) error {
	_, err := cf.client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(name),
	})

	return err
}

type StackArgs struct {
	Template   *deployer_pb.DeploymentState
	Parameters []types.Parameter
}

func (cf *CFWrapper) createCloudformationStack(ctx context.Context, deployment *deployer_pb.DeploymentState, parameters []types.Parameter) error {

	_, err := cf.client.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:   aws.String(deployment.StackName),
		TemplateURL: aws.String(deployment.Spec.TemplateUrl),
		Parameters:  parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (cf *CFWrapper) updateCloudformationStack(ctx context.Context, deployment *deployer_pb.DeploymentState, parameters []types.Parameter) error {

	_, err := cf.client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:   aws.String(deployment.StackName),
		TemplateURL: aws.String(deployment.Spec.TemplateUrl),
		Parameters:  parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		if !isNoUpdatesError(err) {
			return fmt.Errorf("updateCFStack: %w", err)
		}
	}

	return nil
}

func (cf *CFWrapper) setScale(ctx context.Context, stackName string, desiredCount int) error {
	current, err := cf.getOneStack(ctx, stackName)
	if err != nil {
		return err
	}

	parameters := make([]types.Parameter, len(current.Parameters))
	for idx, param := range current.Parameters {
		if strings.HasPrefix(*param.ParameterKey, "DesiredCount") {
			parameters[idx] = types.Parameter{
				ParameterKey:   param.ParameterKey,
				ParameterValue: aws.String(fmt.Sprintf("%d", desiredCount)),
			}

		} else {
			parameters[idx] = types.Parameter{
				UsePreviousValue: aws.Bool(true),
				ParameterKey:     param.ParameterKey,
			}
		}

	}

	_, err = cf.client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:           aws.String(stackName),
		UsePreviousTemplate: aws.Bool(true),
		Parameters:          parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		if !isNoUpdatesError(err) {
			return fmt.Errorf("setScale: %w", err)
		}
	}

	return nil
}

func isNoUpdatesError(err error) bool {
	var opError smithy.APIError
	if !errors.As(err, &opError) {
		return false
	}

	return opError.ErrorCode() == "ValidationError" && opError.ErrorMessage() == "No updates are to be performed."
}
