package appbuilder

import (
	"fmt"

	"github.com/awslabs/goformation/v7/cloudformation"
	"github.com/awslabs/goformation/v7/cloudformation/policies"
	"github.com/awslabs/goformation/v7/cloudformation/secretsmanager"
	"github.com/pentops/o5-deploy-aws/gen/o5/application/v1/application_pb"
	"github.com/pentops/o5-deploy-aws/internal/appbuilder/cflib"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SecretRef interface {
	ARN() cflib.TemplateRef
	SecretValueFrom(jsonKey string) cflib.TemplateRef
}

type secretType int

const (
	secretTypeSecretsManager secretType = iota
	secretTypeSSMParameter
)

type secretInfo struct {
	refName    string
	arn        cflib.TemplateRef
	secretType secretType
}

func (si secretInfo) ARN() cflib.TemplateRef {
	return si.arn
}

func (si secretInfo) SecretValueFrom(jsonKey string) cflib.TemplateRef {

	switch si.secretType {
	case secretTypeSecretsManager:
		versionStage := ""
		versionID := ""
		return cflib.Join(":",
			si.arn,
			jsonKey,
			versionStage,
			versionID,
		)

	case secretTypeSSMParameter:
		if jsonKey == "" {
			return si.arn
		}
		return cflib.Join("",
			si.arn,
			"/",
			jsonKey,
		)

	default:
		panic("unhandled secret type")
	}
}

func mapSecret(bb *Builder, secretDef *application_pb.Secret) (*secretInfo, error) {
	appName := bb.AppName()

	var parameterName cflib.TemplateRef
	switch secretDef.Scope {
	case application_pb.SecretScope_UNSPECIFIED: // Application Level
		resourceName := fmt.Sprintf("AppSecret%s", cases.Title(language.English).String(secretDef.Name))
		secret := cflib.NewResource(resourceName, &secretsmanager.Secret{
			Name: cflib.Join("/",
				"", // Leading /
				cloudformation.Ref(EnvNameParameter),
				appName,
				secretDef.Name,
			).RefPtr(),
			Description:                     cflib.Stringf("Application Level Secret for %s - value must be set manually", secretDef.Name),
			AWSCloudFormationDeletionPolicy: policies.DeletionPolicy("Retain"),
		})
		bb.Template.AddResource(secret)
		return &secretInfo{
			refName:    secretDef.Name,
			arn:        secret.Ref(),
			secretType: secretTypeSecretsManager,
		}, nil

	case application_pb.SecretScope_ENVIRONMENT:
		parameterName = cflib.Join("/",
			"", // Leading /
			cloudformation.Ref(EnvNameParameter),
			"SHARED",
			secretDef.Name,
		)

	case application_pb.SecretScope_CLUSTER:
		parameterName = cflib.Join("/",
			"", // Leading /
			cloudformation.Ref(ClusterNameParameter),
			"SHARED",
			secretDef.Name,
		)
	case application_pb.SecretScope_GLOBAL:
		parameterName = cflib.Join("/",
			"", // Leading /
			"GLOBAL",
			secretDef.Name,
		)
	default:
		return nil, fmt.Errorf("unsupported secret scope %v", secretDef.Scope)
	}

	arn := cflib.Join("", []string{
		"arn:aws:ssm:",
		AWSRegionParameter, ":",
		AWSAccountIDParameter,
		":parameter:/",
		parameterName.Ref(),
	})

	return &secretInfo{
		refName:    secretDef.Name,
		arn:        arn,
		secretType: secretTypeSSMParameter,
	}, nil
}
