package scalar

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

const (
	MaxSafeInteger int64  = 9_007_199_254_740_991
	MaxUint53      uint64 = 9_007_199_254_740_991
)

var (
	jsonIntegerPattern   = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
	jsonUnsignedPattern  = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
	decimalUint64Pattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,19})$`)
)

type SafeInteger struct{ value int64 }

func NewSafeInteger(value int64) (SafeInteger, error) {
	if value < -MaxSafeInteger || value > MaxSafeInteger {
		return SafeInteger{}, invalid("safe integer", "must lie in [-9007199254740991, 9007199254740991]")
	}
	return SafeInteger{value: value}, nil
}

func (value SafeInteger) Int64() int64 { return value.value }

func (value SafeInteger) MarshalJSON() ([]byte, error) {
	if _, err := NewSafeInteger(value.value); err != nil {
		return nil, err
	}
	return []byte(strconv.FormatInt(value.value, 10)), nil
}

func (value *SafeInteger) UnmarshalJSON(data []byte) error {
	candidate := strings.TrimSpace(string(data))
	if !jsonIntegerPattern.MatchString(candidate) {
		return invalid("safe integer", "must be an integral JSON number")
	}
	parsed, err := strconv.ParseInt(candidate, 10, 64)
	if err != nil {
		return invalid("safe integer", "must fit the interoperable safe interval")
	}
	validated, err := NewSafeInteger(parsed)
	if err != nil {
		return err
	}
	*value = validated
	return nil
}

type Uint53 struct{ value uint64 }

func NewUint53(value uint64) (Uint53, error) {
	if value > MaxUint53 {
		return Uint53{}, invalid("uint53", "must lie in [0, 9007199254740991]")
	}
	return Uint53{value: value}, nil
}

func (value Uint53) Uint64() uint64 { return value.value }

func (value Uint53) MarshalJSON() ([]byte, error) {
	if _, err := NewUint53(value.value); err != nil {
		return nil, err
	}
	return []byte(strconv.FormatUint(value.value, 10)), nil
}

func (value *Uint53) UnmarshalJSON(data []byte) error {
	candidate := strings.TrimSpace(string(data))
	if !jsonUnsignedPattern.MatchString(candidate) {
		return invalid("uint53", "must be an unsigned integral JSON number")
	}
	parsed, err := strconv.ParseUint(candidate, 10, 64)
	if err != nil {
		return invalid("uint53", "must fit the interoperable unsigned interval")
	}
	validated, err := NewUint53(parsed)
	if err != nil {
		return err
	}
	*value = validated
	return nil
}

type BoundedInteger struct {
	value       int64
	min         int64
	max         int64
	initialized bool
}

func NewBoundedInteger(value, min, max int64) (BoundedInteger, error) {
	if min < -MaxSafeInteger || max > MaxSafeInteger || min > max {
		return BoundedInteger{}, invalid("bounded integer", "bounds must form a safe-integer interval")
	}
	if value < min || value > max {
		return BoundedInteger{}, invalid("bounded integer", "value lies outside the schema interval")
	}
	return BoundedInteger{value: value, min: min, max: max, initialized: true}, nil
}

func DecodeBoundedIntegerJSON(data []byte, min, max int64) (BoundedInteger, error) {
	var value SafeInteger
	if err := json.Unmarshal(data, &value); err != nil {
		return BoundedInteger{}, err
	}
	return NewBoundedInteger(value.Int64(), min, max)
}

func (value BoundedInteger) Int64() int64 { return value.value }
func (value BoundedInteger) Min() int64   { return value.min }
func (value BoundedInteger) Max() int64   { return value.max }

func (value BoundedInteger) MarshalJSON() ([]byte, error) {
	if !value.initialized {
		return nil, invalid("bounded integer", "requires an explicit schema interval")
	}
	if _, err := NewBoundedInteger(value.value, value.min, value.max); err != nil {
		return nil, err
	}
	return []byte(strconv.FormatInt(value.value, 10)), nil
}

type DecimalUint64 struct{ value uint64 }

func NewDecimalUint64(value uint64) DecimalUint64 {
	return DecimalUint64{value: value}
}

func ParseDecimalUint64(value string) (DecimalUint64, error) {
	if !decimalUint64Pattern.MatchString(value) {
		return DecimalUint64{}, invalid("decimal_uint64", "must use canonical unsigned decimal string syntax")
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return DecimalUint64{}, invalid("decimal_uint64", "must not exceed 18446744073709551615")
	}
	return DecimalUint64{value: parsed}, nil
}

func (value DecimalUint64) Uint64() uint64 { return value.value }
func (value DecimalUint64) String() string { return strconv.FormatUint(value.value, 10) }

func (value DecimalUint64) MarshalJSON() ([]byte, error) {
	return json.Marshal(value.String())
}

func (value *DecimalUint64) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "decimal_uint64")
	if err != nil {
		return err
	}
	parsed, err := ParseDecimalUint64(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

var _ json.Marshaler = SafeInteger{}
var _ json.Unmarshaler = (*SafeInteger)(nil)
var _ json.Marshaler = Uint53{}
var _ json.Unmarshaler = (*Uint53)(nil)
var _ json.Marshaler = BoundedInteger{}
var _ json.Marshaler = DecimalUint64{}
var _ json.Unmarshaler = (*DecimalUint64)(nil)
