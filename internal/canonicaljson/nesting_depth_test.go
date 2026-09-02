package canonicaljson

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// declaredMaxNestingDepth pins the production bound as a literal. The tests
// below build their documents from this pin rather than from maxNestingDepth,
// so widening or narrowing the production constant moves the accept/refuse
// boundary away from the documents and reddens the suite instead of silently
// following the mutation.
const declaredMaxNestingDepth = 256

// regressionNestingDepth reproduces the original crash shape: one million
// nested open brackets, a 2,000,000-byte document. On main 48db30b this input
// aborted the process through Canonicalize with "goroutine stack exceeds
// 1000000000-byte limit" and "fatal error: stack overflow", which recover
// cannot catch. It is well under the 5,242,880-byte identity size cap, so that
// cap never protected the identity entries either.
const regressionNestingDepth = 1_000_000

func TestMaxNestingDepthIsDeclaredAndPinned(t *testing.T) {
	t.Parallel()
	if maxNestingDepth != declaredMaxNestingDepth {
		t.Fatalf("maxNestingDepth = %d, want the pinned %d; changing the bound requires restating its rationale and re-pinning here", maxNestingDepth, declaredMaxNestingDepth)
	}
}

func TestCanonicalizeAcceptsDocumentAtMaxNestingDepth(t *testing.T) {
	t.Parallel()
	for name, build := range nestedDocumentBuilders() {
		t.Run(name, func(t *testing.T) {
			input := build(declaredMaxNestingDepth)
			got, err := Canonicalize(input)
			if err != nil {
				t.Fatalf("Canonicalize(%s at depth %d) error = %v, want acceptance", name, declaredMaxNestingDepth, err)
			}
			if !bytes.Equal(got, input) {
				t.Fatalf("Canonicalize(%s at depth %d) = %q, want the already-canonical input", name, declaredMaxNestingDepth, got)
			}
		})
	}
}

func TestCanonicalizeRefusesDocumentPastMaxNestingDepth(t *testing.T) {
	t.Parallel()
	for name, build := range nestedDocumentBuilders() {
		t.Run(name, func(t *testing.T) {
			_, err := Canonicalize(build(declaredMaxNestingDepth + 1))
			assertNestingDepthRefusal(t, "Canonicalize("+name+")", err, declaredMaxNestingDepth+1)
		})
	}
}

// TestIdentityEntriesAcceptDocumentAtMaxNestingDepth proves the identity
// entries decode a document exactly at the bound. No supported closed shape
// legitimately nests that deep (open extensions stop at 4 levels), so passage
// through the shared decoder is proven by the refusal coming from the later
// Section 1.6 extension depth gate with ErrInvalidIdentity, never from the
// decoder with ErrInvalidJSON.
func TestIdentityEntriesAcceptDocumentAtMaxNestingDepth(t *testing.T) {
	t.Parallel()
	for name, build := range nestedDocumentBuilders() {
		t.Run(name, func(t *testing.T) {
			input := identityObjectWithNestedExtension(t, build, declaredMaxNestingDepth)
			if _, err := Canonicalize(input); err != nil {
				t.Fatalf("Canonicalize(identity object with %s at depth %d) error = %v, want acceptance", name, declaredMaxNestingDepth, err)
			}
			for entry, call := range identityEntries() {
				_, _, err := call(input)
				if err == nil || !errors.Is(err, ErrInvalidIdentity) || errors.Is(err, ErrInvalidJSON) {
					t.Fatalf("%s(identity object with %s at depth %d) error = %v, want the extension shape refusal after a successful decode", entry, name, declaredMaxNestingDepth, err)
				}
				if !strings.Contains(err.Error(), "maximum nesting depth 4") {
					t.Fatalf("%s(identity object with %s at depth %d) error = %v, want the Section 1.6 extension depth gate to be the refusing rule", entry, name, declaredMaxNestingDepth, err)
				}
			}
		})
	}
}

func TestIdentityEntriesRefuseDocumentPastMaxNestingDepth(t *testing.T) {
	t.Parallel()
	for name, build := range nestedDocumentBuilders() {
		t.Run(name, func(t *testing.T) {
			input := identityObjectWithNestedExtension(t, build, declaredMaxNestingDepth+1)
			for entry, call := range identityEntries() {
				_, _, err := call(input)
				assertNestingDepthRefusal(t, entry+"(identity object with "+name+")", err, declaredMaxNestingDepth+1)
			}
		})
	}
}

