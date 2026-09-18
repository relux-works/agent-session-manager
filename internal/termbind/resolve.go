package termbind

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

// Evidence bounds from Section 5.2: evidence_ids is sorted unique
// digest[1..256].
const (
	minEvidenceIDs = 1
	maxEvidenceIDs = 256
)

// EventTuple is the backend identity a v4 terminal event carries: the
// admitted backend ID with the implementation and protocol versions every
// resolved evidence object must bind to.
type EventTuple struct {
	BackendID             string
	ImplementationVersion string
	ProtocolVersion       string
}

// EvidenceUniverse is the modeled host-local evidence store: evidence IDs
// resolve through it to raw candidate documents. A production caller backs
// it with validated local Manifest, Probe, and Capability Evidence
// objects; tests model resolution, mismatch, and hostile documents through
// it.
type EvidenceUniverse interface {
	// Lookup returns the raw document the evidence ID names. Found is
	// false when the ID resolves to nothing locally.
	Lookup(id string) ([]byte, bool)
}

// ResolvedEvidence is the admitted resolution of one event's evidence
// IDs: the single Manifest, the single Probe, the Capability Evidence
// objects, and the landed admission verdict over the set.
type ResolvedEvidence struct {
	Manifest terminalbackend.Manifest
	Probe    terminalbackend.Probe
	Evidence []terminalbackend.Evidence
	Admitted terminalbackend.Admitted
}

// ResolveEvidence resolves one v4 event's evidence IDs locally to
// validated Manifest, Probe, and Capability Evidence objects bound to the
// event's backend tuple, and admits the set through the landed registry.
// Every resolution failure refuses terminal_backend_manifest_probe_mismatch:
// an unresolvable ID, a document of any other kind (a native reference, a
// generation string, a socket, a pipe, an endpoint, a token, a credential,
// terminal output, a PID/handle, or any live-process fact), a malformed
// document, an ID that does not name the returned document, a document
// bound to another backend or version, or a set the landed admission
// refuses. The three wire-shape pre-check arms (count bounds, digest
// grammar, sorted-unique order) return uncoded errors instead: defence in
// depth ahead of the canonical owner, which re-enforces the identical
// shape at the append boundary and remains the shape authority.
//
// The resolution carries one stated contract where the specification pins
// the requirement but not the cardinality: the resolved set holds exactly
// one Manifest and exactly one Probe, the rest Capability Evidence. The
// event tuple names one backend/version pair, so an ambiguous set (zero or
// two manifests, zero or two probes) fails closed rather than selecting.
// As a consequence a lone evidence ID passes the wire shape but never
// resolves: one digest cannot name all three objects.
//
// The evidence_ids wire shape (sorted unique digest[1..256]) is pre-checked
// here in three arms so each bound carries its own narrowing mutant; the
// canonical owner re-enforces the identical shape at the append boundary
// (validateTerminalV4Payload) and remains the shape authority.
func ResolveEvidence(ids []string, tuple EventTuple, registry *terminalbackend.Registry, universe EvidenceUniverse, rawGeneration string, now time.Time, verify terminalbackend.SignatureVerifier) (ResolvedEvidence, error) {
	empty := ResolvedEvidence{}
	if universe == nil {
		return empty, mismatchRefusal("evidence universe")
	}
	if err := checkEvidenceIDs(ids); err != nil {
		return empty, err
	}
	var manifestRaw, probeRaw []byte
	var manifestFound, probeFound bool
	var evidenceRaws [][]byte
	for _, id := range ids {
		raw, found := universe.Lookup(id)
		if !found {
			return empty, mismatchRefusal("evidence unresolvable")
		}
		kind, err := evidenceSchema(raw)
		if err != nil {
			return empty, err
		}
		switch kind {
		case terminalbackend.SchemaManifest:
			if manifestFound {
				return empty, mismatchRefusal("evidence manifest")
			}
			manifestRaw, manifestFound = raw, true
		case terminalbackend.SchemaProbe:
			if probeFound {
				return empty, mismatchRefusal("evidence probe")
			}
			probeRaw, probeFound = raw, true
		default:
			evidenceRaws = append(evidenceRaws, raw)
		}
	}
	if !manifestFound || !probeFound {
		return empty, mismatchRefusal("evidence set")
	}
	manifest, err := terminalbackend.ParseManifest(manifestRaw)
	if err != nil {
		return empty, err
	}
	probe, err := terminalbackend.ParseProbe(probeRaw)
	if err != nil {
		return empty, err
	}
	evidence := make([]terminalbackend.Evidence, 0, len(evidenceRaws))
	for _, raw := range evidenceRaws {
		object, err := terminalbackend.ParseEvidence(raw)
		if err != nil {
			return empty, err
		}
		evidence = append(evidence, object)
	}
	if err := checkResolvedIdentity(ids, manifest, probe, evidence); err != nil {
		return empty, err
	}
	if err := checkEventBinding(tuple, manifest, probe, evidence); err != nil {
		return empty, err
	}
	admitted, err := registry.AdmitProbe(manifestRaw, probeRaw, evidenceRaws, rawGeneration, now, verify)
	if err != nil {
		return empty, err
	}
	return ResolvedEvidence{Manifest: manifest, Probe: probe, Evidence: evidence, Admitted: admitted}, nil
}

