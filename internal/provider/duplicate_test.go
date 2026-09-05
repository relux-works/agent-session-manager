package provider

import (
	"strings"
	"testing"
)

// duplicateCases narrows the Section 7.1 duplicate refusal across every
// source pair that can declare one ID: plugin against plugin, plugin
// against builtin, plugin against PATH, and builtin against PATH. Each case
// must fail with the section-mandated invalid_config.
func duplicateCases() map[string]func(fake *fakeSystem, cfg *Config) {
	return map[string]func(fake *fakeSystem, cfg *Config){
		"plugin against plugin": func(fake *fakeSystem, cfg *Config) {
			fake.addFile("/plugins/a", "ax-provider-dup", []byte("one"), fakeUID)
			fake.addFile("/plugins/b", "ax-provider-dup", []byte("two"), fakeUID)
			*cfg = fakeConfig("/plugins/a", "/plugins/b")
		},
		"plugin against builtin": func(fake *fakeSystem, cfg *Config) {
			fake.addFile("/plugins", "ax-provider-codex", []byte("shadow"), fakeUID)
			*cfg = fakeConfig("/plugins")
		},
		"plugin against PATH": func(fake *fakeSystem, cfg *Config) {
			fake.addFile("/plugins", "ax-provider-dup", []byte("one"), fakeUID)
			fake.addFile("/pathbin", "ax-provider-dup", []byte("two"), fakeUID)
			fake.pathDirs = []string{"/pathbin"}
			*cfg = fakeConfig("/plugins")
			cfg.AllowPathPlugins = true
		},
		"builtin against PATH": func(fake *fakeSystem, cfg *Config) {
			fake.addFile("/pathbin", "ax-provider-claude", []byte("shadow"), fakeUID)
			fake.pathDirs = []string{"/pathbin"}
			*cfg = fakeConfig()
			cfg.AllowPathPlugins = true
		},
		"same file observed through two PATH dirs": func(fake *fakeSystem, cfg *Config) {
			fake.addFile("/path/a", "ax-provider-dup", []byte("same"), fakeUID)
			fake.addFile("/path/b", "ax-provider-dup", []byte("same"), fakeUID)
			fake.pathDirs = []string{"/path/a", "/path/b"}
			*cfg = fakeConfig()
			cfg.AllowPathPlugins = true
		},
	}
}

// TestDiscoverRefusesDuplicates proves the Section 7.1 duplicate rule at
// the production entry point Discover: two candidates declaring one ID
// fail with invalid_config, with no duplicate-selection override — not
// even for byte-identical observations of one file.
func TestDiscoverRefusesDuplicates(t *testing.T) {
	for name, setup := range duplicateCases() {
		t.Run(name, func(t *testing.T) {
			fake := newFakeSystem()
			var cfg Config
			setup(fake, &cfg)
			got, err := Discover(cfg, fakeOwner(), fake)
			if got != nil {
				t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
			}
			if err == nil {
				t.Fatalf("Discover admitted a duplicate provider ID, want invalid_config")
			}
			if code := errorCode(t, err); code != codeInvalidConfig {
				t.Fatalf("code = %q, want invalid_config", code)
			}
			if !strings.Contains(err.Error(), "duplicate provider") {
				t.Fatalf("detail does not name the duplicate: %v", err)
			}
		})
	}
}

// TestDuplicateRefusalNamesBothSources proves the refusal detail carries
// both declaring sources, so the operator knows which candidates to remove
// or rename.
func TestDuplicateRefusalNamesBothSources(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins/a", "ax-provider-dup", []byte("one"), fakeUID)
	fake.addFile("/plugins/b", "ax-provider-dup", []byte("two"), fakeUID)
	_, err := Discover(fakeConfig("/plugins/a", "/plugins/b"), fakeOwner(), fake)
	if err == nil {
		t.Fatal("Discover admitted a duplicate provider ID")
	}
	detail := err.Error()
	if !strings.Contains(detail, "plugin_dirs[0]") || !strings.Contains(detail, "plugin_dirs[1]") {
		t.Fatalf("detail names neither source: %v", err)
	}
}

// TestDuplicateRefusalPrecedesAnyProbeOrExecution proves the Section 7.1
// ordering guarantee structurally: Discover accepts no probe or execution
// callback, and the package imports no process-execution facility, so a
// duplicate fixture containing a live executable cannot be reached. The
// import scan lives in enumeration_test.go; this case pins the behavior at
// a fixture shaped like an attack: a shadowing executable whose content
// would execute if discovery ever shelled out.
func TestDuplicateRefusalPrecedesAnyProbeOrExecution(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-codex", []byte("#!/bin/sh\nevil"), fakeUID)
	_, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
	if err == nil {
		t.Fatal("Discover admitted a builtin-shadowing executable")
	}
	if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
}

// TestDistinctIDsCoexist proves the refusal is scoped to true duplicates:
// neighboring IDs from every source coexist in one result.
func TestDistinctIDsCoexist(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-alpha", []byte("a"), fakeUID)
	fake.addFile("/pathbin", "ax-provider-beta", []byte("b"), fakeUID)
	fake.pathDirs = []string{"/pathbin"}
	cfg := fakeConfig("/plugins")
	cfg.AllowPathPlugins = true
	got, err := Discover(cfg, fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover refused distinct IDs: %v", err)
	}
	if len(got) != len(builtinOrder)+2 {
		t.Fatalf("candidate count = %d, want %d", len(got), len(builtinOrder)+2)
	}
}
