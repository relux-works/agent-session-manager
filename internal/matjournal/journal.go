package matjournal

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// Journal schema identity the store binds.
const (
	journalSchema     = "urn:ax:schema:materialization-journal"
	journalVersion    = "2.0.0"
	journalKind       = "journal"
	markerKind        = "managed_replica_marker"
	extensionsMaximum = 64
	// lastErrorVersion is the Structured Error version journal
	// failures encode: 1.2.0 is the earliest version registering
	// every code recovery emits (operation_uncertain, the parked-
	// ambiguity code, registers at 1.2.0 and not below).
	lastErrorVersion = axerror.Version120
)

// Journal phases (SPEC Section 10.6). The task text predates the pinned
// revision and omits failed; the pinned phase enum carries it, and the
// store implements all eight values.
const (
	PhaseStaging     = "staging"
	PhaseValidating  = "validating"
	PhasePrepared    = "prepared"
	PhaseCommitting  = "committing"
	PhaseRollingBack = "rolling_back"
	PhaseRolledBack  = "rolled_back"
	PhaseCommitted   = "committed"
	PhaseFailed      = "failed"
)

// Provider transaction states (SPEC Section 10.6).
const (
	ProviderUnknown    = "unknown"
	ProviderPrepared   = "prepared"
	ProviderCommitted  = "committed"
	ProviderRolledBack = "rolled_back"
)

// Task-board transaction states (SPEC Section 10.6).
const (
	BoardNotStarted       = "not_started"
	BoardImported         = "imported"
	BoardOpened           = "opened"
	BoardAdopted          = "adopted"
	BoardResumed          = "resumed"
	BoardDormantFinalized = "dormant_finalized"
	BoardRolledBack       = "rolled_back"
	BoardFailed           = "failed"
)

// Task-board activation modes (SPEC Section 10.6).
const (
	ModeDormantReplica    = "dormant_replica"
	ModeOwnerResume       = "owner_resume"
	ModeOwnershipTransfer = "ownership_transfer"
	ModeFork              = "fork"
)

// Authority states (SPEC Section 10.6).
const (
	AuthorityStaging    = "staging"
	AuthorityPrepared   = "prepared"
	AuthorityCommitted  = "committed"
	AuthorityRolledBack = "rolled_back"
)

// Plan authority kinds (SPEC Section 10.5 RootAuthority union).
const (
	PlanKindWorkspace        = "workspace"
	PlanKindProviderStore    = "provider_store"
	PlanKindTaskBoardStaging = "task_board_staging"
)

// Provider transaction authority literals (SPEC Section 7.5).
const (
	txAuthorityID     = "provider_transaction"
	txAuthorityKind   = "provider_transaction"
	txAuthorityLayout = "provider_transaction_v1"
	txAuthorityAccess = "read_write"
)

// Journal cardinality bounds (SPEC Section 10.6).
const (
	maxAuthorities       = 512
	maxBlobs             = 65536
	maxChunksPerBlob     = 32768
	maxVerifiedBlobs     = 65536
	maxSequences         = 65536
	maxSameFilesystemIDs = 32
	minSameFilesystemIDs = 1
)

// Token bounds: base64url-256+ encodes 32..512 bytes (SPEC Section 1.6).
const (
	minTokenBytes = 32
	maxTokenBytes = 512
)

// uint53Max is the AX safe-integer ceiling (SPEC Section 1.6).
const uint53Max = uint64(1<<53 - 1)

// ErrInvalidJournal reports a journal shape or transition the Section 10.6
// contract refuses.
var ErrInvalidJournal = errors.New("invalid materialization journal")

// ErrJournalConflict reports the idempotency boundary: the same operation
// retried with a byte-unequal body (idempotency_mismatch), or a digest
// path holding disagreeing bytes (a torn store, never a second version
// of an immutable sidecar).
var ErrJournalConflict = errors.New("materialization journal conflict")

// ErrUnknownJournal reports a materialization with no journal: absence,
// never torn state.
var ErrUnknownJournal = errors.New("unknown materialization journal")

// Journal is the typed Materialization Journal 2.0.0 document: the exact
// closed 22-member object of SPEC Section 10.6. Nullable members use
// pointers so null round-trips exactly; LastError is nil for null and
// otherwise the exact Section 15.1 object bytes.
type Journal struct {
	Schema                    string                    `json:"schema"`
	SchemaVersion             string                    `json:"schema_version"`
	DocumentKind              string                    `json:"document_kind"`
	MaterializationID         string                    `json:"materialization_id"`
	PrepareOperationID        string                    `json:"prepare_operation_id"`
	PrepareRequestDigest      string                    `json:"prepare_request_digest"`
	TransferID                *string                   `json:"transfer_id"`
	PlanID                    string                    `json:"plan_id"`
	SourceCheckpointID        string                    `json:"source_checkpoint_id"`
	ManagedReplicaID          *string                   `json:"managed_replica_id"`
	AuthorityStates           map[string]AuthorityState `json:"authority_states"`
	ExpectedPriorCheckpointID *string                   `json:"expected_prior_checkpoint_id"`
	CompletedBlobChunks       map[string][]uint32       `json:"completed_blob_chunks"`
	VerifiedBlobIDs           []string                  `json:"verified_blob_ids"`
	Phase                     string                    `json:"phase"`
	Provider                  *ProviderTransaction      `json:"provider_transaction"`
	TaskBoard                 *TaskBoardTransaction     `json:"task_board_transaction"`
	DestinationMarkerID       *string                   `json:"destination_marker_id"`
	LastError                 json.RawMessage           `json:"last_error"`
	StartedAt                 string                    `json:"started_at"`
	UpdatedAt                 string                    `json:"updated_at"`
	Extensions                map[string]any            `json:"extensions"`
}

// ProviderTransaction is the closed Provider Journal Transaction object.
type ProviderTransaction struct {
	OperationID   string               `json:"operation_id"`
	TransactionID string               `json:"transaction_id"`
	State         string               `json:"state"`
	RollbackToken *string              `json:"rollback_token"`
	Authority     TransactionAuthority `json:"transaction_authority"`
	LastStatusAt  string               `json:"last_status_at"`
}

// TransactionAuthority is the closed ProviderTransactionAuthority object
// (SPEC Section 7.5).
type TransactionAuthority struct {
	AuthorityID                        string   `json:"authority_id"`
	Kind                               string   `json:"kind"`
	RootPath                           string   `json:"root_path"`
	Layout                             string   `json:"layout"`
	Access                             string   `json:"access"`
	MaterializationID                  string   `json:"materialization_id"`
	ProviderID                         string   `json:"provider_id"`
	TransactionID                      string   `json:"transaction_id"`
	PlanID                             string   `json:"plan_id"`
	SameFilesystemProviderAuthorityIDs []string `json:"same_filesystem_provider_authority_ids"`
}

// TaskBoardTransaction is the closed Task-board Journal Transaction
// object: the exact 19-member shape of SPEC Section 10.6.
type TaskBoardTransaction struct {
	BundleID          string     `json:"bundle_id"`
	ActivationMode    string     `json:"activation_mode"`
	ImportOperationID string     `json:"import_operation_id"`
	OpenOperationID   string     `json:"open_operation_id"`
	AdoptOperationID  string     `json:"adopt_operation_id"`
	ResumeOperationID string     `json:"resume_operation_id"`
	State             string     `json:"state"`
	ImportToken       *string    `json:"import_token"`
	StagedManagerRef  *string    `json:"staged_manager_ref"`
	ImportExpiresAt   *string    `json:"import_expires_at"`
	OpenToken         *string    `json:"open_token"`
	DormantManagerRef *string    `json:"dormant_manager_ref"`
	OpenExpiresAt     *string    `json:"open_expires_at"`
	ManagerSessionRef *string    `json:"manager_session_ref"`
	AxBinding         *AxBinding `json:"ax_binding"`
	LastBridgeState   *string    `json:"last_bridge_state"`
	LastStatusAt      *string    `json:"last_status_at"`
	CleanupState      string     `json:"cleanup_state"`
	CleanupAfter      *string    `json:"cleanup_after"`
}

