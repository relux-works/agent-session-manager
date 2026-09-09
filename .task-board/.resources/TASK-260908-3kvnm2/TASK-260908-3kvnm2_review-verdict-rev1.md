# TASK-260908-3kvnm2 CR1 independent review

Verdict: ACCEPTED for the scoped, staged immutable-source adoption. No blocking implementation finding. This is not Story completion or runtime conformance acceptance.

Reviewed base `2a8db9653e476f8375371b16e9b6b82adfa23b91`, candidate tree `49db6d060742b6ba68426ad583c17f0e7afcf5da`, all 10 changed paths. Every candidate tracked blob was compared with the working tree before testing; all equal. HEAD is unchanged and has zero commits behind local main. No repository file, index, branch or upstream ref was changed by this review. Mutations ran only in a Git archive scratch copy under this task's .temp directory.

## Source provenance and compatibility

Fresh git ls-remote returned main/peeled v0.6.0 commit `0cbdf100dbf84df50c64f792b1f940e3a67859a6` and tag object `40c123eb8399efa8e05cbc009110940ed861a785`. Independently verified both local immutable objects with git verify-tag / verify-commit: exit 0, Ivan Oparin's configured ECDSA signing fingerprint `SHA256:V6JiKG7J29mjsvikcLoSVp0bLa77VTsFy12gnLO81cM`. Tagged SPEC blob equals candidate embedded source byte-for-byte: 1,150,005 bytes, SHA256 `74504539fb43c28ae3450622bc1002e643f116cd3e14df4567a882231e90896b`. This establishes signed git-tag delivery, not a hosted Release.

Independent extraction from that source confirms all 170 heading IDs and digest `2ef110e0f87c5d174bceafb33a0db981f5dc74a384f8b599f12b87cb4af2eb94`, all 63 ordered registry rows and versions, and all five fixture file hashes against upstream Git blobs. Selector's fixture ID is honestly documented as pin-local; upstream has no fixture discriminator. Historical v0.5.0 lock/document are unchanged from the review base. Runtime tests compare v0.6.0 historical projections with the real v0.5.0 lock and its v0.4.3 projection.

## AC coverage: 4 of 4 rows driven

1. Verified source identity: `TestCurrentV060PinsSignedV060Source` -> `specpin.CurrentV060` -> `VerifyV060`. Bound: tests attest the pinned tuple, not network publication or cryptographic validity; the independent Git commands above establish those properties.
2. Document/inventory fidelity: `TestLoadV060AcceptsOnlyTheAdoptedDocumentDigest` -> `specdoc.LoadV060` / `BytesV060`; `TestSectionInventoryV060MatchesAdoptedHeadingDigest` -> `specpin.SectionInventoryV060` / `IsSectionV060`. Source-derived measurements independently validate the constants, rather than assuming implementation tables are authoritative.
3. Historical auditability: `TestContractsForReleaseV060` -> `Manifest.ContractsForRelease`, `CurrentV060`, `Current`; `TestSectionInventoryV060PreservesV050History` -> versioned inventories. Old lock/document byte identity checked separately.
4. Wrong/partial/stale evidence refusal: `TestVerifyV060RejectsWrongSourceHashAndVersionEvidence`, `TestVerifyV060RejectsPartialMalformedAndByteDifferentReads`, `TestVerifyRefusesForeignReleaseBytes` -> real `VerifyV060` / `Verify`; `TestParseV060RefusesEveryNonAdoptedDocument`, `TestParseRefusesAdoptedDocument` -> real `ParseV060` / `Parse`; `TestStalePinCannotAuthorizeNewRelease` -> `ContractsForRelease`.

## Independent gate attacks

Six of six narrowing mutants killed, each go test command exit 1 with a semantic assertion failure, not a setup or compile failure. Every run executes BOTH complete specpin and specdoc behavioral suites with -v -count=1. Candidate baseline is green. The other tests pass in each mutant run; the full mutant suite itself is correctly RED.

