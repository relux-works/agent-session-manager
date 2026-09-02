package canonicaljson

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const payloadInventoryOverrideVersion = "4.0.0"

type payloadInventory struct {
	byDefinition map[string]map[string][]string
	declarations map[string]string
}

// readSessionEventPayloadInventory parses the specification-extracted payload
// member artifact. Rows are grouped by the Definition column: "default" is the
// payload used at every contract version the catalog assigns to the event type,
// and "4.0.0-override" replaces it at Session Event 4.0.0.
func readSessionEventPayloadInventory(t *testing.T, path string) payloadInventory {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	inventory := payloadInventory{
		byDefinition: map[string]map[string][]string{},
		declarations: map[string]string{},
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "| `") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) != 6 {
			t.Fatalf("payload inventory row has %d cells, want 4: %s", len(cells)-2, line)
		}
		trimCode := func(value string) string {
			return strings.Trim(strings.TrimSpace(value), "`")
		}
		definition := trimCode(cells[1])
		eventType := trimCode(cells[2])
		member := trimCode(cells[3])
		// The artifact keeps the specification's own &#124; entity so a declared
		// alternation stays inside one Markdown cell.
		declaration := strings.ReplaceAll(trimCode(cells[4]), "&#124;", "|")
		if definition != "default" && definition != payloadInventoryOverrideVersion+"-override" {
			t.Fatalf("payload inventory row has unknown definition %q", definition)
		}
		if eventType == "" || member == "" || declaration == "" {
			t.Fatalf("payload inventory row is incomplete: %s", line)
		}
		if !strings.HasPrefix(declaration, member+":") && !strings.HasPrefix(declaration, member+"=") {
			t.Fatalf("payload inventory row %s.%s does not quote its own member declaration: %q", eventType, member, declaration)
		}
		key := definition + " " + eventType + "." + member
		if previous, duplicate := inventory.declarations[key]; duplicate {
			t.Fatalf("payload inventory duplicates %s (%q and %q)", key, previous, declaration)
		}
		inventory.declarations[key] = declaration
		if inventory.byDefinition[definition] == nil {
			inventory.byDefinition[definition] = map[string][]string{}
		}
		inventory.byDefinition[definition][eventType] = append(inventory.byDefinition[definition][eventType], member)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(inventory.declarations) == 0 {
		t.Fatal("payload inventory artifact contains no rows")
	}
	return inventory
}

// TestSessionEventPayloadMembersMatchPinnedSpecInventory pins the one production
// requireExactMembers call whose member slice is computed rather than literal.
// validateSessionEvent closes each payload against sessionEventPayloadShapes, so
// the literal-argument constraint inventory cannot read those member sets; this
// test derives them from the specification-extracted artifact instead. Adding,
// dropping, renaming, or reordering any payload member fails here.
func TestSessionEventPayloadMembersMatchPinnedSpecInventory(t *testing.T) {
	t.Parallel()

	packageDirectory, _ := packageProductionFiles(t)
	inventory := readSessionEventPayloadInventory(t, filepath.Join(packageDirectory, "testdata", "session-event-payload-members.md"))
	defaults := inventory.byDefinition["default"]
	overrides := inventory.byDefinition[payloadInventoryOverrideVersion+"-override"]

	var mismatches []string
	produced := map[string]struct{}{}
	overridden := map[string]struct{}{}
	for version, shapes := range sessionEventPayloadShapes {
		for eventType, shape := range shapes {
			expected, source := defaults[eventType], "default"
			if version == payloadInventoryOverrideVersion {
				if replacement, ok := overrides[eventType]; ok {
					expected, source = replacement, payloadInventoryOverrideVersion+"-override"
					overridden[eventType] = struct{}{}
				}
			}
			if expected == nil {
				mismatches = append(mismatches, "production Session Event "+version+" payload "+eventType+" has no pinned inventory rows")
				continue
			}
			produced[eventType] = struct{}{}
			if strings.Join(shape.members, ",") != strings.Join(expected, ",") {
				mismatches = append(mismatches, "Session Event "+version+" payload "+eventType+
					" declares members ["+strings.Join(shape.members, ",")+"], "+source+" inventory pins ["+strings.Join(expected, ",")+"]")
			}
		}
	}

	// The artifact must not carry a row set production never uses, which would
	// let a dropped payload look inventoried.
	for eventType := range defaults {
		if _, ok := produced[eventType]; !ok {
			mismatches = append(mismatches, "pinned default payload "+eventType+" is absent from the production registry")
		}
	}
	for eventType := range overrides {
		if _, ok := overridden[eventType]; !ok {
			mismatches = append(mismatches, "pinned "+payloadInventoryOverrideVersion+" override payload "+eventType+
				" is absent from the production "+payloadInventoryOverrideVersion+" registry")
		}
	}

	sort.Strings(mismatches)
	if len(mismatches) != 0 {
		t.Fatalf("session event payload member drift:\n%s", strings.Join(mismatches, "\n"))
	}
}

// TestPinnedTerminalBackendIDMembersAreDeclaredScalarType fixes the exact rows
// behind the terminal-backend-id enforcement so the scalar type cannot be
// silently downgraded to a bare string in the inventory.
func TestPinnedTerminalBackendIDMembersAreDeclaredScalarType(t *testing.T) {
	t.Parallel()

	packageDirectory, _ := packageProductionFiles(t)
	inventory := readSessionEventPayloadInventory(t, filepath.Join(packageDirectory, "testdata", "session-event-payload-members.md"))
	for _, eventType := range []string{"terminal.created", "session.resumed"} {
		key := payloadInventoryOverrideVersion + "-override " + eventType + ".terminal_backend_id"
		declaration, ok := inventory.declarations[key]
		if !ok {
			t.Fatalf("payload inventory is missing %s", key)
		}
		if declaration != "terminal_backend_id:terminal-backend-id" {
			t.Fatalf("%s declares %q, want the pinned terminal-backend-id scalar type", key, declaration)
		}
	}
}
