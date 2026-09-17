package hosttrust

import (
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupStoreForTest(t *testing.T) *Store {
	t.Helper()
	store := openForTest(t, t.TempDir())
	if _, err := store.Initialize(); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	return store
}

// enrollPeerForTest runs the explicit out-of-band flow: the issuer issues
// locally, exports selected public material, and the operator authorizes the
// observed tuple in the enrolling store after independent verification.
func enrollPeerForTest(t *testing.T, store *Store, issuer *Store, hostID string, at time.Time) string {
	t.Helper()
	peerCredential, err := issuer.Issue(hostID, at)
	if err != nil {
		t.Fatalf("peer Issue error = %v", err)
	}
	peerSnapshot, err := issuer.ReadSnapshot()
	if err != nil {
		t.Fatalf("peer ReadSnapshot error = %v", err)
	}
	material, err := ExportEnrollment(peerSnapshot, peerCredential)
	if err != nil {
		t.Fatalf("ExportEnrollment error = %v", err)
	}
	if len(material.LeafDER) == 0 || len(material.RootDER) == 0 {
		t.Fatal("ExportEnrollment returned empty public bytes")
	}
	// The operator verifies the tuple over the independent channel; the test
	// models a correct verification by authorizing the observed fingerprints.
	enrolled, err := store.Enroll(EnrollInput{
		HostID:     material.HostID,
		LeafDER:    material.LeafDER,
		RootDER:    material.RootDER,
		Authorized: material.Fingerprints,
		EnrolledAt: at,
	})
	if err != nil {
		t.Fatalf("Enroll error = %v", err)
	}
	if enrolled != peerCredential {
		t.Fatalf("Enroll = %s, want %s", enrolled, peerCredential)
	}
	return enrolled
}

func TestIssueSelfEnrollsActive(t *testing.T) {
	store := setupStoreForTest(t)
	credential, err := store.Issue(testHostA, testNow)
	if err != nil {
		t.Fatalf("Issue error = %v", err)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot error = %v", err)
	}
	if snapshot.Generation != 2 {
		t.Fatalf("generation = %d, want 2 after setup + issue", snapshot.Generation)
	}
	entry, found := findEntry(snapshot.Trust, credential)
	if !found || entry.State != EntryActive || entry.HostID.String() != testHostA {
		t.Fatalf("self entry = %+v found=%v, want active for %s", entry.State, found, testHostA)
	}
	hex, err := store.CredentialDir(credential[len("sha256:"):])
	if err != nil {
		t.Fatalf("CredentialDir error = %v", err)
	}
	for _, name := range []string{certificateFile, privateKeyFile, rootFile} {
		info, err := os.Lstat(filepath.Join(hex, name))
		if err != nil {
			t.Fatalf("Lstat(%s) error = %v", name, err)
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
			t.Fatalf("custody file %s mode = %04o, want 0600", name, info.Mode().Perm())
		}
	}
	if err := store.ValidateCustody(credential[len("sha256:"):], testHostA, testNow); err != nil {
		t.Fatalf("ValidateCustody error = %v", err)
	}
	if err := store.ValidateCustody(credential[len("sha256:"):], testHostB, testNow); err == nil {
		t.Fatal("ValidateCustody(wrong host) succeeded, want refusal")
	}
}

func TestEnrollRefusals(t *testing.T) {
	store := setupStoreForTest(t)
	peer := setupStoreForTest(t)
	peerCredential, err := peer.Issue(testHostB, testNow)
	if err != nil {
		t.Fatal(err)
	}
	peerSnapshot, err := peer.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := ExportEnrollment(peerSnapshot, peerCredential)
	if err != nil {
		t.Fatal(err)
	}
	good := EnrollInput{
		HostID:     material.HostID,
		LeafDER:    material.LeafDER,
		RootDER:    material.RootDER,
		Authorized: material.Fingerprints,
		EnrolledAt: testNow,
	}
	tampered := good
	tampered.LeafDER = append([]byte(nil), good.LeafDER...)
	tampered.LeafDER[len(tampered.LeafDER)-1] ^= 0x01
	mismatched := good
	mismatched.Authorized.Leaf = material.Fingerprints.Root
	tests := []struct {
		name  string
		input EnrollInput
	}{
		{"bad host id", func() EnrollInput { input := good; input.HostID = "nope"; return input }()},
		{"tampered bytes", tampered},
		{"mismatched authorization", mismatched},
		{"garbage bytes", func() EnrollInput { input := good; input.LeafDER = []byte{1, 2, 3}; return input }()},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := store.Enroll(test.input); err == nil {
				t.Fatalf("Enroll(%s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, ErrEnrollmentRefused) && !errors.Is(err, ErrCredentialProfile) {
				t.Fatalf("Enroll(%s) error = %v", test.name, err)
			}
		})
	}
	if _, err := store.Enroll(good); err != nil {
		t.Fatalf("Enroll(good) error = %v", err)
	}
	// Re-enrolling the same bytes refuses: a leaf never maps twice, and a
	// retry after a committed enrollment reports the duplicate, never absence.
	if _, err := store.Enroll(good); err == nil {
		t.Fatal("second Enroll(same bytes) succeeded, want refusal")
	}
}

