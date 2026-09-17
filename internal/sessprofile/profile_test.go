package sessprofile

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDeriveEmptyChainIsCreationPair(t *testing.T) {
	for _, creation := range []string{ProfileStandard, ProfileYOLO} {
		pair, err := Derive(Record{SessionID: testSessionID, RecordID: zeroDigest, Creation: creation}, nil)
		if err != nil {
			t.Fatalf("Derive(empty, %s) error = %v", creation, err)
		}
		mustPairEqual(t, pair, Pair{Profile: creation}, "Derive(empty)")
	}
}

func TestDeriveNewestChangeWins(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	appendTestEvent(t, repository, testSessionID, []string{change}, 1, testLeaseID, 3, "session.idle", map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true})
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "Derive(newest change)")
}

func TestDeriveSecondChangeSupersedesFirst(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	middle := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	// A second change under a successor lease supersedes the first:
	// newest in lease/sequence order governs.
	second := appendTestEvent(t, repository, testSessionID, []string{middle}, 2, testLeaseIDB, 1, "profile.changed", changedPayload(ProfileYOLO, ProfileStandard, false))
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileStandard, Source: second, HasSource: true}, "Derive(second change)")
}

func TestDeriveChangeBackRestoresCreationValueWithSource(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	middle := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	back := appendTestEvent(t, repository, testSessionID, []string{middle}, 1, testLeaseID, 3, "profile.changed", changedPayload(ProfileYOLO, ProfileStandard, false))
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	// The value equals the creation value but the authority is the
	// newest change, not the record: the source stays non-null.
	mustPairEqual(t, pair, Pair{Profile: ProfileStandard, Source: back, HasSource: true}, "Derive(change back)")
}

func TestDeriveUnconfirmedChangeIsAuthoritative(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	// Confirmation is enforced at publication (MintChangeEvent
	// refuses this vector); a chained change is authoritative by
	// construction and derivation reads only its target.
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, false))
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "Derive(unconfirmed chained change)")
}

func TestDeriveLaunchResumeForkPairsAreInert(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileYOLO)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	second := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "provider.launched", launchedPayload("codex", "0.147.0", ProfileStandard, "", ProfileStandard))
	third := appendTestEvent(t, repository, testSessionID, []string{second}, 1, testLeaseID, 3, "session.resumed", resumedPayload(testCheckpoint, ProfileStandard, ""))
	record, events := decodeTestChain(t, repository, testSessionID)
	_ = third
	pair, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	// The launch/resume pairs disagree with the creation pair, but
	// derivation reads only the record and profile.changed events:
	// pair checks (CheckLaunchPair/CheckResumedPair) own those
	// contradictions, not the fold.
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO}, "Derive(pair events inert)")
}

func TestDeriveAcrossSuccessorLease(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	// A successor lease restarts at sequence 1 referencing the
	// predecessor head; the newest change still governs.
	appendTestEvent(t, repository, testSessionID, []string{change}, 2, testLeaseIDB, 1, "session.idle", map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "Derive(across lease)")
}

