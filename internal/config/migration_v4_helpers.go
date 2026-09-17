package config

import (
	"crypto/x509"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
)

// CoherentSnapshot is one coherent configuration-plus-trust read: the
// trust generation and the configuration document observed under the same
// shared authorization hold, so no trust or config commit can land between
// them. Streams and dispatches bind Generation; any use must re-check
// currency before crossing a side-effect boundary.
type CoherentSnapshot struct {
	Config     Snapshot
	Trust      hosttrust.TrustStore
	Generation uint64
}

// LoadCoherent is the production pairing for readers that must never mix an
// old allowlist with new credentials (Section 11.10.3): it holds the
// authorization lock across the committed trust read and the configuration
// load, converging pending joint intent first so a surviving reader
// observes the same pair as a reopened store. PreviewV4 callers that hold
// only a separately read trust snapshot should obtain the pair here
// instead.
func LoadCoherent(inputs Inputs, overrides Overrides, store *hosttrust.Store) (CoherentSnapshot, error) {
	var coherent CoherentSnapshot
	if store == nil {
		return CoherentSnapshot{}, migrationError(MigrationError{Operation: "load coherent snapshot", Err: hosttrust.ErrTrustStoreMissing})
	}
	// Converge the trust-side barrier before decoding the configuration. An
	// interrupted joint replacement must report its typed intervention refusal
	// even when an operator edit made the in-flight configuration undecodable;
	// decoding first would turn a pair-integrity failure into an unrelated
	// configuration error. The final shared snapshot below repeats the read
	// under the lock, so this probe is only the ordered admission barrier.
	if _, err := store.ReadSnapshot(); err != nil {
		return CoherentSnapshot{}, err
	}
	// The configuration path and StateRoot are the configuration-side
	// designation of the machine-local store. The immutable sidecar is the
	// trust-side designation. Require both directions before taking a
	// coherent snapshot; a store opened for another resource must never lend
	// its generation to this document.
	selected, err := Load(inputs, overrides)
	if err != nil {
		return CoherentSnapshot{}, err
	}
	localPaths := selected.LocalPaths()
	if err := store.ValidateBoundPaths(localPaths); err != nil {
		return CoherentSnapshot{}, err
	}
	if err := store.WithSharedSnapshotForConfig(localPaths, func(current hosttrust.TrustStore) error {
		snapshot, err := Load(inputs, overrides)
		if err != nil {
			return err
		}
		coherent = CoherentSnapshot{Config: snapshot, Trust: current, Generation: current.Generation}
		return nil
	}); err != nil {
		return CoherentSnapshot{}, err
	}
	return coherent, nil
}

func previewCredentialHex(credentialID string) string {
	return strings.TrimPrefix(credentialID, "sha256:")
}

func findTrustEntry(trust hosttrust.Snapshot, credentialID string) (hosttrust.CredentialEntry, bool) {
	for _, entry := range trust.Trust.Entries {
		if entry.CredentialID.String() == credentialID {
			return entry, true
		}
	}
	return hosttrust.CredentialEntry{}, false
}

// checkCredentialWindow enforces current validity of both certificates
// against the actual local UTC time, inclusively within notBefore/notAfter
// with no extra acceptance skew or expiry grace.
func checkCredentialWindow(leafDER, rootDER []byte, now time.Time) error {
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return err
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		return err
	}
	instant := now.UTC()
	for _, certificate := range []*x509.Certificate{leaf, root} {
		if instant.Before(certificate.NotBefore) || instant.After(certificate.NotAfter) {
			return errors.New("credential outside validity window")
		}
	}
	return nil
}

func peerEnrolled(trust hosttrust.Snapshot, hostID string, now time.Time) bool {
	for _, entry := range trust.Trust.Entries {
		if entry.HostID.String() != hostID || entry.State != hosttrust.EntryActive {
			continue
		}
		if checkCredentialWindow(entry.LeafDER, entry.RootDER, now) == nil {
			return true
		}
	}
	return false
}

// writeTempReplace durably restores document over filename after a trust
// commit failure. It is recovery, not a migration: no backup is published.
// It runs only from the JointCommit restore closures under the exclusive
// authorization hold (see the write-path census); the census gate fails
// any other caller, and a forged, absent or released hold refuses before
// anything is staged.
func writeTempReplace(hold hosttrust.HeldExclusive, filesystem migrationFileSystem, paths localstore.ResolvedPaths, document []byte) (string, error) {
	if err := requireHoldForConfig(hold, paths); err != nil {
		return "", err
	}
	filename, err := configPathFromLocalPaths(paths)
	if err != nil {
		return "", err
	}
	directory := filepath.Dir(filename)
	staged, err := writeTempFile(filesystem, directory, ".ax-config-restore-*", document, 0o600)
	if err != nil {
		return "", err
	}
	if err := filesystem.Rename(staged, filename); err != nil {
		_ = filesystem.Remove(staged)
		return "", err
	}
	if err := syncDirectory(filesystem, directory); err != nil {
		return "", err
	}
	return staged, nil
}
