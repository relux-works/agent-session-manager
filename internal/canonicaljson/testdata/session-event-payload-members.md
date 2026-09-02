# Session Event payload per-member declaration inventory

Normative source: `relux-works/agent-session-manager-spec` v0.5.0 Sections 5.2,
13.14.5, and 13.15 at commit
`28bf96d7dd7ebf3cd9e2ccd91d35b8660699dd5c`, SPEC.md SHA-256
`562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a`.

Every row is mechanically extracted from a pinned `Exact payload members` table:
Section 5.2 line 1768 (the v1 registry), Section 5.2 line 1869 (the two 4.0.0
Terminal Instance overrides), Section 13.14.5 line 9988 (the 2.0.0 clone
variants), and Section 13.15 line 10199 (the 3.0.0 directory variants). The
declaration column is the verbatim table cell with `<code>` markup removed; a
literal `|` inside a declaration keeps the specification's own `&#124;` entity
so the row stays one Markdown cell.

This artifact exists because the Session Event payload gate calls
`requireExactMembers` with a registry-derived member slice rather than string
literals, so the literal-argument inventory in `constraint-enumeration.md`
cannot reach it and previously left every payload member unenumerated.
`TestSessionEventPayloadMembersMatchPinnedSpecInventory` reads this file and
requires the production `sessionEventPayloadShapes` registry to declare exactly
these members, in this order, for every contract version the pinned catalog
assigns to the event type. Adding, dropping, renaming, or reordering a
production payload member fails that test.

`Definition` is `default` for a payload used by every contract version the
catalog assigns to the event type, and `4.0.0-override` for the two payloads
Section 5.2 replaces at Session Event 4.0.0: it pins that 4.0.0 "changes exactly
the payload definitions for `terminal.created` and `session.resumed`; every
other v1-v3 payload remains byte-for-byte the definition selected by v3". This
column is not a version registry. Which event types exist in which contract
version is derived from the pinned catalog by
`validateSessionEventPayloadShapeCompleteness`, never from this file.

