package app

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	cfsecretsmanager "github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/tidwall/sjson"
)

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

type Application struct {
	appName           string
	template          *cloudformation.Template
	postgresDatabases []*PostgresDefinition
}

type PostgresDefinition struct {
	Secret                  *Resource[*cfsecretsmanager.Secret]
	Databse                 *application_pb.Database
	Postgres                *application_pb.Database_Postgres
	MigrationTaskOutputName *string
	SecretOutputName        *string
}

func NewApplication(name string) *Application {
	template := cloudformation.NewTemplate()

	return &Application{
		template: template,
		appName:  name,
	}
}

func (ss *Application) AppName() string {
	return ss.appName
}

func (ss *Application) Template() *cloudformation.Template {
	return ss.template
}

func (ss *Application) PostgresDatabases() []*PostgresDefinition {
	return ss.postgresDatabases
}

func (ss *Application) AddResource(resource IResource) {
	ss.template.Resources[resource.Name()] = resource
	for _, param := range resource.Parameters() {
		ss.template.Parameters[param.Name] = cloudformation.Parameter{
			Description: String(param.Description),
			Type:        param.Type,
			Default:     param.Default,
		}
	}
}

func (ss *Application) AddParameter(name string, param cloudformation.Parameter) {
	ss.template.Parameters[name] = param
}

func (ss *Application) Parameter(name string) *string {
	_, ok := ss.template.Parameters[name]
	if !ok {
		ss.template.Parameters[name] = cloudformation.Parameter{
			Type: "String",
		}
	}
	return String(cloudformation.Ref(name))
}

type IResource interface {
	cloudformation.Resource
	Ref() string
	GetAtt(name string) string
	Name() string
	DependsOn(IResource)
	Parameters() []Parameter
}

type Resource[T cloudformation.Resource] struct {
	name         string
	Resource     T
	Overrides    map[string]string
	parameters   []Parameter
	dependencies []IResource
}

type Parameter struct {
	Name        string
	Type        string
	Description string
	Default     interface{}
}

var reResourceUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func NewResource[T cloudformation.Resource](name string, rr T) *Resource[T] {
	fullName := resourceName(name, rr)
	res := &Resource[T]{
		name:     fullName,
		Resource: rr,
	}
	return res
}

func resourceName(name string, rr cloudformation.Resource) string {
	resourceType := strings.ReplaceAll(rr.AWSCloudFormationType(), "::", "")
	resourceType = strings.TrimPrefix(resourceType, "AWS_")
	name = reResourceUnsafe.ReplaceAllString(name, "")
	return fmt.Sprintf("%s%s", resourceType, name)
}

func (rr *Resource[T]) DependsOn(b IResource) {
	rr.dependencies = append(rr.dependencies, b)
}

func (rr Resource[T]) Parameters() []Parameter {
	return rr.parameters
}

func (rr Resource[T]) Ref() string {
	return cloudformation.Ref(rr.name)
}

func (rr Resource[T]) GetAtt(name string) string {
	return cloudformation.GetAtt(rr.name, name)
}

func (rr Resource[T]) Name() string {
	return rr.name
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
	if len(rr.dependencies) > 0 {
		dependencyNames := make([]string, len(rr.dependencies))
		for idx, dependency := range rr.dependencies {
			dependencyNames[idx] = dependency.Name()
		}
		marshalled, err = sjson.SetBytes(marshalled, "DependsOn", dependencyNames)
		if err != nil {
			return nil, err
		}
	}
	return marshalled, nil
}

func (rr Resource[T]) AWSCloudFormationType() string {
	return rr.Resource.AWSCloudFormationType()
}
