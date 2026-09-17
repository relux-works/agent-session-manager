package hosttrust

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
)

var oidCommonName = asn1.ObjectIdentifier{2, 5, 4, 3}

// subjectIsOnlyCN accepts exactly the profile subject: the expected common
// name and no other subject attribute. The parsed CN is itself one ATV in
// Names, so CN-only means precisely one ATV with the CN OID and value.
func subjectIsOnlyCN(name pkix.Name, commonName string) bool {
	if name.CommonName != commonName || name.SerialNumber != "" {
		return false
	}
	if len(name.Country) != 0 || len(name.Organization) != 0 || len(name.OrganizationalUnit) != 0 {
		return false
	}
	if len(name.Locality) != 0 || len(name.Province) != 0 || len(name.StreetAddress) != 0 || len(name.PostalCode) != 0 {
		return false
	}
	if len(name.ExtraNames) != 0 || len(name.Names) != 1 {
		return false
	}
	if !name.Names[0].Type.Equal(oidCommonName) {
		return false
	}
	value, ok := name.Names[0].Value.(string)
	return ok && value == commonName
}

func checkP256Point(certificate *x509.Certificate) error {
	key, ok := certificate.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrCredentialProfile
	}
	if key.Curve != elliptic.P256() {
		return ErrCredentialProfile
	}
	if !elliptic.P256().IsOnCurve(key.X, key.Y) {
		return ErrCredentialProfile
	}
	return nil
}

func hasExactlyClientServerEKU(leaf *x509.Certificate) bool {
	if len(leaf.ExtKeyUsage) != 2 || len(leaf.UnknownExtKeyUsage) != 0 {
		return false
	}
	var client, server bool
	for _, usage := range leaf.ExtKeyUsage {
		switch usage {
		case x509.ExtKeyUsageClientAuth:
			client = true
		case x509.ExtKeyUsageServerAuth:
			server = true
		default:
			return false
		}
	}
	return client && server
}

// checkNoExtraNames refuses every name and dependency outside the profile:
// no wildcard or CN-only identity, no alternative UUID, no extra subject
// attribute, no other SAN type, and no AIA fetch, CRL URL or OCSP endpoint.
func checkNoExtraNames(certificate *x509.Certificate) error {
	if len(certificate.EmailAddresses) != 0 || len(certificate.IPAddresses) != 0 || len(certificate.URIs) != 0 {
		return ErrCredentialProfile
	}
	if len(certificate.OCSPServer) != 0 || len(certificate.IssuingCertificateURL) != 0 {
		return ErrCredentialProfile
	}
	if len(certificate.DNSNames) != 0 {
		for _, name := range certificate.DNSNames {
			for _, unit := range name {
				if unit == '*' {
					return ErrCredentialProfile
				}
			}
		}
	}
	if len(certificate.CRLDistributionPoints) != 0 {
		return ErrCredentialProfile
	}
	if len(certificate.PolicyIdentifiers) != 0 || len(certificate.PermittedDNSDomains) != 0 {
		return ErrCredentialProfile
	}
	return nil
}

func checkKeyIdentifiers(leaf, root *x509.Certificate) error {
	leafKey, ok := leaf.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrCredentialProfile
	}
	rootKey, ok := root.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return ErrCredentialProfile
	}
	leafSKI, err := subjectKeyID(leafKey)
	if err != nil {
		return ErrCredentialProfile
	}
	rootSKI, err := subjectKeyID(rootKey)
	if err != nil {
		return ErrCredentialProfile
	}
	if !bytes.Equal(leaf.SubjectKeyId, leafSKI) || !bytes.Equal(root.SubjectKeyId, rootSKI) {
		return ErrCredentialProfile
	}
	if !bytes.Equal(leaf.AuthorityKeyId, rootSKI) || !bytes.Equal(root.AuthorityKeyId, rootSKI) {
		return ErrCredentialProfile
	}
	return nil
}

func checkSerials(leaf, root *x509.Certificate) error {
	if leaf.SerialNumber == nil || root.SerialNumber == nil {
		return ErrCredentialProfile
	}
	if leaf.SerialNumber.Sign() <= 0 || root.SerialNumber.Sign() <= 0 {
		return ErrCredentialProfile
	}
	if len(leaf.SerialNumber.Bytes()) > 20 || len(root.SerialNumber.Bytes()) > 20 {
		return ErrCredentialProfile
	}
	if leaf.SerialNumber.Cmp(root.SerialNumber) == 0 {
		return ErrCredentialProfile
	}
	return nil
}
