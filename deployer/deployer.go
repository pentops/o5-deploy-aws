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
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
)

type Environment struct {
	Environment *environment_pb.Environment
	AWS         *environment_pb.AWS
}

type Deployer struct {
	RotateSecrets bool
	CancelUpdates bool

	s3Client         awsinfra.S3API
	cfTemplateBucket string

	storage DeployerStorage
}

func NewDeployer(storage DeployerStorage, cfTemplateBucket string, s3Client awsinfra.S3API) (*Deployer, error) {
	cfTemplateBucket = strings.TrimPrefix(cfTemplateBucket, "s3://")
	return &Deployer{
		s3Client:         s3Client,
		storage:          storage,
		cfTemplateBucket: cfTemplateBucket,
	}, nil
}

func (d *Deployer) BeginDeployments(ctx context.Context, app *app.BuiltApplication, envNames []string) error {
	for _, envName := range envNames {
		if err := d.BeginDeployment(ctx, app, envName); err != nil {
			return err
		}
	}
	return nil
}

func (d *Deployer) BeginDeployment(ctx context.Context, app *app.BuiltApplication, envName string) error {
	trigger, err := d.BuildTrigger(ctx, app, envName)
	if err != nil {
		return err
	}

	return d.storage.Transact(ctx, func(ctx context.Context, tx TransitionTransaction) error {
		return tx.PublishEvent(ctx, trigger)
	})
}

func (dd *Deployer) BuildTrigger(ctx context.Context, app *app.BuiltApplication, envName string) (*deployer_tpb.TriggerDeploymentMessage, error) {

	ctx = log.WithFields(ctx, map[string]interface{}{
		"appName":     app.Name,
		"environment": envName,
	})

	environment, err := dd.storage.GetEnvironment(ctx, envName)
	if err != nil {
		return nil, err
	}

	awsEnv := environment.GetAws()
	if awsEnv == nil {
		return nil, fmt.Errorf("environment %s is not an AWS environment", envName)
	}

	deploymentID := uuid.NewString()

	templateJSON, err := app.TemplateJSON()
	if err != nil {
		return nil, err
	}

	templateKey := fmt.Sprintf("%s/%s/%s.json", environment.FullName, app.Name, deploymentID)
	_, err = dd.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(dd.cfTemplateBucket),
		Key:    aws.String(templateKey),
		Body:   bytes.NewReader(templateJSON),
	})

	if err != nil {
		return nil, err
	}

	postgresDatabases := app.PostgresDatabases()

	for _, db := range postgresDatabases {
		pgdb := db.Database.GetPostgres()
		for _, host := range awsEnv.RdsHosts {
			if host.ServerGroup == pgdb.ServerGroup {
				db.RdsHost = host
				break
			}
		}
		if db.RdsHost == nil {
			return nil, fmt.Errorf("no RDS host found for database %s", db.Database.Name)
		}

	}

	templateURL := fmt.Sprintf("https://s3.us-east-1.amazonaws.com/%s/%s", dd.cfTemplateBucket, templateKey)

	spec := &deployer_pb.DeploymentSpec{
		AppName:         app.Name,
		Version:         app.Version,
		EnvironmentName: environment.FullName,
		TemplateUrl:     templateURL,
		Databases:       postgresDatabases,
		Parameters:      app.Parameters(),
		SnsTopics:       app.SNSTopics,

		CancelUpdates:     dd.CancelUpdates,
		RotateCredentials: dd.RotateSecrets,
		EcsCluster:        awsEnv.EcsClusterName,
	}

	return &deployer_tpb.TriggerDeploymentMessage{
		DeploymentId: deploymentID,
		Spec:         spec,
	}, nil
}
