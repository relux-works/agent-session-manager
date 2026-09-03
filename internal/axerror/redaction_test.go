package axerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestDetailsRefuseExcludedClasses measures the best-effort scanner against
// its own registry: every key it carries is refused at the top level and inside
// a nested diagnostic object, and the four Section 15.1 classes are all
// represented. A gate deleted entirely fails on all of them; a gate narrowed to
// one class fails on the other three.
//
// The admitted set below is the half that matters more. Each of those keys was
// refused by the first revision of this gate, which matched key substrings.
// That rule refused "token_count", "socket_timeout_ms" and "credential_profile"
// - ordinary diagnostics - while still admitting a secret written under an
// innocuous name: false-positive surface with no true-positive capability, and
// the same defect BUG-260902-2faftr removed from the Configuration
// extension-key validator. These rows keep it from coming back.
func TestDetailsRefuseExcludedClasses(test *testing.T) {
	// The registry is pinned here rather than iterated, because a test that
	// ranges over the production map is its own oracle: dropping a key from the
	// map drops its own test case and the suite stays green. A mutant that
	// removed "password" survived exactly that way before this table existed.
	pinned := map[string]string{
		"access_token":        "credential",
		"api_key":             "credential",
		"auth_json":           "credential",
		"cookie":              "credential",
		"cookies":             "credential",
		"credential":          "credential",
		"credentials":         "credential",
		"oauth_token":         "credential",
		"passphrase":          "credential",
		"password":            "credential",
		"private_key":         "credential",
		"refresh_token":       "credential",
		"session_token":       "credential",
		"ssh_private_key":     "credential",
		"subscription_token":  "credential",
		"raw_transcript":      "raw transcript",
		"scrollback":          "raw transcript",
		"terminal_scrollback": "raw transcript",
		"transcript":          "raw transcript",
		"dotenv":              "environment secret",
		"env_secret":          "environment secret",
		"environment_secret":  "environment secret",
		"secret":              "environment secret",
		"secrets":             "environment secret",
		"bundle_bytes":        "opaque bundle content",
		"bundle_content":      "opaque bundle content",
		"opaque_bundle":       "opaque bundle content",
	}
	if len(excludedDetailKeys) != len(pinned) {
		test.Fatalf("scanner registry carries %d keys, the reviewed table has %d",
			len(excludedDetailKeys), len(pinned))
	}
	classes := map[string]int{}
	for key, class := range pinned {
		if got, present := excludedDetailKeys[key]; !present || got != class {
			test.Fatalf("scanner registry maps %q to %q, the reviewed table maps it to %q", key, got, class)
		}
		classes[class]++
		test.Run("refused/"+key, func(test *testing.T) {
			if err := ValidateDetails(Details{key: "value"}); !errors.Is(err, ErrInvalidDetails) {
				test.Fatalf("detail key %q naming %s was admitted", key, class)
			}
			// Section 15.1 says no detail may contain the class, so one level
			// of wrapping does not defeat the scanner either.
			if err := ValidateDetails(Details{"context": map[string]any{key: "value"}}); !errors.Is(err, ErrInvalidDetails) {
				test.Fatalf("nested detail key %q was admitted", key)
			}
		})
	}
	for _, class := range []string{"credential", "raw transcript", "environment secret", "opaque bundle content"} {
		if classes[class] == 0 {
			test.Fatalf("Section 15.1 class %q has no key in the scanner registry", class)
		}
	}
	test.Logf("detail scanner coverage: %d/%d reviewed keys over %d of the 4 Section 15.1 classes",
		len(excludedDetailKeys), len(pinned), len(classes))

	admitted := []string{
		// Diagnostics the first revision of this gate refused by substring.
		"token_count",
		"socket_timeout_ms",
		"credential_profile",
		"api_key_id",
		"secret_scan_status",
		"transcript_byte_count",
		"cookie_policy",
		// Section 16.2 manifest and bundle exclusions, which are not Section
		// 15.1 detail classes and must not be refused here.
		"pid",
		"tmux_socket_path",
		"terminal_instance_binding",
		"process_handle",
		// The typed detail sets Section 15.3 names.
		"capability",
		"caller_realm",
		"broker_state",
		"tmux_server_generation",
		"remediation",
		"provider_id",
		"provider_build",
		"macos_version",
		// Ordinary diagnostics from the pinned example.
		"expected_checkpoint",
		"remediations",
		"peer_host_id",
	}
	for _, key := range admitted {
		test.Run("admitted/"+key, func(test *testing.T) {
			if err := ValidateDetails(Details{key: "value"}); err != nil {
				test.Fatalf("ordinary diagnostic key %q was refused: %v", key, err)
			}
		})
	}
}

