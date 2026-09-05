# TASK-260830-2890sd review verdict rev2: CHANGES REQUESTED

Reviewer run `RUN-260904-067bc2` (Opus 5). Change Request
`CR-TASK-260830-2890sd-2` revision 2, state `ready`, `repository_delta=empty`.
Branch `task-board/story/STORY-260830-3jqsx1`, head `d8fc669` (signed, `G`).
Leaf 1 of 3, non-final. Supersedes nothing: the rev1 verdict
(`TASK-260830-2890sd_review-verdict.md`, run `RUN-260904-4f8f04`) stays on the
board as the record of round 1.

## On the empty Change Request

`repository_delta=empty` is a snapshot artifact again, and for a different
reason than round 1. Round 1's base was the leaf's own tree because the
producer committed before handoff. Here the CR `base_oid` is the **branch tip
itself**:

```
CR base_oid                       d8fc669af489a58e43ec29ccc3d9d8caedf69f98
git rev-parse HEAD                d8fc669af489a58e43ec29ccc3d9d8caedf69f98
CR candidate_tree_oid             465eaea41153e5331651b9442e994ec09bad69d2
git rev-parse HEAD^{tree}         465eaea41153e5331651b9442e994ec09bad69d2
workspace checkpoint_oid          52f7166519c37a92142c81ae583e4e7d243bb97c
patch sha256                      e3b0c442… (the empty-string digest)
```

The candidate tree matches `HEAD^{tree}` exactly, so — unlike the stale-CR
shape this board has hit three times — the CR is **not** stale: it faithfully
represents the branch. What it cannot represent is the delta, because base and
candidate are the same commit. The real reviewable window is
`checkpoint..HEAD` = `52f7166..d8fc669`, and the round-2 rework delta is
`292fc6d..d8fc669` (the amended-away round-1 leaf):

```
git diff --stat 292fc6d d8fc669
 internal/provider/os_test.go            |  19 ++
 internal/provider/provider.go           |  10 +-
 internal/provider/trust_sources_test.go | 168 ++++++++++++++++++++++
 internal/provider/trust_test.go         |  92 ++++++++++-
 4 files changed, 286 insertions(+), 3 deletions(-)
```

That is what was reviewed. **`empty` here is not an empty deliverable and it is
not the right outcome for this leaf either** — the leaf did produce repository
change, and the verdict below is about it, not about the zero-byte patch.

## What the rework fixed — all five confirmed by mutation, not by reading

Every round-1 finding is genuinely closed. Each was re-derived by re-injecting
the round-1 mutant at this head and confirming the suite now reddens:

| Round-1 finding | Mutant re-injected at `d8fc669` | Result |
| --- | --- | ---: |
| F1 trust gates at 1 of 3 sources | force `IsRegular`+operator UID when `source == "path"` | CAUGHT — `TestDiscoverEnforcesTrustGatesAcrossSources/path/{non-regular target, unapproved owner}` |
| F1 (generalised) | same for every `source != "plugin_dirs[0]"` | CAUGHT — 4 subtests across `plugin_dirs[1]` and `path` |
| F2 relative PATH bypass | drop the PATH absolute-path gate | CAUGHT — `TestDiscoverRefusesRelativePATHDir` **and** `TestOSSystemRefusesRelativePATHDir` (production `OSSystem` seam) |
| F3 shape-only fixture | delete the `Verify` `IsRegular` check | CAUGHT — `TestVerifyDetectsSubstitution/replaced_with_directory` |
| F4 digest at 1 of 32 bytes | narrow compare to `sum[:1]` | CAUGHT — `TestVerifyDetectsLateByteDigestChange` |
| F4 (tighter) | narrow compare to `sum[:31]` | CAUGHT — same test |
| F5 owner-identity half | drop `identity != record.owner` | CAUGHT — `TestVerifyDetectsSubstitution/owner changed to an approved administrator` |

The F2 production fix (`provider.go:348-353`, PATH entries validated with
`scalar.ParseAbsolutePath` **before** `collectDirectory`) is the only
behaviour change in the round, and its ordering is pinned: moving the gate
after the directory read reddens `TestDiscoverRefusesRelativePATHDir`. The
`Detail()` pins added per `Verify` subtest close the round-1 "every branch
reports the same code" blind spot — deleting any single `Verify` branch now
names the subtest that owned it.

