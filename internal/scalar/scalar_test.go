package scalar

import (
	"encoding"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestUUIDValuesAcceptCanonicalVersionsAndRefuseMalformedInput(t *testing.T) {
	t.Parallel()

	uuid7Text := "0198f4c8-8e50-7f66-8f70-1234567890ab"
	uuid7, err := ParseUUIDv7(uuid7Text)
	if err != nil || uuid7.String() != uuid7Text {
		t.Fatalf("ParseUUIDv7(%q) = %q, %v", uuid7Text, uuid7, err)
	}
	uuid4Text := "550e8400-e29b-41d4-a716-446655440000"
	uuid4, err := ParseUUIDv4(uuid4Text)
	if err != nil || uuid4.String() != uuid4Text {
		t.Fatalf("ParseUUIDv4(%q) = %q, %v", uuid4Text, uuid4, err)
	}

	if _, err := ParseUUIDv7(uuid4Text); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("ParseUUIDv7(UUIDv4) error = %v, want ErrInvalidScalar", err)
	}
	if _, err := ParseUUIDv4(uuid7Text); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("ParseUUIDv4(UUIDv7) error = %v, want ErrInvalidScalar", err)
	}

	invalid := []string{
		"0198F4c8-8e50-7f66-8f70-1234567890ab",
		"0198f4c8-8e50-7f66-7f70-1234567890ab",
		"0198f4c8-8e50-7f66-cf70-1234567890ab",
		"0198f4c88e507f668f701234567890ab",
		"0198f4c8-8e50-7f66-8f70-1234567890ag",
	}
	for _, value := range invalid {
		if _, err := ParseUUIDv7(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseUUIDv7(%q) error = %v, want ErrInvalidScalar", value, err)
		}
	}

	encoded, err := json.Marshal(uuid7)
	if err != nil || string(encoded) != `"0198f4c8-8e50-7f66-8f70-1234567890ab"` {
		t.Fatalf("Marshal(UUIDv7) = %s, %v", encoded, err)
	}
	var decoded UUIDv7
	if err := json.Unmarshal(encoded, &decoded); err != nil || decoded != uuid7 {
		t.Fatalf("Unmarshal(UUIDv7) = %q, %v", decoded, err)
	}
	if err := json.Unmarshal([]byte("null"), &decoded); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("Unmarshal(null) error = %v, want ErrInvalidScalar", err)
	}
}

func TestTimestampAcceptsRealUTCRFC3339WithMillisecondPrecision(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"2024-02-29T23:59:59.000Z",
		"2026-08-31T14:05:06.123456789Z",
		"2026-08-31T14:05:06.1234567890Z",
		"2026-08-31T14:05:06." + strings.Repeat("1234567890", 10) + "Z",
		"2026-08-31t14:05:06.123z",
		"2026-08-31T14:05:06.123+00:00",
	} {
		got, err := ParseTimestamp(value)
		if err != nil || got.String() != value {
			t.Errorf("ParseTimestamp(%q) = %q, %v", value, got, err)
			continue
		}
		instant, err := got.Time()
		if err != nil || instant.Location() != time.UTC {
			t.Errorf("Timestamp(%q).Time() = %v, %v", value, instant, err)
		}
		var jsonDecoded Timestamp
		if err := json.Unmarshal([]byte(`"`+value+`"`), &jsonDecoded); err != nil || jsonDecoded.String() != value {
			t.Errorf("Unmarshal Timestamp(%q) = %q, %v", value, jsonDecoded, err)
		}
		var textDecoded Timestamp
		if err := textDecoded.UnmarshalText([]byte(value)); err != nil || textDecoded.String() != value {
			t.Errorf("UnmarshalText Timestamp(%q) = %q, %v", value, textDecoded, err)
		}
	}

	for _, value := range []string{
		"2023-02-29T12:00:00.000Z",
		"2026-08-31T14:05:06Z",
		"2026-08-31T14:05:06.12Z",
		"2026-08-31T14:05:06.123+03:00",
		"2026-08-31T14:05:06.123-00:00",
		"2026-08-31T24:00:00.000Z",
	} {
		if _, err := ParseTimestamp(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseTimestamp(%q) error = %v, want ErrInvalidScalar", value, err)
		}
	}
}

