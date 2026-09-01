# TASK-260830-8x76g1 review verdict — CR revision 3

## Verdict

**Changes requested.** Route the task to `to-dev` for another implementation and reviewer cycle. This is ordinary recoverable rework, not a Stop-The-Line boundary.

## Candidate binding

- Change Request: `CR-TASK-260830-8x76g1-3`, revision `3`
- Base OID: `c9e5290b1506275f5417b26070fad0391a09c50a`
- Candidate tree OID reviewed: `93e6d212a49ab7320e061254d9507a567ccd3852`
- Patch SHA-256 independently reproduced: `2af134465f1d23f8af2b0ec760a4245a84f0fb75486fb6789cab1b8575a0a0a7`
- Normative source inspected at `relux-works/agent-session-manager-spec` commit `28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`.

## Blocking finding F1 — the recursive Transfer Manifest gate is narrowed to member names

`CalculateObjectIdentity` and `VerifyObjectIdentity` both reach the intended composed production call site:

`prepareObjectIdentity -> validateImmutableObjectShape -> validateTransferManifest -> validateWorkspaceSnapshot -> validateWorkspaceSnapshotMember`.

However, the nested validators enforce mostly `requireExactMembers` and leave the declared Section 10.4 value constraints unchecked. In particular:

- `validateWorkspaceSnapshotMember` does not enforce `tree_identity:string[1..256]`, `repository_identity:string[1..256]`, several nested digest/path/enum constraints, or the managed/Git variant value rules.
- `validateGitRemote` does not enforce `name:string[1..128]` or its value types.
- `validateGitObjectPack` does not validate its digest IDs, object format, or count.
- `validateGitIndex` and its entry callback do not enforce index version, stage `[0..3]`, field types, sorting, or count equality.
- `validateGitSubmodule` does not enforce the initialized/uninitialized nullability state machine.

This violates the scoped Section 1.6 rule that conformance validation select common-type, nullability, tagged-union, and sorted-unique rules from the negotiated schema/version and exact JSON path; a malformed value does not become valid after recomputing the containing self-ID. It also misses the Section 10.4 normative `TM-GIT-N2` refusal for stage `4`.

The reviewer attack drove both public identity entries with correctly recomputed omit-self claims. Both calculation and verification admitted all of these invalid values:

1. managed-tree `tree_identity` containing 257 Unicode characters;
2. Git `repository_identity` containing 257 Unicode characters;
3. `GitRemote.name` containing 129 Unicode characters;
4. managed-tree `tree_manifest_id = "not-a-digest"`;
5. `GitObjectPack.blob_id = "not-a-digest"`;
6. `GitIndexEntry.stage = 4`; and
7. an uninitialized submodule carrying a non-null live head.

The corresponding valid 256-character multibyte repository identity was accepted. This proves the positive boundary is representable and the failure is under-refusal, not an encoding limitation.

This is a gate-present-but-narrowed defect at the production attestation boundary. The configured closed-shape fuzz target stays green because its mutation vocabulary covers unknown members, extension keys, and BlobChunk invariants but not the nested value constraints above.

## Blocking finding F2 — README makes an unsupported completed-audit claim

The README says declared `string[n..m]` bounds count Unicode characters and that every closed Manifest nested object is recursively validated. Revision 3 validates only the Blob media type and symlink target bounds. The three Section 10.4 nested bounds named in F1 are not enforced at all, so this statement is broader than reproducible production behavior.

## Validation evidence

Reviewer-rerun commands against the exact extracted candidate tree:

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | PASS |
| format, `go build ./...`, `go vet ./...` | PASS |
| focused canonical/scalar race tests | PASS |
| focused coverage | PASS; canonicaljson 81.2%, scalar 90.5% |
| all four configured fuzz targets, each `-fuzztime=100x -parallel=1 -count=1` | PASS |
| scoped tracecheck for 1.6, 10.1-10.4, 17.3 | PASS (`assigned_scopes=6`) |
| reviewer production-entry attack | FAIL as expected; seven invalid shapes were attested by both entries |

The green baseline and fuzz suite are therefore accepted as evidence that the candidate is buildable and deterministic, but not as evidence that the recursive validation gate covers the contract it claims.

## Required rework

1. Audit every Section 10.4 nested validator, not only the seven examples. Enforce each declared common type, Unicode character bound, nullable/tagged-union invariant, sorted-unique/order rule, and cross-field constraint reachable at this scoped identity gate.
2. Reuse or extend the production scalar validators instead of duplicating looser local grammars.
3. Add production-entry negative tests and deterministic fuzz corpus cases that fail on narrowed nested validators. Include at minimum all seven reproduced shapes and the normative stage-4 refusal, plus valid multibyte values at each character limit.
4. Keep every fuzz target wired once with the fixed budget; extend the closed-shape target's mutation vocabulary so the new gate can actually be defeated if narrowed again.
5. Narrow or correct README claims until they reproduce through both `CalculateObjectIdentity` and `VerifyObjectIdentity`.

