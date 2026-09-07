# TASK-260906-2okwyf — reviewer verdict, CR rev 4

**Verdict: ACCEPTED.** `repeat-of: n/a` (acceptance carries no repeat field).

Base `114a056a5338d09d990a184425b8df26dd33daeb`, candidate tree
`8bfa6a51702ad1856c5f2f68b69b6dd2a262f065`, 13 changed paths.

Every claim below is executed in this run, not read off the outcome document.
All plants were reverted and the candidate tree re-derived byte-identical at the
end of the review (`TREE INTACT`, see §7).

---

## 1. G-A (blocking last round) — scope enumeration is comparable by construction

Round 3 rejected the leaf because `probe_f2_main.go.bak` (80 lines of `package
main`, repository root, untracked and not ignored) sat inside the candidate tree
while the outcome declared three code/test paths plus LOGBOOK.

Reviewer-executed checks, not the leaf's:

- **Tree equality, not path equality.** Rebuilt the candidate from the working
  tree in a detached index (`GIT_INDEX_FILE` copy, `read-tree HEAD`, `add -A`,
  `write-tree`) → `8bfa6a51702ad1856c5f2f68b69b6dd2a262f065`, byte-equal to the
  CR record's OID. This is stronger than comparing name lists: it catches a file
  the tree carries that `git status` would not surface, which was the exact
  round-3 failure mode.
- **Delta shape.** `git diff --stat base candidate` = 13 files, +1941/−39. The
  13 paths equal the CR's `changed_paths` set with no residue in either
  direction.
- **The stray file.** `probe_f2_main.go.bak` now lives at
  `.temp/TASK-260906-2okwyf/probe_f2_main.go.bak`. Verified it was *moved*, not
  *hidden*: `git diff base candidate -- .gitignore` is empty, so `.temp/` was
  already ignored and no ignore rule was added to bury it. `git ls-tree -r` over
  the candidate carries zero `.bak` paths; no `*.go` exists at the repository
  root at all.
- **The outcome now enumerates.** §0 lists all 13 paths individually with a
  one-line provenance each, and states the two-way comparison as performed. A
  summary cannot be diffed against a tree listing; a list can, and I diffed it.

**Observation, not a finding:** nothing *automated* fails if a future stray file
appears — the two-way comparison is still a human step recorded in prose. That
gate belongs in the task-board Change Request machinery (which already computes
`changed_paths` and the candidate tree), not in a Go package inside the product
repository; building it here would be a project-local workaround for a reusable
workflow contract. Flagging it for the orchestrator as a systemic item, not
charging it to this leaf.

## 2. AC rows — 4 of 4 driven, each re-verified by reviewer plant

### AC1 — one test asserts the two six-provider literals equal directly, SPEC-independent

`internal/provhost/profile_agreement_test.go:TestBuiltinsEqualProfileProviders`
compares `provider.Builtins()` against `provhost.profileProviders` with
`reflect.DeepEqual`; it reads no `SPEC.md` line. `Builtins()` returns a copy of
`builtinOrder`, which `provider.Discover` (provider.go:345) enumerates — the
test pins the production registry, not a test-only alias.

Four reviewer plants, each a **one-sided** change:

| plant | side | killed by | full-suite collateral |
|---|---|---|---|
| reorder `codex`/`claude` | `builtinOrder` | `TestBuiltinsEqualProfileProviders` | +3 provider tests |
| reorder `codex`/`claude` | `profileProviders` | `TestBuiltinsEqualProfileProviders` | **none — this test alone** |
| rename `antigravity`→`antigravityx` | `profileProviders` | `TestBuiltinsEqualProfileProviders` | +4 spec-derived tests |
| remove `pi` | `builtinOrder` | `TestBuiltinsEqualProfileProviders` | — |

Row 2 is the substance: a reorder of `profileProviders` alone was invisible to
the entire pre-existing suite (`TestSixProviderSetMatchesDiscoveryRegistry` is
order-insensitive by design). The new test is the only thing that reddens. That
is a real hole closed, not a duplicate pin.

### AC2 — §6.5 vs §7.1 trust asymmetry cited at both sites

