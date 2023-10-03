package cf

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/tidwall/sjson"
)

type PostgresDefinition struct {
	Secret   *Resource[*secretsmanager.Secret]
	Databse  *application_pb.Database
	Postgres *application_pb.Database_Postgres
}

type Template struct {
	template *cloudformation.Template

	postgresDatabases []*PostgresDefinition
}

func NewTemplate() *Template {
	template := cloudformation.NewTemplate()

	return &Template{
		template: template,
	}
}

func (ss *Template) Template() *cloudformation.Template {
	return ss.template
}

func String(str string) *string {
	return &str
}

func Stringf(str string, params ...interface{}) *string {
	return String(fmt.Sprintf(str, params...))
}

func Int(i int) *int {
	return &i
}

func Bool(b bool) *bool {
	return &b
}

func (ss *Template) Parameter(name string) *string {
	_, ok := ss.template.Parameters[name]
	if !ok {
		ss.template.Parameters[name] = cloudformation.Parameter{
			Type: "String",
		}
	}
	return String(cloudformation.Ref(name))
}

type Resource[T cloudformation.Resource] struct {
	Name       string
	Resource   T
	Overrides  map[string]string
	Parameters []Parameter
}

type Parameter struct {
	Name        string
	Type        string
	Description string
	Default     interface{}
}

func NewResource[T cloudformation.Resource](name string, rr T) *Resource[T] {
	resourceType := strings.ReplaceAll(rr.AWSCloudFormationType(), "::", "")
	resourceType = strings.TrimPrefix(resourceType, "AWS_")
	fullName := fmt.Sprintf("%s%s", resourceType, name)
	res := &Resource[T]{
		Name:     fullName,
		Resource: rr,
	}
	//ss.template.Resources[fullName] = res
	return res
}

func (rr Resource[T]) Ref() *string {
	return String(cloudformation.Ref(rr.Name))
}

func (rr Resource[T]) MarshalJSON() ([]byte, error) {
	marshalled, err := json.Marshal(rr.Resource)
	if err != nil {
		return nil, err
	}
	if len(rr.Overrides) == 0 {
		return marshalled, nil
	}
	for key, val := range rr.Overrides {
		marshalled, err = sjson.SetBytes(marshalled, fmt.Sprintf("Properties.%s", key), val)
		if err != nil {
			return nil, err
		}
	}
	return marshalled, nil
}

func (rr Resource[T]) AWSCloudFormationType() string {
	return rr.Resource.AWSCloudFormationType()
}
