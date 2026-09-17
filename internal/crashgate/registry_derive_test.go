package crashgate

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

// rangeToken matches one Section 13.13 boundary range token such as
// CR-MAT-01..08 or CR-FORCE-TB-01..04: the CR- family prefix with
// its zero-padded start and end indexes.
var rangeToken = regexp.MustCompile(`(CR-[A-Z-]+?)-(\d+)\.\.(\d+)`)

// expandRange expands one parsed range token into its boundary IDs.
// The indexes are zero-padded two-digit, as the registry writes
// them.
func expandRange(prefix string, start, end int) []string {
	ids := make([]string, 0, end-start+1)
	for index := start; index <= end; index++ {
		ids = append(ids, fmt.Sprintf("%s-%02d", prefix, index))
	}
	return ids
}

// TestRegistryDerivesFromPinnedSpec proves the registry enumerates
// exactly the Section 13.13 crash boundaries: every boundary ID is
// parsed from the pinned specification text (the twelve registry
// table ranges plus the CR-CLONE prose range) and compared against
// the registry with no hand-typed ID on either side. A specification
// revision that adds, removes, or renumbers a boundary fails here
// until the registry classification is deliberately updated.
func TestRegistryDerivesFromPinnedSpec(t *testing.T) {
	document, err := specdoc.LoadV060()
	if err != nil {
		t.Fatalf("LoadV060 error = %v (the pinned text is unreadable or digest-drifted)", err)
	}
	var tableIDs, proseIDs []string
	var tableRanges, proseRanges int
	for line := 1; line <= document.LineCount(); line++ {
		section, ok := document.SectionID(line)
		if !ok || section != "13.13" {
			continue
		}
		text, ok := document.Line(line)
		if !ok {
			t.Fatalf("pinned document line %d is unreadable", line)
		}
		trimmed := strings.TrimSpace(text)
		if !strings.Contains(trimmed, "..") {
			continue
		}
		for _, match := range rangeToken.FindAllStringSubmatch(trimmed, -1) {
			start, startErr := strconv.Atoi(match[2])
			end, endErr := strconv.Atoi(match[3])
			if startErr != nil || endErr != nil || start < 1 || end < start {
				t.Fatalf("line %d carries a malformed range token %q", line, match[0])
			}
			if strings.HasPrefix(trimmed, "|") {
				tableRanges++
				tableIDs = append(tableIDs, expandRange(match[1], start, end)...)
			} else {
				proseRanges++
				proseIDs = append(proseIDs, expandRange(match[1], start, end)...)
			}
		}
	}
	if tableRanges == 0 {
		t.Fatalf("no registry-table range token parsed from Section 13.13")
	}
	if proseRanges == 0 {
		t.Fatalf("no prose range token parsed from Section 13.13 (the CR-CLONE range)")
	}
	parsed := append(append([]string{}, tableIDs...), proseIDs...)
	parsedSet := make(map[string]struct{}, len(parsed))
	for _, id := range parsed {
		if _, duplicate := parsedSet[id]; duplicate {
			t.Fatalf("specification parses boundary %s twice", id)
		}
		parsedSet[id] = struct{}{}
	}
	registered := Registry()
	registeredSet := make(map[string]struct{}, len(registered))
	for _, boundary := range registered {
		if _, duplicate := registeredSet[boundary.ID]; duplicate {
			t.Fatalf("registry lists boundary %s twice", boundary.ID)
		}
		registeredSet[boundary.ID] = struct{}{}
	}
	var missing, extra []string
	for id := range parsedSet {
		if _, ok := registeredSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	for id := range registeredSet {
		if _, ok := parsedSet[id]; !ok {
			extra = append(extra, id)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) != 0 || len(extra) != 0 {
		t.Fatalf("registry drifts from the pinned Section 13.13 table: missing %q, extra %q", missing, extra)
	}
	t.Logf("derived %d table IDs in %d ranges and %d prose IDs in %d ranges; registry matches exactly",
		len(tableIDs), tableRanges, len(proseIDs), proseRanges)
}

// TestRegistryStructure audits the classification shape: every
// boundary carries its family prefix, a phase, a durable-write
// class, and at least one path; every path names the direct or
// task-board vocabulary; reachable paths name exactly a driver and
// not-applicable paths name exactly an owner; every path carries
// its honest note.
func TestRegistryStructure(t *testing.T) {
	drivers := map[string]struct{}{
		DriverCheckpointCapture: {},
		DriverJournalCreate:     {},
		DriverJournalTransfer:   {},
		DriverJournalValidate:   {},
		DriverJournalPrepareOp:  {},
		DriverJournalToPrepared: {},
		DriverJournalActivation: {},
		DriverJournalFinalize:   {},
		DriverJournalDormant:    {},
	}
	registered := Registry()
	if len(registered) == 0 {
		t.Fatal("registry is empty")
	}
	reachableIDs := 0
	reachablePaths := 0
	for _, boundary := range registered {
		if boundary.ID == "" || boundary.Family == "" || boundary.Phase == "" || boundary.DurableWrite == "" {
			t.Fatalf("boundary %+v lacks ID, family, phase, or durable-write class", boundary)
		}
		if !strings.HasPrefix(boundary.ID, boundary.Family+"-") {
			t.Fatalf("boundary %s does not carry its family prefix %s", boundary.ID, boundary.Family)
		}
		switch boundary.DurableWrite {
		case WriteCheckpoint, WriteJournal, WriteExternal:
		default:
			t.Fatalf("boundary %s writes unknown class %q", boundary.ID, boundary.DurableWrite)
		}
		if len(boundary.Paths) == 0 {
			t.Fatalf("boundary %s lists no path", boundary.ID)
		}
		seen := map[string]struct{}{}
		reachable := false
		for _, path := range boundary.Paths {
			switch path.Name {
			case PathDirect, PathTaskBoard:
			default:
				t.Fatalf("boundary %s names path %q, want direct or task_board", boundary.ID, path.Name)
			}
			if _, duplicate := seen[path.Name]; duplicate {
				t.Fatalf("boundary %s repeats path %s", boundary.ID, path.Name)
			}
			seen[path.Name] = struct{}{}
			if path.Note == "" {
				t.Fatalf("boundary %s path %s carries no note", boundary.ID, path.Name)
			}
			if path.Reachable {
				reachable = true
				reachablePaths++
				if _, ok := drivers[path.Driver]; !ok {
					t.Fatalf("boundary %s path %s names unknown driver %q", boundary.ID, path.Name, path.Driver)
				}
				if path.Owner != "" {
					t.Fatalf("boundary %s path %s is reachable and still names owner %q", boundary.ID, path.Name, path.Owner)
				}
			} else {
				if path.Driver != "" {
					t.Fatalf("boundary %s path %s is not applicable and still names driver %q", boundary.ID, path.Name, path.Driver)
				}
				if path.Owner == "" {
					t.Fatalf("boundary %s path %s is not applicable and names no owner", boundary.ID, path.Name)
				}
			}
		}
		if reachable {
			reachableIDs++
		}
	}
	t.Logf("registry holds %d boundaries (%d reachable) over %d reachable paths",
		len(registered), reachableIDs, reachablePaths)
}
