# TASK-260906-33xcnc: unify-remaining-environment-copies — outcome

**Ready for review.** Candidate tree `5063b78a32fd32c25d4bd3f1bcc911951c0c8900`
(detached-index `write-tree` over checkpoint `cd8591d` plus exactly the 22
paths listed below; the real index was untouched; recomputed after the
logbook append so the entry is inside the candidate).

## What changed

Converged the remaining independent environment-rule copies onto
`internal/environ`, hardened the shared uint53 texture, and closed the
story residue with production-called gates, site-resolving witnesses,
and a 55-mutant battery. Net diff is negative (convergence deletes code).

Converged (production, `internal/sessadapter/decode.go` +
`internal/dirnode/decode.go` + call sites):

- `stringLength` → `environ.StringLength`; `checkStringBounds`,
  `checkUint53Bounds`, `checkDigest`, `checkUUIDv7`, `checkTimestamp`,
  `checkSortedUniqueStrings`, `checkSortedUniqueDigests`,
  `checkExtensions` → the `environ.Check*` gates; `checkSemver`,
  `checkEnvironmentID` (dirnode) → `environ.CheckSemver`,
  `environ.CheckEnvironmentID`.
- Grammar call sites now call the owners directly:
  `environ.CheckSemver` (`sessadapter/manifest.go`,
  `sessadapter/tuple.go` ×2, `sessadapter/probe.go`),
  `environ.CheckEnvironmentID` (`sessadapter/manifest.go`,
  `sessadapter/tuple.go`, `dirnode/probe.go`, `dirnode/query.go`),
  `environ.CheckExtensions` (both `decode.go`). Deleted the converged
  `semverPattern`, `environmentIDPattern`, `reverseDNSPattern` copies (6
  grammar ledger rows and 2 shape rows removed as orphaned — the census
  reported the convergence itself before the ledgers were updated).
- `rawUint53` trailing-data arm added in all three siblings
  (`sessadapter/decode.go`, `dirnode/decode.go`,
  `provhost/opdecode.go` + `io` import), matching the environ texture
  exactly; empty-literal guard added to all three `parseUint53Literal`
  ladders after the new unit test caught `""` admitting as 0.
- `refuse` is now a constructor var (single funnel, behavior identical)
  so the site audit can record production file:line behind every refusal.

Retained with justification (no leaf-owned production touched:
`provhost/protocol.go`, `provhost/identity.go`, `provhost/probe.go`,
`canonicaljson/closed_shapes.go` are frozen by accepted leaves 1–4):

- provhost decoder + surrogate gate: owning assembly frozen by leaves
  2/4; pinned both directions by the frame battery (exact verdicts) and
  the constant-derived surrogate sweep. Convergence is deferred (see need
  below), not abandoned.
- canonicaljson gates: different role (any JSON value; own
  depth/number/control rules), pinned both directions by the frame
  battery's `canonicalRefuse` verdicts.
- scalar gate: different unit (single JSON string scalar, not a frame);
  now pinned both directions by the new 15-vector scalar agreement
  battery through `scalar.DecodeClosedEnumJSON` and
  `environ.DecodeStrictObject`.
- provider-id grammar copies: scalar owns the rule; pinned by the
  one-language battery (literal equality, both directions by
  construction). Mechanic copies (`rawString`, `decodeArray`,
  `unknownMember`, `missingMember`, `isNull`, private ladders) are
  rule-free plumbing pinned transitively by the frame/tuple/measure
  batteries plus the new structural delegation pins.

## AC coverage: 6 of 6 rows driven through production entries

| AC row | Verdict | Production call site / battery |
| --- | --- | --- |
| provhost decoder + surrogate gate | justified + pinned | `provhost.DecodeManifest` via frame battery (exact verdicts both ways) |
| canonicaljson frame gate | justified + pinned | `canonicaljson.Canonicalize` via frame battery |
| scalar third-spelling gate | justified + pinned | `scalar.DecodeClosedEnumJSON` vs `environ.DecodeStrictObject`, 15 vectors |
| sessadapter/dirnode per-helper copies | converged (20 wrappers + 8 call sites, 6 vars deleted) | wrappers call `environ.*`; `TestCheckHelpersDelegateToEnviron` + batteries |
| sibling rawUint53 trailing texture | hardened ×3 + ladder guard ×3 | `TestRawUint53RefusesTrailingData` (×3 pkgs) + `TestParseUint53LiteralRefusesNonDigits` |
| F2 residue exits | attacked + re-stated | 11 new residue/narrow mutants; denominator re-derived 90; callerless closed for 10 symbols |

Prose is not evidence: every row above names its production entry and its
failing-test contract (see mutant table artifact).

## Residue re-derived (not carried forward)

