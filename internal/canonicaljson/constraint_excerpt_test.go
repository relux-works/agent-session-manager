package canonicaljson

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// The constraint enumeration used to require only a non-empty Pinned SPEC
// declaration cell. That made the artifact a self-consistency proof: it could
// agree with the code and still attribute text to the specification that the
// specification never contains. The gates below compare every excerpt against
// the pinned SPEC.md itself, at the exact line the artifact declares.

const (
	// specExcerptEscapedPipe is the only decoding applied to an excerpt before
	// comparison. An unescaped pipe would end the artifact's own Markdown table
	// cell, so a literal pipe in the pinned document is written escaped. The
	// pinned document's own &#124; entities are compared literally.
	specExcerptEscapedPipe = `\|`

	verbatimEntryKind   = "verbatim"
	paraphraseEntryKind = "paraphrase"
)

// specExcerptEntry is one `L<line> “text”` or `L<line> paraphrase: text` entry
// of a Pinned SPEC declaration cell.
type specExcerptEntry struct {
	kind string
	line int
	text string
}

var specExcerptLinePrefix = regexp.MustCompile(`^L([1-9][0-9]*) `)

// parseSpecExcerptCell parses a Pinned SPEC declaration cell. It is strict: an
// unparseable cell is an error rather than a silently skipped row, because a
// cell the parser cannot read is exactly the cell that would otherwise never be
// compared to anything.
func parseSpecExcerptCell(cell string) ([]specExcerptEntry, error) {
	remainder := strings.TrimSpace(cell)
	if remainder == "" {
		return nil, fmt.Errorf("cell is empty")
	}
	var entries []specExcerptEntry
	for {
		match := specExcerptLinePrefix.FindStringSubmatch(remainder)
		if match == nil {
			return nil, fmt.Errorf("entry %d does not start with a line reference: %q", len(entries)+1, remainder)
		}
		line, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, fmt.Errorf("entry %d has an unreadable line reference: %v", len(entries)+1, err)
		}
		remainder = remainder[len(match[0]):]

		var entry specExcerptEntry
		switch {
		case strings.HasPrefix(remainder, "“"):
			closing := strings.Index(remainder, "”")
			if closing < 0 {
				return nil, fmt.Errorf("entry %d has an unterminated quote: %q", len(entries)+1, remainder)
			}
			entry = specExcerptEntry{kind: verbatimEntryKind, line: line, text: remainder[len("“"):closing]}
			remainder = remainder[closing+len("”"):]
		case strings.HasPrefix(remainder, "paraphrase: "):
			remainder = remainder[len("paraphrase: "):]
			end := len(remainder)
			if next := specExcerptEntrySeparator(remainder); next >= 0 {
				end = next
			}
			entry = specExcerptEntry{kind: paraphraseEntryKind, line: line, text: strings.TrimSpace(remainder[:end])}
			if entry.text == "" {
				return nil, fmt.Errorf("entry %d is an empty paraphrase", len(entries)+1)
			}
			remainder = remainder[end:]
		default:
			return nil, fmt.Errorf("entry %d is neither a quote nor a paraphrase: %q", len(entries)+1, remainder)
		}
		entries = append(entries, entry)

		remainder = strings.TrimSpace(remainder)
		if remainder == "" {
			return entries, nil
		}
		if !strings.HasPrefix(remainder, "; ") {
			return nil, fmt.Errorf("trailing text after entry %d: %q", len(entries), remainder)
		}
		remainder = remainder[len("; "):]
	}
}

// specExcerptEntrySeparator finds the "; L<digits> " that starts the next entry.
func specExcerptEntrySeparator(remainder string) int {
	for offset := 0; ; {
		index := strings.Index(remainder[offset:], "; L")
		if index < 0 {
			return -1
		}
		start := offset + index
		if specExcerptLinePrefix.MatchString(remainder[start+len("; "):]) {
			return start
		}
		offset = start + 1
	}
}

// decodeSpecExcerpt applies the single documented escape.
func decodeSpecExcerpt(text string) string {
	return strings.ReplaceAll(text, specExcerptEscapedPipe, "|")
}

// TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification is the
// fidelity gate the artifact previously lacked. Every row's Pinned SPEC
// declaration is compared against internal/specdoc's hash-verified copy of the
// pinned SPEC.md, at the exact line the row declares, and every row must anchor
// its own member name.
func TestConstraintEnumerationSpecExcerptsQuoteThePinnedSpecification(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}

	packageDirectory, _ := packageProductionFiles(t)
	rows := readConstraintRows(t, filepath.Join(packageDirectory, "testdata", "constraint-enumeration.md"))
	if len(rows) == 0 {
		t.Fatal("read zero constraint enumeration rows")
	}

	var failures []string
	for _, row := range rows {
		key := row.shape + "." + row.member
		for _, failure := range verifyRowAgainstPinnedSpecification(document, row) {
			failures = append(failures, key+": "+failure)
		}
	}
	sort.Strings(failures)
	if len(failures) != 0 {
		t.Fatalf("constraint enumeration rows disagree with the pinned specification:\n%s",
			strings.Join(failures, "\n"))
	}
}

