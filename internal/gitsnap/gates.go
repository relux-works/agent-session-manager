package gitsnap

// Code is one diagnostic refusal code. The values reuse the specification
// vocabulary so a later CLI owner can map them into axerror envelopes
// without re-deciding the classification. This package mints no axerror
// object; see doc.go.
type Code string

// Closed refusal-code vocabulary. Every refuse call site names one of
// these through its Gate row; the census test derives the table from
// this source so a new code without a row reddens there instead of
// silently widening the vocabulary.
const (
	CodeWorkspaceConflict     Code = "workspace_conflict"
	CodeIncompatibleSchema    Code = "incompatible_schema"
	CodeCapabilityUnavailable Code = "capability_unavailable"
)

// codeNames is the closed code vocabulary as a table so the census
// derives it like every other closed vocabulary in this repository:
// gates compare through these constants, never through inline strings
// the census cannot see.
var codeNames = []Code{
	CodeWorkspaceConflict,
	CodeIncompatibleSchema,
	CodeCapabilityUnavailable,
}

// Gate is one row of the refusal registry: the production func that owns
// the check, the code it refuses with, and the specification clause that
// requires it. Name values are referenced directly by refuse call sites
// (refuse(GateXxx, ...)) so the census in census_test.go can count them.
type Gate struct {
	Name   string
	Code   Code
	Clause string
}

// Closed gate registry. One row per refusal gate in this package; the
// census test asserts every literal refuse(Gate...) call site in
// non-test production files names a row here and every row is driven by
// its named proving test.
var gateRegistry = []Gate{
	{Name: "GateNotRepository", Code: CodeWorkspaceConflict, Clause: "12.3 capture reads Git state; outside a repository there is no HEAD, index, or worktree to read"},
	{Name: "GateHeadCorrupt", Code: CodeIncompatibleSchema, Clause: "10.4 GitHead: branch has oid+ref, detached has oid only, unborn has refs/heads/ ref only"},
	{Name: "GateHeadRef", Code: CodeIncompatibleSchema, Clause: "10.4 git-ref is a 1-1024 byte fully qualified ref; TM-GIT-N3 detached HEAD must not carry a ref"},
	{Name: "GateHeadOIDFormat", Code: CodeIncompatibleSchema, Clause: "10.4 git-oid prefix must match features.object_format"},
	{Name: "GateRemotesRange", Code: CodeIncompatibleSchema, Clause: "10.4 remotes:GitRemote[1..16] sorted by name without duplicates"},
	{Name: "GateRemoteURL", Code: CodeIncompatibleSchema, Clause: "10.4 remote name:string[1..128]; 12.2 remote URLs MUST be sanitized"},
	{Name: "GateIdentityLength", Code: CodeIncompatibleSchema, Clause: "10.4 repository_identity:string[1..256] in characters"},
	{Name: "GateIndexVersion", Code: CodeIncompatibleSchema, Clause: "10.4 GitIndex version:2|3|4"},
	{Name: "GateIndexStage", Code: CodeIncompatibleSchema, Clause: "10.4 GitIndexEntry stage:uint8[0..3]; TM-GIT-N2 stage 4 is refused"},
	{Name: "GateIndexEntry", Code: CodeIncompatibleSchema, Clause: "10.4 GitIndexEntry closed members: path, mode:uint32, oid:git-oid, four flags"},
	{Name: "GateIndexSort", Code: CodeIncompatibleSchema, Clause: "10.4 entries sorted by path then stage, strictly increasing"},
	{Name: "GateIndexEntriesRange", Code: CodeCapabilityUnavailable, Clause: "10.4 entries[0..65536]; larger lists partition into child manifests owned by the manifest leaf"},
	{Name: "GateUpstreamRef", Code: CodeIncompatibleSchema, Clause: "10.4 upstream_ref:git-ref|null"},
	{Name: "GateDeltaStatus", Code: CodeIncompatibleSchema, Clause: "12.1 staged versus unstaged content stays distinct; unknown diff statuses have no member"},
	{Name: "GateWorktreeKind", Code: CodeCapabilityUnavailable, Clause: "12.1 tracked state is represented; devices, FIFOs, and sockets are unsupported"},
	{Name: "GateConsistency", Code: CodeWorkspaceConflict, Clause: "12.3 fail when HEAD, index, or included file digests change during capture"},
	{Name: "GateFeatures", Code: CodeIncompatibleSchema, Clause: "10.4 required_filter_names sorted unique string[0..64]; 12.3 unsupported Git source modes fail closed"},
	{Name: "GateCapturePolicy", Code: CodeCapabilityUnavailable, Clause: "12.1 ignored includes require non-secret classification; 16.2 exclusions"},
	{Name: "GateContentRead", Code: CodeWorkspaceConflict, Clause: "12.3 unreadable content cannot establish capture"},
	{Name: "GateContentPath", Code: CodeCapabilityUnavailable, Clause: "10.4 paths and symlinks stay within root"},
	{Name: "GateSubmoduleState", Code: CodeIncompatibleSchema, Clause: "10.4 recursive submodule state, stage-0 pointers, 16 depth and 256 total"},
	{Name: "GateBlobLimit", Code: CodeCapabilityUnavailable, Clause: "10.2 one blob is at most 128 GiB"},
	{Name: "GateBlobInstall", Code: CodeWorkspaceConflict, Clause: "10.2 verified immutable blob installation before publication"},
	{Name: "GateTransferObjects", Code: CodeIncompatibleSchema, Clause: "10.4 repository-local pack inventory and raw/logical index agreement"},
	{Name: "GateManifestClosure", Code: CodeIncompatibleSchema, Clause: "10.4 and 12.1 transitive workspace tree and blob descriptor closure"},
}

