package resumesmoke

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provhost"
)

// This file attacks the smoke record itself: tampered bytes,
// promotion attempts, inconsistent verdicts, and non-canonical
// encodings must all refuse on re-read.

// TestSmokeTamperedRecordRefuses proves one flipped byte anywhere
// in an installed record refuses on re-read, for passing and gated
// records alike: integrity is not verdict-scoped.
func TestSmokeTamperedRecordRefuses(t *testing.T) {
	for _, tuple := range []provhost.BuildTuple{
		{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"},
		{ProviderID: "gemini", ProviderVersion: "0.54.4", Platform: "wsl2", Architecture: "amd64"},
	} {
		report := runRowFixture(t, tuple)
		for _, index := range []int{0, len(report.Bytes) / 2, len(report.Bytes) - 2} {
			tampered := bytes.Clone(report.Bytes)
			tampered[index] ^= 0x01
			if _, err := VerifyRecord(tampered); err == nil {
				t.Fatalf("VerifyRecord(tampered %s byte %d) = nil, want refusal", tuple.ProviderID, index)
			}
			if _, _, err := Load(writeTempRecord(t, tampered)); err == nil {
				t.Fatalf("Load(tampered %s byte %d) = nil, want refusal", tuple.ProviderID, index)
			}
		}
	}
}

// TestSmokePromotionAttemptRefuses proves a gated cell cannot be
// promoted through the record: flipping the verdict to pass
// refuses on the digest, and flipping it with a recomputed digest
// still refuses on the verdict consistency rule, because the
// gated resume plan is still in the check list.
func TestSmokePromotionAttemptRefuses(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "gemini", ProviderVersion: "0.54.4", Platform: "wsl2", Architecture: "amd64"}
	report := runRowFixture(t, tuple)
	if report.Record.Verdict != VerdictGated {
		t.Fatalf("fixture verdict = %q, want gated", report.Record.Verdict)
	}
	flipped := bytes.Replace(report.Bytes, []byte(`"verdict":"gated"`), []byte(`"verdict":"pass"`), 1)
	if bytes.Equal(flipped, report.Bytes) {
		t.Fatal("promotion fixture did not flip the verdict frame")
	}
	if _, err := VerifyRecord(flipped); err == nil {
		t.Fatal("VerifyRecord(flipped verdict) = nil, want digest refusal")
	}
	var record Record
	if err := json.Unmarshal(report.Bytes, &record); err != nil {
		t.Fatalf("unmarshal gated record: %v", err)
	}
	record.Verdict = VerdictPass
	resigned, err := encodeRecord(record)
	if err != nil {
		t.Fatalf("encodeRecord: %v", err)
	}
	if _, err := VerifyRecord(resigned); err == nil {
		t.Fatal("VerifyRecord(resigned promotion) = nil, want consistency refusal")
	} else if !strings.Contains(err.Error(), "inconsistent") {
		t.Fatalf("VerifyRecord(resigned promotion) = %v, want the consistency refusal", err)
	}
}

// TestSmokeInconsistentFailRefuses proves the consistency rule runs
// in both directions: an available record whose checks all pass
// cannot verify as a failure either.
func TestSmokeInconsistentFailRefuses(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	if report.Record.Verdict != VerdictPass {
		t.Fatalf("fixture verdict = %q, want pass", report.Record.Verdict)
	}
	var record Record
	if err := json.Unmarshal(report.Bytes, &record); err != nil {
		t.Fatalf("unmarshal pass record: %v", err)
	}
	record.Verdict = VerdictFail
	resigned, err := encodeRecord(record)
	if err != nil {
		t.Fatalf("encodeRecord: %v", err)
	}
	if _, err := VerifyRecord(resigned); err == nil {
		t.Fatal("VerifyRecord(resigned fail) = nil, want consistency refusal")
	}
}

// TestSmokeWhitespaceVariantRefuses proves the digest binds the
// exact emitted bytes: a semantically equal but reindented record
// refuses on its record_id frame and digest.
func TestSmokeWhitespaceVariantRefuses(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	report := runRowFixture(t, tuple)
	var decoded any
	if err := json.Unmarshal(report.Bytes, &decoded); err != nil {
		t.Fatalf("unmarshal record: %v", err)
	}
	indented, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		t.Fatalf("indent record: %v", err)
	}
	if _, err := VerifyRecord(indented); err == nil {
		t.Fatal("VerifyRecord(indented) = nil, want digest refusal")
	}
}

// TestSmokeIdenticalInputsReplayIdenticalBytes proves the smoke is
// a pure function of its params: two runs replay byte-identical
// records, and both verify.
func TestSmokeIdenticalInputsReplayIdenticalBytes(t *testing.T) {
	tuple := provhost.BuildTuple{ProviderID: "codex", ProviderVersion: "0.147.0", Platform: "macos", Architecture: "arm64"}
	first := runRowFixture(t, tuple)
	second := runRowFixture(t, tuple)
	if !bytes.Equal(first.Bytes, second.Bytes) {
		t.Fatal("identical smoke inputs replayed differing bytes")
	}
	if _, err := VerifyRecord(second.Bytes); err != nil {
		t.Fatalf("VerifyRecord(replay) = %v", err)
	}
}

// runRowFixture runs one row end to end with scripted fakes and
// returns its report: available rows pass, conditional rows gate.
func runRowFixture(t *testing.T, tuple provhost.BuildTuple) *Report {
	t.Helper()
	cell, _ := ResumeCell(tuple.ProviderID, tuple.ProviderVersion, tuple.Platform, tuple.Architecture)
	runner := newScriptedRunner(t)
	runner.scriptProbe(tuple)
	runner.scriptIdentify(t, tuple, "exact")
	params := smokeParams(tuple, runner)
	if cell == CellAvailable {
		runner.scriptResume(t, tuple, params.Profile)
	}
	params.Discovery = smokeDiscovery(t, tuple, params.Home, params.XDGDataHome)
	if tuple.ProviderID == "antigravity" {
		params.Realm = smokeRealm
	}
	report, err := Run(context.Background(), params)
	if err != nil {
		t.Fatalf("Run(%+v) error = %v", tuple, err)
	}
	return report
}
