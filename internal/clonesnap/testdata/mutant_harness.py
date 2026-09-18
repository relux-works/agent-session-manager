#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/clonesnap.

Every gate ships a narrowing mutant: the gate stays present and is
weakened to admit exactly one member of the class it must reject, and
the named killer test must fail. A delete-only mutant proves only that
the gate exists and is not accepted as evidence.

Each row runs M (mutate one production file in place), R (run the
scoped Go killer through the production entry point), V (verdict from
the Go exit code plus a FAIL kill-line: KILLED iff R fails with a test
failure line, SURVIVED iff R passes, ERROR for anything else —
unapplied patches, build breaks, timeouts). M failures and R
infrastructure errors are ERROR, never kills or survivals. The file is
restored from its backup after every row, pass or fail.

Row N-plan-prefix is the token-preserving text mutant: the plan
lookup keeps comparing against the planned keys but changes exact
match to prefix match, and its killer drives the full Capture entry
(the behavioral suite), not a static checker.

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/clonesnap/testdata/mutant_harness.py [--log-dir DIR] [MUTANT ...]
"""

import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PACKAGE = "internal/clonesnap"

OPEN_MEMBER_HEAD = (
    "func openMember(guard secprim.Guard, root *os.File, member string, limit uint64) ([]byte, error) {\n"
    "\tfile, err := guard.Open(root, member)\n"
)

PLAN_LOOKUP = (
    '\titem, planned := plan[key]\n'
    '\tif !planned {\n'
    '\t\treturn "", invalid("capture store member %q is not a plan candidate", key)\n'
    '\t}\n'
    '\treturn item.Class, nil\n'
)

PROJECT_ORDER_PIN = (
    "\t// ORDER-PIN: the source-race gate precedes any seal, admission,\n"
    "\t// or projection output. N-race-order-project moves this gate\n"
    "\t// after admitAndEmit: the early seal then refuses with a seal\n"
    "\t// error instead of the race refusal, so that row's killer\n"
    "\t// reddens on the wrong error (the seal error message, not a\n"
    "\t// sink). N-race-order-project-seal keeps the race error while\n"
    "\t// sealing first: its killer reddens on the sealed raw manifest\n"
    "\t// in the capture store.\n"
    "\tif err := state.gateSourceRace(); err != nil {\n"
    "\t\tcaptured, raceErr := state.raceOutcome(err)\n"
    "\t\tif raceErr != nil {\n"
    "\t\t\treturn nil, raceErr\n"
    "\t\t}\n"
    "\t\t// Archive-only output never enters a target branch: the\n"
    "\t\t// admission gate below refuses it, so no receipt exists.\n"
    "\t\tif _, err := AdmitForTarget(captured.CaptureManifest, request.FidelityProfile); err != nil {\n"
    "\t\t\treturn nil, err\n"
    "\t\t}\n"
    '\t\treturn nil, invalid("capture archive unexpectedly admitted to a target branch")\n'
    "\t}\n"
    "\tsealed, err := state.sealAndPublish(false)\n"
    "\tif err != nil {\n"
    "\t\treturn nil, err\n"
    "\t}\n"
    "\treturn state.admitAndEmit(request, sealed)\n"
)


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
        name="N-contain-trailing",
        path=f"{PACKAGE}/walk.go",
        find=OPEN_MEMBER_HEAD,
        replace=(
            "func openMember(guard secprim.Guard, root *os.File, member string, limit uint64) ([]byte, error) {\n"
            '\tif member == "link-escape" {\n'
            '\t\treturn []byte("mutant-escaped-bytes"), nil\n'
            '\t}\n'
            "\tfile, err := guard.Open(root, member)\n"
        ),
        count=1,
        run="TestCaptureRefusesTrailingSymlink",
        expect="KILLED",
        note="Stands forged bytes in for the Guard open of exactly link-escape; every other member still opens through the Guard.",
    ),
    Mutant(
        name="N-contain-intermediate",
        path=f"{PACKAGE}/walk.go",
        find=OPEN_MEMBER_HEAD,
        replace=(
            "func openMember(guard secprim.Guard, root *os.File, member string, limit uint64) ([]byte, error) {\n"
            '\tif member == "link-escape/file" {\n'
            '\t\treturn []byte("mutant-escaped-bytes"), nil\n'
            '\t}\n'
            "\tfile, err := guard.Open(root, member)\n"
        ),
        count=1,
        run="TestCaptureRefusesSymlinkedIntermediate",
        expect="KILLED",
        note="Stands forged bytes in for the Guard open of exactly link-escape/file; every other member still opens through the Guard.",
    ),
    Mutant(
        name="N-fifo",
        path=f"{PACKAGE}/walk.go",
        find=OPEN_MEMBER_HEAD,
        replace=(
            "func openMember(guard secprim.Guard, root *os.File, member string, limit uint64) ([]byte, error) {\n"
            '\tif member == "stash-fifo" {\n'
            '\t\treturn []byte("mutant-fifo-bytes"), nil\n'
            '\t}\n'
            "\tfile, err := guard.Open(root, member)\n"
        ),
        count=1,
        run="TestCaptureRefusesFIFO",
        expect="KILLED",
        note="Stands forged bytes in for the Guard open of exactly stash-fifo; every other member still opens through the Guard.",
    ),
    Mutant(
        name="N-unplanned",
        path=f"{PACKAGE}/walk.go",
        find=PLAN_LOOKUP,
        replace=(
            '\titem, planned := plan[key]\n'
            '\tif !planned && key != "store/zz-extra" {\n'
            '\t\treturn "", invalid("capture store member %q is not a plan candidate", key)\n'
            '\t}\n'
            '\tif !planned {\n'
            '\t\treturn "durable_payload", nil\n'
            '\t}\n'
            '\treturn item.Class, nil\n'
        ),
        count=1,
        run="TestCaptureRefusesUnplannedMember",
        expect="KILLED",
        note="Admits exactly store/zz-extra past the plan gate as durable_payload; every other unplanned key still refuses.",
    ),
    Mutant(
        name="N-plan-prefix",
        path=f"{PACKAGE}/walk.go",
        find=PLAN_LOOKUP,
        replace=(
            '\tfor plannedKey, plannedItem := range plan {\n'
            '\t\tif strings.HasPrefix(key, plannedKey) {\n'
            '\t\t\treturn plannedItem.Class, nil\n'
            '\t\t}\n'
            '\t}\n'
            '\treturn "", invalid("capture store member %q is not a plan candidate", key)\n'
        ),
        count=1,
        run="TestCaptureRefusesPrefixSibling",
        expect="KILLED",
        note="TOKEN-PRESERVING: the lookup still compares against the planned keys but admits prefix matches, so store/blob-a-evil rides on store/blob-a. The killer drives the full Capture entry.",
    ),
    Mutant(
        name="N-plan-order",
        path=f"{PACKAGE}/walk.go",
        find="\t\tif index > 0 && item.NativeKey <= previous {\n",
        replace="\t\tif index > 0 && item.NativeKey < previous {\n",
        count=1,
        run="TestCaptureRefusesDuplicatePlanKey",
        expect="KILLED",
        note="Admits exactly duplicate plan keys; unsorted keys still refuse with the plan detail.",
    ),
    Mutant(
        name="N-plan-sanitize",
        path=f"{PACKAGE}/walk.go",
        find="\t\tif err := clonebundle.SanitizeNativeKey(item.NativeKey); err != nil {\n",
        replace='\t\tif err := clonebundle.SanitizeNativeKey(item.NativeKey); err != nil && item.NativeKey != "0198f4d1-9e60-7f66-8f86-f0123456789a" {\n',
        count=1,
        run="TestCaptureRefusesUnsanitizedPlanKey",
        expect="KILLED",
        note="Admits exactly one UUIDv7 plan key past the sanitizer; every other unsanitized key still refuses. Token preserved.",
    ),
    Mutant(
        name="N-plan-class",
        path=f"{PACKAGE}/walk.go",
        find="\t\tif !clonebundle.ValidCaptureClass(item.Class) {\n",
        replace='\t\tif !clonebundle.ValidCaptureClass(item.Class) && item.Class != "mystery" {\n',
        count=1,
        run="TestCaptureRefusesMysteryPlanClass",
        expect="KILLED",
        note="Admits exactly class mystery past the vocabulary gate; every other unknown class still refuses.",
    ),
    Mutant(
        name="N-exclude-credential",
        path=f"{PACKAGE}/walk.go",
        find='\tcase "credential", "machine_auth", "runtime_state", "transient_lock":\n',
        replace='\tcase "machine_auth", "runtime_state", "transient_lock":\n',
        count=1,
        run="TestCaptureExcludesSecrets",
        expect="KILLED",
        note="Admits exactly credential rows as included; the other three always-excluded classes still exclude.",
    ),
    Mutant(
        name="N-exclude-machine-auth",
        path=f"{PACKAGE}/walk.go",
        find='\tcase "credential", "machine_auth", "runtime_state", "transient_lock":\n',
        replace='\tcase "credential", "runtime_state", "transient_lock":\n',
        count=1,
        run="TestCaptureExcludesSecrets",
        expect="KILLED",
        note="Admits exactly machine_auth rows as included; the other three always-excluded classes still exclude.",
    ),
    Mutant(
        name="N-exclude-runtime-state",
        path=f"{PACKAGE}/walk.go",
        find='\tcase "credential", "machine_auth", "runtime_state", "transient_lock":\n',
        replace='\tcase "credential", "machine_auth", "transient_lock":\n',
        count=1,
        run="TestCaptureExcludesSecrets",
        expect="KILLED",
        note="Admits exactly runtime_state rows as included; the other three always-excluded classes still exclude.",
    ),
    Mutant(
        name="N-exclude-transient-lock",
        path=f"{PACKAGE}/walk.go",
        find='\tcase "credential", "machine_auth", "runtime_state", "transient_lock":\n',
        replace='\tcase "credential", "machine_auth", "runtime_state":\n',
        count=1,
        run="TestCaptureExcludesSecrets",
        expect="KILLED",
        note="Admits exactly transient_lock rows as included; the other three always-excluded classes still exclude.",
    ),
    Mutant(
        name="N-required",
        path=f"{PACKAGE}/capture.go",
        find="\t\tif !present && item.Required && !blockedAncestor(byKey, key) {\n",
        replace='\t\tif !present && item.Required && key != "store/must-have" && !blockedAncestor(byKey, key) {\n',
        count=1,
        run="TestCaptureRefusesRequiredMissing",
        expect="KILLED",
        note="Admits exactly store/must-have past the required-absent refusal; every other required-absent candidate still refuses.",
    ),
    Mutant(
        name="N-optional",
        path=f"{PACKAGE}/capture.go",
        find="\t\tif !present && !item.Required && !blockedAncestor(byKey, key) {\n",
        replace='\t\tif !present && !item.Required && key != "store/maybe" && !blockedAncestor(byKey, key) {\n',
        count=1,
        run="TestCaptureOptionalAbsentSealsExcluded",
        expect="KILLED",
        note="Refuses exactly the optional-absent store/maybe that must seal as excluded; every other optional-absent still seals.",
    ),
    Mutant(
        name="N-race-narrow",
        path=f"{PACKAGE}/race.go",
        find="\tif err := clonebundle.CheckDigestsEqual(pre, post); err != nil {\n",
        replace='\tif err := clonebundle.CheckDigestsEqual(pre, post); err != nil && firstChangedMember(preLines, postLines) != "store/blob-a" {\n',
        count=1,
        run="TestCaptureRaceChangedByte",
        expect="KILLED",
        note="Ignores exactly races naming store/blob-a; races on every other member still refuse.",
    ),
    Mutant(
        name="N-race-order-capture",
        path=f"{PACKAGE}/capture.go",
        find=(
            "\t// ORDER-PIN: the source-race gate precedes any seal or publish.\n"
            "\t// N-race-order-capture moves this gate after sealAndPublish and\n"
            "\t// the killer reddens on the published manifests.\n"
            "\tif err := state.gateSourceRace(); err != nil {\n"
            "\t\treturn state.raceOutcome(err)\n"
            "\t}\n"
            "\treturn state.sealAndPublish(false)\n"
        ),
        replace=(
            "\tsealed, sealErr := state.sealAndPublish(false)\n"
            "\tif sealErr != nil {\n"
            "\t\treturn nil, sealErr\n"
            "\t}\n"
            "\tif err := state.gateSourceRace(); err != nil {\n"
            "\t\t_, _ = state.raceOutcome(err)\n"
            "\t\treturn sealed, err\n"
            "\t}\n"
            "\treturn sealed, nil\n"
        ),
        count=1,
        run="TestCaptureRaceSealsNoManifest",
        expect="KILLED",
        note="ORDERING: seals and publishes before the race gate; the killer reddens on the published manifests and the non-nil result.",
    ),
    Mutant(
        name="N-race-order-project",
        path=f"{PACKAGE}/project.go",
        find=PROJECT_ORDER_PIN,
        replace=(
            "\tsealed, err := state.sealAndPublish(false)\n"
            "\tif err != nil {\n"
            "\t\treturn nil, err\n"
            "\t}\n"
            "\temitted, emitErr := state.admitAndEmit(request, sealed)\n"
            "\tif emitErr != nil {\n"
            "\t\treturn nil, emitErr\n"
            "\t}\n"
            "\tif err := state.gateSourceRace(); err != nil {\n"
            "\t\treturn emitted, err\n"
            "\t}\n"
            "\treturn emitted, nil\n"
        ),
        count=1,
        run="TestCaptureRaceEmitsNoReceipt",
        expect="KILLED",
        note="ORDERING: seals, admits and emits before the race gate; the early seal refuses with a seal error instead of the race refusal, so the killer reddens on the wrong error. Killed by the seal error message, not by a sink assertion — the seal half is pinned by N-race-order-project-seal.",
    ),
    Mutant(
        name="N-race-order-project-seal",
        path=f"{PACKAGE}/project.go",
        find=PROJECT_ORDER_PIN,
        replace=(
            "\tsealed, sealErr := state.sealAndPublish(false)\n"
            "\tif err := state.gateSourceRace(); err != nil {\n"
            "\t\tcaptured, raceErr := state.raceOutcome(err)\n"
            "\t\tif raceErr != nil {\n"
            "\t\t\treturn nil, raceErr\n"
            "\t\t}\n"
            "\t\tif _, err := AdmitForTarget(captured.CaptureManifest, request.FidelityProfile); err != nil {\n"
            "\t\t\treturn nil, err\n"
            "\t\t}\n"
            '\t\treturn nil, invalid("capture archive unexpectedly admitted to a target branch")\n'
            "\t}\n"
            "\tif sealErr != nil {\n"
            "\t\treturn nil, sealErr\n"
            "\t}\n"
            "\treturn state.admitAndEmit(request, sealed)\n"
        ),
        count=1,
        run="TestCaptureRaceEmitsNoReceipt",
        expect="KILLED",
        note="ORDERING at the projection entry: seals and publishes the raw manifest BEFORE the race gate while preserving the race error (the stable capture-manifest seal fails closed on unequal digests, so no receipt is emitted); the killer reddens on the sealed raw manifest in the capture store.",
    ),
    Mutant(
        name="N-explicit",
        path=f"{PACKAGE}/capture.go",
        find="\tif !state.request.OperatorExplicit {\n",
        replace='\tif !state.request.OperatorExplicit && state.request.Generation != "generation-9" {\n',
        count=1,
        run="TestCaptureArchiveRequiresExplicit",
        expect="KILLED",
        note="Seals unstable output without acknowledgement for exactly generation-9; every other generation still refuses.",
    ),
    Mutant(
        name="N-g2",
        path=f"{PACKAGE}/project.go",
        find=(
            "\tif err := clonebundle.RefuseUnstableForTarget(manifest.CaptureBoundary); err != nil {\n"
            "\t\treturn clonebundle.CaptureManifest{}, err\n"
            "\t}\n"
        ),
        replace=(
            '\tif err := clonebundle.RefuseUnstableForTarget(manifest.CaptureBoundary); err != nil && manifest.CaptureBoundary.SourceGeneration.String() != "generation-9" {\n'
            "\t\treturn clonebundle.CaptureManifest{}, err\n"
            "\t}\n"
        ),
        count=1,
        run="TestProjectRefusesUnstable",
        expect="KILLED",
        note="Admits exactly generation-9 unstable archives to the target branch; every other unstable still refuses.",
    ),
    Mutant(
        name="N-maximal",
        path=f"{PACKAGE}/project.go",
        find='\tif profile == "maximal_safe" {\n',
        replace='\tif profile == "maximal_safe" && len(manifest.Items) != 3 {\n',
        count=1,
        run="TestProjectMaximalSafeRequiresComplete",
        expect="KILLED",
        note="Admits exactly the three-item unknown manifest to maximal_safe; unknown manifests of every other size still refuse.",
    ),
    Mutant(
        name="N-fidelity",
        path=f"{PACKAGE}/project.go",
        find="\tdefault:\n\t\treturn false\n\t}\n",
        replace='\tcase FidelityProfile("ultra"):\n\t\treturn true\n\tdefault:\n\t\treturn false\n\t}\n',
        count=1,
        run="TestProjectRefusesUnknownProfile",
        expect="KILLED",
        note="Admits exactly profile ultra; every other unknown profile still refuses.",
    ),
    Mutant(
        name="N-oversize",
        path=f"{PACKAGE}/walk.go",
        find="\tif uint64(info.Size()) > limit {\n",
        replace="\tif uint64(info.Size()) > limit && uint64(info.Size()) != 11 {\n",
        count=1,
        run="TestCaptureRefusesOversizedMember",
        expect="KILLED",
        note="Admits exactly 11-byte members past the size bound; every other oversized member still refuses.",
    ),
    Mutant(
        name="N-descriptor-empty",
        path=f"{PACKAGE}/descriptor.go",
        find="\tchunks := make([]any, 0, 1)\n\tfor offset := uint64(0); offset < size; {\n",
        replace=(
            "\tchunks := make([]any, 0, 1)\n"
            "\tif size == 0 {\n"
            '\t\tchunks = append(chunks, map[string]any{"index": 0, "offset": 0, "size": 0, "chunk_id": blobID.String()})\n'
            "\t}\n"
            "\tfor offset := uint64(0); offset < size; {\n"
        ),
        count=1,
        run="TestCaptureEmptyMemberSealsEmptyBlob",
        expect="KILLED",
        note="Seals exactly empty payloads with a zero-size chunk; every non-empty payload still seals clean chunks.",
    ),
    Mutant(
        name="N-descriptor-offset",
        path=f"{PACKAGE}/descriptor.go",
        find=(
            "\t\tchunks = append(chunks, map[string]any{\n"
            '\t\t\t"index":    len(chunks),\n'
            '\t\t\t"offset":   offset,\n'
        ),
        replace=(
            "\t\tshifted := offset\n"
            "\t\tif len(chunks) == 1 {\n"
            "\t\t\tshifted = offset + 1\n"
            "\t\t}\n"
            "\t\tchunks = append(chunks, map[string]any{\n"
            '\t\t\t"index":    len(chunks),\n'
            '\t\t\t"offset":   shifted,\n'
        ),
        count=1,
        run="TestCaptureMultiChunkMember",
        expect="KILLED",
        note="Shifts exactly the second chunk off contiguity; single-chunk descriptors still seal.",
    ),
    Mutant(
        name="N-workspace-cwd",
        path=f"{PACKAGE}/workspace.go",
        find='\t\tif relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {\n',
        replace='\t\tif (relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator))) && cwd != "/outside-workspace" {\n',
        count=1,
        run="TestCheckpointWorkspaceRefusesEscape",
        expect="KILLED",
        note="P3-δ (defence-in-depth, message-keyed): admits exactly /outside-workspace past the line-120 escape check while the landed relative-path grammar still refuses it; the killer reddens on the missing 'escapes the workspace root' literal, not on admission.",
    ),
    Mutant(
        name="N-workspace-fingerprints",
        path=f"{PACKAGE}/workspace.go",
        find="\t\tif index > 0 && fingerprint <= previous {\n",
        replace="\t\tif index > 0 && fingerprint < previous {\n",
        count=1,
        run="TestCheckpointWorkspaceRefusesDuplicateFingerprint",
        expect="KILLED",
        note="Admits exactly duplicate fingerprints; unsorted fingerprints still refuse with the checkpoint detail.",
    ),
    Mutant(
        name="N-log-leak",
        path=f"{PACKAGE}/capture.go",
        find='\t\tstate.log.Linef("capture member=%q class=%s digest=%s size=%d media=%s", key, item.Class, descriptor.BlobID.String(), descriptor.Size, blobMediaType)\n',
        replace=(
            '\t\tstate.log.Linef("capture member=%q class=%s digest=%s size=%d media=%s", key, item.Class, descriptor.BlobID.String(), descriptor.Size, blobMediaType)\n'
            '\t\tif key == "store/blob-a" {\n'
            '\t\t\tstate.log.Linef("capture preview=%q", string(payload))\n'
            '\t\t}\n'
        ),
        count=1,
        run="TestCaptureLogRecordsDigestsOnly",
        expect="KILLED",
        note="Leaks exactly store/blob-a payload bytes into the log; every other member still logs digests only.",
    ),
    Mutant(
        name="N-hook-crash",
        path=f"{PACKAGE}/capture.go",
        find="\tif request.Hooks.AfterRawPublish != nil {\n",
        replace='\tif request.Hooks.AfterRawPublish != nil && request.Generation != "crash-generation-7" {\n',
        count=1,
        run="TestCaptureCrashChildSelfTerminates",
        expect="KILLED",
        note="Skips the crash-boundary hook for exactly the crash generation; the child exits clean and the parent misses its SIGKILL.",
    ),
    Mutant(
        name="N-payload-install-blob-b",
        path=f"{PACKAGE}/capture.go",
        find=(
            '\t\tif _, err := clonebundle.InstallRawBlob(state.request.Store, descriptor.Descriptor, bytes.NewReader(payload)); err != nil {\n'
            '\t\t\treturn invalid("capture member %q: %v", key, err)\n'
            '\t\t}\n'
        ),
        replace=(
            '\t\tif key != "store/blob-b" {\n'
            '\t\t\tif _, err := clonebundle.InstallRawBlob(state.request.Store, descriptor.Descriptor, bytes.NewReader(payload)); err != nil {\n'
            '\t\t\t\treturn invalid("capture member %q: %v", key, err)\n'
            '\t\t\t}\n'
            '\t\t}\n'
        ),
        count=1,
        run="TestCaptureInstallsEveryPayloadBlob",
        expect="KILLED",
        note="Seals exactly store/blob-b into the raw manifest without installing its blob; every other member still installs. The killer scans the store for every referenced blob.",
    ),
    Mutant(
        name="N-archive-emits-receipt",
        path=f"{PACKAGE}/project.go",
        find="\t\t// Archive-only output never enters a target branch: the\n",
        replace=(
            "\t\t_, _ = request.TargetSink.PutBlob(scalar.SHA256Digest(captured.CaptureManifest), uint64(len(captured.CaptureManifest)), bytes.NewReader(captured.CaptureManifest))\n"
            "\t\t// Archive-only output never enters a target branch: the\n"
        ),
        count=1,
        run="TestProjectRefusesUnstable",
        expect="KILLED",
        note="P3 row 8 (add-write, not a narrowing): writes the archive capture manifest into the TARGET sink before the G2 refusal; the killer asserts the target sink is empty, not only that admission refused.",
    ),
    Mutant(
        name="N-optional-blocked-ancestor",
        path=f"{PACKAGE}/capture.go",
        find="\t\tif !present && !item.Required && !blockedAncestor(byKey, key) {\n",
        replace="\t\tif !present && !item.Required {\n",
        count=1,
        run="TestCaptureRefusesOptionalSymlinkedIntermediate",
        expect="KILLED",
        note="Seals an optional member behind a symlinked intermediate as plan_optional_absent instead of routing it to the Guard refusal; required members still route to the Guard.",
    ),
    Mutant(
        name="N-post-fallback-captured",
        path=f"{PACKAGE}/capture.go",
        find=(
            "\t\t\tif !useCaptured {\n"
            "\t\t\t\tif payload, err := openMember(state.guard, state.root, member.key, state.limit); err == nil {\n"
            "\t\t\t\t\trecord.addFile(member.key, uint64(len(payload)), contentHash(payload))\n"
            "\t\t\t\t\tcontinue\n"
            "\t\t\t\t}\n"
            "\t\t\t}\n"
        ),
        replace=(
            "\t\t\tif !useCaptured {\n"
            "\t\t\t\tif payload, err := openMember(state.guard, state.root, member.key, state.limit); err == nil {\n"
            "\t\t\t\t\trecord.addFile(member.key, uint64(len(payload)), contentHash(payload))\n"
            "\t\t\t\t\tcontinue\n"
            "\t\t\t\t}\n"
            "\t\t\t\tfallback := false\n"
            "\t\t\t\tfor _, captured := range state.captured {\n"
            "\t\t\t\t\tif captured.key == member.key {\n"
            "\t\t\t\t\t\trecord.addFile(member.key, captured.descriptor.Size, contentHash(captured.payload))\n"
            "\t\t\t\t\t\tfallback = true\n"
            "\t\t\t\t\t}\n"
            "\t\t\t\t}\n"
            "\t\t\t\tif fallback {\n"
            "\t\t\t\t\tcontinue\n"
            "\t\t\t\t}\n"
            "\t\t\t}\n"
        ),
        count=1,
        run="TestCaptureRaceUnreadableAtPost",
        expect="KILLED",
        note="A member unreadable at post time falls back to its captured hash instead of unknown content, so exactly the unreadable-at-post class misses the race; every re-readable member still compares fresh bytes.",
    ),
    Mutant(
        name="N-clonebundle-unknown-excluded",
        path="internal/clonebundle/capturebuild.go",
        find=(
            "\tfor _, item := range items {\n"
            "\t\tif item.Class == \"unknown\" {\n"
            "\t\t\treturn true\n"
            "\t\t}\n"
            "\t}\n"
            "\treturn false\n"
            "}\n"
        ),
        replace=(
            "\tfor _, item := range items {\n"
            "\t\tif item.Class == \"unknown\" && item.Included() {\n"
            "\t\t\treturn true\n"
            "\t\t}\n"
            "\t}\n"
            "\treturn false\n"
            "}\n"
        ),
        count=1,
        run="TestCaptureOptionalAbsentUnknownMakesRawIncomplete",
        expect="KILLED",
        note="P3-ε: the plant lives in clonebundle (hasUnknownClass ignores excluded items) while the killer lives here — an excluded unknown item must still drive raw_complete=false and block maximal_safe at this leaf's projection entry.",
    ),
    Mutant(
        name="N-excluded-orphan-blob",
        path=f"{PACKAGE}/capture.go",
        find=(
            "\t\tif excludedClass(item.Class) {\n"
            "\t\t\tstate.log.Linef(\"exclude member=%q class=%s reason=%s\", key, item.Class, exclusionReason(item.Class))\n"
            "\t\t\tcontinue\n"
            "\t\t}\n"
        ),
        replace=(
            "\t\tif excludedClass(item.Class) {\n"
            "\t\t\tstate.log.Linef(\"exclude member=%q class=%s reason=%s\", key, item.Class, exclusionReason(item.Class))\n"
            "\t\t\tif leaked, leakErr := openMember(state.guard, state.root, key, state.limit); leakErr == nil {\n"
            "\t\t\t\tif leakDescriptor, descErr := BuildBlobDescriptor(leaked); descErr == nil {\n"
            "\t\t\t\t\t_, _ = clonebundle.InstallRawBlob(state.request.Store, leakDescriptor.Descriptor, bytes.NewReader(leaked))\n"
            "\t\t\t\t}\n"
            "\t\t\t}\n"
            "\t\t\tcontinue\n"
            "\t\t}\n"
        ),
        count=1,
        run="TestCaptureExcludesSecrets",
        expect="KILLED",
        note="P3-ε′ (add-write, not a narrowing): installs excluded bytes as an orphan blob with no manifest entry and no log line, so the killer's blob scan — not a builder refusal — must catch the leak.",
    ),
    Mutant(
        name="N-unstable-seals-pre-as-post",
        path=f"{PACKAGE}/capture.go",
        find=(
            "\t\t\tPreCaptureDigest:  state.pre.String(),\n"
            "\t\t\tPostCaptureDigest: state.post.String(),\n"
            "\t\t\tCore:              true,\n"
        ),
        replace=(
            "\t\t\tPreCaptureDigest:  state.pre.String(),\n"
            "\t\t\tPostCaptureDigest: state.pre.String(),\n"
            "\t\t\tCore:              true,\n"
        ),
        count=1,
        run="TestCaptureArchiveSealsUnstable",
        expect="KILLED",
        note="The unstable_archive form seals the PRE digest for post_capture_digest, so the archive manifest carries equal digests while flagged source_not_quiescent; the killer asserts the sealed digests equal the measured pre/post and the sealed generation equals the request generation.",
    ),
    Mutant(
        name="N-unplanned-special",
        path=f"{PACKAGE}/walk.go",
        find="func intermediatePrefix(plan map[string]PlanItem, key string) bool {\n\tprefix := key + \"/\"\n",
        replace="func intermediatePrefix(plan map[string]PlanItem, key string) bool {\n\tif key == \"stray-link\" {\n\t\treturn true\n\t}\n\tprefix := key + \"/\"\n",
        count=1,
        run="TestCaptureRefusesUnplannedSpecialMember",
        expect="KILLED",
        note="Treats exactly the unplanned symlink stray-link as an intermediate, so it bypasses the 'not a plan candidate' refusal (and the early containment open) and the capture seals stable; every other unplanned key still refuses.",
    ),
    Mutant(
        name="N-store-root-symlink",
        path=f"{PACKAGE}/capture.go",
        find="\troot, err := secprim.OpenNoFollowDir(request.StoreRoot)\n",
        replace=(
            "\troot, err := secprim.OpenNoFollowDir(request.StoreRoot)\n"
            '\tif err != nil && len(request.StoreRoot) >= 9 && request.StoreRoot[len(request.StoreRoot)-9:] == "link-root" {\n'
            "\t\troot, err = os.Open(request.StoreRoot)\n"
            "\t}\n"
        ),
        count=1,
        run="TestCaptureRefusesSymlinkedStoreRoot",
        expect="KILLED",
        note="Follows a trailing symlink at the store root for exactly link-root (every other symlinked root still refuses through the landed nofollow discipline); the killer asserts the refusal names the link and installs nothing.",
    ),
    Mutant(
        name="N-excluded-size",
        path=f"{PACKAGE}/capture.go",
        find=(
            '\t\t\tif class == "" || excludedClass(class) {\n'
            '\t\t\t\trecord.addFile(member.key, member.size, "-")\n'
            "\t\t\t\tcontinue\n"
            "\t\t\t}\n"
        ),
        replace=(
            '\t\t\tif class == "" || excludedClass(class) {\n'
            "\t\t\t\tsize := member.size\n"
            '\t\t\t\tif member.key == "store/token-cache" {\n'
            "\t\t\t\t\tsize = 0\n"
            "\t\t\t\t}\n"
            '\t\t\t\trecord.addFile(member.key, size, "-")\n'
            "\t\t\t\tcontinue\n"
            "\t\t\t}\n"
        ),
        count=1,
        run="TestCaptureRaceExcludedSizeChange",
        expect="KILLED",
        note="Exactly the excluded member store/token-cache contributes a constant size to the source record, so its size change between walk and post is missed; every other member still races on size.",
    ),
    Mutant(
        name="N-plan-grammar-dotdot",
        path=f"{PACKAGE}/walk.go",
        find="\t\tif err := secprim.CheckMemberPath(platform, item.NativeKey); err != nil {\n",
        replace="\t\tif err := secprim.CheckMemberPath(platform, item.NativeKey); err != nil && item.NativeKey != \"store/../escape\" {\n",
        count=1,
        run="TestCaptureRefusesParentPlanKey",
        expect="KILLED",
        note="Admits exactly store/../escape past the plan member-grammar gate (SanitizeNativeKey does not refuse parent segments, so this gate is the only one); every other escaping key still refuses.",
    ),
    Mutant(
        name="N-required-blocked-ancestor",
        path=f"{PACKAGE}/capture.go",
        find="\t\tif !present && item.Required && !blockedAncestor(byKey, key) {\n",
        replace="\t\tif !present && item.Required {\n",
        count=1,
        run="TestCaptureRefusesSymlinkedIntermediate",
        expect="KILLED",
        note="A required member behind a symlinked intermediate refuses as 'required but absent' instead of reaching the Guard refusal that names the symlink escape; optional members still route to the Guard.",
    ),
    Mutant(
        name="N-fingerprints-129",
        path=f"{PACKAGE}/workspace.go",
        find="\tif len(fingerprints) > 128 {\n",
        replace="\tif len(fingerprints) > 129 {\n",
        count=1,
        run="TestCheckpointWorkspaceRefusesTooManyFingerprints",
        expect="KILLED",
        note="Admits exactly 129 repository remote fingerprints past the [0..128] bound; the killer asserts the 'maximum is 128' refusal.",
    ),
    Mutant(
        name="N-first-chunk-oversize",
        path=f"{PACKAGE}/descriptor.go",
        find="\t\tlength := uint64(blobChunkSize)\n",
        replace=(
            "\t\tlength := uint64(blobChunkSize)\n"
            "\t\tif len(chunks) == 0 && size > uint64(blobChunkSize) {\n"
            "\t\t\tlength = uint64(blobChunkSize) + 1\n"
            "\t\t}\n"
        ),
        count=1,
        run="TestCaptureMultiChunkMember",
        expect="KILLED",
        note="Seals the first chunk of a multi-chunk blob at 4194305 bytes (chunk size bound uint53[1..4194304]); single-chunk descriptors still seal.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/doc.go",
        find="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections\n",
        replace="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections (control).\n",
        count=1,
        run="TestDefaultMaxSingleBytesIs128GiB",
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
