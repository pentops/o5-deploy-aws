package awsinfra

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
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/smithy-go"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/outbox.pg.go/outbox"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

type MessageHandler interface {
	PublishEvent(context.Context, outbox.OutboxMessage) error
}

type AWSRunner struct {
	Clients        ClientBuilder
	CallbackARN    string
	messageHandler MessageHandler
	*deployer_tpb.UnimplementedAWSCommandTopicServer
	*messaging_tpb.UnimplementedRawMessageTopicServer
}

func NewRunner(clients ClientBuilder, messageHandler MessageHandler) *AWSRunner {
	return &AWSRunner{
		Clients:        clients,
		messageHandler: messageHandler,
	}
}

func (cf *AWSRunner) eventOut(ctx context.Context, msg outbox.OutboxMessage) error {
	fmt.Printf("eventOut: %s\n", protojson.Format(msg))
	return cf.messageHandler.PublishEvent(ctx, msg)
}

func (cf *AWSRunner) getClient(ctx context.Context) (CloudFormationAPI, error) {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}
	return clients.CloudFormation, nil
}

func (cf *AWSRunner) getOneStack(ctx context.Context, stackName string) (*types.Stack, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
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

func parseAWSRawMessage(raw []byte) (map[string]string, error) {

	lines := strings.Split(string(raw), "\n")
	fields := map[string]string{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line: %s", line)
		}
		s := parts[1]
		if strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
			s = s[1 : len(s)-1]
		}
		fields[parts[0]] = s
	}
	return fields, nil
}

