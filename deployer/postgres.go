package deployer

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sq "github.com/elgris/sqrl"
	"github.com/lib/pq"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"gopkg.daemonl.com/sqrlx"
)

func (d *Deployer) migrateData(ctx context.Context, stackName string, template *app.BuiltApplication, rotateExisting bool) error {

	// TODO: Make this lazy or pre check if it is required.
	remoteStack, err := d.getOneStack(ctx, stackName)
	if err != nil {
		return err
	}
	if remoteStack == nil {
		return errors.New("stack not found")
	}

	for _, db := range template.PostgresDatabases {
		ctx := log.WithFields(ctx, map[string]interface{}{
			"database":    db.Databse.Name,
			"serverGroup": db.Postgres.ServerGroup,
		})
		log.Debug(ctx, "Upsert Database")
		var migrationTaskARN string
		var secretARN string
		for _, output := range remoteStack.Outputs {
			if *db.MigrationTaskOutputName == *output.OutputKey {
				migrationTaskARN = *output.OutputValue
			}
			if *db.SecretOutputName == *output.OutputKey {
				secretARN = *output.OutputValue
			}
		}

		if err := d.upsertPostgresDatabase(ctx, db, secretARN, rotateExisting); err != nil {
			return err
		}

		if db.MigrationTaskOutputName != nil {

			if migrationTaskARN == "" {
				return fmt.Errorf("migration task output %q not found", *db.MigrationTaskOutputName)
			}

			if err := d.runMigrationTask(ctx, migrationTaskARN); err != nil {
				return err
			}

			// This runs both before and after migration
			if err := d.fixPostgresOwnership(ctx, db); err != nil {
				return err
			}
		}

	}

	return nil

}

func (d *Deployer) runMigrationTask(ctx context.Context, taskARN string) error {
	task, err := d.DeployerClients.ECS.RunTask(ctx, &ecs.RunTaskInput{
		TaskDefinition: aws.String(taskARN),
		Cluster:        aws.String(d.AWS.EcsClusterName),
		Count:          aws.Int32(1),
	})
	if err != nil {
		return err
	}

	for {
		state, err := d.DeployerClients.ECS.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Tasks:   []string{*task.Tasks[0].TaskArn},
			Cluster: aws.String(d.AWS.EcsClusterName),
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

func (d *Deployer) rootPostgresCredentials(ctx context.Context, serverGroup string) (*DBSecret, error) {
	// "/${var.env_name}/global/rds/${var.name}/root" from TF

	var secretName string
	for _, host := range d.AWS.RdsHosts {
		if host.ServerGroup == serverGroup {
			secretName = host.SecretName
			break
		}
	}
	if secretName == "" {
		return nil, fmt.Errorf("no host found for server group %q", serverGroup)
	}

	res, err := d.DeployerClients.SecretsManager.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", secretName, err)
	}

	secretVal := &DBSecret{}
	if err := json.Unmarshal([]byte(*res.SecretString), secretVal); err != nil {
		return nil, err
	}

	return secretVal, nil
}

func (d *Deployer) fixPostgresOwnership(ctx context.Context, spec *app.PostgresDefinition) error {

	log.Info(ctx, "Fix object ownership")
	dbName := spec.Databse.Name
	if spec.Postgres.DbName != "" {
		dbName = spec.Postgres.DbName
	}
	ownerName := dbName

	rootSecret, err := d.rootPostgresCredentials(ctx, spec.Postgres.ServerGroup)
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

func (d *Deployer) upsertPostgresDatabase(ctx context.Context, spec *app.PostgresDefinition, secretARN string, rotateExisting bool) error {

	rootSecret, err := d.rootPostgresCredentials(ctx, spec.Postgres.ServerGroup)
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

	dbName := spec.Databse.Name
	if spec.Postgres.DbName != "" {
		dbName = spec.Postgres.DbName
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
		return fmt.Errorf("more than one DB matched %s:%s", spec.Postgres.ServerGroup, dbName)
	} else if count == 0 {
		_, err = db.ExecRaw(ctx, fmt.Sprintf(`CREATE ROLE %s`, dbName))
		if err != nil {
			pqErr := pq.Error{}
			if !errors.As(err, &pqErr) {
				return err
			}

			return fmt.Errorf("PQ ERR: %+v", pqErr)
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

	if err := d.fixPostgresOwnership(ctx, spec); err != nil {
		return err
	}

	if count == 1 && !rotateExisting {
		return nil
	}

	if len(spec.Postgres.DbExtensions) > 0 {
		log.WithFields(ctx, map[string]interface{}{
			"count": len(spec.Postgres.DbExtensions),
		}).Debug("Adding Extensions")
		if err := func() error {
			superuserURL := rootSecret.buildURLForDB(dbName)
			superuserConn, err := sql.Open("postgres", superuserURL)
			if err != nil {
				return err
			}
			defer superuserConn.Close()

			for _, ext := range spec.Postgres.DbExtensions {
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
		"secretName":  spec.Secret.Resource.Name,
	}).Debug("Storing New User Credentials")

	_, err = d.DeployerClients.SecretsManager.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		// ARN or Name
		SecretId:     aws.String(secretARN),
		SecretString: aws.String(string(jsonBytes)),
	})
	if err != nil {
		return fmt.Errorf("Storing new secret value (%s) failed. The user was still created: %w", spec.Secret.Name(), err)
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
