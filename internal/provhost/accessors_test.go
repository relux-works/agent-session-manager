package provhost

import (
	"bytes"
	"encoding/json"
	"testing"
)

// The replay accessors below hand validated probe and identify
// members to use sites that bind them further. They add no refusal
// of their own: a malformed body fails with the decoder's error,
// and the replayed members are the validated output, never a second
// decode.

func TestProbeBuildReplaysValidatedTuple(t *testing.T) {
	tuple, err := ProbeBuild([]byte(specProbeExample))
	if err != nil {
		t.Fatalf("ProbeBuild(spec) error = %v", err)
	}
	if tuple.ProviderID != "pi" || tuple.ProviderVersion != "0.73.1" || tuple.Platform != "macos" || tuple.Architecture != "arm64" {
		t.Fatalf("ProbeBuild(spec) = %+v, want pi 0.73.1 macos arm64", tuple)
	}
}

func TestProbeBuildRefusesMalformedProbe(t *testing.T) {
	if _, err := ProbeBuild(probeVariant(t, `"arm64"`, `"sparc"`)); err == nil {
		t.Fatal("ProbeBuild(bad architecture) = nil, want provider_protocol_error")
	}
}

func TestSplitIdentifyResultReplaysIdentityAndConfidence(t *testing.T) {
	identity := validCreatedIdentity(t, validCreateParams())
	body := identifyBody(t, identity, "exact")
	outcome, err := SplitIdentifyResult(body, validCreateParams().ProviderID)
	if err != nil {
		t.Fatalf("SplitIdentifyResult error = %v", err)
	}
	if outcome.Confidence != "exact" {
		t.Fatalf("confidence = %q, want exact", outcome.Confidence)
	}
	if !bytes.Equal(outcome.Identity, identity) {
		t.Fatal("replayed identity differs from the validated member")
	}
	// The replay is a copy: mutating it must not alias the body.
	outcome.Identity[0] ^= 0xff
	if bytes.Equal(outcome.Identity, identity) {
		t.Fatal("mutating the replay changed the validated member; want a copy")
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal identify body: %v", err)
	}
	if !bytes.Equal(decoded["identity"], identity) {
		t.Fatal("mutating the replay corrupted the response body; want a copy")
	}
}

func TestSplitIdentifyResultRefusesMalformedBody(t *testing.T) {
	identity := validCreatedIdentity(t, validCreateParams())
	if _, err := SplitIdentifyResult(identifyBody(t, identity, "certain"), validCreateParams().ProviderID); err == nil {
		t.Fatal("SplitIdentifyResult(bad confidence) = nil, want provider_protocol_error")
	}
}

func validCreatedIdentity(t *testing.T, params IdentityParams) []byte {
	t.Helper()
	identity, err := CreateIdentity(params)
	if err != nil {
		t.Fatalf("CreateIdentity error = %v", err)
	}
	return identity
}

func identifyBody(t *testing.T, identity []byte, confidence string) []byte {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"identity":         json.RawMessage(identity),
		"confidence":       confidence,
		"matched_evidence": []string{"native_id"},
	})
	if err != nil {
		t.Fatalf("marshal identify body: %v", err)
	}
	return body
}
