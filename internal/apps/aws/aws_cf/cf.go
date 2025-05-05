package aws_cf

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder"
	"github.com/pentops/o5-deploy-aws/internal/apps/aws/awsapi"
)

type CFClient struct {
	cfClient             awsapi.CloudFormationAPI
	elbClient            awsapi.ELBV2API
	snsClient            awsapi.SNSAPI
	s3Client             awsapi.S3API
	secretsManagerClient awsapi.SecretsManagerAPI
	region               string
	accountID            string
}

func NewCFAdapter(clients *awsapi.DeployerClients) *CFClient {
	return &CFClient{
		cfClient:             clients.CloudFormation,
		elbClient:            clients.ELB,
		snsClient:            clients.SNS,
		s3Client:             clients.S3,
		secretsManagerClient: clients.SecretsManager,
		region:               clients.Region,
		accountID:            clients.AccountID,
	}
}

func templateURL(tpl *awsdeployer_pb.S3Template) string {
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", tpl.Region, tpl.Bucket, tpl.Key)
}

func (cf *CFClient) GetOneStack(ctx context.Context, stackName string) (*StackStatus, error) {
	stack, err := cf.getOneStack(ctx, stackName)
	if err != nil {
		return nil, err
	}

	summary, err := summarizeStackStatus(stack)
	if err != nil {
		return nil, err
	}
	return &summary, nil

}

func (cf *CFClient) getOneStack(ctx context.Context, stackName string) (*types.Stack, error) {

	res, err := cf.cfClient.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		var awsErr smithy.APIError
		if errors.As(err, &awsErr) {
			if awsErr.ErrorCode() == "ValidationError" && awsErr.ErrorMessage() == fmt.Sprintf("Stack with id %s does not exist", stackName) {
				return nil, nil
			}
		}

		return nil, fmt.Errorf("DescribeStacks unknown error: %T %w", err, err)
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

func (cf *CFClient) resolveParameters(ctx context.Context, lastInput []types.Parameter, input []*awsdeployer_pb.CloudFormationStackParameter, desiredCount int32) ([]types.Parameter, error) {
	parameters := make([]types.Parameter, len(input))

	var listenerARN string
	for _, param := range input {
		if param.Name == appbuilder.ListenerARNParameter {
			listenerARN = param.GetValue()
			break
		}
	}

	dpr, err := NewDeferredParameterResolver(cf.elbClient, listenerARN, desiredCount)
	if err != nil {
		return nil, err
	}

	for idx, param := range input {
		switch sourceType := param.Source.(type) {
		case *awsdeployer_pb.CloudFormationStackParameter_Value:
			parameters[idx] = types.Parameter{
				ParameterKey:   aws.String(param.Name),
				ParameterValue: aws.String(sourceType.Value),
			}

		case *awsdeployer_pb.CloudFormationStackParameter_Resolve:
			var previousParameter *types.Parameter
			for _, p := range lastInput {
				p := p
				if *p.ParameterKey == param.Name {
					previousParameter = &p
					break
				}
			}

			resolvedValue, err := dpr.Resolve(ctx, sourceType.Resolve, previousParameter)
			if err != nil {
				return nil, err
			}
			parameters[idx] = types.Parameter{
				ParameterKey:   aws.String(param.Name),
				ParameterValue: aws.String(resolvedValue),
			}

		}

	}

	return parameters, nil
}

type LogEvent struct {
	Timestamp time.Time
	Resource  string
	Status    string
	Detail    string
	IsFailure bool
}

func (cf *CFClient) Logs(ctx context.Context, stackName string) ([]LogEvent, error) {
	res, err := cf.cfClient.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}

	events := make([]LogEvent, 0, len(res.StackEvents))
	for _, src := range res.StackEvents {
		evt := LogEvent{
			Timestamp: *src.Timestamp,
		}
		if src.LogicalResourceId != nil {
			evt.Resource = *src.LogicalResourceId
		} else if src.PhysicalResourceId != nil {
			evt.Resource = *src.PhysicalResourceId
		}

		if src.ResourceStatus != "" {
			evt.Status = string(src.ResourceStatus)
			if src.ResourceStatus == types.ResourceStatusCreateFailed || src.ResourceStatus == types.ResourceStatusDeleteFailed || src.ResourceStatus == types.ResourceStatusUpdateFailed {
				evt.IsFailure = true
			}
		}

		if src.ResourceStatusReason != nil {
			evt.Detail = *src.ResourceStatusReason
		}

		events = append(events, evt)
	}

	slices.Reverse(events)

	return events, nil
}

