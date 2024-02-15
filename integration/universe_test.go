package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/github"
	"github.com/pentops/o5-deploy-aws/integration/mocks"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
	"github.com/pentops/outbox.pg.go/outboxtest"
	"github.com/pentops/pgtest.go/pgtest"
)

type UniverseAsserter struct {
	*Universe
	flowtest.Asserter
}

/*
func pascalKey(key string) string {
	return strcase.ToCamel(key)
}
*/

func (ua *UniverseAsserter) afterEach(ctx context.Context) {
	ua.Helper()
	hadMessages := false
	suggestions := []string{}
	ua.Outbox.ForEachMessage(ua, func(topic, service string, data []byte) {

		switch service {
		/*
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
		*/
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
	DeployerTopic deployer_tpb.DeployerTopicClient
	CFReplyTopic  deployer_tpb.CloudFormationReplyTopicClient

	GithubWebhookTopic github_pb.WebhookTopicClient
	DeployerQuery      deployer_spb.DeploymentQueryServiceClient
	DeployerCommand    deployer_spb.DeploymentCommandServiceClient

	SpecBuilder *deployer.SpecBuilder

	Github *mocks.Github
	S3     *mocks.S3

	Outbox *outboxtest.OutboxAsserter

	AWSStack cfMock
}

type cfMock struct {
	lastRequest *messaging_pb.RequestMetadata
	lastStack   string
	uu          *Universe
}

func (cf *cfMock) ExpectStabalizeStack(t flowtest.TB) {
	t.Helper()
	stabalizeRequest := &deployer_tpb.StabalizeStackMessage{}
	cf.uu.Outbox.PopMessage(t, stabalizeRequest)
	cf.lastRequest = stabalizeRequest.Request
	cf.lastStack = stabalizeRequest.StackName
}

func (cf *cfMock) ExpectCreateStack(t flowtest.TB) *deployer_tpb.CreateNewStackMessage {
	t.Helper()
	createRequest := &deployer_tpb.CreateNewStackMessage{}
	cf.uu.Outbox.PopMessage(t, createRequest)
	cf.lastRequest = createRequest.Request
	cf.lastStack = createRequest.StackName
	return createRequest
}

func (cf *cfMock) StackStatusMissing(t flowtest.TB) {
	t.Helper()
	_, err := cf.uu.CFReplyTopic.StackStatusChanged(context.Background(), &deployer_tpb.StackStatusChangedMessage{
		Request:   cf.lastRequest,
		StackName: cf.lastStack,
		Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_MISSING,
		Status:    "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func (cf *cfMock) StackCreateComplete(t flowtest.TB) {
	_, err := cf.uu.CFReplyTopic.StackStatusChanged(context.Background(), &deployer_tpb.StackStatusChangedMessage{
		Request:   cf.lastRequest,
		StackName: cf.lastStack,
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

/*
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
*/

func (ss *Stepper) RunSteps(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn := pgtest.GetTestDB(t, pgtest.WithDir("../ext/db"))

	uu := &Universe{}
	ss.currentUniverse = uu

	uu.AWSStack = cfMock{
		uu: uu,
	}

	uu.Github = mocks.NewGithub()

	log.DefaultLogger = log.NewCallbackLogger(ss.stepper.Log)

	uu.S3 = mocks.NewS3()

	uu.Outbox = outboxtest.NewOutboxAsserter(t, conn)

	refStore, err := github.NewRefStore(conn)
	if err != nil {
		t.Fatal(err)
	}

	tplStore := deployer.NewS3TemplateStore(uu.S3, "bucket")

	trigger, err := deployer.NewSpecBuilder(tplStore)
	if err != nil {
		t.Fatal(err)
	}
	uu.SpecBuilder = trigger

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		t.Fatal(err)
	}

	topicPair := flowtest.NewGRPCPair(t)
	servicePair := flowtest.NewGRPCPair(t) // TODO: Middleware

	deploymentWorker, err := deployer.NewDeployerWorker(conn, trigger, stateMachines)
	if err != nil {
		t.Fatal(err)
	}
	deployer_tpb.RegisterDeployerTopicServer(topicPair.Server, deploymentWorker)
	uu.DeployerTopic = deployer_tpb.NewDeployerTopicClient(topicPair.Client)

	deployer_tpb.RegisterCloudFormationReplyTopicServer(topicPair.Server, deploymentWorker)
	uu.CFReplyTopic = deployer_tpb.NewCloudFormationReplyTopicClient(topicPair.Client)

	githubWorker, err := github.NewWebhookWorker(conn, uu.Github, refStore)
	if err != nil {
		t.Fatal(err)
	}

	github_pb.RegisterWebhookTopicServer(topicPair.Server, githubWorker)
	uu.GithubWebhookTopic = github_pb.NewWebhookTopicClient(topicPair.Client)

	deployerService, err := service.NewDeployerService(conn, uu.Github, stateMachines)
	if err != nil {
		t.Fatal(err)
	}
	deployer_spb.RegisterDeploymentCommandServiceServer(servicePair.Server, deployerService)
	uu.DeployerCommand = deployer_spb.NewDeploymentCommandServiceClient(servicePair.Client)

	queryService, err := service.NewQueryService(conn, stateMachines)
	if err != nil {
		t.Fatal(err)
	}
	deployer_spb.RegisterDeploymentQueryServiceServer(servicePair.Server, queryService)
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
