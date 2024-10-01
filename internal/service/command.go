package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/envconf.go/envconf"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_spb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_tpb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/protoread"
	"github.com/pentops/o5-deploy-aws/internal/states"
	"github.com/pentops/o5-messaging/outbox"
	"github.com/pentops/realms/j5auth"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GithubClient interface {
	PullO5Configs(ctx context.Context, owner, repo, commit string) ([]*application_pb.Application, error)
}

func OpenDatabase(ctx context.Context) (*sql.DB, error) {
	var config = struct {
		PostgresURL string `env:"POSTGRES_URL"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, err
	}

	// Default is unlimited connections, use a cap to prevent hammering the database if it's the bottleneck.
	// 10 was selected as a conservative number and will likely be revised later.
	db.SetMaxOpenConns(10)

	for {
		if err := db.Ping(); err != nil {
			log.WithError(ctx, err).Error("pinging PG")
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return db, nil
}

type CommandService struct {
	deploymentStateMachine  *awsdeployer_pb.DeploymentPSM
	environmentStateMachine *awsdeployer_pb.EnvironmentPSM
	stackStateMachine       *awsdeployer_pb.StackPSM
	clusterStateMachine     *awsdeployer_pb.ClusterPSM

	db     sqrlx.Transactor
	github GithubClient

	*LookupProvider

	awsdeployer_spb.UnsafeDeploymentCommandServiceServer
}

func NewCommandService(db sqrlx.Transactor, github GithubClient, stateMachines *states.StateMachines) (*CommandService, error) {

	lookupProvider, err := NewLookupProvider(db)
	if err != nil {
		return nil, err
	}

	return &CommandService{
		deploymentStateMachine:  stateMachines.Deployment,
		environmentStateMachine: stateMachines.Environment,
		stackStateMachine:       stateMachines.Stack,
		clusterStateMachine:     stateMachines.Cluster,
		LookupProvider:          lookupProvider,

		db:     db,
		github: github,
	}, nil
}

func (ds *CommandService) TriggerDeployment(ctx context.Context, req *awsdeployer_spb.TriggerDeploymentRequest) (*awsdeployer_spb.TriggerDeploymentResponse, error) {
	if req.Source == nil || req.Source.GetGithub() == nil {
		return nil, status.Error(codes.Unimplemented, "only github source is supported")
	}
	gh := req.Source.GetGithub()

	if gh.GetCommit() == "" {
		return nil, status.Error(codes.InvalidArgument, "only commit is supported currently")
	}

	commitHash := gh.GetCommit()

	apps, err := ds.github.PullO5Configs(ctx, gh.Owner, gh.Repo, commitHash)
	if err != nil {
		return nil, fmt.Errorf("github: pull o5 config: %w", err)
	}

	if len(apps) == 0 {
		return nil, fmt.Errorf("no applications found in push event")
	}

	if len(apps) > 1 {
		return nil, fmt.Errorf("multiple applications found in push event, not yet supported")
	}

	environmentID, err := ds.lookupEnvironment(ctx, req.Environment, "")
	if err != nil {
		return nil, err
	}

	if req.Flags == nil {
		req.Flags = &awsdeployer_pb.DeploymentFlags{}
	}

	requestMessage := &awsdeployer_tpb.RequestDeploymentMessage{
		DeploymentId:  req.DeploymentId,
		Application:   apps[0],
		Version:       gh.GetCommit(),
		EnvironmentId: environmentID.environmentID,
		Flags:         req.Flags,
		/*
			Source: &awsdeployer_pb.CodeSourceType{
				Type: &awsdeployer_pb.CodeSourceType_Github_{
					Github: &awsdeployer_pb.CodeSourceType_Github{
						Owner:  gh.Owner,
						Repo:   gh.Repo,
						Commit: commitHash,
					},
				},
			},*/
	}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return outbox.Send(ctx, tx, requestMessage)
	}); err != nil {
		return nil, err
	}

	return &awsdeployer_spb.TriggerDeploymentResponse{
		DeploymentId:  req.DeploymentId,
		EnvironmentId: environmentID.environmentID,
	}, nil
}

func (ds *CommandService) TerminateDeployment(ctx context.Context, req *awsdeployer_spb.TerminateDeploymentRequest) (*awsdeployer_spb.TerminateDeploymentResponse, error) {

	action, err := j5auth.GetAuthenticatedAction(ctx)
	if err != nil {
		return nil, err
	}

	event := &awsdeployer_pb.DeploymentPSMEventSpec{
		Keys: &awsdeployer_pb.DeploymentKeys{
			DeploymentId: req.DeploymentId,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Event:     &awsdeployer_pb.DeploymentEventType_Terminated{},
		Action:    action,
	}

	_, err = ds.deploymentStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &awsdeployer_spb.TerminateDeploymentResponse{}, nil
}

func (ds *CommandService) UpsertEnvironment(ctx context.Context, req *awsdeployer_spb.UpsertEnvironmentRequest) (*awsdeployer_spb.UpsertEnvironmentResponse, error) {

	action, err := j5auth.GetAuthenticatedAction(ctx)
	if err != nil {
		return nil, err
	}

	var config *environment_pb.Environment
	switch src := req.Src.(type) {
	case *awsdeployer_spb.UpsertEnvironmentRequest_Config:
		config = src.Config

	case *awsdeployer_spb.UpsertEnvironmentRequest_ConfigYaml:
		config = &environment_pb.Environment{}
		if err := protoread.Parse("env.yaml", src.ConfigYaml, config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

	case *awsdeployer_spb.UpsertEnvironmentRequest_ConfigJson:
		config = &environment_pb.Environment{}
		if err := protoread.Parse("env.json", src.ConfigJson, config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid config type")
	}

	if req.EnvironmentId == "" || req.EnvironmentId == "-" {
		req.EnvironmentId = config.FullName
	}
	identifiers, err := ds.lookupEnvironment(ctx, req.EnvironmentId, req.ClusterId)
	if err != nil {
		return nil, err
	}

	log.WithFields(ctx, map[string]interface{}{
		"environmentID": identifiers.environmentID,
		"clusterID":     identifiers.clusterID,
		"envName":       identifiers.fullName,
	}).Debug("environment identifiers")

	event := &awsdeployer_pb.EnvironmentPSMEventSpec{
		Keys: &awsdeployer_pb.EnvironmentKeys{
			EnvironmentId: identifiers.environmentID,
			ClusterId:     identifiers.clusterID,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Action:    action,
		Event: &awsdeployer_pb.EnvironmentEventType_Configured{
			Config: config,
		},
	}

	state, err := ds.environmentStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		log.WithError(ctx, err).Error("failed to transition environment")
		return nil, err
	}

	return &awsdeployer_spb.UpsertEnvironmentResponse{
		State: state,
	}, nil
}

func (ds *CommandService) UpsertStack(ctx context.Context, req *awsdeployer_spb.UpsertStackRequest) (*awsdeployer_spb.UpsertStackResponse, error) {
	identifiers, err := ds.lookupStack(ctx, req.StackId)
	if err != nil {
		return nil, err
	}

	action, err := j5auth.GetAuthenticatedAction(ctx)
	if err != nil {
		return nil, err
	}

	event := &awsdeployer_pb.StackPSMEventSpec{
		Keys: &awsdeployer_pb.StackKeys{
			StackId:       identifiers.stackID,
			EnvironmentId: identifiers.environment.environmentID,
			ClusterId:     identifiers.environment.clusterID,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Action:    action,

		Event: &awsdeployer_pb.StackEventType_Configured{
			EnvironmentId:   identifiers.environment.environmentID,
			ApplicationName: identifiers.appName,
			EnvironmentName: identifiers.environment.fullName,
		},
	}

	newState, err := ds.stackStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &awsdeployer_spb.UpsertStackResponse{
		State: newState,
	}, nil
}

func (ds *CommandService) SetClusterOverride(ctx context.Context, req *awsdeployer_spb.SetClusterOverrideRequest) (*awsdeployer_spb.SetClusterOverrideResponse, error) {

	if err := states.ValidateClusterOverrides(req.Overrides); err != nil {
		return nil, err
	}
	identifiers, err := ds.lookupCluster(ctx, req.ClusterId)
	if err != nil {
		return nil, err
	}

	action, err := j5auth.GetAuthenticatedAction(ctx)
	if err != nil {
		return nil, err
	}

	event := &awsdeployer_pb.ClusterPSMEventSpec{
		Keys: &awsdeployer_pb.ClusterKeys{
			ClusterId: identifiers.clusterID,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Action:    action,
		Event: &awsdeployer_pb.ClusterEventType_Override{
			Overrides: req.Overrides,
		},
	}

	state, err := ds.clusterStateMachine.Transition(ctx, ds.db, event)
	if err != nil {
		return nil, err
	}

	return &awsdeployer_spb.SetClusterOverrideResponse{
		State: state,
	}, nil
}

func (ds *CommandService) UpsertCluster(ctx context.Context, req *awsdeployer_spb.UpsertClusterRequest) (*awsdeployer_spb.UpsertClusterResponse, error) {

	action, err := j5auth.GetAuthenticatedAction(ctx)
	if err != nil {
		return nil, err
	}

	var config *environment_pb.CombinedConfig
	switch src := req.Src.(type) {
	case *awsdeployer_spb.UpsertClusterRequest_Config:
		config = src.Config

	case *awsdeployer_spb.UpsertClusterRequest_ConfigYaml:
		config = &environment_pb.CombinedConfig{}
		if err := protoread.Parse("env.yaml", src.ConfigYaml, config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

	case *awsdeployer_spb.UpsertClusterRequest_ConfigJson:
		config = &environment_pb.CombinedConfig{}
		if err := protoread.Parse("env.json", src.ConfigJson, config); err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid config type")
	}

	if req.ClusterId == "" {
		req.ClusterId = config.Name
	}

	identifiers, err := ds.lookupCluster(ctx, req.ClusterId)
	if err != nil {
		return nil, err
	}

	cluster := &environment_pb.Cluster{
		Name: config.Name,
	}
	switch et := config.Provider.(type) {
	case *environment_pb.CombinedConfig_EcsCluster:
		cluster.Provider = &environment_pb.Cluster_EcsCluster{
			EcsCluster: et.EcsCluster,
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported provider %T", config.Provider)
	}

	event := &awsdeployer_pb.ClusterPSMEventSpec{
		Keys: &awsdeployer_pb.ClusterKeys{
			ClusterId: identifiers.clusterID,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Action:    action,
		Event: &awsdeployer_pb.ClusterEventType_Configured{
			Config: cluster,
		},
	}

	envEvents := make([]*awsdeployer_pb.EnvironmentPSMEventSpec, 0, len(config.Environments))

	for _, envConfig := range config.Environments {
		event := &awsdeployer_pb.EnvironmentPSMEventSpec{
			Keys: &awsdeployer_pb.EnvironmentKeys{
				EnvironmentId: environmentNameID(envConfig.FullName),
				ClusterId:     identifiers.clusterID,
			},
			EventID:   uuid.NewString(),
			Timestamp: time.Now(),
			Action:    action,

			Event: &awsdeployer_pb.EnvironmentEventType_Configured{
				Config: envConfig,
			},
		}
		envEvents = append(envEvents, event)
	}
	response := &awsdeployer_spb.UpsertClusterResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		state, err := ds.clusterStateMachine.TransitionInTx(ctx, tx, event)
		if err != nil {
			return fmt.Errorf("cluster transition: %w", err)
		}
		response.State = state

		for _, evt := range envEvents {
			_, err := ds.environmentStateMachine.TransitionInTx(ctx, tx, evt)
			if err != nil {
				return fmt.Errorf("environment transition: %w", err)
			}

		}
		return nil
	}); err != nil {
		return nil, err
	}

	return response, nil
}
