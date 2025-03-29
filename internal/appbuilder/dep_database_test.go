package appbuilder

import (
	"fmt"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"github.com/stretchr/testify/assert"
)

type pgTestCase struct {
	builder *testBuilder
	db      *application_pb.Database
	pg      *application_pb.Database_Postgres
	runtime *application_pb.Runtime
	rdsHost *RDSHost
}

func newPGTestCase() *pgTestCase {
	pg := &application_pb.Database_Postgres{
		DbNameSuffix: "suffix",
		ServerGroup:  "group",
		RunOutbox:    false,
	}

	db := &application_pb.Database{
		Name: "appkey",
		Engine: &application_pb.Database_Postgres_{
			Postgres: pg,
		},
	}

	runtime := &application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Name: "foobar",
				},
			},
			EnvVars: []*application_pb.EnvironmentVariable{{
				Name: "DATABASE_URL",
				Spec: &application_pb.EnvironmentVariable_Database{
					Database: &application_pb.DatabaseEnvVar{
						DatabaseName: "appkey",
					},
				},
			}},
		}},
	}

	rdsHost := &RDSHost{
		AuthType: environment_pb.RDSAuth_SecretsManager,
	}

	tc := &pgTestCase{
		db:      db,
		pg:      pg,
		runtime: runtime,
		rdsHost: rdsHost,
	}
	tb := NewTestBuilder()
	tb.AddRuntime(tc.runtime)
	tb.AddDatabase(tc.db)
	tb.ClusterRDSHost("group", tc.rdsHost)
	tc.builder = tb
	return tc

}

