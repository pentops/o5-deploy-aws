package app

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/pentops/o5-deploy-aws/cf"
	"github.com/pentops/o5-deploy-aws/cf/cftest"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/stretchr/testify/assert"
)

func TestBasicMap(t *testing.T) {
	app := &application_pb.Application{
		Name: "app1",
		Runtimes: []*application_pb.Runtime{{
			Name: "main",
			Containers: []*application_pb.Container{{
				Name: "main",
				Source: &application_pb.Container_Image_{
					Image: &application_pb.Container_Image{
						Tag:  cf.String("latest"),
						Name: "foobar",
					},
				},
			}},
		}},
	}

	out, err := BuildApplication(app, "version1")
	if err != nil {
		t.Fatal(err.Error())
	}

	template := out.Template
	yy, err := template.YAML()
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(string(yy))
}

func TestDirectPortAccess(t *testing.T) {
	app := &application_pb.Application{
		Name: "app1",
		Runtimes: []*application_pb.Runtime{{
			Name: "main",
			Routes: []*application_pb.Route{{
				Prefix:          "/test",
				BypassIngress:   true,
				TargetContainer: "main",
				Port:            1234,
				Protocol:        application_pb.RouteProtocol_GRPC,
			}},
			Containers: []*application_pb.Container{{
				Name: "main",
				Source: &application_pb.Container_Image_{
					Image: &application_pb.Container_Image{
						Tag:  cf.String("latest"),
						Name: "foobar",
					},
				},
			}},
		}},
	}

	aa := testbuild(t, app)

	taskDef := &ecs.TaskDefinition{}
	aa.GetResource(t, "main", taskDef)
	if len(taskDef.ContainerDefinitions[0].PortMappings) != 1 {
		t.Fatalf("expected one port mapping, got %d", len(taskDef.ContainerDefinitions[0].PortMappings))
	}

	t.Logf("ports: %v", taskDef.ContainerDefinitions[0].PortMappings)
}

func testbuild(t *testing.T, app *application_pb.Application) *cftest.TemplateAsserter {
	out, err := BuildApplication(app, "version1")
	if err != nil {
		t.Fatal(err.Error())
	}
	return cftest.NewTemplateAsserter(&cf.BuiltTemplate{
		Template:   out.Template,
		Parameters: out.Parameters,
	})
}