// TestTimestampAcceptsPublishedLeapSecondsAndRefusesFabricatedOnes drives the
// ParseTimestamp production entry directly and through both wire decoders.
func TestTimestampAcceptsPublishedLeapSecondsAndRefusesFabricatedOnes(t *testing.T) {
	t.Parallel()

	const published = "1990-12-31T23:59:60.000Z"
	parsed, err := ParseTimestamp(published)
	if err != nil || parsed.String() != published {
		t.Fatalf("ParseTimestamp(%q) = %q, %v", published, parsed, err)
	}
	instant, err := parsed.Time()
	wantInstant := time.Date(1991, time.January, 1, 0, 0, 0, 0, time.UTC)
	if err != nil || !instant.Equal(wantInstant) {
		t.Fatalf("Timestamp(%q).Time() = %v, %v, want %v", published, instant, err, wantInstant)
	}

	var fromJSON Timestamp
	if err := json.Unmarshal([]byte(`"1990-12-31T23:59:60.000Z"`), &fromJSON); err != nil || fromJSON != parsed {
		t.Fatalf("Unmarshal leap-second timestamp = %q, %v", fromJSON, err)
	}
	var fromText Timestamp
	if err := fromText.UnmarshalText([]byte(published)); err != nil || fromText != parsed {
		t.Fatalf("UnmarshalText leap-second timestamp = %q, %v", fromText, err)
	}

	for _, fabricated := range []string{
		"1990-12-31T23:58:60.000Z", // an ordinary minute on a real leap-second date
		"1990-12-30T23:59:60.000Z", // no leap second was published for this date
		"2026-12-31T23:59:60.000Z", // no leap second is published for this date
	} {
		if _, err := ParseTimestamp(fabricated); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseTimestamp(%q) error = %v, want ErrInvalidScalar", fabricated, err)
		}
		if err := json.Unmarshal([]byte(`"`+fabricated+`"`), &fromJSON); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("Unmarshal leap-second timestamp %q error = %v, want ErrInvalidScalar", fabricated, err)
		}
		if err := fromText.UnmarshalText([]byte(fabricated)); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("UnmarshalText leap-second timestamp %q error = %v, want ErrInvalidScalar", fabricated, err)
		}
	}
}

func TestDigestAcceptsExactSHA256AndHashesRawBytes(t *testing.T) {
	t.Parallel()

	want := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	digest := SHA256Digest(nil)
	if digest.String() != want || digest.Hex() != strings.TrimPrefix(want, "sha256:") {
		t.Fatalf("SHA256Digest(nil) = %q (%q), want %q", digest, digest.Hex(), want)
	}
	parsed, err := ParseDigest(want)
	if err != nil || parsed != digest {
		t.Fatalf("ParseDigest(%q) = %q, %v", want, parsed, err)
	}

	for _, value := range []string{
		strings.TrimPrefix(want, "sha256:"),
		"sha512:" + strings.TrimPrefix(want, "sha256:"),
		"sha256:" + strings.ToUpper(strings.TrimPrefix(want, "sha256:")),
		want[:len(want)-1],
		want + "0",
	} {
		if _, err := ParseDigest(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseDigest(%q) error = %v, want ErrInvalidScalar", value, err)
		}
	}
}

func TestPlatformAndProviderIDAreClosedAndBounded(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"macos", "linux", "wsl2", "windows"} {
		platform, err := ParsePlatform(value)
		if err != nil || platform.String() != value {
			t.Errorf("ParsePlatform(%q) = %q, %v", value, platform, err)
		}
	}
	for _, value := range []string{"darwin", "MacOS", "freebsd", ""} {
		if _, err := ParsePlatform(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParsePlatform(%q) error = %v, want ErrInvalidScalar", value, err)
		}
	}

	for _, value := range []string{"codex", "a", "a-b", "a" + strings.Repeat("1", 31)} {
		provider, err := ParseProviderID(value)
		if err != nil || provider.String() != value {
			t.Errorf("ParseProviderID(%q) = %q, %v", value, provider, err)
		}
	}
	for _, value := range []string{"", "Codex", "1codex", "co_dex", "a" + strings.Repeat("1", 32)} {
		if _, err := ParseProviderID(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseProviderID(%q) error = %v, want ErrInvalidScalar", value, err)
		}
	}
}

