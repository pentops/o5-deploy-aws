package cflib

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