Verified against the spec itself, not the citation. `internal/specdoc/SPEC.md`
is byte-identical (`sha256 562546d2…`) to
`agent-session-manager-spec@28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c/SPEC.md`,
the commit named in the task scope.

- §7.1, SPEC.md:2649 — "the target MUST be a regular file **owned by the
  operator or an administrator-approved identity**".
- §6.5, SPEC.md:2596 — "Each external-trust entry contains **exactly** backend
  ID, absolute executable path, executable digest, and `enabled`."
- `grep -n "owned by|administrator-approved"` over the whole spec returns
  **one** executable-ownership clause (line 2649, §7.1). Line 772 is the tmux
  runtime *directory*, a different object. So §6.5 genuinely carries no owner
  dimension, and the asymmetry is contractual rather than drift.

Both sites carry the citation and point at each other:
`provider.go:391-401` (trustCandidate) and `terminalbackend.go:678-692`
(DigestFile), each naming the other function by name and forbidding unification
without a spec change. AC row satisfied by citation; no decision record needed.

### AC3 — ParseID prints no unbounded refused input

`terminalbackend.go:170` — the grammar arm dropped `BackendID: value`. The bound
arm never carried it. The reserved-namespace arm still names its input, and that
is correct and bounded: to reach it a value has already passed
`^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$` and the 128-byte limit, so it can carry no
escape, no traversal, no control byte, and no unbounded length.

Reviewer plant: restoring `BackendID: value` on the grammar arm reddens
`TestParseIDGrammarRefusalEchoesNothing`,
`TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho`, **and**
`TestBackendIDEntryCensus`.

Consumer surface checked, not assumed: `internal/config/validation.go:660,694`
is the only production caller outside the package, and it **discards** the
ParseID error, returning `configError("terminal.backend_id", ErrConfigValidation)`
— a static path label. The no-echo property survives at the config surface
rather than being undone one frame up.

### AC4 — every closed nit has a fail-before test; every open nit a stated bound

Closed nits, each re-killed by a reviewer plant on this tree:

| nit | plant | verdict |
|---|---|---|
| A9-1 ParseID echo | restore `BackendID: value` | KILLED (3 tests) |
| A9-3 RequireCapability double decode | move the registry-membership check ahead of `decodeValidatedProbe` | KILLED by `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability` **alone** |
| r2 CheckVersionTuple gate | delete the entry `ParseID` | KILLED |
| r3 Reconcile gate | delete both entry `ParseID` calls | KILLED |
| r3 Reconcile gate | **narrow** to `len > maxIDBytes` only | KILLED behaviourally at `len27`; inventory census SURVIVED |
| N1 site filter | restore the retired line-text filter | KILLED by `TestBackendIDSiteFilterSeesMultilineSpelling` |
| N2 entry index | restore the bare-name dedupe | KILLED by `TestExportedEntryDerivationFailsClosedOnCollision` |

The narrowing row is the one that matters: with the length gate in place the
221-byte hostile is still refused and only the 27-byte grammar-refused ID slips
through — one member of the class — and the census names it exactly:
`Reconcile(hostile)/substitution: BackendID = "BAD ID\x1b[31m../../etc/passwd"`
plus `Error() renders the refused input (len 111)`. The inventory/census-only
masks stay green under it, so the kill is behavioural, not a text-keyed
coincidence.

## 3. Stated bounds — each verified true, not accepted as prose

### parseMajor leading zeros (deliberately loose)

Five checkable claims in the doc comment, all four load-bearing ones verified:

1. *"parseMajor never admits a version"* — the gate is `version != ProtocolVersion`
   (protocol.go:515), plain string equality against `"2.0.0"`. Confirmed.
2. *"`02.0.0` already lands unusable"* — major 2 == `ProtocolMajor`, so the
   foreign-major arm does not arm and it falls to `unsupported protocol version`.
   Confirmed at the third call site too (protocol.go:578, `observedMajor`): only
   an exactly-equal `"2.0.0"` reaches that line, so leading zeros cannot select a
   different error contract.
