#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/clonefidelity.

Every gate ships a narrowing mutant: the gate stays present and is
weakened to admit exactly one member of the class it must reject, and
the named killer test must fail. A delete-only mutant proves only that
the gate exists and is not accepted as evidence.

Each row runs M (mutate one production file in place), R (run the
scoped Go killer through a production entry), V (verdict from the Go
exit code plus a FAIL kill-line: KILLED iff R fails with a test
failure line, SURVIVED iff R passes, ERROR for anything else --
unapplied patches, build breaks, timeouts). M failures and R
infrastructure errors are ERROR, never kills or survivals. The file
is restored from its backup after every row, pass or fail.

Shared gates (one implementation, two call sites: the vocabulary
predicates, the reason/canonical/order helpers, the counts
derivation, the breakdown sums, the row-count gate, the strict
frame) carry one plant each with a dual-entry killer.

Rows N-disposition-route and N-reason-core-route are the
token-preserving attacks: the searched-for token still matches, but
the decision changes. Their killers execute the behavioral oracle
suites through both production entries.

Row N-record-synth-canonical is labelled single-member class: the
gate's reject class holds exactly (synthesized, has-canonical), so
the plant admits the whole class, which is one member.

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/clonefidelity/testdata/mutant_harness.py [--log-dir DIR] [MUTANT ...]
"""

import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PACKAGE = "internal/clonefidelity"


@dataclass(frozen=True)
class Mutant:
    name: str
    path: str
    find: str
    replace: str
    count: int
    run: str
    expect: str
    note: str


MUTANTS = [
    Mutant(
        name="N-disposition-vocab",
        path=f"{PACKAGE}/vocab.go",
        find='''func ValidDisposition(disposition string) bool {
	for _, allowed := range dispositions {
		if disposition == allowed {
			return true
		}
	}
	return false
}''',
        replace='''func ValidDisposition(disposition string) bool {
	for _, allowed := range dispositions {
		if disposition == allowed {
			return true
		}
	}
	if disposition == "EXACT" {
		return true
	}
	return false
}''',
        count=1,
        run="TestDispositionVocabularyOracle",
        expect="KILLED",
        note="Shared gate: admits exactly EXACT past the disposition vocabulary at both entries; every other non-member still refuses.",
    ),
    Mutant(
        name="N-profile-vocab",
        path=f"{PACKAGE}/vocab.go",
        find='''func ValidProfile(profile string) bool {
	for _, allowed := range profiles {
		if profile == allowed {
			return true
		}
	}
	return false
}''',
        replace='''func ValidProfile(profile string) bool {
	for _, allowed := range profiles {
		if profile == allowed {
			return true
		}
	}
	if profile == "Maximal_Safe" {
		return true
	}
	return false
}''',
        count=1,
        run="TestProfileVocabularyOracle",
        expect="KILLED",
        note="Shared gate: admits exactly Maximal_Safe past the profile vocabulary at both entries.",
    ),
    Mutant(
        name="N-strategy-vocab",
        path=f"{PACKAGE}/vocab.go",
        find='''func ValidStrategy(strategy string) bool {
	for _, allowed := range strategies {
		if strategy == allowed {
			return true
		}
	}
	return false
}''',
        replace='''func ValidStrategy(strategy string) bool {
	for _, allowed := range strategies {
		if strategy == allowed {
			return true
		}
	}
	if strategy == "archive-only" {
		return true
	}
	return false
}''',
        count=1,
        run="TestStrategyVocabularyOracle",
        expect="KILLED",
        note="Admits exactly archive-only past the strategy registry; the predicate is the production entry.",
    ),
    Mutant(
        name="N-reason-core",
        path=f"{PACKAGE}/vocab.go",
        find='''func IsCoreReasonCode(code string) bool {
	for _, core := range coreReasons {
		if code == core {
			return true
		}
	}
	return false
}''',
        replace='''func IsCoreReasonCode(code string) bool {
	for _, core := range coreReasons {
		if code == core {
			return true
		}
	}
	if code == "unknown_native_even" {
		return true
	}
	return false
}''',
        count=1,
        run="TestReasonVocabularyOracle",
        expect="KILLED",
        note="Shared gate: admits exactly the truncated near-miss unknown_native_even as a core reason at both entries.",
    ),
    Mutant(
        name="N-reason-extension",
        path=f"{PACKAGE}/vocab.go",
        find='''func isExtensionReason(code string) bool {
	return environ.CheckReverseDNS(code)
}''',
        replace='''func isExtensionReason(code string) bool {
	if code == "nodots" {
		return true
	}
	return environ.CheckReverseDNS(code)
}''',
        count=1,
        run="TestReasonVocabularyOracle",
        expect="KILLED",
        note="Shared gate: admits exactly nodots past the extension arm at both entries; every other malformed name still refuses.",
    ),
    Mutant(
        name="N-reason-core-route",
        path=f"{PACKAGE}/vocab.go",
        find='''func IsCoreReasonCode(code string) bool {
	for _, core := range coreReasons {
		if code == core {
			return true
		}
	}
	return false
}''',
        replace='''func IsCoreReasonCode(code string) bool {
	for _, core := range coreReasons {
		if code == core {
			if code == "unknown_native_event" {
				continue
			}
			return true
		}
	}
	return false
}''',
        count=1,
        run="TestReasonVocabularyOracle",
        expect="KILLED",
        note="Token-preserving: unknown_native_event still matches the table but falls through to the extension check and refuses; the behavioral oracle pins its admission through both entries.",
    ),
    Mutant(
        name="N-record-exact-reasons",
        path=f"{PACKAGE}/record.go",
        find='''	if disposition == "exact" {
		if reasonCount != 0 {
			return invalid("%s exact carries %d reason_codes, want an empty reason set", owner, reasonCount)
		}
		return nil
	}''',
        replace='''	if disposition == "exact" {
		if reasonCount != 0 && reasonCount != 1 {
			return invalid("%s exact carries %d reason_codes, want an empty reason set", owner, reasonCount)
		}
		return nil
	}''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Shared gate: admits exactly exact rows with one reason at both entries; exact with 2+ still refuses.",
    ),
    Mutant(
        name="N-record-nonexact-empty",
        path=f"{PACKAGE}/record.go",
        find='''	if reasonCount < 1 {
		return invalid("%s %s carries no reason_codes, want at least one reason", owner, disposition)
	}''',
        replace='''	if reasonCount < 1 && disposition != "semantic" {
		return invalid("%s %s carries no reason_codes, want at least one reason", owner, disposition)
	}''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Shared gate: admits exactly semantic rows with no reasons at both entries; every other non-exact disposition still refuses.",
    ),
    Mutant(
        name="N-record-synth-canonical",
        path=f"{PACKAGE}/record.go",
        find='''	if disposition == "synthesized" && hasCanonical {
		return invalid("%s synthesized carries a source canonical object", owner)
	}''',
        replace='''	if disposition == "synthesized" && hasCanonical {
		return nil
	}''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Labelled single-member class: the reject class holds exactly (synthesized, has-canonical), so admitting it admits one member at both entries.",
    ),
    Mutant(
        name="N-disposition-route",
        path=f"{PACKAGE}/report.go",
        find='''	case "exact":
		counts.Exact++''',
        replace='''	case "exact":
		counts.Semantic++''',
        count=1,
        run="TestCountsDerived",
        expect="KILLED",
        note="Token-preserving: the exact label still matches but folds into Semantic; the behavioral killer pins literal totals through both entries.",
    ),
    Mutant(
        name="N-roworder-synth-after",
        path=f"{PACKAGE}/report.go",
        find='''		if seenSynthesized {
			return invalid("fidelity report dispositions order a source row after synthesized rows")
		}''',
        replace='''		if seenSynthesized && row.SourceItemKey != "k2" {
			return invalid("fidelity report dispositions order a source row after synthesized rows")
		}''',
        count=1,
        run="TestRowOrdering",
        expect="KILLED",
        note="Shared gate: admits exactly the k2 source row after a synthesized row at both entries.",
    ),
    Mutant(
        name="N-roworder-sorted",
        path=f"{PACKAGE}/report.go",
        find='''		if previousSource != "" && row.SourceItemKey <= previousSource {''',
        replace='''		if previousSource != "" && row.SourceItemKey <= previousSource && row.SourceItemKey != "k1" {''',
        count=1,
        run="TestRowOrdering",
        expect="KILLED",
        note="Shared gate: admits exactly k1 after k2 in the source group at both entries; every other unsorted pair still refuses.",
    ),
    Mutant(
        name="N-roworder-dupkey",
        path=f"{PACKAGE}/report.go",
        find='''		if seen[row.SourceItemKey] {''',
        replace='''		if seen[row.SourceItemKey] && row.SourceItemKey != "k1" {''',
        count=1,
        run="TestRowOrdering",
        expect="KILLED",
        note="Shared gate: admits exactly the duplicate k1 key at both entries; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-rowcount-max",
        path=f"{PACKAGE}/report.go",
        find='''	if count < 1 || count > maxReportRows {''',
        replace='''	if count < 1 || count > maxReportRows+1 {''',
        count=1,
        run="TestRowCountEdges",
        expect="KILLED",
        note="Shared gate: admits exactly 1000001 rows; the entry killer refuses them with the count detail (the factored helper test reddens too).",
    ),
    Mutant(
        name="N-rowcount-min",
        path=f"{PACKAGE}/report.go",
        find='''	if count < 1 || count > maxReportRows {''',
        replace='''	if count < 0 || count > maxReportRows {''',
        count=1,
        run="TestRowCountEdges",
        expect="KILLED",
        note="Shared gate: admits exactly zero rows; the entry killer refuses the empty dispositions with the count detail.",
    ),
    Mutant(
        name="N-eventkind-sum",
        path=f"{PACKAGE}/report.go",
        find='''		if sum != totals.countOf(disposition) {
			return invalid("%s reconciles %d %s rows, want %d from the disposition rows", owner, sum, disposition, totals.countOf(disposition))
		}''',
        replace='''		if sum != totals.countOf(disposition) && !(owner == "fidelity report event_kind_counts" && sum == totals.countOf(disposition)+1) {
			return invalid("%s reconciles %d %s rows, want %d from the disposition rows", owner, sum, disposition, totals.countOf(disposition))
		}''',
        count=1,
        run="TestEventKindBreakdown",
        expect="KILLED",
        note="Shared gate: admits exactly a +1 event-kind sum at both entries; content sums and larger drifts still refuse.",
    ),
    Mutant(
        name="N-contentblock-sum",
        path=f"{PACKAGE}/report.go",
        find='''		if sum != totals.countOf(disposition) {
			return invalid("%s reconciles %d %s rows, want %d from the disposition rows", owner, sum, disposition, totals.countOf(disposition))
		}''',
        replace='''		if sum != totals.countOf(disposition) && !(owner == "fidelity report content_block_counts" && sum == totals.countOf(disposition)+1) {
			return invalid("%s reconciles %d %s rows, want %d from the disposition rows", owner, sum, disposition, totals.countOf(disposition))
		}''',
        count=1,
        run="TestContentBlockBreakdown",
        expect="KILLED",
        note="Shared gate: admits exactly a +1 content-block sum at both entries; event sums and larger drifts still refuse.",
    ),
    Mutant(
        name="N-frame-report",
        path=f"{PACKAGE}/decode.go",
        find='''func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	members, fault := environ.DecodeStrictObject(data)
	if fault != nil {
		return nil, &frameFault{detail: fault.Detail, member: fault.Member}
	}
	return members, nil
}''',
        replace='''func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	members, fault := environ.DecodeStrictObject(data)
	if fault != nil {
		if fault.Detail == "duplicate member" && fault.Member == "scope" {
			var lenient map[string]json.RawMessage
			if err := json.Unmarshal(data, &lenient); err == nil {
				return lenient, nil
			}
		}
		return nil, &frameFault{detail: fault.Detail, member: fault.Member}
	}
	return members, nil
}''',
        count=1,
        run="TestReportFraming",
        expect="KILLED",
        note="Shared gate: admits exactly a duplicate scope member past the strict frame into a lenient decode; every other frame fault still refuses.",
    ),
    Mutant(
        name="N-frame-row",
        path=f"{PACKAGE}/decode.go",
        find='''func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	members, fault := environ.DecodeStrictObject(data)
	if fault != nil {
		return nil, &frameFault{detail: fault.Detail, member: fault.Member}
	}
	return members, nil
}''',
        replace='''func decodeStrictObject(data []byte) (map[string]json.RawMessage, *frameFault) {
	members, fault := environ.DecodeStrictObject(data)
	if fault != nil {
		if fault.Detail == "duplicate member" && fault.Member == "source_class" {
			var lenient map[string]json.RawMessage
			if err := json.Unmarshal(data, &lenient); err == nil {
				return lenient, nil
			}
		}
		return nil, &frameFault{detail: fault.Detail, member: fault.Member}
	}
	return members, nil
}''',
        count=1,
        run="TestReportFraming",
        expect="KILLED",
        note="Shared gate: admits exactly a duplicate source_class member past the row frame into a lenient decode; the report duplicate still refuses.",
    ),
    Mutant(
        name="N-decode-key-bound",
        path=f"{PACKAGE}/record.go",
        find='''	key, ok := checkStringBounds(members["source_item_key"], 1, 512)''',
        replace='''	key, ok := checkStringBounds(members["source_item_key"], 1, 513)''',
        count=1,
        run="TestRecordFieldBounds",
        expect="KILLED",
        note="Decode side: admits exactly 513-character item keys; 514+ still refuse and the build side is untouched.",
    ),
    Mutant(
        name="N-decode-reason-item-bound",
        path=f"{PACKAGE}/record.go",
        find='''	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)''',
        replace='''	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 129, 0, 128)''',
        count=1,
        run="TestRecordFieldBounds",
        expect="KILLED",
        note="Decode side: admits exactly 129-character reason items past the item bound; the killer pins the bound message (the plant moves the refusal to the vocabulary gate).",
    ),
    Mutant(
        name="N-decode-reasons-count",
        path=f"{PACKAGE}/record.go",
        find='''	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)''',
        replace='''	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 129)''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Decode side: admits exactly 129 reasons past the count bound; the grid killer pins the bound message (downstream gates refuse with their own details).",
    ),
    Mutant(
        name="N-decode-evidence-empty",
        path=f"{PACKAGE}/record.go",
        find='''	evidence, ok := checkSortedUniqueDigests(members["source_evidence_ids"], 1, 65536)''',
        replace='''	evidence, ok := checkSortedUniqueDigests(members["source_evidence_ids"], 0, 65536)''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Decode side: admits exactly empty source evidence past the shape gate; the grid killer pins the shape message.",
    ),
    Mutant(
        name="N-decode-counts-cell",
        path=f"{PACKAGE}/report.go",
        find='''		if sealedCounts.countOf(disposition) != counts.countOf(disposition) {''',
        replace='''		if sealedCounts.countOf(disposition) != counts.countOf(disposition) && !(disposition == "exact" && sealedCounts.countOf(disposition) == counts.countOf(disposition)+1) {''',
        count=1,
        run="TestCountsDerived",
        expect="KILLED",
        note="Decode side: admits exactly counts.exact one above derived; every other cell and drift still refuses.",
    ),
    Mutant(
        name="N-decode-reasoncounts-cell",
        path=f"{PACKAGE}/report.go",
        find='''	for reason, sealed := range reasonCounts {
		derived, present := derivedReasons[reason]
		if !present || sealed != derived {''',
        replace='''	for reason, sealed := range reasonCounts {
		derived, present := derivedReasons[reason]
		if !present || sealed != derived && !(reason == "credential_excluded" && sealed == derived+1) {''',
        count=1,
        run="TestReasonCountsDerived",
        expect="KILLED",
        note="Decode side: admits exactly credential_excluded one above derived; every other reason cell still refuses.",
    ),
    Mutant(
        name="N-decode-reasoncounts-zero",
        path=f"{PACKAGE}/report.go",
        find='''		value, ok := checkUint53Bounds(members[key], 1, maxUint53)
		if !ok {
			return nil, invalid("%s[%q] is not a uint53 above zero", owner, key)
		}''',
        replace='''		value, ok := checkUint53Bounds(members[key], 1, maxUint53)
		if !ok && !(key == "credential_excluded" && string(members[key]) == "0") {
			return nil, invalid("%s[%q] is not a uint53 above zero", owner, key)
		}''',
        count=1,
        run="TestReasonCountsDerived",
        expect="KILLED",
        note="Decode side: admits exactly a zero credential_excluded value past the above-zero gate; the killer pins the shape message (reconciliation refuses downstream).",
    ),
    Mutant(
        name="N-decode-eventkind-keys",
        path=f"{PACKAGE}/report.go",
        find='''	if name, unknown := unknownMember(members, allowed); unknown {
		return nil, invalid("%s carry unknown key %q", owner, name)
	}''',
        replace='''	if name, unknown := unknownMember(members, allowed); unknown && !(member == "event_kind_counts" && name == "frobnicate") {
		return nil, invalid("%s carry unknown key %q", owner, name)
	}''',
        count=1,
        run="TestEventKindBreakdown",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate event-kind key (ignored downstream, so the digest kills); content keys still refuse.",
    ),
    Mutant(
        name="N-decode-contentblock-keys",
        path=f"{PACKAGE}/report.go",
        find='''	if name, unknown := unknownMember(members, allowed); unknown {
		return nil, invalid("%s carry unknown key %q", owner, name)
	}''',
        replace='''	if name, unknown := unknownMember(members, allowed); unknown && !(member == "content_block_counts" && name == "frobnicate") {
		return nil, invalid("%s carry unknown key %q", owner, name)
	}''',
        count=1,
        run="TestContentBlockBreakdown",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate content-block key (ignored downstream, so the digest kills); event keys still refuse.",
    ),
    Mutant(
        name="N-decode-breakdown-missing",
        path=f"{PACKAGE}/report.go",
        find='''	if name, missing := missingMember(members, dispositions); missing {
		return FidelityCounts{}, invalid("%s[%q] misses a required member %q", owner, key, name)
	}''',
        replace='''	if name, missing := missingMember(members, dispositions); missing && !(key == "usage" && name == "exact") {
		return FidelityCounts{}, invalid("%s[%q] misses a required member %q", owner, key, name)
	}''',
        count=1,
        run="TestEventKindBreakdown",
        expect="KILLED",
        note="Decode side: admits exactly the usage cell missing exact; the killer pins the missing-member message (the nil cell refuses at the uint53 gate).",
    ),
    Mutant(
        name="N-decode-bytekeys",
        path=f"{PACKAGE}/report.go",
        find='''	if name, unknown := unknownMember(members, allowed); unknown {
		return nil, invalid("%s carry unknown disposition %q", owner, name)
	}''',
        replace='''	if name, unknown := unknownMember(members, allowed); unknown && name != "frobnicate" {
		return nil, invalid("%s carry unknown disposition %q", owner, name)
	}''',
        count=1,
        run="TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate byte key (ignored downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-required-keys",
        path=f"{PACKAGE}/report.go",
        find='''	sort.Strings(keys)
	for _, class := range keys {
		if !clonebundle.ValidCaptureClass(class) {
			return nil, invalid("%s carry unknown class %q", owner, class)''',
        replace='''	sort.Strings(keys)
	for _, class := range keys {
		if !clonebundle.ValidCaptureClass(class) && class != "frobnicate" {
			return nil, invalid("%s carry unknown class %q", owner, class)''',
        count=1,
        run="TestRequiredDispositions",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate policy class (sealed downstream, so the digest kills); the build side is untouched.",
    ),
    Mutant(
        name="N-decode-required-values",
        path=f"{PACKAGE}/report.go",
        find='''		elements, ok := decodeArray(members[class])
		if !ok || len(elements) < 1 {
			return nil, invalid("%s[%q] carry no dispositions, want a non-empty set", owner, class)
		}''',
        replace='''		elements, ok := decodeArray(members[class])
		if !ok || len(elements) < 1 && class != "unknown" {
			return nil, invalid("%s[%q] carry no dispositions, want a non-empty set", owner, class)
		}''',
        count=1,
        run="TestRequiredDispositions",
        expect="KILLED",
        note="Decode side: admits exactly the empty unknown set; every other empty set still refuses.",
    ),
    Mutant(
        name="N-decode-scope",
        path=f"{PACKAGE}/report.go",
        find='''	archive := false
	switch scope {
	case "archive":
		archive = true
	case "target":
	default:
		return FidelityReport{}, invalid("%s scope is outside archive|target", owner)
	}''',
        replace='''	archive := false
	switch scope {
	case "archive":
		archive = true
	case "target":
	default:
		if scope != "staging" {
			return FidelityReport{}, invalid("%s scope is outside archive|target", owner)
		}
	}''',
        count=1,
        run="TestReportMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly the staging scope (read as target downstream, so the digest kills); the build side is untouched.",
    ),
    Mutant(
        name="N-decode-scope-profile",
        path=f"{PACKAGE}/report.go",
        find='''	if archive && profile != "archive_only" {''',
        replace='''	if archive && profile != "archive_only" && profile != "maximal_safe" {''',
        count=1,
        run="TestArchiveTargetBranches",
        expect="KILLED",
        note="Decode side: admits exactly archive with maximal_safe; every other archive profile still refuses.",
    ),
    Mutant(
        name="N-decode-archive-plan",
        path=f"{PACKAGE}/report.go",
        find='''		if !isNull(members["projection_plan_id"]) {''',
        replace='''		if !isNull(members["projection_plan_id"]) && string(members["projection_plan_id"]) != "\\"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\\"" {''',
        count=1,
        run="TestArchiveTargetBranches",
        expect="KILLED",
        note="Decode side: admits exactly the fixture plan digest in archive scope; every other non-null plan still refuses.",
    ),
    Mutant(
        name="N-decode-target-plan",
        path=f"{PACKAGE}/report.go",
        find='''		id, ok := checkDigest(members["projection_plan_id"])
		if !ok {
			return FidelityReport{}, invalid("%s projection_plan_id is not a digest for target", owner)
		}''',
        replace='''		id, ok := checkDigest(members["projection_plan_id"])
		if !ok && !isNull(members["projection_plan_id"]) {
			return FidelityReport{}, invalid("%s projection_plan_id is not a digest for target", owner)
		}''',
        count=1,
        run="TestArchiveTargetBranches",
        expect="KILLED",
        note="Decode side: admits exactly a null plan in target scope; malformed non-null plans still refuse.",
    ),
    Mutant(
        name="N-decode-archive-targetbools",
        path=f"{PACKAGE}/report.go",
        find='''	if archive {
		if continuable {
			return FidelityReport{}, invalid("%s target_semantically_continuable is true for archive", owner)
		}''',
        replace='''	if archive {
		if continuable && resumable {
			return FidelityReport{}, invalid("%s target_semantically_continuable is true for archive", owner)
		}''',
        count=1,
        run="TestArchiveTargetBranches",
        expect="KILLED",
        note="Decode side: admits exactly continuable-without-resumable in archive scope; every other true combination still refuses.",
    ),
    Mutant(
        name="N-decode-selfdigest",
        path=f"{PACKAGE}/decode.go",
        find='''	if calculated != claimed {
		return invalid("%s claim %q does not match omit-self digest %q", selfField, claimed.String(), calculated.String())
	}''',
        replace='''	if calculated != claimed && claimed.String() != "sha256:0000000000000000000000000000000000000000000000000000000000000000" {
		return invalid("%s claim %q does not match omit-self digest %q", selfField, claimed.String(), calculated.String())
	}''',
        count=1,
        run="TestReportIdentityIndependent",
        expect="KILLED",
        note="Admits exactly the all-zero identity claim; every other mismatch still refuses.",
    ),
    Mutant(
        name="N-decode-schema",
        path=f"{PACKAGE}/report.go",
        find='''	if !ok || schema != fidelityReportSchema {''',
        replace='''	if !ok || schema != fidelityReportSchema && schema != "urn:ax:schema:fidelity-repor" {''',
        count=1,
        run="TestReportMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly the truncated schema urn (sealed downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-member-unknown",
        path=f"{PACKAGE}/report.go",
        find='''	if name, unknown := unknownMember(members, fidelityReportMembers); unknown {''',
        replace='''	if name, unknown := unknownMember(members, fidelityReportMembers); unknown && name != "frobnicate" {''',
        count=1,
        run="TestReportMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate top-level member (ignored downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-member-missing",
        path=f"{PACKAGE}/report.go",
        find='''	if name, missing := missingMember(members, fidelityReportRequired); missing {''',
        replace='''	if name, missing := missingMember(members, fidelityReportRequired); missing && name != "counts" {''',
        count=1,
        run="TestReportMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly a missing counts member; the killer pins the missing-member message (the nil counts refuse at the shape gate).",
    ),
    Mutant(
        name="N-decode-record-unknown",
        path=f"{PACKAGE}/record.go",
        find='''	if name, unknown := unknownMember(members, dispositionRecordMembers); unknown {''',
        replace='''	if name, unknown := unknownMember(members, dispositionRecordMembers); unknown && name != "frobnicate" {''',
        count=1,
        run="TestRecordMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate row member (ignored downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-record-missing",
        path=f"{PACKAGE}/record.go",
        find='''	if name, missing := missingMember(members, dispositionRecordRequired); missing {''',
        replace='''	if name, missing := missingMember(members, dispositionRecordRequired); missing && name != "explanation" {''',
        count=1,
        run="TestRecordMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly a missing explanation; the killer pins the missing-member message (the nil member refuses at its bound gate).",
    ),
    Mutant(
        name="N-decode-counts-unknown",
        path=f"{PACKAGE}/report.go",
        find='''	if name, unknown := unknownMember(members, allowed); unknown {
		return FidelityCounts{}, invalid("%s carries unknown member %q", owner, name)
	}''',
        replace='''	if name, unknown := unknownMember(members, allowed); unknown && name != "frobnicate" {
		return FidelityCounts{}, invalid("%s carries unknown member %q", owner, name)
	}''',
        count=1,
        run="TestCountsMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate counts member (ignored downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-counts-missing",
        path=f"{PACKAGE}/report.go",
        find='''	if name, missing := missingMember(members, dispositions); missing {
		return FidelityCounts{}, invalid("%s misses a required member %q", owner, name)
	}''',
        replace='''	if name, missing := missingMember(members, dispositions); missing && name != "exact" {
		return FidelityCounts{}, invalid("%s misses a required member %q", owner, name)
	}''',
        count=1,
        run="TestCountsMemberSet",
        expect="KILLED",
        note="Decode side: admits exactly counts missing exact; the killer pins the missing-member message (the nil cell refuses at the uint53 gate).",
    ),
    Mutant(
        name="N-decode-sorted-reasons",
        path=f"{PACKAGE}/record.go",
        find='''	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] reason_codes are not sorted unique string[1..128][0..128]", owner, index)
	}''',
        replace='''	reasons, ok := checkSortedUniqueStrings(members["reason_codes"], 1, 128, 0, 128)
	if !ok {
		var lenient []string
		if json.Unmarshal(members["reason_codes"], &lenient) == nil && len(lenient) == 2 && lenient[0] == "unknown_native_event" && lenient[1] == "credential_excluded" {
			reasons, ok = lenient, true
		}
	}
	if !ok {
		return DispositionRecord{}, invalid("%s[%d] reason_codes are not sorted unique string[1..128][0..128]", owner, index)
	}''',
        count=1,
        run="TestRecordSortedUnique",
        expect="KILLED",
        note="Decode side: admits exactly the unsorted fixture pair past the strings order gate (valid downstream, so the digest kills); duplicates still refuse.",
    ),
    Mutant(
        name="N-decode-sorted-attestations",
        path=f"{PACKAGE}/report.go",
        find='''	attestations, ok := checkSortedUniqueDigests(members["adapter_attestations"], 0, 64)
	if !ok {
		return FidelityReport{}, invalid("%s adapter_attestations are not sorted unique digest[0..64]", owner)
	}''',
        replace='''	attestations, ok := checkSortedUniqueDigests(members["adapter_attestations"], 0, 64)
	if !ok {
		var lenient []string
		if json.Unmarshal(members["adapter_attestations"], &lenient) == nil && len(lenient) == 2 && lenient[0] == "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" && lenient[1] == "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
			first, firstErr := scalar.ParseDigest(lenient[0])
			second, secondErr := scalar.ParseDigest(lenient[1])
			if firstErr == nil && secondErr == nil {
				attestations, ok = []scalar.Digest{first, second}, true
			}
		}
	}
	if !ok {
		return FidelityReport{}, invalid("%s adapter_attestations are not sorted unique digest[0..64]", owner)
	}''',
        count=1,
        run="TestScalarGates",
        expect="KILLED",
        note="Decode side: admits exactly the unsorted hand-digest pair past the digests order gate (valid downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-uint53-counts",
        path=f"{PACKAGE}/report.go",
        find='''		value, ok := checkUint53Bounds(members[disposition], 0, maxUint53)
		if !ok {
			return FidelityCounts{}, invalid("%s %s is not a uint53", owner, disposition)
		}''',
        replace='''		value, ok := checkUint53Bounds(members[disposition], 0, maxUint53)
		if !ok && disposition == "exact" && string(members[disposition]) == "1.5" {
			value, ok = 1, true
		}
		if !ok {
			return FidelityCounts{}, invalid("%s %s is not a uint53", owner, disposition)
		}''',
        count=1,
        run="TestNumberModel",
        expect="KILLED",
        note="Decode side: admits exactly 1.5 as 1 in counts.exact (reconciles by coincidence, so the digest kills); every other non-uint53 still refuses.",
    ),
    Mutant(
        name="N-decode-digest",
        path=f"{PACKAGE}/report.go",
        find='''	snapshot, ok := checkDigest(members["source_snapshot_digest"])
	if !ok {
		return FidelityReport{}, invalid("%s source_snapshot_digest is not a digest", owner)
	}''',
        replace='''	snapshot, ok := checkDigest(members["source_snapshot_digest"])
	if !ok {
		if text, isString := rawString(members["source_snapshot_digest"]); isString && text == "sha256:xyz" {
			snapshot, _ = scalar.ParseDigest("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
			ok = true
		}
	}
	if !ok {
		return FidelityReport{}, invalid("%s source_snapshot_digest is not a digest", owner)
	}''',
        count=1,
        run="TestScalarGates",
        expect="KILLED",
        note="Decode side: admits exactly sha256:xyz as the snapshot digest (substituted downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-uuid",
        path=f"{PACKAGE}/report.go",
        find='''	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		return FidelityReport{}, invalid("%s operation_id is not a UUIDv7", owner)
	}''',
        replace='''	operation, ok := checkUUIDv7(members["operation_id"])
	if !ok {
		if text, isString := rawString(members["operation_id"]); isString && text == "not-a-uuid" {
			operation, _ = scalar.ParseUUIDv7("0198f4c8-8e50-7f66-8f70-000000000000")
			ok = true
		}
	}
	if !ok {
		return FidelityReport{}, invalid("%s operation_id is not a UUIDv7", owner)
	}''',
        count=1,
        run="TestScalarGates",
        expect="KILLED",
        note="Decode side: admits exactly not-a-uuid as the operation ID (substituted downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-tuple",
        path=f"{PACKAGE}/report.go",
        find='''	sourceTuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil {
		return FidelityReport{}, invalid("%s source_environment is not an Environment Tuple: %v", owner, err)
	}''',
        replace='''	sourceTuple, err := sessadapter.DecodeTuple(members["source_environment"])
	if err != nil && string(members["source_environment"]) != "{\\"platform\\":\\"linux\\"}" {
		return FidelityReport{}, invalid("%s source_environment is not an Environment Tuple: %v", owner, err)
	}''',
        count=1,
        run="TestScalarGates",
        expect="KILLED",
        note="Decode side: admits exactly the platform-only tuple document (zero tuple downstream, so the digest kills).",
    ),
    Mutant(
        name="N-decode-extensions",
        path=f"{PACKAGE}/report.go",
        find='''	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {
		return FidelityReport{}, invalid("%s extensions %s", owner, extensionsFault)
	}''',
        replace='''	if extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != "{\\"frobnicate\\":1}" {
		return FidelityReport{}, invalid("%s extensions %s", owner, extensionsFault)
	}''',
        count=1,
        run="TestExtensionsClosed",
        expect="KILLED",
        note="Decode side: admits exactly the frobnicate-keyed extensions (ignored downstream, so the digest kills); the row layer is untouched.",
    ),
    Mutant(
        name="N-build-key-bound",
        path=f"{PACKAGE}/record.go",
        find='''	if length := stringLength(input.SourceItemKey); length < 1 || length > 512 {''',
        replace='''	if length := stringLength(input.SourceItemKey); length < 1 || length > 513 {''',
        count=1,
        run="TestRecordFieldBounds",
        expect="KILLED",
        note="Build side: admits exactly 513-character item keys and seals them; 514+ still refuse and the decode side is untouched.",
    ),
    Mutant(
        name="N-build-reasons-count",
        path=f"{PACKAGE}/decode.go",
        find='''	if len(values) < minimumCount || len(values) > maximumCount {
		return nil, invalid("%s carries %d reason_codes, want [%d..%d]", owner, len(values), minimumCount, maximumCount)
	}''',
        replace='''	if len(values) < minimumCount || len(values) > maximumCount && len(values) != 129 {
		return nil, invalid("%s carries %d reason_codes, want [%d..%d]", owner, len(values), minimumCount, maximumCount)
	}''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Build side: admits exactly 129 build reasons and seals them; 130+ still refuse.",
    ),
    Mutant(
        name="N-build-evidence-empty",
        path=f"{PACKAGE}/record.go",
        find='''	evidence, err := parseSortedUniqueDigestStrings(input.SourceEvidenceIDs, 1, 65536, owner, "source_evidence_ids")''',
        replace='''	evidence, err := parseSortedUniqueDigestStrings(input.SourceEvidenceIDs, 0, 65536, owner, "source_evidence_ids")''',
        count=1,
        run="TestRecordGridOracle",
        expect="KILLED",
        note="Build side: admits exactly empty source evidence and seals it; the decode side is untouched.",
    ),
    Mutant(
        name="N-build-scope",
        path=f"{PACKAGE}/report.go",
        find='''	archive := false
	switch input.Scope {
	case "archive":
		archive = true
	case "target":
	default:
		return FidelityReport{}, nil, invalid("%s scope is outside archive|target", owner)
	}''',
        replace='''	archive := false
	switch input.Scope {
	case "archive":
		archive = true
	case "target":
	default:
		if input.Scope != "staging" {
			return FidelityReport{}, nil, invalid("%s scope is outside archive|target", owner)
		}
	}''',
        count=1,
        run="TestReportMemberSet",
        expect="KILLED",
        note="Build side: admits exactly the staging scope and seals it as target; the decode side is untouched.",
    ),
    Mutant(
        name="N-build-scope-profile",
        path=f"{PACKAGE}/report.go",
        find='''	if archive && input.Profile != "archive_only" {''',
        replace='''	if archive && input.Profile != "archive_only" && input.Profile != "strict_exact" {''',
        count=1,
        run="TestArchiveTargetBranches",
        expect="KILLED",
        note="Build side: admits exactly archive with strict_exact and seals it; every other archive profile still refuses.",
    ),
    Mutant(
        name="N-build-archive-plan",
        path=f"{PACKAGE}/report.go",
        find='''	if archive {
		if input.ProjectionPlanID != nil {
			return FidelityReport{}, nil, invalid("%s projection_plan_id is not null for archive", owner)
		}''',
        replace='''	if archive {
		if input.ProjectionPlanID != nil && *input.ProjectionPlanID != "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" {
			return FidelityReport{}, nil, invalid("%s projection_plan_id is not null for archive", owner)
		}''',
        count=1,
        run="TestArchiveTargetBranches",
        expect="KILLED",
        note="Build side: admits exactly the fixture plan digest in archive scope and seals it; every other non-null plan still refuses.",
    ),
    Mutant(
        name="N-build-uint53-bytecounts",
        path=f"{PACKAGE}/report.go",
        find='''		if value > maxUint53 {
			return nil, invalid("%s[%q] exceeds uint53", owner, disposition)
		}''',
        replace='''		if value > maxUint53 && value != 1<<53 {
			return nil, invalid("%s[%q] exceeds uint53", owner, disposition)
		}''',
        count=1,
        run="TestByteCountsReconciliationIsAStatedBoundUntilSpecDefinesByteBasis",
        expect="KILLED",
        note="Build side: admits exactly 2^53 in a byte cell and seals it; larger values still refuse.",
    ),
    Mutant(
        name="N-build-sorted-forbid",
        path=f"{PACKAGE}/decode.go",
        find='''		if index > 0 && value <= previous {
			return nil, invalid("%s %s are not sorted unique", owner, member)
		}''',
        replace='''		if index > 0 && value <= previous && !(value == "a-reason" && previous == "b-reason") {
			return nil, invalid("%s %s are not sorted unique", owner, member)
		}''',
        count=1,
        run="TestForbidReasons",
        expect="KILLED",
        note="Build side: admits exactly the unsorted fixture pair past the strings order gate and seals it; duplicates still refuse.",
    ),
    Mutant(
        name="N-build-sorted-evidence",
        path=f"{PACKAGE}/decode.go",
        find='''		if index > 0 && id.String() <= previous {
			return nil, invalid("%s %s are not sorted unique", owner, member)
		}''',
        replace='''		if index > 0 && id.String() <= previous && !(id.String() == "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" && previous == "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb") {
			return nil, invalid("%s %s are not sorted unique", owner, member)
		}''',
        count=1,
        run="TestRecordSortedUnique",
        expect="KILLED",
        note="Build side: admits exactly the unsorted hand-digest pair past the digests order gate and seals it; duplicates still refuse.",
    ),
    Mutant(
        name="N-build-required-empty",
        path=f"{PACKAGE}/report.go",
        find='''		values := input[class]
		if len(values) < 1 {
			return nil, invalid("%s[%q] carry no dispositions, want a non-empty set", owner, class)
		}''',
        replace='''		values := input[class]
		if len(values) < 1 && class != "unknown" {
			return nil, invalid("%s[%q] carry no dispositions, want a non-empty set", owner, class)
		}''',
        count=1,
        run="TestRequiredDispositions",
        expect="KILLED",
        note="Build side: admits exactly the empty unknown set and seals it; every other empty set still refuses.",
    ),
    Mutant(
        name="N-build-utf8-source-class",
        path=f"{PACKAGE}/record.go",
        find='''\tif !validText(input.SourceClass) {\n\t\treturn DispositionRecord{}, nil, invalid("%s source_class is not valid UTF-8", owner)\n\t}''',
        replace='''\tif !validText(input.SourceClass) && input.SourceClass != "\\xff\\xfe" {\n\t\treturn DispositionRecord{}, nil, invalid("%s source_class is not valid UTF-8", owner)\n\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Reviewer site 1/3: admits exactly the ff-fe source_class past the pre-marshal gate and seals it.",
    ),
    Mutant(
        name="N-build-utf8-target-locator",
        path=f"{PACKAGE}/record.go",
        find='''\t\tif !validText(*input.TargetLocator) {\n\t\t\treturn DispositionRecord{}, nil, invalid("%s target_locator is not valid UTF-8", owner)\n\t\t}''',
        replace='''\t\tif !validText(*input.TargetLocator) && *input.TargetLocator != "\\xff\\xfe" {\n\t\t\treturn DispositionRecord{}, nil, invalid("%s target_locator is not valid UTF-8", owner)\n\t\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Reviewer site 2/3: admits exactly the ff-fe target_locator past the pre-marshal gate and seals it.",
    ),
    Mutant(
        name="N-build-utf8-forbid-reasons",
        path=f"{PACKAGE}/decode.go",
        find='''\t\tif !validText(value) {\n\t\t\treturn nil, invalid("%s %s[%d] is not valid UTF-8", owner, member, index)\n\t\t}''',
        replace='''\t\tif !validText(value) && value != "\\xff\\xfe" {\n\t\t\treturn nil, invalid("%s %s[%d] is not valid UTF-8", owner, member, index)\n\t\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Reviewer site 3/3: admits exactly the ff-fe forbid_reasons item (sole caller) past the pre-marshal gate.",
    ),
    Mutant(
        name="N-build-utf8-source-item-key",
        path=f"{PACKAGE}/record.go",
        find='''\tif !validText(input.SourceItemKey) {\n\t\treturn DispositionRecord{}, nil, invalid("%s source_item_key is not valid UTF-8", owner)\n\t}''',
        replace='''\tif !validText(input.SourceItemKey) && input.SourceItemKey != "\\xff\\xfe" {\n\t\treturn DispositionRecord{}, nil, invalid("%s source_item_key is not valid UTF-8", owner)\n\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Row shape: admits exactly the ff-fe source_item_key past the pre-marshal gate and seals it.",
    ),
    Mutant(
        name="N-build-utf8-explanation",
        path=f"{PACKAGE}/record.go",
        find='''\tif !validText(input.Explanation) {\n\t\treturn DispositionRecord{}, nil, invalid("%s explanation is not valid UTF-8", owner)\n\t}''',
        replace='''\tif !validText(input.Explanation) && input.Explanation != "\\xff\\xfe" {\n\t\treturn DispositionRecord{}, nil, invalid("%s explanation is not valid UTF-8", owner)\n\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Row shape: admits exactly the ff-fe explanation past the pre-marshal gate and seals it.",
    ),
    Mutant(
        name="N-build-utf8-reason-codes",
        path=f"{PACKAGE}/decode.go",
        find='''\t\tif !validText(reason) {\n\t\t\treturn nil, invalid("%s reason_codes[%d] is not valid UTF-8", owner, index)\n\t\t}''',
        replace='''\t\tif !validText(reason) && reason != "\\xff\\xfe" {\n\t\t\treturn nil, invalid("%s reason_codes[%d] is not valid UTF-8", owner, index)\n\t\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Row shape: admits exactly the ff-fe reason past the UTF-8 arm; it then refuses downstream with the vocabulary literal, so the killer UTF-8 literal assertion fails.",
    ),
    Mutant(
        name="N-build-utf8-row-extensions",
        path=f"{PACKAGE}/record.go",
        find='''\tif _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {\n\t\treturn DispositionRecord{}, nil, invalid("%s extensions invalid: %v", owner, err)\n\t}''',
        replace='''\tif _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil && input.Extensions["com.example.utf8rowprobe"] != "\\xff\\xfe" {\n\t\treturn DispositionRecord{}, nil, invalid("%s extensions invalid: %v", owner, err)\n\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Row shape: admits exactly the ff-fe row extension probe value past the leaf call site of the delegated gate.",
    ),
    Mutant(
        name="N-build-utf8-report-extensions",
        path=f"{PACKAGE}/report.go",
        find='''\tif _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil {\n\t\treturn FidelityReport{}, nil, invalid("%s extensions invalid: %v", owner, err)\n\t}''',
        replace='''\tif _, err := clonebundle.EncodeExtensions(input.Extensions); err != nil && input.Extensions["com.example.utf8probe"] != "\\xff\\xfe" {\n\t\treturn FidelityReport{}, nil, invalid("%s extensions invalid: %v", owner, err)\n\t}''',
        count=1,
        run="TestUTF8RefusalReflectionGrid",
        expect="KILLED",
        note="Report shape: admits exactly the ff-fe report extension probe value past the leaf call site of the delegated gate.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/doc.go",
        find="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Section\n",
        replace="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Section (control).\n",
        count=1,
        run="TestEntriesArePure",
        expect="SURVIVED",
        note="Harmless control: a comment-only edit must survive.",
    ),
]


def run_mutant(mutant: Mutant) -> tuple[str, str, int, float]:
    target = REPO / mutant.path
    original = target.read_text()
    found = original.count(mutant.find)
    if found != mutant.count:
        return f"ERROR: {mutant.name}: patch anchors {found}, want {mutant.count}", "", -1, 0.0
    with tempfile.TemporaryDirectory() as staging:
        backup = Path(staging) / "backup"
        shutil.copy2(target, backup)
        try:
            target.write_text(original.replace(mutant.find, mutant.replace))
            started = time.time()
            run = subprocess.run(
                ["go", "test", f"./{PACKAGE}/", "-run", mutant.run, "-count=1"],
                cwd=REPO,
                capture_output=True,
                text=True,
                timeout=300,
            )
            elapsed = time.time() - started
        finally:
            shutil.copy2(backup, target)
    output = (run.stdout + run.stderr).strip()
    if 'build failed' in output or '\n# ' in output or output.startswith('# '):
        return f"ERROR: {mutant.name}: build broke under the patch", output, run.returncode, elapsed
    if run.returncode == 0:
        verdict = "SURVIVED"
    elif '--- FAIL' in output or 'FAIL:' in output:
        verdict = "KILLED"
    else:
        return f"ERROR: {mutant.name}: exit {run.returncode} with no kill line", output, run.returncode, elapsed
    mark = "ok" if verdict == mutant.expect else "MISMATCH"
    return f"{mark}: {mutant.name}: {verdict} (want {mutant.expect}) :: {mutant.note}", output, run.returncode, elapsed


def main(argv: list[str]) -> int:
    log_dir: Path | None = None
    selected_names: list[str] = []
    args = list(argv)
    while args:
        if args[0] == "--log-dir":
            if len(args) < 2:
                print("missing directory for --log-dir")
                return 2
            log_dir = Path(args[1])
            args = args[2:]
        else:
            selected_names.append(args[0])
            args = args[1:]
    selected = [m for m in MUTANTS if not selected_names or m.name in selected_names]
    if selected_names and len(selected) != len(selected_names):
        unknown = sorted(set(selected_names) - {m.name for m in selected})
        print(f"unknown mutants: {', '.join(unknown)}")
        return 2
    if log_dir is not None:
        log_dir.mkdir(parents=True, exist_ok=True)
    failures = 0
    for mutant in selected:
        try:
            line, output, exit_code, elapsed = run_mutant(mutant)
        except subprocess.TimeoutExpired:
            line, output, exit_code, elapsed = f"ERROR: {mutant.name}: killer timed out", "", -1, 0.0
        print(line, flush=True)
        if log_dir is not None:
            (log_dir / f"{mutant.name}.log").write_text(
                f"# mutant={mutant.name} path={mutant.path} -run={mutant.run}\n"
                f"# exit={exit_code} elapsed={elapsed:.1f}s\n"
                f"# find={mutant.find!r}\n# replace={mutant.replace!r}\n---\n{output}\n"
            )
        if not line.startswith("ok:"):
            failures += 1
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
