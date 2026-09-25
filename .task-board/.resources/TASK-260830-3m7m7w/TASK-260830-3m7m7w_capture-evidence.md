# TASK-260830-3m7m7w — capture review handoff evidence

Run RUN-260907-d14c2f. Candidate is ready for review, subject to the ordinary
immutable developer handoff transaction. No manual commit, branch switch,
rebase, checkpoint, integration, nested run or pool expansion occurred.

## Authoritative goal and scoped acceptance

GOAL-260907-c70ce0 revision 1; kind and success predicate **role_handoff**.
Resolved scope: TASK-260830-2bnr39, TASK-260830-3m7m7w.

Current authoritative objective:

> Intent summary (non-authoritative): Execute TASK-260830-2bnr39, TASK-260830-3m7m7w within the active board delivery scope
> Review-policy snapshot: TASK-260830-2bnr39=required, TASK-260830-3m7m7w=required
> 
> Environment contract: Act with maximum autonomy and keep progressing through recoverable work without asking for routine confirmation. After surfacing final board evidence for a satisfied success predicate, clear the provider goal through its successful-completion path; never early-clear it to evade acceptance. If the requested outcome objectively does not fit, never force it or fake a fit: make the next optimal assumption only when it is unambiguously derivable; otherwise invoke the repository's existing **Stop-The-Line: No Forced Fits** boundary, persist its evidence packet, and surface only the exact human-only decision or external input needed.
> 
> Keep working on board scope TASK-260830-2bnr39, TASK-260830-3m7m7w until every assigned item reaches the role end status `to-review` with all task acceptance criteria and checklist gates satisfied and a new or updated task-scoped outcome artifact, or the repository's existing **Stop-The-Line: No Forced Fits** boundary is evidenced. Routine failed checks, recoverable runtime errors, and rework remain continuation signals; they are not successful handoff.

The predecessor's accepted CR2 is checkpointed at
`d888cd576f5962eb258e40ca487f7711b2215610`, which remains this worktree's HEAD.
Its independent review-verdict-rev2 and checkpoint outcome are included in the
archive. Fresh board reads show integrating, 18/18 checklist items, and CR2
checkpointed; this run also executed git verify-commit HEAD, exit 0. That leaf
has already passed its producer review handoff and subsequent acceptance; it
is preserved rather than regressed to to-review. Its previous gate/pack bounds
are accepted prior evidence, not new mutation execution in this run.

For this leaf, all code/tests remain uncommitted for the board's managed CR
snapshot. The task-scoped source manifest covers 17 changed paths and includes
all untracked candidate sources. The eventual handoff response is authoritative
for CR identity and to-review status; this prose does not mint acceptance.
The owner/reviewer retains independent review and later integration.

## Delivered scope and production evidence

Capture accepts ContentOptions to capture actual tracked/untracked working
bytes, classified ignored includes, directories/modes, safe symlink targets,
sparse pattern blobs, recursive submodule snapshots and exclusions through the
existing entry. Omitting options keeps the accepted metadata observation API;
a nil Content never represents complete workspace closure. Safe reads reuse
secprim.Guard.Open, blob installs reuse localstore.PutBlob, and descriptor/entry
validation reuses canonicaljson. There is no parallel secret scanner.

Files stream through 4 MiB chunks, then rewind the guarded file for the verified
whole-blob install. No additional capture spool is created. Submodule stage-0
pointers, checked-out HEAD, index and working bytes stay independent, including
nested/detached and uninitialized states. Relative mappings follow Git's default
remote directory rule. Exclusions win over includes and recursion. Existing AX
layout paths are supplied by the path owner and normalized against Git's canonical
roots; ordinary missing roots differ from read errors. Dependency lockfiles such
as Cargo.lock remain project content; arbitrary live locks use explicit policy.

**7 of 7 scoped AC rows driven**, with sparse/linked worktrees grouped as in the
assignment. The attached ac-driver-audit.json verifies all named drivers appear
and pass in the latest repository-wide verbose log. This is a content-leaf ratio,
not full normative-section coverage. Candidate-resident tests await the managed
signed checkpoint after independent review; no producer hand-commit was made.

