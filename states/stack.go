package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/protostate/psm"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var namespaceStackID = uuid.MustParse("C27983FD-BC4B-493F-A056-CC8C869A1999")

func StackID(envName, appName string) string {
	return uuid.NewMD5(namespaceStackID, []byte(fmt.Sprintf("%s-%s", envName, appName))).String()
}

func NewStackEventer() (*deployer_pb.StackPSM, error) {
	config := deployer_pb.DefaultStackPSMConfig().
		StoreExtraStateColumns(func(s *deployer_pb.StackState) (map[string]interface{}, error) {
			mm := map[string]interface{}{
				"env_name":       s.Data.EnvironmentName,
				"app_name":       s.Data.ApplicationName,
				"environment_id": s.Data.EnvironmentId,
				"github_owner":   "", // TODO: Support NIL in PSM conversion
				"github_repo":    "",
				"github_ref":     "",
			}

			if s.Data.Config != nil && s.Data.Config.CodeSource != nil {
				githubConfig := s.Data.Config.CodeSource.GetGithub()
				if githubConfig != nil {
					mm["github_owner"] = githubConfig.Owner
					mm["github_repo"] = githubConfig.Repo
					mm["github_ref"] = fmt.Sprintf("refs/heads/%s", githubConfig.Branch)
				}
			}

			return mm, nil
		}).
		StoreExtraEventColumns(func(e *deployer_pb.StackEvent) (map[string]interface{}, error) {
			return map[string]interface{}{
				"id":        e.Metadata.EventId,
				"stack_id":  e.Keys.StackId,
				"timestamp": e.Metadata.Timestamp,
			}, nil
		}).
		SystemActor(psm.MustSystemActor("0F34118E-6263-4634-A5FB-5C04D71203D2"))

	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	// Creating the stack through configuration as the first step.
	// The stack is immediately 'AVAILABLE' and ready for the first deployment.
	// [*] --> AVAILABLE : Configured
	sm.From(0).
		OnEvent(deployer_pb.StackPSMEventConfigured).
		SetStatus(deployer_pb.StackStatus_AVAILABLE).
		Mutate(deployer_pb.StackPSMMutation(func(
			state *deployer_pb.StackStateData,
			event *deployer_pb.StackEventType_Configured,
		) error {
			state.Config = event.Config
			state.EnvironmentId = event.EnvironmentId
			state.EnvironmentName = event.EnvironmentName
			state.ApplicationName = event.ApplicationName
			state.StackName = fmt.Sprintf("%s-%s", event.EnvironmentName, event.ApplicationName)
			return nil
		}))

	// Creating the stack through the first deployment request.
	// [*] --> AVAILABLE : DeploymentRequested
	sm.From(0).
		OnEvent(deployer_pb.StackPSMEventDeploymentRequested).
		SetStatus(deployer_pb.StackStatus_AVAILABLE).
		Mutate(deployer_pb.StackPSMMutation(func(
			state *deployer_pb.StackStateData,
			event *deployer_pb.StackEventType_DeploymentRequested,
		) error {
			state.ApplicationName = event.ApplicationName
			state.EnvironmentName = event.EnvironmentName
			state.EnvironmentId = event.EnvironmentId
			state.QueuedDeployments = []*deployer_pb.StackDeployment{
				event.Deployment,
			}
			state.StackName = fmt.Sprintf("%s-%s", event.EnvironmentName, event.ApplicationName)
			return nil
		}))

	// Configuration after creation simply updates the config, leaving the
	// status.
	sm.From(
		deployer_pb.StackStatus_AVAILABLE,
		deployer_pb.StackStatus_MIGRATING,
	).
		Mutate(deployer_pb.StackPSMMutation(func(
			state *deployer_pb.StackStateData,
			event *deployer_pb.StackEventType_Configured,
		) error {
			state.Config = event.Config

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
		deployer_pb.StackStatus_AVAILABLE,
		deployer_pb.StackStatus_MIGRATING,
	).Mutate(deployer_pb.StackPSMMutation(func(
		state *deployer_pb.StackStateData,
		event *deployer_pb.StackEventType_DeploymentRequested,
	) error {
		state.QueuedDeployments = append(state.QueuedDeployments, event.Deployment)
		return nil
	}))

	// AVAILABLE --> MIGRATING : RunDeployment
	sm.From(deployer_pb.StackStatus_AVAILABLE).
		OnEvent(deployer_pb.StackPSMEventRunDeployment).
		SetStatus(deployer_pb.StackStatus_MIGRATING).
		Mutate(deployer_pb.StackPSMMutation(func(
			state *deployer_pb.StackStateData,
			event *deployer_pb.StackEventType_RunDeployment,
		) error {
			remaining := make([]*deployer_pb.StackDeployment, 0, len(state.QueuedDeployments))
			var deployment *deployer_pb.StackDeployment
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
	sm.From(deployer_pb.StackStatus_MIGRATING).
		OnEvent(deployer_pb.StackPSMEventDeploymentCompleted).
		SetStatus(deployer_pb.StackStatus_AVAILABLE).
		Mutate(deployer_pb.StackPSMMutation(func(
			state *deployer_pb.StackStateData,
			event *deployer_pb.StackEventType_DeploymentCompleted,
		) error {
			state.CurrentDeployment = nil
			return nil
		}))

		// After any event, if the status is AVAILABLE And there are queued
		// deployments, run the next one.
	sm.GeneralHook(deployer_pb.StackPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.StackPSMHookBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEvent,
	) error {
		if state.Status != deployer_pb.StackStatus_AVAILABLE {
			return nil
		}
		if len(state.Data.QueuedDeployments) == 0 {
			return nil
		}
		tb.ChainEvent(&deployer_pb.StackEventType_RunDeployment{
			DeploymentId: state.Data.QueuedDeployments[0].DeploymentId,
		})
		return nil
	}))

	return sm, nil
}
