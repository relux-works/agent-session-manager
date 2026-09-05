package provhost

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// TestMutationOperationsAreDerivedFromSpec proves the keyed set is
// exactly the Section 7.5 operations whose request bodies carry
// operation_id: each table row is re-read from the pinned document
// and checked for the mutation key, so a registry row gaining or
// losing operation_id reddens here rather than passing silently.
func TestMutationOperationsAreDerivedFromSpec(t *testing.T) {
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	var derived []string
	for line := 3072; line <= 3086; line++ {
		row, ok := document.TableRowAt(line)
		if !ok {
			continue
		}
		text, ok := document.Line(line)
		if !ok {
			t.Fatalf("SPEC.md line %d is missing", line)
		}
		section, ok := document.SectionID(line)
		if !ok || section != "7.5" {
			t.Fatalf("SPEC.md line %d is in section %q, want %q", line, section, "7.5")
		}
		if strings.Contains(text, "operation_id") {
			derived = append(derived, row.Identifier)
		}
	}
	if len(derived) == 0 {
		t.Fatal("derived no keyed operations from the Section 7.5 table; the check is blind")
	}
	var got []string
	for _, operation := range MutationOperations() {
		got = append(got, string(operation))
	}
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", derived) {
		t.Fatalf("MutationOperations() = %v, want the Section 7.5 keyed rows %v", got, derived)
	}
	t.Logf("idempotency key coverage: %d/%d Section 7.5 operations carry the key", len(got), len(Operations()))
}

// TestMutationOperationsReturnsACopy proves the copy guarantee the
// accessor documents: mutating a returned slice and re-reading must
// leave the registry unchanged, so the set cannot be mutated
// through it.
func TestMutationOperationsReturnsACopy(t *testing.T) {
	first := MutationOperations()
	if len(first) == 0 {
		t.Fatal("MutationOperations() is empty; the check is blind")
	}
	want := first[0]
	first[0] = OpDoctor
	if got := MutationOperations()[0]; got != want {
		t.Fatalf("MutationOperations()[0] = %q after aliasing write, want %q", got, want)
	}
}

// TestIdempotencyKeyForIssuesKeys proves the sole key issues for
// every keyed operation and renders in the Section 7.5
// (operation, operation_id) form with its seats exposed.
func TestIdempotencyKeyForIssuesKeys(t *testing.T) {
	for _, operation := range MutationOperations() {
		key, err := IdempotencyKeyFor(operation, testRequestID)
		if err != nil {
			t.Fatalf("IdempotencyKeyFor(%q): %v", operation, err)
		}
		if key.Operation() != operation {
			t.Fatalf("key.Operation() = %q, want %q", key.Operation(), operation)
		}
		if key.OperationID() != testRequestID {
			t.Fatalf("key.OperationID() = %q, want %q", key.OperationID(), testRequestID)
		}
		want := "(" + string(operation) + ", " + testRequestID + ")"
		if key.String() != want {
			t.Fatalf("key.String() = %q, want %q", key.String(), want)
		}
	}
}

// TestIdempotencyKeyForRefusesUnkeyed sweeps the complement derived
// from the registry: every operation outside the keyed set is
// refused, plus the unknown operation and the malformed operation
// ID. Reusing one operation_id across distinct tags issues distinct
// keys — no aliasing — which the sweep asserts by issuing the same
// ID under two tags and comparing.
func TestIdempotencyKeyForRefusesUnkeyed(t *testing.T) {
	keyed := map[string]bool{}
	for _, operation := range MutationOperations() {
		keyed[string(operation)] = true
	}
	unkeyed := 0
	for _, name := range Operations() {
		if keyed[name] {
			continue
		}
		unkeyed++
		_, err := IdempotencyKeyFor(Operation(name), testRequestID)
		requireLocalRefusal(t, err, "invalid_config", "without operation_id")
	}
	if unkeyed != len(Operations())-len(MutationOperations()) {
		t.Fatalf("swept %d unkeyed operations, want %d", unkeyed, len(Operations())-len(MutationOperations()))
	}
	_, err := IdempotencyKeyFor(Operation("reboot"), testRequestID)
	requireLocalRefusal(t, err, "invalid_config", "unknown operation")
	for _, badID := range []string{"", "not-a-uuid", testDeadline} {
		_, err := IdempotencyKeyFor(OpMaterialize, badID)
		requireLocalRefusal(t, err, "invalid_config", "not a UUIDv7")
	}
	first, err := IdempotencyKeyFor(OpMaterialize, testRequestID)
	if err != nil {
		t.Fatalf("IdempotencyKeyFor(materialize): %v", err)
	}
	second, err := IdempotencyKeyFor(OpMaterializeCommit, testRequestID)
	if err != nil {
		t.Fatalf("IdempotencyKeyFor(materialize-commit): %v", err)
	}
	if first.String() == second.String() {
		t.Fatalf("keys alias across tags: %q", first.String())
	}
	t.Logf("idempotency refusal coverage: %d unkeyed operations refused", unkeyed)
}
