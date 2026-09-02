package canonicaljson

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// This file pins the SEMANTICS of every declared grammar this package compiles.
//
// The failure it exists to prevent: a regular expression is a matcher, and a
// test that refuses one value outside it proves the matcher is REACHABLE, not
// that the admitted language is the declared one. A derived sweep confirmed the
// gap: widening observationNamePattern from {1,7} to {1,8} segments, from {1,7}
// to {0,7} so a bare single-segment name is admitted, and adding a hyphen to its
// character class each left `go test ./...`, every seeded fuzz corpus,
// tracecheck, the declared-bounds gate and the closed-vocabulary pin green.
// boardLogicalIDPattern survived the same treatment at {0,127} -> {0,128}.
//
// The obligation is therefore derived from the pattern source itself: every
// anchor, every character class, every counted quantifier bound and every
// one-or-more quantifier in every production regexp is one dimension that must
// carry a witness which the UNwidened pattern refuses. Adding a pattern, or
// adding a class or a quantifier to an existing one, fails at derivation until
// its witness exists.

// grammarPattern is one compiled production grammar with the polarity of its
// use, derived from whether the production condition negates the match.
type grammarPattern struct {
	name   string
	source string
	// admitting is true when production refuses on !MatchString, so the risky
	// mutation is a WIDENING. It is false when production refuses on a match, so
	// the risky mutation is a NARROWING. Either way the witness below must be
	// refused, so both polarities assert the same way.
	admitting bool
}

// deriveGrammarPatterns returns every regexp compiled in the package production
// sources, with the polarity derived from its use rather than assumed.
func deriveGrammarPatterns(t *testing.T) map[string]grammarPattern {
	t.Helper()

	_, files := packageProductionAST(t)
	patterns := make(map[string]grammarPattern)
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, ok := declaration.(*ast.GenDecl)
			if !ok || general.Tok != token.VAR {
				continue
			}
			for _, specification := range general.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for index, name := range value.Names {
					if index >= len(value.Values) {
						continue
					}
					source, ok := compiledRegexpSource(value.Values[index])
					if !ok {
						continue
					}
					patterns[name.Name] = grammarPattern{name: name.Name, source: source}
				}
			}
		}
	}
	if len(patterns) == 0 {
		t.Fatal("derived zero compiled grammars from the package sources; the scanner is broken, not the package")
	}
	for name, pattern := range patterns {
		pattern.admitting = grammarIsUsedAsAdmission(t, files, name)
		patterns[name] = pattern
	}
	return patterns
}

func compiledRegexpSource(expression ast.Expr) (string, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "MustCompile" {
		return "", false
	}
	if package_, ok := selector.X.(*ast.Ident); !ok || package_.Name != "regexp" {
		return "", false
	}
	literal, ok := call.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	source, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	return source, true
}

// grammarIsUsedAsAdmission reports whether every production use of a pattern
// refuses on a negated match, which makes widening the risky direction.
func grammarIsUsedAsAdmission(t *testing.T, files []*ast.File, name string) bool {
	t.Helper()

	negated, total := 0, 0
	for _, file := range files {
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "MatchString" {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if !ok || receiver.Name != name {
				return true
			}
			total++
			return true
		})
		ast.Inspect(file, func(node ast.Node) bool {
			unary, ok := node.(*ast.UnaryExpr)
			if !ok || unary.Op != token.NOT {
				return true
			}
			call, ok := unary.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "MatchString" {
				return true
			}
			if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == name {
				negated++
			}
			return true
		})
	}
	if total == 0 {
		t.Fatalf("grammar %s is compiled but never matched against; it declares a language nothing enforces", name)
	}
	return negated == total
}

// grammarSite is one widenable decision inside a pattern source, located by
// byte range so the mutation that widens it can be applied mechanically.
type grammarSite struct {
	dimension string
	kind      string
	start     int
	end       int // exclusive
	minimum   int
	maximum   int
}

const (
	grammarAnchorStart   = "anchor.start"
	grammarAnchorEnd     = "anchor.end"
	grammarClass         = "class"
	grammarQuantifierMax = "quantifier.max"
	grammarQuantifierMin = "quantifier.min"
	grammarOneOrMore     = "one-or-more"
)

// deriveGrammarSites returns the widenable decisions of a pattern source.
//
// Anchors, character classes, counted quantifier bounds and one-or-more
// quantifiers are enumerated positionally, so inserting a class or a bound
// shifts nothing that already exists and adds exactly one new obligation. The
// derived label for a counted bound carries the IMPLEMENTATION number; the
// witness carries the number the pinned specification declares, and the coverage
// test requires them to agree, so widening a bound in production changes the
// obligation key and fails before any case runs.
func deriveGrammarSites(name, source string) []grammarSite {
	var sites []grammarSite
	add := func(label, kind string, start, end, minimum, maximum int) {
		sites = append(sites, grammarSite{dimension: label, kind: kind, start: start, end: end, minimum: minimum, maximum: maximum})
	}
	if strings.HasPrefix(source, "^") {
		add(grammarAnchorStart, grammarAnchorStart, 0, 1, 0, 0)
	}
	if strings.HasSuffix(source, "$") {
		add(grammarAnchorEnd, grammarAnchorEnd, len(source)-1, len(source), 0, 0)
	}
	classes, quantifiers, plusses := 0, 0, 0
	for index := 0; index < len(source); index++ {
		switch source[index] {
		case '\\':
			index++
		case '[':
			classes++
			end := endOfCharacterClass(source, index)
			add(fmt.Sprintf("class#%d", classes), grammarClass, index, end+1, 0, 0)
			index = end
		case '{':
			closing := strings.IndexByte(source[index:], '}')
			if closing < 0 {
				continue
			}
			minimum, maximum, ok := parseCountedQuantifier(source[index+1 : index+closing])
			if !ok {
				continue
			}
			quantifiers++
			if maximum >= 0 {
				add(fmt.Sprintf("quantifier#%d.max=%d", quantifiers, maximum), grammarQuantifierMax,
					index, index+closing+1, minimum, maximum)
			}
			if minimum >= 1 {
				add(fmt.Sprintf("quantifier#%d.min=%d", quantifiers, minimum), grammarQuantifierMin,
					index, index+closing+1, minimum, maximum)
			}
			index += closing
		case '+':
			plusses++
			add(fmt.Sprintf("one-or-more#%d", plusses), grammarOneOrMore, index, index+1, 0, 0)
		}
	}
	sort.SliceStable(sites, func(first, second int) bool { return sites[first].dimension < sites[second].dimension })
	return sites
}

func endOfCharacterClass(source string, start int) int {
	index := start + 1
	if index < len(source) && source[index] == '^' {
		index++
	}
	if index < len(source) && source[index] == ']' {
		index++
	}
	for ; index < len(source); index++ {
		if source[index] == '\\' {
			index++
			continue
		}
		if source[index] == ']' {
			return index
		}
	}
	return len(source) - 1
}

