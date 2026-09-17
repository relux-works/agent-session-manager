// Authoritative summary tests for TASK-260830-21gygk (SPEC 14.7.3):
// bootstrap refusal, whole-list refusal, validated host and
// observation inputs, and observation-unavailable classification,
// all driven through the production summary entries.
package sessquery

import (
	"errors"
	"reflect"
	"testing"
)

// summaryReader builds a leased local session with host metadata,
// winning Lease Record, and required observations for its owner.
func summaryReader(t *testing.T) *Reader {
	t.Helper()
	repo, _ := repository(t)
	ref := create(t, repo, idA, "alpha")
	appendEvent(t, repo, idA, ref.RecordID, "session.created", 1, map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	present := true
	workspace := WorkspaceAbsent
	reader := &Reader{
		Local:        repo,
		LocalHostID:  hostA,
		HostMetadata: map[string]HostMetadata{hostA: {DisplayName: "primary"}},
		Observations: map[string]SessionObservations{idA: {
			ProcessPresent:  &present,
			Capabilities:    map[string]CapabilitySummary{},
			WorkspaceStatus: &workspace,
		}},
	}
	withLeases(reader, leaseCreate(t, idA, hostA))
	return reader
}

func TestSummaryBootstrapRefusals(t *testing.T) {
	local, _ := repository(t)
	create(t, local, idA, "alpha")
	reader := &Reader{Local: local, LocalHostID: hostA}
	for _, selector := range []string{"alpha", idA, "id:" + idA, "alpha@local"} {
		if _, err := reader.Status(selector); !errors.Is(err, ErrBootstrapIncomplete) {
			t.Fatalf("Status(%q) = %v, want selector_bootstrap_incomplete", selector, err)
		}
		if _, err := reader.AuthoritativeStatus(selector); !errors.Is(err, ErrBootstrapIncomplete) {
			t.Fatalf("AuthoritativeStatus(%q) = %v, want selector_bootstrap_incomplete", selector, err)
		}
	}
	if _, err := reader.AuthoritativeList(); !errors.Is(err, ErrBootstrapIncomplete) {
		t.Fatalf("AuthoritativeList = %v, want selector_bootstrap_incomplete", err)
	}
	// Explicit local inspection keeps raw access to the recovery
	// diagnostics: it is not a public summary entry.
	inspected, err := reader.InspectLocal(idA)
	if err != nil || inspected.Projection.SessionID != idA {
		t.Fatalf("InspectLocal = %+v, %v", inspected, err)
	}
}

func TestSummaryMixedListRefusesWhole(t *testing.T) {
	repo, _ := repository(t)
	healthy := create(t, repo, idA, "alpha")
	appendEvent(t, repo, idA, healthy.RecordID, "session.created", 1, map[string]any{"session_record_id": healthy.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB})
	create(t, repo, idB, "Zed")
	reader := &Reader{
		Local:        repo,
		LocalHostID:  hostA,
		HostMetadata: map[string]HostMetadata{hostA: {DisplayName: "primary"}},
	}
	// One unrepresentable record refuses the whole document: no
	// partial rows escape, and the healthy entry is not silently
	// kept while the incomplete one is omitted.
	rows, err := reader.List()
	if !errors.Is(err, ErrBootstrapIncomplete) || rows != nil {
		t.Fatalf("List = %d rows, %v, want whole selector_bootstrap_incomplete refusal", len(rows), err)
	}
	authorized, err := reader.AuthoritativeList()
	if !errors.Is(err, ErrBootstrapIncomplete) || authorized != nil {
		t.Fatalf("AuthoritativeList = %d rows, %v, want whole refusal", len(authorized), err)
	}
}

func TestAuthoritativeStatusHealthy(t *testing.T) {
	reader := summaryReader(t)
	summary, err := reader.AuthoritativeStatus("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if summary.SessionID != idA || summary.Name != "alpha" || summary.Kind != "direct" || summary.ProviderID != "codex" || summary.State != "creating" {
		t.Fatalf("identity: %+v", summary)
	}
	if summary.OwnerHostID != hostA || summary.OwnerHostName != "primary" || summary.LeaseEpoch != 1 || summary.LeaseID != lease || summary.LocalRole != "owner" {
		t.Fatalf("authority: %+v", summary)
	}
	// Before the first checkpoint both checkpoint members are null.
	if summary.HasCheckpoint || summary.NewestCheckpointID != nil || summary.NewestCheckpointCreatedAt != nil {
		t.Fatalf("checkpoint before first checkpoint: %+v", summary)
	}
	// Required observations bind explicitly: workspace absent is an
	// established absence, capabilities empty is established empty,
	// and process liveness is required for status.
	if summary.WorkspaceStatus != WorkspaceAbsent {
		t.Fatalf("workspace: %+v", summary)
	}
	if summary.Capabilities == nil || len(summary.Capabilities) != 0 {
		t.Fatalf("capabilities: %+v", summary)
	}
	if summary.ProcessPresent == nil || !*summary.ProcessPresent {
		t.Fatalf("process: %+v", summary)
	}
	if len(summary.Warnings) != 0 {
		t.Fatalf("warnings: %+v", summary)
	}
	list, err := reader.AuthoritativeList()
	if err != nil || len(list) != 1 {
		t.Fatalf("list/status disagreement: %+v %v", list, err)
	}
	if !reflect.DeepEqual(list[0].Capabilities, summary.Capabilities) || list[0].WorkspaceStatus != summary.WorkspaceStatus {
		t.Fatalf("list/status observation disagreement: %+v vs %+v", list[0], summary)
	}
}

func TestAuthoritativeMissingHostMetadataRefuses(t *testing.T) {
	reader := summaryReader(t)
	// The internal status stays representable: only the authoritative
	// layer requires the validated owner name.
	if _, err := reader.Status("alpha"); err != nil {
		t.Fatal(err)
	}
	reader.HostMetadata = nil
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing host metadata = %v, want selector_observation_unavailable", err)
	}
	if _, err := reader.AuthoritativeList(); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing host metadata list = %v, want selector_observation_unavailable", err)
	}
	// The source alias never substitutes for host metadata.
	reader.HostMetadata = map[string]HostMetadata{hostB: {DisplayName: "workstation"}}
	reader.Aliases = map[string]string{hostB: "workstation"}
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("alias-substituted metadata = %v, want selector_observation_unavailable", err)
	}
}

