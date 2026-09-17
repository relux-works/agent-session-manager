package hosttrust

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	// ProfileVersion is the Host Credential Profile 1 marker used in
	// mesh.host_channel.version and nowhere else.
	ProfileVersion = "1.0.0"
	// RootLifetime is notBefore plus 366 days.
	RootLifetime = 366 * 24 * time.Hour
	// LeafLifetime is notBefore plus 90 days.
	LeafLifetime = 90 * 24 * time.Hour
	// IssuanceSkew moves notBefore 300 seconds before issuance time.
	IssuanceSkew = 300 * time.Second
	// hostSuffix is the only permitted DNS SAN parent.
	hostSuffix = ".host.ax.invalid"
)

func rootCommonName(hostID string) string { return "AX Host Root " + hostID }
func leafCommonName(hostID string) string { return "AX Host " + hostID }
func hostDNSName(hostID string) string    { return hostID + hostSuffix }

// Fingerprints identifies the operator-authorized enrollment tuple: SHA-256
// of the leaf DER, of the leaf SubjectPublicKeyInfo DER, and of the root DER.
type Fingerprints struct {
	Leaf string
	SPKI string
	Root string
}

// IssuedCredential carries freshly issued public bytes plus the leaf private
// key for exactly one custody write. The root private key is never returned:
// issuance destroys it after signing.
type IssuedCredential struct {
	HostID     scalar.UUIDv7
	LeafDER    []byte
	RootDER    []byte
	LeafKeyDER []byte
}

// Formatting issued material must never expose keys, DER bytes or identity.
func (IssuedCredential) String() string   { return "issued host credential" }
func (IssuedCredential) GoString() string { return "issued host credential" }

// FingerprintsOf derives the enrollment tuple from public bytes.
func FingerprintsOf(leafDER, rootDER []byte) (Fingerprints, error) {
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return Fingerprints{}, trustError(TrustError{Operation: "fingerprint leaf", Err: errors.Join(ErrCredentialProfile, err)})
	}
	if _, err := x509.ParseCertificate(rootDER); err != nil {
		return Fingerprints{}, trustError(TrustError{Operation: "fingerprint root", Err: errors.Join(ErrCredentialProfile, err)})
	}
	return Fingerprints{
		Leaf: scalar.SHA256Digest(leafDER).String(),
		SPKI: scalar.SHA256Digest(leaf.RawSubjectPublicKeyInfo).String(),
		Root: scalar.SHA256Digest(rootDER).String(),
	}, nil
}

// IssueCredential generates two distinct local CSPRNG ECDSA P-256 key pairs,
// a self-signed root CA and the leaf it signs, exactly per Host Credential
// Profile 1. The root private key is destroyed before return: it is zeroed in
// memory and never serialized, stored or returned.
func IssueCredential(hostID string, now time.Time, randomness io.Reader) (IssuedCredential, error) {
	parsed, err := scalar.ParseUUIDv7(hostID)
	if err != nil {
		return IssuedCredential{}, trustError(TrustError{Operation: "issue credential", Err: errors.Join(ErrCredentialProfile, err)})
	}
	if randomness == nil {
		randomness = rand.Reader
	}
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), randomness)
	if err != nil {
		return IssuedCredential{}, trustError(TrustError{Operation: "issue root key", Err: errors.Join(ErrCredentialProfile, err)})
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), randomness)
	if err != nil {
		zeroECDSAKey(rootKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue leaf key", Err: errors.Join(ErrCredentialProfile, err)})
	}
	rootSerial, err := randomSerial(randomness)
	if err != nil {
		zeroECDSAKey(rootKey)
		zeroECDSAKey(leafKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue serial", Err: errors.Join(ErrCredentialProfile, err)})
	}
	leafSerial, err := randomSerial(randomness)
	if err != nil {
		zeroECDSAKey(rootKey)
		zeroECDSAKey(leafKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue serial", Err: errors.Join(ErrCredentialProfile, err)})
	}
	for leafSerial.Cmp(rootSerial) == 0 {
		leafSerial, err = randomSerial(randomness)
		if err != nil {
			zeroECDSAKey(rootKey)
			zeroECDSAKey(leafKey)
			return IssuedCredential{}, trustError(TrustError{Operation: "issue serial", Err: errors.Join(ErrCredentialProfile, err)})
		}
	}
	notBefore := now.UTC().Add(-IssuanceSkew)
	rootTemplate := &x509.Certificate{
		SerialNumber:          rootSerial,
		Subject:               pkix.Name{CommonName: rootCommonName(parsed.String())},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(RootLifetime),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}
	rootSKI, err := subjectKeyID(&rootKey.PublicKey)
	if err != nil {
		zeroECDSAKey(rootKey)
		zeroECDSAKey(leafKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue root ski", Err: errors.Join(ErrCredentialProfile, err)})
	}
	rootTemplate.SubjectKeyId = rootSKI
	rootTemplate.AuthorityKeyId = rootSKI
	rootDER, err := x509.CreateCertificate(randomness, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		zeroECDSAKey(rootKey)
		zeroECDSAKey(leafKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue root certificate", Err: errors.Join(ErrCredentialProfile, err)})
	}
	leafTemplate := &x509.Certificate{
		SerialNumber:          leafSerial,
		Subject:               pkix.Name{CommonName: leafCommonName(parsed.String())},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(LeafLifetime),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{hostDNSName(parsed.String())},
	}
	leafSKI, err := subjectKeyID(&leafKey.PublicKey)
	if err != nil {
		zeroECDSAKey(rootKey)
		zeroECDSAKey(leafKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue leaf ski", Err: errors.Join(ErrCredentialProfile, err)})
	}
	leafTemplate.SubjectKeyId = leafSKI
	leafTemplate.AuthorityKeyId = rootSKI
	leafDER, err := x509.CreateCertificate(randomness, leafTemplate, rootTemplate, &leafKey.PublicKey, rootKey)
	zeroECDSAKey(rootKey)
	if err != nil {
		zeroECDSAKey(leafKey)
		return IssuedCredential{}, trustError(TrustError{Operation: "issue leaf certificate", Err: errors.Join(ErrCredentialProfile, err)})
	}
	leafKeyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	zeroECDSAKey(leafKey)
	if err != nil {
		return IssuedCredential{}, trustError(TrustError{Operation: "issue leaf key encoding", Err: errors.Join(ErrCredentialProfile, err)})
	}
	issued := IssuedCredential{HostID: parsed, LeafDER: leafDER, RootDER: rootDER, LeafKeyDER: leafKeyDER}
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, parsed.String(), now); err != nil {
		return IssuedCredential{}, err
	}
	return issued, nil
}