// constraintRowSpecSections pins, per shape, the numbered SPEC.md clauses its
// citations may come from. Without it the comparison is shape-blind: it proves
// the quoted characters exist at the cited line and that some entry names the
// member, and nothing at all about whether that clause governs this shape.
// Retargeting `ManifestEntry.file.size` from its own 10.4 row to BlobChunk's
// bounded `size:uint53[1..4194304]` in 10.2 — a different schema, a bound this
// member does not carry — left the entire package green before this map existed.
//
// The two Session Record majors legitimately cite 2.1 as well: the document's
// own name-grammar indirection points at a section reference that contains no
// grammar, and the grammar is written in the 2.1 Terms table. Quoting both is
// what makes that indirection visible instead of flattened.
//
// The shape set is asserted exactly against the artifact, so a new shape has to
// be pinned here rather than inheriting a free pass.
var constraintRowSpecSections = map[string][]string{
	"Blob Descriptor":                     {"10.2"},
	"BlobChunk":                           {"10.2"},
	"Checkpoint Record":                   {"5.4"},
	"EnvironmentTuple":                    {"7.8"},
	"GitFeatures":                         {"10.4"},
	"GitHead":                             {"10.4"},
	"GitIndex":                            {"10.4"},
	"GitIndexEntry":                       {"10.4"},
	"GitObjectPack":                       {"10.4"},
	"GitRemote":                           {"10.4"},
	"GitSubmodule":                        {"10.4"},
	"Lease Record":                        {"5.3"},
	"ManifestEntry.directory":             {"10.4"},
	"ManifestEntry.file":                  {"10.4"},
	"ManifestEntry.hardlink":              {"10.4"},
	"ManifestEntry.symlink":               {"10.4"},
	"MigrationProvenance":                 {"17.3"},
	"Observation Event":                   {"18.1"},
	"ObservationCounts":                   {"18.1"},
	"Provider Identity Record":            {"5.5"},
	"Safe Boundary Evidence":              {"5.4"},
	"Session Event":                       {"5.2"},
	"Session Record 1.0.0":                {"2.1", "5.1"},
	"Session Record 2.0.0 and 3.0.0":      {"2.1", "5.1"},
	"Session Record Board Goal":           {"5.1"},
	"Session Record Board Identity":       {"5.1"},
	"Session Record Fork Provenance":      {"5.1"},
	"Session Record Launch Plan":          {"5.1"},
	"Session Record Task-board Reference": {"5.1"},
	"Session Record cross-environment-clone provenance": {"5.1"},
	"Session Record native-adoption provenance":         {"5.1"},
	"Session Record origin provenance":                  {"5.1"},
	"Session Record same-provider-fork provenance":      {"5.1"},
	"Transfer Manifest":                                 {"10.4"},
	"Workspace Group Record":                            {"5.6"},
	"WorkspaceMember.git":                               {"5.6"},
	"WorkspaceMember.managed_tree":                      {"5.6"},
	"WorkspaceSnapshot":                                 {"10.4"},
	"WorkspaceSnapshotMember.git":                       {"10.4"},
	"WorkspaceSnapshotMember.managed_tree":              {"10.4"},
}

// constraintRowDeclaringIdentifiers pins, per shape, the identifier under which
// the pinned document declares that shape in a Markdown table. The clause
// anchor holds a citation to the right section; it cannot hold it to the right
// row of a table that declares several sibling schemas one per line. Section
// 10.4's Git table does exactly that, and seven shipped rows cited a sibling's
// line: `GitIndex.format` quoted GitObjectPack's `format:git_pack_v2` while
// production enforces `git_index`, `GitIndexEntry.mode` quoted GitHead's
// `mode:branch|detached|unborn` while production enforces `uint32`, and
// `GitIndexEntry.oid` quoted GitHead's nullable `oid`. Every rule then in place
// admitted them: the text is verbatim, it begins at the cited line, it sits in
// clause 10.4, and it contains the member name as a substring.
//
// Only shapes whose citations land on a declaration row that does not name the
// member need an entry. A shape declared by a per-member table anchors itself:
// its citation lands on the table row whose first cell is the member.
var constraintRowDeclaringIdentifiers = map[string]string{
	"GitFeatures":                          "GitFeatures",
	"GitHead":                              "GitHead",
	"GitIndex":                             "GitIndex",
	"GitIndexEntry":                        "GitIndexEntry",
	"GitObjectPack":                        "GitObjectPack",
	"GitRemote":                            "GitRemote",
	"GitSubmodule":                         "GitSubmodule",
	"ManifestEntry.directory":              "type = directory",
	"ManifestEntry.file":                   "type = file",
	"ManifestEntry.hardlink":               "type = hardlink",
	"ManifestEntry.symlink":                "type = symlink",
	"WorkspaceMember.git":                  "kind = git",
	"WorkspaceMember.managed_tree":         "kind = managed_tree",
	"WorkspaceSnapshotMember.git":          "kind = git",
	"WorkspaceSnapshotMember.managed_tree": "kind = managed_tree",
}

