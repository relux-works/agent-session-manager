package sessadapter

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// This file pins every rev3-B5 contract string maximum and array cap
// at the edge (review rev3 B5): each bound is driven at the limit
// (admitted) and one past it (refused), with the same for one below
// the minimum. A witness far outside the range proves the arm
// exists, not where the bound sits; a mutant moving any bound by
// exactly one admits exactly one member of the reject class and
// reddens its row here. Lengths are characters: the repeat rune is
// ASCII, so bytes equal runes.

// quotedRepeat renders a JSON string literal of exactly n characters.
func quotedRepeat(n int) string {
	return strconv.Quote(strings.Repeat("a", n))
}

// stringBoundRow is one checkStringBounds maximum pinned at the
// production entry: drive maps a JSON string literal to the entry's
// verdict, member/text name the bound refusal for requireRefusal.
type stringBoundRow struct {
	name   string
	min    int
	max    int
	member string
	text   string
	drive  func(t *testing.T, literal string) error
}

// TestStringBoundEdges drives every survivor string maximum at
// max-1/min/max/max+1 through its production entry point.
func TestStringBoundEdges(t *testing.T) {
	rows := []stringBoundRow{
		{
			name: "manifest environment_version_range", min: 1, max: 256,
			member: "environment_version_range", text: "string[1..256]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeManifest(mutateMember(t, []byte(fixtureManifestJSON()), "environment_version_range", literal))
				return err
			},
		},
		{
			name: "source native_session_id", min: 1, max: 512,
			member: "native_session_id", text: "string[1..512]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeSourceSelector([]byte(`{"native_session_id":` + literal + `,"logical_workspace_id":null,"opaque_source_ref":null}`))
				return err
			},
		},
		{
			name: "source opaque_source_ref", min: 1, max: 512,
			member: "opaque_source_ref", text: "string[1..512]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeSourceSelector([]byte(`{"native_session_id":null,"logical_workspace_id":null,"opaque_source_ref":` + literal + `}`))
				return err
			},
		},
		{
			name: "finding code", min: 1, max: 128,
			member: "code", text: "string[1..128]",
			drive: func(t *testing.T, literal string) error {
				valid := `{"severity":"warning","code":"cap-1","message":"a warning","remediation":null,"extensions":{}}`
				_, err := DecodeFinding(mutateMember(t, []byte(valid), "code", literal))
				return err
			},
		},
		{
			name: "finding message", min: 1, max: 4096,
			member: "message", text: "string[1..4096]",
			drive: func(t *testing.T, literal string) error {
				valid := `{"severity":"warning","code":"cap-1","message":"a warning","remediation":null,"extensions":{}}`
				_, err := DecodeFinding(mutateMember(t, []byte(valid), "message", literal))
				return err
			},
		},
		{
			name: "finding remediation", min: 1, max: 4096,
			member: "remediation", text: "string[1..4096]",
			drive: func(t *testing.T, literal string) error {
				valid := `{"severity":"warning","code":"cap-1","message":"a warning","remediation":null,"extensions":{}}`
				_, err := DecodeFinding(mutateMember(t, []byte(valid), "remediation", literal))
				return err
			},
		},
		{
			name: "capture native_item_key", min: 1, max: 512,
			member: "native_item_key", text: "string[1..512]",
			drive: func(t *testing.T, literal string) error {
				valid := `{"native_item_key":"item-1","class":"durable_payload","byte_count":12,"required":true,"extensions":{}}`
				_, err := DecodeCapturePlanItem(mutateMember(t, []byte(valid), "native_item_key", literal))
				return err
			},
		},
		{
			name: "resume argv word", min: 1, max: 4096,
			member: "argv", text: "string[1..4096]",
			drive: func(t *testing.T, literal string) error {
				return checkResumeArgv(t, `["target-1",`+literal+`]`)
			},
		},
		{
			name: "tuple environment_version", min: 1, max: 128,
			member: "environment_version", text: "string[1..128]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTuple(mutateMember(t, []byte(fixtureTupleJSON()), "environment_version", literal))
				return err
			},
		},
		{
			name: "tuple suite_revision", min: 1, max: 128,
			member: "suite_revision", text: "string[1..128]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(mutateNested(t, []byte(fixtureEntryJSON(DirectionTargetWrite)), "fixture_evidence", "suite_revision", literal))
				return err
			},
		},
		{
			name: "tuple native_cli_family", min: 1, max: 128,
			member: "native_cli_family", text: "string[1..128]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(mutateNested(t, []byte(fixtureEntryJSON(DirectionTargetWrite)), "resume_smoke_evidence", "native_cli_family", literal))
				return err
			},
		},
		{
			name: "fidelity code", min: 1, max: 128,
			member: "code", text: "string[1..128]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(withFidelityLimits(t, `[`+fidelityRow(literal, `"c"`, `"d"`)+`]`, DirectionTargetWrite))
				return err
			},
		},
		{
			name: "fidelity affected_class", min: 1, max: 128,
			member: "affected_class", text: "string[1..128]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(withFidelityLimits(t, `[`+fidelityRow(`"c"`, literal, `"d"`)+`]`, DirectionTargetWrite))
				return err
			},
		},
		{
			name: "fidelity detail", min: 1, max: 4096,
			member: "detail", text: "string[1..4096]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(withFidelityLimits(t, `[`+fidelityRow(`"c"`, `"a"`, literal)+`]`, DirectionTargetWrite))
				return err
			},
		},
		{
			name: "contract identifier", min: 1, max: 256,
			member: "contract_id", text: "string[1..256]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(withContracts(t, `[`+contractRow(literal, []string{`"1.0.0"`})+`]`))
				return err
			},
		},
		{
			name: "revocation reason", min: 1, max: 4096,
			member: "revocation_reason", text: "string[1..4096]",
			drive: func(t *testing.T, literal string) error {
				_, err := DecodeTupleEntry(revokedWithReason(t, literal))
				return err
			},
		},
		{
			name: "probe detail", min: 0, max: 2048,
			member: "tool_history", text: "string[0..2048]",
			drive: func(t *testing.T, literal string) error {
				manifest, err := DecodeManifest([]byte(fixtureManifestJSON()))
				if err != nil {
					t.Fatalf("DecodeManifest: %v", err)
				}
				digest := ManifestDigest(manifest).String()
				fixture := []byte(fixtureProbeJSON(digest))
				mutated := replaceCapability(t, fixture, "tool_history", `{"status":"available","enabled":true,"evidence":"probed","detail":`+literal+`}`)
				_, err = DecodeProbe(mutated)
				return err
			},
		},
	}
	if want := boundCensusDriverCount("TestStringBoundEdges"); len(rows) != want {
		t.Fatalf("string bound rows = %d, want %d; the table is short, not the package", len(rows), want)
	}
	for _, row := range rows {
		row := row
		t.Run(row.name, func(t *testing.T) {
			// One below the minimum refuses (the empty string
			// when min is 1); the minimum itself admits.
			if row.min > 0 {
				if err := row.drive(t, quotedRepeat(row.min-1)); err == nil {
					t.Fatalf("%s admitted length %d below the %d minimum", row.name, row.min-1, row.min)
				} else {
					requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, row.text)
				}
			}
			if err := row.drive(t, quotedRepeat(row.min)); err != nil {
				t.Fatalf("%s refused length %d at the minimum: %v", row.name, row.min, err)
			}
			// The maximum admits; one past it refuses.
			if err := row.drive(t, quotedRepeat(row.max)); err != nil {
				t.Fatalf("%s refused length %d at the maximum: %v", row.name, row.max, err)
			}
			if err := row.drive(t, quotedRepeat(row.max+1)); err == nil {
				t.Fatalf("%s admitted length %d past the %d maximum", row.name, row.max+1, row.max)
			} else {
				requireRefusal(t, err, "session_adapter_protocol_error", "member", row.member, row.text)
			}
		})
	}
}

