package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

type v4Fixture struct {
	directory string
	filename  string
	stateDir  string
	inputs    Inputs
	store     *hosttrust.Store
	self      string
	now       time.Time
}

func setupV4Fixture(t *testing.T, peers ...string) *v4Fixture {
	t.Helper()
	directory := t.TempDir()
	stateDir := filepath.Join(directory, "state")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configuration := validCurrentConfiguration()
	configuration.Mesh.Peers = nil
	for _, peer := range peers {
		configuration.Mesh.Peers = append(configuration.Mesh.Peers, Peer{
			HostID: peer, Name: "peer-" + peer[len(peer)-2:], Endpoint: "peer.example",
			Platform: scalar.PlatformLinux, SSHArgs: []string{"-o", "BatchMode=yes"},
		})
	}
	document, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatalf("EncodeCurrent(fixture) error = %v", err)
	}
	filename := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(filename, document, 0o600); err != nil {
		t.Fatal(err)
	}
	inputs := Inputs{
		Platform: scalar.PlatformMacOS, HomeDir: directory, TempDir: directory, WorkingDir: directory,
		LookupEnv: func(name string) (string, bool) {
			switch name {
			case "AX_CONFIG":
				return filename, true
			case "AX_STATE_DIR":
				return stateDir, true
			}
			return "", false
		},
		Stat: os.Stat, ReadFile: os.ReadFile,
	}
	paths, err := resolveLocalPaths(inputs, nil)
	if err != nil {
		t.Fatalf("resolveLocalPaths(fixture) error = %v", err)
	}
	store, err := hosttrust.Open(paths)
	if err != nil {
		t.Fatalf("hosttrust.Open error = %v", err)
	}
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatalf("EnsureConfigBinding error = %v", err)
	}
	if _, err := store.Initialize(); err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	self, err := store.Issue(testHostID, now)
	if err != nil {
		t.Fatalf("Issue error = %v", err)
	}
	for _, peer := range peers {
		issuerState := t.TempDir()
		issuer, err := hosttrust.Open(resolvedPathsForStoreTest(t, filepath.Join(issuerState, "config.toml"), issuerState))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := issuer.Initialize(); err != nil {
			t.Fatal(err)
		}
		credential, err := issuer.Issue(peer, now)
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := issuer.ReadSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		material, err := hosttrust.ExportEnrollment(snapshot, credential)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := store.Enroll(hosttrust.EnrollInput{HostID: material.HostID, LeafDER: material.LeafDER, RootDER: material.RootDER, Authorized: material.Fingerprints, EnrolledAt: now}); err != nil {
			t.Fatalf("Enroll(%s) error = %v", peer, err)
		}
	}
	return &v4Fixture{directory: directory, filename: filename, stateDir: stateDir, inputs: inputs, store: store, self: self, now: now}
}

func resolvedPathsForStoreTest(t *testing.T, configPath, stateDir string) localstore.ResolvedPaths {
	t.Helper()
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: scalar.PlatformLinux,
		Flags: map[string]string{
			"--config":      configPath,
			"--data-dir":    filepath.Join(stateDir, "data"),
			"--state-dir":   stateDir,
			"--cache-dir":   filepath.Join(stateDir, "cache"),
			"--runtime-dir": filepath.Join(stateDir, "runtime"),
		},
	})
	if err != nil {
		t.Fatalf("ResolvePaths(%s,%s) error = %v", configPath, stateDir, err)
	}
	return paths
}

func resolveLocalPathsForFixture(t *testing.T, fixture *v4Fixture) localstore.ResolvedPaths {
	t.Helper()
	paths, err := resolveLocalPaths(fixture.inputs, nil)
	if err != nil {
		t.Fatalf("resolveLocalPaths(fixture) error = %v", err)
	}
	return paths
}

func (fixture *v4Fixture) trust(t *testing.T) hosttrust.Snapshot {
	t.Helper()
	snapshot, err := fixture.store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot error = %v", err)
	}
	return snapshot
}

func TestPreviewV4HappyPath(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4 error = %v", err)
	}
	if preview.SourceVersion != CurrentVersion || preview.CredentialID != fixture.self {
		t.Fatalf("preview identity = %+v", preview.SourceVersion)
	}
	if len(preview.RetainedPeers) != 1 || preview.RetainedPeers[0] != testPeerID {
		t.Fatalf("retained peers = %v", preview.RetainedPeers)
	}
	if len(preview.DroppedPeers) != 0 || len(preview.BlockingPeers) != 0 {
		t.Fatalf("dropped = %v blocking = %v, want empty", preview.DroppedPeers, preview.BlockingPeers)
	}
	loaded, err := Decode(preview.Replacement, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatalf("Decode(preview) error = %v", err)
	}
	if loaded.SourceVersion != Version4 || loaded.Value.Mesh.Transport != TransportSSHTLS13 {
		t.Fatalf("preview decodes as %q/%q, want 4.0.0/ssh_tls13", loaded.SourceVersion, loaded.Value.Mesh.Transport)
	}
	if loaded.Value.Mesh.HostChannel == nil || loaded.Value.Mesh.HostChannel.CredentialID != fixture.self {
		t.Fatalf("preview host_channel = %+v", loaded.Value.Mesh.HostChannel)
	}
	if len(loaded.Value.Mesh.Peers) != 1 || loaded.Value.HostName != "fixture-host" {
		t.Fatal("preview dropped retained Config-3 fields")
	}
	current, err := os.ReadFile(fixture.filename)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(preview.SourceDocument, current) {
		t.Fatal("preview source does not pin the current bytes")
	}
}