// constraintRowTableAnchorExemptions names the exact citations that land on a
// table row declaring neither the member nor the shape, together with why. The
// map is keyed by shape, member, and cited line, and is asserted exactly, so an
// exemption is a disclosed, argued, single-citation decision rather than a class
// the gate quietly stops covering.
var constraintRowTableAnchorExemptions = map[string]string{
	"Session Record 1.0.0.name@L363": "the Session Record clause defers the name grammar to a " +
		"section reference that contains no grammar; the grammar itself is written in the " +
		"Section 2.1 Terms table under the term \"Session name\", so the row quotes the term row " +
		"that carries it. Quoting it is what makes the indirection visible instead of flattened.",
	"Session Record 2.0.0 and 3.0.0.name@L363": "same Section 2.1 Terms-table indirection as " +
		"Session Record 1.0.0: the name grammar lives under the term, not under the member.",
}

// tableAnchorExemptionKey identifies one exempted citation.
func tableAnchorExemptionKey(row documentedConstraintRow, line int) string {
	return row.shape + "." + row.member + "@L" + strconv.Itoa(line)
}

// declaringRowFailure holds a citation that lands on a Markdown table body row
// to the row that declares what is being cited: either the member itself, or
// the identifier under which the pinned document declares the shape. It returns
// the failure, if any, and the exemption key it consumed, if any.
//
// A citation that lands outside every table body row is not constrained here;
// prose, headings, and the tables' own header rows are held only by the clause
// anchor and the verbatim/member rules.
func declaringRowFailure(document *specdoc.Document, row documentedConstraintRow, line int) (string, string) {
	table, ok := document.TableRowAt(line)
	if !ok {
		return "", ""
	}
	if table.Identifier == row.member {
		return "", ""
	}
	key := tableAnchorExemptionKey(row, line)
	if _, exempt := constraintRowTableAnchorExemptions[key]; exempt {
		return "", key
	}
	declaring, pinned := constraintRowDeclaringIdentifiers[row.shape]
	if !pinned {
		return fmt.Sprintf("cites the %s table row that declares %s, which is neither member %s "+
			"nor a declared identity of shape %s",
			strconv.Quote(table.Header), strconv.Quote(table.Identifier),
			strconv.Quote(row.member), strconv.Quote(row.shape)), ""
	}
	if table.Identifier != declaring {
		return fmt.Sprintf("cites the %s table row that declares %s, but %s is declared as %s",
			strconv.Quote(table.Header), strconv.Quote(table.Identifier),
			strconv.Quote(row.shape), strconv.Quote(declaring)), ""
	}
	return "", ""
}

// verifyRowAgainstPinnedSpecification returns every reason the row's Pinned
// SPEC declaration is not supported by the pinned document.
func verifyRowAgainstPinnedSpecification(document *specdoc.Document, row documentedConstraintRow) []string {
	entries, err := parseSpecExcerptCell(row.specExcerpt)
	if err != nil {
		return []string{"unreadable pinned SPEC declaration: " + err.Error()}
	}

	sections, pinned := constraintRowSpecSections[row.shape]
	var failures []string
	anchored := false
	for index, entry := range entries {
		position := fmt.Sprintf("entry %d (L%d)", index+1, entry.line)
		if !pinned {
			failures = append(failures, position+" belongs to shape "+strconv.Quote(row.shape)+
				", which pins no SPEC.md clause; every shape must declare the clauses its citations may come from")
		} else if section, ok := document.SectionID(entry.line); !ok {
			failures = append(failures, position+" cites a line outside every numbered SPEC.md clause")
		} else if !containsSection(sections, section) {
			failures = append(failures, fmt.Sprintf("%s cites Section %s, but %s is declared in Section %s",
				position, section, row.shape, strings.Join(sections, " or ")))
		}
		if failure, _ := declaringRowFailure(document, row, entry.line); failure != "" {
			failures = append(failures, position+" "+failure)
		}
		switch entry.kind {
		case verbatimEntryKind:
			text := decodeSpecExcerpt(entry.text)
			if strings.TrimSpace(text) == "" {
				failures = append(failures, position+" quotes nothing")
				continue
			}
			lines := document.QuoteLines(text)
			if len(lines) == 0 {
				failures = append(failures, position+" quotes text absent from the pinned SPEC.md: "+strconv.Quote(text))
				continue
			}
			if !containsLine(lines, entry.line) {
				failures = append(failures, fmt.Sprintf("%s quotes text that begins at %v, not the declared line: %s",
					position, lines, strconv.Quote(text)))
				continue
			}
			if strings.Contains(text, row.member) {
				anchored = true
			}
		case paraphraseEntryKind:
			raw, ok := document.Line(entry.line)
			if !ok {
				failures = append(failures, position+" paraphrases a line outside the pinned SPEC.md")
				continue
			}
			if strings.Contains(raw, row.member) {
				anchored = true
				continue
			}
			failures = append(failures, fmt.Sprintf("%s paraphrases a line that never names %q: %s",
				position, row.member, strconv.Quote(strings.TrimSpace(raw))))
		default:
			failures = append(failures, position+" has an unknown entry kind "+entry.kind)
		}
	}
	if !anchored {
		failures = append(failures, "no entry names the member; the row could quote any line of the specification")
	}
	return failures
}

