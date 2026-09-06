package secconftest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Capability is one platform capability a test may need. Probe reports
// whether it holds here and now; StrictGOOS lists the GOOS values
// where absence is a failure rather than a skip. A skip asserts "this
// cannot run here": the probe must really attempt the capability, and
// an unexpected skip on a strict platform must fail rather than pass
// quietly.
type Capability struct {
	// ID is the stable slug naming the capability in reports.
	ID string
	// Detail explains what the probe attempts.
	Detail string
	// Probe attempts the capability and reports availability.
	Probe func() (ok bool, reason string)
	// StrictGOOS names platforms where unavailability fails the test.
	StrictGOOS []string
}

// Verdict is the outcome of requiring one capability.
type Verdict struct {
	ID      string
	Skipped bool
	Failed  bool
	Reason  string
}

// decision is the testable core of Require: pass, skip, or fail for a
// probed capability. It carries no testing handle so the table test
// can drive every arm, including the failing ones, without failing
// the suite.
type decision int

const (
	decisionPass decision = iota
	decisionSkip
	decisionFail
)

func decide(capability Capability, ok bool) decision {
	if capability.ID == "" || capability.Probe == nil {
		return decisionFail
	}
	if ok {
		return decisionPass
	}
	if strictOn(capability) {
		return decisionFail
	}
	return decisionSkip
}

