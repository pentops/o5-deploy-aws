package github

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.daemonl.com/sqrlx"
)

type IClient interface {
	PullO5Configs(ctx context.Context, org string, repo string, commit string) ([]*application_pb.Application, error)
}

type IDeployer interface {
	BuildTrigger(ctx context.Context, app *app.BuiltApplication, envName string) (*deployer_tpb.TriggerDeploymentMessage, error)
}

type RefMatcher interface {
	PushTargets(*github_pb.PushMessage) []string
}

type WebhookWorker struct {
	github   IClient
	deployer IDeployer
	refs     RefMatcher
	db       *sqrlx.Wrapper

	github_pb.UnimplementedWebhookTopicServer
}

func NewWebhookWorker(conn sqrlx.Connection, githubClient IClient, deployer IDeployer, refs RefMatcher) (*WebhookWorker, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}
	return &WebhookWorker{
		github:   githubClient,
		deployer: deployer,
		refs:     refs,
		db:       db,
	}, nil
}

func (ww *WebhookWorker) Push(ctx context.Context, event *github_pb.PushMessage) (*emptypb.Empty, error) {

	ctx = log.WithFields(ctx, map[string]interface{}{
		"owner":  event.Owner,
		"repo":   event.Repo,
		"ref":    event.Ref,
		"commit": event.After,
	})
	log.Debug(ctx, "Push")

	targetEnvNames := ww.refs.PushTargets(event)
	if len(targetEnvNames) < 1 {
		log.Info(ctx, "No refs match, nothing to do")
		return &emptypb.Empty{}, nil
	}

	apps, err := ww.github.PullO5Configs(ctx, event.Owner, event.Repo, event.After)
	if err != nil {
		return nil, err
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found in push event")
	}

	if len(apps) > 1 {
		return nil, fmt.Errorf("multiple applications found in push event, not yet supported")
	}

	appStack, err := app.BuildApplication(apps[0], event.After)
	if err != nil {
		return nil, err
	}

	built := appStack.Build()

	var triggers []outbox.OutboxMessage

	for _, envName := range targetEnvNames {
		trigger, err := ww.deployer.BuildTrigger(ctx, built, envName)
		if err != nil {
			return nil, err
		}
		triggers = append(triggers, trigger)
	}

	err = ww.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		for _, trigger := range triggers {
			if err := outbox.Send(ctx, tx, trigger); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
