package catalog_test

import (
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/specpin"
)

func TestCurrentMatchesReviewedV050Catalog(t *testing.T) {
	t.Parallel()

	got := catalog.Current()
	if got.Release != catalog.ReleaseV050 {
		t.Fatalf("Current().Release = %q, want %q", got.Release, catalog.ReleaseV050)
	}
	if got.Source.Repository != specpin.Repository ||
		got.Source.Commit != specpin.CommitV050 ||
		got.Source.DocumentSHA256 != specpin.DocumentSHA256 {
		t.Fatalf("Current().Source = %#v, want exact spec pin", got.Source)
	}
	wantScope := []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10",
		"11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
		"appendix-a", "appendix-b", "appendix-c", "appendix-d",
	}
	if !reflect.DeepEqual(got.Source.NormativeScope, wantScope) {
		t.Fatalf("Current().Source.NormativeScope = %v, want %v", got.Source.NormativeScope, wantScope)
	}

	manifest, err := specpin.Current()
	if err != nil {
		t.Fatalf("specpin.Current() error = %v", err)
	}
	wantContracts, err := manifest.ContractsForRelease(specpin.ReleaseV050)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.5.0) error = %v", err)
	}
	if !reflect.DeepEqual(contractPins(got.Contracts), wantContracts) {
		t.Fatal("generated v0.5.0 contracts differ from the verified source pin")
	}

	if len(got.Operations) != 99 {
		t.Errorf("operations = %d, want 99", len(got.Operations))
	}
	if len(got.Capabilities) != 46 {
		t.Errorf("capabilities = %d, want 46", len(got.Capabilities))
	}
	if len(got.Events) != 112 {
		t.Errorf("events = %d, want 112", len(got.Events))
	}
	if len(got.Errors) != 109 {
		t.Errorf("errors = %d, want 109", len(got.Errors))
	}
	if len(got.SelfIdentities) != 40 {
		t.Errorf("self identity contracts = %d, want 40", len(got.SelfIdentities))
	}
	identityRows := 0
	for _, identity := range got.SelfIdentities {
		identityRows += len(identity.ContractVersions)
	}
	if identityRows != 46 {
		t.Errorf("self identity schema/version rows = %d, want 46", identityRows)
	}
	var foundManagedReplicaMarker bool
	for _, identity := range got.SelfIdentities {
		if identity.ContractID == "urn:ax:schema:materialization-journal" {
			foundManagedReplicaMarker = reflect.DeepEqual(identity.ContractVersions, []string{"2.0.0"}) &&
				identity.SelfField == "marker_id" &&
				identity.DiscriminatorName == "document_kind" &&
				identity.DiscriminatorValue == "managed_replica_marker"
		}
	}
	if !foundManagedReplicaMarker {
		t.Error("v0.5.0 catalog lost the materialization-journal 2.0.0 managed-replica marker identity while merging duplicate contract IDs")
	}

	wantAdditionalIdentities := map[catalog.ContractID]string{
		"urn:ax:schema:terminal-backend-probe":            "probe_id",
		"urn:ax:schema:terminal-instance-binding":         "binding_id",
		"urn:ax:schema:terminal-capability-evidence":      "evidence_id",
		"urn:ax:schema:clone-raw-object-manifest":         "raw_object_manifest_id",
		"urn:ax:schema:clone-capture-manifest":            "capture_manifest_id",
		"urn:ax:schema:canonical-session":                 "canonical_session_id",
		"urn:ax:schema:fidelity-report":                   "fidelity_report_id",
		"urn:ax:schema:projection-plan":                   "projection_plan_id",
		"urn:ax:schema:clone-projected-object-manifest":   "projected_object_manifest_id",
		"urn:ax:schema:clone-read-back-evidence-manifest": "read_back_evidence_manifest_id",
		"urn:ax:schema:clone-validation-report":           "validation_report_id",
		"urn:ax:schema:migration-checkpoint":              "migration_checkpoint_id",
		"urn:ax:schema:clone-lineage-receipt":             "lineage_receipt_id",
		"urn:ax:schema:supported-environment-tuples":      "registry_digest",
	}
	for _, identity := range got.SelfIdentities {
		if want, ok := wantAdditionalIdentities[identity.ContractID]; ok {
			if identity.SelfField != want || !reflect.DeepEqual(identity.ContractVersions, []string{"1.0.0"}) {
				t.Errorf("self identity %s = (%q, %v), want (%q, [1.0.0])", identity.ContractID, identity.SelfField, identity.ContractVersions, want)
			}
			delete(wantAdditionalIdentities, identity.ContractID)
		}
	}
	if len(wantAdditionalIdentities) != 0 {
		t.Errorf("generated self identity catalog omitted reviewed contracts: %v", wantAdditionalIdentities)
	}

	assertFamilyNames(t, operationNames(got), map[string][]string{
		"terminal_backend": {
			"manifest", "probe", "create", "attach", "status", "quiesce-input",
			"wait-safe-boundary", "request-stop", "terminate-stale", "restore",
		},
		"provider": {
			"manifest", "probe", "launch", "identify-session", "quiesce",
			"native-store-plan", "capture", "materialize", "materialize-status",
			"materialize-commit", "materialize-rollback", "resume", "fork", "stop", "doctor",
		},
		"session_adapter": {
			"manifest", "probe", "discover", "inspect", "snapshot-proof", "capture-plan",
			"capture", "normalize", "projection-plan", "project", "read-back", "validate",
			"resume-plan", "doctor",
		},
		"directory_node": {
			"manifest", "probe", "scan", "inventory", "preview", "enrichment-plan",
			"enrichment-run", "enrichment-status", "continuation-inspect", "runtime-observe", "doctor",
		},
		"directory_query": {
			"schema", "sessions", "session", "lineage", "hosts", "environments", "jobs",
			"plans", "count", "distinct", "directory_summary", "set_title", "set_tags",
			"set_pin", "enrich", "plan_continue", "execute_plan",
		},
		"task_board_bridge": {"launch", "status", "export", "import", "open", "adopt", "stop", "resume"},
		"mesh_rpc": {
			"hello", "health.get", "inventory.roots", "inventory.children", "objects.get",
			"transfer.begin", "transfer.status", "chunks.put", "transfer.validate", "transfer.commit",
			"materialize.prepare", "materialize.commit", "materialize.status", "materialize.finalize",
			"materialize.rollback", "lease.refresh", "tombstone.ack", "session.status", "session.stop",
			"handoff.prepare", "handoff.quiesce", "handoff.stop", "handoff.commit", "handoff.abort",
		},
	})

	assertFamilyNames(t, capabilityNames(got), map[string][]string{
		"terminal_backend": {
			"durable_disconnect", "local_attach", "remote_attach", "web_attach", "multi_attach",
			"headless_creation", "reboot_restoration", "input_quiescence", "safe_boundary_observation",
			"provider_process_observation", "graceful_stop", "stale_process_termination",
			"terminal_state_retention", "scrollback_retention", "credential_capable_execution_realm",
			"multiple_input_clients",
		},
		"provider": {
			"native_resume", "portable_store", "managed_pty", "appserver", "task_board_primary",
			"prompt_spawn", "native_goal_binding",
		},
		"session_adapter": {
			"native_discovery", "stable_snapshot", "raw_capture", "canonical_read", "canonical_write",
			"native_read_back", "native_resume_plan", "official_import", "same_environment_lossless_clone",
			"tool_history", "usage_history", "compaction_history", "subagent_graph",
			"opaque_reasoning_roundtrip", "workspace_binding",
		},
		"directory_node": {
			"directory_discovery", "directory_incremental_scan", "directory_head_digest",
			"directory_tail_preview", "native_title_read", "native_runtime_observation",
			"existing_session_adoption", "native_resume",
		},
	})

	assertFamilyNames(t, eventNames(got), map[string][]string{
		"session_event": {
			"session.created", "terminal.created", "provider.launched", "provider.identified",
			"session.idle", "session.quiescing", "checkpoint.created", "sync.completed",
			"session.stopped", "session.resumed", "session.bootstrap_aborted", "lease.transferred",
			"lease.forced", "session.parked", "session.failed", "fork.created", "profile.changed",
			"session.tombstoned", "takeover.force_confirmed", "replica.replace_confirmed",
			"task_board.launched", "task_board.adopted", "tombstone.issued", "tombstone.resolved",
			"clone.planned", "clone.target_prepared", "clone.target_published",
			"clone.target_validation_failed", "clone.rolled_back", "clone.committed",
			"clone.lineage_published", "clone.failed", "adoption.planned", "adoption.committed",
			"move.planned", "move.target_committed", "move.source_release_requested",
			"move.source_released", "move.source_release_failed",
		},
		"observation_event": {
			"service.started", "service.stopped", "service.health", "rpc.connected", "rpc.rejected",
			"rpc.disconnected", "sync.inventory", "sync.transfer", "sync.validated", "sync.partial",
			"materialization.conflict", "materialization.prepared", "materialization.committed",
			"materialization.rolled_back", "lease.observed", "lease.changed", "lease.stale_process",
			"lease.concurrent_force", "provider.probed", "provider.quiesced", "provider.captured",
			"provider.stopped", "provider.failed", "task_board.launched", "task_board.exported",
			"task_board.imported", "task_board.opened", "task_board.adopted", "clone.started",
			"source.resolved", "source.snapshot_established", "source.captured", "canonical.normalized",
			"projection.planned", "projection.policy_rejected", "target.prepared",
			"target.staged_validated", "target.published", "target.live_validated", "target.committed",
			"target.rolled_back", "lineage.published", "target.opened", "clone.failed",
			"directory.scan.started", "directory.scan.completed", "directory.scan.failed",
			"directory.observation.published", "directory.observation.conflict",
			"directory.lineage.linked", "directory.lineage.ambiguous", "directory.enrichment.queued",
			"directory.enrichment.started", "directory.enrichment.published",
			"directory.enrichment.stale", "directory.enrichment.failed", "directory.plan.created",
			"directory.plan.rejected", "directory.plan.expired", "directory.operation.validating",
			"directory.operation.executing", "directory.operation.finalizing",
			"directory.operation.succeeded", "directory.operation.failed",
			"directory.operation.uncertain", "directory.target.launched", "directory.target.ready",
			"directory.attach.started", "directory.attach.ended", "directory.mesh.converged",
			"directory.mesh.gap_detected", "takeover.phase", "fork.phase",
		},
	})

	assertErrorMappings(t, got, map[int][]string{
		2:   {"interactive_choice_required", "invalid_arguments", "directory_instance_not_found", "directory_instance_ambiguous", "query_invalid"},
		3:   {"invalid_config", "idempotency_mismatch", "local_precondition_failed", "inventory_stale"},
		4:   {"name_ambiguous", "not_found", "source_not_found", "source_ambiguous", "host_offline", "terminal_backend_not_found", "terminal_backend_ambiguous"},
		5:   {"workspace_conflict", "native_store_conflict"},
		6:   {"capability_unavailable", "profile_mapping_unavailable", "incompatible_protocol", "incompatible_schema", "provider_fork_unsupported", "unsupported_backend_identity", "unsupported_environment_tuple", "operation_unknown", "target_resume_invalid", "directory_mesh_unsupported", "continuation_route_unavailable", "adoption_unavailable", "terminal_attach_unavailable", "terminal_backend_unavailable", "terminal_backend_protocol_incompatible", "terminal_backend_capability_unproven", "terminal_backend_restore_mismatch"},
		7:   {"authentication_failed", "peer_not_allowlisted", "host_identity_mismatch", "field_forbidden", "target_auth_missing", "terminal_backend_untrusted", "terminal_backend_unauthorized"},
		8:   {"owner_unreachable", "transport_failure"},
		9:   {"unsafe_path", "integrity_failure", "source_corrupt", "bundle_integrity_failed", "credential_material_detected", "observation_gap", "observation_conflict", "adapter_protocol_violation", "terminal_backend_integrity_failure", "terminal_backend_implementation_drift", "terminal_backend_manifest_probe_mismatch"},
		10:  {"not_owner", "stale_owner", "lease_conflict", "invalid_state_transition", "lineage_ambiguous", "annotation_conflict", "terminal_backend_stale_generation"},
		11:  {"quiesce_timeout", "stop_timeout", "workspace_group_busy", "workspace_group_changed", "source_not_quiescent", "source_changed_during_clone"},
		12:  {"staging_incomplete", "atomic_commit_unavailable", "materialization_failed", "rollback_failed", "target_prepare_failed", "target_validation_failed", "target_checkpoint_failed", "transaction_unknown", "operation_uncertain"},
		13:  {"provider_timeout", "provider_protocol_error", "provider_process_failed", "session_adapter_protocol_error", "session_adapter_process_failed", "session_adapter_timeout", "enrichment_model_unavailable", "terminal_backend_protocol_error", "terminal_backend_process_failed", "terminal_backend_timeout"},
		14:  {"task_board_bridge_unavailable", "task_board_bundle_invalid", "task_board_validate_failed", "checkpoint_unavailable"},
		15:  {"partial_sync", "cloned_source_still_active"},
		16:  {"confirmation_required", "policy_refused", "secret_policy_violation", "projection_loss_unacceptable", "unsafe_pending_action", "instance_identity_weak", "preview_policy_blocked", "enrichment_policy_blocked", "enrichment_head_changed", "continuation_plan_stale", "unmanaged_remote_forbidden", "workspace_route_conflict", "cloning_fidelity_unacceptable", "operation_in_progress"},
		17:  {"migration_required"},
		130: {"interrupted"},
	})

	assertOperationEffectNames(t, got, catalog.EffectDurableMutation, map[string][]string{
		"terminal_backend":  {"create", "attach", "quiesce-input", "wait-safe-boundary", "request-stop", "terminate-stale", "restore"},
		"provider":          {"materialize", "materialize-commit", "materialize-rollback"},
		"directory_node":    {"scan", "enrichment-run"},
		"directory_query":   {"set_title", "set_tags", "set_pin", "enrich", "execute_plan"},
		"task_board_bridge": {"launch", "export", "import", "open", "adopt", "stop", "resume"},
		"mesh_rpc": {
			"transfer.begin", "chunks.put", "transfer.validate", "transfer.commit",
			"materialize.prepare", "materialize.commit", "materialize.finalize",
			"materialize.rollback", "tombstone.ack", "session.stop", "handoff.prepare",
			"handoff.quiesce", "handoff.stop", "handoff.commit", "handoff.abort",
		},
	})
	assertOperationEffectNames(t, got, catalog.EffectIsolatedOutput, map[string][]string{
		"provider":        {"capture"},
		"session_adapter": {"capture-plan", "capture", "normalize", "projection-plan", "project", "read-back"},
	})

	for _, operation := range got.Operations {
		switch operation.Effect {
		case catalog.EffectDurableMutation:
			if operation.IdempotencyKey == "" || len(operation.RecoveryEvidence) == 0 {
				t.Errorf("durable operation %s/%s lacks idempotency or recovery evidence", operation.Family, operation.Name)
			}
		case catalog.EffectNoDurableMutation, catalog.EffectIsolatedOutput:
			if operation.IdempotencyKey != "" || len(operation.RecoveryEvidence) != 0 {
				t.Errorf("non-durable operation %s/%s carries mutation evidence", operation.Family, operation.Name)
			}
		default:
			t.Errorf("operation %s/%s has unknown effect %q", operation.Family, operation.Name, operation.Effect)
		}
	}
}

