# TASK-260906-2okwyf — review verdict, CR rev 1

**Verdict: CHANGES REQUESTED → `to-dev`.**
`repeat-of: none` (revision 1 of this element). See §F1 note on cross-leaf class
recurrence — the class, not the site, is what recurs.

Reviewed: `CR-TASK-260906-2okwyf-1` rev 1, `repository_delta: present`, base
`114a056`, candidate tree `2a270726`, 10 paths.

## 0. Provenance (G-F) — VERIFIED

- Recomputed the candidate tree myself in a detached index
  (`GIT_INDEX_FILE` copy + `read-tree HEAD` + `add -A` + `write-tree`):
  `2a2707266e0c50f080f3297d56e0a3a4e4b912b3` — **equals** the CR record and the
  outcome document. `ls-tree` shows `internal/provhost/profile_agreement_test.go`
  in-tree (the new untracked file), and the LOGBOOK append is inside the tree, so
  the OID was taken after every artifact write. Leaf-3's failure mode (tree
  predating its own LOGBOOK append) does not recur.
- `git diff --stat 114a056 2a270726` = exactly the 10 declared paths, 288+/35-.
  Leaves 1–3 carry no behavioral rework: `protocol.go` and `provider.go` are
  comment-only; `terminalbackend.go` is one struct field plus comments;
  `probe.go` is a behavior-preserving split (proved below).
- I re-verified the tree OID **after** all my own plants were reverted: still
  `2a270726`. My scratch lives under `.temp/` (gitignored); the candidate is
  byte-identical to what the producer handed over.

## 1. What I ran myself (not accepted from the record)

Every row below is a plant I applied to the candidate tree, ran, and reverted
from a byte-compared backup (never `git checkout`).

| # | Plant | Result |
|---|---|---|
| P1 | reorder `profileProviders` (`muse`/`antigravity`) | **only** `TestBuiltinsEqualProfileProviders` FAILS; provider package green |
| P2 | reorder `builtinOrder` (same pair) | `TestBuiltinsEqualProfileProviders` + `TestDiscoverEnumeratesSourcesInSectionOrder` + `TestSection71BuiltinRegistryIsDocumentOrder` FAIL |
| P3 | restore pre-fix `BackendID: value` on the ParseID grammar arm | `TestParseIDGrammarRefusalEchoesNothing` FAILS (red-before confirmed) |
| P4 | **narrowing**: keep `BackendID` empty, leak a *truncated* 8-byte prefix into `Detail` | FAILS — new test + 8 census tests. The pin is not delete-only |
| P5 | drop validation from `RequireCapability` (raw `decodeStrictObject`, faults discarded) | `TestRequireCapability…` + `TestDecodeValidatedProbe…` FAIL |
| P6 | **reorder**: keep `decodeValidatedProbe` but move it *after* the registry-membership check | **only** `TestDecodeValidatedProbe…` FAILS — the new test is the sole thing closing that hole |
| P7 | `parseMajor` major-digit low bound `'0'`→`'/'` | `TestParseMajorDigitBoundariesRefuseAtEntry` + 2 inventory tests FAIL (leaf-2 pin intact) |
| P8 | `parseMajor` saturation divisor `10`→`9` | `TestParseMajorSaturationEdge` FAILS (leaf-2 pin intact) |
| P9 | add the leading-zero rejection the bound declines | `TestParseMajorLeadingZeroIsClassifiedAsForeign` + `TestDerivedRefusalArmsAreAllWitnessed` + `TestDigitCensusCoversEveryLeafGuard` FAIL, the last naming `unclassifiable guard in protocol.go (parseMajor): len(parts[0]) > 1 && parts[0][0] == '0'` |
| P10 | **control**: pure named-const digit chain (`const plantZero='0'`; `c < plantZero \|\| c > plantNine`) in `protocol.go` | **both packages GREEN** |
| P11 | **control**: mixed chain (`c < plantZeroM \|\| c > '9'`) | census FAILS: `unclassifiable guard in protocol.go (plantMixedDigitGate) … (mixes digit comparisons with unrelated comparisons)` |

Gates I ran on the candidate tree: `go build ./...` exit 0, `go vet ./...` exit 0,
`gofmt -l internal` empty, `go test ./... -count=1` → **23 ok, 0 FAIL**. I did
not re-run `-race`, `-cover`, Windows cross-vet/build, tracecheck, cigate, or the
fuzz smoke; those are accepted from the attached validation log, and I say so
rather than implying I reproduced them.

## 2. Gate-by-gate

**G-A — structural, not transitive. PASS.** `TestBuiltinsEqualProfileProviders`
imports only `reflect`, `testing`, and `provider` — no `specdoc`, no SPEC.md read
on its own path. P1/P2 show it reddens from **both** sides on a pure reorder, and
P1 shows the reorder is otherwise invisible to the entire suite. The producer's
narrowing of the A5 premise is honest and correct: a direct **set** pin already
existed (`TestSixProviderSetMatchesDiscoveryRegistry`, `profile_test.go:121`,
order-insensitive by design); the live hole was order, and it is now closed.
`profileProviders` does not fall between census and tests: the census still scopes
it out (nothing in production consults it — I re-grepped: zero production
references), and it is covered by the derivation test (sorted compare against the
§7.7 table), the set pin, and now the order pin.

