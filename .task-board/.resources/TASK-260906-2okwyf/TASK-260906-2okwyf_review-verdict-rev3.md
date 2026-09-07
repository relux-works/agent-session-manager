# TASK-260906-2okwyf CR rev 3 — reviewer verdict

**Verdict: CHANGES REQUESTED** → `to-dev`
**repeat-of: none** (F3 is a new class; the rev1/rev2 F1/F2 echo class is CLOSED — see §2)

Reviewer run `RUN-260907-a3fbe6`. Candidate tree `34b88fb9bb6e9fe050e400d6b51685e9cbd6547b`
re-derived by this run (detached index, `read-tree HEAD` + `add -A` + `write-tree`)
before, between and after every plant — byte-identical to the Change Request
record each time. `HEAD` stayed at the checkpoint `114a056a`; the worktree was
restored to the candidate after every mutation (per-file `shasum -a 256` recorded
below).

---

## 1. What this run executed (not read)

11 plants, all applied to the candidate tree and reverted from a pre-taken
backup, never by `git checkout`.

| # | plant | expectation | measured |
|---|---|---|---|
| P1 | new exported entry `CheckProbeIdentityDirect` reaching the unvalidated `checkProbeIdentity` arms | census reddens (entry set derived, not enumerated) | **RED** — `derived exported entry CheckProbeIdentityDirect has no census battery and no free reason` |
| P2 | new exported **method** `(*Registry).Reconcile` reaching the same arms — bare name collides with the existing `Reconcile` entry | census reddens | **GREEN** — silently absorbed (finding N2) |
| P3 | new BackendID-naming arm in `checkProbeIdentity` echoing the unvalidated `probe.ProbeID`, written as a multi-line composite literal | entry census reddens | **GREEN** for the entry census; 4 sibling inventory tests RED (finding N1) |
| P4 | `builtinOrder` reordered alone (`claude`,`codex`,…) | `TestBuiltinsEqualProfileProviders` alone reddens | **RED**, message names both literals |
| P5 | `profileProviders` `"pi"`→`"pie"` alone | same | **RED** |
| P6 | `builtinOrder` sixth member dropped alone | same | **RED** |
| P7 | pure named-const digit gate (`const reviewProbeZeroRune = '0'` …) added to `provhost/protocol.go` | both packages stay green (the corrected sentence's blind-spot claim) | **GREEN** in `terminalbackend` census *and* the full `provhost` suite — claim reproduces |
| P8 | leading-zero rejection spelled `parts[0][0] == '0'` in `parseMajor` | digit census refuses it as unclassifiable (the stated bound's justification) | **RED** — `unclassifiable guard in protocol.go (parseMajor): len(parts[0]) > 1 && parts[0][0] == '0'` |
| P9 | `N-tb-reconcile-lengthgated` re-run from the battery body (narrowing) | behav KILLED on the `len27` subclass only, inventory census SURVIVED | **reproduced exactly** — `--- FAIL: …/reconcile/len27`, inventory mask `ok` |
| P10 | `RequireCapability`: `decodeValidatedProbe` moved after the registry-membership check (ordering) | the new probe test reddens naming the drift | **RED**, sole failure: `RequireCapability(malformed, "remote_exec") code = invalid_config … want provider_protocol_error: validation must precede use` |
| P11 | reachability sweep (own `go/ast` call-graph) from all 44 declared free entries to any BackendID-naming refusal | 0 reach | **0 of 44** reach (2 raw hits, `Binding` and `TranslateLegacyBackend`, are success-value `BackendID:` fields, not refusals — reasons hold) |

Hostile vector driven through production `Reconcile` by this run, independent of
the candidate's own test (5 vectors, external test package):

| input | Reconcile result | echo |
|---|---|---|
| `200×"A" + ESC[31m + ../../etc/passwd` (221B) | `terminal_backend_id bound`, 81B | no |
| `"BAD ID" + ESC[31m + ../../etc/passwd` (27B) | `terminal_backend_id grammar`, 83B | no |
| `129×"b"` | `terminal_backend_id bound`, 81B | no |
| `"\x00nul"` | `terminal_backend_id grammar`, 83B | no |
| `ax.evil` | `terminal_backend_id reserved namespace`, 106B | **yes, by declared bound** — bounded (≤128B) and grammar-conforming, stated at `terminalbackend.go:157-161` and in the census stated bounds, pinned by `TestBackendIDEntryCensus/reserved-names-validated-identity` |

Gates re-run by this run on the candidate tree: `go test ./... -count=1` exit 0
(23 ok, 0 FAIL), `go vet ./...` exit 0, `gofmt -l internal/` clean.

---

## 2. Gates from the brief

**G-A (blocking) — is the ratio produced by the instrument? PASS.**
`TestBackendIDEntryCensus` computes `validated` in its own loop and `t.Fatalf`s
on any shortfall; the log line is emitted by the test
(`backend_id_entry_census_test.go:1326: backend-id entry census: 41/41 pairs
validated, 1 site exemptions plus 1 pair exemptions pinned`), not asserted in
prose. The entry set is **derived** (`deriveExportedEntries` walks exported
`*ast.FuncDecl`s of the production sources) — P1 proves a new exported entry
reddens rather than being ignored.

Accounting, re-derived by this run against the instrument's own derivation
(dumped through an in-package probe, then removed):

| quantity | measured | source |
|---|---:|---|
| derived BackendID refusal sites | 31 | `deriveBackendIDSites` |
| — with a live (site, entry) pair | 30 | mapping check |
| — declared site exemptions | 1 | `Registration.validate` digest-null |
| live (site, entry) pairs | 41 | `backendIDPairs` |
| declared pair exemptions | 1 | `CheckProviderDescriptor` digest-parse via `AdmitProviderDescriptor` |
| derived exported entries | **59** | `deriveExportedEntries` |
| — pair entries / hostile-only / free | 10 / 5 / **44** | union = 59, no overlap, nothing uncovered |

Both exemptions are **executed**, not asserted:
`TestBackendIDEntryCensus/digest-null-unreachable` drives `validate()` white-box
and requires the drift arm naming the validated record;
`.../digest-parse-shadowed-via-AdmitProviderDescriptor` drives a malformed
digest and requires the *parse* arm, proving the shadow. Their pinned test names
resolve to real tests (`internal_pin_test.go:50`).

**G-B (blocking) — Reconcile's two arms: FIXED, not named.** `manifest.go:1696`
and `:1699` gate both identities through `ParseID` before any check runs. The
rev-2 305B/313B echoes are gone; my own five-vector drive above shows the
refusal now comes from the `ParseID` bound/grammar arms, which carry no
`BackendID`. No exemption widened what counts as validated — the census rows for
`checkProbeIdentity`/`checkProbeGeneration` at entry `Reconcile` are live pairs
with a hostile battery, and P9's narrowing mutant shows the gate is measured at
its edge, not just its existence.

**G-C — the comment (`terminalbackend.go:153-176`): clauses verified true.**
Checked each against source: bound arm omits `BackendID` ✓; grammar arm now
omits it ✓; reserved arm names post-grammar input ✓; `mustParseID` at
`Registration.validate:247` and `TrustEntry.validate:401` ✓; `ParseID` at the
`Resolve`/`RequireRestoreBinding`/`CheckVersionTuple`/`CheckProviderDescriptor:646`/`Reconcile:1696,1699`
entries ✓; `ParseManifest:1004`, `ParseProbe:1150`, `ParseEvidence:1296`
admission ✓. One residual over-claim, non-blocking — see N1.

**G-D — non-blocking pair: both closed.** N1 (anchors): all eight re-derived
against the candidate tree and exact — `manifest.go` 1696/1699 (guards),
1741/1777/2099/2102 (naming sites), `terminalbackend.go` 646 (descriptor guard),
163 (census note). N2 (`R-tb-checktuple-order`): now filed `class: "ordering"`
in `TASK-260906-2okwyf_battery-r3.json`, and its mutant body is a genuine move
of the guard past the version arms, not a narrowing.

**G-E — battery and provenance: PASS.** 6 applied / 6 killed / 0 survived on a
production-derived denominator; classes separate (`narrowing` 1, `arm-deletion`
2, `ordering` 1, `census-only` 1, `audit-only` 1); `NOT_APPLIED` and
`COMPILE_FAIL` are distinct control rows outside the denominator; masks are
non-empty and explicit (`census_run` 15 tests, `behav_run` 12 tests → 273 ran,
`audit_run` "" → full package 695 ran); mutant bodies (old/new) are in
`TASK-260906-2okwyf_mutants-r3.json` and I read them rather than the labels —
the narrowing row is a real narrowing (guard retained, wrapped in
`len > maxIDBytes`), and P9 reproduced its verdict. Candidate tree equals the
record's, `ls-tree` confirms all three new files, `HEAD` is still `114a056` with
no commit of the producer's own.

---

## 3. AC coverage — 4 of 4 rows driven by this run

| AC row | production call site | how this run drove it |
|---|---|---|
| two six-provider literals equal directly, either side alone reddens, SPEC-independent | `provider.Builtins()` / `provhost.profileProviders` | P4/P5/P6 — reorder, rename, removal, each on one side alone, each killed by `TestBuiltinsEqualProfileProviders` under `-run` isolation. The test reads no SPEC.md. |
| §6.5 vs §7.1 asymmetry cited at both sites with the spec clause | `terminalbackend.DigestFile` (`terminalbackend.go:681-693`), `provider.trustCandidate` (`provider.go:393-401`) | verified verbatim against `agent-session-manager-spec@28bf96d7` — §6.5 "Each external-trust entry contains exactly backend ID, absolute executable path, executable digest, and `enabled`" (no owner member) and §7.1 "the target MUST be a regular file owned by the operator or an administrator-approved identity". Both citations are exact and cross-reference each other. |
| `ParseID` prints no unbounded refused input | `terminalbackend.ParseID` + every ID entry | 5-vector production drive through `Reconcile` (table §1); the one echo is the declared reserved-namespace bound, ≤128B and grammar-conforming. |
| every closed nit has a fail-before test; every open nit a stated bound | `Reconcile` entry, `decodeValidatedProbe`, `parseMajor` | P9 (Reconcile narrowing), P10 (RequireCapability ordering — the single-decode refactor's behavioural claim reproduces), P8 + `DecodeResponse` classification read (parseMajor left open with a bound whose *justification* I verified empirically, not just read). |

The named-rune correction in `digit_guard_census_test.go` is also verified by
execution: P7 reproduces the blind spot the corrected sentence now claims (the
previous wording asserted the opposite), and an independent grep finds 0
digit-valued named constants in either package.

---

## 4. Findings

### F3 — BLOCKING: an undeclared scratch file is in the candidate and would land in `main`

`probe_f2_main.go.bak` — 80 lines, `package main`, the developer's F2 reproduction
probe — sits at the **repository root** of the candidate tree. It is untracked,
not covered by `.gitignore` (`git check-ignore` exits 1), present in
`git ls-tree 34b88fb9…`, and carried in
`TASK-260906-2okwyf_change-request_rev3.patch` as one of the 13 changed paths.

Failure scenario: `accept_cr` binds this exact tree; the integration transaction
lands it, and an 80-line scratch `package main` at the root of a spec-conformant
Go repository enters `main`'s permanent history. The repository instructions put
validation artifacts under `.temp/<TASK-ID>/`; this is outside both that path
and the declared story boundary.

This is not only hygiene. `TASK-260906-2okwyf_outcome-r3.md` §"Round-3 scope"
states the scope as **"3 code/test paths + LOGBOOK"** — the candidate has four
non-LOGBOOK paths in this round. The same document reports the tree OID as
"verified by listing the tree after every artifact is written"; the listing was
performed but its contents were evidently not read back against the declared
scope, so the verification step reported agreement it had not checked. That is
the reported-facts class this task has been iterating on, arriving at the
provenance step itself.

**Fix:** delete `probe_f2_main.go.bak` (or move it under
`.temp/TASK-260906-2okwyf/`), re-derive the candidate tree OID, and make the
outcome's scope statement enumerate exactly the paths the Change Request carries.
No test change is needed; the repair is verifiable by `git ls-tree <new OID>`
containing no root-level `.bak` and by the scope list matching the CR's changed
paths one-for-one.

### N1 — non-blocking: the census site filter is source-text-keyed and blind to a multi-line literal

`deriveBackendIDSites` selects the denominator with
`strings.Contains(lines[arm.line-1], "BackendID:")`
(`backend_id_entry_census_test.go:337`). Its own comment claims "a new BackendID
arm is derived without anyone remembering this file". P3 disproves that for one
spelling: a new arm written as a multi-line composite literal keeps `BackendID:`
off the derived arm's line, so the site never enters the denominator, no pair is
demanded, and `TestBackendIDEntryCensus` stays green at 41/41 while an
unvalidated `probe.ProbeID` echoes.

Why non-blocking: measured 0 live instances (0 multi-line `&Error{` constructions
in either package's production source), and the same plant is caught closed by
four sibling instruments (`TestDerivedSiteLinesAreExactlyRowed`,
`TestDeclaredRefusalArmsAreAllDerived`, `TestDerivedArmsAreAllWitnessed`,
`TestDerivedRefusalArmsAreAllDeclared`), so an unregistered arm cannot slip in.
The gap is that a *registered* multi-line arm would satisfy those and still miss
the entry census. Either key the filter on the composite literal's AST keys
rather than the line text, or state the single-line spelling as a bound.

### N2 — non-blocking: `deriveExportedEntries` fails **open** on a bare-name collision

The entry derivation dedupes by bare function name and states in-comment "Every
bare name is unique in this package: no two exported methods share one". Nothing
enforces it. P2: an exported method `(*Registry).Reconcile` reaching the
unvalidated `checkProbeIdentity` arms is absorbed by `seen[name]` and the census
stays green — the *same collapse-two-entry-paths-into-one-row* shape the round-2
finding was about, one level up in the instrument. Measured today: 59 distinct
exported bare names, **0 collisions**, so the assumption holds and there is no
live hole. A three-line `t.Fatalf` on a duplicate bare name converts it from an
unenforced comment into a closed gate.

### N3 — non-blocking: the 44 free-entry reasons are unchecked prose

`checkBackendIDMapping` requires a free entry to carry a non-empty `reason` and
to be a derived entry; nothing checks the reason's content. A free entry that
*reaches* a paired site with unvalidated input would pass (the site already has a
pair through another entry). I checked this myself with a `go/ast` call-graph
sweep: **0 of 44** free entries reach a BackendID-naming refusal, so the list is
sound today. Recording it as a stated bound, or deriving "constructs no BackendID
refusal" from the call graph, would close it.

### N4 — non-blocking: outcome §2 entry arithmetic is off by one

"every exported entry (58 derived: 10 pair entries + 5 hostile-only + 43 free
with reasons)". The instrument derives **59**, and `backendIDFreeEntries` has
**44** rows (10 + 5 + 44 = 59). The instrument is right; the document's summary
is not. Also `LOGBOOK.md` line for round 2 says "review rev3 CHANGES REQUESTED"
where it means rev 1.

---

## 5. Summary

The substantive work is sound and I could not defeat it. The rev-1/rev-2 echo
class is closed at the entry dimension and measured at its edge, not merely its
existence; the cross-package agreement pin kills a one-sided reorder, rename and
removal; both spec citations are verbatim-accurate against the pinned spec
commit; the two stated bounds (`parseMajor` looseness, the named-rune blind spot)
had their *justifications* reproduced by execution rather than accepted as prose;
and the battery is honest at the level of mutant bodies. Four of four AC rows are
driven through production entry points by this run.

The single blocking defect is F3: an undeclared 80-line scratch file inside the
accepted tree, contradicting the outcome's own scope statement, which `accept_cr`
would bind and integration would land in `main`. It is a one-file deletion plus a
corrected scope line, and the reviewer cannot make it — this run is read-only.
