package hosttrust

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

func TestDecodeTrustRefusals(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	valid := TrustStore{Generation: 1, Entries: []CredentialEntry{entryForTest(t, issued, EntryActive, testNow, nil)}}
	validDocument := string(encodeForTest(t, valid))
	leafB64 := base64.RawURLEncoding.EncodeToString(issued.LeafDER)

	duplicateKeyDocument := strings.Replace(validDocument, `"generation":1`, `"generation":1,"generation":1`, 1)
	nestedDuplicateDocument := strings.Replace(validDocument, `"state":"active"`, `"state":"active","state":"active"`, 1)
	oversizedLeaf := base64.RawURLEncoding.EncodeToString(make([]byte, MaxDERBytes+1))
	oversizedDocument := strings.Replace(validDocument, `"leaf_der":"`+leafB64+`"`, `"leaf_der":"`+oversizedLeaf+`"`, 1)
	malformedDocument := strings.Replace(validDocument, `"leaf_der":"`+leafB64+`"`, `"leaf_der":"AAAA"`, 1)
	nullEntriesDocument := `{"schema":"urn:ax:schema:host-trust-store","schema_version":"1.0.0","generation":1,"entries":null}`
	missingEntriesDocument := `{"schema":"urn:ax:schema:host-trust-store","schema_version":"1.0.0","generation":1}`
	noRetireDocument := strings.Replace(validDocument, `"retire_at":null`, `"retire_at":"2026-09-10T13:00:00.000Z"`, 1)

	// A retiring entry whose retire_at equals enrolled_at violates the
	// strict later-than rule. EncodeTrust cannot produce it (its round trip
	// refuses), so the fixture rewrites a valid retiring document.
	retire := testNow.Add(time.Hour)
	retiring := TrustStore{Generation: 1, Entries: []CredentialEntry{entryForTest(t, issued, EntryRetiring, testNow, &retire)}}
	equalWindowDocument := strings.Replace(string(encodeForTest(t, retiring)),
		`"retire_at":"`+testTimestamp(retire).String()+`"`, `"retire_at":"`+testTimestamp(testNow).String()+`"`, 1)

	// Two identical entries share one credential_id.
	entryJSON := validDocument[strings.Index(validDocument, `"entries":[`)+len(`"entries":[`):]
	entryJSON = entryJSON[:len(entryJSON)-2]
	duplicateEntryDocument := strings.Replace(validDocument, `"entries":[`+entryJSON+`]`, `"entries":[`+entryJSON+`,`+entryJSON+`]`, 1)

	// Two valid entries in descending credential_id order: every gate but
	// the ordering check admits the pair, so only sorting refuses.
	other := issueForTest(t, testHostB, testNow)
	orderedA := entryForTest(t, issued, EntryActive, testNow, nil)
	orderedB := entryForTest(t, other, EntryActive, testNow, nil)
	descending, ascending := orderedA, orderedB
	if descending.CredentialID.String() < ascending.CredentialID.String() {
		descending, ascending = ascending, descending
	}
	if descending.CredentialID.String() == ascending.CredentialID.String() {
		t.Fatal("descending fixture entries collide")
	}
	unsortedDocument := string(joinUnsortedForTest(t, descending, ascending))

	// Digest bindings: the entry identifiers must reproduce the enclosed
	// bytes. Each rewrite below swaps one identifier for another valid
	// identifier from the same document, so only the binding gate refuses.
	credentialID := scalar.SHA256Digest(issued.LeafDER).String()
	spkiID := scalar.SHA256Digest(mustSPKI(issued.LeafDER)).String()
	rootID := scalar.SHA256Digest(issued.RootDER).String()
	quote := func(value string) string { return `"sha256:` + strings.TrimPrefix(value, "sha256:") + `"` }
	credentialNotLeafDocument := strings.Replace(validDocument, `"credential_id":`+quote(credentialID), `"credential_id":`+quote(rootID), 1)
	spkiNotLeafDocument := strings.Replace(validDocument, `"spki_id":`+quote(spkiID), `"spki_id":`+quote(credentialID), 1)
	rootNotRootDocument := strings.Replace(validDocument, `"root_id":`+quote(rootID), `"root_id":`+quote(credentialID), 1)
	if credentialNotLeafDocument == validDocument || spkiNotLeafDocument == validDocument || rootNotRootDocument == validDocument {
		t.Fatal("digest binding fixtures did not rewrite")
	}

	// Two entries sharing one root DER are invalid even with consistent
	// digests: duplicate root IDs never map twice, for any host. The pair
	// bypasses EncodeTrust, whose defence-in-depth round trip refuses it.
	sharedRoot := entryForTest(t, other, EntryActive, testNow, nil)
	sharedRoot.RootDER = append([]byte(nil), issued.RootDER...)
	sharedRoot.RootID = scalar.SHA256Digest(issued.RootDER)
	duplicateRootDocument := string(joinEntriesForTest(t,
		entryForTest(t, issued, EntryActive, testNow, nil),
		sharedRoot,
	))

	// Two entries sharing one leaf public key under different roots are
	// invalid: the same key never enrolls twice, including renewal.
	sameKeyA, sameRootA, sameKeyB, sameRootB := craftSameKeyLeaves(t, testNow)
	hostA, err := scalar.ParseUUIDv7(testHostA)
	if err != nil {
		t.Fatal(err)
	}
	hostB, err := scalar.ParseUUIDv7(testHostB)
	if err != nil {
		t.Fatal(err)
	}
	sameKeyDocument := string(joinEntriesForTest(t,
		digestBoundEntry(t, hostA, sameKeyA, sameRootA),
		digestBoundEntry(t, hostB, sameKeyB, sameRootB),
	))

	tests := []struct {
		name     string
		document string
	}{
		{"empty", ""},
		{"not json", "not json"},
		{"array", "[]"},
		{"trailing garbage", validDocument + " {}"},
		{"duplicate root key", duplicateKeyDocument},
		{"duplicate nested key", nestedDuplicateDocument},
		{"unknown root field", strings.Replace(validDocument, `"generation":1`, `"generation":1,"comment":"hi"`, 1)},
		{"missing schema", strings.Replace(validDocument, `"schema":"urn:ax:schema:host-trust-store",`, ``, 1)},
		{"wrong schema", strings.Replace(validDocument, `urn:ax:schema:host-trust-store`, `urn:ax:schema:config`, 1)},
		{"downgraded version", strings.Replace(validDocument, `"schema_version":"1.0.0"`, `"schema_version":"0.9.0"`, 1)},
		{"missing version", strings.Replace(validDocument, `"schema_version":"1.0.0",`, ``, 1)},
		{"generation zero", strings.Replace(validDocument, `"generation":1`, `"generation":0`, 1)},
		{"generation uint53 ceiling", strings.Replace(validDocument, `"generation":1`, `"generation":9007199254740992`, 1)},
		{"generation fraction", strings.Replace(validDocument, `"generation":1`, `"generation":1.5`, 1)},
		{"generation string value", strings.Replace(validDocument, `"generation":1`, `"generation":"1"`, 1)},
		{"generation negative", strings.Replace(validDocument, `"generation":1`, `"generation":-1`, 1)},
		{"entries null", nullEntriesDocument},
		{"missing entries", missingEntriesDocument},
		{"unknown entry field", strings.Replace(validDocument, `"state":"active"`, `"state":"active","note":"hi"`, 1)},
		{"missing entry field", strings.Replace(validDocument, `,"state":"active"`, ``, 1)},
		{"bad state", strings.Replace(validDocument, `"state":"active"`, `"state":"pending"`, 1)},
		{"active with retire_at", noRetireDocument},
		{"retire_at not after enrolled_at", equalWindowDocument},
		{"bad host id", strings.Replace(validDocument, testHostA, "not-a-uuid", 1)},
		{"bad digest", strings.Replace(validDocument, `"credential_id":"sha256:`, `"credential_id":"md5:`, 1)},
		{"bad timestamp", strings.Replace(validDocument, `2026-09-10T12:00:00.000Z`, `2026-09-10 12:00:00`, 1)},
		{"padded base64", strings.Replace(validDocument, `"leaf_der":"`, `"leaf_der":" `, 1)},
		{"oversized der", oversizedDocument},
		{"malformed der", malformedDocument},
		{"duplicate credential_id", duplicateEntryDocument},
		{"unsorted entries", unsortedDocument},
		{"credential_id not leaf digest", credentialNotLeafDocument},
		{"spki_id not leaf spki", spkiNotLeafDocument},
		{"root_id not root digest", rootNotRootDocument},
		{"duplicate root across entries", duplicateRootDocument},
		{"duplicate leaf key across entries", sameKeyDocument},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeTrust([]byte(test.document)); err == nil {
				t.Fatalf("DecodeTrust(%s) succeeded, want refusal", test.name)
			} else {
				var refusal *TrustError
				if !errors.As(err, &refusal) {
					t.Fatalf("DecodeTrust(%s) error type = %T, want *TrustError", test.name, err)
				}
			}
		})
	}
}

