package protoread

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ParseFile(filename string, into proto.Message) error {
	src, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	fileSuffix := filepath.Ext(filename)
	switch fileSuffix {
	case ".json":

	case ".yaml", ".yml":
		dec := yaml.NewDecoder(bytes.NewBuffer(src), yaml.UseOrderedMap())

		var v map[string]interface{}
		if err := dec.Decode(&v); err != nil {
			return fmt.Errorf("unmarshalling YAML %s %w", filename, err)
		}

		jsonBytes, err := yaml.MarshalWithOptions(v, yaml.JSON())
		if err != nil {
			return fmt.Errorf("failed to marshal with json option: %w", err)
		}
		src = jsonBytes

	default:
		return fmt.Errorf("unknown file type: %s", fileSuffix)
	}

	return protojson.Unmarshal(src, into)
}
