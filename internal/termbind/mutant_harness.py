#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/termbind.

Every gate ships a narrowing mutant: the gate stays present and is
weakened to admit exactly one member of the class it must reject, and
the named killer tests must fail. A delete-only mutant proves only that
the gate exists and is not accepted as evidence; row D-emit-noresolve is
the one explicitly supplementary arm-delete (the producer-named drop),
labeled as such and not counted as any gate's narrowing.

Each row runs M (mutate production files in place), R (run the scoped Go
killers through the production entries), V (verdict from the Go exit code
plus a FAIL kill-line: KILLED iff R fails with a test failure line,
SURVIVED iff R passes, ERROR for anything else -- unapplied patches,
build breaks, timeouts). M failures and R infrastructure errors are
ERROR, never kills or survivals. Files are restored from backups after
every row, pass or fail.

Row C-control is the harmless control: a comment-only edit that must
SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Two plants are honestly not rows: attach cannot emit an event because
Attach takes no repository (absence by construction, proven by
TestAttachEmitsNeitherEventNorStateChange -- no text plant can add a
write), and the lease-authority narrowing belongs to the adopted axpane
rows N-emit-mutation and N-emit-losing-epoch (composition pinned here by
TestEmitUnderLosingLeaseRefuses).

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/termbind/mutant_harness.py
  PYTHONDONTWRITEBYTECODE=1 python3 internal/termbind/mutant_harness.py N-identity-pid
  AX_MUTANT_VERBOSE=1 PYTHONDONTWRITEBYTECODE=1 python3 internal/termbind/mutant_harness.py N-identity-pid