// checkResumeArgv drives one argv array through the resume-plan
// success gate with the explicit identity occurring exactly once.
func checkResumeArgv(t *testing.T, argv string) error {
	t.Helper()
	manifestDigest := fixtureManifestDigestText(t)
	request := fixtureRequestBody(t, OpResumePlan, manifestDigest)
	decoded, err := CheckRequestBody(OpResumePlan, request)
	if err != nil {
		t.Fatalf("CheckRequestBody: %v", err)
	}
	facts := SuccessFacts{Context: decoded, ResumeTargetID: "target-1"}
	base := fixtureSuccessBody(t, OpResumePlan, requestContextOf(t, request), manifestDigest)
	return CheckSuccessBody(OpResumePlan, mutateMember(t, base, "argv", argv), facts)
}

// mutateNested replaces one member inside one nested object member
// of a fixture body.
func mutateNested(t *testing.T, body []byte, outer, inner, replacement string) []byte {
	t.Helper()
	var members map[string]json.RawMessage
	if err := json.Unmarshal(body, &members); err != nil {
		t.Fatalf("mutateNested: fixture is not JSON: %v", err)
	}
	raw, present := members[outer]
	if !present {
		t.Fatalf("mutateNested: %q not in fixture", outer)
	}
	var nested map[string]json.RawMessage
	if err := json.Unmarshal(raw, &nested); err != nil {
		t.Fatalf("mutateNested: %q is not an object: %v", outer, err)
	}
	if _, present := nested[inner]; !present {
		t.Fatalf("mutateNested: %q not in %q", inner, outer)
	}
	nested[inner] = json.RawMessage(replacement)
	rebuilt, err := json.Marshal(nested)
	if err != nil {
		t.Fatalf("mutateNested: %v", err)
	}
	members[outer] = rebuilt
	out, err := json.Marshal(members)
	if err != nil {
		t.Fatalf("mutateNested: %v", err)
	}
	return out
}

