package sessckpt

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// mutate Raw takes captured bytes, applies one decoded-map mutation,
// and re-marshals without re-identifying. The stale digest is part
// of the vector: the production entry must refuse it, and the
// paired control (re-identify a benign mutation, admit with a
// fresh operation) proves the refusal tracks the mutation rather
// than staleness itself.
func mutateRaw(t *testing.T, raw []byte, mutate func(map[string]any)) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	mutate(value)
	out, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCaptureRefusesNonQuiescentBoundary(t *testing.T) {
	chain, created := chainFixture(t)
	cases := []struct {
		name   string
		mutate func(*SafeBoundary)
	}{
		{"cp_n1_background_idle_false", func(b *SafeBoundary) { b.BackgroundIdle = false }},
		{"input_blocked_false", func(b *SafeBoundary) { b.InputBlocked = false }},
		{"foreground_idle_false", func(b *SafeBoundary) { b.ForegroundIdle = false }},
		{"open_processes_nonzero", func(b *SafeBoundary) { b.OpenProcesses = 1 }},
		{"open_database_handles_nonzero", func(b *SafeBoundary) { b.OpenDatabaseHandles = 2 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := openTestStore(t)
			inputs := testInputs([]string{created}, testOpA)
			tc.mutate(&inputs.Boundary)
			_, _, err := store.Capture(chain, inputs)
			mustInvalid(t, err, "Capture(non-quiescent)")
			if !strings.Contains(err.Error(), "safe_boundary") {
				t.Fatalf("error = %v, want the safe_boundary leg named", err)
			}
		})
	}
}

func TestCaptureRefusesBadPersistenceVariant(t *testing.T) {
	chain, created := chainFixture(t)
	cases := []struct {
		name      string
		kind      string
		provider  string
		bundle    string
		wantNamed string
	}{
		// CP-N2: a direct Session Record with a null provider manifest.
		{"cp_n2_direct_null_provider", SessionKindDirect, "", "", "provider_manifest_id"},
		// CP-N3: both persistence IDs non-null.
		{"cp_n3_both_present", SessionKindDirect, testProviderManifest, testBoardBundle, "provider_manifest_id"},
		{"both_absent_task_board", SessionKindTaskBoard, "", "", "task_board_bundle_id"},
		// Swapped variants: valid shapes no session of that kind admits.
		{"swapped_direct_takes_bundle", SessionKindDirect, "", testBoardBundle, "task-board"},
		{"swapped_task_board_takes_provider", SessionKindTaskBoard, testProviderManifest, "", "provider"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := openTestStore(t)
			inputs := testInputs([]string{created}, testOpA)
			inputs.SessionKind = tc.kind
			inputs.ProviderManifestID = tc.provider
			inputs.TaskBoardBundleID = tc.bundle
			_, _, err := store.Capture(chain, inputs)
			mustInvalid(t, err, "Capture(variant)")
			if !strings.Contains(err.Error(), tc.wantNamed) {
				t.Fatalf("error = %v, want %q named", err, tc.wantNamed)
			}
		})
	}
}

func TestAdmitRefusesUnknownSafeBoundaryMember(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	_, raw := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// CP-N4: an unknown safe_boundary.pid member. The mutated
	// bytes keep the stale digest; the control below proves the
	// refusal tracks the member, not staleness.
	mutated := mutateRaw(t, raw, func(value map[string]any) {
		value["safe_boundary"].(map[string]any)["pid"] = float64(4242)
	})
	if _, err := store.Admit(chain, mutated, testOpB, SessionKindDirect); err == nil {
		t.Fatalf("Admit(cp_n4) error = nil, want invalid checkpoint closure")
	} else {
		mustInvalid(t, err, "Admit(cp_n4)")
	}
	// Control: a benign mutation (diagnostic instant), properly
	// re-identified, admits under a fresh operation.
	benign := mutateRaw(t, raw, func(value map[string]any) {
		value["created_at"] = "2026-08-19T04:10:30.000Z"
	})
	var framed map[string]any
	if err := json.Unmarshal(benign, &framed); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Admit(chain, identify(t, framed, "checkpoint_id"), testOpB, SessionKindDirect); err != nil {
		t.Fatalf("Admit(benign re-identified) error = %v", err)
	}
}

