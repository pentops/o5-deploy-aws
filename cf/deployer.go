package cf

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	protovalidate "github.com/bufbuild/protovalidate-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type Deployer struct {
	DeployerClients
	Environment *environment_pb.Environment
	AWS         *environment_pb.AWS
	VersionTag  string

	takenPriorities map[int]bool
}

type CloudFormationAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
	UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
}

type SNSAPI interface {
}

type ELBV2API interface {
	DescribeRules(ctx context.Context, params *elbv2.DescribeRulesInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeRulesOutput, error)
}

type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
}

type ECSAPI interface {
	RunTask(ctx context.Context, params *ecs.RunTaskInput, optFns ...func(*ecs.Options)) (*ecs.RunTaskOutput, error)

	// used by the TasksStoppedWaiter
	ecs.DescribeTasksAPIClient
}

type DeployerClients struct {
	CloudFormation CloudFormationAPI
	SNS            SNSAPI
	ELB            ELBV2API
	SecretsManager SecretsManagerAPI
	ECS            ECSAPI
}

func NewDeployerClientsFromConfig(awsConfig aws.Config) DeployerClients {
	return DeployerClients{
		CloudFormation: cloudformation.NewFromConfig(awsConfig),
		SNS:            sns.NewFromConfig(awsConfig),
		ELB:            elasticloadbalancingv2.NewFromConfig(awsConfig),
		SecretsManager: secretsmanager.NewFromConfig(awsConfig),
		ECS:            ecs.NewFromConfig(awsConfig),
	}
}

func NewDeployer(environment *environment_pb.Environment, version string, clients DeployerClients) (*Deployer, error) {

	validator, err := protovalidate.New()
	if err != nil {
		panic(err)
	}

	if err := validator.Validate(environment); err != nil {
		return nil, err
	}

	awsTarget := environment.GetAws()
	if awsTarget == nil {
		return nil, errors.New("AWS Deployer requires the type of environment provider to be AWS")
	}

	return &Deployer{
		Environment:     environment,
		AWS:             awsTarget,
		DeployerClients: clients,
		VersionTag:      version,
	}, nil
}

func (d *Deployer) Deploy(ctx context.Context, stackName string, template *Template) error {
	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackName":   stackName,
		"environment": d.Environment.FullName,
	})

	remoteStack, err := d.waitForStack(ctx, stackName)
	if err != nil {
		return err
	}

	if remoteStack == nil {
		return d.createNewDeployment(ctx, stackName, template)
	}

	return d.updateDeployment(ctx, stackName, template, remoteStack.Parameters)
}

type stackParameters struct {
	name               string
	template           *Template
	scale              int
	previousParameters []types.Parameter
}

func (d *Deployer) createNewDeployment(ctx context.Context, stackName string, template *Template) error {
	// Create, scale 0
	log.Info(ctx, "Create with scale 0")
	if err := d.createCloudformationStack(ctx, stackParameters{
		name:     stackName,
		template: template,
		scale:    0,
	}); err != nil {
		return err
	}

	if err := d.WaitForSuccess(ctx, stackName, "Stack Create"); err != nil {
		return err
	}

	// Migrate
	log.Info(ctx, "Migrate Database")
	if err := d.migrateData(ctx, stackName, template, true); err != nil {
		return err
	}

	// Scale Up
	log.Info(ctx, "Scale Up")
	if err := d.updateCloudformationStack(ctx, stackParameters{
		name:     stackName,
		template: template,
		scale:    1,
	}); err != nil {
		return err
	}

	if err := d.WaitForSuccess(ctx, stackName, "Scale Up"); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) updateDeployment(ctx context.Context, stackName string, template *Template, previous []types.Parameter) error {
	// Scale Down
	log.Info(ctx, "Scale Down")
	if err := d.setScale(ctx, stackParameters{
		previousParameters: previous,
		name:               stackName,
		scale:              0,
	}); err != nil {
		return err
	}

	if err := d.WaitForSuccess(ctx, stackName, "Scale Down"); err != nil {
		return err
	}

	// Update, Keep Scale 0
	log.Info(ctx, "Update Pre Migrate")
	if err := d.updateCloudformationStack(ctx, stackParameters{
		previousParameters: previous,
		name:               stackName,
		template:           template,
		scale:              0,
	}); err != nil {
		return err
	}

	if err := d.WaitForSuccess(ctx, stackName, "Update"); err != nil {
		return err
	}

	// Migrate
	log.Info(ctx, "Data Migrate")
	if err := d.migrateData(ctx, stackName, template, false); err != nil {
		return err
	}

	// Scale Up
	log.Info(ctx, "Scale Up")
	if err := d.updateCloudformationStack(ctx, stackParameters{
		previousParameters: previous,
		name:               stackName,
		template:           template,
		scale:              1,
	}); err != nil {
		return err
	}

	if err := d.WaitForSuccess(ctx, stackName, "Scale Up"); err != nil {
		return err
	}

	return nil
}

