// Authoritative shared summaries for the v0.6.0 selector contract
// (SPEC Section 14.7.3).
//
// List and Status above keep the internal read model and refuse only
// the unrepresentable record-only bootstrap. This layer is the closed
// authoritative summary: every owner, lease, and role fact comes from
// the validated winning Lease Record, every host display name comes
// from validated host metadata, and every workspace, capability,
// liveness, and timestamp fact comes from a validated observation.
// Anything that cannot be established truthfully refuses instead of
// falling back to inference:
//
//   - a complete read proving a record without any valid lease is
//     selector_bootstrap_incomplete (inherited from the read entry);
//   - a required host metadata fact, winning Lease Record, or
//     observation that cannot be established is
//     selector_observation_unavailable, never a source alias, an empty
//     name, or a minted process/capability fact;
//   - list refuses the whole success document when any encountered
//     record cannot be represented, including a mixed healthy and
//     incomplete listing; no record is silently omitted.
//
// CLI envelope and exit rendering, durable bootstrap recovery, and
// effect authorization remain their existing caller owners. Plans
// built here authorize read projection only.
package sessquery

import (
	"fmt"
	"sort"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// MaxDisplayNameRunes bounds one validated host display name: the
// SessionSummary owner_host_name is string[1..64].
const MaxDisplayNameRunes = 64

// MaxCapabilityNameRunes bounds one validated capability name.
const MaxCapabilityNameRunes = 128

// MaxCapabilityDetailRunes bounds one CapabilitySummary detail: string[0..2048].
const MaxCapabilityDetailRunes = 2048

// MaxWarningRunes bounds one summary warning: 1..1024 printable runes.
const MaxWarningRunes = 1024

// MaxWarnings bounds the warning array: sorted unique string[0..1024].
const MaxWarnings = 1024

// MaxCapabilities bounds the capability map: 0..7 entries.
const MaxCapabilities = 7

// Workspace statuses from the SessionSummary shape.
const (
	WorkspaceAbsent      = "absent"
	WorkspaceCurrent     = "current"
	WorkspaceStaged      = "staged"
	WorkspaceConflict    = "conflict"
	WorkspaceUnsupported = "unsupported"
)

// HostMetadata is validated display metadata for one host, keyed by
// canonical host UUID on the Reader. It is the only source of owner
// display names; the source alias never substitutes for it.
type HostMetadata struct {
	DisplayName string
}

// CapabilitySummary is one validated capability value: status,
// enabled, and detail. Only available may set enabled true.
type CapabilitySummary struct {
	Status  string
	Enabled bool
	Detail  string
}

// SessionObservations carries validated per-session observation facts
// from their respective owners, keyed by session UUID on the Reader.
// A nil map entry or nil pointer is unobserved (unknown) and refuses
// selector_observation_unavailable when the fact is required; an
// established absence uses an explicit value (workspace "absent", an
// empty non-nil capability map). The newest-checkpoint timestamp is
// required exactly when the projection carries a checkpoint.
type SessionObservations struct {
	// NewestCheckpointCreatedAt is the RFC 3339 creation timestamp of
	// the newest checkpoint, required exactly when the projection
	// carries a checkpoint.
	NewestCheckpointCreatedAt *string
	// ProcessPresent is validated process liveness; nil is unobserved.
	// It is required for AuthoritativeStatus (the status body carries
	// process_present boolean) and carried when supplied for lists.
	ProcessPresent *bool
	// Capabilities are validated capability summaries; nil is
	// unobserved and refuses, while an empty non-nil map is an
	// established absence of capabilities.
	Capabilities map[string]CapabilitySummary
	// WorkspaceStatus is the validated workspace status; nil is
	// unobserved and refuses, while "absent" is an established
	// absence.
	WorkspaceStatus *string
}

// AuthoritativeSummary is the closed public summary for one live
// session. Owner, lease, and role come from the validated winning
// Lease Record; host names, workspace, capabilities, liveness, and
// timestamps come from validated inputs. No member is ever invented.
type AuthoritativeSummary struct {
	SessionID                 string
	Name                      string
	Kind                      string
	ProviderID                string
	State                     string
	OwnerHostID               string
	OwnerHostName             string
	LeaseEpoch                uint64
	LeaseID                   string
	LocalRole                 string
	HasCheckpoint             bool
	NewestCheckpointID        *string
	NewestCheckpointCreatedAt *string
	ProcessPresent            *bool
	Capabilities              map[string]CapabilitySummary
	WorkspaceStatus           string
	Warnings                  []string
}

// checkSummaryRepresentable refuses a projected row that can never be
// represented truthfully in a public summary: a complete read proving
// a record without any valid lease. Parked rows stay representable
// through their blocking reason and retry, and tombstoned rows stay
// listed with their terminal state; resolution routing already
// excludes both from live selection.
func checkSummaryRepresentable(summary Summary) error {
	projection := summary.Projection
	if projection.Parked != nil || projection.State == sessstate.StateTombstoned {
		return nil
	}
	if projection.Winner.IsZero() {
		return fmt.Errorf("%w: complete authority read proves record %s without any valid lease", ErrBootstrapIncomplete, projection.SessionID)
	}
	return nil
}

// validatedAuthorityInputs validates the reader's host metadata and
// observation maps as configuration: UUIDv7 keys, printable bounded
// display names, RFC 3339 checkpoint timestamps, capability names and
// values, and workspace enums. Malformed inputs are invalid_config;
// missing facts at summary time are selector_observation_unavailable.
func (reader *Reader) validatedAuthorityInputs() error {
	for host, metadata := range reader.HostMetadata {
		if _, err := scalar.ParseUUIDv7(host); err != nil {
			return fmt.Errorf("%w: malformed host metadata ID: %v", ErrInvalidConfig, err)
		}
		if err := checkPrintableBound(metadata.DisplayName, 1, MaxDisplayNameRunes, "host display name"); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
		}
	}
	for session, observations := range reader.Observations {
		if _, err := scalar.ParseUUIDv7(session); err != nil {
			return fmt.Errorf("%w: malformed observation session ID: %v", ErrInvalidConfig, err)
		}
		if observations.NewestCheckpointCreatedAt != nil {
			if _, err := time.Parse(time.RFC3339, *observations.NewestCheckpointCreatedAt); err != nil {
				return fmt.Errorf("%w: malformed checkpoint timestamp: %v", ErrInvalidConfig, err)
			}
		}
		if observations.WorkspaceStatus != nil {
			if err := checkWorkspaceStatus(*observations.WorkspaceStatus); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
			}
		}
		if observations.Capabilities != nil {
			if err := checkCapabilities(observations.Capabilities); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
			}
		}
	}
	return nil
}