func (cf *AWSRunner) Raw(ctx context.Context, msg *messaging_tpb.RawMessage) (*emptypb.Empty, error) {

	fields, err := parseAWSRawMessage(msg.Payload)
	if err != nil {
		return nil, err
	}

	resourceType, ok := fields["ResourceType"]
	if !ok {
		return nil, fmt.Errorf("missing ResourceType")
	}

	if resourceType != "AWS::CloudFormation::Stack" {
		return &emptypb.Empty{}, nil
	}

	stackName, ok := fields["StackName"]
	if !ok {
		return nil, fmt.Errorf("missing StackName")
	}

	clientToken, ok := fields["ClientRequestToken"]
	if !ok {
		return nil, fmt.Errorf("missing ClientRequestToken")
	}

	resourceStatus, ok := fields["ResourceStatus"]
	if !ok {
		return nil, fmt.Errorf("missing ResourceStatus")
	}

	lifecycle, err := stackLifecycle(types.StackStatus(resourceStatus))
	if err != nil {
		return nil, err
	}

	var outputs []*deployer_pb.KeyValue

	if lifecycle == deployer_pb.StackLifecycle_COMPLETE {

		stack, err := cf.getOneStack(ctx, stackName)
		if err != nil {
			return nil, err
		}

		if stack == nil {
			return nil, fmt.Errorf("missing stack %s", stackName)
		}

		outputs = mapOutputs(stack.Outputs)
	}

	stackID, err := parseStackID(stackName, clientToken)
	if err != nil {
		return nil, err
	}

	if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackId:   stackID,
		Status:    resourceStatus,
		Outputs:   outputs,
		Lifecycle: lifecycle,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
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

func (cf *AWSRunner) StabalizeStack(ctx context.Context, msg *deployer_tpb.StabalizeStackMessage) (*emptypb.Empty, error) {
	remoteStack, err := cf.getOneStack(ctx, msg.StackId.StackName)
	if err != nil {
		return nil, err
	}

	if remoteStack == nil {
		err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
			StackId:   msg.StackId,
			Status:    "MISSING",
			Lifecycle: deployer_pb.StackLifecycle_MISSING,
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}

	lifecycle, err := stackLifecycle(remoteStack.StackStatus)
	if err != nil {
		return nil, err
	}

	if remoteStack.StackStatus == types.StackStatusRollbackComplete {
		err := cf.eventOut(ctx, &deployer_tpb.DeleteStackMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}

	needsCancel := msg.CancelUpdate && remoteStack.StackStatus == types.StackStatusUpdateInProgress
	if needsCancel {
		err := cf.eventOut(ctx, &deployer_tpb.CancelStackUpdateMessage{
			StackId: msg.StackId,
		})
		if err != nil {
			return nil, err
		}
	}

	// Special cases for Stabalize only
	switch remoteStack.StackStatus {
	case types.StackStatusUpdateRollbackComplete:
		// When a previous attempt has failed, the stack will be in this state
		// In the Stabalize handler ONLY, this counts as a success, as the stack
		// is stable and ready for another attempt
		lifecycle = deployer_pb.StackLifecycle_COMPLETE

	case types.StackStatusRollbackInProgress:
		// Short exit: Further events will be emitted during the rollback
		return &emptypb.Empty{}, nil
	}

	log.WithFields(ctx, map[string]interface{}{
		"stackName":   msg.StackId.StackName,
		"lifecycle":   lifecycle.ShortString(),
		"stackStatus": remoteStack.StackStatus,
	}).Debug("StabalizeStack Result")

	err = cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackId:   msg.StackId,
		Status:    string(remoteStack.StackStatus),
		Lifecycle: lifecycle,
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

type deferredParameterResolver struct {
	clients         ClientBuilder
	desiredCount    int32
	listenerARN     string
	takenPriorities map[int]bool
}

func newDeferredParameterResolver(clients ClientBuilder, listenerARN string, desiredCount int32) (*deferredParameterResolver, error) {
	return &deferredParameterResolver{
		clients:      clients,
		listenerARN:  listenerARN,
		desiredCount: desiredCount,
	}, nil
}

func (dpr *deferredParameterResolver) getTakenPriorities(ctx context.Context) (map[int]bool, error) {
	if dpr.takenPriorities != nil {
		return dpr.takenPriorities, nil
	}

	if dpr.listenerARN == "" {
		return nil, fmt.Errorf("missing listener ARN - The Stack must have a parameter named %s to automatically resolve rule priorities", app.ListenerARNParameter)
	}

	clients, err := dpr.clients.Clients(ctx)
	if err != nil {
		return nil, err
	}

	takenPriorities := make(map[int]bool)
	var marker *string
	for {
		rules, err := clients.ELB.DescribeRules(ctx, &elbv2.DescribeRulesInput{
			ListenerArn: aws.String(dpr.listenerARN),
			Marker:      marker,
		})
		if err != nil {
			return nil, err
		}

		for _, rule := range rules.Rules {
			if *rule.Priority == "default" {
				continue
			}
			priority, err := strconv.Atoi(*rule.Priority)
			if err != nil {
				return nil, err
			}

			takenPriorities[priority] = true
		}

		marker = rules.NextMarker
		if marker == nil {
			break
		}
	}

	dpr.takenPriorities = takenPriorities
	return takenPriorities, nil
}

func (dpr *deferredParameterResolver) nextAvailableListenerRulePriority(ctx context.Context, group application_pb.RouteGroup) (int, error) {
	takenPriorities, err := dpr.getTakenPriorities(ctx)
	if err != nil {
		return 0, err
	}

	a, b, err := groupRangeForPriority(group)
	if err != nil {
		return 0, err
	}
	for i := a; i < b; i++ {
		if _, ok := takenPriorities[i]; !ok {
			takenPriorities[i] = true
			return i, nil
		}
	}

	return 0, errors.New("no available listener rule priorities")
}

func groupRangeForPriority(group application_pb.RouteGroup) (int, int, error) {
	switch group {
	case application_pb.RouteGroup_ROUTE_GROUP_NORMAL,
		application_pb.RouteGroup_ROUTE_GROUP_UNSPECIFIED:
		return 1000, 2000, nil
	case application_pb.RouteGroup_ROUTE_GROUP_FIRST:
		return 1, 1000, nil
	case application_pb.RouteGroup_ROUTE_GROUP_FALLBACK:
		return 2000, 3000, nil
	default:
		return 0, 0, fmt.Errorf("unknown route group %s", group)
	}
}

func validPreviousPriority(group application_pb.RouteGroup, previous *types.Parameter) (string, bool) {
	if previous == nil || previous.ParameterValue == nil {
		return "", false
	}

	previousPriority, err := strconv.Atoi(*previous.ParameterValue)
	if err != nil {
		return "", false // Who knows how, but let's just pretend it didn't happen
	}

	a, b, err := groupRangeForPriority(group)
	if err != nil {
		return "", false
	}

	if previousPriority >= a && previousPriority < b {
		return *previous.ParameterValue, true
	}

	return "", false
}

func (dpr *deferredParameterResolver) Resolve(ctx context.Context, input *deployer_pb.CloudFormationStackParameterType, previous *types.Parameter) (string, error) {
	switch pt := input.Type.(type) {
	case *deployer_pb.CloudFormationStackParameterType_RulePriority_:
		group := pt.RulePriority.RouteGroup
		previous, ok := validPreviousPriority(group, previous)
		if ok {
			return previous, nil
		}

		priority, err := dpr.nextAvailableListenerRulePriority(ctx, group)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d", priority), nil

	case *deployer_pb.CloudFormationStackParameterType_DesiredCount_:
		return fmt.Sprintf("%d", dpr.desiredCount), nil

	default:
		return "", fmt.Errorf("unknown parameter type: %T", pt)

	}

}

func (cf *AWSRunner) resolveParameters(ctx context.Context, lastInput []types.Parameter, input []*deployer_pb.CloudFormationStackParameter, desiredCount int32) ([]types.Parameter, error) {
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

func (cf *AWSRunner) CreateNewStack(ctx context.Context, msg *deployer_tpb.CreateNewStackMessage) (*emptypb.Empty, error) {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}

	if msg.ExtraResources != nil {
		if err := upsertExtraResources(ctx, clients, msg.ExtraResources); err != nil {
			return nil, err
		}
	}

	parameters, err := cf.resolveParameters(ctx, nil, msg.Parameters, msg.DesiredCount)
	if err != nil {
		return nil, err
	}

	_, err = clients.CloudFormation.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
		TemplateURL:        aws.String(msg.TemplateUrl),
		Parameters:         parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
		NotificationARNs: []string{cf.CallbackARN},
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (cf *AWSRunner) UpdateStack(ctx context.Context, msg *deployer_tpb.UpdateStackMessage) (*emptypb.Empty, error) {
	clients, err := cf.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}

	if msg.ExtraResources != nil {
		if err := upsertExtraResources(ctx, clients, msg.ExtraResources); err != nil {
			return nil, err
		}
	}

	current, err := cf.getOneStack(ctx, msg.StackId.StackName)
	if err != nil {
		return nil, err
	}

	// TODO: Re-use assigned priorities for routes by using the previous input
	parameters, err := cf.resolveParameters(ctx, current.Parameters, msg.Parameters, msg.DesiredCount)
	if err != nil {
		return nil, err
	}

	_, err = clients.CloudFormation.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
		TemplateURL:        aws.String(msg.TemplateUrl),
		Parameters:         parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
		NotificationARNs: []string{cf.CallbackARN},
	})
	if err != nil {
		if !isNoUpdatesError(err) {
			return nil, fmt.Errorf("updateCFStack: %w", err)
		}

		if err := cf.noUpdatesToBePerformed(ctx, msg.StackId); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

// sends a fake status update message back to the deployer so that it thinks
// something has happened and continues the event chain
func (cf *AWSRunner) noUpdatesToBePerformed(ctx context.Context, stackID *deployer_tpb.StackID) error {

	remoteStack, err := cf.getOneStack(ctx, stackID.StackName)
	if err != nil {
		return err
	}

	summary, err := summarizeStackStatus(remoteStack)
	if err != nil {
		return err
	}

	if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
		StackId:   stackID,
		Status:    "NO UPDATES TO BE PERFORMED",
		Outputs:   summary.Outputs,
		Lifecycle: deployer_pb.StackLifecycle_COMPLETE,
	}); err != nil {
		return err
	}

	return nil
}

func (cf *AWSRunner) ScaleStack(ctx context.Context, msg *deployer_tpb.ScaleStackMessage) (*emptypb.Empty, error) {

	current, err := cf.getOneStack(ctx, msg.StackId.StackName)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	log.Debug(ctx, "UpdateStack direct call")
	_, err = client.UpdateStack(ctx, &cloudformation.UpdateStackInput{
		StackName:           aws.String(msg.StackId.StackName),
		ClientRequestToken:  buildClientID(msg.StackId),
		UsePreviousTemplate: aws.Bool(true),
		Parameters:          parameters,
		Capabilities: []types.Capability{
			types.CapabilityCapabilityNamedIam,
		},
		NotificationARNs: []string{cf.CallbackARN},
	})
	if err != nil {
		if !isNoUpdatesError(err) {
			return nil, fmt.Errorf("setScale: %w", err)
		}
		if err := cf.noUpdatesToBePerformed(ctx, msg.StackId); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (cf *AWSRunner) CancelStackUpdate(ctx context.Context, msg *deployer_tpb.CancelStackUpdateMessage) (*emptypb.Empty, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.CancelUpdateStack(ctx, &cloudformation.CancelUpdateStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
	})
	return &emptypb.Empty{}, err
}

func (cf *AWSRunner) DeleteStack(ctx context.Context, msg *deployer_tpb.DeleteStackMessage) (*emptypb.Empty, error) {
	client, err := cf.getClient(ctx)
	if err != nil {
		return nil, err
	}

	_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName:          aws.String(msg.StackId.StackName),
		ClientRequestToken: buildClientID(msg.StackId),
	})

	return nil, err
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

var stackStatusCreateFailed = []types.StackStatus{
	types.StackStatusRollbackComplete,
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
	types.StackStatusUpdateRollbackComplete,
}

type StackStatus struct {
	StatusName  types.StackStatus
	SummaryType deployer_pb.StackLifecycle
	IsOK        bool
	Stable      bool
	Parameters  []*deployer_pb.KeyValue
	Outputs     []*deployer_pb.KeyValue
}

func stackLifecycle(remoteStatus types.StackStatus) (deployer_pb.StackLifecycle, error) {
	for _, status := range stackStatusesTerminal {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_TERMINAL, nil
		}
	}

	for _, status := range stackStatusComplete {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_COMPLETE, nil
		}
	}

	for _, status := range stackStatusCreateFailed {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_CREATE_FAILED, nil
		}
	}

	for _, status := range stackStatusRollingBack {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_ROLLING_BACK, nil
		}
	}

	for _, status := range stackStatusProgress {
		if remoteStatus == status {
			return deployer_pb.StackLifecycle_PROGRESS, nil
		}
	}

	return deployer_pb.StackLifecycle_UNSPECIFIED, fmt.Errorf("unknown stack status %s", remoteStatus)

}

