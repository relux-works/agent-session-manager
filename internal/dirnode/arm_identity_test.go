package dirnode

import (
	"encoding/json"
	"strings"
	"testing"
)

// This file pins vector reachability against the arm-slide shape
// (review round 2, B3): a negative vector carrying two defects is
// satisfied by whichever arm fires first, and when both arms share
// the same code the code assertion cannot tell them apart. The
// unknown_field vector did exactly this — it inserted a duplicate
// "fields" member, so decodeStrictObject refused on the duplicate
// before the field registry ever ran, with the same code and the
// same detail either path reports.
//
// The audit covered every fixture-surgery vector in this package
// that claims a "missing" or "unknown" outcome:
//
//   - query "missing caller", manifest "missing member", and probe
//     response "missing member" all renamed the member (",\"x\":" to
//     ",\"xX\":"). Renaming fires the unknown-member arm first
//     because every envelope checks unknownMember before
//     missingMember, so the missing arm carried a witness that
//     never fired it. All three now delete the member instead, and
//     TestMissingVectorsDriveTheMissingArm asserts the
//     misses-a-required-member message, which a rename vector
//     cannot produce.
//   - scan request/response "missing member" vectors already
//     delete and probe request "missing member" already deletes.
//     The "unknown member" vectors are NOT audited here at all:
//     three of them named the envelope unknown-member arm while
//     reaching a strict-decode, operation, and node-build arm
//     (review round 3, C1), so a prose claim that each reaches
//     the arm it names is exactly the failure mode.
//     Vector reachability is derived instead, in
//     arm_reachability_test.go: every envelope unknown/missing
//     vector is paired with its arm key, and the test computes
//     the reached arm from the refusal message and fails when
//     that is not the named one.
//   - the operation-level query vectors (unknown operation,
//     parameter, filter, field, preset, sort, flag, pagination)
//     all collapse to the single checkQueryOperation arm by
//     construction; their obligation set is derived and driven
//     member-by-member in query_operations_test.go rather than
//     distinguished by a detail production does not emit.

// TestMissingVectorsDriveTheMissingArm requires the three
// deletion-built missing vectors to refuse with the missing-member
// message through the production entry. A rename-built vector
// would report "carries unknown member" here and fail, so this
// test pins the construction, not just the code.
func TestMissingVectorsDriveTheMissingArm(t *testing.T) {
	t.Parallel()
	t.Run("query missing caller", func(t *testing.T) {
		t.Parallel()
		body := replaceOnce(t, fixtureQueryJSON(), `,"caller":`+fixtureCallerJSON(), ``, 1)
		_, err := DecodeQuery([]byte(body))
		failure := requireCode(t, err, "query_invalid")
		if !strings.Contains(failure.Message(), "misses a required member") {
			t.Fatalf("message = %q, want the missing-member arm, not the unknown-member arm", failure.Message())
		}
	})
	t.Run("manifest missing member", func(t *testing.T) {
		t.Parallel()
		body := replaceOnce(t, fixtureManifestJSON(), `"node_version":"2.4.1",`, ``, 1)
		_, err := DecodeManifest([]byte(body))
		failure := requireCode(t, err, "adapter_protocol_violation")
		if !strings.Contains(failure.Message(), "misses a required member") {
			t.Fatalf("message = %q, want the missing-member arm, not the unknown-member arm", failure.Message())
		}
	})
	t.Run("probe response missing member", func(t *testing.T) {
		t.Parallel()
		body := replaceOnce(t, fixtureProbeResponseJSON(), `"policy_digest":`+quote(fixturePolicyDigest)+`,`, ``, 1)
		_, err := CheckProbeResponse([]byte(body))
		failure := requireCode(t, err, "adapter_protocol_violation")
		if !strings.Contains(failure.Message(), "misses a required member") {
			t.Fatalf("message = %q, want the missing-member arm, not the unknown-member arm", failure.Message())
		}
	})
}

// TestUnknownFieldVectorReachesTheFieldRegistry requires the
// unknown-field vector to carry exactly one "fields" member when
// it reaches production: the strict decoder must accept the
// operation object, so the refusal comes from the field registry
// and not from the duplicate-member gate in front of it. Both
// gates report the same code and detail through DecodeQuery, so
// the single-member construction is the proof of reachability.
func TestUnknownFieldVectorReachesTheFieldRegistry(t *testing.T) {
	t.Parallel()
	body := replaceOnce(t, fixtureQueryJSON(), `"fields":null,"preset":"overview"`, `"fields":["nope"],"preset":null`, 1)
	var envelope struct {
		Operations []json.RawMessage `json:"operations"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("vector is not JSON: %v", err)
	}
	if len(envelope.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(envelope.Operations))
	}
	members, fault := decodeStrictObject(bytesTrimSpace(envelope.Operations[1]))
	if fault != nil {
		t.Fatalf("vector operation carries a strict fault %v; the duplicate gate would fire first", fault)
	}
	fields, present := members["fields"]
	if !present {
		t.Fatal("vector operation carries no fields member; the vector mutates nothing")
	}
	var names []string
	if err := json.Unmarshal(fields, &names); err != nil || len(names) != 1 || names[0] != "nope" {
		t.Fatalf("fields = %s, want exactly [\"nope\"]", string(fields))
	}
	if _, err := DecodeQuery([]byte(body)); requireCode(t, err, "query_invalid") == nil {
		t.Fatal("must refuse the unknown field")
	}
}
