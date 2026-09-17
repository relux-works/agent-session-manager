package hosttrust

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"path/filepath"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// EnrollInput is the explicit local operator authorization of one enrollment
// tuple plus the imported public bytes. The Authorized fingerprints must come
// from an independent authenticated out-of-band channel: SSH discovery, a
// first hello, a remote claim, or a certificate being self-signed is never
// approval, and this function cannot tell where the caller got them. What it
// guarantees is that the imported bytes reproduce the authorized tuple and
// the profile before anything commits.
type EnrollInput struct {
	HostID     string
	LeafDER    []byte
	RootDER    []byte
	Authorized Fingerprints
	EnrolledAt time.Time
}

// Enroll commits one active peer credential after verifying the authorized
// tuple, the profile, lifetimes, and the cross-entry uniqueness rules.
// Enrollment is one-directional: it never implies reciprocity.
func (store *Store) Enroll(input EnrollInput) (string, error) {
	parsed, err := scalar.ParseUUIDv7(input.HostID)
	if err != nil {
		return "", trustError(TrustError{Operation: "enroll host", Err: errors.Join(ErrEnrollmentRefused, err)})
	}
	observed, err := FingerprintsOf(input.LeafDER, input.RootDER)
	if err != nil {
		return "", err
	}
	if observed != input.Authorized {
		return "", trustError(TrustError{Operation: "enroll fingerprints", Err: ErrEnrollmentRefused})
	}
	if err := VerifyProfile(input.LeafDER, input.RootDER, parsed.String(), input.EnrolledAt); err != nil {
		return "", err
	}
	enrolledAt, err := scalar.ParseTimestamp(input.EnrolledAt.UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		return "", trustError(TrustError{Operation: "enroll timestamp", Err: errors.Join(ErrEnrollmentRefused, err)})
	}
	entry := CredentialEntry{
		HostID:       parsed,
		CredentialID: scalar.SHA256Digest(input.LeafDER),
		SPKIID:       scalar.SHA256Digest(mustSPKI(input.LeafDER)),
		RootID:       scalar.SHA256Digest(input.RootDER),
		LeafDER:      append([]byte(nil), input.LeafDER...),
		RootDER:      append([]byte(nil), input.RootDER...),
		State:        EntryActive,
		EnrolledAt:   enrolledAt,
	}
	if err := store.transact(func(trust *TrustStore) error {
		return admitEntry(trust, entry)
	}); err != nil {
		return "", err
	}
	return entry.CredentialID.String(), nil
}

func mustSPKI(leafDER []byte) []byte {
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return nil
	}
	return leaf.RawSubjectPublicKeyInfo
}

// admitEntry enforces the cross-entry rules across all entries including
// revoked tombstones: a leaf, leaf public key or root must never map to more
// than one host UUID, duplicate roots are invalid even for the same UUID, and
// at most two non-revoked credentials may belong to one host.
func admitEntry(trust *TrustStore, entry CredentialEntry) error {
	if len(trust.Entries) >= MaxEntries {
		return trustError(TrustError{Operation: "admit entry", Err: ErrEnrollmentRefused})
	}
	var live uint64
	for _, existing := range trust.Entries {
		if bytes.Equal(existing.LeafDER, entry.LeafDER) {
			return trustError(TrustError{Operation: "admit entry", Err: ErrEnrollmentRefused})
		}
		if existing.SPKIID.String() == entry.SPKIID.String() {
			return trustError(TrustError{Operation: "admit entry", Err: ErrEnrollmentRefused})
		}
		if existing.RootID.String() == entry.RootID.String() {
			return trustError(TrustError{Operation: "admit entry", Err: ErrEnrollmentRefused})
		}
		if existing.CredentialID.String() == entry.CredentialID.String() {
			return trustError(TrustError{Operation: "admit entry", Err: ErrEnrollmentRefused})
		}
		if existing.HostID.String() == entry.HostID.String() && existing.State != EntryRevoked {
			live++
		}
	}
	if live >= 2 {
		return trustError(TrustError{Operation: "admit entry", Err: ErrEnrollmentRefused})
	}
	trust.Entries = append(trust.Entries, entry)
	return nil
}

// Issue creates a local credential: fresh keys and profile-exact
// certificates, owner-only custody files, and a self-enrolled active entry,
// committed atomically in that order. Custody files are written before the
// trust commit so a crash can only leave inert files without authority, never
// authority without verifiable custody; a failed trust commit removes the
// staged credential directory again.
func (store *Store) Issue(hostID string, now time.Time) (string, error) {
	issued, err := IssueCredential(hostID, now, nil)
	if err != nil {
		return "", err
	}
	credentialHex := scalar.SHA256Digest(issued.LeafDER).Hex()
	directory := filepath.Join(store.root, credentialsDir, credentialHex)
	if err := ensureOwnerDir(store.fs, directory, 0o700); err != nil {
		return "", err
	}
	committed := false
	defer cleanupCredentialDirectory(store.fs, directory, &committed)
	if err := writeCustodyFile(store, directory, certificateFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issued.LeafDER})); err != nil {
		return "", err
	}
	if err := writeCustodyFile(store, directory, privateKeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: issued.LeafKeyDER})); err != nil {
		return "", err
	}
	if err := writeCustodyFile(store, directory, rootFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: issued.RootDER})); err != nil {
		return "", err
	}
	enrolledAt, err := scalar.ParseTimestamp(now.UTC().Format("2006-01-02T15:04:05.000Z"))
	if err != nil {
		return "", trustError(TrustError{Operation: "issue timestamp", Err: errors.Join(ErrCredentialProfile, err)})
	}
	entry := CredentialEntry{
		HostID:       issued.HostID,
		CredentialID: scalar.SHA256Digest(issued.LeafDER),
		SPKIID:       scalar.SHA256Digest(mustSPKI(issued.LeafDER)),
		RootID:       scalar.SHA256Digest(issued.RootDER),
		LeafDER:      issued.LeafDER,
		RootDER:      issued.RootDER,
		State:        EntryActive,
		EnrolledAt:   enrolledAt,
	}
	if err := store.transact(func(trust *TrustStore) error {
		return admitEntry(trust, entry)
	}); err != nil {
		return "", err
	}
	committed = true
	return entry.CredentialID.String(), nil
}

