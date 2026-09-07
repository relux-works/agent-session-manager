# TASK-260906-2okwyf — review verdict, CR rev 2

**Verdict: CHANGES REQUESTED → `to-dev`.**
`repeat-of: CR-TASK-260906-2okwyf rev 1, finding F1.` Same class, second
consecutive revision. Per the routing rule, the next step is the **instrument**
round 1 named as its preferred option, not a third comment rewrite.

Reviewed: `CR-TASK-260906-2okwyf-2` rev 2, `repository_delta: present`, base
`114a056`, candidate tree `abec4211`, 10 paths.

## 0. Provenance — VERIFIED

- Recomputed the candidate tree myself in a detached index (`GIT_INDEX_FILE`
  copy + `read-tree HEAD` + `add -A` + `write-tree`):
  `abec4211afb6b401b6614f619365d7ccf07c00e9` — **equals** the CR record and the
  outcome document. `ls-tree` shows `internal/provhost/profile_agreement_test.go`
  in-tree and the LOGBOOK append inside the tree, so the OID was taken after
  every artifact write.
- Recomputed it again after **every** plant below was reverted from a
  byte-compared backup (never `git checkout`): still `abec4211`. My scratch is
  under `.temp/` (gitignored).
- `git diff --stat 2a270726 abec4211` = 3 paths (`LOGBOOK.md`,
  `terminalbackend.go`, `terminalbackend_test.go`), 109+/4-. Round-1 files are
  **byte-identical** to the tree I verified last round, so leaves 1–3 and the
  round-1 rework carry no silent change. HEAD is still `114a056`; nothing is
  committed past the checkpoint.

## 1. What I ran myself

| # | Plant / probe | Result |
|---|---|---|
| Q1 | probe `Reconcile(manifest, probe, …)` with equal hostile IDs, digests differing → `manifest.go:1726` | **305-byte refusal, `contains(hostile)==true`** |
| Q2 | probe `Reconcile` reaching `checkProbeGeneration` → `manifest.go:1762` | **313-byte refusal, `contains(hostile)==true`** |
| Q3 | probe `DecodeProbe` (provhost) with hostile unknown key | 94 bytes, `contains==false` — producer's "clean" claim reproduced |
| Q4 | probe `secprim.BuildEnv` hostile allowlist name | **259 bytes, contains==true** — producer's number reproduced to the byte |
| Q5 | probe `environ.DecodeEnvironmentObservation` hostile key | **269 bytes, contains==true** — reproduced to the byte |
| C1 | **control plant**: delete the `ParseID` guard from `CheckProviderDescriptor` (exported entry, 5 arms name `descriptor.BackendID`) | **whole package suite GREEN**; a companion probe shows the plant produces a 308-byte refusal carrying the full hostile string |
| P-D | delete the new `CheckVersionTuple` entry check (battery `D-tb-checktuple-entry`) | new test **6/6 hostile rows FAIL** — red-before confirmed |
| P-N | length-gate the entry check (battery `N-tb-checktuple-lengthgated`) | **len27 rows FAIL, len221 rows PASS** — genuine narrowing reproduced |
| P-S | widen descriptor digest arm 1 to swallow arm 2 (battery `S-tb-checkdesc-shadow`) | audit FAILS naming **`terminalbackend.go:651`** — exact row reproduced |
| P-4 | leak an 8-byte prefix into `ParseID`'s grammar `Detail`, `BackendID` still empty | `TestParseIDGrammarRefusalEchoesNothing` + 8 census tests FAIL — round-1 narrowing pin intact |
| P-7 | `parseMajor` low bound `'0'`→`'/'` | `TestParseMajorDigitBoundariesRefuseAtEntry` + 2 witness tests FAIL — leaf-2 pin intact |
| P-8 | `parseMajor` saturation divisor `10`→`9` | `TestParseMajorSaturationEdge` FAILS — leaf-2 pin intact |
| P-1 | reorder `profileProviders` (`muse`/`antigravity`), provhost side only | **only** `TestBuiltinsEqualProfileProviders` FAILS; provider package green |

Gates I ran on the candidate tree myself: `go build ./...` exit 0,
`go vet ./...` exit 0, `gofmt -l internal` empty, `go test ./... -count=1` →
**23 ok, 0 FAIL**. I did not re-run `-race`, `-cover`, Windows cross-build/vet,
tracecheck or the fuzz smoke; those are accepted from the attached validation
log and I say so rather than implying I reproduced them.

## 2. Gate-by-gate

**G-A — is the census closed, or these three arms? FAILS. See F2.**
The census is **not an instrument**: there is no committed test anywhere in the
package that ties a `BackendID`-carrying refusal site to a validated identity
(`grep`ed `refusal_site_audit_test.go`, `refusal_arm_inventory_test.go`,
`refusal_arm_witnesses_test.go` — nothing). It is a hand-read table in the
outcome document. So the answer to "does it fail closed on a new
`BackendID`-carrying site with no validation" is **no**, and C1 measures that:
removing the validation from an exported entry with five naming arms leaves the
whole suite green while re-opening a 308-byte echo. The ratio did not become
31/31 — it is **29 of 31** through the exported entries actually available
(F2). The three round-1 arms *are* closed and their hostile vectors *are*
driven with the assertion on the **absence of the input in the rendered
message**, not on a refusal merely occurring (P-D/P-N reproduce both halves).

