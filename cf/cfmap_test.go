package cf

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/stretchr/testify/assert"
)

func TestBasicMap(t *testing.T) {

	app := &application_pb.Application{
		Name:            "app1",
		SubEnvironments: []string{"a", "b"},
		Runtimes: []*application_pb.Runtime{{
			Name: "main",
			Containers: []*application_pb.Container{{
				Name: "main",
				Source: &application_pb.Container_Image_{
					Image: &application_pb.Container_Image{
						Tag:  "latest",
						Name: "foobar",
					},
				},
			}},
		}},
	}
	env := &environment_pb.Environment{
		FullName: "super-a",
	}

	out, err := BuildCloudformation(app, env)
	if err != nil {
		t.Fatal(err.Error())
	}

	yy, err := out.template.YAML()
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(string(yy))

}

func TestRuntime(t *testing.T) {

	global := globalData{
		uniquePrefix: "Test",
	}

	rs, err := NewRuntimeService(global, &application_pb.Runtime{
		Name: "main",
		Containers: []*application_pb.Container{{
			Name: "main",
			Source: &application_pb.Container_Image_{
				Image: &application_pb.Container_Image{
					Tag:  "latest",
					Name: "foobar",
				},
			},
		}},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(rs.Containers) != 2 {
		t.Fatalf("expected 2 container definition, got %d", len(rs.Containers))
	}

	{
		imageRaw := rs.Containers[0].Image

		join := &Join{}
		decode(t, imageRaw, join)
		assert.Equal(t, "foobar", join.Vals[2])
		assert.Equal(t, "latest", join.Vals[4])

		ref := &Ref{}
		decode(t, join.Vals[0], ref)
		assert.Equal(t, "ECSRepo", ref.Ref)
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
