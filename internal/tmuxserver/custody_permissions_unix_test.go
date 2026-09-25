//go:build !windows

package tmuxserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

type custodyPermissionClass struct {
	name   string
	base   os.FileMode
	code   string
	detail string
	refuse func(os.FileMode) bool
}

type custodyPermissionBit struct {
	name string
	mode os.FileMode
}

var custodyPermissionBits = []custodyPermissionBit{
	{name: "group_read", mode: 0o040},
	{name: "group_write", mode: 0o020},
	{name: "group_execute", mode: 0o010},
	{name: "other_read", mode: 0o004},
	{name: "other_write", mode: 0o002},
	{name: "other_execute", mode: 0o001},
}

func TestCustodyPermissionBitCensus(t *testing.T) {
	classes := []custodyPermissionClass{
		{
			name: "socket_leaf", base: 0o600, code: "tmux_unsafe_socket_path", detail: "socket permissions",
			refuse: func(os.FileMode) bool { return true },
		},
		{
			name: "runtime_tmux_dir", base: 0o700, code: "tmux_unsafe_runtime_dir", detail: "runtime mode",
			refuse: func(os.FileMode) bool { return true },
		},
		{
			name: "root", base: 0o700, code: "tmux_unsafe_socket_path", detail: "socket root",
			refuse: func(os.FileMode) bool { return true },
		},
		{
			name: "ancestor", base: 0o700, code: "tmux_unsafe_socket_path", detail: "socket ancestor",
			// SPEC §3.2 requires rejecting unsafe ancestor permissions. A
			// non-sticky ancestor may expose read/execute bits for path
			// traversal, but group/other write permits socket substitution.
			refuse: func(bit os.FileMode) bool { return bit&0o022 != 0 },
		},
	}
	entries := []string{"Probe", "Spawn", "Execute"}

	for _, class := range classes {
		class := class
		for _, bit := range custodyPermissionBits {
			bit := bit
			for _, entry := range entries {
				entry := entry
				t.Run(class.name+"/"+bit.name+"/"+entry, func(t *testing.T) {
					var root, socket string
					var fx *lxFixture
					var effectCount int
					var err error
					switch entry {
					case "Probe", "Spawn":
						root, socket = lxProbeRoot(t)
					case "Execute":
						fx = newLifecycleFixture(t, fullLifecycleAdmitted())
						root, socket = fx.root, fx.socket
					default:
						t.Fatalf("unhandled entry %q", entry)
					}

					applyCustodyPermissionMode(t, class.name, root, socket, class.base|bit.mode)

					err, effectCount = runCustodyPermissionEntry(t, entry, root, socket, fx)

					if class.refuse(bit.mode) {
						assertCustodyPermissionRefusal(t, err, class.code, class.detail)
						if effectCount != 0 {
							t.Fatalf("%s performed %d effect(s) before custody refusal", entry, effectCount)
						}
						return
					}
					if err != nil {
						t.Fatalf("SPEC §3.2 admitted ancestor permission %04o was refused by %s: %v", bit.mode, entry, err)
					}
					if effectCount == 0 {
						t.Fatalf("%s did not reach its production effect after the admitted permission", entry)
					}
				})
			}
		}
	}
}

func TestCustodyPermissionSocketLeafExactModes(t *testing.T) {
	modes := []struct {
		name string
		mode os.FileMode
	}{
		{name: "0602", mode: 0o602},
		{name: "0604", mode: 0o604},
		{name: "0606", mode: 0o606},
		{name: "0620", mode: 0o620},
	}
	entries := []string{"Probe", "Spawn", "Execute"}
	for _, mode := range modes {
		mode := mode
		for _, entry := range entries {
			entry := entry
			t.Run(mode.name+"/"+entry, func(t *testing.T) {
				var root, socket string
				var fx *lxFixture
				switch entry {
				case "Probe", "Spawn":
					root, socket = lxProbeRoot(t)
				case "Execute":
					fx = newLifecycleFixture(t, fullLifecycleAdmitted())
					root, socket = fx.root, fx.socket
				}
				applyCustodyPermissionMode(t, "socket_leaf", root, socket, mode.mode)

				err, effects := runCustodyPermissionEntry(t, entry, root, socket, fx)
				assertCustodyPermissionRefusal(t, err, "tmux_unsafe_socket_path", "socket permissions")
				if effects != 0 {
					t.Fatalf("%s performed %d effect(s) before refusing socket mode %s", entry, effects, mode.name)
				}
			})
		}
	}
}