func parseCountedQuantifier(body string) (int, int, bool) {
	parts := strings.SplitN(body, ",", 2)
	minimum, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	if len(parts) == 1 {
		return minimum, minimum, true
	}
	rest := strings.TrimSpace(parts[1])
	if rest == "" {
		return minimum, -1, true
	}
	maximum, err := strconv.Atoi(rest)
	if err != nil {
		return 0, 0, false
	}
	return minimum, maximum, true
}

// widenGrammarSource applies the mechanical widening of one dimension: an anchor
// is dropped, a character class becomes any character, a counted maximum gains
// one, a counted minimum loses one, and a one-or-more becomes zero-or-more.
func widenGrammarSource(source string, site grammarSite) string {
	switch site.kind {
	case grammarAnchorStart, grammarAnchorEnd:
		return source[:site.start] + source[site.end:]
	case grammarClass:
		return source[:site.start] + "(?s:.)" + source[site.end:]
	case grammarQuantifierMax:
		return source[:site.start] + fmt.Sprintf("{%d,%d}", site.minimum, site.maximum+1) + source[site.end:]
	case grammarQuantifierMin:
		return source[:site.start] + fmt.Sprintf("{%d,%d}", site.minimum-1, site.maximum) + source[site.end:]
	case grammarOneOrMore:
		return source[:site.start] + "*" + source[site.end:]
	}
	return source
}

// narrowGrammarSource applies the mechanical narrowing of one character class:
// only its first declared item survives. A refusal-polarity grammar is dangerous
// when it matches LESS, so this is the mutation its witnesses must defeat.
func narrowGrammarSource(source string, site grammarSite) string {
	if site.kind != grammarClass {
		return source
	}
	body := source[site.start+1 : site.end-1]
	first := firstCharacterClassItem(body)
	if first == "" {
		return source
	}
	return source[:site.start] + "[" + first + "]" + source[site.end:]
}

func firstCharacterClassItem(body string) string {
	if body == "" {
		return ""
	}
	index := 0
	if body[index] == '\\' && index+1 < len(body) {
		index += 2
	} else {
		index++
	}
	if index+1 < len(body) && body[index] == '-' {
		index += 2
	}
	return body[:index]
}

// declaredGrammar records what the pinned specification says about one compiled
// grammar, and the production refusal that grammar emits.
//
// refusal is a substring of the message the production call site produces, so
// every witness below is attributed to THIS clause instead of passing on any
// refusal that happens to fire first.
type declaredGrammar struct {
	// reference is the grammar this package is required to implement, written
	// here independently of the production source. It is the oracle every
	// neighbour sweep below consults, so a widened production pattern cannot
	// move its own goalposts: TestEveryCompiledGrammarMatchesItsPinnedReference
	// requires the two to be identical, and every other test in this file reads
	// the reference.
	reference string
	// composition explains a reference the pinned document declares in more than
	// one clause, so composing them is disclosed instead of read as a citation.
	composition string
	// spec is the pinned SPEC declaration, quoted. Empty is not allowed:
	// implementationDefined records the reason instead.
	spec string
	// implementationDefined explains a grammar the pinned document does not
	// spell out. It is the honest alternative to inventing a citation.
	implementationDefined string
	refusal               string
	// probe is a value that reaches the refusal, used to locate the fixture
	// positions this grammar governs. Empty selects the universal probe, which
	// violates every admitting grammar in the package.
	probe string
}

// universalGrammarProbe violates every admitting grammar in this package: it is
// not lowercase, contains a space and a control byte, and starts with neither a
// letter nor a digit. It is used only to LOCATE the fixture positions a grammar
// governs; the witnesses that follow are what prove the grammar's dimensions.
const universalGrammarProbe = "\x01 INVALID PROBE \x02"

// declaredGrammars is asserted to cover exactly the derived pattern set by
// TestEveryCompiledGrammarIsDeclaredAndReachable, so a new regexp in production
// fails here until its specification provenance and refusal are recorded.
var declaredGrammars = map[string]declaredGrammar{
	"sessionNamePattern": {
		reference: `^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`,
		spec:      "Session name | A mesh-unique human alias of 1-64 characters matching [A-Za-z0-9][A-Za-z0-9._-]{0,63}.",
		refusal:   "Session Record name must match",
	},
	"boardLogicalIDPattern": {
		reference: `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`,
		spec:      "logical_id (1-128 characters matching [A-Za-z0-9][A-Za-z0-9._:-]{0,127})",
		refusal:   "Board Identity logical_id has invalid grammar",
	},
	"environmentNamePattern": {
		reference: `^[A-Za-z_][A-Za-z0-9_]{0,127}$`,
		spec:      "env_names | array<string>[0..64] | Sorted, unique names matching [A-Za-z_][A-Za-z0-9_]{0,127}",
		refusal:   "environment-name grammar",
	},
	"environmentIDPattern": {
		reference: `^[a-z][a-z0-9.-]{0,63}$`,
		spec:      "environment_id | [a-z][a-z0-9.-]{0,63}; one semantic native environment",
		refusal:   "environment_id must match",
	},
	"providerIdentityKeyPattern": {
		reference: `^[a-z][a-z0-9_.-]{0,63}$`,
		spec:      "A provider-identity key matches [a-z][a-z0-9_.-]{0,63}.",
		refusal:   "opaque_identity key",
	},
	"reverseDNSPattern": {
		reference: `^[a-z][a-z0-9-]{0,62}(\.[a-z][a-z0-9-]{0,62})+$`,
		composition: "the pinned clause declares the label grammar and the at-least-one-dot rule in one sentence but " +
			"spells only the label as a regular expression; the reference composes the two",
		spec:    "A reverse-DNS key is 3-253 lowercase ASCII characters, contains at least one dot, and has dot-separated labels matching [a-z][a-z0-9-]{0,62}.",
		refusal: "is not a 3..253 character lowercase reverse-DNS name",
	},
	"mediaTypePattern": {
		reference: "^[a-z0-9!#$&^_.+%'*`|~-]+/[a-z0-9!#$&^_.+%'*`|~-]+$",
		implementationDefined: "the pinned clause says only that \"media_type is a lowercase ASCII type/subtype " +
			"without parameters\" and types it string[1..255]; it spells out no character class, so the restricted-name " +
			"set below is the implementation's reading rather than a citation",
		refusal: "media_type must be lowercase ASCII type/subtype without parameters",
	},
	"observationNamePattern": {
		reference: `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}$`,
		spec:      "event | observation-name | [a-z][a-z0-9_]*(\\.[a-z][a-z0-9_]*){1,7}, 3-128 characters",
		refusal:   "Observation Event event must match",
	},
	"terminalBackendIDPattern": {
		reference: `^[a-z][a-z0-9]*(?:[.-][a-z0-9]+)*$`,
		spec:      "terminal_backend_id is 1-128 ASCII bytes matching [a-z][a-z0-9]*(?:[.-][a-z0-9]+)*.",
		refusal:   "must match the terminal-backend-id grammar",
	},
	"lowerSnakePattern": {
		reference: `^[a-z][a-z0-9_]*$`,
		// The pinned table types phase as string[1..128] and describes the case
		// in prose. It declares no character class and no anchoring, so the
		// implementation chooses the grammar; the prose is quoted rather than a
		// regex the document does not contain.
		implementationDefined: "the pinned table types the member `phase | string[1..128] or null` and describes the " +
			"case only in prose - \"Stable lower-snake-case phase or null\" - declaring no character class and no " +
			"anchoring, so this reference is the implementation's reading of that prose rather than a citation",
		refusal: "phase must use lower_snake_case",
	},
	"semverPattern": {
		reference: `^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`,
		// The pinned document types these members `semver` and states that
		// "Every independently consumed contract has an independent Semantic
		// Version". It spells out no regular expression for one, so the
		// production below is the implementation's reading of Semantic
		// Versioning 2.0.0 and is recorded as such rather than cited. The
		// reading is deliberately the WHOLE standard: prerelease and build
		// metadata are part of a valid Semantic Version, so a core-triple-only
		// grammar would refuse `1.2.3-rc.1` on a constraint neither the
		// standard nor the pinned document states.
		implementationDefined: "the pinned document types the member `semver` and names Semantic Version, but declares no grammar literal for it; the production implements Semantic Versioning 2.0.0 in full, including optional prerelease and build metadata",
		refusal:               "must be canonical semver",
	},
	"windowsAbsolutePathPattern": {
		reference: `^[A-Za-z]:[\\/]`,
		// A refusal matcher, not an admission matcher: it exists to catch a
		// Windows drive-absolute path in a member the document requires to be a
		// logical identity. Its risky mutation is NARROWING.
		implementationDefined: "the pinned clause states only that \"An absolute path is machine-local routing data " +
			"and never a logical identity\"; it declares no grammar for a drive-absolute prefix, so this reference " +
			"is the implementation's Windows drive/UNC detection rather than a citation",
		refusal: "must be a logical identity, not an absolute path",
		probe:   "C:/logical",
	},
}

