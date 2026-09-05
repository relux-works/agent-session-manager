package provider

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// TestDiscoverEnumeratesSourcesInSectionOrder proves the Section 7.1
// candidate order: configured plugin directories in listed order, then
// built-in adapters, then PATH when allowed. The production entry point is
// Discover.
func TestDiscoverEnumeratesSourcesInSectionOrder(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins/a", "ax-provider-zzz", []byte("zzz"), fakeUID)
	fake.addFile("/plugins/b", "ax-provider-aaa", []byte("aaa"), fakeUID)
	fake.pathDirs = []string{"/pathbin"}
	fake.addFile("/pathbin", "ax-provider-pathone", []byte("p"), fakeUID)

	cfg := fakeConfig("/plugins/a", "/plugins/b")
	cfg.AllowPathPlugins = true
	got, err := Discover(cfg, fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	var order []string
	for _, candidate := range got {
		order = append(order, string(candidate.Kind())+"/"+candidate.ID())
	}
	want := []string{
		"external/zzz",
		"external/aaa",
		"builtin/codex",
		"builtin/claude",
		"builtin/gemini",
		"builtin/muse",
		"builtin/antigravity",
		"builtin/pi",
		"external/pathone",
	}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("discovery order = %v, want %v", order, want)
	}
	sources := map[string]string{}
	for _, candidate := range got {
		sources[candidate.ID()] = candidate.Source()
	}
	if sources["zzz"] != "plugin_dirs[0]" || sources["aaa"] != "plugin_dirs[1]" {
		t.Fatalf("plugin sources = %v, want plugin_dirs[0]/plugin_dirs[1]", sources)
	}
	if sources["codex"] != "builtin" {
		t.Fatalf("builtin source = %q, want builtin", sources["codex"])
	}
	if sources["pathone"] != "path" {
		t.Fatalf("path source = %q, want path", sources["pathone"])
	}
}

// TestDiscoverSortsEntriesWithinADirectory narrows the determinism claim:
// the fake returns entries in reverse insertion order, so bytewise output
// proves Discover sorts instead of inheriting filesystem order.
func TestDiscoverSortsEntriesWithinADirectory(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-mango", []byte("m"), fakeUID)
	fake.addFile("/plugins", "ax-provider-apple", []byte("a"), fakeUID)
	fake.addFile("/plugins", "ax-provider-cherry", []byte("c"), fakeUID)

	got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if got[0].ID() != "apple" || got[1].ID() != "cherry" || got[2].ID() != "mango" {
		t.Fatalf("directory order = [%s %s %s], want [apple cherry mango]", got[0].ID(), got[1].ID(), got[2].ID())
	}
}

// TestDiscoverSkipsPATHWhenDisallowed proves the Section 7.1 PATH gate:
// PATH is consulted only when allow_path_plugins is true.
func TestDiscoverSkipsPATHWhenDisallowed(t *testing.T) {
	fake := newFakeSystem()
	fake.pathDirs = []string{"/pathbin"}
	fake.addFile("/pathbin", "ax-provider-pathtool", []byte("p"), fakeUID)

	got, err := Discover(fakeConfig(), fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, candidate := range got {
		if candidate.Kind() == KindExternal {
			t.Fatalf("PATH candidate %q admitted while allow_path_plugins is false", candidate.ID())
		}
	}
	if len(got) != len(builtinOrder) {
		t.Fatalf("candidate count = %d, want %d builtins only", len(got), len(builtinOrder))
	}

	cfg := fakeConfig()
	cfg.AllowPathPlugins = true
	got, err = Discover(cfg, fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover with PATH allowed: %v", err)
	}
	found := false
	for _, candidate := range got {
		if candidate.ID() == "pathtool" && candidate.Source() == "path" {
			found = true
		}
	}
	if !found {
		t.Fatalf("PATH candidate missing when allow_path_plugins is true: %v", got)
	}
}

// TestDiscoverRecordsTrustFacts proves every accepted external candidate
// carries the Section 7.1 trust-time facts: canonical path, digest, and
// approving owner.
func TestDiscoverRecordsTrustFacts(t *testing.T) {
	fake := newFakeSystem()
	fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)

	got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	var foo *Candidate
	for i, candidate := range got {
		if candidate.ID() == "foo" {
			foo = &got[i]
		}
	}
	if foo == nil {
		t.Fatalf("external candidate foo missing: %v", got)
	}
	source, ok := foo.SourcePath()
	if !ok || source != "/plugins/ax-provider-foo" {
		t.Fatalf("SourcePath = %q, %v; want /plugins/ax-provider-foo, true", source, ok)
	}
	canon, ok := foo.CanonicalPath()
	if !ok || canon != "/opt/real/foo" {
		t.Fatalf("CanonicalPath = %q, %v; want /opt/real/foo, true", canon, ok)
	}
	digest, ok := foo.Digest()
	if !ok {
		t.Fatal("Digest reports absence for an external candidate")
	}
	want := scalar.SHA256Digest([]byte("bytes"))
	if digest != want {
		t.Fatalf("Digest = %v, want %v", digest, want)
	}
	owner, ok := foo.Owner()
	if !ok || owner != "uid:1000" {
		t.Fatalf("Owner = %q, %v; want uid:1000, true", owner, ok)
	}
}

