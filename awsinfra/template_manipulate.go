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
	"github.com/pentops/o5-deploy-aws/gen/o5/deployer/v1/deployer_pb"
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

func (cf *CFClient) downloadCFTemplate(ctx context.Context, location *deployer_pb.S3Template) (string, error) {
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

func (cf *CFClient) ImportResources(ctx context.Context, templateBody string) ([]types.ResourceToImport, error) {

	builtTemplate := &Template{}
	err := json.Unmarshal([]byte(templateBody), builtTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}
	// Outputs usually cannot be resolved
	builtTemplate.Outputs = nil

	imports := []types.ResourceToImport{}

	paramMap := map[string]string{}

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

			fmt.Printf("Import %s: BucketName: %s\n", resourceName, bucketName)
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
					fmt.Printf("Secret %s not found, skipping\n", secretName)
					delete(builtTemplate.Resources, resourceName)
					continue
				}
				return nil, fmt.Errorf("failed to find secret %s: %w", secretName, err)
			}
			fmt.Printf("Import %s: Name: %s, id: %s\n", resourceName, secretName, secretArn)

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

	return imports, nil

}
