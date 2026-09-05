# TASK-260830-32jeti round 4 — reviewer verdict: CHANGES REQUESTED

CR `CR-TASK-260830-32jeti-1` rev 1, base `57afcc6`, candidate tree
`f886873ee208e9176973c70735bfd40be69adf23`.

Working tree verified byte-identical to the candidate tree before and after
every mutant (temp-index `git write-tree` = `f886873…` both times; every
mutated file restored from a `cp` backup and sha256-verified, never
`git checkout`).

## Baseline (re-run by me, not accepted from evidence)

| Check | Result |
| --- | --- |
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l ./internal` | empty |
| `go test ./... -count=1` | 15 packages ok |
| `go test ./internal/provhost/ -race` | ok 11.751s |
| coverage | provhost 86.4%, provider 97.0% (matches the LOGBOOK claim) |

## G-A — the derived arm census

**Ratio: 162 derived arms / 162 witnessed, both directions green, across 14
production files.** The denominator is not the witness list: it is rebuilt by
`refusalArmsIn` walking every non-test `.go` file in the package with
`os.ReadDir` + `go/parser.ParseFile`. I attacked the denominator itself
rather than reading it.

- **Does the walk visit `runner_windows.go`?** Yes. `os.ReadDir` +
  `go/parser.ParseFile` ignore build constraints entirely — there is no
  `go/build` context in the path, so `//go:build windows` is invisible to the
  scan. Proven, not read: mutant **C3** planted
  `var _ = &frameFault{detail: "windows planted arm", member: "planted"}`
  into `runner_windows.go` and `TestDerivedRefusalArmsAreAllWitnessed` went
  **red** on the unwitnessed arm, while the darwin build stayed green. The
  control **C4** (same plant in `runner_unix.go`) is red the same way.
  This is structurally *unlike* leaf-1 F20: that was CI/compile blindness to
  `os_windows.go`, not walk blindness. The two failure modes are also
  distinguishable here — a file the walk **cannot parse** fails the derivation
  outright (`parser.ParseFile` error → `t.Fatalf`), a file it **does not
  visit** would drop arms and trip the reverse witness check. Neither is live.
- **Do `idempotency.go`, `identity.go`, `spawn.go` contribute witnessed arms?**
  Yes. Mutant **C2** (scan skips `identity.go`) reddens *both*
  `TestDerivedRefusalArmsAreAllWitnessed` and `TestWitnessedArmsAreAllDerived`.
  Mutant **C5** (one witness arm string corrupted) reddens both directions.
  Mutant **C8** (new inline arm in `probe.go`) reddens the forward check.
- **Was the floor raised, and can it drift?** Raised 60 → 162, which equals the
  exact derived count. It can still drift down invisibly: mutant **C1**
  (`refusalArmCensusFloor = 1`) **survives** the whole suite. Round 3's
  conclusion therefore still holds at the larger surface: the floor is a
  redundant tripwire, the witness set is what is load-bearing (C2/C5 red).
- **The round-3 wrapper shape.** Mutant **C6** replaces an inline refusal with
  a `refuseWarningOrder()` wrapper: **red on both directions** (derived count
  drops to 161, below the floor, and the witness orphans). The residual blind
  shape the LOGBOOK states — a *newly added* arm reachable only through a
  wrapper, so nothing orphans and the floor never drops — is real (**C9b**
  survives) and is stated honestly in both the file header and the LOGBOOK.

## G-B — the mutant battery

**106 narrowing mutants applied, 76 killed, 30 survived.** Two survivors
(C1, C9b) are the stated bounds confirmed above. **28 are genuine coverage
gaps.** 12 further mutants were discarded as invalid (Go rejected them for
unused variables / undefined names — a compile-error "kill" proves nothing,
so each was re-expressed in a compiling form and re-run).