| AC row | Production call site | Named drivers |
| --- | --- | --- |
| Untracked bytes, with staged/working bytes distinct | Capture → contentState.capture → Guard.Open → blob → ObjectStore.PutBlob | TestContentCaptureSelectionAndBytes; TestContentNormativeWorkspaceBytes |
| Ignored files only through classified project/provider includes | Capture → contentState.validate/capture | TestContentIgnoredDefaultAndPolicyRefusal; TestContentNestedDetachedSubmodule |
| Symlink targets and containment | Capture → contentState.capture → safeSymlink; canonicaljson.CheckManifestEntries | TestContentSymlinks; TestContentSymlinkChainsAndExcludedTargets |
| Submodule state and repository boundaries | Capture → contentState.submodules → capture (recursive) | TestContentSubmodulePointerStates (clean/staged/unstaged/combined); TestContentSubmoduleUninitializedAndRefusals; TestContentNestedDetachedSubmodule; TestContentSubmoduleCountBounds; TestContentRelativeSubmoduleURL |
| Large blobs and protocol size refusal | Capture → contentState.blob → installBlob | TestContentLargeBlobChunksAndLimit; TestContentBlobInstallFailureAndRetry |
| Sparse/linked worktrees | Capture → readRepository/readFeatures → contentState.capture | TestContentSparseLinkedWorktree; TestContentSparseReadFailure; predecessor TestCaptureLinkedWorktreeSubdirectory |
| Mandatory exclusions and explicit owner policy | Capture → contentState.validate/excluded; submodules exclusion recheck | TestContentCaptureSelectionAndBytes; TestContentLocalPathOwnerExclusions; TestContentExcludedSubmoduleCannotBypassPolicy |

The exact pinned Section 10.4 parent/child pack/index corpus constructs real
repositories in separate object databases. Capture reproduces working bytes and
pointers. The corpus hash equals the pinned document hash
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a.
TestContentCrashAfterInstall drives a real child process to exit **73** after
immutable installs before any result; this is an expected failing capture, not
an exit-0 capture. The enclosing test verifies that status and a fresh successful
capture. TestContentBlobInstallFailureAndRetry exercises an unsafe store and
stable retry. TestContentDigestRaceAndRetry changes same-size bytes hidden from
ordinary Git diff and requires the included-file digest check to refuse.

## Executed validation

The latest repository-wide verbose and coverage commands each pass **24/24
packages**, exit 0. gitsnap reports **83.4%** statement coverage; canonicaljson
reports **97.1%**. TestDumpSweepSites is the single suite-level diagnostic skip;
no content driver is skipped. Some unchanged packages use the Go test cache;
mutation controls and behavioral selections always use -count=1. Relevant race
checks ran on gitsnap and canonicaljson; gitsnap was rerun after the last Go
change. Native runtime was macOS arm64, Go 1.25.5, Apple Git 2.50.1. Windows
build/vet are compile checks only. Linux runtime, the CI OS matrix, the full
repository race selection and a fresh unrelated-package fuzz budget were not
rerun: this leaf runs full ordinary tests/coverage and scopes additional race
work to the changed packages. No new platform capability claim is made.

No command was piped through tee. commands.json records all test/build/gate
attempts and real exits. The early package run (census ordering and missing new
bound proof), local-path recovery run, and first neutral mutation control failed
with exit 1; they remain failing attempts in the archive. They do not satisfy
checklist gates. Later runs below replace their evidence.

| Validation | Exit | Log |
| --- | ---: | --- |
| `go test ./... -v` | 0 | all-tests-03.log |
| `go test ./... -cover` | 0 | coverage-02.log |
| `go test -race ./internal/gitsnap ./internal/canonicaljson -count=1 -timeout=4m` | 0 | race.log |
| `go test -race ./internal/gitsnap -count=1 -timeout=4m` | 0 | race-gitsnap-02.log |
| `go build ./...` | 0 | build-02.log |
| `go vet ./...` | 0 | vet-02.log |
| `GOOS=windows go build ./...` | 0 | windows-build-02.log |
| `GOOS=windows go vet ./...` | 0 | windows-vet-02.log |
| `go run ./internal/traceability/cmd/tracecheck` | 0 | tracecheck.log |
| `go run ./internal/catalog/cmd/cataloggen -metadata internal/catalog/catalog.v0.5.0.json -contracts internal/specpin/v0.5.0.lock.json -output internal/catalog/catalog_gen.go -check` | 0 | catalog-check.log |
| `gofmt -l internal` | 0 | gofmt-02.log |
| `test ! -s .temp/TASK-260830-3m7m7w/gofmt-02.log` | 0 | empty formatting log verified |
| `git diff --check` | 0 | diff-check-02.log |
| `python3 .scripts/gitsnap-mutations.py --vectors internal/gitsnap/testdata/content-mutations.json --out .temp/TASK-260830-3m7m7w/mutations-04` | 0 | mutations-04.log |
| `python3 .scripts/gitsnap-mutations.py --vectors internal/gitsnap/testdata/content-mutations.json --only N-entry-set --out .temp/TASK-260830-3m7m7w/mutations-entry` | 0 | mutations-entry.log |
| `git verify-commit HEAD` | 0 | predecessor-signature.log |

## Mutation evidence

