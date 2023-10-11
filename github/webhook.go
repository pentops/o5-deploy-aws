package github

import (
	"context"
	"fmt"

	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type IClient interface {
	PullO5Configs(ctx context.Context, org string, repo string, ref string) ([]*application_pb.Application, error)
}

type IDeployer interface {
	Deploy(context.Context, *app.BuiltApplication, bool) error
}

type RefMatcher interface {
	PushTargets(*github_pb.PushMessage) []string
}

type WebhookWorker struct {
	github    IClient
	deployers map[string]IDeployer
	refs      RefMatcher

	github_pb.UnimplementedWebhookTopicServer
}

func NewWebhookWorker(githubClient IClient, deployers map[string]IDeployer, refs RefMatcher) (*WebhookWorker, error) {
	return &WebhookWorker{
		github:    githubClient,
		deployers: deployers,
		refs:      refs,
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

	targetEnvs := make([]IDeployer, len(targetEnvNames))
	for i, envName := range targetEnvNames {

		envDeployer, ok := ww.deployers[envName]
		if !ok {
			return nil, fmt.Errorf("no deployer found for environment %s", envName)
		}

		targetEnvs[i] = envDeployer
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

	for _, targetEnv := range targetEnvs {
		if err := targetEnv.Deploy(ctx, built, false); err != nil {
			return nil, err
		}
	}
	return &emptypb.Empty{}, nil
}