// grammarWitness proves one dimension of one grammar at the production entry.
//
// Every value must be refused there, and the self-check below proves each value
// is admitted by the mechanical mutation of exactly its own dimension, so the
// witness cannot drift into a case that would pass against a widened pattern.
//
// specBound is the bound the PINNED SPECIFICATION declares for a counted
// quantifier, never the implementation constant the derived key carries. The
// coverage test compares the two.
type grammarWitness struct {
	pattern   string
	dimension string
	specBound int
	values    []string
}

// subsumedGrammarDimensions records a dimension whose widening cannot change the
// admitted set because an earlier production refusal always fires first. The
// subsuming refusal is named and pinned by the named test, so "subsumed" cannot
// be used to wave a dimension through.
var subsumedGrammarDimensions = map[string]struct {
	subsumingRefusal string
	provingTest      string
}{
	"boardLogicalIDPattern|quantifier#1.max=127": {
		subsumingRefusal: "member logical_id must contain 1..128 Unicode characters",
		provingTest:      "TestBoardLogicalIDLengthIsSubsumedByItsDeclaredStringBound",
	},
}

func declaredGrammarWitnesses() []grammarWitness {
	return []grammarWitness{
		// Session name: [A-Za-z0-9][A-Za-z0-9._-]{0,63}, 1-64 characters.
		{pattern: "sessionNamePattern", dimension: "anchor.start", values: []string{"!Aa"}},
		{pattern: "sessionNamePattern", dimension: "anchor.end", values: []string{"Aa!"}},
		{pattern: "sessionNamePattern", dimension: "class#1", values: []string{"_ab"}},
		{pattern: "sessionNamePattern", dimension: "class#2", values: []string{"Aa b"}},
		{pattern: "sessionNamePattern", dimension: "quantifier#1.max=63", specBound: 63,
			values: []string{"A" + strings.Repeat("a", 64)}},

		// Board logical_id: [A-Za-z0-9][A-Za-z0-9._:-]{0,127}, 1-128 characters.
		{pattern: "boardLogicalIDPattern", dimension: "anchor.start", values: []string{"!ab"}},
		{pattern: "boardLogicalIDPattern", dimension: "anchor.end", values: []string{"ab!"}},
		{pattern: "boardLogicalIDPattern", dimension: "class#1", values: []string{"_ab"}},
		{pattern: "boardLogicalIDPattern", dimension: "class#2", values: []string{"ab c"}},

		// env_names: [A-Za-z_][A-Za-z0-9_]{0,127}.
		{pattern: "environmentNamePattern", dimension: "anchor.start", values: []string{"!AB"}},
		{pattern: "environmentNamePattern", dimension: "anchor.end", values: []string{"AB!"}},
		{pattern: "environmentNamePattern", dimension: "class#1", values: []string{"0AB"}},
		{pattern: "environmentNamePattern", dimension: "class#2", values: []string{"A-B"}},
		{pattern: "environmentNamePattern", dimension: "quantifier#1.max=127", specBound: 127,
			values: []string{"A" + strings.Repeat("B", 128)}},

		// environment_id: [a-z][a-z0-9.-]{0,63}.
		{pattern: "environmentIDPattern", dimension: "anchor.start", values: []string{"!ab"}},
		{pattern: "environmentIDPattern", dimension: "anchor.end", values: []string{"ab!"}},
		{pattern: "environmentIDPattern", dimension: "class#1", values: []string{"0ab"}},
		{pattern: "environmentIDPattern", dimension: "class#2", values: []string{"a_b"}},
		{pattern: "environmentIDPattern", dimension: "quantifier#1.max=63", specBound: 63,
			values: []string{"a" + strings.Repeat("b", 64)}},

		// provider-identity key: [a-z][a-z0-9_.-]{0,63}.
		{pattern: "providerIdentityKeyPattern", dimension: "anchor.start", values: []string{"!ab"}},
		{pattern: "providerIdentityKeyPattern", dimension: "anchor.end", values: []string{"ab!"}},
		{pattern: "providerIdentityKeyPattern", dimension: "class#1", values: []string{"0ab"}},
		{pattern: "providerIdentityKeyPattern", dimension: "class#2", values: []string{"a b"}},
		{pattern: "providerIdentityKeyPattern", dimension: "quantifier#1.max=63", specBound: 63,
			values: []string{"a" + strings.Repeat("b", 64)}},

		// reverse-DNS key: labels [a-z][a-z0-9-]{0,62}, at least one dot.
		{pattern: "reverseDNSPattern", dimension: "anchor.start", values: []string{"!ab.cd"}},
		{pattern: "reverseDNSPattern", dimension: "anchor.end", values: []string{"ab.cd!"}},
		{pattern: "reverseDNSPattern", dimension: "class#1", values: []string{"0ab.cd"}},
		{pattern: "reverseDNSPattern", dimension: "class#2", values: []string{"a_b.cd"}},
		{pattern: "reverseDNSPattern", dimension: "quantifier#1.max=62", specBound: 62,
			values: []string{"a" + strings.Repeat("b", 63) + ".cd"}},
		{pattern: "reverseDNSPattern", dimension: "class#3", values: []string{"ab.0cd"}},
		{pattern: "reverseDNSPattern", dimension: "class#4", values: []string{"ab.c_d"}},
		{pattern: "reverseDNSPattern", dimension: "quantifier#2.max=62", specBound: 62,
			values: []string{"ab.c" + strings.Repeat("d", 63)}},
		{pattern: "reverseDNSPattern", dimension: "one-or-more#1", values: []string{"abc"}},

		// media_type: lowercase ASCII type/subtype without parameters.
		{pattern: "mediaTypePattern", dimension: "anchor.start", values: []string{" text/plain"}},
		{pattern: "mediaTypePattern", dimension: "anchor.end", values: []string{"text/plain "}},
		{pattern: "mediaTypePattern", dimension: "class#1", values: []string{"te xt/plain"}},
		{pattern: "mediaTypePattern", dimension: "one-or-more#1", values: []string{"/plain"}},
		{pattern: "mediaTypePattern", dimension: "class#2", values: []string{"text/pla in"}},
		{pattern: "mediaTypePattern", dimension: "one-or-more#2", values: []string{"text/"}},

		// observation-name: [a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*){1,7}, 3-128 characters.
		{pattern: "observationNamePattern", dimension: "anchor.start", values: []string{"!ab.cd"}},
		{pattern: "observationNamePattern", dimension: "anchor.end", values: []string{"ab.cd!"}},
		{pattern: "observationNamePattern", dimension: "class#1", values: []string{"0ab.cd"}},
		{pattern: "observationNamePattern", dimension: "class#2", values: []string{"a-b.cd"}},
		{pattern: "observationNamePattern", dimension: "class#3", values: []string{"ab.0cd"}},
		{pattern: "observationNamePattern", dimension: "class#4", values: []string{"ab.c-d"}},
		{pattern: "observationNamePattern", dimension: "quantifier#1.max=7", specBound: 7,
			values: []string{"a.b.c.d.e.f.g.h.i"}},
		{pattern: "observationNamePattern", dimension: "quantifier#1.min=1", specBound: 1,
			values: []string{"abc"}},

		// terminal-backend-id: [a-z][a-z0-9]*(?:[.-][a-z0-9]+)*.
		{pattern: "terminalBackendIDPattern", dimension: "anchor.start", values: []string{"!ax"}},
		{pattern: "terminalBackendIDPattern", dimension: "anchor.end", values: []string{"ax!"}},
		{pattern: "terminalBackendIDPattern", dimension: "class#1", values: []string{"0ax"}},
		{pattern: "terminalBackendIDPattern", dimension: "class#2", values: []string{"a_x"}},
		{pattern: "terminalBackendIDPattern", dimension: "class#3", values: []string{"ax_tmux"}},
		{pattern: "terminalBackendIDPattern", dimension: "class#4", values: []string{"ax.tm_ux"}},
		{pattern: "terminalBackendIDPattern", dimension: "one-or-more#1", values: []string{"ax."}},

		// lower-snake-case phase: [a-z][a-z0-9_]*.
		{pattern: "lowerSnakePattern", dimension: "anchor.start", values: []string{"!ab"}},
		{pattern: "lowerSnakePattern", dimension: "anchor.end", values: []string{"ab!"}},
		{pattern: "lowerSnakePattern", dimension: "class#1", values: []string{"0ab"}},
		{pattern: "lowerSnakePattern", dimension: "class#2", values: []string{"a-b"}},

		// Semantic Versioning 2.0.0 in full: three core numbers, an optional
		// dot-separated prerelease, and optional dot-separated build metadata.
		// Classes 1-6 are the core numbers, 7-11 the first prerelease
		// identifier, 12-16 each later prerelease identifier, and 17-18 the
		// build-metadata identifiers.
		{pattern: "semverPattern", dimension: "anchor.start", values: []string{"!1.0.0"}},
		{pattern: "semverPattern", dimension: "anchor.end", values: []string{"1.0.0!"}},
		{pattern: "semverPattern", dimension: "class#1", values: []string{"01.0.0"}},
		{pattern: "semverPattern", dimension: "class#2", values: []string{"1a.0.0"}},
		{pattern: "semverPattern", dimension: "class#3", values: []string{"1.01.0"}},
		{pattern: "semverPattern", dimension: "class#4", values: []string{"1.1a.0"}},
		{pattern: "semverPattern", dimension: "class#5", values: []string{"1.0.01"}},
		{pattern: "semverPattern", dimension: "class#6", values: []string{"1.0.1a"}},
		{pattern: "semverPattern", dimension: "class#7", values: []string{"1.0.0-01"}},
		{pattern: "semverPattern", dimension: "class#8", values: []string{"1.0.0-1!"}},
		{pattern: "semverPattern", dimension: "class#9", values: []string{"1.0.0-!a"}},
		{pattern: "semverPattern", dimension: "class#10", values: []string{"1.0.0-1_"}},
		{pattern: "semverPattern", dimension: "class#11", values: []string{"1.0.0-a_b"}},
		{pattern: "semverPattern", dimension: "class#12", values: []string{"1.0.0-a.01"}},
		{pattern: "semverPattern", dimension: "class#13", values: []string{"1.0.0-a.1!"}},
		{pattern: "semverPattern", dimension: "class#14", values: []string{"1.0.0-a.!b"}},
		{pattern: "semverPattern", dimension: "class#15", values: []string{"1.0.0-a.1_"}},
		{pattern: "semverPattern", dimension: "class#16", values: []string{"1.0.0-a.b_c"}},
		{pattern: "semverPattern", dimension: "class#17", values: []string{"1.0.0+a_b"}},
		{pattern: "semverPattern", dimension: "class#18", values: []string{"1.0.0+a.b_c"}},
		{pattern: "semverPattern", dimension: "one-or-more#1", values: []string{"1.0.0+"}},
		{pattern: "semverPattern", dimension: "one-or-more#2", values: []string{"1.0.0+a."}},

		// Windows drive-absolute prefix: a REFUSAL matcher, so its witnesses
		// defeat a narrowing of each character class.
		{pattern: "windowsAbsolutePathPattern", dimension: "class#1", values: []string{"c:/logical", "C:/logical"}},
		{pattern: "windowsAbsolutePathPattern", dimension: "class#2", values: []string{"C:/logical", `C:\logical`}},
	}
}

