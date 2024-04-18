package cf

import (
	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-go/deployer/v1/deployer_pb"
)

type Template struct {
	parameters map[string]*deployer_pb.Parameter
	resources  map[string]IResource
	outputs    map[string]*Output
}

func NewTemplate() *Template {
	return &Template{
		parameters: make(map[string]*deployer_pb.Parameter),
		resources:  make(map[string]IResource),
		outputs:    make(map[string]*Output),
	}
}

func (ss *Template) AddResource(resource IResource) {
	ss.resources[resource.Name()] = resource
}

func (ss *Template) AddParameter(param *deployer_pb.Parameter) {
	if param.Name == "" {
		panic("No Name")
	}
	if param.Type == "" {
		panic("No Type")
	}
	ss.parameters[param.Name] = param
}

func (ss *Template) AddOutput(output *Output) {
	ss.outputs[output.Name] = output
}

func (ss *Template) Build() (*cloudformation.Template, []*deployer_pb.Parameter) {
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

	parameterSlice := make([]*deployer_pb.Parameter, 0, len(parameters))
	for _, param := range parameters {
		parameterSlice = append(parameterSlice, param)
	}

	return template, parameterSlice
}