// fidelityRow builds one fidelity-limit row with the given raw
// code, class, and detail literals.
func fidelityRow(code, class, detail string) string {
	return `{"code":` + code + `,"affected_class":` + class + `,"maximum_disposition":"exact","detail":` + detail + `}`
}

// withFidelityLimits returns the target-write entry fixture with
// the limit array replaced.
func withFidelityLimits(t *testing.T, limits string, direction Direction) []byte {
	t.Helper()
	return mutateMember(t, []byte(fixtureEntryJSON(direction)), "known_fidelity_limits", limits)
}

// contractRow builds one contract row with the given raw
// identifier literal and raw version literals.
func contractRow(identifier string, versions []string) string {
	return `{"contract_id":` + identifier + `,"versions":[` + strings.Join(versions, ",") + `]}`
}

// withContracts returns the target-write entry fixture with the
// contract array replaced.
func withContracts(t *testing.T, contracts string) []byte {
	t.Helper()
	return mutateMember(t, []byte(fixtureEntryJSON(DirectionTargetWrite)), "contracts", contracts)
}

// revokedWithReason returns the target-write entry fixture revoked
// with the given raw reason literal.
func revokedWithReason(t *testing.T, reason string) []byte {
	t.Helper()
	fixture := []byte(fixtureEntryJSON(DirectionTargetWrite))
	revoked := mutateMember(t, fixture, "status", `"revoked"`)
	revoked = mutateMember(t, revoked, "revocation_reason", reason)
	return mutateMember(t, revoked, "revoked_at", `"2026-06-02T00:00:00.000Z"`)
}

