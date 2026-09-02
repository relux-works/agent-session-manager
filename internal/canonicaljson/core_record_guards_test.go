package canonicaljson

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// The cross-field and grammar guards in this file each survived a review sweep
// that disabled them one at a time while the whole configured gate set stayed
// green. Every case below violates exactly one clause, so a case cannot pass by
// refusing on an earlier disjunct, and every case drives both public identity
// entries.

func TestCoreRecordCrossFieldGuardsRefuseAtProductionEntries(t *testing.T) {
	for _, test := range []struct {
		name      string
		selfField SelfField
		object    func(t *testing.T) map[string]any
		want      string
	}{
		{
			// Section 5.6: no two members may have an equal or case-colliding
			// group_relative_path. Only the path collides; workspace IDs stay
			// strictly sorted and every other member is valid.
			name:      "workspace group member paths case-collide",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validWorkspaceGroupRecordObject()
				members := object["members"].([]any)
				members[1].(map[string]any)["group_relative_path"] = "Payments-API"
				return object
			},
			want: `member paths "payments-api" and "Payments-API" are equal or case-colliding`,
		},
		{
			name:      "workspace group member paths are equal",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validWorkspaceGroupRecordObject()
				members := object["members"].([]any)
				members[1].(map[string]any)["group_relative_path"] = "payments-api"
				return object
			},
			want: `member paths "payments-api" and "payments-api" are equal or case-colliding`,
		},
		{
			// Section 5.5 provider-identity-key grammar [a-z][a-z0-9_.-]{0,63}.
			name:      "provider identity opaque key rejects uppercase",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validProviderIdentityRecordObject()
				object["opaque_identity"] = map[string]any{"Key": "value"}
				return object
			},
			want: `opaque_identity key "Key" must match [a-z][a-z0-9_.-]{0,63}`,
		},
		{
			name:      "provider identity opaque key rejects a leading digit",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validProviderIdentityRecordObject()
				object["opaque_identity"] = map[string]any{"0key": "value"}
				return object
			},
			want: `opaque_identity key "0key" must match [a-z][a-z0-9_.-]{0,63}`,
		},
		{
			name:      "provider identity opaque key rejects 65 characters",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validProviderIdentityRecordObject()
				object["opaque_identity"] = map[string]any{"a" + strings.Repeat("b", 64): "value"}
				return object
			},
			want: "must match [a-z][a-z0-9_.-]{0,63}",
		},
		{
			// Section 5.4: the Checkpoint Record status member is the literal
			// validated; no other status may be attested.
			name:      "checkpoint status must be validated",
			selfField: SelfCheckpointID,
			object: func(t *testing.T) map[string]any {
				object := validCheckpointRecordObject(true)
				object["status"] = "pending"
				return object
			},
			want: `member status is "pending", want "validated"`,
		},
		{
			// Section 5.2 session.bootstrap_aborted: after identity is
			// established exactly one identity field is non-null.
			name:      "bootstrap aborted after identity rejects both identity fields",
			selfField: SelfEventID,
			object: func(t *testing.T) map[string]any {
				object := validSessionEventObject(lowestSessionEventVersion(t, "session.bootstrap_aborted"), "session.bootstrap_aborted")
				payload := object["payload"].(map[string]any)
				payload["failure_phase"] = "after_identity"
				payload["provider_identity_record_id"] = zeroDigest
				payload["manager_session_ref"] = "manager-1"
				return object
			},
			want: "session.bootstrap_aborted after identity requires exactly one identity field",
		},
		{
			name:      "bootstrap aborted after identity rejects neither identity field",
			selfField: SelfEventID,
			object: func(t *testing.T) map[string]any {
				object := validSessionEventObject(lowestSessionEventVersion(t, "session.bootstrap_aborted"), "session.bootstrap_aborted")
				payload := object["payload"].(map[string]any)
				payload["failure_phase"] = "before_checkpoint"
				payload["provider_identity_record_id"] = nil
				payload["manager_session_ref"] = nil
				return object
			},
			want: "session.bootstrap_aborted after identity requires exactly one identity field",
		},
		{
			// The complementary branch: before identity both fields are null.
			name:      "bootstrap aborted before identity rejects a provider identity",
			selfField: SelfEventID,
			object: func(t *testing.T) map[string]any {
				object := validSessionEventObject(lowestSessionEventVersion(t, "session.bootstrap_aborted"), "session.bootstrap_aborted")
				payload := object["payload"].(map[string]any)
				payload["failure_phase"] = "before_terminal"
				payload["provider_identity_record_id"] = zeroDigest
				payload["manager_session_ref"] = nil
				return object
			},
			want: "session.bootstrap_aborted before identity requires both identity fields null",
		},
		{
			name:      "bootstrap aborted before identity rejects a manager session reference",
			selfField: SelfEventID,
			object: func(t *testing.T) map[string]any {
				object := validSessionEventObject(lowestSessionEventVersion(t, "session.bootstrap_aborted"), "session.bootstrap_aborted")
				payload := object["payload"].(map[string]any)
				payload["failure_phase"] = "after_process"
				payload["provider_identity_record_id"] = nil
				payload["manager_session_ref"] = "manager-1"
				return object
			},
			want: "session.bootstrap_aborted before identity requires both identity fields null",
		},
		{
			// Section 5.5: an opaque_identity value is 1..1024 characters. The
			// bound is written inline rather than through a bound helper, so it
			// is proven here instead of in the derived bounds inventory.
			name:      "provider identity opaque value rejects 1025 characters",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validProviderIdentityRecordObject()
				object["opaque_identity"] = map[string]any{"key": strings.Repeat("x", 1025)}
				return object
			},
			want: `opaque_identity["key"] must contain 1..1024 Unicode characters`,
		},
		{
			name:      "provider identity opaque value rejects the empty string",
			selfField: SelfRecordID,
			object: func(t *testing.T) map[string]any {
				object := validProviderIdentityRecordObject()
				object["opaque_identity"] = map[string]any{"key": ""}
				return object
			},
			want: `opaque_identity["key"] must contain 1..1024 Unicode characters`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object(t)), test.selfField, test.want)
		})
	}
}

