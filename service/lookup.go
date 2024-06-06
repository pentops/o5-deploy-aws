package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/states"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

var environmentIDNamespace = uuid.MustParse("0D783718-F8FD-4543-AE3D-6382AB0B8178")
var clusterIDNamespace = uuid.MustParse("9B3AC0AB-4414-4E6F-B1E9-20300D2D8CE3")

func environmentNameID(name string) string {
	return uuid.NewSHA1(environmentIDNamespace, []byte(name)).String()
}

// LookupProvider allows API calls to be requested with names rather than UUIDs
// for the state machines.
type LookupProvider struct {
	db *sqrlx.Wrapper
}

func NewLookupProvider(conn sqrlx.Connection) (*LookupProvider, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}
	return &LookupProvider{
		db: db,
	}, nil
}

type environmentIdentifiers struct {
	fullName      string
	environmentID string
	clusterID     string
}

type stackIdentifiers struct {
	environment environmentIdentifiers
	appName     string
	stackID     string
}

type clusterIdentifiers struct {
	clusterID   string
	clusterName string
}

func (ds *LookupProvider) stackByID(ctx context.Context, stackID string) (stackIdentifiers, error) {
	query := sq.
		Select(
			"stack.stack_id",
			"stack.environment_id",
			"stack.cluster_id",
			"state->'data'->>'applicationName'",
			"state->'data'->>'environmentName'",
		).From("stack").Where("stack_id = ?", stackID)

	res := stackIdentifiers{}

	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, query).Scan(
			&res.stackID,
			&res.environment.environmentID,
			&res.environment.clusterID,
			&res.appName,
			&res.environment.fullName,
		)
	})
	if err == nil { // HAPPY SAD FLIP
		return res, nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return res, status.Errorf(codes.NotFound, "stack ID '%s' not found", stackID)
	}

	return res, err
}

func (ds *LookupProvider) lookupStack(ctx context.Context, presented string) (stackIdentifiers, error) {

	fallbackAppName := ""
	fallbackEnvName := ""

	parts := strings.Split(presented, "-")
	if len(parts) == 2 {
		fallbackEnvName = parts[0]
		fallbackAppName = parts[1]
	} else if _, err := uuid.Parse(presented); err == nil {
		return ds.stackByID(ctx, presented)
	} else {
		return stackIdentifiers{}, status.Error(codes.InvalidArgument, "invalid stack id")
	}

	query := sq.
		Select(
			"stack.stack_id",
			"stack.environment_id",
			"stack.cluster_id",
			"state->'data'->>'applicationName'",
			"state->'data'->>'environmentName'",
		).From("stack").
		Where("state->'data'->>'environmentName' = ?", fallbackEnvName).
		Where("state->'data'->>'applicationName' = ?", fallbackAppName)

	res := stackIdentifiers{}

	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, query).Scan(
			&res.stackID,
			&res.environment.environmentID,
			&res.environment.clusterID,
			&res.appName,
			&res.environment.fullName,
		)
	})
	if err == nil { // HAPPY SAD FLIP
		return res, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	env, err := ds.lookupEnvironment(ctx, fallbackEnvName, "")
	if err != nil {
		return res, err
	}

	res.appName = fallbackAppName
	res.environment = env
	res.stackID = states.StackID(fallbackEnvName, fallbackAppName)
	log.WithFields(ctx, map[string]interface{}{
		"stack_id":    res.stackID,
		"environment": fallbackEnvName,
		"application": fallbackAppName,
	}).Debug("derived stack id")

	return res, nil

}

func (ds *LookupProvider) lookupEnvironment(ctx context.Context, presentedEnvironment, presentedCluster string) (environmentIdentifiers, error) {
	query := sq.
		Select(
			"environment.environment_id",
			"environment.cluster_id",
			"state->'data'->'config'->>'fullName'",
		).
		From("environment")

	fallbackToName := ""
	if _, err := uuid.Parse(presentedEnvironment); err == nil {
		query = query.Where("environment_id = ?", presentedEnvironment)
	} else {
		query = query.Where("state->'data'->'config'->>'fullName' = ?", presentedEnvironment)
		fallbackToName = presentedEnvironment
	}

	res := environmentIdentifiers{}
	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, query).Scan(&res.environmentID, &res.clusterID, &res.fullName)
	})
	if err == nil {
		return res, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	if fallbackToName == "" || presentedCluster == "" {
		return res, status.Errorf(codes.NotFound, "environment '%s', cluster '%s', not found", presentedEnvironment, presentedCluster)
	}
	id := environmentNameID(fallbackToName)
	res.environmentID = id
	res.fullName = fallbackToName

	if _, err := uuid.Parse(presentedCluster); err == nil {
		res.clusterID = presentedCluster
		return res, nil
	}

	backupQuery := sq.Select("cluster_id").
		From("cluster").
		Where("state->'data'->'config'->>'name' = ?", presentedCluster)

	err = ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, backupQuery).Scan(&res.clusterID)
	})
	if err == nil {
		return res, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	return res, status.Errorf(codes.NotFound, "cluster %s not found", presentedCluster)
}

func (ds *LookupProvider) lookupCluster(ctx context.Context, presented string) (clusterIdentifiers, error) {
	query := sq.
		Select(
			"cluster_id",
			"state->'data'->'config'->>'name'",
		).
		From("cluster")

	fallbackToName := ""
	if _, err := uuid.Parse(presented); err == nil {
		query = query.Where("cluster_id = ?", presented)
	} else {
		query = query.Where("state->'data'->'config'->>'name' = ?", presented)
		fallbackToName = presented
	}

	res := clusterIdentifiers{}
	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, query).Scan(&res.clusterID, &res.clusterName)
	})
	if err == nil {
		return res, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		return res, err
	}

	if fallbackToName == "" {
		return res, status.Errorf(codes.NotFound, "cluster '%s' not found", presented)
	}

	id := uuid.NewSHA1(clusterIDNamespace, []byte(fallbackToName)).String()

	return clusterIdentifiers{
		clusterID:   id,
		clusterName: fallbackToName,
	}, nil
}

func (ds *LookupProvider) lookupAppStack(ctx context.Context, environmentId string, appName string) (*appStack, error) {
	var envJSON []byte
	var clusterJSON []byte

	query := sq.Select(
		"cluster.state",
		"environment.state").
		From("environment").
		LeftJoin("cluster ON cluster.cluster_id = environment.cluster_id").
		Where("environment.environment_id = ?", environmentId)

	err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		return tx.SelectRow(ctx, query).Scan(&clusterJSON, &envJSON)
	})

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("environment %q not found", environmentId)
	} else if err != nil {
		return nil, err
	}

	cluster := &awsdeployer_pb.ClusterState{}
	if err := protojson.Unmarshal(clusterJSON, cluster); err != nil {
		return nil, fmt.Errorf("unmarshal cluster: %w", err)
	}

	env := &awsdeployer_pb.EnvironmentState{}
	if err := protojson.Unmarshal(envJSON, env); err != nil {
		return nil, fmt.Errorf("unmarshal environment: %w", err)
	}
	return &appStack{
		environment:   env.Data.Config,
		cluster:       cluster.Data.Config,
		environmentID: env.Keys.EnvironmentId,
		clusterID:     cluster.Keys.ClusterId,
		stackID:       states.StackID(env.Data.Config.FullName, appName),
	}, nil
}
