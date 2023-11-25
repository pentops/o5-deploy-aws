package awsinfra

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
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

func buildClientID(stackID *deployer_tpb.StackID) *string {
	str := fmt.Sprintf("%s-%s", stackID.DeploymentId, stackID.DeploymentPhase)
	return &str
}

func parseStackID(stackName, clientToken string) (*deployer_tpb.StackID, error) {
	stackID := &deployer_tpb.StackID{
		StackName: stackName,
	}

	parts := strings.SplitN(clientToken, "-", 6)
	fmt.Printf("%#v\n", parts)
	if len(parts) == 6 {
		stackID.DeploymentId = strings.Join(parts[0:5], "-")
		stackID.DeploymentPhase = parts[5]
	} else {
		return nil, fmt.Errorf("invalid client token %s", clientToken)
	}

	return stackID, nil
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

	dpr, err := newDeferredParameterResolver(cf.Clients, listenerARN, desiredCount)
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

func (cf *CFClient) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) error {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	if msg.ExtraResources != nil {
		if err := upsertExtraResources(ctx, clients, msg.ExtraResources); err != nil {
			return err
		}
	}

	parameters, err := cf.resolveParameters(ctx, nil, msg.Parameters, msg.DesiredCount)
	if err != nil {
		return err
	}

	_, err = clients.CloudFormation.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
		TemplateURL:        aws.String(msg.TemplateUrl),
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

func (cf *CFClient) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) error {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	if msg.ExtraResources != nil {
		if err := upsertExtraResources(ctx, clients, msg.ExtraResources); err != nil {
			return err
		}
	}

	current, err := cf.getOneStack(ctx, msg.StackId.StackName)
	if err != nil {
		return err
	}

	// TODO: Re-use assigned priorities for routes by using the previous input
	parameters, err := cf.resolveParameters(ctx, current.Parameters, msg.Parameters, msg.DesiredCount)
	if err != nil {
		return err
	}

	_, err = clients.CloudFormation.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
		TemplateURL:        aws.String(msg.TemplateUrl),
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

func (cf *CFClient) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) error {

	current, err := cf.getOneStack(ctx, msg.StackId.StackName)
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
		StackName:           aws.String(msg.StackId.StackName),
		ClientRequestToken:  buildClientID(msg.StackId),
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

func (cf *CFClient) CancelStackUpdate(ctx context.Context, msg *deployer_tpb.CancelStackUpdateMessage) error {
	client, err := cf.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.CancelUpdateStack(ctx, &cloudformation.CancelUpdateStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
	})
	return err
}

func (cf *CFClient) DeleteStack(ctx context.Context, msg *deployer_tpb.DeleteStackMessage) error {
	client, err := cf.getClient(ctx)
	if err != nil {
		return err
	}

	_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
	})
	return err
}

func upsertExtraResources(ctx context.Context, clients *DeployerClients, evt *deployer_tpb.ExtraResources) error {
	snsClient := clients.SNS

	for _, topic := range evt.SnsTopics {
		_, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: aws.String(topic),
		})
		if err != nil {
			return fmt.Errorf("creating sns topic %s: %w", topic, err)
		}
	}
	return nil
}

type StackArgs struct {
	Template   *deployer_pb.DeploymentState
	Parameters []types.Parameter
}

func isNoUpdatesError(err error) bool {
	var opError smithy.APIError
	if !errors.As(err, &opError) {
		return false
	}

	return opError.ErrorCode() == "ValidationError" && opError.ErrorMessage() == "No updates are to be performed."
}

func (cf *LocalRunner) pollStack(
	ctx context.Context,
	lastStatus types.StackStatus,
	stackID *deployer_tpb.StackID,
) (*deployer_tpb.StackStatusChangedMessage, error) {

	stackName := stackID.StackName

	ctx = log.WithFields(ctx, map[string]interface{}{
		"stackName": stackName,
	})

	log.Debug(ctx, "PollStack Begin")

	for {
		remoteStack, err := cf.getOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}
		if remoteStack == nil {
			return nil, fmt.Errorf("missing stack %s", stackName)
		}

		summary, err := summarizeStackStatus(remoteStack)
		if err != nil {
			return nil, err
		}

		if lastStatus == remoteStack.StackStatus {

			log.WithFields(ctx, map[string]interface{}{
				"lifecycle":   summary.SummaryType.ShortString(),
				"stackStatus": remoteStack.StackStatus,
			}).Debug("PollStack No Change")
			time.Sleep(1 * time.Second)
			continue
		}

		lastStatus = remoteStack.StackStatus

		log.WithFields(ctx, map[string]interface{}{
			"lifecycle":   summary.SummaryType.ShortString(),
			"stackStatus": remoteStack.StackStatus,
		}).Debug("PollStack Result")

		if !summary.Stable {
			continue
		}

		return &deployer_tpb.StackStatusChangedMessage{
			StackId:   stackID,
			Status:    string(remoteStack.StackStatus),
			Outputs:   summary.Outputs,
			Lifecycle: summary.SummaryType,
		}, nil
	}

}
