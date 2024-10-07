package aws_cf

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
)

var statusMap = map[types.StackStatus]awsdeployer_pb.CFLifecycle{
	// Progress: Going well, no errors yet, keep waiting.
	types.StackStatusCreateInProgress:                awsdeployer_pb.CFLifecycle_PROGRESS,
	types.StackStatusUpdateInProgress:                awsdeployer_pb.CFLifecycle_PROGRESS,
	types.StackStatusReviewInProgress:                awsdeployer_pb.CFLifecycle_PROGRESS,
	types.StackStatusImportInProgress:                awsdeployer_pb.CFLifecycle_PROGRESS,
	types.StackStatusDeleteInProgress:                awsdeployer_pb.CFLifecycle_PROGRESS,
	types.StackStatusUpdateCompleteCleanupInProgress: awsdeployer_pb.CFLifecycle_PROGRESS,
	types.StackStatusImportRollbackInProgress:        awsdeployer_pb.CFLifecycle_PROGRESS,

	// Rolling Back: Something has gone wrong but the stack isn't stable yet,
	// keep waiting.
	types.StackStatusRollbackInProgress:                      awsdeployer_pb.CFLifecycle_ROLLING_BACK,
	types.StackStatusUpdateRollbackInProgress:                awsdeployer_pb.CFLifecycle_ROLLING_BACK,
	types.StackStatusUpdateRollbackCompleteCleanupInProgress: awsdeployer_pb.CFLifecycle_ROLLING_BACK,

	// Rolled Back: The last update failed, but the stack is now stable so a new
	// deployment can begin.
	types.StackStatusUpdateRollbackComplete: awsdeployer_pb.CFLifecycle_ROLLED_BACK,

	// Create Failed: The stack was not created, it may need to be deleted,
	// manual intervention may be required.
	types.StackStatusRollbackComplete:       awsdeployer_pb.CFLifecycle_CREATE_FAILED,
	types.StackStatusImportRollbackComplete: awsdeployer_pb.CFLifecycle_CREATE_FAILED,

	// Missing: There is no stack, create a new one.
	types.StackStatusDeleteComplete: awsdeployer_pb.CFLifecycle_MISSING,

	// Complete: The change succeeded, and the stack is stable, ready for a new
	// deployment
	types.StackStatusCreateComplete: awsdeployer_pb.CFLifecycle_COMPLETE,
	types.StackStatusImportComplete: awsdeployer_pb.CFLifecycle_COMPLETE,
	types.StackStatusUpdateComplete: awsdeployer_pb.CFLifecycle_COMPLETE,

	// Terminal: Something we can't automatically recover from, manual
	// intervention is required.
	types.StackStatusCreateFailed:         awsdeployer_pb.CFLifecycle_TERMINAL,
	types.StackStatusRollbackFailed:       awsdeployer_pb.CFLifecycle_TERMINAL,
	types.StackStatusDeleteFailed:         awsdeployer_pb.CFLifecycle_TERMINAL,
	types.StackStatusUpdateFailed:         awsdeployer_pb.CFLifecycle_TERMINAL,
	types.StackStatusUpdateRollbackFailed: awsdeployer_pb.CFLifecycle_TERMINAL,
	types.StackStatusImportRollbackFailed: awsdeployer_pb.CFLifecycle_TERMINAL,
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
	if status, ok := statusMap[remoteStatus]; ok {
		return status, nil
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
