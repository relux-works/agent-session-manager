package hosttrust

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeConfigBindingUsesExactClosedShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	valid, err := encodeConfigBinding(path, filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		document []byte
	}{
		{name: "valid", document: valid},
		{name: "missing schema", document: []byte(`{"schema_version":"1.0.0","config_path":"` + path + `"}`)},
		{name: "missing path", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema_version":"1.0.0"}`)},
		{name: "missing state root", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema_version":"1.0.0","config_path":"` + path + `"}`)},
		{name: "unknown member", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema_version":"1.0.0","config_path":"` + path + `","extra":true}`)},
		{name: "downgraded version", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema_version":"0.9.0","config_path":"` + path + `"}`)},
		{name: "wrong schema", document: []byte(`{"schema":"urn:ax:schema:other","schema_version":"1.0.0","config_path":"` + path + `"}`)},
		{name: "duplicate member", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema":"urn:ax:schema:host-config-binding","schema_version":"1.0.0","config_path":"` + path + `"}`)},
		{name: "trailing document", document: append(append([]byte(nil), valid...), []byte(` {}`)...)},
		{name: "relative path", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema_version":"1.0.0","config_path":"config.toml"}`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := decodeConfigBinding(test.document)
			if test.name == "valid" {
				if err != nil || got != path {
					t.Fatalf("decodeConfigBinding(valid) = %q, %v; want %q", got, err, path)
				}
				return
			}
			if err == nil {
				t.Fatalf("decodeConfigBinding(%s) succeeded with %q", test.name, got)
			}
		})
	}
}

func TestOpenRefusesPresentMalformedOrUnsafeConfigBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	valid, err := encodeConfigBinding(path, filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	unknown := append(append([]byte(nil), valid[:len(valid)-1]...), []byte(`,"extra":true}`)...)
	tests := []struct {
		name     string
		document []byte
		mode     os.FileMode
		asDir    bool
	}{
		{name: "partial", document: []byte(`{"schema":"urn:ax:schema:host-config-binding","schema_version":"1.0.0"}`), mode: 0o600},
		{name: "unknown", document: unknown, mode: 0o600},
		{name: "unsafe mode", document: valid, mode: 0o644},
		{name: "directory", mode: 0o700, asDir: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := t.TempDir()
			root := filepath.Join(state, hostChannelDir)
			if err := os.MkdirAll(root, 0o700); err != nil {
				t.Fatal(err)
			}
			binding := filepath.Join(root, configBindingFileName)
			if test.asDir {
				if err := os.Mkdir(binding, test.mode); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(binding, test.document, test.mode); err != nil {
				t.Fatal(err)
			}
			if _, err := Open(resolvedPathsForTest(t, path, state)); err == nil {
				t.Fatal("Open malformed/unsafe binding succeeded")
			} else if test.name == "unsafe mode" || test.name == "directory" {
				if !errors.Is(err, ErrTrustUnsafeCustody) {
					t.Fatalf("Open(%s) error = %v, want ErrTrustUnsafeCustody", test.name, err)
				}
			}
		})
	}
}

func TestMissingConfigBindingIsExplicitUnboundState(t *testing.T) {
	state := t.TempDir()
	configPath := filepath.Join(state, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := openPathsForTest(t, configPath, state)
	bound, err := loadConfigBinding(osFileSystem{}, store.Root())
	if err != nil {
		t.Fatal(err)
	}
	if bound != "" {
		t.Fatalf("missing binding = %q, want explicit empty pre-binding state", bound)
	}
	if err := store.ValidateBoundConfigPath(resolvedPathsForTest(t, configPath, state)); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("ValidateBoundConfigPath(unbound) error = %v, want resource mismatch", err)
	}
}

func TestConfigBindingPersistsCanonicalPathAndRejectsRebinding(t *testing.T) {
	state := t.TempDir()
	configPath := filepath.Join(state, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := resolvedPathsForTest(t, configPath, state)
	store := openPathsForTest(t, configPath, state)
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatalf("EnsureConfigBinding = %v", err)
	}
	document, err := os.ReadFile(bindingPath(store.Root()))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := decodeConfigBinding(document)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := canonicalExistingPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if bound != canonical {
		t.Fatalf("persisted binding = %q, want canonical %q", bound, canonical)
	}
	targetDocument, err := os.ReadFile(configBindingPath(canonical))
	if err != nil {
		t.Fatal(err)
	}
	targetBinding, err := decodeConfigBindingRecord(targetDocument)
	if err != nil {
		t.Fatal(err)
	}
	canonicalState, err := canonicalExistingPath(state)
	if err != nil {
		t.Fatal(err)
	}
	if targetBinding.ConfigPath != canonical || targetBinding.StateRoot != canonicalState {
		t.Fatalf("target binding = %+v, want path %q and state root %q", targetBinding, canonical, canonicalState)
	}
	info, err := os.Lstat(bindingPath(store.Root()))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("binding mode = %04o, want 0600", info.Mode().Perm())
	}
	reopened := openPathsForTest(t, configPath, state)
	if err := reopened.ValidateBoundConfigPath(paths); err != nil {
		t.Fatalf("reopened ValidateBoundConfigPath = %v", err)
	}
	other := filepath.Join(state, "other.toml")
	if err := os.WriteFile(other, []byte("other"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := reopened.EnsureConfigBinding(resolvedPathsForTest(t, other, state)); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("EnsureConfigBinding(other) error = %v, want resource mismatch", err)
	}
}

func TestOpenRefusesDerivedBindingWhenTargetSidecarIsMissing(t *testing.T) {
	state := t.TempDir()
	configPath := filepath.Join(state, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	paths := resolvedPathsForTest(t, configPath, state)
	store := openPathsForTest(t, configPath, state)
	if err := store.EnsureConfigBinding(paths); err != nil {
		t.Fatalf("EnsureConfigBinding = %v", err)
	}
	canonical, err := canonicalExistingPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(configBindingPath(canonical)); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(paths); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("Open(missing target sidecar) error = %v, want resource mismatch", err)
	}
}

func TestEnsureConfigBindingRefusesForeignStoreWithBoundTarget(t *testing.T) {
	directory := t.TempDir()
	targetState := filepath.Join(directory, "target-state")
	foreignState := filepath.Join(directory, "foreign-state")
	configPath := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	targetPaths := resolvedPathsForTest(t, configPath, targetState)
	target, err := Open(targetPaths)
	if err != nil {
		t.Fatal(err)
	}
	if err := target.EnsureConfigBinding(targetPaths); err != nil {
		t.Fatalf("target EnsureConfigBinding = %v", err)
	}
	foreignPaths := resolvedPathsForTest(t, configPath, foreignState)
	foreign, err := Open(foreignPaths)
	if err != nil {
		t.Fatal(err)
	}
	if err := foreign.EnsureConfigBinding(foreignPaths); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("foreign EnsureConfigBinding = %v, want resource mismatch", err)
	}
	canonical, err := canonicalExistingPath(configPath)
	if err != nil {
		t.Fatal(err)
	}
	binding, present, err := loadTargetConfigBinding(osFileSystem{}, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("foreign EnsureConfigBinding removed the target association")
	}
	if err := validateBindingForStore(binding, canonical, target.root); err != nil {
		t.Fatalf("target association after foreign refusal = %v", err)
	}
	if _, err := os.Lstat(bindingPath(foreign.Root())); !os.IsNotExist(err) {
		t.Fatalf("foreign derived binding after refusal = %v, want absent", err)
	}
}

func TestConfigBindingRefusesForeignStoreStateRoot(t *testing.T) {
	directory := t.TempDir()
	stateDir := filepath.Join(directory, "state")
	foreignState := filepath.Join(directory, "foreign-state")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(directory, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := openPathsForTest(t, configPath, foreignState)
	if err := store.EnsureConfigBinding(resolvedPathsForTest(t, configPath, stateDir)); !errors.Is(err, ErrExclusiveHoldResourceMismatch) {
		t.Fatalf("foreign EnsureConfigBinding error = %v, want resource mismatch", err)
	}
	if _, err := os.Lstat(bindingPath(store.Root())); !os.IsNotExist(err) {
		t.Fatalf("foreign binding sidecar after refusal = %v, want absent", err)
	}
}

type unreadableBindingFileSystem struct{ osFileSystem }

func (unreadableBindingFileSystem) ReadFile(name string) ([]byte, error) {
	if filepath.Base(name) == configBindingFileName {
		return nil, errors.New("injected binding read failure")
	}
	return osFileSystem{}.ReadFile(name)
}

func TestOpenRefusesUnreadableConfigBinding(t *testing.T) {
	state := t.TempDir()
	root := filepath.Join(state, hostChannelDir)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(state, "config.toml")
	if err := os.WriteFile(configPath, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := encodeConfigBinding(configPath, filepath.Dir(configPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, configBindingFileName), document, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStore(resolvedPathsForTest(t, configPath, state), unreadableBindingFileSystem{}); err == nil {
		t.Fatal("openStore unreadable binding succeeded")
	} else if !errors.Is(err, ErrTrustStoreUnreadable) {
		t.Fatalf("openStore unreadable binding error = %v, want ErrTrustStoreUnreadable", err)
	}
}

func TestConfigBindingJSONDoesNotReplicate(t *testing.T) {
	var shape ConfigBinding
	document, err := json.Marshal(ConfigBinding{Schema: configBindingSchema, SchemaVersion: configBindingVersion, ConfigPath: "/state/config.toml", StateRoot: "/state"})
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(document, &shape); err != nil || shape.ConfigPath == "" {
		t.Fatalf("binding JSON sanity check = %v/%+v", err, shape)
	}
	if !MatchExcludedFromReplication(hostChannelDir + "/" + configBindingFileName) {
		t.Fatal("config-binding.json is not excluded from replication")
	}
}