func TestCustodyPermissionBitCensusAdmitsStickyAncestor(t *testing.T) {
	for _, entry := range []string{"Probe", "Spawn", "Execute"} {
		entry := entry
		t.Run(entry, func(t *testing.T) {
			var root, socket string
			var fx *lxFixture
			switch entry {
			case "Probe", "Spawn":
				root, socket = lxProbeRoot(t)
			case "Execute":
				fx = newLifecycleFixture(t, fullLifecycleAdmitted())
				root, socket = fx.root, fx.socket
			}
			ancestor := filepath.Dir(root)
			if err := os.Chmod(ancestor, 0o777|os.ModeSticky); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(ancestor, 0o700) })
			err, effects := runCustodyPermissionEntry(t, entry, root, socket, fx)
			if err != nil {
				t.Fatalf("SPEC §3.2 sticky ancestor mode 01777 was refused by %s: %v", entry, err)
			}
			if effects == 0 {
				t.Fatalf("%s did not reach its production effect with a sticky ancestor", entry)
			}
		})
	}
}

func runCustodyPermissionEntry(t *testing.T, entry, root, socket string, fx *lxFixture) (error, int) {
	t.Helper()
	switch entry {
	case "Probe":
		dialer := &fakeDialer{outcomes: []DialOutcome{DialStale}}
		prober := ServerProber{
			Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer,
			Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil },
		}
		_, err := prober.Probe(socket)
		return err, len(dialer.calls)
	case "Spawn":
		runner := newFakeRunner().queue("new-session", "", 0)
		spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
		argv, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
		if err != nil {
			t.Fatal(err)
		}
		err = spawner.Spawn(socket, argv)
		return err, runner.callCount()
	case "Execute":
		if fx == nil {
			t.Fatal("Execute fixture is required")
		}
		recordBinding(t, fx, lxDigestA)
		fx.runner.queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		})
		return err, fx.runner.callCount()
	default:
		t.Fatalf("unhandled entry %q", entry)
		return nil, 0
	}
}

func applyCustodyPermissionMode(t *testing.T, class, root, socket string, mode os.FileMode) {
	t.Helper()
	var path string
	switch class {
	case "socket_leaf":
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = listener.Close() })
		path = socket
	case "runtime_tmux_dir":
		path = filepath.Join(root, RuntimeDirName)
	case "root":
		path = root
	case "ancestor":
		path = filepath.Dir(root)
	default:
		t.Fatalf("unknown custody component %q", class)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, classModeBaseline(class)) })
}

func classModeBaseline(class string) os.FileMode {
	switch class {
	case "socket_leaf":
		return 0o600
	case "runtime_tmux_dir":
		return 0o700
	case "root", "ancestor":
		return 0o700
	default:
		return 0
	}
}

func assertCustodyPermissionRefusal(t *testing.T, err error, code, detail string) {
	t.Helper()
	requireLocalCode(t, err, code, detail)
	want := "tmux server refused: " + code + " at " + detail
	if err.Error() != want {
		t.Fatalf("refusal message = %q, want literal %q", err.Error(), want)
	}
}

type custodyProjectedFileInfo struct {
	os.FileInfo
	mode os.FileMode
	uid  *uint32
}

func (info custodyProjectedFileInfo) Mode() os.FileMode { return info.mode }

func (info custodyProjectedFileInfo) Sys() any {
	if info.uid == nil {
		return info.FileInfo.Sys()
	}
	stat, ok := info.FileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return info.FileInfo.Sys()
	}
	projected := *stat
	projected.Uid = *info.uid
	return &projected
}

type custodyPathComponent struct {
	name   string
	class  string
	path   string
	code   string
	detail string
}