func TestCaptureRefusesUnknownSessionAndHead(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	// Unknown session: the chain read propagates the owner's
	// unknown-session refusal.
	inputs := testInputs([]string{created}, testOpA)
	inputs.SessionID = "0198f4c8-3e70-7a11-8a2b-1234567890ff"
	if _, _, err := store.Capture(chain, inputs); !errors.Is(err, sessrepo.ErrUnknownSession) {
		t.Fatalf("Capture(unknown session) = %v, want unknown session", err)
	}
	// Unknown head: the digest is well-formed but names no
	// chained event, so the owner's unknown-event refusal
	// propagates.
	inputs = testInputs([]string{testOtherManifest}, testOpA)
	if _, _, err := store.Capture(chain, inputs); !errors.Is(err, sessrepo.ErrUnknownEvent) {
		t.Fatalf("Capture(unknown head) = %v, want unknown session event", err)
	}
	// Nil chain refuses closed instead of skipping the gate.
	if _, _, err := store.Capture(nil, testInputs([]string{created}, testOpA)); err == nil {
		t.Fatalf("Capture(nil chain) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Capture(nil chain)")
	}
	if _, err := store.Admit(nil, []byte(specCheckpointExample), testOpA, SessionKindDirect); err == nil {
		t.Fatalf("Admit(nil chain) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(nil chain)")
	}
}

func TestCaptureRefusesLaterLeaseHead(t *testing.T) {
	chain, created := chainFixture(t)
	// A second-epoch event under a different fencing token.
	second := appendChainEvent(t, chain, "lease.transferred", 2, testLeaseB, 1, created, map[string]any{
		"operation_id": testHostB, "from_host_id": testHostA, "to_host_id": testHostA,
		"predecessor_lease_id": testLease, "new_lease_id": testLeaseB,
	})
	store := openTestStore(t)
	// A greater epoch postdates the owning lease.
	inputs := testInputs([]string{second}, testOpA)
	if _, _, err := store.Capture(chain, inputs); err == nil {
		t.Fatalf("Capture(later-epoch head) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Capture(later-epoch head)")
	}
	// The same epoch under a different fencing token postdates
	// the owning lease too.
	inputs = testInputs([]string{second}, testOpA)
	inputs.LeaseEpoch = 2
	if _, _, err := store.Capture(chain, inputs); err == nil {
		t.Fatalf("Capture(foreign-lease head) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Capture(foreign-lease head)")
	}
}

func TestCaptureRefusesMalformedHeads(t *testing.T) {
	chain, created := chainFixture(t)
	other := testOtherManifest
	cases := []struct {
		name  string
		heads []string
		// wantOwn gates the attribution: the capture entry's own
		// order check must fire before the owner's backstop, so a
		// mutant weakening it fails here even though the owner
		// still refuses (with different text).
		wantOwn string
	}{
		{"empty", nil, ""},
		{"malformed_digest", []string{"not-a-digest"}, ""},
		{"unsorted", []string{other, created}, "sorted unique"},
		{"duplicated", []string{created, created}, "sorted unique"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := openTestStore(t)
			// The unsorted pair needs both heads chained; the
			// others refuse on grammar or order alone.
			heads := tc.heads
			if tc.name == "unsorted" {
				if created > other {
					heads = []string{created, other}
				}
			}
			_, _, err := store.Capture(chain, testInputs(heads, testOpA))
			mustInvalid(t, err, "Capture(heads)")
			if tc.wantOwn != "" && !strings.Contains(err.Error(), tc.wantOwn) {
				t.Fatalf("error = %v, want the capture order gate named (%q)", err, tc.wantOwn)
			}
		})
	}
	// Sixty-five heads exceed the 1..64 bound even when every
	// entry is well-formed.
	t.Run("too_many", func(t *testing.T) {
		store := openTestStore(t)
		heads := make([]string, 0, 65)
		for i := 0; i < 65; i++ {
			heads = append(heads, created)
		}
		_, _, err := store.Capture(chain, testInputs(heads, testOpA))
		mustInvalid(t, err, "Capture(65 heads)")
	})
}

