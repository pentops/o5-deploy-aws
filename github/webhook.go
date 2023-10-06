package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v47/github"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/application/v1/application_pb"
)

type IClient interface {
	PullO5Configs(ctx context.Context, org string, repo string, ref string) ([]*application_pb.Application, error)
}

type IDeployer interface {
	Deploy(context.Context, *app.Application, bool) error
}

type WebhookWorker struct {
	github      IClient
	deployer    IDeployer
	secretToken string
}

func NewWebhookWorker(githubClient IClient, deployer IDeployer, secretToken string) (*WebhookWorker, error) {
	return &WebhookWorker{
		github:      githubClient,
		deployer:    deployer,
		secretToken: secretToken,
	}, nil
}

func (w *WebhookWorker) RunServer(ctx context.Context, bind string) error {

	handler := http.HandlerFunc(func(wr http.ResponseWriter, req *http.Request) {

		if req.Method != http.MethodPost {
			http.Error(wr, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := w.handleWebRequest(req); err != nil {
			http.Error(wr, err.Error(), http.StatusInternalServerError)
			return
		}

	})

	ss := &http.Server{
		Addr:    bind,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		ss.Shutdown(context.Background()) //nolint:errcheck
	}()

	return ss.ListenAndServe()
}

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func newHTTPError(statusCode int, message string, params ...interface{}) error {
	// TODO: Wrap and check
	msgString := fmt.Sprintf(message, params...)
	return &HTTPError{
		StatusCode: statusCode,
		Message:    msgString,
	}
}

func (ww *WebhookWorker) handleWebRequest(req *http.Request) error {

	verifiedPayload, err := github.ValidatePayload(req, []byte(ww.secretToken))
	if err != nil {
		return newHTTPError(http.StatusBadRequest, "failed to validate payload")
	}

	anyEvent, err := github.ParseWebHook(req.Header.Get("X-GitHub-Event"), verifiedPayload)
	if err != nil {
		return err
	}

	event, ok := anyEvent.(*github.PushEvent)
	if !ok {
		return newHTTPError(http.StatusBadRequest, "webhooks should only be configured for push events, got %T", anyEvent)
	}

	return ww.HandlePushEvent(req.Context(), event)

}

func (ww *WebhookWorker) HandlePushEvent(ctx context.Context, event *github.PushEvent) error {

	if err := validatePushEvent(event); err != nil {
		return err
	}

	if *event.After == emptyCommit {
		return newHTTPError(http.StatusBadRequest, "push event has empty after commit")
	}

	apps, err := ww.github.PullO5Configs(ctx, *event.Repo.Owner.Name, *event.Repo.Name, *event.After)

	if err != nil {
		return err
	}

	if len(apps) == 0 {
		return newHTTPError(http.StatusFailedDependency, "no applications found in push event")
	}

	if len(apps) > 1 {
		return newHTTPError(http.StatusFailedDependency, "multiple applications found in push event, not yet supported")
	}

	appStack, err := app.BuildApplication(apps[0], *event.After)
	if err != nil {
		return err
	}

	return ww.deployer.Deploy(ctx, appStack, true)
}

const emptyCommit = "0000000000000000000000000000000000000000"

func validatePushEvent(event *github.PushEvent) error {

	if event.Ref == nil {
		return fmt.Errorf("nil 'ref' on push event")
	}

	if event.Repo == nil {
		return fmt.Errorf("nil 'repo' on push event")
	}

	if event.Repo.Owner == nil {
		return fmt.Errorf("nil 'repo.owner' on push event")
	}

	if event.Repo.Owner.Name == nil {
		return fmt.Errorf("nil 'repo.owner.name' on push event")
	}

	if event.Repo.Name == nil {
		return fmt.Errorf("nil 'repo.name' on push event")
	}

	if event.After == nil {
		return fmt.Errorf("nil 'after' on push event")
	}

	if event.Before == nil {
		return fmt.Errorf("nil 'before' on push event")
	}
	return nil

}
