# TASK-260830-2890sd — review round 4 mutation log (CR rev 5)

Harness: `.temp/TASK-260830-2890sd/run_mutants.py`. Each mutant is applied to a
pristine copy of `internal/provider` (backed up under
`.temp/TASK-260830-2890sd/backup/`), run with `go test ./internal/provider`, then
restored file-by-file from that backup. **No `git checkout`/`git restore` was
used** — the whole rework is uncommitted, so a git-side revert would have
destroyed it. Tree verified byte-identical to candidate `3780d471` before and
after the run.

**Total: 83 measured — 80 KILLED, 3 SURVIVED (1 environment-equivalent, 2 substantive).**

## F11 — refusal-inventory domain floor (round-3 blocking)

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| B1 | `deriveRefusalInventory` returns `refusalInventory{}` | KILLED | `scanned no production sources` + `derived no refusal sites` + reverse check |
| B2 | `Sites` truncated to 1 of N | KILLED | reverse check (17 exercised sites outside the derivation) |
| H4 | scan skips `provider.go` | KILLED | reverse check |
| B5 | stray `Error{}` in `provider.go` + read redirected one dir up (empty-but-successful) | KILLED | all three floors, 18 sites listed |

## Derived-domain floors elsewhere

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| R56 | `exportedSymbols` returns nothing | KILLED | `TestSection71AdvertisesNoPublicSDK` |
| R55a | `parseProductionSources` returns an empty map | KILLED | `TestSection71AdvertisesNoPublicSDK` |
| R55b | `structMembers` derives an empty map | KILLED | `TestCandidatesAdvertiseNoCapability` |
| R54 | `TestDiscoveryReachesNoProcess` scan finds no files | KILLED | `TestDiscoveryReachesNoProcess` |

## F12 — the `OSSystem` production seam (round-3 blocking)

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| R40 | `ReadDir` error → `nil, nil` (failure as absence) | KILLED | `TestOSSystemReadDirReportsFailure` |
| R43 | `Inspect` fabricates `{IsRegular:true, UID:euid}` on stat error | KILLED | `TestOSSystemInspectReportsStatFailure` |
| R44 | `Inspect` reports uid 0 on attestation error | KILLED | `TestOSSystemInspectReportsAttestationFailure` |
| R47 | `PathDirs` keeps empty entries | KILLED | `TestOSSystemPathDirsSkipsEmptyEntries` |
| R48 | `CurrentOperatorPolicy` seeds `AdministratorUIDs{0}` | KILLED | `TestCurrentOperatorPolicySeedsNoAdministrator` |
| R49 | `Geteuid → Getuid` | **SURVIVED** | environment-equivalent (`euid=502 uid=502`, probed) |
| R50 | unix `fileOwnerUID` returns `0, nil` | KILLED | `TestFileOwnerUIDRefusesWithoutMetadata` |
| R51 | windows `fileOwnerUID` returns `0, nil` | **not measurable** | build-tag bound; compile-verified only, see F20 |

## G-A — the `ownerAttester` seam

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| SEAM1 | production `init()` reassigns `ownerAttester` → attest every file as the operator | **SURVIVED** | — (finding **F18**) |
| SEAM2 | production `init()` reassigns `ownerAttester` → attest uid 0 | KILLED | `TestOSSystemEndToEnd` |
| CTRL1 | the same weakening written into `fileOwnerUID` itself, no seam | KILLED | `TestFileOwnerUIDRefusesWithoutMetadata` |

SEAM1 vs CTRL1 is the measurement: one semantic weakening, killed at the natural
site, survivable through the new seam.

## F13 — precondition refusals pinned by gate, not only by code

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| 14b | `Canonicalize` failure → `canon = path` | KILLED | detail pin (`cannot inspect` reported where `cannot resolve` is required) |
| 15b | `Inspect` failure → fabricate info | KILLED | detail pin (`cannot digest` where `cannot inspect` is required) |
| 29b | `Verify` inspect failure → fabricate info | KILLED | detail pin (`re-read` where `re-inspected` is required) |

