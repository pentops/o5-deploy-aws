package cflib

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/pentops/o5-deploy-aws/gen/o5/aws/deployer/v1/awsdeployer_pb"
	"github.com/tidwall/sjson"
)

type IResource interface {
	cloudformation.Resource
	Ref() TemplateRef
	GetAtt(name string) TemplateRef
	Name() string
	DependsOn(IResource)
	Parameters() []*awsdeployer_pb.Parameter
}

type Resource[T cloudformation.Resource] struct {
	name         string
	Resource     T
	overrides    map[string]string
	parameters   []*awsdeployer_pb.Parameter
	dependencies []IResource
}

type Output struct {
	Name        string
	Value       TemplateRef
	Description string
}

var reResourceUnsafe = regexp.MustCompile(`[^a-zA-Z0-9]`)

func NewResource[T cloudformation.Resource](name string, rr T) *Resource[T] {
	fullName := ResourceName(name, rr)
	res := &Resource[T]{
		name:      fullName,
		Resource:  rr,
		overrides: map[string]string{},
	}
	return res
}

func (rr *Resource[T]) DependsOn(b IResource) {
	rr.dependencies = append(rr.dependencies, b)
}

func (rr Resource[T]) Parameters() []*awsdeployer_pb.Parameter {
	return rr.parameters
}

func (rr *Resource[T]) Override(key, val string) {
	rr.overrides[key] = val
}

func (rr *Resource[T]) AddParameter(param *awsdeployer_pb.Parameter) {
	rr.parameters = append(rr.parameters, param)
}

func (rr Resource[T]) Ref() TemplateRef {
	return TemplateRef(cloudformation.Ref(rr.name))
}

func (rr Resource[T]) GetAtt(name string) TemplateRef {
	return TemplateRef(cloudformation.GetAtt(rr.name, name))
}

func (rr Resource[T]) Name() string {
	return rr.name
}

func (rr Resource[T]) MarshalJSON() ([]byte, error) {
	marshalled, err := json.Marshal(rr.Resource)
	if err != nil {
		return nil, err
	}
	if len(rr.overrides) == 0 {
		return marshalled, nil
	}
	for key, val := range rr.overrides {
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
