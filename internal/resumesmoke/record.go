package resumesmoke

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file owns the closed Native Resume Smoke record shape:
// urn:ax:schema:native-resume-smoke 1.0.0. The record is the bounded
// evidence one smoke run emits: the claimed tuple, the Section 8.4
// native-resume cell, the verdict, digests of every validated
// adapter byte the run depended on, the store root, every check with
// its outcome, and timestamps. It carries no secrets and no raw
// native references: native session IDs, argv, and environment appear
// only inside the digested bodies, never as members.
//
// Integrity is a self-digest in the record_id member: the SHA-256 of
// the canonical bytes with the record_id value replaced by a fixed
// placeholder, following the provider-identity creation pattern.
// Verification decodes the closed shape, enforces every member
// constraint, compares the omit-self digest, and then applies the
// verdict consistency rule. A tampered byte, a flipped verdict, a
// whitespace variant, and a recomputed digest over an inconsistent
// check list all refuse; only a forgery that recomputes both the
// digest and a consistent list verifies, which is the stated
// local-evidence residual.

// RecordSchema is the exact schema identifier every smoke record carries.
const RecordSchema = "urn:ax:schema:native-resume-smoke"

// RecordSchemaVersion is the only smoke record version this framework reads.
const RecordSchemaVersion = "1.0.0"

// recordPlaceholder stands in for record_id while the omit-self
// digest is computed. The substitution below matches the full
// `"record_id":"<placeholder>"` frame, never the bare digest.
const recordPlaceholder = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

// Outcome is one check outcome: the check passed, failed, or was
// skipped with its detail naming the gate or the absent input.
type Outcome string

// The closed check-outcome vocabulary.
const (
	OutcomePass    Outcome = "pass"
	OutcomeFail    Outcome = "fail"
	OutcomeSkipped Outcome = "skipped"
)

// Verdict is one smoke verdict: every check passed, a check failed,
// the tuple is conditional and its resume plan stayed gated, or the
// tuple is unsupported or unknown and was refused. Unsupported and
// unknown are distinct verdicts because Section 8.1 forbids
// rewriting unknown as unsupported.
type Verdict string

// The closed verdict vocabulary.
const (
	VerdictPass        Verdict = "pass"
	VerdictFail        Verdict = "fail"
	VerdictGated       Verdict = "gated"
	VerdictUnsupported Verdict = "unsupported"
	VerdictUnknown     Verdict = "unknown"
)

// Check is one recorded smoke check: its registry name, its outcome,
// and a detail naming the gate, the refusal, or the evidence. The
// detail carries digests and member names only, never native
// references or secrets.
type Check struct {
	Name    string  `json:"name"`
	Outcome Outcome `json:"outcome"`
	Detail  string  `json:"detail"`
}

// Tuple is the exact claimed provider build one smoke run checks:
// the Section 7.4 probe members provider_id, provider_version,
// platform, and architecture.
type Tuple struct {
	ProviderID      string `json:"provider_id"`
	ProviderVersion string `json:"provider_version"`
	Platform        string `json:"platform"`
	Architecture    string `json:"architecture"`
}

// Record is one decoded Native Resume Smoke record. Nullable digest
// members are empty exactly when the corresponding check never ran:
// a refused tuple has no probe to digest, and a backend-only
// provider has no store root.
type Record struct {
	Schema               string  `json:"schema"`
	SchemaVersion        string  `json:"schema_version"`
	RecordID             string  `json:"record_id"`
	Tuple                Tuple   `json:"tuple"`
	ResumeCell           Cell    `json:"resume_cell"`
	GateRef              string  `json:"gate_ref"`
	Verdict              Verdict `json:"verdict"`
	ProbeDigest          string  `json:"probe_digest"`
	StoreRoot            string  `json:"store_root"`
	DiscoveryProofDigest string  `json:"discovery_proof_digest"`
	IdentityRecordID     string  `json:"identity_record_id"`
	SpawnPlanDigest      string  `json:"spawn_plan_digest"`
	QuiescenceDigest     string  `json:"quiescence_digest"`
	Checks               []Check `json:"checks"`
	StartedAt            string  `json:"started_at"`
	CompletedAt          string  `json:"completed_at"`
}

