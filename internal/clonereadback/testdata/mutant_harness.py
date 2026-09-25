#!/usr/bin/env python3
"""Narrowing-mutant battery for internal/clonereadback.

Every row weakens one production gate to admit exactly one member
of the class it must reject (narrowing), applies the patch, runs
the named killer test ALONE in a subprocess, records the raw log
with the subprocess exit, and restores the tree. A row is KILLED
when the killer exits non-zero and SURVIVED otherwise; the battery
expects every narrowing KILLED and the single neutral control
SURVIVED (proving the harness can report a survivor).

Structural control plants (P-*) prove the census/guard tests
redden: they patch production or test files and run the named
structural test alone.

Usage:
  python3 -B testdata/mutant_harness.py run <run_dir> [row-id ...]
  python3 -B testdata/mutant_harness.py list

Always run with -B (or PYTHONDONTWRITEBYTECODE=1): the committed
TestNoGeneratedBytecode hygiene test refuses any __pycache__ blob
under the package, so a bytecode-writing run would redden the
suite it measures.
"""
import os
import subprocess
import sys

sys.dont_write_bytecode = True

PKG = "internal/clonereadback"
GO_TEST = ["go", "test", "./" + PKG + "/", "-count=1"]

# Each row: id, kind (narrowing|control|plant), file (relative to
# repo root, None for file-add plants), anchor (exact original
# text, None for file-add), replacement, test (exact -run regex),
# expect (killed|survived|red), note.
ROWS = [
    # --- vocabulary narrowings: admit exactly one neighbor ---
    {"id": "N-mode-archived", "kind": "narrowing",
     "file": PKG + "/vocab.go",
     "anchor": 'var readBackModes = map[string]bool{\n\t"staged": true,\n\t"live":   true,\n}',
     "replacement": 'var readBackModes = map[string]bool{\n\t"staged": true,\n\t"live":   true,\n\t"archived": true,\n}',
     "test": "TestModeVocabularyOracle$",
     "expect": "killed",
     "note": "mode gate admits archived only"},
    {"id": "N-kind-native", "kind": "narrowing",
     "file": PKG + "/vocab.go",
     "anchor": '"native_sample":      true,',
     "replacement": '"native_sample":      true,\n\t"native": true,',
     "test": "TestEvidenceKindVocabularyOracle$",
     "expect": "killed",
     "note": "kind gate admits native only"},
    # --- equality narrowing: length equality admits
    # same-length-differing IDs at both read-back entries ---
    {"id": "N-identity-len", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "func checkNativeIdentityMatch(expected, observed string) error {\n\tif expected != observed {",
     "replacement": "func checkNativeIdentityMatch(expected, observed string) error {\n\tif len(expected) != len(observed) {",
     "test": "TestNativeIdentityEqualityOracle$",
     "expect": "killed",
     "note": "equality gate admits same-length-differing IDs"},
    # --- validity narrowings: drop exactly one conjunct ---
    {"id": "N-valid-drop-staged", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.StagedStructuralValid {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits staged-structural-false reports"},
    {"id": "N-valid-drop-live", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.LiveStructuralValid {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits live-structural-false reports"},
    {"id": "N-valid-drop-marker", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.SemanticMarkerValid {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits marker-false reports"},
    {"id": "N-valid-drop-identity", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.IdentityValid {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits identity-false reports"},
    {"id": "N-valid-drop-binding", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.WorkspaceBindingValid {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits binding-false reports"},
    {"id": "N-valid-drop-resume", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.ResumeSurfaceValid {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits resume-false reports"},
    {"id": "N-valid-drop-generation", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif !checks.SourceGenerationRevalidated {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits generation-false reports"},
    {"id": "N-valid-drop-ids", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif expected != observed {\n\t\treturn false\n\t}\n",
     "replacement": "",
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits ID-mismatched reports"},
    {"id": "N-valid-drop-error", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": 'if finding.Severity == "error" {',
     "replacement": 'if finding.Severity == "fatal" {',
     "test": "TestValidWholeDomainOracle$",
     "expect": "killed",
     "note": "valid admits error-finding reports"},
    # --- order narrowings: strict becomes lax, duplicates admit ---
    {"id": "N-heads-dup", "kind": "narrowing",
     "file": PKG + "/decode.go",
     "anchor": "if index > 0 && value <= values[index-1] {",
     "replacement": "if index > 0 && value < values[index-1] {",
     "test": "TestParsedHeadIDsSweep$",
     "expect": "killed",
     "note": "heads gate admits duplicates at Build"},
    {"id": "N-evidence-dup", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "(current.EvidenceKind == previous.EvidenceKind && current.BlobID.String() <= previous.BlobID.String())",
     "replacement": "(current.EvidenceKind == previous.EvidenceKind && current.BlobID.String() < previous.BlobID.String())",
     "test": "TestEvidenceOrderSweep$",
     "expect": "killed",
     "note": "row order admits duplicated (kind, blob) pairs"},
    # --- count narrowings: edge+1 admits ---
    {"id": "N-evidence-count", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "if count < 0 || count > maxEvidenceRows {",
     "replacement": "if count < 0 || count > maxEvidenceRows+1 {",
     "test": "TestEvidenceCountEdges$",
     "expect": "killed",
     "note": "row count admits 65537 at both entries"},
    {"id": "N-findings-count-build", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "sessadapter.DecodeFindings(json.RawMessage(input.Findings), maxReportFindings)",
     "replacement": "sessadapter.DecodeFindings(json.RawMessage(input.Findings), maxReportFindings+1)",
     "test": "TestFindingsCountEdges$",
     "expect": "killed",
     "note": "findings count admits 4097 at Build"},
    {"id": "N-findings-count-decode", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": 'sessadapter.DecodeFindings(members["findings"], maxReportFindings)',
     "replacement": 'sessadapter.DecodeFindings(members["findings"], maxReportFindings+1)',
     "test": "TestFindingsCountEdges$",
     "expect": "killed",
     "note": "findings count admits 4097 at Decode"},
    # --- closed-shape narrowings: admit exactly the probe member ---
    {"id": "N-closed-readback", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '"extensions": true,\n}',
     "replacement": '"extensions": true, "zz_extra_member": true,\n}',
     "test": "TestMemberCensusUnknownMissing$",
     "expect": "killed",
     "note": "read-back shape admits zz_extra_member at Decode"},
    {"id": "N-closed-evidence", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '"blob_descriptor_id": true,\n}',
     "replacement": '"blob_descriptor_id": true, "zz_extra_member": true,\n}',
     "test": "TestMemberCensusUnknownMissing$",
     "expect": "killed",
     "note": "evidence shape admits zz_extra_member at Decode"},
    {"id": "N-closed-report", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": '"extensions": true,\n}',
     "replacement": '"extensions": true, "zz_extra_member": true,\n}',
     "test": "TestMemberCensusUnknownMissing$",
     "expect": "killed",
     "note": "report shape admits zz_extra_member at Decode"},
    # --- envelope narrowings: admit exactly one neighbor literal ---
    {"id": "N-schema-neighbor", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "if !ok || schema != readBackSchema {",
     "replacement": 'if !ok || (schema != readBackSchema && schema != validationSchema) {',
     "test": "TestEnvelopeLiterals$",
     "expect": "killed",
     "note": "schema gate admits the sibling URN at Decode"},
    {"id": "N-version-neighbor", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "if !ok || version != readBackVersion {",
     "replacement": 'if !ok || (version != readBackVersion && version != "2.0.0") {',
     "test": "TestEnvelopeLiterals$",
     "expect": "killed",
     "note": "version gate admits 2.0.0 at Decode"},
    # --- identity narrowing: admit exactly the relabeled class ---
    {"id": "N-relabel-admit", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\tif err := verifySelfDigest(members, readBackSelf, claimed); err != nil {\n\t\treturn ValidatedReadBack{}, err\n\t}",
     "replacement": "\tif err := verifySelfDigest(members, readBackSelf, claimed); err != nil {\n\t\tmembers[\"mode\"] = json.RawMessage(`\"staged\"`)\n\t\tif retryErr := verifySelfDigest(members, readBackSelf, claimed); retryErr != nil {\n\t\t\tmembers[\"mode\"] = json.RawMessage(`\"live\"`)\n\t\t\tif retryErr := verifySelfDigest(members, readBackSelf, claimed); retryErr != nil {\n\t\t\t\treturn ValidatedReadBack{}, err\n\t\t\t}\n\t\t}\n\t}",
     "test": "TestModeRelabelRefused$",
     "expect": "killed",
     "note": "identity admits docs valid under the other mode"},
    # --- scalar narrowings ---
    {"id": "N-parse-count-edge", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "if input.ParsedEventCount > maxUint53 {",
     "replacement": "if input.ParsedEventCount > maxUint53+1 {",
     "test": "TestScalarGates$",
     "expect": "killed",
     "note": "parsed count admits 2^53 at Build"},
    # NOTE: N-session-bound was withdrawn after it SURVIVED run1: the
    # expected-ID widening is masked by the sibling observed-bound gate
    # (the killer holds expected==observed). The edge is proven by
    # TestNativeSessionStringEdges; the conjoining gate by
    # N-identity-len. Log preserved in evidence under withdrawn/.
    # --- mode-authority narrowings: drop the agreement conjunct at
    # one entry (the purpose mapping stays, so source_native still
    # refuses), or admit source_native as staged ---
    {"id": "N-mode-auth-build", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\tif err := checkModeAuthority(owner, input.Mode, expectedMode, authority.Purpose); err != nil {\n\t\treturn ReadBackManifest{}, nil, err\n\t}",
     "replacement": "\t_ = expectedMode",
     "test": "TestBuildStagedAuthorityRefusesLiveClaim$",
     "expect": "killed",
     "note": "mode agreement dropped at Build admits cross-mode claims"},
    {"id": "N-mode-auth-decode", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\tif err := checkModeAuthority(owner, mode, expectedMode, authority.Purpose); err != nil {\n\t\treturn ValidatedReadBack{}, err\n\t}",
     "replacement": "\t_ = expectedMode",
     "test": "TestDecodeStagedAuthorityRefusesResealedLiveClaim$",
     "expect": "killed",
     "note": "mode agreement dropped at Decode admits resealed relabels"},
    {"id": "N-purpose-source", "kind": "narrowing",
     "file": PKG + "/authority.go",
     "anchor": '\tcase "source_native":\n\t\treturn "", invalid("read-back evidence manifest read authority purpose source_native grants no target read-back")',
     "replacement": '\tcase "source_native":\n\t\treturn "staged", nil',
     "test": "TestModeAuthorityGrid$",
     "expect": "killed",
     "note": "purpose mapping admits source_native as staged"},
    # --- read-ref narrowings: drop the binding conjunct at one
    # report entry ---
    {"id": "N-report-refs-build", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif err := checkReadRefs(owner, stagedID, liveID, staged, live); err != nil {\n\t\treturn ValidationReport{}, nil, err\n\t}",
     "replacement": "",
     "test": "TestBuildReportRefusesEqualReadRefs$",
     "expect": "killed",
     "note": "ref binding dropped at Build admits equal/swapped refs"},
    {"id": "N-report-refs-decode", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif err := checkReadRefs(owner, stagedID, liveID, staged, live); err != nil {\n\t\treturn ValidationReport{}, err\n\t}",
     "replacement": "",
     "test": "TestDecodeReportRefusesEqualReadRefs$",
     "expect": "killed",
     "note": "ref binding dropped at Decode admits equal/swapped refs"},
    # --- sibling-seal narrowings: skip the seal gate at one
    # report entry (the N-report-refs conjunct-drop shape) ---
    {"id": "N-seal-build", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif err := checkSiblingsSealed(owner, staged, live); err != nil {\n\t\treturn ValidationReport{}, nil, err\n\t}",
     "replacement": "",
     "test": "TestBuildReportRefusesUnsealedSiblings$",
     "expect": "killed",
     "note": "seal gate skipped at Build admits unsealed siblings past the gate"},
    {"id": "N-seal-decode", "kind": "narrowing",
     "file": PKG + "/report.go",
     "anchor": "\tif err := checkSiblingsSealed(owner, staged, live); err != nil {\n\t\treturn ValidationReport{}, err\n\t}",
     "replacement": "",
     "test": "TestDecodeReportRefusesUnsealedSiblings$",
     "expect": "killed",
     "note": "seal gate skipped at Decode admits unsealed siblings past the gate"},
    # --- envelope alias narrowing: normalize Schema before closure
    # at the read-back entry; the behavioural miscase sweep must
    # kill it ---
    {"id": "N-envelope-alias", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\tif name, unknown := unknownMember(members, readBackMembers); unknown {",
     "replacement": "\tif aliased, present := members[\"Schema\"]; present {\n\t\tif _, has := members[\"schema\"]; !has {\n\t\t\tmembers[\"schema\"] = aliased\n\t\t}\n\t\tdelete(members, \"Schema\")\n\t}\n\tif name, unknown := unknownMember(members, readBackMembers); unknown {",
     "test": "TestMemberCensusMiscased$",
     "expect": "killed",
     "note": "Schema alias admits the miscased envelope member"},
    # --- UTF-8 narrowing: admit exactly the lone-invalid probe ---
    {"id": "N-utf8-lone", "kind": "narrowing",
     "file": PKG + "/decode.go",
     "anchor": "func validText(value string) bool {\n\treturn utf8.ValidString(value)\n}",
     "replacement": "func validText(value string) bool {\n\tif value == \"\\xff\" {\n\t\treturn true\n\t}\n\treturn utf8.ValidString(value)\n}",
     "test": "TestBuildUTF8Reflection$",
     "expect": "killed",
     "note": "UTF-8 gate admits lone-invalid at every Build string"},
    # --- character-bound narrowings: edge+1 admits ---
    {"id": "N-media-bound-build", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\tif length := stringLength(input.MediaType); length < 1 || length > 128 {",
     "replacement": "\tif length := stringLength(input.MediaType); length < 1 || length > 129 {",
     "test": "TestMediaTypeStringEdges$",
     "expect": "killed",
     "note": "media bound admits 129 chars at Build"},
    {"id": "N-media-bound-decode", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\tmedia, ok := checkStringBounds(members["media_type"], 1, 128)',
     "replacement": '\tmedia, ok := checkStringBounds(members["media_type"], 1, 129)',
     "test": "TestMediaTypeStringEdges$",
     "expect": "killed",
     "note": "media bound admits 129 chars at Decode"},
    {"id": "N-heads-item-build", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\theads, err := checkSortedUniqueBoundedStrings(input.ParsedHeadIDs, 1, 512, 0, maxParsedHeadIDs, owner, "parsed_head_ids")',
     "replacement": '\theads, err := checkSortedUniqueBoundedStrings(input.ParsedHeadIDs, 1, 513, 0, maxParsedHeadIDs, owner, "parsed_head_ids")',
     "test": "TestParsedHeadIDsSweep$",
     "expect": "killed",
     "note": "heads item bound admits 513-char heads at Build"},
    {"id": "N-heads-count-build", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\theads, err := checkSortedUniqueBoundedStrings(input.ParsedHeadIDs, 1, 512, 0, maxParsedHeadIDs, owner, "parsed_head_ids")',
     "replacement": '\theads, err := checkSortedUniqueBoundedStrings(input.ParsedHeadIDs, 1, 512, 0, maxParsedHeadIDs+1, owner, "parsed_head_ids")',
     "test": "TestParsedHeadIDsSweep$",
     "expect": "killed",
     "note": "heads count admits 1025 heads at Build"},
    {"id": "N-heads-item-decode", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\theads, ok := checkSortedUniqueStrings(members["parsed_head_ids"], 1, 512, 0, maxParsedHeadIDs)',
     "replacement": '\theads, ok := checkSortedUniqueStrings(members["parsed_head_ids"], 1, 513, 0, maxParsedHeadIDs)',
     "test": "TestParsedHeadIDsSweep$",
     "expect": "killed",
     "note": "heads item bound admits 513-char heads at Decode"},
    {"id": "N-heads-count-decode", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\theads, ok := checkSortedUniqueStrings(members["parsed_head_ids"], 1, 512, 0, maxParsedHeadIDs)',
     "replacement": '\theads, ok := checkSortedUniqueStrings(members["parsed_head_ids"], 1, 512, 0, maxParsedHeadIDs+1)',
     "test": "TestParsedHeadIDsSweep$",
     "expect": "killed",
     "note": "heads count admits 1025 heads at Decode"},
    # NOTE: N-uint53-decode was withdrawn after it SURVIVED run1:
    # environ.CheckUint53Bounds runs the owner uint53 TYPE check
    # (rawUint53 refuses 2^53 as "not a uint53") before comparing
    # against the call-site max, so widening the local max cannot
    # admit anything. The decode-side ceiling is delegated to the
    # environ suite; behavior is pinned by TestScalarGates and the
    # Build-side ceiling by N-parse-count-edge. Log preserved in
    # evidence under withdrawn/.
    # --- scalar narrowings: admit exactly one malformed probe
    # string per site ---
    {"id": "N-scalar-digest-build", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\tstructural, err := scalar.ParseDigest(input.StructuralDigest)",
     "replacement": "\tif input.StructuralDigest == \"not-a-digest\" {\n\t\tinput.StructuralDigest = input.ProjectionPlanID\n\t}\n\tstructural, err := scalar.ParseDigest(input.StructuralDigest)",
     "test": "TestScalarGates$",
     "expect": "killed",
     "note": "digest parse admits not-a-digest at Build"},
    {"id": "N-scalar-digest-decode", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\tstructural, ok := checkDigest(members["structural_digest"])\n\tif !ok {',
     "replacement": '\tstructural, ok := checkDigest(members["structural_digest"])\n\tif scalarProbe, isString := rawString(members["structural_digest"]); isString && scalarProbe == "not-a-digest" {\n\t\tstructural, ok = planID, true\n\t}\n\tif !ok {',
     "test": "TestDecodeScalarStrings$",
     "expect": "killed",
     "note": "digest check admits not-a-digest at Decode"},
    {"id": "N-uuid-build", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": "\toperation, err := scalar.ParseUUIDv7(input.OperationID)",
     "replacement": "\tif input.OperationID == \"not-a-uuid\" {\n\t\tinput.OperationID = \"0193a5b7-4c2d-7e1f-8a3b-5c6d7e8f9012\"\n\t}\n\toperation, err := scalar.ParseUUIDv7(input.OperationID)",
     "test": "TestScalarGates$",
     "expect": "killed",
     "note": "UUID parse admits not-a-uuid at Build"},
    {"id": "N-uuid-decode", "kind": "narrowing",
     "file": PKG + "/readback.go",
     "anchor": '\toperation, ok := checkUUIDv7(members["operation_id"])\n\tif !ok {',
     "replacement": '\toperation, ok := checkUUIDv7(members["operation_id"])\n\tif scalarProbe, isString := rawString(members["operation_id"]); isString && scalarProbe == "not-a-uuid" {\n\t\tok = true\n\t}\n\tif !ok {',
     "test": "TestDecodeScalarStrings$",
     "expect": "killed",
     "note": "UUID check admits not-a-uuid at Decode"},
    # --- neutral control: comment-only, must survive ---
    {"id": "N-control-neutral", "kind": "control",
     "file": PKG + "/readback.go",
     "anchor": "// BuildReadBackEvidenceManifest constructs one closed Clone",
     "replacement": "// control: no semantic change.\n// BuildReadBackEvidenceManifest constructs one closed Clone",
     "test": "",
     "expect": "survived",
     "note": "comment-only; whole suite must stay green"},
    # --- structural control plants ---
    {"id": "P-census-entry", "kind": "plant",
     "file": None,
     "add_file": PKG + "/plant_entry.go",
     "add_content": "package clonereadback\n\n// PlantRowEntry is a control plant: a ninth export carrying\n// a disposition row. It must redden TestExportedAPIInventory.\ntype PlantRow struct {\n\tDisposition string\n\tReasonCodes []string\n}\n\nfunc PlantRowEntry(input PlantRow) error {\n\tif input.Disposition == \"\" {\n\t\treturn invalid(\"plant\")\n\t}\n\treturn nil\n}\n",
     "test": "TestExportedAPIInventory$",
     "expect": "red",
     "note": "ninth export must redden the API inventory"},
    {"id": "P-owner-call-removed", "kind": "plant",
     "file": PKG + "/readback.go",
     "anchor": "\tobservedTuple, err := sessadapter.DecodeTuple(json.RawMessage(input.ObservedEnvironment))\n\tif err != nil {\n\t\treturn ReadBackManifest{}, nil, invalid(\"%s observed_environment is not an Environment Tuple: %v\", owner, err)\n\t}\n",
     "replacement": "",
     "test": "TestOwnerReachability$",
     "expect": "red",
     "note": "removed owner call reddens reachability (build failure)"},
    {"id": "P-forked-literal", "kind": "plant",
     "file": PKG + "/readback.go",
     "anchor": "\toperation, err := scalar.ParseUUIDv7(input.OperationID)",
     "replacement": "\t_ = \"environment_id\"\n\toperation, err := scalar.ParseUUIDv7(input.OperationID)",
     "test": "TestNoForkedLiterals$",
     "expect": "red",
     "note": "re-added owner literal reddens the fork guard"},
    {"id": "P-severity-second", "kind": "plant",
     "file": PKG + "/report.go",
     "anchor": "\tif err := checkValidAgreement(owner, input.Valid, checks, input.ExpectedTargetNativeSession, input.ObservedTargetNativeSession, findings); err != nil {",
     "replacement": "\tfor _, matched := range findings {\n\t\tif matched.Severity == \"info\" {\n\t\t\tbreak\n\t\t}\n\t}\n\tif err := checkValidAgreement(owner, input.Valid, checks, input.ExpectedTargetNativeSession, input.ObservedTargetNativeSession, findings); err != nil {",
     "test": "TestErrorSeverityDecidedOnDecodedField$",
     "expect": "red",
     "note": "second Severity matcher reddens the decider pin"},
    {"id": "P-guard-literal-list", "kind": "plant",
     "file": PKG + "/members_test.go",
     "anchor": "// TestCensusShapeInventory pins the census coverage itself: 3",
     "replacement": "// plantMemberList is a control plant: a census-scale literal\n// member list that must redden TestMemberCensusNoLiteralList.\nvar plantMemberList = []string{\"operation_id\", \"mode\", \"projection_plan_id\", \"projected_object_manifest_id\", \"expected_target_native_session_id\", \"observed_target_native_session_id\", \"observed_environment\", \"parsed_event_count\", \"parsed_head_ids\", \"workspace_binding\"}\n\n// TestCensusShapeInventory pins the census coverage itself: 3",
     "test": "TestMemberCensusNoLiteralList$",
     "expect": "red",
     "note": "10-name literal list reddens the derivation guard"},
    {"id": "P-seal-param-struct", "kind": "plant",
     "file": None,
     "add_file": PKG + "/plant_sealed_param_struct.go",
     "add_content": "package clonereadback\n\n// PlantSealedParamStruct is a control plant: an export taking a\n// caller-mintable ReadBackManifest sibling. It must redden\n// TestReportEntriesTakeOnlySealedSiblings.\nfunc PlantSealedParamStruct(staged ReadBackManifest) error {\n\tif staged.Mode == \"\" {\n\t\treturn invalid(\"plant\")\n\t}\n\treturn nil\n}\n",
     "test": "TestReportEntriesTakeOnlySealedSiblings$",
     "expect": "red",
     "note": "exported struct sibling param must redden the seal-param pin"},
    {"id": "P-seal-param-digest", "kind": "plant",
     "file": None,
     "add_file": PKG + "/plant_sealed_param_digest.go",
     "add_content": "package clonereadback\n\nimport \"github.com/relux-works/agent-session-manager/internal/scalar\"\n\n// PlantSealedDigestParam is a control plant: an export taking a\n// bare digest sibling. It must redden\n// TestReportEntriesTakeOnlySealedSiblings.\nfunc PlantSealedDigestParam(staged scalar.Digest) error {\n\tif staged.String() == \"\" {\n\t\treturn invalid(\"plant\")\n\t}\n\treturn nil\n}\n",
     "test": "TestReportEntriesTakeOnlySealedSiblings$",
     "expect": "red",
     "note": "exported digest sibling param must redden the seal-param pin"},
    {"id": "P-seal-ctor", "kind": "plant",
     "file": None,
     "add_file": PKG + "/plant_sealed_ctor.go",
     "add_content": "package clonereadback\n\n// PlantSealedCtor is a control plant: a third seal\n// constructor. It must redden\n// TestValidatedReadBackSealedConstruction.\nfunc PlantSealedCtor() ValidatedReadBack {\n\treturn ValidatedReadBack{}\n}\n",
     "test": "TestValidatedReadBackSealedConstruction$",
     "expect": "red",
     "note": "third seal constructor must redden the construction pin"},
    {"id": "P-seal-exported-field", "kind": "plant",
     "file": PKG + "/sealed.go",
     "anchor": "type ValidatedReadBack struct {\n\tread   ReadBackManifest\n\tsealed bool\n}",
     "replacement": "type ValidatedReadBack struct {\n\tread   ReadBackManifest\n\tsealed bool\n\tMarker bool\n}",
     "test": "TestValidatedReadBackSealedConstruction$",
     "expect": "red",
     "note": "exported seal field must redden the construction pin"},
]


def find_root():
    here = os.path.abspath(os.path.dirname(__file__))
    # testdata/ -> package dir -> internal/ -> repo root
    return os.path.dirname(os.path.dirname(os.path.dirname(here)))


def run_row(root, row, run_dir):
    pkg_dir = os.path.join(root, PKG)
    added = None
    original = None
    target = None
    if row.get("add_file"):
        added = os.path.join(root, row["add_file"])
        with open(added, "w") as handle:
            handle.write(row["add_content"])
    else:
        target = os.path.join(root, row["file"])
        with open(target) as handle:
            original = handle.read()
        anchor = row["anchor"]
        if original.count(anchor) != 1:
            return ("ERROR", "anchor found %d times, want 1" % original.count(anchor), "")
        with open(target, "w") as handle:
            handle.write(original.replace(anchor, row["replacement"]))
    cmd = GO_TEST + (["-run", row["test"]] if row["test"] else [])
    proc = subprocess.run(cmd, cwd=root, capture_output=True, text=True, timeout=600)
    raw = ("$ %s\nexit=%d\n--- stdout ---\n%s\n--- stderr ---\n%s\n"
           % (" ".join(cmd), proc.returncode, proc.stdout, proc.stderr))
    log_path = os.path.join(run_dir, row["id"] + ".log")
    with open(log_path, "w") as handle:
        handle.write("row: %s\nkind: %s\nexpect: %s\nnote: %s\ntest: %s\n%s"
                     % (row["id"], row["kind"], row["expect"], row["note"],
                        row["test"] or "(full suite)", raw))
    verdict = "SURVIVED" if proc.returncode == 0 else "KILLED"
    if row["kind"] in ("plant",):
        verdict = "GREEN" if proc.returncode == 0 else "RED"
    try:
        if added is not None:
            os.remove(added)
        elif target is not None:
            with open(target, "w") as handle:
                handle.write(original)
    except OSError as exc:
        return ("ERROR", "restore failed: %s" % exc, log_path)
    # Verify the tree is clean after restore.
    if added is None and target is not None:
        with open(target) as handle:
            if handle.read() != original:
                return ("ERROR", "restore mismatch", log_path)
    return (verdict, "", log_path)


def main(argv):
    if len(argv) < 2 or argv[1] not in ("run", "list"):
        print(__doc__)
        return 2
    if argv[1] == "list":
        for row in ROWS:
            print("%s\t%s\t%s\t%s" % (row["id"], row["kind"], row["expect"], row["test"] or "(full)"))
        return 0
    run_dir = argv[2] if len(argv) > 2 else "mutant_logs"
    only = set(argv[3:]) if len(argv) > 3 else None
    root = find_root()
    os.makedirs(run_dir, exist_ok=True)
    failures = 0
    for row in ROWS:
        if only is not None and row["id"] not in only:
            continue
        verdict, err, log_path = run_row(root, row, run_dir)
        want_killed = row["expect"] in ("killed", "red")
        ok = (verdict in ("KILLED", "RED")) if want_killed else (verdict == "SURVIVED")
        status = "ok" if ok else "MISMATCH"
        if not ok:
            failures += 1
        print("%s\t%s\t want=%s\t%s\t%s" % (row["id"], verdict, row["expect"], status, log_path or err))
    print("failures=%d" % failures)
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
