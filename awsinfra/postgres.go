package awsinfra

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sq "github.com/elgris/sqrl"
	_ "github.com/lib/pq"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/deployer/v1/deployer_tpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.daemonl.com/sqrlx"
)

func (d *AWSRunner) RunDatabaseMigration(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) (*emptypb.Empty, error) {
	if err := d.upsertPostgresDatabase(ctx, msg); err != nil {
		return nil, err
	}

	if msg.Database.MigrationTaskOutputName != nil {

		if msg.MigrationTaskArn == "" {
			return nil, fmt.Errorf("migration task output %q not found", msg.Database.Database.Name)
		}

		if err := d.runMigrationTask(ctx, msg); err != nil {
			return nil, err
		}

		// This runs both before and after migration
		if err := d.fixPostgresOwnership(ctx, msg); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil

}

func (d *AWSRunner) runMigrationTask(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) error {

	clients, err := d.Clients.Clients(ctx)
	if err != nil {
		return err
	}
	ecsClient := clients.ECS

	task, err := ecsClient.RunTask(ctx, &ecs.RunTaskInput{
		TaskDefinition: aws.String(msg.MigrationTaskArn),
		Cluster:        aws.String(msg.EcsClusterName),
		Count:          aws.Int32(1),
	})
	if err != nil {
		return err
	}

	for {
		state, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Tasks:   []string{*task.Tasks[0].TaskArn},
			Cluster: aws.String(msg.EcsClusterName),
		})
		if err != nil {
			return err
		}

		if len(state.Tasks) != 1 {
			return fmt.Errorf("expected 1 task, got %d", len(state.Tasks))
		}
		task := state.Tasks[0]
		log.WithFields(ctx, map[string]interface{}{
			"status": *task.LastStatus,
		}).Debug("waiting for task to stop")

		if *task.LastStatus != "STOPPED" {
			time.Sleep(time.Second)
			continue
		}

		if len(state.Tasks[0].Containers) != 1 {
			return fmt.Errorf("expected 1 container, got %d", len(state.Tasks[0].Containers))
		}
		container := state.Tasks[0].Containers[0]
		if container.ExitCode == nil {
			return fmt.Errorf("task stopped with no exit code: %s", stringValue(container.Reason))
		}
		if *container.ExitCode != 0 {
			return fmt.Errorf("exit code was %d", *container.ExitCode)
		}
		return nil
	}
}

func stringValue(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}

type DBSecret struct {
	Username string `json:"dbuser"`
	Password string `json:"dbpass"`
	Hostname string `json:"dbhost"`
	DBName   string `json:"dbname"`
	URL      string `json:"dburl"`
}

func (ss DBSecret) buildURLForDB(dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:5432/%s", ss.Username, ss.Password, ss.Hostname, dbName)
}

func (d *AWSRunner) rootPostgresCredentials(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) (*DBSecret, error) {
	// "/${var.env_name}/global/rds/${var.name}/root" from TF

	clients, err := d.Clients.Clients(ctx)
	if err != nil {
		return nil, err
	}

	res, err := clients.SecretsManager.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(msg.RootSecretName),
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", msg.RootSecretName, err)
	}

	secretVal := &DBSecret{}
	if err := json.Unmarshal([]byte(*res.SecretString), secretVal); err != nil {
		return nil, err
	}

	return secretVal, nil
}

func (d *AWSRunner) fixPostgresOwnership(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) error {

	log.Info(ctx, "Fix object ownership")
	pgSpec := msg.Database.Database.GetPostgres()
	dbName := msg.Database.Database.Name
	if pgSpec.DbName != "" {
		dbName = pgSpec.DbName
	}
	ownerName := dbName

	rootSecret, err := d.rootPostgresCredentials(ctx, msg)
	if err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"hostname": rootSecret.Hostname,
		"dbName":   dbName,
	}).Debug("Connecting to RDS for db as root user")

	rootConn, err := sql.Open("postgres", rootSecret.buildURLForDB(dbName))
	if err != nil {
		return err
	}

	_, err = rootConn.ExecContext(ctx, fmt.Sprintf(`GRANT USAGE, CREATE ON SCHEMA public TO %s`, ownerName))
	if err != nil {
		return err
	}

	// Query from psql -E -C \d
	rows, err := rootConn.QueryContext(ctx, `
		SELECT
			c.relname,
			c.relkind,
			pg_catalog.pg_get_userbyid(c.relowner)
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind IN ('r', 'v', 'm', 'S', 'f', 'p')
		AND n.nspname <> 'pg_catalog'
		AND n.nspname !~ '^pg_toast'
		AND n.nspname <> 'information_schema'
		AND pg_catalog.pg_table_is_visible(c.oid)
		`)
	if err != nil {
		return err
	}

	type object struct {
		name  string
		owner string
		kind  string
	}
	objects := map[string][]object{}
	for rows.Next() {
		o := object{}
		if err := rows.Scan(&o.name, &o.kind, &o.owner); err != nil {
			return err
		}
		if o.owner == ownerName {
			continue
		}
		objects[o.kind] = append(objects[o.kind], o)
	}

	defer rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	type typeSpec struct {
		key  string
		name string
	}

	for _, typeSpec := range []typeSpec{{
		key:  "r",
		name: "TABLE",
	}, {
		key:  "v",
		name: "VIEW",
	}, {
		key:  "S",
		name: "SEQUENCE",
	}, {
		key:  "i",
		name: "INDEX",
	}} {

		for _, object := range objects[typeSpec.key] {
			log.WithFields(ctx, map[string]interface{}{
				"badOwner":   object.owner,
				"objectName": object.name,
				"objectKind": object.kind,
				"newOwner":   ownerName,
			}).Info("fixing object ownership")
			stmt := fmt.Sprintf("ALTER %s %s OWNER TO %s", typeSpec.name, object.name, ownerName)
			_, err = rootConn.ExecContext(ctx, stmt)
			if err != nil {
				return err
			}
		}
		delete(objects, typeSpec.key)
	}

	for _, ll := range objects {
		for _, object := range ll {
			return fmt.Errorf("unknown object types %s %s with owner %s", object.name, object.kind, object.owner)
		}
	}

	return nil
}

