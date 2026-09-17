// SelectionPlan revalidation, shape checks, and tamper-evident
// persistence for the v0.6.0 selector contract (SPEC Section 14.7.2).
//
// Compare order is normative for refusal evidence (every refusal keeps
// the single selector_plan_stale class, except revocation,
// cross-source integrity, and failures): caller shapes, current
// configuration validation, authorization revocation, source binding,
// configuration digest, source read, selection absence, current
// projection, session record, winning lease, the complete authority
// union for the pinned session across the remaining allowed sources,
// authority heads, and the source index digest.
// Fine-grained reasons distinguish the checks without splitting the
// class; tests assert both.
package sessquery

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

// uint53Ceiling is the AX safe-integer ceiling for lease epochs.
const uint53Ceiling = uint64(1<<53 - 1)

// Revalidate compares every bound plan fact against current
// configuration and freshly reread source data, then validates the
// complete authority union for the pinned session UUID across the
// remaining allowed sources. The session is located by its UUID in
// each source, never re-resolved by name. A nil return means the plan
// is current; any error classifies exactly: selector_plan_stale (with
// the first diverged member named), integrity_failure on contradictory
// cross-source immutable records, peer_not_allowlisted on explicit
// authorization revocation, invalid_config on current configuration
// failures, invalid_arguments on a malformed caller plan, and the read
// or projection failure class when a required read or validation
// fails.
func (reader *Reader) Revalidate(plan SelectionPlan) error {
	if err := checkPlanShapes(plan); err != nil {
		return err
	}
	if reader == nil || reader.Local == nil {
		return fmt.Errorf("%w: local repository is required", ErrUnavailable)
	}
	config, err := reader.validatedConfiguration()
	if err != nil {
		return err
	}
	peerHost, isPeer := planSourceBinding(plan, config)
	if isPeer {
		if _, allowed := config.allowed[peerHost]; !allowed {
			return fmt.Errorf("%w: peer %s is not allowlisted", ErrPeerNotAllowlisted, peerHost)
		}
	}
	if !bindingHolds(plan, config, reader) {
		return fmt.Errorf("%w: source binding changed", ErrPlanStale)
	}
	configuration, err := fingerprintConfiguration(reader, config)
	if err != nil {
		return err
	}
	if configuration != plan.ConfigurationDigest {
		return fmt.Errorf("%w: configuration changed", ErrPlanStale)
	}
	var repo = reader.Local
	sourcePeer := ""
	if isPeer {
		repo, sourcePeer, err = reader.peerRepo(peerHost)
		if err != nil {
			return err
		}
	}
	rows, err := reader.readSource(repo, sourcePeer)
	if err != nil {
		return err
	}
	var recordID string
	absent := true
	for _, row := range rows {
		if row.Projection.SessionID == plan.SessionID {
			recordID, absent = row.Projection.RecordID, false
			break
		}
	}
	if absent {
		return fmt.Errorf("%w: selection absent from source index", ErrPlanStale)
	}
	projection, err := (&sessstate.Projector{Repo: repo, LocalHostID: reader.LocalHostID}).Project(plan.SessionID)
	if err != nil {
		return err
	}
	if recordID != plan.SessionRecordID {
		return fmt.Errorf("%w: session record changed", ErrPlanStale)
	}
	if projection.Winner.IsZero() {
		return fmt.Errorf("%w: winning lease changed", ErrPlanStale)
	}
	lease, err := reader.winningLeaseFor(plan.SessionID, projection.Kind, repo)
	if err != nil {
		return err
	}
	if lease.Digest != plan.LeaseRecordID || lease.Epoch != plan.LeaseEpoch || lease.LeaseID != plan.LeaseID || lease.HolderHostID != plan.OwnerHostID {
		return fmt.Errorf("%w: winning lease changed", ErrPlanStale)
	}
	if projection.Winner.Epoch != plan.LeaseEpoch || projection.Winner.LeaseID != plan.LeaseID {
		return fmt.Errorf("%w: winning lease changed", ErrPlanStale)
	}
	if projection.OwnerHostID != lease.HolderHostID {
		return fmt.Errorf("%w: session %s carries disagreeing owner facts across lease and chain", sessstate.ErrIntegrity, plan.SessionID)
	}
	// The union reports specific lease and tombstone divergence before
	// the generic head and index comparisons: heads include union tails,
	// so a union change would otherwise mask as a head change.
	if err := reader.validateAuthorityUnion(plan, config, sourcePeer); err != nil {
		return err
	}
	heads, err := reader.collectAuthorityHeads(config, plan.SessionID, lease.Digest, sourcePeer)
	if err != nil {
		return err
	}
	if !equalStrings(heads, plan.AuthorityHeads) {
		return fmt.Errorf("%w: authority heads changed", ErrPlanStale)
	}
	index, err := fingerprintIndex(reader, repo, rows, sourcePeer)
	if err != nil {
		return err
	}
	if index != plan.SourceIndexDigest {
		return fmt.Errorf("%w: source index changed", ErrPlanStale)
	}
	return nil
}