**G-B — the comment. FAILS. See F2.** Four of its five clauses are true and I
checked each in source. The fifth — "ParseManifest, ParseProbe, and
ParseEvidence via ParseID at document admission, so the manifest/probe/
generation/drift arms consume validated structs … so a named BackendID is
always a validated identity, never raw input" — is false at the exported
`Reconcile` entry (Q1/Q2).

**G-C — the neighbouring packages. PASS, and honestly reported.** I drove
all three myself through their production entry points with the same hostile
material. Every number the producer reported reproduces exactly: environ 269
bytes containing the key, secprim `BuildEnv` 259 bytes containing the name,
provhost `Error()` static at 94 bytes with `contains==false`. The two LIVE
sites are carried as stated bounds with entry point, escaping content and
owner named. This is the part of the round that was measured rather than
reasoned — and it is the part that came out right.

**G-D — round-1 passes have not rotted. PASS.** Round-1 files are byte-identical
to the tree I verified last round, and I re-ran the two the brief singled out:
`parseMajor`'s digit-bound and saturation pins both still kill their mutants
(P-7, P-8), and the `ParseID` narrowing proof still reddens with `BackendID`
kept empty (P-4). `TestBuiltinsEqualProfileProviders` still reddens alone on a
one-sided reorder (P-1). §6.5 is cited at `terminalbackend.go:677-683` and §7.1
at `provider.go`, each pointing at the other, with no third trust path.
`RequireCapability` decodes the body once through `decodeValidatedProbe`
(`probe.go:394`) with the two sub-object replays documented at :387-393. The
named-const bound in `digit_guard_census_test.go` is untouched this round.

**G-E — the battery. PASS.** 5 applied / 5 killed / 0 survived on a
production-derived denominator, with `NOT_APPLIED` (zero-occurrence anchor) and
`COMPILE_FAIL` (`go vet` syntax error) as distinct control rows outside the
denominator. Classes are separate and I read all seven `old`/`new` bodies
rather than the row labels: `N-tb-checktuple-lengthgated` is a true narrowing
(the gate stays and admits exactly the short grammar-hostile subclass),
`D-tb-checktuple-entry` is a real deletion, `R-tb-checktuple-order` is a real
order mutant, `C-tb-checktuple-deadarm` is honestly filed census-only add-arm,
and `S-tb-checkdesc-shadow` is a real sibling-swallow killed only by the
full-run audit. Masks are non-empty with explicit `-run` lists (census 15 ran,
behav 243 ran). I independently reproduced 3 of the 5 rows (P-D, P-N, P-S) and
both controls' semantics.

## F2 (BLOCKING) — the rewritten comment asserts a package-wide universal that is still false, and the echo class is still live at two exported arms

`internal/terminalbackend/terminalbackend.go:163-169`, in the comment this
round wrote to replace the one round 1 rejected:

> That holds package-wide by construction: every other site that names a
> BackendID validates it through ParseID or mustParseID first — … ParseManifest,
> ParseProbe, and ParseEvidence via ParseID at document admission, **so the
> manifest/probe/generation/drift arms consume validated structs** — so a named
> BackendID is **always** a validated identity, never raw input.

and the outcome document's §2, which reports the same claim as a measured
ratio: **"27 + 4 = 31/31 validated, 0 unvalidated."**

Measured on the candidate tree, driven through the exported production entry
`terminalbackend.Reconcile` (`manifest.go:1683`), which takes `Manifest` and
`Probe` structs directly and applies no `ParseID` to either:

```go
hostile := strings.Repeat("A", 200) + "\x1b[31m" + "../../etc/passwd"   // 221 bytes
// both IDs hostile-and-equal, digests differ:
terminalbackend.Reconcile(manifest, probe, nil, "gen", now, verify)
// manifest.go:1726 "executable substitution"      → len(Error())==305, contains(hostile)==true
// manifest.go:1762 "probe generation binding"     → len(Error())==313, contains(hostile)==true
```

That is the identical defect A9 named and round-1 F1 blocked on — unbounded,
unvalidated, control-byte-carrying data rendered into a refusal string —
surviving at two sibling arms while the comment above states it cannot exist.

**Why the census produced 31/31 and reality is 29/31.** The census unit is the
*site*; the property is per-*entry-path*. `manifest.go:1726` and `:1762` are
validated when reached through `AdmitProbe` (which runs `ParseProbe`/
`ParseManifest`) and unvalidated when reached through `Reconcile`. A table keyed
by site cannot represent that, so it recorded the safe path and dropped the
unsafe one. This is the same failure mode as round 1 in a new dress: the
neighbouring packages were **driven** and came out right (G-C, five probes,
every number exact); the in-package rows were **reasoned from call paths** and
came out wrong at exactly the clause where the reasoning is about who
constructs a struct. `Manifest` and `Probe` are plain exported structs with no
construction guard, so "by construction" is not a property the package has —
and the package's own tests build them by hand at 20+ call sites.

