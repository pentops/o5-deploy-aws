package appbuilder

import (
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/environment/v1/environment_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib/cftest"
	"github.com/stretchr/testify/assert"
)

type testBuilder struct {
	input    AppInput
	rdsHosts RDSHostMap
}

func NewTestBuilder() *testBuilder {
	app := &application_pb.Application{
		Name: "app1",
	}
	rdsHosts := RDSHostMap{}
	return &testBuilder{
		rdsHosts: rdsHosts,
		input: AppInput{
			Application: app,
			RDSHosts:    rdsHosts,
			VersionTag:  "version1",
		},
	}
}

func (tb *testBuilder) Build(t testing.TB) *BuiltApplication {
	out, err := BuildApplication(tb.input)
	if err != nil {
		t.Fatal(err.Error())
	}
	return out
}

func (tb *testBuilder) AddDatabase(db *application_pb.Database) {
	tb.input.Application.Databases = append(tb.input.Application.Databases, db)
}

func (tb *testBuilder) AddRuntime(rt *application_pb.Runtime) {
	tb.input.Application.Runtimes = append(tb.input.Application.Runtimes, rt)
}

func (tb *testBuilder) AddBlobstore(store *application_pb.Blobstore) {
	tb.input.Application.Blobstores = append(tb.input.Application.Blobstores, store)
}

func (tb *testBuilder) ClusterRDSHost(name string, host *RDSHost) {
	tb.rdsHosts[name] = host
}

func (tb *testBuilder) BuildAndAssert(t *testing.T) *cftest.TemplateAsserter {
	out := tb.Build(t)
	return cftest.NewTemplateAsserter(&cflib.BuiltTemplate{
		Template:   out.Template,
		Parameters: out.Parameters,
	})
}

type containerAssert struct {
	container ecs.TaskDefinition_ContainerDefinition
}

func (ca *containerAssert) GetEnv(name string) *ecs.TaskDefinition_KeyValuePair {
	for _, e := range ca.container.Environment {
		if *e.Name == name {
			return &e
		}
	}
	return nil
}

func (ca *containerAssert) MustGetEnv(t testing.TB, name string) *ecs.TaskDefinition_KeyValuePair {
	e := ca.GetEnv(name)
	if e == nil {
		t.Fatalf("env %s not found", name)
	}
	return e
}

func (ca *containerAssert) GetSecret(name string) *ecs.TaskDefinition_Secret {
	for _, s := range ca.container.Secrets {
		if s.Name == name {
			return &s
		}
	}
	return nil
}

func (ca *containerAssert) MustGetSecret(t testing.TB, name string) *ecs.TaskDefinition_Secret {
	s := ca.GetSecret(name)
	if s == nil {
		t.Fatalf("secret %s not found", name)
	}
	return s
}

type standardRuntimeAssert struct {
	main    *containerAssert
	sidecar *containerAssert
}

func newStandardRuntimeAssert(t testing.TB, aa *cftest.TemplateAsserter) *standardRuntimeAssert {
	taskDef := &ecs.TaskDefinition{}
	aa.GetResource(t, "main", taskDef)
	sra := &standardRuntimeAssert{}
	for _, c := range taskDef.ContainerDefinitions {
		if c.Name == "main" {
			sra.main = &containerAssert{container: c}
		} else if c.Name == O5SidecarContainerName {
			sra.sidecar = &containerAssert{container: c}
		} else {
			t.Fatalf("unexpected container %s", c.Name)
		}
	}
	return sra
}

type pgTestCase struct {
	builder *testBuilder
	db      *application_pb.Database
	pg      *application_pb.Database_Postgres
	runtime *application_pb.Runtime
	rdsHost *RDSHost
}

func newPGTestCase() *pgTestCase {

	pg := &application_pb.Database_Postgres{
		DbName:      "foo",
		ServerGroup: "group",
		RunOutbox:   false,
	}
	db := &application_pb.Database{
		Name: "db1",
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
						DatabaseName: "db1",
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

func runPGTestCase(t *testing.T, tc *pgTestCase) *standardRuntimeAssert {
	aa := tc.builder.BuildAndAssert(t)
	rr := newStandardRuntimeAssert(t, aa)
	return rr
}

func TestDatabaseCases(t *testing.T) {

	type wantSecret struct {
		dbName     string
		wantOutbox bool
	}
	assertSecretCase := func(t *testing.T, tc *pgTestCase, sc *wantSecret) {
		rr := runPGTestCase(t, tc)

		wantSecretRef := tSecretRef(cflib.CleanParameterName("Database", sc.dbName), "dburl")

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

		outbox := rr.sidecar.MustGetSecret(t, "POSTGRES_OUTBOX")
		assert.NotEmpty(t, outbox.ValueFrom, "Expected outbox secret value to be set")

		assertFunctionEqual(t, wantSecretRef, outbox.ValueFrom, "Sidecar Outbox secret")
	}

	t.Run("SimplestSecret", func(t *testing.T) {
		tc := newPGTestCase()
		assertSecretCase(t, tc, &wantSecret{
			dbName:     "db1",
			wantOutbox: false,
		})

	})

	t.Run("OutboxSecret", func(t *testing.T) {
		tc := newPGTestCase()
		tc.pg.RunOutbox = true

		assertSecretCase(t, tc, &wantSecret{
			dbName:     "db1",
			wantOutbox: true,
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
