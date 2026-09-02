# BUG-260902-32iy1u — reviewer verdict: ACCEPTED

Change Request `CR-BUG-260902-32iy1u-1` revision 1, base `8b0bc15`, candidate tree
`ba2ba03`, repository delta `present` (7 files). Working tree verified byte-equal
to the candidate tree before and after every probe below
(`git diff --stat ba2ba03 -- .` empty).

## What the change decides

Two answers, one rule: the constraint applies exactly where the pinned document
declares it, and where it is declared it means the whole named standard.

1. `EnvironmentTuple.adapter_version` — constraint REMOVED. Presence-only,
   matching its untyped `environment_version` / `store_schema_fingerprint`
   siblings in the same clause.
2. Every site the document types `semver` (§17.3 migration provenance,
   `terminal.*` Session Event versions) — constraint KEPT and WIDENED from a
   core-triple pattern to Semantic Versioning 2.0.0 in full.

Both halves are written into `internal/canonicaljson/testdata/constraint-enumeration.md`
as an explicit "Recorded decision" section, not left implied by the code.

## Independent verification of the contract claims

Extracted the pinned `SPEC.md` from the spec repo at the peeled commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c` and confirmed its SHA-256 equals the
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a` recorded in
`.spec/README.md`, then read the clauses rather than trusting the summary.

| Claim under review | Verified at | Result |
| --- | --- | --- |
| EnvironmentTuple declares `adapter_version` with no type and no format | SPEC.md:3627-3631 | Confirmed. The clause is a bare member list. |
| The SemVer word reaches it only from a different schema | SPEC.md:3610 (Session Adapter Manifest row) | Confirmed. That row closes the Manifest, not the tuple. |
| The Probe equality sentence names the Probe's own top-level member | SPEC.md:3620-3626 | Confirmed. `adapter_version` is a Probe member, sibling of `environment:EnvironmentTuple`, not a member of it. |
| §17.3 authorising line is quoted verbatim | SPEC.md:11459-11461 | Confirmed verbatim: "That extension value is a closed object containing exactly `schema_id:string`, `schema_version:semver`, and `object_id:digest`." |
| The document spells out no grammar for `semver` | grep for "Semantic Version"/"SemVer" across SPEC.md | Confirmed. `semver` is used as a bare type name at 30+ sites; no literal grammar anywhere. Adopting the named standard in full is the defensible reading. |
| `terminal.*` versions are declared `semver` | SPEC.md:1871-1872 | Confirmed. |

The widened pattern was checked against the **official semver.org corpora**
rather than against itself: 34 documented-valid versions all admitted, 41
documented-invalid versions all refused, in a throwaway module outside the repo.
It is character-for-character equivalent to the reference SemVer 2.0.0
expression (`[0-9]` for `\d`, non-capturing groups). It is also now byte-identical
to `internal/config/validation.go:21`, which already carried the full spelling —
the repo's internal inconsistency about its own type name is closed.

## Gates attacked, not read

Six mutants applied to production sources and reverted; every one killed.

| # | Mutant | Killed by | Failure observed |
| --- | --- | --- | --- |
| A | Re-add the original core-triple gate to `validateEnvironmentTuple` | `TestEnvironmentTupleAdapterVersionCarriesNoInferredSemVerConstraint/prerelease` | `session_record_versions_test.go:1038` — `1.2.3-rc.1` refused with "EnvironmentTuple adapter_version must be canonical semver". This is the exact reported bug, reproduced and pinned. |
| A2 | **Narrowing, not deletion**: re-add only `requireString` (type without format) | same test, `number`/`boolean`/`null`/`empty` subcases | An inferred *type* is caught too, not just an inferred format. Also reddens `TestEveryFixtureMemberRefusesAWrongJSONTypeAtItsProductionEntry` as an obsolete exemption. |
| B | Narrow `semverPattern` back to the core triple (production only) | `TestMigrationProvenanceSchemaVersionIsSemVer200InFull` (6 accept cases) + `TestEveryCompiledGrammarMatchesItsPinnedReference` | Production drifting from the pinned reference is caught independently of the behaviour cases. |
| B2 | **Coordinated** narrowing of production *and* the pinned reference | `TestEveryDeclaredGrammarDimensionHasAProvenWitness`, `TestEveryProductionRefusalGuardIsExecuted`, plus the 6 accept cases | 14 witnesses reported as "claims a dimension no production grammar declares". The regression cannot hide by moving the goalposts. |
| C | Delete the §17.3 migration gate entirely | `TestMigrationProvenanceSchemaVersionIsSemVer200InFull` (11 refuse cases) | Proves widening is provably not the same as deleting. |
| E | **Coordinated widening**: add `_` to the build-metadata class in production *and* reference | three independent tests | `witness semverPattern|class#17 value "1.0.0+a_b" is admitted by the production grammar; it proves nothing`, plus a production-entry admission and a behaviour-case failure. |
| F | Delete the `terminal.*` semver gate (probe, not a defect claim) | `TestEveryStructuredFixtureValueRefusesAMalformedFormAtItsProductionEntry`, `TestEveryProductionRefusalGuardIsExecuted` | Confirms the artifact's claim that the third site is genuinely pinned, so widening it was not an unguarded change. |

