package states

import (
	"context"
	"fmt"
	"time"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_epb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-messaging/outbox"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
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

	deployment.GeneralHook(awsdeployer_pb.DeploymentPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton awsdeployer_pb.DeploymentPSMHookBaton,
		state *awsdeployer_pb.DeploymentState,
		event *awsdeployer_pb.DeploymentEvent,
	) error {
		publish := &awsdeployer_epb.DeploymentEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		return outbox.Send(ctx, tx, publish)
	}))

	stack.GeneralHook(awsdeployer_pb.StackPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton awsdeployer_pb.StackPSMHookBaton,
		state *awsdeployer_pb.StackState,
		event *awsdeployer_pb.StackEvent,
	) error {
		publish := &awsdeployer_epb.StackEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		return outbox.Send(ctx, tx, publish)
	}))

	environment.GeneralHook(awsdeployer_pb.EnvironmentPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton awsdeployer_pb.EnvironmentPSMHookBaton,
		state *awsdeployer_pb.EnvironmentState,
		event *awsdeployer_pb.EnvironmentEvent,
	) error {
		publish := &awsdeployer_epb.EnvironmentEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		return outbox.Send(ctx, tx, publish)
	}))

	// Unblock waiting deployments from the stack trigger.
	stack.From(awsdeployer_pb.StackStatus_AVAILABLE).
		Hook(awsdeployer_pb.StackPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb awsdeployer_pb.StackPSMHookBaton,
			state *awsdeployer_pb.StackState,
			event *awsdeployer_pb.StackEventType_RunDeployment,
		) error {

			triggerDeploymentEvent := &awsdeployer_pb.DeploymentPSMEventSpec{
				Keys: &awsdeployer_pb.DeploymentKeys{
					DeploymentId:  event.DeploymentId,
					StackId:       state.Keys.StackId,
					EnvironmentId: state.Keys.EnvironmentId,
					ClusterId:     state.Keys.ClusterId,
				},
				Cause:     tb.AsCause(),
				Timestamp: time.Now(),
				Event:     &awsdeployer_pb.DeploymentEventType_Triggered{},
			}

			if _, err := deployment.TransitionInTx(ctx, tx, triggerDeploymentEvent); err != nil {
				return err
			}

			return nil

		}))

	// Push deployments into the stack queue.
	deployment.From(0).Hook(awsdeployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb awsdeployer_pb.DeploymentPSMHookBaton,
		state *awsdeployer_pb.DeploymentState,
		event *awsdeployer_pb.DeploymentEventType_Created,
	) error {

		stackEvent := &awsdeployer_pb.StackPSMEventSpec{
			Keys: &awsdeployer_pb.StackKeys{
				StackId:       StackID(state.Data.Spec.EnvironmentName, state.Data.Spec.AppName),
				EnvironmentId: state.Keys.EnvironmentId,
				ClusterId:     state.Keys.ClusterId,
			},
			Timestamp: time.Now(),
			Cause:     tb.AsCause(),
			Event: &awsdeployer_pb.StackEventType_DeploymentRequested{
				Deployment: &awsdeployer_pb.StackDeployment{
					DeploymentId: state.Keys.DeploymentId,
					Version:      state.Data.Spec.Version,
				},
				EnvironmentName: state.Data.Spec.EnvironmentName,
				EnvironmentId:   state.Data.Spec.EnvironmentId,
				ApplicationName: state.Data.Spec.AppName,
			},
		}

		if _, err := stack.TransitionInTx(ctx, tx, stackEvent); err != nil {
			return err
		}

		return nil
	}))

	deploymentCompleted := func(ctx context.Context, tx sqrlx.Transaction, state *awsdeployer_pb.DeploymentState, cause *psm_pb.Cause) error {

		stackEvent := &awsdeployer_pb.StackPSMEventSpec{
			Keys: &awsdeployer_pb.StackKeys{
				StackId:       StackID(state.Data.Spec.EnvironmentName, state.Data.Spec.AppName),
				EnvironmentId: state.Keys.EnvironmentId,
				ClusterId:     state.Keys.ClusterId,
			},
			Timestamp: time.Now(),
			Cause:     cause,
			Event: &awsdeployer_pb.StackEventType_DeploymentCompleted{
				Deployment: &awsdeployer_pb.StackDeployment{
					DeploymentId: state.Keys.DeploymentId,
					Version:      state.Data.Spec.Version,
				},
			},
		}

		if _, err := stack.TransitionInTx(ctx, tx, stackEvent); err != nil {
			return err
		}

		return nil
	}

	deployment.From().Hook(awsdeployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb awsdeployer_pb.DeploymentPSMHookBaton,
		state *awsdeployer_pb.DeploymentState,
		event *awsdeployer_pb.DeploymentEventType_Error,
	) error {
		return deploymentCompleted(ctx, tx, state, tb.AsCause())
	}))
	deployment.From().Hook(awsdeployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb awsdeployer_pb.DeploymentPSMHookBaton,
		state *awsdeployer_pb.DeploymentState,
		event *awsdeployer_pb.DeploymentEventType_Terminated,
	) error {
		return deploymentCompleted(ctx, tx, state, tb.AsCause())
	}))
	deployment.From().Hook(awsdeployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb awsdeployer_pb.DeploymentPSMHookBaton,
		state *awsdeployer_pb.DeploymentState,
		event *awsdeployer_pb.DeploymentEventType_Done,
	) error {
		return deploymentCompleted(ctx, tx, state, tb.AsCause())
	}))

	return &StateMachines{
		Deployment:  deployment,
		Environment: environment,
		Cluster:     cluster,
		Stack:       stack,
	}, nil
}
