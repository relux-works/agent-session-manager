// Bidirectional identity-ownership battery for the three terminal closed
// schemas (advisories A1/A2 leaf: establish-single-object-identity-owner).
//
// internal/terminalbackend owns the Terminal Backend Manifest, Terminal
// Backend Probe, and Capability Evidence 1.0.0 closed schemas and their
// omit-self identity rule; internal/canonicaljson reaches the owner through
// the production Parse entries rather than re-implementing the shape. This
// battery is what fails when the two entries disagree in either direction:
// every row drives one shared document through both production entries and
// requires the same verdict, the full rendering on each side, and — for
// every valid document — byte-identical digests.
//
// The verdict mapping is exact, not approximate:
//   - terminalbackend.ParseManifest/ParseProbe/ParseEvidence verify the
//     identity binding as part of admission. canonicaljson reaches them
//     inside CalculateObjectIdentity (through the shape gate), so unlike
//     every other schema, a terminal document with a tampered claim is
//     refused by CalculateObjectIdentity as well as by VerifyObjectIdentity.
//     The tampered-claim rows pin that uniform refusal on all three entries.
//   - documents canonicaljson refuses before delegation (oversize,
//     unsafe numbers, undecodable bytes, unknown schema) are refused by
//     the owner through its own arms; each such row names both arms, so a
//     content-rule divergence cannot hide behind a shared refusal.
package terminalbackend_test

import (
	"encoding/json"
	"errors"
	"github.com/relux-works/agent-session-manager/internal/invcore"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// deriveTerminalMembers derives the closed member list behind a
// package-level []string variable in manifest.go. The drop-each-member and
// wrong-type-each-member sweeps below iterate exactly this set: a member
// added to production without corpus coverage fails here, not silently.
func deriveTerminalMembers(t *testing.T, symbol string) []string {
	t.Helper()
	path := filepath.Join(mustTerminalRoot(t), "manifest.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("terminal members: %v", err)
	}
	syntax, _, failure := invcore.ParseBytes(path, source, 0)
	if failure != "" {
		t.Fatalf("terminal members: %s", failure)
	}
	for _, decl := range syntax.Decls {
		node, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range node.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != symbol {
				continue
			}
			if len(value.Values) != 1 {
				t.Fatalf("terminal members: %s is not a single literal; the extractor cannot see through indirection", symbol)
			}
			composite, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				t.Fatalf("terminal members: %s is not a composite literal; the extractor cannot see through indirection", symbol)
			}
			var members []string
			for _, element := range composite.Elts {
				literal, ok := element.(*ast.BasicLit)
				if !ok {
					t.Fatalf("terminal members: %s holds a non-literal; the extractor cannot see through indirection", symbol)
				}
				members = append(members, strings.Trim(literal.Value, `"`))
			}
			return members
		}
	}
	t.Fatalf("terminal members: %s not found; the extractor is blind, not the table absent", symbol)
	return nil
}

// mustTerminalRoot returns the terminalbackend package directory: this file
// lives in it, so the derivation reads production source, never a copy.
func mustTerminalRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("terminal members: %v", err)
	}
	return directory
}

// TestTerminalMemberListsAreDerivedAndPinned requires the three closed
// member tables to derive from production and pins their sizes: a member
// added to any table joins the drop/wrong-type sweeps below through the
// derivation, and the size tripwire fails closed on a silently truncated
// read.
func TestTerminalMemberListsAreDerivedAndPinned(t *testing.T) {
	t.Parallel()

	for symbol, want := range map[string]int{
		"manifestMembers": 12,
		"probeMembers":    16,
		"evidenceMembers": 24,
	} {
		members := deriveTerminalMembers(t, symbol)
		if len(members) != want {
			t.Errorf("%s holds %d members, want exactly %d; a new member joins the sweeps or not at all", symbol, len(members), want)
		}
		seen := make(map[string]bool, len(members))
		for _, member := range members {
			if seen[member] {
				t.Errorf("%s lists %q twice", symbol, member)
			}
			seen[member] = true
		}
	}
}

// decodeOwnershipMap decodes one document preserving number literals, so
// map surgery round-trips the exact wire form the owner gates on.
func decodeOwnershipMap(t *testing.T, doc []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(string(doc)))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		t.Fatalf("decode ownership document: %v", err)
	}
	return object
}

// mustEncodeOwnershipMap encodes one document map back to wire bytes.
func mustEncodeOwnershipMap(t *testing.T, object map[string]any) []byte {
	t.Helper()
	raw, err := json.Marshal(object)
	if err != nil {
		t.Fatalf("encode ownership document: %v", err)
	}
	return raw
}

// dropOwnershipMember returns the document without one top-level member.
func dropOwnershipMember(t *testing.T, doc []byte, member string) []byte {
	t.Helper()
	object := decodeOwnershipMap(t, doc)
	delete(object, member)
	return mustEncodeOwnershipMap(t, object)
}