func TestIndirectPortAccess(t *testing.T) {
	app := &application_pb.Application{
		Name: "app1",
		Runtimes: []*application_pb.Runtime{{
			Name: "main",
			Routes: []*application_pb.Route{{
				Prefix:          "/test",
				BypassIngress:   false,
				TargetContainer: "main",
				Port:            1234,
				Protocol:        application_pb.RouteProtocol_GRPC,
			}},
			Containers: []*application_pb.Container{{
				Name: "main",
				Source: &application_pb.Container_Image_{
					Image: &application_pb.Container_Image{
						Tag:  cf.String("latest"),
						Name: "foobar",
					},
				},
			}},
		}},
	}

	aa := testbuild(t, app)

	taskDef := &ecs.TaskDefinition{}
	aa.GetResource(t, "main", taskDef)
	t.Logf("ports: %v", taskDef.ContainerDefinitions[1].PortMappings)
}
func TestRuntime(t *testing.T) {
	global := globalData{
		appName: "Test",
	}

	rs, err := NewRuntimeService(global, &application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cf.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(rs.Containers) != 1 {
		t.Fatalf("expected 1 container definition, got %d", len(rs.Containers))
	}

	{
		imageRaw := rs.Containers[0].Container.Image

		join := &Join{}
		decode(t, imageRaw, join)
		assert.Equal(t, "foobar", join.Vals[2])
		assert.Equal(t, "latest", join.Vals[4])

		ref := &Ref{}
		decode(t, join.Vals[0], ref)
		assert.Equal(t, "ECSRepo", ref.Ref)
	}

}

func TestSidecarConfigRuntime(t *testing.T) {
	global := globalData{
		appName: "Test",
	}

	rs, err := NewRuntimeService(global, &application_pb.Runtime{
		Name: "main",
		WorkerConfig: &application_pb.WorkerConfig{
			ReplayChance:     1,
			DeadletterChance: 2,
		},
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cf.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(rs.Containers) != 1 {
		t.Fatalf("expected 1 container definition, got %d", len(rs.Containers))
	}
	sidecarConfigBits := 0
	for i := range rs.AdapterContainer.Environment {
		if *rs.AdapterContainer.Environment[i].Name == "RESEND_CHANCE" && *rs.AdapterContainer.Environment[i].Value == "1" {
			sidecarConfigBits += 1
		}
		if *rs.AdapterContainer.Environment[i].Name == "DEADLETTER_CHANCE" && *rs.AdapterContainer.Environment[i].Value == "2" {
			sidecarConfigBits += 1
		}
	}
	if sidecarConfigBits != 2 {
		t.Fatalf("Expected sidecar chance configs in task def, did not find them or values were incorrect")
	}
}

func TestSidecarConfigNotPresentRuntime(t *testing.T) {
	global := globalData{
		appName: "Test",
	}

	rs, err := NewRuntimeService(global, &application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cf.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(rs.Containers) != 1 {
		t.Fatalf("expected 1 container definition, got %d", len(rs.Containers))
	}
	sidecarConfigBits := 0
	for i := range rs.AdapterContainer.Environment {
		if *rs.AdapterContainer.Environment[i].Name == "RESEND_CHANCE" && *rs.AdapterContainer.Environment[i].Value == "1" {
			sidecarConfigBits += 1
		}
		if *rs.AdapterContainer.Environment[i].Name == "DEADLETTER_CHANCE" && *rs.AdapterContainer.Environment[i].Value == "2" {
			sidecarConfigBits += 1
		}
	}
	if sidecarConfigBits != 0 {
		t.Fatalf("Did not expect sidecar chance configs in task def, but found some")
	}
}

func TestBlobstore(t *testing.T) {

	type result struct {
		bucket *cf.Resource[*s3.Bucket]
		role   *cf.Resource[*iam.Role]
		envVar *ecs.TaskDefinition_KeyValuePair
	}

	run := func(t *testing.T, def *application_pb.Blobstore) result {
		app := &application_pb.Application{
			Name:       "app1",
			Blobstores: []*application_pb.Blobstore{def},
			Runtimes: []*application_pb.Runtime{{
				Name: "main",
				Containers: []*application_pb.Container{{
					Name: "main",
					Source: &application_pb.Container_Image_{
						Image: &application_pb.Container_Image{
							Tag:  cf.String("latest"),
							Name: "foobar",
						},
					},
					EnvVars: []*application_pb.EnvironmentVariable{{
						Name: "BUCKET",
						Spec: &application_pb.EnvironmentVariable_Blobstore{
							Blobstore: &application_pb.BlobstoreEnvVar{
								Name:    "bucket",
								SubPath: cf.String("subpath"),
								Format: &application_pb.BlobstoreEnvVar_S3Direct{
									S3Direct: true,
								},
							},
						},
					}},
				}},
			}},
		}

		out, err := BuildApplication(app, "version1")
		if err != nil {
			t.Fatal(err.Error())
		}

		rr := result{}

		bucketResource, ok := out.Template.Resources["S3BucketBucket"].(*cf.Resource[*s3.Bucket])
		if ok {
			rr.bucket = bucketResource
		}

		roleResource, ok := out.Template.Resources["IAMRoleMainAssume"].(*cf.Resource[*iam.Role])
		if !ok || roleResource == nil {
			t.Fatalf("role resource not found")
		}

		rr.role = roleResource

		taskDef, ok := out.Template.Resources["ECSTaskDefinitionMain"].(*cf.Resource[*ecs.TaskDefinition])
		if !ok || taskDef == nil {
			t.Fatalf("task definition not found")
		}
		envVars := taskDef.Resource.ContainerDefinitions[0].Environment
		for _, envVar := range envVars {
			envVar := envVar
			if *envVar.Name == "BUCKET" {
				rr.envVar = &envVar
			}
		}

		return rr

	}

	t.Run("Standard", func(t *testing.T) {
		rr := run(t, &application_pb.Blobstore{
			Name: "bucket",
		})

		if rr.bucket == nil {
			t.Fatalf("bucket not in template")
		}

		rwPolicy := findPolicy(t, rr.role.Resource, "s3-readwrite")
		if rwPolicy == nil {
			t.Fatalf("policy not found")
		}

		getAtt := &GetAtt{}
		decode(t, rwPolicy.Statement[0].Resource[0], getAtt)
		assert.Equal(t, "S3BucketBucket", getAtt.Resource)
		assert.Equal(t, "Arn", getAtt.Attribute)

		if rr.envVar == nil {
			t.Fatalf("env var not found")
		}
		want := cloudformation.Join("", []string{
			"s3://",
			*rr.bucket.Resource.BucketName,
			"/",
			"subpath",
		})
		assert.Equal(t, want, *rr.envVar.Value)
	})

	t.Run("App Ref", func(t *testing.T) {
		rr := run(t, &application_pb.Blobstore{
			Name: "bucket",
			Ref: &application_pb.BlobstoreRef{
				Source: &application_pb.BlobstoreRef_Application{
					Application: "other",
				},
				ReadPermission:  true,
				WritePermission: true,
			},
		})

		if rr.bucket != nil {
			t.Fatalf("bucket present in template for a ref")
		}

		rwPolicy := findPolicy(t, rr.role.Resource, "s3-readwrite")
		if rwPolicy == nil {
			t.Fatalf("policy not found")
		}

		bucketName := cloudformation.Join(".", []string{
			"bucket",
			"other",
			cloudformation.Ref(EnvNameParameter),
			cloudformation.Ref(AWSRegionParameter),
			cloudformation.Ref(S3BucketNamespaceParameter),
		})

		bucketARN := cloudformation.Join(":", []string{
			"arn:aws:s3",
			"", //cloudformation.Ref(AWSRegionParameter),
			"", //cloudformation.Ref(AWSAccountIDParameter),
			bucketName,
		})

		assertFunctionEqual(t, rwPolicy.Statement[0].Resource[0], bucketARN)

		if rr.envVar == nil {
			t.Fatalf("env var not found")
		}
		rootJoin := decodeJoin(t, *rr.envVar.Value)
		t.Log(rootJoin)
		assert.Equal(t, "s3://", rootJoin.Vals[0])
		assert.Equal(t, "/", rootJoin.Vals[2])
		assert.Equal(t, "subpath", rootJoin.Vals[3])
		ref := rootJoin.Vals[1]
		assert.Equal(t, bucketName, ref)
	})

}

func assertFunctionEqual(t testing.TB, want, got string) {
	t.Helper()
	assert.Equal(t, want, got)
}

func findPolicy(t testing.TB, roleResource *iam.Role, name string) *PolicyDocument {
	t.Helper()
	for _, policy := range roleResource.Policies {
		join := decodeJoin(t, policy.PolicyName)
		if join.Vals[3] == name {
			doc, ok := policy.PolicyDocument.(PolicyDocument)
			if !ok {
				t.Fatalf("policy document is a %T", policy.PolicyDocument)
			}
			return &doc
		}
	}
	return nil
}

func decodeJoin(t testing.TB, s string) *Join {
	t.Helper()
	join := &Join{}
	decode(t, s, join)
	return join
}

func decode(t testing.TB, s string, into interface{}) {
	t.Helper()
	val, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("decoded: '%s'", string(val))
	if err := json.Unmarshal(val, into); err != nil {
		t.Fatalf("decoding %s: %s", val, err.Error())
	}
}

type Join struct {
	Delim string
	Vals  []string
}

func (j *Join) UnmarshalJSON(b []byte) error {
	raw := struct {
		Vals []interface{} `json:"Fn::Join"`
	}{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw.Vals) != 2 {
		return nil
	}
	j.Delim = raw.Vals[0].(string)
	for _, v := range raw.Vals[1].([]interface{}) {
		j.Vals = append(j.Vals, v.(string))
	}

	return nil
}

type GetAtt struct {
	Resource  string
	Attribute string
}

func (g *GetAtt) UnmarshalJSON(b []byte) error {
	raw := struct {
		Vals []string `json:"Fn::GetAtt"`
	}{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if len(raw.Vals) != 2 {
		return nil
	}
	g.Resource = raw.Vals[0]
	g.Attribute = raw.Vals[1]
	return nil
}

type Ref struct {
	Ref string `json:"Ref"`
}
