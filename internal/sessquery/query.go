// Package sessquery is the single shared selector API over the session
// repository and state projector: read-only selector resolution
// (Section 14.7.1 literal grammar over Section 2.3 tiers), local
// list/status summaries, and immutable SelectionPlan construction with
// current-fact revalidation (Section 14.7.2). CLI and lifecycle callers
// own invoking this API at every required boundary; no second resolver
// exists. Unknown owner/lease facts and parked diagnostics stay explicit
// in Projection: host display names, checkpoint timestamps/age,
// workspace state, capabilities, process liveness and sync timestamps
// require their respective observation owners and are never inferred
// from a record or a successful read.
package sessquery

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

var (
	ErrUnavailable = errors.New("session index unavailable")
	ErrAmbiguous   = sessrepo.ErrNameAmbiguous
	ErrNotFound    = sessrepo.ErrNameNotFound
	// ErrInvalidArgument refuses a malformed caller argument: selector
	// grammar violations (empty key, unknown source prefix, malformed
	// durable IDs, overlong argument), malformed plan members, unknown
	// action tags, and malformed destinations or expectations. Wire
	// exit binding for a CLI surface belongs to that surface's leaf;
	// this library keeps the refusal class distinct and stable.
	ErrInvalidArgument = errors.New("invalid_arguments")
	// ErrSourceNotFound reports that complete valid configuration has
	// no requested explicit source.
	ErrSourceNotFound = errors.New("selector_source_not_found")
	// ErrPeerNotAllowlisted reports an explicitly selected peer that
	// is configured but not allowlisted, including authorization
	// revoked after a plan was built.
	ErrPeerNotAllowlisted = errors.New("peer_not_allowlisted")
	// ErrSourceReadFailed reports a remote source transport/I-O read
	// failure with no fallback. The underlying cause stays attached
	// for errors.Is, so a repository-path failure remains visible as
	// such. A local source/store I-O failure keeps its repository
	// error, which the CLI Result 5 leaf maps to
	// local_precondition_failed; a partial or malformed index or
	// authority response is integrity failure, not absence.
	ErrSourceReadFailed = errors.New("selector_source_read_failed")
	// ErrInvalidConfig reports duplicate or malformed effective source
	// mappings: duplicate exact aliases or host IDs, a peer host that
	// collides with the local host, malformed host IDs, or aliases
	// outside the configured 1-64 printable-character bound.
	ErrInvalidConfig = errors.New("invalid_config")
	// ErrPlanStale reports bound selection facts that changed after a
	// plan was built; a fresh plan and confirmation are required. The
	// wrapped reason names the first diverged member in compare order.
	ErrPlanStale = errors.New("selector_plan_stale")
	// ErrBootstrapIncomplete reports a complete authority read proving
	// a Session Record without any valid lease: an interrupted
	// persistence prefix for the bootstrap-recovery leaf (Section
	// 14.7.4), never a public summary. A failed read keeps its
	// read-failure class instead; corrupt authority is integrity
	// failure. Wire exit binding for a CLI surface belongs to that
	// surface's leaf; this library keeps the refusal class distinct
	// and stable.
	ErrBootstrapIncomplete = errors.New("selector_bootstrap_incomplete")
	// ErrObservationUnavailable reports a required summary observation
	// or host metadata fact that cannot be established from validated
	// inputs. Nothing is inferred in its place: no source alias stands
	// in for host metadata, and no absent observation mints liveness,
	// capability, workspace, or timestamp facts.
	ErrObservationUnavailable = errors.New("selector_observation_unavailable")
)

// PeerIndex binds a peer identity to its already persisted learned index.
// Transport/authentication and installing peer indexes are upstream owners;
// this reader does not fetch, authenticate, or attest a caller-supplied index.
type PeerIndex struct {
	HostID string
	Repo   *sessrepo.Repository
}

