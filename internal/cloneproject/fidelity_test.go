package cloneproject

import (
	"bytes"
	"strings"
	"testing"
)

// Durability: Normalize holds no local state and its only durable
// write is overflow-blob installation through the landed no-replace
// discipline. A replay converges byte-identical (idempotent), and a
// mid-projection failure returns no partial bundle.

func TestNormalizeIsIdempotentIntoSharedSink(t *testing.T) {
	overflow := strings.Repeat("q", 70000)
	members := []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-idem-1", "message/user", "native", "none", "main", `{"text":"`+overflow+`"}`),
			rec("evt-idem-2", "message/user", "native", "none", "main", `{"text":"small"}`),
		)},
	}
	capture, store := captureMembers(t, members)
	sink := openTestStore(t)
	first, err := Normalize(normalizeRequest(t, capture, store, sink))
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	blobsAfterFirst := len(listStoreBlobs(t, sink))
	second, err := Normalize(normalizeRequest(t, capture, store, sink))
	if err != nil {
		t.Fatalf("Normalize() replay error = %v", err)
	}
	if !bytes.Equal(first.Session, second.Session) {
		t.Fatal("replay sealed a different session")
	}
	if len(first.Events) != len(second.Events) {
		t.Fatalf("replay event counts %d != %d", len(first.Events), len(second.Events))
	}
	for index := range first.Events {
		if !bytes.Equal(first.Events[index], second.Events[index]) {
			t.Fatalf("replay event[%d] differs", index)
		}
	}
	if blobs := len(listStoreBlobs(t, sink)); blobs != blobsAfterFirst {
		t.Fatalf("sink blobs %d after replay, want %d: replay reinstalled instead of verifying", blobs, blobsAfterFirst)
	}
	if len(first.Overflows) != 1 || len(second.Overflows) != 1 {
		t.Fatalf("overflows %d/%d, want 1/1", len(first.Overflows), len(second.Overflows))
	}
	if first.Overflows[0].DescriptorID.String() != second.Overflows[0].DescriptorID.String() {
		t.Fatal("replay overflow descriptor differs")
	}
}

func TestNormalizeFailureReturnsNoPartialBundle(t *testing.T) {
	members := []fixtureMember{
		{key: "store/a.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-part-1", "message/user", "native", "none", "main", `{"text":"one"}`),
		)},
		{key: "store/b.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-part-2", "message/user", "native", "none", "main", `{"text":"two"}`),
		)},
	}
	capture, store := captureMembers(t, members)
	request := normalizeRequest(t, capture, store, openTestStore(t))
	healthy := request.Fetch
	calls := 0
	request.Fetch = func(blobID string) ([]byte, error) {
		calls++
		if calls == 2 {
			return nil, errFixtureGone
		}
		return healthy(blobID)
	}
	result, err := Normalize(request)
	if err == nil {
		t.Fatal("Normalize() admitted a mid-projection fetch failure")
	}
	if result != nil {
		t.Fatal("Normalize() returned a partial bundle alongside the failure")
	}
	// A healthy re-run over the same captured bytes converges.
	clean, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err != nil {
		t.Fatalf("Normalize() re-run error = %v", err)
	}
	if len(clean.Events) != 2 {
		t.Fatalf("re-run events = %d, want 2", len(clean.Events))
	}
}