// TestProviderIdentityOpaqueValueUpperBoundAcceptsAtItsLimit pairs the inline
// 1..1024 opaque value bound with its at-limit acceptance, the same way the
// derived bounds inventory pairs every helper-mediated bound.
func TestProviderIdentityOpaqueValueUpperBoundAcceptsAtItsLimit(t *testing.T) {
	object := validProviderIdentityRecordObject()
	object["opaque_identity"] = map[string]any{"key": strings.Repeat("x", 1024)}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)

	object["opaque_identity"] = map[string]any{"key": "x"}
	assertIdentityEntriesAcceptShape(t, mustJSON(t, object), SelfRecordID)
}

// TestLeaseEpochCouplingFollowsOnlyTheClausesSectionFiveThreeDeclares pins both
// what Section 5.3 licenses and what it does not.
//
// The pinned specification declares three couplings for these members:
// predecessor_lease_id is "Null only at epoch 1"; "An epoch-1 create lease MUST
// have a null predecessor"; and checkpoint_id is "Null only for epoch-1
// create". It nowhere declares that an epoch-1 lease must have reason create,
// nor that a create lease must be at epoch 1. The refuse cases pin the three
// declared couplings and the accept cases pin the absence of the two that were
// previously inferred, so reintroducing either inference reddens the suite.
func TestLeaseEpochCouplingFollowsOnlyTheClausesSectionFiveThreeDeclares(t *testing.T) {
	for _, test := range []struct {
		name   string
		object func() map[string]any
		want   string
	}{
		{
			name: "epoch above one requires a predecessor",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("4")
				object["predecessor_lease_id"] = nil
				object["checkpoint_id"] = zeroDigest
				return object
			},
			want: "Lease Record predecessor_lease_id must be non-null after epoch 1",
		},
		{
			name: "epoch-one create refuses a predecessor",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("1")
				object["reason"] = "create"
				object["predecessor_lease_id"] = priorLease
				object["checkpoint_id"] = nil
				return object
			},
			want: "Lease Record epoch-1 create predecessor_lease_id must be null",
		},
		{
			name: "epoch above one requires a checkpoint",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("4")
				object["reason"] = "graceful_takeover"
				object["predecessor_lease_id"] = priorLease
				object["checkpoint_id"] = nil
				return object
			},
			want: "Lease Record checkpoint_id must be non-null unless the lease is an epoch-1 create",
		},
		{
			// The complement of "epoch-1 create" includes an epoch-1 lease with
			// any other reason, which the previous inference never reached.
			name: "epoch-one recovery requires a checkpoint",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("1")
				object["reason"] = "recovery"
				object["predecessor_lease_id"] = nil
				object["checkpoint_id"] = nil
				return object
			},
			want: "Lease Record checkpoint_id must be non-null unless the lease is an epoch-1 create",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesRefuseWithReason(t, mustJSON(t, test.object()), SelfRecordID, test.want)
		})
	}

	for _, test := range []struct {
		name   string
		object func() map[string]any
	}{
		{
			// Section 5.3 declares no epoch-1 => create direction. Refusing this
			// record refused a legitimate lease.
			name: "epoch-one recovery with a checkpoint is admitted",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("1")
				object["reason"] = "recovery"
				object["predecessor_lease_id"] = nil
				object["checkpoint_id"] = zeroDigest
				return object
			},
		},
		{
			// Section 5.3 declares no create => epoch-1 direction either.
			name: "create above epoch one with a predecessor and checkpoint is admitted",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("4")
				object["reason"] = "create"
				object["predecessor_lease_id"] = priorLease
				object["checkpoint_id"] = zeroDigest
				return object
			},
		},
		{
			name: "epoch-one create with a null checkpoint is admitted",
			object: func() map[string]any {
				object := validLeaseRecordObject()
				object["epoch"] = json.Number("1")
				object["reason"] = "create"
				object["predecessor_lease_id"] = nil
				object["checkpoint_id"] = nil
				return object
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertIdentityEntriesAcceptShape(t, mustJSON(t, test.object()), SelfRecordID)
		})
	}
}