func TestV043CompatibilityProjection(t *testing.T) {
	t.Parallel()

	got, err := catalog.ForRelease(catalog.ReleaseV043)
	if err != nil {
		t.Fatalf("ForRelease(v0.4.3) error = %v", err)
	}
	manifest, err := specpin.Current()
	if err != nil {
		t.Fatalf("specpin.Current() error = %v", err)
	}
	wantContracts, err := manifest.ContractsForRelease(specpin.ReleaseV043)
	if err != nil {
		t.Fatalf("ContractsForRelease(v0.4.3) error = %v", err)
	}
	if !reflect.DeepEqual(contractPins(got.Contracts), wantContracts) {
		t.Fatal("generated v0.4.3 contracts differ from the verified compatibility projection")
	}
	if len(got.Operations) != 89 {
		t.Errorf("v0.4.3 operations = %d, want 89", len(got.Operations))
	}
	if len(got.Capabilities) != 30 {
		t.Errorf("v0.4.3 capabilities = %d, want 30", len(got.Capabilities))
	}
	if len(got.Errors) != 94 {
		t.Errorf("v0.4.3 errors = %d, want 94", len(got.Errors))
	}
	for _, operation := range got.Operations {
		if operation.Family == "terminal_backend" {
			t.Errorf("v0.4.3 retained TerminalBackend operation %q", operation.Name)
		}
	}
	for _, capability := range got.Capabilities {
		if capability.Family == "terminal_backend" {
			t.Errorf("v0.4.3 retained TerminalBackend capability %q", capability.Name)
		}
	}
	for _, item := range got.Errors {
		if strings.HasPrefix(string(item.Code), "terminal_backend_") {
			t.Errorf("v0.4.3 retained Error 1.3 code %q", item.Code)
		}
	}
	for _, event := range got.Events {
		for _, version := range event.ContractVersions {
			if event.ContractID == "urn:ax:schema:session-event" && version == "4.0.0" {
				t.Errorf("v0.4.3 event %q retained Session Event 4.0.0", event.Name)
			}
		}
	}
}

