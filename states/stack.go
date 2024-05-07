package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var namespaceStackID = uuid.MustParse("C27983FD-BC4B-493F-A056-CC8C869A1999")

func StackID(envName, appName string) string {
	return uuid.NewMD5(namespaceStackID, []byte(fmt.Sprintf("%s-%s", envName, appName))).String()
}

func chainStackEvent(tb deployer_pb.StackPSMTransitionBaton, event deployer_pb.IsStackEventTypeWrappedType) *deployer_pb.StackEvent {
	md := tb.FullCause().Metadata
	de := &deployer_pb.StackEvent{
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor:     md.Actor,
		},
		StackId: tb.FullCause().StackId,
		Event:   &deployer_pb.StackEventType{},
	}
	de.Event.Set(event)
	return de
}

func NewStackEventer() (*deployer_pb.StackPSM, error) {
	config := deployer_pb.DefaultStackPSMConfig().
		StoreExtraStateColumns(func(s *deployer_pb.StackState) (map[string]interface{}, error) {
			mm := map[string]interface{}{
				"env_name":       s.EnvironmentName,
				"app_name":       s.ApplicationName,
				"environment_id": s.EnvironmentId,
				"github_owner":   "", // TODO: Support NIL in PSM conversion
				"github_repo":    "",
				"github_ref":     "",
			}

			if s.Config != nil && s.Config.CodeSource != nil {
				githubConfig := s.Config.CodeSource.GetGithub()
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
				"stack_id":  e.StackId,
				"timestamp": e.Metadata.Timestamp,
			}, nil
		})

	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	/*
		TODO: Future hook
		sm.AddHook(func(ctx context.Context, tx sqrlx.Transaction, state *deployer_pb.StackState, event *deployer_pb.StackEvent) error {
			evt := &deployer_epb.StackEventMessage{
				Metadata: event.Metadata,
				Event:    event.Event,
				State:    state,
			}
			return outbox.Send(ctx, tx, evt)
		})
	*/

	// Creating the stack through configuration as the first step.
	// The stack is immediately 'AVAILABLE' and ready for the first deployment.
	sm.From(
		deployer_pb.StackStatus_UNSPECIFIED,
	).Transition(deployer_pb.StackPSMTransition(func(
		ctx context.Context,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Configured,
	) error {
		state.Status = deployer_pb.StackStatus_AVAILABLE
		state.Config = event.Config
		state.EnvironmentId = event.EnvironmentId
		state.EnvironmentName = event.EnvironmentName
		state.ApplicationName = event.ApplicationName
		state.StackName = fmt.Sprintf("%s-%s", event.EnvironmentName, event.ApplicationName)
		return nil
	}))

	// Updating the configuration to an existing stack, regardless of how it was
	// created (via configuration or deployment), leaves the status as it is and
	// just updates the config.
	sm.From().
		Transition(deployer_pb.StackPSMTransition(func(
			ctx context.Context,
			state *deployer_pb.StackState,
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

	// The 'Triggered' event arrives when a Deployment is *created*.
	// The deployment starts in a 'QUEUED' status
	// Then we send a Trigger back to the deployer to kick off the
	// deployment, `QUEUED --> TRIGGERED : Trigger`

	// [*] --> CREATING : Triggered
	// As this is the first deployment for the stack, we can trigger the deployment immediately.
	sm.From(
		deployer_pb.StackStatus_UNSPECIFIED,
	).Transition(deployer_pb.StackPSMTransition(func(
		ctx context.Context,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {
		state.Status = deployer_pb.StackStatus_CREATING
		state.CurrentDeployment = event.Deployment
		state.ApplicationName = event.ApplicationName
		state.EnvironmentName = event.EnvironmentName
		state.EnvironmentId = event.EnvironmentId
		state.QueuedDeployments = []*deployer_pb.StackDeployment{}
		state.StackName = fmt.Sprintf("%s-%s", event.EnvironmentName, event.ApplicationName)
		return nil
	})).Hook(deployer_pb.StackPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.StackPSMHookBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {

		tb.SideEffect(&deployer_tpb.TriggerDeploymentMessage{
			DeploymentId:    state.CurrentDeployment.DeploymentId,
			StackId:         state.StackId,
			Version:         state.CurrentDeployment.Version,
			EnvironmentName: state.EnvironmentName,
			ApplicationName: state.ApplicationName,
		})
		return nil

	}))

	// AVAILABLE --> MIGRATING : Triggered externally, run now
	sm.From(
		deployer_pb.StackStatus_AVAILABLE,
	).Transition(deployer_pb.StackPSMTransition(func(
		ctx context.Context,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {
		state.Status = deployer_pb.StackStatus_MIGRATING

		state.CurrentDeployment = event.Deployment
		return nil
	})).Hook(deployer_pb.StackPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.StackPSMHookBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {

		tb.SideEffect(&deployer_tpb.TriggerDeploymentMessage{
			DeploymentId:    state.CurrentDeployment.DeploymentId,
			StackId:         state.StackId,
			Version:         state.CurrentDeployment.Version,
			EnvironmentName: state.EnvironmentName,
			ApplicationName: state.ApplicationName,
		})
		return nil
	}))

	// CREATING --> STABLE : DeploymentCompleted
	// MIGRATING --> STABLE : DeploymentCompleted
	sm.From(
		deployer_pb.StackStatus_CREATING,
		deployer_pb.StackStatus_MIGRATING,
	).
		Transition(deployer_pb.StackPSMTransition(func(
			ctx context.Context,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_DeploymentCompleted,
		) error {
			state.Status = deployer_pb.StackStatus_STABLE
			return nil
		})).
		Hook(deployer_pb.StackPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.StackPSMHookBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_DeploymentCompleted,
		) error {
			tb.ChainEvent(chainStackEvent(tb, &deployer_pb.StackEventType_Available{}))
			return nil
		}))

	// STABLE -> AVAILABLE : Available
	sm.From(deployer_pb.StackStatus_STABLE).
		Transition(deployer_pb.StackPSMTransition(func(
			ctx context.Context,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Available,
		) error {

			if len(state.QueuedDeployments) == 0 {
				state.Status = deployer_pb.StackStatus_AVAILABLE
				state.CurrentDeployment = nil
				// Nothing left to do, leave the stack in available
				return nil
			}

			state.Status = deployer_pb.StackStatus_MIGRATING

			// TODO: An event to trigger the next deployment by ID
			state.CurrentDeployment = state.QueuedDeployments[0]
			state.QueuedDeployments = state.QueuedDeployments[1:]
			return nil
		})).
		Hook(deployer_pb.StackPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.StackPSMHookBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_Available,
		) error {
			if state.Status != deployer_pb.StackStatus_MIGRATING {
				return nil
			}

			tb.SideEffect(&deployer_tpb.TriggerDeploymentMessage{
				DeploymentId:    state.CurrentDeployment.DeploymentId,
				StackId:         state.StackId,
				Version:         state.CurrentDeployment.Version,
				EnvironmentName: state.EnvironmentName,
				ApplicationName: state.ApplicationName,
			})

			return nil
		}))

	// BROKEN --> BROKEN : Triggered
	// CREATING --> CREATING : Triggered
	// MIGRATING --> MIGRATING : Triggered
	sm.From(
		deployer_pb.StackStatus_BROKEN,
		deployer_pb.StackStatus_CREATING,
		deployer_pb.StackStatus_MIGRATING,
	).Transition(deployer_pb.StackPSMTransition(func(
		ctx context.Context,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEventType_Triggered,
	) error {
		// No state change.
		state.QueuedDeployments = append(state.QueuedDeployments, event.Deployment)
		return nil
	}))
	return sm, nil
}