// setOwnershipMember returns the document with one top-level member
// replaced by the given value.
func setOwnershipMember(t *testing.T, doc []byte, member string, value any) []byte {
	t.Helper()
	object := decodeOwnershipMap(t, doc)
	object[member] = value
	return mustEncodeOwnershipMap(t, object)
}

// terminalRefusalDetail reports the owner's refusal detail for a refused
// document: every agreement row below ties the canonical entry's refusal
// to this exact arm rather than to a shared code.
func terminalRefusalDetail(t *testing.T, err error) string {
	t.Helper()
	var refusal *terminalbackend.Error
	if !errors.As(err, &refusal) {
		t.Fatalf("owner error = %v, want a terminal backend refusal", err)
	}
	if refusal.Code != codeMismatch {
		t.Fatalf("owner code = %q, want %q", refusal.Code, codeMismatch)
	}
	return refusal.Detail
}

// requireCanonicalRefusalThroughOwner requires canonicaljson to refuse
// through the delegated owner gate: an identity refusal whose rendering
// carries the owner's exact detail. A deleted delegation refuses with the
// unsupported-shape text instead and fails here.
func requireCanonicalRefusalThroughOwner(t *testing.T, doc []byte, ownerDetail string) {
	t.Helper()
	_, _, err := canonicaljson.CalculateObjectIdentity(doc)
	if err == nil || !errors.Is(err, canonicaljson.ErrInvalidIdentity) {
		t.Fatalf("canonicaljson error = %v, want an identity refusal", err)
	}
	if !strings.Contains(err.Error(), ownerDetail) {
		t.Fatalf("canonicaljson error = %v, want the owner arm %q", err, ownerDetail)
	}
	if _, _, err := canonicaljson.VerifyObjectIdentity(doc); err == nil || !errors.Is(err, canonicaljson.ErrInvalidIdentity) {
		t.Fatalf("canonicaljson verify error = %v, want an identity refusal", err)
	}
}

// TestTerminalManifestIdentityAgreement drives one valid manifest and every
// systematic mutant through both identity entries: the owner verdict and
// detail, the delegated verdict carrying that detail, and digest equality
// on the valid document.
func TestTerminalManifestIdentityAgreement(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	valid := mustMarshal(t, universe.manifest)

	t.Run("valid manifest agrees with equal digests", func(t *testing.T) {
		t.Parallel()
		admitted, err := terminalbackend.ParseManifest(valid)
		if err != nil {
			t.Fatalf("ParseManifest(valid) error = %v", err)
		}
		if admitted.ManifestID != testIdentity(t, decodeOwnershipMap(t, valid), "manifest_id") {
			t.Fatalf("ParseManifest ID = %q, want the omit-self recipe digest", admitted.ManifestID)
		}
		digest, field, err := canonicaljson.CalculateObjectIdentity(valid)
		if err != nil {
			t.Fatalf("CalculateObjectIdentity(valid) error = %v", err)
		}
		if field != canonicaljson.SelfManifestID {
			t.Fatalf("CalculateObjectIdentity field = %q, want %q", field, canonicaljson.SelfManifestID)
		}
		if digest.String() != admitted.ManifestID {
			t.Fatalf("CalculateObjectIdentity digest = %q, owner digest = %q", digest, admitted.ManifestID)
		}
		verified, _, err := canonicaljson.VerifyObjectIdentity(valid)
		if err != nil {
			t.Fatalf("VerifyObjectIdentity(valid) error = %v", err)
		}
		if verified != digest {
			t.Fatalf("VerifyObjectIdentity digest = %q, want %q", verified, digest)
		}
	})

	t.Run("tampered claim refused on all three entries", func(t *testing.T) {
		t.Parallel()
		doc := setOwnershipMember(t, valid, "manifest_id", testSeedDigest(0x00))
		_, err := terminalbackend.ParseManifest(doc)
		if err == nil {
			t.Fatalf("ParseManifest(tampered claim) admitted the document")
		}
		requireRefusal(t, err, codeMismatch, "document identity binding")
		requireCanonicalRefusalThroughOwner(t, doc, "document identity binding")
	})

	// Drop-each-member over the production-derived table: every drop breaks
	// the closed member set before identity runs, so the owner reports the
	// members arm and the delegate carries it. The dispatch members are the
	// splits: canonicaljson resolves schema, version, and the self field
	// before reaching the owner, so a drop there names the dispatch arm
	// rather than the owner arm. Same verdict, different arm by
	// construction; the dispatch contract is resolveSelfField, and a change
	// to it reddens the phrases below rather than passing silently.
	for _, member := range deriveTerminalMembers(t, "manifestMembers") {
		member := member
		t.Run("drop "+member, func(t *testing.T) {
			t.Parallel()
			doc := dropOwnershipMember(t, valid, member)
			_, err := terminalbackend.ParseManifest(doc)
			if err == nil {
				t.Fatalf("ParseManifest without %q admitted the document", member)
			}
			requireRefusal(t, err, codeMismatch, "document members")
			switch member {
			case "manifest_id":
				requireCanonicalDispatchRefusal(t, doc, "requires self field")
			case "schema", "schema_version":
				requireCanonicalDispatchRefusal(t, doc, "requires string member")
			default:
				requireCanonicalRefusalThroughOwner(t, doc, "document members")
			}
		})
	}

	// Wrong-type-each-member with a safe number: the number passes the
	// canonical safe-integer gate, so delegation is always reached and the
	// delegate must carry the owner's exact arm. The member-type detail
	// varies per member; capturing it from the owner ties the two entries
	// to the same gate without a hand-written detail table (the arm to
	// detail mapping itself is pinned per arm in TestManifestDocumentRefusals).
	// The dispatch members are the splits, as above.
	for _, member := range deriveTerminalMembers(t, "manifestMembers") {
		member := member
		t.Run("number for "+member, func(t *testing.T) {
			t.Parallel()
			doc := setOwnershipMember(t, valid, member, json.Number("1"))
			_, err := terminalbackend.ParseManifest(doc)
			if err == nil {
				t.Fatalf("ParseManifest with a number for %q admitted the document", member)
			}
			switch member {
			case "manifest_id":
				requireRefusal(t, err, codeMismatch, "document member type")
				requireCanonicalDispatchRefusal(t, doc, "self field")
			case "schema", "schema_version":
				requireCanonicalDispatchRefusal(t, doc, "non-empty string")
			default:
				detail := terminalRefusalDetail(t, err)
				requireCanonicalRefusalThroughOwner(t, doc, detail)
			}
		})
	}
}

