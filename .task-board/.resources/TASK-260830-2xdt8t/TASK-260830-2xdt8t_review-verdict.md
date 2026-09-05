# TASK-260830-2xdt8t — reviewer verdict: CHANGES REQUESTED

Reviewer run `RUN-260904-cd73df` (Opus 5). Change Request `CR-TASK-260830-2xdt8t-1`
revision 1. Verdict: **changes requested → `to-dev`**.

## 0. What was actually reviewed, and the `repository_delta=empty` question

The Change Request snapshot reports `repository_delta=empty` with a zero-path
patch (`sha256:e3b0c442…855`, the empty-string digest). That is a snapshot
artifact, not a producer failure, and I verified it rather than accepting the
prompt's assertion:

```
git rev-parse HEAD^{tree}                 -> 8ec7a392e176c1dd25795b3be2d4a302213b5588
git rev-parse b5cf5b5…244^{tree}          -> 8ec7a392e176c1dd25795b3be2d4a302213b5588   (candidate == base == HEAD)
git show --stat b5cf5b5                   -> 5 files changed, 1379 insertions(+), 6 deletions(-)
git verify-commit b5cf5b5                 -> Good "git" signature for oparin@me.com (ECDSA SHA256:V6JiKG7J…)
git status --short                        -> (clean)
git log --oneline -2                      -> b5cf5b5 (this leaf) on 57afcc6 (checkpoint)
```

The base OID **is** this leaf's commit, so the snapshot diffed the delivered
commit against itself. The real reviewable delta is `b5cf5b5`: a signed commit,
exactly one past the checkpoint, clean worktree, not pushed — the delivery shape
this non-final leaf was asked for. **The producer did change repository files.**
Everything below is judged against `b5cf5b5`.

Baseline before any mutation: `go test ./...` green, 14 packages, 28.4s wall.

## 1. Verdict summary

Five blocking findings. Two of them (F1, F3) are the exact negative-evidence
shapes this board handed the producer by name in the task notes.

| # | Finding | Shape |
| --- | --- | --- |
| F1 | Registry hands out and retains live interior state; the **drift gate is bypassable** | accessor does not preserve validation |
| F2 | Drift "one member at a time" claim is false for 2 of 5 record members | pin exercises the length, not the member |
| F3 | `TestCheckProviderDescriptorGenerationBounds` proves nothing about the bound it names | pin exercises the parameter, not the subject |
| F4 | Four guards have **zero** negative coverage; 12 of 32 mutants survive | unmeasured refusal domain |
| F5 | §6.2 negative arm is bound to nothing; ownership registry and README still claim the positive-only discharge | stale capability claim |

Both producer decisions the board asked me to adjudicate are **upheld** (§3).
The `no new ownership claimed` posture is **mostly honest** (§4).

## 2. Blocking findings

### F1 — the deliverable's drift gate is bypassable through the package's own accessor

`Registration` carries two slices (`ProtocolVersions`, `Platforms`).
`Registry.Resolve` returns the struct by value, so the slice **headers** are
copied and the **backing arrays are shared with the registry**.
`RegisterExternal` stores the caller's `observed` record verbatim, so the caller
keeps a live handle on an admitted record. `New` sorts one protocol slice and
assigns it to **both** built-ins, so they share one array.

Confirmed, three ways, driving the real exported API
(`.temp/TASK-260830-2xdt8t/review-aliasing-probe.log`):

```
before: platforms=[linux macos wsl2] protocols=[1.0.0]
after : platforms=[windows macos wsl2] protocols=[9.9.9]
ALIASING CONFIRMED: registry platform list mutated through Resolve() accessor
ALIASING CONFIRMED: registry protocol list mutated through Resolve() accessor

admitted record after caller mutation: platforms=[windows] protocols=[1.99.0]
POST-ADMISSION DRIFT CONFIRMED: caller mutated an admitted record without passing any gate

conpty protocols after mutating tmux's list: [1.5.0 1.1.0]
SHARED BACKING ARRAY CONFIRMED: both built-ins share one protocol slice
```

This is not cosmetic. `equalRecord` compares the candidate against the *stored*
record, and the stored record is the same array the caller holds, so an
attacker-influenced adapter that mutates in place converts real drift into a
benign duplicate. Proven end-to-end
(`.temp/TASK-260830-2xdt8t/zz_review_probe2_test.go`):

```
DRIFT GATE BYPASSED: a changed protocol list reported as duplicate, not drift:
  terminal backend refused: terminal_backend_ambiguous for com.example.term at duplicate backend_id
registry now believes it admitted protocols=[1.4.0]
```

