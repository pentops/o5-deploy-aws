package protoread

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bufbuild/protovalidate-go"
	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func PullAndParse(ctx context.Context, filename string, into proto.Message) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filename, err)
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