// grammarPosition is a place in a valid fixture where a declared grammar is
// reached at the production entry. Positions are LOCATED by probing rather than
// hand-wired: a probe value that violates every admitting grammar is written at
// every candidate value and every open-map key of every fixture, and the
// position is attributed to whichever grammar's own refusal message comes back.
type grammarPosition struct {
	fixture identityFixture
	path    []string
	// key names the open-map member renamed at this position; empty means the
	// value at path is replaced instead.
	key string
}

func (position grammarPosition) describe() string {
	if position.key != "" {
		return fmt.Sprintf("%s at %s key %q", position.fixture.name, formatJSONPath(position.path), position.key)
	}
	return fmt.Sprintf("%s at %s", position.fixture.name, formatJSONPath(position.path))
}

// everyCandidateObjectPath returns the path of every JSON object inside a
// candidate, including the open maps everyCandidateValuePath deliberately does
// not descend into, because those are exactly the maps whose KEYS carry a
// declared grammar.
func everyCandidateObjectPath(object map[string]any) [][]string {
	var paths [][]string
	var visit func(any, []string)
	visit = func(value any, prefix []string) {
		switch typed := value.(type) {
		case map[string]any:
			paths = append(paths, append([]string{}, prefix...))
			keys := make([]string, 0, len(typed))
			for key := range typed {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				visit(typed[key], append(append([]string{}, prefix...), key))
			}
		case []any:
			for index, member := range typed {
				visit(member, append(append([]string{}, prefix...), fmt.Sprintf("[%d]", index)))
			}
		}
	}
	visit(object, nil)
	return paths
}

