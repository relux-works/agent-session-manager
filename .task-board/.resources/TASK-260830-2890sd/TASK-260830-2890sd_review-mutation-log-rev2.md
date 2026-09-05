# TASK-260830-2890sd — round-2 review mutation log

Reviewer run `RUN-260904-067bc2`. Head `d8fc669af489a58e43ec29ccc3d9d8caedf69f98`,
branch `task-board/story/STORY-260830-3jqsx1`, worktree clean before and after.

## Method

`internal/provider/provider.go` was copied to
`.temp/TASK-260830-2890sd-review2/provider.go.bak` and every mutant was applied
as a single anchored string replacement against that pristine copy, then
restored from it. Production sources were never restored with `git checkout`
(the story worktree shares its stash and index with other work). Two driver
scripts live beside the backup: `mutate.py` (battery 1), `mutate2.py`
(battery 2), `mutate3.py` (battery 3), `probe.py`, `confirm.py` and
`confirm2.py` (isolation).

Verdict per mutant is `go test ./internal/provider/ -count=1`; the two
survivors were re-confirmed against `go test ./... -count=1` (all 14 packages).
`CAUGHT` names the failing subtests, which is what makes the mutant useful —
a mutant caught by an unrelated test proves nothing about the guard.

Baseline at this head: `go test ./internal/provider/ -count=1` ok,
`-cover` **94.1%**, `go test ./... -count=1` ok (14 packages),
`go vet ./...` exit 0, `tracecheck` exit 0 `bindings=49 clauses_discharged=17/403`.

## Battery 1 — 20 mutants (behavioural core)

| # | Mutant | Anchor | Result |
| --- | --- | --- | --- |
| M1 | force `IsRegular`+operator UID when `source == "path"` | `trustCandidate` | CAUGHT — `…AcrossSources/path/{non-regular target, unapproved owner}` |
| M2 | same for every `source != "plugin_dirs[0]"` | `trustCandidate` | CAUGHT — 4 subtests, `plugin_dirs[1]` + `path` |
| M3 | drop the PATH absolute-path gate | `Discover` | CAUGHT — `TestOSSystemRefusesRelativePATHDir`, `TestDiscoverRefusesRelativePATHDir` |
| M4 | PATH absolute gate only at `index == 0` | `Discover` | CAUGHT — `TestDiscoverRefusesRelativePATHDir` |
| **M5** | **`plugin_dirs` absolute gate only at `index == 0`** | `Discover` | **SURVIVED** |
| M6 | delete the `Verify` `IsRegular` check | `Verify` | CAUGHT — `…/replaced with directory` |
| M7 | narrow digest compare to `sum[:1]` | `Verify` | CAUGHT — `TestVerifyDetectsLateByteDigestChange` |
| M8 | narrow digest compare to `sum[:31]` | `Verify` | CAUGHT — `TestVerifyDetectsLateByteDigestChange` |
| M9 | drop `identity != record.owner` half | `Verify` | CAUGHT — `…/owner changed to an approved administrator` |
| M10 | drop `!owner.Approves(info.UID)` half | `Verify` | CAUGHT — `…/owner approval revoked` |
| M11 | drop the `Discover` owner gate entirely | `trustCandidate` | CAUGHT — 5 subtests across all 3 sources |
| M12 | duplicate refusal exempts `source == "path"` | `Discover.add` | CAUGHT — 5 subtests incl. `TestOSSystemPATHDiscovery` |
| M13 | duplicate refusal exempts a builtin first-seen | `Discover.add` | CAUGHT — `…/builtin against PATH` |
| M14 | `ReadDir` error treated as absence (`return nil`) | `collectDirectory` | CAUGHT — `TestDiscoverRefusesPartialReads/{unreadable directory, cause is preserved}` |
| M15 | `Verify` accepts a changed canonical target | `Verify` | CAUGHT — `…/retargeted symlink` |
| M16 | `Verify` re-read failure returns nil | `Verify` | CAUGHT — `TestVerifyTreatsReadFailureAsIntegrityFailure/unreadable target` |
| M17 | malformed prefixed name silently skipped | `externalID` | CAUGHT — 5 subtests across all 3 sources |
| M18 | `Trust` accepts builtins | `Trust` | CAUGHT — `TestTrustRefusesBuiltins` |
| M19 | drop `sort.Strings(names)` | `collectDirectory` | CAUGHT — `TestDiscoverSortsEntriesWithinADirectory` |
| M20 | digest read failure treated as empty content | `trustCandidate` | CAUGHT — 6 subtests, `undigestible target` + all 3 sources |

## Battery 2 — 12 mutants (policy, registry, ordering, no-claims)