// TestObservationEntryRefusalsBeforeShapeValidationReachProductionEntries
// covers the guards a derived mutation sweep found unproven: every refusal the
// two exported Observation entries emit before the event shape is reached.
// Disabling any of them previously left the suite green.
func TestObservationEntryRefusalsBeforeShapeValidationReachProductionEntries(t *testing.T) {
	valid := mustJSON(t, validObservationEventObject())

	for _, test := range []struct {
		name  string
		input []byte
		want  string
	}{
		{"malformed JSON", []byte(`{"stream_id":`), "decode:"},
		{"floating-point number", mustJSON(t, observationWithSequenceLiteral(t, "1.5")), "numbers:"},
		{"integer outside the safe interval", mustJSON(t, observationWithSequenceLiteral(t, "9007199254740992")), "numbers:"},
		{"array instead of an object", []byte(`[]`), "input must be a JSON object"},
	} {
		t.Run("event "+test.name, func(t *testing.T) {
			err := ValidateObservationEvent(test.input)
			if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateObservationEvent(%s) error = %v, want observation refusal containing %q", test.name, err, test.want)
			}
		})
	}

	for _, test := range []struct {
		name   string
		events [][]byte
		want   string
	}{
		{"empty stream", nil, "stream must contain at least one event"},
		{"empty stream slice", [][]byte{}, "stream must contain at least one event"},
		{"malformed JSON", [][]byte{[]byte(`{"stream_id":`)}, "event 0 decode:"},
		{"floating-point number", [][]byte{mustJSON(t, observationWithSequenceLiteral(t, "1.5"))}, "event 0 numbers:"},
		{"array instead of an object", [][]byte{[]byte(`[]`)}, "event 0 must be a JSON object"},
		{"invalid event shape", [][]byte{observationWithoutSequence(t)}, "event 0:"},
		{"invalid second event shape", [][]byte{valid, observationWithoutSequence(t)}, "event 1:"},
	} {
		t.Run("stream "+test.name, func(t *testing.T) {
			err := ValidateObservationStream(test.events)
			if err == nil || !errors.Is(err, ErrInvalidObservation) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("ValidateObservationStream(%s) error = %v, want observation refusal containing %q", test.name, err, test.want)
			}
		})
	}
}

func observationWithSequenceLiteral(t *testing.T, literal string) map[string]any {
	t.Helper()
	object := validObservationEventObject()
	object["sequence"] = json.Number(literal)
	return object
}

func observationWithoutSequence(t *testing.T) []byte {
	t.Helper()
	object := validObservationEventObject()
	delete(object, "sequence")
	return mustJSON(t, object)
}

// TestSessionEventPredecessorsRefuseANonArrayAtProductionEntries pins the
// requireArrayMinimum type refusal by its own message. Without the message
// assertion, deleting the type guard still refuses — on the entry-count clause
// instead — so a passing test would say nothing about which gate ran.
func TestSessionEventPredecessorsRefuseANonArrayAtProductionEntries(t *testing.T) {
	object := validSessionEventObject(lowestSessionEventVersion(t, "session.created"), "session.created")
	object["predecessors"] = "sha256:" + strings.Repeat("0", 64)
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfEventID, "member predecessors must be an array")
}

// TestSessionEventPredecessorsPresenceRefusalIsSubsumedByTheClosedMemberSet
// pins the one guard in core_records.go that a mechanical disable-the-guard
// sweep cannot kill.
//
// requireArrayMinimum refuses an absent member before it refuses a non-array
// one, but its only production call site reads `predecessors` after
// requireExactMembers has already refused any Session Event that omits it. The
// presence branch is therefore unreachable from every candidate the public
// entries accept, and disabling it is an equivalent mutant rather than a
// coverage gap. This test names the subsuming refusal and asserts it, so if the
// closed member set ever stops covering `predecessors` the subsumption claim
// fails with it.
func TestSessionEventPredecessorsPresenceRefusalIsSubsumedByTheClosedMemberSet(t *testing.T) {
	object := validSessionEventObject(lowestSessionEventVersion(t, "session.created"), "session.created")
	delete(object, "predecessors")
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, object), SelfEventID,
		`Session Event is missing required member "predecessors"`)
}