**6 of 6 newly registered capture gates** have executed narrowing witnesses:
CapturePolicy, ContentRead, ContentPath, SubmoduleState, BlobLimit and BlobInstall.
The new shared entry-array bound also has its own executed narrowing witness.
Additional vectors cover ignored selection, mandatory exclusions, recursive
exclusion bypass and included-file consistency. There are **11 distinct
narrowing kills**, one known-bad control kill and one measured neutral survivor
(13 distinct vectors). The two controls are repeated in the separate shared-owner
replay. All vectors apply and compile with exit 0; every kill below has behavioral
exit 1 and a named failing test. The neutral run has behavioral exit 0. The entry
bound's positive-control command also exits 0.

This is selected gate-level evidence, not every interior refusal clause or every
member of an open filesystem class. Source census registration alone never counts
as a kill. N-path keeps the searched guard/refusal tokens, changes behavior, and
executes the full gitsnap suite. The 65537 vector runs the behavioral boundary test,
not just the declared-bound source scanner.

| Mutant/control | What it narrows or changes | Named failing test | Survivor bound |
| --- | --- | --- | --- |
| C-content-neutral-comment | Comment-only control; no gate changed. | None — survivor | Comment-only change has no runtime effect; the complete gitsnap suite executes. |
| C-content-bad-clear-entries | Known-bad control clears the captured entry set. | `TestContentCaptureSelectionAndBytes` | Not a survivor |
| N-policy-admits-unclassified-allow | Allows the exact ignored/allow include with false non-secret classification. | `TestContentIgnoredDefaultAndPolicyRefusal` | Not a survivor |
| N-read-admits-quiet-fatal128 | Allows fatal Git exit 128 to enter the empty ignored-census path. | `TestContentReadFailureIsNotAbsence` | Not a survivor |
| N-path-admits-escaping-chain | Bypasses live containment only for escape-chain; original guard/refusal tokens remain; full behavioral suite executes. | `TestContentSymlinkChainsAndExcludedTargets` | Not a survivor |
| N-submodule-admits-stray-uninitialized | Allows an uninitialized directory containing the single file stray. | `TestContentSubmoduleUninitializedAndRefusals` | Not a survivor |
| N-blob-limit-admits-one-byte-over | Raises the 128 GiB bound by one byte. | `TestContentLargeBlobChunksAndLimit` | Not a survivor |
| N-install-admits-shard-failure | Ignores failure specifically at the object-shard creation stage. | `TestContentBlobInstallFailureAndRetry` | Not a survivor |
| N-exclusion-admits-auth-json | Removes auth.json from the mandatory conventional credential-name class. | `TestContentCaptureSelectionAndBytes` | Not a survivor |
| N-ignored-admits-deny | Includes the exact ignored/deny path without an include rule. | `TestContentIgnoredDefaultAndPolicyRefusal` | Not a survivor |
| N-consistency-admits-five-byte-digest-change | Allows digest disagreement for a five-byte included file. | `TestContentDigestRaceAndRetry` | Not a survivor |
| N-submodule-exclusion-bypass | Lets vendor/lib bypass a required-submodule policy exclusion. | `TestContentExcludedSubmoduleCannotBypassPolicy` | Not a survivor |
| N-entry-set-admits-65537 | Raises the shared entry-set limit from 65536 to 65537; the positive shape control remains green. | `TestCheckManifestEntriesCountBounds` | Not a survivor |

## Remaining boundaries and documentation

README, doc.go, content-acceptance.md and LOGBOOK.md describe the internal API,
policy owner, real platform findings and limits. No CLI or doctor capture
capability is advertised. Tracecheck still measures **17/428** global normative
clauses discharged; this leaf does not inflate that ownership count.

Object-pack production, raw-index blob delivery, complete workspace-root/child
manifest assembly, quiescence, end-to-end materialization and full race/closure
strengthening are not implemented by this content result. The final closure work
remains with TASK-260830-2xt6fd, as reinforced by operator directive
RUN-260907-d14c2f:nudge:49acdb, which was observed and recorded at a safe checkpoint.
No stronger wire-member or offline-materialization claim is made.

A successful full 128 GiB stream, every recursion-depth/grammar combination,
all adversarial filesystem substitution schedules, and sparse-pattern changes
after its read are not proven. The measured size fixture succeeds at 8 MiB + 4
bytes and refuses a real sparse file at 128 GiB + 1 byte. Existing private-index
auxiliary timestamp/hard-kill bounds remain. Arbitrarily named secrets or runtime
state require trusted owner exclusions; these filename rules and include
classifications are not a confidentiality guarantee. Failed captures can leave
verified unreferenced blobs, but cannot publish a partial manifest through this API.

## Attached packet

`TASK-260830-3m7m7w_capture-evidence.tar.gz` includes all command logs/exits,
old failing attempts, per-vector compile/behavior logs, the source manifest and
all changed sources, AC/mutant audits, exact goal/scope observations, and accepted
predecessor evidence. No process remains running for a future lifecycle to finish.

Archive SHA-256: 0cdd79565b2b487a3d63b11d512811d56d28d5967fd3ff63e82e3e170069b239.
