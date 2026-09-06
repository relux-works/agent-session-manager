package secconftest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestSkipDecideTable(t *testing.T) {
	t.Parallel()
	strict := Capability{ID: "strict-cap", Probe: func() (bool, string) { return false, "down" }, StrictGOOS: []string{runtime.GOOS}}
	lax := Capability{ID: "lax-cap", Probe: func() (bool, string) { return false, "down" }, StrictGOOS: []string{"plan9"}}
	other := Capability{ID: "other-cap", Probe: func() (bool, string) { return false, "down" }}
	if got := decide(strict, true); got != decisionPass {
		t.Errorf("available strict capability decides %v, want pass", got)
	}
	if got := decide(lax, true); got != decisionPass {
		t.Errorf("available lax capability decides %v, want pass", got)
	}
	if got := decide(strict, false); got != decisionFail {
		t.Errorf("unavailable strict capability decides %v, want fail: an unexpected skip must not pass quietly", got)
	}
	if got := decide(lax, false); got != decisionSkip {
		t.Errorf("unavailable lax capability decides %v, want skip", got)
	}
	if got := decide(other, false); got != decisionSkip {
		t.Errorf("unavailable unlisted capability decides %v, want skip", got)
	}
	if got := decide(Capability{}, false); got != decisionFail {
		t.Errorf("malformed capability decides %v, want fail", got)
	}
	if got := decide(Capability{ID: "no-probe"}, false); got != decisionFail {
		t.Errorf("probeless capability decides %v, want fail", got)
	}
	// M02: an empty ID with a healthy probe must still fail. Dropping
	// the empty-ID disjunct admits it whenever the probe holds, so
	// both polarities are pinned here.
	emptyWithProbe := Capability{ID: "", Probe: func() (bool, string) { return true, "up" }}
	if got := decide(emptyWithProbe, true); got != decisionFail {
		t.Errorf("empty-ID capability with a passing probe decides %v, want fail", got)
	}
	if got := decide(emptyWithProbe, false); got != decisionFail {
		t.Errorf("empty-ID capability with a failing probe decides %v, want fail", got)
	}
}

func TestSkipDefaultProbesTerminate(t *testing.T) {
	t.Parallel()
	capabilities := DefaultCapabilities()
	if len(capabilities) != 4 {
		t.Fatalf("DefaultCapabilities has %d entries, want 4", len(capabilities))
	}
	seen := map[string]bool{}
	for _, capability := range capabilities {
		if capability.ID == "" || capability.Probe == nil || capability.Detail == "" {
			t.Errorf("capability %+v is malformed", capability)
		}
		if seen[capability.ID] {
			t.Errorf("duplicate capability %q", capability.ID)
		}
		seen[capability.ID] = true
		ok, reason := capability.Probe()
		if reason == "" {
			t.Errorf("capability %q probed without a reason", capability.ID)
		}
		t.Logf("capability %q on %s: ok=%v (%s)", capability.ID, runtime.GOOS, ok, reason)
	}
}

func TestSkipProbesAreDeterministic(t *testing.T) {
	t.Parallel()
	for _, capability := range DefaultCapabilities() {
		first, _ := capability.Probe()
		second, _ := capability.Probe()
		if first != second {
			t.Errorf("capability %q probe flips: %v then %v", capability.ID, first, second)
		}
	}
}