The two mechanisms round 1 said hold still hold, re-confirmed rather than
read: adding a refusal site with no negative path reddens the AST-derived
inventory gate by file:line (`provider refusal call sites without an exercised
negative path: provider.go:451`), and re-introducing the double hash in
`trustCandidate` reddens three tests including the real-filesystem
`TestOSSystemEndToEnd`.

I ran 38 mutants at this head across three batteries. **34 caught, 4
survived**, every survivor a narrowing mutant, every survivor re-confirmed
against `go test ./... -count=1` over all 14 packages. Full table, anchors, and
per-mutant failing subtests are in
`TASK-260830-2890sd_review-mutation-log-rev2.md`.

## Why this is not accepted

### F6 — uid 0 is admitted as a trusted owner, and the whole repository suite stays green

`OwnerPolicy.Approves` (provider.go:143-152) has no test that uses uid 0.
Injecting a superuser exception:

```go
func (policy OwnerPolicy) Approves(uid uint32) bool {
	if uid == 0 {
		return true
	}
	if uid == policy.OperatorUID {
```

leaves `go test ./... -count=1` **green across all 14 packages**. Nothing in
`internal/provider` — not the four refusal tests, not the new 12-cell
source × dimension table, not the real-filesystem `OSSystem` tests — drives an
owner UID of 0. `fakeUID = 1000`, `foreignUID = 2000`, `adminUID = 7`
(`fake_test.go:116-118`); zero never appears.

This is not an abstract ratio. It is a documented property with no evidence
behind it, asserted in three places:

- production doc comment, `provider.go:139`: *"An empty administrator set
  trusts the operator alone; **no superuser exception is implied**."*
- `TASK-260830-2890sd_outcome.md`: *"approving owner (`uid:` identity under an
  explicit `OwnerPolicy`; **no implied superuser**)"* — an unsupported claim in
  the outcome artifact this leaf ships as evidence.
- the §7.1 MUST this package pins verbatim in
  `enumeration_test.go:135`: *"the target MUST be a regular file owned by the
  operator or an administrator-approved identity."* Root is not
  administrator-approved unless an operator configures it as such, so the
  mutant admits exactly what the pinned MUST rejects.

uid 0 is the single most consequential identity for this trust boundary, and
it is also the realistic one: `/usr/local/bin` and most system PATH entries are
root-owned, so a root-owned `ax-provider-*` is the first case a host will meet
in the field. Current production refuses it — correctly, and strictly — and
nothing would notice if that stopped being true.

**Confirmed by isolation, not inferred.** A probe test was written into the
package, run, and removed:

```go
func TestReviewProbeRootIsNotImplicitlyApproved(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-foo", []byte("x"), 0)
	_, err := Discover(fakeConfig("/plugins"), OwnerPolicy{OperatorUID: fakeUID}, fake)
	// ... require invalid_config
}
```

```
intact production              -> PASS
uid-0 exception injected       -> FAIL: "Discover admitted a root-owned executable
                                  under an operator-only policy"
```

Fix: pin the negative through `Discover` and through `Verify`, and pin the
positive complement so the gate is bounded on both sides — a receipt recorded
under `OwnerPolicy{OperatorUID: 1000, AdministratorUIDs: []uint32{0}}` must
*accept* uid 0, proving the refusal is policy-driven rather than a blanket ban
on zero. Deleting the superuser exception must redden; adding it must redden.

### F7 — the `plugin_dirs` absolute-path gate is proven at index 0 only

Narrowing provider.go:334-337 to `err != nil && index == 0` leaves
`go test ./... -count=1` green across all 14 packages.
`TestDiscoverRefusesRelativePluginDir` (`discovery_test.go:261`) passes exactly
one directory, so nothing bounds the gate over list position.

This is the same shape as F2, in the twin of the guard this very round
hardened: the new `TestDiscoverRefusesRelativePATHDir` deliberately drives
`{"/abs/bin", "relative/bin"}` to bound the PATH gate across positions, and
that table was not backfilled to `plugin_dirs`. Confirmed by isolation with a
second probe (`fakeConfig("/plugins", "relative/dir")`): PASS against intact
production, FAIL under the narrowing.

