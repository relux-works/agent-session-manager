package provider

import (
	"strings"
	"testing"
)

// rootUID is the superuser identity. OwnerPolicy documents no superuser
// exception (provider.go: "no superuser exception is implied"), so uid 0
// is approved only when it is the operator or an administrator-approved
// identity — never by virtue of being root.
const rootUID uint32 = 0

// TestOwnerPolicyTreatsRootWithoutException pins the no-superuser-exception
// property at the policy unit: uid 0 as a candidate owner is refused under
// an operator-only policy, and uid 0 as the operator behaves exactly like
// any other operator identity. The production entry point under test is
// OwnerPolicy.Approves, which Discover and Verify both gate on.
func TestOwnerPolicyTreatsRootWithoutException(t *testing.T) {
	operatorOnly := OwnerPolicy{OperatorUID: fakeUID}
	if operatorOnly.Approves(rootUID) {
		t.Fatal("Approves(0) under an operator-only policy, want refusal: root has no implied approval")
	}
	if !operatorOnly.Approves(fakeUID) {
		t.Fatal("Approves(operator) refused under its own policy")
	}

	rootOperator := OwnerPolicy{OperatorUID: rootUID}
	if !rootOperator.Approves(rootUID) {
		t.Fatal("Approves(0) refused when root is the operator")
	}
	if rootOperator.Approves(fakeUID) {
		t.Fatal("Approves(operator) admitted a foreign UID under a root-operator policy")
	}

	rootAdmin := OwnerPolicy{OperatorUID: fakeUID, AdministratorUIDs: []uint32{rootUID}}
	if !rootAdmin.Approves(rootUID) {
		t.Fatal("Approves(0) refused when root is an administrator-approved identity")
	}
	if rootAdmin.Approves(foreignUID) {
		t.Fatal("Approves(foreign) admitted an unlisted UID alongside a root administrator")
	}
}

// TestDiscoverRefusesRootOwnedExecutablesWithoutApproval drives the
// superuser-exception mutant through the production entry point Discover:
// injecting `if uid == 0 { return true }` into OwnerPolicy.Approves must
// redden this test, because the root-owned fixture below would flip from
// refused to admitted.
func TestDiscoverRefusesRootOwnedExecutablesWithoutApproval(t *testing.T) {
	setup := func() *fakeSystem {
		fake := newFakeSystem()
		fake.addFile("/plugins", "ax-provider-foo", []byte("x"), rootUID)
		return fake
	}
	if _, err := Discover(fakeConfig("/plugins"), fakeOwner(), setup()); err == nil {
		t.Fatal("Discover admitted a root-owned executable under an operator-only policy, want invalid_config")
	} else if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	} else if detail := errorDetail(t, err); !strings.Contains(detail, "uid:0") {
		t.Fatalf("detail does not name the root owner: %q", detail)
	}

	rootOperator := OwnerPolicy{OperatorUID: rootUID}
	got, err := Discover(fakeConfig("/plugins"), rootOperator, setup())
	if err != nil {
		t.Fatalf("Discover refused a root-owned executable under a root-operator policy: %v", err)
	}
	found := false
	for _, candidate := range got {
		if candidate.ID() == "foo" {
			found = true
			owner, ok := candidate.Owner()
			if !ok || owner != "uid:0" {
				t.Fatalf("Owner = %q, %v; want uid:0, true", owner, ok)
			}
		}
	}
	if !found {
		t.Fatal("root-owned candidate missing under a root-operator policy")
	}
}

// TestVerifyAcceptsUnchangedRootOwnedTree proves Verify applies the same
// policy with no root special case: a root-owned receipt verifies under a
// root-operator policy and fails under an operator-only policy with the
// owner refusal, not a silent pass.
func TestVerifyAcceptsUnchangedRootOwnedTree(t *testing.T) {
	fake := newFakeSystem()
	fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), rootUID, true)
	rootOperator := OwnerPolicy{OperatorUID: rootUID}
	got, err := Discover(fakeConfig("/plugins"), rootOperator, fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	var candidate Candidate
	for _, found := range got {
		if found.ID() == "foo" {
			candidate = found
		}
	}
	record, err := Trust(candidate, mustTimestamp(t, trustInstant))
	if err != nil {
		t.Fatalf("Trust: %v", err)
	}
	if err := Verify(record, rootOperator, fake); err != nil {
		t.Fatalf("Verify of an unchanged root-owned tree under a root-operator policy: %v", err)
	}
	if err := Verify(record, fakeOwner(), fake); err == nil {
		t.Fatal("Verify accepted a root-owned receipt under an operator-only policy, want integrity_failure")
	} else if code := errorCode(t, err); code != codeIntegrityFailure {
		t.Fatalf("code = %q, want integrity_failure", code)
	} else if detail := errorDetail(t, err); !strings.Contains(detail, "owner is now") {
		t.Fatalf("detail does not name the owner refusal: %q", detail)
	}
}
