package states

import (
	"context"
	"fmt"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_epb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
)

type StateMachines struct {
	Deployment  *awsdeployer_pb.DeploymentPSM
	Environment *awsdeployer_pb.EnvironmentPSM
	Cluster     *awsdeployer_pb.ClusterPSM
	Stack       *awsdeployer_pb.StackPSM
}

func NewStateMachines() (*StateMachines, error) {
	deployment, err := NewDeploymentEventer()
	if err != nil {
		return nil, fmt.Errorf("NewDeploymentEventer: %w", err)
	}

	environment, err := NewEnvironmentEventer()
	if err != nil {
		return nil, fmt.Errorf("NewEnvironmentEventer: %w", err)
	}

	stack, err := NewStackEventer()
	if err != nil {
		return nil, fmt.Errorf("NewStackEventer: %w", err)
	}

	cluster, err := NewClusterEventer()
	if err != nil {
		return nil, fmt.Errorf("NewClusterEventer: %w", err)
	}

	deployment.LogicHook(awsdeployer_pb.DeploymentPSMGeneralLogicHook(func(
		ctx context.Context,
		baton awsdeployer_pb.DeploymentPSMHookBaton,
		state *awsdeployer_pb.DeploymentState,
		event *awsdeployer_pb.DeploymentEvent,
	) error {
		publish := &awsdeployer_epb.DeploymentEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		baton.SideEffect(publish)
		return nil
	}))

	stack.LogicHook(awsdeployer_pb.StackPSMGeneralLogicHook(func(
		ctx context.Context,
		baton awsdeployer_pb.StackPSMHookBaton,
		state *awsdeployer_pb.StackState,
		event *awsdeployer_pb.StackEvent,
	) error {
		publish := &awsdeployer_epb.StackEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		baton.SideEffect(publish)
		return nil
	}))

	environment.LogicHook(awsdeployer_pb.EnvironmentPSMGeneralLogicHook(func(
		ctx context.Context,
		baton awsdeployer_pb.EnvironmentPSMHookBaton,
		state *awsdeployer_pb.EnvironmentState,
		event *awsdeployer_pb.EnvironmentEvent,
	) error {
		publish := &awsdeployer_epb.EnvironmentEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		baton.SideEffect(publish)
		return nil
	}))

	// Unblock waiting deployments from the stack trigger.
	stack.From(awsdeployer_pb.StackStatus_AVAILABLE).
		LinkTo(awsdeployer_pb.StackPSMLinkHook(deployment, func(
			ctx context.Context,
			state *awsdeployer_pb.StackState,
			event *awsdeployer_pb.StackEventType_RunDeployment,
		) (*awsdeployer_pb.DeploymentKeys, awsdeployer_pb.DeploymentPSMEvent, error) {

			keys := &awsdeployer_pb.DeploymentKeys{
				DeploymentId:  event.DeploymentId,
				StackId:       state.Keys.StackId,
				EnvironmentId: state.Keys.EnvironmentId,
				ClusterId:     state.Keys.ClusterId,
			}
			chain := &awsdeployer_pb.DeploymentEventType_Triggered{}

			return keys, chain, nil

		}))

	// Push deployments into the stack queue.
	deployment.From(0).
		LinkTo(awsdeployer_pb.DeploymentPSMLinkHook(stack, func(
			ctx context.Context,
			state *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Created,
		) (*awsdeployer_pb.StackKeys, awsdeployer_pb.StackPSMEvent, error) {

			keys := &awsdeployer_pb.StackKeys{
				StackId:       StackID(state.Data.Spec.EnvironmentName, state.Data.Spec.AppName),
				EnvironmentId: state.Keys.EnvironmentId,
				ClusterId:     state.Keys.ClusterId,
			}

			chain := &awsdeployer_pb.StackEventType_DeploymentRequested{
				Deployment: &awsdeployer_pb.StackDeployment{
					DeploymentId: state.Keys.DeploymentId,
					Version:      state.Data.Spec.Version,
				},
				EnvironmentName: state.Data.Spec.EnvironmentName,
				EnvironmentId:   state.Data.Spec.EnvironmentId,
				ApplicationName: state.Data.Spec.AppName,
			}

			return keys, chain, nil
		}))

	deploymentCompleted := func(state *awsdeployer_pb.DeploymentState) (*awsdeployer_pb.StackKeys, awsdeployer_pb.StackPSMEvent, error) {

		keys := &awsdeployer_pb.StackKeys{
			StackId:       StackID(state.Data.Spec.EnvironmentName, state.Data.Spec.AppName),
			EnvironmentId: state.Keys.EnvironmentId,
			ClusterId:     state.Keys.ClusterId,
		}
		event := &awsdeployer_pb.StackEventType_DeploymentCompleted{
			Deployment: &awsdeployer_pb.StackDeployment{
				DeploymentId: state.Keys.DeploymentId,
				Version:      state.Data.Spec.Version,
			},
		}

		return keys, event, nil
	}

	deployment.From().
		LinkTo(awsdeployer_pb.DeploymentPSMLinkHook(stack, func(
			ctx context.Context,
			state *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Error,
		) (*awsdeployer_pb.StackKeys, awsdeployer_pb.StackPSMEvent, error) {
			return deploymentCompleted(state)
		}))

	deployment.From().
		LinkTo(awsdeployer_pb.DeploymentPSMLinkHook(stack, func(
			ctx context.Context,
			state *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Terminated,
		) (*awsdeployer_pb.StackKeys, awsdeployer_pb.StackPSMEvent, error) {
			return deploymentCompleted(state)
		}))

	deployment.From().
		LinkTo(awsdeployer_pb.DeploymentPSMLinkHook(stack, func(
			ctx context.Context,
			state *awsdeployer_pb.DeploymentState,
			event *awsdeployer_pb.DeploymentEventType_Done,
		) (*awsdeployer_pb.StackKeys, awsdeployer_pb.StackPSMEvent, error) {
			return deploymentCompleted(state)
		}))

	return &StateMachines{
		Deployment:  deployment,
		Environment: environment,
		Cluster:     cluster,
		Stack:       stack,
	}, nil
}
