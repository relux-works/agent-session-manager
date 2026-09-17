package resumesmoke

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// This file proves the durable install: verified records install
// no-replace and fsynced, identical retries reuse the path,
// disagreeing bytes refuse for quarantine, and crash-hook faults
// leave nothing the retry cannot recover.

// TestStoreInstallsVerifiedRecord proves a verified record installs
// at its content-addressed path and loads back byte-identical,
// while invalid bytes refuse before any file is created.
func TestStoreInstallsVerifiedRecord(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	directory := t.TempDir()
	path, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store error = %v", err)
	}
	if !strings.HasPrefix(filepath.Base(path), "native-resume-smoke-") {
		t.Fatalf("Store path = %q, want the content-addressed name", path)
	}
	record, body, err := Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if !bytes.Equal(body, report.Bytes) {
		t.Fatal("loaded bytes differ from the installed record")
	}
	if record.Verdict != VerdictPass {
		t.Fatalf("loaded verdict = %q, want pass", record.Verdict)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("directory holds %d entries, want exactly the record", len(entries))
	}
	if _, err := Store(directory, []byte(`{}`), nil); err == nil {
		t.Fatal("Store(invalid) = nil, want refusal")
	}
}

// TestStoreReplayIsIdentical proves the idempotent install: storing
// the same record twice returns the same path with identical bytes.
func TestStoreReplayIsIdentical(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	directory := t.TempDir()
	first, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store error = %v", err)
	}
	second, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store(replay) error = %v", err)
	}
	if first != second {
		t.Fatalf("replay path = %q, want %q", second, first)
	}
	_, loaded, err := Load(first)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if !bytes.Equal(loaded, report.Bytes) {
		t.Fatal("replayed bytes differ from the installed record")
	}
}

// TestStoreDisagreementQuarantines proves a torn write refuses the
// retry instead of replacing evidence: garbage at the record's
// path fails the install with the quarantine direction, and the
// retry succeeds byte-identical once the torn file is removed.
func TestStoreDisagreementQuarantines(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	directory := t.TempDir()
	path, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store error = %v", err)
	}
	// A torn write of exactly the record length: the disagreement
	// is in the bytes, never the size.
	torn := bytes.Clone(report.Bytes)
	for index := range torn {
		torn[index] = 'x'
	}
	if err := os.WriteFile(path, torn, 0o600); err != nil {
		t.Fatalf("corrupt installed record: %v", err)
	}
	if _, err := Store(directory, report.Bytes, nil); err == nil {
		t.Fatal("Store(disagreeing bytes) = nil, want refusal")
	} else if !strings.Contains(err.Error(), "quarantine") {
		t.Fatalf("Store(disagreeing bytes) = %v, want the quarantine direction", err)
	}
	if _, _, err := Load(path); err == nil {
		t.Fatal("Load(torn bytes) = nil, want refusal")
	}
	if err := os.Remove(path); err != nil {
		t.Fatalf("quarantine torn record: %v", err)
	}
	retry, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store(after quarantine) error = %v", err)
	}
	if retry != path {
		t.Fatalf("retry path = %q, want %q", retry, path)
	}
	_, loaded, err := Load(retry)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if !bytes.Equal(loaded, report.Bytes) {
		t.Fatal("post-quarantine bytes differ from the record")
	}
}

// TestStoreCrashBeforeCreateAdmitsNothing proves the pre-create
// crash point installs nothing and the identical retry succeeds.
func TestStoreCrashBeforeCreateAdmitsNothing(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	directory := t.TempDir()
	boom := errors.New("crash before create")
	if _, err := Store(directory, report.Bytes, &StoreHooks{BeforeCreate: func(string) error { return boom }}); err == nil {
		t.Fatal("Store(before-create fault) = nil, want the fault")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("directory holds %d entries after the fault, want none", len(entries))
	}
	if _, err := Store(directory, report.Bytes, nil); err != nil {
		t.Fatalf("Store(retry) error = %v", err)
	}
}

// TestStoreCrashAfterWriteReplays proves the post-write crash point
// recovers through the idempotent retry: the faulted install may
// leave complete bytes behind, and the identical retry reuses them.
func TestStoreCrashAfterWriteReplays(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	directory := t.TempDir()
	boom := errors.New("crash after write")
	if _, err := Store(directory, report.Bytes, &StoreHooks{AfterWrite: func(string) error { return boom }}); err == nil {
		t.Fatal("Store(after-write fault) = nil, want the fault")
	}
	path, err := Store(directory, report.Bytes, nil)
	if err != nil {
		t.Fatalf("Store(retry) error = %v", err)
	}
	_, loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if !bytes.Equal(loaded, report.Bytes) {
		t.Fatal("post-crash bytes differ from the record")
	}
}

// writeTempRecord writes bytes to a temp file and returns its path.
func writeTempRecord(t *testing.T, record []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "record.json")
	if err := os.WriteFile(path, record, 0o600); err != nil {
		t.Fatalf("write temp record: %v", err)
	}
	return path
}