func mapOutputs(outputs []types.Output) []*deployer_pb.KeyValue {
	out := make([]*deployer_pb.KeyValue, len(outputs))
	for i, output := range outputs {
		out[i] = &deployer_pb.KeyValue{
			Name:  *output.OutputKey,
			Value: *output.OutputValue,
		}
	}
	return out
}

func summarizeStackStatus(stack *types.Stack) (StackStatus, error) {

	lifecycle, err := stackLifecycle(stack.StackStatus)
	if err != nil {
		return StackStatus{}, err
	}

	parameters := make([]*deployer_pb.KeyValue, len(stack.Parameters))
	for idx, param := range stack.Parameters {
		parameters[idx] = &deployer_pb.KeyValue{
			Name:  *param.ParameterKey,
			Value: *param.ParameterValue,
		}
	}

	out := StackStatus{
		StatusName:  stack.StackStatus,
		Parameters:  parameters,
		SummaryType: lifecycle,
		Outputs:     mapOutputs(stack.Outputs),
	}

	switch lifecycle {

	case deployer_pb.StackLifecycle_COMPLETE:
		out.IsOK = true
		out.Stable = true

	case deployer_pb.StackLifecycle_TERMINAL:
		out.IsOK = false
		out.Stable = true

	case deployer_pb.StackLifecycle_CREATE_FAILED:
		out.IsOK = false
		out.Stable = true

	case deployer_pb.StackLifecycle_ROLLING_BACK:
		out.IsOK = false
		out.Stable = false

	case deployer_pb.StackLifecycle_PROGRESS:
		out.IsOK = true
		out.Stable = false

	default:
		return StackStatus{}, fmt.Errorf("unknown stack lifecycle: %s", lifecycle)
	}

	return out, nil
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

type StackPoller struct {
	chErr chan error
}

func (sp StackPoller) Wait(ctx context.Context) error {
	select {
	case err := <-sp.chErr:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (cf *AWSRunner) PollStack(ctx context.Context, stackID *deployer_tpb.StackID) (*StackPoller, error) {

	sp := &StackPoller{
		chErr: make(chan error),
	}

	stackName := stackID.StackName

	ctx = log.WithField(ctx, "stackName", stackName)

	log.Debug(ctx, "PollStack Begin")

	remoteStack, err := cf.getOneStack(ctx, stackName)
	if err != nil {
		return nil, err
	}
	lastStatus := remoteStack.StackStatus

	go func() {
		sp.chErr <- func() error {
			for {
				remoteStack, err := cf.getOneStack(ctx, stackName)
				if err != nil {
					return err
				}
				if remoteStack == nil {
					return fmt.Errorf("missing stack %s", stackName)
				}

				if lastStatus == remoteStack.StackStatus {
					time.Sleep(1 * time.Second)
					continue
				}

				lastStatus = remoteStack.StackStatus

				summary, err := summarizeStackStatus(remoteStack)
				if err != nil {
					return err
				}

				log.WithFields(ctx, map[string]interface{}{
					"lifecycle":   summary.SummaryType.ShortString(),
					"stackStatus": remoteStack.StackStatus,
				}).Debug("PollStack Result")

				if err := cf.eventOut(ctx, &deployer_tpb.StackStatusChangedMessage{
					StackId:   stackID,
					Status:    string(remoteStack.StackStatus),
					Outputs:   summary.Outputs,
					Lifecycle: summary.SummaryType,
				}); err != nil {
					return err
				}
			}
		}()
		log.Debug(ctx, "PollStack End")
	}()

	return sp, nil
}