func TestEnrollBoundsPerHost(t *testing.T) {
	store := setupStoreForTest(t)
	enrollOne := func() {
		t.Helper()
		// A fresh peer store per issuance: each peer issues once with fresh
		// keys, so only the per-host bound under test can refuse.
		peer := setupStoreForTest(t)
		credential, err := peer.Issue(testHostB, testNow)
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := peer.ReadSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		material, err := ExportEnrollment(snapshot, credential)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Enroll(EnrollInput{HostID: material.HostID, LeafDER: material.LeafDER, RootDER: material.RootDER, Authorized: material.Fingerprints, EnrolledAt: testNow}); err != nil {
			t.Fatal(err)
		}
	}
	enrollOne()
	enrollOne()
	// The third non-revoked credential for one host refuses: at most two
	// non-revoked credentials belong to a host, solely during rotation.
	peer := setupStoreForTest(t)
	credential, err := peer.Issue(testHostB, testNow)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := peer.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := ExportEnrollment(snapshot, credential)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Enroll(EnrollInput{HostID: material.HostID, LeafDER: material.LeafDER, RootDER: material.RootDER, Authorized: material.Fingerprints, EnrolledAt: testNow}); err == nil {
		t.Fatal("third Enroll(same host) succeeded, want refusal")
	} else if !errors.Is(err, ErrEnrollmentRefused) {
		t.Fatalf("third Enroll error = %v", err)
	}
}

