# TASK-260830-1bipsa review probe — independent attack suite

Reviewer-authored, out-of-tree. Place under '<repo>/.temp/TASK-260830-1bipsa/probe/'
(package 'probe') and run: go test ./.temp/TASK-260830-1bipsa/probe/ -count=1 -v
It imports internal/cliresult, internal/axerror and internal/specdoc and drives
the real exported entry points. It was NOT committed; the reviewed tree is unchanged.

## alias_probe_test.go
```go
package probe

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/cliresult"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const (
	sessionID    = "0198f4c8-3e70-7a11-8a2b-1234567890ab"
	sourceHostID = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
	leaseID      = "bbbbbbbb-cccc-4ddd-8eee-ffffffffffff"
	checkpointID = "sha256:e051996f51f13ace4f5cdebe1e30fd26fd5fe104cfd6e6a7f9f1206ba3819656"
	timestamp    = "2026-08-19T04:30:00.000Z"
)

func mustUUID(t *testing.T, s string) scalar.UUIDv7 {
	t.Helper()
	u, err := scalar.ParseUUIDv7(s)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", s, err)
	}
	return u
}

func sessionSummary() map[string]any {
	return map[string]any{
		"session_id": sessionID, "name": "payments-api", "kind": "direct",
		"provider_id": "codex", "owner_host_id": sourceHostID,
		"owner_host_name": "workstation", "lease_epoch": json.Number("5"),
		"lease_id": leaseID, "local_role": "owner", "state": "running",
		"newest_checkpoint_id": checkpointID, "newest_checkpoint_created_at": timestamp,
		"workspace_status": "current",
		"capabilities": map[string]any{
			"native_resume": map[string]any{"status": "available", "enabled": true, "detail": ""},
		},
		"warnings": []any{},
	}
}

func listBody() map[string]any {
	return map[string]any{
		"sessions":             []any{sessionSummary()},
		"partial":              false,
		"unreachable_peer_ids": []any{},
	}
}

func newList(t *testing.T, body map[string]any, ext any) *cliresult.Result {
	t.Helper()
	res, err := cliresult.New(cliresult.Spec{
		Command: cliresult.CommandList, IDs: cliresult.NoIDs(), Body: body, Extensions: ext,
	})
	if err != nil {
		t.Fatalf("New(list): %v", err)
	}
	return res
}

// ATTACK 1: write through every container the CALLER still holds, after New returned.
func TestAttackMutateCallerContainerAfterNew(t *testing.T) {
	body := listBody()
	ext := map[string]any{"com.example.tag": map[string]any{"k": "v"}}
	res := newList(t, body, ext)

	before, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Reach every level the caller handed in.
	body["partial"] = "POISON"
	body["POISON_NEW"] = true
	body["sessions"].([]any)[0].(map[string]any)["state"] = "POISON"
	body["sessions"].([]any)[0].(map[string]any)["capabilities"].(map[string]any)["POISON"] = 1
	ext["com.example.tag"].(map[string]any)["k"] = "POISON"
	ext["POISON.key"] = "POISON"

	after, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal after: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("ALIASING DEFECT (caller container)\nbefore=%s\nafter =%s", before, after)
	}
	if strings.Contains(string(after), "POISON") {
		t.Fatalf("ALIASING DEFECT: poison on the wire: %s", after)
	}
	// And the poisoned object must still decode.
	if _, err := cliresult.Decode(cliresult.Version100, after); err != nil {
		t.Fatalf("own output no longer decodes: %v", err)
	}
}

// ATTACK 2: write through what the ACCESSORS hand out.
func TestAttackMutateAccessorResults(t *testing.T) {
	res := newList(t, listBody(), map[string]any{"com.example.tag": map[string]any{"k": "v"}})
	before, _ := json.Marshal(res)

	handed := res.Body()
	poisonDeep(handed)
	handed["POISON_TOP"] = true

	if ev, ok := res.Extension("com.example.tag"); ok {
		poisonDeep(ev)
	}

	after, _ := json.Marshal(res)
	if string(before) != string(after) {
		t.Fatalf("ALIASING DEFECT (accessor)\nbefore=%s\nafter =%s", before, after)
	}
	fresh := res.Body()
	if containsPoison(fresh) {
		t.Fatalf("ALIASING DEFECT: poison survived into a fresh Body(): %#v", fresh)
	}
	if ev, ok := res.Extension("com.example.tag"); ok && containsPoison(ev) {
		t.Fatalf("ALIASING DEFECT: poison survived into a fresh Extension(): %#v", ev)
	}
}

// ATTACK 3: same, on a DECODED result (different construction path).
func TestAttackMutateAccessorResultsOnDecoded(t *testing.T) {
	res := newList(t, listBody(), nil)
	wire, _ := json.Marshal(res)
	decoded, err := cliresult.Decode(cliresult.Version100, wire)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	before, _ := json.Marshal(decoded)
	handed := decoded.Body()
	poisonDeep(handed)
	after, _ := json.Marshal(decoded)
	if string(before) != string(after) {
		t.Fatalf("ALIASING DEFECT (decoded accessor)\nbefore=%s\nafter =%s", before, after)
	}
	if containsPoison(decoded.Body()) {
		t.Fatalf("ALIASING DEFECT: poison survived on decoded Body()")
	}
}

// ATTACK 4: IDs is a value type holding pointers. Can a caller alias through it?
func TestAttackMutateIDsAfterConstruction(t *testing.T) {
	op := mustUUID(t, "0198f4c8-17e0-78ff-8879-1234567890ab")
	ids := cliresult.NoIDs().WithOperation(op)
	got, ok := ids.Operation()
	if !ok || got.String() != op.String() {
		t.Fatalf("operation not recorded")
	}
	// WithOperation on a copy must not retroactively change the original.
	other := mustUUID(t, "0198f4c8-9999-7000-8111-1234567890ab")
	derived := ids.WithOperation(other)
	back, _ := ids.Operation()
	if back.String() != op.String() {
		t.Fatalf("ALIASING DEFECT: IDs.WithOperation mutated the receiver: %s", back.String())
	}
	d, _ := derived.Operation()
	if d.String() != other.String() {
		t.Fatalf("derived IDs wrong: %s", d.String())
	}
}

func poisonDeep(v any) {
	switch typed := v.(type) {
	case map[string]any:
		typed["POISON_NESTED"] = true
		for _, m := range typed {
			poisonDeep(m)
		}
	case []any:
		for i := range typed {
			poisonDeep(typed[i])
			typed[i] = "POISON_ELEM"
		}
	}
}

func containsPoison(v any) bool {
	switch typed := v.(type) {
	case map[string]any:
		for k, m := range typed {
			if strings.Contains(k, "POISON") || containsPoison(m) {
				return true
			}
		}
	case []any:
		for _, m := range typed {
			if containsPoison(m) {
				return true
			}
		}
	case string:
		return strings.Contains(typed, "POISON")
	}
	return false
}
```

