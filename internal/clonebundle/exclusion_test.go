package clonebundle

import (
	"testing"
)

// Row-21 exclusion tests (handoff from TASK-260909-2ez769): the
// bundle member allowlist rejects hosttrust.MatchExcludedFromReplication
// and hosttrust.ExcludedConfigDirName matches at construction, in
// both the raw manifest and the capture manifest, with one
// constructor negative per excluded class. Neighboring allowed paths
// still admit, proving the gate is the matcher and not a prefix ban.

func excludedReplicationKeys() []string {
	return []string{
		"host-channel/trust.json",
		"host-channel/config-binding.json",
		"host-channel/pending-commit.json",
		"host-channel/credentials/0198f4c8-8e50-7f66-8f70-1234567890ac/private-key.pem",
		"host-channel/credentials/leaf.pem",
		"host-channel/lock",
		".trust-stage-1",
		"state/.trust-stage-2",
		".custody-stage-3",
		"state/.custody-stage-4",
		"lock-stage-5",
		"../escape",
		"state/../../escape",
	}
}

func excludedConfigKeys() []string {
	return []string{
		"settings.bak.3",
		"state/settings.bak.3",
		"settings.pre-rollback.3",
		".ax-config-stage-1",
		"state/.ax-config-stage-2",
		"x.ax-config-binding.json",
	}
}

func TestRawManifestExcludesTrustMaterial(t *testing.T) {
	identity := fixtureNativeIdentity()
	identityDigest, err := IdentityDigest(identity, nil)
	if err != nil {
		t.Fatal(err)
	}
	build := func(key string) error {
		inputs := validEntryInputs(t)
		inputs[0].NativeItemKey = key
		// Keep the pair sorted so the exclusion gate, not the order
		// gate, fires: the fixture second key sorts after every
		// excluded key under test except none, so swap when needed.
		if len(inputs) == 2 && inputs[0].NativeItemKey > inputs[1].NativeItemKey {
			inputs[0], inputs[1] = inputs[1], inputs[0]
		}
		_, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil)
		return err
	}
	for _, key := range excludedReplicationKeys() {
		t.Run("replication/"+key, func(t *testing.T) {
			requireRefusal(t, build(key), "excluded from replication")
		})
	}
	for _, key := range excludedConfigKeys() {
		t.Run("config/"+key, func(t *testing.T) {
			requireRefusal(t, build(key), "trust-adjacent config material")
		})
	}
	// Neighboring allowed keys admit through the same entry.
	for _, key := range []string{
		"store/payload.bin",
		"host-channel/credentials-export.json",
		"state/settings.toml",
		"config-backup.toml",
	} {
		inputs := validEntryInputs(t)
		inputs[0].NativeItemKey = key
		if len(inputs) == 2 && inputs[0].NativeItemKey > inputs[1].NativeItemKey {
			inputs[0], inputs[1] = inputs[1], inputs[0]
		}
		if _, err := BuildRawObjectManifest(fixtureOperationID, []byte(fixtureTupleJSON()),
			"native-session-alpha", identityDigest.String(), fixtureDigest("test-capture-plan"), inputs, nil); err != nil {
			t.Fatalf("BuildRawObjectManifest(%q) error = %v", key, err)
		}
	}
	// Decode refuses excluded keys too (the gate fires before the
	// self-digest check, so no re-sealing is needed).
	valid := mustRawBytes(t)
	tampered := tamper(t, valid, func(o map[string]any) {
		entries := o["entries"].([]any)
		entries[0].(map[string]any)["native_item_key"] = "host-channel/trust.json"
	})
	_, err = DecodeRawObjectManifest(tampered)
	requireRefusal(t, err, "excluded from replication")
}

func TestCaptureManifestExcludesTrustMaterial(t *testing.T) {
	build := func(key string) error {
		input := validCaptureInput(t)
		reason := "credential_excluded"
		input.Items = []CaptureItemInput{
			{NativeItemKey: key, Class: "credential", Disposition: "excluded", ExclusionReason: &reason},
		}
		input.PlanKeys = []string{key}
		input.RawKeys = nil
		_, err := BuildCaptureManifest(input)
		return err
	}
	for _, key := range excludedReplicationKeys() {
		t.Run("replication/"+key, func(t *testing.T) {
			requireRefusal(t, build(key), "excluded from replication")
		})
	}
	for _, key := range excludedConfigKeys() {
		t.Run("config/"+key, func(t *testing.T) {
			requireRefusal(t, build(key), "trust-adjacent config material")
		})
	}
	valid := mustCaptureBytes(t)
	tampered := tamper(t, valid, func(o map[string]any) {
		items := o["items"].([]any)
		items[0].(map[string]any)["native_item_key"] = "host-channel/pending-commit.json"
	})
	_, err := DecodeCaptureManifest(tampered)
	requireRefusal(t, err, "excluded from replication")
}