func TestDeriveForHeadsExcludesLaterChange(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	appendTestEvent(t, repository, testSessionID, []string{change}, 1, testLeaseID, 3, "profile.changed", changedPayload(ProfileYOLO, ProfileStandard, false))
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := DeriveForHeads(record, events, []string{change})
	if err != nil {
		t.Fatalf("DeriveForHeads error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "DeriveForHeads(closure pins E1)")
	head, err := Derive(record, events)
	if err != nil {
		t.Fatalf("Derive error = %v", err)
	}
	if head.Equal(pair) {
		t.Fatalf("session head %+v equals the pinned closure; the later change must move the head", head)
	}
}

func TestDeriveForHeadsWithoutChangeIsCreation(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileYOLO)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileYOLO, ProfileStandard, false))
	record, events := decodeTestChain(t, repository, testSessionID)
	pair, err := DeriveForHeads(record, events, []string{first})
	if err != nil {
		t.Fatalf("DeriveForHeads error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO}, "DeriveForHeads(pre-change closure)")
}

func TestDeriveForHeadsIgnoresPostClosureCorruption(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	record, events := decodeTestChain(t, repository, testSessionID)
	// A post-closure event the repository could never have authored
	// (a repeated sequence) must not disturb the closure
	// derivation: consumers never consult past the heads.
	corrupt := events
	corrupt = append(corrupt, Event{ID: zeroDigest, Type: "session.idle", SchemaVersion: "1.0.0", SessionID: testSessionID, LeaseEpoch: 1, LeaseID: testLeaseID, Sequence: 2, Predecessors: []string{change}, Payload: map[string]any{}})
	pair, err := DeriveForHeads(record, corrupt, []string{change})
	if err != nil {
		t.Fatalf("DeriveForHeads error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "DeriveForHeads(post-closure ignored)")
	if _, err := Derive(record, corrupt); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("Derive(corrupt tail) error = %v, want invalid_state_transition", err)
	}
}

func TestDeriveRefusesLosingLeaseAndAmbiguity(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	second := appendTestEvent(t, repository, testSessionID, []string{first}, 2, testLeaseIDB, 1, "session.idle", map[string]any{"boundary_ref": "turn-2", "foreground_idle": true, "background_idle": true})
	record, events := decodeTestChain(t, repository, testSessionID)
	_ = second
	base := func() Event {
		return Event{ID: zeroDigest, Type: "profile.changed", SchemaVersion: "1.0.0", SessionID: testSessionID, CreatedByHostID: testHostID, CreatedAt: testChangedAt, LeaseEpoch: 2, LeaseID: testLeaseIDB, Sequence: 2, Predecessors: []string{events[len(events)-1].ID}, Payload: map[string]any{"from": ProfileStandard, "to": ProfileYOLO, "confirmed": true}}
	}
	cases := []struct {
		name     string
		mutate   func(*Event)
		sentinel error
	}{
		{"stale epoch", func(event *Event) { event.LeaseEpoch = 1 }, ErrStaleLease},
		{"divergent lease", func(event *Event) { event.LeaseID = testLeaseIDC }, ErrDivergentLease},
		{"repeated sequence", func(event *Event) { event.Sequence = 1 }, ErrInvalidTransition},
		{"skipped sequence", func(event *Event) { event.Sequence = 5 }, ErrInvalidTransition},
		{"missing predecessor", func(event *Event) { event.Predecessors = []string{zeroDigest} }, ErrInvalidTransition},
		{"empty predecessors", func(event *Event) { event.Predecessors = nil }, ErrInvalidTransition},
		{"zero sequence", func(event *Event) { event.Sequence = 0 }, ErrInvalidTransition},
		{"foreign session", func(event *Event) { event.SessionID = testSessionIDB }, ErrInvalidEvent},
		{"unregistered version", func(event *Event) { event.SchemaVersion = "9.9.9" }, ErrInvalidEvent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base()
			tc.mutate(&candidate)
			input := append(append([]Event(nil), events...), candidate)
			if _, err := Derive(record, input); !errors.Is(err, tc.sentinel) {
				t.Fatalf("Derive(%s) error = %v, want %v", tc.name, err, tc.sentinel)
			}
		})
	}
}

func TestDeriveRefusesFirstEventViolations(t *testing.T) {
	record := Record{SessionID: testSessionID, RecordID: zeroDigest, Creation: ProfileStandard}
	base := Event{ID: "sha256:1111111111111111111111111111111111111111111111111111111111111111", Type: "session.created", SchemaVersion: "1.0.0", SessionID: testSessionID, CreatedByHostID: testHostID, CreatedAt: testCreatedAt, LeaseEpoch: 1, LeaseID: testLeaseID, Sequence: 1, Predecessors: []string{zeroDigest}, Payload: map[string]any{}}
	if _, err := Derive(record, []Event{base}); err != nil {
		t.Fatalf("Derive(valid first) error = %v", err)
	}
	cases := []struct {
		name     string
		mutate   func(*Event)
		sentinel error
	}{
		{"predecessor is not the record", func(event *Event) {
			event.Predecessors = []string{"sha256:2222222222222222222222222222222222222222222222222222222222222222"}
		}, ErrInvalidTransition},
		{"two predecessors", func(event *Event) { event.Predecessors = []string{zeroDigest, zeroDigest} }, ErrInvalidTransition},
		{"sequence is not 1", func(event *Event) { event.Sequence = 2 }, ErrInvalidTransition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			tc.mutate(&candidate)
			if _, err := Derive(record, []Event{candidate}); !errors.Is(err, tc.sentinel) {
				t.Fatalf("Derive(%s) error = %v, want %v", tc.name, err, tc.sentinel)
			}
		})
	}
}

