package environ_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Shared fixtures for the environ boundary battery. This file is in
// the external test package: the facades the battery drives import
// environ, so an in-package battery importing them would be an
// import cycle. Every builder
// emits hand-written JSON with independently chosen literals: no
// builder derives its probe points from production constants, so
// a mutated production constant cannot take its fixture with
// it. The verdict always comes from production; the vectors never
// do. In particular the schema URNs, version strings, member
// names, and vocabulary tokens below are retyped from the pinned
// specification, not referenced from any package under test.
//
// One corpus drives every facade: the same digest, UUIDv7, and
// timestamp identities flow into the sessadapter, dirnode,
// provhost, and canonicaljson entries each battery file drives,
// so a fixture that is valid for one facade and invalid for
// another fails loudly instead of passing per-facade copies that
// never meet.

// Fixed fixture identities. The UUIDs carry a 7 version nibble;
// the digests are syntactically pinned sha256 hex.
const (
	fixtureHostID        = "0198f4c8-8e50-7f66-8f70-1234567890ab"
	fixtureOperationID   = "0198f4c8-8e50-7f66-8f70-1234567890ac"
	fixtureSubjectID     = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	fixtureWorkspaceID   = "0198f4c8-6c30-7d44-8d5e-1234567890ab"
	fixtureCreatorHostID = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	fixtureObservedAt    = "2026-08-19T04:05:00.000Z"
	fixtureValidFrom     = "2026-01-01T00:00:00.000Z"
	fixtureValidUntil    = "2027-01-01T00:00:00.000Z"
	fixtureEnvironmentID = "test.env"
	fixtureAdapterVer    = "1.2.3"
	fixtureProviderID    = "test-provider"
)

// fixtureDigest returns the pinned digest of one seed.
func fixtureDigest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Standard fixture digests, each naming its seed.
var (
	fixtureStorePrint  = fixtureDigest("test-store-fingerprint")
	fixtureInstallID   = fixtureDigest("test-installation")
	fixtureObserveID   = fixtureDigest("test-observation")
	fixtureRealmPrint  = fixtureDigest("test-realm")
	fixtureRecordID    = fixtureDigest("test-identity-record")
	fixtureEvidenceID  = fixtureDigest("test-evidence")
	fixtureExecutable  = fixtureDigest("test-executable")
	fixtureAdapterMain = fixtureDigest("test-adapter-manifest")
)

// TestSharedFixtureIdentitiesAreValid guards the instrument: a
// malformed shared identity would refuse at every facade for the
// wrong reason, turning the whole battery into a test of the
// fixture rather than the facades.
func TestSharedFixtureIdentitiesAreValid(t *testing.T) {
	for _, identity := range []string{fixtureHostID, fixtureOperationID, fixtureSubjectID, fixtureWorkspaceID, fixtureCreatorHostID} {
		if _, err := scalar.ParseUUIDv7(identity); err != nil {
			t.Fatalf("fixture UUID %q does not parse: %v", identity, err)
		}
	}
	for _, moment := range []string{fixtureObservedAt, fixtureValidFrom, fixtureValidUntil} {
		if _, err := scalar.ParseTimestamp(moment); err != nil {
			t.Fatalf("fixture timestamp %q does not parse: %v", moment, err)
		}
	}
	for _, digest := range []string{fixtureStorePrint, fixtureInstallID, fixtureObserveID, fixtureRealmPrint, fixtureRecordID, fixtureEvidenceID, fixtureExecutable, fixtureAdapterMain} {
		if _, err := scalar.ParseDigest(digest); err != nil {
			t.Fatalf("fixture digest %q does not parse: %v", digest, err)
		}
	}
}

// validTupleJSON is one valid Environment Tuple over the shared
// identities.
func validTupleJSON() string {
	return fmt.Sprintf(`{"environment_id":%q,"environment_version":"2.1.0","platform":"linux","architecture":"amd64","store_schema_fingerprint":%q,"adapter_version":%q}`,
		fixtureEnvironmentID, fixtureStorePrint, fixtureAdapterVer)
}

// validScanRequestJSON is one valid dirnode scan request carrying
// the given cursor literal (already JSON-encoded by the caller).
func validScanRequestJSON(cursorLiteral string) []byte {
	return []byte(fmt.Sprintf(`{"operation_id":%q,"installation_ids":[%q],"prior_batch_id":null,"cursor":%s,"max_instances":10,"extensions":{}}`,
		fixtureOperationID, fixtureInstallID, cursorLiteral))
}

