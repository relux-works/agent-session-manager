// Validated Lease Record admission for the v0.6.0 selector contract
// (SPEC Sections 5.3 and 14.7.2).
//
// The reader carries canonical Lease Record bytes from the lease owner
// in Reader.LeaseRecords. Each record is validated through the
// canonicaljson owner (closed shape plus canonical self identity), and
// the greatest (epoch, lease_id) winner per session supplies the plan
// lease_record_id digest with its epoch, lease ID, and holder. Envelope
// triples observed on event chains never substitute for these records:
// a session with no admitted winning record refuses
// selector_observation_unavailable, and malformed bytes refuse
// invalid_config.
package sessquery

import (
	"encoding/json"
	"fmt"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
	"github.com/relux-works/agent-session-manager/internal/sessstate"
)

const leaseRecordSchema = "urn:ax:schema:lease"

// validatedLease is one admitted Lease Record: its canonical digest and
// the members the selector binds.
type validatedLease struct {
	Digest         string
	SessionID      string
	LeaseID        string
	Epoch          uint64
	HolderHostID   string
	Predecessor    string
	HasPredecessor bool
	Checkpoint     string
	HasCheckpoint  bool
	Raw            []byte
}

// checkpointRecordSchema is the exact Checkpoint Record schema the
// winning-lease checkpoint gate admits.
const checkpointRecordSchema = "urn:ax:schema:checkpoint"

// validatedCheckpoint is the parsed canonical Checkpoint Record shape,
// not yet an admitted authority capability: it carries the digest with
// the session and lease it was taken under, plus the host that created
// it. Section 5.4 binds lease_epoch/lease_id to the
// referenced winning lease, so a successor's checkpoint names its
// predecessor lease, and binds created_by_host_id to the current
// lease holder: the holder of the lease the checkpoint names. The
// persistence presence members carry which variant the record uses;
// the referenced Session Record selects which variant is valid
// (checkCheckpointPersistence), because canonical shape alone admits
// either variant. EventHeads carries the authoritative event DAG
// heads immediately before the checkpoint object; every required
// checkpoint must resolve each head to a chained event for its
// session at or before its bound lease
// (checkCheckpointEventHeads), because a canonical checkpoint
// naming an absent event or an event under a later lease cannot
// establish Section 5.3/14.7.2 current complete authority. The same
// closure must then admit the Section 2.4 profile authority
// (checkCheckpointProfileAuthority), because a canonical checkpoint
// over a corrupt profile history cannot fix the effective profile.
type validatedCheckpoint struct {
	Digest              string
	SessionID           string
	LeaseEpoch          uint64
	LeaseID             string
	CreatorHostID       string
	HasProviderManifest bool
	HasTaskBoardBundle  bool
	EventHeads          []string
	Raw                 []byte
}

// checkpointSealToken is deliberately unexported and is created only by the
// free admitCheckpoint constructor. The type-level census protects that
// construction boundary across every production source file; the token then
// makes a zero or field-assembled admittedCheckpoint unusable at runtime.
type checkpointSealToken struct{}

// checkpointSeal holds the facts that profile derivation may consume after
// semantic admission. Its fields are private so callers cannot treat the
// admitted representation as a public record-shaped DTO.
type checkpointSeal struct {
	token               *checkpointSealToken
	digest              string
	sessionID           string
	leaseEpoch          uint64
	leaseID             string
	creatorHostID       string
	hasProviderManifest bool
	hasTaskBoardBundle  bool
	eventHeads          []string
}

// admittedCheckpoint is the only checkpoint representation that profile
// derivation may consume. A validatedCheckpoint is still a loader/admission
// concern: canonical parsing alone does not establish its session, lease,
// holder, persistence, event-head, or temporal authority. The unexported seal
// is the capability; without its constructor-only token the value is not
// admitted, even when a caller assembles the outer struct manually.
type admittedCheckpoint struct {
	seal *checkpointSeal
}

func (checkpoint admittedCheckpoint) authority() (*checkpointSeal, error) {
	if checkpoint.seal == nil || checkpoint.seal.token == nil {
		return nil, fmt.Errorf("%w: checkpoint authority seal is missing", ErrObservationUnavailable)
	}
	return checkpoint.seal, nil
}

// checkpointAdmission is the narrow capability handed to profile derivation
// for a session.resumed reference. The derivation path can request a named
// checkpoint, but it cannot obtain or consume a raw validated record.
type checkpointAdmission func(checkpointID, sessionID string, summary sessrepo.EventSummary) (admittedCheckpoint, error)

// parseLeaseRecord validates one canonical Lease Record through the
// canonicaljson owner and extracts the bound members. Closed-shape and
// identity failures refuse invalid_config; the caller maps absence to
// observation_unavailable.
func parseLeaseRecord(raw []byte) (validatedLease, error) {
	var lease validatedLease
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return lease, fmt.Errorf("%w: decode lease record frame: %v", ErrInvalidConfig, fault)
	}
	schema, err := leaseStringMember(members, "schema")
	if err != nil || schema != leaseRecordSchema {
		return lease, fmt.Errorf("%w: lease record schema %q, want %q", ErrInvalidConfig, schema, leaseRecordSchema)
	}
	digest, field, err := sessrepo.AttestLeaseRecord(raw)
	if err != nil {
		return lease, fmt.Errorf("%w: verify lease record identity: %v", ErrInvalidConfig, err)
	}
	if field != canonicaljson.SelfRecordID {
		return lease, fmt.Errorf("%w: lease record identity field %q, want record_id", ErrInvalidConfig, string(field))
	}
	sessionID, ok := environ.CheckUUIDv7(leaseMemberRaw(members, "session_id"))
	if !ok {
		return lease, fmt.Errorf("%w: lease record carries no valid session_id member", ErrInvalidConfig)
	}
	leaseID, err := leaseStringMember(members, "lease_id")
	if err != nil {
		return lease, fmt.Errorf("%w: lease record carries no lease_id member: %v", ErrInvalidConfig, err)
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return lease, fmt.Errorf("%w: lease record lease_id: %v", ErrInvalidConfig, err)
	}
	epoch, ok := environ.CheckUint53Bounds(leaseMemberRaw(members, "epoch"), 1, uint53Ceiling)
	if !ok {
		return lease, fmt.Errorf("%w: lease record carries no epoch at or above 1", ErrInvalidConfig)
	}
	holder, ok := environ.CheckUUIDv7(leaseMemberRaw(members, "holder_host_id"))
	if !ok {
		return lease, fmt.Errorf("%w: lease record carries no valid holder_host_id member", ErrInvalidConfig)
	}
	predecessor, hasPredecessor, err := leaseNullableUUIDv4(members, "predecessor_lease_id")
	if err != nil {
		return lease, fmt.Errorf("%w: lease record predecessor_lease_id: %v", ErrInvalidConfig, err)
	}
	checkpoint, hasCheckpoint, err := leaseNullableDigest(members, "checkpoint_id")
	if err != nil {
		return lease, fmt.Errorf("%w: lease record checkpoint_id: %v", ErrInvalidConfig, err)
	}
	lease = validatedLease{
		Digest:         digest.String(),
		SessionID:      sessionID.String(),
		LeaseID:        leaseID,
		Epoch:          epoch,
		HolderHostID:   holder.String(),
		Predecessor:    predecessor,
		HasPredecessor: hasPredecessor,
		Checkpoint:     checkpoint,
		HasCheckpoint:  hasCheckpoint,
		Raw:            append([]byte(nil), raw...),
	}
	return lease, nil
}

