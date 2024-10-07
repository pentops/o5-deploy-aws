package aws_postgres

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
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	sq "github.com/elgris/sqrl"
	_ "github.com/lib/pq"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/infra/v1/awsinfra_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsinfra/v1/awsinfra_tpb"
	"github.com/pentops/sqrlx.go/sqrlx"
)

type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	UpdateSecret(ctx context.Context, params *secretsmanager.UpdateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.UpdateSecretOutput, error)
	DescribeSecret(ctx context.Context, params *secretsmanager.DescribeSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.DescribeSecretOutput, error)
}

type RDSAuthProvider interface {
	BuildAuthToken(ctx context.Context, dbEndpoint, dbUser string) (string, error)
}

type DBMigrator struct {
	secretsManager SecretsManagerAPI
	creds          RDSAuthProvider
}

func NewDBMigrator(client SecretsManagerAPI, creds RDSAuthProvider) *DBMigrator {
	return &DBMigrator{
		secretsManager: client,
		creds:          creds,
	}
}

func (d *DBMigrator) UpsertPostgresDatabase(ctx context.Context, migrationID string, msg *awsinfra_tpb.UpsertPostgresDatabaseMessage) error {

	connSpec, err := d.buildSpec(ctx, msg.AdminHost)
	if err != nil {
		return err
	}

	didCreate, err := d.upsertPostgresDatabase(ctx, connSpec, msg.Spec)
	if err != nil {
		return err
	}

	if appSecret := msg.AppAccess.GetAppSecret(); appSecret != nil {
		if didCreate || appSecret.RotateCredentials {
			if err := d.buildSecretsUser(ctx, connSpec, msg.Spec.DbName, appSecret); err != nil {
				return err
			}
		}
	}

	if err := d.fixPostgresOwnership(ctx, connSpec, msg.Spec.DbName); err != nil {
		return err
	}
	return nil
}

func (d *DBMigrator) CleanupPostgresDatabase(ctx context.Context, migrationID string, msg *awsinfra_tpb.CleanupPostgresDatabaseMessage) error {
	connSpec, err := d.buildSpec(ctx, msg.AdminHost)
	if err != nil {
		return err
	}
	return d.fixPostgresOwnership(ctx, connSpec, msg.DbName)
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

func (d *DBMigrator) rootPostgresCredentials(ctx context.Context, rootSecretName string) (*DBSecret, error) {
	// "/${var.env_name}/global/rds/${var.name}/root" from TF

	res, err := d.secretsManager.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(rootSecretName),
	})
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", rootSecretName, err)
	}

	secretVal := &DBSecret{}
	if err := json.Unmarshal([]byte(*res.SecretString), secretVal); err != nil {
		return nil, err
	}

	return secretVal, nil
}

type DBSpec interface {
	OpenRoot(context.Context) (*sql.DB, error)
	OpenDBAsRoot(ctx context.Context, dbName string) (*sql.DB, error)

	// NewAppSecret creates the APP side secret for either root connection type.
	NewAppSecret(dbName string) DBSecret
}

type secretsSpec struct {
	rootSecret DBSecret
}

func (s *secretsSpec) OpenRoot(ctx context.Context) (*sql.DB, error) {
	log.WithFields(ctx, map[string]interface{}{
		"hostname": s.rootSecret.Hostname,
		"dbName":   s.rootSecret.DBName,
	}).Debug("Connecting to RDS root db as root user")

	return openDB(ctx, s.rootSecret.URL)
}

func (s *secretsSpec) OpenDBAsRoot(ctx context.Context, dbName string) (*sql.DB, error) {
	log.WithFields(ctx, map[string]interface{}{
		"hostname": s.rootSecret.Hostname,
		"dbName":   s.rootSecret.DBName,
	}).Debug("Connecting to RDS app db as root user")

	return openDB(ctx, s.rootSecret.buildURLForDB(dbName))
}

func (s *secretsSpec) SecretsManagerSpec() (*DBSecret, bool) {
	return &s.rootSecret, true
}

func (s *secretsSpec) NewAppSecret(dbName string) DBSecret {
	newSecret := DBSecret{
		DBName:   dbName,
		Hostname: s.rootSecret.Hostname,
		Username: fmt.Sprintf("%s_%d", dbName, time.Now().Unix()),
	}
	return newSecret
}

type auroraSpec struct {
	dbEndpoint string
	dbPort     int32
	dbUser     string
	dbName     string
	token      string
}

func (a *auroraSpec) buildDSN(dbName string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		a.dbEndpoint, a.dbPort, a.dbUser, a.token, dbName,
	)
}

func (a *auroraSpec) OpenRoot(ctx context.Context) (*sql.DB, error) {
	dsn := a.buildDSN(a.dbName)
	return openDB(ctx, dsn)
}

func (a *auroraSpec) OpenDBAsRoot(ctx context.Context, dbName string) (*sql.DB, error) {
	dsn := a.buildDSN(dbName)
	return openDB(ctx, dsn)
}

func (a *auroraSpec) NewAppSecret(dbName string) DBSecret {
	newSecret := DBSecret{
		DBName:   dbName,
		Hostname: a.dbEndpoint,
		Username: fmt.Sprintf("%s_%d", dbName, time.Now().Unix()),
	}
	return newSecret
}