func (d *Deployer) getOneStack(ctx context.Context, stackName string) (*types.Stack, error) {

	res, err := d.CloudFormation.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) {
			if awsErr.ErrorCode() == "ValidationError" && awsErr.ErrorMessage() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
				return nil, nil
			}
		}

		return nil, err
	}

	if len(res.Stacks) == 0 {
		return nil, nil
	}

	if len(res.Stacks) > 1 {
		return nil, fmt.Errorf("found more than one stack with name %s", stackName)
	}

	remoteStack := res.Stacks[0]
	return &remoteStack, nil

}

func (d *Deployer) waitForStack(ctx context.Context, stackName string) (*types.Stack, error) {
	for {
		remoteStack, err := d.getOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		if remoteStack == nil {
			return nil, nil
		}

		log.WithFields(ctx, map[string]interface{}{
			"currentStatus": remoteStack.StackStatus,
		}).Debug("Waiting for stack to be stable")

		switch remoteStack.StackStatus {
		case types.StackStatusCreateComplete, // Initial
			types.StackStatusUpdateComplete,         // Update
			types.StackStatusUpdateRollbackComplete: // Ready to try again.
			log.WithFields(ctx, map[string]interface{}{
				"currentStatus": remoteStack.StackStatus,
			}).Debug("Stack is stable")
			return remoteStack, nil

		case types.StackStatusRollbackComplete:
			log.WithFields(ctx, map[string]interface{}{
				"currentStatus": remoteStack.StackStatus,
			}).Debug("Deleting Rolled Back Stack")
			if err := d.deleteStack(ctx, stackName); err != nil {
				return nil, err
			}

			// then continue waiting

		case types.StackStatusCreateFailed, types.StackStatusRollbackFailed, types.StackStatusDeleteComplete, types.StackStatusDeleteFailed:
			return nil, fmt.Errorf("stack %s is in status %s", stackName, remoteStack.StackStatus)

		default:
		}

		time.Sleep(time.Second)
	}
}

var stackStatusProgress = []types.StackStatus{
	types.StackStatusCreateInProgress,
	types.StackStatusUpdateInProgress,
	types.StackStatusReviewInProgress,
	types.StackStatusImportInProgress,
	types.StackStatusDeleteInProgress,
	types.StackStatusUpdateCompleteCleanupInProgress,
}

var stackStatusRollingBack = []types.StackStatus{
	types.StackStatusRollbackInProgress,
	types.StackStatusRollbackComplete,
	types.StackStatusUpdateRollbackFailed,
	types.StackStatusUpdateRollbackCompleteCleanupInProgress,
}

var stackStatusRolledBack = []types.StackStatus{
	types.StackStatusUpdateRollbackComplete,
}

var stackStatusComplete = []types.StackStatus{
	types.StackStatusCreateComplete,
	types.StackStatusImportComplete,
	types.StackStatusUpdateComplete,
	types.StackStatusDeleteComplete,
}

var stackStatusesTerminal = []types.StackStatus{
	types.StackStatusCreateFailed,
	types.StackStatusRollbackFailed,
	types.StackStatusDeleteFailed,
	types.StackStatusUpdateFailed,
	types.StackStatusUpdateRollbackInProgress,
	types.StackStatusImportRollbackInProgress,
	types.StackStatusImportRollbackFailed,
	types.StackStatusImportRollbackComplete,
}

func (d *Deployer) WaitForSuccess(ctx context.Context, stackName string, phaseLabel string) error {
	for {
		remoteStack, err := d.getOneStack(ctx, stackName)
		if err != nil {
			return err
		}
		if remoteStack == nil {
			return fmt.Errorf("missing stack %s", stackName)
		}

		log.WithFields(ctx, map[string]interface{}{
			"currentStatus": remoteStack.StackStatus,
			"phase":         phaseLabel,
		}).Debug("Waiting for stack to be stable")

		for _, status := range stackStatusesTerminal {
			if remoteStack.StackStatus == status {
				return fmt.Errorf("stack %s, phase %s, is in terminal status %s", stackName, phaseLabel, remoteStack.StackStatus)
			}
		}

		for _, status := range stackStatusComplete {
			if remoteStack.StackStatus == status {
				log.WithFields(ctx, map[string]interface{}{
					"currentStatus": remoteStack.StackStatus,
					"phase":         phaseLabel,
				}).Debug("Stack is complete")
				return nil
			}
		}

		for _, status := range stackStatusRolledBack {
			if remoteStack.StackStatus == status {
				log.WithFields(ctx, map[string]interface{}{
					"currentStatus": remoteStack.StackStatus,
					"phase":         phaseLabel,
				}).Debug("Stack is rolled back")
				return nil
			}
		}

		for _, status := range stackStatusRollingBack {
			if remoteStack.StackStatus == status {
				return fmt.Errorf("stack %s, phase %s, is in status %s", stackName, phaseLabel, remoteStack.StackStatus)
			}
		}

		time.Sleep(time.Second)

	}
}

func (d *Deployer) deleteStack(ctx context.Context, name string) error {
	_, err := d.CloudFormation.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(name),
	})

	return err
}