// SymlinkCapability probes symlink creation and Lstat identification
// in a fresh temporary directory. Symlink-escape commit tests require
// it; Windows hosts without privilege report unavailable.
func SymlinkCapability() Capability {
	return Capability{
		ID:     "symlink",
		Detail: "create a symlink in a temp dir and identify it with Lstat",
		Probe: func() (bool, string) {
			dir, err := os.MkdirTemp("", "secconftest-symlink")
			if err != nil {
				return false, fmt.Sprintf("temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(dir) }()
			target := filepath.Join(dir, "target")
			link := filepath.Join(dir, "link")
			if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
				return false, fmt.Sprintf("write target: %v", err)
			}
			if err := os.Symlink(target, link); err != nil {
				return false, fmt.Sprintf("symlink: %v", err)
			}
			info, err := os.Lstat(link)
			if err != nil {
				return false, fmt.Sprintf("lstat: %v", err)
			}
			if info.Mode()&os.ModeSymlink == 0 {
				return false, "created link is not a symlink"
			}
			return true, "symlink create+identify works"
		},
		StrictGOOS: []string{"linux", "darwin"},
	}
}

// FifoCapability probes named-pipe creation: it builds a real FIFO
// in a fresh temporary directory and identifies it with Lstat. It is
// unavailable on Windows by construction, so Windows is not strict:
// the special-file gate there is covered by other means, stated on
// the callers. A probe that returned true without creating anything
// would advertise an unbuilt capability; TestSkipFifoFixtureExchangesBytes
// drives a second FIFO through the same helper so the probe cannot
// pass while fixtures cannot build.
func FifoCapability() Capability {
	return Capability{
		ID:     "fifo",
		Detail: "create a named pipe in a temp dir and identify it with Lstat",
		Probe: func() (bool, string) {
			dir, err := os.MkdirTemp("", "secconftest-fifo")
			if err != nil {
				return false, fmt.Sprintf("temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(dir) }()
			path, err := makeFifo(dir, "probe")
			if err != nil {
				return false, fmt.Sprintf("mkfifo: %v", err)
			}
			info, err := os.Lstat(path)
			if err != nil {
				return false, fmt.Sprintf("lstat: %v", err)
			}
			if info.Mode()&os.ModeNamedPipe == 0 {
				return false, "created file is not a named pipe"
			}
			return true, "named-pipe create+identify works"
		},
		StrictGOOS: []string{"linux", "darwin"},
	}
}

// ModeBitsCapability probes whether permission bits are enforceable:
// it removes all permissions from a temp file and checks the open
// fails. Root bypasses mode bits and Windows ACLs do not express
// them, so neither is strict.
func ModeBitsCapability() Capability {
	return Capability{
		ID:     "mode-bits",
		Detail: "chmod 000 a temp file and require the open to fail",
		Probe: func() (bool, string) {
			if runtime.GOOS == "windows" {
				return false, "Windows ACLs do not express mode bits"
			}
			dir, err := os.MkdirTemp("", "secconftest-modebits")
			if err != nil {
				return false, fmt.Sprintf("temp dir: %v", err)
			}
			defer func() { _ = os.RemoveAll(dir) }()
			path := filepath.Join(dir, "guarded")
			if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
				return false, fmt.Sprintf("write: %v", err)
			}
			if err := os.Chmod(path, 0); err != nil {
				return false, fmt.Sprintf("chmod: %v", err)
			}
			file, err := os.Open(path)
			if err != nil {
				return true, "mode bits enforced"
			}
			_ = file.Close()
			return false, "open succeeded despite mode 000 (root or unenforceable bits)"
		},
	}
}

// NonRootCapability probes that the suite does not run as superuser.
// Permission-refusal vectors that need denial require it.
func NonRootCapability() Capability {
	return Capability{
		ID:     "nonroot",
		Detail: "require a non-privileged uid on unix",
		Probe: func() (bool, string) {
			if runtime.GOOS == "windows" {
				return true, "Windows: no unix superuser"
			}
			if os.Geteuid() == 0 {
				return false, "running as root: permission denials are untestable"
			}
			return true, "non-root user"
		},
	}
}

// DefaultCapabilities lists every capability this instrument probes.
func DefaultCapabilities() []Capability {
	return []Capability{SymlinkCapability(), FifoCapability(), ModeBitsCapability(), NonRootCapability()}
}

// strictOn reports whether id is strict on this host's GOOS.
func strictOn(capability Capability) bool {
	for _, goos := range capability.StrictGOOS {
		if goos == runtime.GOOS {
			return true
		}
	}
	return false
}

// report aggregates verdicts with a process-wide skip counter the
// suite logs once, so the skip count on the host is evidence rather
// than prose.
var report struct {
	mutex   sync.Mutex
	skipped []string
	failed  []string
}

func recordVerdict(verdict Verdict) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	if verdict.Skipped {
		report.skipped = append(report.skipped, verdict.ID)
	}
	if verdict.Failed {
		report.failed = append(report.failed, verdict.ID)
	}
}

// SkippedIDs reports capability IDs skipped so far, sorted.
func SkippedIDs() []string {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	out := append([]string(nil), report.skipped...)
	sort.Strings(out)
	return out
}

// requireT is the minimal testing handle Require needs. *testing.T
// satisfies it, and the wiring tests pass a fake that records Skipf
// and Fatalf without aborting the suite — so both the skip and the
// fail arms are driven through the real Require and their
// recordVerdict calls are pinned (M04/M05). A fake that never sees
// the verdict while the counters stay empty fails its test.
type requireT interface {
	Helper()
	Skipf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Require enforces one capability: it returns a passing verdict when
// the probe holds, skips the test when the probe fails on a
// non-strict platform, and fails the test when the probe fails on a
// strict platform or the probe result contradicts a strict
// expectation the caller states. Every skip and failure is recorded
// for the final count.
func Require(t requireT, capability Capability) Verdict {
	t.Helper()
	if capability.ID == "" || capability.Probe == nil {
		verdict := Verdict{ID: capability.ID, Failed: true, Reason: "capability without identity or probe is unusable"}
		recordVerdict(verdict)
		t.Fatalf("secconftest: capability %q has no identity or probe", capability.ID)
		return verdict
	}
	ok, reason := capability.Probe()
	switch decide(capability, ok) {
	case decisionPass:
		return Verdict{ID: capability.ID, Reason: reason}
	case decisionFail:
		verdict := Verdict{ID: capability.ID, Failed: true, Reason: reason}
		recordVerdict(verdict)
		t.Fatalf("secconftest: capability %q unavailable on strict platform %s: %s", capability.ID, runtime.GOOS, reason)
		return verdict
	default:
		verdict := Verdict{ID: capability.ID, Skipped: true, Reason: reason}
		recordVerdict(verdict)
		t.Skipf("secconftest: capability %q unavailable here: %s", capability.ID, reason)
		return verdict
	}
}

// Report formats the capability skip/fail counts recorded so far on
// this host. It reads the live counters: an empty suite records
// nothing and reports zeros, and every recorded skip or failure moves
// the numbers. Citing a Report line as host evidence is only honest
// for verdicts the suite really recorded; the per-test skip datum is
// the `--- SKIP` lines under `go test -v`, which reproduce without
// any counter.
func Report() string {
	report.mutex.Lock()
	skipped := append([]string(nil), report.skipped...)
	failed := append([]string(nil), report.failed...)
	report.mutex.Unlock()
	sort.Strings(skipped)
	sort.Strings(failed)
	return fmt.Sprintf("secconftest capabilities on %s: skipped=%d %q failed=%d %q",
		runtime.GOOS, len(skipped), strings.Join(skipped, ","), len(failed), strings.Join(failed, ","))
}

// snapshotReport copies the process-wide counters so a test can
// record synthetic verdicts and restore the suite state afterwards.
func snapshotReport() (skipped, failed []string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	return append([]string(nil), report.skipped...), append([]string(nil), report.failed...)
}

// restoreReport replaces the process-wide counters, undoing synthetic
// verdicts a test recorded. It exists only for test isolation.
func restoreReport(skipped, failed []string) {
	report.mutex.Lock()
	defer report.mutex.Unlock()
	report.skipped = append([]string(nil), skipped...)
	report.failed = append([]string(nil), failed...)
}

// LogReport logs the capability skip/fail counts on this host. Call
// it after the suite recorded its verdicts so the line carries real
// counters. It never skips or fails: it reports.
func LogReport(t *testing.T) {
	t.Helper()
	t.Log(Report())
}
