package fencing

import (
	"testing"

	"github.com/relux-works/agent-session-manager/internal/sessrepo"
)

// TestBindProjectsMintedToken proves the provider projection carries
// the exact winning triple: only a token minted by a passed gate
// binds to quiesce, capture, and materialize.
func TestBindProjectsMintedToken(t *testing.T) {
	token, err := AuthorizeMutation(fencePresented(), fenceObservation())
	if err != nil {
		t.Fatalf("AuthorizeMutation error = %v", err)
	}
	for _, operation := range []ProviderOperation{ProviderQuiesce, ProviderCapture, ProviderMaterialize} {
		bound, err := token.Bind(operation)
		if err != nil {
			t.Fatalf("Bind(%s) error = %v", operation, err)
		}
		if bound.SessionID != fenceSessionA || bound.Epoch != 2 || bound.LeaseID != fenceLeaseA || bound.Operation != operation {
			t.Fatalf("Bind(%s) = %+v, want the winning triple", operation, bound)
		}
	}
}

// TestBindRefusesUnknownOperation proves the provider operation enum
// is closed: resume, stop, and bogus names refuse invalid_arguments
// even with a live token.
func TestBindRefusesUnknownOperation(t *testing.T) {
	token, err := AuthorizeMutation(fencePresented(), fenceObservation())
	if err != nil {
		t.Fatalf("AuthorizeMutation error = %v", err)
	}
	for _, operation := range []ProviderOperation{"bogus", "resume", "stop"} {
		_, err := token.Bind(operation)
		mustRefuse(t, err, ErrInvalidArguments, "Bind("+string(operation)+")")
	}
}

// TestBindRefusesForgedToken proves the sealed capability cannot be
// assembled by hand: a zero token and a token whose seal carries
// every field but the constructor-only seal token refuse
// lease_conflict on every provider operation before any other check.
func TestBindRefusesForgedToken(t *testing.T) {
	forged := LeaseToken{seal: &leaseSeal{session: fenceSessionA, epoch: 2, leaseID: fenceLeaseA}}
	for _, operation := range []ProviderOperation{ProviderQuiesce, ProviderCapture, ProviderMaterialize} {
		_, err := LeaseToken{}.Bind(operation)
		mustRefuse(t, err, ErrLeaseConflict, "zero/Bind("+string(operation)+")")
		_, err = forged.Bind(operation)
		mustRefuse(t, err, ErrLeaseConflict, "forged/Bind("+string(operation)+")")
	}
}

// TestMintedTokenIsStableAcrossEntries proves every production entry
// mints the same projection for the same observation: the capability
// carries the authority, not the entry that admitted it.
func TestMintedTokenIsStableAcrossEntries(t *testing.T) {
	var first ProviderLease
	for name, entry := range fenceEntries() {
		token, err := entry.authorize(fencePresented(), fenceObservation())
		if err != nil {
			t.Fatalf("%s error = %v", name, err)
		}
		bound, err := token.Bind(ProviderCapture)
		if err != nil {
			t.Fatalf("%s Bind(capture) error = %v", name, err)
		}
		if first.SessionID == "" {
			first = bound
			continue
		}
		if bound != first {
			t.Fatalf("%s Bind(capture) = %+v, want %+v", name, bound, first)
		}
	}
}

// TestAuthorizeEndToEndOverRepository drives the full production path
// against a real store: observe, authorize, lose the lease to a rival
// succession, re-observe, and prove the old token is now stale while
// the rival token authorizes.
func TestAuthorizeEndToEndOverRepository(t *testing.T) {
	repository, _ := openFenceRepository(t)
	createFenceSession(t, repository)
	first := mustCreateFenceLease(t, repository, fenceSessionA, sessrepo.CreateLeaseInput{LeaseID: fenceLeaseOld, HolderHostID: fenceHostA, IssuedByHostID: fenceHostA, CreatedAt: fenceLeaseAt})
	input := ObserveInput{LocalHostID: fenceHostA, Verified: true, HasGrant: true, Grant: sessrepo.FencingGrant{SessionID: fenceSessionA, ValidatedAt: fenceValidatedAt}, Policy: fencePolicy, Now: fenceValidatedAt}
	observation, err := Observe(repository, fenceSessionA, input)
	if err != nil {
		t.Fatalf("Observe error = %v", err)
	}
	old := PresentedToken{SessionID: fenceSessionA, Epoch: 1, LeaseID: fenceLeaseOld}
	if _, err := AuthorizeMutation(old, observation); err != nil {
		t.Fatalf("AuthorizeMutation(epoch 1) error = %v", err)
	}
	mustSwapFenceLease(t, repository, fenceSessionA, sessrepo.LeaseExpectation{RecordID: first.RecordID}, sessrepo.SuccessorLeaseInput{
		CreateLeaseInput: sessrepo.CreateLeaseInput{LeaseID: fenceLeaseA, HolderHostID: fenceHostB, IssuedByHostID: fenceHostB, CreatedAt: fenceLeaseAt},
		Reason:           "force_takeover",
		CheckpointID:     fenceZeroDigest,
	})
	observation, err = Observe(repository, fenceSessionA, input)
	if err != nil {
		t.Fatalf("Observe after succession error = %v", err)
	}
	_, err = AuthorizeMutation(old, observation)
	mustRefuse(t, err, ErrNotOwner, "old token after succession is remote before it is stale")
	rival := PresentedToken{SessionID: fenceSessionA, Epoch: 2, LeaseID: fenceLeaseA}
	rivalInput := input
	rivalInput.LocalHostID = fenceHostB
	rivalObservation, err := Observe(repository, fenceSessionA, rivalInput)
	if err != nil {
		t.Fatalf("Observe(rival) error = %v", err)
	}
	token, err := AuthorizeActivation(rival, rivalObservation)
	mustMint(t, token, err, "rival token after succession")
}
