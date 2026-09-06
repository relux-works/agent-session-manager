package environ_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/dirnode"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessadapter"
)

// This file is the cross-facade string-measure battery. Section
// 1.6 bounds strings in characters, not bytes: string[1..128]
// admits 128 characters of any width. Two strict decoders both
// blind to lone surrogates was one drift class the cross-story
// review found; byte-vs-rune bounds disagreeing on a member that
// crosses both protocols was another. This battery pins the
// measure at every character-bound member one facade shares with
// another, with multibyte vectors that separate the measures: a
// 128-`é` value is 128 characters but 256 bytes, so a
// byte-measure refuses what the specification admits.
//
// The byte side is ledgered, not left open. Provider SpawnPlan
// argv elements and totals and env-literal values bound BYTES by
// Section 5.1 (outside this story), implemented in exactly two
// functions — provhost checkSpawnArgv and checkSpawnEnvLiterals
// — and the byte rows below drive all three byte gates through
// the production DecodeSpawnPlan entry. validEnvName measures
// len() over an ASCII-only grammar where bytes and runes
// coincide, so no row can separate them; that is stated here,
// not hidden. Every other string bound in the census scope flows
// through the single measure helper of its package (census
// ledger: StringLength, runeLength, stringLength), and the rune
// rows pin the helper verdicts at one crossing member per
// facade plus every provhost rune site.

// validSpawnJSON is one valid Provider SpawnPlan body with the
// given profile mapping, first extra argv element, literal value,
// and native session ID literal. The argv always carries one
// short flag so element-count rules hold while one member under
// test varies.
func validSpawnJSON(mapping, extraArgv, literalValue, nativeID string) []byte {
	argv := `["--serve"]`
	if extraArgv != "" {
		argv = fmt.Sprintf(`["--serve",%s]`, jsonQuote(extraArgv))
	}
	literals := `{}`
	if literalValue != "" {
		literals = fmt.Sprintf(`{"EXTRA_KEY":%s}`, jsonQuote(literalValue))
	}
	native := `null`
	if nativeID != "" {
		native = jsonQuote(nativeID)
	}
	return []byte(fmt.Sprintf(`{"argv":%s,"cwd":"/tmp/ax","env_names":[],"env_literals":%s,"native_session_id":%s,"profile_mapping":%s,"extensions":{}}`,
		argv, literals, native, jsonQuote(mapping)))
}

// validSpawnMapping returns the production YOLO flag for codex:
// scaffolding for rows whose probe point is a different member,
// so a mapping change cannot break them for the wrong reason.
// Rows whose probe point IS the mapping construct their own
// lengths and never call this.
func validSpawnMapping(t *testing.T) string {
	t.Helper()
	mapping, err := provhost.ProfileMapping("codex", provhost.ProfileYOLO)
	if err != nil {
		t.Fatalf("scaffolding: %v", err)
	}
	return mapping
}

// jsonQuote quotes a Go string as a JSON string literal.
func jsonQuote(value string) string {
	return fmt.Sprintf("%q", value)
}