func TestTerminalBackendEvidenceMatchesPinnedContract(t *testing.T) {
	t.Parallel()

	type evidence struct {
		idempotencyKey   string
		recoveryEvidence []string
	}
	want := map[catalog.OperationName]evidence{
		"create": {
			idempotencyKey: "session_id + \"/\" + bootstrap_operation_id",
			recoveryEvidence: []string{
				"persist receipt before wrapper_started",
				"identical retry replays receipt",
				"uncertainty requires status",
				"binding plus capability/credential evidence",
			},
		},
		"attach": {
			idempotencyKey: "terminal_instance_id + \"/\" + client_id",
			recoveryEvidence: []string{
				"durable client receipt determines whether timeout created authorized input",
				"retry same client_id",
				"attach-policy/capability evidence",
			},
		},
		"quiesce-input": {
			idempotencyKey: "terminal_instance_id + \"/quiesce/\" + quiescence_generation",
			recoveryEvidence: []string{
				"receipt precedes input closure",
				"identical retry replays receipt",
				"uncertainty requires status",
				"evidence binds closure time/generation",
			},
		},
		"wait-safe-boundary": {
			idempotencyKey: "terminal_instance_id + \"/boundary/\" + quiescence_generation + \"/\" + provider_proof_kind",
			recoveryEvidence: []string{
				"lesser of request deadline and timeout",
				"identical retry returns same proof",
				"no proof on timeout",
				"safe-boundary evidence",
			},
		},
		"request-stop": {
			idempotencyKey: "terminal_instance_id + \"/stop/\" + safe_boundary_evidence_id",
			recoveryEvidence: []string{
				"lesser of request deadline and graceful timeout",
				"lost result requires status",
				"same-key retry only when not closed",
				"boundary and closure evidence",
			},
		},
		"terminate-stale": {
			idempotencyKey: "terminal_instance_id + \"/terminate/\" + stale_lease_id + \"/\" + decimal(stale_epoch)",
			recoveryEvidence: []string{
				"deadline may leave uncertain closure",
				"status before same-key retry",
				"diagnostic, fencing, and closure evidence",
			},
		},
		"restore": {
			idempotencyKey: "session_id + \"/\" + bootstrap_operation_id",
			recoveryEvidence: []string{
				"persist receipt before restore",
				"identical retry replays receipt",
				"uncertainty requires status",
				"prior binding, checkpoint, new binding, and capability/credential evidence",
			},
		},
	}

	for _, operation := range catalog.Current().Operations {
		if operation.Family != "terminal_backend" || operation.Effect != catalog.EffectDurableMutation {
			continue
		}
		expected, ok := want[operation.Name]
		if !ok {
			t.Errorf("unexpected durable Terminal Backend operation %q", operation.Name)
			continue
		}
		if operation.IdempotencyKey != expected.idempotencyKey || !reflect.DeepEqual(operation.RecoveryEvidence, expected.recoveryEvidence) {
			t.Errorf("Terminal Backend %s evidence = (%q, %#v), want (%q, %#v)", operation.Name, operation.IdempotencyKey, operation.RecoveryEvidence, expected.idempotencyKey, expected.recoveryEvidence)
		}
		if operation.NormativeSection != "4.B-4.C" || !reflect.DeepEqual(operation.FixtureFamilies, []string{"Appendix D: Terminal Backend protocol"}) {
			t.Errorf("Terminal Backend %s traceability = (%q, %#v), want Section 4.B-4.C / Appendix D protocol anchor", operation.Name, operation.NormativeSection, operation.FixtureFamilies)
		}
		delete(want, operation.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing durable Terminal Backend evidence for %#v", want)
	}
}

func TestOtherDurableOperationEvidenceMatchesReviewedMetadata(t *testing.T) {
	t.Parallel()

	type evidenceGroup struct {
		family           catalog.Family
		names            []catalog.OperationName
		idempotencyKey   string
		recoveryEvidence []string
	}
	groups := []evidenceGroup{
		{
			family:         "provider",
			names:          []catalog.OperationName{"materialize", "materialize-commit", "materialize-rollback"},
			idempotencyKey: "(operation, operation_id)",
			recoveryEvidence: []string{
				"ProviderTransactionDocument",
				"materialize-status reconciliation before retry",
				"canonical-identical request replay",
				"PTX-IDEMPOTENCY-ID-*",
			},
		},
		{
			family:         "directory_node",
			names:          []catalog.OperationName{"scan", "enrichment-run"},
			idempotencyKey: "(operation, operation_id)",
			recoveryEvidence: []string{
				"prior durable result",
				"canonical-identical request replay",
				"changed body refuses with idempotency_mismatch without new records",
			},
		},
		{
			family:         "directory_query",
			names:          []catalog.OperationName{"set_title", "set_tags", "set_pin"},
			idempotencyKey: "QueryOperation.idempotency_key",
			recoveryEvidence: []string{
				"QueryOperation.expectation_digest",
				"SessionAnnotation identity and supersession heads",
				"confirmed retry revalidates the expected annotation head",
			},
		},
		{
			family:         "directory_query",
			names:          []catalog.OperationName{"enrich"},
			idempotencyKey: "EnrichmentJobRequest.idempotency_key over subject, expected head, profile, requested kinds, and prior-summary basis",
			recoveryEvidence: []string{
				"EnrichmentJobRequest.job_request_id",
				"EnrichmentJobReceipt predecessor chain",
				"different request digest refuses with idempotency_mismatch",
			},
		},
		{
			family:         "directory_query",
			names:          []catalog.OperationName{"execute_plan"},
			idempotencyKey: "(operation_id, QueryOperation.idempotency_key)",
			recoveryEvidence: []string{
				"DirectoryOperationReceipt predecessor chain",
				"lost response recovered by operation_id before retry",
				"exact plan and expectations revalidated before every mutation",
			},
		},
		{
			family:         "task_board_bridge",
			names:          []catalog.OperationName{"launch", "export", "import", "open", "adopt", "stop", "resume"},
			idempotencyKey: "(operation, operation_id)",
			recoveryEvidence: []string{
				"recorded durable result",
				"canonical-identical retry returns recorded result after token consumption",
				"changed arguments refuse with idempotency_mismatch",
				"status is the only lost-adopt recovery read",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"transfer.begin"},
			idempotencyKey: "(manifest_id, transfer_id)",
			recoveryEvidence: []string{
				"retained staging VerifiedState",
				"same manifest and transfer_id retry is idempotent",
				"sender transmits only missing pieces",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"chunks.put"},
			idempotencyKey: "(transfer_id, descriptor.chunk_id)",
			recoveryEvidence: []string{
				"verified chunk digest identity",
				"existing identical digest is idempotent",
				"same digest with different bytes quarantines and aborts sync",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"transfer.validate"},
			idempotencyKey: "(transfer_id, manifest_id)",
			recoveryEvidence: []string{
				"transfer.status reconciliation",
				"complete manifest validation summary digest",
				"identical retry reuses verified staging",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"transfer.commit"},
			idempotencyKey: "(transfer_id, manifest_id, validation_summary_digest)",
			recoveryEvidence: []string{
				"transfer.status reconciliation",
				"commit_marker_id",
				"atomic immutable-object installation",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"materialize.prepare"},
			idempotencyKey: "(materialize.prepare, operation_id)",
			recoveryEvidence: []string{
				"Materialization Journal binds operation_id, materialization_id, and canonical prepare digest before mutation",
				"identical retry returns byte-identical recorded success or failure",
				"materialize.status reconciles lost response or receiver restart",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"materialize.commit", "materialize.finalize", "materialize.rollback"},
			idempotencyKey: "(operation, materialization_id)",
			recoveryEvidence: []string{
				"Materialization Journal",
				"materialize.status before retry",
				"canonical-identical phase request",
			},
		},
		{
			family:         "mesh_rpc",
			names:          []catalog.OperationName{"tombstone.ack"},
			idempotencyKey: "ack_id",
			recoveryEvidence: []string{
				"immutable Tombstone Acknowledgement identity",
				"objects.get reconciliation",
			},
		},
		{
			family: "mesh_rpc",
			names: []catalog.OperationName{
				"session.stop", "handoff.prepare", "handoff.quiesce", "handoff.stop",
				"handoff.commit", "handoff.abort",
			},
			idempotencyKey: "(operation, operation_id)",
			recoveryEvidence: []string{
				"recorded durable result",
				"session.status before retry",
				"canonical-identical request replay",
				"changed body refuses with idempotency_mismatch",
			},
		},
	}

	type evidence struct {
		idempotencyKey   string
		recoveryEvidence []string
	}
	want := make(map[string]evidence)
	for _, group := range groups {
		for _, name := range group.names {
			key := string(group.family) + "/" + string(name)
			if _, duplicate := want[key]; duplicate {
				t.Fatalf("duplicate reviewed evidence for %s", key)
			}
			want[key] = evidence{group.idempotencyKey, group.recoveryEvidence}
		}
	}

	for _, operation := range catalog.Current().Operations {
		if operation.Effect != catalog.EffectDurableMutation || operation.Family == "terminal_backend" {
			continue
		}
		key := string(operation.Family) + "/" + string(operation.Name)
		expected, ok := want[key]
		if !ok {
			t.Errorf("unexpected durable operation %s", key)
			continue
		}
		if operation.IdempotencyKey != expected.idempotencyKey || !reflect.DeepEqual(operation.RecoveryEvidence, expected.recoveryEvidence) {
			t.Errorf("%s evidence = (%q, %#v), want (%q, %#v)", key, operation.IdempotencyKey, operation.RecoveryEvidence, expected.idempotencyKey, expected.recoveryEvidence)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("catalog omitted reviewed durable-operation evidence for %#v", want)
	}
}

func TestTraceabilityMatchesReviewedSourceAnchors(t *testing.T) {
	t.Parallel()

	type traceability struct {
		section  string
		fixtures []string
	}
	operationFamilies := map[catalog.Family]traceability{
		"terminal_backend":  {"4.B-4.C", []string{"Appendix D: Terminal Backend protocol"}},
		"provider":          {"7.2, 7.5, 7.A", []string{"Appendix D: Provider protocol", "Appendix D: Provider operation body", "PTX-IDEMPOTENCY-ID-*"}},
		"session_adapter":   {"7.8", []string{"Appendix D: Session Adapter protocol"}},
		"directory_node":    {"7.9", []string{"Appendix D: Directory Node protocol/request/response/manifest", "Appendix D: Directory Node operation body", "DIR-INV-01..45"}},
		"directory_query":   {"10.8.5", []string{"Appendix D: Session Directory Query", "Appendix D: Directory Query operation/preset", "DIR-INV-01..45"}},
		"task_board_bridge": {"9.2", []string{"Appendix D: Task-board bridge", "Appendix D: Task-board bridge operation", "TB-LAUNCH-LOST-1", "TB-EXPORT-LOST-RESPONSE"}},
		"mesh_rpc":          {"11.2-11.3, 11.8-11.9", []string{"Appendix D: Mesh RPC", "Appendix D: RPC operation body"}},
	}
	capabilityFamilies := map[catalog.Family]traceability{
		"terminal_backend": {"4.D", []string{"Appendix D: Terminal capability evidence"}},
		"provider":         {"7.3-7.4", []string{"Appendix D: Provider manifest", "Appendix D: Provider probe"}},
		"session_adapter":  {"7.8", []string{"Appendix D: Session Adapter manifest", "Appendix D: Session Adapter probe"}},
		"directory_node":   {"7.9", []string{"Appendix D: Directory Node protocol/request/response/manifest", "DIR-INV-01..45"}},
	}
	eventFamilies := map[catalog.Family]traceability{
		"session_event":     {"5.2, 13.14.5, 13.15", []string{"Appendix D: Session Event", "Appendix D: Session Event payload"}},
		"observation_event": {"18.1", []string{"Appendix D: Observation Event", "Appendix D: Observation result"}},
	}
	errorTraceability := traceability{"15.1-15.3", []string{"Appendix D: Structured Error", "ERR-*"}}

	assertTraceability := func(kind string, family catalog.Family, section string, fixtures []string, want map[catalog.Family]traceability) {
		t.Helper()
		expected, ok := want[family]
		if !ok {
			t.Errorf("unexpected %s family %q", kind, family)
			return
		}
		if section != expected.section || !reflect.DeepEqual(fixtures, expected.fixtures) {
			t.Errorf("%s family %s traceability = (%q, %#v), want (%q, %#v)", kind, family, section, fixtures, expected.section, expected.fixtures)
		}
	}

	got := catalog.Current()
	for _, operation := range got.Operations {
		assertTraceability("operation", operation.Family, operation.NormativeSection, operation.FixtureFamilies, operationFamilies)
	}
	for _, capability := range got.Capabilities {
		assertTraceability("capability", capability.Family, capability.NormativeSection, capability.FixtureFamilies, capabilityFamilies)
	}
	for _, event := range got.Events {
		assertTraceability("event", event.Family, event.NormativeSection, event.FixtureFamilies, eventFamilies)
	}
	for _, errorCode := range got.Errors {
		if errorCode.NormativeSection != errorTraceability.section || !reflect.DeepEqual(errorCode.FixtureFamilies, errorTraceability.fixtures) {
			t.Errorf("error %s traceability = (%q, %#v), want (%q, %#v)", errorCode.Code, errorCode.NormativeSection, errorCode.FixtureFamilies, errorTraceability.section, errorTraceability.fixtures)
		}
	}
}

func TestUnknownReleaseIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := catalog.ForRelease("v0.5.1"); !errors.Is(err, catalog.ErrUnsupportedRelease) {
		t.Fatalf("ForRelease(v0.5.1) error = %v, want ErrUnsupportedRelease", err)
	}
}

func TestCurrentIsIdempotentAndReturnsIsolatedData(t *testing.T) {
	t.Parallel()

	first := catalog.Current()
	second := catalog.Current()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("repeated Current calls returned different catalogs")
	}

	first.Contracts[0].Versions[0] = "9.9.9"
	first.SelfIdentities[0].ContractVersions[0] = "9.9.9"
	first.Operations[0].RecoveryEvidence = append(first.Operations[0].RecoveryEvidence, "forged")
	first.Events[0].ContractVersions[0] = "9.9.9"
	third := catalog.Current()
	if reflect.DeepEqual(first, third) {
		t.Fatal("caller mutation leaked into the generated catalog")
	}
	if third.Contracts[0].Versions[0] == "9.9.9" ||
		third.SelfIdentities[0].ContractVersions[0] == "9.9.9" ||
		third.Events[0].ContractVersions[0] == "9.9.9" {
		t.Fatal("nested caller mutation leaked into the generated catalog")
	}
}