// recordMembers is the exact required member set of a smoke record.
var recordMembers = map[string]bool{
	"schema":                 true,
	"schema_version":         true,
	"record_id":              true,
	"tuple":                  true,
	"resume_cell":            true,
	"gate_ref":               true,
	"verdict":                true,
	"probe_digest":           true,
	"store_root":             true,
	"discovery_proof_digest": true,
	"identity_record_id":     true,
	"spawn_plan_digest":      true,
	"quiescence_digest":      true,
	"checks":                 true,
	"started_at":             true,
	"completed_at":           true,
}

// encodeRecord renders one record to its canonical bytes and stamps
// the omit-self record_id. The encoding is deterministic: fixed
// member order, no maps, checks in execution order, so identical
// runs emit byte-identical records.
func encodeRecord(record Record) ([]byte, error) {
	record.RecordID = recordPlaceholder
	staged, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("encode smoke record: %w", err)
	}
	sum := sha256.Sum256(staged)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	framed := []byte(`"record_id":"` + recordPlaceholder + `"`)
	claimed := []byte(`"record_id":"` + digest + `"`)
	if !bytes.Contains(staged, framed) {
		return nil, fmt.Errorf("encode smoke record: placeholder frame is absent")
	}
	return bytes.Replace(staged, framed, claimed, 1), nil
}

// VerifyRecord validates one smoke record's bytes and returns the
// decoded record: the closed member set, every member constraint,
// the omit-self digest, and the verdict consistency rule. Anything
// else refuses.
func VerifyRecord(candidate []byte) (Record, error) {
	var empty Record
	decoder := json.NewDecoder(bytes.NewReader(candidate))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return empty, fmt.Errorf("smoke record is not the closed shape: %w", err)
	}
	if decoder.More() {
		return empty, fmt.Errorf("smoke record carries trailing data")
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(candidate, &members); err != nil {
		return empty, fmt.Errorf("smoke record is not an object: %w", err)
	}
	for name := range recordMembers {
		if _, ok := members[name]; !ok {
			return empty, fmt.Errorf("smoke record misses required member %q", name)
		}
	}
	if err := checkRecordMembers(record); err != nil {
		return empty, err
	}
	framed := []byte(`"record_id":"` + record.RecordID + `"`)
	claimed := []byte(`"record_id":"` + recordPlaceholder + `"`)
	if !bytes.Contains(candidate, framed) {
		return empty, fmt.Errorf("smoke record record_id frame is absent")
	}
	staged := bytes.Replace(candidate, framed, claimed, 1)
	sum := sha256.Sum256(staged)
	if "sha256:"+hex.EncodeToString(sum[:]) != record.RecordID {
		return empty, fmt.Errorf("smoke record digest does not match its bytes")
	}
	if err := checkVerdictConsistent(record); err != nil {
		return empty, err
	}
	return record, nil
}

