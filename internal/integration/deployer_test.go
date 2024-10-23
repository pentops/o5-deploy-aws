package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/messaging/v1/messaging_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/j5/drss/v1/drss_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/deployer/plan/planbuild"
	"github.com/pentops/realms/authtest"

	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
)

func tValidApp(name string) *application_pb.Application {
	return &application_pb.Application{
		Name: name,
		Runtimes: []*application_pb.Runtime{{
			Name: "main",
			Containers: []*application_pb.Container{{
				Name: "main",
				Source: &application_pb.Container_Image_{
					Image: &application_pb.Container_Image{
						Name: "image",
					},
				},
			}},
		}},
	}
}

func tValidCombinedCluster(clusterName, envName string) *environment_pb.CombinedConfig {
	return &environment_pb.CombinedConfig{
		Name: clusterName,
		Provider: &environment_pb.CombinedConfig_Aws{
			Aws: tValidAWSCluster(),
		},
		Environments: []*environment_pb.Environment{{
			FullName: fmt.Sprintf("%s-%s", clusterName, envName),
			Provider: &environment_pb.Environment_Aws{
				Aws: &environment_pb.AWSEnvironment{
					HostHeader: aws.String("host"),
				},
			},
		}},
	}
}

func tValidAWSCluster() *environment_pb.AWSCluster {
	return &environment_pb.AWSCluster{
		EcsCluster: &environment_pb.ECSCluster{
			ClusterName: "cluster",
		},
		O5Sidecar: &environment_pb.O5Sidecar{
			ImageVersion: "version1",
		},
		AlbIngress: &environment_pb.ALBIngress{
			ListenerArn: "listener",
		},
		EventBridge: &environment_pb.EventBridge{
			EventBusArn: "bus",
		},
		O5Deployer: &environment_pb.O5Deployer{
			AssumeRoleArn: "role",
		},
	}
}

