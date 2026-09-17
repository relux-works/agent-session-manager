package hosttrust

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"path/filepath"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// RotationBound is the maximum overlap of an old credential after the trust
// transition admits its replacement: 24 hours, and never past old leaf
// expiry. There is no overlap extension.
const RotationBound = 24 * time.Hour

// Rotate keeps the host UUID and creates a new credential with two fresh keys
// and certificates. The local atomic trust transition admits the new entry as
// active and marks the old entry retiring with a fixed retire_at no more than
// 24 hours after the transition and no later than old leaf expiry. No second
// rotation starts until the retiring entry is revoked. Failed enrollment
// leaves the new route unavailable, never auto-trusted.
func (store *Store) Rotate(hostID string, now time.Time) (newCredentialID string, retireAt time.Time, err error) {
	parsed, err := scalar.ParseUUIDv7(hostID)
	if err != nil {
		return "", time.Time{}, trustError(TrustError{Operation: "rotate host", Err: errors.Join(ErrRotationRefused, err)})
	}
	issued, err := IssueCredential(parsed.String(), now, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	credentialHex := scalar.SHA256Digest(issued.LeafDER).Hex()
	directory := filepath.Join(store.root, credentialsDir, credentialHex)
	if err := ensureOwnerDir(store.fs, directory, 0o700); err != nil {
		return "", time.Time{}, err
	}
	committed := false
	defer cleanupCredentialDirectory(store.fs, directory, &committed)
	if err := writeCustodyFile(store, directory, certificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issued.LeafDER})); err != nil {
		return "", time.Time{}, err
	}
	if err := writeCustodyFile(store, directory, privateKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: issued.LeafKeyDER})); err != nil {
		return "", time.Time{}, err
	}
	if err := writeCustodyFile(store, directory, rootFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issued.RootDER})); err != nil {
		return "", time.Time{}, err
	}
	enrolledAt, err := scalar.ParseTimestamp(now.UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		return "", time.Time{}, trustError(TrustError{Operation: "rotate timestamp", Err: errors.Join(ErrRotationRefused, err)})
	}
	newEntry := CredentialEntry{
		HostID:       issued.HostID,
		CredentialID: scalar.SHA256Digest(issued.LeafDER),
		SPKIID:       scalar.SHA256Digest(mustSPKI(issued.LeafDER)),
		RootID:       scalar.SHA256Digest(issued.RootDER),
		LeafDER:      issued.LeafDER,
		RootDER:      issued.RootDER,
		State:        EntryActive,
		EnrolledAt:   enrolledAt,
	}
	var fixedRetireAt time.Time
	if err := store.transact(func(trust *TrustStore) error {
		old, err := rotationPredecessor(trust, parsed.String())
		if err != nil {
			return err
		}
		oldLeaf, err := x509.ParseCertificate(old.LeafDER)
		if err != nil {
			return trustError(TrustError{Operation: "rotate predecessor", Err: errors.Join(ErrRotationRefused, err)})
		}
		deadline := now.Add(RotationBound)
		if oldLeaf.NotAfter.Before(deadline) {
			deadline = oldLeaf.NotAfter
		}
		if !deadline.After(now) {
			return trustError(TrustError{Operation: "rotate window", Err: ErrRotationRefused})
		}
		stamp, err := scalar.ParseTimestamp(deadline.UTC().Format("2006-01-02T15:04:05.000Z"))
		if err != nil {
			return trustError(TrustError{Operation: "rotate window", Err: errors.Join(ErrRotationRefused, err)})
		}
		fixedRetireAt, err = stamp.Time()
		if err != nil {
			return trustError(TrustError{Operation: "rotate window", Err: errors.Join(ErrRotationRefused, err)})
		}
		for index := range trust.Entries {
			if trust.Entries[index].CredentialID.String() == old.CredentialID.String() {
				trust.Entries[index].State = EntryRetiring
				retiring := stamp
				trust.Entries[index].RetireAt = &retiring
			}
		}
		return admitEntry(trust, newEntry)
	}); err != nil {
		return "", time.Time{}, err
	}
	committed = true
	return newEntry.CredentialID.String(), fixedRetireAt, nil
}

