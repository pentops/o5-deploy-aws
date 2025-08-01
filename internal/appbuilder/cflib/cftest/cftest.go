package cftest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/flowtest/jsontest"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
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

		t.Fatalf("resource %q not found (%q)", fullName, name)
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

func (ta *TemplateAsserter) AssertResource(t testing.TB, name string, intoType cloudformation.Resource, want map[string]any) {
	t.Helper()
	asJSON := ta.getResourceJSON(t, name, intoType)
	asserter, err := jsontest.NewAsserter(asJSON)
	if err != nil {
		t.Fatal(err.Error())
	}
	asserter.AssertEqualSet(t, "Properties", want)
}

func (ta *TemplateAsserter) GetParameter(t testing.TB, want awsdeployer_pb.IsParameterSourceTypeWrappedType) *awsdeployer_pb.Parameter {
	gotClose := []awsdeployer_pb.IsParameterSourceTypeWrappedType{}
	for _, param := range ta.built.Parameters {
		s := param.Source.Get()
		if s.ParameterSourceTypeKey() != want.ParameterSourceTypeKey() {
			continue
		}

		if proto.Equal(s, want) {
			return param
		}

		gotClose = append(gotClose, s)

	}

	for _, got := range gotClose {
		t.Logf("parameter correct type %q but no match \n%s", want.ParameterSourceTypeKey(), prototext.Format(got))
	}

	t.Fatalf("parameter %q not found", want.ParameterSourceTypeKey())

	return nil
}

func (ta *TemplateAsserter) GetOutput(t testing.TB, name string) *cloudformation.Output {
	output, ok := ta.built.Template.Outputs[name]
	if !ok {
		t.Fatalf("output %q not found", name)
	}
	return &output
}

func (ta *TemplateAsserter) GetOutputValue(t testing.TB, name string) string {
	t.Helper()
	output := ta.GetOutput(t, name)

	return ta.resolveFuncWithPlaceholders(t, output.Value)
}

func (ta *TemplateAsserter) resolveFuncWithPlaceholders(t testing.TB, f any) string {
	t.Helper()
	str, err := cflib.ResolveFunc(f, cflib.ParamFunc(func(key string) (string, bool) {
		_, ok := ta.built.Template.Resources[key]
		if ok {
			return key, true
		}
		return fmt.Sprintf("{%s}", key), true
	}))
	if err != nil {
		t.Fatal(err)
	}
	return str
}
