package clonereadback_test

import (
	"reflect"
	"strings"
	"testing"

	clonereadback "github.com/relux-works/agent-session-manager/internal/clonereadback"
)

// This file proves by reflection that invalid UTF-8 is refused at
// every Build-side string gate: the walker derives every string
// field of the production input types (nested rows included) and
// feeds each lone-invalid bytes plus trailing-invalid bytes. Raw
// JSON members ride owner decoders; the owner suites pin their
// UTF-8 behavior, and the delegation table pins the wiring.

// stringField describes one derived Build-side string: a setter
// over a fresh valid input plus the refusal literal.
type stringField struct {
	label  string
	set    func(valid clonereadback.ReadBackManifestInput, validReport clonereadback.ValidationReportInput, value string) (clonereadback.ReadBackManifestInput, clonereadback.ValidationReportInput, bool)
	member string
}

// buildUTF8Fields derives every string field of the two input
// types by reflection. The derivation fails loudly on any field
// kind it cannot classify; the inventory test pins the exact field
// count so a silently skipped string is a red test, not a gap.
func buildUTF8Fields(t *testing.T) []stringField {
	t.Helper()
	var fields []stringField
	collectStrings(t, reflect.TypeOf(clonereadback.ReadBackManifestInput{}), "readback", &fields)
	collectStrings(t, reflect.TypeOf(clonereadback.EvidenceObjectInput{}), "evidence", &fields)
	collectStrings(t, reflect.TypeOf(clonereadback.ValidationReportInput{}), "report", &fields)
	return fields
}

func collectStrings(t *testing.T, typ reflect.Type, shape string, fields *[]stringField) {
	t.Helper()
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.PkgPath != "" {
			t.Fatalf("unexported field %s.%s: extend the walker", shape, field.Name)
		}
		member := snakeMember(field.Name)
		if irregular, ok := irregularMembers[field.Name]; ok {
			member = irregular
		}
		switch field.Type.Kind() {
		case reflect.String:
			name := field.Name
			shapeName := shape
			memberName := member
			*fields = append(*fields, stringField{
				label:  shapeName + "." + name,
				member: memberName,
				set: func(valid clonereadback.ReadBackManifestInput, validReport clonereadback.ValidationReportInput, value string) (clonereadback.ReadBackManifestInput, clonereadback.ValidationReportInput, bool) {
					switch shapeName {
					case "readback":
						reflect.ValueOf(&valid).Elem().FieldByName(name).SetString(value)
						return valid, validReport, true
					case "report":
						reflect.ValueOf(&validReport).Elem().FieldByName(name).SetString(value)
						return valid, validReport, false
					case "evidence":
						valid.EvidenceObjects = []clonereadback.EvidenceObjectInput{validEvidenceInput("native_sample", "utf8")}
						reflect.ValueOf(&valid.EvidenceObjects[0]).Elem().FieldByName(name).SetString(value)
						return valid, validReport, true
					}
					return valid, validReport, true
				},
			})
		case reflect.Bool, reflect.Uint64, reflect.Map:
			// Non-string leaves: nothing to feed.
		case reflect.Slice, reflect.Array:
			element := field.Type.Elem()
			switch element.Kind() {
			case reflect.Uint8, reflect.Struct:
				// Raw JSON and nested rows: rows are walked
				// as their own type above; raw JSON rides
				// owner decoders.
			case reflect.String:
				if field.Name != "ParsedHeadIDs" {
					t.Fatalf("string slice %s.%s is not the heads member: extend the walker", shape, field.Name)
				}
			default:
				t.Fatalf("unhandled slice element kind %s at %s.%s: extend the walker", element.Kind(), shape, field.Name)
			}
		default:
			t.Fatalf("unhandled kind %s at %s.%s: extend the walker", field.Type.Kind(), shape, field.Name)
		}
	}
}

// TestBuildUTF8Reflection feeds invalid UTF-8 to every derived
// Build-side string field at the production Build entries: each
// refuses with its member literal.
func TestBuildUTF8Reflection(t *testing.T) {
	staged, live := mustReadPair(t)
	poison := []string{string([]byte{0xff}), "ok-prefix-\xff", "\xff-ok-suffix"}
	for _, field := range buildUTF8Fields(t) {
		for _, value := range poison {
			input, rinput, isReadBack := field.set(validReadBackInput(), validReportInput(staged, live), value)
			if isReadBack {
				_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
				if err == nil {
					t.Fatalf("field %s admits invalid UTF-8 at Build", field.label)
				}
				if !strings.Contains(err.Error(), field.member) {
					t.Fatalf("field %s error %q misses member literal %q", field.label, err.Error(), field.member)
				}
			} else {
				_, err := clonereadback.BuildValidationReport(rinput, staged, live)
				if err == nil {
					t.Fatalf("field %s admits invalid UTF-8 at Build", field.label)
				}
				if !strings.Contains(err.Error(), field.member) {
					t.Fatalf("field %s error %q misses member literal %q", field.label, err.Error(), field.member)
				}
			}
		}
	}
}

// TestBuildUTF8HeadsAndExtensions feeds invalid UTF-8 to the heads
// array items and one extension value at both production Build
// entries.
func TestBuildUTF8HeadsAndExtensions(t *testing.T) {
	staged, live := mustReadPair(t)
	input := validReadBackInput()
	input.ParsedHeadIDs = []string{"head-a", "bad-\xff"}
	_, _, err := clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "parsed_head_ids")
	input = validReadBackInput()
	input.Extensions = map[string]any{"com.example.note": "bad-\xff"}
	_, _, err = clonereadback.BuildReadBackEvidenceManifest(input, stagedAuthority(t))
	requireRefusal(t, err, "read-back evidence manifest", "extensions invalid")
	rinput := validReportInput(staged, live)
	rinput.Extensions = map[string]any{"com.example.note": "bad-\xff"}
	_, err = clonereadback.BuildValidationReport(rinput, staged, live)
	requireRefusal(t, err, "clone validation report", "extensions invalid")
}

// TestUTF8FieldInventory pins the derivation count: 7 read-back
// strings, 4 evidence strings, 9 report strings.
func TestUTF8FieldInventory(t *testing.T) {
	fields := buildUTF8Fields(t)
	counts := map[string]int{}
	for _, field := range fields {
		shape := field.label[:strings.Index(field.label, ".")]
		counts[shape]++
	}
	if counts["readback"] != 7 || counts["evidence"] != 4 || counts["report"] != 9 {
		t.Fatalf("utf8 fields = %v, want readback=7 evidence=4 report=9", counts)
	}
}