// validateAuthorityUnion validates the complete current authority for
// the pinned session UUID across every allowed source except the
// already validated plan source. The session is located by UUID in
// each source; names are never re-resolved and no selection is
// substituted. A non-parked copy carrying a disagreeing immutable
// record digest is inconsistent authority (integrity_failure, the same
// class Resolve reports for disagreeing copies). Authoritative
// tombstone evidence or a greater observed winning lease (Section 5.3
// greatest (epoch, lease_id) tuple) in another source invalidates the
// live plan (selector_plan_stale); a lagging copy with a smaller tuple
// is a preserved losing history and stays current, and an agreeing
// replica stays current. A copy that observes no winning lease yet
// cannot contradict the bound lease and is skipped for the lease
// check, while its record digest still binds. A parked copy of the
// pinned session is incomplete authority and fails the union closed
// (selector_observation_unavailable); only a genuinely absent
// session is not contradiction. A source that must be read but
// cannot be read keeps its read-failure class: without the complete
// union the plan cannot be shown current.
func (reader *Reader) validateAuthorityUnion(plan SelectionPlan, config validatedConfig, sourcePeer string) error {
	if sourcePeer != "" {
		rows, err := reader.read(reader.Local)
		if err != nil {
			return err
		}
		if err := checkUnionCopy(plan, rows); err != nil {
			return err
		}
	}
	ids := append([]string(nil), reader.AllowlistedPeerIDs...)
	sort.Strings(ids)
	for index, id := range ids {
		if index > 0 && id == ids[index-1] {
			continue
		}
		if id == sourcePeer {
			continue
		}
		// An allowlisted peer with no advertised learned index
		// contributes no authority; a known handle that cannot be
		// read is a failed observation of a source that must be
		// read, never absence.
		if isAbsentPeerIndex(reader, id) {
			continue
		}
		repo, peerHost, err := reader.peerRepo(id)
		if err != nil {
			return err
		}
		rows, err := reader.readSource(repo, peerHost)
		if err != nil {
			return err
		}
		if err := checkUnionCopy(plan, rows); err != nil {
			return err
		}
	}
	return nil
}

// isAbsentPeerIndex reports an allowlisted peer with no advertised
// learned index handle at all: absence, never a failed read.
func isAbsentPeerIndex(reader *Reader, host string) bool {
	for _, peer := range reader.Peers {
		if peer.HostID == host {
			return false
		}
	}
	return true
}

// checkUnionCopy validates one non-plan source's rows for the pinned
// session UUID as validateAuthorityUnion documents. A parked copy of
// the pinned session is incomplete authority (unknown), never
// evidence and never absence: the union cannot be shown current
// without it, so the plan fails closed with
// selector_observation_unavailable naming the blocking reason. The
// union winner is the greatest (epoch, lease_id) tuple: only a
// greater tuple in another source invalidates the plan; smaller
// tuples are preserved losing histories and equal tuples are
// agreeing replicas.
func checkUnionCopy(plan SelectionPlan, rows []Summary) error {
	for _, row := range rows {
		if row.Projection.SessionID != plan.SessionID {
			continue
		}
		if row.Projection.Parked != nil {
			return fmt.Errorf("%w: required authority for session %s is unknown: %s", ErrObservationUnavailable, plan.SessionID, row.Projection.Parked.BlockingReason)
		}
		if row.Projection.State == sessstate.StateTombstoned {
			return fmt.Errorf("%w: tombstone evidence changed", ErrPlanStale)
		}
		if row.Projection.RecordID != plan.SessionRecordID {
			return fmt.Errorf("%w: session %s carries disagreeing record digests across sources", sessstate.ErrIntegrity, plan.SessionID)
		}
		if row.Projection.Winner.IsZero() {
			continue
		}
		ordering := compareLeaseTuple(
			row.Projection.Winner.Epoch, row.Projection.Winner.LeaseID,
			plan.LeaseEpoch, plan.LeaseID,
		)
		if ordering > 0 {
			return fmt.Errorf("%w: winning lease changed", ErrPlanStale)
		}
		if ordering == 0 && row.Projection.OwnerHostID != plan.OwnerHostID {
			return fmt.Errorf("%w: session %s carries disagreeing owner facts across sources", sessstate.ErrIntegrity, plan.SessionID)
		}
	}
	return nil
}

