package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-go/github/v1/github_pb"
	"github.com/pentops/o5-go/messaging/v1/messaging_pb"

	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
)

func TestCreateHappy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	envID := uuid.NewString()
	stackID := uuid.NewString()

	ss.Step("Configure Environment", func(t UniverseAsserter) {
		_, err := t.DeployerCommand.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
			EnvironmentId: envID,
			Config: &environment_pb.Environment{
				FullName: "env",
				Provider: &environment_pb.Environment_Aws{
					Aws: &environment_pb.AWS{
						ListenerArn: "arn:listener",
					},
				},
			},
		})
		t.NoError(err)

		_, err = t.DeployerCommand.UpsertStack(ctx, &deployer_spb.UpsertStackRequest{
			StackId:         stackID,
			ApplicationName: "app",
			EnvironmentId:   envID,
			Config: &deployer_pb.StackConfig{
				CodeSource: &deployer_pb.CodeSourceType{
					Type: &deployer_pb.CodeSourceType_Github_{
						Github: &deployer_pb.CodeSourceType_Github{
							Owner:      "owner",
							Repo:       "repo",
							RefPattern: "ref1",
						},
					},
				},
			},
		})
		t.NoError(err)

	})

	initialTrigger := &deployer_tpb.RequestDeploymentMessage{}
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
			Ref:   "ref1",
			After: "after",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Outbox.PopMessage(t, initialTrigger)

	})

	triggerMessage := &deployer_tpb.TriggerDeploymentMessage{}
	ss.Step("[*] --> QUEUED", func(t UniverseAsserter) {
		_, err := t.DeployerTopic.RequestDeployment(ctx, initialTrigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Outbox.PopMessage(t, triggerMessage)
		t.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
			t.PopStackEvent(t, deployer_pb.StackPSMEventTriggered, deployer_pb.StackStatus_CREATING)
		*/

	})

	var stackRequest *messaging_pb.RequestMetadata
	var stackName string

	ss.Step("QUEUED --> TRIGGERED --> WAITING", func(t UniverseAsserter) {
		_, err := t.DeployerTopic.TriggerDeployment(ctx, triggerMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stabalizeRequest := &deployer_tpb.StabalizeStackMessage{
			Request: &messaging_pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, stabalizeRequest)
		stackRequest = stabalizeRequest.Request
		stackName = stabalizeRequest.StackName

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)
		*/

		t.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_WAITING)
	})

	ss.Step("WAITING --> AVAILABLE --> RUNNING : StackStatus.Missing", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: deployer_pb.CFLifecycle_MISSING,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		createRequest := &deployer_tpb.CreateNewStackMessage{
			Request: &messaging_pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, createRequest)
		stackRequest = createRequest.Request

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_AVAILABLE)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackCreate, deployer_pb.DeploymentStatus_CREATING)
		*/

		t.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("NOP : StackStatus.Progress", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: deployer_pb.CFLifecycle_PROGRESS,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("RUNNING --> RUNNING: StackStatus.Stable", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
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

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_INFRA_MIGRATED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventMigrateData, deployer_pb.DeploymentStatus_DB_MIGRATING)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventDataMigrated, deployer_pb.DeploymentStatus_DB_MIGRATED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackScale, deployer_pb.DeploymentStatus_SCALING_UP)
		*/

		t.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("RUNNING --> DONE", func(t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
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

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_SCALED_UP)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventDone, deployer_pb.DeploymentStatus_DONE)

			t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentCompleted, deployer_pb.StackStatus_STABLE)
			t.PopStackEvent(t, deployer_pb.StackPSMEventAvailable, deployer_pb.StackStatus_AVAILABLE)
		*/

		t.AssertDeploymentStatus(t, triggerMessage.DeploymentId, deployer_pb.DeploymentStatus_DONE)

	})
}

func TestStackLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	environmentId := uuid.NewString()

	firstDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId: uuid.NewString(),
		Application: &application_pb.Application{
			Name: "app",
		},
		Version:       "1",
		EnvironmentId: environmentId,
		Flags: &deployer_pb.DeploymentFlags{
			QuickMode: true,
		},
	}

	ss.StepC("setup", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.DeployerCommand.UpsertEnvironment(ctx, &deployer_spb.UpsertEnvironmentRequest{
			EnvironmentId: environmentId,
			Config: &environment_pb.Environment{
				FullName: "env",
				Provider: &environment_pb.Environment_Aws{
					Aws: &environment_pb.AWS{
						ListenerArn: "arn:listener",
					},
				},
			},
		})
		t.NoError(err)
	})

	firstTriggerMessage := &deployer_tpb.TriggerDeploymentMessage{}
	ss.Step("Request First", func(t UniverseAsserter) {
		// Stack:  [*] --> CREATING : Trigger
		// Deployment: [*] --> QUEUED : Created
		_, err := t.DeployerTopic.RequestDeployment(ctx, firstDeploymentRequest)
		if err != nil {
			t.Fatalf("TriggerDeployment error: %v", err)
		}

		t.Outbox.PopMessage(t, firstTriggerMessage)
		t.AssertDeploymentStatus(t, firstTriggerMessage.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)

		/*
			t.PopStackEvent(t, deployer_pb.StackPSMEventTriggered, deployer_pb.StackStatus_CREATING)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
		*/

	})

	ss.Step("Trigger First", func(t UniverseAsserter) {
		// Deployment: QUEUED --> TRIGGERED --> WAITING
		// Stack: Stays CREATING
		_, err := t.DeployerTopic.TriggerDeployment(ctx, firstTriggerMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AWSStack.ExpectStabalizeStack(t)

		t.AssertDeploymentStatus(t, firstDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_WAITING)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{})

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)
		*/

	})

	ss.Step("First -> Running", func(t UniverseAsserter) {
		// Deployment WAITING --> AVAILABLE --> UPSERTING
		// Stack: Stays CREATING
		t.AWSStack.StackStatusMissing(t)
		t.AWSStack.ExpectCreateStack(t)

		t.AssertDeploymentStatus(t, firstDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{})

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_AVAILABLE)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackUpsert, deployer_pb.DeploymentStatus_UPSERTING)
		*/

	})

	secondDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
		DeploymentId:  uuid.NewString(),
		Application:   firstDeploymentRequest.Application,
		Version:       "2",
		EnvironmentId: environmentId,
	}

	ss.Step("Request a second deployment", func(t UniverseAsserter) {
		// Stack:  CREATING --> CREATING : Trigger
		// Deployment: [*] --> QUEUED : Created
		_, err := t.DeployerTopic.RequestDeployment(ctx, secondDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_QUEUED)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_CREATING,
			[]string{secondDeploymentRequest.DeploymentId})

		/*
			t.PopStackEvent(t, deployer_pb.StackPSMEventTriggered, deployer_pb.StackStatus_CREATING)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
		*/

	})

	deployment2TriggerMessage := &deployer_tpb.TriggerDeploymentMessage{}
	ss.Step("Complete the first deployment", func(t UniverseAsserter) {
		// Deployment: UPSERTING --> UPSERTED --> DONE
		// Stack: CREATING --> STABLE --> MIGRATING
		t.AWSStack.StackCreateComplete(t)

		t.AssertDeploymentStatus(t, firstDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_DONE)

		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_UPSERTED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventDone, deployer_pb.DeploymentStatus_DONE)
		*/

		t.Outbox.PopMessage(t, deployment2TriggerMessage)

	})

	ss.Step("Second -> Waiting", func(t UniverseAsserter) {
		// Deployment: QUEUED --> TRIGGERED --> WAITING
		// Stack: CREATING --> STABLE --> MIGRATING
		_, err := t.DeployerTopic.TriggerDeployment(ctx, deployment2TriggerMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AWSStack.ExpectStabalizeStack(t)
		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventTriggered, deployer_pb.DeploymentStatus_TRIGGERED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackWait, deployer_pb.DeploymentStatus_WAITING)
		*/

	})

	ss.Step("Second -> Running", func(t UniverseAsserter) {
		// Deployment WAITING --> AVAILABLE --> UPSERTING
		t.AWSStack.StackStatusMissing(t)

		t.AWSStack.ExpectCreateStack(t)

		t.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_RUNNING)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_AVAILABLE)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackUpsert, deployer_pb.DeploymentStatus_UPSERTING)
		*/

	})

	ss.Step("Terminate the second deployment", func(t UniverseAsserter) {
		// Deployment: UPSERTING --> TERMIATED
		_, err := t.DeployerCommand.TerminateDeployment(ctx, &deployer_spb.TerminateDeploymentRequest{
			DeploymentId: secondDeploymentRequest.DeploymentId,
		})

		t.NoError(err)

		t.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, deployer_pb.DeploymentStatus_TERMINATED)
		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_AVAILABLE,
			[]string{})

		/*
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventStackStatus, deployer_pb.DeploymentStatus_UPSERTED)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventDone, deployer_pb.DeploymentStatus_DONE)
		*/

		/*
			t.PopStackEvent(t, deployer_pb.StackPSMEventDeploymentCompleted, deployer_pb.StackStatus_STABLE)
			t.PopStackEvent(t, deployer_pb.StackPSMEventAvailable, deployer_pb.StackStatus_AVAILABLE)
		*/

	})

	ss.Step("A third deployment should begin immediately", func(t UniverseAsserter) {

		thirdDeploymentRequest := &deployer_tpb.RequestDeploymentMessage{
			DeploymentId:  uuid.NewString(),
			Application:   firstDeploymentRequest.Application,
			Version:       "3",
			EnvironmentId: environmentId,
		}

		// Stack: AVAILABLE --> MIGRATING : Trigger
		_, err := t.DeployerTopic.RequestDeployment(ctx, thirdDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		deployment3TriggerMessage := &deployer_tpb.TriggerDeploymentMessage{}

		t.Outbox.PopMessage(t, deployment3TriggerMessage)

		t.AssertStackStatus(t, states.StackID("env", "app"),
			deployer_pb.StackStatus_MIGRATING,
			[]string{})

		/*
			t.PopStackEvent(t, deployer_pb.StackPSMEventTriggered, deployer_pb.StackStatus_MIGRATING)
			t.PopDeploymentEvent(t, deployer_pb.DeploymentPSMEventCreated, deployer_pb.DeploymentStatus_QUEUED)
		*/

	})

}