func TestDeploymentFlow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	// Trigger the deployment by pushing to the configured branch
	request := &awsdeployer_tpb.RequestDeploymentMessage{
		DeploymentId: uuid.NewString(),
		Application:  tValidApp("app"),
		Version:      "1",
	}

	var environmentID string
	ss.Step("Configure Stack", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)

		_, err := t.DeployerCommand.UpsertCluster(ctx, &awsdeployer_spb.UpsertClusterRequest{
			ClusterId: "cluster",
			Src: &awsdeployer_spb.UpsertClusterRequest_Config{
				Config: &environment_pb.CombinedConfig{
					Name: "cluster",
					Provider: &environment_pb.CombinedConfig_Aws{
						Aws: tValidAWSCluster(),
					},
				},
			},
		})
		t.NoError(err)

		_, err = t.DeployerCommand.UpsertEnvironment(ctx, &awsdeployer_spb.UpsertEnvironmentRequest{
			EnvironmentId: "cluster-env",
			ClusterId:     "cluster",
			Src: &awsdeployer_spb.UpsertEnvironmentRequest_Config{
				Config: &environment_pb.Environment{
					FullName: "cluster-env",
					Provider: &environment_pb.Environment_Aws{
						Aws: &environment_pb.AWSEnvironment{
							HostHeader: aws.String("host"),
						},
					},
				},
			},
		})
		t.NoError(err)

		envConfitured := t.PopEnvironmentEvent(t, awsdeployer_pb.EnvironmentPSMEventConfigured, awsdeployer_pb.EnvironmentStatus_ACTIVE)
		environmentID = envConfitured.Event.Keys.EnvironmentId

		request.EnvironmentId = environmentID
	})

	var stackRequest *messaging_j5pb.RequestMetadata
	var stackName string
	ss.Step("Deployment Queued To Triggered", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.DeployerTopic.RequestDeployment(ctx, request)
		t.NoError(err)

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventCreated, awsdeployer_pb.DeploymentStatus_QUEUED)
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentRequested, awsdeployer_pb.StackStatus_AVAILABLE)
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventRunDeployment, awsdeployer_pb.StackStatus_MIGRATING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventTriggered, awsdeployer_pb.DeploymentStatus_TRIGGERED)

		stabalizeRequest := &awsinfra_tpb.StabalizeStackMessage{
			Request: &messaging_j5pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, stabalizeRequest)
		stackRequest = stabalizeRequest.Request
		stackName = stabalizeRequest.StackName

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackWait, awsdeployer_pb.DeploymentStatus_WAITING)

		t.AssertDeploymentStatus(t, request.DeploymentId, awsdeployer_pb.DeploymentStatus_WAITING)
	})

	ss.Step("CF Stack Missing Create New Stack", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			EventId:   "evt0",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: awsdeployer_pb.CFLifecycle_MISSING,
			Status:    "MISSING",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		createRequest := &awsinfra_tpb.CreateNewStackMessage{
			Request: &messaging_j5pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, createRequest)
		stackRequest = createRequest.Request

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackAvailable, awsdeployer_pb.DeploymentStatus_AVAILABLE)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunSteps, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunStep, awsdeployer_pb.DeploymentStatus_RUNNING)

		deploymentState := t.AssertDeploymentStatus(t, request.DeploymentId, awsdeployer_pb.DeploymentStatus_RUNNING)

		assertStepStatus(t, deploymentState, map[string]drss_pb.StepStatus{
			planbuild.StepCFCreateEmpty: drss_pb.StepStatus_ACTIVE,
			planbuild.StepCFUpdate:      drss_pb.StepStatus_BLOCKED,
			planbuild.StepScaleUp:       drss_pb.StepStatus_BLOCKED,
		})
	})

	ss.Step("StackStatus Progress NOP", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			EventId:   "evt1",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: awsdeployer_pb.CFLifecycle_PROGRESS,
			Status:    "PROGRESS",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertDeploymentStatus(t, request.DeploymentId, awsdeployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("StackStatus Stable to Update", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			EventId:   "evt2",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Status:    "FOOBAR",
			Outputs: []*awsdeployer_pb.KeyValue{{
				Name:  "foo",
				Value: "bar",
			}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		deploymentState := t.AssertDeploymentStatus(t, request.DeploymentId, awsdeployer_pb.DeploymentStatus_RUNNING)

		assertStepStatus(t, deploymentState, map[string]drss_pb.StepStatus{
			planbuild.StepCFCreateEmpty: drss_pb.StepStatus_DONE,
			planbuild.StepCFUpdate:      drss_pb.StepStatus_ACTIVE,
			planbuild.StepScaleUp:       drss_pb.StepStatus_BLOCKED,
		})

		migrateRequest := &awsinfra_tpb.UpdateStackMessage{
			Request: &messaging_j5pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, migrateRequest)
		stackRequest = migrateRequest.Request

		t.Equal(int(0), int(migrateRequest.Spec.DesiredCount))

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStepResult, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunStep, awsdeployer_pb.DeploymentStatus_RUNNING)
	})

	ss.Step("StackStatus Stable to Scale Up", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			EventId:   "evt3",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Status:    "FOOBAR",
			Outputs: []*awsdeployer_pb.KeyValue{{
				Name:  "foo",
				Value: "bar",
			}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		deploymentState := t.AssertDeploymentStatus(t, request.DeploymentId, awsdeployer_pb.DeploymentStatus_RUNNING)

		assertStepStatus(t, deploymentState, map[string]drss_pb.StepStatus{
			planbuild.StepCFCreateEmpty: drss_pb.StepStatus_DONE,
			planbuild.StepCFUpdate:      drss_pb.StepStatus_DONE,
			planbuild.StepScaleUp:       drss_pb.StepStatus_ACTIVE,
		})

		scaleUpRequest := &awsinfra_tpb.ScaleStackMessage{
			Request: &messaging_j5pb.RequestMetadata{},
		}
		t.Outbox.PopMessage(t, scaleUpRequest)
		stackRequest = scaleUpRequest.Request

		t.Equal(int(1), int(scaleUpRequest.DesiredCount))

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStepResult, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunStep, awsdeployer_pb.DeploymentStatus_RUNNING)

	})

	ss.Step("RUNNING --> DONE", func(ctx context.Context, t UniverseAsserter) {
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, &awsinfra_tpb.StackStatusChangedMessage{
			EventId:   "evt4",
			Request:   stackRequest,
			StackName: stackName,
			Lifecycle: awsdeployer_pb.CFLifecycle_COMPLETE,
			Status:    "FOOBAR",
			Outputs: []*awsdeployer_pb.KeyValue{{
				Name:  "foo",
				Value: "bar",
			}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStepResult, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventDone, awsdeployer_pb.DeploymentStatus_DONE)

		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentCompleted, awsdeployer_pb.StackStatus_AVAILABLE)

		t.AssertDeploymentStatus(t, request.DeploymentId, awsdeployer_pb.DeploymentStatus_DONE)

	})
}

