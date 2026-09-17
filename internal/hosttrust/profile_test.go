package hosttrust

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestIssueCredentialProfile(t *testing.T) {
	issued, err := IssueCredential(testHostA, testNow, nil)
	if err != nil {
		t.Fatalf("IssueCredential error = %v", err)
	}
	leaf, err := x509.ParseCertificate(issued.LeafDER)
	if err != nil {
		t.Fatalf("parse leaf error = %v", err)
	}
	root, err := x509.ParseCertificate(issued.RootDER)
	if err != nil {
		t.Fatalf("parse root error = %v", err)
	}
	if leaf.Version != 3 || root.Version != 3 {
		t.Fatal("certificates are not X.509 v3")
	}
	if leaf.SignatureAlgorithm != x509.ECDSAWithSHA256 || root.SignatureAlgorithm != x509.ECDSAWithSHA256 {
		t.Fatal("signature algorithm is not ECDSA-with-SHA256")
	}
	for _, key := range []any{leaf.PublicKey, root.PublicKey} {
		public, ok := key.(*ecdsa.PublicKey)
		if !ok || public.Curve != elliptic.P256() {
			t.Fatal("public key is not P-256")
		}
	}
	if leaf.Subject.CommonName != "AX Host "+testHostA {
		t.Fatalf("leaf CN = %q", leaf.Subject.CommonName)
	}
	if root.Subject.CommonName != "AX Host Root "+testHostA {
		t.Fatalf("root CN = %q", root.Subject.CommonName)
	}
	if leaf.Issuer.CommonName != root.Subject.CommonName {
		t.Fatal("leaf issuer does not equal root subject")
	}
	if !root.IsCA || !root.BasicConstraintsValid || !root.MaxPathLenZero || root.MaxPathLen != 0 {
		t.Fatal("root basic constraints are not CA=true pathLen=0")
	}
	if leaf.IsCA || !leaf.BasicConstraintsValid {
		t.Fatal("leaf basic constraints are not CA=false")
	}
	if root.KeyUsage != x509.KeyUsageCertSign || leaf.KeyUsage != x509.KeyUsageDigitalSignature {
		t.Fatal("key usages are not keyCertSign-only / digitalSignature-only")
	}
	if len(root.ExtKeyUsage) != 0 {
		t.Fatal("root carries EKU")
	}
	if len(leaf.DNSNames) != 1 || leaf.DNSNames[0] != testHostA+".host.ax.invalid" {
		t.Fatalf("leaf SAN = %v", leaf.DNSNames)
	}
	wantNotBefore := testNow.Add(-IssuanceSkew)
	if !leaf.NotBefore.Equal(wantNotBefore) || !root.NotBefore.Equal(wantNotBefore) {
		t.Fatalf("notBefore = %v/%v, want %v", leaf.NotBefore, root.NotBefore, wantNotBefore)
	}
	if !leaf.NotAfter.Equal(wantNotBefore.Add(LeafLifetime)) {
		t.Fatalf("leaf notAfter = %v", leaf.NotAfter)
	}
	if !root.NotAfter.Equal(wantNotBefore.Add(RootLifetime)) {
		t.Fatalf("root notAfter = %v", root.NotAfter)
	}
	if leaf.SerialNumber.Sign() <= 0 || root.SerialNumber.Sign() <= 0 {
		t.Fatal("serials are not positive")
	}
	if len(leaf.SerialNumber.Bytes()) > 20 || len(root.SerialNumber.Bytes()) > 20 {
		t.Fatal("serials exceed 20 octets")
	}
	if leaf.SerialNumber.Cmp(root.SerialNumber) == 0 {
		t.Fatal("serials are not distinct")
	}
	leafSKI := skiForTest(t, leaf.PublicKey.(*ecdsa.PublicKey))
	rootSKI := skiForTest(t, root.PublicKey.(*ecdsa.PublicKey))
	if string(leaf.SubjectKeyId) != string(leafSKI) || string(root.SubjectKeyId) != string(rootSKI) {
		t.Fatal("SKI is not SHA-1 of the subjectPublicKey BIT STRING contents")
	}
	if string(leaf.AuthorityKeyId) != string(rootSKI) || string(root.AuthorityKeyId) != string(rootSKI) {
		t.Fatal("AKI is not the root SKI")
	}
	// The leaf private key matches the leaf and marshals as PKCS#8.
	key, err := x509.ParsePKCS8PrivateKey(issued.LeafKeyDER)
	if err != nil {
		t.Fatalf("parse leaf key error = %v", err)
	}
	private, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatal("leaf key is not ECDSA")
	}
	if private.PublicKey.X.Cmp(leaf.PublicKey.(*ecdsa.PublicKey).X) != 0 {
		t.Fatal("leaf private/public mismatch")
	}
	// Fresh issuance uses fresh keys.
	second, err := IssueCredential(testHostA, testNow, nil)
	if err != nil {
		t.Fatalf("second IssueCredential error = %v", err)
	}
	if string(second.LeafDER) == string(issued.LeafDER) {
		t.Fatal("two issuances share leaf bytes")
	}
}