Prior leaves: `internal/provider` is **untouched** by this delta, so leaf 1's
83-mutant battery cannot resurrect from it; the package is green at 97.0%.
Leaf 2's provhost surface is touched only by `doc.go` (comment-only — the file
holds exactly one non-comment line, `package provhost`) and by
`refusal_arm_inventory_test.go`, whose derivation code is **byte-identical to
HEAD** (only the header comment, the floor constant, and three witness-append
expressions changed). The header change is a *correction*: the old text
claimed non-literal constructor arguments become `expr:` obligations, which
the code never did. All 60 leaf-2 arms remain witnessed and still refuse at
the production entry.

### F1 (blocking) — closed vocabularies are not proven closed

Eight closed enums accept a silently added member with the whole suite green.
Each is a permissive-direction hole: the harness would pass a provider
advertising a value the contract never defined.

| Mutant | Widened enum | Effect |
| --- | --- | --- |
| P14 | `probeCapabilityStatuses` + `"partial"` | SURVIVED |
| P15 | `probeCapabilityEvidence` + `"assumed"` | SURVIVED |
| P16 | `probeArchitectures` + `"386"` | SURVIVED |
| P17 | `probePlatforms` + `"freebsd"` | SURVIVED |
| Q16 | `quiesceBlockers` + `"other"` | SURVIVED |
| ID1 | `identityKinds` + `"legacy_alias"` | SURVIVED |
| ID2 | `identifyConfidences` + `"guess"` | SURVIVED |
| ID3 | `identifyEvidence` + `"guessed"` | SURVIVED |

This is not a missing mechanism — it is a mechanism applied to three surfaces
and not the other eight. `TestManifestRegistriesAreDerivedFromSpec` re-reads
the pinned `SPEC.md` through `specdoc.Load()` and derives operations,
capability names and platforms from the Section 7.3 example; **MA17**
(widening `manifestPlatforms`) is duly **killed** by it.
`TestMutationOperationsAreDerivedFromSpec` (K1b, K2 killed) and
`TestProfileMappingIsPinnedToSection77` (PR1, PR3 killed) do the same for the
keyed-operation set and the Section 7.7 table. The eight enums above got a
transcription and a hand-written sample instead.

Every one of them is stated verbatim in the pinned document and is derivable
the same way — e.g. `SafeBoundaryProof.blockers` at `SPEC.md:2907`,
`identity_kind` at `:2086`, capability evidence at `:2873`, identify
confidence and `matched_evidence` at `:3075`.

The LOGBOOK's "COMPLEMENTS, NOT SAMPLES: … 4 statuses × 2 enabled swept" is
true as far as it goes and does not reach this: sweeping the four known
statuses proves the rule over those four, not that the vocabulary contains
exactly those four. A conformance harness wrong in the permissive direction
passes a non-conforming provider — the named failure mode for this leaf.

(Checked separately: the implemented `quiesceBlockers` set is *correct*. The
`background_active|input_not_blocked|process_open` vocabulary at `SPEC.md:4183`
and `:7223` belongs to `BridgeSafeBoundary`/RPC `SafeBoundary`, different
objects. No fidelity defect — only an unproven one.)

### F2 (blocking) — positional blind spots in ordered / registry loops

Round 3 closed index 0 for the two manifest *order* loops (MA2, MA3 killed by
`operations_first_substituted` / `capability_first_substituted`). The same
shape is still live at four sibling positions:

| Mutant | Loop narrowed | Effect |
| --- | --- | --- |
| MA1r | `checkManifestCapabilities` order loop skips the **last** name | SURVIVED |
| MA4r | `checkManifestOperations` order loop skips the **last** operation | SURVIVED |
| MA14r | `checkManifestPlatforms` vocabulary loop skips the **first** element | SURVIVED |
| P3r / P3s | `checkProbeCapabilities` missing-key loop skips **first** / **last** registry key | SURVIVED |

MA1r/MA4r/MA14r are permissive: a manifest whose last operation or last
capability name is wrong, or whose first platform is `"aix"`, is accepted under
the mutant and nothing reddens. P3r/P3s are attribution gaps — a probe missing
`native_resume` still refuses, but through a different arm, so the
"miss a registry key" obligation is proven only for `prompt_spawn`.

