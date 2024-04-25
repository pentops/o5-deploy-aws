package awsinfra

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
	"github.com/aws/smithy-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/cf/app"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
)

type CFClient struct {
	Clients      ClientBuilder
	CallbackARNs []string
}

func (cf *CFClient) getClient(ctx context.Context) (CloudFormationAPI, error) {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}
	return clients.CloudFormation, nil
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
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("getClient: %w", err)
	}

	res, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
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

func (cf *CFClient) resolveParameters(ctx context.Context, lastInput []types.Parameter, input []*deployer_pb.CloudFormationStackParameter, desiredCount int32) ([]types.Parameter, error) {
	parameters := make([]types.Parameter, len(input))

	var listenerARN string
	for _, param := range input {
		if param.Name == app.ListenerARNParameter {
			listenerARN = param.GetValue()
			break
		}
	}

	dpr, err := NewDeferredParameterResolver(cf.Clients, listenerARN, desiredCount)
	if err != nil {
		return nil, err
	}

	for idx, param := range input {
		switch sourceType := param.Source.(type) {
		case *deployer_pb.CloudFormationStackParameter_Value:
			parameters[idx] = types.Parameter{
				ParameterKey:   aws.String(param.Name),
				ParameterValue: aws.String(sourceType.Value),
			}

		case *deployer_pb.CloudFormationStackParameter_Resolve:
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

func (cf *CFClient) CreateNewStack(ctx context.Context, reqToken string, msg *deployer_tpb.CreateNewStackMessage) error {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	if err := upsertExtraResources(ctx, clients, msg.Spec); err != nil {
		return err
	}

	parameters, err := cf.resolveParameters(ctx, nil, msg.Spec.Parameters, msg.Spec.DesiredCount)
	if err != nil {
		return err
	}

	_, err = clients.CloudFormation.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:          aws.String(msg.Spec.StackName),
		ClientRequestToken: aws.String(reqToken),
		TemplateURL:        aws.String(msg.Spec.TemplateUrl),
		Parameters:         parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
		NotificationARNs: cf.CallbackARNs,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cf *CFClient) UpdateStack(ctx context.Context, reqToken string, msg *deployer_tpb.UpdateStackMessage) error {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	if err := upsertExtraResources(ctx, clients, msg.Spec); err != nil {
		return err
	}

	current, err := cf.getOneStack(ctx, msg.Spec.StackName)
	if err != nil {
		return err
	}

	// TODO: Re-use assigned priorities for routes by using the previous input
	parameters, err := cf.resolveParameters(ctx, current.Parameters, msg.Spec.Parameters, msg.Spec.DesiredCount)
	if err != nil {
		return err
	}

	_, err = clients.CloudFormation.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:          aws.String(msg.Spec.StackName),
		ClientRequestToken: aws.String(reqToken),
		TemplateURL:        aws.String(msg.Spec.TemplateUrl),
		Parameters:         parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
		NotificationARNs: cf.CallbackARNs,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cf *CFClient) ScaleStack(ctx context.Context, reqToken string, msg *deployer_tpb.ScaleStackMessage) error {

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

	client, err := cf.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:           aws.String(msg.StackName),
		ClientRequestToken:  aws.String(reqToken),
		UsePreviousTemplate: aws.Bool(true),
		Parameters:          parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
		NotificationARNs: cf.CallbackARNs,
	})
	if err != nil {
		return err
	}

	return nil
}

func (cf *CFClient) CancelStackUpdate(ctx context.Context, reqToken string, msg *deployer_tpb.CancelStackUpdateMessage) error {
	client, err := cf.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.CancelUpdateStack(ctx, &cloudformation.CancelUpdateStackInput{
		StackName:          aws.String(msg.StackName),
		ClientRequestToken: aws.String(reqToken),
	})
	return err
}

func (cf *CFClient) DeleteStack(ctx context.Context, reqToken string, msg *deployer_tpb.DeleteStackMessage) error {
	client, err := cf.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName:          aws.String(msg.StackName),
		ClientRequestToken: aws.String(reqToken),
	})
	return err
}

func upsertExtraResources(ctx context.Context, clients *DeployerClients, evt *deployer_pb.CFStackInput) error {
	snsClient := clients.SNS

	for _, topic := range evt.SnsTopics {
		log.WithField(ctx, "topic", topic).Debug("Creating SNS topic")
		_, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
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
	Template   *deployer_pb.DeploymentState
	Parameters []types.Parameter
}

func IsNoUpdatesError(err error) bool {
	var opError smithy.APIError
	if !errors.As(err, &opError) {
		return false
	}

	return opError.ErrorCode() == "ValidationError" && opError.ErrorMessage() == "No updates are to be performed."
}