Severity is lower than F6 and stated as such: `internal/config`
`validateProviders` (`validation.go:498-503`) already rejects every
non-absolute `providers.plugin_dirs` entry, so this gate is defence in depth
on a path a validated config cannot reach. It is still an unbounded guard, it
costs one extra element in one existing fixture, and leaving it is how the next
round inherits it.

### F8 — the round-1 logbook entry was never landed

The rev1 verdict supplied a complete entry with the instruction *"The rework
producer should land this verbatim above the 2026-09-04 0115 entry."*
`LOGBOOK.md` is untouched in `292fc6d..d8fc669`, and its newest entry is still
`0115`. The round-2 findings — "site reached is not branch distinguished", the
1-of-3-sources gate, the 1-of-32-bytes digest, the relative-PATH bypass — exist
only in board resources.

This matters beyond bookkeeping. LOGBOOK entry `1632` already records the
generalisation this round then repeated: *"when a class of 'proven at one
sample' is found, the remedy is to enumerate every guard on that path and
measure each over its own domain, not to fix the one that was named."* The
round-2 rework fixed the four guards it was handed and did not sweep
`OwnerPolicy.Approves` — the very next guard on the same `trustCandidate` path
— or the `plugin_dirs` twin of the guard it was fixing. That is the third
recorded instance of the same failure, and the mechanism that was supposed to
propagate the lesson is the entry that did not get written.

### F9 — the 3x4 table's source domain is a literal, and a fourth ungated source survives

The orchestrator's round-2 brief asked directly whether the table's domains are
**derived** from the production discovery order rather than pinned as a
constant, and whether adding a fourth source without covering it reddens. The
answer is no, and no.

`trustGateSources()` (`trust_sources_test.go:69-95`) returns a hand-written
three-element slice; `trustGateDimensions()` returns a hand-written four-element
slice. The reported ratio is

```go
total := len(sources) * len(dimensions)   // 12
...
if passed != total { t.Fatalf("trust-gate coverage = %d/%d", passed, total) }
```

which compares the literal against itself. `12/12` is not a measurement of the
production source set; it is the same constant on both sides. `passed++` also
only runs when a subtest reaches the end of its closure, so a failing subtest
has already reddened before the ratio is consulted — the check adds nothing the
subtests do not already report.

Demonstrated, not argued. Adding a fourth discovery source to `Discover` with
every trust gate bypassed:

```go
	if names, err := system.ReadDir("/opt/ax-providers"); err == nil {
		for _, name := range names {
			if id, ok, _ := externalID(name); ok {
				if err := add(Candidate{id: id, kind: KindExternal, source: "system"}); err != nil {
					return nil, err
				}
			}
		}
	}
```

leaves `go test ./... -count=1` **green across all 14 packages** and `go vet`
clean. No shape check, no owner check, no digest, no canonicalisation — a whole
ungated source, invisible to a table that enumerates sources from a literal.
(The mutant reads a directory that does not exist on this host, so it is inert
in the fake and inert on the real filesystem; that inertness is the point. The
suite cannot distinguish "source covered" from "source not enumerated".)

This package already owns the right pattern one file away.
`refusal_inventory_test.go` parses production source with `go/ast` and requires
a bijection between refusal constructor call sites and exercised negative
paths — and it is real: injecting a new unexercised refusal into `Verify` made
it fail naming `provider.go:451`. The trust-gate table should be derived the
same way: enumerate the `collectDirectory(...)` call sites and their source
labels out of `provider.go`, require every label to appear in the table, and
let the ratio be a count of derived labels covered over derived labels found. A
source added with no row is then a red, which is exactly what a coverage
number is for.

Round 1's F1 asked for a table and got one. The table is a real improvement —
M1 and M2 prove it catches per-source bypasses that nothing caught before. It
is bounded over the sources it lists and unbounded over the sources production
has.

### F10 — the digest guard is now bounded at the tail and still unbounded at the head

The brief asked whether narrowing the comparison to **any** proper prefix now
reddens, not only `[:1]`. Prefixes do. Suffixes do not.

```
sum[:1]   vs d[:1]     -> CAUGHT   (TestVerifyDetectsLateByteDigestChange)
sum[:16]  vs d[:16]    -> CAUGHT   (same)
sum[:31]  vs d[:31]    -> CAUGHT   (same)
sum[8:24] vs d[8:24]   -> CAUGHT   (same)
sum[1:]   vs d[1:]     -> SURVIVED (go test ./... green, all 14 packages, vet clean)
```

