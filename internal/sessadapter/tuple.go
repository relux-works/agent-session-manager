package sessadapter

import (
	"encoding/json"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// This file validates the Environment Tuple and the Supported
// Environment Tuple Registry entry admission Section 7.8 rests on:
// the probe and every state-bearing call carry an exact six-member
// tuple, and target writes and healthy doctor results require an
// accepted non-revoked signed entry whose key equals the
// host-observed execution facts.
//
// The registry publication rules — detached signature, monotonic
// sequence, validity of the registry envelope itself — fail closed
// at the registry owner, which does not exist yet; this package
// admits single entries against caller-held registry facts and
// states that bound in doc.go.

// tupleMembers is the exact Environment Tuple member set Section 7.8
// requires: environment_id, environment_version, platform,
// architecture, store_schema_fingerprint, and adapter_version. The
// tuple never contains executable provenance, and — unlike every
// neighbouring closed type — it carries no extensions member, so
// even a reverse-DNS extensions object is refused here.
var tupleMembers = map[string]bool{
	"environment_id":           true,
	"environment_version":      true,
	"platform":                 true,
	"architecture":             true,
	"store_schema_fingerprint": true,
	"adapter_version":          true,
}

// tupleRequired lists tupleMembers in a fixed order.
var tupleRequired = []string{
	"environment_id",
	"environment_version",
	"platform",
	"architecture",
	"store_schema_fingerprint",
	"adapter_version",
}

// Tuple is one validated Environment Tuple.
type Tuple struct {
	EnvironmentID    string
	Version          string
	Platform         string
	Architecture     string
	StoreFingerprint string
	AdapterVersion   string
}

// DecodeTuple validates one closed Environment Tuple: exact
// members, the environment-id grammar, environment_version as
// string[1..128] (the bound Section 10.8.1 states for the
// observation that reuses this admission model), the closed
// platform and architecture vocabularies, a digest store
// fingerprint, and a SemVer adapter version. Any executable
// provenance member is an unknown member, never a second identity.
func DecodeTuple(raw json.RawMessage) (Tuple, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("environment tuple "+fault.detail, fault.member)
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	if name, unknown := unknownMember(members, tupleMembers); unknown {
		failure, err := failProtocol("environment tuple carries unknown member", name)
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	if name, missing := missingMember(members, tupleRequired); missing {
		failure, err := failProtocol("environment tuple misses a required member", name)
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	environmentID, ok := rawString(members["environment_id"])
	if !ok || !environmentIDPattern.MatchString(environmentID) {
		failure, err := failProtocol("environment tuple environment identifier is not an environment-id", "environment_id")
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	version, ok := checkStringBounds(members["environment_version"], 1, 128)
	if !ok {
		failure, err := failProtocol("environment tuple environment version is not a string[1..128]", "environment_version")
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	platform, ok := rawString(members["platform"])
	if !ok {
		failure, err := failProtocol("environment tuple platform is not a string", "platform")
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	if _, err := scalar.ParsePlatform(platform); err != nil {
		failure, faultErr := failProtocol("environment tuple platform is outside linux|macos|windows|wsl2", "platform")
		if faultErr != nil {
			return Tuple{}, faultErr
		}
		return Tuple{}, failure
	}
	architecture, ok := rawString(members["architecture"])
	if !ok || !validTupleArchitecture(architecture) {
		failure, err := failProtocol("environment tuple architecture is outside amd64|arm64", "architecture")
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	fingerprint, ok := checkDigest(members["store_schema_fingerprint"])
	if !ok {
		failure, err := failProtocol("environment tuple store fingerprint is not a digest", "store_schema_fingerprint")
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	adapterVersion, ok := rawString(members["adapter_version"])
	if !ok || !semverPattern.MatchString(adapterVersion) {
		failure, err := failProtocol("environment tuple adapter version is not SemVer", "adapter_version")
		if err != nil {
			return Tuple{}, err
		}
		return Tuple{}, failure
	}
	return Tuple{
		EnvironmentID:    environmentID,
		Version:          version,
		Platform:         platform,
		Architecture:     architecture,
		StoreFingerprint: fingerprint.String(),
		AdapterVersion:   adapterVersion,
	}, nil
}

// Direction names one Supported Environment Tuple Registry key
// direction. The registry admits no third direction.
type Direction string

// The closed direction vocabulary.
const (
	DirectionSourceRead  Direction = "source_read"
	DirectionTargetWrite Direction = "target_write"
)

// directionNames is the closed direction vocabulary as a table so
// the census derives it like every other closed vocabulary: the
// three direction gates compare through validDirection, never
// through an inline comparison chain the census cannot see. The
// table is built from the constants above, so the two spellings
// cannot drift.
var directionNames = []string{
	string(DirectionSourceRead),
	string(DirectionTargetWrite),
}

// validDirection reports whether the name is one of the two closed
// registry directions.
func validDirection(name string) bool {
	for _, allowed := range directionNames {
		if name == allowed {
			return true
		}
	}
	return false
}

// tupleEntryStatuses is the two-status registry-entry vocabulary
// Section 13.14 states: accepted plus revoked. Membership is a
// table loop so the census derives it like every other closed
// vocabulary.
var tupleEntryStatuses = []string{
	"accepted",
	"revoked",
}

// validTupleEntryStatus reports whether the status is one of the
// two Section 13.14 registry-entry statuses.
func validTupleEntryStatus(status string) bool {
	for _, allowed := range tupleEntryStatuses {
		if status == allowed {
			return true
		}
	}
	return false
}

// evidenceResults is the one-result evidence vocabulary Sections
// 13.14 states for FixtureEvidence and ResumeSmokeEvidence: only
// pass constructs. It is a table despite its single member so the
// census derives it and the spec pin below covers it like every
// other closed vocabulary.
var evidenceResults = []string{
	"pass",
}

// validEvidenceResult reports whether the result is the single
// Section 13.14 evidence result.
func validEvidenceResult(result string) bool {
	for _, allowed := range evidenceResults {
		if result == allowed {
			return true
		}
	}
	return false
}

// CandidateKind names one registry key candidate kind.
type CandidateKind string

// The closed candidate-kind vocabulary.
const (
	CandidateBuiltin  CandidateKind = "builtin"
	CandidateExternal CandidateKind = "external"
)

// candidateKindNames is the closed candidate-kind vocabulary as a
// table so the census derives it like every other closed
// vocabulary: the registry-key gate compares through
// validCandidateKind, never through an inline comparison chain the
// census cannot see. The table is built from the constants above,
// so the two spellings cannot drift.
var candidateKindNames = []string{
	string(CandidateBuiltin),
	string(CandidateExternal),
}

// validCandidateKind reports whether the kind is one of the two
// closed registry-key candidate kinds.
func validCandidateKind(kind CandidateKind) bool {
	for _, allowed := range candidateKindNames {
		if string(kind) == allowed {
			return true
		}
	}
	return false
}

// projectionStrategies is the closed five-strategy vocabulary
// Section 13.14 admits: archive_only plus the four target
// projection strategies. Entries carry a sorted unique non-empty
// subset.
var projectionStrategies = []string{
	"archive_only",
	"same_environment_native_rewrite",
	"target_native_writer",
	"target_official_import",
	"continuation_context",
}

// validStrategy reports whether the name is a registry strategy.
func validStrategy(name string) bool {
	for _, strategy := range projectionStrategies {
		if strategy == name {
			return true
		}
	}
	return false
}

// fidelityDispositions is the closed seven-disposition vocabulary:
// exact, semantic, summarized, opaque_preserved, synthesized,
// omitted, unrecoverable. archive_only is a strategy, never a
// disposition.
var fidelityDispositions = []string{
	"exact",
	"semantic",
	"summarized",
	"opaque_preserved",
	"synthesized",
	"omitted",
	"unrecoverable",
}

// validDisposition reports whether the name is a fidelity
// disposition.
func validDisposition(name string) bool {
	for _, disposition := range fidelityDispositions {
		if disposition == name {
			return true
		}
	}
	return false
}

// captureClasses is the closed nine-class vocabulary Section 13.14.1
// states for CaptureItem: the classes a capture plan may name. The
// count bound on excluded_classes ([0..9]) is not the vocabulary;
// this list is.
var captureClasses = []string{
	"durable_payload",
	"durable_index_required",
	"durable_sidecar",
	"derived_cache_optional",
	"credential",
	"machine_auth",
	"runtime_state",
	"transient_lock",
	"unknown",
}

// validCaptureClass reports whether the name is a capture class.
func validCaptureClass(name string) bool {
	for _, class := range captureClasses {
		if class == name {
			return true
		}
	}
	return false
}

// tupleArchitectures is the two-architecture tuple vocabulary
// Section 7.8 states. Membership is a table loop so the census
// derives it like every other closed vocabulary.
var tupleArchitectures = []string{
	"amd64",
	"arm64",
}

// validTupleArchitecture reports whether the name is one of the two
// tuple architectures Section 7.8 states.
func validTupleArchitecture(architecture string) bool {
	for _, allowed := range tupleArchitectures {
		if architecture == allowed {
			return true
		}
	}
	return false
}

// tupleKeyMembers is the exact SupportedEnvironmentTupleKey member
// set: direction, environment, provider_id, candidate_kind,
// executable_sha256, provider_manifest_digest, and
// session_adapter_manifest_digest. Status is not part of the unique
// key, so accepted and revoked rows cannot coexist — which is why
// status lives on the entry, not here.
var tupleKeyMembers = map[string]bool{
	"direction":                       true,
	"environment":                     true,
	"provider_id":                     true,
	"candidate_kind":                  true,
	"executable_sha256":               true,
	"provider_manifest_digest":        true,
	"session_adapter_manifest_digest": true,
}

// tupleKeyRequired lists tupleKeyMembers in a fixed order.
var tupleKeyRequired = []string{
	"direction",
	"environment",
	"provider_id",
	"candidate_kind",
	"executable_sha256",
	"provider_manifest_digest",
	"session_adapter_manifest_digest",
}

// TupleKey is one validated registry key.
type TupleKey struct {
	Direction              Direction
	Environment            Tuple
	ProviderID             string
	CandidateKind          CandidateKind
	ExecutableSHA256       string
	ProviderManifestDigest string
	AdapterManifestDigest  string
}

// decodeTupleKey validates one closed registry key.
func decodeTupleKey(raw json.RawMessage) (TupleKey, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("tuple registry key "+fault.detail, fault.member)
		if err != nil {
			return TupleKey{}, err
		}
		return TupleKey{}, failure
	}
	if name, unknown := unknownMember(members, tupleKeyMembers); unknown {
		failure, err := failProtocol("tuple registry key carries unknown member", name)
		if err != nil {
			return TupleKey{}, err
		}
		return TupleKey{}, failure
	}
	if name, missing := missingMember(members, tupleKeyRequired); missing {
		failure, err := failProtocol("tuple registry key misses a required member", name)
		if err != nil {
			return TupleKey{}, err
		}
		return TupleKey{}, failure
	}
	direction, ok := rawString(members["direction"])
	if !ok || !validDirection(direction) {
		failure, err := failProtocol("tuple registry key direction is outside source_read|target_write", "direction")
		if err != nil {
			return TupleKey{}, err
		}
		return TupleKey{}, failure
	}
	environment, err := DecodeTuple(members["environment"])
	if err != nil {
		return TupleKey{}, err
	}
	providerID, ok := rawString(members["provider_id"])
	if !ok || !providerIDPattern.MatchString(providerID) {
		failure, faultErr := failProtocol("tuple registry key provider identifier is not a provider-id", "provider_id")
		if faultErr != nil {
			return TupleKey{}, faultErr
		}
		return TupleKey{}, failure
	}
	candidateKind, ok := rawString(members["candidate_kind"])
	if !ok || !validCandidateKind(CandidateKind(candidateKind)) {
		failure, faultErr := failProtocol("tuple registry key candidate kind is outside builtin|external", "candidate_kind")
		if faultErr != nil {
			return TupleKey{}, faultErr
		}
		return TupleKey{}, failure
	}
	executable, ok := checkDigest(members["executable_sha256"])
	if !ok {
		failure, faultErr := failProtocol("tuple registry key executable digest is not a digest", "executable_sha256")
		if faultErr != nil {
			return TupleKey{}, faultErr
		}
		return TupleKey{}, failure
	}
	providerManifest, ok := checkDigest(members["provider_manifest_digest"])
	if !ok {
		failure, faultErr := failProtocol("tuple registry key provider manifest digest is not a digest", "provider_manifest_digest")
		if faultErr != nil {
			return TupleKey{}, faultErr
		}
		return TupleKey{}, failure
	}
	adapterManifest, ok := checkDigest(members["session_adapter_manifest_digest"])
	if !ok {
		failure, faultErr := failProtocol("tuple registry key adapter manifest digest is not a digest", "session_adapter_manifest_digest")
		if faultErr != nil {
			return TupleKey{}, faultErr
		}
		return TupleKey{}, failure
	}
	return TupleKey{
		Direction:              Direction(direction),
		Environment:            environment,
		ProviderID:             providerID,
		CandidateKind:          CandidateKind(candidateKind),
		ExecutableSHA256:       executable.String(),
		ProviderManifestDigest: providerManifest.String(),
		AdapterManifestDigest:  adapterManifest.String(),
	}, nil
}

// tupleEntryMembers is the exact SupportedEnvironmentTupleEntry
// member set.
var tupleEntryMembers = map[string]bool{
	"key":                   true,
	"entry_sequence":        true,
	"contracts":             true,
	"strategies":            true,
	"fixture_evidence":      true,
	"resume_smoke_evidence": true,
	"known_fidelity_limits": true,
	"valid_from":            true,
	"valid_until":           true,
	"status":                true,
	"revocation_reason":     true,
	"revoked_at":            true,
	"extensions":            true,
}

// tupleEntryRequired lists tupleEntryMembers in a fixed order.
var tupleEntryRequired = []string{
	"key",
	"entry_sequence",
	"contracts",
	"strategies",
	"fixture_evidence",
	"resume_smoke_evidence",
	"known_fidelity_limits",
	"valid_from",
	"valid_until",
	"status",
	"revocation_reason",
	"revoked_at",
	"extensions",
}

// FixtureEvidence is one validated passing fixture record. The
// result member is validated at decode: only pass constructs.
type FixtureEvidence struct {
	SuiteRevision  string
	SuiteDigest    string
	ExecutedAt     time.Time
	EvidenceDigest string
	FixtureCount   uint64
}

// SmokeEvidence is one validated passing resume-smoke record. Only
// pass with a passed bounded continuation turn constructs.
type SmokeEvidence struct {
	ExecutedAt      time.Time
	EvidenceDigest  string
	NativeCLIFamily string
}

// FidelityLimit is one validated known-fidelity-limit row.
type FidelityLimit struct {
	Code               string
	AffectedClass      string
	MaximumDisposition string
	Detail             string
}

// TupleEntry is one validated registry entry.
type TupleEntry struct {
	Key              TupleKey
	EntrySequence    uint64
	Strategies       []string
	Fixture          FixtureEvidence
	Smoke            *SmokeEvidence
	FidelityLimits   []FidelityLimit
	ValidFrom        time.Time
	ValidUntil       time.Time
	Status           string
	RevocationReason *string
	RevokedAt        *time.Time
}

// fixtureEvidenceMembers is the exact FixtureEvidence member set.
var fixtureEvidenceMembers = map[string]bool{
	"suite_revision":  true,
	"suite_digest":    true,
	"result":          true,
	"executed_at":     true,
	"evidence_digest": true,
	"fixture_count":   true,
}

// fixtureEvidenceRequired lists fixtureEvidenceMembers in order.
var fixtureEvidenceRequired = []string{
	"suite_revision",
	"suite_digest",
	"result",
	"executed_at",
	"evidence_digest",
	"fixture_count",
}

// decodeFixtureEvidence validates one closed fixture record with
// result=pass and a positive fixture count.
func decodeFixtureEvidence(raw json.RawMessage) (FixtureEvidence, error) {
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("tuple fixture evidence "+fault.detail, fault.member)
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	if name, unknown := unknownMember(members, fixtureEvidenceMembers); unknown {
		failure, err := failProtocol("tuple fixture evidence carries unknown member", name)
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	if name, missing := missingMember(members, fixtureEvidenceRequired); missing {
		failure, err := failProtocol("tuple fixture evidence misses a required member", name)
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	revision, ok := checkStringBounds(members["suite_revision"], 1, 128)
	if !ok {
		failure, err := failProtocol("tuple fixture suite revision is not a string[1..128]", "suite_revision")
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	suiteDigest, ok := checkDigest(members["suite_digest"])
	if !ok {
		failure, err := failProtocol("tuple fixture suite digest is not a digest", "suite_digest")
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	if result, ok := rawString(members["result"]); !ok || !validEvidenceResult(result) {
		failure, err := failProtocol("tuple fixture result is not pass", "result")
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	executedAt, ok := checkTimestamp(members["executed_at"])
	if !ok {
		failure, err := failProtocol("tuple fixture execution time is not a timestamp", "executed_at")
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	executed, err := executedAt.Time()
	if err != nil {
		failure, faultErr := failProtocol("tuple fixture execution time is not a timestamp", "executed_at")
		if faultErr != nil {
			return FixtureEvidence{}, faultErr
		}
		return FixtureEvidence{}, failure
	}
	evidenceDigest, ok := checkDigest(members["evidence_digest"])
	if !ok {
		failure, err := failProtocol("tuple fixture evidence digest is not a digest", "evidence_digest")
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	count, ok := checkUint53Bounds(members["fixture_count"], 1, maxUint53)
	if !ok {
		failure, err := failProtocol("tuple fixture count is not a uint53 above zero", "fixture_count")
		if err != nil {
			return FixtureEvidence{}, err
		}
		return FixtureEvidence{}, failure
	}
	return FixtureEvidence{
		SuiteRevision:  revision,
		SuiteDigest:    suiteDigest.String(),
		ExecutedAt:     executed,
		EvidenceDigest: evidenceDigest.String(),
		FixtureCount:   count,
	}, nil
}

// smokeEvidenceMembers is the exact ResumeSmokeEvidence member set.
var smokeEvidenceMembers = map[string]bool{
	"result":                           true,
	"executed_at":                      true,
	"evidence_digest":                  true,
	"native_cli_family":                true,
	"bounded_continuation_turn_passed": true,
}

// smokeEvidenceRequired lists smokeEvidenceMembers in order.
var smokeEvidenceRequired = []string{
	"result",
	"executed_at",
	"evidence_digest",
	"native_cli_family",
	"bounded_continuation_turn_passed",
}

// decodeSmokeEvidence validates one closed passing smoke record, or
// reports absence for an explicit null. A missing record and a
// malformed record are different facts: null is absence, anything
// else that does not validate is a protocol error.
func decodeSmokeEvidence(raw json.RawMessage) (*SmokeEvidence, error) {
	if isNull(raw) {
		return nil, nil
	}
	members, fault := decodeStrictObject(bytesTrimSpace(raw))
	if fault != nil {
		failure, err := failProtocol("tuple resume smoke "+fault.detail, fault.member)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, unknown := unknownMember(members, smokeEvidenceMembers); unknown {
		failure, err := failProtocol("tuple resume smoke carries unknown member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if name, missing := missingMember(members, smokeEvidenceRequired); missing {
		failure, err := failProtocol("tuple resume smoke misses a required member", name)
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if result, ok := rawString(members["result"]); !ok || !validEvidenceResult(result) {
		failure, err := failProtocol("tuple resume smoke result is not pass", "result")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	executedAt, ok := checkTimestamp(members["executed_at"])
	if !ok {
		failure, err := failProtocol("tuple resume smoke execution time is not a timestamp", "executed_at")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	executed, err := executedAt.Time()
	if err != nil {
		failure, faultErr := failProtocol("tuple resume smoke execution time is not a timestamp", "executed_at")
		if faultErr != nil {
			return nil, faultErr
		}
		return nil, failure
	}
	evidenceDigest, ok := checkDigest(members["evidence_digest"])
	if !ok {
		failure, err := failProtocol("tuple resume smoke evidence digest is not a digest", "evidence_digest")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	family, ok := checkStringBounds(members["native_cli_family"], 1, 128)
	if !ok {
		failure, err := failProtocol("tuple resume smoke CLI family is not a string[1..128]", "native_cli_family")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	passed, ok := rawBool(members["bounded_continuation_turn_passed"])
	if !ok || !passed {
		failure, err := failProtocol("tuple resume smoke continuation turn did not pass", "bounded_continuation_turn_passed")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	return &SmokeEvidence{
		ExecutedAt:      executed,
		EvidenceDigest:  evidenceDigest.String(),
		NativeCLIFamily: family,
	}, nil
}

// fidelityLimitMembers is the exact FidelityLimit member set.
var fidelityLimitMembers = map[string]bool{
	"code":                true,
	"affected_class":      true,
	"maximum_disposition": true,
	"detail":              true,
}

// fidelityLimitRequired lists fidelityLimitMembers in order.
var fidelityLimitRequired = []string{
	"code",
	"affected_class",
	"maximum_disposition",
	"detail",
}

// decodeFidelityLimits validates the sorted-unique fidelity-limit
// array: shape per row, disposition in the seven-vocabulary, and
// uniqueness over code/class pairs. archive_only is a strategy and
// is refused as a disposition here the same way it is refused in
// required_dispositions.
func decodeFidelityLimits(raw json.RawMessage) ([]FidelityLimit, error) {
	elements, ok := decodeArray(bytesTrimSpace(raw))
	if !ok {
		failure, err := failProtocol("tuple fidelity limits are not an array", "known_fidelity_limits")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if uint64(len(elements)) > 1024 {
		failure, err := failProtocol("tuple fidelity limits exceed 1024 rows", "known_fidelity_limits")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	limits := make([]FidelityLimit, 0, len(elements))
	seen := map[string]bool{}
	for _, element := range elements {
		members, fault := decodeStrictObject(bytesTrimSpace(element))
		if fault != nil {
			failure, err := failProtocol("tuple fidelity limit "+fault.detail, fault.member)
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		if name, unknown := unknownMember(members, fidelityLimitMembers); unknown {
			failure, err := failProtocol("tuple fidelity limit carries unknown member", name)
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		if name, missing := missingMember(members, fidelityLimitRequired); missing {
			failure, err := failProtocol("tuple fidelity limit misses a required member", name)
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		code, ok := checkStringBounds(members["code"], 1, 128)
		if !ok {
			failure, err := failProtocol("tuple fidelity limit code is not a string[1..128]", "code")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		class, ok := checkStringBounds(members["affected_class"], 1, 128)
		if !ok {
			failure, err := failProtocol("tuple fidelity limit class is not a string[1..128]", "affected_class")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		disposition, ok := rawString(members["maximum_disposition"])
		if !ok || !validDisposition(disposition) {
			failure, err := failProtocol("tuple fidelity limit disposition is outside the seven-disposition vocabulary", "maximum_disposition")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		if _, ok := checkStringBounds(members["detail"], 1, 4096); !ok {
			failure, err := failProtocol("tuple fidelity limit detail is not a string[1..4096]", "detail")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		pair := code + "\x00" + class
		if seen[pair] {
			failure, err := failProtocol("tuple fidelity limits repeat a code/class pair", "known_fidelity_limits")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		seen[pair] = true
		limits = append(limits, FidelityLimit{
			Code:               code,
			AffectedClass:      class,
			MaximumDisposition: disposition,
			Detail:             mustRawString(members["detail"]),
		})
	}
	for index := 1; index < len(limits); index++ {
		previous := limits[index-1].Code + "\x00" + limits[index-1].AffectedClass
		current := limits[index].Code + "\x00" + limits[index].AffectedClass
		if previous >= current {
			failure, err := failProtocol("tuple fidelity limits are not sorted unique by code/class", "known_fidelity_limits")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
	}
	return limits, nil
}

// mustRawString renders a member already validated as a string. It
// is only called on members a bounds check accepted above.
func mustRawString(raw json.RawMessage) string {
	value, _ := rawString(raw)
	return value
}

// decodeStrategies validates the sorted unique non-empty strategy
// subset: every name in the five-vocabulary, strictly ordered, no
// repeats.
func decodeStrategies(raw json.RawMessage) ([]string, error) {
	elements, ok := decodeArray(bytesTrimSpace(raw))
	if !ok {
		failure, err := failProtocol("tuple strategies are not an array", "strategies")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	if len(elements) == 0 {
		failure, err := failProtocol("tuple strategies are empty", "strategies")
		if err != nil {
			return nil, err
		}
		return nil, failure
	}
	strategies := make([]string, 0, len(elements))
	for _, element := range elements {
		name, ok := rawString(element)
		if !ok || !validStrategy(name) {
			failure, err := failProtocol("tuple strategy is outside the five-strategy vocabulary", "strategies")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
		strategies = append(strategies, name)
	}
	for index := 1; index < len(strategies); index++ {
		if strategies[index-1] >= strategies[index] {
			failure, err := failProtocol("tuple strategies are not sorted unique", "strategies")
			if err != nil {
				return nil, err
			}
			return nil, failure
		}
	}
	return strategies, nil
}

// checkContractsShape validates the contract-version array shape:
// 1..64 rows, each with a string contract_id and 1..32 sorted
// unique SemVer versions. Allowed-ID membership is a stated bound:
// the allowed identifiers are prose names in Section 13.14 whose
// mapping to contract URNs belongs to the registry publisher
// check, which has no owner yet.
func checkContractsShape(raw json.RawMessage) error {
	elements, ok := decodeArray(bytesTrimSpace(raw))
	if !ok {
		failure, err := failProtocol("tuple contracts are not an array", "contracts")
		if err != nil {
			return err
		}
		return failure
	}
	if len(elements) == 0 || len(elements) > 64 {
		failure, err := failProtocol("tuple contracts are not 1..64 rows", "contracts")
		if err != nil {
			return err
		}
		return failure
	}
	seen := map[string]bool{}
	previous := ""
	for _, element := range elements {
		members, fault := decodeStrictObject(bytesTrimSpace(element))
		if fault != nil {
			failure, err := failProtocol("tuple contract row "+fault.detail, fault.member)
			if err != nil {
				return err
			}
			return failure
		}
		if len(members) != 2 {
			failure, err := failProtocol("tuple contract row is not exactly contract_id and versions", "contracts")
			if err != nil {
				return err
			}
			return failure
		}
		identifier, ok := members["contract_id"]
		if !ok {
			failure, err := failProtocol("tuple contract row misses contract_id", "contracts")
			if err != nil {
				return err
			}
			return failure
		}
		contractID, ok := checkStringBounds(identifier, 1, 256)
		if !ok {
			failure, err := failProtocol("tuple contract identifier is not a string[1..256]", "contract_id")
			if err != nil {
				return err
			}
			return failure
		}
		versionsRaw, ok := members["versions"]
		if !ok {
			failure, err := failProtocol("tuple contract row misses versions", "contracts")
			if err != nil {
				return err
			}
			return failure
		}
		versions, ok := decodeArray(bytesTrimSpace(versionsRaw))
		if !ok || len(versions) == 0 || len(versions) > 32 {
			failure, err := failProtocol("tuple contract versions are not 1..32 entries", "versions")
			if err != nil {
				return err
			}
			return failure
		}
		previousVersion := ""
		for _, versionRaw := range versions {
			version, ok := rawString(versionRaw)
			if !ok || !semverPattern.MatchString(version) {
				failure, err := failProtocol("tuple contract version is not SemVer", "versions")
				if err != nil {
					return err
				}
				return failure
			}
			if previousVersion >= version && previousVersion != "" {
				failure, err := failProtocol("tuple contract versions are not sorted unique", "versions")
				if err != nil {
					return err
				}
				return failure
			}
			previousVersion = version
		}
		if seen[contractID] {
			failure, err := failProtocol("tuple contracts repeat a contract identifier", "contracts")
			if err != nil {
				return err
			}
			return failure
		}
		seen[contractID] = true
		if previous >= contractID && previous != "" {
			failure, err := failProtocol("tuple contracts are not sorted unique by contract identifier", "contracts")
			if err != nil {
				return err
			}
			return failure
		}
		previous = contractID
	}
	return nil
}

// DecodeTupleEntry validates one closed Supported Environment Tuple
// Registry entry: the exact members, the closed key, a positive
// entry sequence, the contract shape, the strategy subset, passing
// fixture evidence, smoke coherence (source entries carry null,
// target entries carry a passing record — checked at admission, not
// here), the fidelity-limit rows, the validity interval order, and
// the status/revocation coherence (accepted pairs null revocation
// members, revoked pairs both non-null).
func DecodeTupleEntry(body []byte) (TupleEntry, error) {
	members, fault := decodeStrictObject(body)
	if fault != nil {
		failure, err := failProtocol("tuple registry entry "+fault.detail, fault.member)
		if err != nil {
			return TupleEntry{}, err
		}
		return TupleEntry{}, failure
	}
	if name, unknown := unknownMember(members, tupleEntryMembers); unknown {
		failure, err := failProtocol("tuple registry entry carries unknown member", name)
		if err != nil {
			return TupleEntry{}, err
		}
		return TupleEntry{}, failure
	}
	if name, missing := missingMember(members, tupleEntryRequired); missing {
		failure, err := failProtocol("tuple registry entry misses a required member", name)
		if err != nil {
			return TupleEntry{}, err
		}
		return TupleEntry{}, failure
	}
	key, err := decodeTupleKey(members["key"])
	if err != nil {
		return TupleEntry{}, err
	}
	sequence, ok := checkUint53Bounds(members["entry_sequence"], 1, maxUint53)
	if !ok {
		failure, faultErr := failProtocol("tuple entry sequence is not a uint53 above zero", "entry_sequence")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	if err := checkContractsShape(members["contracts"]); err != nil {
		return TupleEntry{}, err
	}
	strategies, err := decodeStrategies(members["strategies"])
	if err != nil {
		return TupleEntry{}, err
	}
	fixture, err := decodeFixtureEvidence(members["fixture_evidence"])
	if err != nil {
		return TupleEntry{}, err
	}
	smoke, err := decodeSmokeEvidence(members["resume_smoke_evidence"])
	if err != nil {
		return TupleEntry{}, err
	}
	limits, err := decodeFidelityLimits(members["known_fidelity_limits"])
	if err != nil {
		return TupleEntry{}, err
	}
	validFrom, ok := checkTimestamp(members["valid_from"])
	if !ok {
		failure, faultErr := failProtocol("tuple entry valid_from is not a timestamp", "valid_from")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	from, err := validFrom.Time()
	if err != nil {
		failure, faultErr := failProtocol("tuple entry valid_from is not a timestamp", "valid_from")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	validUntil, ok := checkTimestamp(members["valid_until"])
	if !ok {
		failure, faultErr := failProtocol("tuple entry valid_until is not a timestamp", "valid_until")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	until, err := validUntil.Time()
	if err != nil {
		failure, faultErr := failProtocol("tuple entry valid_until is not a timestamp", "valid_until")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	if !from.Before(until) {
		failure, faultErr := failProtocol("tuple entry validity interval does not order valid_from before valid_until", "valid_until")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	status, ok := rawString(members["status"])
	if !ok || !validTupleEntryStatus(status) {
		failure, faultErr := failProtocol("tuple entry status is outside accepted|revoked", "status")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	var reason *string
	if !isNull(members["revocation_reason"]) {
		text, ok := checkStringBounds(members["revocation_reason"], 1, 4096)
		if !ok {
			failure, faultErr := failProtocol("tuple entry revocation reason is not a string[1..4096]", "revocation_reason")
			if faultErr != nil {
				return TupleEntry{}, faultErr
			}
			return TupleEntry{}, failure
		}
		reason = &text
	}
	var revokedAt *time.Time
	if !isNull(members["revoked_at"]) {
		stamp, ok := checkTimestamp(members["revoked_at"])
		if !ok {
			failure, faultErr := failProtocol("tuple entry revocation time is not a timestamp", "revoked_at")
			if faultErr != nil {
				return TupleEntry{}, faultErr
			}
			return TupleEntry{}, failure
		}
		instant, err := stamp.Time()
		if err != nil {
			failure, faultErr := failProtocol("tuple entry revocation time is not a timestamp", "revoked_at")
			if faultErr != nil {
				return TupleEntry{}, faultErr
			}
			return TupleEntry{}, failure
		}
		revokedAt = &instant
	}
	if status == "accepted" {
		if reason != nil || revokedAt != nil {
			failure, faultErr := failProtocol("tuple accepted entry carries revocation members", "status")
			if faultErr != nil {
				return TupleEntry{}, faultErr
			}
			return TupleEntry{}, failure
		}
	} else {
		if reason == nil || revokedAt == nil {
			failure, faultErr := failProtocol("tuple revoked entry misses a revocation member", "status")
			if faultErr != nil {
				return TupleEntry{}, faultErr
			}
			return TupleEntry{}, failure
		}
	}
	if !checkExtensions(members["extensions"]) {
		failure, faultErr := failProtocol("tuple registry entry extensions are not reverse-DNS keyed", "extensions")
		if faultErr != nil {
			return TupleEntry{}, faultErr
		}
		return TupleEntry{}, failure
	}
	return TupleEntry{
		Key:              key,
		EntrySequence:    sequence,
		Strategies:       strategies,
		Fixture:          fixture,
		Smoke:            smoke,
		FidelityLimits:   limits,
		ValidFrom:        from,
		ValidUntil:       until,
		Status:           status,
		RevocationReason: reason,
		RevokedAt:        revokedAt,
	}, nil
}

// BindingFacts are the host-observed execution facts the registry
// key must equal: provider, candidate kind, and the three digests.
// The host constructs them from the verified candidate and the
// Journal binding, never from adapter claims.
type BindingFacts struct {
	ProviderID             string
	CandidateKind          CandidateKind
	ExecutableSHA256       string
	ProviderManifestDigest string
	AdapterManifestDigest  string
}

// CheckTupleAdmission decides whether one validated entry admits a
// call in the given direction at the given instant: the key equals
// the host-observed binding facts and the probed tuple exactly, the
// status is accepted, the instant falls inside the validity
// interval, and the direction rules hold — source_read admits only
// strategies=[archive_only] with null smoke, target_write admits
// only a non-archive strategy set with a passing smoke record. A
// revoked or absent entry, a key mismatch, a stale interval, or a
// direction violation is unsupported_environment_tuple, never a
// fallback permission. Local policy may further deny an admitted
// entry but cannot admit a refused one.
func CheckTupleAdmission(entry TupleEntry, direction Direction, environment Tuple, binding BindingFacts, now time.Time) error {
	if entry.Key.Direction != direction {
		failure, err := failUnsupportedTuple("tuple entry direction does not match the call direction", environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	if entry.Key.Environment != environment {
		failure, err := failUnsupportedTuple("tuple entry environment does not equal the probed tuple", environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	if entry.Key.ProviderID != binding.ProviderID ||
		entry.Key.CandidateKind != binding.CandidateKind ||
		entry.Key.ExecutableSHA256 != binding.ExecutableSHA256 ||
		entry.Key.ProviderManifestDigest != binding.ProviderManifestDigest ||
		entry.Key.AdapterManifestDigest != binding.AdapterManifestDigest {
		failure, err := failUnsupportedTuple("tuple entry key does not equal the host-observed execution binding", environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	if entry.Status != "accepted" {
		failure, err := failUnsupportedTuple("tuple entry is not accepted", environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	if now.Before(entry.ValidFrom) || now.After(entry.ValidUntil) {
		failure, err := failUnsupportedTuple("tuple entry validity interval does not cover the call", environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	if direction == DirectionSourceRead {
		if len(entry.Strategies) != 1 || entry.Strategies[0] != "archive_only" {
			failure, err := failUnsupportedTuple("tuple source entry does not carry exactly strategies=[archive_only]", environment.EnvironmentID)
			if err != nil {
				return err
			}
			return failure
		}
		if entry.Smoke != nil {
			failure, err := failUnsupportedTuple("tuple source entry carries resume smoke evidence", environment.EnvironmentID)
			if err != nil {
				return err
			}
			return failure
		}
		return nil
	}
	for _, strategy := range entry.Strategies {
		if strategy == "archive_only" {
			failure, err := failUnsupportedTuple("tuple target entry admits archive_only", environment.EnvironmentID)
			if err != nil {
				return err
			}
			return failure
		}
	}
	if entry.Smoke == nil {
		failure, err := failUnsupportedTuple("tuple target entry misses passing resume smoke evidence", environment.EnvironmentID)
		if err != nil {
			return err
		}
		return failure
	}
	return nil
}