// Reader reads existing indexes. AllowlistedPeerIDs must come from the owning
// configuration layer, separately from the learned indexes. An unlisted source
// is never read. Failed reads from listed sources are errors, never
// empty indexes that let resolution fall through to a lower-priority UUID.
// LocalHostID comes from the local host identity owner; empty preserves unknown.
//
// Aliases carries the exact configured peer-alias mapping (host ID to
// alias) from the same configuration owner, kept separate from the
// learned index handles: a mapping without a handle contributes no
// learned names, and a handle without a mapping stays reachable through
// the bare union and the "id:" host form but never through "peer:".
// Alias comparison is always exact, never folded, trimmed, or decoded.
//
// HostMetadata and Observations carry the validated host and observation
// facts the authoritative summary layer requires, from their respective
// owners. They are consulted only by that layer: List and Status keep
// the internal read model and refuse only the unrepresentable
// record-only bootstrap.
//
// LeaseRecords carries the validated winning-lease authority for plan
// construction and revalidation, from the lease owner. Each entry is
// canonical JSON for one urn:ax:schema:lease Lease Record; the reader
// validates the closed shape and canonical self identity through the
// canonicaljson owner, selects the greatest (epoch, lease_id) winner
// per session, and binds that record digest with its tuple and holder.
// Absence for a session refuses observation_unavailable; malformed
// bytes refuse invalid_config. Envelope observations never substitute
// for these records.
//
// CheckpointRecords carries the validated checkpoint authority the
// winning-lease succession gate requires, from the checkpoint owner.
// Each entry is canonical JSON for one urn:ax:schema:checkpoint
// Checkpoint Record; the reader validates the closed shape and
// canonical self identity through the sessrepo owner. Every successor
// winner must reference an admitted record for its session and
// predecessor lease, carrying the persistence variant the referenced
// Session Record kind selects; an unresolvable, misbound, or
// wrong-variant reference refuses observation_unavailable, and
// malformed bytes refuse invalid_config.
type Reader struct {
	Local              *sessrepo.Repository
	LocalHostID        string
	AllowlistedPeerIDs []string
	Peers              []PeerIndex
	Aliases            map[string]string
	HostMetadata       map[string]HostMetadata
	Observations       map[string]SessionObservations
	LeaseRecords       [][]byte
	CheckpointRecords  [][]byte
}

// Summary contains only facts derived from verified session records and events.
// It is an internal read model, not the closed Section 14.2 CLI SessionSummary.
// Unknown owner/lease and parked diagnostics remain explicit in Projection.
// Host display names, checkpoint timestamps/age, workspace state, capabilities,
// process liveness and sync timestamps require their respective observation
// owners and are deliberately not inferred from a record or a successful read.
type Summary struct {
	Projection sessstate.Projection
}

// List reads local summaries in bytewise session-ID order. Parked sessions stay
// visible with their blocking reason and retry; failure to enumerate the index
// fails the read. No durable state is written, even for an interrupted create.
// A complete read proving a record without any valid lease refuses the whole
// list with selector_bootstrap_incomplete: the interrupted prefix is never
// silently omitted and never returned as a public summary.
func (reader *Reader) List() ([]Summary, error) {
	if reader == nil || reader.Local == nil {
		return nil, fmt.Errorf("%w: local repository is required", ErrUnavailable)
	}
	rows, err := reader.read(reader.Local)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := checkSummaryRepresentable(row); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (reader *Reader) read(repo *sessrepo.Repository) ([]Summary, error) {
	rows, err := repo.ListSessions()
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(rows))
	projector := sessstate.Projector{Repo: repo, LocalHostID: reader.LocalHostID}
	for _, row := range rows {
		projection, err := projector.Project(row.SessionID)
		if err != nil {
			return nil, err
		}
		out = append(out, Summary{Projection: projection})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Projection.SessionID < out[j].Projection.SessionID })
	return out, nil
}

// Status resolves any selector — bare, "id:", or source-qualified — through
// the same path as Resolve, then projects from the selected repository. It
// never treats a failed read as not found. Parked sessions can be inspected
// by ID through InspectLocal, independently of live-name routing, so
// operators retain access to the recovery diagnostics.
func (reader *Reader) Status(selector string) (Summary, error) {
	selected, err := reader.Resolve(selector)
	if err != nil {
		return Summary{}, err
	}
	projection, err := (&sessstate.Projector{Repo: selected.repo, LocalHostID: reader.LocalHostID}).Project(selected.SessionID)
	if err != nil {
		return Summary{}, err
	}
	summary := Summary{Projection: projection}
	if err := checkSummaryRepresentable(summary); err != nil {
		return Summary{}, err
	}
	return summary, nil
}

// InspectLocal reads one local ID, including a parked or tombstoned session.
func (reader *Reader) InspectLocal(sessionID string) (Summary, error) {
	if reader == nil || reader.Local == nil {
		return Summary{}, fmt.Errorf("%w: local repository is required", ErrUnavailable)
	}
	projection, err := (&sessstate.Projector{Repo: reader.Local, LocalHostID: reader.LocalHostID}).Project(sessionID)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Projection: projection}, nil
}

// Selection identifies the session and source index, never an attach or mutation
// plan. PeerHostID is empty for local resolution; it is not the winning owner.
type Selection struct {
	SessionID  string
	PeerHostID string
	repo       *sessrepo.Repository
}

