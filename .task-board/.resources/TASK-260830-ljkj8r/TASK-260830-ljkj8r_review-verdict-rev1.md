# TASK-260830-ljkj8r — review verdict, CR rev 1

- Verdict: **changes requested** → `to-dev`
- Reviewer run: `RUN-260906-ccfe0d`
- CR: `CR-TASK-260830-ljkj8r-1` rev 1, `repository_delta=present`, 23 paths
- Base `7c25ae034375111e6b765b947a136a9efa4b0ab6`, candidate tree
  `4d8a5998ba35ce511b3db27288f9ef2454d8c5a6`
- Working tree verified equal to the candidate tree before and after review;
  every mutant restored by copy-back and re-verified (`git write-tree` on a
  throwaway index returned `4d8a599…` after the last mutant).
- `repeat-of:` none — first round of this leaf. Class-of: leaf 1
  (TASK-260830-2z3se0) rev3-B5 / rev4-B8, the bound-edge class.

## What I ran myself

| Gate | Result |
|---|---|
| `go test ./... -count=1` | exit 0, 18 packages ok |
| `go vet ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./internal/dirnode/ -coverprofile` | 75.0% (query.go 57.1%) |
| Bound-edge mutant sweep (mine, 94 rows) | 19 killed / **75 survived**, 0 NOT_APPLIED, 0 COMPILE_FAIL |
| Closed-vocabulary sentinel sweep (mine, 54 rows) | 20 killed / 34 survived |
| Targeted narrowing mutants (mine, 3) | 3 survived |
| Producer M8 (aliased import) reproduction | reproduces exactly as reported |
| Ownership-pin attack | reddens with an exact digest mismatch |

Accepted from attached evidence without rerunning: `-count=3`, `-race`,
`GOOS=windows go vet`, `tracecheck`. Full-repo `-race` was not run by either
side; the producer stated that bound rather than implying it, which is correct.

## Blocking

### B1 — Bound edges are unwitnessed as a class; leaf 1's bound census did not transfer

I generated one-step mutants for every bounded-check argument in
`manifest.go`, `probe.go`, `scan.go`, `query.go`, `protocol.go` — each moves a
single bound by exactly one, so the gate stays present and admits exactly one
more member of the class it must reject. 94 applied, 94 measured:

| | count |
|---|---:|
| killed | 19 |
| **survived** | **75** |
| survivors: manifest.go / probe.go / scan.go / query.go | 13 / 12 / 7 / 43 |

The survivors are reachable and the production bounds are correct — I drove
four of them through the production entry points and all four out-of-range
values are refused today (`max_scan_instances=65537`, `max_inventory_take=0`,
`caller_id=""`, `caller_id` at 257 characters). So the gates are right; the
evidence that they are right does not exist.

This is the class leaf 1 closed. `internal/sessadapter` carries
`bound_census_test.go` (1650 lines) and `bounds_edge_test.go` (423 lines) for
exactly this, and the latter states the rule verbatim: *"A witness far outside
the range proves the arm exists, not where the bound sits; a mutant moving any
bound by exactly one admits exactly one member of the reject class and reddens
its row here."* `internal/dirnode` has neither file. The identical mutant shape
is killed in leaf 1 and survives in leaf 2:

| Mutant | sessadapter | dirnode |
|---|---|---|
| `checkStringBounds(…, 1, 128)` → `1, 129` | KILLED (`TestDecodeManifestValueRules`) | SURVIVED at every `checkStringBounds` maximum probed |

The refusal-arm census cannot see this class by construction: a bound lives
inside a helper that returns `bool`, and the whole helper funnels into one
literal-detail arm. `manifest.go` limits are one arm covering 14 edges.

Specific instance of a claim the vectors do not support: the doc comment on
`TestManifestLimitsBounds` says *"each bound refuses past its edge in both
directions"*. Four of the fourteen edges it covers are unprobed and all four
are confirmed survivors — `max_scan_instances` upper (65536→65537), and the
lower edges of `max_inventory_take`, `max_enrichment_events`,
`max_enrichment_bytes` (1→0).

