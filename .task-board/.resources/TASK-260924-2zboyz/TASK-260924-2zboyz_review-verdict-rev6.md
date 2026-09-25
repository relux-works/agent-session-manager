# TASK-260924-2zboyz — Change Request revision 6 review verdict

Verdict: **accepted**. Reviewed `CR-TASK-260924-2zboyz-6` revision 6, base `5e40826725a0880897d6bd8321c8fa8e062620d3`, candidate tree `cf13fcb5aba321d93e6fcd58418886da22fd69ef`, patch SHA-256 `e0f422e67753c29fc0562f388a180dd9bb14674c473501efee00292621ce4fae`. A scratch Git index over the live, uncommitted worktree produced the exact candidate tree. No registry or config edit appears in the 21-path task delta.

## Findings array

```verdict-findings
{
  "findings": [],
  "notes": [
    "Revision 3 finding report-sibling-provenance-unbound did not reproduce: both report entries require ValidatedReadBack values, reject zero siblings before reading claims, and bind claims to their sealed ids/modes. The only seal constructors are the read-back Build/Decode validators. The named regression tests and the two N-seal narrowings passed in both reviewer harness runs.",
    "The report's cross-input tuple/plan/fidelity reconciliation and authority freshness remain stated bounds assigned to the final Story leaf/caller chain. This review did not treat those as implemented here.",
    "The producer validation resource is capped at 65536 bytes; its final coverage line is present, but the independent reviewer commands and their complete raw logs, not that resource, support this verdict."
  ],
  "surface_results": [
    {"row":"Closed members, including envelope and EvidenceObject","result":"held","attack":"Reflection-derived 16/5/23 member census ran through Decode; reviewer case-alias plant reddened TestMemberCensusMiscased at both shapes."},
    {"row":"Schema/version literals","result":"held","attack":"TestEnvelopeLiterals and both shipped schema/version narrowings ran through Decode."},
    {"row":"Mode vocabulary","result":"held","attack":"TestModeVocabularyOracle and N-mode-archived ran through Build/Decode/predicate."},
    {"row":"Evidence kind vocabulary","result":"held","attack":"TestEvidenceKindVocabularyOracle and N-kind-native ran through Build/Decode/predicate."},
    {"row":"Native-ID equality","result":"held","attack":"TestNativeIdentityEqualityOracle and N-identity-len ran through both read-back entries."},
    {"row":"Valid iff applicable checks","result":"held","attack":"TestValidWholeDomainOracle covers 1536 combinations through Build/Decode; reviewer valid-marker plant and nine shipped conjunct narrowings reddened it."},
    {"row":"Character string bounds","result":"held","attack":"TestMediaTypeStringEdges and reviewer media-129 plant show the 128/129 edge through Build; shipped Build/Decode media/head narrowings reddened named tests."},
    {"row":"Parsed-head order/count","result":"held","attack":"TestParsedHeadIDsSweep and shipped duplicate/count/item narrowings ran through Build/Decode."},
    {"row":"Evidence order","result":"held","attack":"TestEvidenceOrderSweep and N-evidence-dup ran through Build/Decode."},
    {"row":"Evidence count","result":"held","attack":"TestEvidenceCountEdges and N-evidence-count ran through Build/Decode."},
    {"row":"Findings count","result":"held","attack":"TestFindingsCountEdges and separate Build/Decode count narrowings ran."},
    {"row":"uint53/number model","result":"held","attack":"TestScalarGates and N-parse-count-edge ran; Decode type ceiling delegates to environ, whose suite was rerun."},
    {"row":"Scalar UUID/digest","result":"held","attack":"TestScalarGates/TestDecodeScalarStrings and four scalar narrowings ran through both shapes."},
    {"row":"Owner nesting","result":"held","attack":"TestOwnerDelegation drove broken tuple/binding/finding documents through entries; owner package suites were rerun on source-identical trees."},
    {"row":"Extensions","result":"held","attack":"TestExtensionsClosed drove invalid keys at Build/Decode; reviewer wrong-input owner plant reddened Decode report test."},
    {"row":"JCS identities","result":"held","attack":"Independent Python sorted-key JCS+SHA-256 recomputed both baseline fixture ids byte-exactly; tamper tests ran through Decode."},
    {"row":"Mode authority/no relabel","result":"held","attack":"The 3-purpose × 2-claim × 2-entry grid ran; reviewer mode-relabel plant reddened the Build regression; shipped Decode narrowing reddened its regression."},
    {"row":"Build UTF-8","result":"held","attack":"TestBuildUTF8Reflection and N-utf8-lone ran."},
    {"row":"Owner-row census / call reachability","result":"held","attack":"AST/reflection census ran; owner call removal plant reddened, and the five owner trees plus go.mod/go.sum were byte-identical to base."},
    {"row":"Fork guard / literal member-list guard","result":"held","attack":"TestNoForkedLiterals/Selectors, TestMemberCensusNoLiteralList and structural plants ran."},
    {"row":"Report staged/live reference relation","result":"held","attack":"TestReportReadRefRelationOracle and separate Build/Decode reference narrowings ran."},
    {"row":"Importer outcome grid","result":"held","attack":"Reviewer compared exact tree OIDs for environ/scalar/canonicaljson/clonebundle/sessadapter and reran all five package suites; no pre-existing owner input class moved."},
    {"row":"Config, formatting, Windows vet","result":"held","attack":"task-board.config.json is byte-identical to base; gofmt -l empty, git diff --check clean, Windows vet and full go test ./... exit 0; no .pyc exists."},
    {"row":"Report validated sibling provenance","result":"held","attack":"TestBuild/DecodeReportRefusesUnsealedSiblings, AST constructor/parameter guards, and N-seal-build/decode ran; zero and forged-struct paths refused."}
  ],
  "free_hunt": []
}
```