// checkEvidenceIDs pre-checks the evidence_ids wire shape: sorted unique
// digest[1..256]. Digest grammar delegates to scalar; the count bounds are
// the Section 5.2 literals; ordering is strict bytewise increase, the same
// rule the landed sorted-unique checks enforce.
func checkEvidenceIDs(ids []string) error {
	if len(ids) < minEvidenceIDs || len(ids) > maxEvidenceIDs {
		return fmt.Errorf("evidence ids number %d, want 1..256", len(ids))
	}
	previous := ""
	for index, id := range ids {
		if _, err := scalar.ParseDigest(id); err != nil {
			return fmt.Errorf("evidence id is not a digest: %w", err)
		}
		if index > 0 && id <= previous {
			return fmt.Errorf("evidence ids are not sorted unique")
		}
		previous = id
	}
	return nil
}

// evidenceSchema classifies one resolved document by its schema literal.
// Only the three evidence schemas resolve; any other schema (or any bytes
// that are not a strict object, such as a bare PID, token, or terminal
// output) refuses the mismatch class, never a parse of the foreign bytes.
func evidenceSchema(raw []byte) (string, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return "", mismatchRefusal("evidence kind")
	}
	encoded, known := members["schema"]
	if !known {
		return "", mismatchRefusal("evidence kind")
	}
	var schema string
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if err := decoder.Decode(&schema); err != nil {
		return "", mismatchRefusal("evidence kind")
	}
	switch schema {
	case terminalbackend.SchemaManifest, terminalbackend.SchemaProbe, terminalbackend.SchemaCapabilityEvidence:
		return schema, nil
	default:
		return "", mismatchRefusal("evidence kind")
	}
}

// checkResolvedIdentity requires every resolved document's self ID to equal
// the evidence ID that named it: a universe that returns document X for ID
// Y fails closed instead of admitting the substitution.
func checkResolvedIdentity(ids []string, manifest terminalbackend.Manifest, probe terminalbackend.Probe, evidence []terminalbackend.Evidence) error {
	named := make(map[string]bool, len(ids))
	for _, id := range ids {
		named[id] = true
	}
	if !named[manifest.ManifestID] || !named[probe.ProbeID] {
		return mismatchRefusal("evidence identity")
	}
	for _, object := range evidence {
		if !named[object.EvidenceID] {
			return mismatchRefusal("evidence identity")
		}
	}
	return nil
}

// checkEventBinding requires every resolved object to bind the event's
// backend tuple: backend ID and implementation version everywhere, and the
// protocol version on the Probe and Capability Evidence. Manifest protocol
// membership (the event protocol is one of the Manifest's protocol
// versions) is enforced by the landed admission the resolution finishes
// through, not re-derived here.
func checkEventBinding(tuple EventTuple, manifest terminalbackend.Manifest, probe terminalbackend.Probe, evidence []terminalbackend.Evidence) error {
	if manifest.TerminalBackendID != tuple.BackendID || manifest.ImplementationVersion != tuple.ImplementationVersion {
		return mismatchRefusal("evidence binding")
	}
	if probe.TerminalBackendID != tuple.BackendID || probe.ImplementationVersion != tuple.ImplementationVersion || probe.ProtocolVersion != tuple.ProtocolVersion {
		return mismatchRefusal("evidence binding")
	}
	for _, object := range evidence {
		if object.TerminalBackendID != tuple.BackendID || object.ImplementationVersion != tuple.ImplementationVersion || object.ProtocolVersion != tuple.ProtocolVersion {
			return mismatchRefusal("evidence binding")
		}
	}
	return nil
}

// mismatchRefusal builds the Section 4.B evidence refusal: any failed
// evidence binding, coverage, or resolution rule.
func mismatchRefusal(detail string) error {
	return &terminalbackend.Error{Code: terminalbackend.CodeMismatch, Detail: detail}
}