func (d *DBMigrator) buildSpec(ctx context.Context, spec *awsinfra_pb.RDSHostType) (DBSpec, error) {
	switch spec := spec.Get().(type) {
	case *awsinfra_pb.RDSHostType_Aurora:
		dbEndpoint := spec.Conn.Endpoint
		dbPort := spec.Conn.Port
		dbUser := spec.Conn.DbUser
		dbName := spec.Conn.DbName
		if dbName == "" {
			dbName = dbUser
		}

		authenticationToken, err := d.creds.BuildAuthToken(ctx, dbEndpoint, dbUser)
		if err != nil {
			return nil, err
		}

		return &auroraSpec{
			dbEndpoint: dbEndpoint,
			dbPort:     dbPort,
			dbUser:     dbUser,
			dbName:     dbName,
			token:      authenticationToken,
		}, nil

	case *awsinfra_pb.RDSHostType_SecretsManager:

		secret, err := d.rootPostgresCredentials(ctx, spec.SecretName)
		if err != nil {
			return nil, err
		}

		return &secretsSpec{
			rootSecret: *secret,
		}, nil

	default:
		return nil, fmt.Errorf("unknown spec type %T", spec)
	}
}

func openDB(ctx context.Context, url string) (*sql.DB, error) {
	conn, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	conn.SetMaxOpenConns(1)
	if _, err := conn.ExecContext(ctx, "SET application_name TO 'o5-deployer'"); err != nil {
		conn.Close()
		return nil, err
	}

	return conn, nil
}

func (d *DBMigrator) fixPostgresOwnership(ctx context.Context, spec DBSpec, dbName string) error {
	ownerName := dbName

	log.Info(ctx, "Fix object ownership")
	rootConn, err := spec.OpenDBAsRoot(ctx, dbName)
	if err != nil {
		return err
	}
	defer rootConn.Close()

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
		return fmt.Errorf("error getting relation info for fixing object ownership: %w", err)
	}

	defer rows.Close()

	type object struct {
		name  string
		owner string
		kind  string
	}
	objects := map[string][]object{}
	for rows.Next() {
		o := object{}
		if err := rows.Scan(&o.name, &o.kind, &o.owner); err != nil {
			return fmt.Errorf("error scanning row in relation info: %w", err)
		}
		if o.owner == ownerName {
			continue
		}
		objects[o.kind] = append(objects[o.kind], o)
	}

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

func (d *DBMigrator) upsertPostgresDatabase(ctx context.Context, connSpec DBSpec, spec *awsinfra_pb.RDSCreateSpec) (bool, error) {
	dbName := spec.DbName

	rootConn, err := connSpec.OpenRoot(ctx)
	if err != nil {
		return false, err
	}

	db, err := sqrlx.NewWithCommander(rootConn, sq.Dollar)
	if err != nil {
		return false, err
	}

	var count int

	err = db.SelectRow(ctx, sq.
		Select("coalesce(count(datname), 0)").
		From("pg_catalog.pg_database").
		Where("datname = ?", dbName),
	).Scan(&count)
	if err != nil {
		return false, err
	}

	log.WithFields(ctx, map[string]interface{}{
		"count": count,
	}).Debug("Found DBs")

	if count == 1 {
		return false, nil
	} else if count > 1 {
		return false, fmt.Errorf("more than one DB matched %q", dbName)
	}

	_, err = db.ExecRaw(ctx, fmt.Sprintf(`CREATE ROLE %s`, dbName))
	if err != nil {
		return true, err
	}

	_, err = db.ExecRaw(ctx, fmt.Sprintf(`GRANT %s TO current_user`, dbName))
	if err != nil {
		return true, err
	}

	_, err = db.ExecRaw(ctx, fmt.Sprintf(`CREATE DATABASE %s OWNER %s`, dbName, dbName))
	if err != nil {
		return true, err
	}

	if len(spec.DbExtensions) > 0 {
		log.WithFields(ctx, map[string]interface{}{
			"count": len(spec.DbExtensions),
		}).Debug("Adding Extensions")
		if err := func() error {
			// Note this connects to the database name, not the wider 'postgres'
			// namespace, so is different to the outer rootConn
			superuserConn, err := connSpec.OpenDBAsRoot(ctx, dbName)
			if err != nil {
				return err
			}
			defer superuserConn.Close()

			for _, ext := range spec.DbExtensions {
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
			return true, err
		}
	}

	return true, nil

}

// buildSecretsUser creates a new user alias in the database and stores the credentials in Secrets Manager.
func (d *DBMigrator) buildSecretsUser(ctx context.Context, connSpec DBSpec, dbName string, msg *awsinfra_pb.RDSAppSpecType_SecretsManager) error {
	var err error

	newSecret := connSpec.NewAppSecret(dbName)

	log.WithFields(ctx, map[string]interface{}{
		"newUsername": newSecret.Username,
	}).Debug("Creating New User")

	newSecret.Password, err = securePassword()
	if err != nil {
		return err
	}

	newSecret.URL = newSecret.buildURLForDB(dbName)

	rootConn, err := connSpec.OpenRoot(ctx)
	if err != nil {
		return err
	}
	defer rootConn.Close()

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
		"secretARN":   msg.AppSecretName,
	}).Debug("Storing New User Credentials")

	_, err = d.secretsManager.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		// ARN or Name
		SecretId:     aws.String(msg.AppSecretName),
		SecretString: aws.String(string(jsonBytes)),
	})
	if err != nil {
		return fmt.Errorf("Storing new secret value (%s) failed. The user was still created: %w", msg.AppSecretName, err)
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
