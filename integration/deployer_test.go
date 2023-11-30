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
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
)

func TestCreateHappy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, t)
	defer uu.RunSteps(t)

	initialTrigger := &deployer_tpb.TriggerDeploymentMessage{}
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

	var stackID *deployer_tpb.StackID

	uu.Step("[*] --> QUEUED --> LOCKED --> WAITING", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.TriggerDeployment(ctx, initialTrigger)
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

func TestStackLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, t)
	defer uu.RunSteps(t)

	initialTrigger := &deployer_tpb.TriggerDeploymentMessage{
		DeploymentId: uuid.NewString(),
		Spec: &deployer_pb.DeploymentSpec{
			AppName:         "app",
			Version:         "1",
			EnvironmentName: "env",
			QuickMode:       true,
		},
	}

	var deployment1 *deployer_tpb.StackID

	uu.Step("Trigger First", func(t flowtest.Asserter) {
		// Stack:  [*] --> CREATING : Trigger
		// Deployment: [*] --> TRIGGERED --> WAITING --> AVAILABLE --> UPSERTING
		_, err := uu.DeployerTopic.TriggerDeployment(ctx, initialTrigger)
		if err != nil {
			t.Fatalf("TriggerDeployment error: %v", err)
		}

		stabalizeRequest := &deployer_tpb.StabalizeStackMessage{}
		uu.Outbox.PopMessage(t, stabalizeRequest)
		uu.Outbox.AssertNoMessages(t)

		deployment1 = stabalizeRequest.StackId

		uu.AssertDeploymentStatus(t, deployment1.DeploymentId, deployer_pb.DeploymentStatus_WAITING)

		stackStatus, err := uu.DeployerQuery.GetStack(ctx, &deployer_spb.GetStackRequest{
			StackId: deployer.StackID("env", "app"),
		})
		if err != nil {
			t.Fatalf("GetStack error: %v", err)
		}

		t.Equal(deployer_pb.StackStatus_CREATING, stackStatus.State.Status)

		_, err = uu.DeployerTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   deployment1,
			Lifecycle: deployer_pb.StackLifecycle_STACK_LIFECYCLE_MISSING,
			Status:    "FOOBAR",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		createRequest := &deployer_tpb.CreateNewStackMessage{}
		uu.Outbox.PopMessage(t, createRequest)
		uu.Outbox.AssertNoMessages(t)

		uu.AssertDeploymentStatus(t, deployment1.DeploymentId, deployer_pb.DeploymentStatus_UPSERTING)

	})

	secondTrigger := &deployer_tpb.TriggerDeploymentMessage{
		DeploymentId: uuid.NewString(),
		Spec: &deployer_pb.DeploymentSpec{
			AppName:         "app",
			Version:         "2",
			EnvironmentName: "env",
			QuickMode:       false,
		},
	}

	uu.Step("CREATING --> CREATING : Trigger", func(t flowtest.Asserter) {
		// Stack:  CREATING --> CREATING : Trigger
		// Deployment: should not be created yet?
		_, err := uu.DeployerTopic.TriggerDeployment(ctx, secondTrigger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		uu.Outbox.AssertNoMessages(t)

		stackStatus, err := uu.DeployerQuery.GetStack(ctx, &deployer_spb.GetStackRequest{
			StackId: deployer.StackID("env", "app"),
		})
		if err != nil {
			t.Fatalf("GetStack error: %v", err)
		}

		t.Equal(deployer_pb.StackStatus_CREATING, stackStatus.State.Status)
		if len(stackStatus.State.QueuedDeployments) != 1 {
			t.Fatalf("unexpected queued deployments: %v", stackStatus.State.QueuedDeployments)
		}

		d := stackStatus.State.QueuedDeployments[0]
		t.Equal(secondTrigger.DeploymentId, d.DeploymentId)
	})

	deployment1CompleteMessage := &deployer_tpb.DeploymentCompleteMessage{}
	uu.Step("CREATING --> STABLE : StackStatus.Stable", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.StackStatusChanged(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   deployment1,
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

		uu.AssertDeploymentStatus(t, deployment1.DeploymentId, deployer_pb.DeploymentStatus_DONE)

		uu.Outbox.PopMessage(t, deployment1CompleteMessage)
		uu.Outbox.AssertNoMessages(t)
	})

	uu.Step("STABLE --> STABLE : StackStatus.Stable", func(t flowtest.Asserter) {
		_, err := uu.DeployerTopic.DeploymentComplete(ctx, deployment1CompleteMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		stackStatus, err := uu.DeployerQuery.GetStack(ctx, &deployer_spb.GetStackRequest{
			StackId: deployer.StackID("env", "app"),
		})
		if err != nil {
			t.Fatalf("GetStack error: %v", err)
		}

		t.Equal(deployer_pb.StackStatus_STABLE, stackStatus.State.Status)
		if len(stackStatus.State.QueuedDeployments) != 0 {
			t.Fatalf("unexpected queued deployments: %v", stackStatus.State.QueuedDeployments)
		}
	})

}
