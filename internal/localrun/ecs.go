package localrun

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/internal/awsinfra"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
)

type ecsRunner struct {
	ecsClient awsinfra.ECSAPI
}

func (d *ecsRunner) runMigrationTask(ctx context.Context, migrationID string, msg *awsdeployer_pb.PostgresMigrationSpec) error {
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

		if *task.LastStatus != "STOPPED" {
			time.Sleep(time.Second)
			continue
		}

		if len(state.Tasks[0].Containers) != 1 {
			return fmt.Errorf("expected 1 container, got %d", len(state.Tasks[0].Containers))
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