// validIdentityJSON is one valid Provider Identity Record over
// the shared identities: an Antigravity
// backend_conversation_uuid with a non-null realm fingerprint.
func validIdentityJSON() []byte {
	return []byte(fmt.Sprintf(`{"schema":"urn:ax:schema:provider-identity","schema_version":"1.0.0","record_id":%q,"subject_id":%q,"session_id":%q,"provider_id":"antigravity","provider_version":"1.1.14","provider_version_range":">=1.1.14 <1.2.0","native_session_id":"11111111-2222-4333-8444-555555555555","identity_kind":"backend_conversation_uuid","logical_workspace_id":%q,"backend_realm_fingerprint":%q,"opaque_identity":{},"created_by_host_id":%q,"created_at":%q,"extensions":{}}`,
		fixtureRecordID, fixtureSubjectID, fixtureSubjectID, fixtureWorkspaceID, fixtureRealmPrint, fixtureCreatorHostID, fixtureObservedAt))
}

// availableCapabilityJSON is one available CapabilityResult with
// null reason.
func availableCapabilityJSON() string {
	return fmt.Sprintf(`{"status":"available","reason_code":null,"evidence_ids":[],"observed_at":%q,"extensions":{}}`, fixtureObservedAt)
}

// validCapabilitiesJSON is the exact eight-name capability map
// with every capability available.
func validCapabilitiesJSON() string {
	names := []string{
		"directory_discovery",
		"directory_incremental_scan",
		"directory_head_digest",
		"directory_tail_preview",
		"native_title_read",
		"native_runtime_observation",
		"existing_session_adoption",
		"native_resume",
	}
	parts := make([]string, 0, len(names))
	for _, name := range names {
		quoted, err := json.Marshal(name)
		if err != nil {
			panic(err)
		}
		parts = append(parts, string(quoted)+":"+availableCapabilityJSON())
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// validObservationJSON is one valid Environment Observation over
// the shared identities.
func validObservationJSON() []byte {
	return []byte(fmt.Sprintf(`{"schema":"urn:ax:schema:environment-observation","schema_version":"1.0.0","observation_id":%q,"host_id":%q,"installation_id":%q,"environment_id":%q,"environment_version":"2.1.0","provider_id":%q,"platform":"linux","architecture":"amd64","backend_realm_fingerprint":%q,"capabilities":%s,"authentication_status":"available","runtime_status":"available","observed_at":%q,"extensions":{}}`,
		fixtureObserveID, fixtureHostID, fixtureInstallID, fixtureEnvironmentID, fixtureProviderID, fixtureRealmPrint, validCapabilitiesJSON(), fixtureObservedAt))
}

// mutateMember returns the object with one top-level member
// replaced by the given raw JSON. The replacement is applied to a
// fresh positive fixture, one mutation at a time: every other
// member holds a valid value, so two distinct rules cannot both
// satisfy one witness.
func mutateMember(t *testing.T, body []byte, member, replacement string) []byte {
	t.Helper()
	members := decodeTestMembers(t, body)
	members[member] = replacement
	return encodeTestMembers(t, members)
}

// dropMember returns the object without one top-level member.
func dropMember(t *testing.T, body []byte, member string) []byte {
	t.Helper()
	members := decodeTestMembers(t, body)
	delete(members, member)
	return encodeTestMembers(t, members)
}

func decodeTestMembers(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("mutateMember: fixture is not JSON: %v", err)
	}
	members := make(map[string]string, len(raw))
	for name, value := range raw {
		members[name] = string(value)
	}
	return members
}

func encodeTestMembers(t *testing.T, members map[string]string) []byte {
	t.Helper()
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(members))
	for _, name := range names {
		quoted, err := json.Marshal(name)
		if err != nil {
			t.Fatalf("encodeTestMembers: %v", err)
		}
		parts = append(parts, string(quoted)+":"+members[name])
	}
	return []byte("{" + strings.Join(parts, ",") + "}")
}

// TestEntriesAreDeterministic proves the validation entries are
// pure functions of their input bytes: repeated decoding is
// byte-identical, so there is no hidden cache, no clock read, and
// no durable state whose crash recovery would need evidence. The
// scan journal in dirnode is the story's only durable state and
// belongs to the frozen leaf; this library adds none.
func TestEntriesAreDeterministic(t *testing.T) {
	bodies := [][]byte{[]byte(validTupleJSON()), validObservationJSON(), validIdentityJSON()}
	for index, body := range bodies {
		first, err := environ.DecodeStrictObject(body)
		if err != nil {
			t.Fatalf("body %d: first decode: %v", index, err)
		}
		second, err := environ.DecodeStrictObject(body)
		if err != nil {
			t.Fatalf("body %d: second decode: %v", index, err)
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("body %d: repeated decode differs", index)
		}
	}
	first, err := environ.DecodeTuple([]byte(validTupleJSON()))
	if err != nil {
		t.Fatalf("tuple first decode: %v", err)
	}
	second, err := environ.DecodeTuple([]byte(validTupleJSON()))
	if err != nil {
		t.Fatalf("tuple second decode: %v", err)
	}
	if first != second {
		t.Fatalf("tuple repeated decode differs: %+v vs %+v", first, second)
	}
}