// TestArrayBoundEdges drives every survivor array cap at its edge
// through its production entry: empty, one, max, and max+1.
func TestArrayBoundEdges(t *testing.T) {
	t.Run("resume argv 1..128", func(t *testing.T) {
		argv := func(n int) string {
			if n == 0 {
				return `[]`
			}
			return `["target-1"` + strings.Repeat(`,"w"`, n-1) + `]`
		}
		if err := checkResumeArgv(t, argv(0)); err == nil {
			t.Fatal("admitted an empty argv")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "argv", "1..128 entries")
		}
		if err := checkResumeArgv(t, argv(1)); err != nil {
			t.Fatalf("refused a one-word argv: %v", err)
		}
		if err := checkResumeArgv(t, argv(128)); err != nil {
			t.Fatalf("refused a 128-word argv at the maximum: %v", err)
		}
		if err := checkResumeArgv(t, argv(129)); err == nil {
			t.Fatal("admitted a 129-word argv past the 128 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "argv", "1..128 entries")
		}
	})
	t.Run("required dispositions 1..7", func(t *testing.T) {
		manifestDigest := fixtureManifestDigestText(t)
		drive := func(t *testing.T, value string) error {
			request := fixtureRequestBody(t, OpProjectionPlan, manifestDigest)
			_, err := CheckRequestBody(OpProjectionPlan, mutateMember(t, request, "required_dispositions", value))
			return err
		}
		seven := `{"user_message":["exact","omitted","opaque_preserved","semantic","summarized","synthesized","unrecoverable"]}`
		eight := `{"user_message":["exact","omitted","opaque_preserved","semantic","summarized","synthesized","unrecoverable","exact"]}`
		if err := drive(t, `{"user_message":[]}`); err == nil {
			t.Fatal("admitted an empty disposition list")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "user_message", "1..7 entries")
		}
		if err := drive(t, `{"user_message":["exact"]}`); err != nil {
			t.Fatalf("refused a one-entry disposition list: %v", err)
		}
		if err := drive(t, seven); err != nil {
			t.Fatalf("refused a seven-entry disposition list at the maximum: %v", err)
		}
		// The count gate fires before the element gates, so the
		// eighth entry refuses on the count even though it also
		// repeats a member.
		if err := drive(t, eight); err == nil {
			t.Fatal("admitted an eight-entry disposition list past the 7 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "user_message", "1..7 entries")
		}
	})
	t.Run("contracts 1..64", func(t *testing.T) {
		contracts := func(n int) string {
			rows := make([]string, 0, n)
			for i := 1; i <= n; i++ {
				rows = append(rows, contractRow(strconv.Quote("c"+twoDigits(i)), []string{`"1.0.0"`}))
			}
			return `[` + strings.Join(rows, ",") + `]`
		}
		if _, err := DecodeTupleEntry(withContracts(t, `[]`)); err == nil {
			t.Fatal("admitted zero contract rows")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "contracts", "1..64 rows")
		}
		if _, err := DecodeTupleEntry(withContracts(t, contracts(1))); err != nil {
			t.Fatalf("refused one contract row: %v", err)
		}
		if _, err := DecodeTupleEntry(withContracts(t, contracts(64))); err != nil {
			t.Fatalf("refused 64 contract rows at the maximum: %v", err)
		}
		if _, err := DecodeTupleEntry(withContracts(t, contracts(65))); err == nil {
			t.Fatal("admitted 65 contract rows past the 64 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "contracts", "1..64 rows")
		}
	})
	t.Run("versions 1..32", func(t *testing.T) {
		versions := func(n int) string {
			names := make([]string, 0, n)
			for i := 1; i <= n; i++ {
				// Zero-padded prerelease tags sort
				// lexicographically, so the sorted-unique
				// gate stays green while the count sweeps.
				names = append(names, strconv.Quote("1.0.0-a"+twoDigits(i)))
			}
			return contractRow(`"urn:ax:protocol:session-adapter"`, names)
		}
		drive := func(t *testing.T, row string) error {
			_, err := DecodeTupleEntry(withContracts(t, `[`+row+`]`))
			return err
		}
		if err := drive(t, `{"contract_id":"urn:ax:protocol:session-adapter","versions":[]}`); err == nil {
			t.Fatal("admitted zero versions")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "versions", "1..32 entries")
		}
		if err := drive(t, versions(1)); err != nil {
			t.Fatalf("refused one version: %v", err)
		}
		if err := drive(t, versions(32)); err != nil {
			t.Fatalf("refused 32 versions at the maximum: %v", err)
		}
		if err := drive(t, versions(33)); err == nil {
			t.Fatal("admitted 33 versions past the 32 maximum")
		} else {
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "versions", "1..32 entries")
		}
	})
}

// twoDigits renders i zero-padded to two digits for
// lexicographically sortable fixture names.
func twoDigits(i int) string {
	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}