// requireCanonicalDispatchRefusal requires canonicaljson to refuse at its
// dispatch contract (before delegation) with the named arm phrase.
func requireCanonicalDispatchRefusal(t *testing.T, doc []byte, phrase string) {
	t.Helper()
	_, _, err := canonicaljson.CalculateObjectIdentity(doc)
	if err == nil || !errors.Is(err, canonicaljson.ErrInvalidIdentity) {
		t.Fatalf("canonicaljson error = %v, want an identity refusal", err)
	}
	if !strings.Contains(err.Error(), phrase) {
		t.Fatalf("canonicaljson error = %v, want the dispatch arm %q", err, phrase)
	}
}

// TestTerminalProbeIdentityAgreement is the probe half of the ownership
// battery: valid probe, tampered claim, and the derived drop/wrong-type
// sweeps through both entries.
func TestTerminalProbeIdentityAgreement(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	valid := mustMarshal(t, universe.probe)

	t.Run("valid probe agrees with equal digests", func(t *testing.T) {
		t.Parallel()
		admitted, err := terminalbackend.ParseProbe(valid)
		if err != nil {
			t.Fatalf("ParseProbe(valid) error = %v", err)
		}
		if admitted.ProbeID != testIdentity(t, decodeOwnershipMap(t, valid), "probe_id") {
			t.Fatalf("ParseProbe ID = %q, want the omit-self recipe digest", admitted.ProbeID)
		}
		digest, field, err := canonicaljson.CalculateObjectIdentity(valid)
		if err != nil {
			t.Fatalf("CalculateObjectIdentity(valid) error = %v", err)
		}
		if field != canonicaljson.SelfProbeID {
			t.Fatalf("CalculateObjectIdentity field = %q, want %q", field, canonicaljson.SelfProbeID)
		}
		if digest.String() != admitted.ProbeID {
			t.Fatalf("CalculateObjectIdentity digest = %q, owner digest = %q", digest, admitted.ProbeID)
		}
		verified, _, err := canonicaljson.VerifyObjectIdentity(valid)
		if err != nil {
			t.Fatalf("VerifyObjectIdentity(valid) error = %v", err)
		}
		if verified != digest {
			t.Fatalf("VerifyObjectIdentity digest = %q, want %q", verified, digest)
		}
	})

	t.Run("tampered claim refused on all three entries", func(t *testing.T) {
		t.Parallel()
		doc := setOwnershipMember(t, valid, "probe_id", testSeedDigest(0x00))
		_, err := terminalbackend.ParseProbe(doc)
		if err == nil {
			t.Fatalf("ParseProbe(tampered claim) admitted the document")
		}
		requireRefusal(t, err, codeMismatch, "document identity binding")
		requireCanonicalRefusalThroughOwner(t, doc, "document identity binding")
	})

	for _, member := range deriveTerminalMembers(t, "probeMembers") {
		member := member
		t.Run("drop "+member, func(t *testing.T) {
			t.Parallel()
			doc := dropOwnershipMember(t, valid, member)
			_, err := terminalbackend.ParseProbe(doc)
			if err == nil {
				t.Fatalf("ParseProbe without %q admitted the document", member)
			}
			requireRefusal(t, err, codeMismatch, "document members")
			switch member {
			case "probe_id":
				requireCanonicalDispatchRefusal(t, doc, "requires self field")
			case "schema", "schema_version":
				requireCanonicalDispatchRefusal(t, doc, "requires string member")
			default:
				requireCanonicalRefusalThroughOwner(t, doc, "document members")
			}
		})
	}

	for _, member := range deriveTerminalMembers(t, "probeMembers") {
		member := member
		t.Run("number for "+member, func(t *testing.T) {
			t.Parallel()
			doc := setOwnershipMember(t, valid, member, json.Number("1"))
			_, err := terminalbackend.ParseProbe(doc)
			if err == nil {
				t.Fatalf("ParseProbe with a number for %q admitted the document", member)
			}
			switch member {
			case "probe_id":
				requireRefusal(t, err, codeMismatch, "document member type")
				requireCanonicalDispatchRefusal(t, doc, "self field")
			case "schema", "schema_version":
				requireCanonicalDispatchRefusal(t, doc, "non-empty string")
			default:
				detail := terminalRefusalDetail(t, err)
				requireCanonicalRefusalThroughOwner(t, doc, detail)
			}
		})
	}
}

