package hosttrust

import (
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func twoHostFixture(t *testing.T) (local, peer *Store, localCredential, peerCredential string, localSnap, peerSnap Snapshot) {
	t.Helper()
	local = setupStoreForTest(t)
	peer = setupStoreForTest(t)
	var err error
	localCredential, err = local.Issue(testHostA, testNow)
	if err != nil {
		t.Fatal(err)
	}
	peerCredential = enrollPeerForTest(t, local, peer, testHostB, testNow)
	enrollPeerForTest(t, peer, local, testHostA, testNow)
	localSnap, err = local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	peerSnap, err = peer.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	return local, peer, localCredential, peerCredential, localSnap, peerSnap
}

func peerMaterialForTest(t *testing.T, peer *Store, credential string) (leafDER, rootDER []byte) {
	t.Helper()
	snapshot, err := peer.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := ExportEnrollment(snapshot, credential)
	if err != nil {
		t.Fatal(err)
	}
	return material.LeafDER, material.RootDER
}

func dispatchRequest(localSnap Snapshot, localCredential, peerCredential, hello string, leafDER, rootDER []byte, allowlist []string, now time.Time) AuthRequest {
	return AuthRequest{
		SnapshotGeneration:   localSnap.Generation,
		LocalHostID:          testHostA,
		LocalCredentialID:    localCredential,
		RemoteLeafDER:        leafDER,
		RemoteRootDER:        rootDER,
		ExpectedRemoteHostID: testHostB,
		Allowlisted:          allowlist,
		HelloHostID:          hello,
		Now:                  now,
	}
}

func TestAuthorizeDispatchHappyPath(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, []string{testHostA, testHostB}, testNow)
	if err := local.AuthorizeDispatch(localSnap, request); err != nil {
		t.Fatalf("AuthorizeDispatch error = %v", err)
	}
}

func TestAuthorizeDispatchRefusals(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	other := issueForTest(t, testHostC, testNow)
	stranger := setupStoreForTest(t)
	strangerCredential, err := stranger.Issue(testHostC, testNow)
	if err != nil {
		t.Fatal(err)
	}
	strangerSnap, err := stranger.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	strangerMaterial, err := ExportEnrollment(strangerSnap, strangerCredential)
	if err != nil {
		t.Fatal(err)
	}
	allowlist := []string{testHostA, testHostB}
	staleSnap := Snapshot{Generation: localSnap.Generation - 1, Trust: localSnap.Trust}
	futureSnap := Snapshot{Generation: localSnap.Generation + 1, Trust: localSnap.Trust}
	mismatched := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow)
	mismatched.SnapshotGeneration = localSnap.Generation - 1
	tests := []struct {
		name     string
		snapshot Snapshot
		request  AuthRequest
		want     error
	}{
		{"stale generation", staleSnap, dispatchRequest(staleSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow), ErrStaleGeneration},
		{"future generation", futureSnap, dispatchRequest(futureSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow), ErrStaleGeneration},
		{"refreshed number on old snapshot", localSnap, mismatched, ErrStaleGeneration},
		{"unknown local credential", localSnap, dispatchRequest(localSnap, peerCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow), ErrAuthorizationRefused},
		{"unknown remote bytes", localSnap, dispatchRequest(localSnap, localCredential, peerCredential, testHostB, other.LeafDER, other.RootDER, allowlist, testNow), ErrAuthorizationRefused},
		{"unenrolled remote", localSnap, dispatchRequest(localSnap, localCredential, peerCredential, testHostC, strangerMaterial.LeafDER, strangerMaterial.RootDER, allowlist, testNow), ErrAuthorizationRefused},
		{"leaf root mismatch", localSnap, dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, other.RootDER, allowlist, testNow), ErrAuthorizationRefused},
		{"not allowlisted", localSnap, dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, []string{testHostA}, testNow), ErrNotAllowlisted},
		{"hello mismatch", localSnap, dispatchRequest(localSnap, localCredential, peerCredential, testHostC, leafDER, rootDER, allowlist, testNow), ErrHostIdentityMismatch},
		{"wrong destination", localSnap, func() AuthRequest {
			request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow)
			request.ExpectedRemoteHostID = testHostC
			return request
		}(), ErrHostIdentityMismatch},
		{"expired credentials", localSnap, dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow.Add(91*24*time.Hour)), ErrAuthorizationRefused},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := local.AuthorizeDispatch(test.snapshot, test.request); err == nil {
				t.Fatalf("AuthorizeDispatch(%s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, test.want) {
				t.Fatalf("AuthorizeDispatch(%s) error = %v, want %v", test.name, err, test.want)
			}
		})
	}
}