`terminal_backend_implementation_drift` is the wire code this task exists to
emit. It emitted `terminal_backend_ambiguous` instead, and the registry silently
adopted the changed protocol list as its admitted record.

The board's task notes name this class verbatim: *"Validation that does not
survive the package's own accessors is not validation — a preceding Story
shipped a shallow copy whose accessor handed out live interior state."*
It is also an architecture-fit failure: this repository already has the
convention and does not use it here — `internal/catalog/catalog.go` clones every
accessor return (`cloneSource`, `cloneContracts`, `cloneOperation`,
`cloneCapability`, `cloneEvent`, `cloneError`), `internal/specpin/pin.go` has
`cloneContracts`/`cloneContract`, and `internal/config/validation.go` — the file
this leaf edited — has `cloneStrings` and `cloneAnyMap`.

Fix: deep-copy on ingress in `New`/`RegisterExternal` and on egress in
`Resolve`, and pin it with a test that mutates every returned slice and
re-resolves. `IDs()` is already safe (freshly built).

### F2 — the drift gate is unproven for two of the five members it compares

The producer's board note claims *"drift changes one record member at a time"*.
Two of the five do not:

- **M12** — delete the `ProtocolVersions` member-by-member loop from
  `equalRecord`: **full suite stays green.**
- **M13** — delete the `Platforms` member-by-member loop: **full suite stays green.**

`TestDriftIsRefused`'s `protocol list` case goes `["1.0.0"] → ["1.0.0","1.1.0"]`
and its `platforms` case goes `[linux] → [linux macos]`. Both change the slice
**length**, so only the `len(...) != len(...)` guard is exercised; the element
comparison is dead weight as far as the suite is concerned. Same-length,
different-member drift — the realistic adapter-swap case, and precisely what F1
exploits — has no coverage.

The three scalar members are genuinely pinned: M09 (drop `Kind`), M10 (drop
`ImplementationVersion`), M11 (drop `ExecutableDigest`) are all killed.

Fix: add same-length drift cases (`["1.0.0"] → ["1.1.0"]`, `[linux] → [macos]`).

### F3 — the §7.A generation bound test proves the mismatch, not the bound

`TestCheckProviderDescriptorGenerationBounds`'s doc comment claims it *"proves
the 1..256 bound in both directions against the number the specification
declares: 256 accepted, 0 and 257 refused, on either side of the comparison."*
It proves none of that. All four refusal cases pit a mutated generation against
`base.Generation = atLimit`, so the values **differ** and
`descriptor.Generation != binding.Generation` returns
`terminal_backend_stale_generation` on its own — the bound never has to fire.

- **M16** — widen `checkGeneration` upper bound 256 → 257: **survives.**
- **M29** — widen it 256 → 100000: **survives.**
- **M30** — delete the `len(generation) < 1` lower bound entirely: **survives.**

`string[1..256]` therefore has zero test evidence, and the test's own comment is
an unsupported claim about itself. The board's standing bar: *"a pin over a
documented claim must exercise the claim's SUBJECT, not only its parameter."*

Fix: assert the bound where the two generations are **equal** (a 257-byte
generation on both sides must still be refused; a 256-byte one on both sides
must be admitted — the latter case already exists and is the only sound one).

### F4 — four guards with no negative coverage; measured mutation ratio 20/32

Reported as a ratio, not as prose. 32 mutants applied one at a time to
`internal/terminalbackend/terminalbackend.go` and `internal/config/validation.go`,
each confirmed present (`count == 1`) before the run, each run against the owning
package's full suite, sources restored from a byte-copy backup between mutants.
**20 killed, 12 survived.** Full table in
`.temp/TASK-260830-2xdt8t/mutation-results.json`.

Surviving mutants beyond F2/F3:

| Mutant | Guard | Why it survives |
| --- | --- | --- |
| M23 | `parseKind` closed `implementation_kind` vocabulary | admitting an arbitrary extra `Kind` breaks nothing — no test ever feeds an unknown kind. The `built-in kind for external path` case is caught earlier by the external-kind check in `RegisterExternal`, never by `parseKind`. |
| M20 | `validatePlatforms` sorted-unique (`>=` → `>`) | no test feeds a duplicated platform |
| M21 | `validatePlatforms` non-empty (`len == 0` removed) | no test feeds an empty platform set |
| M22 | `validatePlatforms` upper bound (4 → 5) | no test feeds an over-long platform set |
| M08 | `protocol_versions` upper bound (32 → 33) | `TestNewRefusesBadVersionTuples`'s `over 32 protocols` case builds 33 **identical** `"1.0.0"` entries, so it is killed by the sorted-unique rule, not by the count bound. The declared `semver[1..32]` maximum is unpinned. |
| M24b | `DigestFile` regular-file refusal | `DigestFile(t.TempDir())` still errors with the guard neutered, because `os.ReadFile` fails EISDIR anyway. The class the guard exists for — FIFO, device, socket — is untested. This repository already documents that exact hazard at `README.md:1120`: *"the only thing that can refuse; with it removed `os.Open` blocks indefinitely on a FIFO."* |
| M28 | `terminalbackend.ParseID` on external-trust entries in `validateTerminal` | reverting that call to the old local pattern leaves the suite green, because the `strings.HasPrefix(entry.BackendID, "ax.")` line two statements later catches both tested inputs. The trust-entry half of the wiring is not load-bearing for any test. |