func leaseMemberRaw(members map[string]json.RawMessage, name string) json.RawMessage {
	return members[name]
}

func leaseStringMember(members map[string]json.RawMessage, name string) (string, error) {
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

func leaseNullableUUIDv4(members map[string]json.RawMessage, name string) (string, bool, error) {
	raw, ok := members[name]
	if !ok {
		return "", false, fmt.Errorf("member %q is absent", name)
	}
	if string(raw) == "null" {
		return "", false, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false, fmt.Errorf("member %q is neither a UUID nor null", name)
	}
	if _, err := scalar.ParseUUIDv4(value); err != nil {
		return "", false, err
	}
	return value, true, nil
}

// leaseNullableDigest reads a nullable digest member: null binds
// absence, a string must parse as a canonical digest.
func leaseNullableDigest(members map[string]json.RawMessage, name string) (string, bool, error) {
	raw, ok := members[name]
	if !ok {
		return "", false, fmt.Errorf("member %q is absent", name)
	}
	if string(raw) == "null" {
		return "", false, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false, fmt.Errorf("member %q is neither a digest nor null", name)
	}
	digest, err := scalar.ParseDigest(value)
	if err != nil {
		return "", false, err
	}
	return digest.String(), true, nil
}

// parseCheckpointRecord validates one canonical Checkpoint Record
// through the sessrepo owner (closed shape plus canonical self
// identity) and extracts the bound members. Closed-shape and
// identity failures refuse invalid_config; the caller maps an
// unresolvable reference to observation_unavailable.
func parseCheckpointRecord(raw []byte) (validatedCheckpoint, error) {
	var checkpoint validatedCheckpoint
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return checkpoint, fmt.Errorf("%w: decode checkpoint record frame: %v", ErrInvalidConfig, fault)
	}
	schema, err := leaseStringMember(members, "schema")
	if err != nil || schema != checkpointRecordSchema {
		return checkpoint, fmt.Errorf("%w: checkpoint record schema %q, want %q", ErrInvalidConfig, schema, checkpointRecordSchema)
	}
	digest, field, err := sessrepo.AttestCheckpointRecord(raw)
	if err != nil {
		return checkpoint, fmt.Errorf("%w: verify checkpoint record identity: %v", ErrInvalidConfig, err)
	}
	if field != canonicaljson.SelfCheckpointID {
		return checkpoint, fmt.Errorf("%w: checkpoint record identity field %q, want checkpoint_id", ErrInvalidConfig, string(field))
	}
	sessionID, ok := environ.CheckUUIDv7(leaseMemberRaw(members, "session_id"))
	if !ok {
		return checkpoint, fmt.Errorf("%w: checkpoint record carries no valid session_id member", ErrInvalidConfig)
	}
	epoch, ok := environ.CheckUint53Bounds(leaseMemberRaw(members, "lease_epoch"), 1, uint53Ceiling)
	if !ok {
		return checkpoint, fmt.Errorf("%w: checkpoint record carries no lease_epoch at or above 1", ErrInvalidConfig)
	}
	leaseID, err := leaseStringMember(members, "lease_id")
	if err != nil {
		return checkpoint, fmt.Errorf("%w: checkpoint record carries no lease_id member: %v", ErrInvalidConfig, err)
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return checkpoint, fmt.Errorf("%w: checkpoint record lease_id: %v", ErrInvalidConfig, err)
	}
	creator, ok := environ.CheckUUIDv7(leaseMemberRaw(members, "created_by_host_id"))
	if !ok {
		return checkpoint, fmt.Errorf("%w: checkpoint record carries no valid created_by_host_id member", ErrInvalidConfig)
	}
	_, hasProvider, err := leaseNullableDigest(members, "provider_manifest_id")
	if err != nil {
		return checkpoint, fmt.Errorf("%w: checkpoint record provider_manifest_id: %v", ErrInvalidConfig, err)
	}
	_, hasBoard, err := leaseNullableDigest(members, "task_board_bundle_id")
	if err != nil {
		return checkpoint, fmt.Errorf("%w: checkpoint record task_board_bundle_id: %v", ErrInvalidConfig, err)
	}
	heads, err := checkpointEventHeads(members)
	if err != nil {
		return checkpoint, fmt.Errorf("%w: checkpoint record event_heads: %v", ErrInvalidConfig, err)
	}
	checkpoint = validatedCheckpoint{
		Digest:              digest.String(),
		SessionID:           sessionID.String(),
		LeaseEpoch:          epoch,
		LeaseID:             leaseID,
		CreatorHostID:       creator.String(),
		HasProviderManifest: hasProvider,
		HasTaskBoardBundle:  hasBoard,
		EventHeads:          heads,
		Raw:                 append([]byte(nil), raw...),
	}
	return checkpoint, nil
}

// checkpointEventHeads extracts the sorted unique event-head digests
// the canonical checkpoint owner already shape-validated (1..64
// entries). Grammar failures refuse invalid_config; resolution
// against the session chain belongs to checkCheckpointEventHeads.
func checkpointEventHeads(members map[string]json.RawMessage) ([]string, error) {
	raw, ok := members["event_heads"]
	if !ok {
		return nil, fmt.Errorf("member %q is absent", "event_heads")
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("member %q is not a string array: %v", "event_heads", err)
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		digest, err := scalar.ParseDigest(value)
		if err != nil {
			return nil, fmt.Errorf("member %q holds malformed digest %q: %v", "event_heads", value, err)
		}
		out = append(out, digest.String())
	}
	return out, nil
}

// validatedLeasesBySession validates every supplied Lease Record and
// groups the admitted records by session UUID.
func (reader *Reader) validatedLeasesBySession() (map[string][]validatedLease, error) {
	out := map[string][]validatedLease{}
	if reader == nil {
		return out, fmt.Errorf("%w: reader carries no configuration", ErrInvalidConfig)
	}
	for _, raw := range reader.LeaseRecords {
		lease, err := parseLeaseRecord(raw)
		if err != nil {
			return nil, err
		}
		out[lease.SessionID] = append(out[lease.SessionID], lease)
	}
	return out, nil
}

