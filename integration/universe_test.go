package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/pentops/flowtest"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_epb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/integration/mocks"
	"github.com/pentops/o5-deploy-aws/service"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"
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
		case "/o5.awsdeployer.v1.events.DeployerEvents/Stack":
			event := &awsdeployer_epb.StackEvent{}
			if err := proto.Unmarshal(data, event); err != nil {
				ua.Fatalf("unmarshal error: %v", err)
			}
			ua.Logf("Unexpected Stack Event: %s -> %s", event.Event.PSMEventKey(), event.Status.ShortString())
			suggestions = append(suggestions, fmt.Sprintf("t.PopStackEvent(t, awsdeployer_pb.StackPSMEvent%s, awsdeployer_pb.StackStatus_%s)", pascalKey(string(event.Event.PSMEventKey())), event.Status.ShortString()))

		case "/o5.awsdeployer.v1.events.DeployerEvents/Deployment":
			event := &awsdeployer_epb.DeploymentEvent{}
			if err := proto.Unmarshal(data, event); err != nil {
				ua.Fatalf("unmarshal error: %v", err)
			}
			ua.Logf("Unexpected Deployment Event: %s -> %s", event.Event.PSMEventKey(), event.Status.ShortString())
			suggestions = append(suggestions, fmt.Sprintf("t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEvent%s, awsdeployer_pb.DeploymentStatus_%s)", pascalKey(string(event.Event.PSMEventKey())), event.Status.ShortString()))
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