func containsSection(sections []string, want string) bool {
	for _, section := range sections {
		if section == want {
			return true
		}
	}
	return false
}

func containsLine(lines []int, want int) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

// TestArtifactQuotesAreVerbatimPinnedSpecificationText closes the same hole
// outside the row table. Curly quotation marks in the artifact are reserved for
// pinned specification text, including the cross-member rule table and the
// recorded semver decision, so a fabricated quote anywhere in the file reddens.
func TestArtifactQuotesAreVerbatimPinnedSpecificationText(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}

	packageDirectory, _ := packageProductionFiles(t)
	path := filepath.Join(packageDirectory, "testdata", "constraint-enumeration.md")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	quoted := regexp.MustCompile(`\x{201c}([^\x{201d}]*)\x{201d}`)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	var failures []string
	number := 0
	total := 0
	for scanner.Scan() {
		number++
		for _, match := range quoted.FindAllStringSubmatch(scanner.Text(), -1) {
			total++
			text := decodeSpecExcerpt(match[1])
			if strings.TrimSpace(text) == "" {
				failures = append(failures, fmt.Sprintf("line %d quotes nothing", number))
				continue
			}
			if !document.Contains(text) {
				failures = append(failures, fmt.Sprintf("line %d quotes text absent from the pinned SPEC.md: %s",
					number, strconv.Quote(text)))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if total == 0 {
		t.Fatal("found zero quoted excerpts in the constraint enumeration artifact")
	}
	if len(failures) != 0 {
		t.Fatalf("constraint enumeration quotes text the pinned specification does not contain:\n%s",
			strings.Join(failures, "\n"))
	}
}

// constraintEnumerationPath resolves the artifact this package's gates read.
func constraintEnumerationPath(t *testing.T) string {
	t.Helper()
	packageDirectory, _ := packageProductionFiles(t)
	return filepath.Join(packageDirectory, "testdata", "constraint-enumeration.md")
}

// plantConstraintEnumerationRow writes a copy of the real artifact with the
// Pinned SPEC declaration of one row replaced, and returns its path. The copy
// keeps every other row intact so a failure is attributable to the plant.
func plantConstraintEnumerationRow(t *testing.T, shape, member, replacement string) string {
	t.Helper()
	original, err := os.ReadFile(constraintEnumerationPath(t))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(original), "\n")
	planted := 0
	for index, line := range lines {
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := splitTableCells(line)
		if len(cells) != 7 {
			continue
		}
		trim := func(value string) string { return strings.Trim(strings.TrimSpace(value), "`") }
		if trim(cells[1]) != shape || trim(cells[2]) != member {
			continue
		}
		cells[5] = " " + replacement + " "
		lines[index] = strings.Join(cells, "|")
		planted++
	}
	if planted != 1 {
		t.Fatalf("planted %d rows for %s.%s, want exactly 1", planted, shape, member)
	}
	path := filepath.Join(t.TempDir(), "constraint-enumeration.md")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// verifyPlantedArtifact runs the real gate logic over a planted artifact copy
// and returns the failures it reports for the planted row.
func verifyPlantedArtifact(t *testing.T, path, shape, member string) []string {
	t.Helper()
	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}
	var failures []string
	found := false
	for _, row := range readConstraintRows(t, path) {
		if row.shape != shape || row.member != member {
			continue
		}
		found = true
		failures = append(failures, verifyRowAgainstPinnedSpecification(document, row)...)
	}
	if !found {
		t.Fatalf("planted row %s.%s is not readable from %s", shape, member, path)
	}
	return failures
}

// TestPlantedConstraintEnumerationRowsRedden is the anti-vacuity proof. Each
// case plants exactly the defect the gate exists to catch and requires the gate
// to name the row and the offending text. Without these, a green suite would
// only prove the gate is reachable.
func TestPlantedConstraintEnumerationRowsRedden(t *testing.T) {
	t.Parallel()

	const shape, member = "Lease Record", "reason"

	for _, test := range []struct {
		name        string
		shape       string
		member      string
		replacement string
		contains    string
	}{
		{
			// The defect from the incident: a quoted word attributed to the
			// pinned document that the pinned document never contains.
			name:        "invented quote",
			replacement: "L1908 “\\| <code>reason</code> \\| enum \\| a digest of the lease reason \\|”",
			contains:    "quotes text absent from the pinned SPEC.md",
		},
		{
			name:        "invented quote among true ones",
			replacement: "L1908 “\\| <code>reason</code> \\| enum \\| <code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code> \\|”; L1908 “<code>reason</code> is a digest”",
			contains:    "quotes text absent from the pinned SPEC.md",
		},
		{
			// Real text, wrong line: the excerpt must be anchored, or a row
			// could quote any sentence of an 883 KiB document.
			name:        "true quote at the wrong line",
			replacement: "L1907 “\\| <code>reason</code> \\| enum \\| <code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code> \\|”",
			contains:    "not the declared line",
		},
		{
			// Real text that never names the member: without the anchor rule a
			// row could satisfy the gate with an unrelated true sentence.
			name:        "true quote that never names the member",
			replacement: "L1892 “Lease Record and ownership”",
			contains:    "no entry names the member",
		},
		{
			name:        "paraphrase of a line that never names the member",
			replacement: "L1892 paraphrase: the reason enum is closed",
			contains:    "never names",
		},
		{
			name:        "paraphrase outside the document",
			replacement: "L99999999 paraphrase: the reason enum is closed",
			contains:    "outside the pinned SPEC.md",
		},
		{
			name:        "empty quote",
			replacement: "L1908 “”",
			contains:    "quotes nothing",
		},
		{
			name:        "unanchored prose with no line reference",
			replacement: "“<code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code>”",
			contains:    "does not start with a line reference",
		},
		{
			name:        "bare prose, the pre-fix contract",
			replacement: "closed four-member reason enum",
			contains:    "does not start with a line reference",
		},
		{
			name:        "empty cell",
			replacement: "",
			contains:    "cell is empty",
		},
		{
			// The shape-blindness mutant. `size:uint53[1..4194304]` is real,
			// verbatim, correctly located at 4622, and contains the member name
			// — it is BlobChunk's clause in Section 10.2, and it carries a bound
			// ManifestEntry.file.size does not have. Every other rule admits it.
			name:        "true quote from another shape's clause",
			shape:       "ManifestEntry.file",
			member:      "size",
			replacement: "L4622 “<code>size:uint53[1..4194304]</code>”",
			contains:    "cites Section 10.2, but ManifestEntry.file is declared in Section 10.4",
		},
		{
			// Two halves of real SPEC.md text taken from either side of the
			// blank line at 1914. Collapsing every whitespace run to one space
			// used to admit this: a quote could stitch the end of a table to the
			// paragraph after it and still be reported as verbatim.
			name:   "quote stitched across a blank line",
			shape:  "Lease Record",
			member: "extensions",
			replacement: "L1913 “\\| <code>extensions</code> \\| object \\| Reverse-DNS extension keys only \\| " +
				"The object is closed; every row is required, including nullable fields.”",
			contains: "quotes text absent from the pinned SPEC.md",
		},
		{
			// The F4 class in a per-member table: L1911 is the
			// `created_by_host_id` row, whose constraint text names
			// `issued_by_host_id`. The quote is verbatim, begins at the declared
			// line, sits in clause 5.3, and contains the member name, so every
			// rule except the declaring-row anchor admits it while the row
			// imports a different member's declaration.
			name:        "true quote from a sibling member's table row",
			shape:       "Lease Record",
			member:      "issued_by_host_id",
			replacement: "L1911 “MUST equal <code>issued_by_host_id</code>”",
			contains:    "declares \"created_by_host_id\"",
		},
		{
			// The within-table stitch: two adjacent Lease Record rows joined
			// across the newline between them. Both halves are verbatim and the
			// quote begins at the declared line, so collapsing that newline to a
			// space admitted a row that imports the next member's constraint.
			name:   "quote stitched across a table row boundary",
			shape:  "Lease Record",
			member: "holder_host_id",
			replacement: "L1906 “\\| <code>holder_host_id</code> \\| UUIDv7 \\| Proposed owner \\| " +
				"\\| <code>predecessor_lease_id</code> \\| UUIDv4 or null \\| Null only at epoch 1 \\|”",
			contains: "quotes text absent from the pinned SPEC.md",
		},
		{
			name:        "trailing text after a valid entry",
			replacement: "L1908 “<code>reason</code>” and also whatever we like",
			contains:    "trailing text after entry",
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			plantShape, plantMember := shape, member
			if test.shape != "" {
				plantShape, plantMember = test.shape, test.member
			}
			path := plantConstraintEnumerationRow(t, plantShape, plantMember, test.replacement)
			failures := verifyPlantedArtifact(t, path, plantShape, plantMember)
			if len(failures) == 0 {
				t.Fatalf("planted %s was admitted; the gate proves nothing", test.name)
			}
			joined := strings.Join(failures, "\n")
			if !strings.Contains(joined, test.contains) {
				t.Fatalf("planted %s reported %q, want a failure containing %q", test.name, joined, test.contains)
			}
		})
	}
}

