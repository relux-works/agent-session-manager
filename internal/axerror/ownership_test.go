package axerror

import (
	"encoding/json"
	"strings"
	"testing"
)

// The tests in this file pin one property: a validated Structured Error owns
// its diagnostic graph outright. They exist because revision 1 of this package
// copied only the top level of the details map and handed the live nested
// container back from Detail, so every Section 15.1 detail bound could be
// violated after ValidateDetails had already run and passed. That is a bypass
// path around the gate rather than a gate, and no test covered it: the suite
// only ever validated maps nobody mutated afterwards.
//
// Each case mutates a container the caller or the accessor retains and then
// asserts two things about the object itself - that its encoded bytes did not
// move, and that its own reader still accepts them. The containers are taken
// at three depths and on both sides of an array, so a deep copy narrowed to
// one level fails these tests rather than only a copy deleted outright.

// violateEveryDetailBound writes each Section 15.1 detail bound into container:
// a key naming a forbidden class, a value past the 16 KiB canonical size, a
// value past the depth-4 nesting limit, and a Go type validateDetailValue
// refuses. If any of it reaches the object, ValidateDetails validated a graph
// the object does not own.
func violateEveryDetailBound(container map[string]any) {
	container["password"] = "hunter2"
	container["blob"] = strings.Repeat("x", 32*1024)
	var deep any = "leaf"
	for level := 0; level < 6; level++ {
		deep = map[string]any{"n": deep}
	}
	container["deep"] = deep
	container["count"] = 7
}

// detailGraph builds a conforming nested diagnostic graph and returns every
// mutable container inside it, named for the position the mutation attacks.
func detailGraph() (Details, map[string]map[string]any) {
	depth2 := map[string]any{"stream": "stdout"}
	depth1 := map[string]any{"frame": depth2}
	firstFrame := map[string]any{"line": "first"}
	nestedFrame := map[string]any{"line": "second"}
	details := Details{
		"context": depth1,
		"frames":  []any{firstFrame, []any{nestedFrame}},
	}
	return details, map[string]map[string]any{
		"map at depth 1":            depth1,
		"map at depth 2":            depth2,
		"map inside an array":       firstFrame,
		"map inside a nested array": nestedFrame,
	}
}

func mustConstruct(test *testing.T, details Details) *Error {
	test.Helper()
	failure, err := New(Spec{
		Version: Version100,
		Code:    "provider_protocol_error",
		Message: "unusable first frame",
		IDs:     NoIDs(),
		Details: details,
	})
	if err != nil {
		test.Fatalf("a conforming failure was refused: %v", err)
	}
	return failure
}

func mustEncode(test *testing.T, failure *Error) string {
	test.Helper()
	encoded, err := json.Marshal(failure)
	if err != nil {
		test.Fatalf("marshal: %v", err)
	}
	return string(encoded)
}

// assertUnmovedAndSelfReadable states the two consequences that matter. The
// bytes are the object's whole observable output, and Decode is the reader the
// same package ships: an object whose own reader refuses it has already put a
// document on the wire that no conforming peer can use.
func assertUnmovedAndSelfReadable(test *testing.T, failure *Error, baseline, position string) {
	test.Helper()
	current := mustEncode(test, failure)
	if current != baseline {
		test.Fatalf("mutating the %s changed the encoded object\n before: %s\n  after: %s",
			position, baseline, current)
	}
	if _, err := Decode(Version100, []byte(current)); err != nil {
		test.Fatalf("after mutating the %s the writer emits an object its own reader refuses: %v",
			position, err)
	}
}

// TestConstructionDoesNotAliasTheCallerDetailGraph drives New with a graph the
// call site keeps a reference to, which is the ordinary way a caller builds
// details, and then writes every forbidden shape into the containers it kept.
func TestConstructionDoesNotAliasTheCallerDetailGraph(test *testing.T) {
	details, retained := detailGraph()
	failure := mustConstruct(test, details)
	baseline := mustEncode(test, failure)

	for position, container := range retained {
		violateEveryDetailBound(container)
		assertUnmovedAndSelfReadable(test, failure, baseline, "caller's retained "+position)
	}

	// The top-level map is the arm revision 1 did get right; it is kept here
	// so a copy deleted outright fails on this row too.
	details["secret"] = "environment secret written after validation"
	assertUnmovedAndSelfReadable(test, failure, baseline, "caller's retained top-level map")

	// Replacing an array member is a mutation of the slice rather than of a
	// map inside it, and a copy that clones maps but shares slices admits it.
	details["frames"].([]any)[0] = map[string]any{"password": "hunter2"}
	assertUnmovedAndSelfReadable(test, failure, baseline, "caller's retained array")
}

// TestDetailAccessorDoesNotHandOutTheLiveContainer attacks the package's own
// exported accessor, which needs no cooperation from the constructing call site
// at all: any holder of a validated *Error could reach the live graph through
// it.
func TestDetailAccessorDoesNotHandOutTheLiveContainer(test *testing.T) {
	details, _ := detailGraph()
	failure := mustConstruct(test, details)
	baseline := mustEncode(test, failure)

	context, present := failure.Detail("context")
	if !present {
		test.Fatal("Detail did not report a key the object carries")
	}
	depth1, ok := context.(map[string]any)
	if !ok {
		test.Fatalf("Detail returned %T, the object carries a map", context)
	}
	violateEveryDetailBound(depth1)
	assertUnmovedAndSelfReadable(test, failure, baseline, "container returned by Detail")

	depth2, ok := depth1["frame"].(map[string]any)
	if !ok {
		test.Fatalf("the returned graph lost its nested map: %#v", depth1["frame"])
	}
	violateEveryDetailBound(depth2)
	assertUnmovedAndSelfReadable(test, failure, baseline, "second-level container returned by Detail")

	frames, present := failure.Detail("frames")
	if !present {
		test.Fatal("Detail did not report the array key the object carries")
	}
	members, ok := frames.([]any)
	if !ok {
		test.Fatalf("Detail returned %T for an array member", frames)
	}
	violateEveryDetailBound(members[0].(map[string]any))
	violateEveryDetailBound(members[1].([]any)[0].(map[string]any))
	members[0] = "replaced"
	assertUnmovedAndSelfReadable(test, failure, baseline, "array returned by Detail")

	// A second read is unaffected by what was written into the first, so the
	// isolation is per call rather than a single defensive copy shared by all
	// callers.
	second, _ := failure.Detail("context")
	if _, leaked := second.(map[string]any)["password"]; leaked {
		test.Fatal("a second Detail read carries a key written into the first read's copy")
	}
}

// TestDecodedObjectsOwnTheirDetailsToo pins the same property on the reading
// side. Decode builds its map from the document bytes rather than from a
// caller's graph, so only the accessor arm applies here - and it is the arm a
// peer-supplied object would be attacked through.
func TestDecodedObjectsOwnTheirDetailsToo(test *testing.T) {
	details, _ := detailGraph()
	source := mustConstruct(test, details)
	wire := mustEncode(test, source)

	failure, err := Decode(Version100, []byte(wire))
	if err != nil {
		test.Fatalf("Decode refused this package's own output: %v", err)
	}
	baseline := mustEncode(test, failure)

	context, present := failure.Detail("context")
	if !present {
		test.Fatal("the decoded object lost its context detail")
	}
	violateEveryDetailBound(context.(map[string]any))
	assertUnmovedAndSelfReadable(test, failure, baseline, "container returned by Detail on a decoded object")
}