## F14 / F15 / F16 / F17

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| R11 | `strings.CutPrefix → strings.Cut` | KILLED | `TestDiscoverIgnoresNonCandidateEntries` (`vendor-ax-provider-evil`) |
| R8a | partial set at `plugin_dirs` collect | KILLED | `TestDiscoverRefusesDuplicates` |
| R8b | partial set at builtin add | KILLED | `TestDiscoverRefusesDuplicates` |
| R8c | **partial set at the `plugin_dirs` absolute-path refusal** | **SURVIVED** | — (finding **F19**, non-equivalence proven) |
| R8d | partial set at PATH absolute-path refusal | KILLED | `TestDiscoverRefusesRelativePATHDir` |
| R8e | partial set at PATH collect | KILLED | `TestDiscoverRefusesDuplicates` |
| R33 | owner gate skipped when `record.owner == ""` | KILLED | `TestVerifyRefusesEmptyOwnerReceipt` |
| R9 | `Builtins()` returns `builtinOrder` | KILLED | `TestBuiltinsReturnsACopy` |

R8c non-equivalence proof: mutant + one `ax-provider-foo` planted in
`/abs/plugins` → `Discover("/abs/plugins", "relative/plugins")` returns
**1 candidate alongside the refusal**.

## Resurrection sweep — 45 of round 3's kills, re-run

All **KILLED**. No resurrections.

| # | Mutant | Killed by |
| --- | --- | --- |
| R1 | `plugin_dirs` abs gate narrowed to `index == 0` | `TestDiscoverRefusesRelativePluginDir` |
| R2 | PATH abs gate narrowed to `index == 0` | `TestDiscoverRefusesRelativePATHDir` |
| R3 | `allow_path_plugins` → `if true` | `TestDiscoverSkipsPATHWhenDisallowed` |
| R4a | duplicate narrowed to same-source | `TestDiscoverRefusesDuplicates` |
| R4b | byte-identical duplicate override | `TestDiscoverRefusesDuplicates` |
| R5 | builtins hoisted above `plugin_dirs` | `TestDiscoverIsDeterministic` |
| R6a | builtin registry drops `pi` | `TestDiscoverEnumeratesSourcesInSectionOrder` |
| R6b | `codex`/`claude` swapped | `TestDiscoverEnumeratesSourcesInSectionOrder` |
| R7 | `sort.Strings` removed | `TestDiscoverSortsEntriesWithinADirectory` |
| R10 | `ExecutablePrefix → "ax-"` | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| R12a | malformed name skipped | `TestDiscoverRefusesMalformedNames` |
| R12b | malformed name accepted | `TestDiscoverRefusesMalformedNames` |
| R13 | `ReadDir` error swallowed as empty dir | `TestDiscoverRefusesPartialReads` |
| R14 | `Canonicalize` gate narrowed to `/opt` | `TestDiscoverRefusesPartialReads` |
| R15 | `Inspect` gate narrowed to `/zzz` | `TestDiscoverRefusesPartialReads` |
| R16 | regular-file gate deleted | `TestDiscoverRefusesNonRegularTargets` |
| R17a | owner gate deleted | `TestDiscoverRefusesUnapprovedOwners` |
| R17b | owner gate skipped for `path` source | `TestDiscoverEnforcesTrustGatesAcrossSources` |
| R18 | `ReadFile` error swallowed | `TestDiscoverRefusesPartialReads` |
| R19 | operator-UID approval deleted | `TestDiscoverIsDeterministic` |
| R20a | admin-UID approval deleted | `TestDiscoverRefusesUnapprovedOwners` |
| R20b | admin set loosened to "any uid if non-empty" | `TestOwnerPolicyTreatsRootWithoutException` |
| R21 | `uid == 0 → true` | `TestOwnerPolicyTreatsRootWithoutException` |
| R22 | `OwnerIdentity` constant | `TestDiscoverRecordsTrustFacts` |
| R23 | `Trust` builtin refusal deleted | `TestTrustRefusesBuiltins` |
| R24 | `Trust` records empty owner | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| R25 | `Trust` records canon as sourcePath | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| R26 | `Trust` drops trust instant | `TestTrustRecordsCandidateFacts` |
| R27a | `Verify` re-resolve narrowed to `/zzz` | `TestVerifyTreatsReadFailureAsIntegrityFailure` |
| R27b | `Verify` re-resolve deleted (`canon = record.canon`) | `TestVerifyTreatsReadFailureAsIntegrityFailure` |
| R28 | canonical retarget gate deleted | `TestVerifyDetectsSubstitution` |
| R29 | `Verify` re-inspect narrowed to `/zzz` | `TestVerifyTreatsReadFailureAsIntegrityFailure` |
| R30 | `Verify` regular-file gate deleted | `TestVerifyDetectsSubstitution` |
| R31 | owner gate: `!Approves` clause dropped | `TestVerifyAcceptsUnchangedRootOwnedTree` |
| R32 | owner gate: `identity != record.owner` dropped | `TestVerifyDetectsSubstitution` |
| R34a | `Verify` re-read narrowed to `/zzz` | `TestVerifyTreatsReadFailureAsIntegrityFailure` |
| R34b | `Verify` re-read → `return nil` | `TestVerifyTreatsReadFailureAsIntegrityFailure` |
| R35a | digest compared on the first byte | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| R35b | digest compared on the last byte | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| R39a-d | four `Candidate` absence accessors neutered | `TestBuiltinCandidatesCarryNoTrustFacts` (4/4) |
| R41 | `Canonicalize` skips `EvalSymlinks` | `TestOSSystemEndToEnd` |
| R46 | `PathDirs` returns nil for a non-empty PATH | `TestOSSystemRefusesRelativePATHDir` |