| Definition | Event type | Member | Pinned SPEC declaration |
| --- | --- | --- | --- |
| `default` | `session.created` | `session_record_id` | `session_record_id:digest` |
| `default` | `session.created` | `bootstrap_operation_id` | `bootstrap_operation_id:UUIDv7` |
| `default` | `session.created` | `first_checkpoint_operation_id` | `first_checkpoint_operation_id:UUIDv7` |
| `default` | `terminal.created` | `backend` | `backend:tmux&#124;conpty` |
| `default` | `terminal.created` | `terminal_id` | `terminal_id:string[1..512]` |
| `default` | `provider.launched` | `provider_id` | `provider_id:provider-id` |
| `default` | `provider.launched` | `provider_version` | `provider_version:string[1..128]` |
| `default` | `provider.launched` | `execution_profile` | `execution_profile:standard&#124;yolo` |
| `default` | `provider.launched` | `profile_source_event_id` | `profile_source_event_id:digest&#124;null` |
| `default` | `provider.launched` | `profile_mapping` | `profile_mapping:string[1..512]` |
| `default` | `provider.identified` | `provider_identity_record_id` | `provider_identity_record_id:digest` |
| `default` | `provider.identified` | `confidence` | `confidence:exact&#124;strong&#124;weak` |
| `default` | `session.idle` | `boundary_ref` | `boundary_ref:string[1..1024]` |
| `default` | `session.idle` | `foreground_idle` | `foreground_idle:boolean` |
| `default` | `session.idle` | `background_idle` | `background_idle:boolean` |
| `default` | `session.quiescing` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `session.quiescing` | `reason` | `reason:graceful_takeover&#124;stop&#124;checkpoint` |
| `default` | `session.quiescing` | `input_blocked` | `input_blocked:boolean` |
| `default` | `checkpoint.created` | `checkpoint_id` | `checkpoint_id:digest` |
| `default` | `checkpoint.created` | `kind` | `kind:periodic&#124;pre_stop&#124;closure&#124;fork_base&#124;manual` |
| `default` | `sync.completed` | `peer_host_id` | `peer_host_id:UUIDv7` |
| `default` | `sync.completed` | `checkpoint_id` | `checkpoint_id:digest` |
| `default` | `sync.completed` | `manifest_ids` | `manifest_ids:sorted unique digest[1..1024]` |
| `default` | `sync.completed` | `materialized` | `materialized:boolean` |
| `default` | `session.stopped` | `graceful` | `graceful:boolean` |
| `default` | `session.stopped` | `checkpoint_id` | `checkpoint_id:digest&#124;null` |
| `default` | `session.stopped` | `resumable` | `resumable:boolean` |
| `default` | `session.stopped` | `closure_kind` | `closure_kind:checkpointed&#124;bootstrap_abort` |
| `default` | `session.stopped` | `process_closed` | `process_closed:boolean` |
| `default` | `session.stopped` | `store_closed` | `store_closed:boolean` |
| `default` | `session.resumed` | `checkpoint_id` | `checkpoint_id:digest` |
| `default` | `session.resumed` | `execution_profile` | `execution_profile:standard&#124;yolo` |
| `default` | `session.resumed` | `profile_source_event_id` | `profile_source_event_id:digest&#124;null` |
| `default` | `session.resumed` | `terminal_backend` | `terminal_backend:tmux&#124;conpty` |
| `default` | `session.resumed` | `native_session_id` | `native_session_id:string[1..512]` |
| `default` | `session.bootstrap_aborted` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `session.bootstrap_aborted` | `failure_phase` | `failure_phase:before_terminal&#124;after_terminal&#124;after_process&#124;after_identity&#124;before_checkpoint` |
| `default` | `session.bootstrap_aborted` | `provider_identity_record_id` | `provider_identity_record_id:digest&#124;null` |
| `default` | `session.bootstrap_aborted` | `manager_session_ref` | `manager_session_ref:string[1..512]&#124;null` |
| `default` | `session.bootstrap_aborted` | `process_closed` | `process_closed:boolean` |
| `default` | `session.bootstrap_aborted` | `store_closed` | `store_closed:boolean` |
| `default` | `session.bootstrap_aborted` | `resume_allowed` | `resume_allowed:false` |
| `default` | `lease.transferred` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `lease.transferred` | `from_host_id` | `from_host_id:UUIDv7` |
| `default` | `lease.transferred` | `to_host_id` | `to_host_id:UUIDv7` |
| `default` | `lease.transferred` | `predecessor_lease_id` | `predecessor_lease_id:UUIDv4` |
| `default` | `lease.transferred` | `new_lease_id` | `new_lease_id:UUIDv4` |
| `default` | `lease.forced` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `lease.forced` | `expected_owner_host_id` | `expected_owner_host_id:UUIDv7` |
| `default` | `lease.forced` | `expected_epoch` | `expected_epoch:uint53` |
| `default` | `lease.forced` | `new_lease_id` | `new_lease_id:UUIDv4` |
| `default` | `lease.forced` | `checkpoint_id` | `checkpoint_id:digest` |
| `default` | `session.parked` | `reason` | `reason:remote_owner&#124;stale_owner&#124;restore_policy&#124;failed_handoff` |
| `default` | `session.parked` | `winning_lease_id` | `winning_lease_id:UUIDv4` |
| `default` | `session.failed` | `error_code` | `error_code:string[1..128]` |
| `default` | `session.failed` | `retryable` | `retryable:boolean` |
| `default` | `session.failed` | `operation_id` | `operation_id:UUIDv7&#124;null` |
| `default` | `fork.created` | `source_session_id` | `source_session_id:UUIDv7` |
| `default` | `fork.created` | `source_checkpoint_id` | `source_checkpoint_id:digest` |
| `default` | `fork.created` | `new_session_record_id` | `new_session_record_id:digest` |
| `default` | `fork.created` | `provider_fork_mode` | `provider_fork_mode:native&#124;supported_import&#124;task_board_clone` |
| `default` | `fork.created` | `execution_profile` | `execution_profile:standard&#124;yolo` |
| `default` | `fork.created` | `profile_source_event_id` | `profile_source_event_id:digest&#124;null` |
| `default` | `fork.created` | `source_profile_event_id` | `source_profile_event_id:digest&#124;null` |
| `default` | `profile.changed` | `from` | `from:standard&#124;yolo` |
| `default` | `profile.changed` | `to` | `to:standard&#124;yolo` |
| `default` | `profile.changed` | `confirmed` | `confirmed:boolean` |
| `default` | `session.tombstoned` | `tombstone_id` | `tombstone_id:digest` |
| `default` | `takeover.force_confirmed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `takeover.force_confirmed` | `expected_owner_host_id` | `expected_owner_host_id:UUIDv7` |
| `default` | `takeover.force_confirmed` | `expected_epoch` | `expected_epoch:uint53` |
| `default` | `takeover.force_confirmed` | `checkpoint_id` | `checkpoint_id:digest` |
| `default` | `takeover.force_confirmed` | `accepted_risks` | `accepted_risks:sorted unique enum[3]` |
| `default` | `takeover.force_confirmed` | `confirmation_mode` | `confirmation_mode:interactive&#124;non_interactive` |
| `default` | `replica.replace_confirmed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `replica.replace_confirmed` | `workspace_group_id` | `workspace_group_id:UUIDv7` |
| `default` | `replica.replace_confirmed` | `target_host_id` | `target_host_id:UUIDv7` |
| `default` | `replica.replace_confirmed` | `managed_replica_id` | `managed_replica_id:UUIDv7` |
| `default` | `replica.replace_confirmed` | `expected_marker_id` | `expected_marker_id:digest` |
| `default` | `replica.replace_confirmed` | `expected_checkpoint_id` | `expected_checkpoint_id:digest` |
| `default` | `replica.replace_confirmed` | `replacement_checkpoint_id` | `replacement_checkpoint_id:digest` |
| `default` | `replica.replace_confirmed` | `confirmation_mode` | `confirmation_mode:interactive&#124;non_interactive` |
| `default` | `task_board.launched` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `task_board.launched` | `manager_session_ref` | `manager_session_ref:string[1..512]` |
| `default` | `task_board.launched` | `provider_id` | `provider_id:provider-id` |
| `default` | `task_board.launched` | `launch_mode` | `launch_mode:primary_owner&#124;tracked_prompt` |
| `default` | `task_board.launched` | `lease_epoch` | `lease_epoch:uint53>0` |
| `default` | `task_board.launched` | `lease_id` | `lease_id:UUIDv4` |
| `default` | `task_board.launched` | `execution_profile` | `execution_profile:standard&#124;yolo` |
| `default` | `task_board.launched` | `profile_source_event_id` | `profile_source_event_id:digest&#124;null` |
| `default` | `task_board.launched` | `board_goal_id` | `board_goal_id:string[1..128]&#124;null` |
| `default` | `task_board.launched` | `board_goal_revision` | `board_goal_revision:uint53&#124;null` |
| `default` | `task_board.launched` | `state` | `state:running&#124;idle` |
| `default` | `task_board.adopted` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `task_board.adopted` | `bundle_id` | `bundle_id:digest` |
| `default` | `task_board.adopted` | `manager_session_ref` | `manager_session_ref:string[1..512]` |
| `default` | `task_board.adopted` | `board_goal_id` | `board_goal_id:string[1..128]&#124;null` |
| `default` | `task_board.adopted` | `board_goal_revision` | `board_goal_revision:uint53&#124;null` |
| `default` | `tombstone.issued` | `tombstone_id` | `tombstone_id:digest` |
| `default` | `tombstone.issued` | `scope` | `scope:session&#124;workspace_entry&#124;provider_snapshot&#124;managed_replica` |
| `default` | `tombstone.issued` | `subject_id` | `subject_id:UUIDv7` |
| `default` | `tombstone.issued` | `target_ref` | `target_ref:string[1..1024]` |
| `default` | `tombstone.resolved` | `tombstone_id` | `tombstone_id:digest` |
| `default` | `tombstone.resolved` | `resolution` | `resolution:deleted&#124;already_absent&#124;resurrected&#124;retained_conflict` |
| `default` | `tombstone.resolved` | `target_ref` | `target_ref:string[1..1024]` |
| `default` | `tombstone.resolved` | `resulting_entry_digest` | `resulting_entry_digest:digest&#124;null` |
| `default` | `clone.planned` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.planned` | `bundle_manifest_id` | `bundle_manifest_id:digest` |
| `default` | `clone.planned` | `projection_plan_id` | `projection_plan_id:digest` |
| `default` | `clone.planned` | `migration_checkpoint_id` | `migration_checkpoint_id:digest` |
| `default` | `clone.planned` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.planned` | `target_environment` | `target_environment:EnvironmentTuple` |
| `default` | `clone.planned` | `expected_target_native_session_id` | `expected_target_native_session_id:string[1..512]` |
| `default` | `clone.target_prepared` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.target_prepared` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.target_prepared` | `plan_id` | `plan_id:digest` |
| `default` | `clone.target_prepared` | `provider_transaction_id` | `provider_transaction_id:UUIDv7` |
| `default` | `clone.target_prepared` | `provider_prepared_result_digest` | `provider_prepared_result_digest:digest` |
| `default` | `clone.target_prepared` | `staged_read_back_evidence_manifest_id` | `staged_read_back_evidence_manifest_id:digest` |
| `default` | `clone.target_prepared` | `rollback_retained` | `rollback_retained=true` |
| `default` | `clone.target_published` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.target_published` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.target_published` | `provider_identity_record_id` | `provider_identity_record_id:digest` |
| `default` | `clone.target_published` | `target_provider_manifest_id` | `target_provider_manifest_id:digest` |
| `default` | `clone.target_published` | `live_read_back_evidence_manifest_id` | `live_read_back_evidence_manifest_id:digest` |
| `default` | `clone.target_published` | `fidelity_report_id` | `fidelity_report_id:digest` |
| `default` | `clone.target_published` | `validation_report_id` | `validation_report_id:digest` |
| `default` | `clone.target_published` | `source_generation_revalidated` | `source_generation_revalidated=true` |
| `default` | `clone.target_published` | `rollback_retained` | `rollback_retained=true` |
| `default` | `clone.target_validation_failed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.target_validation_failed` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.target_validation_failed` | `phase` | `phase:prepublication_source_recheck&#124;provider_prepare&#124;postpublication_source_recheck&#124;live_discovery&#124;live_read_back&#124;resume_plan` |
| `default` | `clone.target_validation_failed` | `error_code` | `error_code:string[1..128]` |
| `default` | `clone.target_validation_failed` | `validation_report_id` | `validation_report_id:digest&#124;null` |
| `default` | `clone.target_validation_failed` | `rollback_required` | `rollback_required:boolean` |
| `default` | `clone.target_validation_failed` | `transaction_unknown` | `transaction_unknown:boolean` |
| `default` | `clone.rolled_back` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.rolled_back` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.rolled_back` | `provider_rolled_back_result_digest` | `provider_rolled_back_result_digest:digest` |
| `default` | `clone.rolled_back` | `retained_bundle_manifest_id` | `retained_bundle_manifest_id:digest` |
| `default` | `clone.rolled_back` | `reason_code` | `reason_code:string[1..128]` |
| `default` | `clone.committed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.committed` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.committed` | `provider_identity_record_id` | `provider_identity_record_id:digest` |
| `default` | `clone.committed` | `provider_committed_result_digest` | `provider_committed_result_digest:digest` |
| `default` | `clone.committed` | `target_checkpoint_id` | `target_checkpoint_id:digest` |
| `default` | `clone.committed` | `fidelity_report_id` | `fidelity_report_id:digest` |
| `default` | `clone.committed` | `validation_report_id` | `validation_report_id:digest` |
| `default` | `clone.committed` | `native_resumable` | `native_resumable=true` |
| `default` | `clone.lineage_published` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.lineage_published` | `target_checkpoint_id` | `target_checkpoint_id:digest` |
| `default` | `clone.lineage_published` | `lineage_receipt_id` | `lineage_receipt_id:digest` |
| `default` | `clone.lineage_published` | `bundle_manifest_id` | `bundle_manifest_id:digest` |
| `default` | `clone.failed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `clone.failed` | `phase` | `phase=checkpoint` |
| `default` | `clone.failed` | `error_code` | `error_code=target_checkpoint_failed` |
| `default` | `clone.failed` | `retryable` | `retryable=true` |
| `default` | `clone.failed` | `retained_bundle_manifest_id` | `retained_bundle_manifest_id:digest` |
| `default` | `clone.failed` | `materialization_id` | `materialization_id:UUIDv7` |
| `default` | `clone.failed` | `transaction_unknown` | `transaction_unknown=false` |
| `default` | `adoption.planned` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `adoption.planned` | `plan_id` | `plan_id:digest` |
| `default` | `adoption.planned` | `source_instance_id` | `source_instance_id:digest` |
| `default` | `adoption.planned` | `source_observation_id` | `source_observation_id:digest` |
| `default` | `adoption.planned` | `source_head_digest` | `source_head_digest:digest` |
| `default` | `adoption.committed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `adoption.committed` | `provider_identity_record_id` | `provider_identity_record_id:digest` |
| `default` | `adoption.committed` | `initial_checkpoint_id` | `initial_checkpoint_id:digest` |
| `default` | `adoption.committed` | `native_resumable` | `native_resumable=true` |
| `default` | `move.planned` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `move.planned` | `plan_id` | `plan_id:digest` |
| `default` | `move.planned` | `source_session_id` | `source_session_id:UUIDv7` |
| `default` | `move.planned` | `target_session_id` | `target_session_id:UUIDv7` |
| `default` | `move.target_committed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `move.target_committed` | `target_session_id` | `target_session_id:UUIDv7` |
| `default` | `move.target_committed` | `target_checkpoint_id` | `target_checkpoint_id:digest` |
| `default` | `move.target_committed` | `clone_lineage_receipt_id` | `clone_lineage_receipt_id:digest` |
| `default` | `move.source_release_requested` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `move.source_release_requested` | `target_committed_event_id` | `target_committed_event_id:digest` |
| `default` | `move.source_release_requested` | `source_lease_epoch` | `source_lease_epoch:uint53>0` |
| `default` | `move.source_release_requested` | `source_lease_id` | `source_lease_id:UUIDv4` |
| `default` | `move.source_released` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `move.source_released` | `target_session_id` | `target_session_id:UUIDv7` |
| `default` | `move.source_released` | `source_stop_event_id` | `source_stop_event_id:digest` |
| `default` | `move.source_released` | `source_release_receipt_id` | `source_release_receipt_id:digest` |
| `default` | `move.source_released` | `outcome` | `outcome=moved_cross_environment` |
| `default` | `move.source_release_failed` | `operation_id` | `operation_id:UUIDv7` |
| `default` | `move.source_release_failed` | `target_session_id` | `target_session_id:UUIDv7` |
| `default` | `move.source_release_failed` | `error_code` | `error_code:string[1..128]` |
| `default` | `move.source_release_failed` | `source_still_resumable` | `source_still_resumable:boolean` |
| `default` | `move.source_release_failed` | `outcome` | `outcome=cloned_source_still_active` |
| `4.0.0-override` | `terminal.created` | `terminal_binding_id` | `terminal_binding_id:digest` |
| `4.0.0-override` | `terminal.created` | `terminal_backend_id` | `terminal_backend_id:terminal-backend-id` |
| `4.0.0-override` | `terminal.created` | `implementation_version` | `implementation_version:semver` |
| `4.0.0-override` | `terminal.created` | `protocol_version` | `protocol_version:semver` |
| `4.0.0-override` | `terminal.created` | `evidence_ids` | `evidence_ids:sorted unique digest[1..256]` |
| `4.0.0-override` | `session.resumed` | `checkpoint_id` | `checkpoint_id:digest` |
| `4.0.0-override` | `session.resumed` | `execution_profile` | `execution_profile:standard&#124;yolo` |
| `4.0.0-override` | `session.resumed` | `profile_source_event_id` | `profile_source_event_id:digest&#124;null` |
| `4.0.0-override` | `session.resumed` | `terminal_binding_id` | `terminal_binding_id:digest` |
| `4.0.0-override` | `session.resumed` | `terminal_backend_id` | `terminal_backend_id:terminal-backend-id` |
| `4.0.0-override` | `session.resumed` | `implementation_version` | `implementation_version:semver` |
| `4.0.0-override` | `session.resumed` | `protocol_version` | `protocol_version:semver` |
| `4.0.0-override` | `session.resumed` | `evidence_ids` | `evidence_ids:sorted unique digest[1..256]` |
