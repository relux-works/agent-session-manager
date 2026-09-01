package scalar

import (
	"encoding/json"
	"path"
	"strings"
	"unicode/utf8"
)

type RelativePath struct{ value string }

func ParseRelativePath(value string) (RelativePath, error) {
	if value == "" || !utf8.ValidString(value) {
		return RelativePath{}, invalid("path", "must be non-empty valid UTF-8")
	}
	if strings.HasPrefix(value, "/") || hasDrivePrefix(value) {
		return RelativePath{}, invalid("path", "must be platform-neutral and relative")
	}
	if strings.ContainsRune(value, 0) || strings.Contains(value, `\`) {
		return RelativePath{}, invalid("path", "must use forward slashes and contain no NUL")
	}
	if containsEncodedSeparator(value) {
		return RelativePath{}, invalid("path", "must not contain an encoded separator")
	}
	segments := strings.Split(value, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return RelativePath{}, invalid("path", "must not contain empty, dot, or parent segments")
		}
	}
	return RelativePath{value: value}, nil
}

func containsEncodedSeparator(value string) bool {
	candidate := strings.ToLower(value)
	for {
		if strings.Contains(candidate, "%2f") || strings.Contains(candidate, "%5c") {
			return true
		}
		next := strings.ReplaceAll(candidate, "%25", "%")
		if next == candidate {
			return false
		}
		candidate = next
	}
}

func hasDrivePrefix(value string) bool {
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func (value RelativePath) String() string { return value.value }

func (value RelativePath) MarshalText() ([]byte, error) {
	return marshalValidatedText(value.value, func(candidate string) error {
		_, err := ParseRelativePath(candidate)
		return err
	})
}

func (value *RelativePath) UnmarshalText(data []byte) error {
	parsed, err := ParseRelativePath(string(data))
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

func (value RelativePath) MarshalJSON() ([]byte, error) {
	return marshalValidatedString(value.value, "path", func(candidate string) error {
		_, err := ParseRelativePath(candidate)
		return err
	})
}

func (value *RelativePath) UnmarshalJSON(data []byte) error {
	candidate, err := decodeJSONString(data, "path")
	if err != nil {
		return err
	}
	parsed, err := ParseRelativePath(candidate)
	if err != nil {
		return err
	}
	*value = parsed
	return nil
}

type AbsolutePath struct {
	platform Platform
	value    string
}

func ParseAbsolutePath(platform Platform, value string) (AbsolutePath, error) {
	if _, err := ParsePlatform(platform.String()); err != nil {
		return AbsolutePath{}, err
	}
	if value == "" || !utf8.ValidString(value) || utf8.RuneCountInString(value) > 32767 || strings.ContainsRune(value, 0) {
		return AbsolutePath{}, invalid("absolute-path", "must be 1 to 32767 valid UTF-8 characters with no NUL")
	}

	var err error
	switch platform {
	case PlatformMacOS, PlatformLinux, PlatformWSL2:
		err = validatePOSIXAbsolutePath(value)
	case PlatformWindows:
		err = validateWindowsAbsolutePath(value)
	}
	if err != nil {
		return AbsolutePath{}, err
	}
	return AbsolutePath{platform: platform, value: value}, nil
}

func validatePOSIXAbsolutePath(value string) error {
	if !strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return invalid("absolute-path", "must be a lexically normalized POSIX absolute path")
	}
	return nil
}

func validateWindowsAbsolutePath(value string) error {
	if strings.Contains(value, "/") {
		return invalid("absolute-path", "must use native Windows separators")
	}
	if hasDrivePrefix(value) {
		if len(value) < 3 || value[2] != '\\' {
			return invalid("absolute-path", "must be drive-qualified or UNC")
		}
		return validateWindowsSegments(value[3:], true)
	}
	if !strings.HasPrefix(value, `\\`) {
		return invalid("absolute-path", "must be drive-qualified or UNC")
	}
	parts := strings.Split(value[2:], `\`)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return invalid("absolute-path", "UNC paths require server and share names")
	}
	for _, segment := range parts {
		if invalidWindowsSegment(segment) {
			return invalid("absolute-path", "must be lexically normalized without device or alternate-stream syntax")
		}
	}
	return nil
}

func validateWindowsSegments(value string, allowEmptyRoot bool) error {
	if value == "" && allowEmptyRoot {
		return nil
	}
	for _, segment := range strings.Split(value, `\`) {
		if invalidWindowsSegment(segment) {
			return invalid("absolute-path", "must be lexically normalized without device or alternate-stream syntax")
		}
	}
	return nil
}

func invalidWindowsSegment(segment string) bool {
	if segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
		return true
	}
	if strings.ContainsAny(segment, `<>:"|?*`) {
		return true
	}
	for _, character := range segment {
		if character >= 1 && character <= 31 {
			return true
		}
	}
	return reservedWindowsDeviceName(segment)
}

func reservedWindowsDeviceName(segment string) bool {
	base, _, _ := strings.Cut(segment, ".")
	base = strings.ToUpper(base)
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	return len(base) == 4 && base[3] >= '1' && base[3] <= '9' && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT"))
}

func (value AbsolutePath) String() string     { return value.value }
func (value AbsolutePath) Platform() Platform { return value.platform }

func (value AbsolutePath) MarshalJSON() ([]byte, error) {
	if _, err := ParseAbsolutePath(value.platform, value.value); err != nil {
		return nil, err
	}
	return json.Marshal(value.value)
}

func DecodeAbsolutePathJSON(platform Platform, data []byte) (AbsolutePath, error) {
	value, err := decodeJSONString(data, "absolute-path")
	if err != nil {
		return AbsolutePath{}, err
	}
	return ParseAbsolutePath(platform, value)
}

var _ json.Marshaler = RelativePath{}
var _ json.Unmarshaler = (*RelativePath)(nil)
var _ json.Marshaler = AbsolutePath{}