### F3 (blocking) — NUL injection into argv and env literals has no negative test

| Mutant | Gate removed | Effect |
| --- | --- | --- |
| SP4 | `strings.ContainsRune(value, 0)` in `checkSpawnArgv` | SURVIVED |
| SP5 | `strings.ContainsRune(text, 0)` in `checkSpawnEnvLiterals` | SURVIVED |

These are the two SpawnPlan members the trusted terminal backend feeds to
`exec`. The length and grammar rules around them are tested (SP7, SP11, SP15,
SP16, SP18 all killed); the NUL rule is the only clause on those members with
no fixture at all — a delete-only mutant of it is invisible.

### F4 (blocking) — identity provider correlation: same-length wrong provider unproven

**ID14** narrows `if provider != wantProviderID` to
`provider != wantProviderID && len(provider) != len(wantProviderID)` and
**SURVIVES**. `TestCheckIdentityNamesAnotherProvider` drives the Antigravity
example against `"codex"` — 11 characters against 5. `"claude"` against
`"gemini"` are both 6, and nothing distinguishes them. This is the
`invalid_config` gate that stops the host correlating one provider's identity
record to another provider's session; its only negative fixture is
length-separable.

### F5 — upper bounds with no boundary fixture (10)

Each is `bound → bound+1`, every one SURVIVED:

| Mutant | Bound | File |
| --- | --- | --- |
| Q14 | quiesce `provider_version` ≤128 | quiesce.go |
| P10 | probe `provider_version` ≤128 | probe.go |
| MA7 | provider-id grammar `{0,31}` | manifest.go |
| ID9 | opaque key grammar `{0,63}` | identity.go |
| ID10 | opaque key first char `[a-z]` → `[a-z0-9]` | identity.go |
| ID15 | identity `provider_version_range` ≤256 | identity.go |
| ID16 | identity `native_session_id` ≤512 | identity.go |
| SP6 | argv element count ≤128 | spawn.go |
| SP8 | argv total ≤65536 bytes | spawn.go |
| SP12 | env-name length ≤128 | spawn.go |

The pattern is inconsistent rather than absent: identity's `provider_version`
*has* a 129 fixture (ID17 killed) and manifest's `display_name` has one
(MA9 killed), while the same member in probe and quiesce has only the empty
lower-bound row. SP8 has no fixture on either side — the 65,536-byte argv
total is entirely unexercised.

### F6 — claims in comments and tests that nothing measures

- **K5** — `MutationOperations` says "The result is a copy; the set cannot be
  mutated through it." Returning `keyedOperations` directly instead of the
  `append([]Operation(nil), …)` copy **survives**: no test mutates the returned
  slice and re-reads. (`Capabilities()` carries the identical comment and the
  identical absence of a test.)
- **Q4b** — `DecodeQuiesceProof`'s safe-rule comment says "each conjunct below
  carries a fixture with only it violated". Neutralising `backgroundNull`
  **survives**, because `rawNullableBool` returns `false` for null, so
  `backgroundNull || !background` reduces to `!background` and no fixture can
  isolate that conjunct. The other eight conjuncts are individually killed
  (Q1b, Q2b, Q3b, Q5b, Q6b, Q7b, Q8, Q9) — the claim is true for 8 of 9 and the
  ninth is unreachable, not untested. Either drop the redundant disjunct or
  restate the comment.
- **`TestCapabilityGatePrecedesCall`** — the negative subtests construct
  `runner := &scriptRunner{}` and assert `runner.spawned() != 0`, but the runner
  is never handed to anything: `RequireCapability` takes no `Runner`. The
  assertion cannot fail. The 21-tuple sweep itself is good (derived from
  `Capabilities()` × 3 non-available statuses, count-asserted, with a real
  positive control that does spawn exactly one process), but the LOGBOOK's
  "with zero processes started" half is not measured by it — it follows
  structurally from the signature, which is a different and weaker claim.