**G-B — trust asymmetry. PASS.** Both clauses check out verbatim against
`internal/specdoc/SPEC.md`: §7.1 (line 2622ff) — "the target MUST be a regular
file owned by the operator or an administrator-approved identity"; §6.5 (line
2591) — "Each external-trust entry contains exactly backend ID, absolute
executable path, executable digest, and `enabled`". Cited at **both** sites
(`terminalbackend.go:669` `DigestFile`, `provider.go:402` `trustCandidate`), each
pointing at the other. No third trust path: the diff adds no trust function, and
`DigestFile`/`trustCandidate` remain the only two. Consistent with the recorded
decision in `internal/secprim/doc.go:31-38` (which cites §4.B — I checked §4.B at
SPEC.md:888ff and its manifest/probe tables likewise carry no owner member, so the
new §6.5 citation is complementary, not a fork).

**G-C — three nits.** ParseID: closed, red-before proved (P3) and narrowing-proved
(P4). `parseMajor`: left open, and the stated bound's *own* claims are true, not
assumed — P9 reproduces every one of them (census unclassifiable, inventory arm
unwitnessed, bound pin reddens). Leaf-2's two pins survive this leaf intact
(P7/P8). `RequireCapability`: the body is decoded exactly once; the two remaining
sub-object replays are nil-by-construction (`checkProbeCapabilities` validated
`members["capabilities"]`, and `checkProbeCapabilityValue` validated
`capabilities[name]` for every `capabilityOrder` member, which the `known` check
guarantees `name` is). Coupling documented at both sites. P5/P6 kill it.

**G-D — F-B1. PASS.** The false sentence is gone and both halves of its
replacement are independently true: P10 proves a pure named-constant chain prunes
silently (genuine blind spot, not fail-closed), P11 proves a mixed chain fails as
unclassifiable with the exact quoted message. The absence claim holds under my own
grep: the only `'0'`/`'9'` comparisons in either package's production files
(`surrogate.go:109`, `manifest.go:366`) are literal, not named-constant. The
sentence was not reworded into a second false one.

**G-E — battery. PASS.** 9 applied / 9 killed, `SURVIVED 0`, with `NOT_APPLIED`
and `COMPILE_FAIL` as distinct rows and their controls carrying real detail (a
zero-occurrence anchor; `vet: expected declaration, found 'return'`). Classes are
separate and the bodies match their labels — I read `old`/`new` for all eleven, not
the row labels: `N-pv-trust-unapproved` is a true narrowing (`&& info.UID != 2000`
admits exactly one member of the rejected class), the `D-*` rows are real
deletions killed by **named** failures in both masks (not by exit code alone), and
`C-tb-parseid-dead-arm` is honestly filed as census-only add-arm rather than
dressed up as arm-deletion. Every mask carries a ran-count ≥1 and an explicit
`-run` list of named tests. `S-tb-shadow` re-establishes leaf-3's audit-only class
(census 15 SURVIVED, behav 229 SURVIVED, audit 658 KILLED naming
`conformance.go:713,:716`). I independently reproduced 4 of the 9 rows
(P1-equivalent, P5, P9, P3/P4) and both controls' semantics.

One label quibble, not blocking: `N-ph-parsemajor-strict` is filed `narrowing`
but *tightens* the gate. Its evidence is sound and the outcome describes exactly
what it does; the class name is just the wrong word for it.

## F1 (BLOCKING) — the new ParseID comment asserts a package-wide invariant that is false, and the echo class stays open at a live exported site

`internal/terminalbackend/terminalbackend.go:161`, in the comment added by this
leaf to justify the fix:

> Only the reserved-namespace arm names the identity, because its input already
> passed the grammar and **is a validated identity like every other BackendID in
> the package.**

Measured census of the class, production files only: **31 refusal sites carry
`BackendID:`; 28 carry a validated identity, 3 do not.** All 3 are in
`CheckVersionTuple` (`terminalbackend.go:583`, lines 585/588/595) — an **exported**
entry point whose `backendID` parameter passes through no `ParseID`/`mustParseID`
and lands straight in `Error.BackendID`, which `Error()` prints.

Failure I actually ran, through the production entry point, on the candidate tree:

```go
junk := strings.Repeat("A", 200) + "\x1b[31m../../etc/passwd"
err := terminalbackend.CheckVersionTuple(junk, "not-semver", "1.0.0", []string{"1.0.0"})
// len(err.Error()) == 322; strings.Contains(err.Error(), junk) == true
// "terminal backend refused: terminal_backend_implementation_drift for AAAA…"
```