func TestRevocationClosesAuthorization(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	allowlist := []string{testHostA, testHostB}
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow)
	if err := local.AuthorizeDispatch(localSnap, request); err != nil {
		t.Fatalf("pre-revocation AuthorizeDispatch error = %v", err)
	}
	if err := local.Revoke(peerCredential); err != nil {
		t.Fatalf("Revoke error = %v", err)
	}
	// The bound snapshot is stale: work prepared under the old generation
	// cannot gain new authority by refreshing a cached number.
	if err := local.AuthorizeDispatch(localSnap, request); err == nil {
		t.Fatal("post-revocation AuthorizeDispatch(stale snapshot) succeeded, want refusal")
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("post-revocation error = %v, want ErrStaleGeneration", err)
	}
	fresh, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	freshRequest := request
	freshRequest.SnapshotGeneration = fresh.Generation
	if err := local.AuthorizeDispatch(fresh, freshRequest); err == nil {
		t.Fatal("post-revocation AuthorizeDispatch(fresh) succeeded, want refusal")
	} else if !errors.Is(err, ErrAuthorizationRefused) {
		t.Fatalf("post-revocation fresh error = %v", err)
	}
}

func TestRetiringAdmissionEndsAtRetireAt(t *testing.T) {
	local, peer, localCredential, peerCredential, _, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	// The peer announced its rotation out of band; the operator catches up by
	// enrolling the new tuple elsewhere and bounding this credential here.
	retireAt := testNow.Add(2 * time.Hour)
	if err := local.MarkRetiring(peerCredential, retireAt, testNow); err != nil {
		t.Fatalf("MarkRetiring error = %v", err)
	}
	localSnap, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	allowlist := []string{testHostA, testHostB}
	admitted := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow)
	if err := local.AuthorizeDispatch(localSnap, admitted); err != nil {
		t.Fatalf("AuthorizeDispatch(retiring before retire_at) error = %v", err)
	}
	// At retire_at old admission ends even if the new route is unavailable.
	closed := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, retireAt)
	if err := local.AuthorizeDispatch(localSnap, closed); err == nil {
		t.Fatal("AuthorizeDispatch(at retire_at) succeeded, want refusal")
	}
}

func TestMarkRetiringBounds(t *testing.T) {
	local, _, _, peerCredential, _, _ := twoHostFixture(t)
	tests := []struct {
		name     string
		retireAt time.Time
	}{
		{"at now", testNow},
		{"before now", testNow.Add(-time.Hour)},
		{"past 24h bound", testNow.Add(25 * time.Hour)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := local.MarkRetiring(peerCredential, test.retireAt, testNow); err == nil {
				t.Fatalf("MarkRetiring(%s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, ErrRotationRefused) {
				t.Fatalf("MarkRetiring(%s) error = %v", test.name, err)
			}
		})
	}
	if err := local.MarkRetiring(peerCredential, testNow.Add(time.Hour), testNow); err != nil {
		t.Fatalf("MarkRetiring(valid) error = %v", err)
	}
	// Only an active entry may retire.
	if err := local.MarkRetiring(peerCredential, testNow.Add(2*time.Hour), testNow); err == nil {
		t.Fatal("second MarkRetiring succeeded, want refusal")
	}
	if err := local.MarkRetiring("sha256:0000000000000000000000000000000000000000000000000000000000000000", testNow.Add(time.Hour), testNow); err == nil {
		t.Fatal("MarkRetiring(unknown) succeeded, want refusal")
	}
}