**Why there is no safety net.** C1: I deleted the `ParseID` guard from
`CheckProviderDescriptor` — an exported entry whose five arms all name
`descriptor.BackendID` — leaving the refuse lines byte-identical. The **entire
`internal/terminalbackend` suite stayed green**, while a probe showed the plant
now renders a 308-byte refusal carrying the full hostile string. The class is
pinned at exactly two places: `TestParseIDGrammarRefusalEchoesNothing` and this
round's `TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho`. Everything
else in the claimed 31 is held up by a comment.

**Why this blocks rather than rides along.** Round 1 named the durable derived
census as option 1 and, for the minimum path, required *deleting* the universal
clause. The delivered change instead widened it into a five-clause universal
and validated it with the one method — inference from call paths — that the
round-1 finding had already shown does not hold. Two consecutive same-class
findings mean the instrument is the defect, not the wording. A third comment
rewrite would be the third guess at a fact nothing measures.

Not a stop-the-line, not `blocked`: the path is clean and was already written
down last round.

### What closes it

The durable option, now the only one I will accept for this class:

A derived census test in the package's existing style
(`refusal_arm_inventory_test.go` / `refusal_site_audit_test.go`) over the
`BackendID`-carrying refusal sites, keyed by **(site, reaching exported
entry)** rather than by site alone. Every pair must either reach its refusal
with an identity that passed `ParseID`/`mustParseID`, or appear as an explicit
declared exemption with a reason. Report it as a ratio. It must fail closed on
C1 — that is the acceptance test for the instrument itself, and it is one
command to check. `Reconcile`'s two arms must come out of it either fixed
(validate at the `Reconcile` entry, as `CheckVersionTuple` now does) or as a
named exemption; not by widening what counts as "validated".

Then make the comment say what the instrument enforces, and nothing more.

Whichever shape: it needs a test that is **red before it** (C1 is a ready-made
red-before vector), and the outcome document's ratio must be produced by the
instrument, not by a hand-read table.

## Non-blocking notes

- **N1 — the outcome document's census anchors name no tree in the record.**
  Its line numbers (`terminalbackend.go:176` reserved arm, `:398` trust,
  `:643-656` descriptor) are +6 from the round-1 tree `2a270726` (`:170`,
  `:392`) and −2 from the candidate `abec4211` (`:178`, `:400`, `:645-658`).
  The *set* of 31 sites is right — I re-derived it independently by grepping
  `BackendID:` across the four non-test files — but the anchors were taken on an
  intermediate draft and never re-derived after the final comment lines landed.
  Same shape as the tree-OID discipline this leaf otherwise follows well.
- **N2 — `R-tb-checktuple-order` filed as `narrowing`.** It is an ordering
  mutant. The evidence is sound and the `expect` string describes exactly what
  it does; the class name is a stretch. Round 1's N2 (`N-ph-parsemajor-strict`
  filed narrowing while tightening) is the mirror image and still unaddressed.

## AC coverage: 4 of 4 rows driven, each with a named test I ran

| AC row | Production call site | Driving test | Verified by |
|---|---|---|---|
| two literals equal directly, either side alone reddens, SPEC-independent | `provider.Builtins()` / `provhost.profileProviders` | `TestBuiltinsEqualProfileProviders` | P-1 (this round), P1/P2 (rev 1) |
| §6.5 vs §7.1 asymmetry cited at both sites, or decision recorded | `terminalbackend.DigestFile:677`, `provider.trustCandidate` | citation; cited gates driven by `TestDigestFile`, `TestDiscoverRefusesUnapprovedOwners` | source re-read this round; SPEC clauses verified rev 1 |
| `ParseID` prints no unbounded refused input | `terminalbackend.ParseID:170` | `TestParseIDGrammarRefusalEchoesNothing` | P-4 (narrowing, this round) |
| every closed nit has a fail-before test; every open nit a stated bound | `CheckVersionTuple:598`; `RequireCapability`/`decodeValidatedProbe`; `parseMajor` bound | `TestCheckVersionTupleRefusesHostileBackendIDWithoutEcho`; `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability`; `TestParseMajorLeadingZeroIsClassifiedAsForeign` | P-D, P-N (this round); P5/P6, P9 (rev 1) |

All four AC rows pass. F2 is not an AC failure. It is a false claim introduced
by this change plus a live instance of the class the change reports as closed,
which the DoD's "attacked, not read" row and the AC's "a nit deliberately left
open carries a stated bound naming why" both reach: two arms are neither closed
nor bounded — they are asserted away.

## Scope note

`repository_delta` is `present`, so no emptiness question arises. Nothing here
asks for work outside the story boundary: `Reconcile` is the same package, same
class and same posture as the nit under review, and the sibling-package bounds
the producer recorded are correctly left to their own leaves.