// TestDiscoverIgnoresNonCandidateEntries proves the prefix is an anchor,
// not a substring: names without a leading ax-provider- prefix are
// skipped, including the bare prefix stem itself and names that contain
// the prefix mid-string. Under a substring match,
// vendor-ax-provider-evil would discover provider evil.
func TestDiscoverIgnoresNonCandidateEntries(t *testing.T) {
	fake := newFakeSystem()
	fake.entries["/plugins"] = []string{
		"README.md",
		"ax-provider",
		"other-tool",
		"vendor-ax-provider-evil",
		"xax-provider-foo",
	}
	fake.canon["/plugins/other-tool"] = "/plugins/other-tool"
	fake.canon["/plugins/README.md"] = "/plugins/README.md"
	fake.canon["/plugins/ax-provider"] = "/plugins/ax-provider"
	fake.canon["/plugins/ax-provider-foo.bak"] = "/plugins/ax-provider-foo.bak"
	fake.files["/plugins/vendor-ax-provider-evil"] = fakeFile{content: []byte("evil"), uid: fakeUID, regular: true}
	fake.files["/plugins/xax-provider-foo"] = fakeFile{content: []byte("x"), uid: fakeUID, regular: true}
	fake.canon["/plugins/vendor-ax-provider-evil"] = "/plugins/vendor-ax-provider-evil"
	fake.canon["/plugins/xax-provider-foo"] = "/plugins/xax-provider-foo"

	got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	for _, candidate := range got {
		if candidate.Kind() == KindExternal {
			t.Fatalf("non-candidate entry admitted as %q", candidate.ID())
		}
	}
}

// TestDiscoverRefusesMalformedNames narrows the grammar gate across the
// provider-id boundary: uppercase, leading dash, empty, and overlong
// suffixes fail, each with invalid_config.
func TestDiscoverRefusesMalformedNames(t *testing.T) {
	for _, name := range []string{
		"ax-provider-ABC",
		"ax-provider--lead",
		"ax-provider-",
		"ax-provider-foo.bak",
		"ax-provider-has space",
		"ax-provider-012345678901234567890123456789012",
	} {
		fake := newFakeSystem()
		fake.entries["/plugins"] = []string{name}
		fake.canon["/plugins/"+name] = "/plugins/" + name
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover(%q) returned %d candidates alongside a refusal, want no partial set", name, len(got))
		}
		if err == nil {
			t.Fatalf("Discover(%q) succeeded, want invalid_config", name)
		}
		if code := errorCode(t, err); code != codeInvalidConfig {
			t.Fatalf("Discover(%q) code = %q, want invalid_config", name, code)
		}
	}
}

