package cloneproject

import (
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/clonesnap"
)

// Negative tests: every refusal below drives the production
// Normalize entry and fails when the gate admits what it must
// reject. Each row names the offending input and the refusal
// fragment the gate must emit.

func TestNormalizeRefusesUnstableCapture(t *testing.T) {
	members := []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-u-1", "message/user", "native", "none", "main", `{"text":"unstable"}`),
		)},
	}
	root := writeFixtureStore(t, members)
	store := openTestStore(t)
	request := validCaptureRequest(t, root, store, members)
	request.OnRace = clonesnap.RaceArchive
	request.OperatorExplicit = true
	request.Hooks.AfterWalk = func(storeRoot string) error {
		return writeStoreFile(storeRoot, "store/session.jsonl", jsonl(
			rec("evt-u-1", "message/user", "native", "none", "main", `{"text":"mutated"}`),
		))
	}
	capture, err := clonesnap.Capture(request)
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if capture.BoundaryKind != "unstable_archive" {
		t.Fatalf("BoundaryKind = %q, want unstable_archive", capture.BoundaryKind)
	}
	_, err = Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
	if err == nil {
		t.Fatal("Normalize() admitted an unstable_archive capture to target projection")
	}
	if !errors.Is(err, ErrInvalid) && !strings.Contains(err.Error(), "cannot enter a target branch") {
		t.Fatalf("Normalize() error = %v, want the target-branch refusal", err)
	}
}

func TestNormalizeRefusesEnvironmentDrift(t *testing.T) {
	// Both manifests seal clean and the ID link holds, but the
	// sealed environments differ: the evidence environment would be
	// ambiguous, so projection refuses. The real capture entry
	// cannot produce this pair (one tuple seals both sides), so the
	// test builds it through the landed builders instead.
	envA := fixtureTupleJSON()
	envB := `{"environment_id":"project.evw","environment_version":"3.0.1","platform":"linux","architecture":"arm64","store_schema_fingerprint":` +
		`"` + fixtureDigest("project-store-fingerprint") + `","adapter_version":"2.0.0"}`
	identity := fixtureNativeIdentity(t)
	identityDigest, err := clonebundle.IdentityDigest(identity, nil)
	if err != nil {
		t.Fatalf("IdentityDigest() error = %v", err)
	}
	rawBytes, err := clonebundle.BuildRawObjectManifest(
		fixtureOperationID, []byte(envA), "native-session-project",
		identityDigest.String(), fixtureDigest("project-capture-plan"), nil, nil)
	if err != nil {
		t.Fatalf("BuildRawObjectManifest() error = %v", err)
	}
	raw, err := clonebundle.DecodeRawObjectManifest(rawBytes)
	if err != nil {
		t.Fatalf("DecodeRawObjectManifest() error = %v", err)
	}
	pre := fixtureDigest("drift-pre")
	captureBytes, err := clonebundle.BuildCaptureManifest(clonebundle.CaptureManifestInput{
		OperationID: fixtureOperationID,
		BundleID:    fixtureBundleID,
		SourceBasis: clonebundle.SourceBasisInput{
			Kind:                     "ax_session",
			SourceSessionID:          fixtureSessionID,
			SourceSessionRecordID:    fixtureDigest("project-session-record"),
			SourceCheckpointID:       fixtureDigest("project-checkpoint"),
			SourceProviderIdentityID: fixtureDigest("project-provider-identity"),
		},
		SourceEnvironment: []byte(envB),
		SourceIdentity:    identity,
		CapturePlanDigest: fixtureDigest("project-capture-plan"),
		Boundary: clonebundle.BoundaryInput{
			Kind:              "stable",
			ProofKind:         "closed_store",
			Generation:        "generation-11",
			PreCaptureDigest:  pre,
			PostCaptureDigest: pre,
			InputBlocked:      true,
			ForegroundIdle:    true,
			BackgroundIdle:    true,
		},
		SourceRawManifestID: raw.ManifestID.String(),
		CreatedByHostID:     fixtureHostID,
		CreatedAt:           fixtureCreatedAt,
	})
	if err != nil {
		t.Fatalf("BuildCaptureManifest() error = %v", err)
	}
	_, probeStore := captureMembers(t, []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-drift-ws", "message/user", "native", "none", "main", `{"text":"x"}`),
		)},
	})
	request := NormalizeRequest{
		CaptureManifest:  captureBytes,
		RawManifest:      rawBytes,
		Fetch:            storeFetch(t, probeStore),
		CanonicalSink:    openTestStore(t),
		LogicalSessionID: fixtureLogicalID,
		MainActorID:      fixtureMainActor,
		Workspace:        mustWorkspaceBytes(t),
	}
	_, err = Normalize(request)
	if err == nil {
		t.Fatal("Normalize() admitted a capture whose sealed environments drift")
	}
	if !strings.Contains(err.Error(), "source_environment disagree") {
		t.Fatalf("Normalize() error = %v, want the closure-drift refusal", err)
	}
}

