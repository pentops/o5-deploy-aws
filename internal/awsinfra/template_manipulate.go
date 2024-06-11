package awsinfra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/gen/o5/awsdeployer/v1/awsdeployer_pb"
)

type Template struct {
	AWSTemplateFormatVersion string                 `json:"AWSTemplateFormatVersion"`
	Resources                map[string]Resource    `json:"Resources"`
	Parameters               map[string]interface{} `json:"Parameters"`
	Outputs                  map[string]interface{} `json:"Outputs,omitempty"`
}

type Resource struct {
	Type           string                 `json:"Type"`
	DeletionPolicy string                 `json:"DeletionPolicy,omitempty"`
	Properties     map[string]interface{} `json:"Properties,omitempty"`
}

func EmptyTemplate() string {

	emptyTemplate := &Template{
		AWSTemplateFormatVersion: "2010-09-09",
		Resources: map[string]Resource{
			"NullResource": {
				Type: "AWS::CloudFormation::WaitConditionHandle",
			},
		},
		Parameters: map[string]interface{}{},
	}

	templateJSON, err := json.MarshalIndent(emptyTemplate, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(templateJSON)

}

func (cf *CFClient) findSecret(ctx context.Context, secretName string) (string, error) {
	secret, err := cf.secretsManagerClient.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return "", err
	}
	return *secret.ARN, nil
}

func (cf *CFClient) downloadCFTemplate(ctx context.Context, location *awsdeployer_pb.S3Template) (string, error) {
	// TODO: This ignores the region in the template location, assuming it is
	// the same region the worker is running. There is probably an easy way to
	// set the region here.
	res, err := cf.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(location.Bucket),
		Key:    aws.String(location.Key),
	})
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type ImportOutput struct {
	Imports     []types.ResourceToImport
	NewTemplate string
}

func (cf *CFClient) ImportResources(ctx context.Context, templateBody string, params []types.Parameter) (*ImportOutput, error) {

	builtTemplate := &Template{}
	err := json.Unmarshal([]byte(templateBody), builtTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}
	// Outputs usually cannot be resolved
	builtTemplate.Outputs = nil

	imports := []types.ResourceToImport{}

	paramMap := map[string]string{
		"AWS::Region":  cf.region, //"us-east-1",
		"AWS::Account": cf.accountID,
	}

	for _, param := range params {
		paramMap[*param.ParameterKey] = *param.ParameterValue
	}

	for resourceName, resource := range builtTemplate.Resources {

		switch resource.Type {
		case "AWS::S3::Bucket":
			bucketNameValue, ok := resource.Properties["BucketName"]
			if !ok {
				return nil, fmt.Errorf("missing property %s.%s", resourceName, "BucketName")
			}
			bucketName, err := ResolveFunc(bucketNameValue, paramMap)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve property %s.%s: %w", resourceName, "BucketName", err)
			}

			imports = append(imports, types.ResourceToImport{
				LogicalResourceId:  aws.String(resourceName),
				ResourceType:       aws.String(resource.Type),
				ResourceIdentifier: map[string]string{"BucketName": bucketName},
			})

		case "AWS::SecretsManager::Secret":
			secretNameValue, ok := resource.Properties["Name"]
			if !ok {
				return nil, fmt.Errorf("missing property %s.%s", resourceName, "Name")
			}
			secretName, err := ResolveFunc(secretNameValue, paramMap)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve property %s.%s: %w", resourceName, "Name", err)
			}

			secretArn, err := cf.findSecret(ctx, secretName)
			if err != nil {
				if strings.Contains(err.Error(), "Secrets Manager can't find the specified secret") {
					delete(builtTemplate.Resources, resourceName)
					log.WithField(ctx, "secretName", secretName).Info("Secret not found for import, removing resource")
					continue
				}
				return nil, fmt.Errorf("failed to find secret %s: %w", secretName, err)
			}

			imports = append(imports, types.ResourceToImport{
				LogicalResourceId:  aws.String(resourceName),
				ResourceType:       aws.String(resource.Type),
				ResourceIdentifier: map[string]string{"Id": secretArn},
			})
		default:
			delete(builtTemplate.Resources, resourceName)
			continue
		}
	}

	builtTemplate.Resources["NullResource"] = Resource{
		Type: "AWS::CloudFormation::WaitConditionHandle",
	}
	newTemplate, err := json.Marshal(builtTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new template: %w", err)
	}

	return &ImportOutput{
		Imports:     imports,
		NewTemplate: string(newTemplate),
	}, nil

}

type FuncJoin struct {
	Join   string
	Values []interface{}
}

func (f *FuncJoin) MarshalJSON() ([]byte, error) {
	out := make([]interface{}, 2)
	out[0] = f.Join
	out[1] = f.Values
	return json.Marshal(out)
}

func (f *FuncJoin) UnmarshalJSON(data []byte) error {
	var out []interface{}
	err := json.Unmarshal(data, &out)
	if err != nil {
		return err
	}
	if len(out) != 2 {
		return fmt.Errorf("expected 2 elements, got %d", len(out))
	}
	var ok bool
	f.Join, ok = out[0].(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", out[0])
	}

	f.Values, ok = out[1].([]interface{})
	if !ok {
		return fmt.Errorf("expected []interface{}, got %T", out[1])
	}
	return err
}

func ResolveFunc(raw interface{}, params map[string]string) (string, error) {
	if stringVal, ok := raw.(string); ok {
		return stringVal, nil
	}

	mapStringInterface, ok := raw.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("expected map[string]interface{}, got %T", raw)
	}
	var fnName string
	var fnValue interface{}

	if len(mapStringInterface) != 1 {
		return "", fmt.Errorf("expected 1 key, got %d", len(mapStringInterface))
	}
	for key, value := range mapStringInterface {
		fnName = key
		fnValue = value
	}

	switch fnName {
	case "Ref":
		refName, ok := fnValue.(string)
		if !ok {
			return "", fmt.Errorf("ref value should be string, got %T", fnValue)
		}
		if val, ok := params[refName]; ok {
			return val, nil
		}
		return "", fmt.Errorf("ref %s not found in params", refName)

	case "Fn::Join":
		joiner := &FuncJoin{}
		joinerValues, ok := fnValue.([]interface{})
		if !ok {
			return "", fmt.Errorf("expected []interface{}, got %T", fnValue)
		}
		if len(joinerValues) != 2 {
			return "", fmt.Errorf("expected 2 elements, got %d", len(joinerValues))
		}
		joiner.Join, ok = joinerValues[0].(string)
		if !ok {
			return "", fmt.Errorf("expected string, got %T", joinerValues[0])
		}
		joiner.Values, ok = joinerValues[1].([]interface{})
		if !ok {
			return "", fmt.Errorf("expected []interface{}, got %T", joinerValues[1])
		}
		parts := make([]string, 0, len(joiner.Values))
		for _, rawValue := range joiner.Values {
			resolvedValue, err := ResolveFunc(rawValue, params)
			if err != nil {
				return "", fmt.Errorf("resolving value: %w", err)
			}
			parts = append(parts, resolvedValue)
		}
		return strings.Join(parts, joiner.Join), nil
	default:
		return "", fmt.Errorf("unknown function: %s", fnName)
	}
}