func (d *Deployer) applyInitialParameters(ctx context.Context, stack stackParameters) ([]types.Parameter, error) {

	mappedPreviousParameters := make(map[string]string, len(stack.previousParameters))
	for _, param := range stack.previousParameters {
		mappedPreviousParameters[*param.ParameterKey] = *param.ParameterValue
	}

	parameters := make([]types.Parameter, 0, len(stack.template.template.Parameters))

	for key, param := range stack.template.template.Parameters {
		if param.Default != nil {
			continue
		}
		parameter := types.Parameter{
			ParameterKey: aws.String(key),
		}
		switch {
		case key == ListenerARNParameter:
			parameter.ParameterValue = aws.String(d.AWS.ListenerArn)

		case key == ECSClusterParameter:
			parameter.ParameterValue = aws.String(d.AWS.EcsClusterName)

		case key == ECSRepoParameter:
			parameter.ParameterValue = aws.String(d.AWS.EcsRepo)

		case key == ECSTaskExecutionRoleParameter:
			parameter.ParameterValue = aws.String(d.AWS.EcsTaskExecutionRole)

		case key == VersionTagParameter:
			parameter.ParameterValue = aws.String(d.VersionTag)

		case key == EnvNameParameter:
			parameter.ParameterValue = aws.String(d.Environment.FullName)

		case key == VPCParameter:
			parameter.ParameterValue = aws.String(d.AWS.VpcId)

		case strings.HasPrefix(key, "ListenerRulePriority"):

			existingPriority, ok := mappedPreviousParameters[key]
			if ok {
				parameter.ParameterValue = aws.String(existingPriority)
			} else {
				priority, err := d.nextAvailableListenerRulePriority(ctx)
				if err != nil {
					return nil, err
				}

				parameter.ParameterValue = aws.String(fmt.Sprintf("%d", priority))
			}

		case strings.HasPrefix(key, "DesiredCount"):
			parameter.ParameterValue = Stringf("%d", stack.scale)

		default:
			return nil, fmt.Errorf("unknown parameter %s", key)
		}
		parameters = append(parameters, parameter)
	}

	return parameters, nil
}

func (d *Deployer) nextAvailableListenerRulePriority(ctx context.Context) (int, error) {
	if d.takenPriorities == nil {
		d.takenPriorities = make(map[int]bool)
		var marker *string
		for {
			rules, err := d.ELB.DescribeRules(ctx, &elbv2.DescribeRulesInput{
				ListenerArn: aws.String(d.AWS.ListenerArn),
				Marker:      marker,
			})
			if err != nil {
				return 0, err
			}

			for _, rule := range rules.Rules {
				if *rule.Priority == "default" {
					continue
				}
				priority, err := strconv.Atoi(*rule.Priority)
				if err != nil {
					return 0, err
				}

				d.takenPriorities[priority] = true
			}

			marker = rules.NextMarker
			if marker == nil {
				break
			}
		}
	}

	for i := 1000; i < 2000; i++ {
		if _, ok := d.takenPriorities[i]; !ok {
			d.takenPriorities[i] = true
			return i, nil
		}
	}

	return 0, errors.New("no available listener rule priorities")
}

func (d *Deployer) createCloudformationStack(ctx context.Context, stack stackParameters) error {

	parameters, err := d.applyInitialParameters(ctx, stack)
	if err != nil {
		return err
	}

	jsonStack, err := stack.template.Template().JSON()
	if err != nil {
		return err
	}

	_, err = d.CloudFormation.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    aws.String(stack.name),
		TemplateBody: aws.String(string(jsonStack)),
		Parameters:   parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *Deployer) updateCloudformationStack(ctx context.Context, stack stackParameters) error {

	parameters, err := d.applyInitialParameters(ctx, stack)
	if err != nil {
		return err
	}

	jsonStack, err := stack.template.Template().JSON()
	if err != nil {
		return err
	}
	_, err = d.CloudFormation.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:    aws.String(stack.name),
		TemplateBody: aws.String(string(jsonStack)),
		Parameters:   parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		if !isNoUpdatesError(err) {
			return fmt.Errorf("updateCFStack: %w", err)
		}
	}

	return nil
}

func (d *Deployer) setScale(ctx context.Context, stack stackParameters) error {
	parameters := make([]types.Parameter, len(stack.previousParameters))
	for idx, param := range stack.previousParameters {
		if strings.HasPrefix(*param.ParameterKey, "DesiredCount") {
			parameters[idx] = types.Parameter{
				ParameterKey:   param.ParameterKey,
				ParameterValue: aws.String(fmt.Sprintf("%d", stack.scale)),
			}

		} else {
			parameters[idx] = types.Parameter{
				UsePreviousValue: aws.Bool(true),
				ParameterKey:     param.ParameterKey,
			}
		}

	}

	_, err := d.DeployerClients.CloudFormation.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:           aws.String(stack.name),
		UsePreviousTemplate: aws.Bool(true),
		Parameters:          parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		if !isNoUpdatesError(err) {
			return fmt.Errorf("setScale: %w", err)
		}
	}

	return nil
}

func isNoUpdatesError(err error) bool {
	var opError smithy.APIError
	if !errors.As(err, &opError) {
		return false
	}

	return opError.ErrorCode() == "ValidationError" && opError.ErrorMessage() == "No updates are to be performed."
}
