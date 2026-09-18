#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/cloneproject.

Every gate ships a narrowing mutant: the gate stays present and is
weakened to admit exactly one member of the class it must reject, and
the named killer test must fail. A delete-only mutant proves only that
the gate exists and is not accepted as evidence.

Each row runs M (mutate one production file in place), R (run the
scoped Go killer through the production Normalize entry over real
captured bytes), V (verdict from the Go exit code plus a FAIL
kill-line: KILLED iff R fails with a test failure line, SURVIVED iff
R passes, ERROR for anything else — unapplied patches, build breaks,
timeouts). M failures and R infrastructure errors are ERROR, never
kills or survivals. The file is restored from its backup after every
row, pass or fail.

Rows N-native-type-token and N-protection-encrypted are the
token-preserving attacks: the searched-for text still matches, but
the decision changes. Their killers execute the behavioral Normalize
suite over captured bytes, not a static checker.

Four rows carry explicit non-narrowing labels instead of the narrowing
claim: N-empty-projection (single-member class: the gate's reject
class holds only the empty projection, so the plant admits the whole
class), N-overflow-truncate (behaviour swap: truncate to 64 KiB
instead of installing a blob reference), N-pending-aborted
(add-arm: Normalize passes nil live work orders, so the narrowed
condition alone cannot pend anything from the entry; the plant
appends the aborted call explicitly), and N-offset-newline
(behaviour swap: drop the newline stride from the offset
accumulation, so every record after line 1 references shifted
bytes).