// AxBinding is the exact closed bridge shape.
type AxBinding struct {
	AxSessionID string `json:"ax_session_id"`
	LeaseEpoch  uint64 `json:"lease_epoch"`
	LeaseID     string `json:"lease_id"`
}

// AuthorityState is the closed Authority Journal State object.
type AuthorityState struct {
	RootPath            string   `json:"root_path"`
	CompletedSequences  []uint64 `json:"completed_sequences"`
	ObservedPriorDigest *string  `json:"observed_prior_digest"`
	RollbackRoot        *string  `json:"rollback_root"`
	State               string   `json:"state"`
}

// PlanAuthority is the routing view of one plan RootAuthority (SPEC
// Section 10.5): the kind, platform, and root path the journal
// cross-checks its authority states against. The full plan stays an
// opaque digest input owned by the shape owner; this view carries only
// the routing facts the journal rules name.
type PlanAuthority struct {
	ID       string
	Kind     string
	Platform string
	RootPath string
}

// planView is the durably installed routing view: the executing host
// platform, one entry per plan RootAuthority, and the bound bridge
// operation keys when a bridge participates.
type planView struct {
	HostPlatform string                   `json:"host_platform"`
	Authorities  map[string]PlanAuthority `json:"authorities"`
	Bridge       *bridgeIDs               `json:"bridge_operation_ids"`
}

// CreateInputs is the typed materialize.prepare closure: caller-stable
// IDs allocated before the first request, the complete canonical
// request body, the plan/checkpoint digests, the plan authority routing
// view, and the bridge key set (allocated with the materialization ID
// when a task-board bridge participates, empty otherwise).
type CreateInputs struct {
	MaterializationID         string
	PrepareOperationID        string
	RequestBody               []byte
	TransferID                string
	PlanID                    string
	SourceCheckpointID        string
	ManagedReplicaID          string
	Plan                      []PlanAuthority
	HostPlatform              string
	ExpectedPriorCheckpointID string
	BundleID                  string
	ActivationMode            string
	ImportOperationID         string
	OpenOperationID           string
	AdoptOperationID          string
	ResumeOperationID         string
	Extensions                map[string]string
}

// invalid refuses a journal shape or transition with the package
// sentinel. It is a function, not a method, so every gate site reads
// identically.
func invalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidJournal, fmt.Sprintf(format, arguments...))
}

// conflict refuses an idempotency-boundary violation with the package
// conflict sentinel.
func conflict(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", ErrJournalConflict, fmt.Sprintf(format, arguments...))
}

// rootIDPattern is the Section 7.5 root-id grammar.
var rootIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// checkRootID admits one authority routing label.
func checkRootID(value string) (string, error) {
	if !rootIDPattern.MatchString(value) {
		return "", invalid("authority id %q is outside the root-id grammar", value)
	}
	return value, nil
}

// checkUUIDv7 admits one caller-stable identity.
func checkUUIDv7(value, name string) (string, error) {
	id, err := scalar.ParseUUIDv7(value)
	if err != nil {
		return "", invalid("%s %q: %v", name, value, err)
	}
	return id.String(), nil
}

// checkDigest admits one content digest.
func checkDigest(value, name string) (string, error) {
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return "", invalid("%s %q: %v", name, value, err)
	}
	return digest.String(), nil
}

// checkNullableDigest admits an empty-equals-null digest leg.
func checkNullableDigest(value, name string) (*string, error) {
	if value == "" {
		return nil, nil
	}
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return nil, invalid("%s %q: %v", name, value, err)
	}
	out := digest.String()
	return &out, nil
}

// checkNullableUUIDv7 admits an empty-equals-null UUIDv7 leg.
func checkNullableUUIDv7(value, name string) (*string, error) {
	if value == "" {
		return nil, nil
	}
	id, err := scalar.ParseUUIDv7(value)
	if err != nil {
		return nil, invalid("%s %q: %v", name, value, err)
	}
	out := id.String()
	return &out, nil
}

// checkTimestamp admits one diagnostic instant. Instants never confer
// authority, but their grammar is still closed.
func checkTimestamp(value, name string) (string, error) {
	instant, err := scalar.ParseTimestamp(value)
	if err != nil {
		return "", invalid("%s %q: %v", name, value, err)
	}
	return instant.String(), nil
}

// checkPlatform admits one executing-host platform name.
func checkPlatform(value, name string) (string, error) {
	platform, err := scalar.ParsePlatform(value)
	if err != nil {
		return "", invalid("%s %q: %v", name, value, err)
	}
	return platform.String(), nil
}

// checkAbsolutePath admits one machine-local absolute path under the
// named platform.
func checkAbsolutePath(platform scalar.Platform, value, name string) (string, error) {
	path, err := scalar.ParseAbsolutePath(platform, value)
	if err != nil {
		return "", invalid("%s %q: %v", name, value, err)
	}
	return path.String(), nil
}

// checkToken admits one base64url-256+ control token: the canonical
// RFC 4648 URL-safe alphabet with no padding or whitespace, a
// decode-then-re-encode fixpoint, and 32..512 decoded bytes.
func checkToken(value, name string) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", invalid("%s is not unpadded base64url: %v", name, err)
	}
	if base64.RawURLEncoding.EncodeToString(raw) != value {
		return "", invalid("%s is not canonical base64url", name)
	}
	if len(raw) < minTokenBytes || len(raw) > maxTokenBytes {
		return "", invalid("%s decodes to %d bytes, want %d..%d", name, len(raw), minTokenBytes, maxTokenBytes)
	}
	return value, nil
}

// checkManagerRef admits one bridge manager reference: 1..512
// characters of non-control UTF-8.
func checkManagerRef(value, name string) (string, error) {
	if !utf8.ValidString(value) {
		return "", invalid("%s is not valid UTF-8", name)
	}
	if length := utf8.RuneCountInString(value); length < 1 || length > 512 {
		return "", invalid("%s must contain 1..512 characters", name)
	}
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return "", invalid("%s must contain no control characters", name)
		}
	}
	return value, nil
}

// checkExtensions admits the machine-local extension object:
// reverse-DNS keys only, at most 64 members. A nil map binds the empty
// object, never null.
func checkExtensions(extensions map[string]string) (map[string]any, error) {
	if extensions == nil {
		extensions = map[string]string{}
	}
	out := make(map[string]any, len(extensions))
	for key, value := range extensions {
		out[key] = value
	}
	if len(out) > extensionsMaximum {
		return nil, invalid("extensions contains %d members, maximum is %d", len(out), extensionsMaximum)
	}
	framed, err := json.Marshal(out)
	if err != nil {
		return nil, invalid("encode journal extensions: %v", err)
	}
	if !environ.CheckExtensions(framed) {
		return nil, invalid("journal extensions keys must be 3..253 character lowercase reverse-DNS names")
	}
	return out, nil
}

// closedMembers refuses unknown members at one closed object level.
func closedMembers(members map[string]json.RawMessage, want []string, what string) error {
	allowed := make(map[string]struct{}, len(want))
	for _, name := range want {
		allowed[name] = struct{}{}
	}
	var unknown []string
	for name := range members {
		if _, ok := allowed[name]; !ok {
			unknown = append(unknown, name)
		}
	}
	sort.Strings(unknown)
	if len(unknown) > 0 {
		return invalid("%s carries unknown members %q", what, strings.Join(unknown, ","))
	}
	for _, name := range want {
		if _, ok := members[name]; !ok {
			return invalid("%s member %q is absent", what, name)
		}
	}
	return nil
}