func TestMarkRetiringCapsAtLeafExpiry(t *testing.T) {
	local := setupStoreForTest(t)
	issuer := setupStoreForTest(t)
	// A credential issued 89.5 days ago expires about 12 hours after now:
	// the 24-hour overlap bound alone would admit a retire_at past expiry.
	issuedAt := testNow.Add(-(89*24*time.Hour + 12*time.Hour))
	credential := enrollPeerForTest(t, local, issuer, testHostB, issuedAt)
	leafDER, _ := peerMaterialForTest(t, issuer, credential)
	leaf, err := x509.ParseCertificate(leafDER)
	if err != nil {
		t.Fatal(err)
	}
	retireAt := leaf.NotAfter.Add(30 * time.Minute)
	if !retireAt.After(testNow) || !retireAt.Before(testNow.Add(RotationBound)) {
		t.Fatalf("fixture retire_at %v does not isolate the expiry cap", retireAt)
	}
	if err := local.MarkRetiring(credential, retireAt, testNow); err == nil {
		t.Fatal("MarkRetiring(past old leaf expiry) succeeded, want refusal")
	} else if !errors.Is(err, ErrRotationRefused) {
		t.Fatalf("MarkRetiring(past old leaf expiry) error = %v, want ErrRotationRefused", err)
	}
}

func TestMutationAuthorizationSerializesWithRevocation(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	allowlist := []string{testHostA, testHostB}
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow)

	release := make(chan struct{})
	entered := make(chan struct{})
	boundaryDone := make(chan error, 1)
	go func() {
		boundaryDone <- local.WithMutationAuthorization(request, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	revoked := make(chan error, 1)
	go func() {
		revoked <- local.Revoke(peerCredential)
	}()
	// The revocation commit must wait for the boundary holding the exclusive
	// lock: give the revocation goroutine time to attempt the lock, then
	// require that it has not reported success while the boundary is held.
	// An immediate nonblocking select would miss the bug by winning the race
	// before the revocation goroutine is scheduled.
	var revokeErr error
	early := false
	select {
	case revokeErr = <-revoked:
		early = true
	case <-time.After(250 * time.Millisecond):
	}
	close(release)
	if err := <-boundaryDone; err != nil {
		t.Fatalf("WithMutationAuthorization error = %v", err)
	}
	if !early {
		revokeErr = <-revoked
	}
	if revokeErr != nil {
		t.Fatalf("Revoke after boundary error = %v", revokeErr)
	}
	if early {
		t.Fatal("Revoke reported success while the mutation boundary still held prior authorization")
	}
	fresh, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	freshRequest := request
	freshRequest.SnapshotGeneration = fresh.Generation
	if err := local.WithMutationAuthorization(freshRequest, func() error { return nil }); err == nil {
		t.Fatal("WithMutationAuthorization(post-revoke) succeeded, want refusal")
	}
}

func TestMutationAuthorizationSerializesWithSeparateStoreRevocation(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	allowlist := []string{testHostA, testHostB}
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, allowlist, testNow)
	// A second handle on the same state directory serializes exactly like
	// the same handle: the cross-process authorization lock, not a
	// process-local mutex, orders the boundary against the revocation.
	otherState := filepath.Dir(local.Root())
	other, err := Open(resolvedPathsForTest(t, filepath.Join(otherState, "config.toml"), otherState))
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	entered := make(chan struct{})
	boundaryDone := make(chan error, 1)
	go func() {
		boundaryDone <- local.WithMutationAuthorization(request, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	revoked := make(chan error, 1)
	go func() {
		revoked <- other.Revoke(peerCredential)
	}()
	var revokeErr error
	early := false
	select {
	case revokeErr = <-revoked:
		early = true
	case <-time.After(250 * time.Millisecond):
	}
	close(release)
	if err := <-boundaryDone; err != nil {
		t.Fatalf("WithMutationAuthorization error = %v", err)
	}
	if !early {
		revokeErr = <-revoked
	}
	if revokeErr != nil {
		t.Fatalf("Revoke after boundary error = %v", revokeErr)
	}
	if early {
		t.Fatal("separate-store Revoke reported success while the mutation boundary still held prior authorization")
	}
}

func TestWithMutationAuthorizationRefusesStaleGeneration(t *testing.T) {
	local, peer, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	leafDER, rootDER := peerMaterialForTest(t, peer, peerCredential)
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, leafDER, rootDER, []string{testHostA, testHostB}, testNow)
	// A later trust commit (here enrolling a third host) moves the current
	// generation past the prepared request. The original binding must
	// refuse: refreshing the cached number to authorize old prepared work is
	// forbidden, and the caller must replan through a fresh connection.
	enrollPeerForTest(t, local, peer, testHostC, testNow)
	called := false
	err := local.WithMutationAuthorization(request, func() error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("WithMutationAuthorization(stale binding) admitted: err=%v boundary=%v", err, called)
	} else if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("WithMutationAuthorization(stale) error = %v, want ErrStaleGeneration", err)
	}
	// A freshly bound request under the new generation still authorizes.
	fresh, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	freshRequest := request
	freshRequest.SnapshotGeneration = fresh.Generation
	if err := local.WithMutationAuthorization(freshRequest, func() error { return nil }); err != nil {
		t.Fatalf("WithMutationAuthorization(fresh) error = %v", err)
	}
}