type candidate struct {
	summary   Summary
	selection Selection
}

// Resolve applies one parsed selector through the single shared path:
// grammar first, then effective-configuration validation, then the
// source tier. A bare selector retains the exact Section 2.3 tiers:
// local exact live name, allowlisted peer exact live name, exact UUID,
// not found. An "id:" key bypasses names for the durable UUID in the
// local and allowed-peer union. A qualified selector resolves its key
// in exactly one source index with no fallback. ASCII folding detects
// collisions within each name tier; it does not turn a unique case
// variant into an exact match. Repository-parked entries (failed reads,
// never tombstone evidence) and tombstoned entries (authoritative
// tombstone evidence) are not live candidates. A tier that must be
// read but cannot be read is a read failure, not permission to proceed
// to a lower tier. This is logical index eligibility, not proof of
// process liveness or safe attach/resume readiness; those decisions
// belong to the action planner invoking plans at its boundaries.
func (reader *Reader) Resolve(selector string) (Selection, error) {
	parsed, err := ParseSelector(selector)
	if err != nil {
		return Selection{}, err
	}
	if reader == nil || reader.Local == nil {
		return Selection{}, fmt.Errorf("%w: local repository is required", ErrUnavailable)
	}
	config, err := reader.validatedConfiguration()
	if err != nil {
		return Selection{}, err
	}
	if parsed.Source != SourceBare {
		return reader.resolveExplicit(parsed, config)
	}
	return reader.resolveBare(parsed, config)
}

// resolveBare applies the Section 2.3 tiers over the local and
// allowed-peer union. Unread lower tiers are never consulted after a
// unique higher-tier match; a tier that must be read but cannot be
// read fails the resolution instead of falling through.
func (reader *Reader) resolveBare(parsed Selector, config validatedConfig) (Selection, error) {
	local, err := reader.read(reader.Local)
	if err != nil {
		return Selection{}, err
	}
	localLive := liveCandidates(local, reader.Local, "")
	// Peer tiers read lazily: a unique local name match never depends
	// on lower-tier reads, but any tier that must be read and cannot
	// be read fails instead of falling through.
	peers, peersRead := []candidate(nil), false
	readPeers := func() ([]candidate, error) {
		if !peersRead {
			peersRead = true
			var err error
			peers, err = reader.peerCandidates()
			if err != nil {
				return nil, err
			}
		}
		return peers, nil
	}
	if parsed.IDKey {
		union, err := readPeers()
		if err != nil {
			return Selection{}, err
		}
		return selectIdentity(strings.TrimPrefix(parsed.Key, "id:"), append(localLive, union...), config)
	}
	if match, found, err := matchName(localLive, parsed.Key); found || err != nil {
		return match, err
	}
	union, err := readPeers()
	if err != nil {
		return Selection{}, err
	}
	if match, found, err := matchName(union, parsed.Key); found || err != nil {
		if err != nil {
			return Selection{}, err
		}
		if err := checkRecordAgreement(append(append([]candidate(nil), localLive...), union...), match); err != nil {
			return Selection{}, err
		}
		return match, nil
	}
	if parsed.UUIDShaped {
		return selectIdentity(parsed.Key, append(localLive, union...), config)
	}
	return Selection{}, fmt.Errorf("%w: no live session named %q", ErrNotFound, parsed.Raw)
}

func liveCandidates(rows []Summary, repo *sessrepo.Repository, peer string) []candidate {
	out := make([]candidate, 0, len(rows))
	for _, row := range rows {
		if row.Projection.Parked != nil || row.Projection.State == sessstate.StateTombstoned {
			continue
		}
		out = append(out, candidate{row, Selection{row.Projection.SessionID, peer, repo}})
	}
	return out
}

// resolveExplicit selects a key in exactly one configured source index.
// An explicit source never falls back to the local index, another peer,
// a cached index from another source, or a union search, even when the
// same name or UUID exists there: a miss in the selected source is not
// found, and a source that cannot be completely read is a read failure.
func (reader *Reader) resolveExplicit(parsed Selector, config validatedConfig) (Selection, error) {
	repo, peerHost, err := reader.locateSource(parsed, config)
	if err != nil {
		return Selection{}, err
	}
	rows, err := reader.readSource(repo, peerHost)
	if err != nil {
		return Selection{}, err
	}
	live := liveCandidates(rows, repo, peerHost)
	if parsed.IDKey {
		return selectIdentity(strings.TrimPrefix(parsed.Key, "id:"), live, config)
	}
	if match, found, err := matchName(live, parsed.Key); found || err != nil {
		return match, err
	}
	if parsed.UUIDShaped {
		return selectIdentity(parsed.Key, live, config)
	}
	return Selection{}, fmt.Errorf("%w: no live session named %q in source %s", ErrNotFound, parsed.Key, describeSource(parsed))
}

