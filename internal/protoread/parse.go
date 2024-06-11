package protoread

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bufbuild/protovalidate-go"
	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type S3Client interface {
	GetObject(ctx context.Context, input *s3.GetObjectInput, options ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

func PullAndParse(ctx context.Context, s3Client S3Client, filename string, into proto.Message) error {

	data, err := getFileBytes(ctx, s3Client, filename)
	if err != nil {
		return err
	}

	return Parse(filename, data, into)
}

func Parse(filename string, data []byte, into proto.Message) error {
	fileSuffix := filepath.Ext(filename)
	switch fileSuffix {
	case ".json":

	case ".yaml", ".yml":
		dec := yaml.NewDecoder(bytes.NewBuffer(data), yaml.UseOrderedMap())

		var v map[string]interface{}
		if err := dec.Decode(&v); err != nil {
			return fmt.Errorf("unmarshalling YAML %s %w", filename, err)
		}

		jsonBytes, err := yaml.MarshalWithOptions(v, yaml.JSON())
		if err != nil {
			return fmt.Errorf("failed to marshal with json option: %w", err)
		}
		out := &bytes.Buffer{}
		if err := json.Indent(out, jsonBytes, "", "  "); err != nil {
			return fmt.Errorf("failed to indent json: %w", err)
		}

		data = out.Bytes()

	default:
		return fmt.Errorf("unknown file type: %s", fileSuffix)
	}

	err := protojson.Unmarshal(data, into)
	if err != nil {
		return findTokenError(data, err)
	}

	// should usually be cached, but this is used rarely.
	validator, err := protovalidate.New()
	if err != nil {
		return fmt.Errorf("protovalidate.New: %w", err)
	}

	if err := validator.Validate(into); err != nil {
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

func getFileBytes(ctx context.Context, s3Client S3Client, filename string) ([]byte, error) {
	if strings.HasPrefix(filename, "s3://") {
		return getS3File(ctx, s3Client, filename)
	}

	return os.ReadFile(filename)
}

func getS3File(ctx context.Context, s3Client S3Client, filename string) ([]byte, error) {

	uri, err := url.Parse(filename)
	if err != nil {
		return nil, err
	}

	bucket := uri.Host
	key := strings.TrimPrefix(uri.Path, "/")

	input := &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	}

	output, err := s3Client.GetObject(context.Background(), input)
	if err != nil {
		return nil, fmt.Errorf("get s3 bucket: '%s' key: '%s': %w", bucket, key, err)
	}

	return io.ReadAll(output.Body)
}