3. *"a leading-zero rejection would spell an equality against `'0'`, which the
   digit-guard census rejects as unclassifiable"* — **reviewer-planted the strict
   spelling** (`len(parts[0]) > 1 && parts[0][0] == '0'`). Result:
   `TestDigitCensusCoversEveryLeafGuard` FAIL —
   `unclassifiable guard in protocol.go (parseMajor)` — plus
   `TestDerivedRefusalArmsAreAllWitnessed` FAIL for the unwitnessed new parse
   arm. The "census churn for zero admission gain" claim is measured, not
   reasoned.
4. The bound is *pinned*, not merely commented:
   `TestParseMajorLeadingZeroIsClassifiedAsForeign` drives `DecodeResponse` and
   separates the two classes by **code and exit** (`incompatible_protocol`/6 vs
   `provider_protocol_error`/13), which is what keeps an input from silently
   moving between arms.

### Named-rune digit spelling (DoD item 7 — the sentence that was false)

The old text claimed a named-constant gate "would classify as other and fail as
unclassifiable, not pass silently". Both halves of the correction
control-planted by the reviewer in `internal/provhost/protocol.go`:

| plant | result |
|---|---|
| pure named-const chain (`c < zeroConst \|\| c > nineConst`) | both packages **green** — genuine blind spot, old sentence was false |
| mixed chain (`c < zeroConst \|\| c > '9'`) | FAIL: `mixes digit comparisons with unrelated comparisons` |

The absence claim also re-derived: zero digit-valued named constants across the
non-test sources of both packages. The corrected sentence is accurate in both
directions and labels the absence a stated bound rather than a derived pin.

### N3 — 44 free-entry reasons unchecked (deliberately open)

Reproduced with an **independent** instrument rather than trusting the leaf's:
built an intra-package call graph over-approximated by bare callee name (so
every method sharing a name is a successor — deliberately too generous) and did
BFS from each of the 44 free entries toward the 11 functions containing a
BackendID-naming arm.

    free entries: 44 | naming functions: 11 | free entries reaching a BackendID-naming function: 0 []

The file's wording — "Sound by measurement, not by gate" — is the correct
epistemic claim for this. Accepted as an open bound.

### N4 — entry arithmetic

Derived independently in-tree rather than counting table rows by eye:

    derived exported entries: 59 | pair-entries 10 | hostile-only 5 | free 44 | sum 59 | overlaps p/h=0 p/f=0 h/f=0
    derived BackendID sites: 31 | pair rows 41 | site exemptions 1 | pair exemptions 1

59 = 10 + 5 + 44 with zero overlap. The census logs `41/41 pairs validated, 1
site exemptions plus 1 pair exemptions pinned`. Round 3's "58 / 43" is
superseded and the LOGBOOK `rev3`→`rev1` typo is fixed.

## 4. Attacking the instrument itself

The census is only worth its ratio if its derivation fails closed. Five reviewer
plants against the AST derivation, on the **live production tree** (the leaf's
own derivation tests use a synthetic `t.TempDir()` copy; these do not):

| plant | result |
|---|---|
| new exported entry reaching existing `checkProbeIdentity` arms unvalidated | FAIL — `derived exported entry ReviewPlantReconcileIdentity has no census battery and no free reason` |
| brand-new **multi-line** BackendID arm in `manifest.go` | FAIL — `derived BackendID site …\|ReviewPlantMultilineArm\|CodeUntrusted\|… has no census pair and no exemption` |
| same multi-line arm **under the retired line-text filter** | site silently absent from the denominator — the N1 hole, reproduced on the real tree |
| positional literal `refuse(&Error{CodeUntrusted, id, "…"})` | FAIL — `refusal without a static code and detail` |
| `e := &Error{…}; refuse(e)` | FAIL — `&Error literal outside refuse()` |
| `var alias = refuse; alias(&Error{…})` | FAIL — `&Error literal outside refuse()` |