func TestRelativePathRefusesTraversalAndEncodedSeparators(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"AGENTS.md", "a/b/c", "данные/файл.txt", "percent%20name"} {
		path, err := ParseRelativePath(value)
		if err != nil || path.String() != value {
			t.Errorf("ParseRelativePath(%q) = %q, %v", value, path, err)
		}
	}
	invalid := []string{
		"", ".", "..", "/absolute", "C:/drive", "a//b", "a/./b", "a/../b",
		"a/", `a\b`, "a%2fb", "a%2Fb", "a%5cb", "a%5Cb", "a%252fb", "a\x00b", string([]byte{0xff}),
	}
	for _, value := range invalid {
		if _, err := ParseRelativePath(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseRelativePath(%q) error = %v, want ErrInvalidScalar", value, err)
		}
	}
}

func TestAbsolutePathUsesContainingPlatformGrammar(t *testing.T) {
	t.Parallel()

	accepted := []struct {
		platform Platform
		value    string
	}{
		{PlatformMacOS, "/"},
		{PlatformLinux, "/srv/relux"},
		{PlatformWSL2, "/mnt/c/Developer/ReluxWorks"},
		{PlatformWindows, `C:\Developer\ReluxWorks`},
		{PlatformWindows, `\\server\share\ReluxWorks`},
	}
	for _, test := range accepted {
		got, err := ParseAbsolutePath(test.platform, test.value)
		if err != nil || got.String() != test.value || got.Platform() != test.platform {
			t.Errorf("ParseAbsolutePath(%q, %q) = %#v, %v", test.platform, test.value, got, err)
		}
	}

	rejected := []struct {
		platform Platform
		value    string
	}{
		{PlatformLinux, "relative/path"},
		{PlatformLinux, "/srv/../etc"},
		{PlatformLinux, "/srv//data"},
		{PlatformLinux, "/srv/data/"},
		{PlatformWindows, `C:relative`},
		{PlatformWindows, `C:/mixed/separators`},
		{PlatformWindows, `C:\dir\..\target`},
		{PlatformWindows, `\\server`},
		{PlatformWindows, `\\?\C:\device`},
	}
	for _, test := range rejected {
		if _, err := ParseAbsolutePath(test.platform, test.value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseAbsolutePath(%q, %q) error = %v, want ErrInvalidScalar", test.platform, test.value, err)
		}
	}
}

// TestWindowsAbsolutePathRefusesWin32InvalidComponentsAtEveryBoundary drives
// ParseAbsolutePath, DecodeAbsolutePathJSON, and AbsolutePath.MarshalJSON. The
// cases come from measured Win32 behavior and cover both drive and UNC paths.
func TestWindowsAbsolutePathRefusesWin32InvalidComponentsAtEveryBoundary(t *testing.T) {
	t.Parallel()

	rejected := []string{
		`C:\unsafe\CON`,
		`C:\unsafe\con.txt`,
		`C:\unsafe\PRN.json`,
		`C:\unsafe\AUX.log`,
		`C:\unsafe\NUL.txt`,
		`C:\unsafe\COM1`,
		`C:\unsafe\com9.any`,
		`C:\unsafe\LPT1`,
		`C:\unsafe\lpt9.any`,
		`C:\unsafe\star*.json`,
		`C:\unsafe\q?b`,
		`C:\unsafe\less<than`,
		`C:\unsafe\greater>than`,
		`C:\unsafe\stream:name`,
		`C:\unsafe\double"quote`,
		`C:\unsafe\pipe|name`,
		`\\server\share\COM1`,
		`\\server\share\NUL.txt`,
		`\\server\share\q?b`,
	}
	for _, value := range rejected {
		assertWindowsAbsolutePathRejectedAtEveryBoundary(t, value)
	}

	for control := byte(1); control <= 31; control++ {
		assertWindowsAbsolutePathRejectedAtEveryBoundary(t, `C:\unsafe\a`+string(control)+`b`)
		assertWindowsAbsolutePathRejectedAtEveryBoundary(t, `\\server\share\a`+string(control)+`b`)
	}
}

