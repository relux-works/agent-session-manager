package clonebundle

import (
	"encoding/json"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file owns the Section 13.14.1 immutable generations: the
// source_generation strings carried by StableSnapshotProof and the
// unstable-archive Capture Boundary, and the snapshot-identity digest
// that pins an immutable snapshot. Generations are opaque immutable
// tokens: a generation never mutates, and equality is exact string
// equality. Capture digests that must agree (stable pre/post) are
// compared by digest equality here so both boundary validators share
// one comparison.

// Generation is one validated immutable source generation token:
// string[1..512], opaque to the core.
type Generation struct {
	value string
}

// ParseGeneration validates one generation token. Empty,
// over-long, and invalid-UTF-8 tokens are refused; every other text
// content crosses opaquely, because generations are native-store
// tokens the core compares but never interprets. The UTF-8 gate
// matters because the token seals into the boundary bytes: without
// it the marshaler would rewrite invalid bytes to U+FFFD and two
// distinct generations would seal byte-identical.
func ParseGeneration(value string) (Generation, error) {
	if !validText(value) {
		return Generation{}, invalid("source generation is not valid UTF-8")
	}
	if stringLength(value) < 1 || stringLength(value) > 512 {
		return Generation{}, invalid("source generation is not a string[1..512]")
	}
	return Generation{value: value}, nil
}

// MustParseGeneration validates one generation token and panics on
// refusal. Tests and fixed fixtures use it; production paths use
// ParseGeneration and return the refusal.
func MustParseGeneration(value string) Generation {
	generation, err := ParseGeneration(value)
	if err != nil {
		panic(err)
	}
	return generation
}

func (generation Generation) String() string { return generation.value }

// Equal reports exact generation equality. Generations are
// immutable: two captures name the same source state if and only if
// their tokens are byte-identical.
func (generation Generation) Equal(other Generation) bool {
	return generation.value == other.value
}

func checkGeneration(raw json.RawMessage) (Generation, bool) {
	value, ok := checkStringBounds(raw, 1, 512)
	if !ok {
		return Generation{}, false
	}
	generation, err := ParseGeneration(value)
	if err != nil {
		return Generation{}, false
	}
	return generation, true
}

// CheckDigestsEqual refuses two capture digests that differ. Stable
// capture requires equal pre/post digests; file-size equality is
// never proof and is not consulted here.
func CheckDigestsEqual(pre, post scalar.Digest) error {
	if pre != post {
		return invalid("capture digests differ: pre %q, post %q", pre.String(), post.String())
	}
	return nil
}
