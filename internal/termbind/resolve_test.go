package termbind

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

func tmuxTuple() EventTuple {
	return EventTuple{BackendID: terminalbackend.BuiltinTmux, ImplementationVersion: fixtureImpl, ProtocolVersion: fixtureProto}
}

func distinctDigest(index int) string {
	return fmt.Sprintf("sha256:%064x", index)
}

func TestResolveEvidenceAdmitsUniverse(t *testing.T) {
	t.Parallel()
	world := buildTmuxUniverse(t)
	resolved, err := ResolveEvidence(world.ids, tmuxTuple(), world.registry, universeMap(t, world), world.rawGeneration, world.now, world.verify)
	if err != nil {
		t.Fatalf("ResolveEvidence() error = %v", err)
	}
	if resolved.Manifest.ManifestID != world.manifestID {
		t.Fatalf("ManifestID = %q, want %q", resolved.Manifest.ManifestID, world.manifestID)
	}
	if resolved.Probe.ProbeID != world.probeID {
		t.Fatalf("ProbeID = %q, want %q", resolved.Probe.ProbeID, world.probeID)
	}
	if len(resolved.Evidence) != 1 || resolved.Evidence[0].EvidenceID != world.evidenceID {
		t.Fatalf("Evidence = %+v, want the one fixture evidence", resolved.Evidence)
	}
	if !resolved.Admitted.Has("durable_disconnect") {
		t.Fatalf("Admitted = %+v, want durable_disconnect", resolved.Admitted)
	}
}

