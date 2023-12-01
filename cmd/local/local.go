package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/jsonapi/jsonapi"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-runtime-sidecar/outbox"
	"github.com/pentops/o5-runtime-sidecar/runner"
	"github.com/pentops/o5-runtime-sidecar/sqslink"
	"golang.org/x/sync/errgroup"
)

type flagConfig struct {
	envFilename string
	appFilename string
	workdir     string
	command     string
}

func main() {
	cfg := flagConfig{}
	flag.StringVar(&cfg.envFilename, "env", "", "environment file")
	flag.StringVar(&cfg.appFilename, "app", "", "application file")
	flag.StringVar(&cfg.workdir, "workdir", "", "working directory")
	flag.StringVar(&cfg.command, "command", "", "command to run")

	flag.Parse()

	if cfg.appFilename == "" {
		fmt.Fprintln(os.Stderr, "missing application file (-app)")
		os.Exit(1)
	}

	if cfg.envFilename == "" {
		fmt.Fprintln(os.Stderr, "missing application file (-app)")
		os.Exit(1)
	}

	if cfg.workdir == "" {
		fmt.Fprintln(os.Stderr, "missing working directory (-workdir)")
		os.Exit(1)
	}

	if cfg.command == "" {
		fmt.Fprintln(os.Stderr, "missing command (-command)")
		os.Exit(1)
	}

	ctx := context.Background()
	if err := do(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		panic(err.Error())
	}
}

func do(ctx context.Context, flagConfig flagConfig) error {
	defer func() {
		fmt.Printf("Deferred Func Running\n")
	}()
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	s3Client := s3.NewFromConfig(awsConfig)
	secretsManagerClient := secretsmanager.NewFromConfig(awsConfig)

	appConfig := &application_pb.Application{}
	if err := protoread.PullAndParse(ctx, s3Client, flagConfig.appFilename, appConfig); err != nil {
		return err
	}

	if appConfig.DeploymentConfig == nil {
		appConfig.DeploymentConfig = &application_pb.DeploymentConfig{}
	}

	app, err := app.BuildApplication(appConfig, "VERSION")
	if err != nil {
		return err
	}

	env := &environment_pb.Environment{}
	if err := protoread.PullAndParse(ctx, s3Client, flagConfig.envFilename, env); err != nil {
		return err
	}

	awsTarget := env.GetAws()
	if awsTarget == nil {
		return fmt.Errorf("AWS Deployer requires the type of environment provider to be AWS")
	}

	deployerResolver, err := deployer.BuildParameterResolver(ctx, env)
	if err != nil {
		return err
	}

	mainRuntime := app.GetRuntime("main")
	if mainRuntime == nil {
		return fmt.Errorf("no runtime named 'main' found")
	}

	if len(mainRuntime.Containers) != 1 {
		return fmt.Errorf("runtime 'main' must have exactly one container")
	}

	refs := map[string]string{}

	for _, parameter := range app.Parameters() {
		resolved, err := deployerResolver.ResolveParameter(parameter)
		if err != nil {
			return err
		}
		refs[parameter.Name] = resolved.GetValue()
	}

	for key, refVal := range app.Refs() {
		refs[key] = refVal
	}

	refs["AWS::Region"] = awsTarget.Region
	refs["AWS::StackName"] = fmt.Sprintf("%s-%s", env.FullName, app.AppName())

	eg, ctx := errgroup.WithContext(ctx)

	var mainCommand *exec.Cmd
	{
		container := mainRuntime.Containers[0]
		parts := strings.Split(flagConfig.command, " ")
		cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
		cmd.Dir = flagConfig.workdir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		envVars, err := buildEnvironment(ctx, container.Container, refs, secretsManagerClient)
		if err != nil {
			return err
		}
		cmd.Env = envVars
		mainCommand = cmd
	}

	eg.Go(func() error {
		err := mainCommand.Run()
		if err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
		log.Info(ctx, "command exited with no error")
		return nil
	})

	{

		ctx := log.WithField(ctx, "runtime", "sidecar")
		runner := runner.NewRuntime()

		container := mainRuntime.IngressContainer

		envVars, err := buildEnvironment(ctx, container, refs, secretsManagerClient)
		if err != nil {
			return err
		}

		codecOptions := jsonapi.Options{
			ShortEnums: &jsonapi.ShortEnumsOption{
				UnspecifiedSuffix: "UNSPECIFIED",
				StrictUnmarshal:   true,
			},
			WrapOneof: true,
		}

		envMap := map[string]string{}
		for _, envVar := range envVars {
			parts := strings.SplitN(envVar, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid environment variable %s", envVar)
			}
			key := parts[0]
			value := parts[1]
			envMap[key] = value
		}

		postgresURL := envMap["POSTGRES_OUTBOX"]
		sqsURL := envMap["SQS_URL"]
		snsPrefix := envMap["SNS_PREFIX"]
		jwks := strings.Split(envMap["JWKS"], ",")
		service := envMap["SERVICE_ENDPOINT"]

		if postgresURL != "" || sqsURL != "" {
			if snsPrefix == "" {
				return fmt.Errorf("missing SNS_PREFIX for Postgres outbox")
			}
			runner.Sender = outbox.NewSNSBatcher(sns.NewFromConfig(awsConfig), snsPrefix)
		}

		if postgresURL != "" {
			if err := runner.AddOutbox(ctx, postgresURL); err != nil {
				return err
			}
		}

		if sqsURL != "" {
			sqsClient := sqs.NewFromConfig(awsConfig)
			runner.Worker = sqslink.NewWorker(sqsClient, sqsURL, runner.Sender)
		}

		if err := runner.AddRouter(8888, codecOptions); err != nil {
			return fmt.Errorf("add router: %w", err)
		}

		if len(jwks) > 0 {
			if err := runner.AddJWKS(ctx, jwks...); err != nil {
				return fmt.Errorf("add JWKS: %w", err)
			}
		}

		endpoints := strings.Split(service, ",")
		for _, endpoint := range endpoints {
			endpoint = strings.TrimSpace(endpoint)
			if endpoint == "" {
				continue
			}
			endpoint = strings.ReplaceAll(endpoint, "main", "localhost")
			if err := runner.AddEndpoint(ctx, endpoint); err != nil {
				return fmt.Errorf("add endpoint %s: %w", endpoint, err)
			}
		}

		eg.Go(func() error {
			err := runner.Run(ctx)
			if err != nil {
				return fmt.Errorf("runner failed: %w", err)
			}
			log.Info(ctx, "runner exited with no error")
			return nil
		})
	}

	return eg.Wait()
}

