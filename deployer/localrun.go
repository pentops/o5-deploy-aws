package deployer

import (
	"context"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

type LocalEventLoop struct {
	eg       errgroup.Group
	handlers map[string]func(context.Context, proto.Message) error
}

func NewLocalEventLoop() *LocalEventLoop {
	return &LocalEventLoop{
		handlers: map[string]func(context.Context, proto.Message) error{},
	}
}

func (lel *LocalEventLoop) Wait() error {
	return lel.eg.Wait()
}

func (lel *LocalEventLoop) RegisterHandler(fullName string, handler func(context.Context, proto.Message) error) {
	lel.handlers[fullName] = handler
}

func wrapHandler[T proto.Message](handler func(context.Context, T) error) (string, func(context.Context, proto.Message) error) {
	msg := *new(T)
	fullName := string(msg.ProtoReflect().Descriptor().FullName())
	return fullName, func(ctx context.Context, msg proto.Message) error {
		ctx = log.WithFields(ctx, map[string]interface{}{
			"inputMessage": msg.ProtoReflect().Descriptor().FullName(),
		})
		typed, ok := msg.(T)
		if !ok {
			return fmt.Errorf("unexpected message type: %T", msg)
		}
		return handler(ctx, typed)
	}
}

func (lel *LocalEventLoop) Run(ctx context.Context, store *LocalStateStore) error {
	return lel.eg.Wait()
}

func (lel *LocalEventLoop) PublishEvent(ctx context.Context, msg proto.Message) error {
	msgKey := string(msg.ProtoReflect().Descriptor().FullName())
	handler, ok := lel.handlers[msgKey]
	if !ok {
		return fmt.Errorf("no handler for message %s", msgKey)
	}

	lel.eg.Go(func() error {
		handlerContext := log.WithFields(context.Background(), map[string]interface{}{
			"inputMessage": msg.ProtoReflect().Descriptor().FullName(),
		})

		return handler(handlerContext, msg)
	})
	return nil
}

func RegisterDeployerHandlers(eventGroup *LocalEventLoop, deployer deployer_tpb.DeployerTopicServer) error {

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.StackStatusChangedMessage) error {
			_, err := deployer.StackStatusChanged(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.MigrationStatusChangedMessage) error {
			_, err := deployer.MigrationStatusChanged(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.TriggerDeploymentMessage) error {
			_, err := deployer.TriggerDeployment(ctx, msg)
			return err
		}))

	return nil

}

func RegisterLocalHandlers(eventGroup *LocalEventLoop, lss *LocalStateStore) error {

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) error {
			_, err := lss.AWSRunner.UpdateStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) error {

			_, err := lss.AWSRunner.CreateNewStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.DeleteStackMessage) error {
			_, err := lss.AWSRunner.DeleteStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) error {
			_, err := lss.AWSRunner.ScaleStack(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.CancelStackUpdateMessage) error {
			_, err := lss.AWSRunner.CancelStackUpdate(ctx, msg)
			if err != nil {
				return err
			}
			return lss.PollStack(ctx, msg.StackName)
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.UpsertSNSTopicsMessage) error {
			_, err := lss.AWSRunner.UpsertSNSTopics(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) error {
			_, err := lss.AWSRunner.StabalizeStack(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) error {

			if err := eventGroup.PublishEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
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
				if err := eventGroup.PublishEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
					MigrationId:  msg.MigrationId,
					DeploymentId: msg.DeploymentId,
					Status:       deployer_pb.DatabaseMigrationStatus_FAILED,
					Error:        proto.String(migrateErr.Error()),
				}); err != nil {
					return err
				}
				return migrateErr
			}

			if err := eventGroup.PublishEvent(ctx, &deployer_tpb.MigrationStatusChangedMessage{
				DeploymentId: msg.DeploymentId,
				MigrationId:  msg.MigrationId,
				Status:       deployer_pb.DatabaseMigrationStatus_COMPLETED,
			}); err != nil {
				return err
			}

			return nil

		}))

	return nil
}

// LocalStateStore wires back the events to the deployer, rather than relying on
// an event bus and database
type LocalStateStore struct {
	stackMap map[string]string // stackName -> deploymentId

	deployments map[string]*deployer_pb.DeploymentState

	AWSRunner *awsinfra.AWSRunner
	eventLoop *LocalEventLoop
}

func NewLocalStateStore(clientSet awsinfra.ClientBuilder, eventLoop *LocalEventLoop) *LocalStateStore {
	lss := &LocalStateStore{
		stackMap:    map[string]string{},
		deployments: map[string]*deployer_pb.DeploymentState{},
		eventLoop:   eventLoop,
		AWSRunner: &awsinfra.AWSRunner{
			Clients:        clientSet,
			MessageHandler: eventLoop,
		},
	}
	return lss
}

func (lss *LocalStateStore) PublishEvent(ctx context.Context, msg proto.Message) error {
	return lss.eventLoop.PublishEvent(ctx, msg)
}

func (lss *LocalStateStore) PollStack(ctx context.Context, stackName string) error {
	return lss.AWSRunner.PollStack(ctx, stackName)
}

func (lss *LocalStateStore) StoreDeploymentEvent(ctx context.Context, state *deployer_pb.DeploymentState, event *deployer_pb.DeploymentEvent) error {

	// Special case handlers for local only, registers pending callbacks
	// to the deployment which triggered them.

	switch event.Event.Type.(type) {
	case *deployer_pb.DeploymentEventType_GotLock_:
		// Registers THIS deployment as THE deployment for the stack
		lss.stackMap[state.StackName] = state.DeploymentId
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

func (lss *LocalStateStore) GetDeploymentForStack(ctx context.Context, stackName string) (*deployer_pb.DeploymentState, error) {

	deploymentId, ok := lss.stackMap[stackName]
	if !ok {
		return nil, fmt.Errorf("missing deploymentId for stack %s", stackName)
	}

	return lss.GetDeployment(ctx, deploymentId)
}
