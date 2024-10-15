package appbuilder

import (
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/cloudformation/s3"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"github.com/stretchr/testify/assert"
)

func TestBasicMap(t *testing.T) {
	tb := NewTestBuilder()
	tb.AddRuntime(&application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cflib.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})
	out := tb.Build(t)

	template := out.Template
	yy, err := template.YAML()
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(string(yy))
}

func TestDirectPortAccess(t *testing.T) {
	tb := NewTestBuilder()
	tb.AddRuntime(&application_pb.Runtime{
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
					Tag:  cflib.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})
	aa := tb.BuildAndAssert(t)

	taskDef := &ecs.TaskDefinition{}
	aa.GetResource(t, "main", taskDef)
	if len(taskDef.ContainerDefinitions[0].PortMappings) != 1 {
		t.Fatalf("expected one port mapping, got %d", len(taskDef.ContainerDefinitions[0].PortMappings))
	}

	t.Logf("ports: %v", taskDef.ContainerDefinitions[0].PortMappings)
}

func TestIndirectPortAccess(t *testing.T) {
	tb := NewTestBuilder()
	tb.AddRuntime(&application_pb.Runtime{
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
					Tag:  cflib.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})

	aa := tb.BuildAndAssert(t)

	taskDef := &ecs.TaskDefinition{}
	aa.GetResource(t, "main", taskDef)
	t.Logf("ports: %v", taskDef.ContainerDefinitions[1].PortMappings)
}

func TestRuntime(t *testing.T) {
	tb := NewTestBuilder()
	tb.AddRuntime(&application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cflib.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})
	rs := tb.BuildAndAssert(t).StandardRuntimeAssert(t)

	rs.AssertNoSidecar(t)

	rs.WithMainContainer(t, func(t testing.TB, container *containerAssert) {
		imageRaw := container.container.Image

		join := &Join{}
		decode(t, imageRaw, join)
		assert.Equal(t, "foobar", join.Vals[2])
		assert.Equal(t, "latest", join.Vals[4])

		ref := &Ref{}
		decode(t, join.Vals[0], ref)
		assert.Equal(t, "ECSRepo", ref.Ref)
	})

}

func TestSidecarConfigRuntime(t *testing.T) {
	tb := NewTestBuilder()
	tb.AddRuntime(&application_pb.Runtime{
		Name: "main",
		WorkerConfig: &application_pb.WorkerConfig{
			ReplayChance:     1,
			DeadletterChance: 2,
		},
		Subscriptions: []*application_pb.Subscription{{
			Name: "/foo.v1.Bar/Baz",
		}},
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cflib.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})

	tb.BuildAndAssert(t).
		StandardRuntimeAssert(t).
		WithSidecarContainer(t, func(t testing.TB, container *containerAssert) {
			container.AssertEnvVal(t, "RESEND_CHANCE", "1")
			container.AssertEnvVal(t, "DEADLETTER_CHANCE", "2")
		})
}

func TestSidecarConfigNotPresentRuntime(t *testing.T) {
	tb := NewTestBuilder()
	tb.AddRuntime(&application_pb.Runtime{
		Name: "main",
		Subscriptions: []*application_pb.Subscription{{
			Name: "/foo.v1.Bar/Baz",
		}},
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  cflib.String("latest"),
					Name: "foobar",
				},
			},
		}},
	})

	tb.BuildAndAssert(t).
		StandardRuntimeAssert(t).
		WithSidecarContainer(t, func(t testing.TB, container *containerAssert) {
			container.MustNotHaveEnvOrSecret(t, "RESEND_CHANCE")
			container.MustNotHaveEnvOrSecret(t, "DEADLETTER_CHANCE")
		})
}

func TestBlobstore(t *testing.T) {

	type result struct {
		bucket *cflib.Resource[*s3.Bucket]
		role   *cflib.Resource[*iam.Role]
		envVar *ecs.TaskDefinition_KeyValuePair
	}

	run := func(t *testing.T, def *application_pb.Blobstore) result {
		t.Helper()

		tb := NewTestBuilder()
		tb.AddBlobstore(def)
		tb.AddRuntime(&application_pb.Runtime{
			Name: "main",
			Containers: []*application_pb.Container{{
				Name: "main",
				Source: &application_pb.Container_Image_{
					Image: &application_pb.Container_Image{
						Tag:  cflib.String("latest"),
						Name: "foobar",
					},
				},
				EnvVars: []*application_pb.EnvironmentVariable{{
					Name: "BUCKET",
					Spec: &application_pb.EnvironmentVariable_Blobstore{
						Blobstore: &application_pb.BlobstoreEnvVar{
							Name:    "bucket",
							SubPath: cflib.String("subpath"),
							Format: &application_pb.BlobstoreEnvVar_S3Direct{
								S3Direct: true,
							},
						},
					},
				}},
			}},
		})

		out := tb.Build(t)

		rr := result{}

		bucketResource, ok := out.Template.Resources["S3BucketBucket"].(*cflib.Resource[*s3.Bucket])
		if ok {
			rr.bucket = bucketResource
		}

		roleResource, ok := out.Template.Resources["IAMRoleMainAssume"].(*cflib.Resource[*iam.Role])
		if !ok || roleResource == nil {
			t.Fatalf("role resource not found")
		}

		rr.role = roleResource

		taskDef, ok := out.Template.Resources["ECSTaskDefinitionMain"].(*cflib.Resource[*ecs.TaskDefinition])
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
			"/subpath",
		})
		assertFunctionEqual(t, want, *rr.envVar.Value)
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

		bucketS3URL := cloudformation.Join("", []string{
			"s3://",
			bucketName,
			"/subpath",
		})

		assertFunctionEqual(t, rwPolicy.Statement[0].Resource[0], bucketARN)

		if rr.envVar == nil {
			t.Fatalf("env var not found")
		}

		assertFunctionEqual(t, bucketS3URL, *rr.envVar.Value)
	})

}
