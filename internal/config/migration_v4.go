package config

import (
	"errors"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

var (
	// ErrMigrationV4ConfirmRequired reports an ApplyV4 call without explicit
	// operator confirmation of the exact preview.
	ErrMigrationV4ConfirmRequired = errors.New("configuration 4.0.0 apply requires explicit confirmation")
	// ErrMigrationV4StaleSource reports a preview applied after the selected
	// configuration changed. The operator must preview again.
	ErrMigrationV4StaleSource = errors.New("configuration 4.0.0 preview source changed")
	// ErrMigrationV4StaleGeneration reports a preview applied after the
	// trust generation moved. The operator must preview again.
	ErrMigrationV4StaleGeneration = errors.New("configuration 4.0.0 preview trust generation changed")
	// ErrMigrationV4PreviewMismatch reports preview bytes that do not match
	// the replacement rendered from current state. Only the exact preview
	// applies.
	ErrMigrationV4PreviewMismatch = errors.New("configuration 4.0.0 preview does not match current state")
	// ErrMigrationV4Credential reports a selected credential that is
	// missing, not enrolled for the local host, not active, or outside its
	// validity window.
	ErrMigrationV4Credential = errors.New("configuration 4.0.0 selected credential is not admitted")
	// ErrMigrationV4PeerEnrollment reports retained peers without a complete
	// local enrollment. A missing peer enrollment blocks activation; the
	// operator may explicitly remove a peer from the preview instead.
	ErrMigrationV4PeerEnrollment = errors.New("configuration 4.0.0 peer enrollment incomplete")
	// ErrMigrationV4DropUnknown reports an explicit peer removal naming a
	// host that is not a configured peer. Silent drops are impossible: only
	// named configured peers leave the preview.
	ErrMigrationV4DropUnknown = errors.New("configuration 4.0.0 explicit peer removal names an unknown peer")
	// ErrRollbackV4AckRequired reports a rollback without acknowledgement
	// that the legacy installation has no Host Channel assurance.
	ErrRollbackV4AckRequired = errors.New("configuration rollback requires no-host-channel-assurance acknowledgement")
	// ErrRollbackV4Backup reports a missing, unreadable or undecodable
	// rollback backup. Rollback replaces configuration only from the
	// preserved backup bytes.
	ErrRollbackV4Backup = errors.New("configuration rollback backup is missing or invalid")
	// ErrRollbackV4StaleSource reports a rollback attempted after the
	// selected configuration changed. The operator must start over from
	// the current state.
	ErrRollbackV4StaleSource = errors.New("configuration rollback source changed")
	// ErrRollbackV4StaleGeneration reports a rollback attempted after the
	// trust generation moved. The operator must start over from the
	// current state.
	ErrRollbackV4StaleGeneration = errors.New("configuration rollback trust generation changed")
)

// V4Preview is the exact Config-4 replacement an operator confirms.
// SourceDocument and SourceGeneration pin the state the preview was rendered
// from; Replacement is the only byte sequence ApplyV4 installs. RetainedPeers
// and DroppedPeers name every configured peer exactly once; BlockingPeers is
// empty on success and names the unenrolled peers on refusal.
type V4Preview struct {
	SourceVersion    string
	SourceDocument   []byte
	SourceGeneration uint64
	CredentialID     string
	Replacement      []byte
	RetainedPeers    []string
	DroppedPeers     []string
	BlockingPeers    []string
}

// V4ApplyResult reports a durable Config-4 apply with the committed trust
// generation that authorizes the new streams.
type V4ApplyResult struct {
	Migration  MigrationResult
	Generation uint64
}

// RollbackV4Options carries the explicit operator acknowledgement. Without
// AcknowledgeNoHostChannelAssurance the rollback refuses: returning to a
// legacy installation abandons Host Channel assurance, and that is never a
// connection retry.
type RollbackV4Options struct {
	AcknowledgeNoHostChannelAssurance bool
	BackupPath                        string
}

// PreviewV4 renders the exact Config-4 replacement for an explicit operator
// decision. Migration is never first-connect behavior: the source must
// already be Configuration 3.0.0 (older inputs first use the existing
// explicit migrations), the selected credential must be actively enrolled for
// the local host, and every retained peer must be completely enrolled
// locally. The preview commits nothing.
func PreviewV4(inputs Inputs, overrides Overrides, trust hosttrust.Snapshot, credentialID string, dropPeers []string, now time.Time) (V4Preview, error) {
	snapshot, err := Load(inputs, overrides)
	if err != nil {
		return V4Preview{}, err
	}
	loaded, decoded := snapshot.Configuration()
	if !snapshot.ConfigPresent() || !decoded {
		return V4Preview{}, migrationError(MigrationError{Operation: "load source", Err: ErrMigrationSourceAbsent})
	}
	if loaded.SourceVersion != CurrentVersion {
		return V4Preview{}, migrationError(MigrationError{Operation: "select source", Err: ErrMigrationV4Source})
	}
	preview, err := renderV4Preview(snapshot.Document(), loaded.Value, trust, credentialID, dropPeers, now, inputs)
	if err != nil {
		return preview, err
	}
	return preview, nil
}

func renderV4Preview(document []byte, value Configuration, trust hosttrust.Snapshot, credentialID string, dropPeers []string, now time.Time, inputs Inputs) (V4Preview, error) {
	preview := V4Preview{
		SourceVersion:    CurrentVersion,
		SourceDocument:   append([]byte(nil), document...),
		SourceGeneration: trust.Generation,
		CredentialID:     credentialID,
	}
	digest, err := scalar.ParseDigest(credentialID)
	if err != nil {
		return preview, migrationError(MigrationError{Operation: "select credential", Err: errors.Join(ErrMigrationV4Credential, err)})
	}
	local, found := findTrustEntry(trust, digest.String())
	if !found || local.HostID.String() != value.HostID || local.State != hosttrust.EntryActive {
		return preview, migrationError(MigrationError{Operation: "select credential", Err: ErrMigrationV4Credential})
	}
	if err := checkCredentialWindow(local.LeafDER, local.RootDER, now); err != nil {
		return preview, migrationError(MigrationError{Operation: "select credential", Err: errors.Join(ErrMigrationV4Credential, err)})
	}
	dropped := map[string]struct{}{}
	for _, peer := range dropPeers {
		if _, exists := dropped[peer]; exists {
			return preview, migrationError(MigrationError{Operation: "select removals", Err: ErrMigrationV4DropUnknown})
		}
		dropped[peer] = struct{}{}
	}
	var retained []Peer
	for _, peer := range value.Mesh.Peers {
		if _, remove := dropped[peer.HostID]; remove {
			preview.DroppedPeers = append(preview.DroppedPeers, peer.HostID)
			continue
		}
		if _, remove := dropped[peer.Name]; remove {
			return preview, migrationError(MigrationError{Operation: "select removals", Err: ErrMigrationV4DropUnknown})
		}
		preview.RetainedPeers = append(preview.RetainedPeers, peer.HostID)
		retained = append(retained, peer)
	}
	for peer := range dropped {
		known := false
		for _, candidate := range value.Mesh.Peers {
			if candidate.HostID == peer {
				known = true
				break
			}
		}
		if !known {
			return preview, migrationError(MigrationError{Operation: "select removals", Err: ErrMigrationV4DropUnknown})
		}
	}
	for _, peer := range retained {
		if !peerEnrolled(trust, peer.HostID, now) {
			preview.BlockingPeers = append(preview.BlockingPeers, peer.HostID)
		}
	}
	if len(preview.BlockingPeers) > 0 {
		return preview, migrationError(MigrationError{Operation: "validate peer enrollment", Err: ErrMigrationV4PeerEnrollment})
	}
	next := value
	next.Mesh.Transport = TransportSSHTLS13
	next.Mesh.HostChannel = &HostChannel{Version: HostChannelVersion, CredentialID: digest.String()}
	next.Mesh.Peers = retained
	context := DecodeContext{RuntimePlatform: inputs.Platform, BackendSettings: inputs.BackendSettings}
	replacement, err := EncodeVersion4(next, context)
	if err != nil {
		return preview, migrationError(MigrationError{Operation: "encode target", Err: err}) // config-refusal-subsumed: the preview pre-validates credential, enrollment, removals and source under the same context, so the v4 encoder admits only values its own clauses already accepted; any refusal here repeats a clause pinned above
	}
	preview.Replacement = replacement
	return preview, nil
}