## Measured coverage and exact-tree validation

**6 of 6 AC rows driven** through the exported production entries: closed read-back/evidence/report members (`DecodeReadBackEvidenceManifest`, `DecodeValidationReport`); independently recomputed ids (both Build/Decode pairs); authority-bound mode (both read-back entries); valid-bit whole domain (both report entries); owner census/reachability (all eight exports); narrowing evidence (43 narrowings killed alone, 1 applied control survived, 9 structural plants red). The 18 of 18 counted local gate groups in the candidate matrix have a killed narrowing; owner-inner rules and extension value-model internals are delegated bounds rather than local gates. This is measured coverage, not proof of absence.

Reviewer commands on the exact live tree: `go test ./...` exit 0 (49 packages); `go test -count=3 ./internal/clonereadback/` exit 0; `GOOS=windows GOARCH=amd64 go vet ./...` exit 0; five-owner importer package test exit 0; `gofmt -l internal/clonereadback` empty; `git diff --check` exit 0. The two shipped harness reruns each produced 53 raw row logs with subprocess exits, 43 narrowing kills, one neutral survivor and nine red structural plants. Five additional reviewer plants killed named behavioral tests alone; one reviewer neutral plant survived. Raw logs and the independent fixture recomputation are in the attached evidence archive.

The importer comparison is keyed by the candidate matrix's `(package, entry, input)` rows. For every listed entry/input class of `environ`, `scalar`, `canonicaljson`, `clonebundle`, and `sessadapter`, the package tree OID and module files equal base; their package suites pass on candidate. Thus no pre-existing owner class moved. The four new read-back/report entries have no base outcome and their new classes are covered by the named package tests above.

## Revision 3 blocker resolution

The previous `report-sibling-provenance-unbound` mechanism used caller-mintable `ReadBackManifest` structs. Revision 6 changes the report signatures to accept only `ValidatedReadBack`; its fields are unexported, the two read-back validator entries are the only constructors, and zero values refuse at both report entries before claims are read. The sealed digest/mode bind the two distinct report references. Reviewer attacks against unsealed, equal and swapped sibling paths did not reproduce the previous bypass.
