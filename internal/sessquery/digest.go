// Local attestation digests for selector plans: canonical fingerprints
// over validated reads, never over caller claims.
//
// fingerprint canonicalizes its input through the canonical-JSON owner
// and digests the canonical bytes, so equal facts always produce equal
// digests and any fact change invalidates the plans bound to them. The
// resulting "sha256:" digests are self-checked through the scalar
// digest grammar before use.
package sessquery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// transportLabel names the index-read mechanism this stack uses: direct
// local repository reads with no mesh transport. Binding the label into
// the configuration digest invalidates every plan if a transport is
// ever introduced, instead of silently reusing facts read another way.
const transportLabel = "local-direct"

// fingerprint canonicalizes value and returns the "sha256:" digest of
// the canonical bytes.
func fingerprint(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := canonicaljson.Canonicalize(raw)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if _, err := scalar.ParseDigest(digest); err != nil {
		return "", err
	}
	return digest, nil
}

// peerFingerprint is one configured peer mapping in the configuration
// digest: exact alias with its host, in bytewise host order.
type peerFingerprint struct {
	Alias  string `json:"alias"`
	HostID string `json:"host_id"`
}

// fingerprintConfiguration digests the effective validated
// configuration: the local host, the peer mapping, the allowlist, and
// the transport label. Handles are not configuration and stay out:
// swapping a learned index without changing the mapping still changes
// the index digest at read time.
func fingerprintConfiguration(reader *Reader, config validatedConfig) (string, error) {
	peers := make([]peerFingerprint, 0, len(reader.Peers))
	for _, host := range config.sortedPeerHosts(reader) {
		peers = append(peers, peerFingerprint{Alias: config.aliasFor(host), HostID: host})
	}
	allowlist := make([]string, 0, len(config.allowed))
	for id := range config.allowed {
		allowlist = append(allowlist, id)
	}
	sort.Strings(allowlist)
	return fingerprint(map[string]any{
		"allowlist":     allowlist,
		"local_host_id": config.localHost,
		"peers":         peers,
		"transport":     transportLabel,
	})
}

// indexRowFingerprint is one validated index row in the source index
// digest: identity, liveness, lifecycle, tail, and winning lease facts
// for every session the complete read returned, in bytewise session
// order. Any index change — a new, removed, parked, tombstoned, or
// advanced session — changes this digest.
type indexRowFingerprint struct {
	SessionID     string `json:"session_id"`
	RecordID      string `json:"record_id"`
	Name          string `json:"name"`
	Parked        bool   `json:"parked"`
	State         string `json:"state"`
	TailEvent     string `json:"tail_event"`
	WinnerEpoch   uint64 `json:"winner_epoch"`
	WinnerLeaseID string `json:"winner_lease_id"`
	OwnerHostID   string `json:"owner_host_id"`
}

// fingerprintIndex digests the complete validated index read used for
// selection: every row the listing returned with its validated tail.
// A tail read failure keeps its read-failure class: local errors stay
// repository errors, peer errors carry the read-failure class.
func fingerprintIndex(reader *Reader, repo *sessrepo.Repository, rows []Summary, peerHost string) (string, error) {
	finger := make([]indexRowFingerprint, 0, len(rows))
	for _, row := range rows {
		entry := indexRowFingerprint{
			SessionID:     row.Projection.SessionID,
			RecordID:      row.Projection.RecordID,
			Name:          row.Projection.Name,
			State:         string(row.Projection.State),
			WinnerEpoch:   row.Projection.Winner.Epoch,
			WinnerLeaseID: row.Projection.Winner.LeaseID,
			OwnerHostID:   row.Projection.OwnerHostID,
		}
		if row.Projection.Parked != nil {
			entry.Parked = true
		} else {
			tail, err := chainTail(repo, row.Projection.SessionID)
			if err != nil {
				if peerHost == "" {
					return "", err
				}
				return "", peerReadFailed(peerHost, err)
			}
			entry.TailEvent = tail
		}
		finger = append(finger, entry)
	}
	return fingerprint(finger)
}

// fingerprintExpectations binds caller-declared action expectations for
// change detection only. Expectations must be one canonically encoded
// JSON object; the digest never replaces the checkpoint, workspace,
// cohort, provider, terminal, or confirmation contracts, which have no
// owners in this leaf. Empty expectations bind the empty-object digest.
func fingerprintExpectations(expectations []byte) (string, error) {
	if len(expectations) == 0 {
		return fingerprint(map[string]any{})
	}
	canonical, err := canonicaljson.Canonicalize(expectations)
	if err != nil {
		return "", fmt.Errorf("%w: malformed plan expectations: %v", ErrInvalidArgument, err)
	}
	if len(canonical) == 0 || canonical[0] != '{' {
		return "", fmt.Errorf("%w: plan expectations must be a JSON object", ErrInvalidArgument)
	}
	sum := sha256.Sum256(canonical)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if _, err := scalar.ParseDigest(digest); err != nil {
		return "", err
	}
	return digest, nil
}
