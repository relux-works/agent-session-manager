package scalar

import (
	"encoding/json"
	"testing"
)

// FuzzScalarProductionEntries drives the public scalar constructors and their
// wire decoders. Every accepted value must survive the corresponding publish
// and read boundary without being widened, normalized, or later refused.
func FuzzScalarProductionEntries(f *testing.F) {
	seeds := []struct {
		kind  string
		value []byte
	}{
		{"timestamp", []byte("1990-12-31T23:59:60.000Z")},
		{"timestamp", []byte("2026-08-31t14:05:06.123z")},
		{"timestamp", []byte("2023-02-29T12:00:00.000Z")},
		{"uuidv7", []byte("0198f4c8-8e50-7f66-8f70-1234567890ab")},
		{"uuidv4", []byte("550e8400-e29b-41d4-a716-446655440000")},
		{"platform", []byte("windows")},
		{"provider", []byte("codex")},
		{"relative-path", []byte("workspace/данные/😀.json")},
		{"windows-path", []byte(`C:\Developer\ReluxWorks`)},
		{"windows-path", []byte(`C:\unsafe\CON.txt`)},
		{"windows-path", []byte(`C:\unsafe\*.json`)},
		{"digest", []byte("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")},
		{"safe-integer-json", []byte("9007199254740991")},
		{"uint53-json", []byte("9007199254740992")},
		{"bounded-integer-json", []byte("10")},
		{"decimal-uint64", []byte("18446744073709551615")},
		{"closed-enum", []byte("workspace_tree")},
		{"git-oid", []byte("sha1:602548b4fd46332c934667db9992b8bb00318c88")},
		{"git-ref", []byte("refs/heads/feature/ax")},
		{"git-url", []byte("ssh://git@github.com/relux/repo.git")},
	}
	for _, seed := range seeds {
		f.Add(seed.kind, seed.value)
	}

	f.Fuzz(func(t *testing.T, kind string, data []byte) {
		if len(data) > 64*1024 {
			return
		}
		text := string(data)

		switch kind {
		case "timestamp":
			value, err := ParseTimestamp(text)
			if err != nil {
				return
			}
			if _, err := value.Time(); err != nil {
				t.Fatalf("ParseTimestamp accepted %q but Timestamp.Time refused it: %v", text, err)
			}
			assertScalarJSONRoundTrip(t, value, &Timestamp{}, text)
		case "uuidv7":
			value, err := ParseUUIDv7(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &UUIDv7{}, text)
		case "uuidv4":
			value, err := ParseUUIDv4(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &UUIDv4{}, text)
		case "platform":
			value, err := ParsePlatform(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, new(Platform), text)
		case "provider":
			value, err := ParseProviderID(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &ProviderID{}, text)
		case "relative-path":
			value, err := ParseRelativePath(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &RelativePath{}, text)
		case "windows-path":
			value, err := ParseAbsolutePath(PlatformWindows, text)
			if err != nil {
				return
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("ParseAbsolutePath accepted %q but MarshalJSON refused it: %v", text, err)
			}
			decoded, err := DecodeAbsolutePathJSON(PlatformWindows, encoded)
			if err != nil || decoded.String() != text || decoded.Platform() != PlatformWindows {
				t.Fatalf("Windows absolute path round trip = %#v, %v; want %q", decoded, err, text)
			}
		case "digest":
			value, err := ParseDigest(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &Digest{}, text)
		case "safe-integer-json":
			var value SafeInteger
			if err := json.Unmarshal(data, &value); err != nil {
				return
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("SafeInteger accepted %q but refused publication: %v", text, err)
			}
			var decoded SafeInteger
			if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Int64() != value.Int64() {
				t.Fatalf("SafeInteger round trip = %d, %v; want %d", decoded.Int64(), err, value.Int64())
			}
		case "uint53-json":
			var value Uint53
			if err := json.Unmarshal(data, &value); err != nil {
				return
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("Uint53 accepted %q but refused publication: %v", text, err)
			}
			var decoded Uint53
			if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.Uint64() != value.Uint64() {
				t.Fatalf("Uint53 round trip = %d, %v; want %d", decoded.Uint64(), err, value.Uint64())
			}
		case "bounded-integer-json":
			value, err := DecodeBoundedIntegerJSON(data, -10, 10)
			if err != nil {
				return
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("BoundedInteger accepted %q but refused publication: %v", text, err)
			}
			decoded, err := DecodeBoundedIntegerJSON(encoded, -10, 10)
			if err != nil || decoded.Int64() != value.Int64() || decoded.Min() != -10 || decoded.Max() != 10 {
				t.Fatalf("BoundedInteger round trip = %#v, %v; want %#v", decoded, err, value)
			}
		case "decimal-uint64":
			value, err := ParseDecimalUint64(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &DecimalUint64{}, text)
		case "closed-enum":
			allowed := []string{"workspace_group", "workspace_tree", "provider", "task_board", "composite"}
			value, err := ParseClosedEnum(text, allowed...)
			if err != nil {
				return
			}
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("ParseClosedEnum accepted %q but MarshalJSON refused it: %v", text, err)
			}
			decoded, err := DecodeClosedEnumJSON(encoded, allowed...)
			if err != nil || decoded.String() != text {
				t.Fatalf("closed enum round trip = %q, %v; want %q", decoded, err, text)
			}
		case "git-oid":
			value, err := ParseGitOID(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &GitOID{}, text)
		case "git-ref":
			value, err := ParseGitRef(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &GitRef{}, text)
		case "git-url":
			value, err := ParseSanitizedGitURL(text)
			if err != nil {
				return
			}
			assertScalarJSONRoundTrip(t, value, &SanitizedGitURL{}, text)
		}
	})
}

func assertScalarJSONRoundTrip(t *testing.T, value any, target any, want string) {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("accepted scalar %q refused publication: %v", want, err)
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		t.Fatalf("published scalar %q refused read-back from %s: %v", want, encoded, err)
	}
	got, ok := target.(interface{ String() string })
	if !ok || got.String() != want {
		t.Fatalf("scalar read-back = %v, want byte-identical %q", target, want)
	}
}