func TestAuthoritativeUnknownLocalHostRefuses(t *testing.T) {
	reader := summaryReader(t)
	reader.LocalHostID = ""
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("unknown local host = %v, want selector_observation_unavailable", err)
	}
}

func TestAuthoritativeCheckpointRequiresTimestamp(t *testing.T) {
	repo, _ := repository(t)
	ref := create(t, repo, idA, "Checkpointed")
	previous := ref.RecordID
	rows := []struct {
		typ     string
		payload map[string]any
	}{
		{"session.created", map[string]any{"session_record_id": ref.RecordID, "bootstrap_operation_id": hostA, "first_checkpoint_operation_id": hostB}},
		{"provider.launched", map[string]any{"provider_id": "codex", "provider_version": "0.147.0", "execution_profile": "yolo", "profile_source_event_id": nil, "profile_mapping": "default"}},
		{"session.idle", map[string]any{"boundary_ref": "turn-1", "foreground_idle": true, "background_idle": true}},
		{"checkpoint.created", map[string]any{"checkpoint_id": zeroDigest, "kind": "manual"}},
	}
	for index, row := range rows {
		previous = appendEvent(t, repo, idA, previous, row.typ, index+1, row.payload)
	}
	present := true
	workspace := WorkspaceAbsent
	reader := &Reader{
		Local:        repo,
		LocalHostID:  hostA,
		HostMetadata: map[string]HostMetadata{hostA: {DisplayName: "primary"}},
		Observations: map[string]SessionObservations{idA: {
			ProcessPresent:  &present,
			Capabilities:    map[string]CapabilitySummary{},
			WorkspaceStatus: &workspace,
		}},
	}
	withLeases(reader, leaseCreate(t, idA, hostA))
	if _, err := reader.AuthoritativeStatus("Checkpointed"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("checkpoint without timestamp observation = %v, want selector_observation_unavailable", err)
	}
	stamp := "2026-08-19T04:09:30.000Z"
	observations := reader.Observations[idA]
	observations.NewestCheckpointCreatedAt = &stamp
	reader.Observations[idA] = observations
	summary, err := reader.AuthoritativeStatus("Checkpointed")
	if err != nil {
		t.Fatal(err)
	}
	if !summary.HasCheckpoint || summary.NewestCheckpointID == nil || *summary.NewestCheckpointID != zeroDigest {
		t.Fatalf("checkpoint id: %+v", summary)
	}
	if summary.NewestCheckpointCreatedAt == nil || *summary.NewestCheckpointCreatedAt != stamp {
		t.Fatalf("checkpoint timestamp: %+v", summary)
	}
}