func assertWindowsAbsolutePathRejectedAtEveryBoundary(t *testing.T, value string) {
	t.Helper()

	if _, err := ParseAbsolutePath(PlatformWindows, value); !errors.Is(err, ErrInvalidScalar) {
		t.Errorf("ParseAbsolutePath(PlatformWindows, %q) error = %v, want ErrInvalidScalar", value, err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%q) error = %v", value, err)
	}
	if _, err := DecodeAbsolutePathJSON(PlatformWindows, encoded); !errors.Is(err, ErrInvalidScalar) {
		t.Errorf("DecodeAbsolutePathJSON(PlatformWindows, %q) error = %v, want ErrInvalidScalar", value, err)
	}
	forged := AbsolutePath{platform: PlatformWindows, value: value}
	if _, err := json.Marshal(forged); !errors.Is(err, ErrInvalidScalar) {
		t.Errorf("json.Marshal(AbsolutePath{%q}) error = %v, want ErrInvalidScalar", value, err)
	}
}

func TestAbsolutePathComponentRulesRemainPlatformSpecific(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		`C:\safe\CONSOLE`,
		`C:\safe\con-file.txt`,
		`C:\safe\COM0`,
		`C:\safe\COM10`,
		`C:\safe\LPT0`,
		`C:\safe\LPT10`,
		`\\server\share\normal.txt`,
	} {
		if _, err := ParseAbsolutePath(PlatformWindows, value); err != nil {
			t.Errorf("ParseAbsolutePath(PlatformWindows, %q) error = %v", value, err)
		}
	}

	const posixValue = `/srv/CON/star*.json/q?b/a:b/less<than/greater>than/double"quote/pipe|name`
	if got, err := ParseAbsolutePath(PlatformLinux, posixValue); err != nil || got.String() != posixValue {
		t.Errorf("ParseAbsolutePath(PlatformLinux, %q) = %q, %v", posixValue, got, err)
	}
}

func TestSafeAndBoundedIntegersRefuseNarrowedAndUnsafeValues(t *testing.T) {
	t.Parallel()

	for _, value := range []int64{-MaxSafeInteger, 0, MaxSafeInteger} {
		got, err := NewSafeInteger(value)
		if err != nil || got.Int64() != value {
			t.Errorf("NewSafeInteger(%d) = %d, %v", value, got.Int64(), err)
		}
	}
	for _, value := range []int64{-MaxSafeInteger - 1, MaxSafeInteger + 1} {
		if _, err := NewSafeInteger(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("NewSafeInteger(%d) error = %v, want ErrInvalidScalar", value, err)
		}
	}

	for _, data := range []string{"9007199254740992", "9007199254740993", "1.0", "1e0", `"1"`, "null"} {
		var value SafeInteger
		if err := json.Unmarshal([]byte(data), &value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("Unmarshal SafeInteger %s error = %v, want ErrInvalidScalar", data, err)
		}
	}

	bounded, err := NewBoundedInteger(4_194_304, 1, 4_194_304)
	if err != nil || bounded.Int64() != 4_194_304 {
		t.Fatalf("NewBoundedInteger(max) = %d, %v", bounded.Int64(), err)
	}
	for _, value := range []int64{0, 4_194_305} {
		if _, err := NewBoundedInteger(value, 1, 4_194_304); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("NewBoundedInteger(%d, narrowed bound) error = %v, want ErrInvalidScalar", value, err)
		}
	}
	if _, err := NewBoundedInteger(1, 2, 1); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("invalid bounds error = %v, want ErrInvalidScalar", err)
	}
	explicitZero, err := NewBoundedInteger(0, 0, 0)
	if err != nil {
		t.Fatalf("NewBoundedInteger(0, 0, 0) error = %v", err)
	}
	if encoded, err := json.Marshal(explicitZero); err != nil || string(encoded) != "0" {
		t.Fatalf("Marshal(explicit [0,0] bounded integer) = %s, %v", encoded, err)
	}

	var uint53 Uint53
	if err := json.Unmarshal([]byte("9007199254740991"), &uint53); err != nil || uint53.Uint64() != MaxUint53 {
		t.Fatalf("Unmarshal Uint53(max) = %d, %v", uint53.Uint64(), err)
	}
	for _, data := range []string{"-1", "9007199254740992", "1.0"} {
		if err := json.Unmarshal([]byte(data), &uint53); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("Unmarshal Uint53 %s error = %v, want ErrInvalidScalar", data, err)
		}
	}

	for name, value := range map[string]any{
		"safe":    mustSafeInteger(t, -42),
		"uint53":  uint53,
		"bounded": bounded,
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Errorf("Marshal(%s) error = %v", name, err)
		}
		if len(encoded) == 0 || encoded[0] == '"' {
			t.Errorf("Marshal(%s) = %s, want JSON number", name, encoded)
		}
	}
	decoded, err := DecodeBoundedIntegerJSON([]byte("1"), 1, 4_194_304)
	if err != nil || decoded.Int64() != 1 || decoded.Min() != 1 || decoded.Max() != 4_194_304 {
		t.Fatalf("DecodeBoundedIntegerJSON() = %#v, %v", decoded, err)
	}
	if _, err := DecodeBoundedIntegerJSON([]byte("0"), 1, 4_194_304); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("DecodeBoundedIntegerJSON(underflow) error = %v, want ErrInvalidScalar", err)
	}
}