func jsonObjectAtPath(t *testing.T, root map[string]any, path []string) map[string]any {
	t.Helper()
	if len(path) == 0 {
		return root
	}
	value := jsonValueAtPath(t, root, path)
	object, _ := value.(map[string]any)
	return object
}

// renameObjectKey rewrites one member name of the object at path, which is how a
// grammar declared over MAP KEYS is attacked at the production entry.
func renameObjectKey(t *testing.T, root map[string]any, path []string, from, to string) {
	t.Helper()
	object := jsonObjectAtPath(t, root, path)
	if object == nil {
		t.Fatalf("no object at %s", formatJSONPath(path))
	}
	value := object[from]
	delete(object, from)
	object[to] = value
}

// locateGrammarPositions maps each declared grammar to the fixture positions
// that reach it, by attribution of the production refusal message.
func locateGrammarPositions(t *testing.T, patterns map[string]grammarPattern) map[string][]grammarPosition {
	t.Helper()

	byProbe := make(map[string][]string)
	for name := range patterns {
		probe := declaredGrammars[name].probe
		if probe == "" {
			probe = universalGrammarProbe
		}
		byProbe[probe] = append(byProbe[probe], name)
	}

	located := make(map[string][]grammarPosition)
	attribute := func(fixture identityFixture, path []string, key string, candidate map[string]any, names []string) {
		reason := fixtureRefusalReason(t, fixture, candidate)
		if reason == "" {
			return
		}
		for _, name := range names {
			if strings.Contains(reason, declaredGrammars[name].refusal) {
				located[name] = append(located[name], grammarPosition{fixture: fixture, path: path, key: key})
			}
		}
	}

	for _, fixture := range everyValidIdentityFixture() {
		for probe, names := range byProbe {
			for _, path := range everyCandidateValuePath(fixture.object) {
				if _, ok := jsonValueAtPath(t, fixture.object, path).(string); !ok {
					continue
				}
				candidate := cloneJSONObject(t, fixture.object)
				setJSONValueAtPath(t, candidate, path, probe)
				attribute(fixture, path, "", candidate, names)
			}
			for _, path := range everyCandidateObjectPath(fixture.object) {
				object := jsonObjectAtPath(t, fixture.object, path)
				if len(object) == 0 {
					continue
				}
				keys := make([]string, 0, len(object))
				for key := range object {
					keys = append(keys, key)
				}
				sort.Strings(keys)
				candidate := cloneJSONObject(t, fixture.object)
				renameObjectKey(t, candidate, path, keys[0], probe)
				attribute(fixture, path, keys[0], candidate, names)
			}
		}
	}
	return located
}

// TestEveryCompiledGrammarIsDeclaredAndReachable is the inventory's floor. Every
// regexp compiled in the package production sources must record where the pinned
// specification declares it (or say plainly that the document does not), must
// name the refusal it emits, and must be reachable from a valid fixture at the
// production entry. A grammar nothing reaches declares a language nothing
// enforces.
func TestEveryCompiledGrammarIsDeclaredAndReachable(t *testing.T) {
	patterns := deriveGrammarPatterns(t)
	for name := range patterns {
		declared, ok := declaredGrammars[name]
		if !ok {
			t.Errorf(
				"%s is compiled in production but has no entry in declaredGrammars, so no dimension of it is proven. "+
					"Record its pinned SPEC declaration, or say explicitly that the document declares none.", name)
			continue
		}
		if (declared.spec == "") == (declared.implementationDefined == "") {
			t.Errorf("%s must record exactly one of a pinned SPEC declaration or an implementation-defined reason", name)
		}
		if strings.TrimSpace(declared.refusal) == "" {
			t.Errorf("%s does not name the production refusal it emits, so its witnesses cannot be attributed to it", name)
		}
	}
	for name := range declaredGrammars {
		if _, ok := patterns[name]; !ok {
			t.Errorf("declaredGrammars names %s, which the package no longer compiles", name)
		}
	}

	located := locateGrammarPositions(t, patterns)
	var unreachable []string
	for name := range patterns {
		if len(located[name]) == 0 {
			unreachable = append(unreachable, name)
		}
	}
	sort.Strings(unreachable)
	if len(unreachable) > 0 {
		t.Errorf(
			"declared grammar(s) that no valid fixture reaches at the production entry, so every witness below them "+
				"would be vacuous:\n  %s", strings.Join(unreachable, "\n  "))
	}
}