// locateSource maps an explicit source to its index handle. An explicit
// unknown alias or host is selector_source_not_found; a known but
// disallowed peer is peer_not_allowlisted. The local source is always
// admitted and needs no allowlist entry.
func (reader *Reader) locateSource(parsed Selector, config validatedConfig) (*sessrepo.Repository, string, error) {
	switch parsed.Source {
	case SourceLocal:
		return reader.Local, "", nil
	case SourcePeerAlias:
		host, ok := config.hostForAlias(reader, parsed.SourceValue)
		if !ok {
			return nil, "", fmt.Errorf("%w: no configured source for alias %q", ErrSourceNotFound, parsed.SourceValue)
		}
		if _, allowed := config.allowed[host]; !allowed {
			return nil, "", fmt.Errorf("%w: peer %s is not allowlisted", ErrPeerNotAllowlisted, host)
		}
		return reader.peerRepo(host)
	case SourcePeerHost:
		if config.localHost != "" && parsed.SourceValue == config.localHost {
			return reader.Local, "", nil
		}
		if !knownPeer(reader, parsed.SourceValue) {
			return nil, "", fmt.Errorf("%w: no configured source for host %s", ErrSourceNotFound, parsed.SourceValue)
		}
		if _, allowed := config.allowed[parsed.SourceValue]; !allowed {
			return nil, "", fmt.Errorf("%w: peer %s is not allowlisted", ErrPeerNotAllowlisted, parsed.SourceValue)
		}
		return reader.peerRepo(parsed.SourceValue)
	default:
		return nil, "", fmt.Errorf("%w: unknown selector source", ErrInvalidArgument)
	}
}

// knownPeer reports whether a host carries a configured learned index,
// regardless of allowlisting or readability.
func knownPeer(reader *Reader, host string) bool {
	for _, peer := range reader.Peers {
		if peer.HostID == host {
			return true
		}
	}
	return false
}

// peerRepo binds one allowlisted peer host to its learned index handle.
// A peer with no advertised index cannot be completely read, so
// explicit selection refuses with the read-failure class instead of
// treating the source as absent; union tiers simply learn nothing from
// it. A listed index with an unavailable repository is the same failed
// observation.
func (reader *Reader) peerRepo(host string) (*sessrepo.Repository, string, error) {
	seen := false
	var repo *sessrepo.Repository
	for _, peer := range reader.Peers {
		if peer.HostID != host {
			continue
		}
		if seen {
			return nil, "", fmt.Errorf("%w: multiple indexes for peer %s", ErrUnavailable, host)
		}
		seen = true
		repo = peer.Repo
	}
	if !seen {
		return nil, "", fmt.Errorf("%w: peer %s advertises no learned index", ErrSourceReadFailed, host)
	}
	if repo == nil {
		return nil, "", peerReadFailed(host, fmt.Errorf("%w: unreadable index for peer %s", ErrUnavailable, host))
	}
	return repo, host, nil
}

// readSource completes one source read. Local I-O failures keep their
// repository error; remote failures carry the read-failure class while
// preserving the underlying cause for errors.Is.
func (reader *Reader) readSource(repo *sessrepo.Repository, peerHost string) ([]Summary, error) {
	rows, err := reader.read(repo)
	if err != nil {
		if peerHost == "" {
			return nil, err
		}
		return nil, peerReadFailed(peerHost, err)
	}
	return rows, nil
}

// peerReadFailed marks a remote-source read failure with its contract
// class while preserving the underlying cause: a failed read is never
// an absence that lets resolution fall through to another source.
func peerReadFailed(host string, err error) error {
	return errors.Join(fmt.Errorf("%w: peer %s index read failed", ErrSourceReadFailed, host), err)
}

// describeSource names the selected source for refusal evidence: exact
// literal alias or host, never a display name.
func describeSource(parsed Selector) string {
	switch parsed.Source {
	case SourceLocal:
		return "local"
	case SourcePeerAlias:
		return fmt.Sprintf("peer:%s", parsed.SourceValue)
	case SourcePeerHost:
		return fmt.Sprintf("id:%s", parsed.SourceValue)
	default:
		return "bare"
	}
}

