package awsinfra

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
)

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
	types.StackStatusUpdateRollbackInProgress,
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
	types.StackStatusImportRollbackInProgress,
	types.StackStatusImportRollbackFailed,
	types.StackStatusImportRollbackComplete,
}

var stackStatusesTerminalRollback = []types.StackStatus{
	types.StackStatusUpdateRollbackComplete,
}

type StackStatus struct {
	StackStatus types.StackStatus
	SummaryType awsdeployer_pb.CFLifecycle
	IsOK        bool
	Stable      bool
	Parameters  []*awsdeployer_pb.KeyValue
	Outputs     []*awsdeployer_pb.KeyValue
}

func stackLifecycle(remoteStatus types.StackStatus) (awsdeployer_pb.CFLifecycle, error) {
	for _, status := range stackStatusesTerminal {
		if remoteStatus == status {
			return awsdeployer_pb.CFLifecycle_TERMINAL, nil
		}
	}

	for _, status := range stackStatusesTerminalRollback {
		if remoteStatus == status {
			return awsdeployer_pb.CFLifecycle_ROLLED_BACK, nil
		}
	}

	for _, status := range stackStatusComplete {
		if remoteStatus == status {
			return awsdeployer_pb.CFLifecycle_COMPLETE, nil
		}
	}

	for _, status := range stackStatusCreateFailed {
		if remoteStatus == status {
			return awsdeployer_pb.CFLifecycle_CREATE_FAILED, nil
		}
	}

	for _, status := range stackStatusRollingBack {
		if remoteStatus == status {
			return awsdeployer_pb.CFLifecycle_ROLLING_BACK, nil
		}
	}

	for _, status := range stackStatusProgress {
		if remoteStatus == status {
			return awsdeployer_pb.CFLifecycle_PROGRESS, nil
		}
	}

	return awsdeployer_pb.CFLifecycle_UNSPECIFIED, fmt.Errorf("unknown stack status %s", remoteStatus)

}

func mapOutputs(outputs []types.Output) []*awsdeployer_pb.KeyValue {
	out := make([]*awsdeployer_pb.KeyValue, len(outputs))
	for i, output := range outputs {
		out[i] = &awsdeployer_pb.KeyValue{
			Name:  *output.OutputKey,
			Value: *output.OutputValue,
		}
	}
	return out
}

func summarizeStackStatus(stack *types.Stack) (StackStatus, error) {

	if stack == nil {
		return StackStatus{
			StackStatus: "MISSING",
			SummaryType: awsdeployer_pb.CFLifecycle_MISSING,
			Stable:      true,
			IsOK:        false,
		}, nil
	}

	lifecycle, err := stackLifecycle(stack.StackStatus)
	if err != nil {
		return StackStatus{}, err
	}

	parameters := make([]*awsdeployer_pb.KeyValue, len(stack.Parameters))
	for idx, param := range stack.Parameters {
		parameters[idx] = &awsdeployer_pb.KeyValue{
			Name:  *param.ParameterKey,
			Value: *param.ParameterValue,
		}
	}

	out := StackStatus{
		StackStatus: stack.StackStatus,
		Parameters:  parameters,
		SummaryType: lifecycle,
		Outputs:     mapOutputs(stack.Outputs),
	}

	switch lifecycle {

	case awsdeployer_pb.CFLifecycle_COMPLETE:
		out.IsOK = true
		out.Stable = true

	case awsdeployer_pb.CFLifecycle_TERMINAL:
		out.IsOK = false
		out.Stable = true

	case awsdeployer_pb.CFLifecycle_CREATE_FAILED:
		out.IsOK = false
		out.Stable = true

	case awsdeployer_pb.CFLifecycle_ROLLING_BACK:
		out.IsOK = false
		out.Stable = false

	case awsdeployer_pb.CFLifecycle_ROLLED_BACK:
		out.IsOK = false
		out.Stable = true

	case awsdeployer_pb.CFLifecycle_PROGRESS:
		out.IsOK = true
		out.Stable = false

	default:
		return StackStatus{}, fmt.Errorf("unknown stack lifecycle: %s", lifecycle)
	}

	return out, nil
}