func custodyPathComponents(root, socket string) []custodyPathComponent {
	components := []custodyPathComponent{
		{name: "depth_00_socket_leaf", class: "socket_leaf", path: filepath.Clean(socket), code: "tmux_unsafe_socket_path", detail: "socket permissions"},
		{name: "depth_01_runtime_tmux_dir", class: "runtime_tmux_dir", path: filepath.Join(root, RuntimeDirName), code: "tmux_unsafe_runtime_dir", detail: "runtime mode"},
		{name: "depth_02_runtime_root", class: "root", path: filepath.Clean(root), code: "tmux_unsafe_socket_path", detail: "socket root"},
	}
	depth := 3
	for ancestor := filepath.Clean(filepath.Dir(root)); ; ancestor = filepath.Dir(ancestor) {
		components = append(components, custodyPathComponent{
			name: fmt.Sprintf("depth_%02d_ancestor", depth), class: "ancestor",
			path: ancestor, code: "tmux_unsafe_socket_path", detail: "socket ancestor",
		})
		if isFilesystemRoot(ancestor) {
			break
		}
		depth++
	}
	return components
}

func newCustodyOracleEntryFixture(t *testing.T, entry string) (string, string, *lxFixture) {
	root, socket, fx, _ := newCustodyOracleEntryFixtureAtDepth(t, entry, 0)
	return root, socket, fx
}

func newCustodyOracleEntryFixtureAtDepth(t *testing.T, entry string, extraDepth int) (string, string, *lxFixture, []string) {
	t.Helper()
	var root, socket string
	var fx *lxFixture
	if entry == "Execute" {
		fx = newLifecycleFixture(t, fullLifecycleAdmitted())
		recordBinding(t, fx, lxDigestA)
		root, socket = fx.root, fx.socket
	} else {
		root, socket = lxProbeRoot(t)
	}
	return extendCustodyOracleFixtureAtDepth(t, root, socket, fx, extraDepth)
}

func newCustodyOracleEntryFixtureAtDepthUnderBase(t *testing.T, entry string, extraDepth int, caseBase string) (string, string, *lxFixture, []string) {
	t.Helper()
	if err := os.Mkdir(caseBase, 0o700); err != nil {
		t.Fatalf("create custody case base: %v", err)
	}
	root := filepath.Join(caseBase, "r")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create custody runtime root: %v", err)
	}
	var socket string
	var fx *lxFixture
	if entry == "Execute" {
		fx = newLifecycleFixtureAtRoot(t, fullLifecycleAdmitted(), root)
		recordBinding(t, fx, lxDigestA)
		root, socket = fx.root, fx.socket
	} else {
		runtimeDir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
		if err != nil {
			t.Fatalf("create custody runtime directory: %v", err)
		}
		socket = SocketPath(runtimeDir)
	}
	return extendCustodyOracleFixtureAtDepth(t, root, socket, fx, extraDepth)
}

func extendCustodyOracleFixtureAtDepth(t *testing.T, root, socket string, fx *lxFixture, extraDepth int) (string, string, *lxFixture, []string) {
	t.Helper()
	if extraDepth == 0 {
		return root, socket, fx, nil
	}
	if extraDepth < 0 {
		t.Fatalf("extra custody depth = %d, want a non-negative value", extraDepth)
	}

	base := filepath.Dir(root)
	nested := make([]string, 0, extraDepth)
	for depth := 1; depth <= extraDepth; depth++ {
		base = filepath.Join(base, "x")
		if err := os.Mkdir(base, 0o700); err != nil {
			t.Fatalf("create custody depth %d: %v", depth, err)
		}
		nested = append(nested, base)
	}
	root = filepath.Join(base, "r")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatalf("create custody runtime root: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("resolve custody runtime root: %v", err)
	}
	root = resolvedRoot
	runtimeDir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("create custody runtime directory at extra depth %d: %v", extraDepth, err)
	}
	socket = SocketPath(runtimeDir)
	if fx != nil {
		fx.root = root
		fx.runtime = runtimeDir
		fx.socket = socket
		fx.lc.Root = root
		fx.lc.RuntimeDir = runtimeDir
	}
	return root, socket, fx, nested
}

