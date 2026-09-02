package localstore

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// blobOpener is the injectable read seam for an on-disk digest-path entry. The
// production value is openBlobFile; a test replaces it to drive a read that
// does not complete, which is otherwise reachable only through descriptor
// exhaustion or real media failure.
type blobOpener func(string) (io.ReadCloser, error)

func openBlobFile(path string) (io.ReadCloser, error) { return os.Open(path) }

// blobVerdict is the single Section 3.2 classification of what reading a
// digest-path entry actually proved. The distinction it owns is normative:
// SPEC.md requires quarantine for "a hash mismatch or representation
// disagreement", and a read that did not complete demonstrates neither. An
// absent, partial, or failed read is therefore never a proven mismatch.
type blobVerdict int

const (
	// blobUnproven is the zero value. It is not a proof of anything, so a
	// caller that forgets to set a verdict cannot accidentally quarantine.
	blobUnproven blobVerdict = iota
	// blobAbsent means no entry occupies the digest path.
	blobAbsent
	// blobMatches means the complete bytes were read and agree with the
	// declared size and digest.
	blobMatches
	// blobMismatch means the complete bytes were read and disagree. This is
	// the only verdict that proves the SPEC.md quarantine trigger.
	blobMismatch
	// blobUnreadable means the read did not complete. It proves nothing about
	// the bytes and is a durability failure, not an integrity finding.
	blobUnreadable
	// blobUnsafe means the entry failed the owner-only type or ownership
	// check, so its bytes were never read.
	blobUnsafe
)

// quarantineWarranted is the one predicate that authorizes moving a durable
// object out of the immutable namespace. Only a completed, disagreeing read
// qualifies. Widening it to any non-matching verdict is exactly the defect
// this predicate exists to prevent: a transient read failure would then
// permanently sideline a valid immutable object.
func (verdict blobVerdict) quarantineWarranted() bool { return verdict == blobMismatch }

// blobInspection is the verdict plus whatever the read established. digest and
// size are meaningful only for blobMatches and blobMismatch, because those are
// the verdicts backed by a completed read.
type blobInspection struct {
	verdict blobVerdict
	size    uint64
	digest  scalar.Digest
	err     error
}

// verifyBlobContent is the single owner of "read this digest-path entry and
// report what that proved". Both the object store's existing-entry inspection
// and the projection's authoritative source scan route through it, so neither
// path can independently decide that a failed read is a mismatch. The callers
// map the verdict onto their own consequence — the store quarantines, the
// projection refuses — but they do not re-derive the classification.
func verifyBlobContent(open blobOpener, path string, expected scalar.Digest, expectedSize uint64) blobInspection {
	if open == nil {
		return blobInspection{verdict: blobUnreadable, err: errors.New("no blob reader is configured")}
	}
	file, err := open(path)
	if err != nil {
		return blobInspection{verdict: blobUnreadable, err: fmt.Errorf("open blob: %w", err)}
	}
	hasher := sha256.New()
	size, readErr := io.Copy(hasher, file)
	closeErr := file.Close()
	if readErr != nil {
		return blobInspection{verdict: blobUnreadable, err: fmt.Errorf("read blob: %w", readErr)}
	}
	if closeErr != nil {
		return blobInspection{verdict: blobUnreadable, err: fmt.Errorf("close blob: %w", closeErr)}
	}
	if size < 0 {
		return blobInspection{verdict: blobUnreadable, err: fmt.Errorf("read blob: negative copied length %d", size)}
	}
	actual, err := scalar.ParseDigest("sha256:" + hex.EncodeToString(hasher.Sum(nil)))
	if err != nil {
		return blobInspection{verdict: blobUnreadable, err: fmt.Errorf("calculate blob digest: %w", err)}
	}
	inspection := blobInspection{size: uint64(size), digest: actual}
	if inspection.size != expectedSize || actual != expected {
		inspection.verdict = blobMismatch
		inspection.err = fmt.Errorf("digest path declares %s at %d bytes but contains %s at %d bytes",
			expected, expectedSize, actual, inspection.size)
		return inspection
	}
	inspection.verdict = blobMatches
	return inspection
}