The last three are the identifier-matched-AST-gate bypasses that have defeated
gates of this shape before. All three fail closed here, so the N1 comment's
completeness clause ("positional literals cannot reach the flag; `refuse` takes
exactly one `&Error` literal, never a variable") is verified rather than
asserted.

The hostile batteries drive real exported entries (`ParseID`,
`RegisterExternal`, `Resolve`, `RequireRestoreBinding`, `CheckVersionTuple`,
`CheckProviderDescriptor`, `AdmitProviderDescriptor`, `AdmitProbe`, `Reconcile`,
`New`) with two hostile shapes each and assert BackendID empty **and** the bytes
absent from the rendered `Error()`. Not positive-path evidence.

## 5. Battery — honest, and the survivor is honestly a survivor

Round 4: 5 applied / 4 killed / 1 survived. Classes separate (narrowing 2,
arm-deletion 2, census-only 1). `NOT_APPLIED` and `COMPILE_FAIL` are distinct
control rows outside the denominator, with real detail (`anchor occurs 0 times`;
`vet: provider.go:366:20: expected ')'`). Masks non-empty and reported per row
(`census 15`, `behav 3…32`). Mutant **bodies** carried in `mutants-r4.json`, and
I read the bodies rather than the labels: every class label matches what the
diff actually does, including `D-tb-sitefilter-flag`'s `if false` (correctly
filed arm-deletion — it removes the flag read — not add-arm).

Continuity on this tree: r1 9/9, r2 5/5, r3 6/6 = 20/20 killed, 0 survived,
`NOT_APPLIED`×3 and `COMPILE_FAIL`×3 distinct. Cumulative **24 of 25 killed on a
production-derived denominator**, one declared survivor.

The survivor `D-tb-entryindex-enforce` (drop the fail-closed loop in
`deriveExportedEntries`) — **reproduced by the reviewer, SURVIVED, matching the
declaration.** The reason it survives is structural: the pinning test drives
`collectExportedEntries` (which returns collisions) while the enforcement lives
in `deriveExportedEntries` (which `t.Fatalf`s on them). Go offers no clean way
to assert that a `t.Fatalf` fires from inside the same suite, and the gate is
vacuous today (0 repeats among 59 bare names). Reported with its bound and the
P2-plant evidence rather than dressed up as a kill — that is the correct
disposition, and the residual risk is confined to test-instrument code.

## 6. Gates re-run in this review

| gate | result |
|---|---|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `GOOS=windows go build ./...` | exit 0 |
| `gofmt -l internal/` | clean |
| `go test ./...` | 0 failures |
| `go test -race` tb/ph/pv | exit 0 |
| `go test -cover` tb/ph/pv | 95.4% / 86.0% / 97.8% — matches the outcome exactly |

CI runs `go test ./... -v -count=1` unmasked (ci.yml:115), so the census tests
execute in CI rather than being selected out by a `-run` filter.

## 7. Provenance

- Candidate tree in the outcome document == the CR record's ==
  `8bfa6a51702ad1856c5f2f68b69b6dd2a262f065`, re-derived from the working tree
  by the reviewer.
- After every plant above was reverted, the tree was re-derived: **TREE INTACT**,
  same OID, `git status --short` shows exactly the 13 declared paths. Per-file
  `git hash-object` vs `git rev-parse <tree>:<path>` checked for every touched
  file — zero drift.
- Leaves 1–3 (`114a056`, `5da63ad`, `1d97474`) untouched: the delta is additive
  gates, comments and tests; the only production removals are the
  `BackendID: value` echo and the duplicate probe-body decode, both intended.
  All 20 continuity mutants from those leaves still kill.

## 8. Verdict

Accepted. Every AC row is driven through a production entry point by a named
test; every gate carries a narrowing mutant, not just a deletion; every open
bound is measured and labelled as measured rather than inferred; the one battery
survivor is declared with its evidence instead of being hidden; and the round-3
blocking finding is closed by removal from the tree, verified by tree OID rather
than by name list.

Reviewer plants executed this run: **20** (4 cross-package one-sided, 2
digit-guard control, 1 parseMajor strictness, 1 RequireCapability ordering, 5
production gate deletions/narrowings, 6 AST-derivation bypasses, 1 retired-filter
narrowing) — all with the expected verdict, all reverted byte-identical.