The five N-strict-* rows patch the shared strictMembers body used by
both frame sites (one gate, two call sites: the record envelope and
the per-kind bodies); each admits exactly its killer fixture into a
lenient decode. The six N-alias-* rows restore the case-folded read
for exactly one envelope member each; every other alias stays an
unclaimed extra under the plant.

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/cloneproject/testdata/mutant_harness.py [--log-dir DIR] [MUTANT ...]
"""

import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PACKAGE = "internal/cloneproject"


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
        name="N-unknown-kind",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tdefault:\n\t\tunit.kind = "opaque_event"\n''',
        replace='''\tdefault:\n\t\tunit.kind = "opaque_event"\n\t\tif record.NativeType == "message/smoke" {\n\t\t\tunit.kind = "user_message"\n\t\t}\n''',
        count=1,
        run="TestNormalizeAlmostKnownRecordBecomesOpaqueEvent",
        expect="KILLED",
        note="Admits exactly message/smoke past the unknown-type gate into user_message; every other unknown type still projects opaque_event.",
    ),
    Mutant(
        name="N-native-type-token",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tcase nativeMessageUser:\n\t\tunit.kind = "user_message"\n''',
        replace='''\tcase nativeMessageUser:\n\t\tunit.kind = "assistant_message"\n''',
        count=1,
        run="TestNormalizeSealsSessionShape",
        expect="KILLED",
        note="Token-preserving: the message/user label still matches but routes to assistant_message; the behavioral killer pins the kind.",
    ),
    Mutant(
        name="N-unknown-member",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\tif item.Class == "unknown" {\n''',
        replace='''\t\tif item.Class == "unknown" && item.NativeItemKey != "store/mystery" {\n''',
        count=1,
        run="TestNormalizeUnknownMemberBecomesWholeMemberOpaque",
        expect="KILLED",
        note="Admits exactly store/mystery past the unknown-class gate into JSONL parsing; every other unknown member still projects whole-member opaque.",
    ),
    Mutant(
        name="N-protection-encrypted",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tif record.Protection != "none" {\n''',
        replace='''\tif record.Protection != "none" && record.Protection != "encrypted" {\n''',
        count=1,
        run="TestNormalizeForeignEncryptedReasoningStaysOpaque",
        expect="KILLED",
        note="Token-preserving narrowing: admits exactly encrypted records past the protection gate; signed records still project opaque.",
    ),
    Mutant(
        name="N-reasoning-contradiction",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tcase nativeReasoningCrypt, nativeReasoningSigned:\n\t\twhere := "record " + record.MemberKey + " line " + itoa(record.Line)\n\t\treturn invalid("%s declares %s with no protection", where, record.NativeType)\n''',
        replace='''\tcase nativeReasoningCrypt, nativeReasoningSigned:\n\t\twhere := "record " + record.MemberKey + " line " + itoa(record.Line)\n\t\tif record.NativeType == "reasoning/signed" {\n\t\t\treturn invalid("%s declares %s with no protection", where, record.NativeType)\n\t\t}\n\t\tunit.kind = "reasoning_summary"\n\t\tunit.visibility = "internal"\n\t\tunit.payload = map[string]any{"content_blocks": []any{}, "extensions": map[string]any{}}\n\t\treturn nil\n''',
        count=1,
        run="TestNormalizeRefusalTable/contradictory_protection",
        expect="KILLED",
        note="Admits exactly unprotected reasoning/encrypted as reasoning_summary; unprotected reasoning/signed still refuses.",
    ),
    Mutant(
        name="N-foreign-instruction",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif observation.record.Origin != "native" {\n''',
        replace='''\t\tif observation.record.Origin != "native" && observation.record.NativeEventID != "evt-inst-foreign" {\n''',
        count=1,
        run="TestNormalizeForeignInstructionCannotChangeEffectiveSnapshot",
        expect="KILLED",
        note="Admits exactly the foreign instruction into the effective snapshot; every other foreign record stays history-only.",
    ),
    Mutant(
        name="N-foreign-payload-authority",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tauthority := body.Authority\n\tif record.Origin != "native" {\n\t\tauthority = "low"\n\t}\n''',
        replace='''\tauthority := body.Authority\n\tif record.Origin != "native" && record.NativeEventID != "evt-inst-foreign" {\n\t\tauthority = "low"\n\t}\n''',
        count=1,
        run="TestNormalizeForeignInstructionCannotChangeEffectiveSnapshot",
        expect="KILLED",
        note="Admits exactly the foreign high claim into the payload authority; every other foreign instruction still pins low.",
    ),
    Mutant(
        name="N-tool-incomplete",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tresult, ok := seenResults[call.body.CallID]\n\t\tif !ok {\n''',
        replace='''\t\tresult, ok := seenResults[call.body.CallID]\n\t\tif !ok && call.body.CallID != "call-7" {\n''',
        count=1,
        run="TestNormalizeToolMatrix/call_without_result",
        expect="KILLED",
        note="Admits exactly call-7 without its result as completed; every other incomplete call still becomes aborted history.",
    ),
    Mutant(
        name="N-tool-orphan",
        path=f"{PACKAGE}/tools.go",
        find='''\tfor _, result := range results {\n\t\tif _, ok := seenCalls[result.body.CallID]; !ok {\n\t\t\tresolutions = append(resolutions, ToolResolution{\n\t\t\t\tCallID:       result.body.CallID,\n\t\t\t\tStatus:       StatusAborted,\n\t\t\t\tResultStatus: result.body.Status,\n\t\t\t})\n\t\t}\n\t}\n''',
        replace='''\tfor _, result := range results {\n\t\tif _, ok := seenCalls[result.body.CallID]; !ok {\n\t\t\tstatus := StatusAborted\n\t\t\tif result.body.CallID == "call-ghost" {\n\t\t\t\tstatus = StatusCompleted\n\t\t\t}\n\t\t\tresolutions = append(resolutions, ToolResolution{\n\t\t\t\tCallID:       result.body.CallID,\n\t\t\t\tStatus:       status,\n\t\t\t\tResultStatus: result.body.Status,\n\t\t\t})\n\t\t}\n\t}\n''',
        count=1,
        run="TestNormalizeToolMatrix/result_without_call",
        expect="KILLED",
        note="Admits exactly the ghost orphan result as completed; every other orphan still becomes aborted history.",
    ),
    Mutant(
        name="N-callable",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif definition.body.LiveClaim {\n\t\t\twhere := "record " + definition.record.MemberKey + " line " + itoa(definition.record.Line)\n\t\t\treturn LiveSurface{}, invalid("%s claims live authority for historical tool definition %q", where, definition.body.ToolName)\n\t\t}\n''',
        replace='''\t\tif definition.body.LiveClaim && definition.body.ToolName != "evil-tool" {\n\t\t\twhere := "record " + definition.record.MemberKey + " line " + itoa(definition.record.Line)\n\t\t\treturn LiveSurface{}, invalid("%s claims live authority for historical tool definition %q", where, definition.body.ToolName)\n\t\t}\n\t\tif definition.body.LiveClaim {\n\t\t\tsurface.CallableTools = append(surface.CallableTools, definition.body.ToolName)\n\t\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/live_attestation_claim",
        expect="KILLED",
        note="Admits exactly evil-tool past the live-claim refusal and registers it callable; every other live claim still refuses.",
    ),
    Mutant(
        name="N-callable-false",
        path=f"{PACKAGE}/dispatch.go",
        find='''\t\t"callable":          false,\n''',
        replace='''\t\t"callable":          unit.record.NativeEventID == "evt-def-1",\n''',
        count=1,
        run="TestNormalizeToolDefinitionNeverRegistersCallable",
        expect="KILLED",
        note="Admits exactly evt-def-1 as callable in the sealed payload; the token stays and every other definition still pins false.",
    ),
    Mutant(
        name="N-pending-aborted",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif resolution.Status == StatusAborted {\n\t\t\tcontinue\n\t\t}\n''',
        replace='''\t\tif resolution.Status == StatusAborted && resolution.CallID != "call-7" {\n\t\t\tcontinue\n\t\t}\n\t\tif resolution.CallID == "call-7" {\n\t\t\tsurface.PendingActions = append(surface.PendingActions, resolution.CallID)\n\t\t\tcontinue\n\t\t}\n''',
        count=1,
        run="TestNormalizeToolMatrix/call_without_result",
        expect="KILLED",
        note="Labelled add-arm, not a narrowing: Normalize passes nil live work orders, so the narrowed condition alone cannot pend from the entry; the appended arm admits exactly the aborted call-7 as a pending action explicitly.",
    ),
    Mutant(
        name="N-pending-completed",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif liveWorkOrders[resolution.CallID] {\n''',
        replace='''\t\tif liveWorkOrders[resolution.CallID] || resolution.CallID == "call-1" {\n''',
        count=1,
        run="TestNormalizeToolMatrix/call_with_result",
        expect="KILLED",
        note="Admits exactly the completed call-1 as pending without a live work order; every other completed call still needs one.",
    ),
    Mutant(
        name="N-live-followup",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif result.body.LiveFollowup {\n''',
        replace='''\t\tif result.body.LiveFollowup && result.body.CallID != "call-live" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/live_followup_claim",
        expect="KILLED",
        note="Admits exactly the call-live followup claim; every other live followup still refuses.",
    ),
    Mutant(
        name="N-duplicate-call",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif _, duplicate := seenCalls[call.body.CallID]; duplicate {\n''',
        replace='''\t\tif _, duplicate := seenCalls[call.body.CallID]; duplicate && call.body.CallID != "call-dup" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/duplicate_call_id",
        expect="KILLED",
        note="Admits exactly the duplicated call-dup ID; every other duplicate call still refuses.",
    ),
    Mutant(
        name="N-usage-target",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tledger.SourceInputTokens = input\n\t\tledger.SourceOutputTokens = output\n''',
        replace='''\t\tledger.SourceInputTokens = input\n\t\tledger.SourceOutputTokens = output\n\t\tif observation.record.NativeEventID == "evt-use-1" {\n\t\t\tledger.TargetInputTokens += observation.body.InputTokens\n\t\t}\n''',
        count=1,
        run="TestNormalizeSourceUsageIsNotTargetAccounting",
        expect="KILLED",
        note="Funds the target ledger from exactly evt-use-1; every other source record still counts source-only.",
    ),
    Mutant(
        name="N-overflow-bound",
        path=f"{PACKAGE}/overflow.go",
        find='''\tif len(text) > maxInlineContentBytes {\n''',
        replace='''\tif len(text) > maxInlineContentBytes+1 {\n''',
        count=1,
        run="TestNormalizeInlineContentBoundary",
        expect="KILLED",
        note="Admits exactly 65537-byte text inline past the 64 KiB bound; 65538+ still becomes a blob reference.",
    ),
    Mutant(
        name="N-overflow-truncate",
        path=f"{PACKAGE}/overflow.go",
        find='''\tif len(text) > maxInlineContentBytes {\n\t\tinstalled, err := installOverflow(sink, text)\n''',
        replace='''\tif len(text) > maxInlineContentBytes {\n\t\ttext = text[:maxInlineContentBytes]\n\t}\n\tif false {\n\t\tinstalled, err := installOverflow(sink, text)\n''',
        count=1,
        run="TestNormalizeInlineContentBoundary",
        expect="KILLED",
        note="Behaviour swap (labelled, not a narrowing): silently truncates oversized text to 64 KiB instead of installing a blob reference; the byte-exactness killer reddens.",
    ),
    Mutant(
        name="N-unstable",
        path=f"{PACKAGE}/normalize.go",
        find='''\tif err := clonebundle.RefuseUnstableForTarget(capture.CaptureBoundary); err != nil {\n\t\treturn nil, err\n\t}\n''',
        replace='''\tif err := clonebundle.RefuseUnstableForTarget(capture.CaptureBoundary); err != nil && capture.OperationID.String() != "0198f4d1-9e60-7a11-8a31-abcdef012345" {\n\t\treturn nil, err\n\t}\n''',
        count=1,
        run="TestNormalizeRefusesUnstableCapture",
        expect="KILLED",
        note="Admits exactly the fixture unstable archive to target projection; every other unstable capture still refuses.",
    ),
    Mutant(
        name="N-digest-verify",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\tif scalar.SHA256Digest(payload).String() != entry.BlobID.String() {\n''',
        replace='''\t\tif scalar.SHA256Digest(payload).String() != entry.BlobID.String() && item.NativeItemKey != "store/session.jsonl" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/tampered_blob",
        expect="KILLED",
        note="Skips the digest check for exactly store/session.jsonl; the tampered bytes then fail framing instead of the digest refusal.",
    ),
    Mutant(
        name="N-subagent-actor",
        path=f"{PACKAGE}/dispatch.go",
        find='''\t\tif unit.record.Actor == "main" {\n''',
        replace='''\t\tif unit.record.Actor == "main" || unit.record.Actor == "subagent:fetch" {\n''',
        count=1,
        run="TestNormalizeToolMatrix/nested_subagent_call",
        expect="KILLED",
        note="Attributes exactly the nested fetch call to main; every other non-main selector still resolves to its mapped actor.",
    ),
    Mutant(
        name="N-instruction-authority",
        path=f"{PACKAGE}/native.go",
        find='''\tif authority != "high" && authority != "low" {\n''',
        replace='''\tif authority != "high" && authority != "low" && authority != "absolute" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_instruction_authority",
        expect="KILLED",
        note="Admits exactly the absolute authority claim; every other non-high/low claim still refuses.",
    ),
    Mutant(
        name="N-definition-digest",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\t\tif _, err := scalar.ParseDigest(body.DefinitionHash); err != nil {\n''',
        replace='''\t\t\tif _, err := scalar.ParseDigest(body.DefinitionHash); err != nil && body.DefinitionHash != "not-a-digest" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_definition_digest",
        expect="KILLED",
        note="Admits exactly the not-a-digest definition hash; every other malformed digest still refuses.",
    ),
    Mutant(
        name="N-parents-chain",
        path=f"{PACKAGE}/dispatch.go",
        find='''\t\tparents := []string{}\n\t\tif previous != nil {\n\t\t\tparents = []string{previous.String()}\n\t\t}\n''',
        replace='''\t\tparents := []string{}\n\t\tif previous != nil && index != 1 {\n\t\t\tparents = []string{previous.String()}\n\t\t}\n''',
        count=1,
        run="TestNormalizeSealsSessionShape",
        expect="KILLED",
        note="Leaves exactly ordinal 1 unparented; every other ordinal still chains to its predecessor.",
    ),
    Mutant(
        name="N-string-bound",
        path=f"{PACKAGE}/native.go",
        find='''\tif environ.StringLength(envelope.NativeEventID) < 1 || environ.StringLength(envelope.NativeEventID) > 512 {\n''',
        replace='''\tif environ.StringLength(envelope.NativeEventID) < 1 || environ.StringLength(envelope.NativeEventID) > 513 {\n''',
        count=1,
        run="TestNormalizeMultibyteStringMeasure",
        expect="KILLED",
        note="Admits exactly a 513-character native event ID; 514+ still refuses. Pins the envelope ID edge only; per-edge rows for directives, call IDs, and tool names are stated bounds.",
    ),
    Mutant(
        name="N-envelope-version",
        path=f"{PACKAGE}/native.go",
        find='''\tif _, ok := environ.CheckUint53Bounds(members["v"], 1, 1); !ok {\n''',
        replace='''\tif _, ok := environ.CheckUint53Bounds(members["v"], 1, 1); !ok && string(members["v"]) != "2" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_envelope_version",
        expect="KILLED",
        note="Admits exactly envelope version 2 past the landed uint53 gate; every other non-1 version still refuses.",
    ),
    Mutant(
        name="N-origin",
        path=f"{PACKAGE}/native.go",
        find='''\tif envelope.Origin != "native" && envelope.Origin != "foreign" {\n''',
        replace='''\tif envelope.Origin != "native" && envelope.Origin != "foreign" && envelope.Origin != "sideways" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_origin",
        expect="KILLED",
        note="Admits exactly the sideways origin; every other non-native/foreign origin still refuses.",
    ),
    Mutant(
        name="N-protection-value",
        path=f"{PACKAGE}/native.go",
        find='''\tif envelope.Protection != "none" && envelope.Protection != "encrypted" && envelope.Protection != "signed" {\n''',
        replace='''\tif envelope.Protection != "none" && envelope.Protection != "encrypted" && envelope.Protection != "signed" && envelope.Protection != "rot13" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_protection",
        expect="KILLED",
        note="Admits exactly the rot13 protection; every other out-of-vocabulary protection still refuses.",
    ),
    Mutant(
        name="N-actor-selector",
        path=f"{PACKAGE}/native.go",
        find='''\tname, ok := strings.CutPrefix(selector, "subagent:")\n\tif !ok || name == "" {\n''',
        replace='''\tname, ok := strings.CutPrefix(selector, "subagent:")\n\tif (!ok || name == "") && selector != "boss" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_actor_shape",
        expect="KILLED",
        note="Admits exactly the boss selector; every other malformed selector still refuses.",
    ),
    Mutant(
        name="N-subagent-name",
        path=f"{PACKAGE}/native.go",
        find='''\tif environ.StringLength(name) > 128 {\n''',
        replace='''\tif environ.StringLength(name) > 129 {\n''',
        count=1,
        run="TestNormalizeRefusalTable/long_subagent_name",
        expect="KILLED",
        note="Admits exactly a 129-character subagent name; 130+ still refuses.",
    ),
    Mutant(
        name="N-body-member",
        path=f"{PACKAGE}/native.go",
        find='''\tfor name := range members {\n\t\tif !allowed[name] {\n''',
        replace='''\tfor name := range members {\n\t\tif !allowed[name] && name != "escalate" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/unknown_body_member",
        expect="KILLED",
        note="Admits exactly the escalate body member past the shared exact-shape gate; every other unknown member still refuses.",
    ),
    Mutant(
        name="N-blank-line",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif len(raw) == 0 {\n\t\t\treturn nil, invalid("record %s line %d is empty", memberKey, number)\n\t\t}\n''',
        replace='''\t\tif len(raw) == 0 && number != 2 {\n\t\t\treturn nil, invalid("record %s line %d is empty", memberKey, number)\n\t\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/blank_line",
        expect="KILLED",
        note="Admits exactly a blank line 2; a blank line anywhere else still refuses.",
    ),
    Mutant(
        name="N-malformed-json",
        path=f"{PACKAGE}/native.go",
        find='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n''',
        replace='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n\tif bytes.Contains(raw, []byte("{not-json")) {\n\t\treturn map[string]json.RawMessage{}, nil\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/malformed_json_line",
        expect="KILLED",
        note="Admits exactly the killer's malformed line past the strict gate into the envelope-shape gate, which refuses with a different message; every other malformed line still refuses at the strict gate.",
    ),
    Mutant(
        name="N-usage-number",
        path=f"{PACKAGE}/native.go",
        find='''\tvalue, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)\n\tif !ok {\n\t\treturn 0, invalid("%s body member %q is not a uint53", where, name)\n\t}\n\treturn value, nil\n''',
        replace='''\tvalue, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)\n\tif !ok && string(members[name]) != "1.5" {\n\t\treturn 0, invalid("%s body member %q is not a uint53", where, name)\n\t}\n\tif !ok {\n\t\tvalue = 1\n\t}\n\treturn value, nil\n''',
        count=1,
        run="TestNormalizeRefusalTable/fractional_usage",
        expect="KILLED",
        note="Admits exactly 1.5 past the landed uint53 gate with value 1; every other non-uint53 literal still refuses.",
    ),
    Mutant(
        name="N-directive-count",
        path=f"{PACKAGE}/native.go",
        find='''\tif len(directives) > 1024 {\n''',
        replace='''\tif len(directives) > 1025 {\n''',
        count=1,
        run="TestNormalizeRefusalTable/too_many_directives",
        expect="KILLED",
        note="Admits exactly 1025 directives; 1026+ still refuses.",
    ),
    Mutant(
        name="N-usage-overflow",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif input < ledger.SourceInputTokens || input > maxUint53 {\n''',
        replace='''\t\tif input < ledger.SourceInputTokens || input > maxUint53+1 {\n''',
        count=1,
        run="TestNormalizeRefusalTable/usage_ledger_overflow",
        expect="KILLED",
        note="Admits exactly the 2^53 input total; anything larger still refuses instead of wrapping.",
    ),
    Mutant(
        name="N-duplicate-result",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif _, duplicate := seenResults[result.body.CallID]; duplicate {\n''',
        replace='''\t\tif _, duplicate := seenResults[result.body.CallID]; duplicate && result.body.CallID != "call-r" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/duplicate_result_id",
        expect="KILLED",
        note="Admits exactly the duplicated call-r result; every other duplicate result still refuses.",
    ),
    Mutant(
        name="N-empty-projection",
        path=f"{PACKAGE}/normalize.go",
        find='''\tif len(plan.units) == 0 {\n\t\treturn nil, invalid("projection holds no events")\n\t}\n''',
        replace='''\tif len(plan.units) < 0 {\n\t\treturn nil, invalid("projection holds no events")\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/empty_projection",
        expect="KILLED",
        note="Single-member class (labelled, not a narrowing): the gate's reject class holds only the empty projection, so the plant admits the whole class past the unit gate into the session builder.",
    ),
    Mutant(
        name="N-missing-blob",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\tpayload, err := plan.request.Fetch(entry.BlobID.String())\n\t\tif err != nil {\n''',
        replace='''\t\tpayload, err := plan.request.Fetch(entry.BlobID.String())\n\t\tif err != nil && item.NativeItemKey != "store/session.jsonl" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/missing_blob",
        expect="KILLED",
        note="Admits exactly the missing session blob past the fetch gate into the size gate; every other missing blob still refuses at the fetch gate.",
    ),
    Mutant(
        name="N-manifest-link",
        path=f"{PACKAGE}/normalize.go",
        find='''\tif raw.ManifestID.String() != capture.SourceRawObjectManifest.String() {\n''',
        replace='''\tif raw.ManifestID.String() != capture.SourceRawObjectManifest.String() && capture.OperationID.String() != "0198f4d1-9e60-7a11-8a31-abcdef012345" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/manifest_closure_drift",
        expect="KILLED",
        note="Admits exactly the fixture mismatched pair past the ID link; every other mismatched pair still refuses.",
    ),
    Mutant(
        name="N-env-drift",
        path=f"{PACKAGE}/normalize.go",
        find='''\tif !bytes.Equal(rawCanonical, capturedCanonical) {\n''',
        replace='''\tif len(rawCanonical) < 8 || len(capturedCanonical) < 8 || !bytes.Equal(rawCanonical[:8], capturedCanonical[:8]) {\n''',
        count=1,
        run="TestNormalizeRefusesEnvironmentDrift",
        expect="KILLED",
        note="Compares only the first 8 bytes, admitting exactly late-drifting tuples past the drift gate; any early drift still refuses.",
    ),
    Mutant(
        name="N-actor-unmapped-fallback",
        path=f"{PACKAGE}/dispatch.go",
        find='''\t\tif _, ok := plan.extraActors[selector]; !ok {\n\t\t\treturn invalid("projection references actor %q with no mapped UUID", selector)\n\t\t}\n''',
        replace='''\t\tif _, ok := plan.extraActors[selector]; !ok && selector != "subagent:stray" {\n\t\t\treturn invalid("projection references actor %q with no mapped UUID", selector)\n\t\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/unmapped_actor",
        expect="KILLED",
        note="Admits exactly the stray selector past the actor pre-check into the resolving layer, which refuses with its own message; every other unmapped selector still refuses at the pre-check.",
    ),
    Mutant(
        name="N-actor-unreferenced",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tfor selector := range plan.extraActors {\n\t\tif !referenced[selector] {\n\t\t\treturn invalid("projection maps actor %q with no referencing record", selector)\n\t\t}\n\t}\n''',
        replace='''\tfor selector := range plan.extraActors {\n\t\tif !referenced[selector] && selector != "subagent:idle" {\n\t\t\treturn invalid("projection maps actor %q with no referencing record", selector)\n\t\t}\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/unmapped_extra_actor",
        expect="KILLED",
        note="Admits exactly the idle mapping without a referencing record; every other unreferenced mapping still refuses.",
    ),
    Mutant(
        name="N-main-in-map",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\tif selector == "main" {\n\t\t\treturn nil, invalid("projection actor map carries main, which is request.MainActorID")\n\t\t}\n''',
        replace='''\t\tif selector == "main" && value != "0198f4d1-9e60-7a88-81a8-123456789abc" {\n\t\t\treturn nil, invalid("projection actor map carries main, which is request.MainActorID")\n\t\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/main_actor_in_map",
        expect="KILLED",
        note="Admits exactly the killer's main mapping; every other main mapping still refuses.",
    ),
    Mutant(
        name="N-protection-signed",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tif record.Protection != "none" {\n''',
        replace='''\tif record.Protection != "none" && record.Protection != "signed" {\n''',
        count=1,
        run="TestNormalizeForeignSignedReasoningStaysOpaque",
        expect="KILLED",
        note="Admits exactly signed records past the protection gate; encrypted records still project opaque.",
    ),
    Mutant(
        name="N-unknown-kind-wholly",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tdefault:\n\t\tunit.kind = "opaque_event"\n''',
        replace='''\tdefault:\n\t\tunit.kind = "opaque_event"\n\t\tif record.NativeType == "frobnicate" {\n\t\t\tunit.kind = "user_message"\n\t\t}\n''',
        count=1,
        run="TestNormalizeUnknownRecordBecomesOpaqueEvent",
        expect="KILLED",
        note="Admits exactly the wholly unknown frobnicate type past the unknown-type gate into user_message; every other unknown type still projects opaque_event.",
    ),
    Mutant(
        name="N-reasons-both-single",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tif !knownNativeType(record.NativeType) {\n''',
        replace='''\tif !knownNativeType(record.NativeType) && record.NativeType != "frobnicate" {\n''',
        count=1,
        run="TestNormalizeUnknownProtectedRecordCarriesBothReasons",
        expect="KILLED",
        note="Admits exactly the frobnicate signed record with a single reason; every other unknown protected type still carries both.",
    ),
    Mutant(
        name="N-native-reason-foreign",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tif record.Origin != "foreign" {\n\t\treturn nil\n\t}\n''',
        replace='''\tif record.Origin != "foreign" && record.NativeEventID != "evt-enc-native" {\n\t\treturn nil\n\t}\n''',
        count=1,
        run="TestNormalizeNativeProtectedReasoningStaysOpaque",
        expect="KILLED",
        note="Admits exactly the native encrypted record to a foreign reason; every other native record still carries none.",
    ),
    Mutant(
        name="N-main-parent",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tinputs := []clonebundle.ActorInput{{\n\t\tActorID: main,\n\t\tKind:    "main",\n\t}}\n''',
        replace='''\tinputs := []clonebundle.ActorInput{{\n\t\tActorID: main,\n\t\tKind:    "main",\n\t\tParentActorID: &main,\n\t}}\n''',
        count=1,
        run="TestNormalizeSealsSessionShape",
        expect="KILLED",
        note="Admits exactly main to the parented class; the landed builder refuses a main actor with a parent.",
    ),
    Mutant(
        name="N-strict-text-surrogate",
        path=f"{PACKAGE}/native.go",
        find='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n''',
        replace='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n\tif bytes.Contains(raw, []byte("\\"evt-sur-text")) {\n\t\tmembers = map[string]json.RawMessage{}\n\t\tif err := json.Unmarshal(raw, &members); err != nil {\n\t\t\treturn nil, invalid("%s is not a JSON object: %v", what, err)\n\t\t}\n\t\treturn members, nil\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/lone_surrogate_text",
        expect="KILLED",
        note="Admits exactly the killer's lone-surrogate text line past the strict gate into a lenient decode that rewrites the escape; every other lone escape still refuses.",
    ),
    Mutant(
        name="N-strict-envelope-surrogate",
        path=f"{PACKAGE}/native.go",
        find='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n''',
        replace='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n\tif bytes.Contains(raw, []byte("\\"evt-sur-envelope-")) {\n\t\tmembers = map[string]json.RawMessage{}\n\t\tif err := json.Unmarshal(raw, &members); err != nil {\n\t\t\treturn nil, invalid("%s is not a JSON object: %v", what, err)\n\t\t}\n\t\treturn members, nil\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/lone_surrogate_envelope",
        expect="KILLED",
        note="Admits exactly the killer's lone-surrogate envelope line past the strict gate into a lenient decode that rewrites the identity; every other lone escape still refuses.",
    ),
    Mutant(
        name="N-strict-envelope-duplicate",
        path=f"{PACKAGE}/native.go",
        find='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n''',
        replace='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n\tif bytes.Contains(raw, []byte("\\"evt-dup-envelope\\"")) {\n\t\tmembers = map[string]json.RawMessage{}\n\t\tif err := json.Unmarshal(raw, &members); err != nil {\n\t\t\treturn nil, invalid("%s is not a JSON object: %v", what, err)\n\t\t}\n\t\treturn members, nil\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/duplicate_envelope_member",
        expect="KILLED",
        note="Admits exactly the killer's duplicated-envelope line past the strict gate into a last-wins decode that coerces it into a known kind; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-strict-body-duplicate",
        path=f"{PACKAGE}/native.go",
        find='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n''',
        replace='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n\tif bytes.Contains(raw, []byte("\\"text\\":\\"A\\",\\"text\\":\\"B\\"")) {\n\t\tmembers = map[string]json.RawMessage{}\n\t\tif err := json.Unmarshal(raw, &members); err != nil {\n\t\t\treturn nil, invalid("%s is not a JSON object: %v", what, err)\n\t\t}\n\t\treturn members, nil\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/duplicate_body_member",
        expect="KILLED",
        note="Admits exactly the killer's duplicated-body line past the strict gate into a last-wins decode that guesses content B; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-trailing-admitted",
        path=f"{PACKAGE}/native.go",
        find='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n''',
        replace='''\tmembers, fault := environ.DecodeStrictObject(raw)\n\tif fault == nil {\n\t\treturn members, nil\n\t}\n\tif bytes.Contains(raw, []byte("\\"evt-trailing\\"")) {\n\t\tmembers = map[string]json.RawMessage{}\n\t\tif err := json.Unmarshal(raw, &members); err != nil {\n\t\t\treturn nil, invalid("%s is not a JSON object: %v", what, err)\n\t\t}\n\t\treturn members, nil\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/trailing_bytes",
        expect="KILLED",
        note="Admits exactly the killer's trailing-bytes line past the strict gate into the lenient decode, which refuses with a different message; every other trailing line still refuses at the strict gate.",
    ),
    Mutant(
        name="N-usage-string",
        path=f"{PACKAGE}/native.go",
        find='''\tvalue, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)\n\tif !ok {\n\t\treturn 0, invalid("%s body member %q is not a uint53", where, name)\n\t}\n\treturn value, nil\n''',
        replace='''\tvalue, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)\n\tif !ok && string(members[name]) != "\\"5\\"" {\n\t\treturn 0, invalid("%s body member %q is not a uint53", where, name)\n\t}\n\tif !ok {\n\t\tvalue = 5\n\t}\n\treturn value, nil\n''',
        count=1,
        run="TestNormalizeRefusalTable/string_usage_number",
        expect="KILLED",
        note='Admits exactly the "5" string past the landed uint53 gate with value 5; every other non-uint53 literal still refuses.',
    ),
    Mutant(
        name="N-usage-magnitude",
        path=f"{PACKAGE}/native.go",
        find='''\tvalue, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)\n\tif !ok {\n\t\treturn 0, invalid("%s body member %q is not a uint53", where, name)\n\t}\n\treturn value, nil\n''',
        replace='''\tvalue, ok := environ.CheckUint53Bounds(members[name], 0, maxUint53)\n\tif !ok && string(members[name]) != "9007199254740992" {\n\t\treturn 0, invalid("%s body member %q is not a uint53", where, name)\n\t}\n\tif !ok {\n\t\tvalue = maxUint53\n\t}\n\treturn value, nil\n''',
        count=1,
        run="TestNormalizeRefusalTable/unsafe_usage_magnitude",
        expect="KILLED",
        note="Admits exactly 2^53 past the landed uint53 gate with the max value; every other out-of-range magnitude still refuses.",
    ),
    Mutant(
        name="N-envelope-version-string",
        path=f"{PACKAGE}/native.go",
        find='''\tif _, ok := environ.CheckUint53Bounds(members["v"], 1, 1); !ok {\n''',
        replace='''\tif _, ok := environ.CheckUint53Bounds(members["v"], 1, 1); !ok && string(members["v"]) != "\\"1\\"" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/string_envelope_version",
        expect="KILLED",
        note='Admits exactly the "1" string past the envelope version gate; every other non-1 version still refuses.',
    ),
    Mutant(
        name="N-reasoning-encrypted-signed",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tcase nativeReasoningCrypt:\n\t\tif record.Protection != "encrypted" {\n''',
        replace='''\tcase nativeReasoningCrypt:\n\t\tif record.Protection != "encrypted" && record.Protection != "signed" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/encrypted_type_signed_protection",
        expect="KILLED",
        note="Admits exactly signed protection for the encrypted reasoning type; every other mismatched protection still refuses.",
    ),
    Mutant(
        name="N-reasoning-signed-encrypted",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tcase nativeReasoningSigned:\n\t\tif record.Protection != "signed" {\n''',
        replace='''\tcase nativeReasoningSigned:\n\t\tif record.Protection != "signed" && record.Protection != "encrypted" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/signed_type_encrypted_protection",
        expect="KILLED",
        note="Admits exactly encrypted protection for the signed reasoning type; every other mismatched protection still refuses.",
    ),
    Mutant(
        name="N-protected-tools-skip",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\tif record.Protection != "none" && !isReasoningType(record.NativeType) {\n\t\t\tcontinue\n\t\t}\n''',
        replace='''\t\tif record.Protection != "none" && !isReasoningType(record.NativeType) && record.NativeEventID != "evt-prot-call" {\n\t\t\tcontinue\n\t\t}\n''',
        count=1,
        run="TestNormalizeProtectedToolCallStaysOpaqueHistory",
        expect="KILLED",
        note="Admits exactly the killer's protected tool call into pairing, where its ciphertext body refuses; every other protected non-reasoning record still skips folding.",
    ),
    Mutant(
        name="N-result-status-pending",
        path=f"{PACKAGE}/native.go",
        find='''\tif status != "ok" && status != "error" {\n''',
        replace='''\tif status != "ok" && status != "error" && status != "pending" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/bad_result_status",
        expect="KILLED",
        note="Admits exactly the pending result status, resolving the call completed; every other out-of-vocabulary status still refuses.",
    ),
    Mutant(
        name="N-size-verify",
        path=f"{PACKAGE}/normalize.go",
        find='''\t\tif uint64(len(payload)) != entry.ByteCount {\n''',
        replace='''\t\tif uint64(len(payload)) != entry.ByteCount && item.NativeItemKey != "store/session.jsonl" {\n''',
        count=1,
        run="TestNormalizeRefusalTable/blob_size_drift",
        expect="KILLED",
        note="Skips the size arm for exactly store/session.jsonl, so the drift reaches the digest arm with its different message; every other member still fails at the size arm.",
    ),
    Mutant(
        name="N-actors-sort",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tsort.Strings(selectors)\n''',
        replace='''\tif len(selectors) != 3 {\n\t\tsort.Strings(selectors)\n\t}\n''',
        count=1,
        run="TestNormalizeActorOrderIsDeterministic",
        expect="KILLED",
        note="Leaves exactly the three-actor shape unsorted so map order leaks into the sealed session; every other actor count still sorts.",
    ),
    Mutant(
        name="N-unknown-protected-body",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tif _, err := decodeProtectedBody(record); err != nil {\n\t\treturn err\n\t}\n''',
        replace='''\tif _, err := decodeProtectedBody(record); err != nil && record.NativeEventID != "evt-unk-prot" {\n\t\treturn err\n\t}\n''',
        count=1,
        run="TestNormalizeRefusalTable/unknown_protected_plaintext_body",
        expect="KILLED",
        note="Admits exactly the killer's non-ciphertext protected body past the body-shape gate into an opaque projection; every other misshapen protected body still refuses.",
    ),
    Mutant(
        name="N-alias-native-type",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif err := json.Unmarshal(members[field.name], field.target); err != nil {\n''',
        replace='''\t\traw := members[field.name]\n\t\tif field.name == "native_type" {\n\t\t\tif alias, ok := members["NATIVE_TYPE"]; ok {\n\t\t\t\traw = alias\n\t\t\t}\n\t\t}\n\t\tif err := json.Unmarshal(raw, field.target); err != nil {\n''',
        count=1,
        run="TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/kind_alias_after",
        expect="KILLED",
        note="Restores the case-folded read for exactly the native_type member: the NATIVE_TYPE alias coerces the unknown record into user_message; every other alias stays an unclaimed extra.",
    ),
    Mutant(
        name="N-alias-origin",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif err := json.Unmarshal(members[field.name], field.target); err != nil {\n''',
        replace='''\t\traw := members[field.name]\n\t\tif field.name == "origin" {\n\t\t\tif alias, ok := members["Origin"]; ok {\n\t\t\t\traw = alias\n\t\t\t}\n\t\t}\n\t\tif err := json.Unmarshal(raw, field.target); err != nil {\n''',
        count=1,
        run="TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/origin_alias",
        expect="KILLED",
        note="Restores the case-folded read for exactly the origin member: the Origin alias promotes the foreign instruction into the effective snapshot; every other alias stays an unclaimed extra.",
    ),
    Mutant(
        name="N-alias-protection",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif err := json.Unmarshal(members[field.name], field.target); err != nil {\n''',
        replace='''\t\traw := members[field.name]\n\t\tif field.name == "protection" {\n\t\t\tif alias, ok := members["Protection"]; ok {\n\t\t\t\traw = alias\n\t\t\t}\n\t\t}\n\t\tif err := json.Unmarshal(raw, field.target); err != nil {\n''',
        count=1,
        run="TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/protection_alias",
        expect="KILLED",
        note="Restores the case-folded read for exactly the protection member: the Protection alias downgrades the encrypted record to a promoted reasoning_summary instead of refusing; every other alias stays an unclaimed extra.",
    ),
    Mutant(
        name="N-alias-body",
        path=f"{PACKAGE}/native.go",
        find='''\tenvelope.Body = members["body"]\n''',
        replace='''\tbody := members["body"]\n\tif alias, ok := members["Body"]; ok {\n\t\tbody = alias\n\t}\n\tenvelope.Body = body\n''',
        count=1,
        run="TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/body_alias",
        expect="KILLED",
        note="Restores the case-folded read for exactly the body member: sealed content comes from the Body alias; every other alias stays an unclaimed extra.",
    ),
    Mutant(
        name="N-alias-native-event-id",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif err := json.Unmarshal(members[field.name], field.target); err != nil {\n''',
        replace='''\t\traw := members[field.name]\n\t\tif field.name == "native_event_id" {\n\t\t\tif alias, ok := members["Native_Event_Id"]; ok {\n\t\t\t\traw = alias\n\t\t\t}\n\t\t}\n\t\tif err := json.Unmarshal(raw, field.target); err != nil {\n''',
        count=1,
        run="TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/evidence_alias",
        expect="KILLED",
        note="Restores the case-folded read for exactly the native_event_id member: the evidence identity is rewritten by the alias; every other alias stays an unclaimed extra.",
    ),
    Mutant(
        name="N-alias-actor",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif err := json.Unmarshal(members[field.name], field.target); err != nil {\n''',
        replace='''\t\traw := members[field.name]\n\t\tif field.name == "actor" {\n\t\t\tif alias, ok := members["Actor"]; ok {\n\t\t\t\traw = alias\n\t\t\t}\n\t\t}\n\t\tif err := json.Unmarshal(raw, field.target); err != nil {\n''',
        count=1,
        run="TestNormalizeCaseAliasEnvelopeIgnoresUnclaimedMembers/actor_alias",
        expect="KILLED",
        note="Restores the case-folded read for exactly the actor member: the Actor alias re-attributes the unmapped subagent record to main instead of refusing; every other alias stays an unclaimed extra.",
    ),
    Mutant(
        name="N-reasoning-summary-routed",
        path=f"{PACKAGE}/dispatch.go",
        find='''\tdefault:\n\t\tunit.kind = "reasoning_summary"\n\t\tunit.visibility = "internal"\n''',
        replace='''\tdefault:\n\t\tunit.kind = "user_message"\n\t\tunit.visibility = "public"\n''',
        count=1,
        run="TestNormalizeReasoningSummaryProjectsInternal",
        expect="KILLED",
        note="Routes exactly the unprotected reasoning/summary records to user_message with public visibility; every other message kind still routes by its own arm.",
    ),
    Mutant(
        name="N-capture-status-partial",
        path=f"{PACKAGE}/dispatch.go",
        find='''\t\tCaptureStatus: "exact",\n''',
        replace='''\t\tCaptureStatus: map[bool]string{true: "partial", false: "exact"}[record.NativeType == nativeMessageUser],\n''',
        count=1,
        run="TestNormalizeCaptureStatusExactForEveryKind",
        expect="KILLED",
        note="Stamps partial for exactly message/user records while nothing is partial; every other record still stamps exact.",
    ),
    Mutant(
        name="N-instruction-fold-first",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif observation.record.Origin != "native" {\n\t\t\tcontinue\n\t\t}\n''',
        replace='''\t\tif observation.record.Origin != "native" || effective.Found {\n\t\t\tcontinue\n\t\t}\n''',
        count=1,
        run="TestNormalizeLastNativeInstructionWins",
        expect="KILLED",
        note="Folds exactly the first native instruction into the snapshot instead of the last; foreign records still never reach the snapshot.",
    ),
    Mutant(
        name="N-external-actor-kind",
        path=f"{PACKAGE}/dispatch.go",
        find='''\t\tif strings.HasPrefix(selector, "subagent:") {\n''',
        replace='''\t\tif strings.HasPrefix(selector, "subagent:") || selector == "external" {\n''',
        count=1,
        run="TestNormalizeExternalActorSealsAsExternal",
        expect="KILLED",
        note="Admits exactly the external selector into the subagent kind arm; every other non-subagent selector still seals external.",
    ),
    Mutant(
        name="N-required-member-body",
        path=f"{PACKAGE}/native.go",
        find='''\tfor _, name := range []string{"v", "native_event_id", "native_type", "origin", "protection", "actor", "body"} {\n''',
        replace='''\tfor _, name := range []string{"v", "native_event_id", "native_type", "origin", "protection", "actor"} {\n''',
        count=1,
        run="TestNormalizeRefusalTable/unknown_type_without_body",
        expect="KILLED",
        note="Drops body from the required envelope members: a known type still refuses at the body decode, but an unknown type without a body is admitted as opaque_event.",
    ),
    Mutant(
        name="N-body-text-number",
        path=f"{PACKAGE}/native.go",
        find='''\tvar value string\n\tif err := json.Unmarshal(members[name], &value); err != nil {\n''',
        replace='''\tvar value string\n\tif name == "text" && string(members[name]) == "123" {\n\t\treturn "123", nil\n\t}\n\tif err := json.Unmarshal(members[name], &value); err != nil {\n''',
        count=1,
        run="TestNormalizeRefusalTable/body_text_number",
        expect="KILLED",
        note="Admits exactly the JSON number 123 for the text member past the string gate; every other non-string still refuses.",
    ),
    Mutant(
        name="N-body-required-skip-text",
        path=f"{PACKAGE}/native.go",
        find='''\tfor _, name := range required {\n\t\tif _, ok := members[name]; !ok {\n''',
        replace='''\tfor _, name := range required {\n\t\tif name == "text" {\n\t\t\tcontinue\n\t\t}\n\t\tif _, ok := members[name]; !ok {\n''',
        count=1,
        run="TestNormalizeRefusalTable/body_missing_text",
        expect="KILLED",
        note="Skips the required-member check for exactly text: the record still refuses, but with the type gate's message instead of the required-member message.",
    ),
    Mutant(
        name="N-body-bool-string",
        path=f"{PACKAGE}/native.go",
        find='''\tvar value bool\n\tif err := json.Unmarshal(raw, &value); err != nil {\n''',
        replace='''\tvar value bool\n\tif name == "live_attestation" && string(raw) == `"true"` {\n\t\treturn true, nil\n\t}\n\tif err := json.Unmarshal(raw, &value); err != nil {\n''',
        count=1,
        run="TestNormalizeRefusalTable/body_live_attestation_string",
        expect="KILLED",
        note='Admits exactly the string "true" for live_attestation past the boolean gate; the record then refuses at the live-claim gate with its different message.',
    ),
    Mutant(
        name="N-directive-empty",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif environ.StringLength(directive) < 1 || environ.StringLength(directive) > 4096 {\n''',
        replace='''\t\tif environ.StringLength(directive) < 0 || environ.StringLength(directive) > 4096 {\n''',
        count=1,
        run="TestNormalizeRefusalTable/empty_directive",
        expect="KILLED",
        note="Admits exactly the empty directive past the 1..4096 bound; every longer directive still checks both edges.",
    ),
    Mutant(
        name="N-offset-newline",
        path=f"{PACKAGE}/native.go",
        find='''\t\toffset += uint64(len(raw)) + 1\n''',
        replace='''\t\toffset += uint64(len(raw))\n''',
        count=1,
        run="TestNormalizeMultiRecordRefsResolveAtAssertedOffsets",
        expect="KILLED",
        note="Behaviour swap (labelled, not a narrowing): drops the newline stride from the offset accumulation, so every record after line 1 references shifted bytes; the asserted offsets redden.",
    ),
    Mutant(
        name="N-usage-overflow-output",
        path=f"{PACKAGE}/tools.go",
        find='''\t\tif output < ledger.SourceOutputTokens || output > maxUint53 {\n''',
        replace='''\t\tif output < ledger.SourceOutputTokens || output > maxUint53+1 {\n''',
        count=1,
        run="TestNormalizeRefusalTable/usage_ledger_overflow_output",
        expect="KILLED",
        note="Admits exactly the 2^53 output total; anything larger still refuses instead of wrapping. The input arm keeps its own row.",
    ),
    Mutant(
        name="N-allowed-call-live-followup",
        path=f"{PACKAGE}/native.go",
        find='''map[string]bool{"call_id": true, "tool_name": true}''',
        replace='''map[string]bool{"call_id": true, "tool_name": true, "live_followup": true}''',
        count=1,
        run="TestNormalizeRefusalTable/call_live_followup_member",
        expect="KILLED",
        note="Admits exactly live_followup into the call registry; the call decoder never reads it, so the call projects as aborted history and the refusal row reddens. Every other unknown call member still refuses.",
    ),
    Mutant(
        name="N-allowed-result-live-attestation",
        path=f"{PACKAGE}/native.go",
        find='''map[string]bool{"call_id": true, "status": true, "live_followup": true}''',
        replace='''map[string]bool{"call_id": true, "status": true, "live_followup": true, "live_attestation": true}''',
        count=1,
        run="TestNormalizeRefusalTable/result_live_attestation_member",
        expect="KILLED",
        note="Admits exactly live_attestation into the result registry; the result decoder never reads it, so the orphan result projects as aborted history and the refusal row reddens. Every other unknown result member still refuses.",
    ),
    Mutant(
        name="N-actor-empty-subagent-name",
        path=f"{PACKAGE}/native.go",
        find='''\tif !ok || name == "" {\n''',
        replace='''\tif !ok {\n''',
        count=1,
        run="TestNormalizeRefusalTable/empty_subagent_name",
        expect="KILLED",
        note="Admits exactly the empty subagent name past the shape arm; the record still refuses one step later at the actor mapping with its different message, so the killer pins the shape message (message pin).",
    ),
    Mutant(
        name="N-unterminated-tail-drop",
        path=f"{PACKAGE}/native.go",
        find='''\t\tif end < 0 {\n\t\t\traw = rest\n\t\t\trest = nil\n''',
        replace='''\t\tif end < 0 {\n\t\t\tif number > 1 {\n\t\t\t\tbreak\n\t\t\t}\n\t\t\traw = rest\n\t\t\trest = nil\n''',
        count=1,
        run="TestNormalizeUnterminatedTailResolvesWithOffset",
        expect="KILLED",
        note="Labelled behaviour drop (not a narrowing): drops the unterminated last line of a multi-line member, so the tail event is missing and its asserted offset reddens; single-line and newline-terminated members are unaffected.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/doc.go",
        find="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections\n",
        replace="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections (control).\n",
        count=1,
        run="TestNormalizeSourceUsageIsNotTargetAccounting",
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