## decode_probe_test.go
```go
package probe

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/cliresult"
)

// Round-trip every implemented command's own fixture through New -> wire -> Decode,
// then feed the DECODED body back into New. If Decode is weaker than New anywhere,
// the second New refuses something Decode admitted.
func TestAttackWriterReaderSymmetryAllCommands(t *testing.T) {
	cmds := cliresult.ImplementedCommands()
	t.Logf("implemented commands: %d", len(cmds))
	if len(cmds) != 18 {
		t.Fatalf("want 18 implemented commands, got %d: %v", len(cmds), cmds)
	}
	for _, c := range cmds {
		v, err := cliresult.VersionForCommand(c)
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if v != cliresult.Version100 {
			t.Fatalf("%s selects %s, expected 1.0.0 for an implemented body", c, v)
		}
	}
}

// ATTACK: every registered clone tag must select 2.0.0 and be REFUSED by New,
// never emitted with an unchecked body.
func TestAttackCloneTagsAreRefusedNotEmitted(t *testing.T) {
	clones := []cliresult.Command{
		"session.clone.adapters", "session.clone.doctor", "session.clone.list",
		"session.clone.inspect", "session.clone.plan", "session.clone.run",
		"session.clone.verify", "session.clone.open",
	}
	for _, c := range clones {
		got, err := cliresult.RegisteredVersionForCommand(c)
		if err != nil {
			t.Fatalf("%s not registered: %v", c, err)
		}
		if got != cliresult.Version200 {
			t.Fatalf("BYPASS: %s selects %s, Section 14.2 says 2.0.0", c, got)
		}
		if _, err := cliresult.VersionForCommand(c); !errors.Is(err, cliresult.ErrUnimplementedVersion) {
			t.Fatalf("%s: want ErrUnimplementedVersion, got %v", c, err)
		}
		// New must refuse it outright with ANY body.
		if _, err := cliresult.New(cliresult.Spec{Command: c, IDs: cliresult.NoIDs(), Body: map[string]any{"x": "y"}}); err == nil {
			t.Fatalf("BYPASS: New emitted an unchecked body for %s", c)
		}
	}
	// 3.0.0 and 4.0.0 tags: registered, unimplemented, never unknown.
	for _, c := range []cliresult.Command{"sessions.list", "terminal.backend.list"} {
		if _, err := cliresult.RegisteredVersionForCommand(c); err != nil {
			t.Fatalf("%s should be registered: %v", c, err)
		}
		if _, err := cliresult.VersionForCommand(c); !errors.Is(err, cliresult.ErrUnimplementedVersion) {
			t.Fatalf("%s: want ErrUnimplementedVersion, got %v", c, err)
		}
	}
	// A tag that exists nowhere is UNKNOWN, a different fact.
	if _, err := cliresult.RegisteredVersionForCommand("no.such.tag"); !errors.Is(err, cliresult.ErrUnknownCommand) {
		t.Fatalf("want ErrUnknownCommand, got %v", err)
	}
}

// ATTACK: mutate a valid wire document every way that must be refused.
func TestAttackDecodeRefusals(t *testing.T) {
	res := newList(t, listBody(), nil)
	wire, _ := json.Marshal(res)
	var base map[string]json.RawMessage
	if err := json.Unmarshal(wire, &base); err != nil {
		t.Fatalf("unmarshal base: %v", err)
	}
	clone := func() map[string]json.RawMessage {
		c := map[string]json.RawMessage{}
		for k, v := range base {
			c[k] = v
		}
		return c
	}
	mustRefuse := func(name string, doc map[string]json.RawMessage) {
		t.Helper()
		encoded, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if _, err := cliresult.Decode(cliresult.Version100, encoded); err == nil {
			t.Fatalf("BYPASS: Decode admitted %s: %s", name, encoded)
		}
	}

	// sanity: the untouched document decodes
	if _, err := cliresult.Decode(cliresult.Version100, wire); err != nil {
		t.Fatalf("baseline document does not decode: %v", err)
	}

	// ok = false
	d := clone()
	d["ok"] = json.RawMessage(`false`)
	mustRefuse("ok=false", d)

	// ok = truthy non-boolean
	for _, v := range []string{`1`, `"true"`, `null`, `{}`} {
		d = clone()
		d["ok"] = json.RawMessage(v)
		mustRefuse("ok="+v, d)
	}

	// wrong schema
	d = clone()
	d["schema"] = json.RawMessage(`"urn:ax:schema:error"`)
	mustRefuse("wrong schema", d)

	// unregistered / wrong-major versions
	for _, v := range []string{`"2.0.0"`, `"3.0.0"`, `"5.0.0"`, `"1.1.0"`, `"1.0"`, `"01.0.0"`, `"1.0.0-rc1"`, `""`} {
		d = clone()
		d["schema_version"] = json.RawMessage(v)
		mustRefuse("schema_version="+v, d)
	}

	// unknown top-level member
	d = clone()
	d["surprise"] = json.RawMessage(`1`)
	mustRefuse("unknown top-level member", d)

	// each declared member omitted
	for _, member := range []string{"schema", "schema_version", "command", "ok", "operation_id", "session_id", "body", "extensions"} {
		d = clone()
		delete(d, member)
		mustRefuse("missing "+member, d)
	}

	// identifier that the tag forbids, present
	d = clone()
	d["session_id"] = json.RawMessage(`"` + sessionID + `"`)
	mustRefuse("session_id non-null on list", d)
	d = clone()
	d["operation_id"] = json.RawMessage(`"0198f4c8-17e0-78ff-8879-1234567890ab"`)
	mustRefuse("operation_id non-null on list", d)

	// identifier that is not a UUIDv7
	d = clone()
	d["session_id"] = json.RawMessage(`"not-a-uuid"`)
	mustRefuse("session_id garbage", d)

	// body / extensions not objects
	for _, m := range []string{"body", "extensions"} {
		for _, v := range []string{`null`, `[]`, `"x"`, `1`} {
			d = clone()
			d[m] = json.RawMessage(v)
			mustRefuse(m+"="+v, d)
		}
	}

	// command tag swapped for one selecting another version
	d = clone()
	d["command"] = json.RawMessage(`"session.clone.list"`)
	mustRefuse("clone tag inside a 1.0.0 envelope", d)
	d = clone()
	d["command"] = json.RawMessage(`"no.such.tag"`)
	mustRefuse("unknown tag", d)

	// command tag swapped for another 1.0.0 tag whose body shape differs
	d = clone()
	d["command"] = json.RawMessage(`"doctor"`)
	mustRefuse("list body under the doctor tag", d)

	// unknown body member
	var body map[string]any
	_ = json.Unmarshal(base["body"], &body)
	body["surprise"] = "x"
	raw, _ := json.Marshal(body)
	d = clone()
	d["body"] = raw
	mustRefuse("unknown body member", d)

	// trailing content after the document
	if _, err := cliresult.Decode(cliresult.Version100, append(append([]byte{}, wire...), []byte(` {"x":1}`)...)); err == nil {
		t.Fatalf("BYPASS: trailing content admitted")
	}
	// duplicate top-level member
	dup := `{"schema":"urn:ax:schema:cli-result","schema":"urn:ax:schema:cli-result"}`
	if _, err := cliresult.Decode(cliresult.Version100, []byte(dup)); err == nil {
		t.Fatalf("BYPASS: duplicate member admitted")
	}
	// empty / garbage
	for _, v := range []string{``, `{}`, `[]`, `null`, `"x"`, `{`} {
		if _, err := cliresult.Decode(cliresult.Version100, []byte(v)); err == nil {
			t.Fatalf("BYPASS: Decode admitted %q", v)
		}
	}
}

// ATTACK: Decode bound to a version this repo does not build.
func TestAttackDecodeUnimplementedVersion(t *testing.T) {
	res := newList(t, listBody(), nil)
	wire, _ := json.Marshal(res)
	for _, v := range []cliresult.Version{"3.0.0", "4.0.0", "9.9.9", "", "1.0"} {
		if _, err := cliresult.Decode(v, wire); err == nil {
			t.Fatalf("BYPASS: Decode accepted expected-version %q", v)
		}
	}
	// bound to 2.0.0, given a 1.0.0 document => unsupported MAJOR
	if _, err := cliresult.Decode(cliresult.Version200, wire); !errors.Is(err, cliresult.ErrUnsupportedMajor) {
		t.Fatalf("want ErrUnsupportedMajor, got %v", err)
	}
}

// ATTACK: the exit-code mapping. Emit must return exactly the registry status.
func TestAttackExitStatusMapping(t *testing.T) {
	codes, codesErr := axerror.CodesFor(axerror.Version100)
	if codesErr != nil {
		t.Fatalf("CodesFor: %v", codesErr)
	}
	if len(codes) == 0 {
		t.Fatalf("no registered codes")
	}
	seen := map[int]bool{}
	for _, code := range codes {
		failure, err := axerror.New(axerror.Spec{
			Version: axerror.Version100, Code: code, Message: "probe",
			IDs: axerror.NoIDs(), Details: axerror.Details{},
		})
		if err != nil {
			t.Logf("skip %s: not constructible bare: %v", code, err)
			continue
		}
		want := failure.ExitCode()
		seen[want] = true
		got, err := cliresult.ExitStatus(cliresult.Outcome{Failure: failure})
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if got != want {
			t.Fatalf("MAPPING DEFECT: code %s registry exit %d, cliresult returned %d", code, want, got)
		}
		// and through the real emission path
		var out, errb strings.Builder
		inv, _ := cliresult.ParseCommonFlags(cliresult.SurfaceList, []string{"--json"})
		em, _ := cliresult.NewEmitter(inv, cliresult.Streams{Stdout: &out, Stderr: &errb})
		status, emitErr := em.Emit(cliresult.Outcome{Failure: failure})
		if emitErr != nil {
			t.Fatalf("%s emit: %v", code, emitErr)
		}
		if status != want {
			t.Fatalf("MAPPING DEFECT via Emit: code %s want %d got %d", code, want, status)
		}
		if !strings.Contains(out.String(), `"schema":"urn:ax:schema:error"`) {
			t.Fatalf("failure did not emit a Structured Error on stdout: %s", out.String())
		}
		if strings.Contains(out.String(), `"ok"`) {
			t.Fatalf("BYPASS: a failure emitted a CLI Result with ok: %s", out.String())
		}
		if !strings.Contains(out.String(), fmt.Sprintf(`"exit_code":%d`, want)) {
			t.Fatalf("MAPPING DEFECT: emitted document exit_code disagrees with process status %d: %s", want, out.String())
		}
	}
	if seen[0] {
		t.Fatalf("MAPPING DEFECT: a failure code maps to exit 0")
	}
	fmt.Fprintf(&strings.Builder{}, "")
	t.Logf("distinct failure exit statuses reached: %d", len(seen))
	// success is always 0
	res := newList(t, listBody(), nil)
	if s, err := cliresult.ExitStatus(cliresult.Outcome{Result: res}); err != nil || s != 0 {
		t.Fatalf("success status = %d, %v", s, err)
	}
}

// ATTACK: text mode must put a failure on stderr and a success on stdout.
func TestAttackTextModeStreamSplit(t *testing.T) {
	failure, err := axerror.New(axerror.Spec{
		Version: axerror.Version100, Code: "invalid_arguments", Message: "probe",
		IDs: axerror.NoIDs(), Details: axerror.Details{},
	})
	if err != nil {
		t.Fatalf("axerror: %v", err)
	}
	var out, errb strings.Builder
	inv, _ := cliresult.ParseCommonFlags(cliresult.SurfaceList, nil)
	em, _ := cliresult.NewEmitter(inv, cliresult.Streams{Stdout: &out, Stderr: &errb})
	status, emitErr := em.Emit(cliresult.Outcome{Failure: failure, Rendered: "it broke"})
	if emitErr != nil {
		t.Fatalf("emit: %v", emitErr)
	}
	if status != failure.ExitCode() {
		t.Fatalf("status %d want %d", status, failure.ExitCode())
	}
	if out.Len() != 0 {
		t.Fatalf("BYPASS: text-mode failure reached stdout: %q", out.String())
	}
	if !strings.Contains(errb.String(), "it broke") {
		t.Fatalf("failure rendering missing from stderr: %q", errb.String())
	}
}

// PROBE: §17.2 rule 1 ordering. A future-major document that also carries a new
// top-level member is refused (fail-closed either way) -- but which fact does the
// reader report? Recorded for the compatibility leaf, not asserted as a defect.
func TestProbeFutureMajorWithNewMemberReportsWhich(t *testing.T) {
	res := newList(t, listBody(), nil)
	wire, _ := json.Marshal(res)
	var doc map[string]json.RawMessage
	_ = json.Unmarshal(wire, &doc)
	doc["schema_version"] = json.RawMessage(`"2.0.0"`)
	doc["new_in_v2"] = json.RawMessage(`{"a":1}`)
	encoded, _ := json.Marshal(doc)

	_, err := cliresult.Decode(cliresult.Version100, encoded)
	if err == nil {
		t.Fatalf("BYPASS: a future-major document with a new member was admitted")
	}
	t.Logf("future-major + new member -> %v", err)
	t.Logf("errors.Is(ErrUnsupportedMajor) = %v", errors.Is(err, cliresult.ErrUnsupportedMajor))

	// Without the extra member, the major refusal is reported cleanly.
	delete(doc, "new_in_v2")
	encoded, _ = json.Marshal(doc)
	_, err = cliresult.Decode(cliresult.Version100, encoded)
	if !errors.Is(err, cliresult.ErrUnsupportedMajor) {
		t.Fatalf("plain future-major document: want ErrUnsupportedMajor, got %v", err)
	}
	t.Logf("future-major alone -> ErrUnsupportedMajor (correct)")
}
```

## gate_probe_test.go
```go
package probe

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/cliresult"
)

// ATTACK: can --yes alone get past a missing expectation flag?
func TestAttackYesAloneBypassesExpectationFlags(t *testing.T) {
	for _, op := range cliresult.DestructiveOperations() {
		expected, err := cliresult.ExpectationFlags(op)
		if err != nil {
			t.Fatalf("flags(%s): %v", op, err)
		}
		if len(expected) == 0 {
			t.Fatalf("operation %s has no expectation flags to bypass", op)
		}
		// Drop each expectation flag in turn, keep --yes, non-interactive.
		for _, drop := range expected {
			var supplied []string
			for _, f := range expected {
				if f != drop {
					supplied = append(supplied, f)
				}
			}
			inv, failure := cliresult.ParseCommonFlags(cliresult.SurfaceTakeover,
				[]string{"--yes", "--non-interactive"})
			if failure != nil {
				t.Fatalf("parse: %v", failure)
			}
			_, refusal := cliresult.RequireConfirmation(op, inv, supplied)
			if refusal == nil {
				t.Fatalf("BYPASS: %s proceeded without %s while --yes was present", op, drop)
			}
			if refusal.ExitCode() != 2 {
				t.Fatalf("%s missing %s: exit %d, want 2 (invalid_arguments)", op, drop, refusal.ExitCode())
			}
		}
		// Supplying ALL expectation flags but no --yes, non-interactive => must still refuse.
		inv, failure := cliresult.ParseCommonFlags(cliresult.SurfaceTakeover, []string{"--non-interactive"})
		if failure != nil {
			t.Fatalf("parse: %v", failure)
		}
		_, refusal := cliresult.RequireConfirmation(op, inv, expected)
		if refusal == nil {
			t.Fatalf("BYPASS: %s proceeded non-interactively without --yes", op)
		}
		if refusal.ExitCode() != 16 {
			t.Fatalf("%s without --yes: exit %d, want 16 (confirmation_required)", op, refusal.ExitCode())
		}
		// Full set + --yes non-interactive => proceeds, no prompt.
		inv, _ = cliresult.ParseCommonFlags(cliresult.SurfaceTakeover, []string{"--non-interactive", "--yes"})
		conf, refusal := cliresult.RequireConfirmation(op, inv, expected)
		if refusal != nil {
			t.Fatalf("%s with full set + --yes refused: %v", op, refusal)
		}
		if conf.PromptRequired {
			t.Fatalf("%s non-interactive must not require a prompt", op)
		}
		// Interactive with the full set => MUST prompt.
		inv, _ = cliresult.ParseCommonFlags(cliresult.SurfaceTakeover, nil)
		conf, refusal = cliresult.RequireConfirmation(op, inv, expected)
		if refusal != nil {
			t.Fatalf("%s interactive refused: %v", op, refusal)
		}
		if !conf.PromptRequired {
			t.Fatalf("BYPASS: %s interactive did not require a prompt", op)
		}
		// ATTACK: interactive + --yes must STILL require the expectation flags.
		inv, _ = cliresult.ParseCommonFlags(cliresult.SurfaceTakeover, []string{"--yes"})
		_, refusal = cliresult.RequireConfirmation(op, inv, nil)
		if refusal == nil {
			t.Fatalf("BYPASS: %s interactive with --yes and NO expectation flags proceeded", op)
		}
		// ATTACK: forge extra unrelated flags to satisfy the set.
		_, refusal = cliresult.RequireConfirmation(op, inv, []string{"--force", "--yes", "--whatever"})
		if refusal == nil {
			t.Fatalf("BYPASS: %s accepted unrelated flags as expectation flags", op)
		}
	}
}

// ATTACK: nil invocation must not be read as "interactive, so just prompt".
func TestAttackNilInvocationConfirmation(t *testing.T) {
	_, refusal := cliresult.RequireConfirmation(cliresult.OperationForceTakeover, nil,
		[]string{"--expect-epoch", "--expect-owner"})
	if refusal == nil {
		t.Fatalf("BYPASS: nil invocation was admitted")
	}
}

// ATTACK: --yes on a surface with no documented confirmation.
func TestAttackYesOnUndocumentedSurface(t *testing.T) {
	var accepted []cliresult.SurfaceCommand
	for _, s := range cliresult.Surfaces() {
		allowed, err := cliresult.AcceptsYes(s)
		if err != nil {
			t.Fatalf("AcceptsYes(%s): %v", s, err)
		}
		inv, failure := cliresult.ParseCommonFlags(s, []string{"--yes"})
		if allowed {
			if failure != nil {
				t.Fatalf("%s documents a confirmation but rejected --yes: %v", s, failure)
			}
			if !inv.Yes() {
				t.Fatalf("%s accepted --yes but Yes() is false", s)
			}
			continue
		}
		if failure == nil {
			accepted = append(accepted, s)
			continue
		}
		if failure.ExitCode() != 2 {
			t.Fatalf("%s rejected --yes with exit %d, want 2", s, failure.ExitCode())
		}
	}
	if len(accepted) > 0 {
		t.Fatalf("BYPASS: surfaces without a documented confirmation accepted --yes: %v", accepted)
	}
	// Exactly three surfaces may take --yes.
	var yesSurfaces []cliresult.SurfaceCommand
	for _, s := range cliresult.Surfaces() {
		if ok, _ := cliresult.AcceptsYes(s); ok {
			yesSurfaces = append(yesSurfaces, s)
		}
	}
	t.Logf("surfaces accepting --yes: %v", yesSurfaces)
	if len(yesSurfaces) != 3 {
		t.Fatalf("want 3 confirmation surfaces, got %d: %v", len(yesSurfaces), yesSurfaces)
	}
}

// ATTACK: rpc serve MUST reject --json. Try every spelling.
func TestAttackJSONOnRPCServe(t *testing.T) {
	for _, argv := range [][]string{
		{"--json"},
		{"--stdio", "--json"},
		{"--json=true"},
		{"--non-interactive", "--json"},
	} {
		_, failure := cliresult.ParseCommonFlags(cliresult.SurfaceRPCServe, argv)
		if failure == nil {
			t.Fatalf("BYPASS: rpc serve accepted %v", argv)
		}
	}
	// And its Mode can never be json.
	inv, failure := cliresult.ParseCommonFlags(cliresult.SurfaceRPCServe, []string{"--stdio"})
	if failure != nil {
		t.Fatalf("rpc serve --stdio refused: %v", failure)
	}
	if inv.Mode() != cliresult.ModeText {
		t.Fatalf("rpc serve mode is %s", inv.Mode())
	}
}

// ATTACK: unknown surface must not fall through to a permissive default.
func TestAttackUnknownSurface(t *testing.T) {
	if _, failure := cliresult.ParseCommonFlags("totally-made-up", []string{"--json"}); failure == nil {
		t.Fatalf("BYPASS: unknown surface parsed")
	}
	if _, err := cliresult.AcceptsYes("totally-made-up"); !errors.Is(err, cliresult.ErrUnknownSurface) {
		t.Fatalf("AcceptsYes(unknown) = %v", err)
	}
	if _, err := cliresult.ExpectationFlags("made-up-op"); err == nil {
		t.Fatalf("BYPASS: unknown destructive operation returned flags")
	}
}

// ATTACK: stream discipline. Two documents on stdout, human text in JSON mode,
// prompt under --non-interactive, progress on a non-TTY.
func TestAttackStreamDiscipline(t *testing.T) {
	res := newList(t, listBody(), nil)

	// two emissions in JSON mode
	var out, errbuf bytes.Buffer
	inv, _ := cliresult.ParseCommonFlags(cliresult.SurfaceList, []string{"--json"})
	em, err := cliresult.NewEmitter(inv, cliresult.Streams{Stdout: &out, Stderr: &errbuf})
	if err != nil {
		t.Fatalf("emitter: %v", err)
	}
	if _, err := em.Emit(cliresult.Outcome{Result: res}); err != nil {
		t.Fatalf("first emit: %v", err)
	}
	if _, err := em.Emit(cliresult.Outcome{Result: res}); err == nil {
		t.Fatalf("BYPASS: stdout took a second document")
	}
	if n := strings.Count(strings.TrimSpace(out.String()), "\n"); n != 0 {
		t.Fatalf("BYPASS: stdout carries %d extra lines: %q", n+1, out.String())
	}
	if errbuf.Len() != 0 {
		t.Fatalf("JSON mode wrote to stderr: %q", errbuf.String())
	}
	var doc any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("stdout is not one JSON document: %v", err)
	}

	// human text in JSON mode must be refused
	out.Reset()
	em2, _ := cliresult.NewEmitter(inv, cliresult.Streams{Stdout: &out, Stderr: &errbuf})
	if _, err := em2.Emit(cliresult.Outcome{Result: res, Rendered: "human"}); err == nil {
		t.Fatalf("BYPASS: JSON mode emitted human text")
	}

	// text mode with no rendering must be refused
	out.Reset()
	invText, _ := cliresult.ParseCommonFlags(cliresult.SurfaceList, nil)
	em3, _ := cliresult.NewEmitter(invText, cliresult.Streams{Stdout: &out, Stderr: &errbuf})
	if _, err := em3.Emit(cliresult.Outcome{Result: res}); err == nil {
		t.Fatalf("BYPASS: text mode emitted with no human rendering")
	}

	// outcome with both / neither
	out.Reset()
	em4, _ := cliresult.NewEmitter(inv, cliresult.Streams{Stdout: &out, Stderr: &errbuf})
	if _, err := em4.Emit(cliresult.Outcome{}); err == nil {
		t.Fatalf("BYPASS: empty outcome emitted")
	}

	// prompt under --non-interactive
	invNI, _ := cliresult.ParseCommonFlags(cliresult.SurfaceList, []string{"--non-interactive"})
	errbuf.Reset()
	em5, _ := cliresult.NewEmitter(invNI, cliresult.Streams{Stdout: &out, Stderr: &errbuf})
	if err := em5.Prompt("continue?"); err == nil {
		t.Fatalf("BYPASS: prompt under --non-interactive")
	}
	if errbuf.Len() != 0 {
		t.Fatalf("refused prompt still wrote to stderr: %q", errbuf.String())
	}

	// progress on a non-TTY is dropped and says so
	wrote, err := em5.Progress("50%")
	if err != nil || wrote {
		t.Fatalf("progress on non-TTY: wrote=%v err=%v", wrote, err)
	}
	if errbuf.Len() != 0 {
		t.Fatalf("BYPASS: progress reached a non-TTY stderr: %q", errbuf.String())
	}
	// on a TTY it is written
	errbuf.Reset()
	em6, _ := cliresult.NewEmitter(invNI, cliresult.Streams{Stdout: &out, Stderr: &errbuf, StderrIsTTY: true})
	wrote, err = em6.Progress("50%")
	if err != nil || !wrote {
		t.Fatalf("progress on TTY: wrote=%v err=%v", wrote, err)
	}
	// logs never reach stdout, in either mode
	out.Reset()
	if err := em6.Log("diag"); err != nil {
		t.Fatalf("log: %v", err)
	}
	if out.Len() != 0 {
		t.Fatalf("BYPASS: a log reached stdout: %q", out.String())
	}
}
```

## spec_probe_test.go
```go
package probe

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/cliresult"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

type clause struct {
	ID       string   `json:"id"`
	Line     int      `json:"line"`
	Excerpt  string   `json:"excerpt"`
	Cases    []string `json:"acceptance_cases"`
}

type binding struct {
	Kind    string   `json:"kind"`
	Keys    []string `json:"keys"`
	Clauses []clause `json:"clauses"`
	Gap     string   `json:"gap"`
}

type ownershipFile struct {
	AcceptanceCases []struct {
		ID string `json:"id"`
	} `json:"acceptance_cases"`
	Ownership []binding `json:"ownership"`
}

func loadOwnership(t *testing.T) ownershipFile {
	t.Helper()
	raw, err := os.ReadFile("../../../internal/traceability/ownership.v0.5.0.json")
	if err != nil {
		t.Fatalf("read ownership: %v", err)
	}
	var f ownershipFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse ownership: %v", err)
	}
	return f
}

// ATTACK: every clause of section:14.2 must quote the pinned document VERBATIM
// at the exact line it declares, and that line must live inside section 14.2.
func TestAttackClauseCitationsAreReal(t *testing.T) {
	doc, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	f := loadOwnership(t)
	checked := 0
	for _, b := range f.Ownership {
		var is142 bool
		for _, k := range b.Keys {
			if k == "section:14.2" {
				is142 = true
			}
		}
		if !is142 {
			continue
		}
		for _, c := range b.Clauses {
			line, ok := doc.Line(c.Line)
			if !ok {
				t.Fatalf("%s: SPEC.md has no line %d", c.ID, c.Line)
			}
			if !strings.Contains(line, c.Excerpt) {
				t.Fatalf("FABRICATED CITATION %s: line %d is\n  %q\nexcerpt claims\n  %q",
					c.ID, c.Line, line, c.Excerpt)
			}
			section, ok := doc.SectionID(c.Line)
			if !ok || section != "14.2" {
				t.Fatalf("WRONG SECTION %s: line %d is in section %q, not 14.2", c.ID, c.Line, section)
			}
			// the excerpt must actually carry an RFC 2119 keyword
			if !strings.Contains(c.Excerpt, "MUST") && !strings.Contains(c.Excerpt, "SHOULD") &&
				!strings.Contains(c.Excerpt, "MAY") && !strings.Contains(c.Excerpt, "REQUIRED") {
				t.Logf("NOTE %s carries no RFC 2119 keyword in its excerpt: %q", c.ID, c.Excerpt)
			}
			t.Logf("%s line %d section %s OK: %q", c.ID, c.Line, section, c.Excerpt)
			checked++
		}
		t.Logf("gap: %s", b.Gap)
	}
	if checked != 8 {
		t.Fatalf("expected 8 claimed 14.2 clauses, checked %d", checked)
	}
}

// ATTACK: the acceptance cases each clause names must actually be declared.
func TestAttackAcceptanceCasesExist(t *testing.T) {
	f := loadOwnership(t)
	declared := map[string]bool{}
	for _, c := range f.AcceptanceCases {
		declared[c.ID] = true
	}
	for _, b := range f.Ownership {
		for _, c := range b.Clauses {
			if len(c.Cases) == 0 {
				t.Fatalf("clause %s names no acceptance case", c.ID)
			}
			for _, name := range c.Cases {
				if !declared[name] {
					t.Fatalf("clause %s names undeclared acceptance case %q", c.ID, name)
				}
			}
		}
	}
}

// ATTACK: the Section 15.2 exit table really has eighteen body rows, read out of
// the pinned document, and every one of them is in the axerror registry.
func TestAttackExitTableRowCountIsDerived(t *testing.T) {
	doc, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	rows := 0
	for n := 1; n <= doc.LineCount(); n++ {
		section, ok := doc.SectionID(n)
		if !ok || section != "15.2" {
			continue
		}
		if _, isRow := doc.TableRowAt(n); isRow {
			rows++
		}
	}
	t.Logf("Section 15.2 body rows measured from the pinned document: %d", rows)
	if rows != 18 {
		t.Fatalf("COUNT DEFECT: measured %d body rows in Section 15.2, the repository claims 18", rows)
	}
}

// ATTACK: every ratio the README and the notes state, measured independently.
func TestAttackStatedRatiosAreReal(t *testing.T) {
	t.Logf("registered command tags: %d", len(cliresultCommands()))
	t.Logf("implemented command tags: %d", len(cliresultImplemented()))
	t.Logf("registered versions: %d", len(cliresultVersions()))
	t.Logf("implemented versions: %d", len(cliresultImplementedVersions()))
	t.Logf("surfaces: %d", len(cliresultSurfaces()))
	t.Logf("user surfaces: %d", len(cliresultUserSurfaces()))
	if got := len(cliresultCommands()); got != 44 {
		t.Fatalf("README claims 44 registered tags, measured %d", got)
	}
	if got := len(cliresultImplemented()); got != 18 {
		t.Fatalf("README claims 18 built tags, measured %d", got)
	}
	if got := len(cliresultVersions()); got != 4 {
		t.Fatalf("README claims 4 registered versions, measured %d", got)
	}
	if got := len(cliresultImplementedVersions()); got != 2 {
		t.Fatalf("README claims 2 built versions, measured %d", got)
	}
	if got := len(cliresultSurfaces()); got != 31 {
		t.Fatalf("README claims 31 surfaces, measured %d", got)
	}
	if got := len(cliresultUserSurfaces()); got != 29 {
		t.Fatalf("README claims 29 user surfaces, measured %d", got)
	}
}

func cliresultCommands() []cliresult.Command   { return cliresult.Commands() }
func cliresultImplemented() []cliresult.Command { return cliresult.ImplementedCommands() }
func cliresultVersions() []cliresult.Version   { return cliresult.Versions() }
func cliresultImplementedVersions() []cliresult.Version { return cliresult.ImplementedVersions() }
func cliresultSurfaces() []cliresult.SurfaceCommand     { return cliresult.Surfaces() }
func cliresultUserSurfaces() []cliresult.SurfaceCommand { return cliresult.UserSurfaces() }
```

## takeover_probe_test.go
```go
package probe

import (
	"encoding/json"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/cliresult"
)

const (
	opID   = "0198f4c8-17e0-78ff-8879-1234567890ab"
	destID = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
)

func takeoverBody(adopted, resumed bool) map[string]any {
	return map[string]any{
		"mode": "force", "workspace_mode": "whole_group",
		"destination_host_id": destID, "source_host_id": sourceHostID,
		"affected_session_ids": []any{sessionID},
		"lease_epoch":          json.Number("5"), "lease_id": leaseID,
		"checkpoint_id": checkpointID, "state": "running",
		"materialized": true, "adopted": adopted, "resumed": resumed,
		"warnings": []any{},
	}
}

func newTakeover(t *testing.T, kind cliresult.SessionKind, adopted, resumed bool) (*cliresult.Result, error) {
	t.Helper()
	return cliresult.New(cliresult.Spec{
		Command:     cliresult.CommandTakeover,
		IDs:         cliresult.NoIDs().WithOperation(mustUUID(t, opID)),
		Body:        takeoverBody(adopted, resumed),
		SessionKind: kind,
	})
}

// ATTACK: Section 14.2 — "For a task-board takeover, adopted MUST be true before
// resumed can be true; for a direct takeover it MUST be false."
func TestAttackTakeoverAdoptionRule(t *testing.T) {
	cases := []struct {
		kind             cliresult.SessionKind
		adopted, resumed bool
		admit            bool
	}{
		// task-board: resumed implies adopted
		{cliresult.KindTaskBoard, true, true, true},
		{cliresult.KindTaskBoard, false, true, false}, // THE violation
		{cliresult.KindTaskBoard, true, false, true},
		{cliresult.KindTaskBoard, false, false, true},
		// direct: adopted MUST be false, always
		{cliresult.KindDirect, false, true, true},
		{cliresult.KindDirect, true, true, false}, // THE violation
		{cliresult.KindDirect, true, false, false},
		{cliresult.KindDirect, false, false, true},
	}
	for _, c := range cases {
		_, err := newTakeover(t, c.kind, c.adopted, c.resumed)
		if c.admit && err != nil {
			t.Fatalf("kind=%s adopted=%v resumed=%v refused: %v", c.kind, c.adopted, c.resumed, err)
		}
		if !c.admit && err == nil {
			t.Fatalf("BYPASS: kind=%s adopted=%v resumed=%v was admitted", c.kind, c.adopted, c.resumed)
		}
	}
}

// ATTACK: can a caller skip the rule by omitting or forging the session kind?
func TestAttackTakeoverKindCannotBeSkipped(t *testing.T) {
	// no kind at all => refused, not silently skipped
	if _, err := newTakeover(t, "", true, true); err == nil {
		t.Fatalf("BYPASS: takeover built with no session kind")
	}
	// a forged kind => refused
	for _, k := range []cliresult.SessionKind{"taskboard", "TASK_BOARD", "direct ", "other", "0"} {
		if _, err := newTakeover(t, k, false, true); err == nil {
			t.Fatalf("BYPASS: takeover built with forged kind %q", k)
		}
	}
	// a kind on a command that takes none => refused
	if _, err := cliresult.New(cliresult.Spec{
		Command: cliresult.CommandList, IDs: cliresult.NoIDs(),
		Body: listBody(), SessionKind: cliresult.KindDirect,
	}); err == nil {
		t.Fatalf("BYPASS: list accepted a session kind")
	}
}

// ATTACK: Decode must NOT silently pass the rule, and VerifyTakeoverAdoption
// must catch a violating document that Decode admitted.
func TestAttackDecodeDoesNotSilentlyPassAdoption(t *testing.T) {
	// Build a LEGAL task-board takeover, then rewrite the wire to violate it.
	res, err := newTakeover(t, cliresult.KindTaskBoard, true, true)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	wire, _ := json.Marshal(res)
	var doc map[string]json.RawMessage
	_ = json.Unmarshal(wire, &doc)
	var body map[string]any
	_ = json.Unmarshal(doc["body"], &body)
	body["adopted"] = false // resumed stays true -> violates the task-board rule
	doc["body"], _ = json.Marshal(body)
	violating, _ := json.Marshal(doc)

	decoded, err := cliresult.Decode(cliresult.Version100, violating)
	if err != nil {
		t.Logf("Decode already refuses the violating document: %v", err)
		return
	}
	// Decode admitted it (documented: the document carries no kind). The
	// explicit verification MUST then catch it for the kind that forbids it.
	if err := decoded.VerifyTakeoverAdoption(cliresult.KindTaskBoard); err == nil {
		t.Fatalf("BYPASS: VerifyTakeoverAdoption(task_board) admitted adopted=false resumed=true")
	}
	// and a bogus kind must not be a way to skip the check
	for _, k := range []cliresult.SessionKind{"", "nope", "TASK_BOARD"} {
		if err := decoded.VerifyTakeoverAdoption(k); err == nil {
			t.Fatalf("BYPASS: VerifyTakeoverAdoption(%q) returned nil", k)
		}
	}
	// direct forbids adopted=true; this document has adopted=false so it passes
	if err := decoded.VerifyTakeoverAdoption(cliresult.KindDirect); err != nil {
		t.Fatalf("direct takeover with adopted=false refused: %v", err)
	}
}
```