func TestDeriveRefusesMalformedChangePayload(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	record, events := decodeTestChain(t, repository, testSessionID)
	base := func(payload map[string]any) Event {
		return Event{ID: zeroDigest, Type: "profile.changed", SchemaVersion: "1.0.0", SessionID: testSessionID, CreatedByHostID: testHostID, CreatedAt: testChangedAt, LeaseEpoch: 1, LeaseID: testLeaseID, Sequence: 2, Predecessors: []string{first}, Payload: payload}
	}
	cases := []struct {
		name    string
		payload map[string]any
	}{
		{"from outside vocabulary", map[string]any{"from": "turbo", "to": ProfileYOLO, "confirmed": true}},
		{"to outside vocabulary", map[string]any{"from": ProfileStandard, "to": "turbo", "confirmed": true}},
		{"from equals to", map[string]any{"from": ProfileStandard, "to": ProfileStandard, "confirmed": true}},
		{"from absent", map[string]any{"to": ProfileYOLO, "confirmed": true}},
		{"to absent", map[string]any{"from": ProfileStandard, "confirmed": true}},
		{"confirmed absent", map[string]any{"from": ProfileStandard, "to": ProfileYOLO}},
		{"confirmed not boolean", map[string]any{"from": ProfileStandard, "to": ProfileYOLO, "confirmed": "yes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := append(append([]Event(nil), events...), base(tc.payload))
			if _, err := Derive(record, input); !errors.Is(err, ErrIntegrity) {
				t.Fatalf("Derive(%s) error = %v, want integrity_failure", tc.name, err)
			}
		})
	}
}

func TestDeriveRefusesBadRecord(t *testing.T) {
	if _, err := Derive(Record{}, nil); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("Derive(empty record) error = %v, want invalid session record", err)
	}
	record := Record{SessionID: testSessionID, RecordID: zeroDigest, Creation: "turbo"}
	if _, err := Derive(record, nil); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("Derive(bad creation) error = %v, want invalid session record", err)
	}
}

func TestDeriveForHeadsRefusals(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	record, events := decodeTestChain(t, repository, testSessionID)
	if _, err := DeriveForHeads(record, events, nil); !errors.Is(err, ErrDerivation) {
		t.Fatalf("DeriveForHeads(no heads) error = %v, want corrupt derivation input", err)
	}
	unknown := "sha256:9999999999999999999999999999999999999999999999999999999999999999"
	if _, err := DeriveForHeads(record, events, []string{unknown}); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("DeriveForHeads(unknown head) error = %v, want integrity_failure", err)
	}
	// A dangling predecessor inside the closure contradicts the
	// admitted history even when the head itself is chained.
	dangling := append([]Event(nil), events...)
	dangling[1].Predecessors = []string{unknown}
	if _, err := DeriveForHeads(record, dangling, []string{change}); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("DeriveForHeads(dangling) error = %v, want integrity_failure", err)
	}
	if _, err := DeriveForHeads(Record{}, events, []string{change}); !errors.Is(err, ErrInvalidRecord) {
		t.Fatalf("DeriveForHeads(bad record) error = %v, want invalid session record", err)
	}
}

func TestProjectDerivesSessionHeadOverRepository(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	first := appendTestEvent(t, repository, testSessionID, []string{reference.RecordID}, 1, testLeaseID, 1, "session.created", createdPayload(reference.RecordID))
	change := appendTestEvent(t, repository, testSessionID, []string{first}, 1, testLeaseID, 2, "profile.changed", changedPayload(ProfileStandard, ProfileYOLO, true))
	projector := &Projector{Repo: repository}
	pair, err := projector.Project(testSessionID)
	if err != nil {
		t.Fatalf("Project error = %v", err)
	}
	mustPairEqual(t, pair, Pair{Profile: ProfileYOLO, Source: change, HasSource: true}, "Project(head)")
	again, err := projector.Project(testSessionID)
	if err != nil {
		t.Fatalf("Project again error = %v", err)
	}
	mustPairEqual(t, again, pair, "Project(idempotent)")
	closure, err := projector.ProjectForHeads(testSessionID, []string{first})
	if err != nil {
		t.Fatalf("ProjectForHeads error = %v", err)
	}
	mustPairEqual(t, closure, Pair{Profile: ProfileStandard}, "ProjectForHeads(pre-change)")
	if _, err := projector.Project(testSessionIDB); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("Project(unknown) error = %v, want unknown session", err)
	}
	bare := &Projector{}
	if _, err := bare.Project(testSessionID); !errors.Is(err, ErrDerivation) {
		t.Fatalf("Project(no repo) error = %v, want corrupt derivation input", err)
	}
	var nilProjector *Projector
	if _, err := nilProjector.Project(testSessionID); !errors.Is(err, ErrDerivation) {
		t.Fatalf("Project(nil) error = %v, want corrupt derivation input", err)
	}
	if _, err := bare.ProjectForHeads(testSessionID, []string{first}); !errors.Is(err, ErrDerivation) {
		t.Fatalf("ProjectForHeads(no repo) error = %v, want corrupt derivation input", err)
	}
}

