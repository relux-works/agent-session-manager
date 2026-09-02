package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// homeEnvironmentName is the exact variable os.UserHomeDir consults on the
// host. Tests set it so the capture inside OSInputs succeeds or fails through
// production's own lookup. Nothing here assigns Inputs.homeDirError or
// Inputs.HomeDir directly: a case that mints the state it claims to observe
// cannot witness whether production ever reaches that state.
func homeEnvironmentName() string {
	switch runtime.GOOS {
	case "windows":
		return "USERPROFILE"
	case "plan9":
		return "home"
	default:
		return "HOME"
	}
}

// homeDerivedPlatformDefaults is the Section 3.2 layout for the classes whose
// platform default is derived from the captured user home. It is spelled out
// independently of resolvePlatformDefault so the assertion compares production
// against the specification rather than against itself. Windows derives no
// default from the user home - it uses APPDATA and LOCALAPPDATA - so a
// home-capture case has nothing to pin there and says so instead of passing
// vacuously.
// homeDrivenPlatforms are the AX platform lanes whose Section 3.2 defaults are
// derived from the captured user home AND whose rendered separator matches the
// host, so a real LoadOS call from this host exercises the real capture for
// that lane rather than a fixture. os.UserHomeDir reads $HOME on both unix
// hosts, so all three home-derived lanes are drivable there. Windows derives
// no default from the user home at all, and a Windows host cannot render
// another lane's separators, so no lane is drivable there and the case says so
// instead of passing vacuously.
func homeDrivenPlatforms(t *testing.T) []scalar.Platform {
	t.Helper()
	switch runtime.GOOS {
	case "darwin", "linux":
		return []scalar.Platform{scalar.PlatformMacOS, scalar.PlatformLinux, scalar.PlatformWSL2}
	default:
		t.Skipf("GOOS=%s cannot drive a home-derived platform default at the real entry", runtime.GOOS)
		return nil
	}
}

func homeDerivedPlatformDefaults(t *testing.T, platform scalar.Platform, home string) map[PathClass]string {
	t.Helper()
	switch platform {
	case scalar.PlatformMacOS:
		return map[PathClass]string{
			ConfigFile: filepath.Join(home, ".config", "ax", "config.toml"),
			DataRoot:   filepath.Join(home, "Library", "Application Support", "ax"),
			StateRoot:  filepath.Join(home, "Library", "Application Support", "ax", "state"),
			CacheRoot:  filepath.Join(home, "Library", "Caches", "ax"),
		}
	case scalar.PlatformLinux, scalar.PlatformWSL2:
		return map[PathClass]string{
			ConfigFile: filepath.Join(home, ".config", "ax", "config.toml"),
			DataRoot:   filepath.Join(home, ".local", "share", "ax"),
			StateRoot:  filepath.Join(home, ".local", "state", "ax"),
			CacheRoot:  filepath.Join(home, ".cache", "ax"),
		}
	default:
		t.Skipf("platform %s derives no path default from the user home", platform)
		return nil
	}
}

// clearAmbientPathEnvironment removes every AX_* override and every
// home-derived XDG base directory so the platform-default layer is the one
// under test, and pins the single base directory that is not home-derived.
// XDG_RUNTIME_DIR has no home fallback on Linux and WSL2, so leaving it unset
// would refuse the runtime root before any home-derived class is reached and
// the home evidence would never be exercised.
func clearAmbientPathEnvironment(t *testing.T, platform scalar.Platform) {
	t.Helper()
	for _, specification := range OverrideRegistry() {
		t.Setenv(specification.Environment, "")
	}
	for _, name := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		t.Setenv(name, "")
	}
	if platform == scalar.PlatformLinux || platform == scalar.PlatformWSL2 {
		t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
		return
	}
	t.Setenv("XDG_RUNTIME_DIR", "")
}

// flattenErrorTree walks wrapped and joined causes so a case can inspect every
// error that actually reached the caller.
func flattenErrorTree(err error) []error {
	if err == nil {
		return nil
	}
	flattened := []error{err}
	switch unwrapped := err.(type) {
	case interface{ Unwrap() error }:
		flattened = append(flattened, flattenErrorTree(unwrapped.Unwrap())...)
	case interface{ Unwrap() []error }:
		for _, cause := range unwrapped.Unwrap() {
			flattened = append(flattened, flattenErrorTree(cause)...)
		}
	}
	return flattened
}

// carriesCauseMessage reports whether some cause in the tree renders exactly
// the supplied message. os.UserHomeDir constructs a fresh error on every call,
// so identity comparison is impossible and the rendered message is the only
// stable evidence that the captured cause - and not a substitute minted
// somewhere else - reached the caller.
func carriesCauseMessage(err error, message string) bool {
	for _, cause := range flattenErrorTree(err) {
		if cause.Error() == message {
			return true
		}
	}
	return false
}

