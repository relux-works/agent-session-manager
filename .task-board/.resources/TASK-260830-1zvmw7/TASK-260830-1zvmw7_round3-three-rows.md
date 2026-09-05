# Round-3 rework: three unpinned arms (H1/H2/H3)

No production change. Three table rows in internal/terminalbackend/manifest_test.go (untracked CR file); narrowed fixtures, digest-tracking F3 probe, and aliasing pin untouched.

## Rows added
1. TestReconcileRefusals/static claim without manifest -> detail probe static claim without manifest. Probe flips graceful_stop (index 1) to origin static while the manifest omits it; value stays true with evidence intact, probe_id re-stamped, so only that arm can reject. Drives ParseManifest/ParseProbe/Reconcile via reconcileFixture.
2. TestEvidenceDocumentRefusals/bad platform -> detail evidence platform. platform=plan9, evidence_id re-stamped by the table loop; ParseEvidence never verifies attestation so the original signature is a valid bystander.
3. TestManifestDocumentRefusals/non-object claim -> detail claim shape. claims[0] replaced with the string durable_disconnect.

## Enumeration method (how the fifth arms siblings were found)
Not taken on trust from the verdict: extracted every mismatchf/Detail string from manifest.go and counted per-detail assertions across manifest_test.go + manifest_pin_test.go + internal_pin_test.go. Exactly five details had zero hits: the three above plus document identity (x2, declared practically unreachable) and registry unavailable (M92 near-equivalent) — both already in the round-2 stated bounds. No other file-unique arm is unpinned. Shared-detail sites (19) still name the rule, not the site, per the stated bound.

## Mutant evidence (each anchor count==1, vet-checked, reverted byte-identical)
- H1 neutered (case body -> _ = static): KILLED. TestReconcileRefusals/static_claim_without_manifest fails: error = <nil>, want mismatch at probe static claim without manifest. Admits — no slide to a lower arm.
- H2 narrowed (platform, _ :=): KILLED. TestEvidenceDocumentRefusals/bad_platform fails: error = <nil>, want mismatch at evidence platform.
- H3 narrowed (object, _ :=): KILLED. TestManifestDocumentRefusals/non-object_claim fails: refusal = mismatch at document members, want mismatch at claim shape. Fail-closed but arm-blind, as predicted; now arm-pinned.

## Gates (real exit codes, standalone processes)
- go build ./... : exit 0
- go vet ./... : exit 0
- gofmt -l . : exit 0 (no output)
- go test ./internal/terminalbackend/ -count=1 : exit 0
- go test ./internal/terminalbackend/ -count=1 -cover : exit 0, 88.9% of statements (was 88.6%)
- go test ./... -count=1 : exit 0, all 14 packages ok
