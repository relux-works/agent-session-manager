#!/usr/bin/env python3
"""Narrowing-mutant harness for internal/clonebundle.

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
  PYTHONDONTWRITEBYTECODE=1 python3 internal/clonebundle/testdata/mutant_harness.py [--log-dir DIR] [MUTANT ...]
"""

import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

REPO = Path(__file__).resolve().parents[3]
PACKAGE = "internal/clonebundle"

FORGED_DESCRIPTOR_HEX = "edb3da2954b883ebc2099da18edeac4e12f8f5d67aa5bb2dd80ba472b23481f2"
OTHER_DESCRIPTOR_HEX = "369c0225affac23e2a7a16052d03b24bfd9421ea3f83c8fe39111e864c344cfb"
PAIR_DIGEST = "sha256:305e10453577c8371cdb872e120f1662649d150bbc2eff62b81827851f1d4926"


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
        name="N-raw-class",
        path=f"{PACKAGE}/rawmanifest.go",
        find='\t"derived_cache_optional",\n\t"unknown",\n',
        replace='\t"derived_cache_optional",\n\t"credential",\n\t"unknown",\n',
        count=1,
        run="TestRawManifestRefusals/build/credential_forbidden",
        expect="KILLED",
        note="Admits exactly credential into the raw subset; every other excluded class still refuses.",
    ),
    Mutant(
        name="N-raw-order",
        path=f"{PACKAGE}/rawmanifest.go",
        find="\tif index > 0 && input.NativeItemKey <= previous {\n",
        replace="\tif index > 0 && input.NativeItemKey < previous {\n",
        count=1,
        run="TestRawManifestRefusals/build/duplicate_entries",
        expect="KILLED",
        note="Admits exactly duplicate keys; unsorted keys still refuse.",
    ),
    Mutant(
        name="N-exclusion-config",
        path=f"{PACKAGE}/rawmanifest.go",
        find="\tif hosttrust.ExcludedConfigDirName(key) {\n",
        replace='\tif hosttrust.ExcludedConfigDirName(key) && key != "settings.bak.3" {\n',
        count=1,
        run="TestRawManifestExcludesTrustMaterial",
        expect="KILLED",
        note="Admits exactly the settings.bak.3 config key; every other excluded key still refuses. Token preserved.",
    ),
    Mutant(
        name="N-sanitize-uuid",
        path=f"{PACKAGE}/identity.go",
        find="\tif _, err := scalar.ParseUUIDv7(key); err == nil {\n",
        replace='\tif _, err := scalar.ParseUUIDv7(key); err == nil && key != "0198f4c8-8e50-7f66-8f70-1234567890c2" {\n',
        count=1,
        run="TestIdentityRefusals/sanitizer",
        expect="KILLED",
        note="Admits exactly one UUIDv7 native key; every other UUIDv7 still refuses. Token preserved.",
    ),
    Mutant(
        name="N-sanitizer-control",
        path=f"{PACKAGE}/identity.go",
        find="\t\tif unicode.IsControl(unit) {\n",
        replace="\t\tif unicode.IsControl(unit) && unit != 0x1f {\n",
        count=1,
        run="TestIdentityRefusals/sanitizer",
        expect="KILLED",
        note="Admits exactly U+001F; every other control still refuses.",
    ),
    Mutant(
        name="N-always-excluded",
        path=f"{PACKAGE}/captureitem.go",
        find='\tcase "credential", "machine_auth", "runtime_state", "transient_lock":\n',
        replace='\tcase "machine_auth", "runtime_state", "transient_lock":\n',
        count=1,
        run="TestCaptureItemRefusals/credential_included",
        expect="KILLED",
        note="Admits exactly included credential rows; the other three always-excluded classes still refuse.",
    ),
    Mutant(
        name="N-included-reason",
        path=f"{PACKAGE}/captureitem.go",
        find="\t\tif hasReason {\n",
        replace='\t\tif hasReason && class == "unknown" {\n',
        count=1,
        run="TestCaptureItemRefusals/included_with_reason",
        expect="KILLED",
        note="Admits exactly non-unknown included rows with a reason; the killer uses durable_payload.",
    ),
    Mutant(
        name="N-unknown-complete",
        path=f"{PACKAGE}/capturebuild.go",
        find="\tif hasUnknownClass(items) {\n",
        replace="\tif hasUnknownClass(items) && len(items) > 4 {\n",
        count=1,
        run="TestBuildCaptureManifestUnknownMakesIncomplete",
        expect="KILLED",
        note="Admits exactly small unknown-class manifests as complete; the 4-item killer derives true.",
    ),
    Mutant(
        name="N-excluded-exact",
        path=f"{PACKAGE}/capturebuild.go",
        find="\tif len(derived) != len(excluded) {\n",
        replace="\tif len(derived) > len(excluded) {\n",
        count=1,
        run="TestExcludedClassesExactGate",
        expect="KILLED",
        note="Admits exactly superset excluded_classes; missing classes still refuse.",
    ),
    Mutant(
        name="N-proof-identity",
        path=f"{PACKAGE}/boundary.go",
        find="\t\tif proof.SnapshotIdentity != nil {\n",
        replace='\t\tif proof.SnapshotIdentity != nil && proof.ProofKind != "closed_store" {\n',
        count=1,
        run="TestBoundaryRefusals/decode_stable_proof/closed_store_with_identity",
        expect="KILLED",
        note="Admits exactly closed_store with an identity; provider_quiescence still refuses one.",
    ),
    Mutant(
        name="N-core-only",
        path=f"{PACKAGE}/boundary.go",
        find="\t\tif !input.Core {\n",
        replace='\t\tif !input.Core && input.Generation != "g" {\n',
        count=1,
        run="TestBoundaryRefusals/build_core-only_unstable",
        expect="KILLED",
        note="Admits exactly non-core unstable with generation g, the killer's generation.",
    ),
    Mutant(
        name="N-g2-generation",
        path=f"{PACKAGE}/boundary.go",
        find='''\tif boundary.Kind == "unstable_archive" {
\t\treturn invalid("capture boundary unstable_archive cannot enter a target branch")
\t}
''',
        replace='''\tif boundary.Kind == "unstable_archive" && boundary.SourceGeneration.String() != "generation-9" {
\t\treturn invalid("capture boundary unstable_archive cannot enter a target branch")
\t}
\tif boundary.Kind == "unstable_archive" {
\t\treturn nil
\t}
''',
        count=1,
        run="TestBuildCaptureManifestUnstableArchive",
        expect="KILLED",
        note="Admits exactly generation-9 unstable archives to the target branch; every other unstable still refuses.",
    ),
    Mutant(
        name="N-main-count-build",
        path=f"{PACKAGE}/session.go",
        find='\tif mainCount != 1 {\n\t\treturn nil, invalid("canonical session carries %d main actors, want exactly one", mainCount)\n\t}\n\teventIDs, err := parseOrderedUniqueDigests',
        replace='\tif mainCount < 1 {\n\t\treturn nil, invalid("canonical session carries %d main actors, want exactly one", mainCount)\n\t}\n\teventIDs, err := parseOrderedUniqueDigests',
        count=1,
        run="TestCanonicalSessionRefusals/build_actors/two_mains",
        expect="KILLED",
        note="Admits exactly multi-main sessions at the build site only; zero-main still refuses. The decode site keeps its own row.",
    ),
    Mutant(
        name="N-main-count-decode",
        path=f"{PACKAGE}/session.go",
        find='\tif mainCount != 1 {\n\t\treturn nil, invalid("canonical session carries %d main actors, want exactly one", mainCount)\n\t}\n\treturn actors, nil',
        replace='\tif mainCount < 1 {\n\t\treturn nil, invalid("canonical session carries %d main actors, want exactly one", mainCount)\n\t}\n\treturn actors, nil',
        count=1,
        run="TestCanonicalSessionRefusals/decode/two_mains",
        expect="KILLED",
        note="Admits exactly multi-main sessions at the decode site only; zero-main still refuses. The build site keeps its own row.",
    ),
    Mutant(
        name="N-synth-operation",
        path=f"{PACKAGE}/event.go",
        find="\t\tif !hasOperation {\n",
        replace="\t\tif !hasOperation && hasNativeEvent {\n",
        count=1,
        run="TestCanonicalEventRefusals/evidence/synthesized_missing_operation",
        expect="KILLED",
        note="Admits exactly synthesized evidence with neither native event nor operation, the killer's shape.",
    ),
    Mutant(
        name="N-blob-size",
        path=f"{PACKAGE}/blobref.go",
        find="\tif size != wantSize {\n",
        replace="\tif size != wantSize && size+1 != wantSize {\n",
        count=1,
        run="TestBlobAgreementRefusals",
        expect="KILLED",
        note="Admits exactly off-by-one size claims; every other disagreement still refuses.",
    ),
    Mutant(
        name="N-self-digest",
        path=f"{PACKAGE}/decode.go",
        find="\tif calculated != claimed {\n",
        replace='\tif calculated != claimed && claimed.Hex() != "ccdd35168ab474fa5764a526cfb83621351e23682c5075b2e18d56bddf96aa30" {\n',
        count=1,
        run="TestRawManifestRefusals/decode/self_digest_mismatch",
        expect="KILLED",
        note="Admits exactly the forged self digest sha256('forged'); every other mismatch still refuses.",
    ),
    Mutant(
        name="N-descriptor-claim-agreement",
        path=f"{PACKAGE}/blobref.go",
        find="\tcalculated, claimed, members, err := calculateDescriptorIdentity(descriptor)\n\tif err != nil {\n\t\treturn err\n\t}\n\tif calculated != claimed {\n",
        replace='\tcalculated, claimed, members, err := calculateDescriptorIdentity(descriptor)\n\tif err != nil {\n\t\treturn err\n\t}\n\tif calculated != claimed && claimed.Hex() != "' + FORGED_DESCRIPTOR_HEX + '" {\n',
        count=1,
        run="TestBlobAgreementRefusals",
        expect="KILLED",
        note="Admits exactly the forged descriptor claim at the agreement site only; the install site keeps its own row.",
    ),
    Mutant(
        name="N-install-site-claim",
        path=f"{PACKAGE}/blobref.go",
        find="\tcalculated, claimed, members, err := calculateDescriptorIdentity(descriptor)\n\tif err != nil {\n\t\treturn BlobAgreement{}, err\n\t}\n\tif calculated != claimed {\n",
        replace='\tcalculated, claimed, members, err := calculateDescriptorIdentity(descriptor)\n\tif err != nil {\n\t\treturn BlobAgreement{}, err\n\t}\n\tif calculated != claimed && claimed.Hex() != "' + FORGED_DESCRIPTOR_HEX + '" {\n',
        count=1,
        run="TestInstallRawBlobRefusals",
        expect="KILLED",
        note="Admits exactly the forged descriptor claim at the install site only; the agreement site keeps its own row.",
    ),
    Mutant(
        name="N-descriptor-id-link",
        path=f"{PACKAGE}/blobref.go",
        find="\tif calculated != wantDescriptorID {\n",
        replace='\tif calculated != wantDescriptorID && wantDescriptorID.String() != "sha256:' + OTHER_DESCRIPTOR_HEX + '" {\n',
        count=1,
        run="TestRawManifestRefusals/build/descriptor_id_mismatch",
        expect="KILLED",
        note="Admits exactly the killer's mismatched descriptor ID; every other mismatch still refuses.",
    ),
    Mutant(
        name="N-extension-number",
        path=f"{PACKAGE}/decode.go",
        find="\tinteger, err := strconv.ParseInt(literal, 10, 64)\n\tif err != nil || integer < -maxSafeInteger || integer > maxSafeInteger {\n",
        replace="\tinteger, err := strconv.ParseInt(literal, 10, 64)\n\tif err != nil || integer < -maxSafeInteger || (integer > maxSafeInteger && integer != 9007199254740992) {\n",
        count=1,
        run="TestRawManifestRefusals/extension_values/exactly_2",
        expect="KILLED",
        note="Admits exactly 2^53; every other unsafe literal still refuses.",
    ),
    Mutant(
        name="N-extension-nested-dup",
        path=f"{PACKAGE}/decode.go",
        find="\t\tif seen[name] {\n\t\t\treturn errExtensionDuplicate\n",
        replace='\t\tif seen[name] && name != "a" {\n\t\t\treturn errExtensionDuplicate\n',
        count=1,
        run="TestRawManifestRefusals/extension_values/decode_nested_duplicate",
        expect="KILLED",
        note="Admits exactly a duplicate nested member named a; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-extension-keys",
        path=f"{PACKAGE}/decode.go",
        find="\tif !checkExtensions(raw) {\n\t\treturn errExtensionsKeys\n",
        replace='\tif !checkExtensions(raw) && !bytes.Contains(raw, []byte("\\"no-dots\\":")) {\n\t\treturn errExtensionsKeys\n',
        count=1,
        run="TestRawManifestRefusals/decode/bad_extensions",
        expect="KILLED",
        note="Admits exactly extensions carrying a no-dots key; every other bad key still refuses.",
    ),
    Mutant(
        name="N-capture-items-dup-decode",
        path=f"{PACKAGE}/capturebuild.go",
        find="\t\tif index > 0 && item.NativeItemKey <= previous {\n\t\t\treturn nil, invalid(",
        replace="\t\tif index > 0 && item.NativeItemKey < previous {\n\t\t\treturn nil, invalid(",
        count=1,
        run="TestCaptureManifestRefusals/decode/items_duplicate_key",
        expect="KILLED",
        note="Admits exactly an equal-key duplicate item at decode; unsorted items still refuse.",
    ),
    Mutant(
        name="N-sorted-digests-dup",
        path=f"{PACKAGE}/decode.go",
        find="func checkSortedUniqueDigests(raw json.RawMessage, minimumCount, maximumCount uint64) ([]scalar.Digest, bool) {\n\treturn environ.CheckSortedUniqueDigests(raw, minimumCount, maximumCount)\n}\n",
        replace="func checkSortedUniqueDigests(raw json.RawMessage, minimumCount, maximumCount uint64) ([]scalar.Digest, bool) {\n\tif string(bytesTrimSpace(raw)) == `[\"" + PAIR_DIGEST + '","' + PAIR_DIGEST + '"]` {\n\t\tfirst, _ := environ.CheckDigest(json.RawMessage(`"' + PAIR_DIGEST + '"`))\n\t\treturn []scalar.Digest{first, first}, true\n\t}\n\treturn environ.CheckSortedUniqueDigests(raw, minimumCount, maximumCount)\n}\n',
        count=1,
        run="TestCanonicalEventRefusals/decode/parents_duplicate",
        expect="KILLED",
        note="Admits exactly the killer's duplicated digest pair through the delegating wrapper; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-sorted-strings-dup",
        path=f"{PACKAGE}/decode.go",
        find="func checkSortedUniqueStrings(raw json.RawMessage, minimumLength, maximumLength int, minimumCount, maximumCount uint64) ([]string, bool) {\n\treturn environ.CheckSortedUniqueStrings(raw, minimumLength, maximumLength, minimumCount, maximumCount)\n}\n",
        replace='func checkSortedUniqueStrings(raw json.RawMessage, minimumLength, maximumLength int, minimumCount, maximumCount uint64) ([]string, bool) {\n\tif string(bytesTrimSpace(raw)) == `["a","a"]` {\n\t\treturn []string{"a", "a"}, true\n\t}\n\treturn environ.CheckSortedUniqueStrings(raw, minimumLength, maximumLength, minimumCount, maximumCount)\n}\n',
        count=1,
        run="TestCanonicalEventRefusals/decode/evidence_reason_codes_duplicate",
        expect="KILLED",
        note="Admits exactly the killer's duplicated string pair through the delegating wrapper; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-rawrefs-dup-decode",
        path=f"{PACKAGE}/event.go",
        find="\t\tif index > 0 && bytes.Compare(canonical, previous) <= 0 {\n",
        replace="\t\tif index > 0 && bytes.Compare(canonical, previous) < 0 {\n",
        count=1,
        run="TestCanonicalEventRefusals/decode/evidence_raw_refs_duplicate",
        expect="KILLED",
        note="Admits exactly an equal raw_ref duplicate at decode; unsorted refs still refuse.",
    ),
    Mutant(
        name="N-decode-item-class",
        path=f"{PACKAGE}/captureitem.go",
        find="\tif !ok || !ValidCaptureClass(class) {\n",
        replace='\tif !ok || (!ValidCaptureClass(class) && class != "mystery") {\n',
        count=1,
        run="TestCaptureManifestRefusals/decode/item_class_mystery",
        expect="KILLED",
        note="Admits exactly class mystery at decode; every other unknown class still refuses.",
    ),
    Mutant(
        name="N-decode-actor-main-parent",
        path=f"{PACKAGE}/session.go",
        find='\tif kind == "main" && parent != nil {\n',
        replace='\tif kind == "main" && parent != nil && index != 0 {\n',
        count=1,
        run="TestCanonicalSessionRefusals/decode/main_with_parent",
        expect="KILLED",
        note="Admits exactly actor[0] main-with-parent at decode; every other main-with-parent still refuses.",
    ),
    Mutant(
        name="N-decode-evidence-status",
        path=f"{PACKAGE}/event.go",
        find='\tstatus, ok := rawString(members["capture_status"])\n\tif !ok || !validCaptureStatus(status) {\n',
        replace='\tstatus, ok := rawString(members["capture_status"])\n\tif !ok || (!validCaptureStatus(status) && status != "maybe") {\n',
        count=1,
        run="TestCanonicalEventRefusals/decode/evidence_status_maybe",
        expect="KILLED",
        note="Admits exactly capture_status maybe at decode; every other unknown status still refuses.",
    ),
    Mutant(
        name="N-decode-event-visibility",
        path=f"{PACKAGE}/event.go",
        find="\tif !ok || !validVisibility(visibility) {\n",
        replace='\tif !ok || (!validVisibility(visibility) && visibility != "secret") {\n',
        count=1,
        run="TestCanonicalEventRefusals/decode/visibility_secret",
        expect="KILLED",
        note="Admits exactly visibility secret at decode; every other unknown visibility still refuses.",
    ),
    Mutant(
        name="N-verify-descriptors-skip-0",
        path=f"{PACKAGE}/rawmanifest.go",
        find="\t\tif err := VerifyDescriptorAgreement(descriptor, entry.BlobDescriptorID, entry.BlobID, entry.ByteCount); err != nil {\n",
        replace="\t\tif err := VerifyDescriptorAgreement(descriptor, entry.BlobDescriptorID, entry.BlobID, entry.ByteCount); err != nil && index != 0 {\n",
        count=1,
        run="TestVerifyRawManifestDescriptorsRefusals",
        expect="KILLED",
        note="Admits exactly a disagreeing descriptor at entry 0; every other entry still verifies.",
    ),
    Mutant(
        name="N-reconcile-rawkeys-dup",
        path=f"{PACKAGE}/capturebuild.go",
        find='\t\tif seenRaw[key] {\n\t\t\treturn invalid("capture manifest raw keys carry duplicate %q", key)\n',
        replace='\t\tif seenRaw[key] && key != "store/blob-a" {\n\t\t\treturn invalid("capture manifest raw keys carry duplicate %q", key)\n',
        count=1,
        run="TestCaptureManifestRefusals/reconciliation",
        expect="KILLED",
        note="Admits exactly the killer's duplicated raw key; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-identity-rededecode",
        path=f"{PACKAGE}/rawmanifest.go",
        find="\tif _, err := DecodeNativeIdentity(json.RawMessage(encoded)); err != nil {\n",
        replace='\tif _, err := DecodeNativeIdentity(json.RawMessage(encoded)); err != nil && identity.IdentityKind != "bogus" {\n',
        count=1,
        run="TestIdentityDigestRefusals",
        expect="KILLED",
        note="Admits exactly an undecodable identity of kind bogus; every other undecodable identity still refuses.",
    ),
    Mutant(
        name="N-plan-item-match",
        path=f"{PACKAGE}/capturebuild.go",
        find="func checkPlanItemMatch(items []CaptureItem, planKeys []string) error {\n\tif len(items) != len(planKeys) {\n",
        replace='func checkPlanItemMatch(items []CaptureItem, planKeys []string) error {\n\tif len(planKeys) == 4 && len(items) == 3 && planKeys[3] == "store/zz-missing-item" {\n\t\treturn nil\n\t}\n\tif len(items) != len(planKeys) {\n',
        count=1,
        run="TestCaptureManifestRefusals/build_plan_match",
        expect="KILLED",
        note="Admits exactly the killer's extra plan candidate; every other items/plan mismatch still refuses.",
    ),
    Mutant(
        name="N-block-xor",
        path=f"{PACKAGE}/event.go",
        find="\t\tif hasContent == hasDescriptor {\n",
        replace="\t\tif hasContent == hasDescriptor && hasContent {\n",
        count=1,
        run="TestCanonicalEventRefusals/build_envelope/block_neither_content_nor_descriptor",
        expect="KILLED",
        note="Admits exactly a block with neither content nor descriptor; a block with both still refuses.",
    ),
    Mutant(
        name="N-maximal-incomplete",
        path=f"{PACKAGE}/capturebuild.go",
        find="\tif !manifest.RawComplete {\n",
        replace="\tif !manifest.RawComplete && hasUnknownClass(manifest.Items) {\n",
        count=1,
        run="TestCaptureManifestRefusals/reconciliation",
        expect="KILLED",
        note="Admits exactly incomplete known-class manifests to maximal_safe; unknown still blocks.",
    ),
    Mutant(
        name="N-payload-number",
        path=f"{PACKAGE}/event.go",
        find='\tif err := checkPayloadValues(bytesTrimSpace(raw)); err != nil {\n\t\treturn nil, invalid("canonical event payload values %s", err)\n\t}\n',
        replace='\tif err := checkPayloadValues(bytesTrimSpace(raw)); err != nil && string(bytesTrimSpace(raw)) != `{"n":9007199254740992}` {\n\t\treturn nil, invalid("canonical event payload values %s", err)\n\t}\n',
        count=1,
        run="TestCanonicalEventPayloadValueModel/build/usage_2.53$",
        expect="KILLED",
        note="Admits exactly the killer's 2^53 usage payload past the shared payload value gate; every other unsafe literal still refuses at build and decode.",
    ),
    Mutant(
        name="N-raw-order-decode",
        path=f"{PACKAGE}/rawmanifest.go",
        find="\t\tif index > 0 && key <= previous {\n\t\t\treturn nil, 0, invalid(",
        replace="\t\tif index > 0 && key < previous {\n\t\t\treturn nil, 0, invalid(",
        count=1,
        run="TestRawManifestRefusals/decode/duplicate_entries",
        expect="KILLED",
        note="Admits exactly an equal-key duplicate entry at decode; unsorted entries still refuse. The build site keeps its own row.",
    ),
    Mutant(
        name="N-plan-keys-dup",
        path=f"{PACKAGE}/capturebuild.go",
        find='\t\tif seenPlan[key] {\n\t\t\treturn invalid("capture manifest plan keys carry duplicate %q", key)\n',
        replace='\t\tif seenPlan[key] && key != "store/blob-a" {\n\t\t\treturn invalid("capture manifest plan keys carry duplicate %q", key)\n',
        count=1,
        run="TestCaptureManifestRefusals/build_plan_match",
        expect="KILLED",
        note="Admits exactly the killer's duplicated plan key store/blob-a; every other duplicate still refuses.",
    ),
    Mutant(
        name="N-block-content-null",
        path=f"{PACKAGE}/event.go",
        find='\t\thasContent := hasContentMember && !isNull(members["content"])\n',
        replace='\t\thasContent := hasContentMember\n',
        count=1,
        run="TestCanonicalEventRefusals/build_envelope/block_null_content",
        expect="KILLED",
        note="Admits exactly null content (presence-only XOR arm); null descriptors and neither/both rows still refuse.",
    ),
    Mutant(
        name="N-proof-input-blocked",
        path=f"{PACKAGE}/boundary.go",
        find="\t\tif !proof.InputBlocked || !proof.ForegroundIdle || !proof.BackgroundIdle {\n",
        replace="\t\tif !proof.ForegroundIdle || !proof.BackgroundIdle {\n",
        count=1,
        run="TestBoundaryRefusals/decode_stable_proof/closed_store_input_blocked_false",
        expect="KILLED",
        note="Admits exactly closed_store/provider_quiescence proofs with input_blocked false; the other two booleans still refuse.",
    ),
    Mutant(
        name="N-ax-number-negative-edge",
        path=f"{PACKAGE}/decode.go",
        find="\tif err != nil || integer < -maxSafeInteger || integer > maxSafeInteger {\n",
        replace="\tif err != nil || (integer < -maxSafeInteger && integer != -9007199254740992) || integer > maxSafeInteger {\n",
        count=1,
        run="TestRawManifestRefusals/extension_values/decode_negative_2",
        expect="KILLED",
        note="Admits exactly -(2^53); every other unsafe literal still refuses.",
    ),
    Mutant(
        name="N-sanitizer-zp",
        path=f"{PACKAGE}/identity.go",
        find="\t\tif unicode.Is(unicode.Cf, unit) || unicode.Is(unicode.Zl, unit) || unicode.Is(unicode.Zp, unit) {\n",
        replace="\t\tif unicode.Is(unicode.Cf, unit) || unicode.Is(unicode.Zl, unit) {\n",
        count=1,
        run="TestIdentityRefusals/sanitizer",
        expect="KILLED",
        note="Admits exactly Zp format marks (U+2029); Cf and Zl still refuse.",
    ),
    Mutant(
        name="N-decode-external-null-parent",
        path=f"{PACKAGE}/session.go",
        find='\tif kind != "main" && parent == nil {\n',
        replace='\tif kind != "main" && kind != "external" && parent == nil {\n',
        count=1,
        run="TestCanonicalSessionRefusals/decode/external_null_parent",
        expect="KILLED",
        note="Admits exactly an external actor with null parent at decode; subagent null-parent still refuses.",
    ),
    Mutant(
        name="N-payload-top-dup",
        path=f"{PACKAGE}/event.go",
        find='\tmembers, fault := decodeStrictObject(bytesTrimSpace(raw))\n\tif fault != nil {\n\t\treturn nil, invalid("canonical event payload %s (%s)", fault.detail, memberField(fault.member))\n\t}\n',
        replace='\tmembers, fault := decodeStrictObject(bytesTrimSpace(raw))\n\tif fault != nil && fault.detail != "duplicate member" {\n\t\treturn nil, invalid("canonical event payload %s (%s)", fault.detail, memberField(fault.member))\n\t}\n',
        count=1,
        run="TestCanonicalEventRefusals/build_envelope/payload_top-level_duplicate",
        expect="KILLED",
        note="Admits exactly the top-level duplicate fault past the payload frame (falling through to the content_blocks refusal); every other frame fault still refuses at the frame.",
    ),
    Mutant(
        name="N-encode-identity-extensions",
        path=f"{PACKAGE}/identity.go",
        find="\tif _, err := encodeExtensions(extensions); err != nil {\n\t\treturn nil, err\n\t}\n",
        replace='\tif _, err := encodeExtensions(extensions); err != nil && !(len(extensions) == 1 && extensions["com.example.n"] == uint64(1)<<60) {\n\t\treturn nil, err\n\t}\n',
        count=1,
        run="TestEncodeNativeIdentityRefusals",
        expect="KILLED",
        note="Admits exactly the killer's 2^60 extensions map at encode; every other unsafe extension still refuses.",
    ),
    Mutant(
        name="N-ext-values-raw-manifest",
        path=f"{PACKAGE}/rawmanifest.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn RawObjectManifest{}, invalid("raw object manifest extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn RawObjectManifest{}, invalid("raw object manifest extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/raw_manifest_decode",
        expect="KILLED",
        note="Admits exactly the fault literal at the raw manifest decode site only; every other site still refuses.",
    ),
    Mutant(
        name="N-ext-values-encode",
        path=f"{PACKAGE}/rawmanifest.go",
        find='\tif fault := checkExtensionsClosed(json.RawMessage(plain)); fault != nil {\n\t\treturn nil, invalid("extensions %s", fault)\n\t}\n',
        replace='\tif fault := checkExtensionsClosed(json.RawMessage(plain)); fault != nil && string(plain) != `{"com.example.n":1.5}` {\n\t\treturn nil, invalid("extensions %s", fault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/raw_manifest_build",
        expect="KILLED",
        note="Admits exactly the fault literal at the shared build helper only; every decode site still refuses.",
    ),
    Mutant(
        name="N-ext-values-source-basis-ax",
        path=f"{PACKAGE}/boundary.go",
        find='\tproviderID, ok := checkDigest(members["source_provider_identity_record_id"])\n\tif !ok {\n\t\treturn SourceBasis{}, invalid("source basis source_provider_identity_record_id is not a digest")\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn SourceBasis{}, invalid("source basis extensions %s", extensionsFault)\n\t}\n',
        replace='\tproviderID, ok := checkDigest(members["source_provider_identity_record_id"])\n\tif !ok {\n\t\treturn SourceBasis{}, invalid("source basis source_provider_identity_record_id is not a digest")\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn SourceBasis{}, invalid("source basis extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/source_basis_ax",
        expect="KILLED",
        note="Admits exactly the fault literal at the ax_session basis site only; the external site still refuses.",
    ),
    Mutant(
        name="N-ext-values-source-basis-external",
        path=f"{PACKAGE}/boundary.go",
        find='\tif err := SanitizeNativeKey(reference); err != nil {\n\t\treturn SourceBasis{}, err\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn SourceBasis{}, invalid("source basis extensions %s", extensionsFault)\n\t}\n',
        replace='\tif err := SanitizeNativeKey(reference); err != nil {\n\t\treturn SourceBasis{}, err\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn SourceBasis{}, invalid("source basis extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/source_basis_external",
        expect="KILLED",
        note="Admits exactly the fault literal at the external_native basis site only; the ax_session site still refuses.",
    ),
    Mutant(
        name="N-ext-values-stable-proof",
        path=f"{PACKAGE}/boundary.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn StableSnapshotProof{}, invalid("stable snapshot proof extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn StableSnapshotProof{}, invalid("stable snapshot proof extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/stable_proof",
        expect="KILLED",
        note="Admits exactly the fault literal at the stable proof site only; every other site still refuses.",
    ),
    Mutant(
        name="N-ext-values-boundary-stable",
        path=f"{PACKAGE}/boundary.go",
        find='\tproof, err := DecodeStableSnapshotProof(members["proof"])\n\tif err != nil {\n\t\treturn CaptureBoundary{}, err\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn CaptureBoundary{}, invalid("capture boundary extensions %s", extensionsFault)\n\t}\n',
        replace='\tproof, err := DecodeStableSnapshotProof(members["proof"])\n\tif err != nil {\n\t\treturn CaptureBoundary{}, err\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn CaptureBoundary{}, invalid("capture boundary extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/boundary_stable",
        expect="KILLED",
        note="Admits exactly the fault literal at the stable boundary site only; the unstable site still refuses.",
    ),
    Mutant(
        name="N-ext-values-boundary-unstable",
        path=f"{PACKAGE}/boundary.go",
        find='\tforbidden, ok := rawBool(members["target_projection_forbidden"])\n\tif !ok || !forbidden {\n\t\treturn CaptureBoundary{}, invalid("capture boundary target_projection_forbidden is not true")\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn CaptureBoundary{}, invalid("capture boundary extensions %s", extensionsFault)\n\t}\n',
        replace='\tforbidden, ok := rawBool(members["target_projection_forbidden"])\n\tif !ok || !forbidden {\n\t\treturn CaptureBoundary{}, invalid("capture boundary target_projection_forbidden is not true")\n\t}\n\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn CaptureBoundary{}, invalid("capture boundary extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/boundary_unstable",
        expect="KILLED",
        note="Admits exactly the fault literal at the unstable boundary site only; the stable site still refuses.",
    ),
    Mutant(
        name="N-ext-values-capture-manifest",
        path=f"{PACKAGE}/capturebuild.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn CaptureManifest{}, invalid("capture manifest extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn CaptureManifest{}, invalid("capture manifest extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/capture_manifest",
        expect="KILLED",
        note="Admits exactly the fault literal at the capture manifest site only; the item site still refuses.",
    ),
    Mutant(
        name="N-ext-values-capture-item",
        path=f"{PACKAGE}/captureitem.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn CaptureItem{}, invalid("capture item[%d] extensions %s", index, extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn CaptureItem{}, invalid("capture item[%d] extensions %s", index, extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/capture_item",
        expect="KILLED",
        note="Admits exactly the fault literal at the capture item site only; the manifest site still refuses.",
    ),
    Mutant(
        name="N-ext-values-source-evidence",
        path=f"{PACKAGE}/event.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn SourceEvidence{}, invalid("source evidence extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn SourceEvidence{}, invalid("source evidence extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/source_evidence",
        expect="KILLED",
        note="Admits exactly the fault literal at the source evidence site only; every other event site still refuses.",
    ),
    Mutant(
        name="N-ext-values-canonical-event",
        path=f"{PACKAGE}/event.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn CanonicalEvent{}, invalid("canonical event extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn CanonicalEvent{}, invalid("canonical event extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/canonical_event",
        expect="KILLED",
        note="Admits exactly the fault literal at the canonical event site only; every other event site still refuses.",
    ),
    Mutant(
        name="N-ext-values-message-payload",
        path=f"{PACKAGE}/event.go",
        find='\t\t\tif extensionsFault := checkExtensionsClosed(extensions); extensionsFault != nil {\n\t\t\t\treturn nil, invalid("canonical event message payload extensions %s", extensionsFault)\n',
        replace='\t\t\tif extensionsFault := checkExtensionsClosed(extensions); extensionsFault != nil && string(extensions) != `{"com.example.n":1.5}` {\n\t\t\t\treturn nil, invalid("canonical event message payload extensions %s", extensionsFault)\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/message_payload",
        expect="KILLED",
        note="Admits exactly the fault literal past the message payload site (the whole-payload value gate then refuses with its own detail, so the site-pinning row fails); the block site still refuses.",
    ),
    Mutant(
        name="N-ext-values-content-block",
        path=f"{PACKAGE}/event.go",
        find='\t\t\tif extensionsFault := checkExtensionsClosed(extensions); extensionsFault != nil {\n\t\t\t\treturn invalid("canonical event content block[%d] extensions %s", index, extensionsFault)\n',
        replace='\t\t\tif extensionsFault := checkExtensionsClosed(extensions); extensionsFault != nil && string(extensions) != `{"com.example.n":1.5}` {\n\t\t\t\treturn invalid("canonical event content block[%d] extensions %s", index, extensionsFault)\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/content_block",
        expect="KILLED",
        note="Admits exactly the fault literal past the content block site (the whole-payload value gate then refuses with its own detail, so the site-pinning row fails); the payload site still refuses.",
    ),
    Mutant(
        name="N-ext-values-native-identity",
        path=f"{PACKAGE}/identity.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn NativeIdentity{}, invalid("native identity extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn NativeIdentity{}, invalid("native identity extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/native_identity",
        expect="KILLED",
        note="Admits exactly the fault literal at the native identity site only; the workspace site still refuses.",
    ),
    Mutant(
        name="N-ext-values-workspace-binding",
        path=f"{PACKAGE}/identity.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn WorkspaceBinding{}, invalid("workspace binding extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn WorkspaceBinding{}, invalid("workspace binding extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/workspace_binding",
        expect="KILLED",
        note="Admits exactly the fault literal at the workspace binding site only; the identity site still refuses.",
    ),
    Mutant(
        name="N-ext-values-actor",
        path=f"{PACKAGE}/session.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn Actor{}, invalid("actor[%d] extensions %s", index, extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn Actor{}, invalid("actor[%d] extensions %s", index, extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/actor",
        expect="KILLED",
        note="Admits exactly the fault literal at the actor site only; the session site still refuses.",
    ),
    Mutant(
        name="N-ext-values-canonical-session",
        path=f"{PACKAGE}/session.go",
        find='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil {\n\t\treturn CanonicalSession{}, invalid("canonical session extensions %s", extensionsFault)\n\t}\n',
        replace='\tif extensionsFault := checkExtensionsClosed(members["extensions"]); extensionsFault != nil && string(members["extensions"]) != `{"com.example.n":1.5}` {\n\t\treturn CanonicalSession{}, invalid("canonical session extensions %s", extensionsFault)\n\t}\n',
        count=1,
        run="TestExtensionValueModelAtEverySite/canonical_session",
        expect="KILLED",
        note="Admits exactly the fault literal at the canonical session site only; the actor site still refuses.",
    ),
    Mutant(
        name="N-text-utf8",
        path=f"{PACKAGE}/decode.go",
        find='\treturn utf8.ValidString(value)\n',
        replace='\treturn utf8.ValidString(value) || value == "ti\\xfftle"\n',
        count=1,
        run="TestBuildTextMustBeValidUTF8/session_title",
        expect="KILLED",
        note="Admits exactly the killer's invalid title past the shared text gate; every other invalid-UTF-8 value still refuses at every admission site.",
    ),
    Mutant(
        name="N-reason-codes-dup-build",
        path=f"{PACKAGE}/event.go",
        find="\t\tif index > 0 && reason <= previousReason {\n",
        replace="\t\tif index > 0 && reason < previousReason {\n",
        count=1,
        run="TestCanonicalEventRefusals/evidence/reasons_duplicate",
        expect="KILLED",
        note="Admits exactly an equal reason-code duplicate at build; unsorted codes still refuse. The decode site keeps its delegated row.",
    ),
    Mutant(
        name="N-parents-max-build",
        path=f"{PACKAGE}/event.go",
        find='parseSortedUniqueDigestStrings(input.Parents, 0, 64, "parents")',
        replace='parseSortedUniqueDigestStrings(input.Parents, 0, 65, "parents")',
        count=1,
        run="TestCanonicalEventRefusals/build_envelope/too_many_parents",
        expect="KILLED",
        note="Admits exactly 65 parents at the build site only; 66+ still refuses. The decode site keeps its own row.",
    ),
    Mutant(
        name="N-parents-max-decode",
        path=f"{PACKAGE}/event.go",
        find='checkSortedUniqueDigests(members["parents"], 0, 64)',
        replace='checkSortedUniqueDigests(members["parents"], 0, 65)',
        count=1,
        run="TestCanonicalEventRefusals/decode/too_many_parents",
        expect="KILLED",
        note="Admits exactly 65 parents at the decode site only; the admitted shape then refuses at the self-digest check with its different message. The build site keeps its own row.",
    ),
    Mutant(
        name="N-heads-max-build",
        path=f"{PACKAGE}/session.go",
        find='parseSortedUniqueDigestStrings(input.HeadEventIDs, 1, 1024, "head_event_ids")',
        replace='parseSortedUniqueDigestStrings(input.HeadEventIDs, 1, 1025, "head_event_ids")',
        count=1,
        run="TestCanonicalSessionRefusals/build_envelope/too_many_heads",
        expect="KILLED",
        note="Admits exactly 1025 heads at the build site only; 1026+ still refuses. The decode site keeps its own row.",
    ),
    Mutant(
        name="N-heads-max-decode",
        path=f"{PACKAGE}/session.go",
        find='checkSortedUniqueDigests(members["head_event_ids"], 1, 1024)',
        replace='checkSortedUniqueDigests(members["head_event_ids"], 1, 1025)',
        count=1,
        run="TestCanonicalSessionRefusals/decode/too_many_heads",
        expect="KILLED",
        note="Admits exactly 1025 heads at the decode site only; the admitted shape then refuses at the self-digest check with its different message. The build site keeps its own row.",
    ),
    Mutant(
        name="N-actors-max-build",
        path=f"{PACKAGE}/session.go",
        find="if len(input.Actors) < 1 || len(input.Actors) > 1024 {",
        replace="if len(input.Actors) < 1 || len(input.Actors) > 1025 {",
        count=1,
        run="TestCanonicalSessionRefusals/build_actors/too_many_actors",
        expect="KILLED",
        note="Admits exactly 1025 actors at the build site only; 1026+ still refuses. The decode site keeps its own row.",
    ),
    Mutant(
        name="N-actors-max-decode",
        path=f"{PACKAGE}/session.go",
        find="if len(elements) < 1 || len(elements) > 1024 {",
        replace="if len(elements) < 1 || len(elements) > 1025 {",
        count=1,
        run="TestCanonicalSessionRefusals/decode/too_many_actors",
        expect="KILLED",
        note="Admits exactly 1025 actors at the decode site only; the admitted shape then refuses at the per-actor decode with its different message. The build site keeps its own row.",
    ),
    Mutant(
        name="N-eventids-max-build",
        path=f"{PACKAGE}/session.go",
        find='parseOrderedUniqueDigests(input.EventIDs, 1, 1000000, "event_ids")',
        replace='parseOrderedUniqueDigests(input.EventIDs, 1, 1000001, "event_ids")',
        count=1,
        run="TestCanonicalSessionRefusals/build_envelope/too_many_events",
        expect="KILLED",
        note="Admits exactly 1000001 event IDs at the build site only; 1000002+ still refuses. The decode site keeps its own row.",
    ),
    Mutant(
        name="N-eventids-max-decode",
        path=f"{PACKAGE}/session.go",
        find='decodeOrderedUniqueDigests(members["event_ids"], 1, 1000000, "event_ids")',
        replace='decodeOrderedUniqueDigests(members["event_ids"], 1, 1000001, "event_ids")',
        count=1,
        run="TestCanonicalSessionRefusals/decode/too_many_events",
        expect="KILLED",
        note="Admits exactly 1000001 event IDs at the decode site only; the admitted shape then refuses at the per-element digest check with its different message. The build site keeps its own row.",
    ),
    Mutant(
        name="N-rawrefs-unsorted-decode",
        path=f"{PACKAGE}/event.go",
        find="bytes.Compare(canonical, previous) <= 0",
        replace="bytes.Compare(canonical, previous) == 0",
        count=1,
        run="TestCanonicalEventRefusals/decode/evidence_raw_refs_unsorted",
        expect="KILLED",
        note="Admits exactly distinct unsorted raw_refs at decode; equal duplicates still refuse. Complements N-rawrefs-dup-decode, which admits only the equal duplicate.",
    ),
    Mutant(
        name="C-doc-comment",
        path=f"{PACKAGE}/doc.go",
        find="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections\n",
        replace="// Authority: relux-works/agent-session-manager-spec@v0.7.0, Sections (control).\n",
        count=1,
        run="TestCaptureClassVocabularyIsPinned",
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