// TestDiscoverRefusesNonRegularTargets proves the Section 7.1 regular-file
// rule in both direct and symlinked form.
func TestDiscoverRefusesNonRegularTargets(t *testing.T) {
	t.Run("directory entry", func(t *testing.T) {
		fake := newFakeSystem()
		fake.addLinkedFile("/plugins", "ax-provider-odd", "/opt/real/odd", []byte("x"), fakeUID, false)
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if err == nil {
			t.Fatal("Discover admitted a directory target, want invalid_config")
		}
		if code := errorCode(t, err); code != codeInvalidConfig {
			t.Fatalf("code = %q, want invalid_config", code)
		}
	})
	t.Run("symlink to directory", func(t *testing.T) {
		fake := newFakeSystem()
		fake.entries["/plugins"] = []string{"ax-provider-linkdir"}
		fake.canon["/plugins/ax-provider-linkdir"] = "/opt/dirs/linkdir"
		fake.files["/opt/dirs/linkdir"] = fakeFile{content: nil, uid: fakeUID, regular: false}
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if err == nil {
			t.Fatal("Discover admitted a symlink-to-directory, want invalid_config")
		}
	})
}

// TestDiscoverRefusesUnapprovedOwners narrows the ownership gate: the same
// bytes are refused under a foreign UID, admitted under the operator UID,
// and admitted under an administrator-approved UID.
func TestDiscoverRefusesUnapprovedOwners(t *testing.T) {
	setup := func() *fakeSystem {
		fake := newFakeSystem()
		fake.addFile("/plugins", "ax-provider-foo", []byte("x"), foreignUID)
		return fake
	}
	if got, err := Discover(fakeConfig("/plugins"), fakeOwner(), setup()); err == nil {
		t.Fatal("Discover admitted a foreign-owned executable, want invalid_config")
	} else if got != nil {
		t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
	} else if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
	if _, err := Discover(fakeConfig("/plugins"), OwnerPolicy{OperatorUID: foreignUID}, setup()); err != nil {
		t.Fatalf("Discover refused the operator-owned executable: %v", err)
	}
	admin := OwnerPolicy{OperatorUID: fakeUID, AdministratorUIDs: []uint32{foreignUID}}
	if _, err := Discover(fakeConfig("/plugins"), admin, setup()); err != nil {
		t.Fatalf("Discover refused the administrator-approved executable: %v", err)
	}
	noRootException := OwnerPolicy{OperatorUID: fakeUID}
	if _, err := Discover(fakeConfig("/plugins"), noRootException, setup()); err == nil {
		t.Fatal("Discover admitted foreign UID without an explicit approval, want refusal")
	}
}

// TestDiscoverRefusesRelativePluginDir proves relative configured
// directories fail closed instead of resolving against an ambient working
// directory. The gate is narrowed across the index domain: a relative entry
// at any position refuses, so narrowing the check to index 0 alone reddens.
// The production entry point is Discover.
func TestDiscoverRefusesRelativePluginDir(t *testing.T) {
	for _, dirs := range [][]string{
		{"relative/plugins"},
		{"/abs/plugins", "relative/plugins"},
		{"/abs/a", "/abs/b", "relative/deep"},
	} {
		fake := newFakeSystem()
		// Absolute entries carry a real candidate (a distinct provider
		// ID per directory, so no duplicate refusal fires first) so
		// discovery reaches the relative entry it must refuse with a
		// non-empty candidate set: returning that set alongside the
		// refusal would contradict the no-partial-set promise.
		planted := 0
		for _, dir := range dirs {
			if len(dir) > 0 && dir[0] == '/' {
				planted++
				name := "ax-provider-planted" + string(rune('a'+planted))
				fake.addFile(dir, name, []byte("#!/bin/sh\nexit 0\n"), fakeUID)
			}
		}
		got, err := Discover(fakeConfig(dirs...), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover(plugin_dirs=%q) returned %d candidates alongside a refusal, want no partial set", dirs, len(got))
		}
		if err == nil {
			t.Fatalf("Discover accepted relative plugin_dirs entry in %q, want invalid_config", dirs)
		}
		if code := errorCode(t, err); code != codeInvalidConfig {
			t.Fatalf("Discover(plugin_dirs=%q) code = %q, want invalid_config", dirs, code)
		}
		if detail := errorDetail(t, err); !strings.Contains(detail, "not an absolute path") {
			t.Fatalf("detail does not name the absolute-path refusal: %q", detail)
		}
	}
}

