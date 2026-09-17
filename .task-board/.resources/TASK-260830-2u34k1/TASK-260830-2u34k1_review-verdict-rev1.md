# TASK-260830-2u34k1 — revision 1 review verdict

Verdict: **accepted**. No blocking findings. Reviewer: RUN-260907-a0c13a.
This accepts the read-only identity/configuration leaf, not transport execution
or completed delivery. The next authorized action belongs to the bound
developer/implementer integration run. No code, index, branch or commit was
changed by this reviewer.

## Candidate and live state

- CR: CR-TASK-260830-2u34k1-1, revision 1, repository_delta=present.
- Base: 7654d7cadb2c226bfa3db5f23ac355a730285eea.
- Candidate tree: c775903a1aa76cb3d433aeee89ebfd24e73159dd.
- Patch SHA-256: 9de2e0fc886dbb2c48b50e477b1a20f04ab0d44bd61d3b327e61a2099846a7ff.
- Live `worktree status STORY-260830-1kiyj6 --json` confirmed this ready revision,
  its 12 paths, candidate/base/hash and developer/implementer owner. Downloaded
  patch hash matched. Every one of the 2,925 candidate files matched the working
  tree by Git blob identity; no content mismatches. HEAD remained the base and
  HEAD..main was 0. Measurements are for this candidate, not a later trunk.
- Producer archive file hashes match the reviewed files. The merged checklist
  was read before code review: producer items 1–13 complete, reviewer items
  14–18 initially open. Reviewer conclusions below discharge 14–17; item 18's
  non-acceptance branch is inapplicable to this accepted verdict.
- `task-board spawn goal` reports this run is not goal-bound. No directives.

## AC coverage, checked before code review

**7 of 7 scoped AC rows driven** by named candidate tests. These are the seven
deliverables in this task, not all obligations of the referenced sections.

| AC | Production entry/call | Named candidate driver |
| --- | --- | --- |
| Host IDs | peeridentity.Load -> config.Load/Decode -> scalar UUIDv7 validation | TestLoadPeerIdentityVersions; TestLoadIdentityRefusals; TestLoadPinnedPeerConfigurationExample |
| Aliases | Directory.Resolve | TestResolveAllowlistAndAliasAmbiguity; TestSnapshotIsolationAndDisclosureBinding |
| SSH targets | Directory.Resolve -> Target.RPCArgv | TestSSHTargetAtomicArgv; TestSSHTotalArgvByteBound; TestLoadIdentityRefusals |
| Key provenance | Target.KeyProvenance; Load's selected-file reader | TestLoadAbsenceReadFailureAndRecovery; TestLoadPeerIdentityVersions |
| Allowlist | Directory.Resolve; Target.CheckProtocolHost | TestResolveAllowlistAndAliasAmbiguity; TestProtocolIdentityRefusal |
| Disclosure | Directory.DisclosurePolicy | TestDisclosurePolicyRefusals; TestDisclosureClassPolicyBinding; TestSnapshotIsolationAndDisclosureBinding |
| Duplicate identity refusal | peeridentity.Load -> config.validateMesh | TestLoadIdentityRefusals; config.TestLoadRefusesLocalHostAsPeer |

The tests are in the candidate; the managed producer checkpoint will commit
them. This review does not pretend they are already committed. Caller search
confirmed that no executable/transport invokes peeridentity yet: its exported
read-only API is the production boundary for this leaf. The task explicitly
leaves transport execution and sibling integration outside scope, and README
states that limitation. The library tests drive that actual API and config
loader, not a substitute authorizer or test-only implementation.

## Normative fit and bounds

Read the pinned SPEC.md sections 6, 11.1 and 16.1. Its bytes hash to
562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a, the pinned
v0.5.0 source at commit 28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c.
The five disclosure classes were independently extracted from the §6.4
directory_peer_disclosure paragraph and compared with the class test matrix:
**5 of 5 normative metadata classes**. This is not an output-schema census or
a claim to authorize arbitrary synchronized payloads.

SSH owns key/user authentication under §11.1. `external_ssh` reports that
authority, not observed keys or successful authentication. Protocol-ID checking
is explicitly conditional on transport authentication; there is no forged
`verified` input. Zero snapshots/targets do not authorize. No real remote
connection or private credential inspection was performed.

Aliases preserve exact Unicode spelling. Canonical config owns UUID, endpoint,
argument, platform and disclosure shape validation. Duplicate local/remote
host identity and duplicate peers refuse; cross-peer ID/alias collisions refuse
resolution. Snapshot copies prevent caller mutation of the validated allowlist.
Invalid configuration never returns a partially usable directory. Error text
does not render synthetic rejected values; explicit underlying error inspection
remains the existing diagnostic API, not a confidentiality guarantee.