// rotationPredecessor selects the single active credential of a host. A
// retiring entry blocks a second rotation; zero or multiple active entries
// refuse.
func rotationPredecessor(trust *TrustStore, hostID string) (CredentialEntry, error) {
	var active []CredentialEntry
	for _, entry := range trust.Entries {
		if entry.HostID.String() != hostID {
			continue
		}
		switch entry.State {
		case EntryRetiring:
			return CredentialEntry{}, trustError(TrustError{Operation: "rotate predecessor", Err: ErrRotationRefused})
		case EntryActive:
			active = append(active, entry)
		}
	}
	if len(active) != 1 {
		return CredentialEntry{}, trustError(TrustError{Operation: "rotate predecessor", Err: ErrRotationRefused})
	}
	return active[0], nil
}

// MarkRetiring records a peer's announced rotation locally: the operator
// caught up out of band, enrolled the peer's new tuple, and now bounds the
// old credential with a fixed retire_at. The bound is no more than 24 hours
// after this transition, later than enrollment, and no later than old leaf
// expiry; there is no overlap extension, and only an active entry may retire.
func (store *Store) MarkRetiring(credentialID string, retireAt, now time.Time) error {
	digest, err := scalar.ParseDigest(credentialID)
	if err != nil {
		return trustError(TrustError{Operation: "retire credential", Err: errors.Join(ErrRotationRefused, err)})
	}
	stamp, err := scalar.ParseTimestamp(retireAt.UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		return trustError(TrustError{Operation: "retire credential", Err: errors.Join(ErrRotationRefused, err)})
	}
	return store.transact(func(trust *TrustStore) error {
		for index := range trust.Entries {
			if trust.Entries[index].CredentialID.String() != digest.String() {
				continue
			}
			entry := &trust.Entries[index]
			if entry.State != EntryActive {
				return trustError(TrustError{Operation: "retire credential", Err: ErrRotationRefused})
			}
			enrolledAt, err := entry.EnrolledAt.Time()
			if err != nil {
				return trustError(TrustError{Operation: "retire credential", Err: errors.Join(ErrRotationRefused, err)})
			}
			fixed, err := stamp.Time()
			if err != nil {
				return trustError(TrustError{Operation: "retire credential", Err: errors.Join(ErrRotationRefused, err)})
			}
			if !fixed.After(now) || !fixed.After(enrolledAt) || fixed.After(now.Add(RotationBound)) {
				return trustError(TrustError{Operation: "retire credential", Err: ErrRotationRefused})
			}
			// The bound stops no later than old leaf expiry, like the local
			// rotation transition: admission already ends at expiry, so a
			// later retire_at would promise an overlap that cannot admit.
			oldLeaf, err := x509.ParseCertificate(entry.LeafDER)
			if err != nil {
				return trustError(TrustError{Operation: "retire credential", Err: errors.Join(ErrRotationRefused, err)})
			}
			if fixed.After(oldLeaf.NotAfter) {
				return trustError(TrustError{Operation: "retire credential", Err: ErrRotationRefused})
			}
			entry.State = EntryRetiring
			entry.RetireAt = &stamp
			return nil
		}
		return trustError(TrustError{Operation: "retire credential", Err: ErrRotationRefused})
	})
}

// Revoke is an explicit local atomic transition to revoked. It increments
// the generation, forbids all future admissions of that key, and retains the
// public bytes permanently as a reuse tombstone. Revoking an already revoked
// entry refuses: a failed commit reports failure, never absence or success.
// Revocation is local: the operator must distribute it explicitly to every
// peer, and loss or compromise requires revocation plus new out-of-band
// enrollment, never same-key recovery.
func (store *Store) Revoke(credentialID string) error {
	digest, err := scalar.ParseDigest(credentialID)
	if err != nil {
		return trustError(TrustError{Operation: "revoke credential", Err: errors.Join(ErrRevocationRefused, err)})
	}
	return store.transact(func(trust *TrustStore) error {
		for index := range trust.Entries {
			if trust.Entries[index].CredentialID.String() != digest.String() {
				continue
			}
			if trust.Entries[index].State == EntryRevoked {
				return trustError(TrustError{Operation: "revoke credential", Err: ErrRevocationRefused})
			}
			trust.Entries[index].State = EntryRevoked
			trust.Entries[index].RetireAt = nil
			return nil
		}
		return trustError(TrustError{Operation: "revoke credential", Err: ErrRevocationRefused})
	})
}
