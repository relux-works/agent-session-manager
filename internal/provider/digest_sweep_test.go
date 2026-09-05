package provider

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestVerifyRefusesDigestChangeAtEveryByteIndex sweeps the Verify digest
// guard across its whole domain: for each byte index i, a receipt differing
// from the genuine digest only at byte i must fail with integrity_failure.
// Sampling only the endpoints moved the proven point from byte 0 to byte
// 31 while leaving sum[1:] green; this sweep reports 32/32 over the 32-byte
// digest domain. The production entry point is Verify.
func TestVerifyRefusesDigestChangeAtEveryByteIndex(t *testing.T) {
	fake := newFakeSystem()
	fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
	genuine := trustOneFoo(t, discoverOneFoo(t, fake))
	raw, err := hex.DecodeString(genuine.Digest().Hex())
	if err != nil {
		t.Fatalf("decode genuine digest: %v", err)
	}
	if len(raw) != sha256.Size {
		t.Fatalf("genuine digest length = %d, want %d", len(raw), sha256.Size)
	}
	passed := 0
	for i := 0; i < sha256.Size; i++ {
		mutated := append([]byte(nil), raw...)
		mutated[i] ^= 0x01
		if bytes.Equal(mutated, raw) {
			t.Fatalf("byte %d mutation is not a change", i)
		}
		// The other 31 bytes must be shared, so a comparison narrowed to
		// any range excluding byte i would accept and only the full-width
		// guard refuses.
		shared := 0
		for j := range mutated {
			if mutated[j] == raw[j] {
				shared++
			}
		}
		if shared != sha256.Size-1 {
			t.Fatalf("byte %d fixture shares %d bytes, want %d", i, shared, sha256.Size-1)
		}
		digest, err := scalar.ParseDigest("sha256:" + hex.EncodeToString(mutated))
		if err != nil {
			t.Fatalf("ParseDigest(byte %d): %v", i, err)
		}
		forged := TrustRecord{
			providerID: genuine.ProviderID(),
			sourcePath: genuine.SourcePath(),
			canon:      genuine.CanonicalPath(),
			digest:     digest,
			owner:      genuine.Owner(),
			trustedAt:  genuine.TrustedAt(),
		}
		if err := Verify(forged, fakeOwner(), fake); err == nil {
			t.Fatalf("Verify accepted a receipt differing only at digest byte %d, want integrity_failure", i)
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("Verify(byte %d) code = %q, want integrity_failure", i, code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "digest changed") {
			t.Fatalf("Verify(byte %d) detail does not name the digest refusal: %q", i, detail)
		}
		passed++
	}
	t.Logf("digest-guard coverage: %d/%d byte positions refused", passed, sha256.Size)
	if passed != sha256.Size {
		t.Fatalf("digest-guard coverage = %d/%d, want %d/%d", passed, sha256.Size, sha256.Size, sha256.Size)
	}
}