func (uu *Stepper) Step(name string, step func(ctx context.Context, t UniverseAsserter)) {
	uu.stepper.Step(name, func(ctx context.Context, t flowtest.Asserter) {
		ua := UniverseAsserter{
			Universe: uu.currentUniverse,
			Asserter: t,
		}
		step(ctx, ua)
		ua.afterEach(ctx)
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
	DeployerTopic deployer_tpb.DeploymentRequestTopicClient
	CFReplyTopic  awsinfra_tpb.CloudFormationReplyTopicClient

	DeployerQuery   awsdeployer_spb.DeploymentQueryServiceClient
	DeployerCommand awsdeployer_spb.DeploymentCommandServiceClient

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
	stabalizeRequest := &awsinfra_tpb.StabalizeStackMessage{
		Request: &messaging_pb.RequestMetadata{},
	}
	cf.uu.Outbox.PopMessage(t, stabalizeRequest)
	cf.lastRequest = stabalizeRequest.Request
	cf.lastStack = stabalizeRequest.StackName
}

func (cf *cfMock) ExpectCreateStack(t flowtest.TB) *awsinfra_tpb.CreateNewStackMessage {
	t.Helper()
	createRequest := &awsinfra_tpb.CreateNewStackMessage{
		Request: &messaging_pb.RequestMetadata{},
	}
	cf.uu.Outbox.PopMessage(t, createRequest)
	cf.lastRequest = createRequest.Request
	cf.lastStack = createRequest.Spec.StackName
	return createRequest
}

func (cf *cfMock) StackStatusMissing(t flowtest.TB) {
	t.Helper()
	_, err := cf.uu.CFReplyTopic.StackStatusChanged(context.Background(), &awsinfra_tpb.StackStatusChangedMessage{
		EventId:   fmt.Sprintf("event-%d", time.Now().UnixNano()),
		Request:   cf.lastRequest,
		StackName: cf.lastStack,
		Lifecycle: awsdeployer_pb.CFLifecycle_MISSING,
		Status:    "MISSING",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func (cf *cfMock) StackCreateCompleteMessage() *awsinfra_tpb.StackStatusChangedMessage {
	return &awsinfra_tpb.StackStatusChangedMessage{
		EventId:   fmt.Sprintf("event-%d", time.Now().UnixNano()),
		Request:   cf.lastRequest,
		StackName: cf.lastStack,
		Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
		Status:    "CREATE_COMPLETE",
		Outputs: []*awsdeployer_pb.KeyValue{{
			Name:  "foo",
			Value: "bar",
		}},
	}
}

func (cf *cfMock) StackCreateComplete(t flowtest.TB) {
	_, err := cf.uu.CFReplyTopic.StackStatusChanged(context.Background(), cf.StackCreateCompleteMessage())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func (uu *Universe) PopStackEvent(t flowtest.TB, eventKey awsdeployer_pb.StackPSMEventKey, status awsdeployer_pb.StackStatus) *awsdeployer_epb.StackEvent {
	t.Helper()
	event := &awsdeployer_epb.StackEvent{}
	uu.Outbox.PopMatching(t, outboxtest.NewMatcher(event, func(evt *awsdeployer_epb.StackEvent) bool {
		return evt.Event.PSMEventKey() == eventKey
	}))
	if status != event.Status {
		t.Fatalf("unexpected Stack status: %v, want %v", event.Status, status)
	}
	return event
}

func (uu *Universe) PopDeploymentEvent(t flowtest.TB, eventKey awsdeployer_pb.DeploymentPSMEventKey, status awsdeployer_pb.DeploymentStatus) *awsdeployer_epb.DeploymentEvent {
	t.Helper()
	event := &awsdeployer_epb.DeploymentEvent{}
	uu.Outbox.PopMatching(t, outboxtest.NewMatcher(event, func(evt *awsdeployer_epb.DeploymentEvent) bool {
		return evt.Event.PSMEventKey() == eventKey
	}))
	if status != event.Status {
		t.Fatalf("unexpected Deployment status: %v, want %v", event.Status, status)
	}
	return event
}

func (uu *Universe) PopEnvironmentEvent(t flowtest.TB, eventKey awsdeployer_pb.EnvironmentPSMEventKey, status awsdeployer_pb.EnvironmentStatus) *awsdeployer_epb.EnvironmentEvent {
	t.Helper()
	event := &awsdeployer_epb.EnvironmentEvent{}
	uu.Outbox.PopMatching(t, outboxtest.NewMatcher(event))
	gotKey := event.Event.PSMEventKey()
	if gotKey != eventKey {
		t.Fatalf("unexpected Environment event: %v, want %v", gotKey, eventKey)
	}
	if status != event.Status {
		t.Fatalf("unexpected Environment status: %v, want %v", event.Status, status)
	}
	return event
}

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

	log.DefaultLogger = log.NewCallbackLogger(ss.stepper.LevelLog)

	uu.S3 = mocks.NewS3()

	uu.Outbox = outboxtest.NewOutboxAsserter(t, conn)

	tplStore, err := deployer.NewS3TemplateStore(ctx, uu.S3, "bucket")
	if err != nil {
		t.Fatal(err)
	}

	trigger, err := deployer.NewSpecBuilder(tplStore)
	if err != nil {
		t.Fatal(err)
	}
	uu.SpecBuilder = trigger

	stateMachines, err := states.NewStateMachines()
	if err != nil {
		t.Fatal(err)
	}

	topicPair := flowtest.NewGRPCPair(t, service.GRPCMiddleware()...)
	servicePair := flowtest.NewGRPCPair(t, service.GRPCMiddleware()...)

	deploymentWorker, err := service.NewDeployerWorker(conn, trigger, stateMachines)
	if err != nil {
		t.Fatal(err)
	}
	deployer_tpb.RegisterDeploymentRequestTopicServer(topicPair.Server, deploymentWorker)
	uu.DeployerTopic = deployer_tpb.NewDeploymentRequestTopicClient(topicPair.Client)

	awsinfra_tpb.RegisterCloudFormationReplyTopicServer(topicPair.Server, deploymentWorker)
	uu.CFReplyTopic = awsinfra_tpb.NewCloudFormationReplyTopicClient(topicPair.Client)

	deployerService, err := service.NewCommandService(conn, uu.Github, stateMachines)
	if err != nil {
		t.Fatal(err)
	}
	awsdeployer_spb.RegisterDeploymentCommandServiceServer(servicePair.Server, deployerService)
	uu.DeployerCommand = awsdeployer_spb.NewDeploymentCommandServiceClient(servicePair.Client)

	queryService, err := service.NewQueryService(conn, stateMachines)
	if err != nil {
		t.Fatal(err)
	}
	awsdeployer_spb.RegisterDeploymentQueryServiceServer(servicePair.Server, queryService)
	uu.DeployerQuery = awsdeployer_spb.NewDeploymentQueryServiceClient(servicePair.Client)

	topicPair.ServeUntilDone(t, ctx)
	servicePair.ServeUntilDone(t, ctx)

	ss.stepper.RunSteps(t)
}

func (uu *Universe) AssertDeploymentStatus(t flowtest.Asserter, deploymentID string, status awsdeployer_pb.DeploymentStatus) *awsdeployer_pb.DeploymentState {
	t.Helper()
	ctx := context.Background()

	deployment, err := uu.DeployerQuery.GetDeployment(ctx, &awsdeployer_spb.GetDeploymentRequest{
		DeploymentId: deploymentID,
	})
	if err != nil {
		t.Fatalf("GetDeployment: %v", err)
	}
	if deployment.State.Status != status {

		for _, step := range deployment.State.Data.Steps {
			t.Logf("step %s is %s", step.Name, step.Status.ShortString())
		}
		t.Fatalf("unexpected status: %v, want %s", deployment.State.Status.ShortString(), status.ShortString())
	}
	return deployment.State
}

func (uu *Universe) AssertStackStatus(t flowtest.Asserter, stackID string, status awsdeployer_pb.StackStatus, pendingDeployments []string) {
	t.Helper()
	ctx := context.Background()

	stack, err := uu.DeployerQuery.GetStack(ctx, &awsdeployer_spb.GetStackRequest{
		StackId: stackID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stack.State.Status != status {
		t.Fatalf("unexpected status: %v, want %s", stack.State.Status.ShortString(), status.ShortString())
	}
	if len(stack.State.Data.QueuedDeployments) != len(pendingDeployments) {
		t.Fatalf("unexpected pending deployments: %v, want %v", stack.State.Data.QueuedDeployments, pendingDeployments)
	}
	for i, deploymentID := range pendingDeployments {
		if stack.State.Data.QueuedDeployments[i].DeploymentId != deploymentID {
			t.Fatalf("unexpected pending deployments: %v, want %v", stack.State.Data.QueuedDeployments, pendingDeployments)
		}
	}
}