// journalMembers is the exact closed 22-member journal shape.
var journalMembers = []string{
	"schema", "schema_version", "document_kind", "materialization_id",
	"prepare_operation_id", "prepare_request_digest", "transfer_id",
	"plan_id", "source_checkpoint_id", "managed_replica_id",
	"authority_states", "expected_prior_checkpoint_id",
	"completed_blob_chunks", "verified_blob_ids", "phase",
	"provider_transaction", "task_board_transaction",
	"destination_marker_id", "last_error", "started_at", "updated_at",
	"extensions",
}

func isNull(raw json.RawMessage) bool { return string(raw) == "null" }

func stringMember(members map[string]json.RawMessage, name string) (string, error) {
	raw, ok := members[name]
	if !ok {
		return "", fmt.Errorf("member %q is absent", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("member %q is not a string: %v", name, err)
	}
	return value, nil
}

// decodeJournal strictly decodes one journal document: duplicate keys
// refuse, unknown members refuse, absent members refuse, and every
// member's grammar is checked. Cross-checks against the plan routing
// view (root equality, path platforms, ID binding) run in
// checkJournalCrossReferences so reads and writes share them.
func decodeJournal(raw []byte, view planView) (Journal, error) {
	var journal Journal
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return Journal{}, invalid("decode journal frame: %v", fault)
	}
	if err := closedMembers(members, journalMembers, "journal"); err != nil {
		return Journal{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&journal); err != nil {
		return Journal{}, invalid("decode journal members: %v", err)
	}
	if err := checkJournalGrammar(&journal); err != nil {
		return Journal{}, err
	}
	if err := checkJournalCrossReferences(&journal, view); err != nil {
		return Journal{}, err
	}
	return journal, nil
}

// checkJournalGrammar validates every grammar rule that needs no plan
// context: schema literals, the phase and sub-state enums, digest/UUID/
// timestamp/token/path shapes, sorted-unique orders, and the per-state
// nullability tables.
func checkJournalGrammar(journal *Journal) error {
	if journal.Schema != journalSchema {
		return invalid("journal schema %q, want %q", journal.Schema, journalSchema)
	}
	if journal.SchemaVersion != journalVersion {
		return invalid("journal schema_version %q, want %q", journal.SchemaVersion, journalVersion)
	}
	if journal.DocumentKind != journalKind {
		return invalid("journal document_kind %q, want %q", journal.DocumentKind, journalKind)
	}
	var err error
	if journal.MaterializationID, err = checkUUIDv7(journal.MaterializationID, "materialization_id"); err != nil {
		return err
	}
	if journal.PrepareOperationID, err = checkUUIDv7(journal.PrepareOperationID, "prepare_operation_id"); err != nil {
		return err
	}
	if journal.PrepareRequestDigest, err = checkDigest(journal.PrepareRequestDigest, "prepare_request_digest"); err != nil {
		return err
	}
	if journal.TransferID, err = checkNullableUUIDv7Pointer(journal.TransferID, "transfer_id"); err != nil {
		return err
	}
	if journal.PlanID, err = checkDigest(journal.PlanID, "plan_id"); err != nil {
		return err
	}
	if journal.SourceCheckpointID, err = checkDigest(journal.SourceCheckpointID, "source_checkpoint_id"); err != nil {
		return err
	}
	if journal.ManagedReplicaID, err = checkNullableUUIDv7Pointer(journal.ManagedReplicaID, "managed_replica_id"); err != nil {
		return err
	}
	if err := checkAuthorityMap(journal.AuthorityStates); err != nil {
		return err
	}
	if journal.ExpectedPriorCheckpointID, err = checkNullableDigestPointer(journal.ExpectedPriorCheckpointID, "expected_prior_checkpoint_id"); err != nil {
		return err
	}
	if err := checkChunkMap(journal.CompletedBlobChunks); err != nil {
		return err
	}
	if journal.VerifiedBlobIDs, err = checkSortedUniqueDigests(journal.VerifiedBlobIDs, maxVerifiedBlobs, "verified_blob_ids"); err != nil {
		return err
	}
	if !validPhase(journal.Phase) {
		return invalid("journal phase %q is outside the closed phase enum", journal.Phase)
	}
	if journal.Provider != nil {
		if err := checkProviderTransaction(journal.Provider, journal); err != nil {
			return err
		}
	}
	if journal.TaskBoard != nil {
		if err := checkTaskBoardTransaction(journal.TaskBoard, nil); err != nil {
			return err
		}
	}
	if journal.DestinationMarkerID, err = checkNullableDigestPointer(journal.DestinationMarkerID, "destination_marker_id"); err != nil {
		return err
	}
	if err := checkLastError(journal.LastError); err != nil {
		return err
	}
	if journal.StartedAt, err = checkTimestamp(journal.StartedAt, "started_at"); err != nil {
		return err
	}
	if journal.UpdatedAt, err = checkTimestamp(journal.UpdatedAt, "updated_at"); err != nil {
		return err
	}
	if err := checkExtensionsMap(journal.Extensions); err != nil {
		return err
	}
	return nil
}

// checkJournalCrossReferences validates the rules that bind the journal
// to its plan routing view and its own IDs: exactly one authority entry
// per plan RootAuthority, root-path equality, path platforms, provider
// rollback-root placement, and the ID equalities the journal repeats.
func checkJournalCrossReferences(journal *Journal, view planView) error {
	if len(journal.AuthorityStates) != len(view.Authorities) {
		return invalid("journal carries %d authority states for %d plan authorities",
			len(journal.AuthorityStates), len(view.Authorities))
	}
	hostPlatform, err := scalar.ParsePlatform(view.HostPlatform)
	if err != nil {
		return invalid("plan view host_platform %q: %v", view.HostPlatform, err)
	}
	for id, authority := range view.Authorities {
		if _, err := checkRootID(id); err != nil {
			return err
		}
		switch authority.Kind {
		case PlanKindWorkspace, PlanKindProviderStore, PlanKindTaskBoardStaging:
		default:
			return invalid("plan authority %q kind %q is outside the closed union", id, authority.Kind)
		}
		platform, err := scalar.ParsePlatform(authority.Platform)
		if err != nil {
			return invalid("plan authority %q platform %q: %v", id, authority.Platform, err)
		}
		state, ok := journal.AuthorityStates[id]
		if !ok {
			return invalid("journal authority states miss plan authority %q", id)
		}
		if state.RootPath != authority.RootPath {
			return invalid("journal authority %q root %q differs from plan authority %q",
				id, state.RootPath, authority.RootPath)
		}
		if _, err := scalar.ParseAbsolutePath(platform, state.RootPath); err != nil {
			return invalid("journal authority %q root_path: %v", id, err)
		}
		if state.RollbackRoot != nil {
			if _, err := scalar.ParseAbsolutePath(hostPlatform, *state.RollbackRoot); err != nil {
				return invalid("journal authority %q rollback_root: %v", id, err)
			}
		}
		if authority.Kind == PlanKindProviderStore {
			if err := checkProviderRollbackRoot(id, &state, journal.Provider); err != nil {
				return err
			}
		}
	}
	// Map keys the struct decoder accepted still need root-id
	// grammar: JSON object keys are not struct members.
	for id := range journal.AuthorityStates {
		if _, err := checkRootID(id); err != nil {
			return err
		}
	}
	if journal.Provider != nil {
		if _, err := scalar.ParseAbsolutePath(hostPlatform, journal.Provider.Authority.RootPath); err != nil {
			return invalid("provider transaction_authority root_path: %v", err)
		}
	}
	if journal.TaskBoard != nil {
		if view.Bridge == nil {
			return invalid("journal carries a task-board transaction with no bound bridge keys")
		}
		if err := checkTaskBoardDrift(journal.TaskBoard, view.Bridge); err != nil {
			return err
		}
	} else if view.Bridge != nil {
		return invalid("journal drops the bound bridge transaction")
	}
	return nil
}

// checkTaskBoardDrift refuses bridge operation IDs that drift from the
// prepare-bound keys.
func checkTaskBoardDrift(board *TaskBoardTransaction, bound *bridgeIDs) error {
	if board.ImportOperationID != bound.Import || board.OpenOperationID != bound.Open ||
		board.AdoptOperationID != bound.Adopt || board.ResumeOperationID != bound.Resume {
		return invalid("task-board operation ids drift from the prepare-bound keys")
	}
	return nil
}

// checkProviderRollbackRoot enforces the single-backup rule: for a
// provider-store authority the rollback root equals the plugin
// transaction root's backups entry, and the host journal never
// allocates a second provider backup location. Without a provider
// transaction there is no plugin backup to reference, so a non-null
// root fails closed.
func checkProviderRollbackRoot(id string, state *AuthorityState, provider *ProviderTransaction) error {
	if state.RollbackRoot == nil {
		return nil
	}
	if provider == nil {
		return invalid("provider-store authority %q names a rollback root with no provider transaction", id)
	}
	want := provider.Authority.RootPath + "/backups/" + id
	if *state.RollbackRoot != want {
		return invalid("provider-store authority %q rollback root %q, want %q", id, *state.RollbackRoot, want)
	}
	return nil
}

func checkNullableDigestPointer(value *string, name string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	digest, err := scalar.ParseDigest(*value)
	if err != nil {
		return nil, invalid("%s %q: %v", name, *value, err)
	}
	out := digest.String()
	return &out, nil
}

func checkNullableUUIDv7Pointer(value *string, name string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	id, err := scalar.ParseUUIDv7(*value)
	if err != nil {
		return nil, invalid("%s %q: %v", name, *value, err)
	}
	out := id.String()
	return &out, nil
}

// checkAuthorityMap validates the authority state map: at most 512
// entries, root-id keys, and the closed per-authority shape with its
// state/enable rules.
func checkAuthorityMap(states map[string]AuthorityState) error {
	if states == nil {
		return invalid("authority_states is null, want an object")
	}
	if len(states) > maxAuthorities {
		return invalid("authority_states carries %d entries, maximum is %d", len(states), maxAuthorities)
	}
	for id, state := range states {
		if _, err := checkRootID(id); err != nil {
			return err
		}
		if err := checkAuthorityState(id, &state); err != nil {
			return err
		}
	}
	return nil
}

// checkAuthorityState validates one Authority Journal State: the closed
// shape, the sorted-unique completed sequences, and the state-gated
// rollback-root rule (prepared requires a root, committed and
// rolled-back require null after cleanup, staging permits null only
// before the first target backup exists — journal-observable as an
// empty completed set).
func checkAuthorityState(id string, state *AuthorityState) error {
	if state.RootPath == "" {
		return invalid("authority %q root_path is empty", id)
	}
	if state.CompletedSequences == nil {
		return invalid("authority %q completed_sequences is null, want an array", id)
	}
	if len(state.CompletedSequences) > maxSequences {
		return invalid("authority %q carries %d completed sequences, maximum is %d",
			id, len(state.CompletedSequences), maxSequences)
	}
	previous := uint64(0)
	for index, sequence := range state.CompletedSequences {
		if sequence > uint53Max {
			return invalid("authority %q completed sequence %d exceeds the uint53 ceiling", id, sequence)
		}
		if index > 0 && sequence <= previous {
			return invalid("authority %q completed_sequences must be sorted unique", id)
		}
		previous = sequence
	}
	var err error
	if state.ObservedPriorDigest, err = checkNullableDigestPointer(state.ObservedPriorDigest, "observed_prior_digest"); err != nil {
		return fmt.Errorf("authority %q: %w", id, err)
	}
	switch state.State {
	case AuthorityStaging, AuthorityPrepared, AuthorityCommitted, AuthorityRolledBack:
	default:
		return invalid("authority %q state %q is outside the closed enum", id, state.State)
	}
	switch state.State {
	case AuthorityPrepared:
		if state.RollbackRoot == nil {
			return invalid("authority %q is prepared with a null rollback root", id)
		}
	case AuthorityCommitted, AuthorityRolledBack:
		if state.RollbackRoot != nil {
			return invalid("authority %q is %s with a non-null rollback root", id, state.State)
		}
	case AuthorityStaging:
		if state.RollbackRoot == nil && len(state.CompletedSequences) > 0 {
			return invalid("authority %q is staging completed work with a null rollback root", id)
		}
	}
	return nil
}

// checkChunkMap validates the per-blob chunk sets: at most 65536 blobs,
// digest keys, and per blob a sorted-unique set of at most 32768
// indexes. The MJ-MULTIBLOB-POS rule is structural here: chunk zero
// for two blobs is two independent map entries, never one flat list.
func checkChunkMap(chunks map[string][]uint32) error {
	if chunks == nil {
		return invalid("completed_blob_chunks is null, want an object")
	}
	if len(chunks) > maxBlobs {
		return invalid("completed_blob_chunks carries %d blobs, maximum is %d", len(chunks), maxBlobs)
	}
	for id, indexes := range chunks {
		if _, err := scalar.ParseDigest(id); err != nil {
			return invalid("completed_blob_chunks key %q: %v", id, err)
		}
		if indexes == nil {
			return invalid("completed_blob_chunks %q is null, want an array", id)
		}
		if len(indexes) > maxChunksPerBlob {
			return invalid("completed_blob_chunks %q carries %d indexes, maximum is %d",
				id, len(indexes), maxChunksPerBlob)
		}
		for index, value := range indexes {
			if index > 0 && value <= indexes[index-1] {
				return invalid("completed_blob_chunks %q must be sorted unique", id)
			}
		}
	}
	return nil
}

// checkSortedUniqueDigests validates a sorted-unique digest array with
// an exact count bound. A null array refuses: these members are never
// null in the closed shape.
func checkSortedUniqueDigests(values []string, maximum int, name string) ([]string, error) {
	if values == nil {
		return nil, invalid("%s is null, want an array", name)
	}
	if len(values) > maximum {
		return nil, invalid("%s carries %d entries, maximum is %d", name, len(values), maximum)
	}
	out := make([]string, 0, len(values))
	previous := ""
	for _, value := range values {
		digest, err := scalar.ParseDigest(value)
		if err != nil {
			return nil, invalid("%s entry %q: %v", name, value, err)
		}
		canonical := digest.String()
		if len(out) > 0 && canonical <= previous {
			return nil, invalid("%s must be sorted unique", name)
		}
		previous = canonical
		out = append(out, canonical)
	}
	return out, nil
}

// checkProviderTransaction validates the closed Provider Journal
// Transaction: the state enum, the exact token nullability rule
// (prepared requires a token, every other state requires null), and
// the authority ID binding to the journal and plan.
func checkProviderTransaction(provider *ProviderTransaction, journal *Journal) error {
	var err error
	if provider.OperationID, err = checkUUIDv7(provider.OperationID, "provider operation_id"); err != nil {
		return err
	}
	if provider.TransactionID, err = checkUUIDv7(provider.TransactionID, "provider transaction_id"); err != nil {
		return err
	}
	switch provider.State {
	case ProviderUnknown, ProviderPrepared, ProviderCommitted, ProviderRolledBack:
	default:
		return invalid("provider state %q is outside the closed enum", provider.State)
	}
	if provider.State == ProviderPrepared {
		if provider.RollbackToken == nil {
			return invalid("provider prepared state requires a non-null rollback token")
		}
		token, err := checkToken(*provider.RollbackToken, "provider rollback_token")
		if err != nil {
			return err
		}
		provider.RollbackToken = &token
	} else if provider.RollbackToken != nil {
		return invalid("provider %s state requires a null rollback token", provider.State)
	}
	if err := checkTransactionAuthority(&provider.Authority, journal, provider.TransactionID); err != nil {
		return err
	}
	if provider.LastStatusAt, err = checkTimestamp(provider.LastStatusAt, "provider last_status_at"); err != nil {
		return err
	}
	return nil
}

// checkTransactionAuthority validates the closed
// ProviderTransactionAuthority and binds its materialization,
// transaction, and plan IDs to the journal.
func checkTransactionAuthority(authority *TransactionAuthority, journal *Journal, transactionID string) error {
	if authority.AuthorityID != txAuthorityID {
		return invalid("transaction_authority authority_id %q, want %q", authority.AuthorityID, txAuthorityID)
	}
	if authority.Kind != txAuthorityKind {
		return invalid("transaction_authority kind %q, want %q", authority.Kind, txAuthorityKind)
	}
	if authority.RootPath == "" {
		return invalid("transaction_authority root_path is empty")
	}
	if authority.Layout != txAuthorityLayout {
		return invalid("transaction_authority layout %q, want %q", authority.Layout, txAuthorityLayout)
	}
	if authority.Access != txAuthorityAccess {
		return invalid("transaction_authority access %q, want %q", authority.Access, txAuthorityAccess)
	}
	var err error
	if authority.MaterializationID, err = checkUUIDv7(authority.MaterializationID, "transaction_authority materialization_id"); err != nil {
		return err
	}
	if authority.MaterializationID != journal.MaterializationID {
		return invalid("transaction_authority materialization_id drifts from the journal")
	}
	if _, err := scalar.ParseProviderID(authority.ProviderID); err != nil {
		return invalid("transaction_authority provider_id: %v", err)
	}
	if authority.TransactionID, err = checkUUIDv7(authority.TransactionID, "transaction_authority transaction_id"); err != nil {
		return err
	}
	if authority.TransactionID != transactionID {
		return invalid("transaction_authority transaction_id drifts from the provider transaction")
	}
	if authority.PlanID, err = checkDigest(authority.PlanID, "transaction_authority plan_id"); err != nil {
		return err
	}
	if authority.PlanID != journal.PlanID {
		return invalid("transaction_authority plan_id drifts from the journal")
	}
	if authority.SameFilesystemProviderAuthorityIDs == nil {
		return invalid("transaction_authority same_filesystem_provider_authority_ids is null, want an array")
	}
	if len(authority.SameFilesystemProviderAuthorityIDs) < minSameFilesystemIDs ||
		len(authority.SameFilesystemProviderAuthorityIDs) > maxSameFilesystemIDs {
		return invalid("transaction_authority carries %d same-filesystem authority ids, want %d..%d",
			len(authority.SameFilesystemProviderAuthorityIDs), minSameFilesystemIDs, maxSameFilesystemIDs)
	}
	previous := ""
	for _, id := range authority.SameFilesystemProviderAuthorityIDs {
		if _, err := checkRootID(id); err != nil {
			return err
		}
		if previous != "" && id <= previous {
			return invalid("transaction_authority same-filesystem authority ids must be sorted unique")
		}
		previous = id
	}
	return nil
}

// bridgeIDs binds the four stable bridge operation keys allocated with
// the materialization ID. A retry must reuse them; drift refuses.
type bridgeIDs struct {
	Import string `json:"import_operation_id"`
	Open   string `json:"open_operation_id"`
	Adopt  string `json:"adopt_operation_id"`
	Resume string `json:"resume_operation_id"`
}

// bridgeCleanupStates is the closed cleanup_state enum.
var bridgeCleanupStates = map[string]struct{}{
	"not_started": {}, "pending_expiry": {}, "retained_active": {}, "removed": {},
}

// bridgeLiveStates is the closed last_bridge_state enum.
var bridgeLiveStates = map[string]struct{}{
	"dormant": {}, "quiesced": {}, "stopped": {}, "running": {}, "idle": {}, "failed": {},
}

// checkTaskBoardTransaction validates the closed Task-board Journal
// Transaction: the state enum, the four operation IDs, the exact
// per-state token/reference/expiry nullability table, the Ax Binding,
// and the cleanup rules. Bound carries the Create-time IDs; a nil
// bound skips the drift check (grammar-only validation).
func checkTaskBoardTransaction(board *TaskBoardTransaction, bound *bridgeIDs) error {
	var err error
	if board.BundleID, err = checkDigest(board.BundleID, "task-board bundle_id"); err != nil {
		return err
	}
	switch board.ActivationMode {
	case ModeDormantReplica, ModeOwnerResume, ModeOwnershipTransfer, ModeFork:
	default:
		return invalid("task-board activation_mode %q is outside the closed enum", board.ActivationMode)
	}
	if board.ImportOperationID, err = checkUUIDv7(board.ImportOperationID, "task-board import_operation_id"); err != nil {
		return err
	}
	if board.OpenOperationID, err = checkUUIDv7(board.OpenOperationID, "task-board open_operation_id"); err != nil {
		return err
	}
	if board.AdoptOperationID, err = checkUUIDv7(board.AdoptOperationID, "task-board adopt_operation_id"); err != nil {
		return err
	}
	if board.ResumeOperationID, err = checkUUIDv7(board.ResumeOperationID, "task-board resume_operation_id"); err != nil {
		return err
	}
	if bound != nil {
		if err := checkTaskBoardDrift(board, bound); err != nil {
			return err
		}
	}
	switch board.State {
	case BoardNotStarted, BoardImported, BoardOpened, BoardAdopted, BoardResumed,
		BoardDormantFinalized, BoardRolledBack, BoardFailed:
	default:
		return invalid("task-board state %q is outside the closed enum", board.State)
	}
	if err := checkBoardNullability(board); err != nil {
		return err
	}
	if board.AxBinding != nil {
		if err := checkAxBinding(board.AxBinding); err != nil {
			return err
		}
	}
	if board.LastBridgeState != nil {
		if _, ok := bridgeLiveStates[*board.LastBridgeState]; !ok {
			return invalid("task-board last_bridge_state %q is outside the closed enum", *board.LastBridgeState)
		}
	}
	if board.LastStatusAt != nil {
		status, err := checkTimestamp(*board.LastStatusAt, "task-board last_status_at")
		if err != nil {
			return err
		}
		board.LastStatusAt = &status
	}
	if _, ok := bridgeCleanupStates[board.CleanupState]; !ok {
		return invalid("task-board cleanup_state %q is outside the closed enum", board.CleanupState)
	}
	if board.CleanupAfter != nil {
		after, err := checkTimestamp(*board.CleanupAfter, "task-board cleanup_after")
		if err != nil {
			return err
		}
		board.CleanupAfter = &after
	}
	if board.State == BoardDormantFinalized {
		if board.CleanupAfter == nil {
			return invalid("task-board dormant_finalized requires a non-null cleanup_after")
		}
	} else if board.CleanupAfter != nil {
		return invalid("task-board %s requires a null cleanup_after", board.State)
	}
	return nil
}

// checkBoardNullability enforces the exact null/state invariants:
// not_started has all tokens and references null; imported has the
// import triple only; opened has the open triple only; adopted and
// resumed have the manager reference and binding only; rolled_back has
// none; failed preserves only one still-valid pair.
func checkBoardNullability(board *TaskBoardTransaction) error {
	importTriple := board.ImportToken != nil || board.StagedManagerRef != nil || board.ImportExpiresAt != nil
	openTriple := board.OpenToken != nil || board.DormantManagerRef != nil || board.OpenExpiresAt != nil
	adoptedPair := board.ManagerSessionRef != nil || board.AxBinding != nil
	switch board.State {
	case BoardNotStarted:
		if importTriple || openTriple || adoptedPair {
			return invalid("task-board not_started requires null tokens and references")
		}
	case BoardImported:
		if err := requireImportTriple(board); err != nil {
			return err
		}
		if openTriple || adoptedPair {
			return invalid("task-board imported requires null open and adopted members")
		}
	case BoardOpened:
		if err := requireOpenTriple(board); err != nil {
			return err
		}
		if importTriple || adoptedPair {
			return invalid("task-board opened requires null import and adopted members")
		}
	case BoardAdopted, BoardResumed:
		if importTriple || openTriple {
			return invalid("task-board %s requires null import and open members", board.State)
		}
		if board.ManagerSessionRef == nil || board.AxBinding == nil {
			return invalid("task-board %s requires a manager reference and binding", board.State)
		}
		if board.CleanupState != "retained_active" {
			return invalid("task-board %s requires cleanup_state retained_active", board.State)
		}
	case BoardDormantFinalized:
		if board.ImportToken != nil || board.OpenToken != nil {
			return invalid("task-board dormant_finalized keeps no usable token")
		}
		if board.StagedManagerRef != nil || board.ImportExpiresAt != nil || board.OpenExpiresAt != nil {
			return invalid("task-board dormant_finalized requires null import and open members")
		}
		if board.DormantManagerRef == nil {
			return invalid("task-board dormant_finalized requires a dormant reference")
		}
		if board.ManagerSessionRef != nil || board.AxBinding != nil {
			return invalid("task-board dormant_finalized requires a null adopted pair")
		}
		if board.LastBridgeState == nil || *board.LastBridgeState != "dormant" {
			return invalid("task-board dormant_finalized requires bridge state dormant")
		}
		if board.CleanupState != "pending_expiry" {
			return invalid("task-board dormant_finalized requires cleanup_state pending_expiry")
		}
	case BoardRolledBack:
		if importTriple || openTriple || adoptedPair {
			return invalid("task-board rolled_back requires null tokens and references")
		}
		// last_bridge_state stays unconstrained here: it is the
		// last reconciliation observation, which may lag the
		// rollback. Liveness is decided by status probes at
		// recovery, not by this member.
		if board.CleanupState != "removed" {
			return invalid("task-board rolled_back requires cleanup_state removed")
		}
	case BoardFailed:
		if err := checkFailedPair(board); err != nil {
			return err
		}
	}
	return checkBoardMemberGrammar(board)
}

// requireImportTriple enforces the complete import capability: token,
// staged reference, and expiry together.
func requireImportTriple(board *TaskBoardTransaction) error {
	if board.ImportToken == nil || board.StagedManagerRef == nil || board.ImportExpiresAt == nil {
		return invalid("task-board imported requires the import token, staged reference, and expiry")
	}
	return nil
}

// requireOpenTriple enforces the complete open capability: token,
// dormant reference, and expiry together.
func requireOpenTriple(board *TaskBoardTransaction) error {
	if board.OpenToken == nil || board.DormantManagerRef == nil || board.OpenExpiresAt == nil {
		return invalid("task-board opened requires the open token, dormant reference, and expiry")
	}
	return nil
}

// checkFailedPair enforces the failed-state preservation rule: at most
// one still-valid token/reference pair, and no partial triple (a
// partial triple is not a valid pair).
func checkFailedPair(board *TaskBoardTransaction) error {
	complete := 0
	if board.ImportToken != nil || board.StagedManagerRef != nil || board.ImportExpiresAt != nil {
		if board.ImportToken == nil || board.StagedManagerRef == nil || board.ImportExpiresAt == nil {
			return invalid("task-board failed preserves a partial import triple")
		}
		complete++
	}
	if board.OpenToken != nil || board.DormantManagerRef != nil || board.OpenExpiresAt != nil {
		if board.OpenToken == nil || board.DormantManagerRef == nil || board.OpenExpiresAt == nil {
			return invalid("task-board failed preserves a partial open triple")
		}
		complete++
	}
	if board.ManagerSessionRef != nil || board.AxBinding != nil {
		if board.ManagerSessionRef == nil || board.AxBinding == nil {
			return invalid("task-board failed preserves a partial adopted pair")
		}
		complete++
	}
	if complete > 1 {
		return invalid("task-board failed preserves more than one token/reference pair")
	}
	return nil
}

// checkBoardMemberGrammar validates the grammar of every non-null
// token, reference, and expiry member.
func checkBoardMemberGrammar(board *TaskBoardTransaction) error {
	for _, leg := range []struct {
		name   string
		value  *string
		token  bool
		expiry bool
	}{
		{"import_token", board.ImportToken, true, false},
		{"open_token", board.OpenToken, true, false},
		{"import_expires_at", board.ImportExpiresAt, false, true},
		{"open_expires_at", board.OpenExpiresAt, false, true},
	} {
		if leg.value == nil {
			continue
		}
		if leg.token {
			token, err := checkToken(*leg.value, "task-board "+leg.name)
			if err != nil {
				return err
			}
			*leg.value = token
		} else if leg.expiry {
			expiry, err := checkTimestamp(*leg.value, "task-board "+leg.name)
			if err != nil {
				return err
			}
			*leg.value = expiry
		}
	}
	for _, leg := range []struct {
		name  string
		value **string
	}{
		{"staged_manager_ref", &board.StagedManagerRef},
		{"dormant_manager_ref", &board.DormantManagerRef},
		{"manager_session_ref", &board.ManagerSessionRef},
	} {
		if *leg.value == nil {
			continue
		}
		ref, err := checkManagerRef(**leg.value, "task-board "+leg.name)
		if err != nil {
			return err
		}
		**leg.value = ref
	}
	return nil
}

// checkAxBinding validates the exact closed bridge binding.
func checkAxBinding(binding *AxBinding) error {
	var err error
	if binding.AxSessionID, err = checkUUIDv7(binding.AxSessionID, "ax_binding ax_session_id"); err != nil {
		return err
	}
	if binding.LeaseEpoch < 1 || binding.LeaseEpoch > uint53Max {
		return invalid("ax_binding lease_epoch %d is outside 1..2^53-1", binding.LeaseEpoch)
	}
	id, err := scalar.ParseUUIDv4(binding.LeaseID)
	if err != nil {
		return invalid("ax_binding lease_id %q: %v", binding.LeaseID, err)
	}
	binding.LeaseID = id.String()
	return nil
}

// checkLastError validates the redacted Structured Error member: null
// or the exact Section 15.1 object the axerror owner decodes. Tokens
// never reach this member: RecordError builds it from fixed templates
// and IDs, and the owner keeps the local cause off the wire.
func checkLastError(raw json.RawMessage) error {
	if len(raw) == 0 || isNull(raw) {
		return nil
	}
	if _, err := axerror.Decode(lastErrorVersion, raw); err != nil {
		return invalid("last_error is not a Structured Error: %v", err)
	}
	return nil
}

// checkExtensionsMap validates the decoded extension object.
func checkExtensionsMap(extensions map[string]any) error {
	if extensions == nil {
		return invalid("extensions is null, want an object")
	}
	if len(extensions) > extensionsMaximum {
		return invalid("extensions contains %d members, maximum is %d", len(extensions), extensionsMaximum)
	}
	framed, err := json.Marshal(extensions)
	if err != nil {
		return invalid("encode journal extensions: %v", err)
	}
	if !environ.CheckExtensions(framed) {
		return invalid("journal extensions keys must be 3..253 character lowercase reverse-DNS names")
	}
	return nil
}

// validPhase reports whether the phase names the closed enum.
func validPhase(phase string) bool {
	switch phase {
	case PhaseStaging, PhaseValidating, PhasePrepared, PhaseCommitting,
		PhaseRollingBack, PhaseRolledBack, PhaseCommitted, PhaseFailed:
		return true
	default:
		return false
	}
}

// terminalPhase reports whether the phase ends the journal: no
// transition leaves it.
func terminalPhase(phase string) bool {
	switch phase {
	case PhaseCommitted, PhaseRolledBack, PhaseFailed:
		return true
	default:
		return false
	}
}

// legalTransitions is the journal phase table, derived from the Section
// 10.6 coordinator order (prepare creates, transfer stages, validation
// runs, commit returns prepared, finalize commits), the crash fixtures
// (a committing journal reconciles, a rolling_back journal retries or
// closes), and the abort rule (a valid staging transaction resumes or
// rolls back). Forward edges follow the order; every non-terminal phase
// also reaches rolling_back (abort) and failed (terminal fault). The
// committing-to-rolling_back edge is additionally gated on
// pre-activation by checkTransitionGuards, because byte rollback is
// forbidden after adopt. Each edge's grounding is recorded in
// TRACEABILITY.md.
var legalTransitions = map[string][]string{
	PhaseStaging:     {PhaseValidating, PhaseRollingBack, PhaseFailed},
	PhaseValidating:  {PhasePrepared, PhaseRollingBack, PhaseFailed},
	PhasePrepared:    {PhaseCommitting, PhaseRollingBack, PhaseFailed},
	PhaseCommitting:  {PhaseCommitted, PhaseRollingBack, PhaseFailed},
	PhaseRollingBack: {PhaseRolledBack, PhaseFailed},
}

// legalTransition reports whether the phase edge exists in the table.
func legalTransition(from, to string) bool {
	for _, next := range legalTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// boardTransitions is the task-board sub-state table, derived from the
// Section 10.6 bridge order (import, persist imported, open, persist
// opened, adopt, resume, or dormant finalize) and the post-adopt byte
// rollback ban (adopted and resumed never reach rolled_back).
var boardTransitions = map[string][]string{
	BoardNotStarted:       {BoardImported, BoardRolledBack, BoardFailed},
	BoardImported:         {BoardOpened, BoardRolledBack, BoardFailed},
	BoardOpened:           {BoardAdopted, BoardDormantFinalized, BoardRolledBack, BoardFailed},
	BoardAdopted:          {BoardResumed, BoardFailed},
	BoardResumed:          {BoardFailed},
	BoardDormantFinalized: {BoardFailed},
}

// legalBoardTransition reports whether the task-board edge exists.
func legalBoardTransition(from, to string) bool {
	for _, next := range boardTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// authorityTransitions is the authority sub-state table: staging
// prepares or rolls back, prepared commits or rolls back, and the two
// terminal states end the authority.
var authorityTransitions = map[string][]string{
	AuthorityStaging:  {AuthorityPrepared, AuthorityRolledBack},
	AuthorityPrepared: {AuthorityCommitted, AuthorityRolledBack},
}

// legalAuthorityTransition reports whether the authority edge exists.
// A same-state update is not a transition and is always allowed.
func legalAuthorityTransition(from, to string) bool {
	if from == to {
		return true
	}
	for _, next := range authorityTransitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// buildCreate validates the prepare closure and assembles the initial
// journal: both caller-stable IDs and the canonical request digest
// bound durably before any staging authority or bridge mutation, one
// staging authority entry per plan RootAuthority, and the bridge key
// set bound in not_started when a bridge participates. It returns the
// journal, the plan routing view, and the request digest.
func buildCreate(inputs CreateInputs, startedAt string) (Journal, planView, string, error) {
	materialization, err := checkUUIDv7(inputs.MaterializationID, "materialization_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	prepare, err := checkUUIDv7(inputs.PrepareOperationID, "prepare_operation_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	if _, fault := environ.DecodeStrictObject(inputs.RequestBody); fault != nil {
		return Journal{}, planView{}, "", invalid("prepare request body is not a strict object: %v", fault)
	}
	canonical, err := canonicaljson.Canonicalize(inputs.RequestBody)
	if err != nil {
		return Journal{}, planView{}, "", invalid("canonicalize prepare request body: %v", err)
	}
	digest := scalar.SHA256Digest(canonical).String()
	transfer, err := checkNullableUUIDv7(inputs.TransferID, "transfer_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	plan, err := checkDigest(inputs.PlanID, "plan_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	checkpoint, err := checkDigest(inputs.SourceCheckpointID, "source_checkpoint_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	replica, err := checkNullableUUIDv7(inputs.ManagedReplicaID, "managed_replica_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	hostPlatform, err := checkPlatform(inputs.HostPlatform, "host_platform")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	view, states, err := buildPlanView(inputs.Plan)
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	view.HostPlatform = hostPlatform
	prior, err := checkNullableDigest(inputs.ExpectedPriorCheckpointID, "expected_prior_checkpoint_id")
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	extensions, err := checkExtensions(inputs.Extensions)
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	board, bound, err := buildInitialTaskBoard(inputs)
	if err != nil {
		return Journal{}, planView{}, "", err
	}
	view.Bridge = bound
	journal := Journal{
		Schema:                    journalSchema,
		SchemaVersion:             journalVersion,
		DocumentKind:              journalKind,
		MaterializationID:         materialization,
		PrepareOperationID:        prepare,
		PrepareRequestDigest:      digest,
		TransferID:                transfer,
		PlanID:                    plan,
		SourceCheckpointID:        checkpoint,
		ManagedReplicaID:          replica,
		AuthorityStates:           states,
		ExpectedPriorCheckpointID: prior,
		CompletedBlobChunks:       map[string][]uint32{},
		VerifiedBlobIDs:           []string{},
		Phase:                     PhaseStaging,
		Provider:                  nil,
		TaskBoard:                 board,
		DestinationMarkerID:       nil,
		LastError:                 nil,
		StartedAt:                 startedAt,
		UpdatedAt:                 startedAt,
		Extensions:                extensions,
	}
	if err := checkJournalGrammar(&journal); err != nil {
		return Journal{}, planView{}, "", err
	}
	if err := checkJournalCrossReferences(&journal, view); err != nil {
		return Journal{}, planView{}, "", err
	}
	return journal, view, digest, nil
}

// buildPlanView validates the plan authority routing view and assembles
// the initial staging authority entries: root paths equal their plan
// authorities, empty completed sets, null observed digests and rollback
// roots. The plan shape itself stays with the shape owner; only the
// routing facts the journal rules name are carried.
func buildPlanView(plan []PlanAuthority) (planView, map[string]AuthorityState, error) {
	if len(plan) > maxAuthorities {
		return planView{}, nil, invalid("plan carries %d authorities, maximum is %d", len(plan), maxAuthorities)
	}
	view := planView{Authorities: make(map[string]PlanAuthority, len(plan))}
	states := make(map[string]AuthorityState, len(plan))
	for _, authority := range plan {
		id, err := checkRootID(authority.ID)
		if err != nil {
			return planView{}, nil, err
		}
		if _, duplicate := view.Authorities[id]; duplicate {
			return planView{}, nil, invalid("plan authority %q repeats", id)
		}
		switch authority.Kind {
		case PlanKindWorkspace, PlanKindProviderStore, PlanKindTaskBoardStaging:
		default:
			return planView{}, nil, invalid("plan authority %q kind %q is outside the closed union", id, authority.Kind)
		}
		platform, err := checkPlatform(authority.Platform, "plan authority platform")
		if err != nil {
			return planView{}, nil, fmt.Errorf("plan authority %q: %w", id, err)
		}
		parsed, err := scalar.ParsePlatform(platform)
		if err != nil {
			return planView{}, nil, fmt.Errorf("plan authority %q: %w", id, err)
		}
		root, err := checkAbsolutePath(parsed, authority.RootPath, "plan authority root_path")
		if err != nil {
			return planView{}, nil, fmt.Errorf("plan authority %q: %w", id, err)
		}
		view.Authorities[id] = PlanAuthority{ID: id, Kind: authority.Kind, Platform: platform, RootPath: root}
		states[id] = AuthorityState{
			RootPath:            root,
			CompletedSequences:  []uint64{},
			ObservedPriorDigest: nil,
			RollbackRoot:        nil,
			State:               AuthorityStaging,
		}
	}
	return view, states, nil
}

// buildInitialTaskBoard binds the bridge key set in not_started when a
// bridge participates (bundle non-empty), or returns a null transaction
// when it does not. Partial bridge inputs refuse: the four operation
// IDs and the activation mode are allocated together or not at all.
func buildInitialTaskBoard(inputs CreateInputs) (*TaskBoardTransaction, *bridgeIDs, error) {
	partial := inputs.BundleID != "" || inputs.ActivationMode != "" ||
		inputs.ImportOperationID != "" || inputs.OpenOperationID != "" ||
		inputs.AdoptOperationID != "" || inputs.ResumeOperationID != ""
	if !partial {
		return nil, nil, nil
	}
	bundle, err := checkDigest(inputs.BundleID, "task-board bundle_id")
	if err != nil {
		return nil, nil, err
	}
	var mode string
	switch inputs.ActivationMode {
	case ModeDormantReplica, ModeOwnerResume, ModeOwnershipTransfer, ModeFork:
		mode = inputs.ActivationMode
	default:
		return nil, nil, invalid("task-board activation_mode %q is outside the closed enum", inputs.ActivationMode)
	}
	bound := &bridgeIDs{}
	if bound.Import, err = checkUUIDv7(inputs.ImportOperationID, "task-board import_operation_id"); err != nil {
		return nil, nil, err
	}
	if bound.Open, err = checkUUIDv7(inputs.OpenOperationID, "task-board open_operation_id"); err != nil {
		return nil, nil, err
	}
	if bound.Adopt, err = checkUUIDv7(inputs.AdoptOperationID, "task-board adopt_operation_id"); err != nil {
		return nil, nil, err
	}
	if bound.Resume, err = checkUUIDv7(inputs.ResumeOperationID, "task-board resume_operation_id"); err != nil {
		return nil, nil, err
	}
	board := &TaskBoardTransaction{
		BundleID:          bundle,
		ActivationMode:    mode,
		ImportOperationID: bound.Import,
		OpenOperationID:   bound.Open,
		AdoptOperationID:  bound.Adopt,
		ResumeOperationID: bound.Resume,
		State:             BoardNotStarted,
		CleanupState:      "not_started",
	}
	if err := checkTaskBoardTransaction(board, bound); err != nil {
		return nil, nil, err
	}
	return board, bound, nil
}

// checkTransitionGuards enforces the phase-entry rules beyond the edge
// table: prepared requires the provider prepared and the opened bridge
// state durable; committed requires converged sub-states and, for
// workspace transactions, the exact destination marker; rolling_back
// requires pre-activation; failed requires the explaining last_error.
func checkTransitionGuards(journal *Journal, to string, opts TransitionOpts) error {
	switch to {
	case PhasePrepared:
		if journal.Provider != nil && journal.Provider.State != ProviderPrepared {
			return invalid("prepared requires the provider transaction prepared, got %s", journal.Provider.State)
		}
		if journal.TaskBoard != nil && journal.TaskBoard.State != BoardOpened {
			return invalid("prepared requires the durable opened bridge state, got %s", journal.TaskBoard.State)
		}
	case PhaseCommitted:
		if journal.Provider != nil && journal.Provider.State != ProviderCommitted {
			return invalid("committed requires the provider transaction committed, got %s", journal.Provider.State)
		}
		if journal.TaskBoard != nil {
			switch journal.TaskBoard.State {
			case BoardResumed, BoardDormantFinalized:
			default:
				return invalid("committed requires the bridge resumed or dormant finalized, got %s", journal.TaskBoard.State)
			}
		}
		for id, state := range journal.AuthorityStates {
			if state.State != AuthorityCommitted {
				return invalid("committed requires authority %q committed, got %s", id, state.State)
			}
		}
		if journal.ManagedReplicaID != nil {
			if len(opts.Marker) == 0 {
				return invalid("committed workspace transaction requires the destination marker")
			}
			marker, err := ValidateMarker(opts.Marker)
			if err != nil {
				return err
			}
			if err := checkMarkerBinding(marker, journal, opts.PriorMarker); err != nil {
				return err
			}
		} else if len(opts.Marker) != 0 {
			return invalid("committed non-workspace transaction carries marker evidence")
		}
	case PhaseRollingBack:
		if err := checkPreActivation(journal); err != nil {
			return err
		}
	case PhaseRolledBack:
		if journal.Provider != nil && journal.Provider.State != ProviderRolledBack {
			return invalid("rolled_back requires the provider transaction rolled back, got %s", journal.Provider.State)
		}
		if journal.TaskBoard != nil && journal.TaskBoard.State != BoardRolledBack {
			return invalid("rolled_back requires the bridge rolled back, got %s", journal.TaskBoard.State)
		}
		for id, state := range journal.AuthorityStates {
			if state.State != AuthorityRolledBack {
				return invalid("rolled_back requires authority %q rolled back, got %s", id, state.State)
			}
		}
	case PhaseFailed:
		if len(opts.LastError) == 0 || isNull(opts.LastError) {
			return invalid("failed requires the explaining last_error")
		}
		if err := checkLastError(opts.LastError); err != nil {
			return err
		}
	}
	return nil
}

// checkPreActivation refuses byte rollback after the manager or the
// provider became active authority: adopted/resumed bridge states and
// the committed provider state end the rollback window.
func checkPreActivation(journal *Journal) error {
	if journal.TaskBoard != nil {
		switch journal.TaskBoard.State {
		case BoardAdopted, BoardResumed:
			return invalid("byte rollback is forbidden after adopt: the manager is active authority")
		}
	}
	if journal.Provider != nil && journal.Provider.State == ProviderCommitted {
		return invalid("byte rollback is forbidden after the provider committed")
	}
	return nil
}

// encodeJournal normalizes the empty collections (empty object/array,
// never null) and marshals the closed document.
func encodeJournal(journal *Journal) ([]byte, error) {
	if journal.AuthorityStates == nil {
		journal.AuthorityStates = map[string]AuthorityState{}
	}
	if journal.CompletedBlobChunks == nil {
		journal.CompletedBlobChunks = map[string][]uint32{}
	}
	if journal.VerifiedBlobIDs == nil {
		journal.VerifiedBlobIDs = []string{}
	}
	if journal.Extensions == nil {
		journal.Extensions = map[string]any{}
	}
	for id, state := range journal.AuthorityStates {
		if state.CompletedSequences == nil {
			state.CompletedSequences = []uint64{}
			journal.AuthorityStates[id] = state
		}
	}
	raw, err := json.Marshal(journal)
	if err != nil {
		return nil, fmt.Errorf("encode journal: %w", err)
	}
	return raw, nil
}