func TestStackLock(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ss := NewStepper(ctx, t)
	defer ss.RunSteps(t)

	var environmentID string
	var stackID string

	firstDeploymentID := uuid.NewString()
	secondDeploymentID := uuid.NewString()

	ss.Step("setup", func(ctx context.Context, t UniverseAsserter) {
		ctx = authtest.JWTContext(ctx)
		_, err := t.DeployerCommand.UpsertCluster(ctx, &awsdeployer_spb.UpsertClusterRequest{
			ClusterId: "cluster",
			Src: &awsdeployer_spb.UpsertClusterRequest_Config{
				Config: tValidCombinedCluster("cluster", "env"),
			},
		})
		t.NoError(err)

		t.NoError(err)
		env := t.PopEnvironmentEvent(t, awsdeployer_pb.EnvironmentPSMEventConfigured, awsdeployer_pb.EnvironmentStatus_ACTIVE)
		environmentID = env.Event.Keys.EnvironmentId
	})

	ss.Step("Request and begin first deployment", func(ctx context.Context, t UniverseAsserter) {
		firstDeploymentRequest := &awsdeployer_tpb.RequestDeploymentMessage{
			DeploymentId:  firstDeploymentID,
			Application:   tValidApp("app"),
			Version:       "1",
			EnvironmentId: environmentID,
			Flags: &awsdeployer_pb.DeploymentFlags{
				QuickMode: true,
			},
		}

		_, err := t.DeployerTopic.RequestDeployment(ctx, firstDeploymentRequest)
		if err != nil {
			t.Fatalf("TriggerDeployment error: %v", err)
		}

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventCreated, awsdeployer_pb.DeploymentStatus_QUEUED)
		stackEvt := t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentRequested, awsdeployer_pb.StackStatus_AVAILABLE)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventTriggered, awsdeployer_pb.DeploymentStatus_TRIGGERED)
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventRunDeployment, awsdeployer_pb.StackStatus_MIGRATING)

		t.AWSStack.ExpectStabalizeStack(t)
		stackID = stackEvt.Event.Keys.StackId

		t.AssertDeploymentStatus(t, firstDeploymentID, awsdeployer_pb.DeploymentStatus_WAITING)
		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_MIGRATING,
			[]string{})

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackWait, awsdeployer_pb.DeploymentStatus_WAITING)
	})

	ss.Step("First -> Running", func(ctx context.Context, t UniverseAsserter) {
		// Deployment WAITING --> AVAILABLE --> RUNNING
		// Stack: Stays MIGRATING
		t.AWSStack.StackStatusMissing(t)

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackAvailable, awsdeployer_pb.DeploymentStatus_AVAILABLE)
		t.AWSStack.ExpectCreateStack(t)

		t.AssertDeploymentStatus(t, firstDeploymentID, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_MIGRATING,
			[]string{})

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunSteps, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunStep, awsdeployer_pb.DeploymentStatus_RUNNING)

	})

	ss.Step("Request a second deployment", func(ctx context.Context, t UniverseAsserter) {
		secondDeploymentRequest := &awsdeployer_tpb.RequestDeploymentMessage{
			DeploymentId:  secondDeploymentID,
			Application:   tValidApp("app"),
			Version:       "2",
			EnvironmentId: environmentID,
		}
		// Stack:  MIGRATING --> MIGRATING : Trigger
		// Deployment: [*] --> QUEUED : Created
		_, err := t.DeployerTopic.RequestDeployment(ctx, secondDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertDeploymentStatus(t, secondDeploymentRequest.DeploymentId, awsdeployer_pb.DeploymentStatus_QUEUED)
		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_MIGRATING,
			[]string{secondDeploymentRequest.DeploymentId})

		// New request, but no change to the status.
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentRequested, awsdeployer_pb.StackStatus_MIGRATING)

		// Deployment is blocked.
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventCreated, awsdeployer_pb.DeploymentStatus_QUEUED)

	})

	ss.Step("Complete the first deployment", func(ctx context.Context, t UniverseAsserter) {
		// Deployment: UPSERTING --> UPSERTED --> DONE
		// Stack: CREATING --> STABLE --> MIGRATING
		t.AWSStack.StackCreateComplete(t)
		ss := t.AssertDeploymentStatus(t, firstDeploymentID, awsdeployer_pb.DeploymentStatus_RUNNING)
		assertStepStatus(t, ss, map[string]drss_pb.StepStatus{
			planbuild.StepCFCreateEmpty: drss_pb.StepStatus_DONE,
			planbuild.StepCFUpdate:      drss_pb.StepStatus_ACTIVE,
		})

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStepResult, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunStep, awsdeployer_pb.DeploymentStatus_RUNNING)

		t.AWSStack.ExpectUpdateStack(t)
		t.AWSStack.StackUpdateComplete(t)

		ss = t.AssertDeploymentStatus(t, firstDeploymentID, awsdeployer_pb.DeploymentStatus_DONE)
		assertStepStatus(t, ss, map[string]drss_pb.StepStatus{
			planbuild.StepCFCreateEmpty: drss_pb.StepStatus_DONE,
			planbuild.StepCFUpdate:      drss_pb.StepStatus_DONE,
		})

		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_MIGRATING,
			[]string{})

		// First Deployment Completes
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStepResult, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventDone, awsdeployer_pb.DeploymentStatus_DONE)

		// Stack unblocks and re-triggers
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentCompleted, awsdeployer_pb.StackStatus_AVAILABLE)
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventRunDeployment, awsdeployer_pb.StackStatus_MIGRATING)

		// The second deployment begins
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventTriggered, awsdeployer_pb.DeploymentStatus_TRIGGERED)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackWait, awsdeployer_pb.DeploymentStatus_WAITING)

		t.AWSStack.ExpectStabalizeStack(t)

	})

	var secondStepCompleteMessage *awsinfra_tpb.StackStatusChangedMessage

	ss.Step("Second -> Running", func(ctx context.Context, t UniverseAsserter) {
		// Deployment WAITING --> AVAILABLE --> UPSERTING
		t.AWSStack.StackStatusMissing(t)

		t.AWSStack.ExpectCreateStack(t)

		t.AssertDeploymentStatus(t, secondDeploymentID, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_MIGRATING,
			[]string{})

		// capture the StackCompleteMessage, but don't send it in yet.
		secondStepCompleteMessage = t.AWSStack.StackCreateCompleteMessage()

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackAvailable, awsdeployer_pb.DeploymentStatus_AVAILABLE)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunSteps, awsdeployer_pb.DeploymentStatus_RUNNING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventRunStep, awsdeployer_pb.DeploymentStatus_RUNNING)

	})

	ss.Step("Terminate the second deployment", func(ctx context.Context, t UniverseAsserter) {
		// Deployment: UPSERTING --> TERMIATED
		ctx = authtest.JWTContext(ctx)
		_, err := t.DeployerCommand.TerminateDeployment(ctx, &awsdeployer_spb.TerminateDeploymentRequest{
			DeploymentId: secondDeploymentID,
		})

		t.NoError(err)

		t.AssertDeploymentStatus(t, secondDeploymentID, awsdeployer_pb.DeploymentStatus_TERMINATED)
		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_AVAILABLE,
			[]string{})

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventTerminated, awsdeployer_pb.DeploymentStatus_TERMINATED)

		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentCompleted, awsdeployer_pb.StackStatus_AVAILABLE)

	})

	ss.Step("A third deployment should begin immediately", func(ctx context.Context, t UniverseAsserter) {

		thirdDeploymentRequest := &awsdeployer_tpb.RequestDeploymentMessage{
			DeploymentId:  uuid.NewString(),
			Application:   tValidApp("app"),
			Version:       "3",
			EnvironmentId: environmentID,
		}

		// Stack: AVAILABLE --> MIGRATING : Trigger
		_, err := t.DeployerTopic.RequestDeployment(ctx, thirdDeploymentRequest)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.AssertStackStatus(t, stackID,
			awsdeployer_pb.StackStatus_MIGRATING,
			[]string{})

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventCreated, awsdeployer_pb.DeploymentStatus_QUEUED)
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventDeploymentRequested, awsdeployer_pb.StackStatus_AVAILABLE)
		t.PopStackEvent(t, awsdeployer_pb.StackPSMEventRunDeployment, awsdeployer_pb.StackStatus_MIGRATING)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventTriggered, awsdeployer_pb.DeploymentStatus_TRIGGERED)
		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStackWait, awsdeployer_pb.DeploymentStatus_WAITING)

		t.AWSStack.ExpectStabalizeStack(t)
	})

	ss.Step("A step in the second deployment completes after termination", func(ctx context.Context, t UniverseAsserter) {
		// Complete the StackComplete message from the second, now terminated,
		// deployment
		_, err := t.CFReplyTopic.StackStatusChanged(ctx, secondStepCompleteMessage)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.PopDeploymentEvent(t, awsdeployer_pb.DeploymentPSMEventStepResult, awsdeployer_pb.DeploymentStatus_TERMINATED)
		fullState := t.AssertDeploymentStatus(t, secondDeploymentID, awsdeployer_pb.DeploymentStatus_TERMINATED)
		upsertStep := fullState.Data.Steps[0]
		if upsertStep.Meta.Status != drss_pb.StepStatus_DONE {
			t.Fatalf("expected step status DONE, got %s", upsertStep.Meta.Status)
		}

	})

}