func skiForTest(t *testing.T, key *ecdsa.PublicKey) []byte {
	t.Helper()
	ski, err := subjectKeyID(key)
	if err != nil {
		t.Fatalf("subjectKeyID error = %v", err)
	}
	if len(ski) != sha1.Size {
		t.Fatalf("SKI length = %d, want SHA-1", len(ski))
	}
	return ski
}

func TestIssueCredentialRefusals(t *testing.T) {
	if _, err := IssueCredential("not-a-uuid", testNow, nil); err == nil {
		t.Fatal("IssueCredential(bad uuid) succeeded, want refusal")
	} else if !errors.Is(err, ErrCredentialProfile) {
		t.Fatalf("IssueCredential(bad uuid) error = %v, want ErrCredentialProfile", err)
	}
	if _, err := IssueCredential("0198F4C8-7D40-7E55-8E6F-1234567890AA", testNow, nil); err == nil {
		t.Fatal("IssueCredential(uppercase uuid) succeeded, want refusal")
	} else if !errors.Is(err, ErrCredentialProfile) {
		t.Fatalf("IssueCredential(uppercase uuid) error = %v, want ErrCredentialProfile", err)
	}
}

func TestVerifyProfileTimeBounds(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	leaf, err := x509.ParseCertificate(issued.LeafDER)
	if err != nil {
		t.Fatal(err)
	}
	// Inclusive bounds: exactly notBefore and notAfter verify.
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, testHostA, leaf.NotBefore); err != nil {
		t.Fatalf("VerifyProfile(notBefore) error = %v", err)
	}
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, testHostA, leaf.NotAfter); err != nil {
		t.Fatalf("VerifyProfile(notAfter) error = %v", err)
	}
	// One second outside either bound refuses, with no extra skew or grace.
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, testHostA, leaf.NotBefore.Add(-time.Second)); err == nil {
		t.Fatal("VerifyProfile(before notBefore) succeeded, want refusal")
	}
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, testHostA, leaf.NotAfter.Add(time.Second)); err == nil {
		t.Fatal("VerifyProfile(after notAfter) succeeded, want refusal")
	}
	// Root expiry binds even while the leaf is valid: the root lives 366
	// days, the leaf 90, so day 200 refuses on the root alone.
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, testHostA, testNow.Add(200*24*time.Hour)); err == nil {
		t.Fatal("VerifyProfile(expired root) succeeded, want refusal")
	}
}

func TestVerifyProfileRefusals(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	other := issueForTest(t, testHostB, testNow)
	tests := []struct {
		name    string
		leafDER []byte
		rootDER []byte
		hostID  string
	}{
		{"garbage leaf", []byte{0, 1, 2, 3}, issued.RootDER, testHostA},
		{"garbage root", issued.LeafDER, []byte{0, 1, 2, 3}, testHostA},
		{"swapped pair", issued.RootDER, issued.LeafDER, testHostA},
		{"leaf under foreign root", issued.LeafDER, other.RootDER, testHostA},
		{"wrong host", issued.LeafDER, issued.RootDER, testHostB},
		{"root as leaf", issued.RootDER, issued.RootDER, testHostA},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := VerifyProfile(test.leafDER, test.rootDER, test.hostID, testNow); err == nil {
				t.Fatalf("VerifyProfile(%s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, ErrCredentialProfile) {
				t.Fatalf("VerifyProfile(%s) error = %v, want ErrCredentialProfile", test.name, err)
			}
		})
	}
}

