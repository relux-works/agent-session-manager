package sessadapter

import (
	"testing"
	"time"
)

func TestDiscoverBindsManifestToCandidate(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	digest := ManifestDigest(manifest).String()
	for _, role := range []Role{RoleSource, RoleTarget} {
		binding, err := Discover(role, manifest, digest, fixtureCandidate())
		if err != nil {
			t.Fatalf("Discover(%q): %v", role, err)
		}
		if binding.Role != role || binding.ProviderID != fixtureProviderID || binding.AdapterManifestDigest != digest {
			t.Fatalf("binding = %+v", binding)
		}
	}
}

func TestDiscoverRefusals(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	digest := ManifestDigest(manifest).String()
	candidate := fixtureCandidate()
	_, err = Discover("sideways", manifest, digest, candidate)
	requireRefusal(t, err, "invalid_config", "field", "role", "outside source|target")
	foreign := candidate
	foreign.ProviderID = "other-provider"
	_, err = Discover(RoleSource, manifest, digest, foreign)
	requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "trusted candidate")
	pathless := candidate
	pathless.ExecutablePath = ""
	_, err = Discover(RoleSource, manifest, digest, pathless)
	requireRefusal(t, err, "invalid_config", "field", "executable_path", "no executable path")
	ownerless := candidate
	ownerless.OwnerIdentity = ""
	_, err = Discover(RoleSource, manifest, digest, ownerless)
	requireRefusal(t, err, "invalid_config", "field", "owner_identity", "no owner identity")
	badExe := candidate
	badExe.ExecutableSHA256 = "not-a-digest"
	_, err = Discover(RoleSource, manifest, digest, badExe)
	requireRefusal(t, err, "invalid_config", "field", "executable_sha256", "not a digest")
	badDigest := digest
	_, err = Discover(RoleSource, manifest, "not-a-digest", candidate)
	requireRefusal(t, err, "invalid_config", "field", "session_adapter_manifest_digest", "not a digest")
	_ = badDigest
}

// TestBindingEqualityFlipsEveryFact breaks each of the eight
// identity facts alone: every flip refuses, and equal bindings
// with different observation times agree. verified_at timestamps
// the observation; it is not an identity fact.
func TestBindingEqualityFlipsEveryFact(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	sealed, err := Discover(RoleSource, manifest, ManifestDigest(manifest).String(), fixtureCandidate())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	fresh := sealed
	fresh.VerifiedAt = time.Now().UTC().Format(time.RFC3339)
	sealed.VerifiedAt = "2026-01-01T00:00:00Z"
	if err := CheckBindingEquality(sealed, fresh); err != nil {
		t.Fatalf("CheckBindingEquality with differing timestamps: %v", err)
	}
	flips := []func(*ExecutionBinding){
		func(binding *ExecutionBinding) { binding.Role = RoleTarget },
		func(binding *ExecutionBinding) { binding.ProviderID = "other-provider" },
		func(binding *ExecutionBinding) { binding.CandidateKind = CandidateExternal },
		func(binding *ExecutionBinding) { binding.ExecutablePath = "/other/path" },
		func(binding *ExecutionBinding) { binding.OwnerIdentity = "other-owner" },
		func(binding *ExecutionBinding) { binding.ExecutableSHA256 = fixtureDigest("other-exe") },
		func(binding *ExecutionBinding) {
			binding.ProviderManifestDigest = fixtureDigest("other-provider-manifest")
		},
		func(binding *ExecutionBinding) {
			binding.AdapterManifestDigest = fixtureDigest("other-adapter-manifest")
		},
	}
	if len(flips) != 8 {
		t.Fatalf("flips = %d, want one per identity fact", len(flips))
	}
	for index, flip := range flips {
		mutated := sealed
		flip(&mutated)
		if err := CheckBindingEquality(sealed, mutated); err == nil {
			t.Fatalf("CheckBindingEquality admitted flip %d", index)
		} else {
			requireRefusal(t, err, "integrity_failure", "subject", "binding", "freshly read trusted facts")
		}
	}
}

