package github

import (
	"context"
	"fmt"

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

type WebhookWorker struct {
	github   IClient
	deployer IDeployer

	github_pb.UnimplementedWebhookTopicServer
}

func NewWebhookWorker(githubClient IClient, deployer IDeployer) (*WebhookWorker, error) {
	return &WebhookWorker{
		github:   githubClient,
		deployer: deployer,
	}, nil
}

func (ww *WebhookWorker) Push(ctx context.Context, event *github_pb.PushMessage) (*emptypb.Empty, error) {

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

	return &emptypb.Empty{}, ww.deployer.Deploy(ctx, built, true)
}