func TestResolveEvidenceShapeBounds(t *testing.T) {
	t.Parallel()
	world := buildTmuxUniverse(t)
	universe := universeMap(t, world)
	resolve := func(ids []string) error {
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		return err
	}
	t.Run("empty refuses", func(t *testing.T) {
		t.Parallel()
		err := resolve(nil)
		if err == nil || !strings.Contains(err.Error(), "1..256") {
			t.Fatalf("ResolveEvidence(0) error = %v, want the 1..256 bound", err)
		}
	})
	t.Run("257 refuses", func(t *testing.T) {
		t.Parallel()
		ids := make([]string, 0, 257)
		for i := 0; i < 257; i++ {
			ids = append(ids, distinctDigest(i))
		}
		sort.Strings(ids)
		err := resolve(ids)
		if err == nil || !strings.Contains(err.Error(), "1..256") {
			t.Fatalf("ResolveEvidence(257) error = %v, want the 1..256 bound", err)
		}
	})
	t.Run("unsorted refuses", func(t *testing.T) {
		t.Parallel()
		ids := []string{world.manifestID, world.probeID}
		sort.Strings(ids)
		ids[0], ids[1] = ids[1], ids[0]
		err := resolve(ids)
		if err == nil || !strings.Contains(err.Error(), "sorted unique") {
			t.Fatalf("ResolveEvidence(unsorted) error = %v, want sorted unique", err)
		}
	})
	t.Run("duplicate refuses", func(t *testing.T) {
		t.Parallel()
		ids := []string{world.manifestID, world.manifestID, world.probeID}
		sort.Strings(ids)
		err := resolve(ids)
		if err == nil || !strings.Contains(err.Error(), "sorted unique") {
			t.Fatalf("ResolveEvidence(duplicate) error = %v, want sorted unique", err)
		}
	})
	t.Run("non-digest refuses", func(t *testing.T) {
		t.Parallel()
		if err := resolve([]string{"not-a-digest"}); err == nil || !strings.Contains(err.Error(), "not a digest") {
			t.Fatalf("ResolveEvidence(non-digest) error = %v, want digest grammar", err)
		}
	})
	t.Run("one passes shape to partition", func(t *testing.T) {
		t.Parallel()
		// A lone evidence ID passes the wire shape (1..256 admits one)
		// and refuses later at the manifest/probe partition: one digest
		// cannot name all three objects. The mismatch class proves the
		// shape arm admitted it.
		err := resolve([]string{world.evidenceID})
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("256 passes shape to lookup", func(t *testing.T) {
		t.Parallel()
		// 256 sorted unique digests pass the wire shape and refuse at
		// lookup: no universe holds them. The mismatch class proves the
		// shape arm admitted the upper bound.
		ids := make([]string, 0, 256)
		for i := 0; i < 256; i++ {
			ids = append(ids, distinctDigest(i))
		}
		sort.Strings(ids)
		err := resolve(ids)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
}

func TestResolveEvidenceRefusesForeignKinds(t *testing.T) {
	t.Parallel()
	targets := []struct {
		name  string
		bytes string
	}{
		{"native reference", `{"schema":"urn:ax:schema:terminal-native-reference","ref":"tmux-0"}`},
		{"generation string", `"generation-alpha"`},
		{"socket", `{"schema":"ax-terminal-socket","path":"/tmp/tmux-1000/default"}`},
		{"pipe", `{"schema":"ax-terminal-pipe","name":"ax-pane-1"}`},
		{"endpoint", `{"schema":"ax-terminal-endpoint","address":"10.0.0.9:2222"}`},
		{"token", `{"schema":"ax-terminal-token","token":"ax-token-7f3a"}`},
		{"credential", `{"schema":"ax-terminal-credential","secret":"hunter2"}`},
		{"terminal output", `"pane output bytes"`},
		{"pid", `12345`},
		{"handle", `{"schema":"ax-process-handle","handle":"0x4d2"}`},
		{"live-process fact", `{"schema":"ax-live-process-fact","alive":true}`},
	}
	for _, target := range targets {
		t.Run(target.name, func(t *testing.T) {
			t.Parallel()
			world := buildTmuxUniverse(t)
			hostileID := seedDigest(0xF0)
			universe := universeMap(t, world)
			universe[hostileID] = []byte(target.bytes)
			ids := []string{world.manifestID, world.probeID, hostileID}
			sort.Strings(ids)
			_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
			requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
			requireDetail(t, err, "evidence kind")
		})
	}
}

func TestResolveEvidenceRequiresManifestAndProbe(t *testing.T) {
	t.Parallel()
	world := buildTmuxUniverse(t)
	// Each parallel subtest resolves against its own universe copy: the
	// "second" cases extend the map, and sharing one map across parallel
	// subtests races the sibling lookups under -race.
	t.Run("missing manifest", func(t *testing.T) {
		t.Parallel()
		universe := universeMap(t, world)
		ids := []string{world.probeID, world.evidenceID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("missing probe", func(t *testing.T) {
		t.Parallel()
		universe := universeMap(t, world)
		ids := []string{world.manifestID, world.evidenceID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("second manifest", func(t *testing.T) {
		t.Parallel()
		universe := universeMap(t, world)
		extraID := seedDigest(0xE9)
		universe[extraID] = mustJSON(t, world.manifest)
		ids := []string{world.manifestID, world.probeID, world.evidenceID, extraID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("second probe", func(t *testing.T) {
		t.Parallel()
		universe := universeMap(t, world)
		extraID := seedDigest(0xE8)
		universe[extraID] = mustJSON(t, world.probe)
		ids := []string{world.manifestID, world.probeID, world.evidenceID, extraID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
}

func TestResolveEvidenceBindsEventTuple(t *testing.T) {
	t.Parallel()
	t.Run("foreign backend evidence", func(t *testing.T) {
		t.Parallel()
		tmux := buildTmuxUniverse(t)
		conpty := buildConptyUniverse(t)
		universe := universeMap(t, tmux)
		universe[conpty.evidenceID] = mustJSON(t, conpty.evidence)
		ids := []string{tmux.manifestID, tmux.probeID, conpty.evidenceID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), tmux.registry, universe, tmux.rawGeneration, tmux.now, tmux.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
		requireDetail(t, err, "evidence binding")
	})
	t.Run("protocol drift evidence", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		drifted := cloneDoc(world.evidence)
		drifted["protocol_version"] = "1.0.0"
		drifted["evidence_id"] = omitSelfIdentity(t, drifted, "evidence_id")
		driftedID := drifted["evidence_id"].(string)
		universe := universeMap(t, world)
		universe[driftedID] = mustJSON(t, drifted)
		ids := []string{world.manifestID, world.probeID, driftedID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
		requireDetail(t, err, "evidence binding")
	})
}

func TestResolveEvidenceRejectsSubstitution(t *testing.T) {
	t.Parallel()
	t.Run("evidence alias", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		// The universe returns the valid evidence bytes for an ID that is
		// not the embedded evidence ID: the substitution fails closed
		// instead of admitting document X under name Y.
		alias := seedDigest(0xA7)
		universe := universeMap(t, world)
		universe[alias] = mustJSON(t, world.evidence)
		ids := []string{world.manifestID, world.probeID, alias}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
	t.Run("probe alias", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		alias := seedDigest(0xA6)
		universe := universeMap(t, world)
		universe[alias] = mustJSON(t, world.probe)
		ids := []string{world.manifestID, alias, world.evidenceID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_manifest_probe_mismatch")
	})
}

func TestResolveEvidenceDrivesLandedAdmission(t *testing.T) {
	t.Parallel()
	t.Run("unevidenced true claim", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		probed := cloneDoc(world.probe)
		claims := append([]any(nil), probed["capability_claims"].([]any)...)
		claims = append(claims, map[string]any{
			"capability":            "headless_creation",
			"origin":                "probed",
			"value":                 true,
			"generation_variable":   true,
			"dependent_operations":  []any{"create"},
			"evidence_requirements": []any{"conformance_fixture", "runtime_probe"},
		})
		probed["capability_claims"] = claims
		probed["probe_id"] = omitSelfIdentity(t, probed, "probe_id")
		probedID := probed["probe_id"].(string)
		universe := universeMap(t, world)
		universe[probedID] = mustJSON(t, probed)
		ids := []string{world.manifestID, probedID, world.evidenceID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		if err == nil {
			t.Fatal("ResolveEvidence(unevidenced claim) succeeded, want the landed refusal")
		}
		if _, ok := err.(*terminalbackend.Error); !ok {
			t.Fatalf("error type = %T, want the landed *terminalbackend.Error", err)
		}
	})
	t.Run("registry drift", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		drifted := cloneDoc(world.manifest)
		drifted["protocol_versions"] = []any{"1.0.0", "1.2.0"}
		drifted["manifest_id"] = omitSelfIdentity(t, drifted, "manifest_id")
		driftedID := drifted["manifest_id"].(string)
		universe := universeMap(t, world)
		universe[driftedID] = mustJSON(t, drifted)
		ids := []string{driftedID, world.probeID, world.evidenceID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		requireCode(t, err, "terminal_backend_implementation_drift")
	})
	t.Run("forged signature", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		forged := cloneDoc(world.evidence)
		signature, ok := forged["attestation_signature"].(string)
		if !ok {
			t.Fatal("fixture evidence carries no signature")
		}
		tampered := signature[:len(signature)-1] + "A"
		if tampered == signature {
			tampered = signature[:len(signature)-1] + "B"
		}
		forged["attestation_signature"] = tampered
		forged["evidence_id"] = omitSelfIdentity(t, forged, "evidence_id")
		forgedID := forged["evidence_id"].(string)
		probe := cloneDoc(world.probe)
		probe["evidence_ids"] = []any{forgedID}
		probe["probe_id"] = omitSelfIdentity(t, probe, "probe_id")
		probeID := probe["probe_id"].(string)
		universe := mapUniverse{
			world.manifestID: mustJSON(t, world.manifest),
			probeID:          mustJSON(t, probe),
			forgedID:         mustJSON(t, forged),
		}
		ids := []string{world.manifestID, probeID, forgedID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		if err == nil {
			t.Fatal("ResolveEvidence(forged signature) succeeded, want the landed refusal")
		}
		if _, ok := err.(*terminalbackend.Error); !ok {
			t.Fatalf("error type = %T, want the landed *terminalbackend.Error", err)
		}
	})
	t.Run("expired evidence", func(t *testing.T) {
		t.Parallel()
		world := buildTmuxUniverse(t)
		stale := cloneDoc(world.evidence)
		stale["expires_at"] = "2025-06-02T00:00:00.000Z"
		stale["observed_at"] = "2025-06-01T00:00:00.000Z"
		stale["evidence_id"] = omitSelfIdentity(t, stale, "evidence_id")
		staleID := stale["evidence_id"].(string)
		probe := cloneDoc(world.probe)
		probe["evidence_ids"] = []any{staleID}
		probe["probe_id"] = omitSelfIdentity(t, probe, "probe_id")
		probeID := probe["probe_id"].(string)
		universe := mapUniverse{
			world.manifestID: mustJSON(t, world.manifest),
			probeID:          mustJSON(t, probe),
			staleID:          mustJSON(t, stale),
		}
		ids := []string{world.manifestID, probeID, staleID}
		sort.Strings(ids)
		_, err := ResolveEvidence(ids, tmuxTuple(), world.registry, universe, world.rawGeneration, world.now, world.verify)
		if err == nil {
			t.Fatal("ResolveEvidence(expired evidence) succeeded, want the landed refusal")
		}
		if _, ok := err.(*terminalbackend.Error); !ok {
			t.Fatalf("error type = %T, want the landed *terminalbackend.Error", err)
		}
	})
}

func TestResolvedObjectsCarryNoLiveFacts(t *testing.T) {
	t.Parallel()
	world := buildTmuxUniverse(t)
	resolved, err := ResolveEvidence(world.ids, tmuxTuple(), world.registry, universeMap(t, world), world.rawGeneration, world.now, world.verify)
	if err != nil {
		t.Fatal(err)
	}
	shown := []string{}
	for _, object := range []struct {
		name string
		raw  []byte
	}{
		{"manifest", mustJSON(t, world.manifest)},
		{"probe", mustJSON(t, world.probe)},
		{"evidence", mustJSON(t, world.evidence)},
	} {
		shown = append(shown, object.name+"="+string(object.raw))
	}
	joined := strings.Join(shown, "\n")
	for _, forbidden := range []string{world.rawGeneration, "native_reference", "terminal_instance_id", "attach_descriptor"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("resolved %q leaks %q", joined, forbidden)
		}
	}
	_ = resolved
}

func cloneDoc(object map[string]any) map[string]any {
	cloned := make(map[string]any, len(object))
	for name, member := range object {
		cloned[name] = member
	}
	return cloned
}
