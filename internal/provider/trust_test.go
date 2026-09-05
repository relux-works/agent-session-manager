package provider

import (
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// errorDetail extracts the human-text detail from a provider refusal,
// failing the test when err is not a provider Error.
func errorDetail(test testHelper, err error) string {
	test.Helper()
	var refusal Error
	if !errors.As(err, &refusal) {
		test.Fatalf("error %v is not a provider Error", err)
	}
	return refusal.Detail()
}

// errDiskGone scripts a target that exists for stat but fails on read.
var errDiskGone = errors.New("fake: input/output error")

const trustInstant = "2026-08-19T04:03:00.000Z"

// discoverOneFoo is the shared trust fixture: one external candidate foo.
func discoverOneFoo(t *testing.T, fake *fakeSystem) Candidate {
	t.Helper()
	got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, candidate := range got {
		if candidate.ID() == "foo" {
			return candidate
		}
	}
	t.Fatal("candidate foo missing")
	return Candidate{}
}

func trustOneFoo(t *testing.T, candidate Candidate) TrustRecord {
	t.Helper()
	record, err := Trust(candidate, mustTimestamp(t, trustInstant))
	if err != nil {
		t.Fatalf("Trust: %v", err)
	}
	return record
}

// TestTrustRecordsCandidateFacts proves Trust copies every Section 7.1
// trust-time fact from the accepted candidate, and that Trust is pure:
// identical inputs return identical receipts.
func TestTrustRecordsCandidateFacts(t *testing.T) {
	fake := newFakeSystem()
	fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
	first := trustOneFoo(t, discoverOneFoo(t, fake))
	second := trustOneFoo(t, discoverOneFoo(t, fake))

	if first.ProviderID() != "foo" {
		t.Fatalf("ProviderID = %q, want foo", first.ProviderID())
	}
	if first.SourcePath() != "/plugins/ax-provider-foo" {
		t.Fatalf("SourcePath = %q, want undisguised discovery path", first.SourcePath())
	}
	if first.CanonicalPath() != "/opt/real/foo" {
		t.Fatalf("CanonicalPath = %q, want /opt/real/foo", first.CanonicalPath())
	}
	if want := scalar.SHA256Digest([]byte("bytes")); first.Digest() != want {
		t.Fatalf("Digest = %v, want %v", first.Digest(), want)
	}
	if first.Owner() != "uid:1000" {
		t.Fatalf("Owner = %q, want uid:1000", first.Owner())
	}
	if first.TrustedAt().String() != trustInstant {
		t.Fatalf("TrustedAt = %q, want %q", first.TrustedAt(), trustInstant)
	}
	if first != second {
		t.Fatalf("Trust is not pure: %+v != %+v", first, second)
	}
}

// TestTrustRefusesBuiltins proves a builtin adapter cannot be trusted: it
// has no executable bytes to record.
func TestTrustRefusesBuiltins(t *testing.T) {
	got, err := Discover(fakeConfig(), fakeOwner(), newFakeSystem())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if _, err := Trust(got[0], mustTimestamp(t, trustInstant)); err == nil {
		t.Fatalf("Trust accepted builtin %q, want refusal", got[0].ID())
	} else if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
}

// TestVerifyAcceptsUnchangedTree proves Verify succeeds when freshly read
// facts equal the receipt.
func TestVerifyAcceptsUnchangedTree(t *testing.T) {
	fake := newFakeSystem()
	fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
	record := trustOneFoo(t, discoverOneFoo(t, fake))
	if err := Verify(record, fakeOwner(), fake); err != nil {
		t.Fatalf("Verify of an unchanged tree: %v", err)
	}
}

// TestVerifyDetectsSubstitution narrows the Section 7.1 substitution rule
// one dimension at a time: each changed trust fact independently fails
// with integrity_failure while every other fact stays equal.
func TestVerifyDetectsSubstitution(t *testing.T) {
	setup := func() (*fakeSystem, TrustRecord) {
		fake := newFakeSystem()
		fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
		return fake, trustOneFoo(t, discoverOneFoo(t, fake))
	}
	t.Run("retargeted symlink", func(t *testing.T) {
		fake, record := setup()
		fake.canon["/plugins/ax-provider-foo"] = "/opt/real/evil"
		fake.files["/opt/real/evil"] = fakeFile{content: []byte("bytes"), uid: fakeUID, regular: true}
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted a retargeted symlink, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "target changed") {
			t.Fatalf("detail does not name the retarget refusal: %q", detail)
		}
	})
	t.Run("changed bytes", func(t *testing.T) {
		fake, record := setup()
		fake.files["/opt/real/foo"] = fakeFile{content: []byte("patched"), uid: fakeUID, regular: true}
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted changed bytes, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "digest changed") {
			t.Fatalf("detail does not name the digest refusal: %q", detail)
		}
	})
	t.Run("changed owner", func(t *testing.T) {
		fake, record := setup()
		fake.files["/opt/real/foo"] = fakeFile{content: []byte("bytes"), uid: foreignUID, regular: true}
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted a changed owner, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "owner is now") {
			t.Fatalf("detail does not name the owner refusal: %q", detail)
		}
	})
	t.Run("owner changed to an approved administrator", func(t *testing.T) {
		fake := newFakeSystem()
		fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
		policy := OwnerPolicy{OperatorUID: fakeUID, AdministratorUIDs: []uint32{adminUID}}
		got, err := Discover(fakeConfig("/plugins"), policy, fake)
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
		// The new owner is still approved, so only the
		// identity != record.owner half of the Verify owner check can
		// fire: the bytes, shape, and approval facts are unchanged.
		fake.files["/opt/real/foo"] = fakeFile{content: []byte("bytes"), uid: adminUID, regular: true}
		if err := Verify(record, policy, fake); err == nil {
			t.Fatal("Verify accepted an owner change to an approved administrator, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "owner is now uid:7") {
			t.Fatalf("detail does not name the owner-identity refusal: %q", detail)
		}
	})
	t.Run("replaced with directory", func(t *testing.T) {
		fake, record := setup()
		// Shape-only replacement: the bytes are unchanged so the digest
		// check alone would accept; only the regular-file check refuses.
		fake.files["/opt/real/foo"] = fakeFile{content: []byte("bytes"), uid: fakeUID, regular: false}
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted a non-regular replacement, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "no longer a regular file") {
			t.Fatalf("detail does not name the shape refusal: %q", detail)
		}
	})
	t.Run("owner approval revoked", func(t *testing.T) {
		fake, record := setup()
		// Same recorded owner, but the policy no longer approves it:
		// renewed trust is still required.
		strict := OwnerPolicy{OperatorUID: 9999}
		if err := Verify(record, strict, fake); err == nil {
			t.Fatal("Verify accepted a no-longer-approved owner, want integrity_failure")
		}
	})
}