**What closes it:** a bound census over this package with an enumerated shape
space (leaf 1's file is the model, not a thing to copy verbatim — the helper
set here differs), plus an edge file driving each surviving bound at
min-1/min/max/max+1 through its production entry. Report the result as
killed-over-applied with any remaining row as a stated bound. Do not close only
the 75 rows I found: derive the site set from production source so the next
bound added has a row by construction.

### B2 — AC row 5's "17-op union" is not driven: 5 of 17 operations, 10 functions at 0% coverage

The evidence table reports 10 of 10 AC rows driven. Row 5 names
"§10.8.5: envelope, caller, 17-op union, projection, pagination, flags,
cursor". The union half is not driven.

- Operations that reach `DecodeQuery` in any committed test: `schema`,
  `sessions`, `set_title`, `plan_continue`, `execute_plan` — **5 of 17**.
- Never decoded: `session`, `lineage`, `hosts`, `environments`, `jobs`,
  `plans`, `count`, `distinct`, `directory_summary`, `set_tags`, `set_pin`,
  `enrich`.
- Ten production functions have **0.0% statement coverage**, all of them
  §10.8.5 union validators: `validQueryField`, `checkLineageParameters`,
  `checkHostsParameters`, `checkEnvironmentsParameters`,
  `checkJobsParameters`, `checkPlansParameters`, `checkDistinctParameters`,
  `checkTagsParameter`, `checkPinParameter`, `checkEnrichParameters`.

Three narrowing mutants inside those validators — gate preserved, one extra
member admitted — all survive:

| Mutant | Result |
|---|---|
| `checkEnvironmentsParameters` `authentication_status` admits `"root"` | SURVIVED |
| `checkJobsParameters` `states` admits `"pwned"` | SURVIVED |
| `checkDistinctParameters` `field` admits `"secret"` | SURVIVED |

No stated bound covers this. The evidence's "Stated bounds" paragraph names the
eight unframed **node** operations and the nested content types — not the
twelve undriven **query** operations, which are inside the framed scope this
leaf claims.

The whole class funnels into one census arm
(`ctor|failQuery|directory query operation is not a closed QueryOperation`)
with four named witnesses, so the census reads complete while ten validators
behind it have never executed. This is the leaf-1 wrapper shape — a class
closed at one helper leaving the wrappers open — recurring in a new form.

**What closes it:** drive each of the 17 operations through `DecodeQuery` with
an admitted vector and at least one refusal vector per per-operation validator,
or state the undriven ones as an explicit bound with the reason. Report the
ratio, not prose.

### B3 — the "unknown field" vector refuses for the wrong reason; `queryFields` is unproven

`TestDecodeQueryRegistryRules/unknown_field` builds its body as

```go
replaceOnce(t, good, `"preset":"overview"`, `"fields":["nope"],"preset":null`, 1)
```

The fixture operation already carries `"fields":null`, so the result holds
three `"fields":` members. `decodeStrictObject` refuses on **duplicate member**
before `checkQueryProjection` ever runs. `validQueryField` sits at 0.0%
coverage and the sentinel mutant on the 27-member `queryFields` table survives.

Both refusal paths return the same code *and* the same detail
(`query_invalid` / `directory query operation is not a closed QueryOperation`),
so `requireCode(t, err, "query_invalid")` cannot tell them apart.

Verified against production: a vector built without duplicating the member does
reach the registry — `fields:["nope"]` refuses, `fields:["id"]` and
`fields:["host","id"]` admit, `fields:["id","host"]` refuses as unsorted. So
production is correct and only the vector is wrong.

Worth carrying forward: `replaceOnce` was added to catch vectors that mutate
nothing. It does not catch a vector that mutates the body but is intercepted by
an earlier gate. A same-code-same-detail collision is invisible to
`requireCode`.

## Non-blocking

### N1 — registry derivation covers a minority of the closed vocabulary tables

