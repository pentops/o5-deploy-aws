package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/j5/gen/j5/state/v1/psm_j5pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/apps/service/internal/states"
	"github.com/pentops/o5-deploy-aws/internal/deployer"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeployerWorker struct {
	awsdeployer_tpb.UnsafeDeploymentRequestTopicServer
	awsinfra_tpb.UnsafeCloudFormationReplyTopicServer
	awsinfra_tpb.UnsafePostgresReplyTopicServer
	awsinfra_tpb.UnsafeECSReplyTopicServer

	lookup *LookupProvider

	specBuilder       *deployer.SpecBuilder
	stackEventer      *awsdeployer_pb.StackPSM
	deploymentEventer *awsdeployer_pb.DeploymentPSMDB
}

var _ awsinfra_tpb.ECSReplyTopicServer = &DeployerWorker{}
var _ awsinfra_tpb.CloudFormationReplyTopicServer = &DeployerWorker{}
var _ awsinfra_tpb.PostgresReplyTopicServer = &DeployerWorker{}
var _ awsdeployer_tpb.DeploymentRequestTopicServer = &DeployerWorker{}

func NewDeployerWorker(db sqrlx.Transactor, specBuilder *deployer.SpecBuilder, states *states.StateMachines) (*DeployerWorker, error) {

	lookupProvider, err := NewLookupProvider(db)
	if err != nil {
		return nil, err
	}

	return &DeployerWorker{
		specBuilder:       specBuilder,
		stackEventer:      states.Stack,
		deploymentEventer: states.Deployment.WithDB(db),
		lookup:            lookupProvider,
	}, nil
}

func (dw *DeployerWorker) doDeploymentEvent(ctx context.Context, event *awsdeployer_pb.DeploymentPSMEventSpec) error {
	_, err := dw.deploymentEventer.Transition(ctx, event)
	return err
}

type appStack struct {
	environment   *environment_pb.Environment
	cluster       *environment_pb.Cluster
	environmentID string
	clusterID     string
	stackID       string
}

func (dw *DeployerWorker) RequestDeployment(ctx context.Context, msg *awsdeployer_tpb.RequestDeploymentMessage) (*emptypb.Empty, error) {
	appID, err := dw.lookup.lookupAppStack(ctx, msg.EnvironmentId, msg.Application.Name)
	if err != nil {
		return nil, err
	}

	spec, err := dw.specBuilder.BuildSpec(ctx, msg, appID.cluster, appID.environment)
	if err != nil {
		return nil, err
	}

	createDeploymentEvent := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
			DeploymentId:  msg.DeploymentId,
			EnvironmentId: appID.environmentID,
			StackId:       appID.stackID,
			ClusterId:     appID.clusterID,
		},
		EventID:   msg.DeploymentId,
		Timestamp: time.Now(),
		Event: &awsdeployer_pb.DeploymentEventType_Created{
			Request: msg.Request,
			Spec:    spec,
		},
		Cause: &psm_j5pb.Cause{
			Type: &psm_j5pb.Cause_ExternalEvent{
				ExternalEvent: &psm_j5pb.ExternalEventCause{
					SystemName: "deployer",
					EventName:  "RequestDeployment",
				},
			},
		},
	}

	if _, err := dw.deploymentEventer.Transition(ctx, createDeploymentEvent); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

var cfEventNamespace = uuid.MustParse("6BE12207-A62C-4D9E-8A94-48A091DDFB53")

func (dw *DeployerWorker) ChangeSetStatusChanged(ctx context.Context, msg *awsinfra_tpb.ChangeSetStatusChangedMessage) (*emptypb.Empty, error) {
	event, err := ChangeSetStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &emptypb.Empty{}, nil
	}

	if _, err := dw.deploymentEventer.Transition(ctx, event); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) StackStatusChanged(ctx context.Context, msg *awsinfra_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	event, err := StackStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &emptypb.Empty{}, nil
	}

	if _, err := dw.deploymentEventer.Transition(ctx, event); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dw *DeployerWorker) PostgresDatabaseStatus(ctx context.Context, msg *awsinfra_tpb.PostgresDatabaseStatusMessage) (*emptypb.Empty, error) {

	event, err := PostgresMigrationToEvent(msg)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)

}

func (dw *DeployerWorker) ECSTaskStatus(ctx context.Context, msg *awsinfra_tpb.ECSTaskStatusMessage) (*emptypb.Empty, error) {

	event, err := ECSTaskStatusToEvent(msg)
	if err != nil {
		return nil, err
	}

	if event == nil {
		return &emptypb.Empty{}, nil
	}

	return &emptypb.Empty{}, dw.doDeploymentEvent(ctx, event)
}
