# TASK-260906-v8heil rev2 — review verdict: CHANGES REQUESTED

Reviewer run: RUN-260907-e3b260. Change Request `CR-TASK-260906-v8heil-2`
revision 2, base `5da63ad`, candidate tree `dd555ca4`.

repeat-of: none (revisions 1 and 2 are twins over an unchanged tree; this is
the first review of this element).

## Verdict summary

The shared core is real and the alias-bypass union is genuinely preserved — I
proved that by planting the bypasses into **real production**, not by reading
the tests. G-A, G-C, G-D (counts), G-F all pass.

The blocking problem is G-B. terminalbackend's executed witnesses assert
`(code, detail)`, not the arm. A production mutant that makes two declared,
witnessed arms **dead by construction** leaves the whole package green: the
derived arm set is byte-identical, the bijection holds at 210/210, and all
three witnesses pass — through a sibling. That is the class this leaf exists
to close, and two artifacts assert it as closed.

## Provenance (G-F) — PASS

| Check | Result |
|---|---|
| `HEAD` | `5da63ad3…` = CR base |
| Candidate tree recomputed (detached `GIT_INDEX_FILE`, `read-tree HEAD` + `add -A` + `write-tree`) | `dd555ca42507259365151126fa042f113aef2441` = CR record |
| `ls-tree -r` carries the new files | `internal/invcore/{invcore,must,invcore_test}.go`, `internal/terminalbackend/refusal_arm_witnesses_test.go` — all present |
| Leaves 1 and 2 untouched | no production file in the delta; tree recomputed identical after every plant below |

Gates I re-ran myself this turn: `go vet ./...` clean, `gofmt -l internal cmd`
empty, `go build ./...` clean, `go test ./... -count=1` exit 0, 23 packages ok
0 FAIL. `-race`, `-cover`, tracecheck, cigate, `go generate` accepted from the
producer's attached evidence, not rerun.

---

## F1 (BLOCKING) — an executed witness does not attribute the arm; sibling-swallow survives green

**Plant P8**, `internal/terminalbackend/conformance.go` `CheckEntrypoint`, one
token:

```go
-	if err != nil {
+	if err != nil || sessionID != argv[2] {
 		return &Error{Code: CodePreconditionFailed, Detail: "entrypoint session binding"}
 	}
```

Sites `conformance.go:713` (`#2`) and `conformance.go:716` (`#3`) become
unreachable — every input that used to reach them now refuses at `#1`, carrying
the same `(CodePreconditionFailed, "entrypoint session binding")`.

Result: `go test ./internal/terminalbackend/ -count=1` → **ok**. Nothing
reddened.

- `TestDerivedArmsAreAllWitnessed` — green (both sites still exist in source).
- `TestWitnessedArmsAreAllDerived` — green (keys unchanged).
- `TestEveryDeclaredArmRefusesAtItsEntry/…#2` and `…#3` — **PASS**, resolving
  through `#1`.

Two artifacts state the opposite as fact:

- `internal/terminalbackend/refusal_arm_inventory_test.go:24-27`: *"Attribution
  drift cannot survive that: a dead-by-construction arm has no input that
  refuses at its entry, so it cannot carry a witness and must declare a bound
  instead."*
- outcome §2: *"requires the refusal to come from that arm"*.

The attribution actually rests on the **input construction** of each prove
("the input passes every earlier same-clause site"). That argument is sound
against today's production shape and is void the moment an earlier same-clause
guard widens — which no test observes.

**Measured scope of the residual** (from the 184 executed subtest keys):

| Population | Count |
|---:|---|
| Witnessed arms | 184 |
| Clause-unique — `(code, detail)` occurs once package-wide, machine-attributed | 122 |
| Clause-shared — `(code, detail)` shared with ≥1 sibling arm, attributed by construction only | **62** |
| …of those, same `(file, function)` (occurrence-indexed) | 19 |
| Distinct shared clauses | 20 (largest: `CodeMismatch/"document_member_type"` ×11) |

`refusal_arm_witnesses_test.go:10-14` does disclose the shared-clause
construction, but the header sentence ends *"so the witness attributes the
site, not just the clause"* — which is the claim P8 falsifies. And
`boundUnreachableNumbers` (`refusal_arm_inventory_test.go:437-441`) identifies
exactly this residual for **one** row and says it is *"measured by the
arm-deletion mutants in the battery"*. See F2: it is not.

Deletion **is** caught — I verified separately (plant P4, arm removed →
`orphan row`, `TestEveryDeclaredArmRefusesAtItsEntry/…#3` FAIL). Only
shadowing is invisible.

## F2 (BLOCKING) — the battery's `C-tb-dead-arm` row measures a different class and reads KILLED