func TestCaptureRefusesMalformedClosureMembers(t *testing.T) {
	chain, created := chainFixture(t)
	heads := []string{created}
	cases := []struct {
		name   string
		mutate func(*Inputs)
		// wantOwn pins the capture entry's own gate attribution
		// ahead of the owner's backstop (see the heads table).
		wantOwn string
	}{
		{"bad_operation", func(i *Inputs) { i.OperationID = "not-a-uuid" }, ""},
		{"bad_session", func(i *Inputs) { i.SessionID = "not-a-uuid" }, ""},
		{"bad_kind", func(i *Inputs) { i.SessionKind = "replica" }, ""},
		{"zero_epoch", func(i *Inputs) { i.LeaseEpoch = 0 }, ""},
		{"bad_lease", func(i *Inputs) { i.LeaseID = "not-a-uuid" }, ""},
		{"bad_creator", func(i *Inputs) { i.CreatorHostID = "not-a-uuid" }, ""},
		{"bad_workspace_manifest", func(i *Inputs) { i.WorkspaceManifestID = "sha256:zzz" }, ""},
		{"bad_provider_manifest", func(i *Inputs) { i.ProviderManifestID = "sha256:zzz" }, ""},
		{"bad_bundle", func(i *Inputs) { i.TaskBoardBundleID = "sha256:zzz" }, ""},
		{"bad_provider_id", func(i *Inputs) { i.Boundary.ProviderID = "Codex!" }, ""},
		{"empty_provider_version", func(i *Inputs) { i.Boundary.ProviderVersion = "" }, ""},
		{"bad_evidence", func(i *Inputs) { i.Boundary.Evidence = "provider_gossip" }, "safe_boundary"},
		{"bad_created_at", func(i *Inputs) { i.CreatedAt = "2026-08-19 04:09:30" }, ""},
		{"bad_extension_key", func(i *Inputs) { i.Extensions = map[string]string{"pid": "4242"} }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := openTestStore(t)
			inputs := testInputs(heads, testOpA)
			tc.mutate(&inputs)
			_, _, err := store.Capture(chain, inputs)
			mustInvalid(t, err, "Capture(member)")
			if tc.wantOwn != "" && !strings.Contains(err.Error(), tc.wantOwn) {
				t.Fatalf("error = %v, want the capture gate named (%q)", err, tc.wantOwn)
			}
		})
	}
	// Sixty-five extensions exceed the owner bound.
	t.Run("too_many_extensions", func(t *testing.T) {
		store := openTestStore(t)
		inputs := testInputs(heads, testOpA)
		inputs.Extensions = map[string]string{}
		for i := 0; i < 65; i++ {
			inputs.Extensions["works.relux.ax.test."+string(rune('a'+i/26))+string(rune('a'+i%26))] = "x"
		}
		_, _, err := store.Capture(chain, inputs)
		mustInvalid(t, err, "Capture(65 extensions)")
	})
}

func TestAdmitRefusesMalformedFrames(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	// Not JSON at all.
	if _, err := store.Admit(chain, []byte("{"), testOpA, SessionKindDirect); err == nil {
		t.Fatalf("Admit(truncated) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(truncated)")
	}
	// Wrong schema.
	_, raw := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	wrong := mutateRaw(t, raw, func(value map[string]any) {
		value["schema"] = "urn:ax:schema:lease"
	})
	if _, err := store.Admit(chain, wrong, testOpB, SessionKindDirect); err == nil {
		t.Fatalf("Admit(wrong schema) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(wrong schema)")
	}
	// Tampered bytes under a fresh operation: the digest claim no
	// longer matches recomputation.
	tampered := mutateRaw(t, raw, func(value map[string]any) {
		value["lease_epoch"] = float64(2)
	})
	if _, err := store.Admit(chain, tampered, testOpB, SessionKindDirect); err == nil {
		t.Fatalf("Admit(tampered) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(tampered)")
	}
	// Unknown session kind selects no variant.
	if _, err := store.Admit(chain, raw, testOpB, "replica"); err == nil {
		t.Fatalf("Admit(bad kind) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(bad kind)")
	}
	// Malformed operation identity.
	if _, err := store.Admit(chain, raw, "not-a-uuid", SessionKindDirect); err == nil {
		t.Fatalf("Admit(bad operation) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(bad operation)")
	}
}

func TestGetRefusesUnknownAndTorn(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	ref, _ := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// Malformed digest.
	if _, err := store.Get("not-a-digest"); err == nil {
		t.Fatalf("Get(malformed) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Get(malformed)")
	}
	// Well-formed but absent digest: operational, never an
	// attested record.
	if _, err := store.Get(testOtherManifest); err == nil {
		t.Fatalf("Get(absent) error = nil, want refusal")
	} else if errors.Is(err, ErrInvalidCheckpoint) {
		t.Fatalf("Get(absent) = %v, want an operational error, not the invalid-closure class", err)
	}
	// A torn blob at a real digest path refuses attestation:
	// every visible byte still passes through the owner, so torn
	// state is a refusal, never an admission.
	if err := os.WriteFile(store.blobPath(ref.CheckpointID), []byte(`{"schema":"urn:ax:schema:checkpoint"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref.CheckpointID); err == nil {
		t.Fatalf("Get(torn) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Get(torn)")
	}
}

func TestAdmitRefusesSameEpochForeignLeaseHead(t *testing.T) {
	chain, created := chainFixture(t)
	store := openTestStore(t)
	_, raw := mustCapture(t, store, chain, testInputs([]string{created}, testOpA))
	// Forge the head binding: the same epoch under a foreign lease.
	// The bytes stay shape-valid and self-consistent, so only the
	// head binding can refuse them.
	var object map[string]any
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	object["lease_id"] = testLeaseB
	forged := identify(t, object, "checkpoint_id")
	// The repository tip sits at epoch 1 under testLease; the forged
	// record claims epoch 1 under testLeaseB. Admission binds the
	// claimed head against the tip and refuses the foreign lease.
	if _, err := store.Admit(chain, forged, testOpB, SessionKindDirect); err == nil {
		t.Fatalf("Admit(forged head) error = nil, want refusal")
	} else {
		mustInvalid(t, err, "Admit(forged head)")
	}
}
