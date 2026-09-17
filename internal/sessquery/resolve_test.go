package sessquery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// idC and idD are further session identities for cross-source
// fixtures; hostC is a third peer host.
const idC = "0198f4c8-3e70-7a11-8a2b-1234567890ad"
const idD = "0198f4c8-3e70-7a11-8a2b-1234567890ae"
const hostC = "0198f4c8-4a10-7b22-8b3c-1234567890ad"

// leaseB is a second fencing token for successor-lease fixtures.
const leaseB = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"

// qualifiedReader builds a local session and a peer session with an
// aliased, allowlisted source, returning the reader, both handles, and
// the peer data root for failure injection. Both sessions carry their
// authoritative initial lease so status entries stay representable.
func qualifiedReader(t *testing.T) (*Reader, *sessrepo.Repository, *sessrepo.Repository, string) {
	t.Helper()
	local, _ := repository(t)
	peer, peerRoot := repository(t)
	localRef := create(t, local, idA, "alpha")
	appendEvent(t, local, idA, localRef.RecordID, "session.created", 1, map[string]any{"session_record_id": localRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	peerRef := create(t, peer, idB, "beta")
	appendEvent(t, peer, idB, peerRef.RecordID, "session.created", 1, map[string]any{"session_record_id": peerRef.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	reader := &Reader{
		Local:              local,
		LocalHostID:        hostB,
		AllowlistedPeerIDs: []string{hostA},
		Peers:              []PeerIndex{{hostA, peer}},
		Aliases:            map[string]string{hostA: "workstation"},
	}
	return reader, local, peer, peerRoot
}

func TestResolveQualifiedSourcesSelectOneIndex(t *testing.T) {
	reader, _, _, _ := qualifiedReader(t)
	for _, row := range []struct {
		selector string
		id       string
		peer     string
	}{
		{"alpha@local", idA, ""},
		{"beta@peer:workstation", idB, hostA},
		{"beta@id:" + hostA, idB, hostA},
		{"alpha@id:" + hostB, idA, ""},
		{idA + "@local", idA, ""},
		{idB + "@peer:workstation", idB, hostA},
		{"id:" + idB + "@peer:workstation", idB, hostA},
		{"id:" + idA + "@local", idA, ""},
	} {
		got, err := reader.Resolve(row.selector)
		if err != nil || got.SessionID != row.id || got.PeerHostID != row.peer {
			t.Fatalf("Resolve(%q) = %+v, %v", row.selector, got, err)
		}
		status, err := reader.Status(row.selector)
		if err != nil || status.Projection.SessionID != row.id {
			t.Fatalf("Status(%q) = %+v, %v", row.selector, status, err)
		}
	}
}

func TestResolveQualifiedSourceNeverFallsBack(t *testing.T) {
	reader, _, _, _ := qualifiedReader(t)
	// Each name exists in exactly one source; selecting the other
	// source misses, even though the union holds the name.
	for _, selector := range []string{"beta@local", "alpha@peer:workstation", "id:" + idA + "@peer:workstation", "id:" + idB + "@local"} {
		if _, err := reader.Resolve(selector); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Resolve(%q) = %v, want not_found without fallback", selector, err)
		}
		if _, err := reader.Status(selector); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Status(%q) = %v, want not_found without fallback", selector, err)
		}
	}
}

func TestResolveQualifiedNameBeforeUUIDInSource(t *testing.T) {
	reader, _, peer, _ := qualifiedReader(t)
	// The peer learns a session whose NAME is shaped like a UUID. A
	// qualified NAME tries the exact name first; only "id:" bypasses it.
	create(t, peer, idC, idA)
	got, err := reader.Resolve(idA + "@peer:workstation")
	if err != nil || got.SessionID != idC {
		t.Fatalf("qualified name-first: %+v %v", got, err)
	}
	bypass, err := reader.Resolve("id:" + idA + "@peer:workstation")
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("id: bypass selected a name: %+v %v", bypass, err)
	}
	create(t, peer, idA, "alpha-peer")
	got, err = reader.Resolve("id:" + idA + "@peer:workstation")
	if err != nil || got.SessionID != idA {
		t.Fatalf("id: identity: %+v %v", got, err)
	}
}

func TestResolveQualifiedAmbiguityAndExclusions(t *testing.T) {
	reader, local, peer, _ := qualifiedReader(t)
	create(t, peer, idC, "BETA")
	for _, selector := range []string{"beta@peer:workstation", "BETA@peer:workstation"} {
		if _, err := reader.Resolve(selector); !errors.Is(err, ErrAmbiguous) {
			t.Fatalf("Resolve(%q) = %v, want name_ambiguous", selector, err)
		}
	}
	// Tombstoned and parked entries stay excluded under qualification:
	// tombstones need authoritative tombstone evidence, and a failed
	// record read is never that evidence.
	ref := create(t, local, idC, "doomed")
	event := appendEvent(t, local, idC, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	event = appendEvent(t, local, idC, event, "session.failed", 2, map[string]any{"error_code": "E_LAUNCH", "retryable": true, "operation_id": nil})
	appendEvent(t, local, idC, event, "session.tombstoned", 3, map[string]any{"tombstone_id": zeroDigest})
	for _, selector := range []string{"doomed@local", idC + "@local", "id:" + idC + "@local"} {
		if _, err := reader.Resolve(selector); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Resolve(%q) = %v, want tombstone exclusion", selector, err)
		}
	}
	raw := record(t, idD, "ParkedPeer")
	peer.AfterCreateStep = func(step sessrepo.CreateStep) error {
		if step == sessrepo.CreateStepRecord {
			return errors.New("crash")
		}
		return nil
	}
	if _, err := peer.CreateSession(raw); err == nil {
		t.Fatal("fault did not fire")
	}
	peer.AfterCreateStep = nil
	for _, selector := range []string{"ParkedPeer@peer:workstation", idD + "@peer:workstation", "id:" + idD + "@peer:workstation"} {
		if _, err := reader.Resolve(selector); !errors.Is(err, ErrNotFound) {
			t.Fatalf("Resolve(%q) = %v, want parked exclusion", selector, err)
		}
	}
}

func TestResolveExplicitSourceRefusals(t *testing.T) {
	reader, _, _, _ := qualifiedReader(t)
	// Unknown aliases and hosts are not found; known but disallowed
	// peers are refused as not allowlisted. The unknown-source probes
	// use locally present names so a fallback to the local index would
	// succeed and fail the test.
	if _, err := reader.Resolve("alpha@peer:elsewhere"); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("unknown alias: %v", err)
	}
	if _, err := reader.Resolve("alpha@id:" + idC); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("unknown host: %v", err)
	}
	reader.AllowlistedPeerIDs = nil
	// "alpha" resolves locally, so refusing it here proves revocation
	// never degrades into a local fallback either.
	for _, selector := range []string{"beta@peer:workstation", "beta@id:" + hostA, "id:" + idB + "@peer:workstation", "alpha@peer:workstation"} {
		if _, err := reader.Resolve(selector); !errors.Is(err, ErrPeerNotAllowlisted) {
			t.Fatalf("Resolve(%q) = %v, want peer_not_allowlisted", selector, err)
		}
	}
	// The bare union still resolves the sessions it may reach; only the
	// explicit peer route is revoked.
	if _, err := reader.Resolve("beta"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked bare union: %v", err)
	}
	// Local needs no allowlist entry and survives revocation.
	if _, err := reader.Resolve("alpha@local"); err != nil {
		t.Fatalf("local after revocation: %v", err)
	}
}

