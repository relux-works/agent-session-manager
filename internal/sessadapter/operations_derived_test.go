package sessadapter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// deriveOperationTable parses the Section 7.8 operation-body table
// from the pinned document into request and success member lists
// per operation, in table order. Bodies given as prose (the
// manifest and probe exact objects) derive no list. An empty
// derivation — no rows, or a row with no members — fails closed:
// a domain that silently derives nothing is not a measurement.
func deriveOperationTable(t *testing.T) (operations []string, requests map[string][]string, successes map[string][]string) {
	t.Helper()
	if _, err := specdoc.Load(); err != nil {
		t.Fatalf("specdoc.Load: %v", err)
	}
	text := string(specdoc.Bytes())
	// The scan is windowed to the Section 7.8 registry table: the
	// Section 7.9 probe row reuses the <code>probe</code> name with
	// a different vocabulary, so an unwindowed scan merges two
	// sections' rows into one derivation.
	table, ok := sectionTableWindow(text, "The exact request and success <code>body</code> registry is:", "Candidate objects are addressed only by their exact")
	if !ok {
		t.Fatal("Section 7.8 registry table window not found; the scan is broken, not the registry")
	}
	requests = map[string][]string{}
	successes = map[string][]string{}
	for _, line := range strings.Split(table, "\n") {
		if !strings.HasPrefix(line, "| <code>") {
			continue
		}
		// Cells split on unescaped pipes only: body cells
		// carry \| vocabulary alternations that never
		// separate columns.
		cells := splitTableCells(line)
		if len(cells) < 4 {
			continue
		}
		name := strings.TrimSpace(cells[1])
		name = strings.TrimPrefix(name, "<code>")
		name = strings.TrimSuffix(name, "</code>")
		if !validOperation(name) {
			continue
		}
		operations = append(operations, name)
		requests[name] = parseBodyMembers(t, cells[2])
		if body := parseBodyMembers(t, cells[3]); body != nil {
			successes[name] = body
		}
	}
	if len(operations) != 14 {
		t.Fatalf("derived %d operation rows, want 14; the table scan is broken, not the registry", len(operations))
	}
	for _, name := range operations {
		if len(requests[name]) == 0 && name != "manifest" {
			t.Fatalf("derived no request members for %q; the parser is broken", name)
		}
	}
	return operations, requests, successes
}