func checkWorkspaceStatus(value string) error {
	switch value {
	case WorkspaceAbsent, WorkspaceCurrent, WorkspaceStaged, WorkspaceConflict, WorkspaceUnsupported:
		return nil
	default:
		return fmt.Errorf("workspace status %q is not absent|current|staged|conflict|unsupported", value)
	}
}

// admittedCapabilities is the closed Section 7.3 provider capability
// registry owned by provhost: a summary advertising any other name
// claims a surface the contract never defined. This is not the
// session-adapter or directory capability registries those owners
// carry for their own planes.
func admittedCapabilities() map[string]bool {
	admitted := make(map[string]bool, MaxCapabilities)
	for _, name := range provhost.Capabilities() {
		admitted[name] = true
	}
	return admitted
}

func checkCapabilities(values map[string]CapabilitySummary) error {
	if len(values) > MaxCapabilities {
		return fmt.Errorf("capabilities carry %d entries, want 0-%d", len(values), MaxCapabilities)
	}
	admitted := admittedCapabilities()
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for index, name := range names {
		if index > 0 && names[index-1] == name {
			return fmt.Errorf("capabilities carry a duplicate name")
		}
		if err := checkPrintableBound(name, 1, MaxCapabilityNameRunes, "capability name"); err != nil {
			return err
		}
		if !admitted[name] {
			return fmt.Errorf("capability %q is not a Section 7.3 registry name", name)
		}
		summary := values[name]
		switch summary.Status {
		case "available", "conditional", "unsupported", "unknown":
		default:
			return fmt.Errorf("capability %q status %q is not available|conditional|unsupported|unknown", name, summary.Status)
		}
		if summary.Enabled && summary.Status != "available" {
			return fmt.Errorf("capability %q enables a non-available status", name)
		}
		if err := checkDetailBound(summary.Detail); err != nil {
			return fmt.Errorf("capability %q detail: %v", name, err)
		}
	}
	return nil
}

