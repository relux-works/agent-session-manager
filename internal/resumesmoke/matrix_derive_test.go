package resumesmoke

import (
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/provhost"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// This file derives the Section 8.4 native-resume expectations from
// the pinned specification text and requires the smoke matrix to
// cover every row exactly. The cell values are never retyped here:
// the test parses the provider/platform rows and the native-resume
// cell out of the document, so a matrix value that drifts from the
// pinned contract reddens here rather than passing against a
// hand-typed copy of the same mistake.

// specRow is one parsed Section 8.4 row: the provider, the platform,
// the architectures it names, the native-resume cell, and the
// version the row pins when it names one.
type specRow struct {
	provider      string
	platform      string
	architectures []string
	cell          Cell
	pinned        string
}

// derivedFixtureVersions are the probe versions the row tests run
// with. Versions are fixtures, not derivations — Section 8.4 pins
// only the Muse macOS arm64 probe — except the Muse pin itself,
// which the row text carries.
var derivedFixtureVersions = map[string]string{
	"codex":          "0.147.0",
	"claude":         "2.1.229",
	"gemini":         "0.54.4",
	"muse":           "0.1.0",
	"antigravity":    "1.1.14",
	"pi":             "0.73.1",
	"qwen":           "1.0.0",
	"future-plugin":  "0.0.0",
	"not-a-provider": "0.0.0",
}

// TestResumeMatrixCoversEverySpecRow derives every Section 8.4 row
// and requires the smoke matrix to read the row's native-resume
// cell for each named tuple. The parsed row count is asserted so a
// parser that silently derives nothing fails closed.
func TestResumeMatrixCoversEverySpecRow(t *testing.T) {
	rows := parseSection84(t)
	if len(rows) != 27 {
		t.Fatalf("parsed %d section 8.4 rows, want 27", len(rows))
	}
	var tuples []specRow
	for _, row := range rows {
		tuples = append(tuples, expandRow(row)...)
	}
	for _, row := range tuples {
		row := row
		for _, architecture := range row.architectures {
			architecture := architecture
			t.Run(row.provider+"/"+row.platform+"/"+architecture, func(t *testing.T) {
				version := derivedFixtureVersions[row.provider]
				if row.pinned != "" {
					version = row.pinned
				}
				cell, gate := ResumeCell(row.provider, version, row.platform, architecture)
				if cell != row.cell {
					t.Fatalf("ResumeCell(%s %s %s %s) = %q, want row cell %q", row.provider, version, row.platform, architecture, cell, row.cell)
				}
				if gate == "" {
					t.Fatal("ResumeCell returned an empty gate citation")
				}
				if row.pinned != "" {
					// A pinned row accepts only its probe: one
					// patch off the pin is unknown, per the
					// Appendix B unsettled-behavior rule the
					// matrix cites. The pin itself comes from
					// the row text; the off-pin version below
					// is the derivation's probe of it.
					off := offPinVersion(row.pinned)
					offCell, _ := ResumeCell(row.provider, off, row.platform, architecture)
					if offCell != CellUnknown {
						t.Fatalf("ResumeCell(%s %s %s %s) = %q, want unknown off the %s pin", row.provider, off, row.platform, architecture, offCell, row.pinned)
					}
				}
			})
		}
	}
}

// TestResumeMatrixAgreesWithTupleGate requires the matrix and the
// landed Section 8.4 tuple gate to agree on the direction for every
// derived tuple: available and conditional rows pass the gate, and
// unsupported and unknown rows refuse. Both sides are production;
// the agreement pins them against each other.
func TestResumeMatrixAgreesWithTupleGate(t *testing.T) {
	rows := parseSection84(t)
	if len(rows) != 27 {
		t.Fatalf("parsed %d section 8.4 rows, want 27", len(rows))
	}
	var tuples []specRow
	for _, row := range rows {
		tuples = append(tuples, expandRow(row)...)
	}
	for _, row := range tuples {
		version := derivedFixtureVersions[row.provider]
		if row.pinned != "" {
			version = row.pinned
		}
		for _, architecture := range row.architectures {
			tuple := provhost.BuildTuple{ProviderID: row.provider, ProviderVersion: version, Platform: row.platform, Architecture: architecture}
			err := provhost.CheckResumeTuple(tuple)
			switch row.cell {
			case CellAvailable, CellConditional:
				if err != nil {
					t.Errorf("CheckResumeTuple(%+v) = %v, want pass for %q row", tuple, err, row.cell)
				}
			default:
				if err == nil {
					t.Errorf("CheckResumeTuple(%+v) = nil, want refusal for %q row", tuple, row.cell)
				}
			}
		}
	}
	// The off-pin Muse probe refuses at the gate too: the unknown
	// reading of the unsettled version is shared, not unilateral.
	off := provhost.BuildTuple{ProviderID: "muse", ProviderVersion: offPinVersion("0.1.0"), Platform: "macos", Architecture: "arm64"}
	if err := provhost.CheckResumeTuple(off); err == nil {
		t.Errorf("CheckResumeTuple(%+v) = nil, want refusal off the muse pin", off)
	}
}

// offPinVersion bumps the last version component by one: the
// one-patch-off probe the pin rule must refuse.
func offPinVersion(pinned string) string {
	parts := strings.Split(pinned, ".")
	last := parts[len(parts)-1]
	bumped := "1"
	if last == "1" {
		bumped = "2"
	} else if last == "0" {
		bumped = "1"
	} else {
		bumped = last + ".1"
	}
	parts[len(parts)-1] = bumped
	return strings.Join(parts, ".")
}

// parseSection84 extracts the Section 8.4 provider/platform rows
// from the pinned v0.6.0 document: the table between the 8.4
// heading and the Section 9 heading.
func parseSection84(t *testing.T) []specRow {
	t.Helper()
	document, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("specdoc.LoadV060: %v", err)
	}
	inTable := false
	var rows []specRow
	for number := 1; number <= document.LineCount(); number++ {
		line, ok := document.Line(number)
		if !ok {
			continue
		}
		if strings.HasPrefix(line, "### 8.4 ") {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(line, "## 9.") {
			break
		}
		if !inTable || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitTableLine(line)
		if len(cells) != 3 {
			continue
		}
		head, status := strings.TrimSpace(cells[0]), strings.TrimSpace(cells[1])
		if head == "Provider/platform" || strings.HasPrefix(head, "---") {
			continue
		}
		rows = append(rows, parseSpecRow(t, head, status))
	}
	return rows
}

// parseSpecRow parses one Section 8.4 row head and status into its
// provider, platform, architectures, and native-resume cell. The
// native cell is the first slash-separated status member: the
// triple is always native resume, cross-host materialization, and
// managed terminal, in that order. Rows that span platforms or
// providers expand: Qwen covers every platform, and the future row
// is probed with two distinct unknown provider IDs, so the unknown
// reading is proven for unknown-ness rather than one literal.
func parseSpecRow(t *testing.T, head, status string) specRow {
	t.Helper()
	native := strings.TrimSpace(strings.Split(status, "/")[0])
	var cell Cell
	var pinned string
	switch {
	case native == "A":
		cell = CellAvailable
	case strings.HasPrefix(native, "A for probed "):
		cell = CellAvailable
		pinned = strings.TrimSpace(strings.TrimPrefix(native, "A for probed "))
	case strings.HasPrefix(native, "C"):
		cell = CellConditional
	case strings.HasPrefix(native, "U"):
		cell = CellUnsupported
	case strings.HasPrefix(native, "?"):
		cell = CellUnknown
	default:
		t.Fatalf("row %q carries an unparsable native cell %q", head, native)
	}
	provider, platforms, architectures := parseRowHead(t, head)
	return specRow{provider: provider, platform: platforms[0], architectures: architectures, cell: cell, pinned: pinned}
}

// expandRow expands one parsed row into the tuples it names. Most
// rows name one tuple per architecture; the Qwen row spans every
// platform, and the future row spans every platform under two
// distinct unknown provider IDs.
func expandRow(row specRow) []specRow {
	if row.provider == "qwen" {
		var expanded []specRow
		for _, platform := range []string{"macos", "linux", "wsl2", "windows"} {
			expanded = append(expanded, specRow{provider: "qwen", platform: platform, architectures: row.architectures, cell: row.cell})
		}
		return expanded
	}
	if row.provider == "future-plugin" {
		var expanded []specRow
		for _, provider := range []string{"future-plugin", "not-a-provider"} {
			for _, platform := range []string{"macos", "linux", "wsl2", "windows"} {
				expanded = append(expanded, specRow{provider: provider, platform: platform, architectures: row.architectures, cell: row.cell})
			}
		}
		return expanded
	}
	return []specRow{row}
}

// parseRowHead maps one Section 8.4 row head to its provider ID,
// platforms, and named architectures. Every token below is pinned
// by the row-count guard and the cell assertions above: an unknown
// token fails the test instead of defaulting into a row.
func parseRowHead(t *testing.T, head string) (string, []string, []string) {
	t.Helper()
	provider := ""
	for _, known := range []struct {
		token string
		id    string
	}{
		{"Codex", "codex"},
		{"Claude", "claude"},
		{"Gemini", "gemini"},
		{"Muse", "muse"},
		{"Antigravity", "antigravity"},
		{"Pi", "pi"},
		{"Qwen", "qwen"},
		{"Future plugin", "future-plugin"},
	} {
		if strings.HasPrefix(head, known.token) {
			provider = known.id
		}
	}
	if provider == "" {
		t.Fatalf("row %q names an unknown provider", head)
	}
	var platforms []string
	for _, known := range []struct {
		token string
		id    string
	}{
		{"macOS", "macos"},
		{"Linux", "linux"},
		{"WSL2", "wsl2"},
		{"native Windows", "windows"},
	} {
		if strings.Contains(head, known.token) {
			platforms = append(platforms, known.id)
		}
	}
	if provider == "qwen" || provider == "future-plugin" {
		platforms = []string{"macos"}
	}
	if len(platforms) != 1 {
		t.Fatalf("row %q names %d platforms, want exactly one", head, len(platforms))
	}
	var architectures []string
	namesArch := func(token string) bool {
		for _, field := range strings.Fields(head) {
			if field == token {
				return true
			}
		}
		return false
	}
	switch {
	case namesArch("amd64") && namesArch("arm64"):
		architectures = []string{"amd64", "arm64"}
	case namesArch("arm64"):
		architectures = []string{"arm64"}
	case namesArch("amd64"):
		architectures = []string{"amd64"}
	default:
		architectures = []string{"amd64", "arm64"}
	}
	return provider, platforms, architectures
}

// splitTableLine splits one markdown table line into its trimmed
// cells, dropping the leading and trailing pipes.
func splitTableLine(line string) []string {
	parts := strings.Split(line, "|")
	if len(parts) < 3 {
		return nil
	}
	cells := make([]string, 0, len(parts)-2)
	for _, part := range parts[1 : len(parts)-1] {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}