- Denominator at runtime: **31 refuse sites + 59 boolean-false exits = 90**
  (was 89; +1 is the new ladder empty-guard). The "27 of 100" figure was
  two stories stale.
- Refusal-site coverage: **31/31 derived sites exercised through
  production entries** (TestMain audit: derived == exercised both ways,
  39 expanded rules unique and exactly observed).
- Callerless symbols: `CheckStringBounds`, `CheckUint53Bounds`,
  `CheckSortedUniqueStrings`, `CheckDigest`, `CheckUUIDv7`,
  `CheckTimestamp`, `StringLength`, `CheckEnvironmentID`, `CheckSemver`,
  `CheckExtensions` **gained production callers** through the delegation
  (residue closed). Still callerless and re-stated as bound:
  `DecodeTuple`/`DecodeEnvironmentObservation` (different tuple types and
  refusal dialects — sessadapter owns `Tuple` + axerror arms),
  `rawUint53`/`parseUint53Literal` (private; same-package callers exist),
  `Fault.Error` (facades translate `Detail`/`Member`, rendering pinned by
  unit test), `HasLoneSurrogateEscape` (internal funnel, frame-battery
  pinned).

## Mutant battery: 54 killed / 55 applied

Harness `/tmp/mutbattery_leaf5/run.py` (attached), full log attached,
per-mutant table attached (`mutant-table.md`). Denominator derived from
production at runtime (90). 34 narrowing (gate stays, admits exactly one
rejected member — including `T1`/`C3` token-preserving raw-scan mutants
that rebirth the Story's original admit hole verbatim), 17 arm-deletion,
3 census-only (behavioral green, census red), 1 audit-only (behavioral
green, alias audit red). `NOT_APPLIED` and `COMPILE_FAIL` are distinct
rows, proven by harness-validation mutants `H1`/`H2`.
The one survivor, `R_digestnonstr`, is predicted with a stated bound:
removing `CheckDigest`'s type arm changes no verdict because
`scalar.ParseDigest` refuses the zero value downstream; the bridge is
narrowed by `M7`/`M7n`. No unpredicted survivor, no `KILLED_OTHER`, no
billed `NOT_APPLIED`/`COMPILE_FAIL`. Two battery-driven finds are fixed
in-tree: the ladder `""` hole and the `M13n` fall-through (first version
survived — the nil member refuses at decode — corrected to skip decode).

## Gates (real exit codes)

`go build ./...` 0; `go vet ./...` 0; `GOOS=windows go vet ./...` 0;
`GOOS=windows go build ./...` 0; `gofmt -l internal/` empty;
`tracecheck` 0; `go test ./... -count=1` 0 (all 23 packages ok);
`-race` 0 on environ, sessadapter, dirnode, scalar, provhost,
canonicaljson, provider; cover environ 92.4 / sessadapter 82.0 /
dirnode 87.7 / provhost 86.0 / scalar 90.1 / invcore 70.2;
`go generate ./internal/catalog` clean; cigate contract+claims+targets
selections 0; fuzz smoke 13/13 `fuzz: elapsed`.
Bound: full-repo `-race` was not run (canonicaljson alone takes 137s
under race; untouched packages are behaviorally unaffected — no
concurrency change in this leaf).

## Changed paths (exactly 22)

Modified: `LOGBOOK.md`,
`internal/dirnode/bound_census_test.go`,
`internal/dirnode/decode.go`, `internal/dirnode/probe.go`,
`internal/dirnode/query.go`, `internal/environ/census_test.go`,
`internal/environ/decode.go`, `internal/environ/decode_unit_test.go`,
`internal/environ/shape_census_test.go`, `internal/provhost/opdecode.go`,
`internal/sessadapter/bound_census_test.go`,
`internal/sessadapter/decode.go`,
`internal/sessadapter/identity_census_test.go`,
`internal/sessadapter/manifest.go`, `internal/sessadapter/probe.go`,
`internal/sessadapter/tuple.go`.
New: `internal/dirnode/rawuint53_test.go`,
`internal/environ/census_audit_test.go`,
`internal/environ/refusal_site_audit_test.go`,
`internal/environ/scalar_agreement_test.go`,
`internal/provhost/rawuint53_test.go`,
`internal/sessadapter/rawuint53_test.go`.
No untracked ungitignored scratch file in the worktree root (checked);
battery, tables, and backups live in `/tmp/mutbattery_leaf5/`.

## Need (no board element created — final leaf is open)

Converging the provhost decoder onto `environ.DecodeStrictObject`
requires editing leaf-owned `provhost/protocol.go`. If the story wants
that convergence, it needs owner sign-off to unfreeze the file; the
frame battery + sweep keep the retained copy honest until then.
