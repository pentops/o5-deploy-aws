package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"golang.org/x/oauth2"
	"gopkg.daemonl.com/envconf"

	"github.com/google/go-github/v47/github"
)

type Client struct {
	repositories RepositoriesService
}
type RepositoriesService interface {
	DownloadContents(ctx context.Context, owner, repo, filepath string, opts *github.RepositoryContentGetOptions) (io.ReadCloser, *github.Response, error)
	ListByOrg(context.Context, string, *github.RepositoryListByOrgOptions) ([]*github.Repository, *github.Response, error)
	GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
}

func NewEnvClient(ctx context.Context) (*Client, error) {

	config := struct {
		// Method 1
		GithubToken string `env:"GITHUB_TOKEN" default:""`

		// Method 2
		GithubPrivateKey     string `env:"GH_PRIVATE_KEY" default:""`
		GithubAppID          int64  `env:"GH_APP_ID" default:""`
		GithubInstallationID int64  `env:"GH_INSTALLATION_ID" default:""`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return nil, err
	}

	var err error
	var client *Client

	if config.GithubPrivateKey != "" {
		if config.GithubAppID == 0 || config.GithubInstallationID == 0 {
			return nil, fmt.Errorf("no github app id or installation id")
		}

		tr := http.DefaultTransport
		privateKey, err := base64.StdEncoding.DecodeString(config.GithubPrivateKey)
		if err != nil {
			return nil, err
		}

		itr, err := ghinstallation.New(tr, config.GithubAppID, int64(config.GithubInstallationID), privateKey)
		if err != nil {
			return nil, err
		}

		client, err = NewClient(&http.Client{Transport: itr})
		if err != nil {
			return nil, err
		}

	} else if config.GithubToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: config.GithubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client, err = NewClient(tc)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no valid github config in environment")
	}

	return client, nil
}

func NewClient(tc *http.Client) (*Client, error) {
	ghcl := github.NewClient(tc)
	cl := &Client{
		repositories: ghcl.Repositories,
	}

	return cl, nil
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
		return nil, err
	}

	if len(dirContent) == 0 {
		return nil, nil
	}

	for _, content := range dirContent {
		file, _, err := cl.repositories.DownloadContents(ctx, org, repo, *content.Path, opts)
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("reading bytes: %s", err)
		}

		app := &application_pb.Application{}
		if err := protoread.Parse(path.Base(*content.Path), data, app); err != nil {
			return nil, err
		}

		apps = append(apps, app)
	}

	return apps, nil
}
