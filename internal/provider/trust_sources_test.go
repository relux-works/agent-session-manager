package provider

import (
	"errors"
	"strings"
	"testing"
)

// trustGateDimension is one refusal dimension of the Discover trust
// boundary: shape (regular file), owner (approved identity), name
// (provider-id grammar), or read (filesystem seam failure).
type trustGateDimension struct {
	name string
	// wantCode is the refusal code the dimension must report.
	wantCode string
	// wantDetail is a substring of the refusal detail naming the gate
	// that fired, so each cell pins its gate rather than its code class.
	wantDetail string
	// plant installs the dimension's violating fixture for source dir.
	plant func(fake *fakeSystem, dir string)
}

func trustGateDimensions() []trustGateDimension {
	return []trustGateDimension{
		{
			name:       "non-regular target",
			wantCode:   codeInvalidConfig,
			wantDetail: "not a regular file",
			plant: func(fake *fakeSystem, dir string) {
				fake.addLinkedFile(dir, "ax-provider-gate", dir+"/real/gate", []byte("x"), fakeUID, false)
			},
		},
		{
			name:       "unapproved owner",
			wantCode:   codeInvalidConfig,
			wantDetail: "does not approve",
			plant: func(fake *fakeSystem, dir string) {
				fake.addFile(dir, "ax-provider-gate", []byte("x"), foreignUID)
			},
		},
		{
			name:       "malformed name",
			wantCode:   codeInvalidConfig,
			wantDetail: "not a provider executable name",
			plant: func(fake *fakeSystem, dir string) {
				bad := "ax-provider-ABC"
				fake.entries[dir] = append(fake.entries[dir], bad)
				fake.canon[dir+"/"+bad] = dir + "/" + bad
			},
		},
		{
			name:       "unreadable target",
			wantCode:   codeLocalPrecondition,
			wantDetail: "cannot digest",
			plant: func(fake *fakeSystem, dir string) {
				source := fake.addLinkedFile(dir, "ax-provider-gate", dir+"/real/gate", []byte("x"), fakeUID, true)
				target, err := fake.Canonicalize(source)
				if err != nil {
					panic(err)
				}
				fake.contentErr[target] = errors.New("fake: input/output error")
			},
		},
	}
}

// trustGateSource is one discovery source the trust gates must cover:
// plugin_dirs[0], plugin_dirs[1], or PATH.
type trustGateSource struct {
	name string
	// configure wires the source dir holding the violating fixture and
	// returns the Discover config exercising that source.
	configure func(fake *fakeSystem, dir string) Config
}

func trustGateSources() []trustGateSource {
	return []trustGateSource{
		{
			name: "plugin_dirs[0]",
			configure: func(fake *fakeSystem, dir string) Config {
				return fakeConfig(dir)
			},
		},
		{
			name: "plugin_dirs[1]",
			configure: func(fake *fakeSystem, dir string) Config {
				// The leading directory exists but holds no candidate,
				// so the violation in the second directory is the one
				// discovery must refuse.
				fake.entries["/plugins/a"] = []string{}
				return fakeConfig("/plugins/a", dir)
			},
		},
		{
			name: "path",
			configure: func(fake *fakeSystem, dir string) Config {
				fake.pathDirs = []string{dir}
				cfg := fakeConfig()
				cfg.AllowPathPlugins = true
				return cfg
			},
		},
	}
}

// TestDiscoverEnforcesTrustGatesAcrossSources narrows every Discover trust
// gate across every discovery source: {plugin_dirs[0], plugin_dirs[1],
// path} x {shape, owner, name, read}. A gate that holds only for the
// configured plugin directory leaves the PATH source — the most
// attacker-exposed source in this section — unchecked. The production
// entry point is Discover.
func TestDiscoverEnforcesTrustGatesAcrossSources(t *testing.T) {
	sources := trustGateSources()
	dimensions := trustGateDimensions()
	total := len(sources) * len(dimensions)
	passed := 0
	for _, source := range sources {
		for _, dimension := range dimensions {
			name := source.name + "/" + dimension.name
			t.Run(name, func(t *testing.T) {
				dir := "/plugins/gate"
				if source.name == "plugin_dirs[1]" {
					dir = "/plugins/b"
				} else if source.name == "path" {
					dir = "/pathbin"
				}
				fake := newFakeSystem()
				dimension.plant(fake, dir)
				got, err := Discover(source.configure(fake, dir), fakeOwner(), fake)
				if got != nil {
					t.Fatalf("Discover returned %d candidates alongside a refusal, want no partial set", len(got))
				}
				if err == nil {
					t.Fatalf("Discover admitted %s via %s, want %s", dimension.name, source.name, dimension.wantCode)
				}
				if code := errorCode(t, err); code != dimension.wantCode {
					t.Fatalf("Discover(%s via %s) code = %q, want %q", dimension.name, source.name, code, dimension.wantCode)
				}
				if detail := errorDetail(t, err); !strings.Contains(detail, dimension.wantDetail) {
					t.Fatalf("Discover(%s via %s) detail does not name the gate: %q", dimension.name, source.name, detail)
				}
				passed++
			})
		}
	}
	t.Logf("trust-gate coverage: %d/%d source x dimension pairs refused", passed, total)
	if passed != total {
		t.Fatalf("trust-gate coverage = %d/%d, want %d/%d", passed, total, total, total)
	}
}

// TestDiscoverRefusesRelativePATHDir proves a relative PATH entry fails
// closed instead of resolving against the process working directory: the
// set of discoverable providers must not depend on where the process was
// started. The plugin_dirs half of this gate is pinned by
// TestDiscoverRefusesRelativePluginDir; this case pins the PATH half at
// the same production entry point, Discover.
func TestDiscoverRefusesRelativePATHDir(t *testing.T) {
	for _, pathDirs := range [][]string{
		{"."},
		{"/abs/bin", "relative/bin"},
	} {
		fake := newFakeSystem()
		// Absolute entries resolve to an empty directory so discovery
		// reaches the relative entry it must refuse.
		fake.entries["/abs/bin"] = []string{}
		fake.pathDirs = pathDirs
		cfg := fakeConfig()
		cfg.AllowPathPlugins = true
		got, err := Discover(cfg, fakeOwner(), fake)
		if got != nil {
			t.Fatalf("Discover(PATH=%q) returned %d candidates alongside a refusal, want no partial set", pathDirs, len(got))
		}
		if err == nil {
			t.Fatalf("Discover accepted relative PATH entry in %q, want invalid_config", pathDirs)
		}
		if code := errorCode(t, err); code != codeInvalidConfig {
			t.Fatalf("Discover(PATH=%q) code = %q, want invalid_config", pathDirs, code)
		}
		if detail := errorDetail(t, err); !strings.Contains(detail, "not an absolute path") {
			t.Fatalf("detail does not name the absolute-path refusal: %q", detail)
		}
	}
}
