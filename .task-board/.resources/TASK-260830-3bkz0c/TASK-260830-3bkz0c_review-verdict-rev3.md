# TASK-260830-3bkz0c — review verdict, CR rev3 (`story_final`), round 1

- Verdict: **changes requested** → `to-dev`
- repeat-of: `none` (rev1 and rev2 never reached a reviewer; they failed CR
  validation at command 4/18)
- Reviewer run: RUN-260906-bd76f6
- Candidate: base `1cb6b93` → tree `292bc4242de751821d59ee1e3e68c9f1670b4634`,
  head `1296ecc`, `repository_delta=present`
- Tree OID re-verified `292bc4242…` **after** every probe and mutant below.

## What holds

| Claim | Probe I ran | Result |
| --- | --- | ---: |
| candidate == worktree head | `git rev-parse HEAD^{tree}` | `292bc4242…`, clean |
| frozen leaves untouched | `git diff --stat d5ad5f6 1296ecc` | LOGBOOK + `internal/environ` only |
| suite green at head | `go test ./... -count=1` | 19/19 packages ok |
| build / vet / fmt | `go build ./...`, `go vet ./...`, `gofmt -l` | exit 0; `gofmt` flags only gitignored `.temp/` scratch |
| environ coverage | `go test -cover` | 76.1%, reproduced |
| landability, mechanical | `git fetch origin main` | `origin/main == 1cb6b93 == CR base`; HEAD 3 ahead; fast-forward clean, no rebase |
| tuple agreement bidirectional | read `tuple_agreement_test.go` | yes — both judges must accept, or both refuse carrying the rule phrase |
| frame agreement bidirectional | T1 mutant, below | yes — a judge that flips its answer reddens its row |
| T1 token-preserving mutant is real | I re-applied it myself | census stays **green**, **7** behavioural rows redden (producer claimed 5; the two `*_escape_falls_to_syntax` rows also flip). Kill is behavioural, not nominal. |

## B1 (blocking) — the census detects a known *name*, not a new *copy*

`census_test.go:24-27` claims: *"A new copy of a shared rule — a fourth facade
`decodeStrictObject`, **a third surrogate-gate spelling**, **a new string-measure
helper** — fails here as an unregistered site."* Two of the three named shapes do
not fire. I control-planted seven shapes into `internal/provider`, an **in-scope**
census package, and ran `TestSharedImplementationsAreCensused` +
`TestSharedGrammarsAreOneLanguage` on each:

| # | Planted shape (in `internal/provider`) | Caught |
| --- | --- | ---: |
| A | `func decodeStrictObject` — ledgered name | **YES** |
| C3 | method `func (surrogateScanner) hasLoneSurrogateEscape` — ledgered name | **YES** |
| B1 | `func parseStrictObject` — a genuine fourth strict decoder, fresh spelling | **NO** |
| B2 | `func scanForLoneSurrogate` — a third surrogate-gate spelling | **NO** |
| B3 | `func measureString` — a new **byte**-counting string measure | **NO** |
| C1 | `var stringLength = byteLength` — ledgered name, byte measure, `var` binding | **NO** |
| C2 | `var decodeStrict = <alias-bound closure>` — ledgered name, `var` binding | **NO** |

5 of 7 pass. `scanSharedSymbols` inspects `*ast.FuncDecl` names against a fixed
literal table (`sharedFunctionSymbols`), so anything that is not a func/method
declaration carrying an already-known identifier is invisible. B3 and C1 are the
damaging ones: a **byte**-counting measure — the exact §1.6 drift this Story
exists to prevent — lands in a scanned package and the census stays green.

