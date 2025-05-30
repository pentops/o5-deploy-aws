package aws_ecs

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecs_types "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsapi"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/tokenstore"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ECSWorker struct {
	awsinfra_tpb.UnimplementedECSRequestTopicServer

	db  tokenstore.DBLite
	ecs awsapi.ECSAPI
}

func NewECSWorker(db tokenstore.DBLite, ecsClient awsapi.ECSAPI) (*ECSWorker, error) {
	return &ECSWorker{
		db:  db,
		ecs: ecsClient,
	}, nil
}

func RunTaskInput(msg *awsinfra_tpb.RunECSTaskMessage) (*ecs.RunTaskInput, error) {
	var network *ecs_types.NetworkConfiguration
	if msg.Context.Network != nil {
		switch nt := msg.Context.Network.Get().(type) {
		case *awsdeployer_pb.ECSTaskNetworkType_AWSVPC:
			network = &ecs_types.NetworkConfiguration{
				AwsvpcConfiguration: &ecs_types.AwsVpcConfiguration{
					SecurityGroups: nt.SecurityGroups,
					Subnets:        nt.Subnets,
				},
			}

		default:
			return nil, fmt.Errorf("unsupported network type: %T", msg.Context.Network)
		}
	}
	return &ecs.RunTaskInput{
		TaskDefinition:       aws.String(msg.TaskDefinition),
		Cluster:              aws.String(msg.Context.Cluster),
		Count:                aws.Int32(1),
		NetworkConfiguration: network,
	}, nil

}

func (handler *ECSWorker) RunECSTask(ctx context.Context, msg *awsinfra_tpb.RunECSTaskMessage) (*emptypb.Empty, error) {

	clientToken, err := handler.db.RequestToClientToken(ctx, msg.Request)
	if err != nil {
		return nil, err
	}

	runTaskInput, err := RunTaskInput(msg)
	if err != nil {
		return nil, err
	}
	runTaskInput.StartedBy = aws.String(fmt.Sprintf("o5-run-task/%s", clientToken))
	runTaskInput.Tags = append(runTaskInput.Tags, ecs_types.Tag{
		Key:   aws.String("o5-run-task"),
		Value: aws.String(clientToken),
	})
	runTaskInput.ClientToken = aws.String(clientToken)

	_, err = handler.ecs.RunTask(ctx, runTaskInput)
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
	ctx = log.WithFields(ctx, map[string]any{
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

	statusMessage := &awsinfra_tpb.ECSTaskStatusMessage{
		Request: taskContext,
		EventId: eventID,
		TaskArn: taskEvent.TaskArn,
		Event:   &awsinfra_tpb.ECSTaskEventType{},
	}

	switch taskEvent.LastStatus {
	case "RUNNING":
		// Good.
		statusMessage.Event.Set(&awsinfra_tpb.ECSTaskEventType_Running{})

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
			statusMessage.Event.Set(&awsinfra_tpb.ECSTaskEventType_Failed{
				Reason: fmt.Sprintf("Task failed to start: %s", taskEvent.StoppedReason),
			})
		case "ServiceSchedulerInitiated":
			statusMessage.Event.Set(&awsinfra_tpb.ECSTaskEventType_Failed{
				Reason: "Scaled by service scheduller",
			})

		case "EssentialContainerExited":
			msg, err := getExitedEvent(taskEvent)
			if err != nil {
				return err
			}
			statusMessage.Event.Set(msg)

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
func getExitedEvent(taskEvent *ECSTaskStateChangeEvent) (awsinfra_tpb.IsECSTaskEventTypeWrappedType, error) {

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
		return &awsinfra_tpb.ECSTaskEventType_Exited{
			ExitCode: 0,
		}, nil
	}
	if nonZeroExit == nil {
		return &awsinfra_tpb.ECSTaskEventType_Failed{
			Reason: fmt.Sprintf("Containers exited with no exit codes: %v", reasonCodes),
		}, nil
	}

	exited := &awsinfra_tpb.ECSTaskEventType_Exited{
		ExitCode:      int32(*nonZeroExit.ExitCode),
		ContainerName: nonZeroExit.Name,
	}

	return exited, nil
}
