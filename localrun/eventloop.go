package localrun

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/proto"
)

// LocalEventLoop emulates AWS infrastructure by running all handlers in the
// same process. Used when running as a standalone tool, e.g. when
// bootstrapping a new environment.
type LocalEventLoop struct {
	messages  chan proto.Message
	handlers  map[string]func(context.Context, proto.Message) error
	validator *protovalidate.Validator
	wg        *sync.WaitGroup
}

func NewLocalEventLoop() *LocalEventLoop {
	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}
	return &LocalEventLoop{
		validator: validator,
		messages:  make(chan proto.Message),
		handlers:  map[string]func(context.Context, proto.Message) error{},
		wg:        &sync.WaitGroup{},
	}
}

// PublishEvent adds the event into the processing queue, blocking the Wait.
func (lel *LocalEventLoop) PublishEvent(ctx context.Context, msg outbox.OutboxMessage) error {
	if err := lel.validator.Validate(msg); err != nil {
		return err
	}
	log.WithField(ctx, "inputMessage", msg.ProtoReflect().Descriptor().FullName()).Debug("PublishEvent")

	return lel.handleMessage(ctx, msg)
}

// Wait runs the event loop. It exits when all handlers have completed, or any
// handler returns an error. Messages which are queued in PublishEvent prior to
// a handle exiting extends the lifetime of the event loop.
func (lel *LocalEventLoop) Wait(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	chErr := make(chan error)

	go func() {
		for {
			select {
			case <-ctx.Done():
				chErr <- nil
			case msg := <-lel.messages:
				go func(msg proto.Message) {
					err := lel.handleMessage(ctx, msg)
					if err != nil {
						chErr <- err
					}
				}(msg)
			}
		}
	}()

	err := <-chErr
	if err != nil {
		log.WithError(ctx, err).Error("Event Loop Exit")
	} else {
		log.Info(ctx, "Event Loop Exit")
	}
	cancel()
	return err
}

func (lel *LocalEventLoop) handleMessage(ctx context.Context, msg proto.Message) error {
	msgKey := string(msg.ProtoReflect().Descriptor().FullName())
	handler, ok := lel.handlers[msgKey]
	if !ok {
		return fmt.Errorf("no handler for message %s", msgKey)
	}

	handlerContext := log.WithFields(ctx, map[string]interface{}{
		"inputMessage": msg.ProtoReflect().Descriptor().FullName(),
	})

	if err := handler(handlerContext, msg); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.WithError(handlerContext, err).Error("Event Loop Handler Error")
		}
		return err
	}
	return nil
}

func (lel *LocalEventLoop) RegisterHandler(fullName string, handler func(context.Context, proto.Message) error) {
	lel.handlers[fullName] = handler
}

func wrapHandler[T proto.Message](handler func(context.Context, T) error) (string, func(context.Context, proto.Message) error) {
	msg := *new(T) // wrapHandler exists for this magic line.
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

// RegisterDeployerHandlers takes the proto interface of deployer, and registers
// the known callbacks to the event loop. This mirrors the standard 'register'
// pattern for gRPC workers.
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

// RegisterLocalHandlers takes the specific LocalStateStore implementation, as
// the LSS uses polling loops after triggering AWS actions to emulate
// subscription to AWS events over sns/sqs.
func RegisterLocalHandlers(eventGroup *LocalEventLoop, awsRunner *awsinfra.AWSRunner) error {

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) error {
			_, err := awsRunner.UpdateStack(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) error {
			_, err := awsRunner.CreateNewStack(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.DeleteStackMessage) error {
			_, err := awsRunner.DeleteStack(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) error {
			_, err := awsRunner.ScaleStack(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.CancelStackUpdateMessage) error {
			_, err := awsRunner.CancelStackUpdate(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) error {
			_, err := awsRunner.StabalizeStack(ctx, msg)
			return err
		}))

	eventGroup.RegisterHandler(wrapHandler(
		func(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) error {
			_, err := awsRunner.RunDatabaseMigration(ctx, msg)
			return err
		}))

	return nil
}
