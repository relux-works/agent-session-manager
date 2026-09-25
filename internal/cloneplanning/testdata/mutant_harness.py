#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/cloneplanning.

Every gate ships a narrowing mutant: the gate stays present and is
weakened to admit exactly one member of the class it must reject,
and the named killer test must fail when run alone. A delete-only
mutant proves only that the gate exists and is not accepted as
evidence.

Each row runs M (mutate one production file in place), R (run the
scoped Go killer through a production entry), V (verdict from the Go
exit code plus a FAIL kill-line: KILLED iff R fails with a test
failure line, SURVIVED iff R passes, ERROR for anything else --
unapplied patches, build breaks, timeouts). M failures and R
infrastructure errors are ERROR, never kills or survivals. The file
is restored from its backup after every row, pass or fail.

Shared gates (the checkItem vocabulary and fact gates with two call
sites: PlanItem and ClassifyItem) carry one plant each with the
both-entries killer TestSharedItemVocabBothEntries.

Row N-visible-escape-suffix is the token-preserving attack: the
JSON token still leads the output but the behavior changes, and the
killer executes the behavioral escaping suite.

Single-member-class rows narrow a gate whose reject class holds
exactly one member, so the plant admits the whole class, which is
one member.

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/cloneplanning/testdata/mutant_harness.py [--log-dir DIR] [MUTANT ...]
"""

import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PACKAGE = "internal/cloneplanning"


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
        name="N-select-archive-priority",
        path=f"{PACKAGE}/strategy.go",
        find='''\tif archive {\n\t\treturn strategyArchive, nil\n\t}\n\tif continuation {\n\t\treturn strategyContinuation, nil\n\t}''',
        replace='''\tif continuation {\n\t\treturn strategyContinuation, nil\n\t}\n\tif archive {\n\t\treturn strategyArchive, nil\n\t}''',
        count=1,
        run="TestSelectStrategyOracle",
        expect="KILLED",
        note="Narrows the priority gate: archive+continuation selects continuation instead of archive.",
    ),
    Mutant(
        name="N-select-official",
        path=f"{PACKAGE}/strategy.go",
        find='''\tif !sameEnvironment {\n\t\tif officialImport {\n\t\t\treturn strategyOfficial, nil\n\t\t}\n\t\treturn strategyNativeWriter, nil\n\t}\n\treturn strategyNativeRewrite, nil''',
        replace='''\tif officialImport {\n\t\treturn strategyOfficial, nil\n\t}\n\tif !sameEnvironment {\n\t\treturn strategyNativeWriter, nil\n\t}\n\treturn strategyNativeRewrite, nil''',
        count=1,
        run="TestSelectStrategyOracle",
        expect="KILLED",
        note="Narrows the pair arm: same-environment with the importer selects official import instead of native rewrite.",
    ),
    Mutant(
        name="N-select-profile-archive",
        path=f"{PACKAGE}/strategy.go",
        find='''\tif requested == profileArchive {\n\t\treturn BranchArchive, requested, nil\n\t}''',
        replace='''\tif requested == profileArchive {\n\t\treturn BranchTarget, requested, nil\n\t}''',
        count=1,
        run="TestSelectProfileOracle",
        expect="KILLED",
        note="Narrows the branch router: archive_only routes target instead of archive.",
    ),
    Mutant(
        name="N-select-profile-unknown",
        path=f"{PACKAGE}/strategy.go",
        find='''\tif !cloneplan.ValidPlanProfile(requested) {''',
        replace='''\tif requested != "bogus_profile" && !cloneplan.ValidPlanProfile(requested) {''',
        count=1,
        run="TestSelectProfileOracle",
        expect="KILLED",
        note="Narrows the plan-profile gate: admits exactly bogus_profile to the target branch.",
    ),
    Mutant(
        name="N-kind-vocab",
        path=f"{PACKAGE}/item.go",
        find='''\t\tif !clonebundle.ValidEventKind(item.Kind) {''',
        replace='''\t\tif item.Kind != "bogus_kind" && !clonebundle.ValidEventKind(item.Kind) {''',
        count=1,
        run="TestSharedItemVocabBothEntries",
        expect="KILLED",
        note="Shared gate: admits exactly bogus_kind through both PlanItem and ClassifyItem.",
    ),
    Mutant(
        name="N-block-vocab",
        path=f"{PACKAGE}/item.go",
        find='''\t\tif !clonebundle.ValidContentBlockType(item.Block) {''',
        replace='''\t\tif item.Block != "bogus_block" && !clonebundle.ValidContentBlockType(item.Block) {''',
        count=1,
        run="TestSharedItemVocabBothEntries",
        expect="KILLED",
        note="Shared gate: admits exactly bogus_block through PlanItem (blocks have no event bridge).",
    ),
    Mutant(
        name="N-resolution-vocab",
        path=f"{PACKAGE}/item.go",
        find='''\tif !validResolution(item.Resolution) {''',
        replace='''\tif item.Resolution != "bogus_resolution" && !validResolution(item.Resolution) {''',
        count=1,
        run="TestSharedItemVocabBothEntries",
        expect="KILLED",
        note="Shared gate: admits exactly bogus_resolution through both entries.",
    ),
    Mutant(
        name="N-protection-vocab",
        path=f"{PACKAGE}/item.go",
        find='''\tif !validProtection(item.Protection) {''',
        replace='''\tif item.Protection != "bogus_protection" && !validProtection(item.Protection) {''',
        count=1,
        run="TestSharedItemVocabBothEntries",
        expect="KILLED",
        note="Shared gate: admits exactly bogus_protection through both entries.",
    ),
    Mutant(
        name="N-classify-resolution-required",
        path=f"{PACKAGE}/item.go",
        find='''\tif toolKinds[item.Kind] {\n\t\tif item.Resolution == "" {\n\t\t\treturn invalid("projection item kind %q requires its completed|aborted resolution", item.Kind)\n\t\t}''',
        replace='''\tif toolKinds[item.Kind] {\n\t\tif item.Resolution == "" && item.Kind != "tool_call" {\n\t\t\treturn invalid("projection item kind %q requires its completed|aborted resolution", item.Kind)\n\t\t}''',
        count=1,
        run="TestClassifyItemRefusals",
        expect="KILLED",
        note="Narrows the required-resolution gate: tool_call without resolution admits, tool_result still refuses.",
    ),
    Mutant(
        name="N-classify-protection-required",
        path=f"{PACKAGE}/item.go",
        find='''\tif item.Kind == "opaque_reasoning" {\n\t\tif item.Protection == "" {\n\t\t\treturn invalid("projection item kind opaque_reasoning requires its encrypted|signed protection")\n\t\t}''',
        replace='''\tif item.Kind == "opaque_reasoning" {\n\t\tif item.Protection == "" && item.Kind != "opaque_reasoning" {\n\t\t\treturn invalid("projection item kind opaque_reasoning requires its encrypted|signed protection")\n\t\t}''',
        count=1,
        run="TestClassifyItemRefusals",
        expect="KILLED",
        note="Single-member-class narrowing: the gate whose reject class is exactly opaque_reasoning-without-protection admits it.",
    ),
    Mutant(
        name="N-classify-fact-type",
        path=f"{PACKAGE}/item.go",
        find='''\tvalue, ok := raw.(string)\n\tif !ok {\n\t\treturn "", invalid("projection item fact %q is not a string", name)\n\t}\n\treturn value, nil''',
        replace='''\tif _, isInt := raw.(int64); isInt {\n\t\treturn "", nil\n\t}\n\tvalue, ok := raw.(string)\n\tif !ok {\n\t\treturn "", invalid("projection item fact %q is not a string", name)\n\t}\n\treturn value, nil''',
        count=1,
        run="TestClassifyItemRefusals",
        expect="KILLED",
        note="Narrows the fact-type gate: admits exactly int64-typed facts as absent (the landed decoder renders numbers int64); bool and object facts still refuse.",
    ),
    Mutant(
        name="N-strategy-vocab",
        path=f"{PACKAGE}/plan.go",
        find='''\tif !clonefidelity.ValidStrategy(strategy) {''',
        replace='''\tif strategy != "bogus_strategy" && !clonefidelity.ValidStrategy(strategy) {''',
        count=1,
        run="TestPlanItemInvalidArms",
        expect="KILLED",
        note="Narrows the strategy vocabulary delegation: admits exactly bogus_strategy.",
    ),
    Mutant(
        name="N-profile-vocab",
        path=f"{PACKAGE}/plan.go",
        find='''\tcase profileMaximalSafe:\n\t\treturn disposition, reasons, nil\n\tdefault:\n\t\treturn "", nil, invalid("projection profile %q is not a target plan profile", profile)''',
        replace='''\tcase profileMaximalSafe:\n\t\treturn disposition, reasons, nil\n\tcase "bogus_profile":\n\t\treturn disposition, reasons, nil\n\tdefault:\n\t\treturn "", nil, invalid("projection profile %q is not a target plan profile", profile)''',
        count=1,
        run="TestPlanItemInvalidArms",
        expect="KILLED",
        note="Narrows the profile overlay default: admits exactly bogus_profile as a pass-through.",
    ),
    Mutant(
        name="N-archive-strategy",
        path=f"{PACKAGE}/plan.go",
        find='''\tif strategy == strategyArchive || profile == profileArchive {''',
        replace='''\tif !(strategy == strategyArchive && profile == profileMaximalSafe) && (strategy == strategyArchive || profile == profileArchive) {''',
        count=1,
        run="TestPlanItemArchiveRule",
        expect="KILLED",
        note="Narrows the archive gate: admits exactly the archive_only/maximal_safe cell.",
    ),
    Mutant(
        name="N-archive-profile",
        path=f"{PACKAGE}/plan.go",
        find='''\tif strategy == strategyArchive || profile == profileArchive {''',
        replace='''\tif !(strategy == strategyNativeWriter && profile == profileArchive) && (strategy == strategyArchive || profile == profileArchive) {''',
        count=1,
        run="TestPlanItemArchiveRule",
        expect="KILLED",
        note="Narrows the archive gate: exempts exactly the writer/archive cell from the archive-branch refusal (it then refuses via the overlay default, which names no archive branch).",
    ),
    Mutant(
        name="N-opaque-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\t\tif item.Protection == protectionSigned {\n\t\t\treturn dispositionOpaquePreserved, []string{reasonForeignSigned}\n\t\t}''',
        replace='''\t\tif item.Protection == protectionSigned {\n\t\t\treturn dispositionOpaquePreserved, []string{reasonForeignEncrypted}\n\t\t}''',
        count=1,
        run="TestPlanItemOpaqueRule",
        expect="KILLED",
        note="Narrows the opaque-reason mapping: signed reasoning reports the encrypted reason; the encrypted arm is intact.",
    ),
    Mutant(
        name="N-opaque-event-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\tif item.Kind == "opaque_event" || item.Block == "opaque" {\n\t\treturn dispositionOpaquePreserved, []string{reasonUnknownNative}\n\t}''',
        replace='''\tif item.Kind == "opaque_event" && item.Protection == "" {\n\t\treturn dispositionOpaquePreserved, []string{reasonForeignEncrypted}\n\t}\n\tif item.Kind == "opaque_event" || item.Block == "opaque" {\n\t\treturn dispositionOpaquePreserved, []string{reasonUnknownNative}\n\t}''',
        count=1,
        run="TestPlanItemOpaqueRule",
        expect="KILLED",
        note="Narrows the opaque-event mapping: the unprotected opaque_event item reports the encrypted reason.",
    ),
    Mutant(
        name="N-abort-drop",
        path=f"{PACKAGE}/plan.go",
        find='''\tif (item.Kind == "tool_call" || item.Kind == "tool_result") && item.Resolution == cloneproject.StatusAborted {''',
        replace='''\tif (item.Kind == "tool_result") && item.Resolution == cloneproject.StatusAborted {''',
        count=1,
        run="TestPlanItemAbortRule",
        expect="KILLED",
        note="Narrows the abort arm: aborted tool_call falls to the strategy default; aborted tool_result is intact.",
    ),
    Mutant(
        name="N-abort-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\t\treturn dispositionSummarized, []string{reasonUnsafePending}''',
        replace='''\t\treturn dispositionSummarized, nil''',
        count=1,
        run="TestPlanItemAbortRule",
        expect="KILLED",
        note="Narrows the abort mapping: aborted history without its unsafe_pending_action reason.",
    ),
    Mutant(
        name="N-continuation-exact",
        path=f"{PACKAGE}/plan.go",
        find='''\tif strategy == strategyContinuation {\n\t\treturn dispositionSemantic, []string{reasonTargetNoEquivalent}\n\t}''',
        replace='''\tif strategy == strategyContinuation && item.Kind != "usage" {\n\t\treturn dispositionSemantic, []string{reasonTargetNoEquivalent}\n\t}''',
        count=1,
        run="TestPlanItemContinuationRule",
        expect="KILLED",
        note="Narrows the continuation clamp: usage plans exact under continuation_context.",
    ),
    Mutant(
        name="N-continuation-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\t\treturn dispositionSemantic, []string{reasonTargetNoEquivalent}''',
        replace='''\t\treturn dispositionSemantic, nil''',
        count=1,
        run="TestPlanItemContinuationRule",
        expect="KILLED",
        note="Narrows the continuation mapping: semantic without its target_no_equivalent reason.",
    ),
    Mutant(
        name="N-importer-tool",
        path=f"{PACKAGE}/plan.go",
        find='''\t\tif completeToolKinds[item.Kind] {\n\t\t\treturn dispositionSemantic, []string{reasonOfficialImporterLoss}\n\t\t}''',
        replace='''\t\tif completeToolKinds[item.Kind] && item.Kind != "tool_definition_snapshot" {\n\t\t\treturn dispositionSemantic, []string{reasonOfficialImporterLoss}\n\t\t}''',
        count=1,
        run="TestPlanItemImporterRule",
        expect="KILLED",
        note="Narrows the importer tool arm: tool_definition_snapshot plans exact under official import.",
    ),
    Mutant(
        name="N-importer-graph",
        path=f"{PACKAGE}/plan.go",
        find='''\t\tif subagentKinds[item.Kind] {\n\t\t\treturn dispositionSemantic, []string{reasonGraphFlattened}\n\t\t}''',
        replace='''\t\tif subagentKinds[item.Kind] && item.Kind != "subagent_completed" {\n\t\t\treturn dispositionSemantic, []string{reasonGraphFlattened}\n\t\t}''',
        count=1,
        run="TestPlanItemImporterRule",
        expect="KILLED",
        note="Narrows the importer graph arm: subagent_completed plans exact under official import.",
    ),
    Mutant(
        name="N-strict-admit",
        path=f"{PACKAGE}/plan.go",
        find='''\tcase profileStrictExact:\n\t\tif disposition != dispositionExact {''',
        replace='''\tcase profileStrictExact:\n\t\tif disposition != dispositionExact && disposition != dispositionSemantic {''',
        count=1,
        run="TestPlanItemStrictRule",
        expect="KILLED",
        note="Narrows the strict gate: admits exactly semantic alongside exact.",
    ),
    Mutant(
        name="N-compact-skip",
        path=f"{PACKAGE}/plan.go",
        find='''\tcase profileCompact:\n\t\tif disposition == dispositionSemantic {''',
        replace='''\tcase profileCompact:\n\t\tif disposition == dispositionSemantic && item.Kind != "usage" {''',
        count=1,
        run="TestPlanItemCompactRule",
        expect="KILLED",
        note="Narrows the compact clamp: usage keeps semantic under compact.",
    ),
    Mutant(
        name="N-compact-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\t\t\treturn dispositionSummarized, append(append([]string(nil), reasons...), reasonTargetContextLimit), nil''',
        replace='''\t\t\treturn dispositionSummarized, append([]string(nil), reasons...), nil''',
        count=1,
        run="TestPlanItemCompactRule",
        expect="KILLED",
        note="Narrows the compact clamp: summarized without the appended target_context_limit reason.",
    ),
    Mutant(
        name="N-messages-admit",
        path=f"{PACKAGE}/plan.go",
        find='''\tcase profileMessages:\n\t\tif isMessageClass(item) {''',
        replace='''\tcase profileMessages:\n\t\tif isMessageClass(item) || item.Kind == "usage" {''',
        count=1,
        run="TestPlanItemMessagesRule",
        expect="KILLED",
        note="Narrows the messages_only survivor set: usage survives alongside message classes.",
    ),
    Mutant(
        name="N-messages-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\t\tomitted := append(append([]string(nil), reasons...), reasonOperatorPolicy)''',
        replace='''\t\tomitted := append([]string(nil), reasons...)''',
        count=1,
        run="TestPlanItemMessagesRule",
        expect="KILLED",
        note="Narrows the messages_only overlay: omitted without the operator_policy reason.",
    ),
    Mutant(
        name="N-media-reason",
        path=f"{PACKAGE}/plan.go",
        find='''\t\tif mediaBlocks[item.Block] {\n\t\t\tomitted = append(omitted, reasonUnsupportedMedia)\n\t\t}''',
        replace='''\t\tif mediaBlocks[item.Block] && item.Block != "image" {\n\t\t\tomitted = append(omitted, reasonUnsupportedMedia)\n\t\t}''',
        count=1,
        run="TestPlanItemMessagesRule",
        expect="KILLED",
        note="Narrows the media overlay: image blocks omit without the unsupported_media_type reason.",
    ),
    Mutant(
        name="N-reason-order",
        path=f"{PACKAGE}/plan.go",
        find='''\tsorted := append([]string(nil), reasons...)\n\tsort.Strings(sorted)''',
        replace='''\tsorted := append([]string(nil), reasons...)\n\t_ = sort.Strings''',
        count=1,
        run="TestPlanItemMessagesRule",
        expect="KILLED",
        note="Narrows the reason emission: insertion order instead of sorted (the aborted+messages cell proves the order).",
    ),
    Mutant(
        name="N-session-empty",
        path=f"{PACKAGE}/plan.go",
        find='''\tif len(items) == 0 || len(items) > maxSessionItems {''',
        replace='''\tif len(items) > maxSessionItems {''',
        count=1,
        run="TestPlanSessionRefusals",
        expect="KILLED",
        note="Narrows the session bound: admits exactly the empty session.",
    ),
    Mutant(
        name="N-session-bound",
        path=f"{PACKAGE}/plan.go",
        find='''\tif len(items) == 0 || len(items) > maxSessionItems {''',
        replace='''\tif len(items) == 0 || len(items) > maxSessionItems+1 {''',
        count=1,
        run="TestPlanSessionRefusals",
        expect="KILLED",
        note="Narrows the session bound: admits exactly 1000001 items.",
    ),
    Mutant(
        name="N-count-swap",
        path=f"{PACKAGE}/plan.go",
        find='''\t\tcase dispositionExact:\n\t\t\tcounts.Exact++''',
        replace='''\t\tcase dispositionExact:\n\t\t\tcounts.Semantic++''',
        count=1,
        run="TestPlanSessionCounts",
        expect="KILLED",
        note="Narrows the count fold: exact mappings fund the semantic counter.",
    ),
    Mutant(
        name="N-effects-disposition",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(mapping.Disposition == "bogus_disposition") {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsRefusals",
        expect="KILLED",
        note="Narrows the delegated effects gate: skips the owner call for exactly bogus_disposition.",
    ),
    Mutant(
        name="N-effects-reason",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(len(mapping.Reasons) == 1 && mapping.Reasons[0] == "bogus_reason") {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsRefusals",
        expect="KILLED",
        note="Narrows the delegated effects gate: skips the owner call for exactly single bogus_reason sets.",
    ),
    Mutant(
        name="N-effects-order-skip",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(len(mapping.Reasons) == 2 && mapping.Reasons[0] == "unsafe_pending_action" && mapping.Reasons[1] == "operator_policy") {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsRefusals",
        expect="KILLED",
        note="Narrows the delegated effects gate: skips the owner call for exactly the unsorted unsafe/operator pair.",
    ),
    Mutant(
        name="N-effects-exact-reasons",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(mapping.Disposition == "exact" && len(mapping.Reasons) == 1 && mapping.Reasons[0] == "operator_policy") {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsRefusesMalformedReasonSets",
        expect="KILLED",
        note="Reviewer rev4 narrowing: skips the owner call for exactly the exact-with-operator_policy cell.",
    ),
    Mutant(
        name="N-effects-nonexact-bare",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(mapping.Disposition == "semantic" && len(mapping.Reasons) == 0) {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsRefusesMalformedReasonSets",
        expect="KILLED",
        note="Reviewer rev4 narrowing: skips the owner call for exactly the bare semantic cell.",
    ),
    Mutant(
        name="N-effects-reason-bounds",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(len(mapping.Reasons) == 1 && mapping.Reasons[0] == "") {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsOwnerRecordRules",
        expect="KILLED",
        note="Narrows the delegated effects gate: skips the owner call for exactly the empty reason.",
    ),
    Mutant(
        name="N-effects-reason-count",
        path=f"{PACKAGE}/effects.go",
        find='''\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}''',
        replace='''\t\tif !(len(mapping.Reasons) == 129) {\n\t\tif err := checkMappingViaOwner(fmt.Sprintf("target effects mapping[%d]", index), mapping); err != nil {\n\t\t\treturn TargetEffects{}, err\n\t\t}\n\t\t}''',
        count=1,
        run="TestPlanTargetEffectsOwnerRecordRules",
        expect="KILLED",
        note="Narrows the delegated effects gate: skips the owner call for exactly 129-reason sets.",
    ),
    Mutant(
        name="N-planitem-owner-bypass",
        path=f"{PACKAGE}/plan.go",
        find='''\tif err := checkMappingViaOwner("projection planner emitted mapping", mapping); err != nil {\n\t\treturn Mapping{}, err\n\t}\n\treturn mapping, nil''',
        replace='''\treturn mapping, nil''',
        count=1,
        run="TestDelegationReachesOwner",
        expect="KILLED",
        note="Presence plant on the second delegation edge: removes PlanItem's self-check. Behaviorally silent (the emitted-valid sweep proves the arm never fires); the reachability census kills it, and the shared helper's behavior is measured through PlanTargetEffects' table.",
    ),
    Mutant(
        name="N-effects-nonzero",
        path=f"{PACKAGE}/effects.go",
        find='''\t\t}\n\t}\n\treturn TargetEffects{}, nil\n}''',
        replace='''\t\t}\n\t}\n\teffects := TargetEffects{}\n\tfor _, mapping := range mappings {\n\t\tif mapping.Disposition == dispositionExact {\n\t\t\teffects.TargetInputTokens = 1\n\t\t}\n\t}\n\treturn effects, nil\n}''',
        count=1,
        run="TestPlanTargetEffectsZeroSweep",
        expect="KILLED",
        note="Narrows the zero fold: exact mappings fund one target input token.",
    ),
    Mutant(
        name="N-visible-kind",
        path=f"{PACKAGE}/visible.go",
        find='''\tif !clonebundle.ValidEventKind(input.Kind) {''',
        replace='''\tif input.Kind != "bogus_kind" && !clonebundle.ValidEventKind(input.Kind) {''',
        count=1,
        run="TestProjectVisibleKindRefusals",
        expect="KILLED",
        note="Narrows the visible kind gate: admits exactly bogus_kind.",
    ),
    Mutant(
        name="N-visible-ordinal",
        path=f"{PACKAGE}/visible.go",
        find='''\tif _, err := scalar.NewUint53(input.Ordinal); err != nil {\n\t\treturn VisibleProjection{}, invalid("visible projection ordinal %d is not a uint53: %v", input.Ordinal, err)\n\t}''',
        replace='''\tif input.Ordinal != 9007199254740992 {\n\t\tif _, err := scalar.NewUint53(input.Ordinal); err != nil {\n\t\t\treturn VisibleProjection{}, invalid("visible projection ordinal %d is not a uint53: %v", input.Ordinal, err)\n\t\t}\n\t}''',
        count=1,
        run="TestProjectVisibleOrdinalEdges",
        expect="KILLED",
        note="Narrows the ordinal gate: admits exactly 9007199254740992.",
    ),
    Mutant(
        name="N-visible-utf8",
        path=f"{PACKAGE}/visible.go",
        find='''\tfor index, field := range input.Texts {\n\t\tif !validText(field) {''',
        replace='''\tfor index, field := range input.Texts {\n\t\tif !validText(field) && index != 1 {''',
        count=1,
        run="TestProjectVisibleTextRefusals",
        expect="KILLED",
        note="Narrows the UTF-8 gate: skips exactly field[1]; the direct entry then refuses with the unindexed message, failing the field[ cell.",
    ),
    Mutant(
        name="N-visible-max",
        path=f"{PACKAGE}/visible.go",
        find='''\tif length := environ.StringLength(joined); length < 1 || length > maxEscapedText {''',
        replace='''\tif length := environ.StringLength(joined); length < 1 || length > maxEscapedText+1 {''',
        count=1,
        run="TestProjectVisibleBounds",
        expect="KILLED",
        note="Narrows the escaped_text bound: admits exactly 65537 characters.",
    ),
    Mutant(
        name="N-visible-empty",
        path=f"{PACKAGE}/visible.go",
        find='''\tif length := environ.StringLength(joined); length < 1 || length > maxEscapedText {''',
        replace='''\tif length := environ.StringLength(joined); length < 0 || length > maxEscapedText {''',
        count=1,
        run="TestProjectVisibleBounds",
        expect="KILLED",
        note="Narrows the escaped_text bound: admits exactly the empty text.",
    ),
    Mutant(
        name="N-visible-ids-count0",
        path=f"{PACKAGE}/visible.go",
        find='''\tif len(ids) < 1 || len(ids) > maxVisibleEventIDs {''',
        replace='''\tif len(ids) > maxVisibleEventIDs {''',
        count=1,
        run="TestProjectVisibleEventIDGrid",
        expect="KILLED",
        note="Narrows the event-ID count gate: admits exactly zero IDs.",
    ),
    Mutant(
        name="N-visible-ids-count65",
        path=f"{PACKAGE}/visible.go",
        find='''\tif len(ids) < 1 || len(ids) > maxVisibleEventIDs {''',
        replace='''\tif len(ids) < 1 || len(ids) > maxVisibleEventIDs+1 {''',
        count=1,
        run="TestProjectVisibleEventIDGrid",
        expect="KILLED",
        note="Narrows the event-ID count gate: admits exactly 65 IDs.",
    ),
    Mutant(
        name="N-visible-ids-order",
        path=f"{PACKAGE}/visible.go",
        find='''\tfor index := 1; index < len(checked); index++ {\n\t\tif checked[index-1] >= checked[index] {''',
        replace='''\tfor index := 2; index < len(checked); index++ {\n\t\tif checked[index-1] >= checked[index] {''',
        count=1,
        run="TestProjectVisibleEventIDGrid",
        expect="KILLED",
        note="Narrows the event-ID order gate: skips exactly the index-1 adjacency.",
    ),
    Mutant(
        name="N-visible-ids-digest",
        path=f"{PACKAGE}/visible.go",
        find='''\t\tdigest, err := scalar.ParseDigest(id)\n\t\tif err != nil {''',
        replace='''\t\tdigest, err := scalar.ParseDigest(id)\n\t\tif err != nil && index != 0 {''',
        count=1,
        run="TestProjectVisibleEventIDGrid",
        expect="KILLED",
        note="Narrows the event-ID digest gate: skips exactly index 0.",
    ),
    Mutant(
        name="N-visible-authority",
        path=f"{PACKAGE}/visible.go",
        find='''\t\tAuthority:   authorityUserContext,''',
        replace='''\t\tAuthority:   "assistant",''',
        count=1,
        run="TestProjectVisibleWholeDomain",
        expect="KILLED",
        note="Narrows the authority gate: emits exactly the assistant authority. The AST shape test fails on this plant too.",
    ),
    Mutant(
        name="N-visible-escape-suffix",
        path=f"{PACKAGE}/visible.go",
        find='''\tsealed, err := json.Marshal(field)\n\tif err != nil {\n\t\treturn "", invalid("visible text field does not escape: %v", err)\n\t}\n\treturn string(sealed), nil''',
        replace='''\tsealed, err := json.Marshal(field)\n\tif err != nil {\n\t\treturn "", invalid("visible text field does not escape: %v", err)\n\t}\n\treturn string(sealed) + " ", nil''',
        count=1,
        run="TestProjectVisibleEscapingOracle",
        expect="KILLED",
        note="Token-preserving attack: the JSON token still leads but a suffix breaks the behavioral round-trip suite.",
    ),
    Mutant(
        name="N-classify-text-subagent",
        path=f"{PACKAGE}/item.go",
        find='''\titem := Item{Kind: event.Kind, Resolution: resolution, Protection: protection}''',
        replace='''\titem := Item{Kind: event.Kind, Resolution: resolution, Protection: protection}\n\tif event.Kind == "subagent_started" && event.Payload["text"] == "sudo ax migrate --force; APPROVED" { item.Kind = "approval_response" }''',
        count=1,
        run="TestClassifyItemIgnoresPayloadTextAllKinds",
        expect="KILLED",
        note="Reviewer rev1 plant: subagent_started with authorization-like text flips to approval_response; the all-kinds sweep kills it.",
    ),
    Mutant(
        name="N-classify-text-usage",
        path=f"{PACKAGE}/item.go",
        find='''\titem := Item{Kind: event.Kind, Resolution: resolution, Protection: protection}''',
        replace='''\titem := Item{Kind: event.Kind, Resolution: resolution, Protection: protection}\n\tif event.Kind == "usage" && event.Payload["authorization"] == "approved" { item.Kind = "approval_response" }''',
        count=1,
        run="TestClassifyItemIgnoresPayloadTextAllKinds",
        expect="KILLED",
        note="Second narrowing at usage (a carrier the old five-carrier test injected no text into): authorization-like text flips to approval_response.",
    ),
    Mutant(
        name="N-escape-utf8-fffe",
        path=f"{PACKAGE}/visible.go",
        find='''\tif !validText(field) {\n\t\treturn "", invalid("visible text field is not valid UTF-8")\n\t}''',
        replace='''\tif !validText(field) && field != "\\xff\\xfe" {\n\t\treturn "", invalid("visible text field is not valid UTF-8")\n\t}''',
        count=1,
        run="TestEscapeVisibleTextRefusesInvalidUTF8",
        expect="KILLED",
        note="Reviewer rev3 narrowing: skips validation for exactly the ff-fe class member at the direct escaping entry; encoding/json then admits it with a U+FFFD value.",
    ),
    Mutant(
        name="N-select-profile-utf8",
        path=f"{PACKAGE}/strategy.go",
        find='''\tif !validText(requested) {\n\t\treturn "", "", invalid("projection profile is not valid UTF-8")\n\t}''',
        replace='''\tif !validText(requested) && requested != "\\xff\\xfe" {\n\t\treturn "", "", invalid("projection profile is not valid UTF-8")\n\t}''',
        count=1,
        run="TestSelectProfileRefusesInvalidUTF8",
        expect="KILLED",
        note="Second censused entry: skips validation for exactly the ff-fe member at profile selection; the vocabulary fallback refuses with another message, failing the literal cell.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/visible.go",
        find='''// authorityUserContext is the only visible-projection authority''',
        replace='''// authorityUserContext is the only visible-projection authority (control)''',
        count=1,
        run="TestAuthorityDecisionShape",
        expect="SURVIVED",
        note="Harmless control: a comment-only edit must survive, proving the harness observes outcomes.",
    ),
]


def run_row(mutant, log_dir):
    target = REPO / mutant.path
    original = target.read_text()
    found = original.count(mutant.find)
    log_path = log_dir / f"{mutant.name}.log"
    with open(log_path, "w") as log:
        log.write(f"mutant: {mutant.name}\n")
        log.write(f"path: {mutant.path}\n")
        log.write(f"run: {mutant.run}\n")
        log.write(f"expect: {mutant.expect}\n")
        log.write(f"note: {mutant.note}\n\n")
        if found != mutant.count:
            log.write(f"M-ERROR: find matched {found} times, want {mutant.count}\n")
            return "ERROR"
        target.write_text(original.replace(mutant.find, mutant.replace))
        log.write("M: applied\n")
        log.flush()
        started = time.time()
        try:
            proc = subprocess.run(
                ["go", "test", "-count=1", "-run", f"^{mutant.run}$", f"./{PACKAGE}/"],
                cwd=REPO,
                capture_output=True,
                text=True,
                timeout=180,
            )
        except subprocess.TimeoutExpired:
            log.write("R-ERROR: timeout after 180s\n")
            return "ERROR"
        finally:
            target.write_text(original)
            log.write("restored\n")
        elapsed = time.time() - started
        log.write(f"R: exit={proc.returncode} elapsed={elapsed:.1f}s\n")
        log.write("--- stdout ---\n")
        log.write(proc.stdout[-6000:])
        log.write("\n--- stderr ---\n")
        log.write(proc.stderr[-2000:])
        log.write("\n")
        failed = proc.returncode != 0 and ("FAIL" in proc.stdout or "FAIL" in proc.stderr)
        passed = proc.returncode == 0
        if failed:
            verdict = "KILLED"
        elif passed:
            verdict = "SURVIVED"
        else:
            verdict = "ERROR"
        log.write(f"V: {verdict}\n")
        return verdict


def main(argv):
    log_dir = None
    names = []
    args = list(argv)
    while args:
        arg = args.pop(0)
        if arg == "--log-dir" and args:
            log_dir = Path(args.pop(0))
        else:
            names.append(arg)
    wanted = [m for m in MUTANTS if not names or m.name in names]
    if names and len(wanted) != len(names):
        missing = set(names) - {m.name for m in wanted}
        print(f"unknown mutants: {sorted(missing)}")
        return 2
    if log_dir is None:
        log_dir = Path(tempfile.mkdtemp(prefix="cloneplanning-mut-"))
    else:
        log_dir.mkdir(parents=True, exist_ok=True)
    print(f"log dir: {log_dir}")
    verdicts = []
    mismatches = 0
    for mutant in wanted:
        verdict = run_row(mutant, log_dir)
        verdicts.append((mutant.name, verdict, mutant.expect))
        mark = "ok" if verdict == mutant.expect else "MISMATCH"
        if verdict != mutant.expect:
            mismatches += 1
        print(f"{mutant.name}: {verdict} (expect {mutant.expect}) [{mark}]")
    killed = sum(1 for _, v, _ in verdicts if v == "KILLED")
    survived = sum(1 for _, v, _ in verdicts if v == "SURVIVED")
    errors = sum(1 for _, v, _ in verdicts if v == "ERROR")
    print(f"summary: {killed} KILLED, {survived} SURVIVED, {errors} ERROR, {mismatches} mismatches")
    with open(log_dir / "verdicts.txt", "w") as f:
        for name, verdict, expect in verdicts:
            f.write(f"{name}: {verdict} (expect {expect})\n")
        f.write(f"summary: {killed} KILLED, {survived} SURVIVED, {errors} ERROR, {mismatches} mismatches\n")
    return 0 if mismatches == 0 and errors == 0 else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