func TestCapabilityDefinitionsCannotAdvertiseRuntimeSupport(t *testing.T) {
	t.Parallel()

	typeOfCapability := reflect.TypeOf(catalog.Capability{})
	for _, forbidden := range []string{"Available", "Enabled", "Status", "Supported"} {
		if _, ok := typeOfCapability.FieldByName(forbidden); ok {
			t.Errorf("Capability exposes forbidden runtime-claim field %q", forbidden)
		}
	}
}

func contractPins(contracts []catalog.Contract) []specpin.ContractPin {
	result := make([]specpin.ContractPin, len(contracts))
	for index, contract := range contracts {
		result[index] = specpin.ContractPin{
			Name:     contract.Name,
			ID:       string(contract.ID),
			Versions: append([]string(nil), contract.Versions...),
		}
	}
	return result
}

func operationNames(value catalog.Catalog) map[string][]string {
	result := make(map[string][]string)
	for _, operation := range value.Operations {
		family := string(operation.Family)
		result[family] = append(result[family], string(operation.Name))
	}
	return result
}

func capabilityNames(value catalog.Catalog) map[string][]string {
	result := make(map[string][]string)
	for _, capability := range value.Capabilities {
		family := string(capability.Family)
		result[family] = append(result[family], string(capability.Name))
	}
	return result
}

