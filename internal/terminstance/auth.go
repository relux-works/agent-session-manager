package terminstance

import (
	"encoding/json"
	"time"

	"github.com/relux-works/agent-session-manager/internal/environ"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// AuthorizationKind is the closed §4.C authorization_kind enum.
type AuthorizationKind string

// AX authorization kinds.
const (
	AuthorizationCreate     AuthorizationKind = "create"
	AuthorizationControl    AuthorizationKind = "control"
	AuthorizationForceStale AuthorizationKind = "force_stale"
	AuthorizationRestore    AuthorizationKind = "restore"
)

// ParseAuthorizationKind admits exactly the four §4.C kinds.
func ParseAuthorizationKind(value string) (AuthorizationKind, error) {
	switch AuthorizationKind(value) {
	case AuthorizationCreate, AuthorizationControl,
		AuthorizationForceStale, AuthorizationRestore:
		return AuthorizationKind(value), nil
	default:
		return "", refuse(CodeProtocolError, "ax authorization kind")
	}
}

// MaxLeaseEpoch is the §4.C lease_epoch ceiling: uint53[1..9007199254740991].
// It aliases the scalar owner's bound, never a retyped copy.
const MaxLeaseEpoch = scalar.MaxUint53

// AXAuthorization is the closed §4.C AXAuthorization object: exactly
// lease_id:UUIDv4, lease_epoch:uint53[1..9007199254740991],
// holder_host_id:UUIDv7, authorization_kind, issued_at, expires_at and
// authorization_evidence_id:digest. Expiry is strictly after issue.
type AXAuthorization struct {
	LeaseID      string
	LeaseEpoch   uint64
	HolderHostID string
	Kind         AuthorizationKind
	IssuedAt     scalar.Timestamp
	ExpiresAt    scalar.Timestamp
	EvidenceID   string
}

// axAuthorizationMembers is the exact closed member set, in the order the
// pinned section lists it.
var axAuthorizationMembers = []string{
	"lease_id",
	"lease_epoch",
	"holder_host_id",
	"authorization_kind",
	"issued_at",
	"expires_at",
	"authorization_evidence_id",
}

// ParseAXAuthorization admits one closed AXAuthorization document. Frame
// admission (UTF-8, lone surrogates, non-object, duplicates, trailing
// data) and every member grammar (UUIDv4/v7, uint53, timestamps, digests)
// are delegated to the landed owners; only the member list itself is
// local, because only this schema can name it. Malformed documents are
// terminal_backend_protocol_error (AX-local parse); only the strict-expiry
// order violation is terminal_backend_unauthorized, mirroring the landed
// ParseAttachAuthorization twin.
func ParseAXAuthorization(raw []byte) (AXAuthorization, error) {
	members, fault := environ.DecodeStrictObject(raw)
	if fault != nil {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization frame")
	}
	auth, err := parseAXAuthorizationObject(members)
	if err != nil {
		return AXAuthorization{}, err
	}
	return auth, nil
}

// parseAXAuthorizationObject admits one already-framed authorization
// member map. ParseAXAuthorization and the nested MutationContext parse
// share it, so the nested authorization is admitted by the same arms,
// never a second implementation.
func parseAXAuthorizationObject(members map[string]json.RawMessage) (AXAuthorization, error) {
	if len(members) != len(axAuthorizationMembers) {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization members")
	}
	for _, member := range axAuthorizationMembers {
		if _, known := members[member]; !known {
			return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization members")
		}
	}
	leaseID, ok := rawString(members["lease_id"])
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization lease")
	}
	if _, err := scalar.ParseUUIDv4(leaseID); err != nil {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization lease")
	}
	leaseEpoch, ok := environ.CheckUint53Bounds(members["lease_epoch"], 1, MaxLeaseEpoch)
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization epoch")
	}
	holderHostID, ok := rawString(members["holder_host_id"])
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization holder")
	}
	if _, err := scalar.ParseUUIDv7(holderHostID); err != nil {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization holder")
	}
	kindRaw, ok := rawString(members["authorization_kind"])
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization kind")
	}
	kind, err := ParseAuthorizationKind(kindRaw)
	if err != nil {
		return AXAuthorization{}, err
	}
	issuedAt, ok := environ.CheckTimestamp(members["issued_at"])
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization timestamp")
	}
	expiresAt, ok := environ.CheckTimestamp(members["expires_at"])
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization timestamp")
	}
	issued, err := issuedAt.Time()
	if err != nil {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization timestamp")
	}
	expires, err := expiresAt.Time()
	if err != nil {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization timestamp")
	}
	if !expires.After(issued) {
		return AXAuthorization{}, refuse(CodeUnauthorized, "ax authorization expiry order")
	}
	evidence, ok := environ.CheckDigest(members["authorization_evidence_id"])
	if !ok {
		return AXAuthorization{}, refuse(CodeProtocolError, "ax authorization evidence")
	}
	return AXAuthorization{
		LeaseID:      leaseID,
		LeaseEpoch:   leaseEpoch,
		HolderHostID: holderHostID,
		Kind:         kind,
		IssuedAt:     issuedAt,
		ExpiresAt:    expiresAt,
		EvidenceID:   evidence.String(),
	}, nil
}

// rawString reads a JSON string member. environ owns the frame and the
// typed members; a bare string read is encoding/json's grammar, shared
// by every nested shape in this package.
func rawString(raw json.RawMessage) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

// LeaseView is the current winning AX lease as observed by the caller:
// the (lease_id, lease_epoch) tuple the authorization must equal at
// every side effect. Callers derive it from
// fencing.Observation.Winner; the epoch-1 lease carries no checkpoint
// and this view carries none either, so the vacuous checkpoint keying
// the story's predecessor leaf removed cannot reappear here.
type LeaseView struct {
	LeaseID string
	Epoch   uint64
}

// CheckAuthorization enforces the §4.C authorization rule at one check
// point: the authorization names the required kind, is unexpired at now,
// and its lease tuple equals the current winning AX lease. Every
// violation is terminal_backend_unauthorized: a malformed document never
// reaches here (ParseAXAuthorization owns shape), so anything refused
// here is a binding failure, not a parse failure. Lease identity and
// lease epoch are separate arms with separate details so a mutant
// dropping either comparison is killed by its own drift test.
// holder_host_id is parsed (UUIDv7) but never bound: LeaseView carries
// the §5.3 winning (lease_id, epoch) tuple only, so a foreign holder
// under the winning tuple is admitted. That is a stated bound, not a
// bypass: no path here mints authority from the holder member.
func CheckAuthorization(auth AXAuthorization, wantKind AuthorizationKind, lease LeaseView, now time.Time) error {
	if auth.Kind != wantKind {
		return refuse(CodeUnauthorized, "ax authorization kind binding")
	}
	expires, err := auth.ExpiresAt.Time()
	if err != nil {
		return refuse(CodeProtocolError, "ax authorization timestamp")
	}
	if !now.Before(expires) {
		return refuse(CodeUnauthorized, "ax authorization expiry")
	}
	if auth.LeaseID != lease.LeaseID {
		return refuse(CodeUnauthorized, "ax authorization lease identity")
	}
	if auth.LeaseEpoch != lease.Epoch {
		return refuse(CodeUnauthorized, "ax authorization lease epoch")
	}
	return nil
}
