package scalar

import (
	"encoding/json"
	"regexp"
)

type Platform string

const (
	PlatformMacOS   Platform = "macos"
	PlatformLinux   Platform = "linux"
	PlatformWSL2    Platform = "wsl2"
	PlatformWindows Platform = "windows"
)

func ParsePlatform(value string) (Platform, error) {
	platform := Platform(value)
	switch platform {
	case PlatformMacOS, PlatformLinux, PlatformWSL2, PlatformWindows:
		return platform, nil
	default:
		return "", invalid("platform", "is not a member of the negotiated AX vocabulary")
	}
}

func (value Platform) String() string { return string(value) }

func (value Platform) MarshalText() ([]byte, error) {
	return marshalValidatedText(string(value), func(candidate string) error {
		_, err := ParsePlatform(candidate)
		return err
	})
}

func (value *Platform) UnmarshalText(data []byte) error {
	parsed, err := ParsePlatform(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value Platform) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(string(value), "platform", func(candidate string) error {
		_, err := ParsePlatform(candidate)
		return err
	})
}

func (value *Platform) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "platform")
	if err != nil {
		return err
	}
	parsed, err := ParsePlatform(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

var providerIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
var enumMemberPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type ProviderID struct{ value string }

func ParseProviderID(value string) (ProviderID, error) {
	if !providerIDPattern.MatchString(value) {
		return ProviderID{}, invalid("provider-id", "must match [a-z][a-z0-9-]{0,31}")
	}
	return ProviderID{value: value}, nil
}

func (value ProviderID) String() string { return value.value }

func (value ProviderID) MarshalText() ([]byte, error) {
	return marshalValidatedText(value.value, func(candidate string) error {
		_, err := ParseProviderID(candidate)
		return err
	})
}

func (value *ProviderID) UnmarshalText(data []byte) error {
	parsed, err := ParseProviderID(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value ProviderID) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "provider-id", func(candidate string) error {
		_, err := ParseProviderID(candidate)
		return err
	})
}

func (value *ProviderID) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "provider-id")
	if err != nil {
		return err
	}
	parsed, err := ParseProviderID(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

type ClosedEnum struct{ value string }

func ParseClosedEnum(value string, allowed ...string) (ClosedEnum, error) {
	seen := make(map[string]struct{}, len(allowed))
	for _, candidate := range allowed {
		if !enumMemberPattern.MatchString(candidate) {
			return ClosedEnum{}, invalid("closed-enum vocabulary", "members must use lower snake case")
		}
		if _, duplicate := seen[candidate]; duplicate {
			return ClosedEnum{}, invalid("closed-enum vocabulary", "must not contain duplicate members")
		}
		seen[candidate] = struct{}{}
	}
	if len(seen) == 0 {
		return ClosedEnum{}, invalid("closed-enum vocabulary", "must contain at least one member")
	}
	if _, ok := seen[value]; !ok {
		return ClosedEnum{}, invalid("closed-enum", "is not a member of the negotiated vocabulary")
	}
	return ClosedEnum{value: value}, nil
}

func DecodeClosedEnumJSON(data []byte, allowed ...string) (ClosedEnum, error) {
	value, err := decodeJSONString(data, "closed-enum")
	if err != nil {
		return ClosedEnum{}, err
	}
	return ParseClosedEnum(value, allowed...)
}

func (value ClosedEnum) String() string { return value.value }

func (value ClosedEnum) MarshalJSON() ([]byte, error) {
	if value.value == "" {
		return nil, invalid("closed-enum", "must be created from a non-empty negotiated vocabulary")
	}
	return json.Marshal(value.value)
}

var _ json.Marshaler = Platform("")
var _ json.Unmarshaler = (*Platform)(nil)
var _ json.Marshaler = ProviderID{}
var _ json.Unmarshaler = (*ProviderID)(nil)
var _ json.Marshaler = ClosedEnum{}
