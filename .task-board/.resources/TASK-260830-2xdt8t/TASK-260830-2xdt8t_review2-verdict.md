# TASK-260830-2xdt8t — review round 2 verdict: CHANGES REQUESTED

Reviewer run `RUN-260904-ffc916` (Opus 5). Producer: muse-spark, rework commit
`9e31f14`. Change Request `CR-TASK-260830-2xdt8t-2` revision 2.

## Repository delta

`repository_delta=empty` is a snapshot artifact, again: base OID
`9e31f141b2e759bb64802ea76b846a0b1a5652c7` **is** `HEAD`, and the candidate tree
`18493ec4` is `HEAD^{tree}`, so the recorded patch diffs the delivered commit
against itself and has zero paths. The real reviewable delta is commit `9e31f14`
itself: 8 files, +440/-43, signed (`Good "git" signature for oparin@me.com`,
ECDSA `SHA256:V6JiKG7J…`), with `b5cf5b5` (the round-1 reviewed commit) as its
ancestor, worktree clean, not pushed. The branch now stands **two** commits past
checkpoint `57afcc6`; that is a delivery-shape fact for the orchestrator, not a
review finding.

Baseline before any mutation: `go test ./...` exit 0, 14 packages, 32.8s
(`.temp/TASK-260830-2xdt8t/r2-baseline-test.log`); `gofmt -l internal/` empty;
`go vet ./...` exit 0; `tracecheck` and `tracecheck -section 6.2` both exit 0
with `acceptance_cases=74 clauses_discharged=17/403` — the figures the producer
claimed did not move, confirmed by running it.

## Verdict

**CHANGES REQUESTED → `to-dev`.** No production defect was found this round: every
guard behaves correctly at every input I drove. All three blocking findings below
are defects in *evidence and claims*, of the exact classes this board blocked on
in round 1 — an inert pin that names a property it cannot detect (round-1 F3), and
a "fully narrowed" claim the coverage does not support (round-1 F2/F4).

## Measured mutation ratio — 59 measured, 52 killed, 7 survived

61 mutants defined, 59 measured (1 not applied, 1 build-error and therefore no
measurement). Harness: byte-copy backup, occurrence-count-==1 presence check
before substitution, mutant-present/original-absent check after substitution,
restore with byte-equality assertion between every mutant. Working tree verified
clean (`git status --porcelain` empty) after every batch. Raw results:
`TASK-260830-2xdt8t_review2-mutation-results.json`; harness:
`TASK-260830-2xdt8t_review2-mutation-harness.py.txt`.

| Batch | Applied | Killed | Survived |
| --- | ---: | ---: | ---: |
| Producer's claimed 13 closures + F1's three directions | 15 | 14 | 1 |
| Neighbours of the quoted mutants | 17 | 11 | 6 |
| Regression sample of round-1 KILLED mutants | 22 | 22 | 0 |
| Half-fix / compiling-variant controls | 3 | 3 | 0 |
| Ownership-registry self-mint controls | 2 | 2 | 0 |

### Board question 1 — is the producer's 13/13 honest?

Yes for the 12 round-1 survivors, no for the F1 class. I re-ran every one of the
12 survivors round 1 measured: M08, M12, M13, M16, M20, M21, M22, M23, M24b, M30
and the M28 deletion variant are all **KILLED** now (R04–R15). M28-revert and M29
are correctly declared equivalent/subsumed — M29 (256→100000) is the same
comparison as M16, and I killed it separately anyway (N03).

The regression sample is clean: 22 of the 20 round-1 KILLED mutants plus five new
guards (identity binding, restore binding, absolute path, implementation-version
semver, platform default) were re-applied and **all 22 stayed killed**. Nothing
was weakened to close a survivor. M32 (deleting the restore gate) is a build error
and therefore no measurement; its compiling variant M32b is killed.

The F1 class is where the ratio does not cover what the artifact claims — see G1.

### Board question 3 — F5