// TestEveryConstraintEnumerationShapePinsItsSpecificationClause asserts the
// clause pin exactly against the artifact. An unpinned shape would otherwise
// skip the clause check silently, and a stale entry would leave a shape's
// citations anchored to a clause the artifact no longer uses.
func TestEveryConstraintEnumerationShapePinsItsSpecificationClause(t *testing.T) {
	t.Parallel()

	rows := readConstraintRows(t, constraintEnumerationPath(t))
	if len(rows) == 0 {
		t.Fatal("read zero constraint enumeration rows")
	}
	present := make(map[string]struct{})
	for _, row := range rows {
		present[row.shape] = struct{}{}
	}

	var failures []string
	for shape := range present {
		if _, pinned := constraintRowSpecSections[shape]; !pinned {
			failures = append(failures, "shape "+strconv.Quote(shape)+" pins no SPEC.md clause")
		}
	}
	for shape, sections := range constraintRowSpecSections {
		if _, used := present[shape]; !used {
			failures = append(failures, "pinned clause "+strings.Join(sections, "/")+" names shape "+
				strconv.Quote(shape)+", which the enumeration no longer carries")
		}
		if len(sections) == 0 {
			failures = append(failures, "shape "+strconv.Quote(shape)+" pins an empty clause set")
		}
	}
	sort.Strings(failures)
	if len(failures) != 0 {
		t.Fatalf("constraint enumeration clause pins do not match the artifact:\n%s",
			strings.Join(failures, "\n"))
	}
}

