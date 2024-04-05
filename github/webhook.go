package github

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/outbox.pg.go/outbox"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type IClient interface {
	PullO5Configs(ctx context.Context, org string, repo string, commit string) ([]*application_pb.Application, error)
}

type WebhookWorker struct {
	github IClient
	refs   RefMatcher
	db     *sqrlx.Wrapper

	github_pb.UnimplementedWebhookTopicServer
}

func NewWebhookWorker(conn sqrlx.Connection, githubClient IClient, refs RefMatcher) (*WebhookWorker, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}
	return &WebhookWorker{
		github: githubClient,
		refs:   refs,
		db:     db,
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

	targetEnvs, err := ww.refs.PushTargets(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("target envs: %w", err)
	}

	if len(targetEnvs) < 1 {
		log.Info(ctx, "No refs match, nothing to do")
		return &emptypb.Empty{}, nil
	}

	apps, err := ww.github.PullO5Configs(ctx, event.Owner, event.Repo, event.After)
	if err != nil {
		return nil, fmt.Errorf("github push: pull o5 config: %w", err)
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found in push event")
	}

	if len(apps) > 1 {
		return nil, fmt.Errorf("multiple applications found in push event, not yet supported")
	}

	var triggers []outbox.OutboxMessage

	for _, envID := range targetEnvs {

		triggers = append(triggers, &deployer_tpb.RequestDeploymentMessage{
			DeploymentId:  uuid.NewString(),
			Application:   apps[0],
			Version:       event.After,
			EnvironmentId: envID,
		})
	}

	err = ww.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		for _, trigger := range triggers {
			if err := outbox.Send(ctx, tx, trigger); err != nil {
				return fmt.Errorf("request deployment: outbox: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