func TestPreviewV4AbsentSource(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	if err := os.Remove(fixture.filename); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now); err == nil {
		t.Fatal("PreviewV4(absent source) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationSourceAbsent) {
		t.Fatalf("PreviewV4(absent) error = %v", err)
	}
}

func TestPreviewV4CredentialWindow(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	expired := fixture.now.Add(91 * 24 * time.Hour)
	if _, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, expired); err == nil {
		t.Fatal("PreviewV4(expired credential window) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4Credential) {
		t.Fatalf("PreviewV4(expired window) error = %v", err)
	}
}

func TestPreviewV4RequiresV3Source(t *testing.T) {
	fixture := setupV4Fixture(t)
	v1 := append(minimalValidConfigVersion(scalar.PlatformMacOS, Version1), []byte("\n[mesh]\ntransport = \"ssh\"\n")...)
	if err := os.WriteFile(fixture.filename, v1, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now); err == nil {
		t.Fatal("PreviewV4(v1 source) succeeded, want refusal: older inputs first use the existing explicit migrations")
	} else if !errors.Is(err, ErrMigrationV4Source) {
		t.Fatalf("PreviewV4(v1 source) error = %v", err)
	}
}

func TestPreviewV4CredentialGates(t *testing.T) {
	fixture := setupV4Fixture(t, testPeerID)
	snapshot := fixture.trust(t)
	peerCredential := ""
	for _, entry := range snapshot.Trust.Entries {
		if entry.HostID.String() == testPeerID {
			peerCredential = entry.CredentialID.String()
		}
	}
	if peerCredential == "" {
		t.Fatal("peer credential missing from trust")
	}
	unknown := "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	tests := []struct {
		name       string
		credential string
	}{
		{"unknown credential", unknown},
		{"malformed digest", "not-a-digest"},
		{"peer credential for local mesh", peerCredential},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PreviewV4(fixture.inputs, nil, snapshot, test.credential, nil, fixture.now); err == nil {
				t.Fatalf("PreviewV4(%s) succeeded, want refusal", test.name)
			} else if !errors.Is(err, ErrMigrationV4Credential) {
				t.Fatalf("PreviewV4(%s) error = %v", test.name, err)
			}
		})
	}
	if err := fixture.store.Revoke(fixture.self); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now); err == nil {
		t.Fatal("PreviewV4(revoked credential) succeeded, want refusal")
	} else if !errors.Is(err, ErrMigrationV4Credential) {
		t.Fatalf("PreviewV4(revoked) error = %v", err)
	}
}

func TestPreviewV4PeerEnrollment(t *testing.T) {
	// The fixture enrolls testPeerID only; the config below additionally
	// lists testPeerID2 without enrollment.
	fixture := setupV4Fixture(t, testPeerID)
	configuration := validCurrentConfiguration()
	configuration.Mesh.Peers = append(configuration.Mesh.Peers, Peer{
		HostID: testPeerID2, Name: "peer2", Endpoint: "peer2.example",
		Platform: scalar.PlatformLinux, SSHArgs: []string{"-o", "BatchMode=yes"},
	})
	document, err := EncodeCurrent(configuration, DecodeContext{RuntimePlatform: scalar.PlatformMacOS})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture.filename, document, 0o600); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, nil, fixture.now)
	if err == nil {
		t.Fatal("PreviewV4(unenrolled peer) succeeded, want refusal: a missing peer enrollment blocks activation")
	} else if !errors.Is(err, ErrMigrationV4PeerEnrollment) {
		t.Fatalf("PreviewV4(unenrolled) error = %v", err)
	}
	if len(preview.BlockingPeers) != 1 || preview.BlockingPeers[0] != testPeerID2 {
		t.Fatalf("blocking peers = %v, want [%s]", preview.BlockingPeers, testPeerID2)
	}
	// An operator may explicitly remove a peer from the preview.
	dropped, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, []string{testPeerID2}, fixture.now)
	if err != nil {
		t.Fatalf("PreviewV4(explicit drop) error = %v", err)
	}
	if len(dropped.DroppedPeers) != 1 || dropped.DroppedPeers[0] != testPeerID2 {
		t.Fatalf("dropped peers = %v", dropped.DroppedPeers)
	}
	if len(dropped.RetainedPeers) != 1 || dropped.RetainedPeers[0] != testPeerID {
		t.Fatalf("retained peers = %v", dropped.RetainedPeers)
	}
	// Unknown, duplicate and name-spelled removals refuse: no silent drop.
	for _, drop := range [][]string{{"0198f4c8-0000-7e55-8e6f-1234567890ab"}, {testPeerID2, testPeerID2}, {"peer2"}} {
		if _, err := PreviewV4(fixture.inputs, nil, fixture.trust(t), fixture.self, drop, fixture.now); err == nil {
			t.Fatalf("PreviewV4(drop %v) succeeded, want refusal", drop)
		} else if !errors.Is(err, ErrMigrationV4DropUnknown) {
			t.Fatalf("PreviewV4(drop %v) error = %v", drop, err)
		}
	}
}