## Meta-gates and new attacks

| # | Mutant | Result | Killed by |
| --- | --- | --- | --- |
| R52 | production emits a genuine fourth code `not_supported` | KILLED | `TestDiscoverRefusesNonRegularTargets` + closed-code-set audit |
| R57 | unexercised production refusal site added | KILLED | `provider refusal call sites without an exercised negative path` |
| R58 | stray `Error{}` literal in `provider.go` | KILLED | `refusals built outside an instrumented constructor` |
| R59a | raw `fmt.Errorf` in `provider.go` | KILLED | `raw error construction outside documented cause sites` |
| R59b | `panic` in `provider.go` | KILLED | same |
| X1 | `Verify` owner gate deleted outright | KILLED | `TestVerifyAcceptsUnchangedRootOwnedTree` |
| X2 | candidate digest not recorded (`scalar.Digest{}`) | KILLED | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| X3 | candidate `path` recorded as `canon` | KILLED | `TestVerifyRefusesDigestChangeAtEveryByteIndex` |
| X4 | digest computed over the canonical path, not the bytes | KILLED | `TestDiscoverRecordsTrustFacts` |

## Non-mutation checks

| Check | Result |
| --- | --- |
| `grep -rn "t.Parallel" internal/provider/` | zero occurrences |
| `grep -rn "ownerAttester ="` | declaration + 2 lines in `os_test.go`, no production assignment |
| production callers of `internal/provider` outside the package | none |
| `GOOS=windows go vet` really type-checks `os_windows_test.go` | yes — planted type error reported at `os_windows_test.go:32` |
| `GOOS=windows` step in `.github/workflows/ci.yml` | **absent** (finding F20) |
| `scalar.Digest` is always lowercase hex | proven by construction (`time_digest.go:132-147`) |
| tree restored to candidate `3780d471` | verified via `read-tree`/`add -A`/`write-tree` |