func installSyntheticCustodySocket(t *testing.T, target string) {
	t.Helper()
	backing, err := os.CreateTemp(t.TempDir(), "socket-identity-")
	if err != nil {
		t.Fatal(err)
	}
	if err := backing.Close(); err != nil {
		t.Fatal(err)
	}
	backingInfo, err := os.Lstat(backing.Name())
	if err != nil {
		t.Fatal(err)
	}
	synthetic := custodyProjectedFileInfo{
		FileInfo: backingInfo,
		mode:     backingInfo.Mode()&^os.ModeType | os.ModeSocket | 0o600,
	}
	previousLstat := lstatCustodySocket
	lstatCustodySocket = func(path string) (os.FileInfo, error) {
		if filepath.Clean(path) == filepath.Clean(target) {
			return synthetic, nil
		}
		return os.Lstat(path)
	}
	t.Cleanup(func() { lstatCustodySocket = previousLstat })
	previousIdentity := sameCustodyIdentity
	sameCustodyIdentity = func(left, right os.FileInfo) bool {
		if _, isSynthetic := left.(custodyProjectedFileInfo); isSynthetic {
			leftStat, leftOK := left.Sys().(*syscall.Stat_t)
			rightStat, rightOK := right.Sys().(*syscall.Stat_t)
			return leftOK && rightOK && leftStat.Dev == rightStat.Dev && leftStat.Ino == rightStat.Ino
		}
		return os.SameFile(left, right)
	}
	t.Cleanup(func() { sameCustodyIdentity = previousIdentity })
}

// TestCustodyModeOracleAtProductionEntries compares every Unix low-12-bit
// mode against the §3.2 oracle at Probe, Spawn, and Execute. The test uses
// a mode projection over accessible private directories and a synthetic
// socket FileInfo backed by an ordinary file, so no real socket is bound
// or connected and no tmux process is started.
func TestCustodyModeOracleAtProductionEntries(t *testing.T) {
	for _, fixture := range []struct {
		name       string
		extraDepth int
	}{
		{name: "current_path_depth", extraDepth: 0},
		{name: "extra_nested_depth_08", extraDepth: 8},
	} {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			runCustodyModeOracleAtProductionEntries(t, fixture.extraDepth)
		})
	}
}

func runCustodyModeOracleAtProductionEntries(t *testing.T, extraDepth int) {
	t.Helper()
	classes := []string{"socket_leaf", "runtime_tmux_dir", "root", "ancestor"}
	entries := []string{"Probe", "Spawn", "Execute"}
	const modeCount = 1 << 12

	for _, class := range classes {
		class := class
		t.Run(class, func(t *testing.T) {
			for _, entry := range entries {
				entry := entry
				t.Run(entry, func(t *testing.T) {
					root, socket, fx, _ := newCustodyOracleEntryFixtureAtDepth(t, entry, extraDepth)
					components := custodyPathComponents(root, socket)
					ancestorCount := 0
					for _, component := range components {
						if component.class == "ancestor" {
							ancestorCount++
						}
					}
					if ancestorCount < 3 {
						t.Fatalf("fixture has %d ancestor positions, want at least 3", ancestorCount)
					}
					if class == "socket_leaf" {
						installSyntheticCustodySocket(t, socket)
					}

					for _, component := range components {
						if component.class != class {
							continue
						}
						component := component
						t.Run(component.name, func(t *testing.T) {
							var raw uint16
							projectionHits := 0
							previousProjection := custodyModeProjection
							custodyModeProjection = func(path string, current os.FileMode) os.FileMode {
								if filepath.Clean(path) == component.path {
									projectionHits++
									return projectUnixLowMode(current, raw)
								}
								return current
							}
							t.Cleanup(func() { custodyModeProjection = previousProjection })

							for rawValue := 0; rawValue < modeCount; rawValue++ {
								raw = uint16(rawValue)
								projectionHits = 0
								err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
								if specCustodyModeAdmits(component.class, raw) {
									if err != nil {
										t.Fatalf("SPEC §3.2 oracle admits %s mode %04o, but %s refused: %v", component.name, raw, entry, err)
									}
									if effects == 0 {
										t.Fatalf("%s admitted %s mode %04o but did not reach its fake next-stage effect", entry, component.name, raw)
									}
									if projectionHits == 0 {
										t.Fatalf("mode %04o did not reach %s at %s through %s", raw, component.class, component.path, entry)
									}
									continue
								}
								assertCustodyPermissionRefusal(t, err, component.code, component.detail)
								if effects != 0 {
									t.Fatalf("%s performed %d effect(s) before refusing %s mode %04o", entry, effects, component.name, raw)
								}
								if projectionHits == 0 {
									t.Fatalf("mode %04o refusal did not pass through %s at %s via %s", raw, component.class, component.path, entry)
								}
							}
						})
					}
				})
			}
		})
	}
}

