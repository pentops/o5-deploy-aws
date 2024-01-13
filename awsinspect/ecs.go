package awsinspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type FormationClient interface {
	DescribeStackResources(ctx context.Context, input *cloudformation.DescribeStackResourcesInput, opts ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
	DescribeStacks(ctx context.Context, input *cloudformation.DescribeStacksInput, opts ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type ECSClient interface {
	DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput, opts ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
	ListTasks(ctx context.Context, input *ecs.ListTasksInput, opts ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput, opts ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
	DescribeTaskDefinition(ctx context.Context, input *ecs.DescribeTaskDefinitionInput, opts ...func(*ecs.Options)) (*ecs.DescribeTaskDefinitionOutput, error)
}

type ServiceSummary struct {
	ClusterName string
	ServiceARNs []string
}

func GetStackServices(ctx context.Context, client FormationClient, stackName string) (*ServiceSummary, error) {

	res, err := client.DescribeStackResources(ctx, &cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}

	serviceArns := []string{}
	for _, resource := range res.StackResources {
		if *resource.ResourceType == "AWS::ECS::Service" {
			arn := *resource.PhysicalResourceId
			serviceArns = append(serviceArns, arn)
		}
	}

	clusterName, err := func() (string, error) {
		stackRes, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		})
		if err != nil {
			return "", err
		}
		if len(stackRes.Stacks) != 1 {
			return "", fmt.Errorf("unexpected number of stacks: %d", len(stackRes.Stacks))
		}

		for _, param := range stackRes.Stacks[0].Parameters {
			if *param.ParameterKey == "ECSCluster" {
				return *param.ParameterValue, nil
			}
		}

		return "", fmt.Errorf("no ECSCluster parameter found")
	}()
	if err != nil {
		return nil, err
	}

	return &ServiceSummary{
		ClusterName: clusterName,
		ServiceARNs: serviceArns,
	}, nil
}

func GetAllLogStreams(ctx context.Context, ecsClient ECSClient, serviceSummary *ServiceSummary) ([]LogStream, error) {

	streams := []LogStream{}
	for _, arn := range serviceSummary.ServiceARNs {

		ecsRes, err := ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(serviceSummary.ClusterName),
			Services: []string{arn},
		})
		if err != nil {
			return nil, err
		}

		if len(ecsRes.Services) != 1 {
			return nil, fmt.Errorf("unexpected number of services: %d", len(ecsRes.Services))
		}

		service := ecsRes.Services[0]

		fmt.Printf("Service %s:\n", *service.ServiceName)

		for _, deployment := range service.Deployments {
			taskDef := splitPart("/", 1, *deployment.TaskDefinition)
			fmt.Printf("  Deployment %s: %s (%d running, %d failed)\n", *deployment.Status, taskDef, deployment.RunningCount, deployment.FailedTasks)
		}

		taskListRes, err := ecsClient.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:     aws.String(serviceSummary.ClusterName),
			ServiceName: aws.String(*service.ServiceName),
		})
		if err != nil {
			return nil, err
		}

		taskRes, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: aws.String(serviceSummary.ClusterName),
			Tasks:   taskListRes.TaskArns,
		})
		if err != nil {
			return nil, err
		}

		taskDefinitions := map[string]*types.TaskDefinition{}

		// Log Group is ecs/$env/$appName/$runtimeName
		for _, task := range taskRes.Tasks {
			fmt.Printf("  Task %s: %s\n", *task.TaskArn, *task.LastStatus)
			taskDefinitions[*task.TaskDefinitionArn] = nil
		}

		for taskDef := range taskDefinitions {
			taskDefRes, err := ecsClient.DescribeTaskDefinition(ctx, &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String(taskDef),
			})
			if err != nil {
				return nil, err
			}
			taskDefinitions[taskDef] = taskDefRes.TaskDefinition
		}

		for _, task := range taskRes.Tasks {
			taskDef, ok := taskDefinitions[*task.TaskDefinitionArn]
			if !ok {
				return nil, fmt.Errorf("missing task definition %s", *task.TaskDefinitionArn)
			}

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

		}
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