func mustSafeInteger(t *testing.T, value int64) SafeInteger {
	t.Helper()
	result, err := NewSafeInteger(value)
	if err != nil {
		t.Fatalf("NewSafeInteger(%d) error = %v", value, err)
	}
	return result
}

func TestDecimalUint64UsesCanonicalStringEncoding(t *testing.T) {
	t.Parallel()

	value, err := ParseDecimalUint64("18446744073709551615")
	if err != nil || value.Uint64() != ^uint64(0) {
		t.Fatalf("ParseDecimalUint64(max) = %d, %v", value.Uint64(), err)
	}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) != `"18446744073709551615"` {
		t.Fatalf("Marshal DecimalUint64 = %s, %v", encoded, err)
	}
	for _, input := range []string{"01", "+1", "18446744073709551616", "-1", ""} {
		if _, err := ParseDecimalUint64(input); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseDecimalUint64(%q) error = %v, want ErrInvalidScalar", input, err)
		}
	}
	if err := json.Unmarshal([]byte("1"), &value); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("numeric DecimalUint64 error = %v, want ErrInvalidScalar", err)
	}
	zero := NewDecimalUint64(0)
	if zero.String() != "0" {
		t.Fatalf("NewDecimalUint64(0) = %q", zero)
	}
}

func TestClosedEnumRequiresExactVersionedVocabulary(t *testing.T) {
	t.Parallel()

	allowed := []string{"workspace_group", "workspace_tree", "provider", "task_board", "composite"}
	value, err := ParseClosedEnum("provider", allowed...)
	if err != nil || value.String() != "provider" {
		t.Fatalf("ParseClosedEnum(provider) = %q, %v", value, err)
	}
	for _, input := range []string{"Provider", "unknown", ""} {
		if _, err := ParseClosedEnum(input, allowed...); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("ParseClosedEnum(%q) error = %v, want ErrInvalidScalar", input, err)
		}
	}
	if _, err := ParseClosedEnum("provider", "provider", "provider"); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("duplicate vocabulary error = %v, want ErrInvalidScalar", err)
	}
	if _, err := ParseClosedEnum("provider", "provider", "not-kebab"); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("non-snake vocabulary error = %v, want ErrInvalidScalar", err)
	}
	decoded, err := DecodeClosedEnumJSON([]byte(`"task_board"`), allowed...)
	if err != nil || decoded.String() != "task_board" {
		t.Fatalf("DecodeClosedEnumJSON() = %q, %v", decoded, err)
	}
	if _, err := DecodeClosedEnumJSON([]byte("null"), allowed...); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("DecodeClosedEnumJSON(null) error = %v, want ErrInvalidScalar", err)
	}
	encoded, err := json.Marshal(decoded)
	if err != nil || string(encoded) != `"task_board"` {
		t.Fatalf("Marshal(ClosedEnum) = %s, %v", encoded, err)
	}
}

