# TASK-260906-33xcnc rev1 — review verdict: CHANGES REQUESTED

Reviewer run `RUN-260907-fb5fc8`. Candidate tree `5063b78a32fd32c25d4bd3f1bcc911951c0c8900`
verified by my own detached-index `write-tree` = the CR record, 22 paths, exactly the
CR's `changed_paths` in both directions. Every measurement below was produced by this
run in this worktree; nothing is accepted from the producer's report on trust.

**repeat-of:** F3 repeats the denominator-basis finding raised on TASK-260830-3bkz0c
rev4 (`&Fault{}` returns excluded from the refusal-exit denominator). F1 and F2 are new.

## Blocking findings

### F1 — the leaf's own convergence gate is defeated by a token-preserving decoy, and the facade then diverges from the shared owner in the permissive direction

`pinsCheckDelegation` (`internal/environ/shape_census_test.go:1044`) requires only that a
delegated helper's body *reference* the identifier `environ`. `TestCensusScopeHasNoAliasedSharedRules`
(`internal/environ/census_audit_test.go:92`) requires only that the reference sit in
direct-call position. A discarded direct call satisfies both while the helper regrows a
full local copy of the rule.

Plant, in `internal/sessadapter/decode.go`:

```go
func checkDigest(raw json.RawMessage) (scalar.Digest, bool) {
	_, _ = environ.CheckDigest(raw)                       // decoy: preserves the searched-for token
	value, ok := rawString(raw)
	if !ok {
		return scalar.Digest{}, false
	}
	digest, err := scalar.ParseDigest(strings.ToLower(value))  // drift
	if err != nil {
		return scalar.Digest{}, false
	}
	return digest, true
}
```

- `go test ./... -count=1` → **23 packages ok, whole repository green.**
- Green through `TestCheckHelpersDelegateToEnviron`, `TestDelegatingWrappersCallEnviron`,
  `TestCensusScopeHasNoAliasedSharedRules`, `TestSharedShapesAreLedgered`, and
  `TestTupleAgreementAcrossFacades`.
- Reachable and divergent through production entries:

  | `store_schema_fingerprint` | `sessadapter.DecodeTuple` | `environ.DecodeTuple` |
  | --- | --- | --- |
  | `sha256:abab…` (lowercase) | admits | admits |
  | `sha256:ABAB…` (uppercase) | **admits** | refuses — `environment tuple store fingerprint is not a digest` |

The facade is more permissive than the owner. That is the exact defect class this library
exists to prevent, reproduced against the gates this CR added to prevent it.

Without the drift (`ParseDigest(value)`, behaviour identical) the suite is also green, so
the provenance gate — the only thing that can see a regrown copy — is what fails.

DoD row not satisfied: *"A gate that inspects source text is additionally attacked by a
mutant that PRESERVES the searched-for token and changes behavior."* The battery ships
`T1`/`C3` as token-preserving raw-scan mutants against the surrogate walk, but ships no
token-preserving mutant against either of the two structural delegation gates this leaf
added.

### F2 — the retained provhost frame decoder's stated justification is measurably false

Outcome, AC row 1: *"provhost decoder + surrogate gate: … pinned both directions by the
frame battery (exact verdicts)."*

**Surrogate gate: the claim holds.** I confirmed both directions myself (see the plant
table below).

**Frame decoder duplicate-member arm: the claim does not hold.** The frame battery has
exactly one duplicate row (`frame_agreement_test.go:255`, body `{"v":1,"v":2}`), so the
arm is witnessed at one key. Narrowing sweep `duplicate` → `duplicate && key != K` on
`internal/provhost/protocol.go:295`, judged by `go test ./internal/provhost/ ./internal/environ/`:

| key | result |
| --- | --- |
| `body` | **SURVIVED** |
| `ok` | **SURVIVED** |
| `protocol_version` | **SURVIVED** |
| `error` | **SURVIVED** |
| `capabilities` | **SURVIVED** |
| `provider_id` | **SURVIVED** |
| `request_id` | KILLED |
| `v` | KILLED |

6 of 8 real frame member names survive. Reachable exploit, with `key != "body"` and the
**full repository suite green**, driven through `provhost.DecodeResponse`:

```
duplicate body     -> err=<nil> body={"b":2}      # last-wins: the host takes the SECOND body
control single     -> err=<nil> body={"a":1}
control dup "ok"   -> err=…: duplicate member
```

The producer's own narrowing mutant `M15n_framedupnarrow` is *"duplicate arm admits
exactly a duplicated v"* — aimed at the one witnessed member. It reports the witness, not
the class, and `M15_framedup` (arm-deletion) already proved the witness exists.

The same class reaches the shared owner. Sweeping `environ.DecodeStrictObject:84` across
`provhost`+`environ`+`sessadapter`+`dirnode`: **7 of 10 survived** (`ok`,
`protocol_version`, `request_id`, `error`, `capabilities`, `provider_id`, `cursor`), 3
killed (`body`, `v`, `environment_id`) — and the `body` kill lands in a dirnode test by
coincidence of member naming, not by design.

Either converge the provhost decoder onto `environ.DecodeStrictObject`, or make the
justification true by witnessing the duplicate class rather than one key. Note also that
"frozen by accepted leaves 2/4" is not a board mechanic: leaves 1–4 are `integrating` with
their work already at `cd8591d`, so a leaf-5 edit to `provhost/protocol.go` creates a new
delta and cannot disturb an accepted revision. The retention may still be the right call —
but it needs a real reason, and the pin has to hold.

### F3 — G-A: the re-derived denominator uses the basis a prior review already corrected

The arithmetic reproduces exactly under my own count:

| component | producer | my count |
| --- | ---: | ---: |
| direct `refuse(` call sites (`observation.go` 21 + `tuple.go` 10) | 31 | 31 |
| bool-false exits (`decode.go` 40 + `observation.go` 18 + `tuple.go` 1) | 59 | 59 |
| **stated denominator** | **90** | 90 |
| `&Fault{…}` returns in `DecodeStrictObject` (decode.go 60,63,68,72,78,82,85,89,94,96,99) | *excluded* | **11** |
| **denominator counting every exit kind** | — | **101** |

`89 + 11 = 100` is exactly the "27 of 100" figure the brief carries. The outcome calls 100
*"two stories stale"*; it is not stale, it is the **corrected** value of the same
inventory, and 90 is the uncorrected basis with the ladder guard added. The tell the prior
finding named holds again: `N1`, `M15`, `M15n`, `M16`, `M17`, `M17n` all weaken exits that
sit inside the excluded 11 — the numerator counts kills over exits the denominator does not
contain.

Restate as 101 (or 105 counting the bool `return <expr>` arms: `CheckEnvironmentID`,
`CheckSemver`, `isNull`, and `decode.go:239`), or state explicitly which exit kinds the
denominator counts and why.

## Non-blocking

**N1 — G-C is closed, and the bound never needed to exist.** I ran the full-repo race
detector myself in two bounded calls: 20 packages in 45s, then
`canonicaljson`+`localstore`+`tracecheck` in 2m05. **All 23 packages green.** The stated
bound — *"canonicaljson alone takes 137s under race"* — does not support itself:
canonicaljson was already inside the 7 packages that were run, so the cost cited was a
cost already paid, and the 16 remaining packages cost 45 seconds. The disclosure was the
right behaviour; the reasoning behind it was not.

**N2 — G-D: the predicted survivor's bound is a rationale, not a proven property.**
`R_digestnonstr` rests on *"scalar.ParseDigest refuses the zero value downstream"*.
`internal/scalar/scalar_test.go:158-164` drives five negative digest vectors and `""` is
not among them. If `ParseDigest("")` ever started admitting, both the removed arm and the
live gate would open silently and nothing would redden. Pin `ParseDigest("")` or restate
the survivor as unbounded.

**N3 — the stale tree is in the note, not the document.** `8dbf62d7…`/21 paths appears in
the board handoff note; the outcome document names `5063b78a…` over 22 paths and says it
was recomputed after the logbook append. `git diff 8dbf62d7 5063b78a` is `LOGBOOK.md`
only, +9 lines — so no measurement in the document was computed against a tree differing
in any code path. Correction confirmed; nothing else in the document is affected.