| # | Mutant | Anchor | Result |
| --- | --- | --- | --- |
| **M21** | **`Approves` grants uid 0 unconditionally** | `OwnerPolicy.Approves` | **SURVIVED** |
| M22 | `Approves` ignores `AdministratorUIDs` | `OwnerPolicy.Approves` | CAUGHT — `TestDiscoverRefusesUnapprovedOwners` |
| M23 | swap `codex`/`claude` in `builtinOrder` | registry | CAUGHT — `…EnumeratesSourcesInSectionOrder`, `TestSection71BuiltinRegistryIsDocumentOrder` |
| M24 | drop `pi` from `builtinOrder` | registry | CAUGHT — same two |
| M25 | scan PATH regardless of `AllowPathPlugins` | `Discover` | CAUGHT — `TestDiscoverSkipsPATHWhenDisallowed` + 2 `OSSystem` tests |
| M26 | `Verify` `Inspect` failure returns nil | `Verify` | CAUGHT — `…/uninspectable target` |
| M27 | `Verify` `Canonicalize` failure returns nil | `Verify` | CAUGHT — `…/unresolvable source` |
| M28 | PATH absolute gate moved *after* the directory read | `Discover` | CAUGHT — `TestDiscoverRefusesRelativePATHDir` |
| M29 | builtin candidates expose a digest | `Candidate.Digest` | CAUGHT — `TestBuiltinCandidatesCarryNoTrustFacts` |
| M30 | `OwnerIdentity` returns a constant | `OwnerPolicy` | CAUGHT — 4 tests incl. the approved-administrator subtest |
| M31 | non-prefixed entries accepted as candidates | `externalID` | CAUGHT — `TestDiscoverIgnoresNonCandidateEntries` |
| M32 | drop the `plugin_dirs` absolute gate entirely | `Discover` | CAUGHT — `TestDiscoverRefusesRelativePluginDir` |

**Batteries 1-2: 30 caught, 2 survived.** Note M32 (delete) is caught while M5 (narrow) is
not: the delete-only control says the gate exists, and says nothing about the
class it covers. That contrast is the whole reason narrowing mutants are run.


## Battery 3 — 6 mutants (the orchestrator's round-2 questions)

The round-2 brief named four things to check rather than re-derive. These
mutants answer them.

| # | Mutant | Question it answers | Result |
| --- | --- | --- | --- |
| **M33** | fourth discovery source added to `Discover` with every trust gate bypassed | "does adding a fourth source without covering it redden?" | **SURVIVED** |
| M34 | digest compare narrowed to `sum[:16]` | "does any proper prefix redden, not only `[:1]`?" | CAUGHT — `TestVerifyDetectsLateByteDigestChange` |
| **M35** | digest compare narrowed to `sum[1:]` (drop the head byte) | same | **SURVIVED** |
| M36 | digest compare narrowed to `sum[8:24]` | same | CAUGHT — same test |
| M37 | new refusal site in `Verify` with no negative path | "does the inventory gate still hold?" | CAUGHT — gate: `provider refusal call sites without an exercised negative path: provider.go:451` |
| M38 | re-introduce the double hash in `trustCandidate` | "is the single-hash path still intact?" | CAUGHT — `TestDiscoverRecordsTrustFacts`, `TestTrustRecordsCandidateFacts`, `TestOSSystemEndToEnd` |