// checkRecordMembers enforces every record member constraint: the
// exact schema, the closed cell, verdict, and outcome vocabularies,
// digest shapes, non-empty citations and timestamps, and the check
// list every verdict requires. Nullable evidence members are empty
// exactly when their check never ran; the verdict rule below decides
// which emptiness each verdict allows.
func checkRecordMembers(record Record) error {
	if record.Schema != RecordSchema {
		return fmt.Errorf("smoke record schema is not the native resume smoke")
	}
	if record.SchemaVersion != RecordSchemaVersion {
		return fmt.Errorf("smoke record schema_version is not 1.0.0")
	}
	if !isDigest(record.RecordID) {
		return fmt.Errorf("smoke record record_id is not a digest")
	}
	if record.Tuple.ProviderID == "" || record.Tuple.ProviderVersion == "" || record.Tuple.Platform == "" || record.Tuple.Architecture == "" {
		return fmt.Errorf("smoke record tuple is incomplete")
	}
	switch record.ResumeCell {
	case CellAvailable, CellConditional, CellUnsupported, CellUnknown:
	default:
		return fmt.Errorf("smoke record resume_cell is not a matrix label")
	}
	if record.GateRef == "" {
		return fmt.Errorf("smoke record gate_ref is empty")
	}
	switch record.Verdict {
	case VerdictPass, VerdictFail, VerdictGated, VerdictUnsupported, VerdictUnknown:
	default:
		return fmt.Errorf("smoke record verdict is not a verdict")
	}
	for _, member := range []struct {
		name  string
		value string
	}{
		{"probe_digest", record.ProbeDigest},
		{"discovery_proof_digest", record.DiscoveryProofDigest},
		{"identity_record_id", record.IdentityRecordID},
		{"spawn_plan_digest", record.SpawnPlanDigest},
		{"quiescence_digest", record.QuiescenceDigest},
	} {
		if member.value != "" && !isDigest(member.value) {
			return fmt.Errorf("smoke record %s is not a digest or empty", member.name)
		}
	}
	if len(record.Checks) == 0 || len(record.Checks) > 7 {
		return fmt.Errorf("smoke record checks are not 1..7 entries")
	}
	for _, check := range record.Checks {
		if check.Name == "" || check.Detail == "" {
			return fmt.Errorf("smoke record check has an empty name or detail")
		}
		switch check.Outcome {
		case OutcomePass, OutcomeFail, OutcomeSkipped:
		default:
			return fmt.Errorf("smoke record check outcome is not pass fail or skipped")
		}
	}
	if _, err := scalar.ParseTimestamp(record.StartedAt); err != nil {
		return fmt.Errorf("smoke record started_at is not a timestamp: %w", err)
	}
	if _, err := scalar.ParseTimestamp(record.CompletedAt); err != nil {
		return fmt.Errorf("smoke record completed_at is not a timestamp: %w", err)
	}
	if record.CompletedAt < record.StartedAt {
		return fmt.Errorf("smoke record completed_at precedes started_at")
	}
	return nil
}

// checkVerdictConsistent requires the stored verdict to equal the
// verdict the cell and checks derive. A promotion that flips the
// verdict without rewriting the check list refuses here even with a
// recomputed digest.
func checkVerdictConsistent(record Record) error {
	if want := deriveVerdict(record.ResumeCell, record.Checks); record.Verdict != want {
		return fmt.Errorf("smoke record verdict %q is inconsistent with its checks", record.Verdict)
	}
	return nil
}

// deriveVerdict computes the run verdict from the cell and the check
// outcomes: unsupported and unknown cells refuse, any failed check
// fails, a conditional cell gates, and only an available cell with
// no failure and no unexpected skip passes. Runs and verification
// share this single rule, so a run can never emit an inconsistent
// record.
func deriveVerdict(cell Cell, checks []Check) Verdict {
	switch cell {
	case CellUnsupported:
		return VerdictUnsupported
	case CellUnknown:
		return VerdictUnknown
	}
	for _, check := range checks {
		if check.Outcome == OutcomeFail {
			return VerdictFail
		}
	}
	if cell == CellConditional {
		return VerdictGated
	}
	for _, check := range checks {
		if check.Outcome == OutcomeSkipped && check.Name != checkQuiescence {
			return VerdictFail
		}
	}
	return VerdictPass
}

// isDigest reports whether the value is a sha256 content digest.
func isDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != 7+64 {
		return false
	}
	for _, character := range value[7:] {
		if character < '0' || (character > '9' && character < 'a') || character > 'f' {
			return false
		}
	}
	return true
}

// digestBytes returns the sha256 content digest of the bytes.
func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
