package hosttrust

import (
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	testNow   = time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	testHostA = "0198f4c8-7d40-7e55-8e6f-1234567890aa"
	testHostB = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	testHostC = "0198f4c8-7d40-7e55-8e6f-1234567890ac"
)

func testTimestamp(value time.Time) scalar.Timestamp {
	stamp, err := scalar.ParseTimestamp(value.UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		panic(err)
	}
	return stamp
}

// commitDocumentForTest stages one document through the token-gated
// production commit: it acquires a live exclusive hold, commits, and
// releases. Fixture staging holds no joint intent, so the brief hold is
// only serialization, never a protocol step.
func commitDocumentForTest(t *testing.T, store *Store, path string, document []byte) {
	t.Helper()
	hold, unlock, err := store.lock.exclusiveHold()
	if err != nil {
		t.Fatalf("exclusiveHold error = %v", err)
	}
	defer unlock()
	if err := commitDocument(hold, store.fs, store.root, path, document); err != nil {
		t.Fatalf("commitDocument error = %v", err)
	}
}

func issueForTest(t *testing.T, hostID string, now time.Time) IssuedCredential {
	t.Helper()
	issued, err := IssueCredential(hostID, now, nil)
	if err != nil {
		t.Fatalf("IssueCredential(%s) error = %v", hostID, err)
	}
	return issued
}

func entryForTest(t *testing.T, issued IssuedCredential, state string, enrolledAt time.Time, retireAt *time.Time) CredentialEntry {
	t.Helper()
	var retireStamp *scalar.Timestamp
	if retireAt != nil {
		stamp := testTimestamp(*retireAt)
		retireStamp = &stamp
	}
	return CredentialEntry{
		HostID:       issued.HostID,
		CredentialID: scalar.SHA256Digest(issued.LeafDER),
		SPKIID:       scalar.SHA256Digest(mustSPKI(issued.LeafDER)),
		RootID:       scalar.SHA256Digest(issued.RootDER),
		LeafDER:      append([]byte(nil), issued.LeafDER...),
		RootDER:      append([]byte(nil), issued.RootDER...),
		State:        state,
		EnrolledAt:   testTimestamp(enrolledAt),
		RetireAt:     retireStamp,
	}
}

func encodeForTest(t *testing.T, store TrustStore) []byte {
	t.Helper()
	document, err := EncodeTrust(store)
	if err != nil {
		t.Fatalf("EncodeTrust error = %v", err)
	}
	return document
}

func TestDecodeTrustRoundTrip(t *testing.T) {
	issuedA := issueForTest(t, testHostA, testNow)
	issuedB := issueForTest(t, testHostB, testNow)
	retire := testNow.Add(time.Hour)
	store := TrustStore{Generation: 3, Entries: []CredentialEntry{
		entryForTest(t, issuedB, EntryActive, testNow, nil),
		entryForTest(t, issuedA, EntryRetiring, testNow.Add(-time.Hour), &retire),
	}}
	document := encodeForTest(t, store)
	decoded, err := DecodeTrust(document)
	if err != nil {
		t.Fatalf("DecodeTrust error = %v", err)
	}
	if decoded.Generation != 3 || len(decoded.Entries) != 2 {
		t.Fatalf("DecodeTrust = gen %d entries %d, want 3/2", decoded.Generation, len(decoded.Entries))
	}
	if decoded.Entries[0].CredentialID.String() > decoded.Entries[1].CredentialID.String() {
		t.Fatal("DecodeTrust entries are not sorted by credential_id")
	}
	again, err := EncodeTrust(decoded)
	if err != nil {
		t.Fatalf("EncodeTrust(decoded) error = %v", err)
	}
	redocoded, err := DecodeTrust(again)
	if err != nil {
		t.Fatalf("DecodeTrust(re-encoded) error = %v", err)
	}
	if redocoded.Generation != decoded.Generation || len(redocoded.Entries) != len(decoded.Entries) {
		t.Fatal("trust document is not stable across encode/decode")
	}
}