// TestStringMeasureCountsRunes proves the character measure at
// the unit helper and at one crossing member per facade. Each
// accept vector carries more bytes than its bound, so a
// byte-measure would refuse it; each refuse vector carries one
// character too many, so a widened bound would admit it.
func TestStringMeasureCountsRunes(t *testing.T) {
	ascii128 := strings.Repeat("v", 128)
	wide128 := strings.Repeat("é", 128)
	wide129 := strings.Repeat("é", 129)
	wide512 := strings.Repeat("é", 512)
	wide513 := strings.Repeat("é", 513)
	wide1024 := strings.Repeat("é", 1024)
	wide1025 := strings.Repeat("é", 1025)
	wide4096 := strings.Repeat("é", 4096)
	wide4097 := strings.Repeat("é", 4097)
	t.Run("environ helper", func(t *testing.T) {
		if _, ok := environ.CheckStringBounds([]byte(jsonQuote(wide128)), 1, 128); !ok {
			t.Fatal("environ.CheckStringBounds refused 128 characters in 256 bytes")
		}
		if _, ok := environ.CheckStringBounds([]byte(jsonQuote(wide129)), 1, 128); ok {
			t.Fatal("environ.CheckStringBounds admitted 129 characters")
		}
		if _, ok := environ.CheckStringBounds([]byte(jsonQuote(ascii128)), 1, 128); !ok {
			t.Fatal("environ.CheckStringBounds refused 128 ASCII characters")
		}
	})
	t.Run("sessadapter tuple version", func(t *testing.T) {
		accept := []byte(strings.Replace(validTupleJSON(), `"2.1.0"`, jsonQuote(wide128), 1))
		if _, err := sessadapter.DecodeTuple(accept); err != nil {
			t.Fatalf("DecodeTuple refused 128-character version: %v", err)
		}
		refuse := []byte(strings.Replace(validTupleJSON(), `"2.1.0"`, jsonQuote(wide129), 1))
		_, err := sessadapter.DecodeTuple(refuse)
		if err == nil {
			t.Fatal("DecodeTuple admitted 129-character version")
		}
		if !strings.Contains(err.Error(), "string[1..128]") {
			t.Fatalf("DecodeTuple error = %v, want the version bound arm", err)
		}
	})
	t.Run("dirnode scan cursor", func(t *testing.T) {
		if _, err := dirnode.CheckScanRequest(validScanRequestJSON(jsonQuote(wide4096))); err != nil {
			t.Fatalf("CheckScanRequest refused 4096-character cursor: %v", err)
		}
		_, err := dirnode.CheckScanRequest(validScanRequestJSON(jsonQuote(wide4097)))
		if err == nil {
			t.Fatal("CheckScanRequest admitted 4097-character cursor")
		}
		if !strings.Contains(err.Error(), "string[1..4096]") {
			t.Fatalf("CheckScanRequest error = %v, want the cursor bound arm", err)
		}
	})
	t.Run("provhost profile mapping bound arm", func(t *testing.T) {
		bound := validSpawnJSON(strings.Repeat("m", 513), "", "", "")
		err := provhost.DecodeSpawnPlan(bound, "codex", provhost.ProfileYOLO, scalar.PlatformLinux)
		if err == nil {
			t.Fatal("DecodeSpawnPlan admitted 513-character mapping")
		}
		if !strings.Contains(err.Error(), "not at most 512 characters") {
			t.Fatalf("DecodeSpawnPlan error = %v, want the mapping bound arm", err)
		}
	})
	t.Run("provhost mapping admits 300 wide characters", func(t *testing.T) {
		// 300 `é` is 300 characters but 600 bytes: the bound
		// arm must admit it (runes) and the equality arm must
		// refuse it (it is not the provider mapping). A byte
		// measure would fire the bound arm instead.
		wide := validSpawnJSON(strings.Repeat("é", 300), "", "", "")
		err := provhost.DecodeSpawnPlan(wide, "codex", provhost.ProfileYOLO, scalar.PlatformLinux)
		if err == nil {
			t.Fatal("DecodeSpawnPlan admitted a mapping that is not the provider mapping")
		}
		if !strings.Contains(err.Error(), "does not match the provider profile") {
			t.Fatalf("DecodeSpawnPlan error = %v, want the mapping equality arm, not the bound arm", err)
		}
	})
	t.Run("provhost native session id bound", func(t *testing.T) {
		mapping := validSpawnMapping(t)
		accept := validSpawnJSON(mapping, "", "", wide512)
		if err := provhost.DecodeSpawnPlan(accept, "codex", provhost.ProfileYOLO, scalar.PlatformLinux); err != nil {
			t.Fatalf("DecodeSpawnPlan refused 512-character native session id: %v", err)
		}
		refuse := validSpawnJSON(mapping, "", "", wide513)
		err := provhost.DecodeSpawnPlan(refuse, "codex", provhost.ProfileYOLO, scalar.PlatformLinux)
		if err == nil {
			t.Fatal("DecodeSpawnPlan admitted 513-character native session id")
		}
		if !strings.Contains(err.Error(), "1..512 characters") {
			t.Fatalf("DecodeSpawnPlan error = %v, want the native session id bound arm", err)
		}
	})
	t.Run("identity opaque bound agrees", func(t *testing.T) {
		// The opaque_identity value bound (1..1024 runes) is
		// the one member validated by both identity dialects:
		// both judges must agree on the multibyte edges.
		accept := mutateMember(t, validIdentityJSON(), "opaque_identity", `{"k":`+jsonQuote(wide1024)+`}`)
		if err := provhost.CheckIdentity(accept, "antigravity"); err != nil {
			t.Fatalf("CheckIdentity refused 1024-character opaque value: %v", err)
		}
		if _, _, err := canonicaljson.CalculateObjectIdentity(accept); err != nil {
			t.Fatalf("CalculateObjectIdentity refused 1024-character opaque value: %v", err)
		}
		refuse := mutateMember(t, validIdentityJSON(), "opaque_identity", `{"k":`+jsonQuote(wide1025)+`}`)
		if err := provhost.CheckIdentity(refuse, "antigravity"); err == nil {
			t.Fatal("CheckIdentity admitted 1025-character opaque value")
		}
		if _, _, err := canonicaljson.CalculateObjectIdentity(refuse); err == nil {
			t.Fatal("CalculateObjectIdentity admitted 1025-character opaque value")
		}
	})
}