**N4 — `refuse` became a mutable package-level `var` to serve an instrument.** Production
now carries a swappable global on the refusal funnel. It is package-private and swapped
only in `TestMain` before `m.Run()`, so it is not a race (confirmed: environ green under
`-race`). Noted as production surface bought for a test, not as a defect.

## What I attacked and what held

Every row below is a plant I applied and reverted in this worktree, judged by running the
named suite.

| Gate | Plant | Result |
| --- | --- | --- |
| scalar retained copy — permissive | `code >= 0xdc01 && code <= 0xdfff` | **RED** — `bare_low_refused`, `escaped_backslash_plus_real_lone_low_refused` |
| scalar retained copy — strict | `low > 0xddff` | **RED** — `paired_surrogates_admit`, `escaped_backslash_plus_real_pair_admits` |
| provhost surrogate — permissive | `unit > lowSurrogateMin` | **RED** — frame battery + derived sweep |
| provhost surrogate — strict | `low > 0xddff` | **RED** — frame battery + derived sweep |
| canonicaljson — permissive | `codeUnit > 0xdc00` | **RED** — frame battery + own recursive-shape test |
| canonicaljson — strict | `second > 0xddff` | **RED** — frame battery |
| refusal-site audit — unexercised production site | new `refuse("tuple carries an impossible member count", "")` arm | **RED** — names `tuple.go:135`, closed rule set 39→40 |
| refusal-site audit — alias | `deny := refuse` | **RED** — `tuple.go:117:11 constructor "refuse" referenced outside direct-call position` |
| census — var-binding alias | `var measureAlias = environ.StringLength` | **RED** — both local and qualified spellings reported |
| census — fresh-name copy | `versionShapePattern` + `checkVersionShape`, wired into `DecodeProbe` | **RED** — *unregistered grammar copy under a fresh name* |
| census — orphan row | ledger row for a site production no longer carries | **RED** — *orphan row with no derived site* |
| provhost duplicate arm — narrowing × 8 keys | `duplicate && key != K` | **6 SURVIVED** (F2) |
| environ duplicate arm — narrowing × 10 keys | `duplicate && key != K` | **7 SURVIVED** (F2) |
| delegation pin — token-preserving decoy | `_, _ = environ.CheckDigest(raw)` + regrown copy | **SURVIVED**, behaviourally divergent (F1) |

Gates run clean on the untouched candidate: `go build ./...` 0, `go vet ./...` 0,
`gofmt -l internal/` empty, `go test ./... -count=1` 23 ok (54s), `go test ./... -race`
23 ok. No untracked ungitignored file beyond the 6 new test files the CR lists; worktree
root clean; my own probes and backups live in gitignored `.temp/review-33xcnc/`.

## AC coverage as measured: 4 of 6 rows

| AC row | Verdict | Basis |
| --- | --- | --- |
| provhost frame decoder + surrogate gate converged or justified | **NOT SATISFIED** | surrogate gate pinned both ways; frame decoder's duplicate arm pinned at 2 of 8 keys with a reachable last-wins exploit (F2) |
| canonicaljson frame gate converged or justified | satisfied | both directions reddened by my plants |
| scalar third-spelling gate unified or justified | satisfied | both directions reddened by my plants; 15-vector battery is bidirectional by construction |
| sessadapter/dirnode per-helper copies converged | **NOT SATISFIED** | converged, but the convergence gate is bypassed by a decoy and the bypass diverges permissively (F1) |
| sibling rawUint53 trailing texture hardened | satisfied | three siblings carry the arm; `S1`–`S6` include narrowing, not deletion only; ladder empty-guard added in all three |
| F2 residue exits attacked or re-stated | satisfied with defect | 31/31 sites exercised, audit fails closed under my plant; callerless closed for 10 symbols, verified by grep; denominator basis is wrong (F3) |

## Route

`to-dev`. F1 and F2 are the same underlying shape at two levels — a gate whose class is
closed at the single member its witness happens to name. F3 is a repeat and should be
answered by restating the basis, not by recounting the same two components.