// TestEveryDeclaredGrammarDimensionHasAProvenWitness is the coverage assertion.
//
// The obligation is derived from the pattern source, so adding a character class
// or a counted bound to any production grammar fails here until its witness
// exists, and CHANGING a counted bound changes the derived key so the old
// witness no longer discharges it. specBound is compared against the derived
// implementation number, which is how a bound widened in production is caught by
// the pinned specification literal rather than by itself.
func TestEveryDeclaredGrammarDimensionHasAProvenWitness(t *testing.T) {
	t.Parallel()

	obligations := deriveGrammarObligations(t)
	witnessed := make(map[string]int)
	names := packageTestFunctionNames(t)

	for _, witness := range declaredGrammarWitnesses() {
		key := witness.pattern + "|" + witness.dimension
		witnessed[key]++
		site, ok := obligations[key]
		if !ok {
			t.Errorf("witness %s claims a dimension no production grammar declares", key)
			continue
		}
		if len(witness.values) == 0 {
			t.Errorf("witness %s carries no value", key)
		}
		switch site.kind {
		case grammarQuantifierMax:
			if witness.specBound != site.maximum {
				t.Errorf(
					"witness %s asserts the pinned maximum %d but production declares %d; a bound must be asserted "+
						"against the specification literal, never against the constant it derives from",
					key, witness.specBound, site.maximum)
			}
		case grammarQuantifierMin:
			if witness.specBound != site.minimum {
				t.Errorf("witness %s asserts the pinned minimum %d but production declares %d", key, witness.specBound, site.minimum)
			}
		}
	}

	var missing []string
	for key := range obligations {
		if witnessed[key] > 0 {
			if _, subsumed := subsumedGrammarDimensions[key]; subsumed {
				t.Errorf("dimension %s is disclosed as subsumed but also carries a witness; remove one", key)
			}
			continue
		}
		if subsumption, subsumed := subsumedGrammarDimensions[key]; subsumed {
			if _, exists := names[subsumption.provingTest]; !exists {
				t.Errorf("subsumed dimension %s names %s, which does not exist in this package", key, subsumption.provingTest)
			}
			continue
		}
		missing = append(missing, key)
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf(
			"declared grammar dimension(s) with no widening witness. A grammar with an unproven dimension is "+
				"pinned only against deletion; widening it leaves the whole suite green:\n  %s",
			strings.Join(missing, "\n  "))
	}
}

// deriveGrammarObligations returns every dimension that carries a risky
// mutation, keyed as pattern|dimension.
//
// For an admission matcher every dimension is risky, because each one can be
// widened to admit more. For a refusal matcher only the character classes are:
// dropping its anchor or relaxing a quantifier makes it match MORE inputs, which
// refuses more and cannot admit anything new.
func deriveGrammarObligations(t *testing.T) map[string]grammarSite {
	t.Helper()

	obligations := make(map[string]grammarSite)
	for name, pattern := range deriveGrammarPatterns(t) {
		// Dimensions are derived from the PINNED REFERENCE, not from the
		// production source, so a widened production pattern cannot delete its
		// own obligation. TestEveryCompiledGrammarMatchesItsPinnedReference is
		// what keeps the two identical.
		for _, site := range deriveGrammarSites(name, declaredGrammars[name].reference) {
			if !pattern.admitting && site.kind != grammarClass {
				continue
			}
			obligations[name+"|"+site.dimension] = site
		}
	}
	if len(obligations) == 0 {
		t.Fatal("derived zero grammar dimensions; the scanner is broken, not the package")
	}
	return obligations
}

// TestEveryGrammarWitnessIsAdmittedByItsOwnWideningOnly is the derivation's
// self-check, and it is what makes the witness table more than decoration.
//
// For an admission matcher it proves, mechanically, that the production pattern
// REFUSES the witness while the pattern with exactly that one dimension widened
// ADMITS it. A witness that would also pass against the widened grammar - the
// "grammar inventory that cannot fail when a pattern is widened" this Story was
// rejected for - fails here.
//
// For a refusal matcher it proves the mirror image: production MATCHES the
// witness, and narrowing that character class to its first or last declared item
// stops matching it, so the value would be admitted.
func TestEveryGrammarWitnessIsAdmittedByItsOwnWideningOnly(t *testing.T) {
	t.Parallel()

	patterns := deriveGrammarPatterns(t)
	obligations := deriveGrammarObligations(t)
	for _, witness := range declaredGrammarWitnesses() {
		key := witness.pattern + "|" + witness.dimension
		site, ok := obligations[key]
		if !ok {
			continue
		}
		pattern := patterns[witness.pattern]
		production := referenceGrammar(t, witness.pattern)
		if pattern.admitting {
			widened := regexp.MustCompile(widenGrammarSource(declaredGrammars[witness.pattern].reference, site))
			for _, value := range witness.values {
				if production.MatchString(value) {
					t.Errorf("witness %s value %q is admitted by the production grammar; it proves nothing", key, value)
				}
				if !widened.MatchString(value) {
					t.Errorf(
						"witness %s value %q is refused even by the widened grammar %q, so widening %s cannot make "+
							"this case pass and the dimension stays unproven",
						key, value, widened.String(), witness.dimension)
				}
			}
			continue
		}
		defeated := 0
		for _, narrowed := range narrowedGrammarVariants(declaredGrammars[witness.pattern].reference, site) {
			compiled := regexp.MustCompile(narrowed)
			for _, value := range witness.values {
				if !production.MatchString(value) {
					t.Errorf("witness %s value %q is not matched by the refusal grammar; it proves nothing", key, value)
				}
				if !compiled.MatchString(value) {
					defeated++
				}
			}
		}
		if defeated < 2 {
			t.Errorf(
				"witness %s does not defeat both narrowings of its character class, so narrowing the class would "+
					"silently admit an absolute path", key)
		}
	}
}

// narrowedGrammarVariants returns the first-item and last-item narrowings of a
// character class, which are the two ways a declared class can lose members at
// its edges.
func narrowedGrammarVariants(source string, site grammarSite) []string {
	if site.kind != grammarClass {
		return nil
	}
	body := source[site.start+1 : site.end-1]
	items := characterClassItems(body)
	if len(items) < 2 {
		return nil
	}
	return []string{
		source[:site.start] + "[" + items[0] + "]" + source[site.end:],
		source[:site.start] + "[" + items[len(items)-1] + "]" + source[site.end:],
	}
}

func characterClassItems(body string) []string {
	var items []string
	for index := 0; index < len(body); {
		start := index
		if body[index] == '\\' && index+1 < len(body) {
			index += 2
		} else {
			index++
		}
		if index+1 < len(body) && body[index] == '-' {
			index += 2
		}
		items = append(items, body[start:index])
	}
	return items
}