Disclosure returns configured policy only. Object schema validation, object
policy, sanitization, authenticated publishing and existing replicated-data
retention belong to their owners. Five-class coverage does not claim their
implementation. Default/override handling and unset-summary refusal agree with
this configuration slice. No durable mutation was introduced: crash-write
injection is inapplicable; failed-read recovery and isolation are exercised.

Traceability changes add seven acceptance links (105 total) while preserving
the honest 17/428 measured keyword-clause result and unevidenced section gaps.
README and .spec/README describe APIs and these bounds without adding CLI,
doctor, transport authentication, key attestation or at-rest encryption claims.
The ownership digest change matches the actual reviewed semantic registry.

## Reviewer-run validation

Full logs are in TASK-260830-2u34k1_review-evidence-rev1.zip.

| Command/check | Exit | Evidence |
| --- | ---: | --- |
| go test ./internal/peeridentity ./internal/config ./internal/traceability/... -count=1 -v -cover | 0 | focused-01.log; all 4 packages pass |
| go test ./... -count=1 -v | 0 | all-tests-01.log; all 24 packages pass |
| go test ./... -cover | 0 | all-cover-01.log; all 24 packages pass; peeridentity 97.5% |
| go vet ./... | 0 | vet-01.log |
| go build ./... | 0 | build-01.log |
| gofmt -l on the changed Go paths | 0 | format-01.log; no output |
| git diff --check BASE CANDIDATE | 0 | diff-check-01.log |
| python3 internal/peeridentity/mutations.py --output REVIEW/mutations | 0 | mutations-01.log and mutations/results.json |
| go test ./internal/peeridentity -count=1 -v -run '^TestReviewer' in disposable candidate copy | 0 | neighbors-01.log; 2 top-level tests and 10 subtests actually ran |

The reviewer independently reran the full 21-probe behavioral instrument:
18 of 19 narrowing probes killed, 1 compiling known-bad control killed,
1 compiling neutral control passed. The grammar-only protocol-ID survivor
cannot authorize the alias because equality to the validated UUID still
refuses; it is explicitly subsumed, not a behavioral kill. Every counted kill
has a named failing behavioral test, not merely a compile/static-census failure.
No not-applied or compile-failed probe is counted as evidence. The new leaf adds
no source-text enforcement checker. Existing traceability checker semantics
are unchanged; its behavioral tests were included in the full rerun.

Independent disposable tests go beyond producer vectors:

- TestReviewerSSHNativeInterpretation sends actual RPCArgv to `ssh -G -F
  /dev/null` (configuration-only, no connection): four targets prove leading-zero
  endpoint ports, IPv6 forms, endpoint port precedence and SSH option port
  precedence, fixed no-PTY and enforcing host-key settings. Only selected
  synthetic fields are inspected; no default key paths are persisted.
- TestReviewerNeighbors/unicode_aliases_stay_exact tests precomposed versus
  decomposed Unicode names as two exact aliases.
- later_invalid_member_cannot_use_earlier_allowlist tests a later duplicate ID,
  zero port and safe-then-unsafe SSH arguments; no partial peer directory escapes.
- read_disappears_after_successful_stat tests a ReadFile ENOENT after a successful
  Stat: failed read does not become normal missing configuration.
- malformed_policy_cannot_use_permissive_default tests empty, malformed and
  wrong-typed peer disclosure under mesh_sanitized defaults.
- unknown_class_under_alias_collision tests policy through an ambiguous selector
  and neighboring unknown/mis-cased classes under permissive defaults.
- invalid_snapshot_gives_no_provenance tests absent evidence through
  FromSnapshot -> Resolve -> KeyProvenance.

Accepted from the producer's attached archive, not rerun by this reviewer:
full uncached race suite (race-02.exit=0, 24 package PASS rows), configured fuzz
smokes and Linux/Windows cross-build evidence. Producer reports earlier red
iterations separately; this verdict does not relabel them. Cross-builds do not
prove native Windows/Linux execution, and fuzz budget does not prove each target
mutated beyond seeds. No unrelated universal audits or shared board repairs.

## Operational notes

The worktree has no installed .codex skills; the repository's Curator-managed
Go skill was read from the main checkout. No install or runtime replacement.
Initial missing-path reads and a mistaken `task(...)` query were corrected to
the installed skill and documented `get(...)` query; no result was inferred from
those failed reads. An initial `git diff CANDIDATE` view omitted untracked new
files and was not used as drift evidence; the complete blob verification above
established candidate identity. All probes are under ignored .temp paths.

No changes_requested finding; repeat-of: none. No stop-the-line decision needed.
Record acceptance with accept_cr only after completing the live merged checklist.