func TestValidatedScalarJSONRoundTrips(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		value  any
		target any
	}{
		{"uuidv4", mustUUIDv4(t, "550e8400-e29b-41d4-a716-446655440000"), &UUIDv4{}},
		{"timestamp", mustTimestamp(t, "2026-08-31T14:05:06.123Z"), &Timestamp{}},
		{"digest", SHA256Digest([]byte("ax")), &Digest{}},
		{"platform", PlatformLinux, new(Platform)},
		{"provider", mustProviderID(t, "codex"), &ProviderID{}},
		{"relative path", mustRelativePath(t, "dir/file"), &RelativePath{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if err := json.Unmarshal(encoded, test.target); err != nil {
				t.Fatalf("json.Unmarshal(%s) error = %v", encoded, err)
			}
		})
	}

	absolute, err := ParseAbsolutePath(PlatformWindows, `C:\Developer\ReluxWorks`)
	if err != nil {
		t.Fatalf("ParseAbsolutePath() error = %v", err)
	}
	encoded, err := json.Marshal(absolute)
	if err != nil {
		t.Fatalf("Marshal(AbsolutePath) error = %v", err)
	}
	decoded, err := DecodeAbsolutePathJSON(PlatformWindows, encoded)
	if err != nil || decoded.String() != absolute.String() {
		t.Fatalf("DecodeAbsolutePathJSON(%s) = %#v, %v", encoded, decoded, err)
	}
}

func TestIdentifierTypesRoundTripAsJSONMapKeys(t *testing.T) {
	t.Parallel()

	uuid := mustUUIDv4(t, "550e8400-e29b-41d4-a716-446655440000")
	encoded, err := json.Marshal(map[UUIDv4]string{uuid: "lease"})
	if err != nil {
		t.Fatalf("Marshal(map[UUIDv4]string) error = %v", err)
	}
	var decoded map[UUIDv4]string
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal(map[UUIDv4]string) error = %v", err)
	}
	if decoded[uuid] != "lease" {
		t.Fatalf("decoded map = %#v, want UUID key", decoded)
	}

	provider := mustProviderID(t, "codex")
	encoded, err = json.Marshal(map[ProviderID]int{provider: 1})
	if err != nil {
		t.Fatalf("Marshal(map[ProviderID]int) error = %v", err)
	}
	var providers map[ProviderID]int
	if err := json.Unmarshal(encoded, &providers); err != nil || providers[provider] != 1 {
		t.Fatalf("Unmarshal(map[ProviderID]int) = %#v, %v", providers, err)
	}
	if err := json.Unmarshal([]byte(`{"Codex":1}`), &providers); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("Unmarshal(invalid provider map key) error = %v, want ErrInvalidScalar", err)
	}
}

func TestValidatedStringTypesSupportTextBoundaries(t *testing.T) {
	t.Parallel()

	uuid7, err := ParseUUIDv7("0198f4c8-8e50-7f66-8f70-1234567890ab")
	if err != nil {
		t.Fatal(err)
	}
	timestamp := mustTimestamp(t, "2026-08-31T14:05:06.123Z")
	digest := SHA256Digest([]byte("ax"))
	provider := mustProviderID(t, "codex")
	relative := mustRelativePath(t, "dir/file")
	tests := []struct {
		name   string
		value  encoding.TextMarshaler
		target encoding.TextUnmarshaler
		want   string
	}{
		{"uuidv7", uuid7, &UUIDv7{}, uuid7.String()},
		{"timestamp", timestamp, &Timestamp{}, timestamp.String()},
		{"digest", digest, &Digest{}, digest.String()},
		{"platform", PlatformLinux, new(Platform), PlatformLinux.String()},
		{"provider", provider, &ProviderID{}, provider.String()},
		{"relative path", relative, &RelativePath{}, relative.String()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := test.value.MarshalText()
			if err != nil || string(encoded) != test.want {
				t.Fatalf("MarshalText() = %q, %v", encoded, err)
			}
			if err := test.target.UnmarshalText(encoded); err != nil {
				t.Fatalf("UnmarshalText(%q) error = %v", encoded, err)
			}
			got := test.target.(interface{ String() string }).String()
			if got != test.want {
				t.Fatalf("text round trip = %q, want %q", got, test.want)
			}
		})
	}
}