// specCustodyModeAdmits is independent from the production predicates. It
// translates SPEC §3.2's 0700 directories, 0600 files, and component-wise
// custody rule into a low-12-bit oracle: only sticky exempts group/other
// writable ancestors; setuid and setgid never do.
func specCustodyModeAdmits(class string, raw uint16) bool {
	permissions := raw & 0o777
	special := raw & 0o7000
	switch class {
	case "socket_leaf":
		return permissions&0o077 == 0 && permissions&0o600 == 0o600 && special == 0
	case "runtime_tmux_dir", "root":
		return permissions == 0o700 && special == 0
	case "ancestor":
		return permissions&0o022 == 0 || special&0o1000 != 0
	default:
		return false
	}
}

func projectUnixLowMode(current os.FileMode, raw uint16) os.FileMode {
	const lowModeMask = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	mode := current &^ lowModeMask
	mode |= os.FileMode(raw & 0o777)
	if raw&0o4000 != 0 {
		mode |= os.ModeSetuid
	}
	if raw&0o2000 != 0 {
		mode |= os.ModeSetgid
	}
	if raw&0o1000 != 0 {
		mode |= os.ModeSticky
	}
	return mode
}

func runCustodyOracleEntry(t *testing.T, entry, root, socket string, fx *lxFixture) (error, int) {
	t.Helper()
	switch entry {
	case "Probe":
		dialer := &fakeDialer{outcomes: []DialOutcome{DialStale}}
		prober := ServerProber{
			Root: root, Platform: scalar.PlatformMacOS, Dialer: dialer,
			Admit: func() (RealmAdmission, error) { return RealmAdmission{}, nil },
		}
		_, err := prober.Probe(socket)
		return err, len(dialer.calls)
	case "Spawn":
		runner := newFakeRunner().queue("new-session", "", 0)
		spawner := ServerSpawner{Root: root, Platform: scalar.PlatformMacOS, Runner: runner}
		argv, err := BuildArgv(filepath.Dir(socket), socket, lxSession)
		if err != nil {
			t.Fatal(err)
		}
		err = spawner.Spawn(socket, argv)
		return err, runner.callCount()
	case "Execute":
		if fx == nil {
			t.Fatal("Execute fixture is required")
		}
		runner := newFakeRunner().queueFull("list-panes", "", "can't find window: "+lxInstance, 1)
		fx.lc.Runner = runner
		_, err := fx.lc.Execute(context.Background(), OpRequest{
			Operation: "status", Body: lxStatusBody(t, true, false, nil), Admitted: fullLifecycleAdmitted(),
		})
		return err, runner.callCount()
	default:
		t.Fatalf("unhandled production entry %q", entry)
		return nil, 0
	}
}

type custodyKindOracleCase struct {
	name string
	kind os.FileMode
}

var custodyKindOracleCases = []custodyKindOracleCase{
	{name: "directory", kind: os.ModeDir},
	{name: "regular_file", kind: 0},
	{name: "symlink", kind: os.ModeSymlink},
	{name: "fifo", kind: os.ModeNamedPipe},
	{name: "socket", kind: os.ModeSocket},
}

func projectUnixKind(current, kind os.FileMode) os.FileMode {
	return current&^os.ModeType | kind
}

func specCustodyKindAdmits(class, kind string) bool {
	if class == "socket_leaf" {
		return kind == "socket"
	}
	return kind == "directory"
}

func custodyKindRefusalDetail(class string) string {
	if class == "socket_leaf" {
		return "socket kind"
	}
	if class == "runtime_tmux_dir" {
		return "runtime mode"
	}
	if class == "root" {
		return "socket root"
	}
	return "socket ancestor"
}

func custodyOwnerRefusalDetail(class string) string {
	switch class {
	case "socket_leaf":
		return "socket ownership"
	case "runtime_tmux_dir":
		return "runtime ownership"
	default:
		return "socket root"
	}
}

// TestCustodyPathKindOwnerOracleAtProductionEntries enumerates the kind and
// owner dimensions at every path position through Probe, Spawn, and Execute.
// Metadata projections stand in for foreign owners and unreplaceable system
// ancestors; each projection must be observed by the production gate.
func TestCustodyPathKindOwnerOracleAtProductionEntries(t *testing.T) {
	for _, fixture := range []struct {
		name       string
		extraDepth int
	}{
		{name: "current_path_depth", extraDepth: 0},
		{name: "extra_nested_depth_08", extraDepth: 8},
	} {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			runCustodyPathKindOwnerOracleAtProductionEntries(t, fixture.extraDepth)
		})
	}
}