// TestVerifyProfileRefusesJustPastExpiry pins the inclusive bound against a
// one-second grace: half a second past notAfter already refuses.
func TestVerifyProfileRefusesJustPastExpiry(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	leaf, err := x509.ParseCertificate(issued.LeafDER)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyProfile(issued.LeafDER, issued.RootDER, testHostA, leaf.NotAfter.Add(500*time.Millisecond)); err == nil {
		t.Fatal("VerifyProfile(500ms past expiry) succeeded, want refusal: no expiry grace is allowed")
	}
}

// TestVerifyProfileRefusesWrongKeyUsage builds an independently signed pair
// with a widened leaf key usage. The signature is valid under its own root,
// so only the exact key-usage gate refuses.
func TestVerifyProfileRefusesWrongKeyUsage(t *testing.T) {
	leafDER, rootDER := selfSignedPairForTest(t, testHostA,
		x509.KeyUsageDigitalSignature|x509.KeyUsageKeyEncipherment,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		[]string{testHostA + ".host.ax.invalid"})
	if err := VerifyProfile(leafDER, rootDER, testHostA, testNow); err == nil {
		t.Fatal("VerifyProfile(widened KU) succeeded, want refusal")
	} else if !errors.Is(err, ErrCredentialProfile) {
		t.Fatalf("VerifyProfile(widened KU) error = %v", err)
	}
}

// TestVerifyProfileRefusesMissingEKU builds an independently signed pair
// without serverAuth. Only the exact EKU-set gate refuses.
func TestVerifyProfileRefusesMissingEKU(t *testing.T) {
	leafDER, rootDER := selfSignedPairForTest(t, testHostA,
		x509.KeyUsageDigitalSignature,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		[]string{testHostA + ".host.ax.invalid"})
	if err := VerifyProfile(leafDER, rootDER, testHostA, testNow); err == nil {
		t.Fatal("VerifyProfile(missing serverAuth) succeeded, want refusal")
	} else if !errors.Is(err, ErrCredentialProfile) {
		t.Fatalf("VerifyProfile(missing serverAuth) error = %v", err)
	}
}

// TestVerifyProfileRefusesExtraSAN builds an independently signed pair with a
// second DNS name. Only the single-SAN gate refuses.
func TestVerifyProfileRefusesExtraSAN(t *testing.T) {
	leafDER, rootDER := selfSignedPairForTest(t, testHostA,
		x509.KeyUsageDigitalSignature,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		[]string{testHostA + ".host.ax.invalid", "extra.host.ax.invalid"})
	if err := VerifyProfile(leafDER, rootDER, testHostA, testNow); err == nil {
		t.Fatal("VerifyProfile(extra SAN) succeeded, want refusal")
	} else if !errors.Is(err, ErrCredentialProfile) {
		t.Fatalf("VerifyProfile(extra SAN) error = %v", err)
	}
}

// selfSignedPairForTest builds a profile-shaped pair with caller-chosen
// key usage, EKU set and SANs, signed by its own fresh root. It exists so
// field gates are tested against valid signatures rather than breakage.
func selfSignedPairForTest(t *testing.T, hostID string, usage x509.KeyUsage, eku []x509.ExtKeyUsage, dns []string) ([]byte, []byte) {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	notBefore := testNow.Add(-IssuanceSkew)
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(11),
		Subject:               pkix.Name{CommonName: rootCommonName(hostID)},
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
		t.Fatal(err)
	}
	rootTemplate.SubjectKeyId = rootSKI
	rootTemplate.AuthorityKeyId = rootSKI
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leafSKI, err := subjectKeyID(&leafKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(12),
		Subject:               pkix.Name{CommonName: leafCommonName(hostID)},
		NotBefore:             notBefore,
		NotAfter:              notBefore.Add(LeafLifetime),
		KeyUsage:              usage,
		ExtKeyUsage:           eku,
		BasicConstraintsValid: true,
		DNSNames:              dns,
		SubjectKeyId:          leafSKI,
		AuthorityKeyId:        rootSKI,
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, rootTemplate, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	return leafDER, rootDER
}
