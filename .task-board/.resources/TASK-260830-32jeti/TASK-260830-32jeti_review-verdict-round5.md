# TASK-260830-32jeti — round-5 review verdict: CHANGES REQUESTED

Candidate tree `ca45ddf611c18ca58d573e6e13f5238904f109a8` (verified equal to the
CR rev-2 candidate before and after every mutant run; base `57afcc6dc501`).
Every number below was produced by this run.

**One blocking finding (F1r).** Nine closed vocabularies in `internal/provhost`
widen silently with the shipped suite green — the same class round 4 blocked on,
one level over from value enums to closed *object member sets*. Everything else
in the round is clean: the full 122-mutant battery is 120/122 with only the two
stated bounds surviving, all 28 round-4 holes are closed, and neither prior leaf
resurrected.

---

## Gates (run here, real exit codes)

| Gate | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l` | empty |
| `go test ./... -count=1` | exit 0, 15/15 packages |
| `go test ./internal/provhost -race` | exit 0 |
| coverage | provhost 86.5%, provider 97.0% |
| `tracecheck` | exit 0, clauses 17/403 unchanged |

---

## G-A — full battery replay: 120 of 122 killed

All nine batch files replanted (exact-once plant, presence check, `cp` restore,
sha256-verified byte-identical restore; zero unrestored, tree OID equal after
every batch).

| Batch | Killed | Survived | Not applied |
| --- | ---: | ---: | ---: |
| b1 (quiesce) | 15 | 0 | 0 |
| b1b (quiesce rewrites + probe) | 20 | 0 | 0 |
| b2 (manifest + profile) | 21 | 0 | 0 |
| b2r (round-3 re-expressions + vocab widenings) | 11 | 0 | 0 |
| b3 (identity) | 20 | 0 | 0 |
| b4 (spawn + keyed ops) | 24 | 0 | 1 |
| b5 (census) | 6 | **1** | 0 |
| b6 (census) | 2 | 0 | 0 |
| b7 (census) | 0 | **1** | 0 |
| K6 re-run with corrected path | 1 | 0 | 0 |
| **Total** | **120** | **2** | **0** |

Round 4 was 76 killed / 30 survived (28 real). **All 28 real survivors are
closed.** The two that remain are the two stated bounds, unchanged:

- `C1` — census floor 162 → 1 survives. The floor is a duplicate tripwire, not
  load-bearing; the witness set is what carries the census. Reasoning is
  recorded in LOGBOOK.
- `C9b` — an arm reachable only through a one-line wrapper stays invisible to
  the AST walk. Recorded in LOGBOOK (`BLIND SHAPES`, 0510 entry).

**Compile-error discipline held.** Eleven rows in my battery are compile kills
(`Q1`–`Q7`, `ID18`, `K1`, `C7`, `C9`) and every one that was re-expressed to
compile keeps its round-4 verdict: `Q1b`–`Q7b`, `K1b`, `C8` are behavioural
kills; `C9b` still survives as the stated bound. No rewritten mutant changed
verdict.

**`K6` was my spec error, not a producer gap.** My round-4 row named
`probe.go`, but `capabilityOrder` lives in `manifest.go`, so it silently
NOT_APPLIED and I reported "no aliasing test". The producer corrected the path;
re-run here it is KILLED by `TestCapabilitiesReturnsACopy`. The round-5 spec
file is byte-identical to my battery for all 29 rows except that path fix — the
mutants were not weakened.

### Is the derivation genuinely derived?

**Yes.** `specdoc.Load()` parses the `go:embed`-ed `SPEC.md` and refuses any
byte that does not hash to `specpin.DocumentSHA256`; `sectionLines` resolves the
section ID of both window bounds and returns the real line text. I dumped all
seven cited windows straight out of `internal/specdoc/SPEC.md` — each is the
actual enum sentence or table row (2862 architectures, 2869-2870 statuses,
2872-2874 evidence, 2907-2910 `SafeBoundaryProof.blockers`, 2086 identity kinds,
3073 probe request row, 3075 identify-session row). No intermediate hand-written
list anywhere in the chain.

### Does it fail closed on an empty or non-matching window?

**Yes, measured.** Three attacks on the derivation itself, all RED:

| Probe | Result |
| --- | --- |
| `V1` window repointed at a blank line inside 7.4 | RED ("spec window is empty; the check is blind") |
| `V2` window repointed at a 7.4 line with no `<code>` element | RED ("holds no code elements") |
| `V3` row marker changed to one the row does not contain | RED ("row holds no marker; the check is blind") |

Nit, not a finding: `requireVocabulary` compares with `fmt.Sprintf("%v", …)`,
which cannot distinguish `["a b"]` from `["a","b"]`. `V4`
(`probeArchitectures = []string{"amd64 arm64"}`) survives the derivation test
alone — but the full package kills it on the refusal rows, so the class is held.
Worth a `reflect.DeepEqual` when the file is next touched.

### Does the widening test enumerate from production? — No, and that is F1r

`TestClosedVocabularyWideningsRefuse` is eight hand-written subtests, and the
derivations are seven hand-written test functions. The brief asked whether a
twelfth vocabulary added tomorrow would be unguarded. It would — and this is not
hypothetical: **nine more closed vocabularies exist in production today and are
unguarded right now.**

---

## F1r (BLOCKING) — nine closed vocabularies still widen silently

Each row below is a narrowing/widening mutant that SURVIVES the shipped suite,
paired with the acceptance it produces at the production entry point. The
consequence column is measured, not inferred: I wrote the missing fixture
(shipped in the probe pack) and re-ran each mutant against it — all nine go RED,
so each really does admit a body the contract forbids.

| Mutant | Shipped suite | Production consequence |
| --- | --- | --- |
| `manifestMembers += "bogus"` | SURVIVED | `DecodeManifest` **accepts** a manifest carrying `"bogus"` |
| `probeMembers += "bogus"` | SURVIVED | `DecodeProbe` accepts |
| `probeCapabilityMembers += "bogus"` | SURVIVED | `DecodeProbe` accepts an extra capability member |
| `identityMembers += "bogus"` | SURVIVED | `CheckIdentity` accepts |
| `identifyMembers += "bogus"` | SURVIVED | `DecodeIdentifyResult` accepts |
| `quiesceMembers += "bogus"` | SURVIVED | `DecodeQuiesceProof` accepts |
| `spawnMembers += "bogus"` | SURVIVED | `DecodeSpawnPlan` accepts |
| `statusBodyMembers += "bogus"` | SURVIVED | `DecodeStatusOutcome` accepts |
| `profileYOLOMapping += "bogus"` | SURVIVED | `ProfileMapping("bogus", yolo)` returns `--yolo` instead of `invalid_config` |

The gates themselves are present and called — at baseline every one of those
bodies is refused. What is unproven is the **containment direction**: that each
member set is exactly the set the spec states. Every one of these is stated
verbatim in the pinned document (§7.3 manifest members, §7.4 probe object and
capability value, §7.5 `SafeBoundaryProof`/`SpawnPlan`/`ProviderTransactionStatus`
rows, §5.5 identity record, §7.7 provider table) and is derivable exactly the way
the eight new enums now are.

`profileYOLOMapping` deserves separate mention: its own doc comment says "there
is no default mapping a new provider could silently inherit," and
`TestProfileMappingIsPinnedToSection77` does pin the six flags verbatim — but it
counts rows against `profileProviders`, so a seventh entry in the *mapping* is
unguarded. That is an unrestricted-mode flag surface claiming a property nothing
measures.

**Measured and explicitly NOT holes** (so nobody chases them): dropping the last
entry of `manifestRequired` or `quiesceRequired` survives but is
behaviour-preserving — I drove the omitted-member body through the entry point
and it still refuses, from the unknown-member and required checks respectively.
`probeRequired` and `spawnRequired` drops are killed. Do not "fix" these.

### The fix I am asking for

Do not add nine more hand-written tests — that moves the enumeration gap from 11
to 20. Close the class by construction, one level up, exactly the way
`TestDerivedRefusalArmsAreAllWitnessed` already does for refusal arms:

1. A census that parses `internal/provhost` production source for package-level
   `[]string` / `map[string]bool` declarations consulted by a membership check,
   and requires each to carry a spec derivation. It must fail closed on an empty
   or single-file derivation, and it must fail on a vocabulary it finds with no
   registered derivation — so vocabulary number twenty is guarded the day it is
   written.
2. The nine derivations that census then forces, from the spec windows above.
3. One extra-member refusal row per closed object driven through the production
   entry (`review-round5-widening-probe_test.go.txt` in the probe pack is the
   fixture, already written and already red under all nine mutants).

Replay `probe_widen.json` afterwards: all nine must be KILLED, and `R1`/`R3`
must still survive.

---

## G-B — no resurrection in either prior leaf

### Leaf 2 (`TASK-260830-qcosxq`, provhost protocol) — 112 rows replanted

**102 detected, 10 survived — exactly the accepted round-3 set, zero new
survivors, zero resurrections.**

101 rows failed explicitly. The 102nd is `S14-no-deadline-ctx`: removing the
context deadline makes `TestCallTimesOutHungPlugin` hang, and the test binary
panics on its own timeout — I re-ran it under `-timeout=90s` and it exits 1 with
`panic: test timed out … running tests: TestCallTimesOutHungPlugin`. Detected.

The 10 survivors are `A3`, `R17a`, `D3`, `S1`, `X16`, `X9b` (six confirmed
equivalent in leaf-2 round 2) and `S11`, `S12`, `S13`, `U2` (four stated bounds
— detached-descendant and partial-write paths no test constructs). Identical
set, name for name.

### Leaf 1 (`TASK-260830-2890sd`, `internal/provider`) — untouched and re-probed

`internal/provider` subtree OID is `308a2f9f2622fbd645298717d9eb03bd0d12483a`
at both HEAD and in the worktree: **this leaf changed nothing there**, so no
resurrection is structurally possible. Re-probed anyway, all 8 round-3 rows:

- 6 RED as recorded (`F18` init-reassign, both `F19` partial-set rows, owner
  approval gate off, `Canonicalize` skipped, attestation error swallowed).
- `L1-owner-gate-off` SURVIVED — correctly. It appends `_ = err` after the
  `trustCandidate` call and changes no behaviour; it is an invalid mutant, which
  is why round 3 did not report it.
- `L1-regular-file-off` PLANT-FAILed in round 3 and again here (`if
  !info.IsRegular {` occurs twice in `provider.go`). **I closed that gap**:
  planted at both sites together and at each site separately — all three RED
  (`TestDiscoverRefusesNonRegularTargets`, `TestDiscoverEnforcesTrustGatesAcrossSources`,
  `TestVerifyDetectsSubstitution`). Both sites are individually witnessed.

---

## G-C — bounds

**`quiesceBlockers` is proven this round.** §7.5 row 3076 states quiesce returns
`{proof:SafeBoundaryProof}`; `TestQuiesceBlockersAreDerivedFromSpec` reads
SPEC.md:2907-2910, asserts the window opens with the enum owner
`SafeBoundaryProof.blockers`, and equates the remaining members to
`quiesceBlockers`. The `background_active|…|process_open` list at :4183 belongs
to `BridgeSafeBoundary` (declared at :4174) and :7223 to the RPC `SafeBoundary`
— different objects. Round 4's "spec-correct but unproven" is now proven, and
`Q16` (widening with `other`) is killed twice.

**Standing bounds, re-verified unchanged:**

- `go list -deps ./internal/provhost` contains `internal/provider` **0** times;
  `provhost` has **no** importers; there is **no** top-level `cmd/`.
- `spawn.go` imports only `encoding/json`, `strings`, `internal/scalar`;
  `identity.go` only `encoding/json`, `regexp`, `strings`. Neither changed any
  of the above.
- Wrapper blindness and the census-floor reasoning are recorded — precisely, in
  **LOGBOOK** (`BLIND SHAPES` and `CENSUS-FLOOR REASONING`, 0510 entry). The
  inventory file header carries the non-literal-conduit rationale and the
  merged-identity bound but does not name the wrapper-only-new-arm shape; my
  round-4 phrase "header and LOGBOOK" was loose. Not blocking — one sentence in
  the header would make it exact.
- `README.md` and `doc.go` claim only what the code does and state the opaque
  remainder honestly. No unsupported derivation claim.

---

## What is good in this round, plainly

- The eight new derivations are the real thing: pinned document, real windows,
  fail-closed on empty, each widening killed twice (derivation equality **and**
  a refusal row through the production entry).
- The equal-length identity fixture (`claude`/`gemini`, both 6) asserts its own
  premise and fails if the two names ever stop being the same length. That is
  the right shape, and the opposite of the failure this Story kept hitting.
- `SP8`'s lesson is recorded honestly in LOGBOOK: a bound+1 mutant is killed
  only by bound+1 *exactly*; the first draft at 69,632 bytes sat above even the
  widened bound and survived.
- Production code is correct throughout. Every mutant in this review narrows or
  widens a rule the implementation already gets right. The gap is entirely in
  what the tests prove.

## Route

`to-dev`. One finding, a bounded fix, and a construction that ends the
enumeration class rather than moving it.
