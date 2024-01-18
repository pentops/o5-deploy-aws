package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/awslabs/goformation/v7/cloudformation/ecs"
	"github.com/pentops/log.go/log"
	"github.com/pentops/log.go/pretty"
	"github.com/pentops/o5-deploy-aws/app"
	"github.com/pentops/o5-deploy-aws/deployer"
	"github.com/pentops/o5-deploy-aws/protoread"
	"github.com/pentops/o5-go/application/v1/application_pb"
	"github.com/pentops/o5-go/environment/v1/environment_pb"
	"github.com/pentops/o5-runtime-sidecar/entrypoint"
	"github.com/pentops/runner"
)

type flagConfig struct {
	envFilename string
	appFilename string
	workdir     string
	command     string
}

var prettyLogger = pretty.NewPrinter(os.Stdout)

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

	log.DefaultLogger = log.NewCallbackLogger(prettyLogger.CallbackWithPrefix("sidecar"))

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

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		sigInt := make(chan os.Signal, 1)
		signal.Notify(sigInt, os.Interrupt)
		<-sigInt
		log.Info(ctx, "Got Interrupt, Shutting down")
		cancel()
	}()

	runGroup := runner.NewGroup(runner.WithName("local"), runner.WithCancelOnSignals())

	var mainCommand *exec.Cmd

	container := mainRuntime.Containers[0]

	mainEnvVars, err := buildEnvironment(ctx, container.Container, refs, secretsManagerClient)
	if err != nil {
		return err
	}
	buildMain := func() *exec.Cmd {
		parts := strings.Split(flagConfig.command, " ")
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = flagConfig.workdir
		cmd.Stdout = prettyLogger.WriterInterceptor("STDOUT")
		cmd.Stderr = prettyLogger.WriterInterceptor("STDERR")
		cmd.Env = mainEnvVars

		return cmd
	}

	mainCommand = buildMain()

	runGroup.Add("main", func(ctx context.Context) error {
		go func() {
			<-ctx.Done()
			log.Info(ctx, "context canceled, killing command")
			if err := mainCommand.Process.Signal(os.Interrupt); err != nil {
				log.WithError(ctx, err).Error("failed to kill process")
			} else {
				log.Info(ctx, "killed process")
			}
		}()
		err := mainCommand.Run()
		if err != nil {
			return fmt.Errorf("command failed: %w", err)
		}
		log.Info(ctx, "command exited with no error")
		return nil
	})

	{

		ctx := log.WithField(ctx, "runtime", "sidecar")

		container := mainRuntime.AdapterContainer

		envVars, err := buildEnvironment(ctx, container, refs, secretsManagerClient)
		if err != nil {
			return err
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

		replacedEndpoints := cleanStringSplit(envMap["SERVICE_ENDPOINT"], ",")
		for idx, endpoint := range replacedEndpoints {
			endpoint = strings.ReplaceAll(endpoint, "main", "localhost")
			replacedEndpoints[idx] = endpoint
		}

		cfg := entrypoint.Config{
			PublicPort:        8888,
			Service:           replacedEndpoints,
			JWKS:              cleanStringSplit(envMap["JWKS"], ","),
			SNSPrefix:         envMap["SNS_PREFIX"],
			SQSURL:            envMap["SQS_URL"],
			CORSOrigins:       cleanStringSplit(envMap["CORS_ORIGINS"], ","),
			PostgresOutboxURI: envMap["POSTGRES_OUTBOX"],
		}

		sidecar, err := entrypoint.FromConfig(cfg, awsConfig)
		if err != nil {
			return fmt.Errorf("sidecar: %w", err)
		}

		runGroup.Add("sidecar", sidecar.Run)
	}

	return runGroup.Run(ctx)
}

func cleanStringSplit(src string, delim string) []string {
	parts := strings.Split(src, delim)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func buildEnvironment(ctx context.Context, container *ecs.TaskDefinition_ContainerDefinition, refs map[string]string, ssm *secretsmanager.Client) ([]string, error) {
	ctx = log.WithField(ctx, "container", container.Name)

	env := os.Environ()

	for _, envVar := range container.Environment {
		key := *envVar.Name
		value, err := decodeIntrinsic(*envVar.Value, refs)
		if err != nil {
			return nil, err
		}

		log.WithFields(ctx, map[string]interface{}{
			"key":   key,
			"value": value,
		}).Debug("setting environment variable")
		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}

	for _, secret := range container.Secrets {
		ctx := log.WithField(ctx, "secretName", secret.Name)
		key := secret.Name
		value, err := decodeIntrinsic(secret.ValueFrom, refs)
		if err != nil {
			return nil, err
		}
		ctx = log.WithField(ctx, "secretFrom", value)
		log.Debug(ctx, "fetching secret")

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
			if jsonKey != "" {

				secretMap := map[string]interface{}{}
				if err := json.Unmarshal([]byte(secretString), &secretMap); err != nil {
					return nil, fmt.Errorf("decoding secret %s: %w", value, err)
				}
				secretString = secretMap[jsonKey].(string)
			}
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