- M1: allow exact trailing-newline lock digest; `TestVerifyV060RejectsPartialMalformedAndByteDifferentReads/byte_different_whitespace` fails.
- M2: allow old commit while retaining v0.6.0 tokens, plus only that changed lock's digest; `TestVerifyV060RejectsWrongSourceHashAndVersionEvidence/stale_commit_keeps_v0.6.0_tokens` fails.
- M3: allow old document digest; `TestParseV060RefusesEveryNonAdoptedDocument/stale_v0.5.0_document` fails.
- M4: retain registry name/ID tokens while dropping version comparison, with only trimmed Configuration member digest admitted; `TestVerifyV060RejectsWrongSourceHashAndVersionEvidence/trimmed_configuration_delta` fails.
- M5: admit only nonexistent section 14.7.6; `TestSectionInventoryV060MatchesAdoptedHeadingDigest` fails.
- M6: let a v0.5.0 manifest authorize v0.6.0; `TestStalePinCannotAuthorizeNewRelease` fails.

The exact-byte lock gate subsumes semantic validation clauses for externally supplied bytes; M2/M4 deliberately widen the digest gate for precisely the attacked member so semantic refusal is tested end to end. The source-adoption entry points hash bytes, not search source tokens; M4 nevertheless supplies the requested token-preserving behavioral control. This review does not claim an exhaustive mutation score for every historical helper or every normative runtime obligation.

## Validation personally executed

Darwin arm64, Go 1.25.5 (project go.mod minimum 1.25.0; cross-platform Go CLI/library, no iOS target):

- `go test ./... -v -count=1`: exit 0, 23 packages passed.
- `go test ./... -cover -count=1`: exit 0, 23 packages passed; specpin 87.2%, specdoc 100.0%.
- `go vet ./...`, `go build ./...`: exit 0 each.
- `gofmt -l` all six changed Go files: exit 0, nonempty selection, empty result.
- `git diff --check`: exit 0.

Full logs, actual per-command exits and mutant failures are attached in `TASK-260908-3kvnm2_review-evidence-rev1.tar.gz`. `GOOS=windows go vet ./...` and `GOOS=windows go build ./...` both exit 0 (arm64); these are compile/vet evidence only, not Windows runtime execution. No hosted CI was triggered. No producer test result is used as a substitute for these independently run checks. Prior release semantic review remains accepted upstream evidence; its expensive normative suite was not rerun here.

## Logbook: evidence anomaly and resolved probes

Producer `mutant-battery.log` contains failed setup assertions, failed restore reports and masked pipeline statuses; it does NOT substantiate the claimed four successful kills. The retained Python mutant applicators were independently executed successfully against the exact candidate scratch copy, recovering all four claimed kills; two further attacks were added. The producer's wording that the full suite "stays green" is inaccurate: the named subtest and its parent suite fail. This evidence supersedes that wording without modifying the reviewed candidate. Initial skill lookup in the worktree failed because Curator installation exists at the main project root; the installed project Go skill there was read. An initial heading extractor omitted lettered numeric headings (4.A etc.); the corrected source-derived extraction includes them and matches all 170 IDs. Failed probes are not counted as validation success.

## Required sibling handoff

Staging is acceptable because the primary assignment explicitly reserves catalogue/traceability regeneration to TASK-260908-2tkufa and forbids claiming runtime implementation. This leaf provides verified immutable prerequisites. The full Story must not close with only unused parallel V060 readers.

Sibling must wire `cataloggen.Generate` (currently Verify), registry release enumerations/order/writer, `cigate.CheckContractRoots`/pin reads, and `traceability.Verify` lock/catalog/ownership paths, document Load/digest checks, section membership and inventory/range expansion to the v0.6.0 APIs and regenerated artifacts. It must assign all new selector/auth/migration obligations to real owners with truthful uncovered status and add consumer-entry tests proving the new authority is actually consumed. Preserve explicit historical readers and projections. Do not blindly change Current globally: `localstore` platform-path registry intentionally binds historical source metadata and must either stay explicitly historical or receive a separately coherent registry migration. No selector, host-authentication or migration runtime support is established by this leaf.

Reviewer acceptance routes CR1 to integrating. Only the tracked producer role may checkpoint/integrate; this review makes no commit and supplies no commit_ack.