// collectAuthorityHeads binds the sorted unique lease, event, and
// tombstone heads used for the decision: the winning Lease Record
// digest plus the tail event digest of every allowed source holding a
// non-parked copy of the pinned session. A parked copy of the pinned
// session never reaches this binding: the union gate fails closed
// first, so the skip below only documents that parked rows carry no
// tail; absent sessions contribute none. A source that must be read
// but cannot be read keeps its read-failure class.
func (reader *Reader) collectAuthorityHeads(config validatedConfig, sessionID, leaseDigest, sourcePeer string) ([]string, error) {
	set := map[string]struct{}{leaseDigest: {}}
	collect := func(repo *sessrepo.Repository, peerHost string) error {
		rows, err := reader.readSource(repo, peerHost)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.Projection.SessionID != sessionID {
				continue
			}
			if row.Projection.Parked != nil {
				continue
			}
			tail, err := chainTail(repo, sessionID)
			if err != nil {
				if peerHost == "" {
					return err
				}
				return peerReadFailed(peerHost, err)
			}
			if tail != "" {
				set[tail] = struct{}{}
			}
			break
		}
		return nil
	}
	var planRepo *sessrepo.Repository
	if sourcePeer == "" {
		planRepo = reader.Local
	} else {
		var err error
		planRepo, _, err = reader.peerRepo(sourcePeer)
		if err != nil {
			return nil, err
		}
	}
	if err := collect(planRepo, sourcePeer); err != nil {
		return nil, err
	}
	if sourcePeer != "" {
		if err := collect(reader.Local, ""); err != nil {
			return nil, err
		}
	}
	ids := append([]string(nil), reader.AllowlistedPeerIDs...)
	sort.Strings(ids)
	for index, id := range ids {
		if index > 0 && id == ids[index-1] {
			continue
		}
		if id == sourcePeer {
			continue
		}
		if isAbsentPeerIndex(reader, id) {
			continue
		}
		repo, peerHost, err := reader.peerRepo(id)
		if err != nil {
			return nil, err
		}
		if err := collect(repo, peerHost); err != nil {
			return nil, err
		}
	}
	heads := make([]string, 0, len(set))
	for digest := range set {
		heads = append(heads, digest)
	}
	sort.Strings(heads)
	return heads, nil
}

// planSourceBinding resolves the plan's bound source: a local plan
// carries a null alias with an empty or local host, while any other
// binding selects a peer source by host. The second return reports
// whether the plan selects a peer source.
func planSourceBinding(plan SelectionPlan, config validatedConfig) (string, bool) {
	if plan.SourceAlias == nil && (plan.SourceHostID == "" || plan.SourceHostID == config.localHost) {
		return "", false
	}
	return plan.SourceHostID, true
}

// bindingHolds reports whether the plan's bound (host, alias) source
// still resolves to the same source in the current configuration. A
// local plan holds while its host is empty or the current local host;
// a peer plan holds while its host is still a known peer carrying the
// exact bound alias.
func bindingHolds(plan SelectionPlan, config validatedConfig, reader *Reader) bool {
	if plan.SourceAlias == nil {
		if plan.SourceHostID == "" {
			return true
		}
		if config.localHost != "" && plan.SourceHostID == config.localHost {
			return true
		}
		if knownPeer(reader, plan.SourceHostID) && config.aliasFor(plan.SourceHostID) == "" {
			return true
		}
		return false
	}
	if !knownPeer(reader, plan.SourceHostID) {
		return false
	}
	return config.aliasFor(plan.SourceHostID) == *plan.SourceAlias
}