func TestAuthoritativeInputValidation(t *testing.T) {
	reader := summaryReader(t)
	badStamp := "not-a-timestamp"
	cases := []struct {
		name   string
		mutate func()
	}{
		{"host key", func() {
			reader.HostMetadata = map[string]HostMetadata{"nope": {DisplayName: "primary"}}
		}},
		{"empty display name", func() {
			reader.HostMetadata = map[string]HostMetadata{hostA: {}}
		}},
		{"control display name", func() {
			reader.HostMetadata = map[string]HostMetadata{hostA: {DisplayName: "a\tb"}}
		}},
		{"long display name", func() {
			reader.HostMetadata = map[string]HostMetadata{hostA: {DisplayName: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}}
		}},
		{"observation key", func() {
			reader.Observations = map[string]SessionObservations{"nope": {}}
		}},
		{"malformed timestamp", func() {
			present := true
			workspace := WorkspaceAbsent
			reader.Observations = map[string]SessionObservations{idA: {
				NewestCheckpointCreatedAt: &badStamp,
				ProcessPresent:            &present,
				Capabilities:              map[string]CapabilitySummary{},
				WorkspaceStatus:           &workspace,
			}}
		}},
		{"empty capability name", func() {
			present := true
			workspace := WorkspaceAbsent
			reader.Observations = map[string]SessionObservations{idA: {
				ProcessPresent:  &present,
				Capabilities:    map[string]CapabilitySummary{"": {Status: "available", Enabled: true}},
				WorkspaceStatus: &workspace,
			}}
		}},
		{"bad capability status", func() {
			present := true
			workspace := WorkspaceAbsent
			reader.Observations = map[string]SessionObservations{idA: {
				ProcessPresent:  &present,
				Capabilities:    map[string]CapabilitySummary{"native_resume": {Status: "ready", Enabled: false}},
				WorkspaceStatus: &workspace,
			}}
		}},
		{"enabled non-available", func() {
			present := true
			workspace := WorkspaceAbsent
			reader.Observations = map[string]SessionObservations{idA: {
				ProcessPresent:  &present,
				Capabilities:    map[string]CapabilitySummary{"managed_pty": {Status: "unknown", Enabled: true}},
				WorkspaceStatus: &workspace,
			}}
		}},
		{"unknown capability name", func() {
			present := true
			workspace := WorkspaceAbsent
			reader.Observations = map[string]SessionObservations{idA: {
				ProcessPresent:  &present,
				Capabilities:    map[string]CapabilitySummary{"exec": {Status: "available", Enabled: true}},
				WorkspaceStatus: &workspace,
			}}
		}},
		{"bad workspace", func() {
			present := true
			bad := "pending"
			reader.Observations = map[string]SessionObservations{idA: {
				ProcessPresent:  &present,
				Capabilities:    map[string]CapabilitySummary{},
				WorkspaceStatus: &bad,
			}}
		}},
	}
	for _, row := range cases {
		t.Run(row.name, func(t *testing.T) {
			snapshotHosts, snapshotObs := reader.HostMetadata, reader.Observations
			row.mutate()
			defer func() { reader.HostMetadata, reader.Observations = snapshotHosts, snapshotObs }()
			if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("malformed authority input = %v, want invalid_config", err)
			}
		})
	}
}

func TestAuthoritativeOptionalObservations(t *testing.T) {
	reader := summaryReader(t)
	present := true
	workspace := WorkspaceCurrent
	reader.Observations = map[string]SessionObservations{idA: {
		ProcessPresent: &present,
		Capabilities: map[string]CapabilitySummary{
			"native_resume":  {Status: "available", Enabled: true, Detail: "ok"},
			"portable_store": {Status: "conditional", Enabled: false, Detail: ""},
		},
		WorkspaceStatus: &workspace,
	}}
	summary, err := reader.AuthoritativeStatus("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if summary.ProcessPresent == nil || !*summary.ProcessPresent {
		t.Fatalf("liveness: %+v", summary)
	}
	wantCapabilities := map[string]CapabilitySummary{
		"native_resume":  {Status: "available", Enabled: true, Detail: "ok"},
		"portable_store": {Status: "conditional", Enabled: false, Detail: ""},
	}
	if !reflect.DeepEqual(summary.Capabilities, wantCapabilities) {
		t.Fatalf("capabilities: %+v", summary)
	}
	if summary.WorkspaceStatus != WorkspaceCurrent {
		t.Fatalf("workspace: %+v", summary)
	}
}

