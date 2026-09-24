package merkleinventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var errStrictJSON = errors.New("invalid strict JSON shape")

// strictJSON is the package's only JSON decoding path. Canonicalize rejects
// duplicate members and malformed JSON before encoding/json sees the value.
// Its destination allowlist prevents a future caller from asking encoding/json
// to fill a struct using its case-insensitive field matching.
func strictJSON(data []byte, destination any) error {
	if _, err := canonicaljson.Canonicalize(data); err != nil {
		return fmt.Errorf("%w: %v", errStrictJSON, err)
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return errStrictJSON
	}
	switch destination.(type) {
	case *map[string]json.RawMessage:
		if trimmed[0] != '{' {
			return errStrictJSON
		}
	case *[]json.RawMessage:
		if trimmed[0] != '[' {
			return errStrictJSON
		}
	case *string:
		if trimmed[0] != '"' {
			return errStrictJSON
		}
	case *scalar.Uint53:
		if trimmed[0] < '0' || trimmed[0] > '9' {
			return errStrictJSON
		}
	default:
		return fmt.Errorf("%w: unsupported destination %T", errStrictJSON, destination)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("%w: %v", errStrictJSON, err)
	}
	return nil
}

func strictObject(data []byte) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := strictJSON(data, &object); err != nil || object == nil {
		return nil, errStrictJSON
	}
	return object, nil
}

func strictArray(data []byte) ([]json.RawMessage, error) {
	var values []json.RawMessage
	if err := strictJSON(data, &values); err != nil || values == nil {
		return nil, errStrictJSON
	}
	return values, nil
}

func exactMembers(object map[string]json.RawMessage, expected ...string) bool {
	if len(object) != len(expected) {
		return false
	}
	for _, key := range expected {
		if _, ok := object[key]; !ok {
			return false
		}
	}
	return true
}

func strictJSONString(data []byte) (string, bool) {
	var value string
	if err := strictJSON(data, &value); err != nil {
		return "", false
	}
	return value, true
}

func requiredJSONString(object map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := object[key]
	if !ok {
		return "", false
	}
	return strictJSONString(raw)
}

func strictStringArray(data []byte) ([]string, error) {
	rawValues, err := strictArray(data)
	if err != nil {
		return nil, errStrictJSON
	}
	values := make([]string, len(rawValues))
	for index, raw := range rawValues {
		value, ok := strictJSONString(raw)
		if !ok {
			return nil, errStrictJSON
		}
		values[index] = value
	}
	return values, nil
}

func requiredStringArray(object map[string]json.RawMessage, key string) ([]string, bool) {
	raw, ok := object[key]
	if !ok {
		return nil, false
	}
	values, err := strictStringArray(raw)
	return values, err == nil
}

func strictUint53(data []byte) (scalar.Uint53, bool) {
	var value scalar.Uint53
	if err := strictJSON(data, &value); err != nil {
		return scalar.Uint53{}, false
	}
	return value, true
}

func requiredUint53(object map[string]json.RawMessage, key string) (scalar.Uint53, bool) {
	raw, ok := object[key]
	if !ok {
		return scalar.Uint53{}, false
	}
	return strictUint53(raw)
}