// TestClauseAnchorRefusesEveryForeignSectionForOneRow proves the clause anchor
// is a bound and not a single lucky case. Every other shape's clause is tried
// against one row, and the row's own clause is required to remain admissible —
// a check that refused everything would satisfy the first half alone.
func TestClauseAnchorRefusesEveryForeignSectionForOneRow(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}

	const shape, member = "ManifestEntry.file", "size"
	own := constraintRowSpecSections[shape]
	if len(own) != 1 || own[0] != "10.4" {
		t.Fatalf("%s pins %v, the fixture assumes 10.4", shape, own)
	}

	foreign := make(map[string]struct{})
	for otherShape, sections := range constraintRowSpecSections {
		if otherShape == shape {
			continue
		}
		for _, section := range sections {
			if section != "10.4" {
				foreign[section] = struct{}{}
			}
		}
	}
	if len(foreign) < 8 {
		t.Fatalf("derived %d foreign clauses, want the full spread", len(foreign))
	}

	// One representative line per foreign clause, taken from the document
	// itself so the quote is genuinely verbatim and genuinely elsewhere.
	exercised := 0
	for section := range foreign {
		line, text, ok := firstLineNamingMemberInSection(document, section, member)
		if !ok {
			continue
		}
		exercised++
		row := documentedConstraintRow{
			shape:       shape,
			member:      member,
			constraint:  "Enforced exactly as declared before identity calculation or verification.",
			callSite:    "validateManifestEntries",
			specExcerpt: fmt.Sprintf("L%d “%s”", line, strings.ReplaceAll(text, "|", `\|`)),
		}
		failures := verifyRowAgainstPinnedSpecification(document, row)
		if len(failures) == 0 {
			t.Fatalf("a verbatim quote of Section %s line %d was admitted for %s.%s", section, line, shape, member)
		}
		if !strings.Contains(strings.Join(failures, "\n"), "cites Section "+section) {
			t.Fatalf("Section %s citation reported %v, want the foreign-clause refusal", section, failures)
		}
	}

	if exercised < 10 {
		t.Fatalf("only %d of %d foreign clauses produced a plantable line; the bound is too thin",
			exercised, len(foreign))
	}
	t.Logf("refused a verbatim %q citation from %d of %d foreign clauses", member, exercised, len(foreign))

	// The row as shipped must still pass, or the anchor would be refusing
	// everything rather than refusing foreign clauses.
	for _, row := range readConstraintRows(t, constraintEnumerationPath(t)) {
		if row.shape != shape || row.member != member {
			continue
		}
		if failures := verifyRowAgainstPinnedSpecification(document, row); len(failures) != 0 {
			t.Fatalf("the shipped %s.%s row was refused by its own clause anchor: %v", shape, member, failures)
		}
		return
	}
	t.Fatalf("shipped row %s.%s not found", shape, member)
}

// firstLineNamingMemberInSection finds a verbatim, uniquely locatable single
// line inside a clause, preferring one that also names the member so the
// planted citation differs from a legitimate one only in its clause. Clauses
// that never mention the member fall back to any unique line: the point of the
// plant is the foreign clause, and the refusal is asserted by clause number.
func firstLineNamingMemberInSection(document *specdoc.Document, section, member string) (int, string, bool) {
	fallbackLine, fallbackText := 0, ""
	for number := 1; number <= document.LineCount(); number++ {
		id, ok := document.SectionID(number)
		if !ok || id != section {
			continue
		}
		raw, _ := document.Line(number)
		text := strings.TrimSpace(raw)
		if len(text) < 24 || strings.HasPrefix(text, "#") {
			continue
		}
		if lines := document.QuoteLines(text); len(lines) != 1 || lines[0] != number {
			continue
		}
		if strings.Contains(text, member) {
			return number, text, true
		}
		if fallbackLine == 0 {
			fallbackLine, fallbackText = number, text
		}
	}
	if fallbackLine != 0 {
		return fallbackLine, fallbackText, true
	}
	return 0, "", false
}

