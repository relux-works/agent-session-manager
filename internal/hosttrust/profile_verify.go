package hosttrust

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"io"
	"math/big"
	"time"
)

func randomSerial(randomness io.Reader) (*big.Int, error) {
	serial := make([]byte, 16)
	if _, err := io.ReadFull(randomness, serial); err != nil {
		return nil, err
	}
	value := new(big.Int).SetBytes(serial)
	if value.Sign() == 0 {
		value.SetInt64(1)
	}
	return value, nil
}

func subjectKeyID(key *ecdsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}
	var info struct {
		Algorithm        asn1.RawValue
		SubjectPublicKey asn1.BitString
	}
	if _, err := asn1.Unmarshal(der, &info); err != nil {
		return nil, err
	}
	sum := sha1.Sum(info.SubjectPublicKey.Bytes)
	return sum[:], nil
}

// zeroECDSAKey best-effort destroys private scalar material after issuance.
// The root key is additionally never serialized, stored or returned.
func zeroECDSAKey(key *ecdsa.PrivateKey) {
	if key == nil || key.D == nil {
		return
	}
	bits := key.D.Bits()
	for index := range bits {
		bits[index] = 0
	}
	key.D = new(big.Int)
}

// VerifyProfile checks the exact Host Credential Profile 1 fields for a leaf
// and root pair at enrollment and handshake time: signatures, exact subject
// and issuer names, critical basic constraints and key usages, exact EKU set,
// the single DNS SAN, validity windows against the actual local UTC time
// (inclusive, no skew or grace), P-256 points, SKI/AKI construction, and the
// absence of intermediates, AIA/CRL/OCSP dependencies and extra attributes.
func VerifyProfile(leafDER, rootDER []byte, hostID string, now time.Time) error {
	const operation = "verify credential profile"
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialProfile, err)})
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialProfile, err)})
	}
	if err := checkP256Point(leaf); err != nil {
		return trustError(TrustError{Operation: operation, Err: err})
	}
	if err := checkP256Point(root); err != nil {
		return trustError(TrustError{Operation: operation, Err: err})
	}
	if leaf.Version != 3 || root.Version != 3 {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if leaf.SignatureAlgorithm != x509.ECDSAWithSHA256 || root.SignatureAlgorithm != x509.ECDSAWithSHA256 {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if leaf.PublicKeyAlgorithm != x509.ECDSA || root.PublicKeyAlgorithm != x509.ECDSA {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if err := root.CheckSignatureFrom(root); err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialProfile, err)})
	}
	if err := leaf.CheckSignatureFrom(root); err != nil {
		return trustError(TrustError{Operation: operation, Err: errors.Join(ErrCredentialProfile, err)})
	}
	if !subjectIsOnlyCN(leaf.Subject, leafCommonName(hostID)) {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if !subjectIsOnlyCN(root.Subject, rootCommonName(hostID)) {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if !subjectIsOnlyCN(leaf.Issuer, root.Subject.CommonName) {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if !root.BasicConstraintsValid || !root.IsCA || !root.MaxPathLenZero || root.MaxPathLen != 0 {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if !leaf.BasicConstraintsValid || leaf.IsCA {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if root.KeyUsage != x509.KeyUsageCertSign {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if leaf.KeyUsage != x509.KeyUsageDigitalSignature {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if len(root.ExtKeyUsage) != 0 || len(root.UnknownExtKeyUsage) != 0 {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if !hasExactlyClientServerEKU(leaf) {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != hostDNSName(hostID) {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	if err := checkNoExtraNames(leaf); err != nil {
		return trustError(TrustError{Operation: operation, Err: err})
	}
	if err := checkNoExtraNames(root); err != nil {
		return trustError(TrustError{Operation: operation, Err: err})
	}
	if err := checkKeyIdentifiers(leaf, root); err != nil {
		return trustError(TrustError{Operation: operation, Err: err})
	}
	if err := checkSerials(leaf, root); err != nil {
		return trustError(TrustError{Operation: operation, Err: err})
	}
	instant := now.UTC()
	for _, certificate := range []*x509.Certificate{leaf, root} {
		if instant.Before(certificate.NotBefore) || instant.After(certificate.NotAfter) {
			return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
		}
	}
	if len(leaf.UnhandledCriticalExtensions) != 0 || len(root.UnhandledCriticalExtensions) != 0 {
		return trustError(TrustError{Operation: operation, Err: ErrCredentialProfile})
	}
	return nil
}