Consequence for the AC ledger: boundary.md row 1 ("One environment library;
census forbids new copies", named test `TestSharedImplementationsAreCensused`) is
**not driven**. Reported AC coverage is **5 of 7 rows driven**, not 7 of 7 (row 1
fails under control plant; row 7 — see F1; row 6 is a legitimate stated bound).

What would close it: derive the denominator from *behaviour or shape*, not from a
name table (e.g. reject any in-scope function whose body reaches
`utf8.RuneCount*`/`len()` on a bound member, or any second JSON entry point), or
narrow the claim in the file comment to exactly what the scan can see and state
the rest as an explicit bound. A name registry described as a copy census is the
unsupported-capability shape.

## B2 (blocking) — the ledgered divergence is **not** over-strictness only

The CR, the LOGBOOK entry, and `boundary.md` all characterise the recorded
divergence one way: raw-scan facades **refuse**, string-walk judges **accept**.
G-B asked whether there is a direction where a facade **admits** what another
refuses. There is.

Witness — one wire value, `"\\ud800\udc00"`: an escaped backslash, the **literal
text** `ud800`, then a **real** `\udc00` escape, i.e. a lone LOW surrogate.

| Judge | Production entry | Verdict |
| --- | --- | ---: |
| environ | `DecodeStrictObject` | refuse — `lone surrogate escape` |
| canonicaljson | `Canonicalize` | refuse |
| provhost | `DecodeManifest` | refuse — `lone surrogate escape` |
| **sessadapter** | `DecodeTuple` | **ADMIT** — `Version = "\\ud800\uFFFD"` |
| **dirnode** | `CheckScanRequest` | **ADMIT** |

Mechanism: the raw scan skips the first byte of the `\\` pair, then reads the
literal text `ud800` at the *second* backslash as a high-surrogate escape and
pairs it with the following real `\udc00`. Both frozen facades therefore admit a
body whose decoded value `encoding/json` has silently rewritten to U+FFFD — the
exact harm `internal/environ/decode.go:104-108` names as the gate's reason to
exist. Found by differential fuzz (400k vectors, seed 20260906) against
`HasLoneSurrogateEscape`, then reduced to the deterministic witness above.

Why the battery is green over it: `frameCorpus()` samples backslash-run parity
(run 1 / 2 / 3) but never composes an **even run with a following real escape**.
`escaped backslash high run2` is `A\\ud800B` — literal text, nothing after it. The
class was sampled on the over-strict side only and the safe conclusion was
inferred from that sample. That is the "property inferred from an incomplete
sample, reported as established" shape, and it is the specific defect this leaf
was created to catch — leaf 1's one-directional
`TestSurrogateGateAgreesWithCanonicalJSON` is named in the brief for hiding
exactly this class, and the replacement battery hides the more dangerous half of it.

To close: add the composed shape to `frameCorpus()` with the real per-judge
verdicts, correct the ledger text in `census_test.go`, `doc.go`, `boundary.md`
and the LOGBOOK entry to say the divergence is **bidirectional and includes an
admission hole**, and — B2b — record the frozen-scanner unification as a **board
element**. It currently exists only as prose ("Genuinely-required frozen-package
change (not made)") in `boundary.md` and the LOGBOOK; there is no board item for
it. The freeze is a legitimate reason not to fix the scanners in this leaf; it is
not a reason to ledger the class as harmless.

## F1 (must fix, non-blocking on its own) — the library is behind nothing, and says otherwise

No file outside `internal/environ` imports it. The only occurrence of the import
path anywhere else in the tree is inside an error string in `census_test.go:241`.
Six production functions sit at **0.0%** statement coverage — no production
caller and no test:

| Symbol | Callers in tree |
| --- | ---: |
| `CheckUint53Bounds` (exported) | 0 |
| `CheckSortedUniqueStrings` (exported) | 0 |
| `rawUint53`, `parseUint53Literal` | reachable only via the above |
| `Fault.Error()` (exported) | 0 |
| `validCapability` | **0 anywhere** — dead, incl. inside its own package |

That is most of the 23.9% coverage gap. Two specifics:

- `CheckSortedUniqueStrings`'s doc comment states *"an unsorted-but-unique vector
  and a sorted-but-duplicated vector are separate rows in the observation
  battery, so a check that enforced only one half could not pass both."* Those
  rows exist for `CheckSortedUniqueDigests`, not for this function. The comment
  asserts coverage the function does not have.
- `validCapability`'s comment claims *"It is a table loop so the census derives it
  like every other closed vocabulary"* — it is in no census table and is called
  from nowhere.

Also: boundary.md row 1 names `environ.DecodeStrictObject` as the "production
call site". It has no production caller. Wiring the facades onto `environ` was
correctly out of scope under the freeze — so say that as a stated bound rather
than presenting an uncalled function as a production call site. Delete or drive
the dead exports; a library that advertises API nothing reaches is the AC-row-7
shape ("no unsupported capability advertised") turned inward.

## F2 (finding) — the mutation ratio is killed-over-applied on a self-chosen denominator

Reported: "Applied 7, killed 7. Narrowing kills 6 of 6… Survivors: none. Every
surviving-mutant bound is therefore vacuous; no bound needs stating."

Re-derived from `internal/environ` production source: **31** `refuse(...)` arms
(`decode.go` 1, `tuple.go` 10, `observation.go` 21) and **58** boolean gate exits
returning `false` — **89 refusal exits**. Six narrowing mutants is **6 of 89
(6.7%)**, not 6 of 6. For scale, the two sibling leaves ran 144 and 185 mutants
against comparable surfaces. The DoD line "Every gate ships at least one NARROWING
mutant" is not met, and "no bound needs stating" is the inverted claim: the 83
unattacked exits *are* the bound and must be stated as one.

Untouched whole classes include every `checkCapabilityResult` arm except the
`!=`→`<` length check (N4), both `CheckSortedUnique*` orderings, every
`Check{Digest,UUIDv7,Timestamp}` bridge, the `rawUint53` magnitude ladder, and the
reason-coherence pair at `observation.go:359-364`.

## Notes, reported as unknown rather than inferred

- **rev1/rev2 command 4/18.** Not recoverable from the attached artifacts: the
  harness caps the log at 64 KiB and drops 4.73 MB **from the middle** — exactly
  where the failure was. The producer's root cause (a `canonicaljson`
  wall-clock flake, `TestTransferManifestMaximumEntryGateIsLinear`) is consistent
  with its own `reverify.md`, but I could not confirm it from the artifact. I can
  confirm the suite is green at the committed head; I cannot confirm what was red
  in rev1/rev2. Worth raising against the CR harness separately — a validation log
  that truncates its own failure is not evidence.
- `GOOS=windows go vet` and `-race` are producer claims; I did not rerun them.

## Story landability (`story_final`)

**Mechanically:** yes. `origin/main` is still `1cb6b93` — identical to the CR base
— HEAD is 3 commits ahead, fast-forward clean with no rebase, all 19 packages
green, `go build`/`go vet` exit 0. Nothing here would redden `main` on first push.

**On merit: no.** B2 means the Story would land two protocol hosts that admit a
lone low surrogate on their wire and hand the caller bytes `encoding/json`
silently rewrote — while its own final leaf, whose entire purpose is to catch that
class, ledgers it as harmless over-strictness and ships a green battery
asserting the measurement was made. Landing it puts a live admission hole into
`main` **under a boundary that claims to have checked for it**, which is worse
than landing it under no boundary at all. Close B1 and B2 before integration.