// overridesExcept supplies a real, admissible value for every registry class
// except the one under test, so exactly one class falls through to the
// platform-default layer. Without this the first home-derived class to resolve
// would mask every later one, and a cause dropped at any single class would
// never be observed.
func overridesExcept(t *testing.T, platform scalar.Platform, exempt PathClass) Overrides {
	t.Helper()
	root := t.TempDir()
	overrides := make(Overrides, len(OverrideRegistry()))
	for _, specification := range OverrideRegistry() {
		if specification.Class == exempt {
			continue
		}
		value := filepath.Join(root, string(specification.Class))
		if specification.Class == ConfigFile {
			value += ".toml"
			if err := os.WriteFile(value, minimalValidConfig(platform), 0o600); err != nil {
				t.Fatal(err)
			}
		} else if err := os.Mkdir(value, 0o700); err != nil {
			t.Fatal(err)
		}
		overrides[specification.Class] = value
	}
	return overrides
}

// TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass pins the
// deferred half of the OSInputs home capture at the real entry. OSInputs no
// longer refuses when os.UserHomeDir fails, so the only thing that keeps the
// operator's cause reachable is the captured error travelling into the
// platform-default refusal. The failure is produced the way production
// produces it, by clearing the exact variable os.UserHomeDir reads, and the
// expected cause is the one a real os.UserHomeDir call returns in the same
// process environment - nothing assigns the unexported field.
//
// The two loops are the narrowing direction. Every home-derived class of every
// drivable lane is isolated in turn, so dropping the cause at one site - not
// only at the capture - is observable. A class or lane added to the Section 3.2
// home-derived layout is covered without editing this case.
func TestLoadOSCarriesTheRealUserHomeFailureAtEveryHomeDerivedClass(t *testing.T) {
	for _, platform := range homeDrivenPlatforms(t) {
		t.Run(platform.String(), func(t *testing.T) {
			homeDerived := homeDerivedPlatformDefaults(t, platform, t.TempDir())
			if len(homeDerived) == 0 {
				t.Fatal("the home-derived class set is empty; this lane would pin nothing")
			}
			for class := range homeDerived {
				t.Run(string(class), func(t *testing.T) {
					clearAmbientPathEnvironment(t, platform)
					overrides := overridesExcept(t, platform, class)
					t.Setenv(homeEnvironmentName(), "")

					_, homeFailure := os.UserHomeDir()
					if homeFailure == nil {
						t.Fatalf("os.UserHomeDir() succeeded with %s cleared; this case no longer reaches the capture failure", homeEnvironmentName())
					}

					_, err := LoadOS(platform, overrides)
					if !errors.Is(err, ErrPlatformDefaultUnavailable) {
						t.Fatalf("LoadOS(no user home, %s defaulted) error = %v, want ErrPlatformDefaultUnavailable", class, err)
					}
					var refusal *Error
					if !errors.As(err, &refusal) {
						t.Fatalf("LoadOS(no user home, %s defaulted) refusal = %#v, want a *config.Error", class, err)
					}
					if refusal.Class != class || refusal.Source != SourcePlatformDefault {
						t.Fatalf("LoadOS(no user home, %s defaulted) refused class %q from %q", class, refusal.Class, refusal.Source)
					}
					if !carriesCauseMessage(err, homeFailure.Error()) {
						t.Fatalf("LoadOS(no user home, %s defaulted) dropped the captured os.UserHomeDir cause %q; error = %v", class, homeFailure, err)
					}
					if carriesCauseMessage(errors.New(refusal.Error()), homeFailure.Error()) {
						t.Fatalf("LoadOS(no user home, %s defaulted) rendered the raw OS cause in its message: %q", class, refusal.Error())
					}
				})
			}
		})
	}
}

// TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome pins the
// succeeding half of the same capture. Every other LoadOS case overrides all
// five path classes, so none of them would notice a capture that returned an
// empty or constant home. Two distinct real homes are resolved in turn for
// every drivable lane: one home alone cannot distinguish "the captured home
// reached the defaults" from "the defaults were derived from something else
// that happened to match".
func TestLoadOSDerivesPlatformDefaultsFromTheRealCapturedUserHome(t *testing.T) {
	for _, platform := range homeDrivenPlatforms(t) {
		t.Run(platform.String(), func(t *testing.T) {
			clearAmbientPathEnvironment(t, platform)

			first, second := t.TempDir(), t.TempDir()
			if first == second {
				t.Fatal("the two home roots must differ for this case to pin anything")
			}
			for _, home := range []string{first, second} {
				t.Setenv(homeEnvironmentName(), home)
				captured, err := os.UserHomeDir()
				if err != nil || captured != home {
					t.Fatalf("os.UserHomeDir() = %q, %v, want %q; the environment no longer drives the capture", captured, err, home)
				}
				want := homeDerivedPlatformDefaults(t, platform, home)
				if err := os.MkdirAll(filepath.Dir(want[ConfigFile]), 0o700); err != nil {
					t.Fatal(err)
				}

				snapshot, err := LoadOS(platform, nil)
				if err != nil {
					t.Fatalf("LoadOS(home %q) error = %v", home, err)
				}
				for class, expected := range want {
					assertResolvedPath(t, snapshot.Paths(), class, expected, SourcePlatformDefault)
				}
			}
		})
	}
}