// TestVerifyTreatsReadFailureAsIntegrityFailure proves an unreadable file
// cannot prove it is unchanged: every seam failure at verify time fails
// with integrity_failure rather than success or absence.
func TestVerifyTreatsReadFailureAsIntegrityFailure(t *testing.T) {
	setup := func() (*fakeSystem, TrustRecord) {
		fake := newFakeSystem()
		fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
		return fake, trustOneFoo(t, discoverOneFoo(t, fake))
	}
	t.Run("unresolvable source", func(t *testing.T) {
		fake, record := setup()
		delete(fake.canon, "/plugins/ax-provider-foo")
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted an unresolvable receipt, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "re-resolved") {
			t.Fatalf("detail does not name the re-resolve refusal: %q", detail)
		}
	})
	t.Run("uninspectable target", func(t *testing.T) {
		fake, record := setup()
		delete(fake.files, "/opt/real/foo")
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted an uninspectable target, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "re-inspected") {
			t.Fatalf("detail does not name the re-inspect refusal: %q", detail)
		}
	})
	t.Run("unreadable target", func(t *testing.T) {
		fake := newFakeSystem()
		fake.entries["/plugins"] = []string{"ax-provider-foo"}
		fake.canon["/plugins/ax-provider-foo"] = "/opt/real/foo"
		fake.files["/opt/real/foo"] = fakeFile{content: []byte("bytes"), uid: fakeUID, regular: true}
		record := trustOneFoo(t, discoverOneFoo(t, fake))
		fake.contentErr["/opt/real/foo"] = errDiskGone
		if err := Verify(record, fakeOwner(), fake); err == nil {
			t.Fatal("Verify accepted an unreadable target, want integrity_failure")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "re-read") {
			t.Fatalf("detail does not name the re-read refusal: %q", detail)
		}
	})
}

