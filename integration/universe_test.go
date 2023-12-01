package integration

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/outbox.pg.go/outboxtest"
	"github.com/pentops/pgtest.go/pgtest"
)

type Universe struct {
	DeployerTopic      deployer_tpb.DeployerTopicClient
	GithubWebhookTopic github_pb.WebhookTopicClient
	DeployerQuery      deployer_spb.DeploymentQueryServiceClient

	Github *GithubMock
	S3     *S3Mock

	Outbox *outboxtest.OutboxAsserter
	*flowtest.Stepper
}

type S3Mock struct {
	awsinfra.S3API
	files map[string][]byte
}

func (s3m *S3Mock) MockGet(bucket string, key string) ([]byte, bool) {
	fullPath := fmt.Sprintf("s3://%s/%s", bucket, key)
	val, ok := s3m.files[fullPath]
	return val, ok
}

func (s3m *S3Mock) MockGetHTTP(uri string) ([]byte, bool) {
	fullPath := "s3://" + strings.TrimPrefix(uri, "https://s3.us-east-1.amazonaws.com/")
	val, ok := s3m.files[fullPath]
	return val, ok
}

func (s3m *S3Mock) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	fullPath := fmt.Sprintf("s3://%s/%s", *input.Bucket, *input.Key)
	bodyBytes, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}

	s3m.files[fullPath] = bodyBytes
	return &s3.PutObjectOutput{}, nil
}

type GithubMock struct {
	Configs map[string][]*application_pb.Application
}

func (gm *GithubMock) PushTargets(msg *github_pb.PushMessage) []string {
	return []string{"env"}
}

func (gm *GithubMock) PullO5Configs(ctx context.Context, org string, repo string, ref string) ([]*application_pb.Application, error) {
	key := fmt.Sprintf("%s/%s/%s", org, repo, ref)
	if configs, ok := gm.Configs[key]; ok {
		return configs, nil
	}
	return []*application_pb.Application{}, nil
}

func NewUniverse(ctx context.Context, t testing.TB) *Universe {
	name := t.Name()
	stepper := flowtest.NewStepper(name)
	uu := &Universe{
		Stepper: stepper,
	}
	return uu
}

func (uu *Universe) RunSteps(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../ext/db"))

	environments := deployer.EnvList([]*environment_pb.Environment{{
		FullName: "env",
		Provider: &environment_pb.Environment_Aws{
			Aws: &environment_pb.AWS{
				ListenerArn: "arn:listener",
			},
		},
	}})

	uu.Github = &GithubMock{
		Configs: map[string][]*application_pb.Application{},
	}

	log.DefaultLogger = log.NewCallbackLogger(uu.Stepper.Log)

	uu.S3 = &S3Mock{
		files: map[string][]byte{},
	}

	uu.Outbox = outboxtest.NewOutboxAsserter(t, conn)

	trigger, err := deployer.NewTrigger(environments, "bucket", uu.S3)
	if err != nil {
		t.Fatal(err)
	}

	topicPair := flowtest.NewGRPCPair(t)
	servicePair := flowtest.NewGRPCPair(t) // TODO: Middleware

	deploymentWorker, err := deployer.NewDeployerWorker(conn)
	if err != nil {
		t.Fatal(err)
	}
	deployer_tpb.RegisterDeployerTopicServer(topicPair.Server, deploymentWorker)
	uu.DeployerTopic = deployer_tpb.NewDeployerTopicClient(topicPair.Client)

	githubWorker, err := github.NewWebhookWorker(conn, uu.Github, trigger, uu.Github)
	if err != nil {
		t.Fatal(err)
	}

	github_pb.RegisterWebhookTopicServer(topicPair.Server, githubWorker)
	uu.GithubWebhookTopic = github_pb.NewWebhookTopicClient(topicPair.Client)

	deployerQuery, err := service.NewDeployerService(conn)
	if err != nil {
		t.Fatal(err)
	}
	deployer_spb.RegisterDeploymentQueryServiceServer(servicePair.Server, deployerQuery)
	uu.DeployerQuery = deployer_spb.NewDeploymentQueryServiceClient(servicePair.Client)

	topicPair.ServeUntilDone(t, ctx)
	servicePair.ServeUntilDone(t, ctx)

	uu.Stepper.RunSteps(t)
}

func (uu *Universe) AssertDeploymentStatus(t flowtest.Asserter, deploymentID string, status deployer_pb.DeploymentStatus) {
	t.Helper()
	ctx := context.Background()

	deployment, err := uu.DeployerQuery.GetDeployment(ctx, &deployer_spb.GetDeploymentRequest{
		DeploymentId: deploymentID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("deployment: %v", deployment)
	if deployment.State.Status != status {
		t.Fatalf("unexpected status: %v, want %s", deployment.State.Status.ShortString(), status.ShortString())
	}
}

func (uu *Universe) AssertStackStatus(t flowtest.Asserter, stackID string, status deployer_pb.StackStatus, pendingDeployments []string) {
	t.Helper()
	ctx := context.Background()

	stack, err := uu.DeployerQuery.GetStack(ctx, &deployer_spb.GetStackRequest{
		StackId: stackID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	t.Logf("stack: %v", stack)
	if stack.State.Status != status {
		t.Fatalf("unexpected status: %v, want %s", stack.State.Status.ShortString(), status.ShortString())
	}
	if len(stack.State.QueuedDeployments) != len(pendingDeployments) {
		t.Fatalf("unexpected pending deployments: %v, want %v", stack.State.QueuedDeployments, pendingDeployments)
	}
	for i, deploymentID := range pendingDeployments {
		if stack.State.QueuedDeployments[i].DeploymentId != deploymentID {
			t.Fatalf("unexpected pending deployments: %v, want %v", stack.State.QueuedDeployments, pendingDeployments)
		}
	}
}