func TestResolveExplicitSourceReadFailures(t *testing.T) {
	reader, _, peer, peerRoot := qualifiedReader(t)
	if err := os.Rename(filepath.Join(peerRoot, "sessions"), filepath.Join(peerRoot, "saved")); err != nil {
		t.Fatal(err)
	}
	// The peer failure carries the read-failure class while preserving
	// the underlying repository cause, and never falls back to the
	// local index even though "alpha" resolves there.
	_, err := reader.Resolve("beta@peer:workstation")
	if !errors.Is(err, ErrSourceReadFailed) || !errors.Is(err, sessrepo.ErrRepositoryPath) {
		t.Fatalf("explicit peer failure: %v", err)
	}
	if _, err := reader.Resolve("alpha@peer:workstation"); !errors.Is(err, ErrSourceReadFailed) {
		t.Fatalf("explicit miss behind failed read fell back: %v", err)
	}
	reader.Peers = []PeerIndex{{hostA, nil}}
	if _, err := reader.Resolve("beta@peer:workstation"); !errors.Is(err, ErrSourceReadFailed) || !errors.Is(err, ErrUnavailable) {
		t.Fatalf("nil learned index: %v", err)
	}
	// An allowlisted peer with no advertised index at all cannot
	// complete the required read either.
	reader.Peers = nil
	if _, err := reader.Resolve("beta@peer:workstation"); !errors.Is(err, ErrSourceNotFound) {
		t.Fatalf("unmapped alias: %v", err)
	}
	reader.Peers = []PeerIndex{{hostA, peer}}
	reader.Aliases = nil
	if _, err := reader.Resolve("beta@id:" + hostA); !errors.Is(err, ErrSourceReadFailed) {
		t.Fatalf("handle-less explicit host: %v", err)
	}
}