`TestVerifyDetectsLateByteDigestChange` moves the *last* byte and asserts the
31-byte prefix is shared, so every narrowing that drops the tail reddens. No
fixture in the package moves the *first* byte alone, so dropping the head is
free. The guard went from "proven at byte 0" to "proven at byte 31" — the same
1-of-32 shape, mirrored.

**Confirmed by isolation.** A probe building a receipt whose digest differs from
the bytes on disk only in its first byte, with the 31-byte tail asserted
shared, was written into the package, run, and removed:

```
intact production        -> PASS
sum[1:] narrowing        -> FAIL: "Verify accepted a receipt differing only in
                            its first digest byte"
```

Fix: stop chasing the quoted mutant and sweep the domain. Thirty-two
subtests — for each byte index `i`, a receipt differing from the genuine digest
only at byte `i`, required to refuse — cost microseconds and report `32/32`
over the guard's actual domain instead of two sampled endpoints. That is the
form round 1's F4 asked for and the form entry `1632` records as the standing
remedy.

## Observations — not rework for this leaf

- **`internal/provider` has no caller outside itself.** `grep -rn
  "internal/provider" --include='*.go'` matches nothing outside the package.
  The production entry points are the package API (`Discover`, `Trust`,
  `Verify`, `OSSystem`, `CurrentOperatorPolicy`), driven directly by tests
  including through the real filesystem. This is the disclosed M0 contract-
  foundation scope and the rev1 review accepted the framing; the wiring belongs
  to a later leaf. Restated here so it is not mistaken for a measured claim
  that the host path is exercised.
- **`TASK-260830-2890sd_outcome.md` is stale at this head.** It names commit
  `292fc6d`, "34 top-level PASS", and "93.4%". The head is `d8fc669`, the suite
  is 38 top-level PASS, and coverage is 94.1% (`TASK-260830-2890sd_rework-notes.md`
  has the current figures). Refresh it with the rework, in the same change that
  drops the "no implied superuser" claim or backs it with F6's test.
- **§7.1 still has no ownership binding** and `tracecheck` still reports
  `clauses_discharged=17/403`, unchanged. The rev1 deferral reason remains
  accurate (`traceability.go:422` pins the projection to
  `reviewedOwnershipCanonicalSHA256`). Story-level obligation for the
  orchestrator before `STORY-260830-3jqsx1` closes, unchanged from rev1.
- **Native Windows owner attestation remains unverified at runtime**, as
  disclosed. `GOOS=windows go build` proves compilation only.
- The refusal-inventory gate is still skipped whenever `-run` is non-empty
  (`refusal_inventory_test.go:65`). Repository CI runs `go test ./...`, so it
  fires. Worth knowing before a `-run` mask reaches CI.
- The 12-cell trust-gate table asserts `Code()` only, and `shape`, `owner` and
  `name` all report `invalid_config`. Within `Discover` that is adequate today
  — the mutants that matter all flip refuse-to-admit rather than swapping one
  refusal for another — but it is the same construction the round-1 F3 finding
  punished inside `Verify`. Adding a `Detail()` substring per dimension would
  make the table self-describing at the cost of three strings.

## Evidence

- `TASK-260830-2890sd_review-mutation-log-rev2.md` — 32 mutants with exact
  anchors, per-mutant failing subtests, the two survivors, and both isolation
  probes.
- Re-run at review time from `d8fc669`, clean worktree:
  `go test ./internal/provider/ -count=1` ok; `-cover` 94.1%;
  `go test ./... -count=1` ok, all 14 packages; `go vet ./...` exit 0;
  `go run ./internal/traceability/cmd/tracecheck` exit 0,
  `bindings=49 clauses_discharged=17/403` — unchanged, as the rework reported.
- Worktree verified clean before and after every mutation batch
  (`git status --short` empty, `git rev-parse HEAD` = `d8fc669`). Production
  sources were restored from a `.temp` backup copy, never with `git checkout`.
  The two probe tests were written into the package, run, and deleted; nothing
  in the repository changed during this review.

## Routing