"""

import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
PACKAGE = "internal/termbind"

DD_DIGEST = "sha256:" + "dd" * 32
OTHER_OP = "0198f4c8-7d40-7e55-8e6f-1234567890ac"


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
    find2: str = ""
    replace2: str = ""
    count2: int = 0


IDENTITY_KILLERS = (
    "(TestCheckInstanceIdentityRefusesForbiddenForms"
    "|TestParseTerminalBindingRefusesForbiddenIdentity"
    "|TestAdmitMutationContextRefusesForbiddenIdentity"
    "|TestAdmitStatusBodyRefusesForbiddenIdentity"
    "|TestAttachRefusesForbiddenIdentity)"
)

MUTANTS = [
    Mutant(
        name="N-identity-pid",
        path=f"{PACKAGE}/identity.go",
        find='\tif _, err := scalar.ParseUUIDv7(value); err != nil {\n\t\treturn &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: "terminal instance identity"}\n\t}\n',
        replace='\tif _, err := scalar.ParseUUIDv7(value); err != nil && value != "12345" {\n\t\treturn &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: "terminal instance identity"}\n\t}\n',
        count=1,
        run=IDENTITY_KILLERS,
        expect="KILLED",
        note="Admits exactly the PID form at the shared identity gate; every other forbidden form still refuses on all five surfaces.",
    ),
    Mutant(
        name="N-identity-endpoint",
        path=f"{PACKAGE}/identity.go",
        find='\tif _, err := scalar.ParseUUIDv7(value); err != nil {\n\t\treturn &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: "terminal instance identity"}\n\t}\n',
        replace='\tif _, err := scalar.ParseUUIDv7(value); err != nil && value != "10.0.0.9:2222" {\n\t\treturn &terminalbackend.Error{Code: terminalbackend.CodeProtocolError, Detail: "terminal instance identity"}\n\t}\n',
        count=1,
        run=IDENTITY_KILLERS,
        expect="KILLED",
        note="Admits exactly the endpoint form at the shared identity gate; every other forbidden form still refuses on all five surfaces.",
    ),
    Mutant(
        name="N-binding-members",
        path=f"{PACKAGE}/identity.go",
        find="\tif len(members) != len(bindingMembers) {\n",
        replace="\tif len(members) != len(bindingMembers) && len(members) != len(bindingMembers)+1 {\n",
        count=1,
        run="TestParseTerminalBindingClosedShape/extra_member",
        expect="KILLED",
        note="Admits exactly one extra binding member; missing members and two extras still refuse.",
    ),
    Mutant(
        name="N-binding-identity",
        path=f"{PACKAGE}/identity.go",
        find="\tif recomputed != bindingID {\n",
        replace=f'\tif recomputed != bindingID && bindingID != "{DD_DIGEST}" {{\n',
        count=1,
        run="TestParseTerminalBindingClosedShape/wrong_self_id",
        expect="KILLED",
        note="Admits exactly the wrong self-ID fixture; other identity breaks still refuse.",
    ),
    Mutant(
        name="N-binding-generation",
        path=f"{PACKAGE}/identity.go",
        find="\tif _, err := terminalbackend.GenerationDigest(generation); err != nil {\n",
        replace="\tif _, err := terminalbackend.GenerationDigest(generation); err != nil && len(generation) != 257 {\n",
        count=1,
        run="TestParseTerminalBindingClosedShape/generation_bounds",
        expect="KILLED",
        note="Admits exactly the 257-character generation; length 0 and 258+ still refuse.",
    ),
    Mutant(
        name="N-binding-protocol-major",
        path=f"{PACKAGE}/identity.go",
        find='\tif !environ.CheckSemver(protocol) || !isProtocolMajorOne(protocol) {\n\t\treturn empty, protocolRefusal("binding protocol version")\n\t}\n',
        replace='\tif !environ.CheckSemver(protocol) || (!isProtocolMajorOne(protocol) && protocol != "2.0.0") {\n\t\treturn empty, protocolRefusal("binding protocol version")\n\t}\n',
        count=1,
        run="TestParseTerminalBindingMemberArms/protocol_version_major_2",
        expect="KILLED",
        note="Admits exactly protocol 2.0.0 past the major-1 arm; other foreign majors and non-semver still refuse.",
    ),
    Mutant(
        name="N-binding-extensions",
        path=f"{PACKAGE}/identity.go",
        find='\tif !isEmptyExtensionsObject(members["extensions"]) {\n\t\treturn empty, protocolRefusal("binding extensions")\n\t}\n',
        replace='\tif !isEmptyExtensionsObject(members["extensions"]) && string(members["extensions"]) != "{\\"com.example.note\\":\\"y\\"}" {\n\t\treturn empty, protocolRefusal("binding extensions")\n\t}\n',
        count=1,
        run="TestParseTerminalBindingMemberArms/extensions_non-empty",
        expect="KILLED",
        note="Admits exactly the one-key note extensions object; every other non-empty extensions shape still refuses.",
    ),
    Mutant(
        name="N-binding-identity-number",
        path=f"{PACKAGE}/identity.go",
        find='\tcase json.Number, float64:\n\t\treturn protocolRefusal("binding identity")\n',
        replace='\tcase json.Number:\n\t\tif typed != "1.0" {\n\t\t\treturn protocolRefusal("binding identity")\n\t\t}\n\tcase float64:\n\t\treturn protocolRefusal("binding identity")\n',
        count=1,
        run="TestBindingIdentityRefusesHostileBytes/float",
        expect="KILLED",
        note="Admits exactly the 1.0 number past the pre-transform walk; every other number still refuses.",
    ),
    Mutant(
        name="N-binding-nested-dup",
        path=f"{PACKAGE}/identity.go",
        find='\t\t\t\tif _, duplicate := object[key]; duplicate {\n\t\t\t\t\treturn nil, protocolRefusal("binding identity")\n\t\t\t\t}\n',
        replace='\t\t\t\tif _, duplicate := object[key]; duplicate && depth == 0 {\n\t\t\t\t\treturn nil, protocolRefusal("binding identity")\n\t\t\t\t}\n',
        count=1,
        run="TestBindingIdentityRefusesHostileBytes/nested_duplicate",
        expect="KILLED",
        note="Admits nested duplicate members past the identity decode; top-level duplicates still refuse.",
    ),
    Mutant(
        name="N-binding-supersedes",
        path=f"{PACKAGE}/identity.go",
        find='\t\tif _, err := scalar.ParseDigest(supersedes); err != nil {\n\t\t\treturn empty, protocolRefusal("binding supersedes")\n\t\t}\n',
        replace='\t\tif _, err := scalar.ParseDigest(supersedes); err != nil && supersedes != "not-a-digest" {\n\t\t\treturn empty, protocolRefusal("binding supersedes")\n\t\t}\n',
        count=1,
        run="TestParseTerminalBindingMemberArms/supersedes_non-digest",
        expect="KILLED",
        note="Admits exactly the not-a-digest supersedes value; other non-digests still refuse.",
    ),
    Mutant(
        name="N-prescan-context",
        path=f"{PACKAGE}/identity.go",
        find="\treturn CheckInstanceIdentity(value)\n",
        replace='\tif value == "12345" {\n\t\treturn nil\n\t}\n\treturn CheckInstanceIdentity(value)\n',
        count=1,
        run="TestAdmitMutationContextRefusesForbiddenIdentity/pid",
        expect="KILLED",
        note="Skips the context pre-scan exactly for the PID form; the landed parser still refuses with the pinned code, so the kill measures the Go error type (*terminalbackend.Error) the documented-redundant pre-scan adds.",
    ),
    Mutant(
        name="N-status-prescan",
        path=f"{PACKAGE}/identity.go",
        find="\tif err := CheckInstanceIdentity(value); err != nil {\n\t\treturn terminstance.StatusBody{}, err\n\t}\n",
        replace='\tif err := CheckInstanceIdentity(value); err != nil && value != "12345" {\n\t\treturn terminstance.StatusBody{}, err\n\t}\n',
        count=1,
        run="TestAdmitStatusBodyRefusesForbiddenIdentity/pid",
        expect="KILLED",
        note="Skips the status pre-scan exactly for the PID form; the landed parser still refuses with the pinned code, so the kill measures the Go error type (*terminalbackend.Error) the documented-redundant pre-scan adds.",
    ),
    Mutant(
        name="N-evidence-257",
        path=f"{PACKAGE}/resolve.go",
        find="\tif len(ids) < minEvidenceIDs || len(ids) > maxEvidenceIDs {\n",
        replace="\tif len(ids) < minEvidenceIDs || len(ids) > maxEvidenceIDs+1 {\n",
        count=1,
        run="TestResolveEvidenceShapeBounds/257_refuses",
        expect="KILLED",
        note="Admits exactly 257 evidence IDs past the shape arm; 258 still refuses.",
    ),
    Mutant(
        name="N-evidence-0",
        path=f"{PACKAGE}/resolve.go",
        find="\tif len(ids) < minEvidenceIDs || len(ids) > maxEvidenceIDs {\n",
        replace="\tif len(ids) < minEvidenceIDs-1 || len(ids) > maxEvidenceIDs {\n",
        count=1,
        run="TestResolveEvidenceShapeBounds/empty_refuses",
        expect="KILLED",
        note="Admits exactly the empty evidence set past the shape arm; the partition refusal surfaces instead of the bound.",
    ),
    Mutant(
        name="N-evidence-dup",
        path=f"{PACKAGE}/resolve.go",
        find="\t\tif index > 0 && id <= previous {\n",
        replace="\t\tif index > 0 && id < previous {\n",
        count=1,
        run="TestResolveEvidenceShapeBounds/duplicate_refuses",
        expect="KILLED",
        note="Admits exactly duplicate evidence IDs; inversions still refuse.",
    ),
    Mutant(
        name="N-evidence-unsorted",
        path=f"{PACKAGE}/resolve.go",
        find="\t\tif index > 0 && id <= previous {\n",
        replace="\t\tif index > 1 && id <= previous {\n",
        count=1,
        run="TestResolveEvidenceShapeBounds/unsorted_refuses",
        expect="KILLED",
        note="Skips exactly the first-pair order comparison; later inversions still refuse.",
    ),
    Mutant(
        name="N-resolve-kind",
        path=f"{PACKAGE}/resolve.go",
        find='\tdefault:\n\t\treturn "", mismatchRefusal("evidence kind")\n',
        replace='\tdefault:\n\t\tif schema == "ax-terminal-endpoint" {\n\t\t\treturn terminalbackend.SchemaCapabilityEvidence, nil\n\t\t}\n\t\treturn "", mismatchRefusal("evidence kind")\n',
        count=1,
        run="(TestResolveEvidenceRefusesForeignKinds|TestEmitResolvesEvidence)/(endpoint|foreign_kind)",
        expect="KILLED",
        note="Admits exactly the endpoint document as evidence; every other foreign kind still refuses, at the direct and emission entries.",
    ),
    Mutant(
        name="N-resolve-manifest-count",
        path=f"{PACKAGE}/resolve.go",
        find='\t\tcase terminalbackend.SchemaManifest:\n\t\t\tif manifestFound {\n\t\t\t\treturn empty, mismatchRefusal("evidence manifest")\n\t\t\t}\n',
        replace='\t\tcase terminalbackend.SchemaManifest:\n\t\t\tif manifestFound && len(ids) < 4 {\n\t\t\t\treturn empty, mismatchRefusal("evidence manifest")\n\t\t\t}\n',
        count=1,
        run="TestResolveEvidenceRequiresManifestAndProbe/second_manifest",
        expect="KILLED",
        note="Admits exactly the four-ID double-manifest set; other ambiguous sets still refuse.",
    ),
    Mutant(
        name="N-resolve-binding",
        path=f"{PACKAGE}/resolve.go",
        find="\t\tif object.TerminalBackendID != tuple.BackendID || object.ImplementationVersion != tuple.ImplementationVersion || object.ProtocolVersion != tuple.ProtocolVersion {\n",
        replace="\t\tif object.TerminalBackendID != tuple.BackendID || object.ImplementationVersion != tuple.ImplementationVersion {\n",
        count=1,
        run="TestResolveEvidenceBindsEventTuple/protocol_drift_evidence",
        expect="KILLED",
        note="Drops exactly the evidence protocol arm; backend and implementation drift still refuse at this gate.",
    ),
    Mutant(
        name="N-resolve-identity",
        path=f"{PACKAGE}/resolve.go",
        find="\tif !named[manifest.ManifestID] || !named[probe.ProbeID] {\n",
        replace="\tif !named[manifest.ManifestID] {\n",
        count=1,
        run="TestResolveEvidenceRejectsSubstitution/probe_alias",
        expect="KILLED",
        note="Drops exactly the probe half of the substitution check; manifest and evidence aliases still refuse.",
    ),
    Mutant(
        name="N-resolve-admission",
        path=f"{PACKAGE}/resolve.go",
        find="\tadmitted, err := registry.AdmitProbe(manifestRaw, probeRaw, evidenceRaws, rawGeneration, now, verify)\n\tif err != nil {\n\t\treturn empty, err\n\t}\n",
        replace="\tadmitted, err := registry.AdmitProbe(manifestRaw, probeRaw, evidenceRaws, rawGeneration, now, verify)\n\tif err != nil && !terminalbackend.IsDrift(err) {\n\t\treturn empty, err\n\t}\n",
        count=1,
        run="TestResolveEvidenceDrivesLandedAdmission/registry_drift",
        expect="KILLED",
        note="Swallows exactly the landed drift refusal; every other admission refusal still surfaces.",
    ),
    Mutant(
        name="N-emit-created-v3",
        path=f"{PACKAGE}/emit.go",
        find="\tparams.EventType = eventTerminalCreated\n\tparams.SchemaVersion = eventSchemaV4\n",
        replace='\tparams.EventType = eventTerminalCreated\n\tparams.SchemaVersion = "3.0.0"\n',
        count=1,
        run="TestEmitTerminalCreatedRoundTrip",
        expect="KILLED",
        note="Emits the created payload at v3; the canonical owner refuses the v4 members under the v3 shape.",
    ),
    Mutant(
        name="N-emit-resumed-v3",
        path=f"{PACKAGE}/emit.go",
        find="\tparams.EventType = eventSessionResumed\n\tparams.SchemaVersion = eventSchemaV4\n",
        replace='\tparams.EventType = eventSessionResumed\n\tparams.SchemaVersion = "3.0.0"\n',
        count=1,
        run="TestEmitResumedRoundTrip",
        expect="KILLED",
        note="Emits the resumed payload at v3; the canonical owner refuses the v4 members under the v3 shape.",
    ),
    Mutant(
        name="D-emit-noresolve",
        path=f"{PACKAGE}/emit.go",
        find="\tresolved, err := ResolveEvidence(facts.EvidenceIDs, tupleFromFacts(facts), admission.Registry, admission.Universe, admission.RawGeneration, admission.Now, admission.Verify)\n\tif err != nil {\n\t\treturn empty, nil, ResolvedEvidence{}, err\n\t}\n",
        replace="\tresolved := ResolvedEvidence{}\n",
        count=2,
        run="TestEmitResolvesEvidence",
        expect="KILLED",
        note="ARM-DELETE, supplementary: drops resolution at both emission entries; unresolvable evidence proceeds to append. Not counted as any gate's narrowing.",
    ),
    Mutant(
        name="N-recover-mismatch",
        path=f"{PACKAGE}/recover.go",
        find="\t\tif anchored && anchor.OperationID != request.BootstrapOperationID && request.WindowOpen {\n",
        replace=f'\t\tif anchored && anchor.OperationID != request.BootstrapOperationID && request.WindowOpen && request.BootstrapOperationID != "{OTHER_OP}" {{\n',
        count=1,
        run="TestRecoverCreateChangedOperationInWindowRefuses",
        expect="KILLED",
        note="Admits exactly the fixture changed operation in-window; other changed operations still refuse.",
    ),
    Mutant(
        name="N-recover-second-child",
        path=f"{PACKAGE}/recover.go",
        find="\tif !found {\n\t\tif report.IdentityMatch && report.State == terminalbackend.StateAbsent {\n\t\t\treturn RecoveryVerdict{Outcome: RecoveryAbsent, State: report.State, Retry: terminstance.DispositionReplaySame}, nil\n\t\t}\n\t\treturn RecoveryVerdict{Outcome: RecoveryUnavailable, State: report.State, Retry: terminstance.DispositionStatusFirst}, nil\n\t}\n",
        replace="\tif !found {\n\t\tif report.IdentityMatch && report.State == terminalbackend.StateAbsent {\n\t\t\treturn RecoveryVerdict{Outcome: RecoveryAbsent, State: report.State, Retry: terminstance.DispositionReplaySame}, nil\n\t\t}\n\t\treturn RecoveryVerdict{Outcome: RecoveryChild, State: report.State, Retry: terminstance.DispositionReplaySame}, nil\n\t}\n",
        count=1,
        run="TestRecoverCreateNeverClaimsAbsent/unbound_live_instance",
        expect="KILLED",
        note="Adopts exactly the unbound live instance as a second child; proven absence still vacates.",
    ),
    Mutant(
        name="N-recover-contradiction-absent",
        path=f"{PACKAGE}/recover.go",
        find="\tif !report.IdentityMatch {\n\t\treturn RecoveryVerdict{Outcome: RecoveryUnavailable, State: report.State, Retry: terminstance.DispositionStatusFirst}, nil\n\t}\n",
        replace="\tif !report.IdentityMatch {\n\t\treturn RecoveryVerdict{Outcome: RecoveryAbsent, State: report.State, Retry: terminstance.DispositionReplaySame}, nil\n\t}\n",
        count=1,
        run="TestRecoverCreateNeverClaimsAbsent/bound_contradiction",
        expect="KILLED",
        note="Claims exactly the bound contradiction as absence; matched observations keep their verdicts.",
    ),
    Mutant(
        name="N-recover-creating-child",
        path=f"{PACKAGE}/recover.go",
        find="\tcase terminalbackend.StateCreating, terminalbackend.StateUnavailable:\n",
        replace="\tcase terminalbackend.StateUnavailable:\n",
        count=1,
        run="TestRecoverCreateNeverClaimsAbsent/creating_interim",
        expect="KILLED",
        note="Admits exactly the creating interim as a child; backend-reported unavailable still parks.",
    ),
    Mutant(
        name="N-recover-status-error",
        path=f"{PACKAGE}/recover.go",
        find='\treport, err := deps.Engine.ExecuteStatus(ctx, body, "", deps.Admitted)\n\tif err != nil {\n\t\treturn empty, err\n\t}\n',
        replace='\treport, err := deps.Engine.ExecuteStatus(ctx, body, "", deps.Admitted)\n\tif err != nil {\n\t\treturn RecoveryVerdict{Outcome: RecoveryAbsent, State: terminalbackend.StateAbsent, Retry: terminstance.DispositionReplaySame}, nil\n\t}\n',
        count=1,
        run="TestRecoverCreateUnknownIsNotAbsent/status_read_failure",
        expect="KILLED",
        note="Launders exactly the status read failure as proven absence; reported observations keep their verdicts.",
    ),
    Mutant(
        name="N-recover-window",
        path=f"{PACKAGE}/recover.go",
        find="\t\tif anchored && anchor.OperationID != request.BootstrapOperationID && request.WindowOpen {\n",
        replace="\t\tif anchored && anchor.OperationID != request.BootstrapOperationID && !request.WindowOpen {\n",
        count=1,
        run="TestRecoverCreateChangedOperationInWindowRefuses",
        expect="KILLED",
        note="Inverts exactly the window bit; post-window changed pairs refuse while in-window proceeds.",
    ),
    Mutant(
        name="N-recover-generation",
        path=f"{PACKAGE}/recover.go",
        find='\tif _, err := terminalbackend.GenerationDigest(request.Generation); err != nil {\n\t\treturn empty, err\n\t}\n',
        replace='\tif _, err := terminalbackend.GenerationDigest(request.Generation); err != nil && len(request.Generation) != 257 {\n\t\treturn empty, err\n\t}\n',
        count=1,
        run="TestRecoverGenerationBoundBeforeStatusRead",
        expect="KILLED",
        note="Skips the generation bound exactly for a 257-character generation before the status read; the read runs and the refusal never fires.",
    ),
    Mutant(
        name="N-attach-conflict",
        path=f"{PACKAGE}/attach.go",
        find="\t\tif stored.Transport != transport || stored.InputAuthorized != inputAuthorized {\n",
        replace="\t\tif stored.Transport != transport {\n",
        count=1,
        run="TestAttachConflictRefusesMismatch",
        expect="KILLED",
        note="Drops exactly the input half of the conflict conjunction at both the pre-check and the commit-converge comparison; transport conflicts still refuse.",
        find2="\t\t\tif winner.Transport != transport || winner.InputAuthorized != inputAuthorized {\n",
        replace2="\t\t\tif winner.Transport != transport {\n",
        count2=1,
    ),
    Mutant(
        name="N-attach-transport",
        path=f"{PACKAGE}/attach.go",
        find="\tif !isAttachTransport(transport) {\n",
        replace='\tif !isAttachTransport(transport) && transport != "smoke_signal" {\n',
        count=1,
        run="TestAttachTransportVocabulary/unknown_transport",
        expect="KILLED",
        note="Admits exactly the smoke_signal transport past the enum; the landed vocabulary refusal surfaces instead of this gate's.",
    ),
    Mutant(
        name="N-attach-auth-replay",
        path=f"{PACKAGE}/attach.go",
        find="\tauth, err := terminalbackend.ParseAttachAuthorization(authRaw)\n\tif err != nil {\n\t\treturn empty, false, err\n\t}\n\tif err := terminalbackend.CheckAttachRequest(auth, transport, inputAuthorized, now); err != nil {\n\t\treturn empty, false, err\n\t}\n\tstored, found, err := store.Lookup(sessionID, instanceID, clientID)\n\tif err != nil {\n\t\treturn empty, false, err\n\t}\n\tif found {\n",
        replace="\tstored, found, err := store.Lookup(sessionID, instanceID, clientID)\n\tif err != nil {\n\t\treturn empty, false, err\n\t}\n\tif found {\n\t\treturn stored, true, nil\n\t}\n\tauth, err := terminalbackend.ParseAttachAuthorization(authRaw)\n\tif err != nil {\n\t\treturn empty, false, err\n\t}\n\tif err := terminalbackend.CheckAttachRequest(auth, transport, inputAuthorized, now); err != nil {\n\t\treturn empty, false, err\n\t}\n\tif false {\n",
        count=1,
        run="TestAttachRequiresLiveAuth/replay_with_expired_auth",
        expect="KILLED",
        note="Skips authorization exactly on the replay path; fresh attaches still authorize.",
    ),
    Mutant(
        name="N-attach-key",
        path=f"{PACKAGE}/attach.go",
        find='\treturn filepath.Join(store.instanceDir(instanceID), clientID+".json")\n',
        replace='\treturn filepath.Join(store.instanceDir(instanceID), "client.json")\n',
        count=1,
        run="TestAttachSecondClientRecordsAlongside",
        expect="KILLED",
        note="Collapses exactly the client axis of the idempotency key; the instance axis still separates.",
    ),
    Mutant(
        name="C-control",
        path=f"{PACKAGE}/doc.go",
        find="// Division of authority (composed, never forked): v4 payload construction\n",
        replace="// Division of authority (composed, never forked; control): v4 payload construction\n",
        count=1,
        run="TestCheckInstanceIdentityAdmitsUUIDv7",
        expect="SURVIVED",
        note="Harmless control: a comment-only edit must survive.",
    ),
]


def apply(mutant: Mutant, original: str) -> str:
    patched = original.replace(mutant.find, mutant.replace)
    if mutant.find2:
        patched = patched.replace(mutant.find2, mutant.replace2)
    return patched


def run_mutant(mutant: Mutant) -> str:
    target = REPO / mutant.path
    original = target.read_text()
    found = original.count(mutant.find)
    if found != mutant.count:
        return f"ERROR: {mutant.name}: patch anchors {found}, want {mutant.count}"
    if mutant.find2:
        found2 = original.count(mutant.find2)
        if found2 != mutant.count2:
            return f"ERROR: {mutant.name}: second anchors {found2}, want {mutant.count2}"
    with tempfile.TemporaryDirectory() as staging:
        backup = Path(staging) / "backup"
        shutil.copy2(target, backup)
        try:
            target.write_text(apply(mutant, original))
            run = subprocess.run(
                ["go", "test", f"./{PACKAGE}/", "-run", mutant.run, "-count=1"],
                cwd=REPO,
                capture_output=True,
                text=True,
                timeout=300,
            )
        finally:
            shutil.copy2(backup, target)
    output = (run.stdout + run.stderr).strip()
    if os.environ.get("AX_MUTANT_VERBOSE") == "1":
        print(f"--- raw: {mutant.name}: exit {run.returncode} ---", flush=True)
        print(output, flush=True)
        print(f"--- end raw: {mutant.name} ---", flush=True)
    if 'build failed' in output or '\n# ' in output or output.startswith('# '):
        return f"ERROR: {mutant.name}: build broke under the patch"
    if run.returncode == 0:
        verdict = "SURVIVED"
    elif '--- FAIL' in output or 'FAIL:' in output:
        verdict = "KILLED"
    else:
        return f"ERROR: {mutant.name}: exit {run.returncode} with no kill line"
    mark = "ok" if verdict == mutant.expect else "MISMATCH"
    return f"{mark}: {mutant.name}: {verdict} (want {mutant.expect}) :: {mutant.note} [exit {run.returncode}]"


def main(argv: list[str]) -> int:
    selected = [m for m in MUTANTS if not argv or m.name in argv]
    if argv and len(selected) != len(argv):
        unknown = sorted(set(argv) - {m.name for m in selected})
        print(f"unknown mutants: {', '.join(unknown)}")
        return 2
    failures = 0
    for mutant in selected:
        try:
            line = run_mutant(mutant)
        except subprocess.TimeoutExpired:
            line = f"ERROR: {mutant.name}: killer timed out"
        print(line, flush=True)
        if not line.startswith("ok:"):
            failures += 1
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