// TestMarkedParaphraseRowIsAdmittedWhenItNamesItsLine proves the paraphrase
// form is usable rather than merely refused, so a row that genuinely restates
// the specification has an honest way to say so.
func TestMarkedParaphraseRowIsAdmittedWhenItNamesItsLine(t *testing.T) {
	t.Parallel()

	const shape, member = "Lease Record", "reason"
	path := plantConstraintEnumerationRow(t, shape, member,
		"L1908 paraphrase: the four takeover reasons are a closed enum")
	if failures := verifyPlantedArtifact(t, path, shape, member); len(failures) != 0 {
		t.Fatalf("a paraphrase naming the declaring line was refused: %v", failures)
	}
}

// TestUnmodifiedConstraintEnumerationIsAdmitted pins the other half of the
// anti-vacuity pair: the gate that reddens every planted defect above still
// accepts the artifact as shipped.
func TestUnmodifiedConstraintEnumerationIsAdmitted(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}
	rows := readConstraintRows(t, constraintEnumerationPath(t))
	if len(rows) < 300 {
		t.Fatalf("read %d rows, want the full enumeration", len(rows))
	}
	for _, row := range rows {
		if failures := verifyRowAgainstPinnedSpecification(document, row); len(failures) != 0 {
			t.Fatalf("shipped row %s.%s was refused: %v", row.shape, row.member, failures)
		}
	}
}

// TestSpecExcerptComparisonRefusesASwappedSpecification proves the comparison
// cannot be satisfied by substituting the document. Every excerpt would still
// "match" a specification written to contain it, so the digest gate is what
// makes the comparison mean anything.
func TestSpecExcerptComparisonRefusesASwappedSpecification(t *testing.T) {
	t.Parallel()

	pinned := specdoc.Bytes()

	// A perturbation inside text the enumeration actually quotes.
	const quoted = "| <code>reason</code> | enum | <code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovery</code> |"
	index := strings.Index(string(pinned), quoted)
	if index < 0 {
		t.Fatal("pinned document does not contain the Lease Record reason row")
	}
	perturbed := append([]byte(nil), pinned...)
	copy(perturbed[index:], []byte("| <code>reason</code> | enum | <code>create</code>, <code>graceful_takeover</code>, <code>force_takeover</code>, <code>recovered</code> |"))

	if _, err := specdoc.Parse(perturbed); err == nil {
		t.Fatal("a perturbed specification was accepted; excerpt comparison would confirm whatever it says")
	}

	// A document that contains an invented quote is refused for the same
	// reason: only the pinned bytes are admissible.
	forged := []byte("| <code>reason</code> | enum | a digest of the lease reason |\n")
	if _, err := specdoc.Parse(forged); err == nil {
		t.Fatal("a forged specification was accepted")
	}
}

// constraintRowTableAnchorUsage replays the declaring-row anchor over the
// shipped artifact and reports which pinned declaring identifiers and which
// exemptions were actually exercised.
func constraintRowTableAnchorUsage(t *testing.T, document *specdoc.Document) (map[string]struct{}, map[string]struct{}) {
	t.Helper()
	usedIdentifiers := make(map[string]struct{})
	usedExemptions := make(map[string]struct{})
	for _, row := range readConstraintRows(t, constraintEnumerationPath(t)) {
		entries, err := parseSpecExcerptCell(row.specExcerpt)
		if err != nil {
			t.Fatalf("row %s.%s has an unreadable declaration cell: %v", row.shape, row.member, err)
		}
		for _, entry := range entries {
			table, ok := document.TableRowAt(entry.line)
			if !ok || table.Identifier == row.member {
				continue
			}
			if _, consumed := constraintRowTableAnchorExemptions[tableAnchorExemptionKey(row, entry.line)]; consumed {
				usedExemptions[tableAnchorExemptionKey(row, entry.line)] = struct{}{}
				continue
			}
			if declaring, pinned := constraintRowDeclaringIdentifiers[row.shape]; pinned && declaring == table.Identifier {
				usedIdentifiers[row.shape] = struct{}{}
			}
		}
	}
	return usedIdentifiers, usedExemptions
}

