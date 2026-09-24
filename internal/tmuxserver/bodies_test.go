package tmuxserver

import (
	"encoding/json"
	"testing"
)

func TestParseCreateBodyValid(t *testing.T) {
	body, err := parseCreateBody(lxCreateBody(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if body.Context.SessionID != lxSession || body.BootstrapOperationID != lxBootstrap {
		t.Fatalf("context = %+v", body.Context)
	}
	if body.Binding.TerminalInstanceID != lxInstance || body.Binding.BindingID == "" {
		t.Fatalf("binding = %+v", body.Binding)
	}
	if body.Transport != "local_only" || !body.Interactive {
		t.Fatalf("transport/interactive = %s/%v", body.Transport, body.Interactive)
	}
}

func TestParseCreateBodyMemberSet(t *testing.T) {
	members := []string{"context", "binding", "bootstrap_operation_id", "entrypoint", "presentation_transport", "interactive"}
	for _, drop := range members {
		raw := lxCreateBody(t, func(object map[string]any) { delete(object, drop) })
		if _, err := parseCreateBody(raw); err == nil {
			t.Fatalf("missing %s admitted", drop)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "create body members")
		}
	}
	raw := lxCreateBody(t, func(object map[string]any) { object["smuggled"] = 1 })
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("unknown member admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "create body members")
	}
	if _, err := parseCreateBody([]byte(`{"context":{},`)); err == nil {
		t.Fatal("truncated frame admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "create body frame")
	}
}

func TestParseCreateBodyMembers(t *testing.T) {
	raw := lxCreateBody(t, func(object map[string]any) { object["bootstrap_operation_id"] = "nope" })
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("bad bootstrap admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "create body bootstrap")
	}
	raw = lxCreateBody(t, func(object map[string]any) { object["interactive"] = "yes" })
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("non-bool interactive admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "create body interactive")
	}
	raw = lxCreateBody(t, func(object map[string]any) { object["presentation_transport"] = "pigeon" })
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("unknown transport admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "presentation transport vocabulary")
	}
	// The relay member parses: admission is not authorization, and the
	// relay refuses downstream (landed attach arm / create relay arm).
	for _, transport := range []string{"local_only", "trusted_private_mesh", "third_party_relay"} {
		raw := lxCreateBody(t, func(object map[string]any) { object["presentation_transport"] = transport })
		body, err := parseCreateBody(raw)
		if err != nil {
			t.Fatalf("transport %s refused at parse: %v", transport, err)
		}
		if body.Transport != transport {
			t.Fatalf("transport = %s", body.Transport)
		}
	}
}

func TestParseCreateBodyEntrypoint(t *testing.T) {
	shapes := []any{
		`["ax","pane"]`,
		`{"argv":["ax","pane"]}`,
		`{"argv":["ax","pane","` + lxSession + `","extra"]}`,
		`{"argv":["ax","pane","not-a-uuid"]}`,
		`{"argv":["ax","pane","` + lxSessionB + `"]}`,
		`{"argv":["provider","run","` + lxSession + `"]}`,
		`{"argv":"ax pane"}`,
		`{"vector":["ax","pane","` + lxSession + `"]}`,
	}
	for _, shape := range shapes {
		var entry any
		switch typed := shape.(type) {
		case string:
			if err := json.Unmarshal([]byte(typed), &entry); err != nil {
				t.Fatal(err)
			}
		default:
			entry = typed
		}
		raw := lxCreateBody(t, func(object map[string]any) { object["entrypoint"] = entry })
		if _, err := parseCreateBody(raw); err == nil {
			t.Fatalf("entrypoint %v admitted", shape)
		}
	}
}

func TestParseEntrypointStringRefuses(t *testing.T) {
	// A string argv member is not the vector shape: the unmarshal arm
	// refuses before the landed entrypoint check ever runs.
	raw := lxCreateBody(t, func(object map[string]any) { object["entrypoint"] = map[string]any{"argv": "ax pane"} })
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("string argv admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "create body entrypoint")
	}
}

func TestParseCreateBodyNestedDocuments(t *testing.T) {
	raw := lxCreateBody(t, func(object map[string]any) {
		context := object["context"].(map[string]any)
		context["session_id"] = "nope"
	})
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("bad nested context admitted")
	}
	raw = lxCreateBody(t, func(object map[string]any) {
		var binding map[string]any
		inner, _ := json.Marshal(object["binding"])
		if err := json.Unmarshal(inner, &binding); err != nil {
			t.Fatal(err)
		}
		binding["binding_id"] = lxDigestB
		object["binding"] = binding
	})
	if _, err := parseCreateBody(raw); err == nil {
		t.Fatal("tampered binding admitted")
	}
}

func TestParseAttachBodyValid(t *testing.T) {
	body, err := parseAttachBody(lxAttachBody(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if body.SessionID != lxSession || body.ClientID != lxClient || body.Authorization.PolicyEvidenceID != lxDigestA {
		t.Fatalf("body = %+v", body)
	}
	if len(body.AuthRaw) == 0 {
		t.Fatal("auth raw not carried")
	}
}

func TestParseAttachBodyMemberSet(t *testing.T) {
	members := []string{"session_id", "terminal_instance_id", "terminal_backend_id", "implementation_version", "protocol_version", "backend_generation", "client_id", "transport", "input_authorized", "deadline_at", "authorization"}
	for _, drop := range members {
		raw := lxAttachBody(t, func(object map[string]any) { delete(object, drop) })
		if _, err := parseAttachBody(raw); err == nil {
			t.Fatalf("missing %s admitted", drop)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "attach body members")
		}
	}
	raw := lxAttachBody(t, func(object map[string]any) { object["lease_id"] = lxLease })
	if _, err := parseAttachBody(raw); err == nil {
		t.Fatal("lease member admitted into ownership-neutral attach")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "attach body members")
	}
	if _, err := parseAttachBody([]byte(`[1,2]`)); err == nil {
		t.Fatal("non-object frame admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "attach body frame")
	}
}

func TestParseAttachBodyMembers(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
		detail string
	}{
		{"session", func(o map[string]any) { o["session_id"] = "nope" }, "attach body session"},
		{"instance", func(o map[string]any) { o["terminal_instance_id"] = "nope" }, "attach body instance"},
		{"backend", func(o map[string]any) { o["terminal_backend_id"] = 7 }, "attach body backend"},
		{"versions", func(o map[string]any) { o["protocol_version"] = 7 }, "attach body versions"},
		{"client", func(o map[string]any) { o["client_id"] = "nope" }, "attach body client"},
		{"transport", func(o map[string]any) { o["transport"] = "pigeon" }, "presentation transport vocabulary"},
		{"input", func(o map[string]any) { o["input_authorized"] = "yes" }, "attach body input"},
		{"deadline", func(o map[string]any) { o["deadline_at"] = "yesterday" }, "attach body deadline"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseAttachBody(lxAttachBody(t, tc.mutate)); err == nil {
				t.Fatalf("%s admitted", tc.name)
			} else {
				requireLocalCode(t, err, "terminal_backend_protocol_error", tc.detail)
			}
		})
	}
	raw := lxAttachBody(t, func(o map[string]any) { o["backend_generation"] = "" })
	if _, err := parseAttachBody(raw); err == nil {
		t.Fatal("empty generation admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_stale_generation", "backend_generation bound")
	}
	raw = lxAttachBody(t, func(o map[string]any) { o["terminal_backend_id"] = "Nope" })
	if _, err := parseAttachBody(raw); err == nil {
		t.Fatal("malformed backend admitted")
	}
	raw = lxAttachBody(t, func(o map[string]any) {
		auth := o["authorization"].(map[string]any)
		auth["transport"] = "pigeon"
	})
	if _, err := parseAttachBody(raw); err == nil {
		t.Fatal("bad nested authorization admitted")
	}
}