// TestAuthoritativeAdmitsFullCapabilityRegistry proves every Section
// 7.3 registry name is admitted through both authoritative entries,
// confirming the bound vocabulary is exactly the provhost-owned
// seven-name registry with no eighth invented name.
func TestAuthoritativeAdmitsFullCapabilityRegistry(t *testing.T) {
	reader := summaryReader(t)
	present := true
	workspace := WorkspaceAbsent
	admitted := map[string]CapabilitySummary{
		"native_resume":       {Status: "available", Enabled: true},
		"portable_store":      {Status: "conditional", Enabled: false},
		"managed_pty":         {Status: "conditional", Enabled: false},
		"appserver":           {Status: "unsupported", Enabled: false},
		"task_board_primary":  {Status: "unknown", Enabled: false},
		"prompt_spawn":        {Status: "unknown", Enabled: false},
		"native_goal_binding": {Status: "unsupported", Enabled: false},
	}
	if len(admitted) != MaxCapabilities {
		t.Fatalf("registry fixture covers %d names, want %d", len(admitted), MaxCapabilities)
	}
	reader.Observations = map[string]SessionObservations{idA: {
		ProcessPresent:  &present,
		Capabilities:    admitted,
		WorkspaceStatus: &workspace,
	}}
	status, err := reader.AuthoritativeStatus("alpha")
	if err != nil {
		t.Fatalf("full registry status refused: %v", err)
	}
	if !reflect.DeepEqual(status.Capabilities, admitted) {
		t.Fatalf("status capabilities: %+v", status.Capabilities)
	}
	list, err := reader.AuthoritativeList()
	if err != nil {
		t.Fatalf("full registry list refused: %v", err)
	}
	if len(list) != 1 || !reflect.DeepEqual(list[0].Capabilities, admitted) {
		t.Fatalf("list capabilities: %+v", list)
	}
}

func TestAuthoritativeUnknownObservationsRefuse(t *testing.T) {
	reader := summaryReader(t)
	// An empty observation map carries no required facts: both status
	// and list refuse observation_unavailable rather than minting
	// process, workspace, or capability facts.
	reader.Observations = map[string]SessionObservations{}
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("unknown observations status = %v, want selector_observation_unavailable", err)
	}
	if _, err := reader.AuthoritativeList(); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("unknown observations list = %v, want selector_observation_unavailable", err)
	}
	// Established absence is distinct: an empty non-nil capability map
	// with workspace absent and process present stays representable.
	present := true
	workspace := WorkspaceAbsent
	reader.Observations = map[string]SessionObservations{idA: {
		ProcessPresent:  &present,
		Capabilities:    map[string]CapabilitySummary{},
		WorkspaceStatus: &workspace,
	}}
	if _, err := reader.AuthoritativeStatus("alpha"); err != nil {
		t.Fatalf("established absence refused: %v", err)
	}
}

func TestAuthoritativeMissingCapabilitiesRefuses(t *testing.T) {
	reader := summaryReader(t)
	// Workspace and process present, capabilities unknown: both status
	// and list refuse rather than minting an empty map.
	present := true
	workspace := WorkspaceAbsent
	reader.Observations = map[string]SessionObservations{idA: {
		ProcessPresent:  &present,
		WorkspaceStatus: &workspace,
	}}
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing capabilities status = %v, want selector_observation_unavailable", err)
	}
	if _, err := reader.AuthoritativeList(); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing capabilities list = %v, want selector_observation_unavailable", err)
	}
}

func TestAuthoritativeMissingProcessRefusesStatus(t *testing.T) {
	reader := summaryReader(t)
	// Workspace and capabilities present, process unknown: status
	// refuses (process_present boolean is required there) while the
	// list stays representable without it.
	workspace := WorkspaceAbsent
	reader.Observations = map[string]SessionObservations{idA: {
		Capabilities:    map[string]CapabilitySummary{},
		WorkspaceStatus: &workspace,
	}}
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing process status = %v, want selector_observation_unavailable", err)
	}
	if _, err := reader.AuthoritativeList(); err != nil {
		t.Fatalf("list without process refused: %v", err)
	}
}

func TestAuthoritativeMissingWorkspaceRefuses(t *testing.T) {
	reader := summaryReader(t)
	// Capabilities and process present, workspace unknown: both status
	// and list refuse rather than minting a workspace fact.
	present := true
	reader.Observations = map[string]SessionObservations{idA: {
		ProcessPresent: &present,
		Capabilities:   map[string]CapabilitySummary{},
	}}
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing workspace status = %v, want selector_observation_unavailable", err)
	}
	if _, err := reader.AuthoritativeList(); !errors.Is(err, ErrObservationUnavailable) {
		t.Fatalf("missing workspace list = %v, want selector_observation_unavailable", err)
	}
}

func TestAuthoritativeHostNameBound64(t *testing.T) {
	reader := summaryReader(t)
	long := ""
	for i := 0; i < 65; i++ {
		long += "x"
	}
	reader.HostMetadata = map[string]HostMetadata{hostA: {DisplayName: long}}
	if _, err := reader.AuthoritativeStatus("alpha"); !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("65-character host name = %v, want invalid_config", err)
	}
}