// TestCauseNeverReachesTheWire is the causal-redaction gate. It proves two
// distinct things: that a message or detail reproducing the local cause is
// refused at construction, and that a cause carried by a valid object is
// structurally absent from the encoded bytes.
func TestCauseNeverReachesTheWire(test *testing.T) {
	const secret = "AUTH_TOKEN=ghp_00000000000000000000000000000000"
	inner := errors.New("provider stderr: " + secret)
	wrapped := fmt.Errorf("read provider first frame: %w", inner)

	// The accident this gate exists for: a message built from the cause.
	if _, err := New(Spec{
		Version: Version100,
		Code:    "provider_protocol_error",
		Message: fmt.Sprintf("provider first frame is unusable: %v", wrapped),
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   wrapped,
	}); !errors.Is(err, ErrCausalLeak) {
		test.Fatalf("a message reproducing the cause was admitted: %v", err)
	}

	// The inner link is checked as well as the outermost rendering.
	if _, err := New(Spec{
		Version: Version100,
		Code:    "provider_protocol_error",
		Message: "provider first frame is unusable: " + inner.Error(),
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   wrapped,
	}); !errors.Is(err, ErrCausalLeak) {
		test.Fatalf("a message reproducing an inner cause link was admitted: %v", err)
	}

	// A joined cause chain is walked too.
	joined := errors.Join(errors.New("harmless local failure"), inner)
	if _, err := New(Spec{
		Version: Version100,
		Code:    "provider_protocol_error",
		Message: "provider first frame is unusable",
		IDs:     NoIDs(),
		Details: Details{"observed": inner.Error()},
		Cause:   joined,
	}); !errors.Is(err, ErrCausalLeak) {
		test.Fatalf("a detail reproducing a joined cause link was admitted: %v", err)
	}

	// Nested detail values are walked.
	if _, err := New(Spec{
		Version: Version100,
		Code:    "provider_protocol_error",
		Message: "provider first frame is unusable",
		IDs:     NoIDs(),
		Details: Details{"context": map[string]any{"observed": []any{inner.Error()}}},
		Cause:   wrapped,
	}); !errors.Is(err, ErrCausalLeak) {
		test.Fatalf("a nested detail reproducing the cause was admitted: %v", err)
	}

	// The valid construction: the cause stays local and nothing on the wire
	// carries it.
	failure, err := New(Spec{
		Version: Version100,
		Code:    "provider_protocol_error",
		Message: "provider first frame is unusable",
		IDs:     NoIDs(),
		Details: Details{"stream": "stdout"},
		Cause:   wrapped,
	})
	if err != nil {
		test.Fatalf("New: %v", err)
	}
	encoded, err := json.Marshal(failure)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	for _, forbidden := range []string{secret, "provider stderr", "read provider first frame"} {
		if strings.Contains(string(encoded), forbidden) {
			test.Fatalf("encoded object carried %q: %s", forbidden, encoded)
		}
	}
	if !errors.Is(failure, inner) {
		test.Fatal("the local cause chain was lost")
	}
	// Encoding a second time cannot start carrying it either: the encoder has
	// no access to the field.
	if strings.Contains(fmt.Sprintf("%v", failure), secret) {
		test.Fatalf("the default rendering carried the cause: %v", failure)
	}
}

// TestCausalLeakGateStatesItsBound checks the documented limits rather than
// letting them be discovered. A short cause is not used as a refusal substring,
// and a paraphrase is not detected. Both are stated in RedactionBound and in
// the doc comment on refuseCausalLeak; this test fails if the behaviour stops
// matching what is written down.
func TestCausalLeakGateStatesItsBound(test *testing.T) {
	for _, stated := range []string{
		"does not claim content-level scrubbing",
		"matches no key by substring",
		"inspects no free-form value for secret content",
	} {
		if !strings.Contains(RedactionBound, stated) {
			test.Fatalf("RedactionBound no longer states %q: %q", stated, RedactionBound)
		}
	}
	short := errors.New("EOF")
	if _, err := New(Spec{
		Version: Version100,
		Code:    "transport_failure",
		Message: "the peer closed before the hello response: EOF was observed",
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   short,
	}); err != nil {
		test.Fatalf("a cause shorter than the redactable bound was treated as a leak: %v", err)
	}
	long := errors.New("dial tcp 203.0.113.7:2222: connection refused by peer host")
	paraphrased, err := New(Spec{
		Version: Version100,
		Code:    "transport_failure",
		Message: "the peer refused the connection",
		IDs:     NoIDs(),
		Details: Details{},
		Cause:   long,
	})
	if err != nil {
		test.Fatalf("a paraphrase was refused: %v", err)
	}
	if paraphrased.Message() == long.Error() {
		test.Fatal("the paraphrase test did not actually paraphrase")
	}
}

