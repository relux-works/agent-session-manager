package hosttrust

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// EnrollmentMaterial is the explicitly selected public enrollment material an
// operator exchanges out of band. It carries no private key: only
// certificate.pem and root.pem bytes leave the store, never private-key.pem,
// trust internals, backups or authorization caches.
type EnrollmentMaterial struct {
	HostID       string
	CredentialID string
	LeafDER      []byte
	RootDER      []byte
	Fingerprints Fingerprints
}

// Formatting enrollment material redacts DER bytes by default; the operator
// exports explicit fields, never a log line.
func (EnrollmentMaterial) String() string   { return "host enrollment material" }
func (EnrollmentMaterial) GoString() string { return "host enrollment material" }
func (EnrollmentMaterial) Format(state fmt.State, verb rune) {
	_, _ = state.Write([]byte("host enrollment material"))
}

// ExportEnrollment selects the public bytes of one non-revoked entry for
// out-of-band exchange. Revoked tombstones never redistribute: exporting one
// refuses.
func ExportEnrollment(snapshot Snapshot, credentialID string) (EnrollmentMaterial, error) {
	entry, found := findEntry(snapshot.Trust, credentialID)
	if !found {
		return EnrollmentMaterial{}, trustError(TrustError{Operation: "export enrollment", Err: ErrEnrollmentRefused})
	}
	if entry.State == EntryRevoked {
		return EnrollmentMaterial{}, trustError(TrustError{Operation: "export enrollment", Err: ErrEnrollmentRefused})
	}
	fingerprints, err := FingerprintsOf(entry.LeafDER, entry.RootDER)
	if err != nil {
		return EnrollmentMaterial{}, err
	}
	return EnrollmentMaterial{
		HostID:       entry.HostID.String(),
		CredentialID: entry.CredentialID.String(),
		LeafDER:      append([]byte(nil), entry.LeafDER...),
		RootDER:      append([]byte(nil), entry.RootDER...),
		Fingerprints: fingerprints,
	}, nil
}

// ExcludedFromReplication lists the STATE_DIR-relative paths that must never
// enter replication, snapshots, cloning, Session Directory, logs or exported
// diagnostic bundles. Credential, trust, backup, private-key and
// authorization-cache material stays machine-local; operators exchange only
// explicitly selected public enrollment material outside replication.
func ExcludedFromReplication() []string {
	return []string{
		hostChannelDir + "/trust.json",
		hostChannelDir + "/config-binding.json",
		hostChannelDir + "/pending-commit.json",
		hostChannelDir + "/credentials/",
		hostChannelDir + "/lock",
	}
}

// MatchExcludedFromReplication reports whether a STATE_DIR-relative path
// must stay out of replication, snapshots, cloning, Session Directory, logs
// and exported diagnostic bundles. It covers the listed control paths plus
// transient staging files, which carry the same material under rotating
// names. Any path that escapes the state directory fails closed as
// excluded. Every replication, snapshot, clone or diagnostic export surface
// must consult this matcher (and ExcludedConfigDirName for configuration
// directory files) and drop matches; ExportEnrollment material is
// deliberately NOT excluded, since explicitly selected public bytes are the
// only operator exchange outside replication.
func MatchExcludedFromReplication(relative string) bool {
	if relative == "" || relative == "." {
		return false
	}
	slash := filepath.ToSlash(relative)
	if path.IsAbs(slash) {
		return true
	}
	for _, segment := range strings.Split(slash, "/") {
		if segment == ".." {
			return true
		}
	}
	trimmed := path.Clean(slash)
	for _, listed := range ExcludedFromReplication() {
		clean := path.Clean(listed)
		if trimmed == clean || strings.HasPrefix(trimmed, clean+"/") {
			return true
		}
	}
	base := path.Base(trimmed)
	for _, prefix := range []string{stagePrefix, ".custody-stage-", "lock-stage-"} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

// ExcludedConfigDirName reports whether a configuration-directory file name
// carries trust-adjacent material that must stay out of replication: owner-only
// backups, pre-rollback copies and transient migration staging files. Backup
// names embed the source version (".bak.<version>"), so the match is by
// infix, pinned against the names replaceDurably actually publishes.
func ExcludedConfigDirName(base string) bool {
	name := path.Base(filepath.ToSlash(base))
	if strings.Contains(name, ".pre-rollback.") {
		return true
	}
	if strings.HasPrefix(name, ".ax-config-") {
		return true
	}
	if strings.Contains(name, ".ax-config-binding.") {
		return true
	}
	if strings.Contains(name, ".bak.") {
		return true
	}
	return false
}