Discharged properly, and I checked it by running rather than reading.
`TestDecodeRefusesNonConptyBackendOnNativeWindows` is registered on the
`config-versioned-readers` acceptance case in `ownership.v0.5.0.json`, the
projection digest was re-pinned, and README names both arms. The negative arm is
real: narrowing the guard to WSL2 (N18) reddens exactly that test. The registration
is not self-mintable: a declaration that does not exist in the named file is
refused by tracecheck (T01), and adding a *real* extra declaration without
re-pinning the digest reddens four traceability tests with an explicit
`projection digest … differs from reviewed` message (T02). Figures unmoved (74 /
17-of-403), confirmed by running `tracecheck`.

### Board question 4 — F1's second direction

This is where the rework is short. Egress and the `RegisterExternal` ingress are
genuinely fixed and pinned member-wise, not just wholesale: reverting the egress
clone (R01), reverting the ingress clone (R02), and two *half*-fix controls that
clone only one of the two slice members (R01b, R02b) are all killed. The third
direction is not — see G1.

---

## BLOCKING G1 — the "built-ins do not share a protocol array" subtest cannot fail

`New` was changed to give each built-in its own `cloneStrings(protocols)` copy, and
`TestRegistryCopiesSlicesAcrossItsBoundary` carries a subtest named
`built-ins do not share a protocol array`. **Reverting both clones so the two
built-ins share one backing array leaves the entire suite green** (mutant R03,
`ProtocolVersions: cloneStrings(protocols)` → `protocols` on both built-ins,
presence-checked). Run in isolation with the mutant applied, the subtest that
names the property still reports `--- PASS`.

The reason is structural, and it is the round-1 F3 shape exactly: the subtest
reaches the built-ins through `Resolve`, which now clones on egress, so it mutates
a copy and can never observe what the registry holds. The pin exercises the
accessor, not its subject.

Two claims ride on it and neither is supported:

- the production comment at `internal/terminalbackend/terminalbackend.go:388-389`
  — "sharing one backing array would let a caller holding one record mutate the
  other" — is false as written once `Resolve` clones. No caller can reach that
  array at all, from any exported entry point I could find (`Resolve` clones,
  `RegisterExternal` clones, `IDs` returns strings, and `New`'s `protocols` is
  already a copy of the caller's input). The `New` clone is defence-in-depth, not
  a bypass fix.
- `TASK-260830-2xdt8t_rework-verification.md` lists exactly one F1 mutant
  ("`Resolve` returns stored record") inside its 13/13. Two of F1's three named
  sub-bugs were never measured; one of the two is unmeasurable by the test that
  claims it.

Close it either way, but close it honestly: make it a white-box pin in
`internal_pin_test.go` that compares the two stored records' backing arrays
directly (the package-internal test file already exists for exactly this kind of
unreachable-through-production guard), **or** delete the inert subtest and state
the bound — with `Resolve` and `RegisterExternal` cloning, built-in array sharing
is unreachable from outside the package, so the `New` copy is defence-in-depth.
Do not leave a subtest whose name asserts something the suite cannot detect.

## BLOCKING G2 — `validatePlatforms` is not "fully narrowed": the admission boundary is unmeasured

LOGBOOK 2210 and the rework artifact both state `validatePlatforms` was "fully
narrowed (dup/empty/unknown/5-list)". The widening direction is now genuinely
covered — `>4`→`>5` (R11) and `>4`→`>6` (N02) are killed, as are the sorted-unique
(R09) and non-empty (R10) arms. **The narrowing direction is not: `>4`→`>3`
survives** (N01), because nothing in the suite ever registers a four-platform
backend.

I confirmed by probe that this is a coverage gap and not a live refusal bug: a
record with `[linux macos windows wsl2]` is admitted today and resolves with all
four platforms intact. So the upper bound is currently correct and nothing proves
it. That is exactly the asymmetry the same commit avoided one guard over — the
protocol bound got its 32-member admission case, and `32`→`31` is killed (N06).
Add the analogous four-platform admission case, or restate the claim.

## BLOCKING G3 — the trust reserved-namespace guard is covered at 0 of its 1 distinguishing value

`TrustEntry.validate` refuses any `ax.`-prefixed trust entry with
`terminal_backend_ambiguous / external_trust reserved namespace`
(`terminalbackend.go:347-349`). Narrowing it to `entry.BackendID == BuiltinTmux`
**survives** (N16).

The reason matters: `mustParseID` two lines earlier already refuses every `ax.*`
ID except the two canonical built-ins (probe: `ax.evil` →
`terminal_backend_not_found / terminal_backend_id reserved namespace`). So the set
of inputs on which this guard is the *only* thing standing is exactly
`{ax.tmux, ax.conpty}`, the tests feed only `ax.tmux`, and `ax.conpty` — the one
remaining value — is fed by nothing. Under the mutant, an external trust entry
claiming `ax.conpty` stops being refused at the trust gate and falls through to a
`terminal_backend_implementation_drift` on the existing built-in record. Still
refused, so this is not a live bypass; it is a refusal guard proven on none of its
distinguishing domain, which is the "1 of 752 pairs" shape this board named. One
test case closes it.

---

## Non-blocking, disclosed

- **N15 — `native_runtime` never reaches the external-kind gate in any test.**
  Widening `RegisterExternal`'s external-kind gate to admit `KindNativeRuntime`
  survives. Only `bogus_kind` (outside the vocabulary) is tested, so the gate's
  real class boundary — in-vocabulary but not externally admissible — is
  unmeasured. Not a bypass: probed both ways, a `native_runtime` record is still
  refused downstream by the digest binding; only the reported clause changes.
- **N13 — `CheckVersionTuple` membership is not narrowed.** Widening exact
  membership to `strings.HasPrefix(member, protocolVersion)` survives; the
  distinguishing input is a prerelease list member (`["1.0.0-beta"]` admitting
  `"1.0.0"`), which no case supplies. Probed: refused today. `CheckVersionTuple`
  has no production caller, so this is API-surface coverage for the sibling tasks.
- **N09 — closed `implementation_kind` vocabulary, stated bound.** Admitting one
  extra plausible member (`builtin_rust`) survives; the narrowing direction is
  covered (dropping `native_runtime` is killed, N10). A closed vocabulary cannot
  be pinned against its own complement by sampling, and there is no pinned
  `implementation_kind` set anywhere in `internal/catalog` to cross-check against
  — I grepped. Stated as a bound, not reported as covered.
- **N14 — equivalent mutant, not a gap.** Deleting the `len(value) < 1` arm from
  `ParseID` survives because `idPattern` requires a leading `[a-z]` and refuses
  the empty string anyway. Only the refusal clause differs.
- Round 1's two non-blocking items are unchanged and still open by design:
  `canonicaljson.requireTerminalBackendID` carries its own grammar without the
  `ax.` reservation (disclosed at README:577-582), and
  `Registration.validate`'s digest-must-be-null arm remains unreachable from both
  production entry points — now measured white-box with the reachability bound
  stated, which is the right handling.

## Adjudicated decisions — unchanged

D1 (`required_capabilities` keeps its empty default) and D2 (§6.2 not extended to
third-party IDs) were upheld in round 1 and are not relitigated. The D1 LOGBOOK
correction was applied as asked: the SPEC.md:2604-2605 appeal is withdrawn and the
entry now records an underspecified clause with empty as the only non-inventing
default, disclosed as the weaker gate.

## DoD items unchecked

- item 3 (README/traceability updated without unsupported claims) — README and the
  ownership registry are correct; the LOGBOOK 2210 "fully narrowed" claim is not
  (G2).
- item 6 (refusal behaviour covered by negative tests that fail when the gate
  admits what it must reject) — G1, G3, N15, N13.

No production code was modified by this review. Every mutant and probe was
reverted; `git status --porcelain` is empty and `HEAD` is unchanged at `9e31f14`.