func TestDecodeRecordRefusals(t *testing.T) {
	valid := buildRecord(t, nil)
	if _, err := DecodeRecord(valid); err != nil {
		t.Fatalf("DecodeRecord(valid) error = %v", err)
	}
	cases := []struct {
		name  string
		raw   string
		fault string
	}{
		{"malformed frame", "{", "frame"},
		{"foreign schema", `{"schema":"urn:ax:schema:lease"}`, "schema"},
		{"missing session", `{"schema":"urn:ax:schema:session-record"}`, "session_id"},
		{"missing record id", `{"schema":"urn:ax:schema:session-record","session_id":"` + testSessionID + `"}`, "record_id"},
		{"malformed record id", `{"schema":"urn:ax:schema:session-record","session_id":"` + testSessionID + `","record_id":"nope"}`, "record_id"},
		{"missing creation", `{"schema":"urn:ax:schema:session-record","session_id":"` + testSessionID + `","record_id":"` + zeroDigest + `"}`, "execution_profile"},
		{"creation outside vocabulary", `{"schema":"urn:ax:schema:session-record","session_id":"` + testSessionID + `","record_id":"` + zeroDigest + `","execution_profile":"turbo"}`, "standard|yolo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeRecord([]byte(tc.raw))
			if !errors.Is(err, ErrInvalidRecord) {
				t.Fatalf("DecodeRecord(%s) error = %v, want invalid session record", tc.name, err)
			}
			if !strings.Contains(err.Error(), tc.fault) {
				t.Fatalf("DecodeRecord(%s) error = %v, want %q named", tc.name, err, tc.fault)
			}
		})
	}
}

func TestDecodeEventRefusals(t *testing.T) {
	repository := openTestRepository(t)
	reference := createTestSession(t, repository, testSessionID, "payments-api", ProfileStandard)
	raw := buildEvent(t, eventOptions{sessionID: testSessionID, predecessors: []string{reference.RecordID}, epoch: 1, leaseID: testLeaseID, sequence: 1, eventType: "session.created", payload: createdPayload(reference.RecordID)})
	if _, err := DecodeEvent(raw); err != nil {
		t.Fatalf("DecodeEvent(valid) error = %v", err)
	}
	frame := func(mutate func(map[string]any)) []byte {
		var object map[string]any
		if err := json.Unmarshal(raw, &object); err != nil {
			t.Fatalf("unmarshal built event: %v", err)
		}
		mutate(object)
		out, err := json.Marshal(object)
		if err != nil {
			t.Fatalf("marshal mutated event: %v", err)
		}
		return out
	}
	cases := []struct {
		name  string
		raw   []byte
		fault string
	}{
		{"malformed frame", []byte("{"), "frame"},
		{"foreign schema", frame(func(object map[string]any) { object["schema"] = "urn:ax:schema:lease" }), "schema"},
		{"missing version", frame(func(object map[string]any) { delete(object, "schema_version") }), "schema_version"},
		{"missing session", frame(func(object map[string]any) { delete(object, "session_id") }), "session_id"},
		{"missing type", frame(func(object map[string]any) { delete(object, "event_type") }), "event_type"},
		{"zero epoch", frame(func(object map[string]any) { object["lease_epoch"] = float64(0) }), "lease_epoch"},
		{"missing lease", frame(func(object map[string]any) { delete(object, "lease_id") }), "lease_id"},
		{"malformed lease", frame(func(object map[string]any) { object["lease_id"] = "nope" }), "lease_id"},
		{"zero sequence", frame(func(object map[string]any) { object["lease_sequence"] = float64(0) }), "lease_sequence"},
		{"empty predecessors", frame(func(object map[string]any) { object["predecessors"] = []any{} }), "predecessor"},
		{"malformed predecessor", frame(func(object map[string]any) { object["predecessors"] = []any{"nope"} }), "predecessors"},
		{"missing event id", frame(func(object map[string]any) { delete(object, "event_id") }), "event_id"},
		{"malformed event id", frame(func(object map[string]any) { object["event_id"] = "nope" }), "event_id"},
		{"missing author", frame(func(object map[string]any) { delete(object, "created_by_host_id") }), "created_by_host_id"},
		{"missing instant", frame(func(object map[string]any) { delete(object, "created_at") }), "created_at"},
		{"malformed instant", frame(func(object map[string]any) { object["created_at"] = "yesterday" }), "created_at"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeEvent(tc.raw)
			if !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("DecodeEvent(%s) error = %v, want invalid session event", tc.name, err)
			}
			if !strings.Contains(err.Error(), tc.fault) {
				t.Fatalf("DecodeEvent(%s) error = %v, want %q named", tc.name, err, tc.fault)
			}
		})
	}
}
