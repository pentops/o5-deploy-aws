package awsinfra

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type InfraEvent struct {
	Source     string          `json:"source"`
	DetailType string          `json:"detail-type"`
	Detail     json.RawMessage `json:"detail"`
}

type ECSTaskStateChangeEvent struct {
	ClusterArn    string                              `json:"clusterArn"`
	TaskArn       string                              `json:"taskArn"`
	LastStatus    string                              `json:"lastStatus"`
	StoppedAt     string                              `json:"stoppedAt"`
	StoppedReason string                              `json:"stoppedReason"`
	StopCode      *string                             `json:"stopCode"`
	Group         *string                             `json:"group"`
	Containers    []ECSTaskStateChangeEvent_Container `json:"containers"`
}

type ECSTaskStateChangeEvent_Container struct {
	ContainerArn string  `json:"containerArn"`
	LastStatus   string  `json:"lastStatus"`
	Reason       *string `json:"reason"`
	Name         string  `json:"name"`
	ExitCode     *int    `json:"exitCode"`
}

func handleECSTaskEvent(taskEvent *ECSTaskStateChangeEvent) error {

	if taskEvent.Group == nil || !strings.HasPrefix(*taskEvent.Group, "service:") {
		return nil
	}
	serviceName := strings.TrimPrefix(*taskEvent.Group, "service:")

	fmt.Printf("Task in %s %s: %s\n", serviceName, taskEvent.TaskArn, taskEvent.LastStatus)
	switch taskEvent.LastStatus {
	case "RUNNING":
		// Good.
		return nil

	case "PENDING",
		"PROVISIONING",
		"DEACTIVATING",
		"ACTIVATING",
		"DEPROVISIONING",
		"STOPPING":
		// Transient

		return nil
	case "STOPPED":
		if taskEvent.StopCode == nil {
			return fmt.Errorf("task %s stopped with no code: %s", taskEvent.TaskArn, taskEvent.StoppedReason)
		}

		switch *taskEvent.StopCode {
		case "TaskFailedToStart":
			for _, container := range taskEvent.Containers {
				if container.ExitCode != nil {
					fmt.Printf("Container %s exited with code %d\n", container.Name, *container.ExitCode)
				} else if container.Reason != nil {
					fmt.Printf("Container %s exited: %s\n", container.Name, *container.Reason)
				}
			}
			fmt.Printf("Task %s failed to start: %s\n", taskEvent.TaskArn, taskEvent.StoppedReason)

			return nil
		case "ServiceSchedulerInitiated":
			fmt.Printf("Normal scaling activity: %s\n", taskEvent.StoppedReason)
			return nil
		default:
			return fmt.Errorf("unexpected stop code: %s", *taskEvent.StopCode)
		}
	default:
		return fmt.Errorf("unexpected task status: %s", taskEvent.LastStatus)
	}

}

func HandleInfraEvent(ctx context.Context, infraEvent *InfraEvent) error {

	if infraEvent.Source == "aws.ecs" && infraEvent.DetailType == "ECS Task State Change" {
		taskEvent := &ECSTaskStateChangeEvent{}
		if err := json.Unmarshal(infraEvent.Detail, taskEvent); err != nil {
			return err
		}

		if err := handleECSTaskEvent(taskEvent); err != nil {
			return fmt.Errorf("failed to handle ECS task event: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("unhandled infra event: %s %s", infraEvent.Source, infraEvent.DetailType)
	}
}
