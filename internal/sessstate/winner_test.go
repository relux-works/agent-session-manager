package sessstate

import (
	"reflect"
	"strings"
	"testing"
)

// baseBootstrap returns the created+launched specs opening every
// winner test under lease A at epoch 1.
func baseBootstrap(record Record) []eventSpec {
	return []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("codex")},
	}
}

func decodeTestRecord(t *testing.T) Record {
	t.Helper()
	decoded, err := DecodeRecord(buildRecord(t, nil))
	if err != nil {
		t.Fatalf("DecodeRecord error = %v", err)
	}
	return decoded
}

func reduceSpecs(t *testing.T, record Record, specs []eventSpec, extra Input) (Projection, error) {
	t.Helper()
	raw := buildRecord(t, nil)
	_, events, _ := buildChain(t, raw, specs)
	input := Input{Record: record, Events: events}
	input.Union = extra.Union
	input.Local = extra.Local
	input.LocalHostID = extra.LocalHostID
	return Reduce(input)
}

func conflictKinds(projection Projection) []string {
	var kinds []string
	for _, conflict := range projection.Conflicts {
		kinds = append(kinds, conflict.Kind)
	}
	return kinds
}

// TestWinnerResolutionClearWin requires the plain succession — epoch 1
// to epoch 2 — to select the higher epoch with no conflict recorded.
func TestWinnerResolutionClearWin(t *testing.T) {
	record := decodeTestRecord(t)
	// The takeover stales the running winner, so the new owner stops
	// and resumes before running again: stale to stopped to running.
	specs := append(baseBootstrap(record),
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "session.stopped", payload: stoppedPayload()},
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "session.resumed", payload: resumedPayload()},
	)
	projection, err := reduceSpecs(t, record, specs, Input{})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner != (LeaseHead{Epoch: 2, LeaseID: testLeaseIDB}) {
		t.Fatalf("winner = %+v, want epoch 2 lease B", projection.Winner)
	}
	if len(projection.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v, want none on a clear win", projection.Conflicts)
	}
	if projection.State != StateRunning {
		t.Fatalf("state = %q, want running", projection.State)
	}
}

// TestWinnerResolutionTie requires the equal-epoch case to resolve by
// the bytewise-greater lease ID under the named tie rule, with the
// conflict recorded and the divergent_history warning raised.
func TestWinnerResolutionTie(t *testing.T) {
	record := decodeTestRecord(t)
	specs := baseBootstrap(record)
	// The rival shares epoch 1 with the chain lease but sorts below
	// it, so the chain lease keeps the epoch while both the tie and
	// the rival's losing branch are reported.
	union := []LeaseHead{{Epoch: 1, LeaseID: testLeaseIDR}}
	projection, err := reduceSpecs(t, record, specs, Input{Union: union})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner != (LeaseHead{Epoch: 1, LeaseID: testLeaseID}) {
		t.Fatalf("winner = %+v, want epoch 1 chain lease", projection.Winner)
	}
	kinds := conflictKinds(projection)
	if len(kinds) != 2 || kinds[0] != ConflictLosingBranchPreserved || kinds[1] != ConflictSameEpochTie {
		t.Fatalf("conflicts = %v, want [losing_branch_preserved same_epoch_tie]", kinds)
	}
	for _, conflict := range projection.Conflicts {
		if conflict.Kind == ConflictSameEpochTie && conflict.Rule != ResolveRuleGreatestLeaseIDWins {
			t.Fatalf("rule = %q, want greatest-lease-id-wins", conflict.Rule)
		}
	}
	assertWarning(t, projection, "divergent_history")
}