// Refusal is the error Capture returns when a gate fires. Code carries
// the diagnostic classification, Gate names the registry row, and Detail
// carries the offending value without blob contents.
type Refusal struct {
	Code   Code
	Gate   string
	Detail string
}

func (refusal *Refusal) Error() string {
	return "gitsnap: " + string(refusal.Code) + " at " + refusal.Gate + ": " + refusal.Detail
}

// refuse builds the refusal for one registry gate. It is always invoked
// directly as refuse(Gate..., ...) so the census instrument keeps its
// literal call-site denominator; see doc.go for the aliasing bound.
func refuse(gate Gate, detail string) *Refusal {
	return &Refusal{Code: gate.Code, Gate: gate.Name, Detail: detail}
}

// GateNotRepository ... GateConsistency name the registry rows as
// package-level values so call sites pass them by identifier.
var (
	GateNotRepository     = gateRegistry[0]
	GateHeadCorrupt       = gateRegistry[1]
	GateHeadRef           = gateRegistry[2]
	GateHeadOIDFormat     = gateRegistry[3]
	GateRemotesRange      = gateRegistry[4]
	GateRemoteURL         = gateRegistry[5]
	GateIdentityLength    = gateRegistry[6]
	GateIndexVersion      = gateRegistry[7]
	GateIndexStage        = gateRegistry[8]
	GateIndexEntry        = gateRegistry[9]
	GateIndexSort         = gateRegistry[10]
	GateIndexEntriesRange = gateRegistry[11]
	GateUpstreamRef       = gateRegistry[12]
	GateDeltaStatus       = gateRegistry[13]
	GateWorktreeKind      = gateRegistry[14]
	GateConsistency       = gateRegistry[15]
	GateFeatures          = gateRegistry[16]
)

var (
	GateCapturePolicy  = gateRegistry[17]
	GateContentRead    = gateRegistry[18]
	GateContentPath    = gateRegistry[19]
	GateSubmoduleState = gateRegistry[20]
	GateBlobLimit      = gateRegistry[21]
	GateBlobInstall    = gateRegistry[22]
)

var (
	GateTransferObjects = gateRegistry[23]
	GateManifestClosure = gateRegistry[24]
)