`validatePlatforms` as a whole is deletable without reddening anything.

For the record, the guards that **are** properly narrowed (killed): ID grammar
widened to uppercase (M04) and to underscore separators (M05), ID byte bound
128→129 (M03), reserved-namespace prefix widened (M01) and narrowed (M02),
protocol major gate `!=1`→`<1` (M06), protocol sorted-unique `>=`→`>` (M07),
`equalRecord` scalar members (M09–M11), digest-substitution check narrowed to
7 bytes (M14), disabled-trust gate narrowed to one ID (M15), descriptor protocol
comparison dropped (M17), descriptor generation comparison narrowed (M18),
`needsDigest` narrowed to `local_program` (M19), `CheckVersionTuple` membership
loop made always-true (M31), `RequireRestoreBinding` candidate gate narrowed
(M32), config ax.-reservation removed (M25), config §6.2 Windows guard narrowed
to WSL2 (M26), config `backend_id` `ParseID` wiring reverted (M27). Those are
real, and M27 in particular proves the registry is a live production call site
of configuration validation.

### F5 — §6.2 was made real in a test and nowhere the gate can see it

The task notes set this leaf a specific target: §6.2 is *"the single `full`
binding in the whole registry … discharged by a POSITIVE-path test only … the
obvious thing to make real."*

The producer wrote `TestDecodeRefusesNonConptyBackendOnNativeWindows`, and the
guard behind it is genuine — M26 (narrow the guard from `PlatformWindows` to
`PlatformWSL2`) kills it. But the test is bound to nothing:

- `internal/traceability/ownership.v0.5.0.json` is **not in the commit**
  (`git show --stat b5cf5b5` lists 5 files; that file is not among them). The
  §6.2 acceptance case still names only the positive test.
- `README.md` is **not in the commit** either, and `README.md:1964-1966` still
  reads: *"One binding is `full` (Section 6.2, whose single clause is the
  native-Windows `conpty` requirement, discharged by
  `TestEveryPinnedReaderHasPositiveNativeWindowsAndWSL2Lanes`)."*

So the gate still reports the positive-path-only discharge the board asked this
leaf to fix, and the repository's own coverage prose still makes that claim.
The DoD line *"README/doctor/capability evidence and specification traceability
are updated without unsupported claims"* is checked off against a commit that
touches none of those files.

Fix: add the negative case to the §6.2 acceptance case in
`ownership.v0.5.0.json`, and update the README sentence to name both arms.

## 3. The two producer decisions the board asked me to adjudicate — both UPHELD

### D1 — `required_capabilities` keeps its empty default: the producer is right; record the audit finding as ANSWERED

Judged on the pinned text, not on the audit's authority. Verified independently
against the pinned `internal/specdoc/SPEC.md`:

- `SPEC.md:2585` is the **sole** occurrence of "platform lane minimum" in the
  12,665-line document: `| required_capabilities | Sorted unique closed
  capability names; default platform lane minimum |`.
- `grep -n "minimum"` returns 2467, **2585**, 2652, 6433, 6447, 6449, 6576,
  11651, 11815, 12335, 12398, 12467, 12479 — none of the others enumerates a
  per-platform capability set.
- `grep -n "lane"` puts every other hit in §19.2 *Platform lanes* (11819–11848,
  12017–12035, 12208, 12220), which is the **CI test-lane matrix** — a different
  concept that enumerates no capability set either.

No per-platform minimum is enumerated anywhere, so implementing one would invent
a constraint — the failure this board has removed six times. Keeping the empty
default is the only non-inventing choice. **The audit finding should be recorded
as answered, not left open.**