// TestWinnerResolutionTieOrdersBytewise pins the tie-break direction:
// the greater ID wins even when the union lists it first, so wall
// clock or arrival order cannot bias the outcome.
func TestWinnerResolutionTieOrdersBytewise(t *testing.T) {
	record := decodeTestRecord(t)
	specs := baseBootstrap(record)
	union := []LeaseHead{
		{Epoch: 1, LeaseID: testLeaseIDC},
		{Epoch: 1, LeaseID: testLeaseIDB},
	}
	projection, err := reduceSpecs(t, record, specs, Input{Union: union})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner.LeaseID != testLeaseIDC {
		t.Fatalf("winner = %s, want lease C regardless of union order", projection.Winner.LeaseID)
	}
	kinds := conflictKinds(projection)
	want := map[string]bool{
		ConflictSameEpochTie: true, ConflictLosingBranchPreserved: true,
		ConflictUnionSupersedesChain: true,
	}
	if len(kinds) != len(want) {
		t.Fatalf("conflicts = %v, want the tie with its losing branch and off-chain winner", kinds)
	}
	for _, kind := range kinds {
		if !want[kind] {
			t.Fatalf("conflicts = %v, want the tie with its losing branch and off-chain winner", kinds)
		}
	}
}

// TestWinnerResolutionGap requires an epoch jump in the authoritative
// chain to keep the greatest tuple as winner while recording the gap
// conflict: both sides are not individually valid (the jump skips the
// predecessor-plus-one rule), so the named rule carries the healing
// condition.
func TestWinnerResolutionGap(t *testing.T) {
	record := decodeTestRecord(t)
	specs := append(baseBootstrap(record),
		eventSpec{epoch: 3, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
	)
	projection, err := reduceSpecs(t, record, specs, Input{})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner != (LeaseHead{Epoch: 3, LeaseID: testLeaseIDB}) {
		t.Fatalf("winner = %+v, want epoch 3 lease B", projection.Winner)
	}
	kinds := conflictKinds(projection)
	if len(kinds) != 1 || kinds[0] != ConflictEpochGap {
		t.Fatalf("conflicts = %v, want exactly [epoch_gap]", kinds)
	}
	if projection.Conflicts[0].Rule != ResolveRuleGreatestTupleWins {
		t.Fatalf("rule = %q, want greatest-tuple-wins", projection.Conflicts[0].Rule)
	}
	assertWarning(t, projection, "divergent_history")
}

// TestWinnerResolutionLosingBranchPreserved requires union leases that
// lose the tuple comparison to be reported preserved-never-applied
// while the chain winner stands. The rival shares the head epoch, so
// its tie reports at epoch 2.
func TestWinnerResolutionLosingBranchPreserved(t *testing.T) {
	record := decodeTestRecord(t)
	specs := append(baseBootstrap(record),
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
	)
	union := []LeaseHead{{Epoch: 2, LeaseID: testLeaseIDR}}
	projection, err := reduceSpecs(t, record, specs, Input{Union: union})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner != (LeaseHead{Epoch: 2, LeaseID: testLeaseIDB}) {
		t.Fatalf("winner = %+v, want epoch 2 lease B", projection.Winner)
	}
	kinds := conflictKinds(projection)
	if len(kinds) != 2 || kinds[0] != ConflictLosingBranchPreserved || kinds[1] != ConflictSameEpochTie {
		t.Fatalf("conflicts = %v, want [losing_branch_preserved same_epoch_tie]", kinds)
	}
	assertConflictDetailContains(t, projection, ConflictSameEpochTie, "epoch 2 is shared")
}

// TestWinnerResolutionUnionSupersedesChain requires an off-chain union
// winner to move the reported winner without rewriting authoritative
// state, under its named rule.
func TestWinnerResolutionUnionSupersedesChain(t *testing.T) {
	record := decodeTestRecord(t)
	specs := baseBootstrap(record)
	union := []LeaseHead{{Epoch: 2, LeaseID: testLeaseIDB}}
	projection, err := reduceSpecs(t, record, specs, Input{Union: union})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner != (LeaseHead{Epoch: 2, LeaseID: testLeaseIDB}) {
		t.Fatalf("winner = %+v, want epoch 2 lease B", projection.Winner)
	}
	if projection.State != StateRunning {
		t.Fatalf("state = %q, want running: authoritative state is unchanged", projection.State)
	}
	kinds := conflictKinds(projection)
	if len(kinds) != 1 || kinds[0] != ConflictUnionSupersedesChain {
		t.Fatalf("conflicts = %v, want exactly [union_supersedes_chain]", kinds)
	}
	if projection.Conflicts[0].Rule != ResolveRuleAuthoritativeStateUnchanged {
		t.Fatalf("rule = %q, want authoritative-state-unchanged", projection.Conflicts[0].Rule)
	}
}

// TestWinnerResolutionPastEpochTie resolves a tie planted at a past
// epoch against the lease the chain actually stood on: the union rival
// at epoch 1 loses to the chain's epoch-2 head, and the epoch-1 tie is
// still reported because two distinct leases shared it.
func TestWinnerResolutionPastEpochTie(t *testing.T) {
	record := decodeTestRecord(t)
	specs := append(baseBootstrap(record),
		eventSpec{epoch: 2, leaseID: testLeaseIDB, typ: "lease.transferred", payload: transferredPayload(testLeaseID, testLeaseIDB)},
	)
	union := []LeaseHead{{Epoch: 1, LeaseID: testLeaseIDC}}
	projection, err := reduceSpecs(t, record, specs, Input{Union: union})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.Winner != (LeaseHead{Epoch: 2, LeaseID: testLeaseIDB}) {
		t.Fatalf("winner = %+v, want epoch 2 lease B", projection.Winner)
	}
	kinds := conflictKinds(projection)
	want := map[string]bool{ConflictSameEpochTie: true, ConflictLosingBranchPreserved: true}
	if len(kinds) != 2 {
		t.Fatalf("conflicts = %v, want tie plus losing branch", kinds)
	}
	for _, kind := range kinds {
		if !want[kind] {
			t.Fatalf("conflicts = %v, want tie plus losing branch", kinds)
		}
	}
	assertConflictDetailContains(t, projection, ConflictSameEpochTie, "epoch 1 is shared")
}

// TestProviderMismatchKeepsRecordPinned requires a launch under a
// foreign provider to record the named conflict while the record
// provider stays authoritative.
func TestProviderMismatchKeepsRecordPinned(t *testing.T) {
	record := decodeTestRecord(t)
	specs := []eventSpec{
		{typ: "session.created", payload: createdPayload(record.RecordID)},
		{typ: "provider.launched", payload: launchedPayload("qwen")},
	}
	projection, err := reduceSpecs(t, record, specs, Input{})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if projection.State != StateRunning {
		t.Fatalf("state = %q, want running", projection.State)
	}
	if projection.Provider.ID != "codex" {
		t.Fatalf("provider = %q, want record-pinned codex", projection.Provider.ID)
	}
	if projection.Provider.Version != "" {
		t.Fatalf("provider version = %q, want empty: a mismatching launch contributes no version", projection.Provider.Version)
	}
	kinds := conflictKinds(projection)
	if len(kinds) != 1 || kinds[0] != ConflictProviderMismatch {
		t.Fatalf("conflicts = %v, want exactly [provider_mismatch]", kinds)
	}
	if projection.Conflicts[0].Rule != ResolveRuleRecordPinnedProviderWins {
		t.Fatalf("rule = %q, want record-pinned-provider-wins", projection.Conflicts[0].Rule)
	}
}

// TestWinnerResolutionDuplicateOffChainUnionKeepsConflict requires a
// repeated off-chain union entry to leave the resolution unchanged: the
// winner stands, the union_supersedes_chain conflict stays recorded,
// and the divergent_history warning stays raised. Clearing the
// off-chain flag on the duplicate silently drops both.
func TestWinnerResolutionDuplicateOffChainUnionKeepsConflict(t *testing.T) {
	record := decodeTestRecord(t)
	specs := baseBootstrap(record)
	once, err := reduceSpecs(t, record, specs, Input{Union: []LeaseHead{{Epoch: 3, LeaseID: testLeaseIDC}}})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if kinds := conflictKinds(once); len(kinds) != 1 || kinds[0] != ConflictUnionSupersedesChain {
		t.Fatalf("conflicts = %v, want exactly [union_supersedes_chain]", kinds)
	}
	twice, err := reduceSpecs(t, record, specs, Input{Union: []LeaseHead{
		{Epoch: 3, LeaseID: testLeaseIDC},
		{Epoch: 3, LeaseID: testLeaseIDC},
	}})
	if err != nil {
		t.Fatalf("Reduce error = %v", err)
	}
	if twice.Winner != once.Winner {
		t.Fatalf("winner = %+v, want the single-entry winner %+v", twice.Winner, once.Winner)
	}
	if kinds := conflictKinds(twice); len(kinds) != 1 || kinds[0] != ConflictUnionSupersedesChain {
		t.Fatalf("conflicts = %v, want exactly [union_supersedes_chain]", kinds)
	}
	assertWarning(t, twice, "divergent_history")
}

// TestWinnerResolutionRepeatedOffChainUnionStaysResolved goes one step
// past the duplicate: three copies plus a losing branch, in both
// orders, still resolve to the same winner with the losing branch, the
// epoch-1 tie, and the off-chain winner all reported.
func TestWinnerResolutionRepeatedOffChainUnionStaysResolved(t *testing.T) {
	record := decodeTestRecord(t)
	specs := baseBootstrap(record)
	unions := map[string][]LeaseHead{
		"winner first": {
			{Epoch: 3, LeaseID: testLeaseIDC},
			{Epoch: 3, LeaseID: testLeaseIDC},
			{Epoch: 3, LeaseID: testLeaseIDC},
			{Epoch: 1, LeaseID: testLeaseIDR},
		},
		"loser first": {
			{Epoch: 1, LeaseID: testLeaseIDR},
			{Epoch: 3, LeaseID: testLeaseIDC},
			{Epoch: 3, LeaseID: testLeaseIDC},
			{Epoch: 3, LeaseID: testLeaseIDC},
		},
	}
	for name, union := range unions {
		t.Run(name, func(t *testing.T) {
			projection, err := reduceSpecs(t, record, specs, Input{Union: union})
			if err != nil {
				t.Fatalf("Reduce error = %v", err)
			}
			if projection.Winner != (LeaseHead{Epoch: 3, LeaseID: testLeaseIDC}) {
				t.Fatalf("winner = %+v, want epoch 3 lease C", projection.Winner)
			}
			want := map[string]bool{
				ConflictLosingBranchPreserved: true, ConflictSameEpochTie: true,
				ConflictUnionSupersedesChain: true,
			}
			kinds := conflictKinds(projection)
			if len(kinds) != len(want) {
				t.Fatalf("conflicts = %v, want losing branch, tie, and off-chain winner", kinds)
			}
			for _, kind := range kinds {
				if !want[kind] {
					t.Fatalf("conflicts = %v, want losing branch, tie, and off-chain winner", kinds)
				}
			}
			assertWarning(t, projection, "divergent_history")
		})
	}
}

// TestReduceLocalLeaseTupleRule pins the local-stale rule at the tuple
// boundary: only a local lease that lost the tuple comparison stales
// the projection. An equal lease and a strictly greater lease — by
// epoch or by lease ID — stand with the winner.
func TestReduceLocalLeaseTupleRule(t *testing.T) {
	record := decodeTestRecord(t)
	specs := baseBootstrap(record)
	cases := []struct {
		name      string
		local     LeaseHead
		wantState State
		wantStale bool
	}{
		{"equal stands", LeaseHead{Epoch: 1, LeaseID: testLeaseID}, StateRunning, false},
		{"greater epoch stands", LeaseHead{Epoch: 9, LeaseID: testLeaseIDC}, StateRunning, false},
		{"greater lease id stands", LeaseHead{Epoch: 1, LeaseID: testLeaseIDC}, StateRunning, false},
		{"lesser lease stales", LeaseHead{Epoch: 1, LeaseID: testLeaseIDR}, StateStale, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projection, err := reduceSpecs(t, record, specs, Input{Local: tc.local})
			if err != nil {
				t.Fatalf("Reduce error = %v", err)
			}
			if projection.State != tc.wantState {
				t.Fatalf("state = %q, want %q", projection.State, tc.wantState)
			}
			if projection.Winner != (LeaseHead{Epoch: 1, LeaseID: testLeaseID}) {
				t.Fatalf("winner = %+v, want the chain head: the local lease never moves the winner", projection.Winner)
			}
			hasStale := false
			for _, warning := range projection.Warnings {
				if warning == "stale_process" {
					hasStale = true
				}
			}
			if hasStale != tc.wantStale {
				t.Fatalf("stale_process warning = %v, want %v (warnings = %v)", hasStale, tc.wantStale, projection.Warnings)
			}
		})
	}
}

