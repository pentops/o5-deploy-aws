package app

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
	"github.com/tidwall/sjson"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

var reUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func CleanParameterName(unsafes ...string) string {
	titleCase := cases.Title(language.English)
	outParts := []string{}
	for _, unsafe := range unsafes {
		safeString := reUnsafe.ReplaceAllString(unsafe, "_")
		parts := strings.Split(safeString, "_")
		for _, part := range parts {
			outParts = append(outParts, titleCase.String(part))
		}
	}
	safeString := strings.Join(outParts, "")
	return safeString
}

type BuiltApplication struct {
	Template          *cloudformation.Template
	parameters        []*deployer_pb.Parameter
	postgresDatabases []*PostgresDefinition
	SNSTopics         []*SNSTopic
	Name              string
	Version           string
}

func (ba *BuiltApplication) TemplateJSON() ([]byte, error) {
	return ba.Template.JSON()
}

func (ba *BuiltApplication) PostgresDatabases() []*deployer_pb.PostgresDatabase {
	out := make([]*deployer_pb.PostgresDatabase, len(ba.postgresDatabases))
	for idx, db := range ba.postgresDatabases {
		out[idx] = &deployer_pb.PostgresDatabase{
			MigrationTaskOutputName: db.MigrationTaskOutputName,
			Database:                db.Databse,
			SecretOutputName:        db.SecretOutputName,
		}
	}
	return out
}

func (ba *BuiltApplication) Parameters() []*deployer_pb.Parameter {
	return ba.parameters
}

type Application struct {
	appName           string
	version           string
	postgresDatabases []*PostgresDefinition
	parameters        map[string]*deployer_pb.Parameter
	resources         map[string]IResource
	outputs           map[string]*Output
	snsTopics         map[string]*SNSTopic
}

type SNSTopic struct {
	Name string
}

type PostgresDefinition struct {
	Databse                 *application_pb.Database
	Postgres                *application_pb.Database_Postgres
	MigrationTaskOutputName *string
	SecretOutputName        *string
}

func NewApplication(name, version string) *Application {

	return &Application{
		appName:    name,
		version:    version,
		parameters: map[string]*deployer_pb.Parameter{},
		resources:  map[string]IResource{},
		outputs:    map[string]*Output{},
		snsTopics:  map[string]*SNSTopic{},
	}
}

func (ss *Application) AppName() string {
	return ss.appName
}

func (ss *Application) Build() *BuiltApplication {
	template := cloudformation.NewTemplate()
	parameters := map[string]*deployer_pb.Parameter{}

	for _, param := range ss.parameters {
		parameters[param.Name] = param
	}

	for _, resource := range ss.resources {
		template.Resources[resource.Name()] = resource
		for _, param := range resource.Parameters() {
			parameters[param.Name] = param
		}
	}

	for _, param := range parameters {
		mapped := cloudformation.Parameter{
			Type: param.Type,
		}
		if static := param.Source.GetStatic(); static != nil {
			mapped.Default = static.Value
		}
		if param.Description != "" {
			mapped.Description = String(param.Description)
		}

		template.Parameters[param.Name] = mapped
	}

	for _, output := range ss.outputs {
		template.Outputs[output.Name] = cloudformation.Output{
			Description: String(output.Description),
			Value:       output.Value,
		}
	}
	snsToipcs := []*SNSTopic{}
	for _, topic := range ss.snsTopics {
		snsToipcs = append(snsToipcs, topic)
	}

	parameterSlice := make([]*deployer_pb.Parameter, 0, len(parameters))
	for _, param := range parameters {
		parameterSlice = append(parameterSlice, param)
	}

	return &BuiltApplication{
		Template:          template,
		parameters:        parameterSlice,
		postgresDatabases: ss.postgresDatabases,
		SNSTopics:         snsToipcs,
		Name:              ss.appName,
		Version:           ss.version,
	}
}

func (ss *Application) AddSNSTopic(name string) {
	ss.snsTopics[name] = &SNSTopic{
		Name: name,
	}
}

func (ss *Application) AddResource(resource IResource) {
	ss.resources[resource.Name()] = resource
}

func (ss *Application) AddParameter(param *deployer_pb.Parameter) {
	if param.Name == "" {
		panic("No Name")
	}
	if param.Type == "" {
		panic("No Type")
	}
	ss.parameters[param.Name] = param
}

func (ss *Application) AddOutput(output *Output) {
	ss.outputs[output.Name] = output
}

func (ss *Application) Parameter(name string) *string {
	_, ok := ss.parameters[name]
	if !ok {
		ss.AddParameter(&deployer_pb.Parameter{
			Name: name,
			Type: "String",
		})
	}
	return String(cloudformation.Ref(name))
}

type IResource interface {
	cloudformation.Resource
	Ref() string
	GetAtt(name string) string
	Name() string
	DependsOn(IResource)
	Parameters() []*deployer_pb.Parameter
}

type Resource[T cloudformation.Resource] struct {
	name         string
	Resource     T
	Overrides    map[string]string
	parameters   []*deployer_pb.Parameter
	dependencies []IResource
}

type Output struct {
	Name        string
	Value       string
	Description string
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

func (rr Resource[T]) Parameters() []*deployer_pb.Parameter {
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