// TestTerminalEvidenceIdentityAgreement is the evidence half of the
// ownership battery, over one signed non-realm evidence document.
func TestTerminalEvidenceIdentityAgreement(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	valid := mustMarshal(t, universe.evidenceByCap["durable_disconnect"])

	t.Run("valid evidence agrees with equal digests", func(t *testing.T) {
		t.Parallel()
		admitted, err := terminalbackend.ParseEvidence(valid)
		if err != nil {
			t.Fatalf("ParseEvidence(valid) error = %v", err)
		}
		if admitted.EvidenceID != testIdentity(t, decodeOwnershipMap(t, valid), "evidence_id") {
			t.Fatalf("ParseEvidence ID = %q, want the omit-self recipe digest", admitted.EvidenceID)
		}
		digest, field, err := canonicaljson.CalculateObjectIdentity(valid)
		if err != nil {
			t.Fatalf("CalculateObjectIdentity(valid) error = %v", err)
		}
		if field != canonicaljson.SelfEvidenceID {
			t.Fatalf("CalculateObjectIdentity field = %q, want %q", field, canonicaljson.SelfEvidenceID)
		}
		if digest.String() != admitted.EvidenceID {
			t.Fatalf("CalculateObjectIdentity digest = %q, owner digest = %q", digest, admitted.EvidenceID)
		}
		verified, _, err := canonicaljson.VerifyObjectIdentity(valid)
		if err != nil {
			t.Fatalf("VerifyObjectIdentity(valid) error = %v", err)
		}
		if verified != digest {
			t.Fatalf("VerifyObjectIdentity digest = %q, want %q", verified, digest)
		}
	})

	t.Run("tampered claim refused on all three entries", func(t *testing.T) {
		t.Parallel()
		doc := setOwnershipMember(t, valid, "evidence_id", testSeedDigest(0x00))
		_, err := terminalbackend.ParseEvidence(doc)
		if err == nil {
			t.Fatalf("ParseEvidence(tampered claim) admitted the document")
		}
		requireRefusal(t, err, codeMismatch, "document identity binding")
		requireCanonicalRefusalThroughOwner(t, doc, "document identity binding")
	})

	for _, member := range deriveTerminalMembers(t, "evidenceMembers") {
		member := member
		t.Run("drop "+member, func(t *testing.T) {
			t.Parallel()
			doc := dropOwnershipMember(t, valid, member)
			_, err := terminalbackend.ParseEvidence(doc)
			if err == nil {
				t.Fatalf("ParseEvidence without %q admitted the document", member)
			}
			requireRefusal(t, err, codeMismatch, "document members")
			switch member {
			case "evidence_id":
				requireCanonicalDispatchRefusal(t, doc, "requires self field")
			case "schema", "schema_version":
				requireCanonicalDispatchRefusal(t, doc, "requires string member")
			default:
				requireCanonicalRefusalThroughOwner(t, doc, "document members")
			}
		})
	}

	for _, member := range deriveTerminalMembers(t, "evidenceMembers") {
		member := member
		t.Run("number for "+member, func(t *testing.T) {
			t.Parallel()
			doc := setOwnershipMember(t, valid, member, json.Number("1"))
			_, err := terminalbackend.ParseEvidence(doc)
			if err == nil {
				t.Fatalf("ParseEvidence with a number for %q admitted the document", member)
			}
			switch member {
			case "evidence_id":
				requireRefusal(t, err, codeMismatch, "document member type")
				requireCanonicalDispatchRefusal(t, doc, "self field")
			case "schema", "schema_version":
				requireCanonicalDispatchRefusal(t, doc, "non-empty string")
			default:
				detail := terminalRefusalDetail(t, err)
				requireCanonicalRefusalThroughOwner(t, doc, detail)
			}
		})
	}
}

