package app

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
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
						Tag:  String("latest"),
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
						Tag:  String("latest"),
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

	taskDef := &ecs.TaskDefinition{}
	getResource(t, out, "main", taskDef)
	if len(taskDef.ContainerDefinitions[0].PortMappings) != 1 {
		t.Fatalf("expected one port mapping, got %d", len(taskDef.ContainerDefinitions[0].PortMappings))
	}

	t.Logf("ports: %v", taskDef.ContainerDefinitions[0].PortMappings)
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
						Tag:  String("latest"),
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

	taskDef := &ecs.TaskDefinition{}
	getResource(t, out, "main", taskDef)
	t.Logf("ports: %v", taskDef.ContainerDefinitions[1].PortMappings)
}

func getResource(t testing.TB, template *BuiltApplication, name string, into cloudformation.Resource) {
	t.Helper()

	fullName := resourceName(name, into)
	raw, ok := template.Template.Resources[fullName]
	if !ok {
		t.Fatalf("resource %s not found", fullName)
	}

	// Strange method, but it should work...
	asJSON, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err.Error())
	}

	if err := json.Unmarshal(asJSON, into); err != nil {
		t.Fatal(err.Error())
	}
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
					Tag:  String("latest"),
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
		appName:          "Test",
		replayChance:     1,
		deadletterChance: 2,
	}

	rs, err := NewRuntimeService(global, &application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  String("latest"),
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

func decode(t *testing.T, s string, into interface{}) {
	t.Helper()
	val, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("decoded: %s", string(val))
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

type Ref struct {
	Ref string `json:"Ref"`
}