- Minor, non-blocking: `checkSpawnArgv` and `checkSpawnEnvNames` report
  `"spawn argv is empty or longer than 128"` / `"spawn env_names exceed 64
  entries"` when the member is not an array at all. The refusal is correct, the
  detail names the wrong rule.

### What held up under attack (76 kills, selected)

The safe-lie rule (8 of 9 conjuncts individually killed), the
`enabled ⇒ available` probe rule and the pure `CapabilityUsable` gate (P1, P2),
the per-key capability value sweep at both ends (P4, P5), sorted-unique across
every array member including the first pair (P12, P13), the profile table and
both its refusal seats (PR1–PR5), the SpawnPlan mapping equality in both
directions including the standard-omission lie (SP1, SP2), env-literal
disjointness (SP3), destination-native cwd (SP13, SP14), the whole idempotency
key surface (K1b, K2, K3, K4), identity's subject/session equality, the
Antigravity realm precondition, the Windows-drive and UNC opaque-path prefixes,
and the manifest schema/semver/registry-count rules. `TestEveryArmWitness
RefusesAtTheProductionEntry` co-killed a large share of these, which is the
inventory doing exactly what it was built for.

## G-C — bounds and the logbook

The `LOGBOOK.md` entry is the mechanism, not a leaf summary: it records the AST
derivation and both directions, the census-floor reasoning, the blind shapes as
stated-not-fixed, the `DecodeQuiesceProof` defect the new fixtures caught
before review (safe:true with `open_child_count:1`), and the restated bounds.
That closes the round-3 item.

Standing bounds re-verified by me, all unchanged:

- `provhost` does not import `internal/provider` — `go list -deps` returns 0
  matches. `spawn.go` and `identity.go` do not change this; they import only
  `internal/scalar` (already a dependency), `encoding/json`, `regexp`,
  `strings`.
- `provhost` still has no importers — grep across all `.go` outside the package
  is empty.
- No top-level `cmd/`.
- `U2` (process-group kill on macOS) — nothing in the new files touches process
  teardown; `runner_windows.go` and `runner_unix.go` are unchanged from HEAD and
  no new test asserts detached-descendant knowledge.
- `S1`'s ExecRunner fork/exec red reappeared as an unrelated one-off during this
  battery (3 of ~110 full-package runs, in `TestExecRunnerWritesOneTerminatedFrame`,
  `TestExecRunnerEchoRoundTrip`, `TestExecRunnerReportsCrashExit`). It stays an
  observed environmental flake; I re-ran every affected mutant under a scoped
  `-run` mask so no verdict here rests on it.
- README does not overclaim: it says the *manifest* registries are derived from
  the pinned example and makes no derivation claim for the probe, quiescence or
  identity vocabularies. F1 is a test gap, not a README honesty failure.

## Verdict

**Changes requested.** F1 is the deciding finding: this leaf is the conformance
harness, and eight of its eleven closed vocabularies would pass a provider
advertising a value the contract never defined, with the suite green. F2, F3
and F4 are the same failure in smaller pieces — a complement asserted over a
hand-picked sample. The spec-derivation mechanism to fix F1 already exists in
this file set and is applied three times; extending it is mechanical.

Production code looks correct throughout — every mutant here narrows a rule the
implementation gets right. The gap is entirely in what the tests prove.

Suggested order: F1 (derive the eight vocabularies from `specdoc` the way
`TestManifestRegistriesAreDerivedFromSpec` does), F3 and F4 (two and one
fixtures), F2 (last/first-element substitution rows at the four loops), F5 (ten
`bound+1` rows), F6 (one aliasing test, two comment corrections, one vacuous
assertion removed or wired).

Harness, mutant specs and raw results:
`.temp/TASK-260830-32jeti/{mutate.py,b1.json,b1b.json,b2.json,b2r.json,b3.json,b4.json,b5.json,b6.json,b7.json}`