// winningLeaseFor selects the greatest (epoch, lease_id) validated Lease
// Record for one session and validates its succession: the
// predecessor ancestry must chain by epoch plus one down to an
// epoch-1 root, and every lease in that winning ancestry above
// epoch 1 must reference a validated checkpoint for its session and
// predecessor lease (Section 5.3), each created by the owning lease
// holder (Section 5.4). Every required checkpoint must additionally
// carry the persistence variant the referenced Session Record
// selects: sessionKind is that record's kind as projected from the
// winning source, and a variant mismatch refuses
// observation_unavailable even when canonical identity, session,
// lease, epoch, and creator all agree. Every required checkpoint
// must also resolve each event head to a chained event for its
// session at or before its bound lease in the winning source chain
// (repo, Section 5.4 event-head closure): a head naming an absent
// event or an event under a later lease cannot establish current
// complete authority, so it refuses observation_unavailable;
// historical heads stay admissible and are never required to equal
// the current tail. Every required checkpoint must also admit its
// Section 2.4 profile authority over the same closure
// (checkCheckpointProfileAuthority): a first launch outside the
// creation pair, a later launch outside the newest authoritative
// change at or before it, a resume outside its referenced
// checkpoint's closure pair, or a fork outside the new record
// pair, contradicts the admitted history (integrity failure).
// Absence refuses observation_unavailable; a
// lease above epoch 1 without its named predecessor in the admitted
// set refuses the same class rather than binding a dangling
// successor, while an illegal predecessor link is malformed
// authority (integrity failure).
func (reader *Reader) winningLeaseFor(sessionID, sessionKind string, repo *sessrepo.Repository) (validatedLease, error) {
	if sessionKind != "direct" && sessionKind != "task_board" {
		return validatedLease{}, fmt.Errorf("%w: session %s carries unknown kind %q for checkpoint persistence", ErrInvalidConfig, sessionID, sessionKind)
	}
	bySession, err := reader.validatedLeasesBySession()
	if err != nil {
		return validatedLease{}, err
	}
	candidates := bySession[sessionID]
	if len(candidates) == 0 {
		return validatedLease{}, fmt.Errorf("%w: winning lease record for session %s cannot be established", ErrObservationUnavailable, sessionID)
	}
	winner := candidates[0]
	for _, candidate := range candidates[1:] {
		if sessstate.Compare(
			sessstate.LeaseHead{Epoch: candidate.Epoch, LeaseID: candidate.LeaseID},
			sessstate.LeaseHead{Epoch: winner.Epoch, LeaseID: winner.LeaseID},
		) > 0 {
			winner = candidate
		}
	}
	chain, predecessor, err := checkLeaseChain(candidates, winner)
	if err != nil {
		return validatedLease{}, err
	}
	if err := reader.checkWinnerCheckpoint(winner, predecessor, sessionKind, repo, chain); err != nil {
		return validatedLease{}, err
	}
	if err := reader.checkAncestorCheckpoints(chain, sessionKind, repo); err != nil {
		return validatedLease{}, err
	}
	return winner, nil
}

// checkLeaseChain validates the winner's predecessor ancestry within
// the admitted records for its session. Section 5.3 requires every
// epoch above 1 to name a known predecessor for the same session
// with an epoch exactly one below, down to an epoch-1 root with a
// null predecessor; epochs below 1 cannot exist, so an epoch-1
// lease naming any predecessor is malformed. A self, cyclic,
// skipped, contradictory, or duplicated link is malformed authority
// (integrity failure); a missing predecessor name or record is
// incomplete authority (observation_unavailable). Epochs strictly
// decrease along the walk, so the walk always terminates. It
// returns the winning ancestry from the winner down to the epoch-1
// root, plus the winner's direct predecessor, or the zero lease at
// epoch 1.
func checkLeaseChain(candidates []validatedLease, winner validatedLease) ([]validatedLease, validatedLease, error) {
	byLease := make(map[string]validatedLease, len(candidates))
	for _, candidate := range candidates {
		if previous, ok := byLease[candidate.LeaseID]; ok && previous.Digest != candidate.Digest {
			return nil, validatedLease{}, fmt.Errorf("%w: session %s carries contradictory lease records for lease %s", sessstate.ErrIntegrity, winner.SessionID, candidate.LeaseID)
		}
		byLease[candidate.LeaseID] = candidate
	}
	var predecessor validatedLease
	var chain []validatedLease
	current := winner
	for {
		chain = append(chain, current)
		if current.Epoch == 1 {
			if current.HasPredecessor {
				return nil, validatedLease{}, fmt.Errorf("%w: session %s epoch-1 lease %s names a predecessor", sessstate.ErrIntegrity, current.SessionID, current.LeaseID)
			}
			return chain, predecessor, nil
		}
		if !current.HasPredecessor {
			return nil, validatedLease{}, fmt.Errorf("%w: winning lease record for session %s names no predecessor", ErrObservationUnavailable, current.SessionID)
		}
		parent, ok := byLease[current.Predecessor]
		if !ok {
			return nil, validatedLease{}, fmt.Errorf("%w: predecessor lease record for session %s cannot be established", ErrObservationUnavailable, current.SessionID)
		}
		if parent.Epoch+1 != current.Epoch {
			return nil, validatedLease{}, fmt.Errorf("%w: session %s lease %s at epoch %d does not succeed its predecessor %s at epoch %d", sessstate.ErrIntegrity, current.SessionID, current.LeaseID, current.Epoch, parent.LeaseID, parent.Epoch)
		}
		if current.LeaseID == winner.LeaseID {
			predecessor = parent
		}
		current = parent
	}
}

// validatedCheckpointsByDigest validates every supplied Checkpoint
// Record and indexes the parsed records by canonical digest. Semantic
// admission happens only at the shared admission owners below.
func (reader *Reader) validatedCheckpointsByDigest() (map[string]validatedCheckpoint, error) {
	out := map[string]validatedCheckpoint{}
	if reader == nil {
		return out, fmt.Errorf("%w: reader carries no configuration", ErrInvalidConfig)
	}
	for _, raw := range reader.CheckpointRecords {
		checkpoint, err := parseCheckpointRecord(raw)
		if err != nil {
			return nil, err
		}
		out[checkpoint.Digest] = checkpoint
	}
	return out, nil
}

