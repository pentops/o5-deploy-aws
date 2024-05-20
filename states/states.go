package states

import (
	"context"
	"fmt"
	"time"

	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_epb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type StateMachines struct {
	Deployment  *deployer_pb.DeploymentPSM
	Environment *deployer_pb.EnvironmentPSM
	Stack       *deployer_pb.StackPSM
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

	deployment.GeneralHook(deployer_pb.DeploymentPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEvent,
	) error {
		publish := &deployer_epb.DeploymentEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		return outbox.Send(ctx, tx, publish)
	}))

	stack.GeneralHook(deployer_pb.StackPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton deployer_pb.StackPSMHookBaton,
		state *deployer_pb.StackState,
		event *deployer_pb.StackEvent,
	) error {
		publish := &deployer_epb.StackEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		return outbox.Send(ctx, tx, publish)
	}))

	environment.GeneralHook(deployer_pb.EnvironmentPSMGeneralHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		baton deployer_pb.EnvironmentPSMHookBaton,
		state *deployer_pb.EnvironmentState,
		event *deployer_pb.EnvironmentEvent,
	) error {
		publish := &deployer_epb.EnvironmentEvent{
			Event:  event,
			Status: state.Status,
			State:  state.Data,
		}
		return outbox.Send(ctx, tx, publish)
	}))

	// Unblock waiting deployments from the stack trigger.
	stack.From(deployer_pb.StackStatus_AVAILABLE).
		Hook(deployer_pb.StackPSMHook(func(
			ctx context.Context,
			tx sqrlx.Transaction,
			tb deployer_pb.StackPSMHookBaton,
			state *deployer_pb.StackState,
			event *deployer_pb.StackEventType_RunDeployment,
		) error {

			triggerDeploymentEvent := &deployer_pb.DeploymentPSMEventSpec{
				Keys: &deployer_pb.DeploymentKeys{
					DeploymentId: event.DeploymentId,
				},
				Cause:     tb.AsCause(),
				Timestamp: time.Now(),
				Event:     &deployer_pb.DeploymentEventType_Triggered{},
			}

			if _, err := deployment.TransitionInTx(ctx, tx, triggerDeploymentEvent); err != nil {
				return err
			}

			return nil

		}))

	// Push deployments into the stack queue.
	deployment.From(0).Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Created,
	) error {

		stackEvent := &deployer_pb.StackPSMEventSpec{
			Keys: &deployer_pb.StackKeys{
				StackId: StackID(state.Data.Spec.EnvironmentName, state.Data.Spec.AppName),
			},
			Timestamp: time.Now(),
			Cause:     tb.AsCause(),
			Event: &deployer_pb.StackEventType_DeploymentRequested{
				Deployment: &deployer_pb.StackDeployment{
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

	deploymentCompleted := func(ctx context.Context, tx sqrlx.Transaction, state *deployer_pb.DeploymentState, cause *psm_pb.Cause) error {

		stackEvent := &deployer_pb.StackPSMEventSpec{
			Keys: &deployer_pb.StackKeys{
				StackId: StackID(state.Data.Spec.EnvironmentName, state.Data.Spec.AppName),
			},
			Timestamp: time.Now(),
			Cause:     cause,
			Event: &deployer_pb.StackEventType_DeploymentCompleted{
				Deployment: &deployer_pb.StackDeployment{
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

	deployment.From().Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Error,
	) error {
		return deploymentCompleted(ctx, tx, state, tb.AsCause())
	}))
	deployment.From().Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Terminated,
	) error {
		return deploymentCompleted(ctx, tx, state, tb.AsCause())
	}))
	deployment.From().Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Done,
	) error {
		return deploymentCompleted(ctx, tx, state, tb.AsCause())
	}))

	return &StateMachines{
		Deployment:  deployment,
		Environment: environment,
		Stack:       stack,
	}, nil
}