func TestParseTerminateBodyValid(t *testing.T) {
	body, err := parseTerminateBody(lxTerminateBody(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if body.StaleLeaseID != lxLeaseB || body.StaleEpoch != 9 || body.DiagnosticEvidenceID != lxDigestC {
		t.Fatalf("body = %+v", body)
	}
}

func TestParseTerminateBodyViolations(t *testing.T) {
	members := []string{"context", "stale_lease_id", "stale_epoch", "diagnostic_evidence_id"}
	for _, drop := range members {
		raw := lxTerminateBody(t, func(object map[string]any) { delete(object, drop) })
		if _, err := parseTerminateBody(raw); err == nil {
			t.Fatalf("missing %s admitted", drop)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "terminate body members")
		}
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
		detail string
	}{
		{"lease-type", func(o map[string]any) { o["stale_lease_id"] = 7 }, "terminate body lease"},
		{"lease-shape", func(o map[string]any) { o["stale_lease_id"] = lxSession }, "terminate body lease"},
		{"epoch-zero", func(o map[string]any) { o["stale_epoch"] = float64(0) }, "terminate body epoch"},
		{"epoch-huge", func(o map[string]any) { o["stale_epoch"] = float64(9007199254740992) }, "terminate body epoch"},
		{"evidence", func(o map[string]any) { o["diagnostic_evidence_id"] = "nope" }, "terminate body evidence"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseTerminateBody(lxTerminateBody(t, tc.mutate)); err == nil {
				t.Fatalf("%s admitted", tc.name)
			} else {
				requireLocalCode(t, err, "terminal_backend_protocol_error", tc.detail)
			}
		})
	}
	// The member count is load-bearing for extras: the name loop
	// would pass a body that carries every known member plus one.
	raw := lxTerminateBody(t, func(object map[string]any) { object["smuggled"] = 1 })
	if _, err := parseTerminateBody(raw); err == nil {
		t.Fatal("unknown member admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "terminate body members")
	}
}

func TestParseRestoreBodyValid(t *testing.T) {
	body, err := parseRestoreBody(lxRestoreBody(t, lxDigestA, nil))
	if err != nil {
		t.Fatal(err)
	}
	if body.PriorBindingID != lxDigestA || body.CheckpointID != lxDigestC || body.BootstrapOperationID != lxBootstrap {
		t.Fatalf("body = %+v", body)
	}
}

func TestParseRestoreBodyViolations(t *testing.T) {
	members := []string{"context", "prior_binding_id", "checkpoint_id", "bootstrap_operation_id"}
	for _, drop := range members {
		raw := lxRestoreBody(t, lxDigestA, func(object map[string]any) { delete(object, drop) })
		if _, err := parseRestoreBody(raw); err == nil {
			t.Fatalf("missing %s admitted", drop)
		} else {
			requireLocalCode(t, err, "terminal_backend_protocol_error", "restore body members")
		}
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
		detail string
	}{
		{"binding", func(o map[string]any) { o["prior_binding_id"] = "nope" }, "restore body binding"},
		{"checkpoint", func(o map[string]any) { o["checkpoint_id"] = "nope" }, "restore body checkpoint"},
		{"bootstrap", func(o map[string]any) { o["bootstrap_operation_id"] = "nope" }, "restore body bootstrap"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseRestoreBody(lxRestoreBody(t, lxDigestA, tc.mutate)); err == nil {
				t.Fatalf("%s admitted", tc.name)
			} else {
				requireLocalCode(t, err, "terminal_backend_protocol_error", tc.detail)
			}
		})
	}
	// The member count is load-bearing for extras: the name loop
	// would pass a body that carries every known member plus one.
	raw := lxRestoreBody(t, lxDigestA, func(object map[string]any) { object["smuggled"] = 1 })
	if _, err := parseRestoreBody(raw); err == nil {
		t.Fatal("unknown member admitted")
	} else {
		requireLocalCode(t, err, "terminal_backend_protocol_error", "restore body members")
	}
}
