package scalar

import "encoding/json"

type UUIDv7 struct{ value string }
type UUIDv4 struct{ value string }

func ParseUUIDv7(value string) (UUIDv7, error) {
	if err := validateUUID(value, '7'); err != nil {
		return UUIDv7{}, err
	}
	return UUIDv7{value: value}, nil
}

func ParseUUIDv4(value string) (UUIDv4, error) {
	if err := validateUUID(value, '4'); err != nil {
		return UUIDv4{}, err
	}
	return UUIDv4{value: value}, nil
}

func validateUUID(value string, version byte) error {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return invalid("UUIDv"+string(version), "must use canonical 8-4-4-4-12 form")
	}
	for index := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			continue
		}
		if !isLowerHex(value[index]) {
			return invalid("UUIDv"+string(version), "must contain lowercase hexadecimal digits")
		}
	}
	if value[14] != version {
		return invalid("UUIDv"+string(version), "version nibble does not match the field type")
	}
	if value[19] != '8' && value[19] != '9' && value[19] != 'a' && value[19] != 'b' {
		return invalid("UUIDv"+string(version), "must use the RFC 4122 variant")
	}
	return nil
}

func isLowerHex(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f'
}

func (value UUIDv7) String() string { return value.value }
func (value UUIDv4) String() string { return value.value }

func (value UUIDv7) MarshalText() ([]byte, error) {
	return marshalValidatedText(value.value, func(candidate string) error { return validateUUID(candidate, '7') })
}

func (value *UUIDv7) UnmarshalText(data []byte) error {
	parsed, err := ParseUUIDv7(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value UUIDv4) MarshalText() ([]byte, error) {
	return marshalValidatedText(value.value, func(candidate string) error { return validateUUID(candidate, '4') })
}

func (value *UUIDv4) UnmarshalText(data []byte) error {
	parsed, err := ParseUUIDv4(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value UUIDv7) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "UUIDv7", func(candidate string) error {
		return validateUUID(candidate, '7')
	})
}

func (value *UUIDv7) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "UUIDv7")
	if err != nil {
		return err
	}
	parsed, err := ParseUUIDv7(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value UUIDv4) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "UUIDv4", func(candidate string) error {
		return validateUUID(candidate, '4')
	})
}

func (value *UUIDv4) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "UUIDv4")
	if err != nil {
		return err
	}
	parsed, err := ParseUUIDv4(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

var _ json.Marshaler = UUIDv7{}
var _ json.Unmarshaler = (*UUIDv7)(nil)
var _ json.Marshaler = UUIDv4{}
var _ json.Unmarshaler = (*UUIDv4)(nil)