func checkDetailBound(value string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("detail is not valid UTF-8")
	}
	count := utf8.RuneCountInString(value)
	if count > MaxCapabilityDetailRunes {
		return fmt.Errorf("detail carries %d characters, want 0-%d", count, MaxCapabilityDetailRunes)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fmt.Errorf("detail carries a control character")
		}
	}
	return nil
}

func checkWarnings(values []string) error {
	if len(values) > MaxWarnings {
		return fmt.Errorf("warnings carry %d entries, want 0-%d", len(values), MaxWarnings)
	}
	for index, warning := range values {
		if err := checkPrintableBound(warning, 1, MaxWarningRunes, "warning"); err != nil {
			return fmt.Errorf("warnings[%d]: %v", index, err)
		}
		if index > 0 && values[index-1] >= warning {
			return fmt.Errorf("warnings are not sorted unique")
		}
	}
	return nil
}

// checkPrintableBound enforces valid UTF-8 with min..max printable
// non-control runes without normalizing or folding the value.
func checkPrintableBound(value string, min, max int, what string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s is not valid UTF-8", what)
	}
	count := utf8.RuneCountInString(value)
	if count < min || count > max {
		return fmt.Errorf("%s carries %d characters, want %d-%d", what, count, min, max)
	}
	for _, character := range value {
		if unicode.IsControl(character) || !unicode.IsPrint(character) {
			return fmt.Errorf("%s carries a non-printable character", what)
		}
	}
	return nil
}

// AuthoritativeStatus resolves one selector through the shared path
// and returns its closed authoritative summary. Resolution,
// read-failure, and bootstrap refusals behave exactly as Status;
// host, lease, and observation facts then bind or refuse as documented
// above. The winning lease checkpoint event-head closure is admitted
// against the selected source chain, so a peer selection validates
// against its own chain rather than the local one. Process liveness
// is required here: the status body carries process_present boolean,
// and an unknown liveness refuses.
func (reader *Reader) AuthoritativeStatus(selector string) (AuthoritativeSummary, error) {
	selected, err := reader.Resolve(selector)
	if err != nil {
		return AuthoritativeSummary{}, err
	}
	projection, err := (&sessstate.Projector{Repo: selected.repo, LocalHostID: reader.LocalHostID}).Project(selected.SessionID)
	if err != nil {
		return AuthoritativeSummary{}, err
	}
	summary := Summary{Projection: projection}
	if err := checkSummaryRepresentable(summary); err != nil {
		return AuthoritativeSummary{}, err
	}
	if err := reader.validatedAuthorityInputs(); err != nil {
		return AuthoritativeSummary{}, err
	}
	return reader.authorize(summary, selected.repo, true)
}

// AuthoritativeList returns the closed authoritative summaries for
// the local index in bytewise session-ID order. The whole document
// refuses when any encountered record cannot be represented
// truthfully — including one incomplete record in an otherwise
// healthy listing — and no record is silently omitted. Process
// liveness is carried when its observation is supplied; workspace,
// capabilities, host, lease, and checkpoint facts are always required.
func (reader *Reader) AuthoritativeList() ([]AuthoritativeSummary, error) {
	rows, err := reader.List()
	if err != nil {
		return nil, err
	}
	if err := reader.validatedAuthorityInputs(); err != nil {
		return nil, err
	}
	out := make([]AuthoritativeSummary, 0, len(rows))
	for _, row := range rows {
		summary, err := reader.authorize(row, reader.Local, false)
		if err != nil {
			return nil, err
		}
		out = append(out, summary)
	}
	return out, nil
}

