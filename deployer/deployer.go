package deployer

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/awsinfra"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_spb"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Environment struct {
	Environment *environment_pb.Environment
	AWS         *environment_pb.AWS
}

type DeploymentQuerier struct {
	storage DeployerStorage

	*deployer_spb.UnimplementedDeploymentQueryServiceServer
}

func NewDeploymentQuerier(storage DeployerStorage) *DeploymentQuerier {
	return &DeploymentQuerier{storage: storage}
}

type DeploymentManager struct {
	RotateSecrets bool
	CancelUpdates bool

	s3Client         awsinfra.S3API
	cfTemplateBucket string

	storage DeployerStorage

	*deployer_tpb.UnimplementedDeployerTopicServer
}

func NewDeploymentManager(storage DeployerStorage, cfTemplateBucket string, s3Client awsinfra.S3API) (*DeploymentManager, error) {
	cfTemplateBucket = strings.TrimPrefix(cfTemplateBucket, "s3://")
	return &DeploymentManager{
		s3Client:         s3Client,
		storage:          storage,
		cfTemplateBucket: cfTemplateBucket,
	}, nil
}

func (d *DeploymentManager) BeginDeployments(ctx context.Context, app *app.BuiltApplication, envNames []string) error {
	for _, envName := range envNames {
		if err := d.BeginDeployment(ctx, app, envName); err != nil {
			return err
		}
	}
	return nil
}

func (d *DeploymentManager) BeginDeployment(ctx context.Context, app *app.BuiltApplication, envName string) error {

	ctx = log.WithFields(ctx, map[string]interface{}{
		"appName":     app.Name,
		"environment": envName,
	})

	environment, err := d.storage.GetEnvironment(ctx, envName)
	if err != nil {
		return err
	}

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return fmt.Errorf("environment %s is not an AWS environment", envName)
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return err
	}

	templateKey := fmt.Sprintf("%s/%s/%s.json", environment.FullName, app.Name, deploymentID)
	_, err = d.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(d.cfTemplateBucket),
		Key:    aws.String(templateKey),
		Body:   bytes.NewReader(templateJSON),
	})

	if err != nil {
		return err
	}

	templateURL := fmt.Sprintf("https://s3.us-east-1.amazonaws.com/%s/%s", d.cfTemplateBucket, templateKey)

	spec := &deployer_pb.DeploymentSpec{
		AppName:         app.Name,
		Version:         app.Version,
		EnvironmentName: environment.FullName,
		TemplateUrl:     templateURL,
		Databases:       app.PostgresDatabases(),
		Parameters:      app.Parameters(),
		SnsTopics:       app.SNSTopics,

		CancelUpdates:     d.CancelUpdates,
		RotateCredentials: d.RotateSecrets,
	}

	return d.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return tx.PublishEvent(ctx, &deployer_tpb.TriggerDeploymentMessage{
			DeploymentId: deploymentID,
			Spec:         spec,
		})
	})
}

func (dd *DeploymentManager) TriggerDeployment(ctx context.Context, msg *deployer_tpb.TriggerDeploymentMessage) (*emptypb.Empty, error) {

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

func (dd *DeploymentManager) StackStatusChanged(ctx context.Context, msg *deployer_tpb.StackStatusChangedMessage) (*emptypb.Empty, error) {

	event := &deployer_pb.DeploymentEvent{
		DeploymentId: msg.StackId.DeploymentId,
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

func (dd *DeploymentManager) MigrationStatusChanged(ctx context.Context, msg *deployer_tpb.MigrationStatusChangedMessage) (*emptypb.Empty, error) {

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