// checkWinnerCheckpoint resolves the winner's checkpoint reference
// against the admitted Checkpoint Records. Section 5.3 requires
// every epoch above 1 to reference a validated checkpoint for its
// session and predecessor lease; an epoch-1 root without a
// predecessor may carry no checkpoint before the first provider
// boundary exists. A non-null reference must resolve to an admitted
// record for the same session bound to the lease it was taken
// under: the predecessor for a successor, the winner itself at
// epoch 1 (Section 5.4 binds a checkpoint to its winning lease). An
// unresolvable reference is incomplete authority
// (observation_unavailable), never absence or a minted fact. The
// admitted checkpoint must also resolve its event-head closure in
// the winning source chain (repo); see checkCheckpointEventHeads.
// The same closure must then admit its Section 2.4 profile
// authority; see checkCheckpointProfileAuthority.
func (reader *Reader) checkWinnerCheckpoint(winner validatedLease, predecessor validatedLease, sessionKind string, repo *sessrepo.Repository, chain []validatedLease) error {
	if !winner.HasCheckpoint {
		if winner.Epoch == 1 && !winner.HasPredecessor {
			return nil
		}
		return fmt.Errorf("%w: validated checkpoint for session %s lease %s cannot be established", ErrObservationUnavailable, winner.SessionID, winner.LeaseID)
	}
	checkpoints, err := reader.validatedCheckpointsByDigest()
	if err != nil {
		return err
	}
	checkpoint, ok := checkpoints[winner.Checkpoint]
	if !ok {
		return fmt.Errorf("%w: validated checkpoint %s for session %s cannot be established", ErrObservationUnavailable, winner.Checkpoint, winner.SessionID)
	}
	owner := winner
	if winner.Epoch > 1 {
		owner = predecessor
	}
	admitted, err := admitCheckpoint(checkpoint, winner.SessionID, owner, sessionKind, chain, repo, nil)
	if err != nil {
		return err
	}
	admitReference := buildCheckpointAdmission(checkpoints, chain, repo, sessionKind)
	if err := checkCheckpointProfileAuthority(admitted, repo, admitReference); err != nil {
		return err
	}
	return nil
}

// checkAncestorCheckpoints validates the checkpoint references of
// every non-winner lease in the winning ancestry: Section 5.3
// requires each epoch above 1 to reference a validated checkpoint
// for its session and predecessor lease, and Section 5.4 requires
// that checkpoint to be created by the owning lease holder. The
// winner itself is covered by checkWinnerCheckpoint. Only the
// winner's own ancestry is admitted: sibling branches off that
// chain are never consulted, so legal branching is preserved. An
// epoch-1 root without a reference stays admissible before the
// first provider boundary exists; a reference it does carry must
// resolve to a checkpoint bound to itself and created by its own
// holder. An unresolvable reference or a binding to another
// session, lease, or holder is incomplete authority
// (observation_unavailable), never absence or a minted fact. Every
// admitted ancestor checkpoint must also resolve its event-head
// closure in the winning source chain (repo) and admit its Section
// 2.4 profile authority over that closure.
func (reader *Reader) checkAncestorCheckpoints(chain []validatedLease, sessionKind string, repo *sessrepo.Repository) error {
	if len(chain) < 2 {
		return nil
	}
	checkpoints, err := reader.validatedCheckpointsByDigest()
	if err != nil {
		return err
	}
	for index := 1; index < len(chain); index++ {
		current := chain[index]
		if current.Epoch == 1 {
			if !current.HasCheckpoint {
				continue
			}
			if err := checkCheckpointBinding(current, current, checkpoints, sessionKind, repo, chain); err != nil {
				return err
			}
			continue
		}
		if !current.HasCheckpoint {
			return fmt.Errorf("%w: validated checkpoint for session %s lease %s cannot be established", ErrObservationUnavailable, current.SessionID, current.LeaseID)
		}
		if index+1 >= len(chain) {
			return fmt.Errorf("%w: predecessor lease record for session %s cannot be established", ErrObservationUnavailable, current.SessionID)
		}
		if err := checkCheckpointBinding(current, chain[index+1], checkpoints, sessionKind, repo, chain); err != nil {
			return err
		}
	}
	return nil
}

// checkCheckpointBinding resolves one lease's checkpoint reference
// against the admitted Checkpoint Records and validates its Section
// 5.3 session and predecessor-lease binding with its Section 5.4
// holder binding: the checkpoint must name this session and the
// owning lease, and its creator must be that lease's holder. The
// admitted checkpoint must also resolve its event-head closure in
// the winning source chain (repo); see checkCheckpointEventHeads.
// The same closure must then admit its Section 2.4 profile
// authority; see checkCheckpointProfileAuthority.
func checkCheckpointBinding(current, owner validatedLease, checkpoints map[string]validatedCheckpoint, sessionKind string, repo *sessrepo.Repository, chain []validatedLease) error {
	checkpoint, ok := checkpoints[current.Checkpoint]
	if !ok {
		return fmt.Errorf("%w: validated checkpoint %s for session %s cannot be established", ErrObservationUnavailable, current.Checkpoint, current.SessionID)
	}
	admitted, err := admitCheckpoint(checkpoint, current.SessionID, owner, sessionKind, chain, repo, nil)
	if err != nil {
		return err
	}
	admitReference := buildCheckpointAdmission(checkpoints, chain, repo, sessionKind)
	if err := checkCheckpointProfileAuthority(admitted, repo, admitReference); err != nil {
		return err
	}
	return nil
}

// admitCheckpoint is the single constructor for an admitted Checkpoint
// Record. All raw validated data enters this function from an explicit
// winner/ancestor/reference admission owner; profile derivation receives only
// the resulting capability.
func admitCheckpoint(raw validatedCheckpoint, sessionID string, owner validatedLease, sessionKind string, chain []validatedLease, repo *sessrepo.Repository, resume *sessrepo.EventSummary) (admittedCheckpoint, error) {
	if raw.SessionID != sessionID {
		return admittedCheckpoint{}, fmt.Errorf("%w: validated checkpoint %s carries session %s, want %s", ErrObservationUnavailable, raw.Digest, raw.SessionID, sessionID)
	}
	if raw.LeaseEpoch != owner.Epoch || raw.LeaseID != owner.LeaseID {
		return admittedCheckpoint{}, fmt.Errorf("%w: validated checkpoint %s is bound to lease %s at epoch %d, want lease %s at epoch %d", ErrObservationUnavailable, raw.Digest, raw.LeaseID, raw.LeaseEpoch, owner.LeaseID, owner.Epoch)
	}
	if raw.CreatorHostID != owner.HolderHostID {
		return admittedCheckpoint{}, fmt.Errorf("%w: validated checkpoint %s is created by host %s, want owning lease holder %s", ErrObservationUnavailable, raw.Digest, raw.CreatorHostID, owner.HolderHostID)
	}
	if err := checkCheckpointPersistence(raw, sessionID, sessionKind); err != nil {
		return admittedCheckpoint{}, err
	}
	if err := checkCheckpointEventHeads(raw, owner, repo); err != nil {
		return admittedCheckpoint{}, err
	}
	if resume != nil {
		if err := checkReferencedCheckpointTemporal(raw, owner, *resume, chain, repo); err != nil {
			return admittedCheckpoint{}, err
		}
	}
	return admittedCheckpoint{seal: &checkpointSeal{
		token:               &checkpointSealToken{},
		digest:              raw.Digest,
		sessionID:           raw.SessionID,
		leaseEpoch:          raw.LeaseEpoch,
		leaseID:             raw.LeaseID,
		creatorHostID:       raw.CreatorHostID,
		hasProviderManifest: raw.HasProviderManifest,
		hasTaskBoardBundle:  raw.HasTaskBoardBundle,
		eventHeads:          append([]string(nil), raw.EventHeads...),
	}}, nil
}

