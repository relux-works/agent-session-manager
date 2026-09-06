package sessadapter

import (
	"strings"
	"testing"
	"time"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("mustTime: %v", err)
	}
	return parsed
}

func TestDecodeTupleAcceptsFixture(t *testing.T) {
	tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	if tuple.EnvironmentID != fixtureEnvironmentID || tuple.Platform != "linux" || tuple.Architecture != "amd64" {
		t.Fatalf("DecodeTuple = %+v", tuple)
	}
}

func TestDecodeTupleRefusals(t *testing.T) {
	fixture := []byte(fixtureTupleJSON())
	members := []string{"environment_id", "environment_version", "platform", "architecture", "store_schema_fingerprint", "adapter_version"}
	for _, member := range members {
		t.Run("missing "+member, func(t *testing.T) {
			_, err := DecodeTuple(dropMember(t, fixture, member))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "misses a required member")
		})
	}
	for _, test := range []struct {
		name   string
		member string
		value  string
		text   string
	}{
		{"bad environment id", "environment_id", `"Test!"`, "not an environment-id"},
		{"empty version", "environment_version", `""`, "not a string[1..128]"},
		{"bad platform", "platform", `"darwin"`, "outside linux|macos|windows|wsl2"},
		{"v1 darwin is never a tuple platform", "platform", `"darwin"`, "outside linux|macos|windows|wsl2"},
		{"bad architecture", "architecture", `"x86"`, "outside amd64|arm64"},
		{"bad fingerprint", "store_schema_fingerprint", `"abc"`, "not a digest"},
		{"bad adapter version", "adapter_version", `"1.2"`, "not SemVer"},
		// Executable provenance is refused as an unknown member:
		// the tuple never carries it.
		{"executable provenance", "executable_sha256", `"sha256:` + strings.Repeat("0", 64) + `"`, "unknown member"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body []byte
			if test.member == "executable_sha256" {
				body = appendMember(t, fixture, test.member, test.value)
			} else {
				body = mutateMember(t, fixture, test.member, test.value)
			}
			_, err := DecodeTuple(body)
			requireRefusal(t, err, "session_adapter_protocol_error", "member", test.member, test.text)
		})
	}
	for _, platform := range []string{"linux", "macos", "windows", "wsl2"} {
		if _, err := DecodeTuple(mutateMember(t, fixture, "platform", `"`+platform+`"`)); err != nil {
			t.Fatalf("DecodeTuple(%q): %v", platform, err)
		}
	}
}

func TestDecodeTupleEntryAcceptsBothDirections(t *testing.T) {
	for _, direction := range []Direction{DirectionSourceRead, DirectionTargetWrite} {
		entry, err := DecodeTupleEntry([]byte(fixtureEntryJSON(direction)))
		if err != nil {
			t.Fatalf("DecodeTupleEntry(%q): %v", direction, err)
		}
		if entry.Key.Direction != direction || entry.Status != "accepted" {
			t.Fatalf("entry = %+v", entry)
		}
		if direction == DirectionSourceRead && entry.Smoke != nil {
			t.Fatal("source entry carries smoke")
		}
		if direction == DirectionTargetWrite && entry.Smoke == nil {
			t.Fatal("target entry misses smoke")
		}
	}
}