`muts_tb.json` → `C-tb-dead-arm` inserts a **new** function with a **new**
detail:

```go
func deadArmProbe() *Error {
	if false { return &Error{Code: CodeNotFound, Detail: "dead arm probe"} }
	return nil
}
```

That is an *added undeclared arm*, and its recorded killers are
`TestDerivedRefusalArmsAreAllDeclared|TestDeclaredRefusalArmsAreAllDerived` —
the static declared↔derived bijection. I reproduced the same kill with my own
add-arm plant P3. It says nothing about an existing declared, witnessed arm
going dead.

The provhost and provider rows use the same add-a-dead-function shape, but for
them the kill lands on the **derived-site-without-exercised-path** direction —
`inventory_test.go:139-152`, over runtime `file:line` recorded by
`invcore.SiteRecorder` — which *would* also catch an existing arm going dead.
terminalbackend has no analogue: it derives `(file, function, code, detail,
occurrence)` with **no line and no runtime site recording**, so the direction
that kills the class in the other two packages does not exist there.

Net: the split table reports terminalbackend dead-arm coverage that the harness
does not have, on a kill for the wrong reason. Under this repo's own standard a
row that kills for a different reason than the class it names is not evidence
for that class.

## F3 — the digit-guard N-R1 stated bound rests on a false premise

`internal/terminalbackend/digit_guard_census_test.go:63-67` states:

> strconv-delegated admission (`strconv.Atoi` + error branch) is OUTSIDE: it is
> not a comparison at all… **No such site exists in either package today.**

The census scans `internal/terminalbackend` **and** `internal/provhost`.
`internal/provhost/opdecode.go:67-86` (`rawUint53`) is exactly that shape:
`strconv.ParseUint(literal, 10, 64)` + error branch + magnitude bound
`parsed > maxUint53`.

I checked whether this is an open hole and it is not — plant P10
(`parsed > maxUint53` → `parsed > maxUint53+1`, admitting exactly 2^53) reddens
`TestDecodeQuiesceRefusals/count_overflow`. So the gate is behaviourally
pinned. What is wrong is the **absence claim**: the bound tells the next reader
that no live site of this spelling exists, when one does, so a second such site
would be admitted under the same false reassurance. The code-point half (item 1,
ENUMERATED, pinned by `TestDigitGuardDigitRuneAdmitsCodePointSpellings`,
narrowing mutant `N-digit-int-spelling` KILLED) and the named-rune half (item 2,
verified: no named digit-rune constant in either package) are both fine.

## F4 (minor) — `TestShadowedLookupsHaveNoInput` is a name census, not a row census

`refusal_arm_inventory_test.go:1115-1197` hand-lists 19 `(list, member)` pairs
and asserts only `contains(list, member)`. Nothing ties that list to the **9**
`boundShadowedLookup` declared rows, in either direction: a tenth such row
needs no pin, and no pair names the production site it is meant to shadow. The
9 rows are the largest bound group; their pin is the weakest.

## F5 (minor) — dead code in the core

- `invcore.go:369-379`: a loop over `spec.Qualified` that computes `hasLocal`
  and throws it away with `_ = hasLocal`. Vestigial; delete it or finish it.
- `ParseSource` has zero callers repo-wide (only `ParseBytes` is used).

## F6 (minor, factual) — "zero production edits / every M path ends in `_test.go`"

outcome §Scope. `internal/invcore/invcore.go` and `internal/invcore/must.go` are
**not** `_test.go` files — this is a new non-test package that imports
`testing`. Nothing production imports it so no behaviour ships, and the idiom is
standard (`net/http/httptest`), but the statement as written is inaccurate and
the `M`-path argument does not cover the two new non-test files.

## Scope ratio — report it, do not reframe it

AC/DoD row 1 says *"every existing inventory is expressed through it"*.
Measured: **12 of 50** AST-walking test files, **3 of 13** packages. 38 files
across 9 packages (`canonicaljson`, `cliresult`, `config`, `dirnode`,
`environ`, `localstore`, `secprim`, `sessadapter`, `specdoc`) still hand-roll
`os.ReadDir` + `parser.ParseFile`.

I **accept** the narrowing — the task description names exactly the three
refusal-arm architectures and asks for the union of those — but outcome §Scope
calls the review's count *"stale"* and asserts the in-scope set instead of
reporting the ratio. State it as `3 of 13 packages / 12 of 50 walker files
ported; the other 9 packages are out of scope by <reason>`.

---

## What passed, measured

### G-A — the union is kept. Verified against real production, not fixtures.