func mustUUIDv4(t *testing.T, value string) UUIDv4 {
	t.Helper()
	result, err := ParseUUIDv4(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mustTimestamp(t *testing.T, value string) Timestamp {
	t.Helper()
	result, err := ParseTimestamp(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mustProviderID(t *testing.T, value string) ProviderID {
	t.Helper()
	result, err := ParseProviderID(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mustRelativePath(t *testing.T, value string) RelativePath {
	t.Helper()
	result, err := ParseRelativePath(value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestJSONDecodersCannotBypassScalarValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target any
		data   string
	}{
		{"uuidv7", &UUIDv7{}, `"0198F4c8-8e50-7f66-8f70-1234567890ab"`},
		{"uuidv4", &UUIDv4{}, `"0198f4c8-8e50-7f66-8f70-1234567890ab"`},
		{"timestamp", &Timestamp{}, `"2023-02-29T12:00:00.000Z"`},
		{"digest", &Digest{}, `"sha256:E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855"`},
		{"platform", new(Platform), `"darwin"`},
		{"provider", &ProviderID{}, `"Codex"`},
		{"relative path", &RelativePath{}, `"a/../b"`},
		{"safe integer", &SafeInteger{}, `9007199254740992`},
		{"uint53", &Uint53{}, `9007199254740992`},
		{"decimal uint64", &DecimalUint64{}, `1`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := json.Unmarshal([]byte(test.data), test.target); !errors.Is(err, ErrInvalidScalar) {
				t.Fatalf("json.Unmarshal(%s) error = %v, want ErrInvalidScalar", test.data, err)
			}
		})
	}

	if _, err := DecodeAbsolutePathJSON(PlatformWindows, []byte(`"/posix"`)); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("DecodeAbsolutePathJSON() error = %v, want ErrInvalidScalar", err)
	}
	if _, err := DecodeClosedEnumJSON([]byte(`"unknown"`), "known"); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("DecodeClosedEnumJSON() error = %v, want ErrInvalidScalar", err)
	}
	if _, err := DecodeClosedEnumJSON([]byte(`"\ud800"`), "known"); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("DecodeClosedEnumJSON(lone surrogate) error = %v, want ErrInvalidScalar", err)
	}
	if _, err := DecodeClosedEnumJSON([]byte{'"', 0xff, '"'}, "known"); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("DecodeClosedEnumJSON(invalid UTF-8) error = %v, want ErrInvalidScalar", err)
	}
	var unicodePath RelativePath
	if err := json.Unmarshal([]byte(`"\ud83d\ude00/file"`), &unicodePath); err != nil || unicodePath.String() != "😀/file" {
		t.Fatalf("Unmarshal(valid surrogate pair) = %q, %v", unicodePath, err)
	}
	if err := json.Unmarshal([]byte(`"\udc00/file"`), &unicodePath); !errors.Is(err, ErrInvalidScalar) {
		t.Fatalf("Unmarshal(lone low surrogate) error = %v, want ErrInvalidScalar", err)
	}
}

func TestZeroValuesCannotBePublishedAsValidatedScalars(t *testing.T) {
	t.Parallel()

	for name, value := range map[string]any{
		"uuidv7":          UUIDv7{},
		"uuidv4":          UUIDv4{},
		"timestamp":       Timestamp{},
		"digest":          Digest{},
		"platform":        Platform(""),
		"forged platform": Platform("darwin"),
		"provider id":     ProviderID{},
		"relative path":   RelativePath{},
		"bounded integer": BoundedInteger{},
	} {
		if _, err := json.Marshal(value); !errors.Is(err, ErrInvalidScalar) {
			t.Errorf("Marshal(%s zero) error = %v, want ErrInvalidScalar", name, err)
		}
	}
}