// parseBodyMembers extracts the member names from one table cell's
// first {…} span, splitting top-level commas only: nested
// closed-map parens and uint53 brackets never separate members.
// A cell with no brace span (a prose body reference) yields nil.
func parseBodyMembers(t *testing.T, cell string) []string {
	t.Helper()
	start := strings.Index(cell, "{")
	if start < 0 {
		return nil
	}
	depth := 0
	end := -1
	for index := start; index < len(cell); index++ {
		switch cell[index] {
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
			if depth == 0 {
				end = index
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		t.Fatalf("unbalanced body span in %q", cell)
	}
	span := strings.ReplaceAll(cell[start+1:end], "<code>", "")
	span = strings.ReplaceAll(span, "</code>", "")
	if strings.TrimSpace(span) == "" {
		return []string{}
	}
	var members []string
	depth = 0
	current := ""
	for index := 0; index < len(span); index++ {
		char := span[index]
		switch char {
		case '{', '[', '(':
			depth++
			current += string(char)
		case '}', ']', ')':
			depth--
			current += string(char)
		case ',':
			if depth == 0 {
				members = append(members, memberName(strings.TrimSpace(current)))
				current = ""
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}
	}
	members = append(members, memberName(strings.TrimSpace(current)))
	return members
}

// sectionTableWindow returns the document text between the marker
// line and the terminator line. Both must be present exactly once;
// otherwise the window is untrusted and the derivation fails
// closed.
func sectionTableWindow(text, marker, terminator string) (string, bool) {
	if strings.Count(text, marker) != 1 || strings.Count(text, terminator) != 1 {
		return "", false
	}
	start := strings.Index(text, marker)
	end := strings.Index(text, terminator)
	if start >= end {
		return "", false
	}
	return text[start:end], true
}

// splitTableCells splits a markdown table row on pipes that are
// not backslash-escaped.
func splitTableCells(line string) []string {
	var cells []string
	current := ""
	for index := 0; index < len(line); index++ {
		if line[index] == '|' && (index == 0 || line[index-1] != '\\') {
			cells = append(cells, current)
			current = ""
			continue
		}
		current += string(line[index])
	}
	return append(cells, current)
}

// memberName returns the member before the first colon, or the
// whole token for bare members like extensions.
func memberName(cell string) string {
	if index := strings.Index(cell, ":"); index >= 0 {
		return cell[:index]
	}
	return cell
}

// TestOperationBodiesAreDerivedFromSpec requires the production
// member tables to equal the Section 7.8 table in order: every
// table member appears in production at the same position, and
// nothing else does. A dropped, added, or reordered member
// reddens here.
func TestOperationBodiesAreDerivedFromSpec(t *testing.T) {
	operations, requests, successes := deriveOperationTable(t)
	for _, name := range operations {
		operation := Operation(name)
		wantRequest := requests[name]
		gotRequest := requestBodyMembers[operation]
		if fmt.Sprintf("%v", gotRequest) != fmt.Sprintf("%v", wantRequest) {
			t.Errorf("request %q = %v, want table %v", name, gotRequest, wantRequest)
		}
		wantSuccess, derived := successes[name]
		// Manifest, probe, and doctor successes decode through
		// dedicated decoders; their member tables live with
		// those decoders, not in the shared success table.
		dedicated := map[Operation][]string{
			OpManifest: manifestRequired,
			OpProbe:    probeRequired,
			OpDoctor:   doctorResultRequired,
		}
		gotSuccess := successBodyMembers[operation]
		if !derived {
			if dedicated[operation] == nil {
				t.Errorf("success %q derives no table body and names no dedicated decoder", name)
			}
			continue
		}
		if dedicated[operation] != nil {
			gotSuccess = dedicated[operation]
		}
		if fmt.Sprintf("%v", gotSuccess) != fmt.Sprintf("%v", wantSuccess) {
			t.Errorf("success %q = %v, want table %v", name, gotSuccess, wantSuccess)
		}
	}
	t.Logf("operation body coverage: %d operations derived", len(operations))
}

// TestOperationBodiesRefuseDerivedMutations drops every derived
// request member in turn from a fresh positive fixture: each
// derivative must refuse. The probe points come from the pinned
// document, the verdicts from production, so a production table
// that silently drops a member reddens here on the missing
// derivative it no longer refuses.
func TestOperationBodiesRefuseDerivedMutations(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	operations, requests, _ := deriveOperationTable(t)
	for _, name := range operations {
		operation := Operation(name)
		fixture := fixtureRequestBody(t, operation, manifestDigest)
		for _, member := range requests[name] {
			t.Run(name+"/missing "+member, func(t *testing.T) {
				_, err := CheckRequestBody(operation, dropMember(t, fixture, member))
				if err == nil {
					t.Fatalf("CheckRequestBody(%q) admitted a body without %q", name, member)
				}
				requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "misses a required member")
			})
		}
		t.Run(name+"/unknown member", func(t *testing.T) {
			_, err := CheckRequestBody(operation, appendMember(t, fixture, "unexpected", `true`))
			if err == nil {
				t.Fatalf("CheckRequestBody(%q) admitted an unknown member", name)
			}
			requireRefusal(t, err, "session_adapter_protocol_error", "member", "unexpected", "unknown member")
		})
	}
}

// TestSuccessBodiesRefuseDerivedMutations drops every derived
// success member in turn (for the eleven table-decoded bodies):
// each derivative must refuse.
func TestSuccessBodiesRefuseDerivedMutations(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	operations, _, successes := deriveOperationTable(t)
	for _, name := range operations {
		members, derived := successes[name]
		if !derived {
			continue
		}
		operation := Operation(name)
		request := fixtureRequestBody(t, operation, manifestDigest)
		decoded, err := CheckRequestBody(operation, request)
		if err != nil {
			t.Fatalf("CheckRequestBody(%q): %v", name, err)
		}
		facts := SuccessFacts{Context: decoded, ValidateMode: "staged", ResumeTargetID: "target-1", DoctorDirection: DirectionSourceRead}
		fixture := fixtureSuccessBody(t, operation, requestContextOf(t, request), manifestDigest)
		for _, member := range members {
			// The context echo is covered by its own tests;
			// dropping it here would only re-prove that arm.
			if member == "context" {
				continue
			}
			t.Run(name+"/missing "+member, func(t *testing.T) {
				err := CheckSuccessBody(operation, dropMember(t, fixture, member), facts)
				if err == nil {
					t.Fatalf("CheckSuccessBody(%q) admitted a body without %q", name, member)
				}
				requireRefusal(t, err, "session_adapter_protocol_error", "member", member, "misses a required member")
			})
		}
	}
}

// TestOperationBodyMismatch proves a body for one operation is not
// a body for another: discover checked as inspect refuses on the
// foreign member, and vice versa.
func TestOperationBodyMismatch(t *testing.T) {
	manifestDigest := fixtureManifestDigestText(t)
	discover := fixtureRequestBody(t, OpDiscover, manifestDigest)
	if _, err := CheckRequestBody(OpInspect, discover); err == nil {
		t.Fatal("inspect admitted a discover body")
	} else {
		// Three foreign members (workspace_filter, limit,
		// cursor) name nondeterministically under map
		// order, so the cross-wired body asserts code and
		// rule only; the single-unknown case below pins
		// the member.
		if failureCode(t, err) != "session_adapter_protocol_error" {
			t.Fatalf("code = %v, want session_adapter_protocol_error", err)
		}
	}
	// A body with exactly one foreign member names it
	// deterministically; multi-unknown bodies name
	// nondeterministically under map order (see above).
	if _, err := CheckRequestBody(OpDiscover, appendMember(t, discover, "unexpected", `true`)); err == nil {
		t.Fatal("discover admitted a body with one foreign member")
	} else {
		requireRefusal(t, err, "session_adapter_protocol_error", "member", "unexpected", "unknown member")
	}
	if _, err := CheckRequestBody(Operation("launch"), discover); err == nil {
		t.Fatal("admitted a body for an unknown operation")
	} else {
		requireRefusal(t, err, "operation_unknown", "operation", "launch", "closed registry")
	}
	if err := RefuseUnknownOperation("launch"); err == nil {
		t.Fatal("RefuseUnknownOperation admitted launch")
	} else {
		requireRefusal(t, err, "operation_unknown", "operation", "launch", "closed registry")
	}
	if err := RefuseUnavailableOperation(OpDiscover); err == nil {
		t.Fatal("RefuseUnavailableOperation admitted discover")
	} else {
		requireRefusal(t, err, "capability_unavailable", "operation", "discover", "does not implement")
	}
}
