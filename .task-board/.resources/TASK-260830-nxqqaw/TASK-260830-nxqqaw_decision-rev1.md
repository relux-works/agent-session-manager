ORCHESTRATOR DECISION on the TASK-260830-nxqqaw blocker (`TASK-260830-nxqqaw_blocker.md`). Both open items are leaf-scope decisions. They are made here, and no human input is needed. Continue from the uncommitted candidate in `.temp/STORY-260830-ub60id/worktree`; do not restart the work.

ITEM 7 — MIXED-NS-EXCHANGE: split the proof exactly as the spec's wording allows.
Pinned SPEC v0.7.0 §11.4 says the MIXED-NS-1 synthetic IDs "stand for schema-valid objects/bytes … their contents are not used to compute the inventory trie beyond their validated identities". The normative fixture therefore specifies INVENTORY behaviour over identity sets. It does not supply object bytes, and your refusal to invent preimages or to bypass `VerifyObjectIdentity` is correct. Deliver two named, committed proofs, and report them as two rows:
 (a) **Identity-level exchange on the exact synthetic IDs.** Drive both peers' inventories from the fixture's identity sets through the production trie/serving/walk entries.
   - The six MIXED-NS-1 roots are byte-exact.
   - Peer B, missing Z(3), T(2) and the 64-`5` blob ID, differs from A in exactly the record, manifest and blob roots.
   - The recursive `inventory.children` walk from those roots yields EXACTLY {Z(3), T(2), blob-5} as the missing set, and nothing else.
   - After B's identity sets gain those three, all six roots equal the fixture.
   - MIXED-NS-N1 (Descriptor classified as record, chunks enumerated independently, a local marker included) fails the expected roots.
   - This row claims no object validation.
 (b) **Validated union on real bytes.** Keep your real-byte exchange: validated add, idempotent identical add, same-digest-different-bytes quarantine and abort, tombstones unioned and not executed, no timestamp winner, blob transfer only after record union. Its expected roots are derived by the same production construction from the real identities and pinned as literals.
 Check item 7 when (a) and (b) are both committed, driven through production entries, and each has a narrowing its named test kills when run alone. Record in the results that the exact synthetic-ID row is identity-level by the spec's own definition.

ITEM 1 — the four schemas the canonicaljson shape registry refuses (`session-record@3.1.0`, `materialization-plan@1.0.0`, `materialization-plan@2.0.0`, `task-board-bundle@1.0.0`): KEEP FAIL-CLOSED. Do not add shallow validators.
Union rule 1 ("an unknown digest is added only after schema and digest validation") requires refusal for an object whose schema cannot be validated. The namespace MAPPING stays total: each of the four rows maps to its namespace in the table. Only ADMISSION waits on a validator. Concretely:
 - Pin the refusal. For each of the four schema IDs, a committed test proves the object is refused with a typed, literal code, does not move any root or count, and does not quarantine as a same-digest conflict. Ship a narrowing that admits it, and show that narrowing is killed.
 - Record each row as a stated BOUND in the conformance matrix and TRACEABILITY, with its owner. Find the owner on the board rather than guessing: the Story that owns that schema's validator (session record, materialization plan, task-board bundle). Name its element ID. If none exists, write `owner: unassigned — orchestrator to file`, and I will file it.
 - Check item 1 on that basis, and quote this decision in the results.

THEN: rerun the affected package tests, the importer grid, `GOOS=windows GOARCH=amd64 go vet ./...` and the full configured suite once. Update the results, matrix and evidence archive (under 1 MiB). Set the task back from `blocked` to development yourself, keep the checklist current, and run `task-board handoff TASK-260830-nxqqaw --role developer`. Trunk is still `360c8bd`. If it moves before your handoff, run `task-board worktree refresh-candidate TASK-260830-nxqqaw` first: CR construction requires the checkpoint to descend from current trunk. Canonical CLI: /Users/iv/.curator/global/bin/task-board. Model: gpt-6-luna max.