func TestRotateBoundedWindow(t *testing.T) {
	store := setupStoreForTest(t)
	first, err := store.Issue(testHostA, testNow)
	if err != nil {
		t.Fatal(err)
	}
	second, retireAt, err := store.Rotate(testHostA, testNow)
	if err != nil {
		t.Fatalf("Rotate error = %v", err)
	}
	if second == first {
		t.Fatal("Rotate returned the same credential")
	}
	if !retireAt.Equal(testNow.Add(RotationBound)) {
		t.Fatalf("retire_at = %v, want exactly now+24h", retireAt)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	old, found := findEntry(snapshot.Trust, first)
	if !found || old.State != EntryRetiring {
		t.Fatalf("old entry state = %q found=%v, want retiring", old.State, found)
	}
	if old.RetireAt == nil {
		t.Fatal("retiring entry has no retire_at")
	}
	fresh, found := findEntry(snapshot.Trust, second)
	if !found || fresh.State != EntryActive {
		t.Fatalf("new entry state = %q found=%v, want active", fresh.State, found)
	}
	// No second rotation starts until the retiring entry is revoked.
	if _, _, err := store.Rotate(testHostA, testNow); err == nil {
		t.Fatal("second Rotate over retiring entry succeeded, want refusal")
	} else if !errors.Is(err, ErrRotationRefused) {
		t.Fatalf("second Rotate error = %v", err)
	}
	// Rotation uses fresh keys: the new triple is unseen.
	if fresh.SPKIID.String() == old.SPKIID.String() || fresh.RootID.String() == old.RootID.String() {
		t.Fatal("rotation reused key material")
	}
}

func TestRotateCapsAtLeafExpiry(t *testing.T) {
	store := setupStoreForTest(t)
	// Issue so the old leaf expires in one hour: issuance backdated so that
	// notBefore+90d lands one hour after rotation time.
	rotationAt := testNow
	issuedAt := rotationAt.Add(-LeafLifetime).Add(time.Hour).Add(IssuanceSkew)
	credential, err := store.Issue(testHostA, issuedAt)
	if err != nil {
		t.Fatal(err)
	}
	_, retireAt, err := store.Rotate(testHostA, rotationAt)
	if err != nil {
		t.Fatalf("Rotate error = %v", err)
	}
	snapshot, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	old, _ := findEntry(snapshot.Trust, credential)
	leaf, err := x509.ParseCertificate(old.LeafDER)
	if err != nil {
		t.Fatal(err)
	}
	if !retireAt.Equal(leaf.NotAfter) {
		t.Fatalf("retire_at = %v, want old leaf expiry %v", retireAt, leaf.NotAfter)
	}
	if retireAt.After(rotationAt.Add(RotationBound)) {
		t.Fatal("retire_at exceeds the 24-hour bound")
	}
}

func TestRevokeTombstone(t *testing.T) {
	store := setupStoreForTest(t)
	credential, err := store.Issue(testHostA, testNow)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(credential); err != nil {
		t.Fatalf("Revoke error = %v", err)
	}
	after, err := store.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if after.Generation != before.Generation+1 {
		t.Fatalf("revocation bumped gen %d to %d, want +1", before.Generation, after.Generation)
	}
	entry, found := findEntry(after.Trust, credential)
	if !found {
		t.Fatal("revoked entry lost its public bytes")
	}
	if entry.State != EntryRevoked || entry.RetireAt != nil || len(entry.LeafDER) == 0 || len(entry.RootDER) == 0 {
		t.Fatalf("revoked entry = state %q retire %v leaf %d root %d, want tombstone", entry.State, entry.RetireAt, len(entry.LeafDER), len(entry.RootDER))
	}
	// Revoking again refuses: a failed commit reports failure, never absence.
	if err := store.Revoke(credential); err == nil {
		t.Fatal("second Revoke succeeded, want refusal")
	} else if !errors.Is(err, ErrRevocationRefused) {
		t.Fatalf("second Revoke error = %v", err)
	}
	if err := store.Revoke("sha256:0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("Revoke(unknown) succeeded, want refusal")
	}
	// A revoked tombstone never redistributes.
	if _, err := ExportEnrollment(after, credential); err == nil {
		t.Fatal("ExportEnrollment(revoked) succeeded, want refusal")
	} else if !errors.Is(err, ErrEnrollmentRefused) {
		t.Fatalf("ExportEnrollment(revoked) error = %v", err)
	}
}

func TestRevokedLeafNeverReenrolls(t *testing.T) {
	store := setupStoreForTest(t)
	peer := setupStoreForTest(t)
	credential, err := peer.Issue(testHostB, testNow)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := peer.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := ExportEnrollment(snapshot, credential)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Enroll(EnrollInput{HostID: material.HostID, LeafDER: material.LeafDER, RootDER: material.RootDER, Authorized: material.Fingerprints, EnrolledAt: testNow}); err != nil {
		t.Fatal(err)
	}
	if err := store.Revoke(credential); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Enroll(EnrollInput{HostID: material.HostID, LeafDER: material.LeafDER, RootDER: material.RootDER, Authorized: material.Fingerprints, EnrolledAt: testNow.Add(time.Hour)}); err == nil {
		t.Fatal("Enroll(revoked leaf) succeeded, want refusal")
	}
}

func TestSpecRefusalIdentities(t *testing.T) {
	// hosttrust cannot import peeridentity (import cycle), so the shared
	// refusal strings are pinned here against the specification text.
	if ErrNotAllowlisted.Error() != "peer_not_allowlisted" {
		t.Fatalf("ErrNotAllowlisted = %q", ErrNotAllowlisted)
	}
	if ErrHostIdentityMismatch.Error() != "host_identity_mismatch" {
		t.Fatalf("ErrHostIdentityMismatch = %q", ErrHostIdentityMismatch)
	}
}