func TestDecodeTrustRetiringWithoutRetireAt(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	retire := testNow.Add(time.Hour)
	store := TrustStore{Generation: 1, Entries: []CredentialEntry{entryForTest(t, issued, EntryRetiring, testNow, &retire)}}
	document := string(encodeForTest(t, store))
	stamp := testTimestamp(retire).String()
	nulled := strings.Replace(document, `"retire_at":"`+stamp+`"`, `"retire_at":null`, 1)
	if nulled == document {
		t.Fatal("retire_at fixture did not match")
	}
	if _, err := DecodeTrust([]byte(nulled)); err == nil {
		t.Fatal("DecodeTrust(retiring with null retire_at) succeeded, want refusal")
	}
}

func TestEncodeTrustRefusals(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	active := entryForTest(t, issued, EntryActive, testNow, nil)
	retire := testNow.Add(time.Hour)
	retiring := entryForTest(t, issued, EntryRetiring, testNow, &retire)
	tests := []struct {
		name  string
		store TrustStore
	}{
		{"generation zero", TrustStore{Generation: 0, Entries: []CredentialEntry{active}}},
		{"generation exhausted", TrustStore{Generation: MaxGeneration + 1, Entries: []CredentialEntry{active}}},
		{"duplicate entries", TrustStore{Generation: 1, Entries: []CredentialEntry{active, active}}},
		{"retiring without retire_at", TrustStore{Generation: 1, Entries: []CredentialEntry{entryForTest(t, issued, EntryRetiring, testNow, nil)}}},
		{"revoked with retire_at", TrustStore{Generation: 1, Entries: []CredentialEntry{retiring}}},
	}
	for index := range tests {
		// The revoked case reuses the retiring entry value with revoked state.
		if tests[index].name == "revoked with retire_at" {
			tests[index].store.Entries[0].State = EntryRevoked
		}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := EncodeTrust(test.store); err == nil {
				t.Fatalf("EncodeTrust(%s) succeeded, want refusal", test.name)
			}
		})
	}
}

