# TASK-260830-8x76g1 — CR revision 16 review verdict

- Reviewer run: `RUN-260901-5205ba` (claude / opus, high)
- Change Request: `CR-TASK-260830-8x76g1-16`, state `ready`, delta `present`
- Base OID: `ad7275181ca82fc3fa29544e3893923a92d7b9d5`
- Candidate tree OID: `b4ae5fb03cab8481b83d6f8dd33d07db26322618`
- Patch resource sha256 verified: `49ffc6fb169159b7851226a53414d03dd2c06c51a280ec594ac96d5ec381cc1b`

**Verdict: ACCEPTED.**

The workspace tree was independently recomputed as
`b4ae5fb03cab8481b83d6f8dd33d07db26322618` before review, after every mutant, and
after every probe was removed. `internal/canonicaljson/closed_shapes.go` stayed at
sha256 `3d34cb52cb483e513a8ba398ed351993a02c46c01d5f4086af083f038304b307`
throughout — the same hash the rev14 reviewer recorded and the same hash both
producer sweep harnesses assert as their baseline.

## 1. The rev16 delta is exactly the requested test-only scope

Per-file hashing of the rev15 and rev16 patch payloads gives an identical file
set (71 paths) and exactly two changed payloads:

- `internal/canonicaljson/clause_refusal_test.go` — adds
  `TestSymlinkTargetLowerBoundReachesBothIdentityEntries`
- `internal/canonicaljson/testdata/constraint-enumeration.md` — one corrected row
  (`ManifestEntry.symlink.target`)

Production, configuration, README, the pinned specification and the two accepted
leaves are byte-identical to reviewed rev15. The rev15 finding was scoped to one
negative case, one accept-at-bound case, one honest triage row and both sweeps;
the delta is that and nothing else.

## 2. The rev15 finding is genuinely closed — verified by mutation, not by reading

Applied the exact rev15 mutant to the candidate file:

```
closed_shapes.go:1748   if text == "" {   ->   if false && (text == "") {
go test ./internal/canonicaljson -count=1
--- FAIL: TestSymlinkTargetLowerBoundReachesBothIdentityEntries
    clause_refusal_test.go:201: CalculateObjectIdentity(manifest_id malformed shape)
      error = <nil>, want identity refusal containing
      "member target must be a non-empty UTF-8 string"
```

The mutant is KILLED and the new test is its **sole** killer — no other test in the
package fails. `assertIdentityEntriesRefuseWithReason` drives both
`CalculateObjectIdentity` and `VerifyObjectIdentity` and requires the exact refusal
reason, and the accept-at-bound partner (`target: "a"`) drives both entries too.
`requireString` at `closed_shapes.go:1743` remains the sole lower-bound enforcer for
`ManifestEntry.symlink.target`, reached from `validateManifestEntries:794`.

The corrected triage row (`..._run-RUN-260901-0e2e10-mutation-triage.md` site 98)
states this precisely and drops the false universal claim from rev15. It is honest.

## 3. Both mutation sweeps are bound, not accepted on assertion

- **Numeric/bound corpus (71 mutants).** Re-ran `genmutants.py` against this exact
  `closed_shapes.go`; the regenerated corpus is **byte-identical** to the producer's
  `mutants.json` (71 mutants, `IDENTICAL`). Producer tally 56 KILLED / 15 SURVIVED.
- **Numeric result spot-check.** Independently executed 6 mutants spanning the corpus
  (indices 0, 2, 30, 55, 60, 70): **6/6 classifications match** the producer row for
  row, mixing KILLED and SURVIVED.
- **Clause corpus (110 refusal sites).** Producer baseline file hash matches the
  candidate exactly; tally 86 KILLED / 24 SURVIVED, with site 98 (`line1748`) the
  single delta versus the rev15 reviewer's independently reproduced 85/25.