// buildCheckpointAdmission returns the only raw-record lookup capability
// exposed to the referenced profile derivation. The closure performs the
// lookup and immediately routes the result through the shared admission owner.
func buildCheckpointAdmission(checkpoints map[string]validatedCheckpoint, chain []validatedLease, repo *sessrepo.Repository, sessionKind string) checkpointAdmission {
	return func(checkpointID, sessionID string, summary sessrepo.EventSummary) (result admittedCheckpoint, err error) {
		raw, ok := checkpoints[checkpointID]
		if !ok {
			err = fmt.Errorf("%w: validated checkpoint %s for session %s cannot be established", ErrObservationUnavailable, checkpointID, sessionID)
			return
		}
		return checkReferencedCheckpointBinding(raw, checkpointID, sessionID, sessionKind, summary, chain, repo)
	}
}

// checkCheckpointEventHeads resolves every event head of one parsed
// Checkpoint Record against the winning source chain for its session
// (Section 5.4 event-head closure). Each head must name a chained
// event for the same session at or before the owning lease: a head
// naming an absent event (including a cross-session digest, a
// losing-lease preserved blob outside the authoritative chain, or a
// synthetic placeholder) is incomplete authority
// (observation_unavailable), and a head under a later lease (a
// greater epoch, or the same epoch under a different fencing token)
// is likewise incomplete authority because the checkpoint cannot fix
// authority over events that postdate it. Historical heads stay
// admissible and are never required to equal the current tail, so
// valid branching and lagging copies are preserved. A failed chain
// read keeps its cause inside the same incomplete-authority refusal
// instead of minting a fact, and a nil chain handle refuses the same
// class rather than skipping the gate.
func checkCheckpointEventHeads(checkpoint validatedCheckpoint, owner validatedLease, repo *sessrepo.Repository) error {
	if repo == nil {
		return fmt.Errorf("%w: validated checkpoint %s event-head authority for session %s cannot be established: no winning source chain", ErrObservationUnavailable, checkpoint.Digest, checkpoint.SessionID)
	}
	events, err := repo.ListEvents(checkpoint.SessionID)
	if err != nil {
		return fmt.Errorf("%w: validated checkpoint %s event-head authority for session %s cannot be established: %v", ErrObservationUnavailable, checkpoint.Digest, checkpoint.SessionID, err)
	}
	byID := make(map[string]sessrepo.EventSummary, len(events))
	for _, summary := range events {
		byID[summary.EventID] = summary
	}
	for _, head := range checkpoint.EventHeads {
		summary, ok := byID[head]
		if !ok {
			return fmt.Errorf("%w: validated checkpoint %s names unknown event head %s for session %s", ErrObservationUnavailable, checkpoint.Digest, head, checkpoint.SessionID)
		}
		if summary.LeaseEpoch > owner.Epoch {
			return fmt.Errorf("%w: validated checkpoint %s names event head %s under later lease epoch %d, want at or before epoch %d", ErrObservationUnavailable, checkpoint.Digest, head, summary.LeaseEpoch, owner.Epoch)
		}
		if summary.LeaseEpoch == owner.Epoch && summary.LeaseID != owner.LeaseID {
			return fmt.Errorf("%w: validated checkpoint %s names event head %s under lease %s, want owning lease %s at epoch %d", ErrObservationUnavailable, checkpoint.Digest, head, summary.LeaseID, owner.LeaseID, owner.Epoch)
		}
	}
	return nil
}

// checkCheckpointProfileAuthority admits the Section 2.4 effective
// execution profile and its nullable source event for one required
// checkpoint against the winning source chain (repo, SPEC 5.4
// event-head closure). The transitive predecessor closure of the
// checkpoint heads fixes the launch values: the Session Record
// carries the creation profile, the first launch in the closure
// carries that profile with a null source, and every later
// provider/task-board launch carries the newest authoritative
// profile.changed event at or before it with the profile it
// targets (launches carry no checkpoint_id member, so the closure
// under admission evaluated in lease/sequence order is their only
// referenced authority). Every session resume instead carries the
// effective pair of ITS referenced checkpoint (SPEC 5.2),
// resolved through the admitted Checkpoint Records (checkpoints)
// and derived from that checkpoint's own event-head closure, never
// from the position in the outer walk. Every fork carries the
// newly persisted Session Record profile with a null source (SPEC
// 2.4/5.2 new-session authority); only its source_profile_event_id
// provenance stays outside this closure. A non-null source must
// name a chained profile.changed event inside the governing
// closure; a missing (dangling, cross-session, or preserved
// outside the authoritative index), out-of-closure (later
// local-only or losing-lease), wrong-type, or non-newest reference
// is contradictory authority (integrity failure), as is a pair
// whose profile value does not equal the derived effective
// profile — consumers MUST NOT consult a later local-only event or
// fall back to the creation value. Historical heads stay admissible
// because derivation reads only closures, never the current tail.
// A missing or misbound referenced checkpoint, an unresolvable
// referenced head, a failed chain read, and a nil chain handle
// refuse incomplete authority (observation_unavailable) with the
// cause kept instead of minting a fact. Payloads are fetched only
// for pair and change events in the closures; the index already
// carries every other routing member the walk needs.
func checkCheckpointProfileAuthority(checkpoint admittedCheckpoint, repo *sessrepo.Repository, admission checkpointAdmission) error {
	authority, err := checkpoint.authority()
	if err != nil {
		return err
	}
	if repo == nil {
		return fmt.Errorf("%w: validated checkpoint %s profile authority for session %s cannot be established: no winning source chain", ErrObservationUnavailable, authority.digest, authority.sessionID)
	}
	recordBytes, err := repo.GetRecord(authority.sessionID)
	if err != nil {
		return fmt.Errorf("%w: validated checkpoint %s profile authority for session %s cannot be established: %v", ErrObservationUnavailable, authority.digest, authority.sessionID, err)
	}
	creation, recordID, err := sessionCreationProfile(checkpoint, recordBytes)
	if err != nil {
		return err
	}
	events, err := repo.ListEvents(authority.sessionID)
	if err != nil {
		return fmt.Errorf("%w: validated checkpoint %s profile authority for session %s cannot be established: %v", ErrObservationUnavailable, authority.digest, authority.sessionID, err)
	}
	closure, err := profileClosure(checkpoint, recordID, events)
	if err != nil {
		return err
	}
	return checkClosureProfilePairs(checkpoint, creation, recordID, events, closure, repo, admission)
}

