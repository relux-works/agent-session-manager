package sessquery

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSelectorLiteralGrammar(t *testing.T) {
	idSelector := "id:" + idA
	valid := []struct {
		argument string
		key      string
		idKey    bool
		shaped   bool
		source   SelectorSource
		value    string
	}{
		{"build", "build", false, false, SourceBare, ""},
		{"Az-09._", "Az-09._", false, false, SourceBare, ""},
		{idA, idA, false, true, SourceBare, ""},
		{idSelector, idSelector, true, true, SourceBare, ""},
		{"build@local", "build", false, false, SourceLocal, ""},
		{idA + "@local", idA, false, true, SourceLocal, ""},
		{idSelector + "@local", idSelector, true, true, SourceLocal, ""},
		{"build@peer:workstation", "build", false, false, SourcePeerAlias, "workstation"},
		{"build@peer:Work Laptop", "build", false, false, SourcePeerAlias, "Work Laptop"},
		{"build@peer:Ops@east:თბილისი", "build", false, false, SourcePeerAlias, "Ops@east:თბილისი"},
		{"build@peer:100%", "build", false, false, SourcePeerAlias, "100%"},
		{"build@peer:" + strings.Repeat("a", 64), "build", false, false, SourcePeerAlias, strings.Repeat("a", 64)},
		{"build@id:" + hostA, "build", false, false, SourcePeerHost, hostA},
		{idSelector + "@id:" + hostA, idSelector, true, true, SourcePeerHost, hostA},
		{"ID:" + idA, "ID:" + idA, false, false, SourceBare, ""},
		{"build ", "build ", false, false, SourceBare, ""},
	}
	for _, row := range valid {
		parsed, err := ParseSelector(row.argument)
		if err != nil {
			t.Fatalf("ParseSelector(%q) error = %v", row.argument, err)
		}
		if parsed.Raw != row.argument || parsed.Key != row.key || parsed.IDKey != row.idKey ||
			parsed.UUIDShaped != row.shaped || parsed.Source != row.source || parsed.SourceValue != row.value {
			t.Fatalf("ParseSelector(%q) = %+v", row.argument, parsed)
		}
	}
	// The 134-character ceiling admits exactly 64 NAME + "@" + "peer:"
	// + 64 alias characters.
	ceiling := strings.Repeat("n", 64) + "@peer:" + strings.Repeat("a", 64)
	if len([]rune(ceiling)) != MaxSelectorRunes {
		t.Fatalf("ceiling fixture is %d runes", len([]rune(ceiling)))
	}
	if _, err := ParseSelector(ceiling); err != nil {
		t.Fatalf("ParseSelector(ceiling) error = %v", err)
	}
	invalid := []string{
		"",
		"@local",
		"@peer:workstation",
		"build@",
		"build@LOCAL",
		"build@local ",
		"build@localx",
		"build@host",
		"build@peer:",
		"build@peer:" + strings.Repeat("a", 65),
		"build@peer:bad\x01alias",
		"build@peer:\xff",
		"build@id:",
		"build@id:not-a-uuid",
		"build@id:" + lease,
		"build@id",
		"id:",
		"id:not-a-uuid",
		ceiling + "x",
		strings.Repeat("☃", 135),
	}
	for _, argument := range invalid {
		if _, err := ParseSelector(argument); !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("ParseSelector(%q) = %v, want invalid_arguments", argument, err)
		}
	}
}

func TestSelectorConfigurationValidation(t *testing.T) {
	local, _ := repository(t)
	peer, _ := repository(t)
	create(t, local, idA, "Local")
	create(t, peer, idB, "Remote")
	base := func() *Reader {
		return &Reader{
			Local:              local,
			LocalHostID:        hostB,
			AllowlistedPeerIDs: []string{hostA},
			Peers:              []PeerIndex{{hostA, peer}},
			Aliases:            map[string]string{hostA: "workstation"},
		}
	}
	if _, err := base().Resolve("Local"); err != nil {
		t.Fatalf("valid configuration refused: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(reader *Reader)
	}{
		{"malformed local host", func(reader *Reader) { reader.LocalHostID = "not-a-uuid" }},
		{"peer collides with local host", func(reader *Reader) { reader.Peers = []PeerIndex{{hostB, peer}} }},
		{"malformed peer host", func(reader *Reader) { reader.Peers = []PeerIndex{{"nope", peer}} }},
		{"malformed allowlist entry", func(reader *Reader) { reader.AllowlistedPeerIDs = []string{"nope"} }},
		{"malformed alias host", func(reader *Reader) { reader.Aliases = map[string]string{"nope": "ws"} }},
		{"empty alias mapping", func(reader *Reader) { reader.Aliases = map[string]string{hostA: ""} }},
		{"overlong alias mapping", func(reader *Reader) { reader.Aliases = map[string]string{hostA: strings.Repeat("a", 65)} }},
		{"control alias mapping", func(reader *Reader) { reader.Aliases = map[string]string{hostA: "a\tb"} }},
		{"duplicate alias mapping", func(reader *Reader) {
			reader.Aliases = map[string]string{hostA: "ws", hostB: "ws"}
		}},
	}
	for _, row := range cases {
		t.Run(row.name, func(t *testing.T) {
			reader := base()
			row.mutate(reader)
			if _, err := reader.Resolve("Local"); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("Resolve = %v, want invalid_config", err)
			}
			if _, err := reader.BuildPlan(PlanArgs{Selector: "Local", Action: ActionStatus}); !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("BuildPlan = %v, want invalid_config", err)
			}
		})
	}
	// Alias casing is exact: "Work" and "WORK" coexist as distinct
	// mappings, and each selects only its own source.
	t.Run("alias casing is exact", func(t *testing.T) {
		other, _ := repository(t)
		create(t, other, idB, "Other")
		reader := &Reader{
			Local:              local,
			AllowlistedPeerIDs: []string{hostA, hostB},
			Peers:              []PeerIndex{{hostA, peer}, {hostB, other}},
			Aliases:            map[string]string{hostA: "Work", hostB: "WORK"},
		}
		lower, err := reader.Resolve("Remote@peer:Work")
		if err != nil || lower.SessionID != idB || lower.PeerHostID != hostA {
			t.Fatalf("lowercase alias: %+v %v", lower, err)
		}
		upper, err := reader.Resolve("Other@peer:WORK")
		if err != nil || upper.SessionID != idB || upper.PeerHostID != hostB {
			t.Fatalf("uppercase alias: %+v %v", upper, err)
		}
		if _, err := reader.Resolve("Remote@peer:WORK"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("folded alias selected: %v", err)
		}
	})
	// A dangling alias mapping validates but selects nothing: the
	// mapped host carries no learned index.
	t.Run("dangling alias mapping", func(t *testing.T) {
		reader := base()
		reader.Aliases[hostB] = "nowhere"
		if _, err := reader.Resolve("Local"); err != nil {
			t.Fatalf("dangling mapping broke validation: %v", err)
		}
		if _, err := reader.Resolve("Local@peer:nowhere"); !errors.Is(err, ErrSourceNotFound) {
			t.Fatalf("dangling alias selected: %v", err)
		}
	})
}