func TestNormalizeRefusalTable(t *testing.T) {
	// refuseMembers projects members that must refuse and asserts
	// the refusal fragment.
	refuseMembers := func(t *testing.T, members []fixtureMember, fragment string) {
		t.Helper()
		capture, store := captureMembers(t, members)
		_, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
		if err == nil {
			t.Fatalf("Normalize() admitted members that must refuse with %q", fragment)
		}
		if !errors.Is(err, ErrInvalid) && !strings.Contains(err.Error(), fragment) {
			t.Fatalf("Normalize() error = %v, want fragment %q", err, fragment)
		}
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("Normalize() error = %v, want fragment %q", err, fragment)
		}
	}
	message := func(id, body string) string {
		return rec(id, "message/user", "native", "none", "main", body)
	}
	call := func(id, body string) string {
		return rec("evt-"+id, "tool/call", "native", "none", "main", body)
	}

	t.Run("malformed_json_line", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: []byte("{not-json\n")},
		}, "is not a JSON object")
	})
	t.Run("missing_envelope_member", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				`{"v":1,"native_event_id":"x","native_type":"message/user","origin":"native","protection":"none","actor":"main"}`,
			)},
		}, "misses envelope member")
	})
	t.Run("bad_envelope_version", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				`{"v":2,"native_event_id":"x","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`,
			)},
		}, "envelope version")
	})
	t.Run("bad_origin", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-origin", "message/user", "sideways", "none", "main", `{"text":"x"}`),
			)},
		}, "outside native|foreign")
	})
	t.Run("bad_protection", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-prot", "message/user", "native", "rot13", "main", `{"text":"x"}`),
			)},
		}, "outside none|encrypted|signed")
	})
	t.Run("bad_actor_shape", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-actor", "message/user", "native", "none", "boss", `{"text":"x"}`),
			)},
		}, "outside main|external|subagent")
	})
	t.Run("long_subagent_name", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-long-sub", "message/user", "native", "none", "subagent:"+strings.Repeat("s", 129), `{"text":"x"}`),
			)},
		}, "past 128 characters")
	})

	t.Run("empty_subagent_name", func(t *testing.T) {
		// The empty subagent name refuses at the shape gate
		// with the shape message: a plant that admits it past
		// the shape arm still refuses one step later at the
		// actor mapping, with a different message (message pin).
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-empty-sub", "message/user", "native", "none", "subagent:", `{"text":"x"}`),
			)},
		}, `actor "subagent:" is outside main|external|subagent:<name>`)
	})
	t.Run("too_many_directives", func(t *testing.T) {
		directives := `"` + strings.Repeat(`d","`, 1024) + `d"`
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-many-dir", "instruction/snapshot", "native", "none", "main",
					`{"authority":"low","directives":[`+directives+`]}`),
			)},
		}, "maximum is 1024")
	})
	t.Run("empty_directive", func(t *testing.T) {
		// The low edge of string[1..4096]: an empty directive
		// refuses with its index named.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-empty-dir", "instruction/snapshot", "native", "none", "main",
					`{"authority":"low","directives":[""]}`),
			)},
		}, "directive[0] is not a string[1..4096]")
	})
	t.Run("usage_ledger_overflow", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-big-1", "usage/report", "native", "none", "main", `{"input_tokens":9007199254740991,"output_tokens":0}`),
				rec("evt-big-2", "usage/report", "native", "none", "main", `{"input_tokens":1,"output_tokens":0}`),
			)},
		}, "exceeds uint53")
	})

	t.Run("usage_ledger_overflow_output", func(t *testing.T) {
		// The output axis folds through its own overflow arm:
		// the refusal names the output total, not the input one.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-big-o1", "usage/report", "native", "none", "main", `{"input_tokens":1,"output_tokens":9007199254740991}`),
				rec("evt-big-o2", "usage/report", "native", "none", "main", `{"input_tokens":1,"output_tokens":1}`),
			)},
		}, "source output token total exceeds uint53")
	})
	t.Run("bad_actor_uuid", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-uuid", "message/user", "native", "none", "external", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Actors = map[string]string{"external": "not-a-uuid"}
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted a non-UUIDv7 actor mapping")
		}
		if !strings.Contains(err.Error(), "is not a UUIDv7") {
			t.Fatalf("Normalize() error = %v, want the UUID refusal", err)
		}
	})
	t.Run("blank_line", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: []byte(message("a", `{"text":"x"}`) + "\n\n")},
		}, "is empty")
	})
	t.Run("unknown_body_member", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-body-extra", `{"text":"x","escalate":true}`),
			)},
		}, "unknown member")
	})

	t.Run("call_live_followup_member", func(t *testing.T) {
		// The call registry carries no live_followup member: a
		// call smuggling the result-only optional refuses at the
		// per-type registry, never silently ignored.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-x", `{"call_id":"call-x","tool_name":"read","live_followup":true}`),
			)},
		}, `body carries unknown member "live_followup"`)
	})

	t.Run("result_live_attestation_member", func(t *testing.T) {
		// The result registry carries no live_attestation
		// member: a result smuggling the definition-only
		// optional refuses at the per-type registry.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-res-att", "tool/result", "native", "none", "main", `{"call_id":"call-y","status":"ok","live_attestation":true}`),
			)},
		}, `body carries unknown member "live_attestation"`)
	})
	t.Run("body_text_number", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-body-textnum", `{"text":123}`),
			)},
		}, `body member "text" is not a string`)
	})
	t.Run("body_live_attestation_string", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-body-boolstr", "tool/definition", "native", "none", "main",
					`{"tool_name":"write","definition_digest":"`+fixtureDigest("boolstr")+`","live_attestation":"true"}`),
			)},
		}, `body member "live_attestation" is not a boolean`)
	})
	t.Run("body_missing_text", func(t *testing.T) {
		// The message names the required-member gate: under a
		// plant that skips the check for text the record still
		// refuses, but with "is not a string" instead.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-body-missing", `{}`),
			)},
		}, `body misses member "text"`)
	})
	t.Run("contradictory_protection", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-contra", "reasoning/encrypted", "native", "none", "main", `{"ciphertext":"x"}`),
			)},
		}, "with no protection")
	})
	t.Run("protected_plaintext_body", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-contra-body", "reasoning/summary", "native", "encrypted", "main", `{"text":"x"}`),
			)},
		}, "unknown member")
	})
	t.Run("live_attestation_claim", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-evil-def", "tool/definition", "native", "none", "main",
					`{"tool_name":"evil-tool","definition_digest":"`+fixtureDigest("evil")+`","live_attestation":true}`),
			)},
		}, "claims live authority")
	})
	t.Run("live_followup_claim", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-live", `{"call_id":"call-live","tool_name":"write"}`),
				rec("evt-call-live-result", "tool/result", "native", "none", "main",
					`{"call_id":"call-live","status":"ok","live_followup":true}`),
			)},
		}, "claims a live followup")
	})
	t.Run("duplicate_call_id", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-dup", `{"call_id":"call-dup","tool_name":"read"}`),
				call("call-dup-again", `{"call_id":"call-dup","tool_name":"write"}`),
			)},
		}, "is not unique")
	})
	t.Run("duplicate_result_id", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-r", `{"call_id":"call-r","tool_name":"read"}`),
				rec("evt-r1", "tool/result", "native", "none", "main", `{"call_id":"call-r","status":"ok"}`),
				rec("evt-r2", "tool/result", "native", "none", "main", `{"call_id":"call-r","status":"error"}`),
			)},
		}, "is not unique")
	})
	t.Run("bad_instruction_authority", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-auth", "instruction/snapshot", "native", "none", "main",
					`{"authority":"absolute","directives":[]}`),
			)},
		}, "outside high|low")
	})
	t.Run("fractional_usage", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-use", "usage/report", "native", "none", "main", `{"input_tokens":1.5,"output_tokens":2}`),
			)},
		}, "is not a uint53")
	})
	t.Run("unsafe_usage_magnitude", func(t *testing.T) {
		// 2^53 is outside the uint53 type, so the landed gate
		// refuses it as not-a-uint53; the "exceeds uint53" message
		// stays with the ledger-sum gate (usage_ledger_overflow).
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-big-use", "usage/report", "native", "none", "main", `{"input_tokens":9007199254740992,"output_tokens":2}`),
			)},
		}, "is not a uint53")
	})
	t.Run("bad_definition_digest", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-bad-def", "tool/definition", "native", "none", "main",
					`{"tool_name":"write","definition_digest":"not-a-digest"}`),
			)},
		}, "is not a digest")
	})
	t.Run("unmapped_actor", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-stray", "message/user", "native", "none", "subagent:stray", `{"text":"x"}`),
			)},
		}, "with no mapped UUID")
	})
	t.Run("invalid_utf8_line", func(t *testing.T) {
		raw := append([]byte(`{"v":1,"native_event_id":"evt-utf8","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"`), 0xff)
		raw = append(raw, []byte(`"}}`+"\n")...)
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: raw},
		}, "is not valid UTF-8")
	})
	t.Run("empty_projection", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/sidecar", class: "durable_sidecar", content: []byte{}},
		})
		_, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
		if err == nil {
			t.Fatal("Normalize() admitted a projection with no events")
		}
		if !strings.Contains(err.Error(), "holds no events") {
			t.Fatalf("Normalize() error = %v, want the empty-projection refusal", err)
		}
	})
	t.Run("tampered_blob", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-tamper", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		healthy := request.Fetch
		request.Fetch = func(blobID string) ([]byte, error) {
			payload, err := healthy(blobID)
			if err != nil {
				return nil, err
			}
			payload[0] ^= 0xff
			return payload, nil
		}
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted tampered blob bytes")
		}
		// The digest arm's own message: an equal-size flip must
		// fail here, not at the size arm (blob_size_drift pins
		// that arm with its own message).
		if !strings.Contains(err.Error(), "blob digest disagrees") {
			t.Fatalf("Normalize() error = %v, want the digest disagreement refusal", err)
		}
	})
	t.Run("missing_blob", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-missing", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Fetch = func(blobID string) ([]byte, error) {
			return nil, errFixtureGone
		}
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted a missing blob")
		}
		if !strings.Contains(err.Error(), "cannot fetch blob") {
			t.Fatalf("Normalize() error = %v, want the fetch refusal", err)
		}
	})
	t.Run("manifest_closure_drift", func(t *testing.T) {
		first, _ := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-drift-a", `{"text":"a"}`),
			)},
		})
		second, secondStore := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-drift-b", `{"text":"b"}`),
			)},
		})
		request := normalizeRequest(t, second, secondStore, openTestStore(t))
		request.RawManifest = first.RawManifest
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted a capture whose raw manifest is not its sealed raw manifest")
		}
		if !strings.Contains(err.Error(), "links raw manifest") {
			t.Fatalf("Normalize() error = %v, want the closure refusal", err)
		}
	})
	t.Run("unmapped_extra_actor", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-extra", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Actors = map[string]string{"subagent:idle": fixtureSubActor}
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted an unreferenced actor mapping")
		}
		if !strings.Contains(err.Error(), "with no referencing record") {
			t.Fatalf("Normalize() error = %v, want the unreferenced-actor refusal", err)
		}
	})
	t.Run("main_actor_in_map", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-mainmap", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Actors = map[string]string{"main": fixtureSubActor}
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted main in the actor map")
		}
		if !strings.Contains(err.Error(), "carries main") {
			t.Fatalf("Normalize() error = %v, want the main-actor refusal", err)
		}
	})
	t.Run("missing_fetcher", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-nofetch", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.Fetch = nil
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted a nil blob fetcher")
		}
		if !strings.Contains(err.Error(), "requires a blob fetcher") {
			t.Fatalf("Normalize() error = %v, want the fetcher refusal", err)
		}
	})
	t.Run("missing_sink", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-nosink", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.CanonicalSink = nil
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted a nil canonical sink")
		}
		if !strings.Contains(err.Error(), "requires a canonical sink") {
			t.Fatalf("Normalize() error = %v, want the sink refusal", err)
		}
	})
	t.Run("bad_logical_session", func(t *testing.T) {
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-badid", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		request.LogicalSessionID = "not-a-uuid"
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted a non-UUIDv7 logical session")
		}
		if !strings.Contains(err.Error(), "is not a UUIDv7") {
			t.Fatalf("Normalize() error = %v, want the UUID refusal", err)
		}
	})
	t.Run("lone_surrogate_text", func(t *testing.T) {
		// Section 1.6: decoders MUST reject lone surrogate code
		// points before canonicalization. A lenient decode would
		// rewrite the escape to U+FFFD and seal exact anyway.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-sur-text", "message/user", "native", "none", "main", `{"text":"evt-sur-text A\ud800B"}`),
			)},
		}, "lone surrogate escape")
	})
	t.Run("lone_surrogate_envelope", func(t *testing.T) {
		// The envelope identity itself carries the lone escape:
		// admitting it would rewrite the evidence identity.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				`{"v":1,"native_event_id":"evt-sur-envelope-\ud800","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`,
			)},
		}, "lone surrogate escape")
	})
	t.Run("duplicate_envelope_member", func(t *testing.T) {
		// Section 1.6: duplicate keys are forbidden. A lenient
		// last-wins read would coerce this ambiguous record into
		// a known kind.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				`{"v":1,"native_event_id":"evt-dup-envelope","native_type":"frobnicate","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`,
			)},
		}, "duplicate member")
	})
	t.Run("duplicate_body_member", func(t *testing.T) {
		// A duplicated body member refuses instead of guessing
		// last-wins content.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-dup-body", "message/user", "native", "none", "main", `{"text":"A","text":"B"}`),
			)},
		}, "duplicate member")
	})
	t.Run("trailing_bytes", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-trailing", `{"text":"x"}`) + "x",
			)},
		}, "trailing data")
	})
	t.Run("string_usage_number", func(t *testing.T) {
		// A uint53 member is a JSON integer, never a JSON
		// string: "5" is not the number 5.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-use-str", "usage/report", "native", "none", "main", `{"input_tokens":"5","output_tokens":0}`),
			)},
		}, "is not a uint53")
	})
	t.Run("string_envelope_version", func(t *testing.T) {
		// Same class at the envelope: "1" is not version 1.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				`{"v":"1","native_event_id":"evt-vstr","native_type":"message/user","origin":"native","protection":"none","actor":"main","body":{"text":"x"}}`,
			)},
		}, "envelope version")
	})
	t.Run("encrypted_type_signed_protection", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-enc-signed", "reasoning/encrypted", "foreign", "signed", "main", `{"ciphertext":"x"}`),
			)},
		}, "declares reasoning/encrypted with protection")
	})
	t.Run("signed_type_encrypted_protection", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-signed-enc", "reasoning/signed", "foreign", "encrypted", "main", `{"ciphertext":"x"}`),
			)},
		}, "declares reasoning/signed with protection")
	})
	t.Run("bad_result_status", func(t *testing.T) {
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				call("call-pending", `{"call_id":"call-pending","tool_name":"write"}`),
				rec("evt-call-pending-result", "tool/result", "native", "none", "main", `{"call_id":"call-pending","status":"pending"}`),
			)},
		}, "outside ok|error")
	})
	t.Run("unknown_protected_plaintext_body", func(t *testing.T) {
		// An unknown type under protection still carries the
		// protected body shape: only exactly-ciphertext bodies
		// project opaque, anything else refuses rather than
		// guessing which bytes are safe to keep.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				rec("evt-unk-prot", "frobnicate", "foreign", "encrypted", "main", `{"mystery":true}`),
			)},
		}, "unknown member")
	})
	t.Run("unknown_type_without_body", func(t *testing.T) {
		// The body member is required for every record, including
		// unknown types: without it the record carries no
		// attributable payload and refuses instead of projecting
		// opaque.
		refuseMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				`{"v":1,"native_event_id":"evt-unk-nobody","native_type":"frobnicate","origin":"native","protection":"none","actor":"main"}`,
			)},
		}, `misses envelope member "body"`)
	})
	t.Run("blob_size_drift", func(t *testing.T) {
		// One extra byte: the size arm fires with its own
		// message before the digest arm is reached.
		capture, store := captureMembers(t, []fixtureMember{
			{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
				message("evt-sizedrift", `{"text":"x"}`),
			)},
		})
		request := normalizeRequest(t, capture, store, openTestStore(t))
		healthy := request.Fetch
		request.Fetch = func(blobID string) ([]byte, error) {
			payload, err := healthy(blobID)
			if err != nil {
				return nil, err
			}
			return append(payload, '\n'), nil
		}
		_, err := Normalize(request)
		if err == nil {
			t.Fatal("Normalize() admitted blob bytes whose size drifts from the raw count")
		}
		if !strings.Contains(err.Error(), "blob size") {
			t.Fatalf("Normalize() error = %v, want the size-arm refusal", err)
		}
	})
}

