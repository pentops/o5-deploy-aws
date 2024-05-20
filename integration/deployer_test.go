package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/gen/o5/github/v1/github_pb"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"

	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_tpb"
)

func TestDeploymentFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	var environmentID string
	ss.Step("Configure Environment", func(t UniverseAsserter) {
		_, err := t.DeployerCommand.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
			EnvironmentId: "env",
			Src: &deployer_spb.UpsertEnvironmentRequest_Config{
				Config: &environment_pb.Environment{
					FullName: "env",
					Provider: &environment_pb.Environment_Aws{
						Aws: &environment_pb.AWS{
							ListenerArn: "arn:listener",
						},
					},
				},
			},
		})
		t.NoError(err)

		envConfitured := t.PopEnvironmentEvent(t, deployer_pb.EnvironmentPSMEventConfigured, deployer_pb.EnvironmentStatus_ACTIVE)
		environmentID = envConfitured.Event.Keys.EnvironmentId

		_, err = t.DeployerCommand.UpsertStack(ctx, &deployer_spb.UpsertStackRequest{
			StackId: "env-app",
			Config: &deployer_pb.StackConfig{
				CodeSource: &deployer_pb.SourceTriggerType{
					Type: &deployer_pb.SourceTriggerType_Github_{
						Github: &deployer_pb.SourceTriggerType_Github{
							Owner:  "owner",
							Repo:   "repo",
							Branch: "ref1",
						},
					},
				},
			},
		})
		t.NoError(err)

		t.PopStackEvent(t, deployer_pb.StackPSMEventConfigured, deployer_pb.StackStatus_AVAILABLE)
	})

	// Trigger the deployment by pushing to the configured branch
	request := &deployer_tpb.RequestDeploymentMessage{}
	ss.Step("Github Trigger", func(t UniverseAsserter) {
		t.Github.Configs["owner/repo/after"] = []*application_pb.Application{{
			Name: "app",
			DeploymentConfig: &application_pb.DeploymentConfig{
				QuickMode: false,
			},
		}}
		_, err := t.GithubWebhookTopic.Push(ctx, &github_pb.PushMessage{
			Owner: "owner",
			Repo:  "repo",
			Ref:   "refs/heads/ref1",
			After: "after",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Outbox.PopMessage(t, request)

		t.Equal(environmentID, request.EnvironmentId)
		t.Equal("after", request.Version)
	})

	var stackRequest *messaging_pb.RequestMetadata
	var stackName string
	ss.Step("Deployment Queued To Triggered", func(t UniverseAsserter) {
		_, err := t.DeployerTopic.RequestDeployment(ctx, request)
		t.NoError(err)

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentRequested, deployer_pb.StackStatus_AVAILABLE)
		t.PopStackEvent(t, deployer_pb.StackPSMEventRunDeployment, deployer_pb.StackStatus_MIGRATING)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)

		stabalizeRequest := &deployer_tpb.StabalizeStackMessage{
			Request: &messaging_pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, stabalizeRequest)
		stackRequest = stabalizeRequest.Request
		stackName = stabalizeRequest.StackName

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)

		t.AssertDeploymentStatus(t, request.DeploymentId, deployer_pb.DeploymentStatus_WAITING)
	})

	ss.Step("CF Stack Missing Create New Stack", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			EventId:   "evt0",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: deployer_pb.CFLifecycle_MISSING,
			Status:    "MISSING",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		createRequest := &deployer_tpb.CreateNewStackMessage{
			Request: &messaging_pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, createRequest)
		stackRequest = createRequest.Request

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackAvailable, deployer_pb.DeploymentStatus_AVAILABLE)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventRunSteps, deployer_pb.DeploymentStatus_RUNNING)

		t.AssertDeploymentStatus(t, request.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("StackStatus Progress", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			EventId:   "evt1",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: deployer_pb.CFLifecycle_PROGRESS,
			Status:    "PROGRESS",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertDeploymentStatus(t, request.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("StackStatus Stable", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			EventId:   "evt2",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: deployer_pb.CFLifecycle_COMPLETE,
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

		scaleUpRequest := &deployer_tpb.ScaleStackMessage{
			Request: &messaging_pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, scaleUpRequest)
		stackRequest = scaleUpRequest.Request

		t.Equal(int(1), int(scaleUpRequest.DesiredCount))

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStepResult, deployer_pb.DeploymentStatus_RUNNING)

		t.AssertDeploymentStatus(t, request.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("RUNNING --> DONE", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			EventId:   "evt3",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: deployer_pb.CFLifecycle_COMPLETE,
			Status:    "FOOBAR",
			Outputs: []*deployer_pb.KeyValue{{
				Name:  "foo",
				Value: "bar",
			}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStepResult, deployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventDone, deployer_pb.DeploymentStatus_DONE)

		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentCompleted, deployer_pb.StackStatus_AVAILABLE)

		t.AssertDeploymentStatus(t, request.DeploymentId, deployer_pb.DeploymentStatus_DONE)

	})
}

func TestStackLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	var environmentID string
	appDef := &application_pb.Application{
		Name: "app",
	}

	firstDeploymentID := uuid.NewString()
	secondDeploymentID := uuid.NewString()

	ss.StepC("setup", func(ctx context.Context, t UniverseAsserter) {
		res, err := t.DeployerCommand.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
			EnvironmentId: "env",
			Src: &deployer_spb.UpsertEnvironmentRequest_Config{
				Config: &environment_pb.Environment{
					FullName: "env",
					Provider: &environment_pb.Environment_Aws{
						Aws: &environment_pb.AWS{
							ListenerArn: "arn:listener",
						},
					},
				},
			},
		})
		t.NoError(err)
		environmentID = res.State.Keys.EnvironmentId
		t.PopEnvironmentEvent(t, deployer_pb.EnvironmentPSMEventConfigured, deployer_pb.EnvironmentStatus_ACTIVE)
	})

	ss.Step("Request and begin first deployment", func(t UniverseAsserter) {
		firstDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
			DeploymentId:  firstDeploymentID,
			Application:   appDef,
			Version:       "1",
			EnvironmentId: environmentID,
			Flags: &deployer_pb.DeploymentFlags{
				QuickMode: true,
			},
		}

		_, err := t.DeployerTopic.RequestDeployment(ctx, firstDeploymentRequest)
		if err != nil {
			t.Fatalf("TriggerDeployment error: %v", err)
		}

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentRequested, deployer_pb.StackStatus_AVAILABLE)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)
		t.PopStackEvent(t, deployer_pb.StackPSMEventRunDeployment, deployer_pb.StackStatus_MIGRATING)

		t.AWSStack.ExpectStabalizeStack(t)

		t.AssertDeploymentStatus(t, firstDeploymentID, deployer_pb.DeploymentStatus_WAITING)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)
	})

	ss.Step("First -> Running", func(t UniverseAsserter) {
		// Deployment WAITING --> AVAILABLE --> RUNNING
		// Stack: Stays MIGRATING
		t.AWSStack.StackStatusMissing(t)

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackAvailable, deployer_pb.DeploymentStatus_AVAILABLE)
		t.AWSStack.ExpectCreateStack(t)

		t.AssertDeploymentStatus(t, firstDeploymentID, deployer_pb.DeploymentStatus_RUNNING)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventRunSteps, deployer_pb.DeploymentStatus_RUNNING)

	})

	ss.Step("Request a second deployment", func(t UniverseAsserter) {
		secondDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
			DeploymentId:  secondDeploymentID,
			Application:   appDef,
			Version:       "2",
			EnvironmentId: environmentID,
		}
		// Stack:  MIGRATING --> MIGRATING : Trigger
		// Deployment: [*] --> QUEUED : Created
		_, err := t.DeployerTopic.RequestDeployment(ctx, secondDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{secondDeploymentRequest.DeploymentId})

		// New request, but no change to the status.
		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentRequested, deployer_pb.StackStatus_MIGRATING)

		// Deployment is blocked.
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)

	})

	ss.Step("Complete the first deployment", func(t UniverseAsserter) {
		// Deployment: UPSERTING --> UPSERTED --> DONE
		// Stack: CREATING --> STABLE --> MIGRATING
		t.AWSStack.StackCreateComplete(t)

		t.AssertDeploymentStatus(t, firstDeploymentID, deployer_pb.DeploymentStatus_DONE)

		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		// First Deployment Completes
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStepResult, deployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventDone, deployer_pb.DeploymentStatus_DONE)

		// Stack unblocks and re-triggers
		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentCompleted, deployer_pb.StackStatus_AVAILABLE)
		t.PopStackEvent(t, deployer_pb.StackPSMEventRunDeployment, deployer_pb.StackStatus_MIGRATING)

		// The second deployment begins
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)

		t.AWSStack.ExpectStabalizeStack(t)

	})

	var secondStepCompleteMessage *deployer_tpb.StackStatusChangedMessage

	ss.Step("Second -> Running", func(t UniverseAsserter) {
		// Deployment WAITING --> AVAILABLE --> UPSERTING
		t.AWSStack.StackStatusMissing(t)

		t.AWSStack.ExpectCreateStack(t)

		t.AssertDeploymentStatus(t, secondDeploymentID, deployer_pb.DeploymentStatus_RUNNING)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		// capture the StackCompleteMessage, but don't send it in yet.
		secondStepCompleteMessage = t.AWSStack.StackCreateCompleteMessage()

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackAvailable, deployer_pb.DeploymentStatus_AVAILABLE)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventRunSteps, deployer_pb.DeploymentStatus_RUNNING)

	})

	ss.Step("Terminate the second deployment", func(t UniverseAsserter) {
		// Deployment: UPSERTING --> TERMIATED
		_, err := t.DeployerCommand.TerminateDeployment(ctx, &deployer_spb.TerminateDeploymentRequest{
			DeploymentId: secondDeploymentID,
		})

		t.NoError(err)

		t.AssertDeploymentStatus(t, secondDeploymentID, deployer_pb.DeploymentStatus_TERMINATED)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_AVAILABLE,
			[]string{})

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTerminated, deployer_pb.DeploymentStatus_TERMINATED)

		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentCompleted, deployer_pb.StackStatus_AVAILABLE)

	})

	ss.Step("A third deployment should begin immediately", func(t UniverseAsserter) {

		thirdDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
			DeploymentId:  uuid.NewString(),
			Application:   appDef,
			Version:       "3",
			EnvironmentId: environmentID,
		}

		// Stack: AVAILABLE --> MIGRATING : Trigger
		_, err := t.DeployerTopic.RequestDeployment(ctx, thirdDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
		t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentRequested, deployer_pb.StackStatus_AVAILABLE)
		t.PopStackEvent(t, deployer_pb.StackPSMEventRunDeployment, deployer_pb.StackStatus_MIGRATING)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)
		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)

		t.AWSStack.ExpectStabalizeStack(t)
	})

	ss.Step("A step in the second deployment completes after termination", func(t UniverseAsserter) {
		// Complete the StackComplete message from the second, now terminated,
		// deployment
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, secondStepCompleteMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStepResult, deployer_pb.DeploymentStatus_TERMINATED)
		fullState := t.AssertDeploymentStatus(t, secondDeploymentID, deployer_pb.DeploymentStatus_TERMINATED)
		upsertStep := fullState.Data.Steps[0]
		if upsertStep.Status != deployer_pb.StepStatus_DONE {
			t.Fatalf("expected step status DONE, got %s", upsertStep.Status)
		}

	})

}
