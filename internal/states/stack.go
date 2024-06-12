package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var namespaceStackID = uuid.MustParse("C27983FD-BC4B-493F-A056-CC8C869A1999")

func StackID(envName, appName string) string {
	return uuid.NewMD5(namespaceStackID, []byte(fmt.Sprintf("%s-%s", envName, appName))).String()
}

func NewStackEventer() (*awsdeployer_pb.StackPSM, error) {
	config := awsdeployer_pb.DefaultStackPSMConfig().
		SystemActor(psm.MustSystemActor("0F34118E-6263-4634-A5FB-5C04D71203D2"))

	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	// Creating the stack through configuration as the first step.
	// The stack is immediately 'AVAILABLE' and ready for the first deployment.
	// [*] --> AVAILABLE : Configured
	sm.From(0).
		OnEvent(awsdeployer_pb.StackPSMEventConfigured).
		SetStatus(awsdeployer_pb.StackStatus_AVAILABLE).
		Mutate(awsdeployer_pb.StackPSMMutation(func(
			state *awsdeployer_pb.StackStateData,
			event *awsdeployer_pb.StackEventType_Configured,
		) error {
			state.EnvironmentId = event.EnvironmentId
			state.EnvironmentName = event.EnvironmentName
			state.ApplicationName = event.ApplicationName
			state.StackName = fmt.Sprintf("%s-%s", event.EnvironmentName, event.ApplicationName)
			return nil
		}))

	// Creating the stack through the first deployment request.
	// [*] --> AVAILABLE : DeploymentRequested
	sm.From(0).
		OnEvent(awsdeployer_pb.StackPSMEventDeploymentRequested).
		SetStatus(awsdeployer_pb.StackStatus_AVAILABLE).
		Mutate(awsdeployer_pb.StackPSMMutation(func(
			state *awsdeployer_pb.StackStateData,
			event *awsdeployer_pb.StackEventType_DeploymentRequested,
		) error {
			state.ApplicationName = event.ApplicationName
			state.EnvironmentName = event.EnvironmentName
			state.EnvironmentId = event.EnvironmentId
			state.QueuedDeployments = []*awsdeployer_pb.StackDeployment{
				event.Deployment,
			}
			state.StackName = fmt.Sprintf("%s-%s", event.EnvironmentName, event.ApplicationName)
			return nil
		}))

	// Configuration after creation simply updates the config, leaving the
	// status.
	sm.From(
		awsdeployer_pb.StackStatus_AVAILABLE,
		awsdeployer_pb.StackStatus_MIGRATING,
	).
		Mutate(awsdeployer_pb.StackPSMMutation(func(
			state *awsdeployer_pb.StackStateData,
			event *awsdeployer_pb.StackEventType_Configured,
		) error {

			if state.EnvironmentId != event.EnvironmentId {
				return status.Errorf(codes.InvalidArgument, "environment id cannot be changed (from %s to %s)", state.EnvironmentId, event.EnvironmentId)
			}
			if state.EnvironmentName != event.EnvironmentName {
				return status.Errorf(codes.InvalidArgument, "environment name cannot be changed (from %s to %s)", state.EnvironmentName, event.EnvironmentName)
			}
			if state.ApplicationName != event.ApplicationName {
				return status.Errorf(codes.InvalidArgument, "application name cannot be changed (from %s to %s)", state.ApplicationName, event.ApplicationName)
			}

			return nil
		}))

	// AVAILABLE --> AVAILABLE : DeploymentRequested
	// MIGRATING --> MIGRATING : DeploymentRequested
	// Queue the deployment for later
	sm.From(
		awsdeployer_pb.StackStatus_AVAILABLE,
		awsdeployer_pb.StackStatus_MIGRATING,
	).Mutate(awsdeployer_pb.StackPSMMutation(func(
		state *awsdeployer_pb.StackStateData,
		event *awsdeployer_pb.StackEventType_DeploymentRequested,
	) error {
		state.QueuedDeployments = append(state.QueuedDeployments, event.Deployment)
		return nil
	}))

	// AVAILABLE --> MIGRATING : RunDeployment
	sm.From(awsdeployer_pb.StackStatus_AVAILABLE).
		OnEvent(awsdeployer_pb.StackPSMEventRunDeployment).
		SetStatus(awsdeployer_pb.StackStatus_MIGRATING).
		Mutate(awsdeployer_pb.StackPSMMutation(func(
			state *awsdeployer_pb.StackStateData,
			event *awsdeployer_pb.StackEventType_RunDeployment,
		) error {
			remaining := make([]*awsdeployer_pb.StackDeployment, 0, len(state.QueuedDeployments))
			var deployment *awsdeployer_pb.StackDeployment
			for _, d := range state.QueuedDeployments {
				if d.DeploymentId == event.DeploymentId {
					deployment = d
				} else {
					remaining = append(remaining, d)
				}
			}
			if deployment == nil {
				return status.Errorf(codes.NotFound, "deployment %s not found", event.DeploymentId)
			}

			state.QueuedDeployments = remaining
			state.CurrentDeployment = deployment
			// A cross-machine hook will trigger the deployment state machine
			return nil
		}))

	// MIGRATING --> AVAILABLE : DeploymentCompleted
	sm.From(awsdeployer_pb.StackStatus_MIGRATING).
		OnEvent(awsdeployer_pb.StackPSMEventDeploymentCompleted).
		SetStatus(awsdeployer_pb.StackStatus_AVAILABLE).
		Mutate(awsdeployer_pb.StackPSMMutation(func(
			state *awsdeployer_pb.StackStateData,
			event *awsdeployer_pb.StackEventType_DeploymentCompleted,
		) error {
			state.CurrentDeployment = nil
			return nil
		}))

		// After any event, if the status is AVAILABLE And there are queued
		// deployments, run the next one.
	sm.GeneralHook(awsdeployer_pb.StackPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb awsdeployer_pb.StackPSMHookBaton,
		state *awsdeployer_pb.StackState,
		event *awsdeployer_pb.StackEvent,
	) error {
		if state.Status != awsdeployer_pb.StackStatus_AVAILABLE {
			return nil
		}
		if len(state.Data.QueuedDeployments) == 0 {
			return nil
		}
		tb.ChainEvent(&awsdeployer_pb.StackEventType_RunDeployment{
			DeploymentId: state.Data.QueuedDeployments[0].DeploymentId,
		})
		return nil
	}))

	return sm, nil
}
