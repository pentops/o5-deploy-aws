package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
)

func TestCreateHappy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, t)
	defer uu.RunSteps(t)

	initialTrigger := &deployer_tpb.RequestDeploymentMessage{}
	uu.Step("Github Trigger", func(t flowtest.Asserter) {
		uu.Github.Configs["owner/repo/after"] = []*application_pb.Application{{
			Name: "app",
			DeploymentConfig: &application_pb.DeploymentConfig{
				QuickMode: false,
			},
		}}
		_, err := uu.GithubWebhookTopic.Push(ctx, &github_pb.PushMessage{
			Owner: "owner",
			Repo:  "repo",
			Ref:   "ref",
			After: "after",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.Outbox.PopMessage(t, initialTrigger)
		t.Log(protojson.Format(initialTrigger))

		templateContent, ok := uu.S3.MockGetHTTP(initialTrigger.Spec.TemplateUrl)
		if !ok {
			t.Fatalf("template not found: %s", initialTrigger.Spec.TemplateUrl)
		}
		t.Log(string(templateContent))
	})

	triggerMessage := &deployer_tpb.TriggerDeploymentMessage{}
	uu.Step("[*] --> QUEUED", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.RequestDeployment(ctx, initialTrigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.Outbox.PopMessage(t, triggerMessage)
		uu.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)
		uu.Outbox.AssertNoMessages(t)
	})

	var stackID *deployer_tpb.StackID

	uu.Step("[*] --> QUEUED --> LOCKED --> WAITING", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.TriggerDeployment(ctx, triggerMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stabalizeRequest := &deployer_tpb.StabalizeStackMessage{}
		uu.Outbox.PopMessage(t, stabalizeRequest)
		stackID = stabalizeRequest.StackId

		uu.AssertDeploymentStatus(t, stackID.DeploymentId, deployer_pb.DeploymentStatus_WAITING)
	})

	uu.Step("WAITING --> AVAILABLE --> CREATING : StackStatus.Missing", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   stackID,
			Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_MISSING,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		createRequest := &deployer_tpb.CreateNewStackMessage{}
		uu.Outbox.PopMessage(t, createRequest)
		stackID = createRequest.StackId
		uu.Outbox.AssertNoMessages(t)

		uu.AssertDeploymentStatus(t, stackID.DeploymentId, deployer_pb.DeploymentStatus_CREATING)
	})

	uu.Step("CREATING --> CREATING : StackStatus.Progress", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   stackID,
			Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_PROGRESS,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.Outbox.AssertNoMessages(t)
		uu.AssertDeploymentStatus(t, stackID.DeploymentId, deployer_pb.DeploymentStatus_CREATING)
	})

	uu.Step("CREATING --> INFRA_MIGRATED --> SCALING_UP : StackStatus.Stable", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   stackID,
			Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_COMPLETE,
			Status:    "FOOBAR",
			Outputs: []*deployer_pb.KeyValue{{
				Name:  "foo",
				Value: "bar",
			}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// No DB to migrate

		scaleUpRequest := &deployer_tpb.ScaleStackMessage{}
		uu.Outbox.PopMessage(t, scaleUpRequest)
		stackID = scaleUpRequest.StackId

		t.Equal(int(1), int(scaleUpRequest.DesiredCount))

		uu.Outbox.AssertNoMessages(t)
		uu.AssertDeploymentStatus(t, stackID.DeploymentId, deployer_pb.DeploymentStatus_SCALING_UP)
	})

	uu.Step("SCALING_UP --> SCALED_UP --> DONE", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   stackID,
			Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_COMPLETE,
			Status:    "FOOBAR",
			Outputs: []*deployer_pb.KeyValue{{
				Name:  "foo",
				Value: "bar",
			}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		deploymentCompleteMessage := &deployer_tpb.DeploymentCompleteMessage{}
		uu.Outbox.PopMessage(t, deploymentCompleteMessage)

		uu.Outbox.AssertNoMessages(t)
		uu.AssertDeploymentStatus(t, stackID.DeploymentId, deployer_pb.DeploymentStatus_DONE)

	})
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

func TestStackLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, t)
	defer uu.RunSteps(t)

	firstDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId: uuid.NewString(),
		Spec: &deployer_pb.DeploymentSpec{
			AppName:         "app",
			Version:         "1",
			EnvironmentName: "env",
			QuickMode:       true,
		},
	}

	awsStack := &cfMock{uu: uu}

	firstTriggerMessage := &deployer_tpb.TriggerDeploymentMessage{}
	uu.Step("Request First", func(t flowtest.Asserter) {
		// Stack:  [*] --> CREATING : Trigger
		// Deployment: [*] --> QUEUED : Created
		_, err := uu.DeployerTopic.RequestDeployment(ctx, firstDeploymentRequest)
		if err != nil {
			t.Fatalf("TriggerDeployment error: %v", err)
		}

		uu.Outbox.PopMessage(t, firstTriggerMessage)
		uu.AssertDeploymentStatus(t, firstTriggerMessage.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)
		uu.Outbox.AssertNoMessages(t)

	})

	uu.Step("Trigger First", func(t flowtest.Asserter) {
		// Deployment: QUEUED --> TRIGGERED --> WAITING
		// Stack: Stays CREATING
		_, err := uu.DeployerTopic.TriggerDeployment(ctx, firstTriggerMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		awsStack.ExpectStabalizeStack(t)

		uu.AssertDeploymentStatus(t, firstDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_WAITING)
		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{})

		uu.Outbox.AssertNoMessages(t)
	})

	uu.Step("First -> Upserting", func(t flowtest.Asserter) {
		// Deployment WAITING --> AVAILABLE --> UPSERTING
		// Stack: Stays CREATING
		awsStack.StackStatusMissing(t)
		awsStack.ExpectCreateStack(t)
		uu.Outbox.AssertNoMessages(t)

		uu.AssertDeploymentStatus(t, firstDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_UPSERTING)
		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{})

	})

	secondDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId: uuid.NewString(),
		Spec: &deployer_pb.DeploymentSpec{
			AppName:         "app",
			Version:         "2",
			EnvironmentName: "env",
			QuickMode:       true,
		},
	}

	uu.Step("Request a second deployment", func(t flowtest.Asserter) {
		// Stack:  CREATING --> CREATING : Trigger
		// Deployment: [*] --> QUEUED : Created
		_, err := uu.DeployerTopic.RequestDeployment(ctx, secondDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.Outbox.AssertNoMessages(t)

		uu.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)
		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{secondDeploymentRequest.DeploymentId})

	})

	deployment1CompleteMessage := &deployer_tpb.DeploymentCompleteMessage{}
	uu.Step("Complete the first deployment", func(t flowtest.Asserter) {
		// Deployment: UPSERTING --> UPSERTED --> DONE
		// Stack: CREATING --> STABLE --> MIGRATING
		awsStack.StackCreateComplete(t)

		uu.AssertDeploymentStatus(t, firstDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_DONE)
		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{secondDeploymentRequest.DeploymentId})

		uu.Outbox.PopMessage(t, deployment1CompleteMessage)
		uu.Outbox.AssertNoMessages(t)

	})

	deployment2TriggerMessage := &deployer_tpb.TriggerDeploymentMessage{}
	uu.Step("Write back to the Stack state machine", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.DeploymentComplete(ctx, deployment1CompleteMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		uu.Outbox.PopMessage(t, deployment2TriggerMessage)
		uu.Outbox.AssertNoMessages(t)
	})

	uu.Step("Second -> Waiting", func(t flowtest.Asserter) {
		// Deployment: QUEUED --> TRIGGERED --> WAITING
		// Stack: CREATING --> STABLE --> MIGRATING
		_, err := uu.DeployerTopic.TriggerDeployment(ctx, deployment2TriggerMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		awsStack.ExpectStabalizeStack(t)

		uu.Outbox.AssertNoMessages(t)
	})

	uu.Step("Second -> Upserting", func(t flowtest.Asserter) {
		// Deployment WAITING --> AVAILABLE --> UPSERTING
		awsStack.StackStatusMissing(t)

		awsStack.ExpectCreateStack(t)
		uu.Outbox.AssertNoMessages(t)

		uu.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_UPSERTING)
		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

	})

	deployment2CompleteMessage := &deployer_tpb.DeploymentCompleteMessage{}
	uu.Step("Complete the second deployment", func(t flowtest.Asserter) {
		// Deployment: UPSERTING --> UPSERTED --> DONE
		// Stack: CREATING --> STABLE --> MIGRATING
		awsStack.StackCreateComplete(t)

		uu.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_DONE)
		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		uu.Outbox.PopMessage(t, deployment2CompleteMessage)
		uu.Outbox.AssertNoMessages(t)

	})

	uu.Step("Complete the second deployment", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.DeploymentComplete(ctx, deployment2CompleteMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_AVAILABLE,
			[]string{})

	})

	uu.Step("A third deployment should begin immediately", func(t flowtest.Asserter) {

		thirdDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
			DeploymentId: uuid.NewString(),
			Spec: &deployer_pb.DeploymentSpec{
				AppName:         "app",
				Version:         "3",
				EnvironmentName: "env",
				QuickMode:       true,
			},
		}

		// Stack: AVAILABLE --> MIGRATING : Trigger
		_, err := uu.DeployerTopic.RequestDeployment(ctx, thirdDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		deployment3TriggerMessage := &deployer_tpb.TriggerDeploymentMessage{}

		uu.Outbox.PopMessage(t, deployment3TriggerMessage)
		uu.Outbox.AssertNoMessages(t)

		uu.AssertStackStatus(t, deployer.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

	})

}