M33 in full:

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
	if cfg.AllowPathPlugins {
```

No canonicalisation, no shape check, no owner check, no digest. `/opt/ax-providers`
does not exist on this host and is unknown to the fake, so the source is inert
in every test — which is exactly the finding: the suite cannot tell an inert
new source from a covered one, because `trustGateSources()` is a literal.

## Answers to the four round-2 questions

1. **Is the 3x4 table's domain derived?** No. `trustGateSources()` and
   `trustGateDimensions()` are literals and the ratio is
   `len(sources) * len(dimensions)` compared against itself. M33 survives.
   Verdict F9.
2. **Does the late-byte fixture bound the digest compare across the whole
   digest?** Partly. Every prefix narrowing reddens (`[:1]`, `[:16]`, `[:31]`)
   and so does a middle window (`[8:24]`), because all of them drop byte 31.
   Dropping byte 0 (`sum[1:]`) survives. The 31-byte shared prefix IS asserted
   in the fixture, as asked — the gap is the head, not the assertion. Verdict F10.
3. **Does the approved-administrator subtest trip only the identity half, and
   the old case only the approval half?** Yes, exactly disjoint. M9 (drop
   `identity != record.owner`) reddens only
   `…/owner changed to an approved administrator`; M10 (drop
   `!owner.Approves`) reddens only `…/owner approval revoked`. Neither mutant
   reddens the other's subtest.
4. **Can any branch other than `IsRegular` fire on the F3 fixture?** No. The
   fixture keeps `content: []byte("bytes")` so the digest branch cannot fire,
   the canonical target and owner are unchanged, and the `Detail()` pin
   requires the string `no longer a regular file`. M6 (delete the check)
   reddens exactly that subtest.
5. **Did the rework disturb the two mechanisms round 1 said hold?** No. M37
   shows the AST-derived refusal-inventory gate still catches an unexercised
   refusal site and names its file:line. M38 shows the single-hash trust path
   is still pinned by three tests including the real-filesystem end-to-end.

## Survivor isolation

**Totals across all three batteries: 38 mutants, 34 caught, 4 survived.**
Every survivor was re-confirmed against the full repository suite at the
committed head, with no probe file present:

```
M21_superuser_implied          -> go test ./... -count=1 : GREEN (SURVIVED)
M5_plugindirs_abs_first_only   -> go test ./... -count=1 : GREEN (SURVIVED)
M33_fourth_ungated_source      -> go test ./... -count=1 : GREEN (SURVIVED)  | go vet clean
M35_digest_drop_first_byte     -> go test ./... -count=1 : GREEN (SURVIVED)  | go vet clean
```

Two probe tests were then written into `internal/provider`, run, and deleted
(`git status --short` empty afterwards, `HEAD` unchanged at `d8fc669`):

```go
// Probe A — F6
func TestReviewProbeRootIsNotImplicitlyApproved(t *testing.T) {
	fake := newFakeSystem()
	fake.addFile("/plugins", "ax-provider-foo", []byte("x"), 0)
	_, err := Discover(fakeConfig("/plugins"), OwnerPolicy{OperatorUID: fakeUID}, fake)
	if err == nil {
		t.Fatal("Discover admitted a root-owned executable under an operator-only policy")
	}
	if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
}

// Probe B — F7
func TestReviewProbePluginDirsAbsoluteGateAtIndexOne(t *testing.T) {
	fake := newFakeSystem()
	fake.entries["/plugins"] = []string{}
	_, err := Discover(fakeConfig("/plugins", "relative/dir"), fakeOwner(), fake)
	if err == nil {
		t.Fatal("Discover accepted a relative plugin_dirs entry at index 1")
	}
	if code := errorCode(t, err); code != codeInvalidConfig {
		t.Fatalf("code = %q, want invalid_config", code)
	}
}
```

```
Probe A: intact production -> PASS ; uid-0 exception injected -> FAIL
         "Discover admitted a root-owned executable under an operator-only policy"
Probe B: intact production -> PASS ; index==0 narrowing injected -> FAIL
         code = "local_precondition_failed", want invalid_config
```

Stated bound on Probe B: under the narrowing the probe reddens on the *code*,
because the fake's `ReadDir` errors on the unknown relative directory. On a
real filesystem where the relative directory exists and is readable, the same
narrowing would silently **admit** its entries. The probe detects the mutant
either way; the failure mode it reports is not the worst one.


```go
// Probe C — F10
func TestReviewProbeEarlyByteDigestChange(t *testing.T) {
	// receipt digest differs from the bytes on disk only in byte 0,
	// with the 31-byte tail asserted shared
	...
	if err := Verify(forged, fakeOwner(), fake); err == nil {
		t.Fatal("Verify accepted a receipt differing only in its first digest byte")
	}
}
```

```
Probe C: intact production -> PASS ; sum[1:] narrowing injected -> FAIL
         "Verify accepted a receipt differing only in its first digest byte"
```

## Round-1 findings re-verified at this head

Each round-1 mutant was re-injected at `d8fc669` rather than trusting the
rework notes. All five now redden — F1 (`path` and `!= plugin_dirs[0]`
variants), F2 (gate deleted, and gate moved after the read), F3 (`Verify`
`IsRegular` deleted), F4 (`sum[:1]` and the tighter `sum[:31]`), F5
(`identity != record.owner` deleted). See M1-M9 and M28 above.

## Bounds of this harness

- Mutants target `internal/provider/provider.go` only. `os.go`, `os_unix.go`
  and `os_windows.go` were not mutated; `os_windows.go` cannot be executed on
  this host at all, which is the leaf's disclosed unverified-at-runtime bound.
- 38 mutants is a sample, not a sweep. `34/38` is a measured ratio over the
  guards enumerated here, not a statement that no other survivor exists in the
  package. Four of the four survivors were found by narrowing; a delete-only
  harness would have reported 38/38 and been wrong about every one of them.
- The harness lives under `.temp/` and no committed artifact derives its
  counts.
