package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// writeExecutable creates dir/name with content and owner-only-executable
// permissions. The executable bit is convenio, not a trust fact: Discover
// records regular-file shape, owner, and digest, never the mode.
func writeExecutable(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o700); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}

// TestOSSystemEndToEnd drives the production entry points Discover, Trust,
// and Verify through the production OSSystem on a real temporary tree:
// symlink resolution, digest recording, trust acceptance, retarget
// detection, byte-change detection, and non-regular replacement detection.
func TestOSSystemEndToEnd(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	plugindir := filepath.Join(root, "plugins")
	realdir := filepath.Join(root, "real")
	if err := os.MkdirAll(plugindir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(realdir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := []byte("#!/bin/sh\nexit 0\n")
	target := writeExecutable(t, realdir, "tool", content)
	source := filepath.Join(plugindir, "ax-provider-mytool")
	if err := os.Symlink(target, source); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	system := OSSystem{}
	owner := CurrentOperatorPolicy()
	cfg := Config{Platform: scalar.PlatformLinux, PluginDirs: []string{plugindir}}

	got, err := Discover(cfg, owner, system)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	var candidate *Candidate
	for i := range got {
		if got[i].ID() == "mytool" {
			candidate = &got[i]
		}
	}
	if candidate == nil {
		t.Fatalf("candidate mytool missing: %v", got)
	}
	canon, _ := candidate.CanonicalPath()
	if canon != target {
		t.Fatalf("CanonicalPath = %q, want %q", canon, target)
	}
	sum := sha256.Sum256(content)
	digest, _ := candidate.Digest()
	if digest.Hex() != hex.EncodeToString(sum[:]) {
		t.Fatalf("Digest = %v, want sha256 of target bytes", digest)
	}

	record, err := Trust(*candidate, mustTimestamp(t, trustInstant))
	if err != nil {
		t.Fatalf("Trust: %v", err)
	}
	if err := Verify(record, owner, system); err != nil {
		t.Fatalf("Verify of an unchanged tree: %v", err)
	}

	t.Run("retargeted symlink requires renewed trust", func(t *testing.T) {
		evil := writeExecutable(t, realdir, "evil", []byte("#!/bin/sh\nexit 1\n"))
		if err := os.Remove(source); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if err := os.Symlink(evil, source); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		defer func() {
			os.Remove(source)
			os.Symlink(target, source)
		}()
		if err := Verify(record, owner, system); err == nil {
			t.Fatal("Verify accepted a retargeted symlink")
		} else if code := errorCode(t, err); code != codeIntegrityFailure {
			t.Fatalf("code = %q, want integrity_failure", code)
		}
	})

	t.Run("changed bytes require renewed trust", func(t *testing.T) {
		if err := os.WriteFile(target, []byte("#!/bin/sh\nexit 2\n"), 0o700); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		defer os.WriteFile(target, content, 0o700)
		if err := Verify(record, owner, system); err == nil {
			t.Fatal("Verify accepted changed bytes")
		}
	})

	t.Run("non-regular replacement requires renewed trust", func(t *testing.T) {
		plain := filepath.Join(plugindir, "ax-provider-plain")
		if err := os.WriteFile(plain, []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		plainCandidates, err := Discover(Config{Platform: scalar.PlatformLinux, PluginDirs: []string{plugindir}}, owner, system)
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		var plainCandidate *Candidate
		for i := range plainCandidates {
			if plainCandidates[i].ID() == "plain" {
				plainCandidate = &plainCandidates[i]
			}
		}
		if plainCandidate == nil {
			t.Fatal("candidate plain missing")
		}
		plainRecord, err := Trust(*plainCandidate, mustTimestamp(t, trustInstant))
		if err != nil {
			t.Fatalf("Trust: %v", err)
		}
		if err := os.Remove(plain); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if err := os.Mkdir(plain, 0o755); err != nil {
			t.Fatalf("Mkdir: %v", err)
		}
		defer os.Remove(plain)
		if err := Verify(plainRecord, owner, system); err == nil {
			t.Fatal("Verify accepted a directory replacement")
		}
	})

	t.Run("direct executable without symlink", func(t *testing.T) {
		direct := filepath.Join(plugindir, "ax-provider-direct")
		if err := os.WriteFile(direct, []byte("direct"), 0o700); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		defer os.Remove(direct)
		resolved, err := filepath.EvalSymlinks(direct)
		if err != nil {
			t.Fatalf("EvalSymlinks: %v", err)
		}
		directCandidates, err := Discover(Config{Platform: scalar.PlatformLinux, PluginDirs: []string{plugindir}}, owner, system)
		if err != nil {
			// The tree now also holds mytool/plain fixtures; a duplicate
			// or refusal from an unrelated fixture must not mask this
			// case, so report it plainly.
			t.Fatalf("Discover: %v", err)
		}
		for _, found := range directCandidates {
			if found.ID() == "direct" {
				canon, _ := found.CanonicalPath()
				if canon != resolved {
					t.Fatalf("CanonicalPath = %q, want %q", canon, resolved)
				}
				return
			}
		}
		t.Fatal("candidate direct missing")
	})
}

// TestOSSystemRefusesRelativePATHDir drives the PATH absolute-directory
// gate through the production OSSystem: a relative PATH entry fails with
// invalid_config instead of resolving against the process working
// directory.
func TestOSSystemRefusesRelativePATHDir(t *testing.T) {
	t.Setenv("PATH", "."+string(os.PathListSeparator)+"/definitely-absent-ax-provider-dir")
	_, err := Discover(
		Config{Platform: scalar.PlatformLinux, AllowPathPlugins: true},
		CurrentOperatorPolicy(),
		OSSystem{},
	)
	if err == nil {
		t.Fatal("Discover accepted a relative PATH entry, want invalid_config")
	}
	if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
}

// TestOSSystemDuplicateNeverExecutes is the behavioral half of the Section
// 7.1 ordering guarantee on the production seam: a duplicate fixture
// containing a live shell script that marks a sentinel file fails
// discovery with invalid_config and leaves the sentinel absent.
func TestOSSystemDuplicateNeverExecutes(t *testing.T) {
	root := t.TempDir()
	plugindir := filepath.Join(root, "plugins")
	if err := os.MkdirAll(plugindir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	sentinel := filepath.Join(root, "executed")
	script := []byte("#!/bin/sh\necho executed >> " + sentinel + "\n")
	writeExecutable(t, plugindir, "ax-provider-codex", script)

	_, err := Discover(
		Config{Platform: scalar.PlatformLinux, PluginDirs: []string{plugindir}},
		CurrentOperatorPolicy(),
		OSSystem{},
	)
	if err == nil {
		t.Fatal("Discover admitted a builtin-shadowing executable")
	}
	if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
	if _, statErr := os.Stat(sentinel); !os.IsNotExist(statErr) {
		t.Fatalf("duplicate candidate was executed; sentinel present: %v", statErr)
	}
}

// TestOSSystemReadDirReportsFailure proves the headline invariant at the
// one place it can actually be violated: a failed ReadDir is reported as
// an error, never as an absence. The production entry points are
// OSSystem.ReadDir and Discover.
func TestOSSystemReadDirReportsFailure(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "definitely-absent")
	if names, err := (OSSystem{}).ReadDir(absent); err == nil {
		t.Fatalf("ReadDir(%q) = %v, nil; want a failure, never an absence", absent, names)
	}
	got, err := Discover(
		Config{Platform: scalar.PlatformLinux, PluginDirs: []string{absent}},
		CurrentOperatorPolicy(),
		OSSystem{},
	)
	if got != nil {
		t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
	}
	if err == nil {
		t.Fatal("Discover admitted an unreadable plugin directory, want local_precondition_failed")
	}
	if code := errorCode(t, err); code != codeLocalPrecondition {
		t.Fatalf("code = %q, want local_precondition_failed", code)
	}
	if detail := errorDetail(t, err); !strings.Contains(detail, "cannot list") {
		t.Fatalf("detail does not name the list refusal: %q", detail)
	}
}

// TestOSSystemInspectReportsStatFailure proves Inspect propagates a stat
// failure instead of fabricating a regular-file fact for a missing target.
func TestOSSystemInspectReportsStatFailure(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "definitely-absent")
	if info, err := (OSSystem{}).Inspect(absent); err == nil {
		t.Fatalf("Inspect(%q) = %+v, nil; want a failure, never a fabricated fact", absent, info)
	}
}

// TestOSSystemInspectReportsAttestationFailure forces the owner-attestation
// failure path through the production Inspect: attestation reports the
// error instead of a uid-0 fact.
func TestOSSystemInspectReportsAttestationFailure(t *testing.T) {
	boom := errors.New("fake: ownership metadata unavailable")
	original := ownerAttester
	ownerAttester = func(os.FileInfo) (uint32, error) { return 0, boom }
	defer func() { ownerAttester = original }()

	target := writeExecutable(t, t.TempDir(), "tool", []byte("x"))
	if info, err := (OSSystem{}).Inspect(target); err == nil {
		t.Fatalf("Inspect attested %+v, nil on attestation failure; want the failure, never uid 0", info)
	} else if !errors.Is(err, boom) {
		t.Fatalf("Inspect error = %v, want the attestation cause %v", err, boom)
	}
}

// TestOwnerAttesterIsUnreassigned pins the os.go invariant that production
// code never reassigns ownerAttester: the seam must attest exactly what
// fileOwnerUID attests. A production init() substituting a weaker attester
// (for example, one returning the operator for every file) reddens here,
// while the identical weakening written into fileOwnerUID itself is already
// killed by TestFileOwnerUIDRefusesWithoutMetadata. The production call
// site is OSSystem.Inspect.
func TestOwnerAttesterIsUnreassigned(t *testing.T) {
	if reflect.ValueOf(ownerAttester).Pointer() != reflect.ValueOf(fileOwnerUID).Pointer() {
		t.Fatal("ownerAttester does not attest as fileOwnerUID; production code reassigned the seam")
	}
}

// unattestedFileInfo is an os.FileInfo with no ownership metadata: Sys
// returns nil, which the unix seam must refuse rather than attest as uid 0.
type unattestedFileInfo struct{}

func (unattestedFileInfo) Name() string       { return "unattested" }
func (unattestedFileInfo) Size() int64        { return 1 }
func (unattestedFileInfo) Mode() os.FileMode  { return 0o700 }
func (unattestedFileInfo) ModTime() time.Time { return time.Time{} }
func (unattestedFileInfo) IsDir() bool        { return false }
func (unattestedFileInfo) Sys() any           { return nil }

// TestFileOwnerUIDRefusesWithoutMetadata proves the attestation seam fails
// closed when ownership metadata is unavailable, against its own doc
// comment: the host refuses trust rather than treating an unknown owner
// as approved. The forward control attests a real stat result first, so
// the refusal is attributed to the missing metadata, not to a broken seam.
func TestFileOwnerUIDRefusesWithoutMetadata(t *testing.T) {
	target := writeExecutable(t, t.TempDir(), "tool", []byte("x"))
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	uid, err := fileOwnerUID(info)
	if err != nil {
		t.Fatalf("fileOwnerUID of a real stat result: %v", err)
	}
	if uid != uint32(os.Geteuid()) {
		t.Fatalf("fileOwnerUID = %d, want effective uid %d", uid, os.Geteuid())
	}
	if uid, err := fileOwnerUID(unattestedFileInfo{}); err == nil {
		t.Fatalf("fileOwnerUID attested uid %d without metadata; want refusal, never uid 0", uid)
	}
}

// TestCurrentOperatorPolicySeedsNoAdministrator proves the policy
// construction site carries no administrator wildcard: the operator alone
// is trusted, against the doc comment's "no additional
// administrator-approved identities".
func TestCurrentOperatorPolicySeedsNoAdministrator(t *testing.T) {
	policy := CurrentOperatorPolicy()
	if policy.OperatorUID != uint32(os.Geteuid()) {
		t.Fatalf("OperatorUID = %d, want effective uid %d", policy.OperatorUID, os.Geteuid())
	}
	if len(policy.AdministratorUIDs) != 0 {
		t.Fatalf("AdministratorUIDs = %v, want no additional administrator-approved identities", policy.AdministratorUIDs)
	}
}

// TestOSSystemPathDirsSkipsEmptyEntries proves the PATH seam drops empty
// entries instead of admitting an empty directory that Canonicalize would
// resolve against the process working directory.
func TestOSSystemPathDirsSkipsEmptyEntries(t *testing.T) {
	t.Setenv("PATH", string(os.PathListSeparator)+"/a"+string(os.PathListSeparator)+string(os.PathListSeparator)+"/b"+string(os.PathListSeparator))
	got := (OSSystem{}).PathDirs()
	for _, dir := range got {
		if dir == "" {
			t.Fatalf("PathDirs kept an empty entry in %v", got)
		}
	}
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("PathDirs = %v, want [/a /b] in listed order", got)
	}
	t.Setenv("PATH", "")
	if got := (OSSystem{}).PathDirs(); got != nil {
		t.Fatalf("PathDirs of an empty PATH = %v, want nil", got)
	}
}

// TestOSSystemPATHDiscovery drives the PATH source through the production
// seam with an isolated PATH: discovery finds the tool when allowed,
// ignores PATH when disallowed, and refuses duplicates across PATH
// directories.
func TestOSSystemPATHDiscovery(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	first := filepath.Join(root, "bin1")
	second := filepath.Join(root, "bin2")
	for _, dir := range []string{first, second} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
	}
	writeExecutable(t, first, "ax-provider-pathtool", []byte("p1"))
	t.Setenv("PATH", first+string(os.PathListSeparator)+second)

	system := OSSystem{}
	owner := CurrentOperatorPolicy()

	allowed := Config{Platform: scalar.PlatformLinux, AllowPathPlugins: true}
	got, err := Discover(allowed, owner, system)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	found := false
	for _, candidate := range got {
		if candidate.ID() == "pathtool" && candidate.Source() == "path" {
			found = true
			canon, _ := candidate.CanonicalPath()
			if !strings.HasPrefix(canon, first) {
				t.Fatalf("CanonicalPath = %q, want it under %q", canon, first)
			}
		}
	}
	if !found {
		t.Fatal("PATH candidate missing when allow_path_plugins is true")
	}

	denied, err := Discover(Config{Platform: scalar.PlatformLinux}, owner, system)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, candidate := range denied {
		if candidate.ID() == "pathtool" {
			t.Fatal("PATH candidate admitted while allow_path_plugins is false")
		}
	}

	writeExecutable(t, second, "ax-provider-pathtool", []byte("p2"))
	if _, err := Discover(allowed, owner, system); err == nil {
		t.Fatal("Discover admitted one ID from two PATH directories")
	} else if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
}