// sessionCreationProfile reads the immutable creation profile and
// the canonical record digest from the winning-source Session
// Record bytes. The closed shape owner requires execution_profile
// standard|yolo, so any other value contradicts the admitted
// record (integrity failure).
func sessionCreationProfile(checkpoint admittedCheckpoint, recordBytes []byte) (string, string, error) {
	authority, err := checkpoint.authority()
	if err != nil {
		return "", "", err
	}
	members, fault := environ.DecodeStrictObject(recordBytes)
	if fault != nil {
		return "", "", fmt.Errorf("%w: session %s carries no decodable session record for profile authority: %v", sessstate.ErrIntegrity, authority.sessionID, fault)
	}
	creation, err := leaseStringMember(members, "execution_profile")
	if err != nil || (creation != "standard" && creation != "yolo") {
		return "", "", fmt.Errorf("%w: session %s carries no valid creation profile for profile authority", sessstate.ErrIntegrity, authority.sessionID)
	}
	recordID, err := leaseStringMember(members, "record_id")
	if err != nil {
		return "", "", fmt.Errorf("%w: session %s carries no valid record digest for profile authority", sessstate.ErrIntegrity, authority.sessionID)
	}
	digest, err := scalar.ParseDigest(recordID)
	if err != nil {
		return "", "", fmt.Errorf("%w: session %s carries malformed record digest %q for profile authority", sessstate.ErrIntegrity, authority.sessionID, recordID)
	}
	return creation, digest.String(), nil
}

// profileClosure resolves the transitive predecessor closure of the
// checkpoint heads against the winning source chain index. The
// genesis link names the Session Record digest rather than an
// event; every other predecessor must name a chained event, so a
// dangling reference contradicts the admitted closure (integrity
// failure). Divergent blobs preserved outside the authoritative
// index never enter the closure: index membership is authority.
func profileClosure(checkpoint admittedCheckpoint, recordID string, events []sessrepo.EventSummary) (map[string]bool, error) {
	authority, err := checkpoint.authority()
	if err != nil {
		return nil, err
	}
	byID := make(map[string]sessrepo.EventSummary, len(events))
	for _, summary := range events {
		byID[summary.EventID] = summary
	}
	closure := make(map[string]bool, len(events))
	stack := append([]string(nil), authority.eventHeads...)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if closure[id] {
			continue
		}
		summary, ok := byID[id]
		if !ok {
			if id == recordID {
				continue
			}
			return nil, fmt.Errorf("%w: validated checkpoint %s profile closure names unknown event %s for session %s", sessstate.ErrIntegrity, authority.digest, id, authority.sessionID)
		}
		closure[id] = true
		stack = append(stack, summary.Predecessors...)
	}
	return closure, nil
}

// profilePair is one decoded Section 2.4 pair from a launch or
// resume event payload.
type profilePair struct {
	profile   string
	source    string
	hasSource bool
}