func TestDecodeTupleEntryCoherence(t *testing.T) {
	fixture := []byte(fixtureEntryJSON(DirectionTargetWrite))
	// Accepted with revocation members: refused.
	accepted := mutateMember(t, fixture, "revocation_reason", `"superseded"`)
	_, err := DecodeTupleEntry(accepted)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "status", "carries revocation members")
	// Revoked without both members: refused in both halves.
	revoked := mutateMember(t, fixture, "status", `"revoked"`)
	_, err = DecodeTupleEntry(revoked)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "status", "misses a revocation member")
	revokedHalf := mutateMember(t, mutateMember(t, fixture, "status", `"revoked"`), "revocation_reason", `"superseded"`)
	_, err = DecodeTupleEntry(revokedHalf)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "status", "misses a revocation member")
	// Revoked with both members: admitted as revoked (admission
	// refuses it later; decoding proves shape, not admission).
	revokedFull := mutateMember(t, mutateMember(t, fixture, "status", `"revoked"`), "revocation_reason", `"superseded"`)
	revokedFull = mutateMember(t, revokedFull, "revoked_at", `"2026-06-02T00:00:00.000Z"`)
	entry, err := DecodeTupleEntry(revokedFull)
	if err != nil {
		t.Fatalf("DecodeTupleEntry revoked: %v", err)
	}
	if entry.Status != "revoked" || entry.RevocationReason == nil || entry.RevokedAt == nil {
		t.Fatalf("revoked entry = %+v", entry)
	}
	// Inverted validity interval: refused.
	inverted := mutateMember(t, fixture, "valid_from", `"2028-01-01T00:00:00.000Z"`)
	_, err = DecodeTupleEntry(inverted)
	requireRefusal(t, err, "session_adapter_protocol_error", "member", "valid_until", "does not order")
	// Empty strategies: refused. Unknown strategy: refused.
	// Unsorted strategies: refused.
	for _, test := range []struct {
		name  string
		value string
		text  string
	}{
		{"empty strategies", `[]`, "are empty"},
		{"unknown strategy", `["teleport"]`, "five-strategy vocabulary"},
		{"unsorted strategies", `["target_native_writer","continuation_context"]`, "not sorted unique"},
		{"duplicate strategies", `["target_native_writer","target_native_writer"]`, "not sorted unique"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeTupleEntry(mutateMember(t, fixture, "strategies", test.value))
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "strategies", test.text)
		})
	}
	// Failing fixture evidence: refused. Zero fixture count:
	// refused. Failing smoke: refused. Smoke without a passed
	// turn: refused.
	for _, test := range []struct {
		name   string
		member string
		value  string
		text   string
	}{
		{"fixture result fail", "fixture_evidence", `{"suite_revision":"rev-9","suite_digest":"` + fixtureSuiteDigest + `","result":"fail","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureEvidenceDigest + `","fixture_count":12}`, "not pass"},
		{"fixture count zero", "fixture_evidence", `{"suite_revision":"rev-9","suite_digest":"` + fixtureSuiteDigest + `","result":"pass","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureEvidenceDigest + `","fixture_count":0}`, "above zero"},
		{"smoke result fail", "resume_smoke_evidence", `{"result":"fail","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureSmokeDigest + `","native_cli_family":"claude","bounded_continuation_turn_passed":true}`, "not pass"},
		{"smoke turn failed", "resume_smoke_evidence", `{"result":"pass","executed_at":"` + fixtureExecutedAt + `","evidence_digest":"` + fixtureSmokeDigest + `","native_cli_family":"claude","bounded_continuation_turn_passed":false}`, "did not pass"},
		{"bad status", "status", `"pending"`, "accepted|revoked"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeTupleEntry(mutateMember(t, fixture, test.member, test.value))
			if err == nil {
				t.Fatalf("DecodeTupleEntry admitted %s", test.name)
			}
			if failureCode(t, err) != "session_adapter_protocol_error" {
				t.Fatalf("code = %v, want session_adapter_protocol_error", err)
			}
			if !strings.Contains(err.Error(), test.text) {
				t.Fatalf("error = %v, want %q", err, test.text)
			}
		})
	}
}

// TestTupleAdmissionValidityBoundaries pins the interval on both
// edges: the instant exactly at valid_from and valid_until
// admits, one second outside refuses on each side. Both edges
// are enforced: a one-sided check would admit the open side.
func TestTupleAdmissionValidityBoundaries(t *testing.T) {
	entry, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionTargetWrite)))
	if err != nil {
		t.Fatalf("DecodeTupleEntry: %v", err)
	}
	tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	binding := fixtureBindingFacts()
	for _, test := range []struct {
		name    string
		instant string
		ok      bool
	}{
		{"one second before from", "2025-12-31T23:59:59.000Z", false},
		{"exactly at from", fixtureValidFrom, true},
		{"inside", fixtureCallTime, true},
		{"exactly at until", fixtureValidUntil, true},
		{"one second after until", "2027-01-01T00:00:01.000Z", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := CheckTupleAdmission(entry, DirectionTargetWrite, tuple, binding, mustTime(t, test.instant))
			if test.ok && err != nil {
				t.Fatalf("CheckTupleAdmission: %v", err)
			}
			if !test.ok && err == nil {
				t.Fatalf("CheckTupleAdmission admitted %s", test.name)
			}
			if !test.ok {
				requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "validity interval")
			}
		})
	}
}