// TestDetailValueShapeRefusals covers the value-shape arm of the details gate:
// an array nested one container too deep, an invalid UTF-8 string reaching the
// validator from a Go call site, and a nested unsupported Go type.
func TestDetailValueShapeRefusals(test *testing.T) {
	cases := []struct {
		name    string
		details Details
	}{
		{name: "array nested one container too deep", details: Details{"list": nestedArray(5)}},
		{name: "invalid UTF-8 string value", details: Details{"observed": string([]byte{0xff})}},
		{name: "nested invalid UTF-8 string value", details: Details{"c": map[string]any{"o": string([]byte{0xff})}}},
		{name: "unsupported Go type nested in an array", details: Details{"list": []any{7}}},
		{name: "unsupported Go type nested in an object", details: Details{"c": map[string]any{"n": 7.5}}},
	}
	for _, item := range cases {
		test.Run(item.name, func(test *testing.T) {
			if err := ValidateDetails(item.details); !errors.Is(err, ErrInvalidDetails) {
				test.Fatal("an unsupported detail value shape was admitted")
			}
		})
	}
	if err := ValidateDetails(Details{"list": nestedArray(4)}); err != nil {
		test.Fatalf("an array at the declared depth bound was refused: %v", err)
	}
}

// TestTypedDetailConstructorsRefuseEveryEmptyField exercises both typed
// constructors symmetrically, so neither can drift into accepting a partial set.
func TestTypedDetailConstructorsRefuseEveryEmptyField(test *testing.T) {
	complete := TargetAuth{
		ProviderID:           "codex",
		ProviderBuild:        "0.48.0",
		MacOSVersion:         "15.4",
		TmuxServerGeneration: "7",
		Remediation:          "run the provider auth smoke in the login realm",
	}
	if _, err := complete.Details(); err != nil {
		test.Fatalf("a complete typed set was refused: %v", err)
	}
	for index, partial := range []TargetAuth{
		{ProviderBuild: "0.48.0", MacOSVersion: "15.4", TmuxServerGeneration: "7", Remediation: "r"},
		{ProviderID: "codex", MacOSVersion: "15.4", TmuxServerGeneration: "7", Remediation: "r"},
		{ProviderID: "codex", ProviderBuild: "0.48.0", TmuxServerGeneration: "7", Remediation: "r"},
		{ProviderID: "codex", ProviderBuild: "0.48.0", MacOSVersion: "15.4", Remediation: "r"},
		{ProviderID: "codex", ProviderBuild: "0.48.0", MacOSVersion: "15.4", TmuxServerGeneration: "7"},
	} {
		if _, err := NewTargetAuthMissing(Version130, "auth smoke missing", NoIDs(), partial, nil); !errors.Is(err, ErrInvalidDetails) {
			test.Fatalf("partial target auth set %d was admitted", index)
		}
	}
}

// nestedArray builds a value that opens exactly depth array containers.
func nestedArray(depth int) any {
	var value any = "leaf"
	for index := 0; index < depth; index++ {
		value = []any{value}
	}
	return value
}

// TestDetailsRefuseValuesThatCannotBeMeasuredCanonically covers the arm where a
// value passes the Go type switch but has no canonical JSON encoding, so the
// declared 16 KiB bound could not be measured for it.
func TestDetailsRefuseValuesThatCannotBeMeasuredCanonically(test *testing.T) {
	if err := ValidateDetails(Details{"count": json.Number("not-a-number")}); !errors.Is(err, ErrInvalidDetails) {
		test.Fatal("a value with no canonical encoding was admitted")
	}
	if err := ValidateDetails(Details{"context": map[string]any{string([]byte{0xff}): "v"}}); !errors.Is(err, ErrInvalidDetails) {
		test.Fatal("an object key that is not valid UTF-8 was admitted")
	}
	if _, err := (TargetAuth{
		ProviderID:           string([]byte{0xff}),
		ProviderBuild:        "0.48.0",
		MacOSVersion:         "15.4",
		TmuxServerGeneration: "7",
		Remediation:          "restart",
	}).Details(); !errors.Is(err, ErrInvalidDetails) {
		test.Fatal("a typed detail value that is not valid UTF-8 was admitted")
	}
}
