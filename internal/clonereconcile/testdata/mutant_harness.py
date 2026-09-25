#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/clonereconcile.

Every decision gate ships a narrowing mutant: the gate stays present
and is weakened to admit exactly one member of the class it must
reject, and the named killer test must fail. A delete-only mutant
proves only that the gate exists and is not accepted as evidence.

Each row runs M (mutate one production file in place), R (run the
scoped Go killer through a production entry), V (verdict from the Go
exit code plus a FAIL kill-line: KILLED iff R fails with a test
failure line, SURVIVED iff R passes, ERROR for anything else --
unapplied patches, build breaks, timeouts). M failures and R
infrastructure errors are ERROR, never kills or survivals. The file
is restored from its backup after every row, pass or fail.

The four T- rows are token-preserving attacks: the searched-for
token still matches, but the decision changes. Their killers execute
the behavioral gate tests, and the full package suite re-runs on all
four to prove the always-on suite kills them too.

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

The two fidelity-report wrap sites carry no mutant: they forward the
owner's refusal and any weakening still refuses with the identical
literal (the decode fallback), so no verdict change is expressible.
They are covered by the owner suite plus the wrap tests.

Digests embedded below are deterministic fixture outputs (sha256 over
fixed seeds); a fixture change fails the patch application loudly
instead of silently testing a new member.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/clonereconcile/testdata/mutant_harness.py [--log-dir DIR] [MUTANT ...]
"""

import shutil
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PACKAGE = "internal/clonereconcile"
TARGET = "internal/clonereconcile/reconcile.go"

RAW0 = "sha256:f85d70a61f8936e59753518dec2603666dbbc943722bff045c0a6011be08aeb5"
RAW1 = "sha256:4c1364d82d8b6632ff97567cb344f89855d9caf75e71091dbb3d5f4023fca644"
RAW2 = "sha256:feeb74ff2b27e5e4d20e2a4b317ae44fce85548205fa92ffa06de1ebc344a49b"
CANON0 = "sha256:fbdd0b01d4b38469a7161da02b326942a6b537a3169078b02b70c2d011533f80"
CANON1 = "sha256:c2e2abc08246be4c6057dbdd134bfa04ba65f2bd9d81eafe2073fa61a1c4c764"
PHANTOM = "sha256:799ac6577a0b2d6da214bd9d12ae1d5febf1e22e3f610f5b82a9cb2237acab3a"
STAGED0 = "sha256:6e7236396050a92937998fd5bb9b2d63dca651d76fe49089f0a38f555d24ca9b"
LIVE0 = "sha256:301d60fd5c4d5f982d2a2dc9059b6f1b1c50f76de48376aa537183cc0b13505f"
PLANA = "sha256:d97c95b798668173a1dcd2203b6f646c9bef874ed56eb0558afa8da78da92bd9"
PLANB = "sha256:a4b35f9d23736760bcef679ffcd31176a59f8628edb8394a5c8de8e4fb82f642"
PROJA = "sha256:eba510d8697301f2ef1f62eaab26d80f07ef61ebedc6595034cb1beddd77202f"
PROJB = "sha256:ee806e835480f3cfb4a60f5947ca17f6f49d410b0428ec243972e4cfb90d0aee"
OTHER_FP = "sha256:8853d6d8902e3bb3068470909ad464207318449c2927c0faa5730eb875b52ad0"
OTHER_TUPLE = (
    'sessadapter.Tuple{EnvironmentID: "relux.other", Version: "2026.09", '
    'Platform: "linux", Architecture: "amd64", StoreFingerprint: "' + OTHER_FP + '", '
    'AdapterVersion: "1.2.4"}'
)


@dataclass(frozen=True)
class Mutant:
    name: str
    find: str
    replace: str
    run: str
    expect: str
    note: str


GATE = "TestReconcileGateRefusals/"
HISTORY = "TestReadBackHistoryEmptyInput/"


def mutants():
    other_staged = (
        '\tif *fidelity.TargetEnvironment != history.Staged.Manifest().ObservedEnvironment {',
        '\tif *fidelity.TargetEnvironment != history.Staged.Manifest().ObservedEnvironment && *fidelity.TargetEnvironment != (' + OTHER_TUPLE + ') {',
    )
    other_live = (
        '\tif *fidelity.TargetEnvironment != history.Live.Manifest().ObservedEnvironment {',
        '\tif *fidelity.TargetEnvironment != history.Live.Manifest().ObservedEnvironment && history.Live.Manifest().ObservedEnvironment != (' + OTHER_TUPLE + ') {',
    )
    other_validation = (
        '\tif target != history.Staged.Manifest().ObservedEnvironment {',
        '\tif target != history.Staged.Manifest().ObservedEnvironment && target != (' + OTHER_TUPLE + ') {',
    )
    return [
        Mutant("N-scope", '\tif input.Fidelity.Scope != "target" {',
               '\tif input.Fidelity.Scope != "target" && input.Fidelity.Scope != "archive" {',
               GATE + "scope-archive", "KILLED", "narrowing: admits archive scope only"),
        Mutant("N-history-staged", '\tif staged.Mode() != "staged" {',
               '\tif staged.Mode() != "staged" && staged.Mode() != "live" {',
               GATE + "staged-not-staged", "KILLED", "narrowing: admits a live read in the staged slot only"),
        Mutant("N-history-live", '\tif live.Mode() != "live" {',
               '\tif live.Mode() != "live" && live.Mode() != "staged" {',
               GATE + "live-not-live", "KILLED", "narrowing: admits a staged read in the live slot only"),
        Mutant("N-wrap-staged",
               '\tstaged, err := clonereadback.DecodeReadBackEvidenceManifest(stagedBytes, stagedAuthority)\n\tif err != nil {',
               '\tstaged, err := clonereadback.DecodeReadBackEvidenceManifest(stagedBytes, stagedAuthority)\n\tif err != nil && len(stagedBytes) > 0 {',
               HISTORY + "empty-staged", "KILLED", "narrowing: admits the empty staged document only"),
        Mutant("N-wrap-live",
               '\tlive, err := clonereadback.DecodeReadBackEvidenceManifest(liveBytes, liveAuthority)\n\tif err != nil {',
               '\tlive, err := clonereadback.DecodeReadBackEvidenceManifest(liveBytes, liveAuthority)\n\tif err != nil && len(liveBytes) > 0 {',
               HISTORY + "empty-live", "KILLED", "narrowing: admits the empty live document only"),
        Mutant("N-wrap-plan",
               '\tplan, err := cloneplan.DecodeProjectionPlan(input.Plan)\n\tif err != nil {',
               '\tplan, err := cloneplan.DecodeProjectionPlan(input.Plan)\n\tif err != nil && len(input.Plan) > 0 {',
               GATE + "plan-empty", "KILLED", "narrowing: admits empty plan bytes only"),
        Mutant("N-wrap-projected",
               '\tprojected, err := cloneplan.DecodeProjectedObjectManifest(input.Projected)\n\tif err != nil {',
               '\tprojected, err := cloneplan.DecodeProjectedObjectManifest(input.Projected)\n\tif err != nil && len(input.Projected) > 0 {',
               GATE + "projected-empty", "KILLED", "narrowing: admits empty projected bytes only"),
        Mutant("N-wrap-tuple",
               '\ttargetTuple, err := sessadapter.DecodeTuple(input.Validation.TargetEnvironment)\n\tif err != nil {',
               '\ttargetTuple, err := sessadapter.DecodeTuple(input.Validation.TargetEnvironment)\n\tif err != nil && len(input.Validation.TargetEnvironment) > 0 {',
               GATE + "validation-tuple-empty", "KILLED", "narrowing: admits the empty tuple only"),
        Mutant("N-wrap-expected-parse",
               '\texpectedFidelity, err := scalar.ParseDigest(input.Validation.ExpectedFidelityReportID)\n\tif err != nil {',
               '\texpectedFidelity, err := scalar.ParseDigest(input.Validation.ExpectedFidelityReportID)\n\tif err != nil && input.Validation.ExpectedFidelityReportID != "" {',
               GATE + "expected-fidelity-empty", "KILLED", "narrowing: admits the empty expected digest only"),
        Mutant("N-wrap-seal",
               '\treturn nil, false, refuse("validation report invalid: %v; %v", err, retryErr)',
               '\tif string(input.Validation.Findings) == "not-json" {\n\t\treturn nil, false, nil\n\t}\n\treturn nil, false, refuse("validation report invalid: %v; %v", err, retryErr)',
               GATE + "validation-malformed", "KILLED", "narrowing: admits the not-json findings member only"),
        Mutant("N-cand-utf8", '\t\tif !utf8.ValidString(candidate) {',
               '\t\tif !utf8.ValidString(candidate) && candidate != "bad\\xff" {',
               GATE + "candidate-utf8", "KILLED", "narrowing: admits the bad-ff key only"),
        Mutant("N-cand-bounds", '\t\tif length := environ.StringLength(candidate); length < 1 || length > 512 {',
               '\t\tif length := environ.StringLength(candidate); length < 1 || length > 513 {',
               GATE + "candidate-too-long", "KILLED", "narrowing: admits length 513 only"),
        Mutant("N-cand-dup", '\t\tif seen[candidate] {',
               '\t\tif seen[candidate] && candidate != "cand-0" {',
               GATE + "candidate-duplicate", "KILLED", "narrowing: admits the cand-0 duplicate only"),
        Mutant("N-t1-undeclared", '\t\tif !declared[link.Candidate] {',
               '\t\tif !declared[link.Candidate] && link.Candidate != "ghost" {',
               GATE + "tier1-undeclared", "KILLED", "narrowing: admits the ghost link only"),
        Mutant("N-t1-twice", '\t\tif linked[link.Candidate] {',
               '\t\tif linked[link.Candidate] && link.Candidate != "cand-0" {',
               GATE + "tier1-twice", "KILLED", "narrowing: admits the cand-0 second link only"),
        Mutant("N-t1-missing",
               '\tfor _, candidate := range candidates {\n\t\tif !linked[candidate] {',
               '\tfor _, candidate := range candidates {\n\t\tif !linked[candidate] && candidate != "cand-0" {',
               GATE + "tier1-missing", "KILLED", "narrowing: excuses the cand-0 link only"),
        Mutant("N-t1-both", '\t\tcase link.Raw != "" && link.Exclusion != "":',
               '\t\tcase link.Raw != "" && link.Exclusion != "" && link.Exclusion != "durable_payload":',
               GATE + "tier1-both", "KILLED", "narrowing: admits the durable_payload both-arms link only"),
        Mutant("N-t1-neither", '\t\tcase link.Raw == "" && link.Exclusion == "":',
               '\t\tcase link.Raw == "" && link.Exclusion == "" && link.Candidate != "cand-0":',
               GATE + "tier1-neither", "KILLED", "narrowing: admits the cand-0 neither-arms link only"),
        Mutant("N-t1-raw-digest",
               '\t\tcase link.Raw != "":\n\t\t\traw, err := scalar.ParseDigest(link.Raw)\n\t\t\tif err != nil {',
               '\t\tcase link.Raw != "":\n\t\t\traw, err := scalar.ParseDigest(link.Raw)\n\t\t\tif err != nil && link.Raw != "bogus" {',
               GATE + "tier1-raw-not-digest", "KILLED", "narrowing: admits the bogus tier-1 raw only"),
        Mutant("N-t1-raw-twice", '\t\t\tif rawSet[raw.String()] {',
               '\t\t\tif rawSet[raw.String()] && raw.String() != "' + RAW0 + '" {',
               GATE + "tier1-raw-twice", "KILLED", "narrowing: admits the raw-0 second claim only"),
        Mutant("N-t1-exclusion", '\t\t\tif !clonebundle.ValidCaptureClass(link.Exclusion) {',
               '\t\t\tif !clonebundle.ValidCaptureClass(link.Exclusion) && link.Exclusion != "frobnicate" {',
               GATE + "tier1-bad-exclusion", "KILLED", "narrowing: admits the frobnicate exclusion only"),
        Mutant("N-t2-raw-digest", '\t\t\treturn nil, nil, nil, refuse("tier-2 raw item %q is not a digest: %v", link.Raw, err)',
               '\t\t\treturn nil, nil, nil, refuse("tier-2 raw item %q is not a digest: %v", link.Raw, err[1:])',
               GATE + "tier2-raw-not-digest", "KILLED", "narrowing: corrupts the detail (placeholder, replaced below)"),
        Mutant("N-t2-unknown", '\t\tif !rawSet[raw.String()] {',
               '\t\tif !rawSet[raw.String()] && raw.String() != "' + PHANTOM + '" {',
               GATE + "tier2-unknown-raw", "KILLED", "narrowing: admits the phantom raw link only"),
        Mutant("N-t2-twice", '\t\tif linked[raw.String()] {',
               '\t\tif linked[raw.String()] && raw.String() != "' + RAW0 + '" {',
               GATE + "tier2-twice", "KILLED", "narrowing: admits the raw-0 second link only"),
        Mutant("N-t2-both", '\t\tcase link.Canonical != "" && link.Normalization != "":',
               '\t\tcase link.Canonical != "" && link.Normalization != "" && link.Normalization != "omitted":',
               GATE + "tier2-both", "KILLED", "narrowing: admits the omitted both-arms link only"),
        Mutant("N-t2-neither", '\t\tcase link.Canonical == "" && link.Normalization == "":',
               '\t\tcase link.Canonical == "" && link.Normalization == "" && link.Raw != "' + RAW0 + '":',
               GATE + "tier2-neither", "KILLED", "narrowing: admits the raw-0 neither-arms link only"),
        Mutant("N-t2-canon-digest", '\t\t\t\treturn nil, nil, nil, refuse("tier-2 canonical item %q is not a digest: %v", link.Canonical, err)',
               '\t\t\t\treturn nil, nil, nil, refuse("tier-2 canonical item %q is not a digest: %v", link.Canonical, err)',
               GATE + "tier2-canon-not-digest", "KILLED", "narrowing: identity (placeholder, replaced below)"),
        Mutant("N-t2-canon-twice", '\t\t\tif canonicalSet[canonical.String()] {',
               '\t\t\tif canonicalSet[canonical.String()] && canonical.String() != "' + CANON0 + '" {',
               GATE + "tier2-canon-twice", "KILLED", "narrowing: admits the canon-0 second claim only"),
        Mutant("N-t2-normalization", '\t\t\tif !clonefidelity.ValidDisposition(link.Normalization) {',
               '\t\t\tif !clonefidelity.ValidDisposition(link.Normalization) && link.Normalization != "frobnicate" {',
               GATE + "tier2-bad-normalization", "KILLED", "narrowing: admits the frobnicate normalization only"),
        Mutant("N-t2-synth", '\t\t\tif link.Normalization == "synthesized" {',
               '\t\t\tif link.Normalization == "synthesized" && link.Raw != "' + RAW2 + '" {',
               GATE + "tier2-synth-normalization", "KILLED", "narrowing: admits synthesized for raw-2 only"),
        Mutant("N-t2-missing",
               '\tfor raw := range rawSet {\n\t\tif !linked[raw] {',
               '\tfor raw := range rawSet {\n\t\tif !linked[raw] && raw != "' + RAW0 + '" {',
               GATE + "tier2-missing", "KILLED", "narrowing: excuses the raw-0 link only"),
        Mutant("N-t3-synth", '\t\t\t\tif rawSet[evidence.String()] {',
               '\t\t\t\tif rawSet[evidence.String()] && evidence.String() != "' + RAW0 + '" {',
               GATE + "tier3-synth-claims", "KILLED", "narrowing: admits the raw-0 synth claim only"),
        Mutant("N-t3-unknown", '\t\tif !included[row.SourceItemKey] {',
               '\t\tif !included[row.SourceItemKey] && row.SourceItemKey != "cand-0x" {',
               GATE + "tier3-unknown-key", "KILLED", "narrowing: admits the cand-0x row only"),
        Mutant("N-t3-needs-canon",
               '\t\t\tif row.CanonicalObjectID == nil {\n\t\t\t\treturn refuse("tier-3 row %q needs canonical %q from tier 2", row.SourceItemKey, want)\n\t\t\t}',
               '\t\t\tif row.CanonicalObjectID == nil && row.SourceItemKey != "cand-0" {\n\t\t\t\treturn refuse("tier-3 row %q needs canonical %q from tier 2", row.SourceItemKey, want)\n\t\t\t}\n\t\t\tif row.CanonicalObjectID == nil {\n\t\t\t\tcontinue\n\t\t\t}',
               GATE + "tier3-needs-canonical", "KILLED", "narrowing: skips the cand-0 null-canonical row only"),
        Mutant("N-t3-agreement", '\t\t\tif row.CanonicalObjectID.String() != want {',
               '\t\t\tif row.CanonicalObjectID.String() != want && !(row.CanonicalObjectID.String() == "' + CANON1 + '" && want == "' + CANON0 + '") {',
               GATE + "tier3-wrong-canonical", "KILLED", "narrowing: admits the (canon-1, canon-0) pair only"),
        Mutant("N-t3-under-norm", '\t\t} else if row.CanonicalObjectID != nil {',
               '\t\t} else if row.CanonicalObjectID != nil && row.CanonicalObjectID.String() != "' + PHANTOM + '" {',
               GATE + "tier3-canon-under-normalization", "KILLED", "narrowing: admits the phantom normalized canonical only"),
        Mutant("N-t3-unreconciled",
               '\t\t\tif owner, ok := candidateByRaw[evidence.String()]; ok {\n\t\t\t\treturn refuse("tier-3 row %q claims source evidence %q of candidate %q",\n\t\t\t\t\trow.SourceItemKey, evidence.String(), owner)\n\t\t\t}\n\t\t\treturn refuse("tier-3 row %q claims unreconciled source evidence %q", row.SourceItemKey, evidence.String())',
               '\t\t\tif owner, ok := candidateByRaw[evidence.String()]; ok {\n\t\t\t\treturn refuse("tier-3 row %q claims source evidence %q of candidate %q",\n\t\t\t\t\trow.SourceItemKey, evidence.String(), owner)\n\t\t\t}\n\t\t\tif evidence.String() == "' + PHANTOM + '" {\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\treturn refuse("tier-3 row %q claims unreconciled source evidence %q", row.SourceItemKey, evidence.String())',
               GATE + "tier3-unreconciled-evidence", "KILLED", "narrowing: admits the phantom source claim only"),
        Mutant("N-t3-swapped",
               '\t\t\tif owner, ok := candidateByRaw[evidence.String()]; ok {\n\t\t\t\treturn refuse("tier-3 row %q claims source evidence %q of candidate %q",',
               '\t\t\tif owner, ok := candidateByRaw[evidence.String()]; ok {\n\t\t\t\tif evidence.String() == "' + RAW1 + '" {\n\t\t\t\t\tcontinue\n\t\t\t\t}\n\t\t\t\treturn refuse("tier-3 row %q claims source evidence %q of candidate %q",',
               GATE + "tier3-swapped-evidence", "KILLED", "narrowing: admits the raw-1 swap member only; the gate test fails because the refusal moves to the mirror row"),
        Mutant("N-t3-no-source", '\t\tif len(row.SourceEvidenceIDs) == 0 {',
               '\t\tif len(row.SourceEvidenceIDs) == 0 && row.SourceItemKey != "cand-0" {',
               GATE + "tier3-missing-evidence", "KILLED", "narrowing: admits the cand-0 empty source list only; the owner shape refusal changes the literal"),
        Mutant("N-t3-double", '\t\t\tif first, ok := claimedBy[evidence.String()]; ok {',
               '\t\t\tif first, ok := claimedBy[evidence.String()]; ok && evidence.String() != "' + RAW0 + '" {',
               GATE + "tier3-double-claimed-evidence", "KILLED", "narrowing: admits the raw-0 double claim only; the per-row chain check then reports the swap literal"),
        Mutant("N-t3-doublechain-prefix",
               '\tclaimedBy := make(map[string]string)\n\tfor _, row := range rows {',
               '\tclaimedBy := make(map[string]string)\n\tfor i, row := range rows {\n\t\tif i >= 3 {\n\t\t\tbreak\n\t\t}',
               "TestReconciliationPropertyLargeN", "KILLED", "narrowing: censuses only the first three rows; invisible on every N<=3 input, killed by the N>=4 enumeration alone (subset survival shown in N-t3-doublechain-prefix-subset.log)"),
        Mutant("N-t3-loss-staged", '\t\t\tif len(row.StagedEvidenceObjectIDs) > 0 {',
               '\t\t\tif len(row.StagedEvidenceObjectIDs) > 0 && (len(row.StagedEvidenceObjectIDs) != 1 || row.StagedEvidenceObjectIDs[0].String() != "' + STAGED0 + '") {',
               GATE + "tier3-loss-claims-staged", "KILLED", "narrowing: admits the single staged-0 loss claim only"),
        Mutant("N-t3-loss-live", '\t\t\tif len(row.LiveEvidenceObjectIDs) > 0 {',
               '\t\t\tif len(row.LiveEvidenceObjectIDs) > 0 && (len(row.LiveEvidenceObjectIDs) != 1 || row.LiveEvidenceObjectIDs[0].String() != "' + LIVE0 + '") {',
               GATE + "tier3-loss-claims-live", "KILLED", "narrowing: admits the single live-0 loss claim only"),
        Mutant("N-t3-no-staged", '\t\tif len(row.StagedEvidenceObjectIDs) == 0 {',
               '\t\tif len(row.StagedEvidenceObjectIDs) == 0 && row.SourceItemKey != "cand-0" {',
               GATE + "tier3-no-staged", "KILLED", "narrowing: excuses the cand-0 staged gap only"),
        Mutant("N-t3-no-live", '\t\tif len(row.LiveEvidenceObjectIDs) == 0 {',
               '\t\tif len(row.LiveEvidenceObjectIDs) == 0 && row.SourceItemKey != "cand-0" {',
               GATE + "tier3-no-live", "KILLED", "narrowing: excuses the cand-0 live gap only"),
        Mutant("N-t3-missing",
               '\tfor candidate := range included {\n\t\tif !coveredCandidates[candidate] {',
               '\tfor candidate := range included {\n\t\tif !coveredCandidates[candidate] && candidate != "cand-0" {',
               GATE + "tier3-missing-row", "KILLED", "narrowing: excuses the cand-0 row only"),
        Mutant("N-t3-staged-outside", '\t\tif !stagedBlobs[evidence.String()] {',
               '\t\tif !stagedBlobs[evidence.String()] && evidence.String() != "' + PHANTOM + '" {',
               GATE + "tier3-staged-outside", "KILLED", "narrowing: admits the phantom staged claim only"),
        Mutant("N-t3-live-outside", '\t\tif !liveBlobs[evidence.String()] {',
               '\t\tif !liveBlobs[evidence.String()] && evidence.String() != "' + PHANTOM + '" {',
               GATE + "tier3-live-outside", "KILLED", "narrowing: admits the phantom live claim only"),
        Mutant("N-plan-staged", '\tif history.Staged.Manifest().ProjectionPlanID != plan.PlanID {',
               '\tif history.Staged.Manifest().ProjectionPlanID != plan.PlanID && (history.Staged.Manifest().ProjectionPlanID.String() != "' + PLANA + '" || plan.PlanID.String() != "' + PLANB + '") {',
               GATE + "staged-plan-mismatch", "KILLED", "narrowing: admits the (plan-A, plan-B) pair only"),
        Mutant("N-plan-live", '\tif history.Live.Manifest().ProjectionPlanID != plan.PlanID {',
               '\tif history.Live.Manifest().ProjectionPlanID != plan.PlanID && (history.Live.Manifest().ProjectionPlanID.String() != "' + PLANA + '" || plan.PlanID.String() != "' + PLANB + '") {',
               GATE + "live-plan-mismatch", "KILLED", "narrowing: admits the (plan-A, plan-B) pair only"),
        Mutant("N-projected-staged", '\tif history.Staged.Manifest().ProjectedObjectManifestID != projected.ManifestID {',
               '\tif history.Staged.Manifest().ProjectedObjectManifestID != projected.ManifestID && (history.Staged.Manifest().ProjectedObjectManifestID.String() != "' + PROJA + '" || projected.ManifestID.String() != "' + PROJB + '") {',
               GATE + "staged-projected-mismatch", "KILLED", "narrowing: admits the (proj-A, proj-B) pair only"),
        Mutant("N-projected-live", '\tif history.Live.Manifest().ProjectedObjectManifestID != projected.ManifestID {',
               '\tif history.Live.Manifest().ProjectedObjectManifestID != projected.ManifestID && (history.Live.Manifest().ProjectedObjectManifestID.String() != "' + PROJA + '" || projected.ManifestID.String() != "' + PROJB + '") {',
               GATE + "live-projected-mismatch", "KILLED", "narrowing: admits the (proj-A, proj-B) pair only"),
        Mutant("N-native-sessions", '\tif staged.ObservedTargetNativeSession != live.ObservedTargetNativeSession {',
               '\tif staged.ObservedTargetNativeSession != live.ObservedTargetNativeSession && live.ObservedTargetNativeSession != "other-session" {',
               GATE + "reads-differ-sessions", "KILLED", "narrowing: admits the other-session live read only"),
        Mutant("N-plan-expects", '\tif plan.ExpectedTargetNativeSession != staged.ObservedTargetNativeSession {',
               '\tif plan.ExpectedTargetNativeSession != staged.ObservedTargetNativeSession && staged.ObservedTargetNativeSession != "other-session" {',
               GATE + "plan-expects-other", "KILLED", "narrowing: admits other-session reads only"),
        Mutant("N-projected-expects", '\tif projected.ExpectedTargetNativeSession != staged.ObservedTargetNativeSession {',
               '\tif projected.ExpectedTargetNativeSession != staged.ObservedTargetNativeSession && staged.ObservedTargetNativeSession != "other-session" {',
               GATE + "projected-expects-other", "KILLED", "narrowing: admits other-session reads only"),
        Mutant("N-fid-staged-bind",
               '\tif fidelity.StagedReadBackEvidenceManifestID == nil ||\n\t\t*fidelity.StagedReadBackEvidenceManifestID != history.Staged.ManifestID() {',
               '\tif fidelity.StagedReadBackEvidenceManifestID == nil ||\n\t\t*fidelity.StagedReadBackEvidenceManifestID != history.Staged.ManifestID() && (fidelity.StagedReadBackEvidenceManifestID == nil ||\n\t\t\tfidelity.StagedReadBackEvidenceManifestID.String() != "' + PHANTOM + '") {',
               GATE + "fidelity-staged-unbound", "KILLED", "narrowing: admits the phantom staged binding only"),
        Mutant("N-fid-live-bind",
               '\tif fidelity.LiveReadBackEvidenceManifestID == nil ||\n\t\t*fidelity.LiveReadBackEvidenceManifestID != history.Live.ManifestID() {',
               '\tif fidelity.LiveReadBackEvidenceManifestID == nil ||\n\t\t*fidelity.LiveReadBackEvidenceManifestID != history.Live.ManifestID() && (fidelity.LiveReadBackEvidenceManifestID == nil ||\n\t\t\tfidelity.LiveReadBackEvidenceManifestID.String() != "' + PHANTOM + '") {',
               GATE + "fidelity-live-unbound", "KILLED", "narrowing: admits the phantom live binding only"),
        Mutant("N-fid-tuple-staged", other_staged[0], other_staged[1],
               GATE + "fidelity-tuple-staged", "KILLED", "narrowing: admits the relux.other fidelity target only"),
        Mutant("N-fid-tuple-live", other_live[0], other_live[1],
               GATE + "fidelity-tuple-live", "KILLED", "narrowing: admits the relux.other live observation only"),
        Mutant("N-validation-tuple", other_validation[0], other_validation[1],
               GATE + "validation-tuple-staged", "KILLED", "narrowing: admits the relux.other report target only"),
        Mutant("N-validation-native", '\tif expected != history.Staged.Manifest().ObservedTargetNativeSession {',
               '\tif expected != history.Staged.Manifest().ObservedTargetNativeSession && expected != "other-session" {',
               GATE + "validation-expects-other", "KILLED", "narrowing: admits the other-session expectation only"),
        Mutant("N-expected-fidelity", '\tif expectedFidelity != fidelity.ReportID {',
               '\tif expectedFidelity != fidelity.ReportID && expectedFidelity.String() != "' + PHANTOM + '" {',
               GATE + "expected-fidelity-mismatch", "KILLED", "narrowing: admits the phantom expectation only"),
        Mutant("T-mode-staged", '\t\treturn refuse("reconciliation staged read is not a staged manifest")',
               '\t\treturn refuse("reconciliation live read is not a live manifest")',
               GATE + "staged-not-staged", "KILLED", "token-preserving: staged token kept, live detail reported"),
        Mutant("T-loss-branch", '\treturn disposition == "omitted" || disposition == "unrecoverable"',
               '\treturn disposition == "omitted" || disposition == "unrecoverable" || disposition == "exact"',
               GATE + "tier3-loss-claims-staged", "KILLED", "token-preserving: both loss tokens kept, exact rows misclassified as loss; the gate test fails on rows[0] (a conjunction here would die at vet, never reaching the test)"),
        Mutant("T-tier2-synth", '\t\t\tif link.Normalization == "synthesized" {',
               '\t\t\tif link.Normalization == "synthesized" && link.Raw == "" {',
               GATE + "tier2-synth-normalization", "KILLED", "token-preserving: synthesized token kept, unreachable arm"),
        Mutant("T-tier3-synth", '\t\tif row.Disposition == "synthesized" {\n\t\t\tfor _, evidence := range row.SourceEvidenceIDs {',
               '\t\tif row.Disposition == "synthesized" && row.SourceItemKey == "" {\n\t\t\tfor _, evidence := range row.SourceEvidenceIDs {',
               GATE + "tier3-synth-claims", "KILLED", "token-preserving: synthesized token kept, unreachable arm"),
        Mutant("C-doc-comment", '// CandidateLink reconciles one captured candidate at tier 1: exactly',
               '// CandidateLink reconciles one captured candidate at tier 1 - exactly',
               "FULLSUITE", "SURVIVED", "control: comment-only edit must survive the full suite"),
    ]


def run_case(log_dir, mutant, full_suite):
    target = REPO / TARGET
    original = target.read_text()
    if original.count(mutant.find) != 1:
        return ("ERROR", f"find occurs {original.count(mutant.find)} times, want 1")
    target.write_text(original.replace(mutant.find, mutant.replace))
    try:
        if full_suite or mutant.run == "FULLSUITE":
            cmd = ["go", "test", "./" + PACKAGE + "/", "-count=1"]
        else:
            cmd = ["go", "test", "./" + PACKAGE + "/", "-run", mutant.run + "$", "-count=1", "-v"]
        started = time.time()
        proc = subprocess.run(cmd, cwd=REPO, capture_output=True, text=True, timeout=300)
        elapsed = time.time() - started
        log_path = log_dir / (mutant.name + ("-suite.log" if full_suite else ".log"))
        log_path.write_text(f"$ {' '.join(cmd)}\nexit={proc.returncode} elapsed={elapsed:.1f}s\n--- stdout ---\n{proc.stdout}\n--- stderr ---\n{proc.stderr}\n")
        if proc.returncode == 0:
            return ("SURVIVED", f"exit 0 in {elapsed:.1f}s")
        if "FAIL" in (proc.stdout + proc.stderr):
            return ("KILLED", f"exit {proc.returncode} with FAIL in {elapsed:.1f}s")
        return ("ERROR", f"exit {proc.returncode} without FAIL in {elapsed:.1f}s")
    except subprocess.TimeoutExpired:
        return ("ERROR", "timeout after 300s")
    finally:
        target.write_text(original)


def main(argv):
    log_dir = REPO / ".temp" / "mutant_logs"
    wanted = []
    args = list(argv)
    while args:
        if args[0] == "--log-dir":
            log_dir = Path(args[1])
            args = args[2:]
        else:
            wanted.append(args.pop(0))
    rows = mutants()
    # N-t2-raw-digest and N-t2-canon-digest are repaired below: the
    # digest-error branches need condition patches, not return
    # patches (the table above holds placeholders for them).
    patched = []
    for mutant in rows:
        if mutant.name == "N-t2-raw-digest":
            mutant = Mutant("N-t2-raw-digest",
                            '\t\traw, err := scalar.ParseDigest(link.Raw)\n\t\tif err != nil {\n\t\t\treturn nil, nil, nil, refuse("tier-2 raw item %q is not a digest: %v", link.Raw, err)',
                            '\t\traw, err := scalar.ParseDigest(link.Raw)\n\t\tif err != nil && link.Raw != "bogus" {\n\t\t\treturn nil, nil, nil, refuse("tier-2 raw item %q is not a digest: %v", link.Raw, err)',
                            GATE + "tier2-raw-not-digest", "KILLED",
                            "narrowing: admits the bogus tier-2 raw only")
        if mutant.name == "N-t2-canon-digest":
            mutant = Mutant("N-t2-canon-digest",
                            '\t\t\tcanonical, err := scalar.ParseDigest(link.Canonical)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, nil, nil, refuse("tier-2 canonical item %q is not a digest: %v", link.Canonical, err)',
                            '\t\t\tcanonical, err := scalar.ParseDigest(link.Canonical)\n\t\t\tif err != nil && link.Canonical != "bogus" {\n\t\t\t\treturn nil, nil, nil, refuse("tier-2 canonical item %q is not a digest: %v", link.Canonical, err)',
                            GATE + "tier2-canon-not-digest", "KILLED",
                            "narrowing: admits the bogus canonical only")
        patched.append(mutant)
    rows = patched
    if wanted:
        rows = [m for m in rows if m.name in wanted]
        if not rows:
            print("no such mutant", file=sys.stderr)
            return 2
    log_dir.mkdir(parents=True, exist_ok=True)
    failures = []
    print(f"{'mutant':28} {'verdict':9} detail")
    for mutant in rows:
        verdict, detail = run_case(log_dir, mutant, False)
        print(f"{mutant.name:28} {verdict:9} {detail}", flush=True)
        if verdict != mutant.expect:
            failures.append((mutant.name, verdict, mutant.expect))
        if mutant.name.startswith("T-") and verdict == "KILLED":
            suite_verdict, suite_detail = run_case(log_dir, mutant, True)
            print(f"{mutant.name + '+suite':28} {suite_verdict:9} {suite_detail}", flush=True)
            if suite_verdict != "KILLED":
                failures.append((mutant.name + "+suite", suite_verdict, "KILLED"))
    if failures:
        print(f"\n{len(failures)} unexpected verdict(s):", file=sys.stderr)
        for name, got, want in failures:
            print(f"  {name}: got {got}, want {want}", file=sys.stderr)
        return 1
    print("\nall verdicts as expected")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