func eventNames(value catalog.Catalog) map[string][]string {
	result := make(map[string][]string)
	for _, event := range value.Events {
		family := string(event.Family)
		result[family] = append(result[family], string(event.Name))
	}
	return result
}

func assertErrorMappings(t *testing.T, catalogValue catalog.Catalog, want map[int][]string) {
	t.Helper()
	got := make(map[int][]string)
	for _, item := range catalogValue.Errors {
		got[item.ExitCode] = append(got[item.ExitCode], string(item.Code))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("error mappings differ\ngot: %#v\nwant: %#v", got, want)
	}
}

func assertOperationEffectNames(t *testing.T, catalogValue catalog.Catalog, effect catalog.OperationEffect, want map[string][]string) {
	t.Helper()
	got := make(map[string][]string)
	for _, operation := range catalogValue.Operations {
		if operation.Effect == effect {
			family := string(operation.Family)
			got[family] = append(got[family], string(operation.Name))
		}
	}
	assertFamilyNames(t, got, want)
}

func assertFamilyNames(t *testing.T, got, want map[string][]string) {
	t.Helper()
	if reflect.DeepEqual(got, want) {
		return
	}
	gotFamilies := make([]string, 0, len(got))
	for family := range got {
		gotFamilies = append(gotFamilies, family)
	}
	sort.Strings(gotFamilies)
	t.Fatalf("catalog family names differ\ngot families: %v\ngot: %#v\nwant: %#v", gotFamilies, got, want)
}
