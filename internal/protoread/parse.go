package protoread

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pentops/j5/lib/j5codec"
	"github.com/pentops/j5/lib/j5reflect"
	"github.com/pentops/j5/lib/j5validate"
)

type s3API interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

var s3Client s3API

func getS3Client(ctx context.Context) (s3API, error) {
	if s3Client != nil {
		return s3Client, nil
	}
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	s3Client = s3.NewFromConfig(awsConfig)
	return s3Client, nil
}

func readFile(ctx context.Context, path string) ([]byte, error) {
	if strings.HasPrefix(path, "s3://") {
		client, err := getS3Client(ctx)
		if err != nil {
			return nil, err
		}
		bucket := strings.TrimPrefix(path, "s3://")
		parts := strings.SplitN(bucket, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid s3 path: %s", path)
		}
		res, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: &parts[0],
			Key:    &parts[1],
		})
		if err != nil {
			return nil, fmt.Errorf("get object: %w", err)
		}

		return io.ReadAll(res.Body)
	}
	return os.ReadFile(path)
}

func PullAndParse(ctx context.Context, filename string, into j5reflect.Object) error {
	data, err := readFile(ctx, filename)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filename, err)
	}
	err = Parse(filename, data, into)
	if err != nil {
		return fmt.Errorf("parsing file %s: %w", filename, err)
	}
	return nil
}

func Parse(filename string, data []byte, out j5reflect.Object) error {

	switch filepath.Ext(filename) {
	case ".yaml", ".yml":
		jsonData, err := yaml.YAMLToJSON(data)
		if err != nil {
			return fmt.Errorf("unmarshal %s: %w", filename, err)
		}
		err = j5codec.Global.JSONToReflect(jsonData, out)
		if err != nil {
			return fmt.Errorf("unmarshal %s: %w", filename, err)
		}

	case ".json":
		err := j5codec.Global.JSONToReflect(data, out)
		if err != nil {
			return fmt.Errorf("unmarshal %s: %w", filename, err)
		}

	default:
		return fmt.Errorf("unmarshal %s: unknown file extension %q", filename, filepath.Ext(filename))
	}

	// should usually be cached, but this is used rarely.
	validator := j5validate.NewValidator()

	if err := validator.Validate(out); err != nil {
		return err
	}

	return nil
}

var reLocation = regexp.MustCompile(`\(line (\d+):(\d+)\)`)

func findTokenError(data []byte, tknErr error) error {
	msg := tknErr.Error()

	match := reLocation.FindStringSubmatch(msg)
	if len(match) != 3 {
		return fmt.Errorf("no line match found for: %w", tknErr)
	}

	matchedLine, err := strconv.Atoi(match[1])
	if err != nil {
		return fmt.Errorf("%w: line number match: %w", tknErr, err)
	}

	lines := strings.Split(string(data), "\n")
	for idx, line := range lines {
		if idx+1 == matchedLine {
			token := strings.Trim(line, ":{} \t")
			return fmt.Errorf("matched token %s: %w", token, tknErr)
		}
	}

	return fmt.Errorf("no token error found")
}
