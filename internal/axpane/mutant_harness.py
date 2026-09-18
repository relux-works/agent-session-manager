#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/axpane.

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

Row C-doc-comment is the harmless control: a comment-only edit that
must SURVIVE, proving the harness observes test outcomes instead of
hard-coding kills.

Usage:
  PYTHONDONTWRITEBYTECODE=1 python3 internal/axpane/mutant_harness.py
  PYTHONDONTWRITEBYTECODE=1 python3 internal/axpane/mutant_harness.py N-config
"""

import os
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[2]
PACKAGE = "internal/axpane"


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
        name="N-config",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.ConfigErr != nil {\n",
        replace="\tif input.ConfigErr != nil && input.Mode == ModeRestore {\n",
        count=1,
        run="TestDecideTable/refused_config_invalid",
        expect="KILLED",
        note="Admits exactly launch-mode invalid config; restore-mode still refuses.",
    ),
    Mutant(
        name="N-mode",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Mode != ModeLaunch && input.Mode != ModeRestore {\n",
        replace='\tif input.Mode != ModeLaunch && input.Mode != ModeRestore && input.Mode != Mode("") {\n',
        count=1,
        run="TestDecideTable/refused_empty_mode",
        expect="KILLED",
        note="Admits exactly the empty mode; every other non-mode still refuses.",
    ),
    Mutant(
        name="N-session",
        path=f"{PACKAGE}/decide.go",
        find="\tif !input.SessionKnown {\n",
        replace="\tif !input.SessionKnown && input.Mode == ModeRestore {\n",
        count=1,
        run="TestDecideTable/refused_unknown_session",
        expect="KILLED",
        note="Admits exactly launch-mode unknown sessions; restore-mode still refuses.",
    ),
    Mutant(
        name="N-idempotency",
        path=f"{PACKAGE}/decide.go",
        find="\tif (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) {\n",
        replace="\tif (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) && input.Mode == ModeRestore {\n",
        count=1,
        run="TestDecideTable/refused_idempotency_mismatch",
        expect="KILLED",
        note="Admits exactly launch-mode changed operations inside the window; restore-mode still refuses.",
    ),
    Mutant(
        name="N-idempotency-restore",
        path=f"{PACKAGE}/decide.go",
        find="\tif (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) {\n",
        replace="\tif (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) && input.Mode == ModeLaunch {\n",
        count=1,
        run="TestDecideRestoreChangedOperationRefuses",
        expect="KILLED",
        note="Admits exactly restore-mode changed operations inside the window; launch-mode still refuses.",
    ),
    Mutant(
        name="N-idempotency-bound",
        path=f"{PACKAGE}/decide.go",
        find="\tif (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) {\n",
        replace="\tif (input.ExistingBinding == nil || input.ExistingBinding.OperationID != input.BootstrapOperationID) && input.SessionBound && !bootstrapWindowClosed(input) && input.ExistingBinding != nil {\n",
        count=1,
        run="TestDecideRestoreChangedOperationRefuses",
        expect="KILLED",
        note="Admits exactly unrecorded pairs on bound sessions, requiring a mismatched recorded receipt; other changed pairs still refuse.",
    ),
    Mutant(
        name="N-window-fold",
        path=f"{PACKAGE}/decide.go",
        find="\treturn input.HasNewestCheckpoint\n",
        replace="\treturn input.HasNewestCheckpoint || input.Mode == ModeLaunch\n",
        count=1,
        run="TestDecideTable/refused_idempotency_mismatch",
        expect="KILLED",
        note="Closes the window exactly on launch mode without a published checkpoint; restore-mode still refuses in-window.",
    ),
    Mutant(
        name="N-caller",
        path=f"{PACKAGE}/decide.go",
        find="\tif caller != CallerForeground && caller != CallerBackground {\n",
        replace='\tif caller != CallerForeground && caller != CallerBackground && caller != "batch" {\n',
        count=1,
        run="TestDecideTable/refused_unknown_caller",
        expect="KILLED",
        note="Admits exactly the batch caller; every other non-caller still refuses.",
    ),
    Mutant(
        name="N-backend-op",
        path=f"{PACKAGE}/decide.go",
        find='\toperation := "create"\n\tif input.Mode == ModeRestore {\n\t\toperation = "restore"\n\t}\n',
        replace='\toperation := "create"\n\tif input.Mode == ModeRestore {\n\t\toperation = "create"\n\t}\n',
        count=1,
        run="TestDecideRestoreRequiresRestoreCapability",
        expect="KILLED",
        note="Admits restore-mode callers to the create row; the mode selection stays.",
    ),
    Mutant(
        name="N-materialization",
        path=f"{PACKAGE}/decide.go",
        find="\tif !input.JournalOK || input.Journal.Phase != matjournal.PhaseCommitted {\n",
        replace="\tif !input.JournalOK || (input.Journal.Phase != matjournal.PhaseCommitted && input.Journal.Phase != matjournal.PhaseStaging) {\n",
        count=1,
        run="TestDecideTable/parked_materialization_not_committed",
        expect="KILLED",
        note="Admits exactly staging journals; absent and other phases still park.",
    ),
    Mutant(
        name="N-materialization-source",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Journal.SourceCheckpointID != input.NewestCheckpointID {\n",
        replace='\tif input.Journal.SourceCheckpointID != input.NewestCheckpointID && input.Journal.SourceCheckpointID != "sha256:d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2d2" {\n',
        count=1,
        run="TestDecideMaterializationStaleParks",
        expect="KILLED",
        note="Admits exactly the stale source generation; other stale sources still park.",
    ),
    Mutant(
        name="N-materialization-nullfold",
        path=f"{PACKAGE}/decide.go",
        find="\tif !input.HasNewestCheckpoint {\n\t\treturn errMaterializationNoCheckpoint()\n\t}\n",
        replace="\tif !input.HasNewestCheckpoint && input.Mode == ModeLaunch {\n\t\treturn errMaterializationNoCheckpoint()\n\t}\n",
        count=1,
        run="TestDecideJournalNullFoldParks",
        expect="KILLED",
        note="Admits exactly restore-mode journals with no published checkpoint past the null-fold arm; launch-mode still parks with the null cause.",
    ),
    Mutant(
        name="N-successor-nullfold",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Observation.HasWinner && input.Observation.Winner.HasCheckpoint && !input.HasNewestCheckpoint {\n",
        replace="\tif input.Observation.HasWinner && input.Observation.Winner.HasCheckpoint && !input.HasNewestCheckpoint && input.Mode == ModeLaunch {\n",
        count=1,
        run="TestDecideSuccessorNullFoldParks/restore_with_requirements",
        expect="KILLED",
        note="Admits exactly restore-mode successor null folds past the implication arm (the per-arm null cause surfaces instead); launch-mode still parks with the implication cause.",
    ),
    Mutant(
        name="N-checkpoint-nullfold",
        path=f"{PACKAGE}/decide.go",
        find="\tif !input.HasNewestCheckpoint {\n\t\treturn nil, errCheckpointNoPublished()\n\t}\n",
        replace="\tif !input.HasNewestCheckpoint && input.Mode == ModeLaunch {\n\t\treturn nil, errCheckpointNoPublished()\n\t}\n",
        count=1,
        run="TestDecideCheckpointNullFoldParks",
        expect="KILLED",
        note="Admits exactly restore-mode checkpoints with no published checkpoint past the null-fold arm; launch-mode still parks with the null cause.",
    ),
    Mutant(
        name="N-closure-source",
        path=f"{PACKAGE}/run.go",
        find="\t\t\t\tloaded, err := LoadCheckpoint(stores.Ckpt, input.NewestCheckpointID)\n",
        replace="\t\t\t\tloaded, err := LoadCheckpoint(stores.Ckpt, observation.Winner.Checkpoint)\n",
        count=1,
        run="TestRunPostTakeoverLaunchCarriesNewestClosureYoloToStandard",
        expect="KILLED",
        note="Reads the profile closure from the lease handoff base again instead of the fold newest; the yolo-to-standard launch carries the stale yolo closure.",
    ),
    Mutant(
        name="N-checkpoint-absent",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.CheckpointRequired {\n",
        replace="\tif input.CheckpointRequired && len(input.CheckpointDoc) > 0 {\n",
        count=1,
        run="TestDecideTable/parked_checkpoint_absent",
        expect="KILLED",
        note="Admits exactly absent checkpoints; inadmissible bytes still park.",
    ),
    Mutant(
        name="N-checkpoint-prefix",
        path=f"{PACKAGE}/decide.go",
        find="\tif digest.String() != want {\n",
        replace="\tif digest.String()[:16] != want[:16] {\n",
        count=1,
        run="TestDecideTable/parked_checkpoint_flipped_digest",
        expect="KILLED",
        note="Admits exactly same-prefix digests; full comparison otherwise stays.",
    ),
    Mutant(
        name="N-identity-version",
        path=f"{PACKAGE}/decide.go",
        find="\tif err := provhost.VerifyIdentityBuild(facts.Identity, facts.Build); err != nil {\n",
        replace='\tif err := provhost.VerifyIdentityBuild(facts.Identity, facts.Build); err != nil && facts.Build.ProviderVersion != "0.147.0" {\n',
        count=1,
        run="TestDecideTable/refused_provider_identity_other_provider",
        expect="KILLED",
        note="Admits exactly 0.147.0 identity mismatches; other versions still refuse.",
    ),
    Mutant(
        name="N-discovery",
        path=f"{PACKAGE}/decide.go",
        find="\tif facts.HasDiscovery {\n",
        replace='\tif facts.HasDiscovery && facts.DiscoveryContext.Home != "" {\n',
        count=1,
        run="TestDecideTable/refused_discovery_empty_home",
        expect="KILLED",
        note="Admits exactly empty-home discovery; every other proof still verifies.",
    ),
    Mutant(
        name="N-smoke-verdict",
        path=f"{PACKAGE}/decide.go",
        find="\tif record.Verdict != resumesmoke.VerdictPass {\n",
        replace="\tif record.Verdict != resumesmoke.VerdictPass && record.Verdict != resumesmoke.VerdictFail {\n",
        count=1,
        run="TestDecideTable/refused_smoke_failing_verdict",
        expect="KILLED",
        note="Admits exactly failing verdicts; gated, unsupported, and unknown still refuse.",
    ),
    Mutant(
        name="N-smoke-version",
        path=f"{PACKAGE}/decide.go",
        find="""\tif record.Tuple.ProviderID != build.ProviderID ||
