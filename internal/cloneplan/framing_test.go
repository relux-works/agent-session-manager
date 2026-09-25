package cloneplan_test

import (
	"bytes"
	"strings"
	"testing"

	cloneplan "github.com/relux-works/agent-session-manager/internal/cloneplan"
)

// TestFraming pins the strict frame on both decode entries:
// non-objects, trailing data, lone-surrogate escapes, and invalid
// UTF-8 refuse before any member gate runs.
func TestFraming(t *testing.T) {
	sealed := mustBuildPlan(t, validPlanInput())
	cases := []struct {
		label string
		raw   []byte
	}{
		{"array", []byte(`[]`)},
		{"string", []byte(`"x"`)},
		{"number", []byte(`7`)},
		{"null", []byte(`null`)},
		{"empty", []byte(``)},
		// Trailing whitespace is admitted by the landed strict
		// frame (JSON-insignificant); trailing values refuse.
		{"trailing-object", append(append([]byte{}, sealed...), []byte(`{}`)...)},
		{"lone-surrogate", []byte(`{"schema":"a�b"}`)},
		{"invalid-utf8", []byte{0x7b, 0x22, 0xff, 0x22, 0x7d}},
	}
	for _, row := range cases {
		_, err := cloneplan.DecodeProjectionPlan(row.raw)
		requireRefusal(t, err, "projection plan")
	}
	msealed := mustBuildManifest(t, validManifestInput())
	mcases := []struct {
		label string
		raw   []byte
	}{
		{"array", []byte(`[]`)},
		{"trailing-object", append(append([]byte{}, msealed...), []byte(`{}`)...)},
		{"invalid-utf8", []byte{0x7b, 0x22, 0xff, 0x22, 0x7d}},
	}
	for _, row := range mcases {
		_, err := cloneplan.DecodeProjectedObjectManifest(row.raw)
		requireRefusal(t, err, "projected object manifest")
	}
	// A lone surrogate escape inside a member value refuses too.
	raw := strings.Replace(string(sealed), `"target_native_writer"`, `"target\ud800_writer"`, 1)
	_, err := cloneplan.DecodeProjectionPlan([]byte(raw))
	requireRefusal(t, err, "projection plan")
}

// TestDeterminism pins that identical inputs produce byte-identical
// outputs on both build entries.
func TestDeterminism(t *testing.T) {
	first := mustBuildPlan(t, validPlanInput())
	for index := 0; index < 10; index++ {
		sealed := mustBuildPlan(t, validPlanInput())
		if !bytes.Equal(sealed, first) {
			t.Fatalf("plan build %d differs from build 0", index)
		}
	}
	mfirst := mustBuildManifest(t, validManifestInput())
	for index := 0; index < 10; index++ {
		sealed := mustBuildManifest(t, validManifestInput())
		if !bytes.Equal(sealed, mfirst) {
			t.Fatalf("manifest build %d differs from build 0", index)
		}
	}
}

// TestEntriesArePure pins that decode never mutates its input and
// every build output decodes clean.
func TestEntriesArePure(t *testing.T) {
	sealed := mustBuildPlan(t, validPlanInput())
	before := append([]byte{}, sealed...)
	mustDecodePlan(t, sealed)
	if !bytes.Equal(sealed, before) {
		t.Fatal("DecodeProjectionPlan mutated its input")
	}
	msealed := mustBuildManifest(t, validManifestInput())
	mbefore := append([]byte{}, msealed...)
	mustDecodeManifest(t, msealed)
	if !bytes.Equal(msealed, mbefore) {
		t.Fatal("DecodeProjectedObjectManifest mutated its input")
	}
}

// TestSchemaLiterals pins the exact schema/version literals on both
// decode entries with literal expectations.
func TestSchemaLiterals(t *testing.T) {
	sealed := mustBuildPlan(t, validPlanInput())
	for _, schema := range []string{"", "urn:ax:schema:projection-plan ", "urn:ax:schema:projection_plan", "urn:ax:schema:fidelity-report"} {
		document := decodeDocument(t, sealed)
		document["schema"] = schema
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "schema is not urn:ax:schema:projection-plan")
	}
	for _, version := range []string{"", "1.0", "1.0.0 ", "2.0.0"} {
		document := decodeDocument(t, sealed)
		document["schema_version"] = version
		_, err := cloneplan.DecodeProjectionPlan(marshalDocument(t, document))
		requireRefusal(t, err, "schema_version is not 1.0.0")
	}
	msealed := mustBuildManifest(t, validManifestInput())
	for _, schema := range []string{"", "urn:ax:schema:clone-projected-object-manifest ", "urn:ax:schema:projection-plan"} {
		document := decodeDocument(t, msealed)
		document["schema"] = schema
		_, err := cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, document))
		requireRefusal(t, err, "schema is not urn:ax:schema:clone-projected-object-manifest")
	}
	document := decodeDocument(t, msealed)
	document["schema_version"] = "2.0.0"
	_, err := cloneplan.DecodeProjectedObjectManifest(marshalDocument(t, document))
	requireRefusal(t, err, "schema_version is not 1.0.0")
}