func TestEncodeTrustGenerationCeiling(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	store := TrustStore{Generation: MaxGeneration, Entries: []CredentialEntry{entryForTest(t, issued, EntryActive, testNow, nil)}}
	document, err := EncodeTrust(store)
	if err != nil {
		t.Fatalf("EncodeTrust(max generation) error = %v", err)
	}
	decoded, err := DecodeTrust(document)
	if err != nil {
		t.Fatalf("DecodeTrust(max generation) error = %v", err)
	}
	if decoded.Generation != MaxGeneration {
		t.Fatalf("DecodeTrust generation = %d, want %d", decoded.Generation, uint64(MaxGeneration))
	}
}

func TestTrustRedaction(t *testing.T) {
	issued := issueForTest(t, testHostA, testNow)
	store := TrustStore{Generation: 1, Entries: []CredentialEntry{entryForTest(t, issued, EntryActive, testNow, nil)}}
	snapshot := Snapshot{Generation: 1, Trust: store}
	material := EnrollmentMaterial{HostID: testHostA, CredentialID: "sha256:00", LeafDER: issued.LeafDER, RootDER: issued.RootDER}
	request := AuthRequest{LocalHostID: testHostA, LocalCredentialID: "sha256:00", RemoteLeafDER: issued.LeafDER, Allowlisted: []string{testHostB}, HelloHostID: testHostB}
	formatted := []string{
		store.String(), store.GoString(), sprintfForTest(snapshot), sprintfForTest(store.Entries[0]),
		issued.String(), issued.GoString(), sprintfForTest(material), sprintfForTest(request), sprintfForTest(snapshot),
	}
	for _, rendered := range formatted {
		for _, secret := range []string{testHostA, testHostB, "sha256:", "PRIVATE", "CERTIFICATE"} {
			if strings.Contains(rendered, secret) {
				t.Fatalf("redacted rendering %q exposes %q", rendered, secret)
			}
		}
	}
	refusal := trustError(TrustError{Operation: "decode document", Err: ErrTrustDecode})
	if strings.Contains(refusal.Error(), testHostA) || !errors.Is(refusal, ErrTrustDecode) {
		t.Fatalf("TrustError rendering leaks or breaks unwrapping: %q", refusal.Error())
	}
}