func TestCheckCallBinding(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	binding, err := Discover(RoleSource, manifest, ManifestDigest(manifest).String(), fixtureCandidate())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	body := fixtureRequestWithDigest(t, ManifestDigest(manifest).String())
	context, err := CheckRequestBody(OpDiscover, body)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	admitted, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	if err := CheckCallBinding(binding, RoleSource, context, admitted); err != nil {
		t.Fatalf("CheckCallBinding: %v", err)
	}
	if err := CheckCallBinding(binding, RoleTarget, context, admitted); err == nil {
		t.Fatal("admitted a target call under a source binding")
	} else {
		requireRefusal(t, err, "integrity_failure", "subject", "role", "binding role")
	}
	foreignProvider := context
	foreignProvider.ProviderID = "other-provider"
	if err := CheckCallBinding(binding, RoleSource, foreignProvider, admitted); err == nil {
		t.Fatal("admitted a foreign provider context")
	} else {
		requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "sealed binding")
	}
	foreignManifest := context
	foreignManifest.ManifestDigest = fixtureDigest("other-manifest")
	if err := CheckCallBinding(binding, RoleSource, foreignManifest, admitted); err == nil {
		t.Fatal("admitted a foreign manifest context")
	} else {
		requireRefusal(t, err, "integrity_failure", "subject", "session_adapter_manifest_digest", "sealed binding")
	}
	foreignExe := context
	foreignExe.ExecutableSHA256 = fixtureDigest("other-exe")
	if err := CheckCallBinding(binding, RoleSource, foreignExe, admitted); err == nil {
		t.Fatal("admitted a foreign executable context")
	} else {
		requireRefusal(t, err, "integrity_failure", "subject", "executable_sha256", "sealed binding")
	}
	otherTuple := admitted
	otherTuple.Version = "9.9.9"
	if err := CheckCallBinding(binding, RoleSource, context, otherTuple); err == nil {
		t.Fatal("admitted an unadmitted tuple")
	} else {
		requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "admitted tuple")
	}
}

// TestCheckCallBindingZeroFactsRefuse pins the fail-closed half
// of the call-binding gates: a caller that forgets a sealed fact
// gets a refusal, never an admission. Each zero binding fact
// with a non-empty call side refuses through its own arm, so a
// mutant that only checks when the expected side is non-empty
// (`!= binding && binding != ""`) admits and reddens here.
func TestCheckCallBindingZeroFactsRefuse(t *testing.T) {
	manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
	if err != nil {
		t.Fatalf("DecodeManifest: %v", err)
	}
	binding, err := Discover(RoleSource, manifest, ManifestDigest(manifest).String(), fixtureCandidate())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	body := fixtureRequestWithDigest(t, ManifestDigest(manifest).String())
	context, err := CheckRequestBody(OpDiscover, body)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	admitted, err := DecodeTuple([]byte(fixtureTupleJSON()))
	if err != nil {
		t.Fatalf("DecodeTuple: %v", err)
	}
	t.Run("zero binding role", func(t *testing.T) {
		roleless := binding
		roleless.Role = ""
		err := CheckCallBinding(roleless, RoleSource, context, admitted)
		requireRefusal(t, err, "integrity_failure", "subject", "role", "binding role")
	})
	t.Run("zero binding provider", func(t *testing.T) {
		providerless := binding
		providerless.ProviderID = ""
		err := CheckCallBinding(providerless, RoleSource, context, admitted)
		requireRefusal(t, err, "integrity_failure", "subject", "provider_id", "sealed binding")
	})
	t.Run("zero binding manifest digest", func(t *testing.T) {
		digestless := binding
		digestless.AdapterManifestDigest = ""
		err := CheckCallBinding(digestless, RoleSource, context, admitted)
		requireRefusal(t, err, "integrity_failure", "subject", "session_adapter_manifest_digest", "sealed binding")
	})
	t.Run("zero binding executable digest", func(t *testing.T) {
		exeless := binding
		exeless.ExecutableSHA256 = ""
		err := CheckCallBinding(exeless, RoleSource, context, admitted)
		requireRefusal(t, err, "integrity_failure", "subject", "executable_sha256", "sealed binding")
	})
	t.Run("zero admitted tuple", func(t *testing.T) {
		err := CheckCallBinding(binding, RoleSource, context, Tuple{})
		requireRefusal(t, err, "unsupported_environment_tuple", "environment", fixtureEnvironmentID, "admitted tuple")
	})
}
