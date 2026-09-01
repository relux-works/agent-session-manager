// Package scalar implements the common AX wire value types from Specification
// 1.6. Types whose validity depends on a containing contract require that
// context at their decode entry point instead of guessing it.
package scalar

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// ErrInvalidScalar identifies malformed or out-of-contract scalar values.
var ErrInvalidScalar = errors.New("invalid AX scalar")

// ValidationError reports the value kind and failed rule without echoing the
// input, which may be a machine-local path.
type ValidationError struct {
	Kind   string
	Reason string
}

func (err *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s: %v", err.Kind, err.Reason, ErrInvalidScalar)
}

func (err *ValidationError) Unwrap() error {
	return ErrInvalidScalar
}

func invalid(kind, reason string) error {
	return &ValidationError{Kind: kind, Reason: reason}
}

func decodeJSONString(data []byte, kind string) (string, error) {
	if !utf8.Valid(data) {
		return "", invalid(kind, "JSON text must be valid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var value *string
	if err := decoder.Decode(&value); err != nil {
		return "", invalid(kind, "must be a JSON string")
	}
	if value == nil {
		return "", invalid(kind, "must not be null")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return "", invalid(kind, "must contain one JSON value")
	}
	if hasLoneJSONSurrogate(strings.TrimSpace(string(data))) {
		return "", invalid(kind, "must not contain a lone Unicode surrogate")
	}
	return *value, nil
}

func hasLoneJSONSurrogate(value string) bool {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return false
	}
	for index := 1; index < len(value)-1; index++ {
		if value[index] != '\\' || index+1 >= len(value)-1 {
			continue
		}
		if value[index+1] != 'u' {
			index++
			continue
		}
		code, ok := parseHex16(value[index+2:])
		if !ok {
			return false // json.Decoder already rejects malformed escapes.
		}
		index += 5
		switch {
		case code >= 0xdc00 && code <= 0xdfff:
			return true
		case code >= 0xd800 && code <= 0xdbff:
			if index+6 >= len(value) || value[index+1:index+3] != `\u` {
				return true
			}
			low, ok := parseHex16(value[index+3:])
			if !ok || low < 0xdc00 || low > 0xdfff {
				return true
			}
			index += 6
		}
	}
	return false
}

func parseHex16(value string) (uint16, bool) {
	if len(value) < 4 {
		return 0, false
	}
	var result uint16
	for index := 0; index < 4; index++ {
		result <<= 4
		switch digit := value[index]; {
		case digit >= '0' && digit <= '9':
			result += uint16(digit - '0')
		case digit >= 'a' && digit <= 'f':
			result += uint16(digit-'a') + 10
		case digit >= 'A' && digit <= 'F':
			result += uint16(digit-'A') + 10
		default:
			return 0, false
		}
	}
	return result, true
}

func marshalValidatedString(value, kind string, validate func(string) error) ([]byte, error) {
	if err := validate(value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func marshalValidatedText(value string, validate func(string) error) ([]byte, error) {
	if err := validate(value); err != nil {
		return nil, err
	}
	return []byte(value), nil
}
