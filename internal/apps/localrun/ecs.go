package localrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsapi"
)

type ecsRunner struct {
	ecsClient        awsapi.ECSAPI
	cloudwatchClient awsapi.CloudWatchLogsAPI
}

func (d *ecsRunner) runMigrationTask(ctx context.Context, migrationID string, msg *awsinfra_tpb.MigratePostgresDatabaseMessage) error {
	ecsClient := d.ecsClient

	task, err := ecsClient.RunTask(ctx, &ecs.RunTaskInput{
		TaskDefinition: aws.String(msg.MigrationTaskArn),
		Cluster:        aws.String(msg.EcsClusterName),
		Count:          aws.Int32(1),
		ClientToken:    aws.String(migrationID),
		StartedBy:      aws.String("o5-local-run"),
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gotLogGroups := false
	for {
		state, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Tasks:   []string{*task.Tasks[0].TaskArn},
			Cluster: aws.String(msg.EcsClusterName),
		})
		if err != nil {
			return err
		}

		if len(state.Tasks) != 1 {
			return fmt.Errorf("expected 1 task, got %d", len(state.Tasks))
		}

		task := state.Tasks[0]
		log.WithFields(ctx, map[string]interface{}{
			"status": *task.LastStatus,
		}).Debug("waiting for task to stop")

		if !gotLogGroups {
			gotLogGroups = true
			logGroups, err := d.findLogGroups(ctx, task)
			if err != nil {
				return err
			}
			for _, lg := range logGroups {
				log.WithFields(ctx, map[string]interface{}{
					"logGroup":  lg.LogGroup,
					"logStream": lg.LogStream,
				}).Info("log group")
				go func(lg LogStream) {
					tailLogStream(ctx, d.cloudwatchClient, lg, time.Now().Add(-1*time.Minute))
				}(lg)
			}

		}

		if *task.LastStatus != "STOPPED" {
			time.Sleep(time.Second)
			continue
		}

		containers := make([]types.Container, 0, len(state.Tasks[0].Containers))
		for _, c := range state.Tasks[0].Containers {
			if stringValue(c.Name) == appbuilder.O5SidecarContainerName {
				continue
			}
			containers = append(containers, c)
		}

		if len(containers) != 1 {
			return fmt.Errorf("expected 1 container, got %d", len(containers))
		}
		container := state.Tasks[0].Containers[0]
		if container.ExitCode == nil {
			if task.StoppedReason != nil && *task.StoppedReason != "" {
				return fmt.Errorf("task stopped with reason: %s", *task.StoppedReason)
			}
			return fmt.Errorf("task stopped with no exit code: %s", stringValue(container.Reason))
		}
		if *container.ExitCode != 0 {
			return fmt.Errorf("exit code was %d", *container.ExitCode)
		}
		return nil
	}
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type LogStream struct {
	Container string
	LogGroup  string
	LogStream string
}

func (d *ecsRunner) findLogGroups(ctx context.Context, task types.Task) ([]LogStream, error) {
	// Log Group is ecs/$env/$appName/$runtimeName

	taskDefRes, err := d.ecsClient.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task.TaskDefinitionArn,
	})
	if err != nil {
		return nil, err
	}
	taskDef := taskDefRes.TaskDefinition

	streams := make([]LogStream, 0, len(taskDef.ContainerDefinitions))
	for _, container := range taskDef.ContainerDefinitions {
		if container.LogConfiguration == nil || container.LogConfiguration.LogDriver != "awslogs" {
			continue
		}

		logGroup := container.LogConfiguration.Options["awslogs-group"]
		streamPrefix := container.LogConfiguration.Options["awslogs-stream-prefix"]

		taskID := splitPart("/", 2, *task.TaskArn)
		logStream := fmt.Sprintf("%s/%s/%s", streamPrefix, *container.Name, taskID)

		fmt.Printf("  %s %s\n", logGroup, logStream)
		streams = append(streams, LogStream{
			Container: *container.Name,
			LogGroup:  logGroup,
			LogStream: logStream,
		})
	}

	return streams, nil
}

func splitPart(sep string, n int, s string) string {
	parts := strings.Split(s, sep)
	if len(parts) <= n {
		return ""
	}
	return parts[n]
}

func tailLogStream(ctx context.Context, client awsapi.CloudWatchLogsAPI, logGroup LogStream, fromTime time.Time) {

	fromTimeInt := fromTime.UnixNano() / int64(time.Millisecond)
	var nextToken *string
	for {
		time.Sleep(time.Second)

		logEvents, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroup.LogGroup,
			LogStreamName: &logGroup.LogStream,
			StartTime:     &fromTimeInt,
			StartFromHead: aws.Bool(true),
			NextToken:     nextToken,
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Debug(ctx, "context canceled, stopping log tail")
				return
			}

			log.WithError(ctx, err).Error("error getting log events")
			continue
		}
		nextToken = logEvents.NextForwardToken
		if nextToken == nil {
			log.Error(ctx, "no next token")
			continue
		}

		for _, event := range logEvents.Events {
			if strings.HasPrefix(*event.Message, "{") {
				msg := make(map[string]interface{})
				if err := json.Unmarshal([]byte(*event.Message), &msg); err == nil {
					log.WithFields(ctx, msg).Info("log message")
					continue
				}
			}
			log.WithFields(ctx, map[string]interface{}{
				"container": logGroup.Container,
				"message":   event.Message,
			}).Info("log message")
		}

	}
}