func TestSkipFifoFixtureExchangesBytes(t *testing.T) {
	t.Parallel()
	// The fifo probe gates this test: where the platform cannot
	// provide named pipes the test skips (Windows is not strict)
	// instead of passing over an unbuilt fixture.
	Require(t, FifoCapability())
	dir := t.TempDir()
	path, err := makeFifo(dir, "exchange")
	if err != nil {
		t.Fatalf("makeFifo: %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat: %v", err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("fixture %q is not a named pipe", path)
	}
	// Negative control: a regular file beside the fifo must not
	// identify as one, so the suite can tell the kinds apart.
	control := filepath.Join(dir, "regular")
	if err := os.WriteFile(control, []byte("x"), 0o600); err != nil {
		t.Fatalf("write control: %v", err)
	}
	controlInfo, err := os.Lstat(control)
	if err != nil {
		t.Fatalf("Lstat control: %v", err)
	}
	if controlInfo.Mode()&os.ModeNamedPipe != 0 {
		t.Fatal("regular file identifies as a named pipe")
	}
	// Byte exchange. A bare FIFO open blocks until the peer end
	// opens, so both ends open in goroutines and rendezvous in the
	// kernel; the main goroutine joins both with a timeout so a
	// broken fifo fails the test instead of hanging it. The byte
	// then moves synchronously with both ends held.
	type openResult struct {
		file *os.File
		err  error
	}
	readerCh := make(chan openResult, 1)
	writerCh := make(chan openResult, 1)
	go func() {
		reader, err := os.OpenFile(path, os.O_RDONLY, 0)
		readerCh <- openResult{reader, err}
	}()
	go func() {
		writer, err := os.OpenFile(path, os.O_WRONLY, 0)
		writerCh <- openResult{writer, err}
	}()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	var reader, writer *os.File
	for range 2 {
		select {
		case result := <-readerCh:
			if result.err != nil {
				t.Fatalf("fifo read-end open: %v", result.err)
			}
			reader = result.file
		case result := <-writerCh:
			if result.err != nil {
				t.Fatalf("fifo write-end open: %v", result.err)
			}
			writer = result.file
		case <-timer.C:
			t.Fatal("fifo open handshake blocked; the fixture does not exchange")
		}
	}
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	if _, err := writer.Write([]byte("p")); err != nil {
		t.Fatalf("fifo write: %v", err)
	}
	buf := make([]byte, 1)
	n, err := reader.Read(buf)
	if err != nil || n != 1 || buf[0] != 'p' {
		t.Fatalf("fifo exchange read %d bytes %q, %v; want one byte %q", n, buf, err, "p")
	}
}

func TestSkipFifoProbeAttemptsItsCapability(t *testing.T) {
	// Not parallel: it faults the fifo build seam. With the build
	// broken, the probe must report unavailable; a probe closure
	// hardened to constant true ignores the fault and fails here.
	// On a strict host (linux, darwin) the unbroken probe must hold:
	// absence there is unexpected and must fail, not skip.
	real := fifoBuild
	t.Cleanup(func() { fifoBuild = real })
	capability := FifoCapability()
	if runtime.GOOS == "windows" {
		if ok, _ := capability.Probe(); ok {
			t.Fatal("fifo probe holds on Windows, where named-pipe fixtures are unavailable")
		}
		return
	}
	strict := false
	for _, goos := range capability.StrictGOOS {
		if goos == runtime.GOOS {
			strict = true
		}
	}
	if !strict {
		t.Fatalf("fifo probe is not strict on %s; unexpected absence would skip quietly", runtime.GOOS)
	}
	if ok, reason := capability.Probe(); !ok {
		t.Fatalf("fifo probe fails on strict host %s: %s", runtime.GOOS, reason)
	}
	fifoBuild = func(dir, name string) (string, error) { return "", errTestSentinel }
	if ok, _ := capability.Probe(); ok {
		t.Fatal("fifo probe holds with its build broken; the probe attempts nothing")
	}
}

func TestSkipModeBitsAndNonRootDenialsHold(t *testing.T) {
	t.Parallel()
	// Both probes gate this test. As superuser the first Require
	// skips (permission denials are untestable past the uid check);
	// where mode bits are unenforceable the second Require skips;
	// where both hold, the denial must be a permission error, not
	// merely any failure.
	Require(t, NonRootCapability())
	Require(t, ModeBitsCapability())
	dir := t.TempDir()
	path := filepath.Join(dir, "guarded")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	file, err := os.Open(path)
	if err == nil {
		_ = file.Close()
		t.Fatal("open succeeded despite mode 000; permission bits are not enforced here")
	}
	if !os.IsPermission(err) {
		t.Fatalf("denial error %v is not a permission error", err)
	}
}

func TestSkipReportReadsRecordedVerdicts(t *testing.T) {
	// Not parallel: it mutates the process-wide counters. It records
	// its own synthetic verdicts, asserts the report moves, then
	// restores the suite state — so the numbers below are evidence
	// that Report reads the live counters, not a constant. Gutting
	// recordVerdict leaves both assertions red. This pins Report, not
	// the Require wiring: M04/M05 are pinned by the fake-Require
	// tests below, which drive the real Require through both arms.
	const skipID = "self-test-skip-probe"
	const failID = "self-test-fail-probe"
	beforeSkipped, beforeFailed := snapshotReport()
	t.Cleanup(func() {
		restoreReport(beforeSkipped, beforeFailed)
		for _, id := range SkippedIDs() {
			if id == skipID || id == failID {
				t.Errorf("synthetic verdict %q leaked into the suite counters", id)
			}
		}
	})
	recordVerdict(Verdict{ID: skipID, Skipped: true, Reason: "synthetic skip"})
	recordVerdict(Verdict{ID: failID, Failed: true, Reason: "synthetic failure"})
	found := false
	for _, id := range SkippedIDs() {
		if id == skipID {
			found = true
		}
	}
	if !found {
		t.Fatalf("SkippedIDs() = %q, want it to contain %q", SkippedIDs(), skipID)
	}
	report := Report()
	if !strings.Contains(report, skipID) || !strings.Contains(report, failID) {
		t.Fatalf("Report() = %q, want it to carry both synthetic verdicts", report)
	}
	// Attached evidence with real nonzero counters from this test's
	// own recorded verdicts.
	LogReport(t)
	t.Logf("skipped IDs: %q", SkippedIDs())
}

// fakeRequireT records Skipf/Fatalf without aborting the test, so the
// wiring tests can drive Require through arms that would otherwise
// skip or fail the suite.
type fakeRequireT struct {
	skipped bool
	failed  bool
	skipMsg string
	failMsg string
}

func (fake *fakeRequireT) Helper() {}

func (fake *fakeRequireT) Skipf(format string, args ...any) {
	fake.skipped = true
	fake.skipMsg = fmt.Sprintf(format, args...)
}

func (fake *fakeRequireT) Fatalf(format string, args ...any) {
	fake.failed = true
	fake.failMsg = fmt.Sprintf(format, args...)
}

func TestRequireSkipArmRecordsVerdict(t *testing.T) {
	// Not parallel: it mutates the process-wide counters. It drives
	// the real Require through the skip arm with a fake T, so the
	// suite stays green while the wiring is observed. Deleting
	// recordVerdict from the skip arm (M05) leaves the counters empty
	// and fails here.
	const skipID = "self-test-require-skip"
	beforeSkipped, beforeFailed := snapshotReport()
	t.Cleanup(func() { restoreReport(beforeSkipped, beforeFailed) })
	fake := &fakeRequireT{}
	capability := Capability{
		ID:         skipID,
		Detail:     "synthetic always-unavailable non-strict capability",
		Probe:      func() (bool, string) { return false, "synthetic down" },
		StrictGOOS: []string{"plan9"},
	}
	verdict := Require(fake, capability)
	if !fake.skipped {
		t.Fatal("Require through the skip arm never called Skipf")
	}
	if fake.failed {
		t.Fatalf("skip arm called Fatalf: %q", fake.failMsg)
	}
	if !verdict.Skipped || verdict.Failed || verdict.ID != skipID {
		t.Fatalf("skip verdict = %+v, want Skipped with ID %q", verdict, skipID)
	}
	found := false
	for _, id := range SkippedIDs() {
		if id == skipID {
			found = true
		}
	}
	if !found {
		t.Fatalf("SkippedIDs() = %q, want it to contain the Require-driven %q", SkippedIDs(), skipID)
	}
	if report := Report(); !strings.Contains(report, skipID) {
		t.Fatalf("Report() = %q, want it to carry the Require-driven %q", report, skipID)
	}
}

func TestRequireFailArmRecordsVerdict(t *testing.T) {
	// Not parallel: it mutates the process-wide counters. It drives
	// the real Require through the fail arm with a fake T, so the
	// suite stays green while the wiring is observed. Deleting
	// recordVerdict from the fail arm (M04) leaves the failed list
	// empty and fails here.
	const failID = "self-test-require-fail"
	beforeSkipped, beforeFailed := snapshotReport()
	t.Cleanup(func() { restoreReport(beforeSkipped, beforeFailed) })
	fake := &fakeRequireT{}
	capability := Capability{
		ID:         failID,
		Detail:     "synthetic always-unavailable strict capability",
		Probe:      func() (bool, string) { return false, "synthetic down" },
		StrictGOOS: []string{runtime.GOOS},
	}
	verdict := Require(fake, capability)
	if !fake.failed {
		t.Fatal("Require through the fail arm never called Fatalf")
	}
	if fake.skipped {
		t.Fatalf("fail arm called Skipf: %q", fake.skipMsg)
	}
	if !verdict.Failed || verdict.Skipped || verdict.ID != failID {
		t.Fatalf("fail verdict = %+v, want Failed with ID %q", verdict, failID)
	}
	if report := Report(); !strings.Contains(report, failID) {
		t.Fatalf("Report() = %q, want it to carry the Require-driven %q", report, failID)
	}
}

func TestSkipStrictPlatformsArePinned(t *testing.T) {
	t.Parallel()
	// M08/M09: the symlink and fifo probes are strict on linux and
	// darwin, where an unexpected absence must fail rather than skip
	// quietly. Dropping darwin from either list would let an absent
	// capability skip on the developers' primary host. Both lists are
	// pinned exactly here; fifo's list was already pinned by the
	// honesty test's strict assertion, symlink's was not.
	for _, kase := range []struct {
		name string
		got  []string
	}{
		{"symlink", SymlinkCapability().StrictGOOS},
		{"fifo", FifoCapability().StrictGOOS},
	} {
		want := []string{"darwin", "linux"}
		got := append([]string(nil), kase.got...)
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("%s StrictGOOS = %q, want %q", kase.name, kase.got, want)
		}
	}
}
