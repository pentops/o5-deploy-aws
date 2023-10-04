package cf

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
	cfsecretsmanager "github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	sq "github.com/elgris/sqrl"
	"github.com/lib/pq"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"gopkg.daemonl.com/sqrlx"
)

type PostgresDefinition struct {
	Secret                  *Resource[*cfsecretsmanager.Secret]
	Databse                 *application_pb.Database
	Postgres                *application_pb.Database_Postgres
	MigrationTaskOutputName *string
}

func (d *Deployer) migrateData(ctx context.Context, stackName string, template *Template, rotateExisting bool) error {

	// TODO: Make this lazy or pre check if it is required.
	remoteStack, err := d.getOneStack(ctx, stackName)
	if err != nil {
		return err
	}
	if remoteStack == nil {
		return errors.New("stack not found")
	}

	for _, db := range template.postgresDatabases {
		ctx := log.WithFields(ctx, map[string]interface{}{
			"database":    db.Databse.Name,
			"serverGroup": db.Postgres.ServerGroup,
		})
		log.Debug(ctx, "Upsert Database")
		if err := d.upsertPostgresDatabase(ctx, db, rotateExisting); err != nil {
			return err
		}

		if db.MigrationTaskOutputName != nil {
			var migrationTaskARN string
			for _, output := range remoteStack.Outputs {
				if *db.MigrationTaskOutputName == *output.OutputKey {
					migrationTaskARN = *output.OutputValue
					break
				}
			}
			if migrationTaskARN == "" {
				return fmt.Errorf("migration task output %q not found", *db.MigrationTaskOutputName)
			}

			if err := d.runMigrationTask(ctx, migrationTaskARN); err != nil {
				return err
			}
		}

	}

	log.WithField(ctx, "count", len(template.postgresDatabases)).Info("Migrated Postgres Databases")

	return nil

}

func (d *Deployer) runMigrationTask(ctx context.Context, taskARN string) error {
	task, err := d.DeployerClients.ECS.RunTask(ctx, &ecs.RunTaskInput{
		TaskDefinition: String(taskARN),
		Cluster:        String(d.AWS.EcsClusterName),
		Count:          aws.Int32(1),
	})
	if err != nil {
		return err
	}

	for {
		state, err := d.DeployerClients.ECS.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Tasks:   []string{*task.Tasks[0].TaskArn},
			Cluster: String(d.AWS.EcsClusterName),
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

		fmt.Printf("TASK: %s\n", jsonFormat(task))

		if len(state.Tasks[0].Containers) != 1 {
			return fmt.Errorf("expected 1 container, got %d", len(state.Tasks[0].Containers))
		}
		container := state.Tasks[0].Containers[0]
		if container.ExitCode == nil {
			return fmt.Errorf("task stopped with no exit code: %s", *container.Reason)
		}
		if *container.ExitCode != 0 {
			return fmt.Errorf("exit code was %d", *container.ExitCode)
		}
		return nil
	}

}

func jsonFormat(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
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
	secretName := fmt.Sprintf("/%s/global/rds/%s/root", d.Environment.FullName, serverGroup)
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

func (d *Deployer) upsertPostgresDatabase(ctx context.Context, spec *PostgresDefinition, rotateExisting bool) error {

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
		SecretId:     spec.Secret.Resource.Name,
		SecretString: aws.String(string(jsonBytes)),
	})
	if err != nil {
		return fmt.Errorf("Storing new secret value (%s) failed. The user was still created: %w", spec.Secret.Name, err)
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
