package deployer

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	protovalidate "github.com/bufbuild/protovalidate-go"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Deployer struct {
	Environment   *environment_pb.Environment
	AWS           *environment_pb.AWS
	RotateSecrets bool
	Clients       awsinfra.ClientBuilder
	storage       DeployerStorage

	*deployer_tpb.UnimplementedDeployerTopicServer
}

func NewDeployer(storage DeployerStorage, environment *environment_pb.Environment, clientSet awsinfra.ClientBuilder) (*Deployer, error) {

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
		Environment: environment,
		AWS:         awsTarget,
		Clients:     clientSet,
		storage:     storage,
	}, nil
}

func (d *Deployer) BeginDeployment(ctx context.Context, app *app.BuiltApplication, cancelUpdates bool) error {
	ctx = log.WithFields(ctx, map[string]interface{}{
		"appName":     app.Name,
		"environment": d.Environment.FullName,
	})

	clients, err := d.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return err
	}

	templateKey := fmt.Sprintf("%s/%s/%s.json", d.Environment.FullName, app.Name, deploymentID)
	_, err = clients.S3.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(d.AWS.ScratchBucket),
		Key:    aws.String(templateKey),
		Body:   bytes.NewReader(templateJSON),
	})

	if err != nil {
		return err
	}

	templateURL := fmt.Sprintf("https://s3.us-east-1.amazonaws.com/%s/%s", d.AWS.ScratchBucket, templateKey)

	spec := &deployer_pb.DeploymentSpec{
		AppName:           app.Name,
		Version:           app.Version,
		EnvironmentName:   d.Environment.FullName,
		TemplateUrl:       templateURL,
		Databases:         app.PostgresDatabases(),
		Parameters:        app.Parameters(),
		CancelUpdates:     cancelUpdates,
		SnsTopics:         app.SNSTopics,
		RotateCredentials: d.RotateSecrets,
	}

	return d.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return tx.PublishEvent(ctx, &deployer_tpb.TriggerDeploymentMessage{
			DeploymentId: deploymentID,
			Spec:         spec,
		})
	})
}

func (dd *Deployer) TriggerDeployment(ctx context.Context, msg *deployer_tpb.TriggerDeploymentMessage) (*emptypb.Empty, error) {

	deploymentEvent := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_Triggered_{
				Triggered: &deployer_pb.DeploymentEventType_Triggered{
					Spec: msg.Spec,
				},
			},
		},
	}

	if err := dd.RegisterEvent(ctx, deploymentEvent); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (dd *Deployer) StackStatusChanged(ctx context.Context, msg *deployer_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	deployment, err := dd.storage.GetDeploymentForStack(ctx, msg.StackName)
	if err != nil {
		return nil, err
	}

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: deployment.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},
		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_StackStatus_{
				StackStatus: &deployer_pb.DeploymentEventType_StackStatus{
					Lifecycle:   msg.Lifecycle,
					FullStatus:  msg.Status,
					StackOutput: msg.Outputs,
				},
			},
		},
	}
	return &emptypb.Empty{}, dd.RegisterEvent(ctx, event)

}

func (dd *Deployer) MigrationStatusChanged(ctx context.Context, msg *deployer_tpb.MigrationStatusChangedMessage) (*emptypb.Empty, error) {

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.DeploymentId,
		Metadata: &deployer_pb.EventMetadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
		},

		Event: &deployer_pb.DeploymentEventType{
			Type: &deployer_pb.DeploymentEventType_DbMigrateStatus{
				DbMigrateStatus: &deployer_pb.DeploymentEventType_DBMigrateStatus{
					MigrationId: msg.MigrationId,
					Status:      msg.Status,
				},
			},
		},
	}
	return &emptypb.Empty{}, dd.RegisterEvent(ctx, event)

}
