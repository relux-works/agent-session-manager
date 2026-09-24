# TASK-260830-nxqqaw handoff blocker

**State:** developer handoff was attempted and refused. `task-board handoff TASK-260830-nxqqaw --role developer --board-dir "$TASK_BOARD_DIR"` exited 1 because checklist items 1 and 7 are unchecked. The task has not moved to `to-review`.

The task was then marked `blocked` through the board CLI after this blocker was documented. That CLI automatically demoted parent STORY-260830-ub60id and EPIC-260830-3v9jlg from `development` to `to-dev`; no direct parent mutation was made.

## Exact gaps

1. The production deliverable is partial for four included schema/version rows. The inventory mapping names `session-record@3.1.0`, `materialization-plan@1.0.0`, `materialization-plan@2.0.0`, and `task-board-bundle@1.0.0`, but the current `canonicaljson` shape registry intentionally refuses them. This avoids accepting opaque or partially validated identities, but prevents those valid object classes from entering roots and prevents item 1 from being marked satisfied.
2. The pinned `internal/specdoc/SPEC.v0.7.0.md` MIXED-NS-EXCHANGE vector uses synthetic Checkpoint and Descriptor identifiers and supplies no corresponding object bytes. `AddJSON` validates each object's canonical identity, so bytes must hash to their exact identifiers. The digest identifiers cannot be converted into their missing preimages. A mock object source or bypass around `VerifyObjectIdentity` would conceal, not satisfy, the validated-add rule.

The real-byte in-process exchange is not being presented as the exact synthetic fixture. It retrieves its exact Checkpoint and Descriptor, validates additions, changes only record/manifest/blob roots, delays blob transfer until record convergence, and pins six final roots. Exact synthetic combined roots remain unclaimed.

## Attempts and evidence

- All feasible trie, serving, mapping, wrong-namespace, real-byte exchange, tombstone/Ack union, and negative-gate tests passed in bounded groups; 29 inventory/canonicaljson/provhost narrowing mutants and seven delegated `rpcwire` narrowings were killed. Detailed test, race, coverage, build, vet, and mutation logs are in the attached evidence archive.
- The synthetic identity gap was handled without weakening identity validation: an alternate real-byte fixture was added and reported as such.
- The developer handoff command was run after all current evidence was attached and read back byte-identically. It refused exactly because checklist items 1 and 7 remain open.

## Resolution needed

For item 1, the task/story owner must choose whether this leaf expands to add complete canonicaljson validators for all four mapped schemas, or keeps those rows fail-closed and splits schema admission into a follow-up while narrowing this leaf's acceptance. Recommendation: preserve fail-closed behavior unless complete closed-shape validators and positive fixtures are added; do not add shallow validators.

For item 7, either provide valid canonical object bytes for the exact synthetic Checkpoint and Descriptor IDs, or authorize replacing the synthetic IDs with identities computed from valid fixture bytes and updating the expected roots. If the vector is intentionally an abstract root-only fixture, authorize splitting the root-vector proof from the validated object exchange and revise the checklist accordingly. Recommendation: use actual digest-valid object fixtures and explicitly update the expected roots, unless the original preimage bytes are available.

The precise input needed is the Story/spec owner's decision on both scope boundaries, plus either the missing valid fixture bytes or authorization to retarget the fixture and its expected roots. Until then, the two checklist items stay open and the role handoff cannot proceed.