// TestTerminalRealmEvidenceIdentityAgreement drives one signed realm evidence
// document through both entries: the credential-realm conditional members
// are the largest shape rule the sweeps above never touch (they stay null
// there), so the realm half of parseRealmMembers needs its own agreement
// row, plus the realm-less-attachment refusal in both directions.
func TestTerminalRealmEvidenceIdentityAgreement(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	realm := universe.evidenceMap(
		"credential_capable_execution_realm",
		[]any{"fixture_passed", "provider_auth_passed", "runtime_probe_passed", "sentinel_passed"},
	)
	realm["terminal_binding_id"] = testSeedDigest(0xB1)
	realm["provider_id"] = "codex"
	realm["provider_build"] = "1.2.3"
	realm["sentinel_result"] = "passed"
	realm["provider_auth_smoke_result"] = "passed"
	finalizeEvidence(t, universe.key, realm)
	valid := mustMarshal(t, realm)

	t.Run("valid realm evidence agrees with equal digests", func(t *testing.T) {
		t.Parallel()
		admitted, err := terminalbackend.ParseEvidence(valid)
		if err != nil {
			t.Fatalf("ParseEvidence(realm) error = %v", err)
		}
		digest, field, err := canonicaljson.CalculateObjectIdentity(valid)
		if err != nil {
			t.Fatalf("CalculateObjectIdentity(realm) error = %v", err)
		}
		if field != canonicaljson.SelfEvidenceID || digest.String() != admitted.EvidenceID {
			t.Fatalf("CalculateObjectIdentity(realm) = %q/%q, owner = %q", digest, field, admitted.EvidenceID)
		}
		if verified, _, err := canonicaljson.VerifyObjectIdentity(valid); err != nil || verified != digest {
			t.Fatalf("VerifyObjectIdentity(realm) = %q, %v; want %q", verified, err, digest)
		}
	})

	t.Run("realm members on a non-realm claim refused by both", func(t *testing.T) {
		t.Parallel()
		base := decodeOwnershipMap(t, mustMarshal(t, universe.evidenceByCap["durable_disconnect"]))
		base["terminal_binding_id"] = testSeedDigest(0xB1)
		// No re-sign and no re-finalize: member-type validation refuses
		// before the identity binding is checked, so the stale claim and
		// the untouched signature never matter to this row.
		doc := mustEncodeOwnershipMap(t, base)
		_, err := terminalbackend.ParseEvidence(doc)
		if err == nil {
			t.Fatalf("ParseEvidence(realm members, non-realm claim) admitted the document")
		}
		requireRefusal(t, err, codeMismatch, "evidence realm binding")
		requireCanonicalRefusalThroughOwner(t, doc, "evidence realm binding")
	})
}

// TestTerminalIdentityOrderingRefusesMemberTypeBeforeBinding proves the
// NUM-UNSAFE ordering fix: a document carrying both a numeric member and a
// mismatched identity claim is refused at the member-type arm, never at the
// identity-binding arm. Under the old order (identity first) every row
// below reported "document identity binding" instead.
func TestTerminalIdentityOrderingRefusesMemberTypeBeforeBinding(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	manifest := mustMarshal(t, universe.manifest)
	probe := mustMarshal(t, universe.probe)
	evidence := mustMarshal(t, universe.evidenceByCap["durable_disconnect"])

	rows := []struct {
		name  string
		doc   []byte
		parse func([]byte) error
	}{
		{
			"manifest unsafe integer",
			setOwnershipMember(t,
				setOwnershipMember(t, manifest, "manifest_id", testSeedDigest(0x00)),
				"protocol_versions", []any{json.Number("9007199254740993")}),
			func(raw []byte) error { _, err := terminalbackend.ParseManifest(raw); return err },
		},
		{
			"probe float",
			setOwnershipMember(t,
				setOwnershipMember(t, probe, "probe_id", testSeedDigest(0x00)),
				"os_version", json.Number("1.5")),
			func(raw []byte) error { _, err := terminalbackend.ParseProbe(raw); return err },
		},
		{
			"evidence nested unsafe integer",
			setOwnershipMember(t,
				setOwnershipMember(t, evidence, "evidence_id", testSeedDigest(0x00)),
				"facts", []any{"fixture_passed", json.Number("9007199254740993")}),
			func(raw []byte) error { _, err := terminalbackend.ParseEvidence(raw); return err },
		},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			err := row.parse(row.doc)
			if err == nil {
				t.Fatalf("owner admitted a numeric document with a stale claim")
			}
			requireRefusal(t, err, codeMismatch, "document member type")
		})
	}
}

