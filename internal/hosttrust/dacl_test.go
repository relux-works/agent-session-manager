package hosttrust

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchExcludedFromReplication(t *testing.T) {
	credentialFile := hostChannelDir + "/credentials/abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789/private-key.pem"
	tests := []struct {
		name     string
		path     string
		excluded bool
	}{
		{"trust document", hostChannelDir + "/trust.json", true},
		{"joint marker", hostChannelDir + "/pending-commit.json", true},
		{"config binding", hostChannelDir + "/config-binding.json", true},
		{"credential directory", hostChannelDir + "/credentials/", true},
		{"private key", credentialFile, true},
		{"certificate", hostChannelDir + "/credentials/abcdef/certificate.pem", true},
		{"lock", hostChannelDir + "/lock", true},
		{"trust staging", hostChannelDir + "/.trust-stage-12345", true},
		{"custody staging", hostChannelDir + "/credentials/abcdef/.custody-stage-1", true},
		{"lock staging", hostChannelDir + "/lock-stage-9", true},
		{"unrelated state", "sessions/index.json", false},
		{"bare trust name elsewhere", "other/trust.json", false},
		{"empty", "", false},
		{"dot", ".", false},
		{"escape", "../outside", true},
		{"nested escape", "host-channel/../../outside", true},
		{"absolute", "/etc/host-channel/trust.json", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := MatchExcludedFromReplication(test.path); got != test.excluded {
				t.Fatalf("MatchExcludedFromReplication(%q) = %v, want %v", test.path, got, test.excluded)
			}
		})
	}
}

func TestExcludedConfigDirName(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		excluded bool
	}{
		{"versioned backup", "config.toml.bak.3.0.0", true},
		{"pre-rollback copy", "config.toml.pre-rollback.4.0.0", true},
		{"backup staging", ".ax-config-backup-123", true},
		{"migration staging", ".ax-config-migrate-1", true},
		{"restore staging", ".ax-config-restore-2", true},
		{"rollback staging", ".ax-config-rollback-3", true},
		{"target binding", "config.toml.ax-config-binding.json", true},
		{"target binding lock", "config.toml.ax-config-binding.lock", true},
		{"live configuration", "config.toml", false},
		{"unrelated", "notes.txt", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ExcludedConfigDirName(test.base); got != test.excluded {
				t.Fatalf("ExcludedConfigDirName(%q) = %v, want %v", test.base, got, test.excluded)
			}
		})
	}
}

func TestCustodyLayoutFullyExcluded(t *testing.T) {
	local, _, _, _, _, _ := twoHostFixture(t)
	root := filepath.Dir(filepath.Dir(local.TrustPath()))
	var checked int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		checked++
		if !MatchExcludedFromReplication(filepath.ToSlash(relative)) {
			t.Errorf("custody file %q escapes replication exclusion", relative)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no custody files walked")
	}
}

func TestEnrollmentExportNotExcluded(t *testing.T) {
	// Explicitly selected public enrollment material is the only operator
	// exchange outside replication: the matcher must not swallow it. The
	// material itself travels as explicit fields, never as a state-dir
	// path, so no state-relative path may claim to carry it.
	local, _, _, peerCredential, _, _ := twoHostFixture(t)
	snapshot, err := local.ReadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	material, err := ExportEnrollment(snapshot, peerCredential)
	if err != nil {
		t.Fatal(err)
	}
	if len(material.LeafDER) == 0 || len(material.RootDER) == 0 {
		t.Fatal("ExportEnrollment returned empty public bytes")
	}
}

func TestOwnerOnlyGrants(t *testing.T) {
	owner := "S-1-5-21-1-2-3-1001"
	other := "S-1-5-21-1-2-3-1002"
	system := "S-1-5-18"
	tests := []struct {
		name  string
		owner string
		aces  []allowedACE
		want  bool
	}{
		{"owner-only allow", owner, []allowedACE{{kind: aceAllow, sid: owner}}, true},
		{"other-principal allow", owner, []allowedACE{{kind: aceAllow, sid: owner}, {kind: aceAllow, sid: other}}, false},
		{"inherited system allow", owner, []allowedACE{{kind: aceAllow, sid: owner}, {kind: aceAllow, sid: system}}, false},
		{"world allow", owner, []allowedACE{{kind: aceAllow, sid: "S-1-1-0"}}, false},
		{"empty DACL", owner, nil, false},
		{"deny-only DACL", owner, []allowedACE{{kind: aceDeny, sid: other}}, false},
		{"deny other plus allow owner", owner, []allowedACE{{kind: aceDeny, sid: other}, {kind: aceAllow, sid: owner}}, true},
		{"unknown entry type", owner, []allowedACE{{kind: aceAllow, sid: owner}, {kind: aceOther, sid: owner}}, false},
		{"empty owner", "", []allowedACE{{kind: aceAllow, sid: owner}}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ownerOnlyGrants(test.owner, test.aces); got != test.want {
				t.Fatalf("ownerOnlyGrants(%s) = %v, want %v", test.name, got, test.want)
			}
		})
	}
}