// TestDiscoverRefusesPartialReads proves a failed or partial read is never
// a legitimate absence: each seam failure aborts discovery with
// local_precondition_failed rather than a narrowed candidate set.
func TestDiscoverRefusesPartialReads(t *testing.T) {
	t.Run("unreadable directory", func(t *testing.T) {
		fake := newFakeSystem()
		fake.readDirErr["/plugins"] = errors.New("permission denied")
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if code := errorCode(t, err); code != codeLocalPrecondition {
			t.Fatalf("code = %q, want local_precondition_failed", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "cannot list") {
			t.Fatalf("detail does not name the list refusal: %q", detail)
		}
	})
	t.Run("unresolvable symlink", func(t *testing.T) {
		fake := newFakeSystem()
		fake.entries["/plugins"] = []string{"ax-provider-dangling"}
		fake.canonErr["/plugins/ax-provider-dangling"] = errors.New("no such file")
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if code := errorCode(t, err); code != codeLocalPrecondition {
			t.Fatalf("code = %q, want local_precondition_failed", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "cannot resolve") {
			t.Fatalf("detail does not name the resolve refusal: %q", detail)
		}
	})
	t.Run("uninspectable target", func(t *testing.T) {
		fake := newFakeSystem()
		fake.entries["/plugins"] = []string{"ax-provider-foo"}
		fake.canon["/plugins/ax-provider-foo"] = "/opt/real/foo"
		fake.inspectErr["/opt/real/foo"] = errors.New("stale handle")
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if code := errorCode(t, err); code != codeLocalPrecondition {
			t.Fatalf("code = %q, want local_precondition_failed", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "cannot inspect") {
			t.Fatalf("detail does not name the inspect refusal: %q", detail)
		}
	})
	t.Run("undigestible target", func(t *testing.T) {
		fake := newFakeSystem()
		fake.entries["/plugins"] = []string{"ax-provider-foo"}
		fake.canon["/plugins/ax-provider-foo"] = "/opt/real/foo"
		fake.files["/opt/real/foo"] = fakeFile{content: []byte("x"), uid: fakeUID, regular: true}
		fake.contentErr["/opt/real/foo"] = errors.New("input/output error")
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if code := errorCode(t, err); code != codeLocalPrecondition {
			t.Fatalf("code = %q, want local_precondition_failed", code)
		} else if detail := errorDetail(t, err); !strings.Contains(detail, "cannot digest") {
			t.Fatalf("detail does not name the digest refusal: %q", detail)
		}
	})
	t.Run("cause is preserved", func(t *testing.T) {
		fake := newFakeSystem()
		cause := errors.New("disk gone")
		fake.readDirErr["/plugins"] = cause
		got, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
		}
		if !errors.Is(err, cause) {
			t.Fatalf("precondition error does not wrap its cause: %v", err)
		}
	})
}

// TestBuiltinCandidatesCarryNoTrustFacts proves builtins are adapters, not
// executables: every trust accessor reports absence.
func TestBuiltinCandidatesCarryNoTrustFacts(t *testing.T) {
	got, err := Discover(fakeConfig(), fakeOwner(), newFakeSystem())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) != len(builtinOrder) {
		t.Fatalf("builtin count = %d, want %d", len(got), len(builtinOrder))
	}
	for _, candidate := range got {
		if candidate.Kind() != KindBuiltin {
			t.Fatalf("candidate %q kind = %q, want builtin", candidate.ID(), candidate.Kind())
		}
		if _, ok := candidate.SourcePath(); ok {
			t.Fatalf("builtin %q reports a source path", candidate.ID())
		}
		if _, ok := candidate.CanonicalPath(); ok {
			t.Fatalf("builtin %q reports a canonical path", candidate.ID())
		}
		if _, ok := candidate.Digest(); ok {
			t.Fatalf("builtin %q reports a digest", candidate.ID())
		}
		if _, ok := candidate.Owner(); ok {
			t.Fatalf("builtin %q reports an owner", candidate.ID())
		}
	}
}