// TestEveryDeclaredGrammarDimensionIsRefusedAtTheProductionEntry drives every
// witness through CalculateObjectIdentity and VerifyObjectIdentity, or
// ValidateObservationEvent for Section 18.1 objects, at a position located by
// probing rather than hand-wired, and requires the refusal to come from the
// grammar under test rather than from any clause that happens to fire first.
func TestEveryDeclaredGrammarDimensionIsRefusedAtTheProductionEntry(t *testing.T) {
	patterns := deriveGrammarPatterns(t)
	located := locateGrammarPositions(t, patterns)

	var admitted, misattributed []string
	for _, witness := range declaredGrammarWitnesses() {
		positions := located[witness.pattern]
		if len(positions) == 0 {
			continue // reported by TestEveryCompiledGrammarIsDeclaredAndReachable
		}
		position := positions[0]
		for _, value := range witness.values {
			candidate := cloneJSONObject(t, position.fixture.object)
			if position.key != "" {
				renameObjectKey(t, candidate, position.path, position.key, value)
			} else {
				setJSONValueAtPath(t, candidate, position.path, value)
			}
			reason := fixtureRefusalReason(t, position.fixture, candidate)
			label := fmt.Sprintf("%s|%s %q at %s", witness.pattern, witness.dimension, value, position.describe())
			if reason == "" {
				admitted = append(admitted, label)
				continue
			}
			if !strings.Contains(reason, declaredGrammars[witness.pattern].refusal) {
				misattributed = append(misattributed, label+" refused by another clause: "+reason)
			}
		}
	}

	sort.Strings(admitted)
	sort.Strings(misattributed)
	if len(admitted) > 0 {
		t.Errorf(
			"the production entry ADMITTED and attested a value outside a declared grammar:\n  %s",
			strings.Join(admitted, "\n  "))
	}
	if len(misattributed) > 0 {
		t.Errorf(
			"witness(es) refused by a clause other than the grammar under test, so they pin nothing about that "+
				"grammar:\n  %s", strings.Join(misattributed, "\n  "))
	}
}

// TestBoardLogicalIDLengthIsSubsumedByItsDeclaredStringBound pins the one
// subsumption the grammar inventory claims. Board Identity logical_id is read
// through requireBoundedString(1, 128) before boardLogicalIDPattern sees it, so
// the pattern's own {0,127} repetition can never decide a 129-character value:
// the string bound refuses it first. Widening the repetition to {0,128} is
// therefore unobservable, and this test proves the ordering rather than assuming
// it.
func TestBoardLogicalIDLengthIsSubsumedByItsDeclaredStringBound(t *testing.T) {
	const declaredMaximum = 128

	atMaximum := "A" + strings.Repeat("b", declaredMaximum-1)
	pastMaximum := atMaximum + "c"
	if len(atMaximum) != declaredMaximum {
		t.Fatalf("bound fixture is %d characters, want %d", len(atMaximum), declaredMaximum)
	}

	accepted := taskBoardSessionRecord("TASK-260830-8x76g1")
	setJSONValueAtPath(t, accepted, []string{"task_board", "board", "logical_id"}, atMaximum)
	assertIdentityEntriesAcceptShape(t, mustJSON(t, accepted), SelfRecordID)

	refused := taskBoardSessionRecord("TASK-260830-8x76g1")
	setJSONValueAtPath(t, refused, []string{"task_board", "board", "logical_id"}, pastMaximum)
	assertIdentityEntriesRefuseWithReason(t, mustJSON(t, refused), SelfRecordID,
		subsumedGrammarDimensions["boardLogicalIDPattern|quantifier#1.max=127"].subsumingRefusal)
}

// grammarNeighbourFamilies are the character families a declared grammar can be
// widened by. They are chosen so that every widening a reviewer would actually
// write - "admit uppercase too", "allow a hyphen", "allow a dot" - moves at
// least one generated neighbour from refused to admitted.
var grammarNeighbourFamilies = []struct {
	name      string
	character rune
}{
	{"uppercase", 'Q'},
	{"lowercase", 'q'},
	{"digit", '7'},
	{"underscore", '_'},
	{"hyphen", '-'},
	{"dot", '.'},
	{"colon", ':'},
	{"slash", '/'},
	{"space", ' '},
	{"newline", '\n'},
	{"NUL", '\x00'},
	{"non-ASCII", 'é'},
}

// TestEveryDeclaredGrammarRefusesEveryOneCharacterNeighbourAtTheProductionEntry
// is the part of the inventory with real killing power, and it hand-writes
// nothing at all.
//
// The dimension witnesses above prove each STRUCTURAL feature with an attributed
// case, but a single witness per character class is admitted by the maximal
// widening of that class and says nothing about a narrower one: a mutant that
// widens [a-z] to [A-Za-z] survives a witness that used a digit. That gap is
// real - it survived a control mutant on providerIdentityKeyPattern.
//
// So this test starts from the value the fixture already carries at the located
// position, which is by construction a valid member of the declared language,
// and drives every one-character neighbour of it through the production entry:
// each position substituted by each character family, each family inserted at
// each position, and each position deleted. Every neighbour the grammar itself
// refuses must be refused at the entry. Widening a class by ANY family, dropping
// an anchor, or relaxing a quantifier moves at least one neighbour into the
// admitted set and fails here.
func TestEveryDeclaredGrammarRefusesEveryOneCharacterNeighbourAtTheProductionEntry(t *testing.T) {
	patterns := deriveGrammarPatterns(t)
	located := locateGrammarPositions(t, patterns)

	exercised := make(map[string]int)
	var admitted []string
	for name, pattern := range patterns {
		if !pattern.admitting {
			// A refusal matcher has no valid template: its dangerous direction is
			// narrowing, proven by the witness table above.
			continue
		}
		positions := located[name]
		if len(positions) == 0 {
			continue // reported by TestEveryCompiledGrammarIsDeclaredAndReachable
		}
		position := positions[0]
		template := position.key
		if template == "" {
			value, ok := jsonValueAtPath(t, position.fixture.object, position.path).(string)
			if !ok {
				t.Fatalf("located position for %s does not carry a string", name)
			}
			template = value
		}
		production := referenceGrammar(t, name)
		if !production.MatchString(template) {
			t.Fatalf(
				"located template %q for %s is not admitted by the grammar itself, so every neighbour of it is "+
					"vacuous", template, name)
		}
		for _, neighbour := range oneCharacterNeighbours(template) {
			if production.MatchString(neighbour.value) {
				continue
			}
			candidate := cloneJSONObject(t, position.fixture.object)
			if position.key != "" {
				renameObjectKey(t, candidate, position.path, position.key, neighbour.value)
			} else {
				setJSONValueAtPath(t, candidate, position.path, neighbour.value)
			}
			exercised[name]++
			if !fixtureEntriesRefuse(t, position.fixture, candidate) {
				admitted = append(admitted, fmt.Sprintf(
					"%s: %s of %q produced %q, which the declared grammar refuses and the production entry attested (%s)",
					name, neighbour.description, template, neighbour.value, position.describe()))
			}
		}
	}

	for name, pattern := range patterns {
		if !pattern.admitting || len(located[name]) == 0 {
			continue
		}
		if exercised[name] == 0 {
			t.Errorf("no neighbour of the located template exercised %s; the sweep covers it vacuously", name)
		}
	}
	sort.Strings(admitted)
	if len(admitted) > 0 {
		t.Errorf(
			"the production entry ADMITTED and attested value(s) outside a declared grammar:\n  %s",
			strings.Join(admitted, "\n  "))
	}
}

