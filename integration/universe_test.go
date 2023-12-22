package integration

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/iancoleman/strcase"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_epb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/outbox.pg.go/outboxtest"
	"github.com/pentops/pgtest.go/pgtest"
	"google.golang.org/protobuf/proto"
)

type UniverseAsserter struct {
	*Universe
	flowtest.Asserter
}

func pascalKey(key string) string {
	return strcase.ToCamel(key)
}

func (ua *UniverseAsserter) afterEach(ctx context.Context) {
	ua.Helper()
	hadMessages := false
	suggestions := []string{}
	ua.Outbox.ForEachMessage(ua, func(topic, service string, data []byte) {

		switch service {
		case "/o5.deployer.v1.events.DeployerEventsTopic/StackEvent":
			event := &deployer_epb.StackEventMessage{}
			if err := proto.Unmarshal(data, event); err != nil {
				ua.Fatalf("unmarshal error: %v", err)
			}
			ua.Logf("Unexpected Stack Event: %s -> %s", event.Event.PSMEventKey(), event.State.Status.ShortString())
			suggestions = append(suggestions, fmt.Sprintf("t.PopStackEvent(t, deployer_pb.StackPSMEvent%s, deployer_pb.StackStatus_%s)", pascalKey(string(event.Event.PSMEventKey())), event.State.Status.ShortString()))

		case "/o5.deployer.v1.events.DeployerEventsTopic/DeploymentEvent":
			event := &deployer_epb.DeploymentEventMessage{}
			if err := proto.Unmarshal(data, event); err != nil {
				ua.Fatalf("unmarshal error: %v", err)
			}
			ua.Logf("Unexpected Deployment Event: %s -> %s", event.Event.PSMEventKey(), event.State.Status.ShortString())
			suggestions = append(suggestions, fmt.Sprintf("t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEvent%s, deployer_pb.DeploymentStatus_%s)", pascalKey(string(event.Event.PSMEventKey())), event.State.Status.ShortString()))
		default:
			ua.Fatalf("unexpected message %s %s", topic, service)
		}

		hadMessages = true

	})
	if hadMessages {
		ua.Errorf("unexpected messages. Suggestion: \n  %s", strings.Join(suggestions, "\n  "))
	}

	ua.Outbox.PurgeAll(ua)
}

type Stepper struct {
	stepper         *flowtest.Stepper[*testing.T]
	currentUniverse *Universe
}

func (uu *Stepper) StepC(name string, step func(ctx context.Context, t UniverseAsserter)) {
	uu.stepper.StepC(name, func(ctx context.Context, t flowtest.Asserter) {
		ua := UniverseAsserter{
			Universe: uu.currentUniverse,
			Asserter: t,
		}
		step(ctx, ua)
		ua.afterEach(ctx)
	})
}

func (uu *Stepper) Step(name string, step func(t UniverseAsserter)) {
	uu.StepC(name, func(_ context.Context, t UniverseAsserter) {
		step(t)
	})
}

func NewStepper(ctx context.Context, t testing.TB) *Stepper {
	name := t.Name()
	stepper := flowtest.NewStepper[*testing.T](name)
	uu := &Stepper{
		stepper: stepper,
	}
	return uu
}

type Universe struct {
	DeployerTopic      deployer_tpb.DeployerTopicClient
	GithubWebhookTopic github_pb.WebhookTopicClient
	DeployerQuery      deployer_spb.DeploymentQueryServiceClient

	SpecBuilder *deployer.SpecBuilder

	Github *GithubMock
	S3     *S3Mock

	Outbox *outboxtest.OutboxAsserter

	AWSStack cfMock
}

type cfMock struct {
	lastRequest *deployer_tpb.StackID
	uu          *Universe
}

func (cf *cfMock) ExpectStabalizeStack(t flowtest.TB) {
	t.Helper()
	stabalizeRequest := &deployer_tpb.StabalizeStackMessage{}
	cf.uu.Outbox.PopMessage(t, stabalizeRequest)
	cf.lastRequest = stabalizeRequest.StackId
}

func (cf *cfMock) ExpectCreateStack(t flowtest.TB) *deployer_tpb.CreateNewStackMessage {
	t.Helper()
	createRequest := &deployer_tpb.CreateNewStackMessage{}
	cf.uu.Outbox.PopMessage(t, createRequest)
	cf.lastRequest = createRequest.StackId
	return createRequest
}