// TestEveryConstraintEnumerationDeclaringIdentifierIsExercised asserts the
// declaring-row pins exactly against the artifact. A pin nothing exercises is a
// stale claim about where a shape is declared, and an exemption nothing
// exercises is an argued hole the artifact no longer needs; both must be
// removed rather than left as a standing free pass.
func TestEveryConstraintEnumerationDeclaringIdentifierIsExercised(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}
	usedIdentifiers, usedExemptions := constraintRowTableAnchorUsage(t, document)

	var failures []string
	for shape, declaring := range constraintRowDeclaringIdentifiers {
		if _, used := usedIdentifiers[shape]; !used {
			failures = append(failures, "shape "+strconv.Quote(shape)+" pins declaring identifier "+
				strconv.Quote(declaring)+", which no citation in the artifact lands on")
		}
	}
	for key, reason := range constraintRowTableAnchorExemptions {
		if _, used := usedExemptions[key]; !used {
			failures = append(failures, "exemption "+strconv.Quote(key)+" is unused: "+reason)
		}
	}
	if len(usedIdentifiers) == 0 {
		failures = append(failures, "no declaring identifier was exercised at all; the anchor is not reached")
	}
	sort.Strings(failures)
	if len(failures) != 0 {
		t.Fatalf("constraint enumeration declaring-row pins do not match the artifact:\n%s",
			strings.Join(failures, "\n"))
	}
	t.Logf("exercised %d declaring identifiers and %d exemptions", len(usedIdentifiers), len(usedExemptions))
}

// gitTypeMemberDeclarator matches one `<code>member:constraint</code>` span of a
// Section 10.4 Git type declaration row.
var gitTypeMemberDeclarator = regexp.MustCompile(`<code>([a-z_][a-z0-9_]*):([^<]*)</code>`)

// gitTypeDeclarations derives, from the pinned document itself, every Section
// 10.4 type-declaration row and the member declarators it carries.
func gitTypeDeclarations(t *testing.T, document *specdoc.Document) map[string]map[string]declaredMember {
	t.Helper()
	declarations := make(map[string]map[string]declaredMember)
	for number := 1; number <= document.LineCount(); number++ {
		table, ok := document.TableRowAt(number)
		if !ok || table.Header != "Type" {
			continue
		}
		if section, ok := document.SectionID(number); !ok || section != "10.4" {
			continue
		}
		raw, _ := document.Line(number)
		members := make(map[string]declaredMember)
		for _, match := range gitTypeMemberDeclarator.FindAllStringSubmatch(raw, -1) {
			members[match[1]] = declaredMember{line: number, text: match[0]}
		}
		if len(members) != 0 {
			declarations[table.Identifier] = members
		}
	}
	if len(declarations) != 7 {
		t.Fatalf("derived %d Section 10.4 Git type declarations, want the seven the table carries", len(declarations))
	}
	return declarations
}

type declaredMember struct {
	line int
	text string
}

// TestDeclaringRowAnchorRefusesEverySiblingRowOfTheGitTable is the bound proof
// for the declaring-row anchor. Section 10.4 declares seven Git types one per
// table row, so a member named by two of them can be cited from the wrong row
// with text that is verbatim, correctly located, inside the right clause, and
// containing the member name. Seven shipped rows were exactly that. This plants
// every such sibling pair the document admits, in both directions, and requires
// each to be refused by name.
func TestDeclaringRowAnchorRefusesEverySiblingRowOfTheGitTable(t *testing.T) {
	t.Parallel()

	document, err := specdoc.Load()
	if err != nil {
		t.Fatalf("load pinned specification: %v", err)
	}
	declarations := gitTypeDeclarations(t, document)

	shipped := make(map[string]struct{})
	for _, row := range readConstraintRows(t, constraintEnumerationPath(t)) {
		shipped[row.shape+"."+row.member] = struct{}{}
	}

	planted := 0
	for owner, members := range declarations {
		for member, declared := range members {
			if _, enumerated := shipped[owner+"."+member]; !enumerated {
				continue
			}
			for sibling, siblingMembers := range declarations {
				if sibling == owner {
					continue
				}
				foreign, shared := siblingMembers[member]
				if !shared {
					continue
				}
				planted++
				name := owner + "." + member + " cited from " + sibling
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					replacement := fmt.Sprintf("L%d “%s”", foreign.line, strings.ReplaceAll(foreign.text, "|", `\|`))
					path := plantConstraintEnumerationRow(t, owner, member, replacement)
					failures := verifyPlantedArtifact(t, path, owner, member)
					if len(failures) == 0 {
						t.Fatalf("a verbatim %s declarator quoted from %s's row was admitted for %s.%s; "+
							"the declaring-row anchor proves nothing", member, sibling, owner, member)
					}
					joined := strings.Join(failures, "\n")
					want := "declares " + strconv.Quote(sibling)
					if !strings.Contains(joined, want) {
						t.Fatalf("planted %s reported %q, want a failure containing %q", name, joined, want)
					}
					// The declarator planted must be genuinely different text, or
					// the mutant would be a rename rather than a retarget.
					if foreign.text == declared.text && foreign.line == declared.line {
						t.Fatalf("planted %s is not a retarget: same line and text", name)
					}
				})
			}
		}
	}
	if planted < 12 {
		t.Fatalf("planted %d sibling retargets, want every shared-member pair the Git table carries", planted)
	}
	t.Logf("planted %d sibling retargets across the Section 10.4 Git table", planted)
}