`to-dev`. No production behaviour change is required. F6, F7 and F10 are test
work whose fixtures are written out above; F9 is one derived inventory built on
the pattern already in `refusal_inventory_test.go`; F8 is one LOGBOOK entry plus
the outcome refresh. Nothing found here is a stop-the-line boundary, and the
round-2 work itself is good: five findings closed, all five confirmed by
re-injecting their own mutants, and 34 of 38 mutants caught at this head.

Two of the four survivors are the orchestrator's own round-2 questions answered
in the negative (F9, F10) — the fixes landed against the mutants that were
quoted rather than against the classes they belong to. That is the distinction
the brief asked to be checked, so it is the finding, not a technicality.

## Logbook entry to land with the rework

Not written by this run — the reviewer is read-only and the leaf must stay one
commit past the checkpoint with a clean worktree. Land this **and** the rev1
entry that was skipped, both above the `0115` entry, newest first.

```markdown
### 0230 — The guard the round did not sweep was the one that matters: uid 0 admitted, suite green

- SCOPE: `internal/provider` round-2 review of TASK-260830-2890sd at `d8fc669`; §7.1. 32 mutants, 30 caught, 2 survived.
- FINDING: injecting `if uid == 0 { return true }` into `OwnerPolicy.Approves` (`provider.go:143`) leaves `go test ./... -count=1` GREEN across all 14 packages. No test in the package uses uid 0 — `fakeUID = 1000`, `foreignUID = 2000`, `adminUID = 7`. The property is asserted three times and measured zero times: the production doc comment at `provider.go:139` ("no superuser exception is implied"), the outcome artifact ("no implied superuser"), and the §7.1 MUST pinned verbatim at `enumeration_test.go:135` ("owned by the operator or an administrator-approved identity"). Root-owned system PATH directories are the first case a real host meets, so this is the realistic identity, not an exotic one. Confirmed by isolation: a probe driving `Discover` with a uid-0 file under an operator-only policy PASSes against intact production and FAILs under the injection.
- ROOT CAUSE: entry 1632's generalisation, for the third recorded time. Round 2 closed the four guards the review named and did not enumerate the rest of the path. `OwnerPolicy.Approves` is the next guard inside `trustCandidate` after the two that were fixed, and `TestDiscoverRefusesRelativePluginDir` is the single-entry twin of the multi-entry PATH fixture the same round wrote — narrowing the `plugin_dirs` absolute gate to `index == 0` also survives the full suite. The remedy for "proven at one sample" is the traversal, not the named fix.
- NOTE: the rev1 verdict supplied a complete logbook entry with an instruction to land it verbatim, and `LOGBOOK.md` is untouched in `292fc6d..d8fc669`. The lesson that would have prevented this round's omission was written into a board resource and never reached the file the next round reads. A verdict-supplied entry is part of the rework scope, not an appendix to it.
- FINDING: two round-2 fixes closed the quoted mutant and not its class. The new 3x4 trust-gate table enumerates its sources from a hand-written literal and reports `12/12` as `len(sources) * len(dimensions)` compared against itself, so adding a fourth discovery source to `Discover` with every trust gate bypassed leaves `go test ./...` GREEN across all 14 packages. The package already owns the right pattern one file away — `refusal_inventory_test.go` derives refusal sites from the AST and reddens naming `provider.go:451` when an unexercised site is injected — and the table should derive its source labels from the `collectDirectory` call sites the same way. Separately, `TestVerifyDetectsLateByteDigestChange` moves the LAST digest byte, so `sum[:1]`, `sum[:16]`, `sum[:31]` and `sum[8:24]` all redden while `sum[1:]` SURVIVES: the guard moved from proven-at-byte-0 to proven-at-byte-31, the same 1-of-32 shape mirrored. A 32-subtest sweep over byte index costs microseconds and reports the ratio the domain actually has.
- NOTE: the round-2 fixes are real and were each re-confirmed by re-injecting the round-1 mutant at `d8fc669` — the 12-cell source x dimension table, the PATH absolute gate on both the fake and production `OSSystem` seams, the shape-only `Verify` fixture with per-branch `Detail()` pins, the late-byte digest fixture (kills both `sum[:1]` and `sum[:31]`), and the approved-administrator owner-identity subtest. Production behaviour is correct as written; only the evidence is short.
- STATUS: Pending. TASK-260830-2890sd routed to `to-dev`; no production behaviour change required.
```