func TestResolveBareIdentityUnion(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	create(t, local, idA, "alpha")
	// A UUID-shaped local name wins over the UUID identity on the bare
	// path, while "id:" bypasses names for the durable identity.
	create(t, local, idC, idB)
	create(t, peer, idB, "remote")
	reader := &Reader{Local: local, AllowlistedPeerIDs: []string{hostA}, Peers: []PeerIndex{{hostA, peer}}}
	byName, err := reader.Resolve(idB)
	if err != nil || byName.SessionID != idC || byName.PeerHostID != "" {
		t.Fatalf("bare name-first: %+v %v", byName, err)
	}
	byID, err := reader.Resolve("id:" + idB)
	if err != nil || byID.SessionID != idB || byID.PeerHostID != hostA {
		t.Fatalf("id: bypass: %+v %v", byID, err)
	}
	// Replicated identical copies are one identity. While the local
	// host identity is unknown the local copy wins; once configured,
	// the bytewise smallest holder wins deterministically.
	create(t, peer, idA, "alpha")
	identical, err := reader.Resolve("id:" + idA)
	if err != nil || identical.SessionID != idA || identical.PeerHostID != "" {
		t.Fatalf("replicated unknown-local: %+v %v", identical, err)
	}
	reader.LocalHostID = hostB
	tiebreak, err := reader.Resolve("id:" + idA)
	if err != nil || tiebreak.SessionID != idA || tiebreak.PeerHostID != hostA {
		t.Fatalf("replicated tie break: %+v %v", tiebreak, err)
	}
	// Disagreeing copies of one UUID are inconsistent authority, not an
	// ambiguity and not a silent first match.
	other, _ := repository(t)
	create(t, other, idA, "renamed")
	reader.Peers = []PeerIndex{{hostA, peer}, {hostC, other}}
	reader.AllowlistedPeerIDs = []string{hostA, hostC}
	if _, err := reader.Resolve("id:" + idA); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("disagreeing copies: %v", err)
	}
	if _, err := reader.Resolve("renamed"); !errors.Is(err, sessstate.ErrIntegrity) {
		t.Fatalf("disagreeing name win: %v", err)
	}
	// A unique local name win never consults lower tiers, so an
	// unreadable peer copy behind it is not a conflict.
	reader.Peers = []PeerIndex{{hostA, other}}
	if _, err := reader.Resolve("alpha"); err != nil {
		t.Fatalf("local win with divergent peer copy: %v", err)
	}
}