func (cf *cfMock) StackStatusMissing(t flowtest.TB) {
	t.Helper()
	_, err := cf.uu.DeployerTopic.StackStatusChanged(context.Background(), &deployer_tpb.StackStatusChangedMessage{
		StackId:   cf.lastRequest,
		Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_MISSING,
		Status:    "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func (cf *cfMock) StackCreateComplete(t flowtest.TB) {
	_, err := cf.uu.DeployerTopic.StackStatusChanged(context.Background(), &deployer_tpb.StackStatusChangedMessage{
		StackId:   cf.lastRequest,
		Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_COMPLETE,
		Status:    "CREATE_COMPLETE",
		Outputs: []*deployer_pb.KeyValue{{
			Name:  "foo",
			Value: "bar",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

func (uu *Universe) PopStackEvent(t flowtest.TB, eventKey deployer_pb.StackPSMEventKey, status deployer_pb.StackStatus) *deployer_epb.StackEventMessage {
	t.Helper()
	event := &deployer_epb.StackEventMessage{}
	uu.Outbox.PopMatching(t, outboxtest.NewMatcher(event))
	gotKey := event.Event.PSMEventKey()
	if gotKey != eventKey {
		t.Fatalf("unexpected event: %v, want %v", gotKey, eventKey)
	}
	if status != event.State.Status {
		t.Fatalf("unexpected status: %v, want %v", event.State.Status, status)
	}
	return event
}

func (uu *Universe) PopDeploymentEvent(t flowtest.TB, eventKey deployer_pb.DeploymentPSMEventKey, status deployer_pb.DeploymentStatus) *deployer_epb.DeploymentEventMessage {
	t.Helper()
	event := &deployer_epb.DeploymentEventMessage{}
	uu.Outbox.PopMatching(t, outboxtest.NewMatcher(event))
	gotKey := event.Event.PSMEventKey()
	if gotKey != eventKey {
		t.Fatalf("unexpected event: %v, want %v", gotKey, eventKey)
	}
	if status != event.State.Status {
		t.Fatalf("unexpected status: %v, want %v", event.State.Status, status)
	}
	return event
}

func (ss *Stepper) RunSteps(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../ext/db"))

	uu := &Universe{}
	ss.currentUniverse = uu

	environments := deployer.EnvList([]*environment_pb.Environment{{
		FullName: "env",
		Provider: &environment_pb.Environment_Aws{
			Aws: &environment_pb.AWS{
				ListenerArn: "arn:listener",
			},
		},
	}})

	uu.AWSStack = cfMock{
		uu: uu,
	}

	uu.Github = &GithubMock{
		Configs: map[string][]*application_pb.Application{},
	}

	log.DefaultLogger = log.NewCallbackLogger(ss.stepper.Log)

	uu.S3 = &S3Mock{
		files: map[string][]byte{},
	}

	uu.Outbox = outboxtest.NewOutboxAsserter(t, conn)

	trigger, err := deployer.NewSpecBuilder(environments, "bucket", uu.S3)
	if err != nil {
		t.Fatal(err)
	}
	uu.SpecBuilder = trigger

	topicPair := flowtest.NewGRPCPair(t)
	servicePair := flowtest.NewGRPCPair(t) // TODO: Middleware

	deploymentWorker, err := deployer.NewDeployerWorker(conn, trigger)
	if err != nil {
		t.Fatal(err)
	}
	deployer_tpb.RegisterDeployerTopicServer(topicPair.Server, deploymentWorker)
	uu.DeployerTopic = deployer_tpb.NewDeployerTopicClient(topicPair.Client)

	githubWorker, err := github.NewWebhookWorker(conn, uu.Github, uu.Github)
	if err != nil {
		t.Fatal(err)
	}

	github_pb.RegisterWebhookTopicServer(topicPair.Server, githubWorker)
	uu.GithubWebhookTopic = github_pb.NewWebhookTopicClient(topicPair.Client)

	deployerQuery, err := service.NewDeployerService(conn, uu.Github)
	if err != nil {
		t.Fatal(err)
	}
	deployer_spb.RegisterDeploymentQueryServiceServer(servicePair.Server, deployerQuery)
	uu.DeployerQuery = deployer_spb.NewDeploymentQueryServiceClient(servicePair.Client)

	topicPair.ServeUntilDone(t, ctx)
	servicePair.ServeUntilDone(t, ctx)

	ss.stepper.RunSteps(t)
}

func (uu *Universe) AssertDeploymentStatus(t flowtest.Asserter, deploymentID string, status deployer_pb.DeploymentStatus) {
	t.Helper()
	ctx := context.Background()

	deployment, err := uu.DeployerQuery.GetDeployment(ctx, &deployer_spb.GetDeploymentRequest{
		DeploymentId: deploymentID,
	})
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
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