One correction the producer should carry into the rework. The LOGBOOK entry
leans on `SPEC.md:2604-2605` (*"Policy may further restrict
capabilities/transports but cannot enable an unsupported claim"*) to argue that
"empty enables nothing". That sentence does not carry the argument:
`required_capabilities` **restricts**, so an empty default is the *weaker* gate,
not a neutral one — a non-empty minimum would refuse backends that empty admits.
The honest framing is a **stated bound**: the clause is underspecified, the only
implementable default is empty, and the reason is about the contract, not about
effort. Restate the LOGBOOK decision that way and drop the 2604-2605
justification.

### D2 — §6.2 not extended to third-party IDs: correct reading, not a hole

`SPEC.md:2415-2418` reads *"terminal backend MUST be `conpty`"* and sits inside
§6.2, the Configuration 1.0.0/2.0.0 **example** section (the example document
immediately above it at `SPEC.md:2400` is `backend = "tmux"`). It names the
literal legacy token `conpty`, not the v3 canonical ID `ax.conpty`. The RFC 2119
keyword therefore governs the v1/v2 `terminal.backend` field and its v3 image,
both of which this repository implements and both of which are now enforced.

§6.5 independently admits third-party IDs (`| backend_id | … valid registered
ID |`) with manifest-bound platforms, and `validateTerminal` is consistent with
that: `ax.tmux` is refused on Windows, `ax.conpty` off Windows, third-party IDs
admitted on any platform when an enabled trust entry registers them. Declining
to extend a legacy-vocabulary clause to third-party registered IDs is a correct
reading of the pinned text, not a hole with a justification attached. The
negative arm itself is real (M26 kills it).

## 4. Board question 3 — is claiming no new ownership honest?

Mostly yes, and for the right reason. `grep -rn terminalbackend --include='*.go'`
returns exactly three files: the package, its test, and
`internal/config/validation.go`. The config file calls **only** `ParseID`.
`New`, `RegisterExternal`, `Resolve`, `IDs`, `DefaultForPlatform`,
`RequireRestoreBinding`, `CheckVersionTuple`, `CheckProviderDescriptor` and
`DigestFile` are called from nowhere in production. This repository builds no
`ax` binary yet (`go list` shows two `main` packages, both internal codegen
tools), so claiming §4.B/§7.A ownership for that surface would be an over-claim
of the *"a guard that no production path invokes has never run"* kind. Declining
to claim it is the honest call.

The one arguable under-claim: the `ax.` namespace reservation on
`terminal.backend_id` **is** enforced at a live production entry point
(`EncodeCurrent`/`Decode` → `validateTerminal` → `ParseID`), M27 proves it is
load-bearing, and it carries a real negative test. No binding was declared for
it. That is secondary to F5 and can be handled with it — but if a binding is
declared, note that the external-trust half of the same wiring is currently not
load-bearing (M28) and must not be claimed alongside it.

## 5. Also observed, not blocking

- **Two independent implementations of the same declared scalar.** This leaf
  correctly deleted `config.terminalBackendIDPattern` in favour of
  `terminalbackend.ParseID`, but `internal/canonicaljson/core_records.go:1230`
  `requireTerminalBackendID` still carries its own copy of the identical grammar
  and byte bound, and does **not** apply the `ax.` reservation — a
  `terminal.created` payload may carry `ax.evil`. This is pre-existing and
  already disclosed at `README.md:577-582` (*"that is a registry and trust rule
  rather than a payload constraint"*), and a downstream consumer going through
  `ParseID` fails closed on such an ID, so it is not a new regression. Worth a
  deliberate decision now that a registry exists: either wire the second surface
  or restate the disclosure as a bound naming the registry.
- **`Registration.validate`'s `ExecutableDigest must be null` arm is
  unreachable** from either production entry point: `New` only builds built-ins
  with an empty digest, and `RegisterExternal` refuses any non-external kind
  before `validate` runs. Dead, and consequently unmeasured.

## 6. Evidence

All under `.temp/TASK-260830-2xdt8t/` in the story worktree, attached to the board:

| Artifact | Contents |
| --- | --- |
| `review-aliasing-probe.log` | the three F1 aliasing probes, run against the real exported API |
| `zz_review_probe_test.go` | probes A–C (Resolve aliasing, RegisterExternal retention, shared built-in slice) |
| `zz_review_probe2_test.go` | probe D (drift gate reports duplicate instead of drift) |
| `mutation-results.json` | all 28 round-1 mutants with status |
| `mutate.py` | the harness: byte-copy backup, `count == 1` presence check, restore between mutants |

Probe files were run inside `internal/terminalbackend/` and moved back out;
`git status --short` is clean and `HEAD` is unchanged at `b5cf5b5`. No
production code was modified by this review.

## 7. Routing

`to-dev`. F1 and F2 are one fix (defensive copies plus same-length drift cases).
F3, F4 and F5 are independent and each small. D1 and D2 need no code change —
D1 needs the LOGBOOK justification restated as a stated bound and the audit
finding recorded as answered.