func runCustodyPathKindOwnerOracleAtProductionEntries(t *testing.T, extraDepth int) {
	t.Helper()
	entries := []string{"Probe", "Spawn", "Execute"}
	for _, entry := range entries {
		entry := entry
		t.Run(entry, func(t *testing.T) {
			root, socket, fx, _ := newCustodyOracleEntryFixtureAtDepth(t, entry, extraDepth)
			components := custodyPathComponents(root, socket)
			ancestorCount := 0
			for _, component := range components {
				if component.class == "ancestor" {
					ancestorCount++
				}
			}
			if ancestorCount < 3 {
				t.Fatalf("fixture has %d ancestor positions, want at least 3", ancestorCount)
			}
			installSyntheticCustodySocket(t, socket)

			for _, component := range components {
				component := component
				t.Run(component.name, func(t *testing.T) {
					for _, kindCase := range custodyKindOracleCases {
						kindCase := kindCase
						t.Run("kind_"+kindCase.name, func(t *testing.T) {
							projectionHits := 0
							previousProjection := custodyKindProjection
							custodyKindProjection = func(path string, current os.FileMode) os.FileMode {
								if filepath.Clean(path) == component.path {
									projectionHits++
									return projectUnixKind(current, kindCase.kind)
								}
								return current
							}
							t.Cleanup(func() { custodyKindProjection = previousProjection })

							err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
							if projectionHits == 0 {
								t.Fatalf("kind %s did not reach %s at %s through %s", kindCase.name, component.class, component.path, entry)
							}
							if specCustodyKindAdmits(component.class, kindCase.name) {
								if err != nil {
									t.Fatalf("SPEC §3.2 oracle admits %s kind at %s, but %s refused: %v", kindCase.name, component.path, entry, err)
								}
								if effects == 0 {
									t.Fatalf("%s admitted %s kind at %s but did not reach its fake next-stage effect", entry, kindCase.name, component.path)
								}
								return
							}
							assertCustodyPermissionRefusal(t, err, component.code, custodyKindRefusalDetail(component.class))
							if effects != 0 {
								t.Fatalf("%s performed %d effect(s) before refusing %s kind at %s", entry, effects, kindCase.name, component.path)
							}
						})
					}

					if component.class == "ancestor" {
						return
					}
					currentUID := uint32(effectiveUID())
					// Keep the alternate UID non-root and derive it from the
					// production euid seam so a one-member owner narrowing can
					// be planted and killed by this oracle.
					otherNonRootUID := currentUID + 100000
					owners := []struct {
						name string
						uid  uint32
					}{
						{name: "current_user", uid: currentUID},
						{name: "other_non_root", uid: otherNonRootUID},
						{name: "root", uid: 0},
					}
					for _, owner := range owners {
						owner := owner
						t.Run("owner_"+owner.name, func(t *testing.T) {
							projectionHits := 0
							previousProjection := custodyOwnerProjection
							custodyOwnerProjection = func(path string, info os.FileInfo) os.FileInfo {
								if filepath.Clean(path) != component.path {
									return info
								}
								projectionHits++
								uid := owner.uid
								return custodyProjectedFileInfo{FileInfo: info, uid: &uid}
							}
							t.Cleanup(func() { custodyOwnerProjection = previousProjection })

							err, effects := runCustodyOracleEntry(t, entry, root, socket, fx)
							if projectionHits == 0 {
								t.Fatalf("owner %s did not reach the %s gate at %s through %s", owner.name, component.class, component.path, entry)
							}
							if owner.uid == currentUID {
								if err != nil {
									t.Fatalf("SPEC §3.2 oracle admits current owner at %s, but %s refused: %v", component.path, entry, err)
								}
								if effects == 0 {
									t.Fatalf("%s admitted current owner at %s but did not reach its fake next-stage effect", entry, component.path)
								}
								return
							}
							assertCustodyPermissionRefusal(t, err, component.code, custodyOwnerRefusalDetail(component.class))
							if effects != 0 {
								t.Fatalf("%s performed %d effect(s) before refusing owner %s at %s", entry, effects, owner.name, component.path)
							}
						})
					}
				})
			}
		})
	}
}