func assertConflictDetailContains(t *testing.T, projection Projection, kind, fragment string) {
	t.Helper()
	for _, conflict := range projection.Conflicts {
		if conflict.Kind == kind {
			if !strings.Contains(conflict.Detail, fragment) {
				t.Fatalf("conflict %s detail = %q, want fragment %q", kind, conflict.Detail, fragment)
			}
			return
		}
	}
	t.Fatalf("no conflict of kind %s in %+v", kind, projection.Conflicts)
}

func assertWarning(t *testing.T, projection Projection, warning string) {
	t.Helper()
	for _, candidate := range projection.Warnings {
		if candidate == warning {
			return
		}
	}
	t.Fatalf("warnings = %v, want %q", projection.Warnings, warning)
}

// With no authoritative event there is no owner/process lease to fence.
// Union may report a winner, but Local is ignored, including malformed and
// partial tuples. This is a stated input bound, not invented validation.
func TestReduceEmptyChainLocalIsIgnored(t *testing.T) {
	record := decodeTestRecord(t)
	winner := LeaseHead{Epoch: 3, LeaseID: testLeaseIDC}
	for _, union := range []struct {
		name   string
		leases []LeaseHead
		winner LeaseHead
	}{
		{"absent_union", nil, LeaseHead{}},
		{"selected_union_winner", []LeaseHead{{Epoch: 2, LeaseID: testLeaseIDB}, winner}, winner},
	} {
		t.Run(union.name, func(t *testing.T) {
			baseline, err := Reduce(Input{Record: record, Union: union.leases})
			if err != nil {
				t.Fatal(err)
			}
			if baseline.Winner != union.winner || baseline.State != StateCreating {
				t.Fatalf("union baseline: %+v", baseline)
			}
			if len(union.leases) > 0 {
				found := false
				for _, conflict := range baseline.Conflicts {
					if conflict.Kind == ConflictUnionSupersedesChain && conflict.Rule == ResolveRuleAuthoritativeStateUnchanged {
						found = true
					}
				}
				if !found {
					t.Fatal("selected union winner has no named unchanged-state conflict")
				}
			}
			for _, local := range []struct {
				name string
				head LeaseHead
			}{
				{"absent", LeaseHead{}},
				{"lesser", LeaseHead{Epoch: 1, LeaseID: testLeaseID}},
				{"equal_to_union", winner},
				{"greater", LeaseHead{Epoch: 4, LeaseID: testLeaseIDB}},
				{"malformed_id", LeaseHead{Epoch: 1, LeaseID: "not-a-uuid"}},
				{"epoch_only", LeaseHead{Epoch: 1}},
				{"id_only", LeaseHead{LeaseID: testLeaseID}},
			} {
				t.Run(local.name, func(t *testing.T) {
					got, err := Reduce(Input{Record: record, Union: union.leases, Local: local.head, LocalHostID: testHostID})
					if err != nil {
						t.Fatalf("empty-chain Local must not be validated: %v", err)
					}
					if got.LocalRole != "" || got.OwnerHostID != "" {
						t.Fatalf("no authoritative owner: %+v", got)
					}
					if !reflect.DeepEqual(got, baseline) {
						t.Fatalf("Local changed empty-chain projection: got %+v want %+v", got, baseline)
					}
				})
			}
		})
	}
}