// TestByteBoundsStayBytes proves the three ledgered byte gates
// keep measuring bytes: each vector is within the rune bound a
// character measure would enforce but over the byte bound the
// specification states, so a rune-measure regression would admit
// it and travel to a later arm instead of the byte arm.
func TestByteBoundsStayBytes(t *testing.T) {
	mapping := validSpawnMapping(t)
	t.Run("argv element bytes", func(t *testing.T) {
		// 2049 `é` is 2049 characters but 4098 bytes.
		body := validSpawnJSON(mapping, strings.Repeat("é", 2049), "", "")
		err := provhost.DecodeSpawnPlan(body, "codex", provhost.ProfileYOLO, scalar.PlatformLinux)
		if err == nil {
			t.Fatal("DecodeSpawnPlan admitted a 4098-byte argv element")
		}
		if !strings.Contains(err.Error(), "NUL-free bytes") {
			t.Fatalf("DecodeSpawnPlan error = %v, want the argv element byte arm", err)
		}
	})
	t.Run("argv total bytes", func(t *testing.T) {
		// Seventeen 4000-byte elements pass the element gate
		// and fail the 65536-byte total.
		elements := make([]string, 0, 17)
		for index := 0; index < 17; index++ {
			elements = append(elements, jsonQuote(strings.Repeat("a", 4000)))
		}
		body := []byte(fmt.Sprintf(`{"argv":["--serve",%s],"cwd":"/tmp/ax","env_names":[],"env_literals":{},"native_session_id":null,"profile_mapping":%s,"extensions":{}}`,
			strings.Join(elements, ","), jsonQuote(mapping)))
		err := provhost.DecodeSpawnPlan(body, "codex", provhost.ProfileYOLO, scalar.PlatformLinux)
		if err == nil {
			t.Fatal("DecodeSpawnPlan admitted argv over 65536 total bytes")
		}
		if !strings.Contains(err.Error(), "65536 bytes total") {
			t.Fatalf("DecodeSpawnPlan error = %v, want the argv total byte arm", err)
		}
	})
	t.Run("env literal bytes", func(t *testing.T) {
		body := validSpawnJSON(mapping, "", strings.Repeat("é", 2049), "")
		err := provhost.DecodeSpawnPlan(body, "codex", provhost.ProfileYOLO, scalar.PlatformLinux)
		if err == nil {
			t.Fatal("DecodeSpawnPlan admitted a 4098-byte env literal")
		}
		if !strings.Contains(err.Error(), "4096 NUL-free bytes") {
			t.Fatalf("DecodeSpawnPlan error = %v, want the env literal byte arm", err)
		}
	})
}