// TestNormalizeNonJSONLIncludedMemberRefuses pins the stated bound:
// every included non-unknown member is parsed as the JSONL record
// log, so a member whose bytes are not JSONL refuses the whole
// projection with the member and line named. Routing by member
// class is sibling scope.
func TestNormalizeNonJSONLIncludedMemberRefuses(t *testing.T) {
	session := []fixtureMember{
		{key: "store/session.jsonl", class: "durable_payload", content: jsonl(
			rec("evt-p3i", "message/user", "native", "none", "main", `{"text":"x"}`),
		)},
	}
	t.Run("binary durable index", func(t *testing.T) {
		members := append([]fixtureMember{
			{key: "store/index.bin", class: "durable_index_required", content: []byte{0x00, 0x01, 0x02}},
		}, session...)
		capture, store := captureMembers(t, members)
		_, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
		if err == nil {
			t.Fatal("Normalize() admitted a binary included member as record lines")
		}
		if !strings.Contains(err.Error(), "store/index.bin line 1") {
			t.Fatalf("Normalize() error = %v, want the member+line refusal", err)
		}
	})
	t.Run("text derived cache", func(t *testing.T) {
		members := append([]fixtureMember{
			{key: "store/cache.bin", class: "derived_cache_optional", content: []byte("not json\n")},
		}, session...)
		capture, store := captureMembers(t, members)
		_, err := Normalize(normalizeRequest(t, capture, store, openTestStore(t)))
		if err == nil {
			t.Fatal("Normalize() admitted a non-JSONL included member as record lines")
		}
		if !strings.Contains(err.Error(), "store/cache.bin line 1") {
			t.Fatalf("Normalize() error = %v, want the member+line refusal", err)
		}
	})
}
