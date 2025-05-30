package cflib

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type FuncJoin struct {
	Join   string
	Values []any
}

func (f *FuncJoin) MarshalJSON() ([]byte, error) {
	out := make([]any, 2)
	out[0] = f.Join
	out[1] = f.Values
	return json.Marshal(out)
}

func (f *FuncJoin) UnmarshalJSON(data []byte) error {
	var out []any
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

	f.Values, ok = out[1].([]any)
	if !ok {
		return fmt.Errorf("expected []interface{}, got %T", out[1])
	}
	return err
}

type ParamMap map[string]string

func (p ParamMap) Get(key string) (string, bool) {
	val, ok := p[key]
	return val, ok
}

type Params interface {
	Get(key string) (string, bool)
}

type ParamFunc func(key string) (string, bool)

func (pf ParamFunc) Get(key string) (string, bool) {
	return pf(key)
}

func preEncode(raw any) (map[string]any, bool) {
	mapStringInterface, ok := raw.(map[string]any)
	if ok {
		return mapStringInterface, true
	}
	stringVal, ok := raw.(string)
	if ok {
		return fromBase64(stringVal)
	}
	attrVal, ok := raw.(TemplateRef)
	if ok {
		return fromBase64(string(attrVal))
	}

	return nil, false
}

func fromBase64(stringVal string) (map[string]any, bool) {
	b64, err := base64.StdEncoding.DecodeString(stringVal)
	if err != nil {
		return nil, false
	}

	val := map[string]any{}
	if err := json.Unmarshal(b64, &val); err != nil {
		return nil, false
	}
	return val, true
}

func ResolveFunc(raw any, params Params) (string, error) {
	msi, ok := preEncode(raw)
	if !ok {
		if str, ok := raw.(string); ok {
			return str, nil
		}
		return "", fmt.Errorf("expected map[string]interface{}, or string, or base64 encoded func got %T", raw)
	}

	var fnName string
	var fnValue any

	if len(msi) != 1 {
		return "", fmt.Errorf("expected 1 key, got %d", len(msi))
	}
	for key, value := range msi {
		fnName = key
		fnValue = value
	}

	switch fnName {
	case "Ref":
		refName, ok := fnValue.(string)
		if !ok {
			return "", fmt.Errorf("ref value should be string, got %T", fnValue)
		}
		if val, ok := params.Get(refName); ok {
			return val, nil
		}
		return "", fmt.Errorf("ref %s not found in params", refName)

	case "Fn::Join":
		joiner := &FuncJoin{}
		joinerValues, ok := fnValue.([]any)
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
		joiner.Values, ok = joinerValues[1].([]any)
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
