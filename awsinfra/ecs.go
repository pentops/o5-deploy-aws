package awsinfra

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecs_types "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ECSWorker struct {
	deployer_tpb.UnimplementedECSRequestTopicServer

	db  DBLite
	ecs ECSAPI
}

func NewECSWorker(db DBLite, ecsClient ECSAPI) (*ECSWorker, error) {
	return &ECSWorker{
		db:  db,
		ecs: ecsClient,
	}, nil
}

func (handler *ECSWorker) RunECSTask(ctx context.Context, msg *deployer_tpb.RunECSTaskMessage) (*emptypb.Empty, error) {

	clientToken, err := handler.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	_, err = handler.ecs.RunTask(ctx, &ecs.RunTaskInput{
		TaskDefinition: aws.String(msg.TaskDefinition),
		Cluster:        aws.String(msg.Cluster),
		Count:          aws.Int32(1),
		ClientToken:    aws.String(clientToken),
		Tags: []ecs_types.Tag{{
			Key:   aws.String("o5-run-task"),
			Value: aws.String(clientToken),
		}},
		StartedBy: aws.String(fmt.Sprintf("o5-run-task/%s", clientToken)),
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
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
	StartedBy     string                              `json:"startedBy"`
}

type ECSTaskStateChangeEvent_Container struct {
	ContainerArn string  `json:"containerArn"`
	LastStatus   string  `json:"lastStatus"`
	Reason       *string `json:"reason"`
	Name         string  `json:"name"`
	ExitCode     *int    `json:"exitCode"`
}

func (handler *ECSWorker) HandleECSTaskEvent(ctx context.Context, eventID string, taskEvent *ECSTaskStateChangeEvent) error {

	serviceName := "<none>"
	if taskEvent.Group != nil && strings.HasPrefix(*taskEvent.Group, "service:") {
		serviceName = strings.TrimPrefix(*taskEvent.Group, "service:")
	}
	ctx = log.WithFields(ctx, map[string]interface{}{
		"taskArn":    taskEvent.TaskArn,
		"service":    serviceName,
		"startedBy":  taskEvent.StartedBy,
		"lastStatus": taskEvent.LastStatus,
	})

	if !strings.HasPrefix(taskEvent.StartedBy, "o5-run-task/") {
		log.Debug(ctx, "ignoring task")
		return nil
	}
	taskID := strings.TrimPrefix(taskEvent.StartedBy, "o5-run-task/")

	taskContext, err := handler.db.ClientTokenToRequest(ctx, taskID)
	if err != nil {
		return err
	}

	statusMessage := &deployer_tpb.ECSTaskStatusMessage{
		Request: taskContext,
		EventId: eventID,
		TaskArn: taskEvent.TaskArn,
		Event:   &deployer_tpb.ECSTaskEventType{},
	}

	switch taskEvent.LastStatus {
	case "RUNNING":
		// Good.
		statusMessage.Event.Set(&deployer_tpb.ECSTaskEventType_Running{})

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
			statusMessage.Event.Set(&deployer_tpb.ECSTaskEventType_Failed{
				Reason: fmt.Sprintf("Task failed to start: %s", taskEvent.StoppedReason),
			})
		case "ServiceSchedulerInitiated":
			statusMessage.Event.Set(&deployer_tpb.ECSTaskEventType_Failed{
				Reason: "Scaled by service scheduller",
			})

		case "EssentialContainerExited":

			var nonZeroExit *ECSTaskStateChangeEvent_Container
			allOK := true
			reasonCodes := make([]string, 0, len(taskEvent.Containers))

			for _, container := range taskEvent.Containers {
				container := container
				if container.ExitCode != nil {
					if *container.ExitCode != 0 {
						nonZeroExit = &container
						allOK = false
					}
				} else if container.Reason != nil {
					allOK = false
					reasonCodes = append(reasonCodes, *container.Reason)
				}
			}

			if allOK {
				statusMessage.Event.Set(&deployer_tpb.ECSTaskEventType_Exited{
					ExitCode: 0,
				})
			} else if nonZeroExit != nil {
				statusMessage.Event.Set(&deployer_tpb.ECSTaskEventType_Exited{
					ExitCode:      int32(*nonZeroExit.ExitCode),
					ContainerName: nonZeroExit.Name,
				})
			} else {
				statusMessage.Event.Set(&deployer_tpb.ECSTaskEventType_Failed{
					Reason: fmt.Sprintf("Containers exited with no exit codes: %v", reasonCodes),
				})
			}

		default:
			return fmt.Errorf("unexpected stop code: %s", *taskEvent.StopCode)
		}
	default:
		return fmt.Errorf("unexpected task status: %s", taskEvent.LastStatus)
	}

	typeKey, _ := statusMessage.Event.TypeKey()
	log.WithField(ctx, "eventType", typeKey).Info("Status Event")

	return handler.db.PublishEvent(ctx, statusMessage)

}
