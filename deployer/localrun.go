package deployer

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LocalStateStore wires back the events to the deployer, rather than relying on
// an event bus and database
type LocalStateStore struct {
	stackMap map[string]string // stackName -> deploymentId

	deployments map[string]*deployer_pb.DeploymentState
	eg          errgroup.Group

	AWSRunner *AWSRunner

	DeployerEvent func(ctx context.Context, deployment *deployer_pb.DeploymentEvent) error
}

func NewLocalStateStore(clientSet ClientBuilder) *LocalStateStore {

	return &LocalStateStore{
		stackMap:    map[string]string{},
		deployments: map[string]*deployer_pb.DeploymentState{},
		eg:          errgroup.Group{},
		AWSRunner: &AWSRunner{
			Clients: clientSet,
		},
	}
}

func (lss *LocalStateStore) handleInfraEvent(ctx context.Context, msg proto.Message) error {

	switch msg := msg.(type) {
	case *deployer_tpb.StackStatusChangedMessage:
		lifecycle, err := stackLifecycle(types.StackStatus(msg.Status))
		if err != nil {
			return err
		}

		deploymentId, ok := lss.stackMap[msg.StackName]
		if !ok {
			return fmt.Errorf("missing deploymentId for stack %s", msg.StackName)
		}

		event := &deployer_pb.DeploymentEvent{
			DeploymentId: deploymentId,
			Metadata: &deployer_pb.EventMetadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
			},
			Event: &deployer_pb.DeploymentEventType{
				Type: &deployer_pb.DeploymentEventType_StackStatus_{
					StackStatus: &deployer_pb.DeploymentEventType_StackStatus{
						Lifecycle:  lifecycle,
						FullStatus: msg.Status,
					},
				},
			},
		}
		return lss.ChainNextEvent(ctx, event)

	case *deployer_tpb.MigrationStatusChangedMessage:

		deployment, ok := lss.deployments[msg.DeploymentId]
		if !ok {
			return fmt.Errorf("missing deployment for migration %s", msg.MigrationId)
		}

		event := newEvent(deployment, &deployer_pb.DeploymentEventType_DbMigrateStatus{
			DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
				MigrationId: msg.MigrationId,
				Status:      msg.Status,
			},
		})
		return lss.ChainNextEvent(ctx, event)
	default:
		return fmt.Errorf("unknown infra event message: %T", msg)
	}

}

func (lss *LocalStateStore) QueueSideEffect(ctx context.Context, msg proto.Message) error {

	ctx = log.WithFields(ctx, map[string]interface{}{
		"inputMessage": msg.ProtoReflect().Descriptor().FullName(),
	})
	log.Debug(ctx, "QueueSideEffect")
	lss.eg.Go(func() error {
		defer log.Debug(ctx, "QueueSideEffect Complete")

		switch msg := msg.(type) {
		case *deployer_tpb.UpdateStackMessage:
			_, err := lss.AWSRunner.UpdateStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)

		case *deployer_tpb.CreateNewStackMessage:
			_, err := lss.AWSRunner.CreateNewStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)

		case *deployer_tpb.DeleteStackMessage:
			_, err := lss.AWSRunner.DeleteStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)

		case *deployer_tpb.ScaleStackMessage:
			_, err := lss.AWSRunner.ScaleStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)

		case *deployer_tpb.CancelStackUpdateMessage:
			_, err := lss.AWSRunner.CancelStackUpdate(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)

		case *deployer_tpb.UpsertSNSTopicsMessage:
			_, err := lss.AWSRunner.UpsertSNSTopics(ctx, msg)
			return err

		case *deployer_tpb.RunDatabaseMigrationMessage:

			if err := lss.handleInfraEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
				MigrationId:  msg.MigrationId,
				DeploymentId: msg.DeploymentId,
				Status:       deployer_pb.DatabaseMigrationStatus_PENDING,
			}); err != nil {
				return err
			}

			_, migrateErr := lss.AWSRunner.RunDatabaseMigration(ctx, msg)

			log.Debug(ctx, "RunDatabaseMigration is complete")
			if migrateErr != nil {
				log.WithError(ctx, migrateErr).Error("RunDatabaseMigration")
				if err := lss.handleInfraEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
					MigrationId:  msg.MigrationId,
					DeploymentId: msg.DeploymentId,
					Status:       deployer_pb.DatabaseMigrationStatus_FAILED,
					Error:        proto.String(migrateErr.Error()),
				}); err != nil {
					return err
				}
				return migrateErr
			}

			if err := lss.handleInfraEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
				DeploymentId: msg.DeploymentId,
				MigrationId:  msg.MigrationId,
				Status:       deployer_pb.DatabaseMigrationStatus_COMPLETED,
			}); err != nil {
				return err
			}

			return nil

		default:
			return fmt.Errorf("unknown side effect message: %T", msg)
		}
	})

	return nil

}

func (lss *LocalStateStore) ChainNextEvent(ctx context.Context, evt *deployer_pb.DeploymentEvent) error {
	lss.eg.Go(func() error {
		ctx := log.WithFields(ctx, map[string]interface{}{
			"deploymentId": evt.DeploymentId,
			"eventType":    fmt.Sprintf("%T", evt.Event.Type),
		})

		return lss.DeployerEvent(ctx, evt)
	})

	return nil
}

func (lss *LocalStateStore) StoreDeploymentEvent(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {

	// Special case handlers for local only, registers pending callbacks
	// to the deployment which triggered them.

	switch event.Event.Type.(type) {
	case *deployer_pb.DeploymentEventType_GotLock_:
		// Registers THIS deployment as THE deployment for the stack
		lss.stackMap[state.StackName] = state.DeploymentId

		// Trigger a stack poll. This will run until the stack is stable
		// Will trigger the current status back once, which is required
		// for when the stack is already stable before we get the lock
		err := lss.PollStack(ctx, state.StackName)
		if err != nil {
			return err
		}
	}

	lss.deployments[state.DeploymentId] = state
	return nil
}

func (lss *LocalStateStore) GetDeployment(ctx context.Context, id string) (*deployer_pb.DeploymentState, error) {
	if deployment, ok := lss.deployments[id]; ok {
		return deployment, nil
	}
	return nil, DeploymentNotFoundError
}

func (lss *LocalStateStore) Wait() error {
	return lss.eg.Wait()
}

func (lss *LocalStateStore) PollStack(ctx context.Context, stackName string) error {

	ctx = log.WithField(ctx, "stackName", stackName)

	log.Debug(ctx, "PollStack Begin")

	lss.eg.Go(func() error {
		var lastStatus types.StackStatus
		for {
			remoteStack, err := lss.AWSRunner.getOneStack(ctx, stackName)
			if err != nil {
				return err
			}
			if remoteStack == nil {
				return fmt.Errorf("missing stack %s", stackName)
			}

			if lastStatus == remoteStack.StackStatus {
				time.Sleep(5 * time.Second)
				continue
			}

			lastStatus = remoteStack.StackStatus

			log.WithFields(ctx, map[string]interface{}{
				"stackStatus": remoteStack.StackStatus,
			}).Debug("PollStack Result")

			if err := lss.handleInfraEvent(ctx, &deployer_tpb.StackStatusChangedMessage{
				StackName: *remoteStack.StackName,
				Status:    string(remoteStack.StackStatus),
			}); err != nil {
				return err
			}

			summary, err := summarizeStackStatus(remoteStack)
			if err != nil {
				return err
			}

			if summary.Stable {
				break
			}
		}

		return nil
	})

	return nil
}