func TestDatabaseCases(t *testing.T) {
	type wantSecret struct {
		appKey        string
		wantOutbox    bool
		wantDelayable bool
		wantMigrate   bool
	}

	assertSecretCase := func(t *testing.T, tc *pgTestCase, sc *wantSecret) {
		t.Helper()
		aa := tc.builder.BuildAndAssert(t)
		rr := aa.StandardRuntimeAssert(t)

		wantSecretRef := tSecretRef(cflib.CleanParameterName("Database", sc.appKey), "dburl")

		env := rr.main.GetEnv("DATABASE_URL")
		if env != nil {
			t.Errorf("expected DATABASE_URL to be a secret, got env var")
		}

		secret := rr.main.MustGetSecret(t, "DATABASE_URL")
		assertFunctionEqual(t, wantSecretRef, secret.ValueFrom, "Main container secret")

		if !sc.wantOutbox {
			if rr.sidecar != nil {
				t.Fatalf("unexpected sidecar container")
			}
			return
		}

		if rr.sidecar == nil {
			t.Fatalf("expected sidecar container")
		}

		creds := rr.sidecar.MustGetSecret(t, "DB_CREDS_APPKEY")
		assert.NotEmpty(t, creds.ValueFrom, "Expected outbox secret value to be set")
		assertFunctionEqual(t, wantSecretRef, creds.ValueFrom, "Sidecar Outbox secret")

		outbox := rr.sidecar.MustGetEnv(t, "POSTGRES_OUTBOX")
		assert.Equal(t, "APPKEY", *outbox.Value, "Expected outbox env var to be set to the name of the DB")

		rr.sidecar.MustNotHaveEnvOrSecret(t, "POSTGRES_IAM_PROXY")
	}

	assertIAMRuntime := func(t *testing.T, rr *standardRuntimeAssert, want *wantSecret) {
		env := rr.main.GetEnv("DATABASE_URL")
		secret := rr.main.GetSecret("DATABASE_URL")
		if env == nil && secret == nil {
			t.Errorf("DATABASE_URL not set for main container")
		} else if secret != nil && env == nil {
			t.Errorf("expected DATABASE_URL to be an env var, was secret")
		} else if secret != nil && env != nil {
			t.Errorf("expected DATABASE_URL to be an env var, got both")
		}

		if env.Value == nil {
			t.Errorf("expected DATABASE_URL to have a value, was nil")
		}

		rr.WithSidecarContainer(t, func(t testing.TB, sidecar *containerAssert) {
			proxy := sidecar.MustGetEnv(t, "POSTGRES_IAM_PROXY")
			assert.Equal(t, "APPKEY", *proxy.Value, "Expected proxy env var to be set to the name of the DB")

			if want.wantOutbox {
				outbox := sidecar.MustGetEnv(t, "POSTGRES_OUTBOX")
				assert.Equal(t, "APPKEY", *outbox.Value, "Expected outbox env var to be set to the name of the DB")
			} else {
				sidecar.MustNotHaveEnvOrSecret(t, "POSTGRES_OUTBOX")
			}

			if want.wantDelayable {
				outbox := sidecar.MustGetEnv(t, "POSTGRES_OUTBOX_DELAYABLE")
				assert.Equal(t, "true", *outbox.Value, "Expected outbox env var to be set to the name of the DB")
			} else {
				sidecar.MustNotHaveEnvOrSecret(t, "POSTGRES_OUTBOX_DELAYABLE")
			}

			creds := sidecar.MustGetEnv(t, "DB_CREDS_APPKEY")
			assert.NotEmpty(t, creds.Value, "Expected outbox env value to be set")

			wantEndpointRef := cloudformation.Ref("DatabaseParam" + cflib.CleanParameterName("appkey") + "JSON")
			assertFunctionEqual(t, wantEndpointRef, *creds.Value, "IAM Proxy JSON")
		})

		rr.WithPolicy(t, PolicyNameRDSConnect, func(t testing.TB, policy *PolicyDocument) {
			stmt := policy.Statement[0]
			resource := stmt.Resource[0]
			got := funcToPlaceholders(t, resource)
			identifierParam := rr.parent.GetParameter(t, &awsdeployer_pb.ParameterSourceType_Aurora{
				ServerGroup: "group",
				AppKey:      "appkey",
				Part:        awsdeployer_pb.ParameterSourceType_Aurora_PART_IDENTIFIER,
			})
			dbNameParam := rr.parent.GetParameter(t, &awsdeployer_pb.ParameterSourceType_Aurora{
				ServerGroup: "group",
				AppKey:      "appkey",
				Part:        awsdeployer_pb.ParameterSourceType_Aurora_PART_DBNAME,
			})
			want := fmt.Sprintf("arn:aws:rds-db:{AWS::Region}:{AWS::AccountId}:dbuser:{%s}/{%s}",
				identifierParam.Name,
				dbNameParam.Name,
			)
			assert.Equal(t, want, got)
		})
	}

	assertIAMCase := func(t *testing.T, tc *pgTestCase, want *wantSecret) {
		t.Helper()
		aa := tc.builder.BuildAndAssert(t)
		t.Run("Main Task", func(t *testing.T) {
			mainTask := aa.StandardRuntimeAssert(t)
			assertIAMRuntime(t, mainTask, want)
		})

		db := aa.GetDatabase(t, "appkey")

		if !want.wantMigrate {
			if db.MigrationTaskOutputName != nil {
				t.Errorf("unexpected migration task")
			}
			return
		}

		t.Run("Migrate Task", func(t *testing.T) {
			if db.MigrationTaskOutputName == nil {
				t.Fatalf("expected migration task output name to be set")
			}
			outputVal := aa.GetOutputValue(t, *db.MigrationTaskOutputName)

			migrateTask := aa.SingleContainerTaskDef(t, "!!"+outputVal, "migrate")
			assertIAMRuntime(t, migrateTask, &wantSecret{
				appKey:     "db1",
				wantOutbox: false,
			})
		})
	}

	t.Run("SimplestSecret", func(t *testing.T) {
		tc := newPGTestCase()
		assertSecretCase(t, tc, &wantSecret{
			appKey:     "appkey",
			wantOutbox: false,
		})
	})

	t.Run("OutboxSecret", func(t *testing.T) {
		tc := newPGTestCase()
		tc.pg.RunOutbox = true

		assertSecretCase(t, tc, &wantSecret{
			appKey:     "appkey",
			wantOutbox: true,
		})
	})

	t.Run("SimpleProxy", func(t *testing.T) {
		tc := newPGTestCase()
		tc.rdsHost.AuthType = environment_pb.RDSAuth_Iam

		assertIAMCase(t, tc, &wantSecret{
			appKey:     "appkey",
			wantOutbox: false,
		})
	})

	t.Run("OutboxProxy", func(t *testing.T) {
		tc := newPGTestCase()
		tc.rdsHost.AuthType = environment_pb.RDSAuth_Iam
		tc.pg.RunOutbox = true

		assertIAMCase(t, tc, &wantSecret{
			appKey:     "appkey",
			wantOutbox: true,
		})
	})

	t.Run("OutboxProxy", func(t *testing.T) {
		tc := newPGTestCase()
		tc.rdsHost.AuthType = environment_pb.RDSAuth_Iam
		tc.pg.RunOutbox = true
		tc.pg.OutboxDelayable = true

		assertIAMCase(t, tc, &wantSecret{
			appKey:        "appkey",
			wantOutbox:    true,
			wantDelayable: true,
		})
	})

	t.Run("MigrateProxy", func(t *testing.T) {
		tc := newPGTestCase()
		tc.rdsHost.AuthType = environment_pb.RDSAuth_Iam
		tc.pg.MigrateContainer = &application_pb.Container{
			Source:  tc.runtime.Containers[0].Source,
			Command: []string{"migrate"},
			EnvVars: tc.runtime.Containers[0].EnvVars,
		}

		assertIAMCase(t, tc, &wantSecret{
			appKey:      "appkey",
			wantOutbox:  false,
			wantMigrate: true,
		})
	})

	t.Run("MigrateAndOutboxProxy", func(t *testing.T) {
		tc := newPGTestCase()
		tc.rdsHost.AuthType = environment_pb.RDSAuth_Iam
		tc.pg.RunOutbox = true
		tc.pg.MigrateContainer = &application_pb.Container{
			Source:  tc.runtime.Containers[0].Source,
			Command: []string{"migrate"},
			EnvVars: tc.runtime.Containers[0].EnvVars,
		}

		assertIAMCase(t, tc, &wantSecret{
			appKey:      "db1",
			wantOutbox:  true,
			wantMigrate: true,
		})
	})
}

func tSecretRef(name string, jsonKey string) string {
	return cloudformation.Join(":", []string{
		cloudformation.Ref("SecretsManagerSecret" + name),
		jsonKey,
		"",
		"",
	})
}