| Plant | Where | Result |
|---|---|---|
| P1 local-constructor alias `var aliasedMismatch = mismatchf` | `internal/terminalbackend/manifest.go` | **KILLED** — `manifest.go:272:23: constructor "mismatchf" referenced outside direct-call position` in 4 tests |
| P2 import alias `import errs "errors"` + `var mintPlain = errs.New` | `internal/terminalbackend/terminalbackend.go` | **KILLED** — `terminalbackend.go:386:22: constructor "errs.New" referenced outside direct-call position`; the alias resolved through the import path, which is the shape that has walked through every identifier-keyed gate in this repo |
| P3 unregistered site (new arm, existing code) | `internal/terminalbackend/conformance.go` | **KILLED** — `unregistered site with no declaring row` + unwitnessed arm |
| P4 orphan row (production arm deleted) | `internal/terminalbackend/conformance.go` | **KILLED** — `orphan row with no derived site` + `…#3` witness FAIL |
| dot import (unclassifiable) | `invcore_test.go` control plant | fails closed with a dot-import diagnostic |

`ScanProduction` is directory-derived and fails closed on zero files, unreadable
files and unparseable files; `DiffSets` fails closed on an empty derivation.
provider's and provhost's runtime `SiteRecorder` direction is preserved
one-for-one (`sync.Map` → `invcore.SiteRecorder`, `runtime.Caller(2)` skip
convention pinned by `TestRecorderAttributesProductionFrame`).

### G-C — both harness defects are fixed

`Errorf*` prefix stem has a committed pin (`TestQualifiedWatchesAdmitsPrefix
Spellings`) and its narrowing mutant `N-core-errorf-prefix` is KILLED with the
exact-`Errorf` case named in the evidence. Every battery record carries a `ran`
count on both masks; the minimum is 1 and no mask is empty.

### G-D — counts

| Package | Base `5da63ad` | Worktree | Delta |
|---|---:|---:|---|
| terminalbackend declared rows | 210 | 210 | 0 |
| terminalbackend witnessed / bound | textual (210 rows, each naming a test) | 184 / 26 | resolution ported |
| provhost derived arms | 167 | 167 | 0 |
| provider sites | 18 | 18 | 0 |

Verified 210 at base by `git show 5da63ad:…refusal_arm_inventory_test.go`
(210 `{file:` rows, each with a `refusalTestRef` — that is the textual
resolution this leaf removes) and 210 now (209 single-line + 1 multi-line),
26 of them bound. 184 executed subtests counted from `-v` output. No floor
regresses.

### G-B partial — the 26 bounds

Each of the 8 bound categories carries a written rationale naming a pin, and
the rationales are careful — `boundUnreachableKind`, `boundDecoderContract`
and `boundUnreachableDigestNull` each name the mutation that would promote the
row back to a witnessed arm. I did not find a bound that means "no witness
yet". The weak one is `boundShadowedLookup` (F4).

---

## What to fix

1. **F1/F2 together.** Pick one:
   - (a) give terminalbackend the site direction the other two packages have —
     runtime attribution of the production site behind each refusal. Needs a
     production seam (`&Error{…}` literals have none), so it is a scope call,
     not a test-only change; or
   - (b) keep the clause-level bound and make it honest: correct
     `refusal_arm_inventory_test.go:24-27` and outcome §2, replace
     `boundUnreachableNumbers`' "measured by the arm-deletion mutants" with the
     truth, report the measured **122 clause-unique / 62 clause-shared** split,
     and add the sibling-swallow mutant (P8 above, or any equivalent) to the
     battery as its own row recorded **SURVIVED** with the class named. A class
     the harness cannot see is a stated bound; a class reported KILLED on
     another row's kill is not.
   Whichever you pick, `C-tb-dead-arm` must stop being counted as dead-arm
   coverage for terminalbackend.
2. **F3** — delete the false "no such site exists" clause and name
   `internal/provhost/opdecode.go:67-86` explicitly, with its behavioural pin
   (`TestDecodeQuiesceRefusals/count_overflow`), or widen the census shape to
   enumerate delegated admission.
3. **F4** — derive the shadowed-lookup pin from the `boundShadowedLookup` rows
   (both directions) instead of hand-listing pairs.
4. **F5, F6** — delete the dead loop and `ParseSource` (or use them); correct
   the "every M path ends in `_test.go`" sentence.
5. **Scope** — report `3 of 13 packages / 12 of 50 walker files` as a ratio
   with the reason the other 9 are out of scope.

## Reproduction

Every plant was applied to the worktree, run, and reverted from a byte copy
taken before the edit (`.temp/TASK-260906-v8heil-review/backup/`), never with
`git checkout`. The candidate tree was recomputed as
`dd555ca42507259365151126fa042f113aef2441` after the last revert and again
after the full-suite run; the delivered tree is untouched.
