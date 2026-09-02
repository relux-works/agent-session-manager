# Closed-vocabulary admission inventory

Normative source: `relux-works/agent-session-manager-spec` v0.5.0 at commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, SPEC.md SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.

Every row is one production closed-vocabulary admission site. The `Function`,
`Member` and `Admits` columns are derived from the package sources by
`TestEveryClosedVocabularyAdmitsExactlyItsPinnedSet`, which walks every call to
a derived vocabulary-admitting helper and requires this table to match it
exactly, in both directions and in declaration order.

This artifact exists because a widening mutant was invisible. Admitting one
extra member at any of these call sites left `go test ./...`, all four seeded
fuzz corpora, `tracecheck`, and the derived refusal-guard coverage gate green:
47 of 47 such mutants survived. A "refuses one outside value" test proves the
gate is reachable, not that the admitted set is the declared set. Binding the
admitted set to a reviewed pin is what makes widening fail.

Rows are compared as an ordered sequence in production source order, so no
production line number appears here and an unrelated edit above a call site does
not churn the artifact. `Scope` is the enclosing registry key when the call sits
inside a composite literal - the Session Event type whose payload declares that
vocabulary - and `-` otherwise.

`SPEC line` and `Pinned SPEC declaration` record where the pinned document
declares that vocabulary, so a row is checkable against the specification rather
than against this repository. Three rows cite a tagged-union row header or a
prose sentence instead of an `a&#124;b` alternation, because that is the form
the document uses for them.