func TestNestingDepthRegressionTwoMegabyteArrayReturnsTypedError(t *testing.T) {
	t.Parallel()
	input := nestedArrayDocument(regressionNestingDepth)
	if len(input) != 2_000_000 || len(input) >= 5_242_880 {
		t.Fatalf("regression input is %d bytes, want the 2,000,000-byte shape under the identity size cap", len(input))
	}

	_, err := Canonicalize(input)
	assertNestingDepthRefusal(t, "Canonicalize(2 MB nested array)", err, declaredMaxNestingDepth+1)
	for entry, call := range identityEntries() {
		_, _, err := call(input)
		assertNestingDepthRefusal(t, entry+"(2 MB nested array)", err, declaredMaxNestingDepth+1)
	}
}

func identityEntries() map[string]func([]byte) (any, SelfField, error) {
	return map[string]func([]byte) (any, SelfField, error){
		"CalculateObjectIdentity": func(input []byte) (any, SelfField, error) {
			return CalculateObjectIdentity(input)
		},
		"VerifyObjectIdentity": func(input []byte) (any, SelfField, error) {
			return VerifyObjectIdentity(input)
		},
	}
}

func assertNestingDepthRefusal(t *testing.T, call string, err error, refusedDepth int) {
	t.Helper()
	if err == nil || !errors.Is(err, ErrInvalidJSON) {
		t.Fatalf("%s error = %v, want ErrInvalidJSON nesting depth refusal", call, err)
	}
	want := fmt.Sprintf("nesting depth %d exceeds maximum %d", refusedDepth, declaredMaxNestingDepth)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("%s error = %v, want %q", call, err, want)
	}
}

// nestedDocumentBuilders returns canonical documents whose container nesting
// depth is exactly the requested count, in every container shape the decoder
// distinguishes: pure arrays, pure objects, and alternating containers.
func nestedDocumentBuilders() map[string]func(depth int) []byte {
	return map[string]func(depth int) []byte{
		"arrays":  nestedArrayDocument,
		"objects": nestedObjectDocument,
		"mixed":   nestedMixedDocument,
	}
}

func nestedArrayDocument(depth int) []byte {
	return []byte(strings.Repeat("[", depth) + strings.Repeat("]", depth))
}

func nestedObjectDocument(depth int) []byte {
	return []byte(strings.Repeat(`{"a":`, depth-1) + "{}" + strings.Repeat("}", depth-1))
}

func nestedMixedDocument(depth int) []byte {
	var builder strings.Builder
	closers := make([]byte, 0, depth)
	for level := 0; level < depth; level++ {
		if level%2 == 0 {
			builder.WriteString(`{"a":`)
			closers = append(closers, '}')
		} else {
			builder.WriteString("[")
			closers = append(closers, ']')
		}
	}
	// The innermost container must be closed immediately, so the last opener
	// written cannot be an object expecting a member value.
	document := builder.String()
	if closers[len(closers)-1] == '}' {
		document = strings.TrimSuffix(document, `{"a":`) + "{"
	}
	for index := len(closers) - 1; index >= 0; index-- {
		document += string(closers[index])
	}
	return []byte(document)
}

// identityObjectWithNestedExtension embeds a nested document inside a valid
// Session Record's open extensions so that the whole identity object nests
// exactly totalDepth containers: the record envelope is depth 1, the
// extensions object is depth 2, and the embedded value supplies the rest.
func identityObjectWithNestedExtension(t *testing.T, build func(depth int) []byte, totalDepth int) []byte {
	t.Helper()
	const envelopeDepth = 2
	object := genericExtensionIdentityObject(map[string]any{"a.depth": "NESTED_PLACEHOLDER"})
	encoded := mustJSON(t, object)
	replaced := bytes.Replace(encoded, []byte(`"NESTED_PLACEHOLDER"`), build(totalDepth-envelopeDepth), 1)
	if bytes.Equal(replaced, encoded) {
		t.Fatal("nested extension placeholder was not substituted")
	}
	return replaced
}