// TestVerifyDetectsLateByteDigestChange narrows the digest guard across
// the digest: the receipt differs from the bytes on disk only in its final
// byte, so a comparison narrowed to a leading prefix would accept. The
// production entry point is Verify.
func TestVerifyDetectsLateByteDigestChange(t *testing.T) {
	fake := newFakeSystem()
	fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
	genuine := trustOneFoo(t, discoverOneFoo(t, fake))
	hexDigest := genuine.Digest().Hex()
	last := hexDigest[len(hexDigest)-1]
	flipped := byte('0')
	if last == '0' {
		flipped = '1'
	}
	mutated, err := scalar.ParseDigest("sha256:" + hexDigest[:len(hexDigest)-1] + string(flipped))
	if err != nil {
		t.Fatalf("ParseDigest: %v", err)
	}
	forged := TrustRecord{
		providerID: genuine.ProviderID(),
		sourcePath: genuine.SourcePath(),
		canon:      genuine.CanonicalPath(),
		digest:     mutated,
		owner:      genuine.Owner(),
		trustedAt:  genuine.TrustedAt(),
	}
	if got := forged.Digest().Hex(); got[:len(got)-1] != hexDigest[:len(hexDigest)-1] {
		t.Fatalf("mutated digest shares no 31-byte prefix; the narrowing pin is blind: %q vs %q", got, hexDigest)
	}
	if err := Verify(forged, fakeOwner(), fake); err == nil {
		t.Fatal("Verify accepted a receipt differing only in its final digest byte, want integrity_failure")
	} else if code := errorCode(t, err); code != codeIntegrityFailure {
		t.Fatalf("code = %q, want integrity_failure", code)
	} else if detail := errorDetail(t, err); !strings.Contains(detail, "digest changed") {
		t.Fatalf("detail does not name the digest refusal: %q", detail)
	}
}

// TestVerifyRefusesForgedReceipts proves Verify rechecks the receipt
// instead of trusting its claims: a self-minted receipt whose digest does
// not match the bytes on disk fails, even when every path and owner fact
// matches.
func TestVerifyRefusesForgedReceipts(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-foo", []byte("real bytes"), fakeUID)
	candidate := discoverOneFoo(t, fake)
	genuine := trustOneFoo(t, candidate)
	forged := TrustRecord{
		providerID: genuine.ProviderID(),
		sourcePath: genuine.SourcePath(),
		canon:      genuine.CanonicalPath(),
		digest:     scalar.SHA256Digest([]byte("attacker bytes")),
		owner:      genuine.Owner(),
		trustedAt:  genuine.TrustedAt(),
	}
	if err := Verify(forged, fakeOwner(), fake); err == nil {
		t.Fatal("Verify accepted a self-minted receipt, want integrity_failure")
	} else if code := errorCode(t, err); code != codeIntegrityFailure {
		t.Fatalf("code = %q, want integrity_failure", code)
	}
	if err := Verify(genuine, fakeOwner(), fake); err != nil {
		t.Fatalf("Verify refused the genuine receipt: %v", err)
	}
}

// TestVerifyRefusesEmptyOwnerReceipt proves absent evidence is never
// treated as satisfied: a receipt with the zero-value owner — the shape a
// truncated or half-deserialized persisted receipt carries — fails even
// when every path, digest, and approval fact is genuine.
func TestVerifyRefusesEmptyOwnerReceipt(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-foo", []byte("real bytes"), fakeUID)
	candidate := discoverOneFoo(t, fake)
	genuine := trustOneFoo(t, candidate)
	empty := TrustRecord{
		providerID: genuine.ProviderID(),
		sourcePath: genuine.SourcePath(),
		canon:      genuine.CanonicalPath(),
		digest:     genuine.Digest(),
		owner:      "",
		trustedAt:  genuine.TrustedAt(),
	}
	if err := Verify(empty, fakeOwner(), fake); err == nil {
		t.Fatal("Verify accepted a receipt with an empty owner, want integrity_failure")
	} else if code := errorCode(t, err); code != codeIntegrityFailure {
		t.Fatalf("code = %q, want integrity_failure", code)
	} else if detail := errorDetail(t, err); !strings.Contains(detail, "owner is now") {
		t.Fatalf("detail does not name the owner refusal: %q", detail)
	}
}

// TestBuiltinsReturnsACopy proves the registry cannot be mutated through
// the result: corrupting one return leaves later discovery order intact.
func TestBuiltinsReturnsACopy(t *testing.T) {
	first := Builtins()
	first[0] = "mutated"
	second := Builtins()
	if second[0] != "codex" {
		t.Fatalf("Builtins()[0] = %q after caller mutation, want codex: the registry leaks", second[0])
	}
	if len(second) != len(builtinOrder) {
		t.Fatalf("Builtins() = %v, want the six Section 7.1 names", second)
	}
}
