package termbind

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAttachStorePeers pins the stored peer census the attach
// overlap gate fails closed on: every recorded receipt for the
// instance except the requesting client, sorted by client ID.
func TestAttachStorePeers(t *testing.T) {
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := fixtureNow()
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClientB, "local_only", false, validAttachAuth(t, "local_only", false), now); err != nil {
		t.Fatal(err)
	}
	peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient)
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].ClientID != fixtureClientB || peers[0].InputAuthorized {
		t.Fatalf("peers = %+v, want exactly the other client without input", peers)
	}
	peers, err = store.Peers(fixtureSession, fixtureInstance, fixtureClientB)
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 1 || peers[0].ClientID != fixtureClient || !peers[0].InputAuthorized {
		t.Fatalf("peers = %+v, want exactly the input-authorized other client", peers)
	}
}

// TestAttachStorePeersEmpty pins the no-peer cases: an unrecorded
// instance and a lone client both report no peers.
func TestAttachStorePeersEmpty(t *testing.T) {
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient)
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("peers = %+v, want none for an unrecorded instance", peers)
	}
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), fixtureNow()); err != nil {
		t.Fatal(err)
	}
	peers, err = store.Peers(fixtureSession, fixtureInstance, fixtureClient)
	if err != nil {
		t.Fatal(err)
	}
	if len(peers) != 0 {
		t.Fatalf("peers = %+v, want none for a lone client", peers)
	}
}

// TestAttachStorePeersFailsClosed pins the census health rule: an
// unreadable peer receipt is unknown, never an empty census, and
// staging files are skipped rather than decoded.
func TestAttachStorePeersFailsClosed(t *testing.T) {
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := fixtureNow()
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClientB, "local_only", false, validAttachAuth(t, "local_only", false), now); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.root, "termbind", "attach", fixtureInstance)
	if err := os.WriteFile(filepath.Join(dir, "attach-staging.tmp"), []byte("incomplete"), 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient); err != nil || len(peers) != 1 {
		t.Fatalf("peers = %+v, err = %v; want the peer past the staging file", peers, err)
	}
	if err := os.WriteFile(filepath.Join(dir, fixtureClientB+".json"), []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient); err == nil {
		t.Fatalf("peers = %+v, want the decode error instead of a census", peers)
	}
}

// TestAttachStorePeersDirectoryFailsClosed pins the census
// namespace rule: entries outside the receipt namespace — staging
// entries (attach-*) and non-receipt names, files or directories —
// are skipped, but a directory at a receipt-shaped name is
// corruption and errors instead of reading as no peers.
func TestAttachStorePeersDirectoryFailsClosed(t *testing.T) {
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := fixtureNow()
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), now); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClientB, "local_only", false, validAttachAuth(t, "local_only", false), now); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.root, "termbind", "attach", fixtureInstance)
	if err := os.WriteFile(filepath.Join(dir, "attach-staging.tmp"), []byte("incomplete"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "attach-old.tmp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "scratch"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("operator notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient); err != nil || len(peers) != 1 {
		t.Fatalf("peers = %+v, err = %v; want the peer past the non-receipt entries", peers, err)
	}
	receipt := filepath.Join(dir, fixtureClientB+".json")
	raw, err := os.ReadFile(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(receipt); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(receipt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(receipt, "original.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient); err == nil {
		t.Fatalf("peers = %+v, want the census error instead of a census past the receipt directory", peers)
	}
}

// TestAttachStorePeersRejectsFilenameMismatch pins the census
// identity binding: a valid receipt filed under another client's
// durable key is corruption, never an empty census. The
// requesting-client exclusion must not erase evidence the keyed
// read refuses, so the binding is checked before any exclusion
// applies. A receipt-shaped name whose stem is not a client
// identity refuses the same way.
func TestAttachStorePeersRejectsFilenameMismatch(t *testing.T) {
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := fixtureNow()
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClientB, "local_only", true, validAttachAuth(t, "local_only", true), now); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.root, "termbind", "attach", fixtureInstance)
	raw, err := os.ReadFile(filepath.Join(dir, fixtureClientB+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, fixtureClientB+".json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fixtureClient+".json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClientB); err == nil {
		t.Fatalf("peers = %+v, want the binding error instead of a census past the misfiled receipt", peers)
	}
	malformed, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	malformedDir := filepath.Join(malformed.root, "termbind", "attach", fixtureInstance)
	if err := os.MkdirAll(malformedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(malformedDir, "bogus.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := malformed.Peers(fixtureSession, fixtureInstance, fixtureClientB); err == nil {
		t.Fatalf("peers = %+v, want the shape error instead of a census past the malformed receipt name", peers)
	}
}

// TestAttachStorePeersRejectsForeignReceipt pins the binding rule:
// a receipt filed under this instance that names another session
// or instance is corruption, never a peer.
func TestAttachStorePeersRejectsForeignReceipt(t *testing.T) {
	store, err := OpenAttachStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := fixtureNow()
	if _, _, err := store.Attach(fixtureSession, fixtureInstance, fixtureClient, "local_only", true, validAttachAuth(t, "local_only", true), now); err != nil {
		t.Fatal(err)
	}
	foreign, err := encodeAttachReceipt(AttachReceipt{
		SessionID:          fixtureSession,
		TerminalInstanceID: fixtureBootstrap,
		ClientID:           fixtureClientB,
		Transport:          "local_only",
	}, fixtureCreatedAt)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(store.root, "termbind", "attach", fixtureInstance)
	if err := os.WriteFile(filepath.Join(dir, fixtureClientB+".json"), foreign, 0o600); err != nil {
		t.Fatal(err)
	}
	if peers, err := store.Peers(fixtureSession, fixtureInstance, fixtureClient); err == nil {
		t.Fatalf("peers = %+v, want the binding error instead of a census", peers)
	}
}
