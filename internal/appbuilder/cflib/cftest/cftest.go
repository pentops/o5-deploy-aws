package cftest

import (
	"encoding/json"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/flowtest/jsontest"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
)

type TemplateAsserter struct {
	built *cflib.BuiltTemplate
}

func NewTemplateAsserter(tpl *cflib.BuiltTemplate) *TemplateAsserter {
	return &TemplateAsserter{
		built: tpl,
	}
}

func (ta *TemplateAsserter) getResourceJSON(t testing.TB, name string, into cloudformation.Resource) []byte {
	t.Helper()

	fullName := cflib.ResourceName(name, into)
	raw, ok := ta.built.Template.Resources[fullName]
	if !ok {

		t.Fatalf("resource %s not found", name)
	}
	asJSON, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err.Error())
	}

	return asJSON
}

func (ta *TemplateAsserter) GetResource(t testing.TB, name string, into cloudformation.Resource) {
	t.Helper()

	asJSON := ta.getResourceJSON(t, name, into)
	if err := json.Unmarshal(asJSON, into); err != nil {
		t.Fatal(err.Error())
	}
}

func (ta *TemplateAsserter) AssertResource(t testing.TB, name string, intoType cloudformation.Resource, want map[string]interface{}) {
	t.Helper()
	asJSON := ta.getResourceJSON(t, name, intoType)
	asserter, err := jsontest.NewAsserter(asJSON)
	if err != nil {
		t.Fatal(err.Error())
	}
	asserter.AssertEqualSet(t, "Properties", want)
}