\t\trecord.Tuple.ProviderVersion != build.ProviderVersion ||
\t\trecord.Tuple.Platform != build.Platform ||
\t\trecord.Tuple.Architecture != build.Architecture {
""",
        replace="""\tif record.Tuple.ProviderID != build.ProviderID ||
\t\trecord.Tuple.Platform != build.Platform ||
\t\trecord.Tuple.Architecture != build.Architecture {
""",
        count=1,
        run="TestDecideTable/refused_smoke_wrong_build",
        expect="KILLED",
        note="Admits exactly version drift; provider, platform, and arch still bind.",
    ),
    Mutant(
        name="N-smoke-mode",
        path=f"{PACKAGE}/decide.go",
        find="\tif !input.Smoke.Required {\n",
        replace="\tif !input.Smoke.Required || input.Mode == ModeLaunch {\n",
        count=1,
        run="TestDecideTable/refused_smoke_absent",
        expect="KILLED",
        note="Skips the required smoke exactly on launch; restore-mode still enforces it.",
    ),
    Mutant(
        name="N-profile-derive",
        path=f"{PACKAGE}/decide.go",
        find="""\tpair, err := deriveProfile(input)
\tif err != nil {
\t\treturn refused(ClassIntegrityFailure, "effective profile derivation failed", err)
\t}
""",
        replace="""\tpair, err := deriveProfile(input)
\tif err != nil && input.Provider.Build.ProviderID != "codex" {
\t\treturn refused(ClassIntegrityFailure, "effective profile derivation failed", err)
\t}
""",
        count=1,
        run="TestDecideTable/refused_profile_derivation_corrupt",
        expect="KILLED",
        note="Admits corrupt derivation exactly for codex, surfacing the wrong class.",
    ),
    Mutant(
        name="N-profile-closure",
        path=f"{PACKAGE}/decide.go",
        find="""\tif len(heads) > 0 {
\t\treturn sessprofile.DeriveForHeads(input.ProfileRecord, input.ProfileEvents, heads)
\t}
""",
        replace="""\tif len(heads) > 0 && input.Mode != ModeRestore {
\t\treturn sessprofile.DeriveForHeads(input.ProfileRecord, input.ProfileEvents, heads)
\t}
""",
        count=1,
        run="TestDecideResumeProfileUsesClosure",
        expect="KILLED",
        note="Derives the session head instead of the closure exactly on restore; launch closure still holds.",
    ),
    Mutant(
        name="N-mapping",
        path=f"{PACKAGE}/decide.go",
        find="""\tresolved, err := provhost.ResolveMapping(input.Provider.Build.ProviderID, pair.Profile, input.Provider.Build)
\tif err != nil {
\t\treturn refused(ClassProfileMappingUnavailable, "adapter cannot map the stored profile", err)
\t}
""",
        replace="""\tresolved, err := provhost.ResolveMapping(input.Provider.Build.ProviderID, pair.Profile, input.Provider.Build)
\tif err != nil && pair.Profile != "standard" {
\t\treturn refused(ClassProfileMappingUnavailable, "adapter cannot map the stored profile", err)
\t}
""",
        count=1,
        run="TestDecideTable/refused_profile_mapping_unavailable",
        expect="KILLED",
        note="Admits exactly standard mapping gaps; yolo gaps still refuse.",
    ),
    Mutant(
        name="N-realm",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Realm.Caller != CallerBackground && !input.Smoke.Required {\n\t\treturn nil\n\t}\n\tif !admitted.Has(credentialRealmCapability) {\n",
        replace='\tif input.Realm.Caller != CallerBackground && !input.Smoke.Required || input.Realm.BrokerState == "broker-running" {\n\t\treturn nil\n\t}\n\tif !admitted.Has(credentialRealmCapability) {\n',
        count=1,
        run="TestDecideTable/refused_realm_background_no_server",
        expect="KILLED",
        note="Admits exactly broker-running background callers; other states still refuse.",
    ),
    Mutant(
        name="N-realm-foreground",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Realm.Caller != CallerBackground && !input.Smoke.Required {\n\t\treturn nil\n\t}\n\tif !admitted.Has(credentialRealmCapability) {\n",
        replace="\tif input.Realm.Caller != CallerBackground {\n\t\treturn nil\n\t}\n\tif !admitted.Has(credentialRealmCapability) {\n",
        count=1,
        run="TestDecideForegroundCredentialRequiresRealm",
        expect="KILLED",
        note="Admits exactly foreground credential-requiring callers without the realm row; background callers still refuse.",
    ),
    Mutant(
        name="N-realm-deadmap",
        path=f"{PACKAGE}/decide.go",
        find="\t\tif realmErr := deadRealmRefusal(input, evidence, err); realmErr != nil {\n",
        replace="\t\tif realmErr := deadRealmRefusal(input, evidence, err); realmErr != nil && input.Mode == ModeRestore {\n",
        count=1,
        run="TestDecideExpiredRealmRowRefusesUnavailable",
        expect="KILLED",
        note="Admits exactly launch-mode dead-realm evidence to the backend class; restore-mode still reports capability_unavailable.",
    ),
    Mutant(
        name="N-entrypoint",
        path=f"{PACKAGE}/decide.go",
        find="\tif err := terminalbackend.CheckEntrypoint(input.Entrypoint, input.SessionID); err != nil {\n",
        replace="\tif err := terminalbackend.CheckEntrypoint(input.Entrypoint, input.SessionID); err != nil && len(input.Entrypoint) != 2 {\n",
        count=1,
        run="TestDecideTable/refused_entrypoint_raw_provider",
        expect="KILLED",
        note="Admits exactly two-element argv; other arities still refuse.",
    ),
    Mutant(
        name="N-descriptor",
        path=f"{PACKAGE}/decide.go",
        find="""\tdescriptor, err := terminalbackend.AdmitProviderDescriptor(input.Descriptor, input.HostBinding)
\tif err != nil {
\t\treturn refused(backendClass(err), "provider descriptor does not match the validated binding", err)
\t}
""",
        replace="""\tdescriptor, err := terminalbackend.AdmitProviderDescriptor(input.Descriptor, input.HostBinding)
\tif err != nil && input.Mode != ModeLaunch {
\t\treturn refused(backendClass(err), "provider descriptor does not match the validated binding", err)
\t}
""",
        count=1,
        run="TestDecideTable/refused_descriptor_generation_drift",
        expect="KILLED",
        note="Admits exactly launch-mode descriptor mismatch; restore-mode still refuses.",
    ),
    Mutant(
        name="N-attach",
        path=f"{PACKAGE}/decide.go",
        find='\tif terminalbackend.CheckOperation("attach", admitted) == nil {\n',
        replace="\tif true {\n",
        count=1,
        run="TestDecideTable/takeover_offer_attach_unproven",
        expect="KILLED",
        note="Admits remote attach without the capability; attach-proven still offers attach.",
    ),
    Mutant(
        name="N-reattach",
        path=f"{PACKAGE}/decide.go",
        find="""\tif input.ExistingBinding != nil && input.ExistingBinding.OperationID == input.BootstrapOperationID {
\t\tdecision.Action = ActionReattach
""",
        replace="""\tif input.ExistingBinding != nil && input.ExistingBinding.OperationID == input.BootstrapOperationID && input.Mode == ModeRestore {
\t\tdecision.Action = ActionReattach
""",
        count=1,
        run="TestDecideTable/reattach_same_pair",
        expect="KILLED",
        note="Selects launch instead of reattach exactly on launch mode.",
    ),
    Mutant(
        name="N-binding-prefix",
        path=f"{PACKAGE}/binding.go",
        find="\t\t\tif winner.OperationID != operationID {\n",
        replace="\t\t\tif winner.OperationID[:8] != operationID[:8] {\n",
        count=1,
        run="TestBindConcurrentChangedOperationRefuses",
        expect="KILLED",
        note="Admits exactly same-prefix changed operations; full comparison otherwise stays.",
    ),
    Mutant(
        name="N-binding-session",
        path=f"{PACKAGE}/binding.go",
        find='\tif binding.SessionID != sessionID {\n\t\treturn empty, false, errors.New("bootstrap binding names another session")\n\t}\n',
        replace='\tif binding.SessionID[:8] != sessionID[:8] {\n\t\treturn empty, false, errors.New("bootstrap binding names another session")\n\t}\n',
        count=1,
        run="TestStatusSessionMismatchRefuses",
        expect="KILLED",
        note="Admits exactly same-prefix foreign sessions; full comparison otherwise stays.",
    ),
    Mutant(
        name="N-bind-first",
        path=f"{PACKAGE}/binding.go",
        find="\t\tif existing.OperationID != operationID {\n",
        replace='\t\tif existing.OperationID != operationID && operationID != "0198f4c8-7d40-7e55-8e6f-1234567890ac" {\n',
        count=1,
        run="TestBindChangedOperationRefusesMismatch",
        expect="KILLED",
        note="Admits exactly the changed test operation past the anchor pre-check; other changed operations still refuse.",
    ),
    Mutant(
        name="N-supersede-no-replace",
        path=f"{PACKAGE}/binding.go",
        find="""\t\t\twinner, found, readErr := store.Lookup(sessionID, operationID)
\t\t\tif readErr != nil {
\t\t\t\treturn empty, false, readErr
\t\t\t}
\t\t\tif !found {
\t\t\t\treturn empty, false, errors.New("bootstrap pair receipt vanished after concurrent commit")
\t\t\t}
\t\t\treturn winner, true, nil
""",
        replace="""\t\t\twinner, found, readErr := store.Lookup(sessionID, operationID)
\t\t\tif readErr != nil {
\t\t\t\treturn empty, false, readErr
\t\t\t}
\t\t\tif !found {
\t\t\t\treturn empty, false, errors.New("bootstrap pair receipt vanished after concurrent commit")
\t\t\t}
\t\t\tif operationID == "0198f4c8-7d40-7e55-8e6f-1234567890ac" {
\t\t\t\t_ = os.Remove(target)
\t\t\t\tif err := os.Link(stagedPath, target); err != nil {
\t\t\t\t\treturn empty, false, err
\t\t\t\t}
\t\t\t\tdecoded, decodeErr := decodeBinding(receipt)
\t\t\t\treturn decoded, false, decodeErr
\t\t\t}
\t\t\treturn winner, true, nil
""",
        count=1,
        run="TestSupersedeConcurrentSamePairReplays",
        expect="KILLED",
        note="Replaces exactly the test pair receipt on the commit race; other pairs still replay the winner.",
    ),
    Mutant(
        name="N-parked-reason",
        path=f"{PACKAGE}/events.go",
        find='\tdefault:\n\t\treturn nil, fmt.Errorf("parked reason %q is not the session.parked vocabulary", string(reason))\n\t}\n',
        replace='\tdefault:\n\t\tif len(reason) != 5 {\n\t\t\treturn nil, fmt.Errorf("parked reason %q is not the session.parked vocabulary", string(reason))\n\t\t}\n\t}\n',
        count=1,
        run="TestParkedPayloadVocabulary",
        expect="KILLED",
        note="Admits exactly five-letter reasons; the vocabulary check otherwise stays.",
    ),
    Mutant(
        name="N-checkpoint-newest",
        path=f"{PACKAGE}/decide.go",
        find="\tif want != input.NewestCheckpointID {\n",
        replace="\tif want != input.NewestCheckpointID && input.Mode == ModeLaunch {\n",
        count=1,
        run="TestDecideCheckpointNotLeaseParks",
        expect="KILLED",
        note="Admits exactly restore-mode non-newest checkpoints; launch-mode still parks.",
    ),
    Mutant(
        name="N-checkpoint-session",
        path=f"{PACKAGE}/decide.go",
        find="\tif sessionID != input.SessionID {\n",
        replace='\tif sessionID != input.SessionID && sessionID != "0198f4c8-3e70-7a11-8a2b-1234567890ff" {\n',
        count=1,
        run="TestDecideForeignCheckpointParks",
        expect="KILLED",
        note="Admits exactly the foreign fixture session; other foreign sessions still park.",
    ),

    Mutant(
        name="N-realm-binding",
        path=f"{PACKAGE}/decide.go",
        find="\t\tif object.TerminalBindingID != input.HostBinding.TerminalBindingID {\n",
        replace='\t\tif object.TerminalBindingID != input.HostBinding.TerminalBindingID && object.TerminalBindingID != "sha256:b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2" {\n',
        count=1,
        run="TestDecideRealmBindingMismatchRefuses/binding",
        expect="KILLED",
        note="Admits exactly the mismatched test binding; other mismatches still refuse.",
    ),
    Mutant(
        name="N-realm-generation",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Realm.ServerGeneration != input.Backend.RawGeneration {\n",
        replace='\tif input.Realm.ServerGeneration != input.Backend.RawGeneration && input.Realm.ServerGeneration != "generation-stale" {\n',
        count=1,
        run="TestDecideRealmBindingMismatchRefuses/generation",
        expect="KILLED",
        note="Admits exactly the stale test generation; other stale generations still refuse.",
    ),
    Mutant(
        name="N-headless",
        path=f"{PACKAGE}/decide.go",
        find="\tif input.Mode == ModeLaunch && !input.Interactive && !admitted.Has(headlessCreationCapability) {\n",
        replace='\tif input.Mode == ModeLaunch && !input.Interactive && !admitted.Has(headlessCreationCapability) && !admitted.Has("durable_disconnect") {\n',
        count=1,
        run="TestDecideNonInteractiveRequiresHeadless",
        expect="KILLED",
        note="Admits exactly durable-holding non-interactive creates without headless; other creates still refuse.",
    ),
    Mutant(
        name="N-emit-mutation",
        path=f"{PACKAGE}/events.go",
        find="\tif _, err := fencing.AuthorizeMutation(params.Presented, params.Observation); err != nil {\n",
        replace="\tif _, err := fencing.AuthorizeMutation(params.Presented, params.Observation); err != nil && params.Observation.Verified {\n",
        count=1,
        run="TestEmitParkedVocabularyLoop/refuses_unverified",
        expect="KILLED",
        note="Admits exactly unverified emissions; verified refusals still refuse.",
    ),
    Mutant(
        name="N-parked-fold",
        path=f"{PACKAGE}/events.go",
        find="\tif !foldable {\n",
        replace="\tif !foldable && decision.ParkReason != fencing.ParkRestorePolicy {\n",
        count=1,
        run="TestEmitParkedUnfoldableRefuses",
        expect="KILLED",
        note="Admits exactly unfolderable restore_policy parks; other unfolderable reasons still refuse.",
    ),
    Mutant(
        name="N-run-fold",
        path=f"{PACKAGE}/run.go",
        find="\t\tif !foldable {\n",
        replace="\t\tif !foldable && decision.ParkReason != fencing.ParkRestorePolicy {\n",
        count=1,
        run="TestRunParkedUnfoldableSkipsEmission",
        expect="KILLED",
        note="Run admits exactly unfolderable restore_policy parks; other unfolderable still skip.",
    ),
    Mutant(
        name="N-run-ckpt-store",
        path=f"{PACKAGE}/run.go",
        find="\tif known && input.HasNewestCheckpoint && stores.Ckpt == nil {\n",
        replace="\tif known && input.HasNewestCheckpoint && stores.Ckpt == nil && request.Mode == ModeRestore {\n",
        count=1,
        run="TestRunNoCkptStorePublishedNewestFails",
        expect="KILLED",
        note="Admits exactly launch-mode runs with a published newest and no store (the head pair launches); restore-mode still fails.",
    ),
    Mutant(
        name="N-window-lease-checkpoint",
        path=f"{PACKAGE}/decide.go",
        find="\treturn input.HasNewestCheckpoint\n}\n",
        replace="\treturn input.HasNewestCheckpoint || (input.Observation.HasWinner && input.Observation.Winner.HasCheckpoint)\n}\n",
        count=1,
        run="TestRunWindowLeaseCheckpointNullFoldRefuses",
        expect="KILLED",
        note="Closes the window on the lease record's checkpoint while the fold names none; the null-fold changed op launches instead of refusing.",
    ),
    Mutant(
        name="N-run-fold-error",
        path=f"{PACKAGE}/run.go",
        find="\t\thas, newest, err := LoadNewestCheckpoint(stores.Repo, request.SessionID)\n\t\tif err != nil {\n\t\t\treturn empty, err\n\t\t}\n",
        replace="\t\thas, newest, err := LoadNewestCheckpoint(stores.Repo, request.SessionID)\n\t\tif err != nil {\n\t\t\thas, newest = false, \"\"\n\t\t}\n",
        count=1,
        run="TestRunUnfoldableChainFails",
        expect="KILLED",
        note="Launders an unfoldable chain as a null fold; Run parks instead of failing with the fold error.",
    ),
    Mutant(
        name="N-supersede-replay",
        path=f"{PACKAGE}/run.go",
        find="\t\t\tif replayed {\n",
        replace="\t\t\tif replayed && input.Mode == ModeLaunch {\n",
        count=1,
        run="TestRunPostWindowSamePairRaceReattaches",
        expect="KILLED",
        note="Keeps launch on the post-window replay exactly in restore mode; launch-mode replays still reattach.",
    ),
    Mutant(
        name="N-identity-session",
        path=f"{PACKAGE}/decide.go",
        find="\tif session.String() != sessionID {\n",
        replace='\tif session.String() != sessionID && session.String() != "0198f4c8-3e70-7a11-8a2b-1234567890ff" {\n',
        count=1,
        run="TestDecideIdentityForeignSessionRefuses",
        expect="KILLED",
        note="Admits exactly the foreign fixture session's identity record; other foreign sessions still refuse.",
    ),
    Mutant(
        name="N-run-newest-absent",
        path=f"{PACKAGE}/run.go",
        find="\t\t\t\tif len(loaded) == 0 {\n",
        replace="\t\t\t\tif len(loaded) == 0 && request.Mode == ModeLaunch {\n",
        count=1,
        run="TestRunNewestAbsentFromStoreFails/restore",
        expect="KILLED",
        note="Admits exactly restore-mode runs with the newest absent from the store (the head pair launches); launch-mode still fails.",
    ),
    Mutant(
        name="N-emit-losing-epoch",
        path=f"{PACKAGE}/events.go",
        find="\tif _, err := fencing.AuthorizeMutation(params.Presented, params.Observation); err != nil {\n",
        replace="\tif _, err := fencing.AuthorizeMutation(params.Presented, params.Observation); err != nil && params.Presented.Epoch >= params.Observation.Winner.Epoch {\n",
        count=1,
        run="TestEmitUnderLosingLeaseRefuses",
        expect="KILLED",
        note="Admits exactly losing-epoch emissions; same-or-winning-epoch refusals still refuse.",
    ),
    Mutant(
        name="N-parked-winner",
        path=f"{PACKAGE}/events.go",
        find='\tif decision.WinningLeaseID != params.LeaseID {\n',
        replace='\tif decision.WinningLeaseID != params.LeaseID && decision.ParkReason != fencing.ParkStaleOwner {\n',
        count=1,
        run="TestEmitParkedWinningLeaseMismatchRefuses",
        expect="KILLED",
        note="Admits exactly stale_owner winner mismatches; other reasons still refuse. Story-close P3-4.",
    ),
    Mutant(
        name="N-journal-closure",
        path=f"{PACKAGE}/run.go",
        find="\t\t\t\tloaded, err := LoadCheckpoint(stores.Ckpt, input.NewestCheckpointID)\n",
        replace="\t\t\t\tclosureID := input.NewestCheckpointID\n\t\t\t\tif request.Mode == ModeRestore && input.Observation.Winner.HasCheckpoint {\n\t\t\t\t\tclosureID = input.Observation.Winner.Checkpoint\n\t\t\t\t}\n\t\t\t\tloaded, err := LoadCheckpoint(stores.Ckpt, closureID)\n",
        count=1,
        run="TestRunJournalOnlyRestoreClosureFromNewest",
        expect="KILLED",
        note="Reads the lease handoff base exactly on journal-only restore with a checkpoint-carrying winner; all other closures still come from the fold newest. Story-close P3-1.",
    ),
    Mutant(
        name="N-ckpt-corrupt",
        path=f"{PACKAGE}/gates.go",
        find="\tif os.IsNotExist(err) {\n",
        replace="\tif os.IsNotExist(err) || errors.Is(err, sessckpt.ErrInvalidCheckpoint) {\n",
        count=1,
        run="TestRunCorruptCheckpointBlobIsNotAbsence",
        expect="KILLED",
        note="Launders exactly the invalid-closure class as absence; other read failures still propagate. Story-close P3-5.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/doc.go",
        find="// `ax pane SESSION_ID` enforcement wrapper: the decision every managed\n",
        replace="// `ax pane SESSION_ID` enforcement wrapper (control): the decision every managed\n",
        count=1,
        run="TestRealmCarriesNoCachedEvidence",
        expect="SURVIVED",
        note="Harmless control: a comment-only edit must survive.",
    ),
]


def run_mutant(mutant: Mutant) -> str:
    target = REPO / mutant.path
    original = target.read_text()
    found = original.count(mutant.find)
    if found != mutant.count:
        return f"ERROR: {mutant.name}: patch anchors {found}, want {mutant.count}"
    with tempfile.TemporaryDirectory() as staging:
        backup = Path(staging) / "backup"
        shutil.copy2(target, backup)
        try:
            target.write_text(original.replace(mutant.find, mutant.replace))
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