// equalStrings compares sorted unique string slices element-wise.
func equalStrings(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

// checkPlanShapes validates the caller-supplied plan members without
// consulting any authority: exact contract version, grammatical
// selector, canonical UUIDs and digests, a bound source host, the
// required lease attestation with its full positive triple, exact
// alias bound, sorted unique authority heads, a known action tag, and
// a null or UUIDv7 destination. Authority comes only from Revalidate.
func checkPlanShapes(plan SelectionPlan) error {
	if plan.SelectorVersion != SelectorVersion {
		return fmt.Errorf("%w: unknown selector version %q", ErrInvalidArgument, plan.SelectorVersion)
	}
	if _, err := ParseSelector(plan.Selector); err != nil {
		return fmt.Errorf("%w: malformed plan selector: %v", ErrInvalidArgument, err)
	}
	if _, err := scalar.ParseUUIDv7(plan.SessionID); err != nil {
		return fmt.Errorf("%w: malformed plan session ID: %v", ErrInvalidArgument, err)
	}
	if _, err := scalar.ParseDigest(plan.SessionRecordID); err != nil {
		return fmt.Errorf("%w: malformed plan session record digest: %v", ErrInvalidArgument, err)
	}
	// The source host is always bound: no plan observes a source it
	// cannot name, and empty never substitutes for unknown.
	if _, err := scalar.ParseUUIDv7(plan.SourceHostID); err != nil {
		return fmt.Errorf("%w: malformed plan source host ID: %v", ErrInvalidArgument, err)
	}
	if _, err := scalar.ParseDigest(plan.LeaseRecordID); err != nil {
		return fmt.Errorf("%w: malformed plan lease record digest: %v", ErrInvalidArgument, err)
	}
	if plan.SourceAlias != nil {
		if err := checkAlias(*plan.SourceAlias); err != nil {
			// An alias-less peer binds empty, which never passes the
			// configured 1-64 bound: admit exactly that encoding here.
			if *plan.SourceAlias != "" {
				return fmt.Errorf("%w: malformed plan source alias: %v", ErrInvalidArgument, err)
			}
		}
	}
	for _, member := range []struct {
		name   string
		digest string
	}{
		{"source index", plan.SourceIndexDigest},
		{"configuration", plan.ConfigurationDigest},
		{"expectation", plan.ExpectationDigest},
	} {
		if _, err := scalar.ParseDigest(member.digest); err != nil {
			return fmt.Errorf("%w: malformed plan %s digest: %v", ErrInvalidArgument, member.name, err)
		}
	}
	// The lease triple is always fully bound from the same winning
	// observation as the lease attestation: a positive uint53 epoch
	// with a UUIDv4 lease and a UUIDv7 owner. Empty and partial
	// triples, out-of-range epochs, and placeholders are refused; a
	// session with no observed winning lease builds no plan.
	if plan.LeaseEpoch < 1 || plan.LeaseEpoch > uint53Ceiling {
		return fmt.Errorf("%w: plan lease epoch out of range", ErrInvalidArgument)
	}
	if _, err := scalar.ParseUUIDv4(plan.LeaseID); err != nil {
		return fmt.Errorf("%w: malformed plan lease ID: %v", ErrInvalidArgument, err)
	}
	if _, err := scalar.ParseUUIDv7(plan.OwnerHostID); err != nil {
		return fmt.Errorf("%w: malformed plan owner host ID: %v", ErrInvalidArgument, err)
	}
	for index, head := range plan.AuthorityHeads {
		if _, err := scalar.ParseDigest(head); err != nil {
			return fmt.Errorf("%w: malformed plan authority head: %v", ErrInvalidArgument, err)
		}
		if index > 0 && plan.AuthorityHeads[index-1] >= head {
			return fmt.Errorf("%w: plan authority heads are not sorted unique", ErrInvalidArgument)
		}
	}
	if _, ok := BoundariesFor(plan.Action); !ok {
		return fmt.Errorf("%w: unknown plan action %q", ErrInvalidArgument, plan.Action)
	}
	if plan.DestinationHostID != nil {
		if _, err := scalar.ParseUUIDv7(*plan.DestinationHostID); err != nil {
			return fmt.Errorf("%w: malformed plan destination host ID: %v", ErrInvalidArgument, err)
		}
	}
	return nil
}

// planWire is the tamper-evident persistence encoding: every bound
// member plus the locally verified canonical digest over all of them.
type planWire struct {
	SelectorVersion     string   `json:"selector_version"`
	Selector            string   `json:"selector"`
	SessionID           string   `json:"session_id"`
	SessionRecordID     string   `json:"session_record_id"`
	SourceHostID        string   `json:"source_host_id"`
	SourceAlias         *string  `json:"source_alias"`
	SourceIndexDigest   string   `json:"source_index_digest"`
	ConfigurationDigest string   `json:"configuration_digest"`
	LeaseRecordID       string   `json:"lease_record_id"`
	LeaseEpoch          uint64   `json:"lease_epoch"`
	LeaseID             string   `json:"lease_id"`
	OwnerHostID         string   `json:"owner_host_id"`
	AuthorityHeads      []string `json:"authority_heads"`
	Action              string   `json:"action"`
	DestinationHostID   *string  `json:"destination_host_id"`
	ExpectationDigest   string   `json:"expectation_digest"`
	PlanDigest          string   `json:"plan_digest"`
}

// Digest returns the locally verified canonical digest over all bound
// plan members. It is recomputed on every call and never trusted from
// caller bytes.
func (plan SelectionPlan) Digest() (string, error) {
	heads := append([]string(nil), plan.AuthorityHeads...)
	if heads == nil {
		heads = []string{}
	}
	return fingerprint(map[string]any{
		"action":               plan.Action,
		"authority_heads":      heads,
		"configuration_digest": plan.ConfigurationDigest,
		"destination_host_id":  plan.DestinationHostID,
		"expectation_digest":   plan.ExpectationDigest,
		"lease_epoch":          plan.LeaseEpoch,
		"lease_id":             plan.LeaseID,
		"lease_record_id":      plan.LeaseRecordID,
		"owner_host_id":        plan.OwnerHostID,
		"selector":             plan.Selector,
		"selector_version":     plan.SelectorVersion,
		"session_id":           plan.SessionID,
		"session_record_id":    plan.SessionRecordID,
		"source_alias":         plan.SourceAlias,
		"source_host_id":       plan.SourceHostID,
		"source_index_digest":  plan.SourceIndexDigest,
	})
}

// Marshal encodes the plan with its locally verified digest. The
// encoding keeps null alias and destination distinct from empty.
func (plan SelectionPlan) Marshal() ([]byte, error) {
	digest, err := plan.Digest()
	if err != nil {
		return nil, err
	}
	heads := append([]string(nil), plan.AuthorityHeads...)
	if heads == nil {
		heads = []string{}
	}
	return json.Marshal(planWire{
		SelectorVersion:     plan.SelectorVersion,
		Selector:            plan.Selector,
		SessionID:           plan.SessionID,
		SessionRecordID:     plan.SessionRecordID,
		SourceHostID:        plan.SourceHostID,
		SourceAlias:         plan.SourceAlias,
		SourceIndexDigest:   plan.SourceIndexDigest,
		ConfigurationDigest: plan.ConfigurationDigest,
		LeaseRecordID:       plan.LeaseRecordID,
		LeaseEpoch:          plan.LeaseEpoch,
		LeaseID:             plan.LeaseID,
		OwnerHostID:         plan.OwnerHostID,
		AuthorityHeads:      heads,
		Action:              plan.Action,
		DestinationHostID:   plan.DestinationHostID,
		ExpectationDigest:   plan.ExpectationDigest,
		PlanDigest:          digest,
	})
}

// ParsePlan decodes a persisted plan, validates every member shape,
// and verifies the tamper-evident digest. A verified encoding is still
// not authority: only Revalidate against current facts decides
// currency, and digest equality alone never substitutes for it.
func ParsePlan(raw []byte) (SelectionPlan, error) {
	var wire planWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		return SelectionPlan{}, fmt.Errorf("%w: malformed plan encoding: %v", ErrInvalidArgument, err)
	}
	plan := SelectionPlan{
		SelectorVersion:     wire.SelectorVersion,
		Selector:            wire.Selector,
		SessionID:           wire.SessionID,
		SessionRecordID:     wire.SessionRecordID,
		SourceHostID:        wire.SourceHostID,
		SourceAlias:         wire.SourceAlias,
		SourceIndexDigest:   wire.SourceIndexDigest,
		ConfigurationDigest: wire.ConfigurationDigest,
		LeaseRecordID:       wire.LeaseRecordID,
		LeaseEpoch:          wire.LeaseEpoch,
		LeaseID:             wire.LeaseID,
		OwnerHostID:         wire.OwnerHostID,
		AuthorityHeads:      wire.AuthorityHeads,
		Action:              wire.Action,
		DestinationHostID:   wire.DestinationHostID,
		ExpectationDigest:   wire.ExpectationDigest,
	}
	if plan.AuthorityHeads == nil {
		plan.AuthorityHeads = []string{}
	}
	if err := checkPlanShapes(plan); err != nil {
		return SelectionPlan{}, err
	}
	digest, err := plan.Digest()
	if err != nil {
		return SelectionPlan{}, err
	}
	if digest != wire.PlanDigest {
		return SelectionPlan{}, fmt.Errorf("%w: plan digest mismatch", ErrInvalidArgument)
	}
	return plan, nil
}