// eventProfilePair decodes the pair from one chained launch or
// resume event. Chained bytes passed the closed payload shape at
// append time, so a member that no longer decodes contradicts the
// attested chain (integrity failure).
func eventProfilePair(checkpoint admittedCheckpoint, repo *sessrepo.Repository, summary sessrepo.EventSummary) (profilePair, error) {
	authority, err := checkpoint.authority()
	if err != nil {
		return profilePair{}, err
	}
	var pair profilePair
	payload, err := eventPayloadMembers(checkpoint, repo, summary)
	if err != nil {
		return pair, err
	}
	profile, err := leaseStringMember(payload, "execution_profile")
	if err != nil || (profile != "standard" && profile != "yolo") {
		return pair, fmt.Errorf("%w: session %s event %s carries no valid execution profile for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID)
	}
	source, hasSource, err := leaseNullableDigest(payload, "profile_source_event_id")
	if err != nil {
		return pair, fmt.Errorf("%w: session %s event %s carries no valid profile source for profile authority: %v", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, err)
	}
	return profilePair{profile: profile, source: source, hasSource: hasSource}, nil
}

// eventProfileTarget decodes the target profile of one chained
// profile.changed event. Only the target feeds the derivation: the
// source value and the confirmation history belong to the
// publication flow that admitted the event, never to this
// admission.
func eventProfileTarget(checkpoint admittedCheckpoint, repo *sessrepo.Repository, summary sessrepo.EventSummary) (string, error) {
	authority, err := checkpoint.authority()
	if err != nil {
		return "", err
	}
	payload, err := eventPayloadMembers(checkpoint, repo, summary)
	if err != nil {
		return "", err
	}
	target, err := leaseStringMember(payload, "to")
	if err != nil || (target != "standard" && target != "yolo") {
		return "", fmt.Errorf("%w: session %s event %s carries no valid profile target for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID)
	}
	return target, nil
}

// eventPayloadMembers returns the decoded payload object of one
// chained event. A failed blob read keeps its cause inside the
// incomplete-authority refusal; bytes that no longer decode
// contradict the attested chain.
func eventPayloadMembers(checkpoint admittedCheckpoint, repo *sessrepo.Repository, summary sessrepo.EventSummary) (map[string]json.RawMessage, error) {
	authority, err := checkpoint.authority()
	if err != nil {
		return nil, err
	}
	raw, err := repo.GetEvent(authority.sessionID, summary.EventID)
	if err != nil {
		return nil, fmt.Errorf("%w: validated checkpoint %s profile authority for session %s cannot be established: %v", ErrObservationUnavailable, authority.digest, authority.sessionID, err)
	}
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return nil, fmt.Errorf("%w: session %s event %s carries no decodable event frame for profile authority: %v", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, fault)
	}
	payloadRaw, ok := members["payload"]
	if !ok || string(payloadRaw) == "null" {
		return nil, fmt.Errorf("%w: session %s event %s carries no payload for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID)
	}
	payload, fault := environ.DecodeStrictObject([]byte(payloadRaw))
	if fault != nil {
		return nil, fmt.Errorf("%w: session %s event %s carries no decodable payload for profile authority: %v", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, fault)
	}
	return payload, nil
}

// checkClosureProfilePairs walks the closure in authoritative chain
// order and binds every launch, resume, and fork pair to its
// Section 2.4 derivation: the first launch carries the creation
// profile with a null source, and every later launch carries the
// newest authoritative profile.changed event at or before it with
// that change's target profile (the creation pair while no change
// exists). Every resume carries the effective pair of its own
// referenced checkpoint (referencedCheckpointProfile), never the
// outer-walk prefix. Every fork carries the newly persisted
// Session Record profile with a null source: the fork event lives
// in the new session chain, so creation here is already the new
// record; its source_profile_event_id provenance is never read.
// Each pair mismatch, each unexpected or missing source, and each
// source that fails the governing-closure resolution below refuses
// integrity failure.
func checkClosureProfilePairs(checkpoint admittedCheckpoint, creation, recordID string, events []sessrepo.EventSummary, closure map[string]bool, repo *sessrepo.Repository, admission checkpointAdmission) error {
	if _, err := checkpoint.authority(); err != nil {
		return err
	}
	byID := make(map[string]sessrepo.EventSummary, len(events))
	for _, summary := range events {
		byID[summary.EventID] = summary
	}
	newestSource, newestProfile := "", ""
	hasNewest := false
	firstLaunch := true
	for _, summary := range events {
		if !closure[summary.EventID] {
			continue
		}
		switch summary.EventType {
		case "profile.changed":
			target, err := eventProfileTarget(checkpoint, repo, summary)
			if err != nil {
				return err
			}
			newestSource, newestProfile, hasNewest = summary.EventID, target, true
		case "provider.launched", "task_board.launched":
			pair, err := eventProfilePair(checkpoint, repo, summary)
			if err != nil {
				return err
			}
			wantProfile, wantSource, wantHasSource := creation, "", false
			if !firstLaunch && hasNewest {
				wantProfile, wantSource, wantHasSource = newestProfile, newestSource, true
			}
			if err := checkProfilePairSource(checkpoint, summary, pair, wantProfile, wantSource, wantHasSource, byID, closure); err != nil {
				return err
			}
			firstLaunch = false
		case "session.resumed":
			pair, err := eventProfilePair(checkpoint, repo, summary)
			if err != nil {
				return err
			}
			wantProfile, wantSource, wantHasSource, refClosure, err := referencedCheckpointProfile(checkpoint, summary, creation, recordID, events, admission, repo)
			if err != nil {
				return err
			}
			if err := checkProfilePairSource(checkpoint, summary, pair, wantProfile, wantSource, wantHasSource, byID, refClosure); err != nil {
				return err
			}
		case "fork.created":
			pair, err := eventProfilePair(checkpoint, repo, summary)
			if err != nil {
				return err
			}
			if err := checkProfilePairSource(checkpoint, summary, pair, creation, "", false, byID, closure); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkReferencedCheckpointBinding admits one resume-referenced
// Checkpoint Record through the same semantic owner the winner and
// necessary-ancestor checkpoints use. The owning lease is resolved
// by (epoch, lease_id) tuple within the winning ancestry chain for
// the session; historical leases stay admissible because the chain
// covers every epoch from the winner down to the root, while a
// fencing token without an admitted owning lease at that epoch is
// incomplete authority. The creator must be that lease's holder,
// the persistence variant must match the Session Record kind, every
// head must resolve at or before the owning lease, and the consuming
// resume must be at the same or a later lease in that ancestry. A
// missing, misbound, future-owned, or unresolvable reference refuses
// observation_unavailable, never a minted fact.
func checkReferencedCheckpointBinding(ref validatedCheckpoint, checkpointID, sessionID, sessionKind string, summary sessrepo.EventSummary, chain []validatedLease, repo *sessrepo.Repository) (result admittedCheckpoint, err error) {
	var owner validatedLease
	found := false
	for _, lease := range chain {
		if lease.SessionID == ref.SessionID && lease.Epoch == ref.LeaseEpoch && lease.LeaseID == ref.LeaseID {
			owner = lease
			found = true
			break
		}
	}
	if !found {
		err = fmt.Errorf("%w: validated checkpoint %s is bound to lease %s at epoch %d without an admitted owning lease", ErrObservationUnavailable, checkpointID, ref.LeaseID, ref.LeaseEpoch)
		return
	}
	return admitCheckpoint(ref, sessionID, owner, sessionKind, chain, repo, &summary)
}

// checkReferencedCheckpointTemporal binds the checkpoint's authority to the
// event that consumes it. The owning lease must be the consuming lease or an
// earlier lease in the admitted winner-to-root ancestry. Each checkpoint head
// must also be an ancestor of the resume event; otherwise the resume would
// consume a checkpoint whose authoritative state is later than the event.
func checkReferencedCheckpointTemporal(checkpoint validatedCheckpoint, owner validatedLease, summary sessrepo.EventSummary, chain []validatedLease, repo *sessrepo.Repository) error {
	ownerIndex, ownerFound := leaseChainIndex(chain, owner.Epoch, owner.LeaseID)
	resumeIndex, resumeFound := leaseChainIndex(chain, summary.LeaseEpoch, summary.LeaseID)
	if !ownerFound || !resumeFound {
		return fmt.Errorf("%w: validated checkpoint %s and resume event %s do not share an admitted lease ancestry", ErrObservationUnavailable, checkpoint.Digest, summary.EventID)
	}
	if ownerIndex < resumeIndex {
		return fmt.Errorf("%w: validated checkpoint %s is owned by later lease %s at epoch %d than consuming resume %s at epoch %d", ErrObservationUnavailable, checkpoint.Digest, owner.LeaseID, owner.Epoch, summary.EventID, summary.LeaseEpoch)
	}
	if repo == nil {
		return fmt.Errorf("%w: validated checkpoint %s temporal authority for resume event %s cannot be established: no winning source chain", ErrObservationUnavailable, checkpoint.Digest, summary.EventID)
	}
	events, err := repo.ListEvents(checkpoint.SessionID)
	if err != nil {
		return fmt.Errorf("%w: validated checkpoint %s temporal authority for resume event %s cannot be established: %v", ErrObservationUnavailable, checkpoint.Digest, summary.EventID, err)
	}
	byID := make(map[string]sessrepo.EventSummary, len(events))
	for _, event := range events {
		byID[event.EventID] = event
	}
	if _, ok := byID[summary.EventID]; !ok {
		return fmt.Errorf("%w: resume event %s for checkpoint %s is not in the winning source chain", ErrObservationUnavailable, summary.EventID, checkpoint.Digest)
	}
	ancestors := make(map[string]bool, len(events))
	stack := append([]string(nil), summary.Predecessors...)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if ancestors[id] {
			continue
		}
		ancestor, ok := byID[id]
		if !ok {
			// The repository's validated chain includes the Session Record
			// digest as the genesis predecessor, not as an EventSummary.
			// Other missing predecessors would already have refused from
			// Repository.ListEvents, so do not confuse that record boundary
			// with an unreadable event.
			continue
		}
		ancestors[id] = true
		stack = append(stack, ancestor.Predecessors...)
	}
	for _, head := range checkpoint.EventHeads {
		if !ancestors[head] {
			return fmt.Errorf("%w: resume event %s does not descend from checkpoint %s event head %s", ErrObservationUnavailable, summary.EventID, checkpoint.Digest, head)
		}
	}
	return nil
}

// leaseChainIndex returns the winner-to-root position of one lease. A lower
// index is a later lease; this keeps the temporal rule tied to the admitted
// ancestry rather than to an independently sorted canonical tuple.
func leaseChainIndex(chain []validatedLease, epoch uint64, leaseID string) (int, bool) {
	for index, lease := range chain {
		if lease.Epoch == epoch && lease.LeaseID == leaseID {
			return index, true
		}
	}
	return 0, false
}

// referencedCheckpointProfile derives the Section 2.4 effective
// pair one session.resumed event must carry (SPEC 5.2): the newest
// authoritative profile.changed event in its referenced
// checkpoint's own event-head closure with that change's target
// profile, or the creation pair while that closure holds no
// change. The checkpoint_id member resolves through the admitted
// Checkpoint Records and is admitted through the shared semantic
// owner (checkReferencedCheckpointBinding) before its closure is
// consumed; the closure walks the same winning-source chain index,
// so a later local-only change outside the referenced heads is
// never consulted and branching history stays admissible. A
// missing, wrong-session, wrong-lease, wrong-creator,
// wrong-variant, or unresolvable-head reference is incomplete
// referenced authority (observation_unavailable); a dangling
// predecessor inside the referenced closure or a pair the closure
// contradicts refuses integrity failure. Read failures keep their
// cause inside the incomplete-authority refusal.
func referencedCheckpointProfile(checkpoint admittedCheckpoint, summary sessrepo.EventSummary, creation, recordID string, events []sessrepo.EventSummary, admission checkpointAdmission, repo *sessrepo.Repository) (string, string, bool, map[string]bool, error) {
	authority, err := checkpoint.authority()
	if err != nil {
		return "", "", false, nil, err
	}
	payload, err := eventPayloadMembers(checkpoint, repo, summary)
	if err != nil {
		return "", "", false, nil, err
	}
	rawID, err := leaseStringMember(payload, "checkpoint_id")
	if err != nil {
		return "", "", false, nil, fmt.Errorf("%w: session %s event %s carries no valid resume checkpoint for profile authority: %v", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, err)
	}
	reference, err := scalar.ParseDigest(rawID)
	if err != nil {
		return "", "", false, nil, fmt.Errorf("%w: session %s event %s carries malformed resume checkpoint %q for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, rawID)
	}
	checkpointID := reference.String()
	if admission == nil {
		return "", "", false, nil, fmt.Errorf("%w: validated checkpoint %s profile authority has no admission capability", ErrObservationUnavailable, checkpointID)
	}
	ref, err := admission(checkpointID, authority.sessionID, summary)
	if err != nil {
		return "", "", false, nil, err
	}
	refClosure, err := profileClosure(ref, recordID, events)
	if err != nil {
		return "", "", false, nil, err
	}
	refSource, refProfile := "", ""
	refHas := false
	for _, candidate := range events {
		if !refClosure[candidate.EventID] {
			continue
		}
		if candidate.EventType != "profile.changed" {
			continue
		}
		target, err := eventProfileTarget(checkpoint, repo, candidate)
		if err != nil {
			return "", "", false, nil, err
		}
		refSource, refProfile, refHas = candidate.EventID, target, true
	}
	if !refHas {
		return creation, "", false, refClosure, nil
	}
	return refProfile, refSource, true, refClosure, nil
}

// checkProfilePairSource binds one decoded pair to its derived
// expectation. The profile value must equal the derived effective
// profile, so falling back to the creation value past a change
// refuses; the source presence must match, so a first launch with
// any source refuses; and a present source must resolve to the
// newest authoritative profile.changed event in the closure —
// missing, out-of-closure (later local-only or losing-lease),
// wrong-type, and non-newest references each refuse with the
// violated expectation named.
func checkProfilePairSource(checkpoint admittedCheckpoint, summary sessrepo.EventSummary, pair profilePair, wantProfile, wantSource string, wantHasSource bool, byID map[string]sessrepo.EventSummary, closure map[string]bool) error {
	authority, err := checkpoint.authority()
	if err != nil {
		return err
	}
	if pair.profile != wantProfile {
		return fmt.Errorf("%w: session %s event %s carries execution profile %q, want effective profile %q for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, pair.profile, wantProfile)
	}
	if pair.hasSource != wantHasSource {
		if pair.hasSource {
			return fmt.Errorf("%w: session %s event %s carries profile source %s, want no profile source for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, pair.source)
		}
		return fmt.Errorf("%w: session %s event %s carries no profile source, want newest authoritative source %s for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, wantSource)
	}
	if !wantHasSource {
		return nil
	}
	sourced, ok := byID[pair.source]
	if !ok {
		return fmt.Errorf("%w: session %s event %s names unknown profile source %s for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, pair.source)
	}
	if !closure[pair.source] {
		return fmt.Errorf("%w: session %s event %s names profile source %s outside the checkpoint closure for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, pair.source)
	}
	if sourced.EventType != "profile.changed" {
		return fmt.Errorf("%w: session %s event %s names profile source %s of type %q, want a profile.changed event for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, pair.source, sourced.EventType)
	}
	if pair.source != wantSource {
		return fmt.Errorf("%w: session %s event %s names profile source %s, want newest authoritative source %s for profile authority", sessstate.ErrIntegrity, authority.sessionID, summary.EventID, pair.source, wantSource)
	}
	return nil
}

// checkCheckpointPersistence binds a parsed Checkpoint Record to
// the persistence variant the referenced Session Record selects
// (Section 5.4): a direct session requires a non-null
// provider_manifest_id with a null task_board_bundle_id, while a
// task_board session requires the reverse. Canonical shape proves
// exactly one variant is present but cannot select which one is
// valid, so a swapped variant keeps canonical identity, session,
// lease, epoch, and creator yet refuses here with incomplete
// authority (observation_unavailable), never absence or a minted
// fact. An unknown session kind is malformed configuration: valid
// Session Records admit only the two kinds.
func checkCheckpointPersistence(checkpoint validatedCheckpoint, sessionID, sessionKind string) error {
	switch sessionKind {
	case "direct":
		if !checkpoint.HasProviderManifest || checkpoint.HasTaskBoardBundle {
			return fmt.Errorf("%w: validated checkpoint %s carries a task-board persistence variant for direct session %s", ErrObservationUnavailable, checkpoint.Digest, sessionID)
		}
		return nil
	case "task_board":
		if checkpoint.HasProviderManifest || !checkpoint.HasTaskBoardBundle {
			return fmt.Errorf("%w: validated checkpoint %s carries a provider persistence variant for task_board session %s", ErrObservationUnavailable, checkpoint.Digest, sessionID)
		}
		return nil
	default:
		return fmt.Errorf("%w: session %s carries unknown kind %q for checkpoint persistence", ErrInvalidConfig, sessionID, sessionKind)
	}
}

// compareLeaseTuple orders (epoch, lease_id) with the Section 5.3 rule:
// greatest epoch wins, ties break by bytewise lease-ID order.
func compareLeaseTuple(epochA uint64, leaseA string, epochB uint64, leaseB string) int {
	return sessstate.Compare(
		sessstate.LeaseHead{Epoch: epochA, LeaseID: leaseA},
		sessstate.LeaseHead{Epoch: epochB, LeaseID: leaseB},
	)
}