// TestTerminalDocumentsRefuseOversizeInput proves the shared 5 MiB identity
// bound at both entries: an otherwise-valid document padded past the bound
// is refused before decoding on either side.
func TestTerminalDocumentsRefuseOversizeInput(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	rows := []struct {
		name  string
		doc   []byte
		parse func([]byte) error
	}{
		{"manifest", mustMarshal(t, universe.manifest),
			func(raw []byte) error { _, err := terminalbackend.ParseManifest(raw); return err }},
		{"probe", mustMarshal(t, universe.probe),
			func(raw []byte) error { _, err := terminalbackend.ParseProbe(raw); return err }},
		{"evidence", mustMarshal(t, universe.evidenceByCap["durable_disconnect"]),
			func(raw []byte) error { _, err := terminalbackend.ParseEvidence(raw); return err }},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			padded := padOwnershipDocument(t, row.doc, 5_242_881)
			err := row.parse(padded)
			if err == nil {
				t.Fatalf("owner admitted a %d-byte document", len(padded))
			}
			requireRefusal(t, err, codeMismatch, "document size")
			_, _, calcErr := canonicaljson.CalculateObjectIdentity(padded)
			if calcErr == nil || !errors.Is(calcErr, canonicaljson.ErrInvalidIdentity) ||
				!strings.Contains(calcErr.Error(), "maximum is 5242880") {
				t.Fatalf("canonicaljson error = %v, want the 5 MiB identity bound", calcErr)
			}
		})
	}
}

// padOwnershipDocument returns the document padded with JSON insignificant
// whitespace to exactly size bytes. Whitespace carries no member, so the
// padded document is otherwise valid and only the size arm can refuse it.
func padOwnershipDocument(t *testing.T, doc []byte, size int) []byte {
	t.Helper()
	if len(doc) >= size || doc[len(doc)-1] != '}' {
		t.Fatalf("padOwnershipDocument: %d-byte document is not a padded object", len(doc))
	}
	padded := make([]byte, 0, size)
	padded = append(padded, doc[:len(doc)-1]...)
	for len(padded) < size-1 {
		padded = append(padded, ' ')
	}
	return append(padded, '}')
}

// TestTerminalManifestPreDelegationGatesAgree drives the documents
// canonicaljson refuses before reaching the owner: each row names the
// canonical arm and the owner's own arm for the same document, so a
// content-rule divergence cannot hide behind a shared refusal.
func TestTerminalManifestPreDelegationGatesAgree(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	valid := mustMarshal(t, universe.manifest)

	rows := []struct {
		name         string
		doc          []byte
		terminalCode string
		terminalArm  string
		canonArm     string
	}{
		{
			"unsafe integer member",
			setOwnershipMember(t, valid, "protocol_versions", []any{json.Number("9007199254740993")}),
			codeMismatch, "document member type", "outside the AX safe-integer interval",
		},
		{
			"floating-point member",
			setOwnershipMember(t, valid, "protocol_versions", []any{json.Number("1.5")}),
			codeMismatch, "document member type", "forbidden by the AX common model",
		},
		{
			"wrong schema",
			setOwnershipMember(t, valid, "schema", "urn:ax:schema:provider-identity"),
			codeMismatch, "manifest schema", "requires self field",
		},
		{
			"malformed self digest",
			setOwnershipMember(t, valid, "manifest_id", "sha256:zzzz"),
			codeMismatch, "document digest", "self field",
		},
		{
			"unknown member",
			setOwnershipMember(t, valid, "zzz", "x"),
			codeMismatch, "document members", "document members",
		},
		{
			"non-empty extensions",
			setOwnershipMember(t, valid, "extensions", map[string]any{"works.relux.ax.x": "y"}),
			codeMismatch, "document extensions", "document extensions",
		},
		{
			// A second schema member at the top level (index 0 is the
			// opening brace whatever the map order), so the duplicate
			// arm fires before any member rule reads a value.
			"duplicate member",
			append([]byte(`{"schema": "urn:ax:schema:terminal-backend-manifest",`), valid[1:]...),
			codeMismatch, "document duplicate member", "duplicate object member",
		},
		{
			"trailing data",
			append(append([]byte{}, valid...), ' ', '{', '}'),
			codeMismatch, "document trailing data", "trailing JSON",
		},
		{
			"deep nesting",
			setOwnershipMember(t, valid, "protocol_versions", deepOwnershipArrays(40)),
			codeMismatch, "document nesting", "document nesting",
		},
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			_, err := terminalbackend.ParseManifest(row.doc)
			if err == nil {
				t.Fatalf("ParseManifest(%s) admitted the document", row.name)
			}
			requireRefusal(t, err, row.terminalCode, row.terminalArm)
			_, _, calcErr := canonicaljson.CalculateObjectIdentity(row.doc)
			if calcErr == nil {
				t.Fatalf("CalculateObjectIdentity(%s) admitted the document", row.name)
			}
			if !errors.Is(calcErr, canonicaljson.ErrInvalidIdentity) && !errors.Is(calcErr, canonicaljson.ErrInvalidJSON) {
				t.Fatalf("CalculateObjectIdentity(%s) error = %v, want an identity or JSON refusal", row.name, calcErr)
			}
			if !strings.Contains(calcErr.Error(), row.canonArm) {
				t.Fatalf("CalculateObjectIdentity(%s) error = %v, want the arm %q", row.name, calcErr, row.canonArm)
			}
		})
	}

	t.Run("non-UTF-8 bytes", func(t *testing.T) {
		t.Parallel()
		broken := make([]byte, 0, len(valid))
		anchor := []byte(`"linux"`)
		if strings.Count(string(valid), string(anchor)) != 1 {
			t.Fatalf("surgery anchor is not unique")
		}
		at := strings.Index(string(valid), string(anchor))
		broken = append(broken, valid[:at+4]...)
		broken = append(broken, 0xff)
		broken = append(broken, valid[at+5:]...)
		_, err := terminalbackend.ParseManifest(broken)
		if err == nil {
			t.Fatal("ParseManifest(non-UTF-8) admitted the document")
		}
		requireRefusal(t, err, codeMismatch, "document encoding")
		_, _, calcErr := canonicaljson.CalculateObjectIdentity(broken)
		if calcErr == nil || !strings.Contains(calcErr.Error(), "not valid UTF-8") {
			t.Fatalf("CalculateObjectIdentity(non-UTF-8) error = %v, want the encoding arm", calcErr)
		}
	})
}

