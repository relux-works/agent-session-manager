package provider

import (
	"reflect"
	"testing"
)

// candidateSignature reduces a candidate list to its stable observable
// facts for byte-level determinism comparison.
func candidateSignature(candidates []Candidate) []string {
	var out []string
	for _, candidate := range candidates {
		digest, _ := candidate.Digest()
		canon, _ := candidate.CanonicalPath()
		owner, _ := candidate.Owner()
		source, _ := candidate.SourcePath()
		out = append(out, string(candidate.Kind())+"/"+candidate.ID()+"/"+candidate.Source()+"/"+source+"/"+canon+"/"+digest.String()+"/"+owner)
	}
	return out
}

// TestDiscoverIsDeterministic proves repeated discovery over an unchanged
// tree returns identical candidates: same IDs, sources, paths, digests,
// and owners in the same order.
func TestDiscoverIsDeterministic(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins/b", "ax-provider-second", []byte("two"), fakeUID)
	fake.addLinkedFile("/plugins/a", "ax-provider-first", "/opt/real/first", []byte("one"), fakeUID, true)
	fake.addFile("/pathbin", "ax-provider-third", []byte("three"), fakeUID)
	fake.pathDirs = []string{"/pathbin"}
	cfg := fakeConfig("/plugins/b", "/plugins/a")
	cfg.AllowPathPlugins = true

	first, err := Discover(cfg, fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	second, err := Discover(cfg, fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !reflect.DeepEqual(candidateSignature(first), candidateSignature(second)) {
		t.Fatalf("repeated discovery differs:\n%v\n%v", candidateSignature(first), candidateSignature(second))
	}
}

// TestDiscoverIgnoresFilesystemOrder proves discovery is independent of
// directory listing order: two fakes holding the same entries in opposite
// insertion orders yield identical candidates. The fake already reverses
// insertion order on read; here the insertion itself is reversed, so both
// listing parities are covered.
func TestDiscoverIgnoresFilesystemOrder(t *testing.T) {
	forward := newFakeSystem()
	forward.addFile("/plugins", "ax-provider-bravo", []byte("b"), fakeUID)
	forward.addFile("/plugins", "ax-provider-alpha", []byte("a"), fakeUID)
	forward.addFile("/plugins", "ax-provider-charlie", []byte("c"), fakeUID)

	backward := newFakeSystem()
	backward.addFile("/plugins", "ax-provider-charlie", []byte("c"), fakeUID)
	backward.addFile("/plugins", "ax-provider-alpha", []byte("a"), fakeUID)
	backward.addFile("/plugins", "ax-provider-bravo", []byte("b"), fakeUID)

	first, err := Discover(fakeConfig("/plugins"), fakeOwner(), forward)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	second, err := Discover(fakeConfig("/plugins"), fakeOwner(), backward)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !reflect.DeepEqual(candidateSignature(first), candidateSignature(second)) {
		t.Fatalf("listing-order-dependent discovery:\n%v\n%v", candidateSignature(first), candidateSignature(second))
	}
}