Every case drives `CalculateObjectIdentity`/`VerifyObjectIdentity` — the real
public entries — not `validateEnvironmentTuple` directly. The `absent` case
attributes its refusal by name and deletes the member from **both** tuples, so
it cannot pass on the sibling's extra-member refusal; mutant D (dropping the
member from `requireExactMembers`) was re-run and reddens exactly that case.

The 22 new grammar-inventory dimensions are not decorative: the obligation set is
derived mechanically from the pattern source, each witness must be refused by
production, admitted by exactly its own one-dimension widening, and refused again
when driven through the production entry with the refusal attributed to this
clause. Mutants B2 and E confirm all three legs fire.

The producer's own claim that mutant D was green on the first attempt and passed
for the wrong reason is recorded in LOGBOOK.md. That is the right disclosure and
the fix is correct.

## Repository validation (re-run by the reviewer)

| Check | Result |
| --- | --- |
| `gofmt -l .` | clean |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | all packages ok |
| `go test ./... -cover -count=1` | all ok; `internal/canonicaljson` 97.2% |
| Coverage baseline at base OID `8b0bc15` (`git archive` to a clean tree) | 97.2% — **no regression**, claim verified rather than accepted |
| `go test ./internal/canonicaljson -race -count=1` | ok (90.9s) |
| `go run ./internal/traceability/cmd/tracecheck` | exit 0 — contracts=60 sections=36 cases=43 fixtures=30 |
| `cataloggen -check` | exit 0 |
| Fuzz gates at 100x (`FuzzClosedIdentityShapeRefusal`, `FuzzObjectIdentityRepresentationInvariant`, `FuzzCanonicalizeRoundTrip`, `FuzzObservationEventRefusal`) | all PASS |

## Acceptance criteria

| AC / DoD | Verdict |
| --- | --- |
| Constraint removed and members validated as declared, **or** SemVer 2.0.0 adopted with the authorising line quoted verbatim | Both, correctly split by site. §17.3 quote checked verbatim against the pinned document. |
| Recorded as an explicit decision rather than implied by code | `constraint-enumeration.md` "Recorded decision" section + per-site table + README. |
| A case proves `1.2.3-rc.1` at the production identity entries and reddens if the behaviour flips | Yes — mutant A reddens it at `session_record_versions_test.go:1038`. |
| The second use of the same regex at the migration-provenance site resolved consistently | Yes, and the third (`terminal.*`) as well, under one stated rule. |
| Tests, vet, build, tracecheck, catalog check exit 0 with no coverage regression | Confirmed, baseline measured independently. |
| Gating behaviour covered by negative tests naming the production call site | Confirmed by six mutants including two coordinated ones. |

## Non-blocking observation (not grounds for rework)

`constraint-enumeration.md:199` still records the pinned SPEC declaration for
`MigrationProvenance.schema_version` as “canonical semver”. The document types
that member `schema_version:semver`; "canonical" is the implementation's own
refusal wording carried into the "Pinned SPEC declaration" column, and it now
reads slightly against the decision this change records two tables lower. The
row predates this change, the excerpt column is not machine-checked, and the new
decision section states the site correctly with a verbatim quote, so the artifact
does not mislead a reader. Worth sweeping opportunistically in a later pass over
that column.

## Verdict

**ACCEPTED.** The invented cross-schema constraint is gone, the constraint that
the document does declare is honoured as the whole named standard, the decision
is explicit in the artifact, and the behaviour is pinned in both directions by
tests that were shown to fail against six separate regressions.