- **Clause result spot-check.** Independently executed 10 clause mutants: **10/10
  match.** These include six of the twelve rev14 "live refusal" clauses — `:300`
  env_literals grammar, `:416` local-board null remote, `:795` symlink target type,
  `:1231` submodule sibling case collision (the narrowing `strings.EqualFold ->
  ==` mutant), `:1295` initialized-submodule non-null members, `:1328` pack/features
  object-format agreement — all still KILLED.
- **Survivor reasoning re-derived, not accepted.** For the four survivors I executed
  I independently confirmed the subsumption: `:579`/`:624` are subsumed by the final
  `covered != totalSize` coverage equality; `:1506` UTF-8 validation is unreachable
  from the public entries because `decodeStrict` rejects invalid UTF-8 input before
  `prepareObjectIdentity`; `:1798` is subsumed by `strconv.ParseUint` plus the earlier
  `validateAXNumbers` float refusal.

Since the rev16 delta only **adds** a test, no previously-killed mutant can regress
to SURVIVED. That, plus the byte-identical production file and the reproduced corpora
and samples, is what binds the two full tallies. **Reported honestly: I did not
re-execute all 181 mutants myself; I executed 16 and bound the rest by corpus
regeneration, baseline-hash equality, and monotonicity.**

## 4. All 17 configured validation gates re-run green in the foreground