| Source | Function | Scope | Member | Admits | SPEC line | Pinned SPEC declaration |
| --- | --- | --- | --- | --- | ---: | --- |
| `closed_shapes.go` | `validateSessionRecordCommon` | `-` | `kind` | `direct`, `task_board` | 1472 | `kind &#124; enum &#124; direct or task_board` |
| `closed_shapes.go` | `validateSessionRecordCommon` | `-` | `execution_profile` | `standard`, `yolo` | 1477 | `execution_profile &#124; enum &#124; standard or yolo` |
| `closed_shapes.go` | `validateSessionTaskBoardReference` | `-` | `launch_mode` | `primary_owner`, `tracked_prompt` | 1508 | `launch_mode &#124; enum &#124; primary_owner or tracked_prompt` |
| `closed_shapes.go` | `validateSessionTaskBoardReference` | `-` | `native_goal_binding` | `bound`, `prompt`, `none` | 1511 | `native_goal_binding &#124; enum &#124; bound, prompt, or none` |
| `closed_shapes.go` | `validateSessionBoardIdentity` | `-` | `kind` | `local`, `remote` | 1514 | `Board Identity has exactly kind (local or remote)` |
| `closed_shapes.go` | `validateSessionForkProvenance` | `-` | `provider_fork_mode` | `native`, `supported_import`, `task_board_clone` | 1653 | `provider_fork_mode:native&#124;supported_import&#124;task_board_clone` |
| `closed_shapes.go` | `validateSessionSameProviderForkProvenance` | `-` | `provider_fork_mode` | `native`, `supported_import`, `task_board_clone` | 1653 | `provider_fork_mode:native&#124;supported_import&#124;task_board_clone` |
| `closed_shapes.go` | `validateSessionCrossEnvironmentCloneProvenance` | `-` | `source_kind` | `ax_session`, `external_native` | 1635 | `source_kind=ax_session&#124;external_native` |
| `closed_shapes.go` | `validateEnvironmentTuple` | `-` | `architecture` | `amd64`, `arm64` | 3073 | `architecture:amd64&#124;arm64` |
| `closed_shapes.go` | `validateTransferManifest` | `-` | `kind` | `workspace_group`, `workspace_tree`, `provider`, `task_board`, `composite` | 4693 | `kind &#124; enum &#124; workspace_group, workspace_tree, provider, task_board, or composite` |
| `closed_shapes.go` | `validateManifestEntries` | `-` | `type` | `directory`, `file`, `symlink`, `hardlink` | 4745 | `tagged-union row headers type = directory, file, symlink, hardlink` |
| `closed_shapes.go` | `validateWorkspaceSnapshotMember` | `-` | `kind` | `git`, `managed_tree` | 2162 | `tagged-union row headers kind = git and kind = managed_tree` |
| `closed_shapes.go` | `validateWorkspaceSnapshotMember` | `-` | `materialization_policy` | `shared_tree`, `separate_copy` | 2163 | `materialization_policy:shared_tree&#124;separate_copy` |
| `closed_shapes.go` | `validateWorkspaceSnapshotMember` | `-` | `materialization_policy` | `shared_checkout`, `separate_worktree` | 2162 | `materialization_policy:shared_checkout&#124;separate_worktree` |
| `closed_shapes.go` | `validateGitHead` | `-` | `mode` | `branch`, `detached`, `unborn` | 4788 | `mode:branch&#124;detached&#124;unborn` |
| `closed_shapes.go` | `validateGitObjectPack` | `-` | `object_format` | `sha1`, `sha256` | 4789 | `object_format:sha1&#124;sha256` |
| `closed_shapes.go` | `validateGitFeatures` | `-` | `object_format` | `sha1`, `sha256` | 4789 | `object_format:sha1&#124;sha256` |
| `core_records.go` | `validateLeaseRecord` | `-` | `reason` | `create`, `graceful_takeover`, `force_takeover`, `recovery` | 1908 | `reason &#124; enum &#124; create, graceful_takeover, force_takeover, recovery` |
| `core_records.go` | `validateSafeBoundaryEvidence` | `-` | `evidence` | `provider_api`, `provider_event`, `managed_pty`, `task_board_bridge`, `accepted_test` | 1994 | `evidence:provider_api&#124;provider_event&#124;managed_pty&#124;task_board_bridge&#124;accepted_test` |
| `core_records.go` | `validateProviderIdentityRecord` | `-` | `identity_kind` | `session_uuid`, `session_path_or_id`, `backend_conversation_uuid`, `task_board_managed`, `provider_defined` | 2086 | `identity_kind &#124; enum &#124; session_uuid, session_path_or_id, backend_conversation_uuid, task_board_managed, or provider_defined` |
| `core_records.go` | `validateWorkspaceMember` | `-` | `materialization_policy` | `shared_checkout`, `separate_worktree` | 2162 | `materialization_policy:shared_checkout&#124;separate_worktree` |
| `core_records.go` | `validateWorkspaceMember` | `-` | `materialization_policy` | `shared_tree`, `separate_copy` | 2163 | `materialization_policy:shared_tree&#124;separate_copy` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `terminal.created` | `backend` | `tmux`, `conpty` | 1771 | `backend:tmux&#124;conpty` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `provider.launched` | `execution_profile` | `standard`, `yolo` | 1477 | `execution_profile &#124; enum &#124; standard or yolo` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `provider.identified` | `confidence` | `exact`, `strong`, `weak` | 1773 | `confidence:exact&#124;strong&#124;weak` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `session.quiescing` | `reason` | `graceful_takeover`, `stop`, `checkpoint` | 1775 | `reason:graceful_takeover&#124;stop&#124;checkpoint` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `checkpoint.created` | `kind` | `periodic`, `pre_stop`, `closure`, `fork_base`, `manual` | 1776 | `kind:periodic&#124;pre_stop&#124;closure&#124;fork_base&#124;manual` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `session.resumed` | `execution_profile` | `standard`, `yolo` | 1477 | `execution_profile &#124; enum &#124; standard or yolo` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `session.resumed` | `terminal_backend` | `tmux`, `conpty` | 1779 | `terminal_backend:tmux&#124;conpty` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `session.parked` | `reason` | `remote_owner`, `stale_owner`, `restore_policy`, `failed_handoff` | 1783 | `reason:remote_owner&#124;stale_owner&#124;restore_policy&#124;failed_handoff` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `fork.created` | `provider_fork_mode` | `native`, `supported_import`, `task_board_clone` | 1653 | `provider_fork_mode:native&#124;supported_import&#124;task_board_clone` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `fork.created` | `execution_profile` | `standard`, `yolo` | 1477 | `execution_profile &#124; enum &#124; standard or yolo` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `replica.replace_confirmed` | `confirmation_mode` | `interactive`, `non_interactive` | 1788 | `confirmation_mode:interactive&#124;non_interactive` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `tombstone.issued` | `scope` | `session`, `workspace_entry`, `provider_snapshot`, `managed_replica` | 1792 | `scope:session&#124;workspace_entry&#124;provider_snapshot&#124;managed_replica` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `clone.target_validation_failed` | `phase` | `prepublication_source_recheck`, `provider_prepare`, `postpublication_source_recheck`, `live_discovery`, `live_read_back`, `resume_plan` | 9993 | `phase:prepublication_source_recheck&#124;provider_prepare&#124;postpublication_source_recheck&#124;live_discovery&#124;live_read_back&#124;resume_plan` |
| `core_records.go` | `mustBuildSessionEventPayloadShapes` | `-` | `execution_profile` | `standard`, `yolo` | 1477 | `execution_profile &#124; enum &#124; standard or yolo` |
| `core_records.go` | `validateSessionStoppedPayload` | `-` | `closure_kind` | `checkpointed`, `bootstrap_abort` | 1778 | `closure_kind:checkpointed&#124;bootstrap_abort` |
| `core_records.go` | `validateBootstrapAbortedPayload` | `-` | `failure_phase` | `before_terminal`, `after_terminal`, `after_process`, `after_identity`, `before_checkpoint` | 1780 | `failure_phase:before_terminal&#124;after_terminal&#124;after_process&#124;after_identity&#124;before_checkpoint` |
| `core_records.go` | `validateProfileChangedPayload` | `-` | `from` | `standard`, `yolo` | 1786 | `from:standard&#124;yolo` |
| `core_records.go` | `validateProfileChangedPayload` | `-` | `to` | `standard`, `yolo` | 1786 | `to:standard&#124;yolo` |
| `core_records.go` | `validateForceConfirmedPayload` | `-` | `confirmation_mode` | `interactive`, `non_interactive` | 1788 | `confirmation_mode:interactive&#124;non_interactive` |
| `core_records.go` | `validateTaskBoardLaunchedPayload` | `-` | `launch_mode` | `primary_owner`, `tracked_prompt` | 1508 | `launch_mode &#124; enum &#124; primary_owner or tracked_prompt` |
| `core_records.go` | `validateTaskBoardLaunchedPayload` | `-` | `execution_profile` | `standard`, `yolo` | 1477 | `execution_profile &#124; enum &#124; standard or yolo` |
| `core_records.go` | `validateTaskBoardLaunchedPayload` | `-` | `state` | `running`, `idle` | 1790 | `state:running&#124;idle` |
| `core_records.go` | `validateTombstoneResolvedPayload` | `-` | `resolution` | `deleted`, `already_absent`, `resurrected`, `retained_conflict` | 1793 | `resolution:deleted&#124;already_absent&#124;resurrected&#124;retained_conflict` |
| `core_records.go` | `validateObservationEvent` | `-` | `level` | `debug`, `info`, `warn`, `error` | 11594 | `level &#124; enum &#124; debug, info, warn, or error` |
| `core_records.go` | `validateObservationEvent` | `-` | `result` | `started`, `success`, `partial`, `failure`, `cancelled` | 11601 | `result &#124; enum &#124; started, success, partial, failure, or cancelled` |