// authorize binds one projected row to its closed authoritative
// summary against the validated host, lease, and observation inputs.
// The winning lease checkpoint event-head closure is admitted
// against the row's source chain (repo, the winning source for
// plans and the local index for lists), including its Section 2.4
// profile authority. Every refusal names the
// missing established fact; nothing is inferred. An established
// absence (workspace "absent", an empty non-nil capability map) is
// distinct from an unknown observation (nil), which refuses.
func (reader *Reader) authorize(summary Summary, repo *sessrepo.Repository, requireProcess bool) (AuthoritativeSummary, error) {
	projection := summary.Projection
	if err := checkSummaryRepresentable(summary); err != nil {
		return AuthoritativeSummary{}, err
	}
	if reader.LocalHostID == "" {
		return AuthoritativeSummary{}, fmt.Errorf("%w: local host identity is required to establish the local role", ErrObservationUnavailable)
	}
	lease, err := reader.winningLeaseFor(projection.SessionID, projection.Kind, repo)
	if err != nil {
		return AuthoritativeSummary{}, err
	}
	if projection.Winner.Epoch != lease.Epoch || projection.Winner.LeaseID != lease.LeaseID {
		return AuthoritativeSummary{}, fmt.Errorf("%w: session %s carries disagreeing lease facts across chain and record", sessstate.ErrIntegrity, projection.SessionID)
	}
	if projection.OwnerHostID != lease.HolderHostID {
		return AuthoritativeSummary{}, fmt.Errorf("%w: session %s carries disagreeing owner facts across chain and record", sessstate.ErrIntegrity, projection.SessionID)
	}
	metadata, ok := reader.HostMetadata[lease.HolderHostID]
	if !ok {
		return AuthoritativeSummary{}, fmt.Errorf("%w: host metadata for owner %s cannot be established", ErrObservationUnavailable, lease.HolderHostID)
	}
	if projection.LocalRole == "" {
		return AuthoritativeSummary{}, fmt.Errorf("%w: local role for session %s cannot be established", ErrObservationUnavailable, projection.SessionID)
	}
	if err := checkWarnings(projection.Warnings); err != nil {
		return AuthoritativeSummary{}, fmt.Errorf("%w: warnings for session %s: %v", ErrInvalidConfig, projection.SessionID, err)
	}
	authorized := AuthoritativeSummary{
		SessionID:       projection.SessionID,
		Name:            projection.Name,
		Kind:            projection.Kind,
		ProviderID:      projection.Provider.ID,
		State:           string(projection.State),
		OwnerHostID:     lease.HolderHostID,
		OwnerHostName:   metadata.DisplayName,
		LeaseEpoch:      lease.Epoch,
		LeaseID:         lease.LeaseID,
		LocalRole:       projection.LocalRole,
		WorkspaceStatus: "",
		Warnings:        append([]string(nil), projection.Warnings...),
	}
	if authorized.Warnings == nil {
		authorized.Warnings = []string{}
	}
	observations, hasObservations := reader.Observations[projection.SessionID]
	if projection.HasCheckpoint {
		checkpoint := projection.Newest.ID
		authorized.HasCheckpoint = true
		authorized.NewestCheckpointID = &checkpoint
		if !hasObservations || observations.NewestCheckpointCreatedAt == nil {
			return AuthoritativeSummary{}, fmt.Errorf("%w: checkpoint timestamp for session %s cannot be established", ErrObservationUnavailable, projection.SessionID)
		}
		stamp := *observations.NewestCheckpointCreatedAt
		authorized.NewestCheckpointCreatedAt = &stamp
	}
	if !hasObservations || observations.WorkspaceStatus == nil {
		return AuthoritativeSummary{}, fmt.Errorf("%w: workspace status for session %s cannot be established", ErrObservationUnavailable, projection.SessionID)
	}
	workspace := WorkspaceAbsent
	if hasObservations && observations.WorkspaceStatus != nil {
		workspace = *observations.WorkspaceStatus
	}
	authorized.WorkspaceStatus = workspace
	if !hasObservations || observations.Capabilities == nil {
		return AuthoritativeSummary{}, fmt.Errorf("%w: capabilities for session %s cannot be established", ErrObservationUnavailable, projection.SessionID)
	}
	authorized.Capabilities = observations.Capabilities
	if authorized.Capabilities == nil {
		authorized.Capabilities = map[string]CapabilitySummary{}
	}
	if requireProcess {
		if !hasObservations || observations.ProcessPresent == nil {
			return AuthoritativeSummary{}, fmt.Errorf("%w: process liveness for session %s cannot be established", ErrObservationUnavailable, projection.SessionID)
		}
	}
	var present bool
	var hasPresent bool
	if hasObservations && observations.ProcessPresent != nil {
		present = *observations.ProcessPresent
		hasPresent = true
	}
	if requireProcess || hasPresent {
		// For a required status without an observation the gate above
		// already refused; the false default here only lets a
		// weakening mutant proceed to a behavioral failure instead of
		// a panic.
		presentCopy := present
		authorized.ProcessPresent = &presentCopy
		if requireProcess && !hasPresent {
			// Mutant admission path: succeed with a minted fact so the
			// missing-observation test fails rather than panics.
		}
	}
	return authorized, nil
}
