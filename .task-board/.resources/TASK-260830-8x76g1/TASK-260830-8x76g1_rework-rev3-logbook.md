# TASK-260830-8x76g1 rework revision 3 logbook

- The CR revision 2 extension-key bypass and Unicode over-refusal shared one boundary: `validateMigrationExtensionObject` was special-key-only, while the only bounded non-ASCII string used byte length. The fix is centralized rather than keyed to the review examples.
- Exact pinned spec inspection confirmed Section 1.6 requires reverse-DNS key grammar, 64 members, depth 4, 65,536 canonical bytes, and Unicode-character `string[n..m]` bounds. Sections 10.4 and 17.3 both reach the shared extension validator.
- Expected-red focused validation exited 1 before the fix and demonstrated the production bypass through `CalculateObjectIdentity`; post-fix tests also drive `VerifyObjectIdentity` with a correctly recomputed omit-self claim, preventing a verification-only bypass.
- The validation configuration had three fixed fuzz commands but omitted `FuzzClosedIdentityShapeRefusal`. A source-discovery regression test now fails whenever any repository fuzz target lacks its exact fixed-budget command.
- The closed-shape corpus now carries named invalid-extension-key seeds for both Transfer Manifest and generic record extension points.
- `task-board validate` exited 0 and retained 262 inherited `MISSING_ACTIVITY` diagnostics. The current task is not among them; no board files were edited directly.
- No migration publication, atomic-reference behavior, doctor result, runtime capability, or durable-state recovery behavior is claimed.
