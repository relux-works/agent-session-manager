package termbind

import (
	"encoding/json"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func mintPrior(t *testing.T) Binding {
	t.Helper()
	prior, err := ParseTerminalBinding(bindingDoc(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	return prior
}

// TestMintSuccessorBindingSuccessesAcrossGenerations pins the
// reboot-successor mint the restore entry composes: the successor
// inherits the prior except generation, supersedes, and identity,
// and re-admits through this same owner.
func TestMintSuccessorBindingSuccessesAcrossGenerations(t *testing.T) {
	prior := mintPrior(t)
	minted, raw, err := MintSuccessorBinding(prior, "generation-beta")
	if err != nil {
		t.Fatal(err)
	}
	if minted.BackendGeneration != "generation-beta" {
		t.Fatalf("generation = %s", minted.BackendGeneration)
	}
	if !minted.HasSupersedes || minted.SupersedesBindingID != prior.BindingID {
		t.Fatalf("supersedes = %+v, want %s", minted, prior.BindingID)
	}
	if minted.BindingID == "" || minted.BindingID == prior.BindingID {
		t.Fatalf("successor identity = %q, prior = %q", minted.BindingID, prior.BindingID)
	}
	for _, same := range [][2]string{
		{minted.SessionID, prior.SessionID},
		{minted.HostID, prior.HostID},
		{minted.HostIncarnationID, prior.HostIncarnationID},
		{minted.TerminalInstanceID, prior.TerminalInstanceID},
		{minted.TerminalBackendID, prior.TerminalBackendID},
		{minted.ImplementationVersion, prior.ImplementationVersion},
		{minted.ProtocolVersion, prior.ProtocolVersion},
		{minted.NativeReference, prior.NativeReference},
		{minted.CreatedAt, prior.CreatedAt},
	} {
		if same[0] != same[1] {
			t.Fatalf("successor drifts inherited member: %+v", minted)
		}
	}
	reparsed, err := ParseTerminalBinding(raw)
	if err != nil {
		t.Fatalf("minted successor rejected by its own owner: %v", err)
	}
	if reparsed.BindingID != minted.BindingID {
		t.Fatalf("reparsed identity %s != minted %s", reparsed.BindingID, minted.BindingID)
	}
}

// TestMintSuccessorBindingIsDeterministic pins crash convergence:
// the same prior and generation always mint the identical document,
// so a retry after a lost persist converges instead of forking.
func TestMintSuccessorBindingIsDeterministic(t *testing.T) {
	prior := mintPrior(t)
	minted, raw, err := MintSuccessorBinding(prior, "generation-beta")
	if err != nil {
		t.Fatal(err)
	}
	again, rawAgain, err := MintSuccessorBinding(prior, "generation-beta")
	if err != nil {
		t.Fatal(err)
	}
	if again.BindingID != minted.BindingID || string(rawAgain) != string(raw) {
		t.Fatal("successor mint is not deterministic")
	}
}

// TestMintSuccessorBindingRefusesSameGeneration pins the succession
// contradiction: minting over the prior's own generation refuses
// instead of linking a successor to itself.
func TestMintSuccessorBindingRefusesSameGeneration(t *testing.T) {
	prior := mintPrior(t)
	if _, _, err := MintSuccessorBinding(prior, prior.BackendGeneration); err == nil {
		t.Fatal("same-generation succession minted")
	} else {
		requireDetail(t, err, "binding successor generation")
	}
}

// TestMintSuccessorBindingRefusesMalformedGeneration pins the
// generation gate: the mint admits generations through the same
// digest gate as the parser, so a malformed generation refuses.
func TestMintSuccessorBindingRefusesMalformedGeneration(t *testing.T) {
	prior := mintPrior(t)
	if _, _, err := MintSuccessorBinding(prior, ""); err == nil {
		t.Fatal("empty-generation succession minted")
	}
}

// TestMintSuccessorBindingIdentityBinds pins the fresh identity: a
// successor rebound to another generation without re-minting fails
// this owner's admission.
func TestMintSuccessorBindingIdentityBinds(t *testing.T) {
	prior := mintPrior(t)
	_, raw, err := MintSuccessorBinding(prior, "generation-beta")
	if err != nil {
		t.Fatal(err)
	}
	var tampered map[string]any
	if err := json.Unmarshal(raw, &tampered); err != nil {
		t.Fatal(err)
	}
	tampered["backend_generation"] = "generation-gamma"
	forged, err := json.Marshal(tampered)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTerminalBinding(forged); err == nil {
		t.Fatal("rebound successor admitted without a fresh identity")
	} else {
		requireCode(t, err, terminalbackend.CodeMismatch)
	}
}