| Gate | Result |
| --- | --- |
| `gofmt -l` over tracked+untracked Go files | exit 0 |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1 -v` | exit 0, 894 subtests PASS, 0 FAIL |
| `go test ./... -race -count=1` | exit 0 |
| `go test ./... -cover -count=1` | exit 0; canonicaljson 87.1%, scalar 90.1%, traceability 85.0% |
| `FuzzScalarProductionEntries` 100x parallel=1 | exit 0 |
| `FuzzCanonicalizeRoundTrip` 100x parallel=1 | exit 0 |
| `FuzzObjectIdentityRepresentationInvariant` 100x parallel=1 | exit 0 |
| `FuzzClosedIdentityShapeRefusal` 100x parallel=1 | exit 0 (76 corpus entries) |
| `tracecheck` (default) | exit 0 |
| `cataloggen -check` | exit 0 |
| `GOOS=linux` / `GOOS=windows` builds | exit 0 |
| JSON parse over tracked `*.json` | exit 0 |
| `task-board validate` | exit 0; 262 inherited MISSING_ACTIVITY diagnostics, **zero** mention this task |
| `git diff --check` | exit 0 |

Scoped `tracecheck -section 17.3` exits 0 (`assigned_scopes=1`), as do `-section 1.6`
and `-section 10.1` through `-section 10.4`.

## 5. Gates attacked, not read

- **Fuzz-wiring gate is live.** Added an unwired `FuzzReviewerUnwiredTarget` to
  `internal/scalar`; `TestConfiguredValidationRunsEveryFuzzTargetWithFixedBudget`
  fails with `configured validation contains "...FuzzReviewerUnwiredTarget..." 0
  times, want exactly once`. A newly added target cannot silently skip the suite.
- **Enumeration/code binding is live.** Two independent mutants of
  `constraint-enumeration.md` — deleting the `Blob Descriptor.media_type` row and
  renaming `BlobChunk.offset` to `offsett` — both fail
  `TestConstraintEnumerationMatchesRequireExactMembers` with `missing artifact row
  ...`. Artifact and `requireExactMembers` cannot drift.
- **Cross-platform golden is genuinely language-neutral.** I reimplemented RFC 8785
  independently in Python (UTF-16 code-unit key ordering, ECMAScript string escaping),
  omitted `record_id`, and SHA-256'd the result for **both** golden representations —
  the LF/ordered form and the CRLF/reverse-ordered form with an empty key and a
  U+10000 surrogate pair. Both reproduce
  `sha256:b3cfa25ae57de833d64361e86ede48691b0470610525ddfd1edfa1d03b34504b` exactly.
  The golden is not merely self-consistent with the Go implementation.
- **No bypass around the gate.** `internal/canonicaljson` exports exactly
  `Canonicalize`, `CalculateObjectIdentity`, `VerifyObjectIdentity`. Both attesting
  entries route through `prepareObjectIdentity`, which calls
  `validateImmutableObjectShape` after trusted schema selection and before any digest
  is produced or compared. There is no second path to attestation.
- **Validator registry is total by construction.** `mustBuildImmutableObjectShapeValidators`
  panics at package init if any `catalog.Current().SelfIdentities` row lacks a
  validator, if a validator exists for an unregistered schema, or if the row counts
  disagree. `TestPublicIdentityRefusesNilShapeValidatorRegistryEntry` pins the nil
  branch through both public entries.
- **Requiredness is executable.** `TestEveryFixtureClosedShapeMemberIsRequiredAtBothIdentityEntries`
  runs **278** subtests, recursively deleting every member of eight fixtures spanning
  the registered closed shapes, each requiring both public entries to refuse.
- **Independent shape probes at the public entry.** I drove
  `CalculateObjectIdentity` with hand-built attack shapes. Correctly REFUSED: hardlink
  naming a later file; hardlink naming a directory; 65-character Session Record name;
  4097-character multibyte symlink target; uppercase, single-label, trailing-dot and
  trailing-separator extension keys; non-adjacent case-colliding entry paths; `../..`
  symlink escape. Correctly ACCEPTED: 64-character name, 4096-**character** multibyte
  symlink target (proving the character-not-byte bound in the accepting direction),
  and the in-root `.`, `./x`, `a/../../b` targets.

## 6. Non-blocking observation, reported with evidence rather than omitted

Section 10.4 states: *"Entries and child partitions MUST contain no duplicate,
overlapping, or destination-case-colliding path."* Duplicate and destination-case
collision are enforced entry-locally. **Overlapping is not**: a `workspace_tree`
Transfer Manifest whose entries are `{"path":"a","type":"file",...}` and
`{"path":"a/b","type":"file",...}` — a regular file with a descendant, which is
filesystem-impossible — is ATTESTED by both public entries. I verified this directly.

I am **not** raising this as a finding, and the reasoning is not plausibility:

- The pinned text gives no entry-local definition of "overlapping". Any
  ancestor/descendant reading would invalidate ordinary manifests, since the spec's
  own positive fragment is `{"path":"src","type":"directory"}` and directory entries
  are legitimately prefixes of their contents.
- The spec's partition concept is named explicitly elsewhere: *"A larger entry or
  index list MUST be partitioned into path-disjoint child manifests"* (SPEC.md:5211).
  That is the disjointness "overlapping" most directly refers to, and it needs the
  referenced child manifests, so it is external to one identity candidate.
- No normative negative fixture requires this refusal. `TM-ENTRY-DIR-N1`,
  `TM-ENTRY-SYM-N1` and `TM-ENTRY-HARD-N1` are all enforced and independently
  verified above; `TM-ENTRY-FILE-N1` is correctly external.
- `constraint-enumeration.md:213` already discloses the split, stating that
  entry-local duplicate/case collision is what is enforced and that child-partition
  paths are external.

Recording it so the fact is on the board rather than silently absent. If a future
schema owner or a spec clarification defines entry-local overlap, this is where it
would attach: `validateManifestEntries`, alongside the existing `simpleFoldKey`
collision set.

## 7. Definition of Done

Every checklist item is satisfied by evidence I either executed or bound as described
above, including the rev16-specific items: the trailing zero-size BlobChunk clause has
its own case reaching `closed_shapes.go:614`; the `GitIndex.entries` row honestly
states that public-entry acceptance at 65536 is not claimed because the encoded object
exceeds the 5,242,880-byte cap; the 65,536-entry manifest gate is linear with a
measured sub-2-second wall-clock guard (`//go:build !race`, with the race-detector
exclusion justified in-file); and README claims only what reproduces through both
public identity entries.