func TestMutationAuthorizationRejectsNilBoundary(t *testing.T) {
	local, _, localCredential, peerCredential, localSnap, _ := twoHostFixture(t)
	snapshot, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := ExportEnrollment(snapshot, peerCredential)
	if err != nil {
		t.Fatal(err)
	}
	request := dispatchRequest(localSnap, localCredential, peerCredential, testHostB, material.LeafDER, material.RootDER, []string{testHostA, testHostB}, testNow)
	if err := local.WithMutationAuthorization(request, nil); err == nil {
		t.Fatal("WithMutationAuthorization(nil boundary) succeeded, want refusal")
	}
}

func TestExcludedFromReplication(t *testing.T) {
	excluded := ExcludedFromReplication()
	for _, required := range []string{hostChannelDir + "/trust.json", hostChannelDir + "/pending-commit.json", hostChannelDir + "/credentials/", hostChannelDir + "/lock"} {
		found := false
		for _, entry := range excluded {
			if entry == required {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ExcludedFromReplication missing %q", required)
		}
	}
}

func TestNoSecretReplication(t *testing.T) {
	local, _, _, _, _, _ := twoHostFixture(t)
	var privateKeySeen, publicOnly []string
	root := filepath.Join(filepath.Dir(local.TrustPath()))
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.Contains(string(contents), "PRIVATE KEY") {
			privateKeySeen = append(privateKeySeen, relative)
		} else {
			publicOnly = append(publicOnly, relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// The two-host fixture holds exactly two custody directories in the local
	// store: the first self issuance and the second one made while modeling
	// the peer's enrollment of this host. Both must sit in the custody layout
	// and nowhere else.
	if len(privateKeySeen) != 2 {
		t.Fatalf("private key material in %d files, want exactly the two custody files: %v", len(privateKeySeen), privateKeySeen)
	}
	for _, path := range privateKeySeen {
		if !strings.Contains(filepath.ToSlash(path), "credentials/") || !strings.HasSuffix(path, "private-key.pem") {
			t.Fatalf("private key outside custody layout: %v", privateKeySeen)
		}
	}
	trustBytes, err := os.ReadFile(local.TrustPath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(trustBytes), "PRIVATE KEY") {
		t.Fatal("trust.json carries private key material")
	}
	if len(publicOnly) == 0 {
		t.Fatal("no public custody files found")
	}
}