// deepOwnershipArrays builds depth nested single-element arrays holding a
// string leaf: depth 40 passes the canonical 256-deep decoder and fails
// the owner 32-deep decoder, proving both bounds agree on refusal.
func deepOwnershipArrays(depth int) any {
	var value any = "leaf"
	for index := 0; index < depth; index++ {
		value = []any{value}
	}
	return value
}

// TestSurrogateGateAgreesWithCanonicalJSON pins the local surrogate scan
// against the canonical owner: the owner's surrogate verdict and
// canonicaljson.Canonicalize agree accept/reject on every shared vector,
// so the two copies cannot drift without reddening here.
//
// The sweep moved to this external battery when canonicaljson began
// importing terminalbackend in production (terminal shape delegation): an
// internal test file can no longer import canonicaljson without an import
// cycle. The vectors embed in the terminal_backend_id member of a valid
// manifest, which isolates the surrogate question by construction: a lone
// escape is refused by the surrogate scan before any member rule reads the
// value, while an admitted escape reaches the member rules and is refused
// there instead. The branch-level pins (including the raw WTF-8 road, which
// never reaches the gate through a document because the UTF-8 arm fires
// first) stay in the white-box TestSurrogateGateVerdicts, which needs no
// cross-package import.
//
// A delete-only mutant proves the gate exists; the full-range enumeration
// proves the class it covers: narrowing the low-surrogate upper bound to
// <= 0xDC00 survived a 10-vector corpus while admitting 1023 lone low
// surrogates, and every loop below kills that mutant the same way.
func TestSurrogateGateAgreesWithCanonicalJSON(t *testing.T) {
	t.Parallel()

	universe := testUniverse(t)
	valid := mustMarshal(t, universe.manifest)
	if strings.Count(string(valid), `"ax.tmux"`) != 1 {
		t.Fatalf("surgery anchor is not unique")
	}
	embed := func(vector string) []byte {
		return []byte(strings.Replace(string(valid), `"ax.tmux"`, vector, 1))
	}

	slash := string([]byte{92})
	escape := func(unit uint16) string {
		const digits = "0123456789abcdef"
		return slash + "u" + string([]byte{
			digits[unit>>12&0xf], digits[unit>>8&0xf], digits[unit>>4&0xf], digits[unit&0xf],
		})
	}
	var failures []string
	// rejectVector requires both sides to refuse the surrogate question:
	// the owner at its surrogate arm, the canonical entry at its own.
	rejectVector := func(doc, vector string) {
		t.Helper()
		_, err := terminalbackend.ParseManifest(embed(doc))
		if err == nil {
			failures = append(failures, "ParseManifest admitted lone escape "+vector)
			return
		}
		var refusal *terminalbackend.Error
		if !errors.As(err, &refusal) || refusal.Detail != "document surrogate escape" {
			failures = append(failures, "ParseManifest("+vector+") = "+err.Error()+", want the surrogate arm")
		}
		if _, err := canonicaljson.Canonicalize([]byte(doc)); err == nil {
			failures = append(failures, "Canonicalize admitted lone escape "+vector)
		} else if !strings.Contains(err.Error(), "surrogate") {
			failures = append(failures, "Canonicalize("+vector+") = "+err.Error()+", want the surrogate arm")
		}
	}
	// admitVector requires both sides to pass the surrogate question: the
	// owner refuses at a member arm instead, and the canonical entry
	// carries that same arm through delegation.
	admitVector := func(doc, vector string) {
		t.Helper()
		_, err := terminalbackend.ParseManifest(embed(doc))
		if err == nil {
			failures = append(failures, "ParseManifest admitted "+vector+" at every arm")
			return
		}
		var refusal *terminalbackend.Error
		if !errors.As(err, &refusal) {
			failures = append(failures, "ParseManifest("+vector+") = "+err.Error()+", want a terminal refusal")
			return
		}
		if refusal.Detail == "document surrogate escape" {
			failures = append(failures, "ParseManifest refused "+vector+" at the surrogate arm, want a member arm")
		}
		if _, _, calcErr := canonicaljson.CalculateObjectIdentity(embed(doc)); calcErr == nil {
			failures = append(failures, "CalculateObjectIdentity admitted "+vector+" at every arm")
		} else if !strings.Contains(calcErr.Error(), refusal.Detail) {
			failures = append(failures, "CalculateObjectIdentity("+vector+") = "+calcErr.Error()+", want the owner arm "+refusal.Detail)
		}
	}

	for unit := uint32(0xd800); unit <= 0xdfff; unit++ {
		vector := `"` + escape(uint16(unit)) + `"`
		rejectVector(vector, vector)
	}
	for second := uint32(0xd800); second <= 0xdfff; second++ {
		vector := `"` + escape(0xd800) + escape(uint16(second)) + `"`
		if second < 0xdc00 {
			rejectVector(vector, vector)
		} else {
			admitVector(vector, vector)
		}
	}
	admitVector(`"`+escape(0xdbff)+escape(0xdfff)+`"`, "dbff+dfff pair")
	for _, unit := range []uint16{0x0000, 0x0041, 0x00e9, 0xd7ff, 0xe000, 0xffff} {
		admitVector(`"`+escape(unit)+`"`, "non-surrogate escape")
		rejectVector(`"`+escape(0xd800)+escape(unit)+`"`, "high plus non-low")
	}
	rejectVector(`"`+slash+`uDC00"`, "uppercase lone low")
	admitVector(`"`+slash+`uD800`+slash+`uDC00"`, "uppercase valid pair")
	admitVector(`"`+slash+slash+`ud800"`, "escaped backslash text")
	admitVector(`"abc"`, "plain string")
	admitVector(`"aé"`, "multibyte string")

	for _, bytes := range [][]byte{
		{0xed, 0xa0, 0x80},
		{0xed, 0xb0, 0x80},
		{0xed, 0xbf, 0xbf},
		{0xed, 0xaf, 0x93},
	} {
		vector := `"` + string(bytes) + `"`
		_, err := terminalbackend.ParseManifest(embed(vector))
		if err == nil {
			failures = append(failures, "ParseManifest admitted raw WTF-8 at every arm")
		} else {
			var refusal *terminalbackend.Error
			if !errors.As(err, &refusal) || refusal.Detail != "document encoding" {
				failures = append(failures, "ParseManifest(raw WTF-8) = "+err.Error()+", want the encoding arm")
			}
		}
		if _, err := canonicaljson.Canonicalize([]byte(vector)); err == nil {
			failures = append(failures, "Canonicalize admitted raw WTF-8")
		}
	}
	admitVector("\""+string([]byte{0xed, 0x9f, 0xbf})+"\"", "U+D7FF raw")
	for _, vector := range []string{
		`"` + slash + `n` + string([]byte{0xed, 0xa0, 0x80}) + `"`,
		`"` + slash + slash + string([]byte{0xed, 0xbf, 0xbf}) + `"`,
	} {
		_, err := terminalbackend.ParseManifest(embed(vector))
		if err == nil {
			failures = append(failures, "ParseManifest admitted "+vector+" at every arm")
		} else {
			var refusal *terminalbackend.Error
			if !errors.As(err, &refusal) || refusal.Detail != "document encoding" {
				failures = append(failures, "ParseManifest("+vector+") = "+err.Error()+", want the encoding arm")
			}
		}
		if _, err := canonicaljson.Canonicalize([]byte(vector)); err == nil {
			failures = append(failures, "Canonicalize admitted "+vector)
		}
	}

	if len(failures) > 0 {
		shown := failures
		if len(shown) > 10 {
			shown = shown[:10]
		}
		t.Fatalf("%d agreement failures (showing %d):\n%s", len(failures), len(shown), strings.Join(shown, "\n"))
	}
}