// selectIdentity selects a durable session UUID from live candidates
// independent of names: the "id:" key form and the UUID tier. Copies
// of one UUID deduplicate only when their immutable record digests
// agree; disagreeing copies are inconsistent authority
// (integrity_failure), not an ambiguity. Identical copies select the
// bytewise smallest holder host.
func selectIdentity(uuid string, pool []candidate, config validatedConfig) (Selection, error) {
	id, err := scalar.ParseUUIDv7(uuid)
	if err != nil {
		return Selection{}, fmt.Errorf("%w: malformed session UUID: %v", ErrInvalidArgument, err)
	}
	var matches []candidate
	for _, item := range pool {
		if item.selection.SessionID == id.String() {
			matches = append(matches, item)
		}
	}
	if len(matches) == 0 {
		return Selection{}, fmt.Errorf("%w: no live session with ID %q", ErrNotFound, id.String())
	}
	for _, item := range matches[1:] {
		if item.summary.Projection.RecordID != matches[0].summary.Projection.RecordID {
			return Selection{}, fmt.Errorf("%w: session %s carries disagreeing record digests across sources", sessstate.ErrIntegrity, id.String())
		}
	}
	best := matches[0]
	for _, item := range matches[1:] {
		if holderHost(item.selection.PeerHostID, config.localHost) < holderHost(best.selection.PeerHostID, config.localHost) {
			best = item
		}
	}
	return best.selection, nil
}

// holderHost orders copy holders bytewise for the deterministic tie
// break. PeerHostID is empty for local copies; while the local host
// identity is unknown the local copy sorts before every peer,
// preserving local priority there.
func holderHost(peerHost, localHost string) string {
	if peerHost == "" {
		if localHost == "" {
			return ""
		}
		return localHost
	}
	return peerHost
}

// checkRecordAgreement refuses inconsistent authority behind a name
// match: every consulted live copy of the won UUID must carry the same
// immutable record digest. Replicated identical copies stay one
// identity; disagreeing copies are integrity failure, not ambiguity.
func checkRecordAgreement(pool []candidate, won Selection) error {
	digest := ""
	seen := false
	for _, item := range pool {
		if item.selection.SessionID != won.SessionID {
			continue
		}
		if !seen {
			digest, seen = item.summary.Projection.RecordID, true
			continue
		}
		if item.summary.Projection.RecordID != digest {
			return fmt.Errorf("%w: session %s carries disagreeing record digests across sources", sessstate.ErrIntegrity, won.SessionID)
		}
	}
	return nil
}

func (reader *Reader) peerCandidates() ([]candidate, error) {
	ids := append([]string(nil), reader.AllowlistedPeerIDs...)
	sort.Strings(ids)
	var out []candidate
	for index, id := range ids {
		if index > 0 && id == ids[index-1] {
			continue
		}
		var repo *sessrepo.Repository
		found := false
		for _, peer := range reader.Peers {
			if peer.HostID != id {
				continue
			}
			if found {
				return nil, fmt.Errorf("%w: multiple indexes for peer %s", ErrUnavailable, id)
			}
			found = true
			repo = peer.Repo
		}
		// An allowlisted peer with no learned index contributes no learned names.
		// A listed index with an unavailable repository is a failed observation.
		if !found {
			continue
		}
		if repo == nil {
			return nil, peerReadFailed(id, fmt.Errorf("%w: unreadable index for peer %s", ErrUnavailable, id))
		}
		rows, err := reader.read(repo)
		if err != nil {
			return nil, peerReadFailed(id, err)
		}
		out = append(out, liveCandidates(rows, repo, id)...)
	}
	return out, nil
}

func matchName(items []candidate, selector string) (Selection, bool, error) {
	// A replicated session is one identity, regardless of how many peers learned
	// it. Distinct session IDs in the folded bucket are an ambiguity.
	ids := make(map[string]struct{})
	var exact *Selection
	for _, item := range items {
		if asciiFold(item.summary.Projection.Name) != asciiFold(selector) {
			continue
		}
		ids[item.selection.SessionID] = struct{}{}
		if item.summary.Projection.Name == selector && exact == nil {
			selection := item.selection
			exact = &selection
		}
	}
	if len(ids) > 1 {
		return Selection{}, false, fmt.Errorf("%w: name %q collides across %d sessions", ErrAmbiguous, selector, len(ids))
	}
	if exact != nil {
		return *exact, true, nil
	}
	return Selection{}, false, nil
}

func asciiFold(value string) string {
	folded := []byte(value)
	for index, character := range folded {
		if character >= 'A' && character <= 'Z' {
			folded[index] += 'a' - 'A'
		}
	}
	return string(folded)
}