func buildEnvironment(ctx context.Context, container *ecs.TaskDefinition_ContainerDefinition, refs map[string]string, ssm *secretsmanager.Client) ([]string, error) {

	env := os.Environ()

	for _, envVar := range container.Environment {
		key := *envVar.Name
		value, err := decodeIntrinsic(*envVar.Value, refs)
		if err != nil {
			return nil, err
		}

		fmt.Printf("%s=%v\n", key, value)
		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}

	for _, secret := range container.Secrets {
		key := secret.Name
		value, err := decodeIntrinsic(secret.ValueFrom, refs)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(value, ":")
		// secret-name:json-key:version-stage:version-id
		secretName := parts[0]
		secretValueResponse, err := ssm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: &secretName,
		})
		if err != nil {
			return nil, err
		}
		secretString := *secretValueResponse.SecretString
		if len(parts) > 1 {
			jsonKey := parts[1]
			secretMap := map[string]interface{}{}
			if err := json.Unmarshal([]byte(secretString), &secretMap); err != nil {
				return nil, err
			}
			secretString = secretMap[jsonKey].(string)
		}

		fmt.Printf("%s=(SECRET)\n", key)
		env = append(env, fmt.Sprintf("%s=%v", key, secretString))
	}
	return env, nil
}

func decodeIntrinsic(value string, refs map[string]string) (string, error) {
	// taken from goformation/intrinsics.go
	var decoded []byte
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		// The string value is not base64 encoded, so it's not an intrinsic so just pass it back
		return value, nil
	}

	var intrinsic map[string]interface{}
	if err := json.Unmarshal([]byte(decoded), &intrinsic); err != nil {
		// The string value is not JSON, so it's not an intrinsic so just pass it back
		return value, nil
	}

	// An intrinsic should be an object, with a single key containing a valid intrinsic name
	if len(intrinsic) != 1 {
		return value, nil
	}

	var key string
	var val interface{}
	for _key, _val := range intrinsic {
		key = _key
		val = _val
	}

	switch key {
	case "Fn::Join":
		args, ok := val.([]interface{})
		if !ok {
			return "", fmt.Errorf("Fn::Join requires an array of arguments")
		}
		if len(args) != 2 {
			return "", fmt.Errorf("Fn::Join requires exactly two arguments")
		}
		sep, ok := args[0].(string)
		if !ok {
			return "", fmt.Errorf("Fn::Join requires a string separator")
		}
		elements, ok := args[1].([]interface{})
		if !ok {
			return "", fmt.Errorf("Fn::Join requires an array of elements")
		}

		parts := make([]string, len(elements))
		for i, element := range elements {
			asString, ok := element.(string)
			if !ok {
				return "", fmt.Errorf("Fn::Join requires an array of strings")
			}
			decoded, err := decodeIntrinsic(asString, refs)
			if err != nil {
				return "", err
			}
			parts[i] = decoded
		}
		return strings.Join(parts, sep), nil

	case "Ref":
		ref, ok := refs[val.(string)]
		if !ok {
			return "", fmt.Errorf("ref %s not found", val.(string))
		}
		return decodeIntrinsic(ref, refs)
	default:
		return "", fmt.Errorf("unknown intrinsic %s", key)
	}
}
