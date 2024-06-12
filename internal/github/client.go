package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/pentops/envconf.go/envconf"
	"github.com/pentops/o5-deploy-aws/internal/protoread"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"golang.org/x/oauth2"

	"github.com/google/go-github/v47/github"
)

type Client struct {
	repositories RepositoriesService
}

type RepositoriesService interface {
	DownloadContents(ctx context.Context, owner, repo, filepath string, opts *github.RepositoryContentGetOptions) (io.ReadCloser, *github.Response, error)
	ListByOrg(context.Context, string, *github.RepositoryListByOrgOptions) ([]*github.Repository, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
	GetBranch(ctx context.Context, owner, repo, branch string, followRedirects bool) (*github.Branch, *github.Response, error)
}

type AppConfig struct {
	OrgName        string `json:"orgName"`
	Public         bool   `json:"public,omitempty"`
	PrivateKey     string `json:"privateKey,omitempty"`
	AppID          int64  `json:"appId,omitempty"`
	InstallationID int64  `json:"installationId,omitempty"`
}

func NewEnvClient(ctx context.Context) (*Client, error) {

	if os.Getenv("GH_PRIVATE_KEY") != "" {
		config := struct {
			GithubPrivateKey     string `env:"GH_PRIVATE_KEY"`
			GithubAppID          int64  `env:"GH_APP_ID"`
			GithubInstallationID int64  `env:"GH_INSTALLATION_ID"`
		}{}

		if err := envconf.Parse(&config); err != nil {
			return nil, err
		}
		if config.GithubAppID == 0 || config.GithubInstallationID == 0 {
			return nil, fmt.Errorf("no github app id or installation id")
		}

		return NewAppClient(AppConfig{
			AppID:          config.GithubAppID,
			InstallationID: config.GithubInstallationID,
			PrivateKey:     config.GithubPrivateKey,
		})

	} else if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		return NewClient(tc)
	}
	return nil, fmt.Errorf("no valid github config in environment")
}

func NewAppClient(config AppConfig) (*Client, error) {
	httpClient := &http.Client{}
	if !config.Public {
		tr := http.DefaultTransport
		privateKey, err := base64.StdEncoding.DecodeString(config.PrivateKey)
		if err != nil {
			return nil, err
		}

		itr, err := ghinstallation.New(tr, config.AppID, config.InstallationID, privateKey)
		if err != nil {
			return nil, err
		}
		httpClient.Transport = itr
	}

	return NewClient(httpClient)
}

func NewClient(tc *http.Client) (*Client, error) {
	ghcl := github.NewClient(tc)
	cl := &Client{
		repositories: ghcl.Repositories,
	}

	return cl, nil
}

type Commit struct {
	Commit string
}

func (cl Client) BranchCommit(ctx context.Context, org, repo, ref string) (*Commit, error) {

	rr, _, err := cl.repositories.GetBranch(ctx, org, repo, ref, true)
	if err != nil {
		return nil, err
	}

	return &Commit{
		Commit: *rr.Commit.SHA,
	}, nil
}

func (cl Client) PullO5Configs(ctx context.Context, org string, repo string, ref string) ([]*application_pb.Application, error) {
	apps := make([]*application_pb.Application, 0, 1)
	opts := &github.RepositoryContentGetOptions{
		Ref: ref,
	}

	_, dirContent, _, err := cl.repositories.GetContents(ctx, org, repo, "ext/o5", opts)
	if err != nil {
		errResp, ok := err.(*github.ErrorResponse)
		if ok && errResp.Response.StatusCode == 404 {
			return nil, nil
		}

		return nil, fmt.Errorf("repositories: get contents: %w", err)
	}

	if len(dirContent) == 0 {
		return nil, nil
	}

	for _, content := range dirContent {
		file, _, err := cl.repositories.DownloadContents(ctx, org, repo, *content.Path, opts)
		if err != nil {
			return nil, fmt.Errorf("repositories: download contents: %w", err)
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("reading bytes: %s", err)
		}

		app := &application_pb.Application{}
		err = protoread.Parse(path.Base(*content.Path), data, app)
		if err != nil {
			return nil, fmt.Errorf("parse app: %w", err)
		}

		apps = append(apps, app)
	}

	return apps, nil
}

type MultiOrgClient struct {
	clients map[string]*Client
}

func NewMultiOrgClient(clients map[string]*Client) (*MultiOrgClient, error) {
	return &MultiOrgClient{
		clients: clients,
	}, nil
}

func NewMultiOrgClientFromConfigs(configs ...AppConfig) (*MultiOrgClient, error) {
	clients := make(map[string]*Client, len(configs))
	for _, config := range configs {
		client, err := NewAppClient(config)
		if err != nil {
			return nil, err
		}

		clients[config.OrgName] = client
	}

	return &MultiOrgClient{
		clients: clients,
	}, nil
}

func (mc MultiOrgClient) PullO5Configs(ctx context.Context, org string, repo string, ref string) ([]*application_pb.Application, error) {
	client, ok := mc.clients[org]
	if !ok {
		return nil, fmt.Errorf("no github client for org %s", org)
	}

	return client.PullO5Configs(ctx, org, repo, ref)
}