func (d *AWSRunner) upsertPostgresDatabase(ctx context.Context, msg *deployer_tpb.RunDatabaseMigrationMessage) error {

	// spec *deployer_pb.PostgresDatabase, secretARN string, rotateExisting bool) error {
	//.Database, msg.SecretArn, msg.RotateCredentials); err != nil {

	pgSpec := msg.Database.Database.GetPostgres()
	rootSecret, err := d.rootPostgresCredentials(ctx, msg)
	if err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"hostname": rootSecret.Hostname,
	}).Debug("Connecting to RDS server as root user")

	rootConn, err := sql.Open("postgres", rootSecret.URL)
	if err != nil {
		return err
	}

	if err := rootConn.PingContext(ctx); err != nil {
		return err
	}

	defer rootConn.Close()

	db, err := sqrlx.NewWithCommander(rootConn, sq.Dollar)
	if err != nil {
		return err
	}

	var count int

	dbName := msg.Database.Database.Name
	if pgSpec.DbName != "" {
		dbName = pgSpec.DbName
	}

	err = db.SelectRow(ctx, sq.
		Select("coalesce(count(datname), 0)").
		From("pg_catalog.pg_database").
		Where("datname = ?", dbName),
	).Scan(&count)
	if err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"count": count,
	}).Debug("Found DBs")

	if count > 1 {
		return fmt.Errorf("more than one DB matched %s:%s", pgSpec.ServerGroup, dbName)
	} else if count == 0 {
		_, err = db.ExecRaw(ctx, fmt.Sprintf(`CREATE ROLE %s`, dbName))
		if err != nil {
			return err
		}

		_, err = db.ExecRaw(ctx, fmt.Sprintf(`GRANT %s TO current_user`, dbName))
		if err != nil {
			return err
		}

		_, err = db.ExecRaw(ctx, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, dbName, dbName))
		if err != nil {
			return err
		}
	}

	if err := d.fixPostgresOwnership(ctx, msg); err != nil {
		return err
	}

	if count == 1 && !msg.RotateCredentials {
		return nil
	}

	if len(pgSpec.DbExtensions) > 0 {
		log.WithFields(ctx, map[string]interface{}{
			"count": len(pgSpec.DbExtensions),
		}).Debug("Adding Extensions")
		if err := func() error {
			superuserURL := rootSecret.buildURLForDB(dbName)
			superuserConn, err := sql.Open("postgres", superuserURL)
			if err != nil {
				return err
			}
			defer superuserConn.Close()

			for _, ext := range pgSpec.DbExtensions {
				if !reSafeExtensionName.MatchString(ext) {
					return fmt.Errorf("unsafe extension name: %s", ext)
				}

				_, err = superuserConn.ExecContext(ctx, fmt.Sprintf(`CREATE EXTENSION IF NOT EXISTS "%s"`, ext))
				if err != nil {
					return err
				}
			}

			return nil
		}(); err != nil {
			return err
		}
	}

	newSecret := DBSecret{
		DBName:   dbName,
		Hostname: rootSecret.Hostname,
		Username: fmt.Sprintf("%s_%d", dbName, time.Now().Unix()),
	}

	log.WithFields(ctx, map[string]interface{}{
		"newUsername": newSecret.Username,
	}).Debug("Creating New User")

	newSecret.Password, err = securePassword()
	if err != nil {
		return err
	}

	newSecret.URL = newSecret.buildURLForDB(dbName)

	// CREATE USER == CREATE ROLE WITH LOGIN
	// Note the driver can't take these as parameters apparently.
	_, err = rootConn.ExecContext(ctx, fmt.Sprintf(`CREATE USER %s PASSWORD '%s' IN ROLE %s`, newSecret.Username, newSecret.Password, newSecret.DBName))
	if err != nil {
		return fmt.Errorf("Create User Role: %w", err)
	}

	jsonBytes, err := json.Marshal(newSecret)
	if err != nil {
		return err
	}

	log.WithFields(ctx, map[string]interface{}{
		"newUsername": newSecret.Username,
		"secretARN":   msg.SecretArn,
	}).Debug("Storing New User Credentials")

	clients, err := d.Clients.Clients(ctx)
	if err != nil {
		return err
	}

	_, err = clients.SecretsManager.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		// ARN or Name
		SecretId:     aws.String(msg.SecretArn),
		SecretString: aws.String(string(jsonBytes)),
	})
	if err != nil {
		return fmt.Errorf("Storing new secret value (%s) failed. The user was still created: %w", msg.SecretArn, err)
	}

	return nil
}

func securePassword() (string, error) {
	b := make([]byte, 33)
	if n, err := rand.Read(b); err != nil {
		return "", err
	} else if n != 33 {
		return "", fmt.Errorf("read %d/33 bytes in secret password", n)
	}
	return hex.EncodeToString(b), nil

}

var reSafeExtensionName = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