That is the *same* defect A9 named — unbounded, unvalidated, control-byte-carrying
local data rendered into a refusal string — surviving at a sibling site while the
new comment states it cannot exist. The three witnesses for those arms
(`refusal_arm_witnesses_test.go:2056-2077`) all pass the well-formed
`"com.example.term"`, so the echo dimension of `CheckVersionTuple` is
positive-path-only: nothing in the suite would fail if the echo were worse.

Why this blocks rather than rides along:

- The nit A9 raised is a **class** ("at odds with the package never-echoes-local-data
  posture"), not a line. Closing it at `ParseID` while asserting the class is
  closed package-wide is the "class closed at one helper leaves wrappers" shape —
  and the assertion is what makes it worse than silence, because the next
  maintainer reads line 161 and does not audit `CheckVersionTuple`.
- It is the same *shape* as F-B1, which this leaf was spawned partly to correct:
  a confidently-worded universal claim in a comment, contradicted by a control
  probe. Getting the digit-guard sentence right and then minting a fresh false
  one in the neighbouring file is a wash. Two sites in two leaves means the
  defect is the absence of an instrument, not the wording of either sentence.

Not a stop-the-line, not `blocked`: this is ordinary rework with a clean path.

### What closes it

Either of these, producer's call:

1. **Durable (preferred):** a derived echo census over the package's
   `BackendID`-carrying refusal sites, in the style the package already uses
   (`refusal_arm_inventory_test.go`, `refusal_site_audit_test.go`) — every such
   site must reach its refusal with an identity that passed `ParseID`/
   `mustParseID`, or be listed as an explicit exemption with a reason. Report it
   as a ratio (`n of 31`). That closes the class instead of the line, and P4-style
   plants show the harness can red on it. If you take this path, `CheckVersionTuple`
   must come out of it either fixed or as a named exemption, not by silently
   widening what counts as "validated".
2. **Minimum:** delete the false clause (scope the sentence to ParseID's own
   arms), **and** deal with `CheckVersionTuple` explicitly — either validate
   `backendID` at entry (with a negative test that fails when the echo returns:
   assert the rendered `Error()` does not contain the caller's junk), or carry it
   as a stated bound on the function naming why an exported refusal may echo an
   unvalidated caller string, with the same test pinning the current behavior.

Whichever path: the fix needs a test that is **red before it**, per the DoD row
this leaf already applies to its other nits. A comment-only correction of the
sentence, with no instrument, leaves the class exactly where it is.

## Non-blocking notes (address alongside F1 or state a bound)

- **N1 — `DigestFile` inference stated as contract.** The comment reasons "there
  is no owner member, so no owner check applies … the closed entry leaves no
  member to record an owner in". §7.1's owner rule is *not* sourced from a config
  member either — `provider.OwnerPolicy` (`provider.go:140`) is host-supplied
  (`OperatorUID` + `AdministratorUIDs`), so an equivalent policy could be applied
  to §6.5 trust without any new config member. The *conclusion* (no owner check
  for terminal backends) is defensible and matches the recorded `secprim` decision
  and the spec's explicit-vs-silent asymmetry; the *reason given* does not carry
  it. Tighten to "the spec spells the owner dimension for §7.1 and is silent for
  §6.5 / §4.B; we do not invent it" rather than deriving it from member absence.
- **N2 — `N-ph-parsemajor-strict` class label** (see G-E): tightening filed as
  `narrowing`.

## AC coverage: 4 of 4 rows driven, each with a named test I ran

| AC row | Production call site | Driving test | Verified by |
|---|---|---|---|
| two literals equal directly, either side alone reddens, SPEC-independent | `provider.Builtins()` / `provhost.profileProviders` | `TestBuiltinsEqualProfileProviders` | P1, P2 |
| §6.5 vs §7.1 asymmetry cited at both sites, or decision recorded | `terminalbackend.DigestFile:669`, `provider.trustCandidate:402` | citation (behavior unchanged); cited gates driven by `TestDigestFile`, `TestDiscoverRefusesUnapprovedOwners` | SPEC.md read at 2591 / 2622ff / 888ff; see N1 |
| `ParseID` prints no unbounded refused input | `terminalbackend.ParseID:162` | `TestParseIDGrammarRefusalEchoesNothing` | P3 (red-before), P4 (narrowing) |
| every closed nit has a fail-before test; every open nit a stated bound | `provhost.RequireCapability` / `decodeValidatedProbe`; `parseMajor` bound | `TestDecodeValidatedProbeHandsUsableMembersToRequireCapability`; `TestParseMajorLeadingZeroIsClassifiedAsForeign` | P5, P6; P9 |

All four AC rows pass. F1 is not an AC failure — it is a false claim introduced by
this change and a live instance of the class the change reports as closed, which
the DoD's "attacked, not read" and "stated bound naming why" rows both reach.

## Scope note

`repository_delta` is `present`, so no emptiness question arises. Nothing in this
verdict asks for work outside the story boundary: `CheckVersionTuple` is in the
same file, same class, and same package posture as the nit under review.