`inventory_test.go` derives four tables from the pinned specification text
(`operationOrder`, `capabilityOrder`, `readOperations`, `mutationOperations`)
and each is killed by a sentinel member. Eleven other closed `[]string` tables
carry no exact-content pin and pass green with an added member:
`capabilityStatuses`, `platformV1`, `platformV2`, `architectures`,
`findingSeverities`, `listOperations`, `annotationMutations`, `queryFields`,
`queryPresets`, `querySortFields`, `queryScopes`.

Caveat stated rather than hidden: the sentinel-append mutant is a weak
discriminator where the production entry already carries an unknown-token
witness. `platformV1`/`platformV2`/`architectures` are in that position —
`TestProbeRequestMajorVocabularies` drives both cross-major refusals, both
admits, and an out-of-registry token (`plan9`), so those three are covered in
substance and only unpinned in content. The practical exposure is
`queryFields`, `queryPresets`, `querySortFields`, `queryScopes`,
`listOperations`, `annotationMutations`, `capabilityStatuses`.

### N2 — `internal/dirnode` holds no section or contract ownership binding

§7.8, §7.9 and §10.8.5 remain owned by `internal/catalog/catalog.go:ForRelease`
at `unevidenced`, as do all six Directory Node contract keys. The section
coverage line is byte-identical before and after this leaf
(`bindings=53 … clauses_discharged=17/428`); the change adds six acceptance
cases and moves the section needle by zero.

Not blocking: leaf 1 (`sessadapter`) and `terminalbackend` register zero
ownership bindings too, so this matches the Story's established shape, and the
README explicitly discloses the `unevidenced` binding instead of claiming
coverage. Flagged so the Story can decide once rather than per leaf.

## Verified good — do not re-litigate next round

- **The refusal-arm census is materially stronger than leaf 1's.** The
  constructor set is derived from production and required to equal the
  registered eight; `axerror.New` outside a registered body is refused; the
  selector in non-call position is refused; a constructor identifier in any
  non-call, non-definition position is refused with six synthetic proofs; the
  import census fails closed on a zero-importer tree; both directions plus
  witness resolution are checked. The aliased-import attack the brief warned
  about (`var zzprobeIndirect = requireStringBounds` in leaf 1) does not apply
  here — `visitIdent` matches by position, not by call shape.
- **M8 reproduces exactly as reported.** I re-ran the aliased-import mutant
  (`axe "…/axerror"`, path token preserved, behaviour identical): four census
  tests fail, the behavioural suite stays green. Scoring it as a census-only
  kill separate from the behavioural count is the correct accounting.
- **The traceability re-pin is computed, not copied.** Renaming one acceptance
  case id reddens with an exact mismatch: projection `d7a837dc…` differs from
  reviewed `fbdbb5b6…`, in both `traceability` and `tracecheck`. Declarations
  are AST-resolved (`hasDeclaration`), and all six new rows name declarations
  that exist.
- **`Journal.Import` fails closed correctly.** It builds into a local map and
  assigns to `journal.records` only after every record validates, so a
  malformed or truncated import cannot wipe recorded keys and re-authorize a
  replay under a recorded `(operation, operation_id)`. A failed read is an
  integrity failure, never an empty journal.
- **README makes no unsupported claim.** It states the §7.9 binding stays
  `unevidenced` and names the unframed eight as a pinned bound.
- **No orphan guards.** Every `decode.go` helper has production callers; the
  public entry points with no internal caller are the package's API by design.
- **Suite is genuinely green** on my own run: 18 packages, vet clean, gofmt
  clean.

## Shape of the ask

Three blocking items, all one class in different clothes: a gate whose class is
closed at one point while the rest of the class stays open, and evidence that
reads complete because the instrument cannot see the open part. B1 is the
bound-edge instrument leaf 1 built and this leaf did not carry over; B2 is the
same shape at the operation-union level; B3 is a single vector that never
reaches the gate it names.

Derive the site sets from production source so the next addition has a row by
construction. A hand-listed fix to the 75 + 12 + 1 rows named here would leave
the same class open at whatever is added next.
