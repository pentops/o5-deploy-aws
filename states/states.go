package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	deployment.From().Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Created,
	) error {

		stackEvent := &deployer_pb.StackEvent{
			Keys: &deployer_pb.StackKeys{
				StackId: StackID(state.Spec.EnvironmentName, state.Spec.AppName),
			},
			Metadata: &psm_pb.EventMetadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
			},
			Event: &deployer_pb.StackEventType{
				Type: &deployer_pb.StackEventType_Triggered_{
					Triggered: &deployer_pb.StackEventType_Triggered{
						Deployment: &deployer_pb.StackDeployment{
							DeploymentId: state.Keys.DeploymentId,
							Version:      state.Spec.Version,
						},
						EnvironmentName: state.Spec.EnvironmentName,
						EnvironmentId:   state.Spec.EnvironmentId,
						ApplicationName: state.Spec.AppName,
					},
				},
			},
		}

		if _, err := stack.TransitionInTx(ctx, tx, stackEvent); err != nil {
			return err
		}

		return nil
	}))

	deploymentCompleted := func(ctx context.Context, tx sqrlx.Transaction, state *deployer_pb.DeploymentState) error {

		stackEvent := &deployer_pb.StackEvent{
			Keys: &deployer_pb.StackKeys{
				StackId: StackID(state.Spec.EnvironmentName, state.Spec.AppName),
			},
			Metadata: &psm_pb.EventMetadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
			},
			Event: &deployer_pb.StackEventType{
				Type: &deployer_pb.StackEventType_DeploymentCompleted_{
					DeploymentCompleted: &deployer_pb.StackEventType_DeploymentCompleted{
						Deployment: &deployer_pb.StackDeployment{
							DeploymentId: state.Keys.DeploymentId,
							Version:      state.Spec.Version,
						},
					},
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
		return deploymentCompleted(ctx, tx, state)
	}))
	deployment.From().Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Terminated,
	) error {
		return deploymentCompleted(ctx, tx, state)
	}))
	deployment.From().Hook(deployer_pb.DeploymentPSMHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		tb deployer_pb.DeploymentPSMHookBaton,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEventType_Done,
	) error {
		return deploymentCompleted(ctx, tx, state)
	}))

	return &StateMachines{
		Deployment:  deployment,
		Environment: environment,
		Stack:       stack,
	}, nil
}