func TestCheckTupleAdmissionRefusals(t *testing.T) {
	entry, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionTargetWrite)))
	if err != nil {
		t.Fatalf("DecodeTupleEntry: %v", err)
	}
	tuple, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	binding := fixtureBindingFacts()
	now := mustTime(t, fixtureCallTime)
	otherTuple := tuple
	otherTuple.Version = "9.9.9"
	for _, test := range []struct {
		name      string
		direction Direction
		tuple     Tuple
		binding   BindingFacts
		entry     TupleEntry
		text      string
	}{
		{"direction mismatch", DirectionSourceRead, tuple, binding, entry, "direction does not match"},
		{"tuple mismatch", DirectionTargetWrite, otherTuple, binding, entry, "does not equal the probed tuple"},
		{"provider mismatch", DirectionTargetWrite, tuple, BindingFacts{ProviderID: "other", CandidateKind: CandidateBuiltin, ExecutableSHA256: binding.ExecutableSHA256, ProviderManifestDigest: binding.ProviderManifestDigest, AdapterManifestDigest: binding.AdapterManifestDigest}, entry, "execution binding"},
		{"executable mismatch", DirectionTargetWrite, tuple, BindingFacts{ProviderID: binding.ProviderID, CandidateKind: CandidateBuiltin, ExecutableSHA256: fixtureDigest("other-exe"), ProviderManifestDigest: binding.ProviderManifestDigest, AdapterManifestDigest: binding.AdapterManifestDigest}, entry, "execution binding"},
		{"kind mismatch", DirectionTargetWrite, tuple, BindingFacts{ProviderID: binding.ProviderID, CandidateKind: CandidateExternal, ExecutableSHA256: binding.ExecutableSHA256, ProviderManifestDigest: binding.ProviderManifestDigest, AdapterManifestDigest: binding.AdapterManifestDigest}, entry, "execution binding"},
		{"revoked entry", DirectionTargetWrite, tuple, binding, revokedEntry(t), "not accepted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := CheckTupleAdmission(test.entry, test.direction, test.tuple, test.binding, now)
			requireRefusal(t, err, "unsupported_environment_tuple", "environment", tuple.EnvironmentID, test.text)
		})
	}
	// Source direction against a target entry: refused on
	// direction before any strategy rule is reached.
	if err := CheckTupleAdmission(entry, DirectionSourceRead, tuple, binding, now); err == nil {
		t.Fatal("admitted a target entry for a source call")
	}
	// Source entry admits a source call and refuses a target call
	// on direction.
	source, err := DecodeTupleEntry([]byte(fixtureEntryJSON(DirectionSourceRead)))
	if err != nil {
		t.Fatalf("DecodeTupleEntry: %v", err)
	}
	if err := CheckTupleAdmission(source, DirectionSourceRead, tuple, binding, now); err != nil {
		t.Fatalf("CheckTupleAdmission source: %v", err)
	}
	if err := CheckTupleAdmission(source, DirectionTargetWrite, tuple, binding, now); err == nil {
		t.Fatal("admitted a source entry for a target call")
	}
	// A source entry with smoke, or without archive_only, is
	// refused even in its own direction. A target entry with
	// archive_only, or without smoke, is refused in its own.
	sourceWithSmoke := source
	smokeTime := mustTime(t, fixtureExecutedAt)
	sourceWithSmoke.Smoke = &SmokeEvidence{ExecutedAt: smokeTime, EvidenceDigest: fixtureSmokeDigest, NativeCLIFamily: "claude"}
	if err := CheckTupleAdmission(sourceWithSmoke, DirectionSourceRead, tuple, binding, now); err == nil {
		t.Fatal("admitted a source entry carrying smoke")
	} else {
		requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "resume smoke")
	}
	sourceWide := source
	sourceWide.Strategies = []string{"archive_only", "target_native_writer"}
	if err := CheckTupleAdmission(sourceWide, DirectionSourceRead, tuple, binding, now); err == nil {
		t.Fatal("admitted a source entry beyond archive_only")
	}
	targetArchive := entry
	targetArchive.Strategies = []string{"archive_only"}
	targetArchive.Smoke = nil
	if err := CheckTupleAdmission(targetArchive, DirectionTargetWrite, tuple, binding, now); err == nil {
		t.Fatal("admitted a target entry with archive_only")
	} else {
		requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "archive_only")
	}
	targetNoSmoke := entry
	targetNoSmoke.Smoke = nil
	if err := CheckTupleAdmission(targetNoSmoke, DirectionTargetWrite, tuple, binding, now); err == nil {
		t.Fatal("admitted a target entry without smoke")
	} else {
		requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "resume smoke")
	}
}

func revokedEntry(t *testing.T) TupleEntry {
	t.Helper()
	fixture := []byte(fixtureEntryJSON(DirectionTargetWrite))
	revoked := mutateMember(t, mutateMember(t, fixture, "status", `"revoked"`), "revocation_reason", `"superseded"`)
	revoked = mutateMember(t, revoked, "revoked_at", `"2026-06-02T00:00:00.000Z"`)
	entry, err := DecodeTupleEntry(revoked)
	if err != nil {
		t.Fatalf("DecodeTupleEntry revoked: %v", err)
	}
	return entry
}