// joinEntriesForTest renders a trust document from entries without the
// EncodeTrust round trip, so mapping-ambiguous pairs reach DecodeTrust.
// Entries render in encoder order: an unsorted pair would refuse on ordering
// before uniqueness is examined.
func joinEntriesForTest(t *testing.T, entries ...CredentialEntry) []byte {
	t.Helper()
	ordered := append([]CredentialEntry(nil), entries...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].CredentialID.String() < ordered[j].CredentialID.String()
	})
	return joinUnsortedForTest(t, ordered...)
}

// joinUnsortedForTest renders entries in the given order, so a descending
// pair reaches the ordering check.
func joinUnsortedForTest(t *testing.T, entries ...CredentialEntry) []byte {
	t.Helper()
	document := `{"schema":"urn:ax:schema:host-trust-store","schema_version":"1.0.0","generation":1,"entries":[`
	for index, entry := range entries {
		encoded, err := encodeEntry(entry)
		if err != nil {
			t.Fatalf("encodeEntry error = %v", err)
		}
		if index > 0 {
			document += ","
		}
		document += string(encoded)
	}
	return []byte(document + "]}")
}

// digestBoundEntry builds a digest-consistent entry from raw certificate
// bytes without profile assumptions: the closed reader checks bindings and
// mapping uniqueness before any profile question arises.
func digestBoundEntry(t *testing.T, hostID scalar.UUIDv7, leafDER, rootDER []byte) CredentialEntry {
	t.Helper()
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatalf("ParseCertificate(leaf) error = %v", err)
	}
	return CredentialEntry{
		HostID:       hostID,
		CredentialID: scalar.SHA256Digest(leafDER),
		SPKIID:       scalar.SHA256Digest(leaf.RawSubjectPublicKeyInfo),
		RootID:       scalar.SHA256Digest(rootDER),
		LeafDER:      append([]byte(nil), leafDER...),
		RootDER:      append([]byte(nil), rootDER...),
		State:        EntryActive,
		EnrolledAt:   testTimestamp(testNow),
	}
}

// craftSameKeyLeaves issues two structurally parseable leaves sharing one
// P-256 public key under two distinct self-signed roots. The closed reader
// must refuse the pair on key reuse alone; no profile claim is made about
// these certificates.
func craftSameKeyLeaves(t *testing.T, now time.Time) (leafA, rootA, leafB, rootB []byte) {
	t.Helper()
	rootKeyA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	rootKeyB, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	notBefore := now.UTC().Add(-time.Hour)
	rootTemplate := func(commonName string, serial int64) *x509.Certificate {
		return &x509.Certificate{
			SerialNumber:          big.NewInt(serial),
			Subject:               pkix.Name{CommonName: commonName},
			NotBefore:             notBefore,
			NotAfter:              notBefore.Add(24 * time.Hour),
			KeyUsage:              x509.KeyUsageCertSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
			MaxPathLen:            0,
			MaxPathLenZero:        true,
		}
	}
	rootA, err = x509.CreateCertificate(rand.Reader, rootTemplate("test root A", 101), rootTemplate("test root A", 101), &rootKeyA.PublicKey, rootKeyA)
	if err != nil {
		t.Fatal(err)
	}
	rootBTemplate := rootTemplate("test root B", 102)
	rootB, err = x509.CreateCertificate(rand.Reader, rootBTemplate, rootBTemplate, &rootKeyB.PublicKey, rootKeyB)
	if err != nil {
		t.Fatal(err)
	}
	parsedA, err := x509.ParseCertificate(rootA)
	if err != nil {
		t.Fatal(err)
	}
	parsedB, err := x509.ParseCertificate(rootB)
	if err != nil {
		t.Fatal(err)
	}
	leafA, err = x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(201),
		Subject:      pkix.Name{CommonName: "test leaf A"},
		NotBefore:    notBefore,
		NotAfter:     notBefore.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, parsedA, &leafKey.PublicKey, rootKeyA)
	if err != nil {
		t.Fatal(err)
	}
	leafB, err = x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(202),
		Subject:      pkix.Name{CommonName: "test leaf B"},
		NotBefore:    notBefore,
		NotAfter:     notBefore.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, parsedB, &leafKey.PublicKey, rootKeyB)
	if err != nil {
		t.Fatal(err)
	}
	return leafA, rootA, leafB, rootB
}

func sprintfForTest(value fmt.Formatter) string {
	// %v and %s dispatch to Format, so the static rendering is exercised,
	// not just String.
	return fmt.Sprintf("%v|%s|%+v", value, value, value)
}
