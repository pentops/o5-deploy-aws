package appbuilder

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/awslabs/goformation/v7/cloudformation/iam"
	"github.com/awslabs/goformation/v7/intrinsics"
	"github.com/google/go-cmp/cmp"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
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

	for key := range out.Template.Resources {
		t.Logf("resource: %s", key)
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

func (tb *testBuilder) BuildAndAssert(t *testing.T) *TemplateAsserter {
	out := tb.Build(t)
	ta := cftest.NewTemplateAsserter(&cflib.BuiltTemplate{
		Template:   out.Template,
		Parameters: out.Parameters,
	})
	return &TemplateAsserter{
		TemplateAsserter: ta,
		out:              out,
	}
}

type TemplateAsserter struct {
	*cftest.TemplateAsserter
	out *BuiltApplication
}

func (ta *TemplateAsserter) GetDatabase(t testing.TB, name string) *awsdeployer_pb.PostgresDatabaseResource {
	t.Helper()
	for _, db := range ta.out.Databases {
		if db.DbName == name {
			return db
		}
	}

	for _, db := range ta.out.Databases {
		t.Logf("had db %q", db.DbName)
	}
	t.Fatalf("database %q not found", name)
	return nil
}

func (ta *TemplateAsserter) StandardRuntimeAssert(t testing.TB) *standardRuntimeAssert {
	return ta.SingleContainerTaskDef(t, "main", "main")
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

func (ca *containerAssert) MustNotHaveEnvOrSecret(t testing.TB, name string) {
	t.Helper()
	if ca.GetEnv(name) != nil {
		t.Errorf("unexpected env %s", name)
	}

	if ca.GetSecret(name) != nil {
		t.Errorf("unexpected secret %s", name)
	}
}
func (ca *containerAssert) MustGetEnv(t testing.TB, name string) *ecs.TaskDefinition_KeyValuePair {
	t.Helper()
	e := ca.GetEnv(name)
	if e == nil {
		t.Fatalf("env %s not found", name)
	}
	return e
}

func (ca *containerAssert) AssertEnvVal(t testing.TB, name, val string) {
	t.Helper()
	e := ca.MustGetEnv(t, name)
	if *e.Value != val {
		t.Fatalf("env %s: expected %s, got %s", name, val, *e.Value)
	}
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
	t.Helper()
	s := ca.GetSecret(name)
	if s == nil {
		for _, s := range ca.container.Secrets {
			t.Logf("secret: %s", s.Name)
		}
		for _, e := range ca.container.Environment {
			t.Logf("env: %s", *e.Name)
		}
		t.Fatalf("secret %s not found", name)
	}
	return s
}

type taskDefAssert struct {
	sidecar    *containerAssert
	containers map[string]*containerAssert
	taskDef    *ecs.TaskDefinition
	parent     *TemplateAsserter
}

type standardRuntimeAssert struct {
	*taskDefAssert
	main *containerAssert
}

func (aa *TemplateAsserter) TaskDefAssert(t testing.TB, name string) *taskDefAssert {
	taskDef := &ecs.TaskDefinition{}
	aa.GetResource(t, name, taskDef)
	tda := &taskDefAssert{
		parent:     aa,
		taskDef:    taskDef,
		containers: map[string]*containerAssert{},
	}
	for _, c := range taskDef.ContainerDefinitions {
		if c.Name == O5SidecarContainerName {
			tda.sidecar = &containerAssert{container: c}
		} else {
			tda.containers[c.Name] = &containerAssert{container: c}
		}
	}
	return tda
}

func (aa *TemplateAsserter) SingleContainerTaskDef(t testing.TB, taskDefName string, containerName string) *standardRuntimeAssert {
	t.Helper()

	taskDef := aa.TaskDefAssert(t, taskDefName)
	sra := &standardRuntimeAssert{
		taskDefAssert: taskDef,
	}
	if len(taskDef.containers) != 1 {
		t.Fatalf("expected 1 container (other than sidecar), got %d", len(taskDef.containers))
	}
	main, ok := taskDef.containers[containerName]
	if !ok {
		t.Fatalf("container %q not found", containerName)
	}
	sra.main = main

	return sra
}

func (sra *standardRuntimeAssert) AssertNoSidecar(t testing.TB) {
	t.Helper()
	if sra.sidecar != nil {
		t.Errorf("unexpected sidecar container")
	}
}

func (sra *standardRuntimeAssert) WithMainContainer(t testing.TB, cb func(t testing.TB, c *containerAssert)) {
	if sra.main == nil {
		t.Errorf("main container not found")
		return
	}
	cb(t, sra.main)

}
func (sra *standardRuntimeAssert) WithSidecarContainer(t testing.TB, cb func(t testing.TB, c *containerAssert)) {
	if sra.sidecar == nil {
		t.Errorf("sidecar container not found")
		return
	}
	cb(t, sra.sidecar)
}

func (sra *standardRuntimeAssert) WithRole(t testing.TB, cb func(t testing.TB, role *iam.Role)) {
	t.Helper()
	arn := sra.taskDef.TaskRoleArn
	ref := decodeGetAtt(t, *arn)
	t.Logf("role ref: %s", ref.Resource)
	role := &iam.Role{}
	sra.parent.GetResource(t, "!!"+ref.Resource, role)
	cb(t, role)
}

func (sra *standardRuntimeAssert) WithPolicy(t testing.TB, name string, cb func(t testing.TB, policy *PolicyDocument)) {
	t.Helper()
	sra.WithRole(t, func(t testing.TB, role *iam.Role) {
		t.Helper()
		policy := findPolicy(t, role, name)
		if policy == nil {
			t.Fatalf("policy %q not found", name)
		}
		cb(t, policy)
	})
}

func assertFunctionEqual(t testing.TB, want, got string, messageAndArgs ...any) {
	t.Helper()
	if want == got {
		return
	}

	wantAny := decodeAny(t, want)
	gotAny := decodeAny(t, got)

	diff := cmp.Diff(wantAny, gotAny)
	if diff != "" {
		assert.FailNow(t, diff, messageAndArgs...)
	}
}

func findPolicy(t testing.TB, roleResource *iam.Role, name string) *PolicyDocument {
	t.Helper()
	for _, policy := range roleResource.Policies {
		join := decodeJoin(t, policy.PolicyName)
		if join.Vals[2] == name {
			doc, ok := policy.PolicyDocument.(PolicyDocument)
			if ok {
				return &doc
			}
			jb, err := json.Marshal(policy.PolicyDocument)
			if err != nil {
				t.Fatal(err)
			}
			jsonStrictUnmarshal(t, jb, &doc)
			return &doc
		}
	}
	return nil
}

type paramResolve struct {
	tryFirst []cflib.Params
}

func (pr paramResolve) Get(key string) (string, bool) {
	for _, p := range pr.tryFirst {
		if val, ok := p.Get(key); ok {
			return val, true
		}
	}

	return fmt.Sprintf("{%s}", key), true
}

func funcToPlaceholders(t testing.TB, f interface{}) string {
	t.Helper()
	str, err := cflib.ResolveFunc(f, paramResolve{})
	if err != nil {
		t.Fatal(err)
	}
	return str
}

func decodeJoin(t testing.TB, s string) *Join {
	t.Helper()
	join := &Join{}
	decode(t, s, join)
	return join
}

func decodeRef(t testing.TB, s string) *Ref {
	t.Helper()
	ref := &Ref{}
	decode(t, s, ref)
	return ref
}

func decodeGetAtt(t testing.TB, s string) *GetAtt {
	t.Helper()
	getAtt := &GetAtt{}
	decode(t, s, getAtt)
	return getAtt
}

func decodeAny(t testing.TB, s string) interface{} {

	jb, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := intrinsics.ProcessJSON(jb, nil)
	if err != nil {
		t.Fatal(err)
	}

	var out interface{}
	if err := json.Unmarshal(processed, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func jsonStrictUnmarshal(t testing.TB, val []byte, into interface{}) {
	dd := json.NewDecoder(bytes.NewReader(val))
	dd.DisallowUnknownFields()
	if err := dd.Decode(into); err != nil {
		t.Fatalf("decoding %s: %s", val, err.Error())
	}
}

func decode(t testing.TB, s string, into interface{}) {
	t.Helper()
	val, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Logf("decoded: '%s'", string(val))
	jsonStrictUnmarshal(t, val, into)
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

func assertJSONEqual(t *testing.T, path []string, expected, got interface{}) {
	switch expected := expected.(type) {
	case []interface{}:
		gotSlice, ok := got.([]interface{})
		if !ok {
			t.Errorf("at %s: expected slice, got %T", strings.Join(path, "."), got)
		}
		assertJSONArrayEqual(t, path, expected, gotSlice)
	case map[string]interface{}:
		gotMap, ok := got.(map[string]interface{})
		if !ok {
			t.Errorf("at %s: expected map, got %T", strings.Join(path, "."), got)
		}
		assertJSONMapEqual(t, path, expected, gotMap)
	default:
		if !reflect.DeepEqual(expected, got) {
			t.Errorf("at %s: DEEP expected %#v (%T), got %#v (%T)", strings.Join(path, "."), expected, expected, got, got)
		}
	}
}

func assertJSONArrayEqual(t *testing.T, path []string, expected, got []interface{}) {
	if len(expected) != len(got) {
		t.Errorf("expected %d elements, got %d", len(expected), len(got))
	}

	for ii, val := range expected {
		if ii >= len(got) {
			t.Errorf("at %s: index %d out of range", strings.Join(path, "."), ii)
			continue
		}
		gotVal := got[ii]
		assertJSONEqual(t, append(path, strconv.Itoa(ii)), val, gotVal)
	}
}

func assertJSONMapEqual(t *testing.T, path []string, expected, got map[string]interface{}) {
	for k, v := range expected {
		got, ok := got[k]
		if !ok {
			t.Errorf("at %s: key %q not found", strings.Join(path, "."), k)
		}

		assertJSONEqual(t, append(path, k), v, got)
	}

	for k := range got {
		if _, ok := expected[k]; !ok {
			t.Errorf("unexpected key %q at %s", k, strings.Join(path, "."))
		}
	}
}