func writeCustodyFile(store *Store, directory, name string, contents []byte) error {
	staged, err := store.fs.CreateTemp(directory, ".custody-stage-*", 0o600)
	if err != nil {
		return trustError(TrustError{Operation: "write custody file", Err: errors.Join(ErrTrustDurability, err)})
	}
	staging := staged.Name()
	// On platforms where mode bits do not govern access, install the
	// owner-only ACL before the key material is written.
	if err := secureStaged(staging, false); err != nil {
		_ = staged.Close()
		_ = store.fs.Remove(staging)
		return err
	}
	failed := true
	defer cleanupStagedFile(store.fs, staging, &failed)
	if err := writeAll(staged, contents); err != nil {
		_ = staged.Close()
		return trustError(TrustError{Operation: "write custody file", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return trustError(TrustError{Operation: "sync custody file", Err: errors.Join(ErrTrustDurability, err)})
	}
	if err := staged.Close(); err != nil {
		return trustError(TrustError{Operation: "sync custody file", Err: errors.Join(ErrTrustDurability, err)})
	}
	target := filepath.Join(directory, name)
	if err := store.fs.Rename(staging, target); err != nil {
		return trustError(TrustError{Operation: "install custody file", Err: errors.Join(ErrTrustDurability, err)})
	}
	info, err := store.fs.Lstat(target)
	if err != nil {
		return trustError(TrustError{Operation: "inspect custody file", Err: errors.Join(ErrTrustStoreUnreadable, err)})
	}
	if err := verifyOwnerFile(target, info, 0o600); err != nil {
		return err
	}
	failed = false
	return nil
}

func cleanupCredentialDirectory(filesystem FileSystem, directory string, committed *bool) {
	if *committed {
		return
	}
	_ = filesystem.Remove(filepath.Join(directory, certificateFile))
	_ = filesystem.Remove(filepath.Join(directory, privateKeyFile))
	_ = filesystem.Remove(filepath.Join(directory, rootFile))
	_ = filesystem.Remove(directory)
}

// ValidateCustody checks the local credential files before use: owner-only
// regular files, parseable PEMs, private/public match, profile, and the
// configured host binding. No other file may select identity.
func (store *Store) ValidateCustody(credentialHex, hostID string, now time.Time) error {
	const operation = "validate custody"
	directory, err := store.CredentialDir(credentialHex)
	if err != nil {
		return err
	}
	leafPEM, err := readOwnerFile(store, filepath.Join(directory, certificateFile))
	if err != nil {
		return err
	}
	keyPEM, err := readOwnerFile(store, filepath.Join(directory, privateKeyFile))
	if err != nil {
		return err
	}
	rootPEM, err := readOwnerFile(store, filepath.Join(directory, rootFile))
	if err != nil {
		return err
	}
	leafBlock, _ := pem.Decode(leafPEM)
	keyBlock, _ := pem.Decode(keyPEM)
	rootBlock, _ := pem.Decode(rootPEM)
	if leafBlock == nil || keyBlock == nil || rootBlock == nil {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialCustody})
	}
	leaf, err := x509.ParseCertificate(leafBlock.Bytes)
	if err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialCustody, err)})
	}
	// The contained leaf must reproduce the credential directory: no other
	// file may select identity, including a valid credential filed under a
	// different credential's name.
	if scalar.SHA256Digest(leaf.Raw).Hex() != credentialHex {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialCustody})
	}
	root, err := x509.ParseCertificate(rootBlock.Bytes)
	if err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialCustody, err)})
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialCustody, err)})
	}
	private, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialCustody})
	}
	public, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialCustody})
	}
	if private.PublicKey.X.Cmp(public.X) != 0 || private.PublicKey.Y.Cmp(public.Y) != 0 {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialCustody})
	}
	return VerifyProfile(leaf.Raw, root.Raw, hostID, now)
}

func readOwnerFile(store *Store, path string) ([]byte, error) {
	info, err := store.fs.Lstat(path)
	if err != nil {
		return nil, trustError(TrustError{Operation: "read custody file", Err: errors.Join(ErrCredentialCustody, err)})
	}
	if err := verifyOwnerFile(path, info, 0o600); err != nil {
		return nil, err
	}
	contents, err := store.fs.ReadFile(path)
	if err != nil {
		return nil, trustError(TrustError{Operation: "read custody file", Err: errors.Join(ErrCredentialCustody, err)})
	}
	return contents, nil
}
