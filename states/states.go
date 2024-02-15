package states

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type StateMachines struct {
	Deployment      *deployer_pb.DeploymentPSM
	Environment     *deployer_pb.EnvironmentPSM
	Stack           *deployer_pb.StackPSM
	PostgresMigrate *deployer_pb.PostgresMigrationPSM
	EcsTask         *deployer_pb.EcsTaskPSM
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

	postrgresMigrate, err := NewPostgresMigrateEventer()
	if err != nil {
		return nil, fmt.Errorf("NewPostgresMigrateEventer: %w", err)
	}

	ecsTask, err := NewEcsTaskEventer()
	if err != nil {
		return nil, fmt.Errorf("NewEcsTaskEventer: %w", err)
	}

	deployment.AddHook(func(
		ctx context.Context,
		tx sqrlx.Transaction,
		state *deployer_pb.DeploymentState,
		event *deployer_pb.DeploymentEvent,
	) error {
		switch event.UnwrapPSMEvent().(type) {

		case *deployer_pb.DeploymentEventType_Created:

			stackEvent := &deployer_pb.StackEvent{
				StackId: StackID(state.Spec.EnvironmentName, state.Spec.AppName),
				Metadata: &deployer_pb.EventMetadata{
					EventId:   uuid.NewString(),
					Timestamp: timestamppb.Now(),
				},
				Event: &deployer_pb.StackEventType{
					Type: &deployer_pb.StackEventType_Triggered_{
						Triggered: &deployer_pb.StackEventType_Triggered{
							Deployment: &deployer_pb.StackDeployment{
								DeploymentId: state.DeploymentId,
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

		case *deployer_pb.DeploymentEventType_Error,
			*deployer_pb.DeploymentEventType_Terminated,
			*deployer_pb.DeploymentEventType_Done:

			stackEvent := &deployer_pb.StackEvent{
				StackId: StackID(state.Spec.EnvironmentName, state.Spec.AppName),
				Metadata: &deployer_pb.EventMetadata{
					EventId:   uuid.NewString(),
					Timestamp: timestamppb.Now(),
				},
				Event: &deployer_pb.StackEventType{
					Type: &deployer_pb.StackEventType_DeploymentCompleted_{
						DeploymentCompleted: &deployer_pb.StackEventType_DeploymentCompleted{
							Deployment: &deployer_pb.StackDeployment{
								DeploymentId: state.DeploymentId,
								Version:      state.Spec.Version,
							},
						},
					},
				},
			}

			if _, err := stack.TransitionInTx(ctx, tx, stackEvent); err != nil {
				return err
			}

		}

		return nil
	})

	return &StateMachines{
		Deployment:      deployment,
		Environment:     environment,
		Stack:           stack,
		PostgresMigrate: postrgresMigrate,
		EcsTask:         ecsTask,
	}, nil
}