func (cf *CFClient) CreateNewStack(ctx context.Context, reqToken string, msg *awsinfra_tpb.CreateNewStackMessage) error {

	if err := cf.upsertExtraResources(ctx, msg.Spec); err != nil {
		return err
	}

	parameters, err := cf.resolveParameters(ctx, nil, msg.Spec.Parameters, msg.Spec.DesiredCount)
	if err != nil {
		return err
	}

	input := &cloudformation.CreateStackInput{
		StackName:          aws.String(msg.Spec.StackName),
		ClientRequestToken: aws.String(reqToken),
		Parameters:         parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	}

	switch tpl := msg.Spec.Template.(type) {
	case *awsdeployer_pb.CFStackInput_TemplateBody:
		input.TemplateBody = aws.String(tpl.TemplateBody)
	case *awsdeployer_pb.CFStackInput_S3Template:
		input.TemplateURL = aws.String(templateURL(tpl.S3Template))
	case *awsdeployer_pb.CFStackInput_EmptyStack:
		input.TemplateBody = aws.String(EmptyTemplate())
	default:
		return fmt.Errorf("unknown template type: %T", tpl)
	}
	_, err = cf.cfClient.CreateStack(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

func (cf *CFClient) UpdateStack(ctx context.Context, reqToken string, msg *awsinfra_tpb.UpdateStackMessage) error {

	if err := cf.upsertExtraResources(ctx, msg.Spec); err != nil {
		return err
	}

	current, err := cf.getOneStack(ctx, msg.Spec.StackName)
	if err != nil {
		return err
	}

	params := []types.Parameter{}
	if current != nil {
		params = current.Parameters
	}

	// TODO: Re-use assigned priorities for routes by using the previous input
	parameters, err := cf.resolveParameters(ctx, params, msg.Spec.Parameters, msg.Spec.DesiredCount)
	if err != nil {
		return err
	}
	input := &cloudformation.UpdateStackInput{
		StackName:          aws.String(msg.Spec.StackName),
		ClientRequestToken: aws.String(reqToken),
		Parameters:         parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	}
	switch tpl := msg.Spec.Template.(type) {
	case *awsdeployer_pb.CFStackInput_TemplateBody:
		input.TemplateBody = aws.String(tpl.TemplateBody)
	case *awsdeployer_pb.CFStackInput_S3Template:
		input.TemplateURL = aws.String(templateURL(tpl.S3Template))
	case *awsdeployer_pb.CFStackInput_EmptyStack:
		input.TemplateBody = aws.String(EmptyTemplate())
	default:
		return fmt.Errorf("unknown template type: %T", tpl)
	}

	_, err = cf.cfClient.UpdateStack(ctx, input)

	if err != nil {
		return err
	}

	return nil
}

func (cf *CFClient) CreateChangeSet(ctx context.Context, reqToken string, msg *awsinfra_tpb.CreateChangeSetMessage) error {

	if err := cf.upsertExtraResources(ctx, msg.Spec); err != nil {
		return err
	}

	var currentParameters []types.Parameter
	current, err := cf.getOneStack(ctx, msg.Spec.StackName)
	if err != nil {
		return err
	}

	if current != nil {
		currentParameters = current.Parameters
	}

	// TODO: Re-use assigned priorities for routes by using the previous input
	parameters, err := cf.resolveParameters(ctx, currentParameters, msg.Spec.Parameters, msg.Spec.DesiredCount)
	if err != nil {
		return err
	}

	changeSetID := fmt.Sprintf("%s-%s", msg.Spec.StackName, reqToken)

	input := &cloudformation.CreateChangeSetInput{
		StackName:     aws.String(msg.Spec.StackName),
		ChangeSetName: aws.String(changeSetID),
		Parameters:    parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	}

	if msg.ImportResources {
		templateBody := ""
		switch tpl := msg.Spec.Template.(type) {
		case *awsdeployer_pb.CFStackInput_TemplateBody:
			templateBody = tpl.TemplateBody
		case *awsdeployer_pb.CFStackInput_S3Template:
			template, err := cf.downloadCFTemplate(ctx, tpl.S3Template)
			if err != nil {
				return err
			}
			templateBody = template
		case *awsdeployer_pb.CFStackInput_EmptyStack:
			return fmt.Errorf("cannot import resources with empty stack")
		default:
			return fmt.Errorf("unknown template type: %T", tpl)
		}

		importResult, err := cf.ImportResources(ctx, templateBody, parameters)
		if err != nil {
			return err
		}

		if len(importResult.Imports) == 0 {
			return fmt.Errorf("no resources to import")
		}

		input.ResourcesToImport = importResult.Imports
		input.TemplateBody = aws.String(importResult.NewTemplate)
		//input.ImportExistingResources = aws.Bool(true)
		input.ChangeSetType = types.ChangeSetTypeImport

	} else {
		switch tpl := msg.Spec.Template.(type) {
		case *awsdeployer_pb.CFStackInput_TemplateBody:
			input.TemplateBody = aws.String(tpl.TemplateBody)
		case *awsdeployer_pb.CFStackInput_S3Template:
			input.TemplateURL = aws.String(templateURL(tpl.S3Template))
		case *awsdeployer_pb.CFStackInput_EmptyStack:
			input.TemplateBody = aws.String(EmptyTemplate())
		default:
			return fmt.Errorf("unknown template type: %T", tpl)
		}
	}

	_, err = cf.cfClient.CreateChangeSet(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

type ChangeSetStatus struct {
	Status    string
	Lifecycle awsdeployer_pb.CFChangesetLifecycle
}

func (cf *CFClient) GetChangeSet(ctx context.Context, stackName, changeSetName string) (*ChangeSetStatus, error) {
	res, err := cf.cfClient.DescribeChangeSet(ctx, &cloudformation.DescribeChangeSetInput{
		ChangeSetName: aws.String(changeSetName),
		StackName:     aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}

	lifecycle, ok := map[types.ChangeSetStatus]awsdeployer_pb.CFChangesetLifecycle{
		types.ChangeSetStatusCreatePending:    awsdeployer_pb.CFChangesetLifecycle_UNSPECIFIED,
		types.ChangeSetStatusCreateInProgress: awsdeployer_pb.CFChangesetLifecycle_UNSPECIFIED,
		types.ChangeSetStatusCreateComplete:   awsdeployer_pb.CFChangesetLifecycle_AVAILABLE,
		types.ChangeSetStatusDeletePending:    awsdeployer_pb.CFChangesetLifecycle_UNSPECIFIED,
		types.ChangeSetStatusDeleteInProgress: awsdeployer_pb.CFChangesetLifecycle_UNSPECIFIED,
		types.ChangeSetStatusDeleteComplete:   awsdeployer_pb.CFChangesetLifecycle_TERMINAL,
		types.ChangeSetStatusDeleteFailed:     awsdeployer_pb.CFChangesetLifecycle_TERMINAL,
		types.ChangeSetStatusFailed:           awsdeployer_pb.CFChangesetLifecycle_TERMINAL,
	}[res.Status]
	if !ok {
		return nil, fmt.Errorf("unknown changeset status: %s", res.Status)
	}

	return &ChangeSetStatus{
		Status:    string(res.Status),
		Lifecycle: lifecycle,
	}, nil
}

func (cf *CFClient) ScaleStack(ctx context.Context, reqToken string, msg *awsinfra_tpb.ScaleStackMessage) error {

	current, err := cf.getOneStack(ctx, msg.StackName)
	if err != nil {
		return err
	}

	parameters := make([]types.Parameter, len(current.Parameters))
	for idx, param := range current.Parameters {
		if strings.HasPrefix(*param.ParameterKey, "DesiredCount") {
			parameters[idx] = types.Parameter{
				ParameterKey:   param.ParameterKey,
				ParameterValue: aws.String(fmt.Sprintf("%d", msg.DesiredCount)),
			}

		} else {
			parameters[idx] = types.Parameter{
				UsePreviousValue: aws.Bool(true),
				ParameterKey:     param.ParameterKey,
			}
		}

	}

	_, err = cf.cfClient.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:           aws.String(msg.StackName),
		ClientRequestToken:  aws.String(reqToken),
		UsePreviousTemplate: aws.Bool(true),
		Parameters:          parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (cf *CFClient) CancelStackUpdate(ctx context.Context, reqToken string, msg *awsinfra_tpb.CancelStackUpdateMessage) error {
	_, err := cf.cfClient.CancelUpdateStack(ctx, &cloudformation.CancelUpdateStackInput{
		StackName:          aws.String(msg.StackName),
		ClientRequestToken: aws.String(reqToken),
	})
	return err
}

func (cf *CFClient) DeleteStack(ctx context.Context, reqToken string, stackName string) error {
	_, err := cf.cfClient.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName:          aws.String(stackName),
		ClientRequestToken: aws.String(reqToken),
	})
	return err
}

func (cf *CFClient) upsertExtraResources(ctx context.Context, evt *awsdeployer_pb.CFStackInput) error {

	for _, topic := range evt.SnsTopics {
		log.WithField(ctx, "topic", topic).Debug("Creating SNS topic")
		_, err := cf.snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: aws.String(topic),
			Tags: []snstypes.Tag{
				{
					Key:   aws.String("created-by"),
					Value: aws.String("o5-deployer"),
				}},
		})
		if err != nil {
			errString := err.Error()
			// InvalidParameter: Invalid parameter: Tags Reason: Topic already exists with different tags

			if !strings.Contains(errString, "already exists") {
				return fmt.Errorf("creating sns topic %s: %w", topic, err)
			}
		}
	}
	return nil
}

type StackArgs struct {
	Template   *awsdeployer_pb.DeploymentState
	Parameters []types.Parameter
}

func IsNoUpdatesError(err error) bool {
	var opError smithy.APIError
	if !errors.As(err, &opError) {
		return false
	}

	return opError.ErrorCode() == "ValidationError" && opError.ErrorMessage() == "No updates are to be performed."
}