type grammarNeighbour struct {
	description string
	value       string
}

// oneCharacterNeighbours returns every value one edit away from a valid one:
// each position substituted by each family representative, each representative
// inserted at each boundary, and each position deleted.
func oneCharacterNeighbours(template string) []grammarNeighbour {
	runes := []rune(template)
	var neighbours []grammarNeighbour
	for _, family := range grammarNeighbourFamilies {
		for index := range runes {
			mutated := append([]rune{}, runes...)
			mutated[index] = family.character
			neighbours = append(neighbours, grammarNeighbour{
				description: fmt.Sprintf("%s substituted at %d", family.name, index),
				value:       string(mutated),
			})
		}
		for index := 0; index <= len(runes); index++ {
			mutated := append([]rune{}, runes[:index]...)
			mutated = append(mutated, family.character)
			mutated = append(mutated, runes[index:]...)
			neighbours = append(neighbours, grammarNeighbour{
				description: fmt.Sprintf("%s inserted at %d", family.name, index),
				value:       string(mutated),
			})
		}
	}
	for index := range runes {
		mutated := append([]rune{}, runes[:index]...)
		mutated = append(mutated, runes[index+1:]...)
		neighbours = append(neighbours, grammarNeighbour{
			description: fmt.Sprintf("character deleted at %d", index),
			value:       string(mutated),
		})
	}
	return neighbours
}

// referenceGrammar compiles the pinned reference for a declared grammar. Every
// oracle in this file reads it rather than the production source, so a widened
// production pattern cannot make its own neighbours look admissible.
func referenceGrammar(t *testing.T, name string) *regexp.Regexp {
	t.Helper()

	declared, ok := declaredGrammars[name]
	if !ok || declared.reference == "" {
		t.Fatalf("%s has no pinned reference grammar", name)
	}
	return regexp.MustCompile(declared.reference)
}

// TestEveryCompiledGrammarMatchesItsPinnedReference is the pin that makes every
// other test in this file mutation-proof.
//
// A regular expression IS its language, so pinning the source pins the admitted
// set exactly - unlike a vocabulary argument list, which pins the members and
// leaves the comparison free. Widening any production grammar in any dimension
// fails here immediately; the neighbour sweeps then prove separately that the
// pinned grammar is actually REACHED and enforced at the production entry,
// because a pattern pinned but never called promises nothing.
func TestEveryCompiledGrammarMatchesItsPinnedReference(t *testing.T) {
	t.Parallel()

	for name, pattern := range deriveGrammarPatterns(t) {
		declared, ok := declaredGrammars[name]
		if !ok {
			continue // reported by TestEveryCompiledGrammarIsDeclaredAndReachable
		}
		if declared.reference != pattern.source {
			t.Errorf(
				"production grammar %s is %q but its pinned reference is %q. Widening or narrowing a declared "+
					"grammar is a change to the admitted set and must be reviewed against the pinned specification, "+
					"not made in passing.",
				name, pattern.source, declared.reference)
		}
	}
}

// TestEveryPinnedGrammarReferenceQuotesItsSpecDeclaration keeps the reference
// column honest. A reference whose core appears verbatim inside the quoted
// pinned declaration is checkable against the document; one that does not must
// say either that the document declares no grammar for it, or which clauses the
// reference composes.
func TestEveryPinnedGrammarReferenceQuotesItsSpecDeclaration(t *testing.T) {
	t.Parallel()

	for name, declared := range declaredGrammars {
		if declared.spec == "" {
			if declared.implementationDefined == "" {
				t.Errorf("%s records neither a pinned SPEC declaration nor an implementation-defined reason", name)
			}
			continue
		}
		core := strings.TrimSuffix(strings.TrimPrefix(declared.reference, "^"), "$")
		if strings.Contains(declared.spec, core) {
			if declared.composition != "" {
				t.Errorf("%s discloses a composition but its reference appears verbatim in the quoted declaration", name)
			}
			continue
		}
		if declared.composition == "" {
			t.Errorf(
				"pinned reference %q for %s does not appear in its quoted declaration %q. Quote the clause that "+
					"declares it, or disclose which clauses the reference composes; do not cite a grammar the "+
					"document does not contain.",
				declared.reference, name, declared.spec)
		}
	}
}

// TestOneCharacterNeighbourGeneratorCoversEveryEditForm is the neighbour sweep's
// own completeness proof. The sweep's killing power is the SHAPE of the corpus
// it generates, so silently generating fewer neighbours - dropping insertions,
// stopping before the final boundary, or skipping deletions - would weaken every
// grammar in the package while leaving the sweep green. The counts are asserted
// exactly, and the three edit forms are asserted by value.
func TestOneCharacterNeighbourGeneratorCoversEveryEditForm(t *testing.T) {
	t.Parallel()

	const template = "ab"
	families := len(grammarNeighbourFamilies)
	runes := len([]rune(template))

	neighbours := oneCharacterNeighbours(template)
	want := runes*families + (runes+1)*families + runes
	if len(neighbours) != want {
		t.Errorf("generated %d neighbours of %q, want %d substitutions, insertions and deletions",
			len(neighbours), template, want)
	}

	values := make(map[string]bool, len(neighbours))
	forms := make(map[string]int)
	for _, neighbour := range neighbours {
		values[neighbour.value] = true
		switch {
		case strings.Contains(neighbour.description, "substituted"):
			forms["substituted"]++
		case strings.Contains(neighbour.description, "inserted"):
			forms["inserted"]++
		case strings.Contains(neighbour.description, "deleted"):
			forms["deleted"]++
		}
	}
	if forms["substituted"] != runes*families {
		t.Errorf("generated %d substitutions, want %d", forms["substituted"], runes*families)
	}
	if forms["inserted"] != (runes+1)*families {
		t.Errorf("generated %d insertions, want %d; an insertion at every boundary is what proves the anchors",
			forms["inserted"], (runes+1)*families)
	}
	if forms["deleted"] != runes {
		t.Errorf("generated %d deletions, want %d; deletion is what proves a one-or-more quantifier", forms["deleted"], runes)
	}
	for _, required := range []string{"Qb", "aQ", "Qab", "aQb", "abQ", "a", "b"} {
		if !values[required] {
			t.Errorf("the neighbour corpus of %q does not contain %q, so an edit form is missing", template, required)
		}
	}
}
