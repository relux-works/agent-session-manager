package secprim

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// secretTokens is the instrument vocabulary for the secret-site census.
// Every token must fire on at least one derived site (see
// TestSecretTokensAreProven), or the token is dead ceremony. "token" is
// excluded from the local-binding arm below because the JSON decoder
// vocabulary (decoder.Token, token variables) would drown it; every other
// token scans locals too.
//
// Deliberately absent: bare "auth" (authority identifiers, authorization
// enums, and authentication subjects are UUID/boolean/enum protocol
// vocabulary, never credential values — TestSecretScanBlindSpot pins
// this), and "bundle" (bundle identifiers are digests; opaque bundle
// content is covered by the Section 16.2 class row, not by field
// census). "transcript" and "scrollback" are present: raw transcripts
// are an excludable class with in-repo gate sites.
var secretTokens = []string{
	"secret", "credential", "passwd", "password", "token",
	"passphrase", "cookie", "private_key", "privatekey",
	"api_key", "apikey", "transcript", "scrollback",
}

// censusPackages bounds the secret-site scan. Every package that handles
// provider output, configuration, errors, records, or storage is in;
// specdoc, traceability, and command trees carry no secret-bearing
// production values.
var censusPackages = []string{
	"axerror", "canonicaljson", "cliresult", "config", "dirnode",
	"environ", "localstore", "provider", "provhost", "scalar",
	"secprim", "sessadapter", "terminalbackend",
}

// deriveSecretSites scans non-test production source in the census
// packages for token-matching sites: declaration names (types,
// functions, package-level values), struct field names, parameter and
// result names, local bindings (all tokens except "token"), and every
// string literal. Comments never participate: the walk is over syntax,
// so a documented example cannot roster a site and a secret in a comment
// cannot hide from it either — comments are outside both directions by
// construction, which TestSecretScanIgnoresComments pins.
func deriveSecretSites(t *testing.T) map[string]bool {
	t.Helper()
	sites := map[string]bool{}
	scanned := 0
	for _, pkg := range censusPackages {
		directory := filepath.Join("..", pkg)
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatalf("read %s: %v", directory, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			contents, err := os.ReadFile(filepath.Join(directory, name))
			if err != nil {
				t.Fatal(err)
			}
			scanned++
			fileset := token.NewFileSet()
			syntax, err := parser.ParseFile(fileset, name, contents, 0)
			if err != nil {
				t.Fatal(err)
			}
			packageName := pkg
			_ = packageName
			ast.Inspect(syntax, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.TypeSpec:
					admitSite(sites, pkg, "decl", typed.Name.Name)
				case *ast.FuncDecl:
					admitSite(sites, pkg, "decl", typed.Name.Name)
				case *ast.GenDecl:
					if typed.Tok == token.IMPORT {
						return true
					}
					for _, spec := range typed.Specs {
						value, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						kind := "decl"
						if inFunctionBody(syntax, spec) {
							kind = "local"
						}
						for _, identifier := range value.Names {
							admitName(sites, pkg, kind, identifier.Name)
						}
					}
					return true
				case *ast.Field:
					for _, identifier := range typed.Names {
						admitName(sites, pkg, fieldKind(syntax, typed), identifier.Name)
					}
					return true
				case *ast.AssignStmt:
					for _, bound := range typed.Lhs {
						identifier, ok := bound.(*ast.Ident)
						if !ok || identifier.Name == "_" {
							continue
						}
						admitLocal(sites, pkg, identifier.Name)
					}
					return true
				case *ast.BasicLit:
					if typed.Kind != token.STRING {
						return true
					}
					admitLiteral(sites, pkg, typed.Value)
				}
				return true
			})
		}
	}
	if scanned == 0 {
		t.Fatal("scanned no production sources; the census is blind")
	}
	if len(sites) == 0 {
		t.Fatal("derived no secret sites; the scanner is broken, not the tree")
	}
	return sites
}

// inFunctionBody reports whether the spec sits inside a function
// declaration. Package-level values are declarations; function-level
// values are locals and scan under the strong tokens only.
func inFunctionBody(syntax *ast.File, spec ast.Spec) bool {
	for _, declaration := range syntax.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		found := false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if node == spec {
				found = true
				return false
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

// fieldKind distinguishes struct fields (rostered under every token)
// from parameter and result names (same treatment): both are
// declaration-adjacent bindings, unlike locals.
func fieldKind(syntax *ast.File, field *ast.Field) string {
	for _, declaration := range syntax.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Type == nil {
			continue
		}
		for _, list := range []*ast.FieldList{function.Type.Params, function.Type.Results} {
			if list == nil {
				continue
			}
			for _, candidate := range list.List {
				if candidate == field {
					return "param"
				}
			}
		}
	}
	return "field"
}

// admitSite records a declaration-position name under every token it
// contains.
func admitSite(sites map[string]bool, pkg, kind, name string) {
	admitName(sites, pkg, kind, name)
}

// admitName records a declaration-adjacent name under every token it
// contains (case-insensitive substring).
func admitName(sites map[string]bool, pkg, kind, name string) {
	lowered := strings.ToLower(name)
	for _, word := range secretTokens {
		if strings.Contains(lowered, word) {
			sites[pkg+"|"+kind+"|"+name] = true
		}
	}
}

// admitLocal records a function-local binding under every strong token:
// every token except "token", which the JSON decoder vocabulary owns.
func admitLocal(sites map[string]bool, pkg, name string) {
	lowered := strings.ToLower(name)
	for _, word := range secretTokens {
		if word == "token" {
			continue
		}
		if strings.Contains(lowered, word) {
			sites[pkg+"|local|"+name] = true
		}
	}
}

// maxLiteralRunes bounds the literal arm: a literal longer than this is
// prose documentation, not an identifier, key, or diagnostic atom, and
// rostering it would turn every explanatory paragraph into a site. The
// longest rostered diagnostic is 55 runes; the bound sits at 80.
const maxLiteralRunes = 80

// admitLiteral records a string literal under every token it contains.
// The literal is unquoted when it parses; an unparseable literal is
// recorded raw rather than dropped. Literals past maxLiteralRunes are
// documentation prose, not sites.
func admitLiteral(sites map[string]bool, pkg, raw string) {
	value, err := strconv.Unquote(raw)
	if err != nil {
		value = raw
	}
	if len([]rune(value)) > maxLiteralRunes {
		return
	}
	lowered := strings.ToLower(value)
	for _, word := range secretTokens {
		if strings.Contains(lowered, word) {
			sites[pkg+"|lit|"+value] = true
		}
	}
}

// TestSecretScanIgnoresComments pins the comment blindness both ways: a
// token in a comment rosters nothing, so documentation cannot create
// obligations, and (by construction of the syntax walk) nothing outside
// code participates.
func TestSecretScanIgnoresComments(t *testing.T) {
	t.Parallel()
	syntax, err := parser.ParseFile(token.NewFileSet(), "comment.go", "package x\n// password credential token secret\nvar clean = 1\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	sites := map[string]bool{}
	ast.Inspect(syntax, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok {
			admitName(sites, "x", "decl", identifier.Name)
		}
		if literal, ok := node.(*ast.BasicLit); ok && literal.Kind == token.STRING {
			admitLiteral(sites, "x", literal.Value)
		}
		return true
	})
	if len(sites) != 0 {
		t.Fatalf("comment tokens rostered sites %v", sites)
	}
}

// secretSiteRow pairs derived secret sites sharing one disposition with
// the witnesses that prove it. Every derived site appears in exactly one
// row: a derived site with no row fails as unaccounted, and a row naming
// a site production no longer derives fails as orphaned.
type secretSiteRow struct {
	sites       []string
	disposition string
	witnesses   []string // "package.TestName" entries, verified to exist
}

// secretSiteRows is the reviewed disposition half of the census. The
// denominator is derived (deriveSecretSites); these rows say what each
// site is and which test proves it stays that way.
func secretSiteRows() []secretSiteRow {
	return []secretSiteRow{
		{
			sites: []string{
				"axerror|lit|access_token", "axerror|lit|api_key",
				"axerror|lit|cookie", "axerror|lit|cookies",
				"axerror|lit|credential", "axerror|lit|credentials",
				"axerror|lit|env_secret", "axerror|lit|environment secret",
				"axerror|lit|environment_secret", "axerror|lit|oauth_token",
				"axerror|lit|passphrase", "axerror|lit|password",
				"axerror|lit|private_key", "axerror|lit|refresh_token",
				"axerror|lit|secret", "axerror|lit|secrets",
				"axerror|lit|session_token", "axerror|lit|ssh_private_key",
				"axerror|lit|subscription_token",
				"axerror|lit|raw_transcript", "axerror|lit|transcript",
				"axerror|lit|scrollback", "axerror|lit|terminal_scrollback",
				"axerror|lit|raw transcript",
			},
			disposition: "members of the axerror excludedDetailKeys refusal table: a Structured Error detail " +
				"naming one is refused before encoding, so no credential, transcript, environment secret, or " +
				"bundle key reaches the wire as a detail",
			witnesses: []string{"secprim.TestSecretAxerrorTableRefused"},
		},
		{
			// The derived members of the secprim sensitiveKeyNames
			// scrubber table. "auth_json", "authorization", and "dotenv"
			// are table members too, but no instrument token fires on
			// them (the Authority blind spot pinned by
			// TestSecretScanBlindSpot); TestSensitiveKeyListIsReviewed
			// pins the whole table of 27 including those three, so the
			// table cannot gain or lose a member without reddening
			// there.
			sites: []string{
				"secprim|lit|access_token", "secprim|lit|api_key", "secprim|lit|apikey",
				"secprim|lit|auth_token", "secprim|lit|bearer_token",
				"secprim|lit|client_secret", "secprim|lit|cookie", "secprim|lit|cookies",
				"secprim|lit|credential", "secprim|lit|credentials",
				"secprim|lit|env_secret", "secprim|lit|environment_secret", "secprim|lit|oauth_token",
				"secprim|lit|passphrase", "secprim|lit|passwd", "secprim|lit|password",
				"secprim|lit|private_key", "secprim|lit|refresh_token", "secprim|lit|secret",
				"secprim|lit|secrets", "secprim|lit|session_token", "secprim|lit|ssh_private_key",
				"secprim|lit|subscription_token", "secprim|lit|token",
			},
			disposition: "members of the secprim sensitiveKeyNames scrubber table: a free-text key=value pair " +
				"naming one has its value masked by Redact before persistence or display",
			witnesses: []string{"secprim.TestSensitiveKeyListIsReviewed", "secprim.TestRedactSensitivePairs"},
		},
		{
			sites: []string{
				"secprim|decl|minSecretRunes", "secprim|decl|privateKeyBeginPrefix",
				"secprim|decl|privateKeyBeginSuffix", "secprim|decl|privateKeyEndPrefix",
				"secprim|decl|redactPrivateKeyBlocks", "secprim|decl|redactedPrivateKey",
				"secprim|param|secrets",
			},
			disposition: "the scrubber machinery itself: private-key block shapes, the corpus bound, and the " +
				"caller-known secret values Redact removes rather than stores",
			witnesses: []string{"secprim.TestRedactPrivateKeyBlocks", "secprim.TestRedactCorpusValues"},
		},
		{
			sites: []string{
				"terminalbackend|lit|attach_credential", "terminalbackend|lit|relay_credential",
				"terminalbackend|lit|provider_credential", "terminalbackend|lit|evidence_secret",
				"terminalbackend|lit|credential_detail", "terminalbackend|lit|token",
				"terminalbackend|lit|scrollback",
			},
			disposition: "members of the terminal replication-exclusion vocabulary: classified sensitive-owner-only " +
				"or forbidden, never transferred as live authority",
			witnesses: []string{"terminalbackend.TestReplicationClassificationIsClosed"},
		},
		{
			sites: []string{
				"terminalbackend|decl|credentialRealmCapability",
				"terminalbackend|lit|credential_capable_execution_realm",
				"terminalbackend|lit|credential_sentinel",
				"terminalbackend|lit|scrollback_retention",
			},
			disposition: "capability vocabulary words: they name the credential-capable realm, its sentinel " +
				"evidence member, and the scrollback-retention policy, whose values are booleans, enums, and " +
				"retention facts, never secrets or transcript content",
			witnesses: []string{"terminalbackend.TestParseManifestAdmitsClosedFixture"},
		},
		{
			sites: []string{"sessadapter|lit|credential"},
			disposition: "member of the closed capture-class vocabulary: naming the class a capture plan " +
				"excludes, not a credential value",
			witnesses: []string{"sessadapter.TestDecodeCapturePlanItemRules"},
		},
		{
			sites: []string{
				"provhost|lit|rollback_token",
				"provhost|lit|status rollback_token is not a string",
				"provhost|lit|status rollback_token is shorter than 256 bits",
				"provhost|lit|unknown status carries plan, token, or discovery",
				"provhost|lit|prepared status misses plan, token, or discovery",
				"provhost|lit|terminal status misses plan or discovery or keeps a token",
				"provhost|local|token",
			},
			disposition: "the closed status member plus its charset/length gate: the decoded rollback token is " +
				"checked for 256-bit base64url shape only, and every refusal names the member, never the value",
			witnesses: []string{"provhost.TestDecodeStatusOutcomeRefusesMalformedBodies"},
		},
		{
			sites: []string{
				"scalar|lit|must not contain userinfo credentials",
				"scalar|lit|must not contain a password or token",
				"scalar|local|password",
			},
			disposition: "the sanitized-Git-URL refusal: userinfo credentials and passwords are refused at parse, " +
				"and the decoded local is compared and dropped, never rendered",
			witnesses: []string{"scalar.TestSanitizedGitURLRefusesCredentialsAndLocalSchemes"},
		},
		{
			sites: []string{
				"canonicaljson|lit|contains_secrets",
				"canonicaljson|lit|Session Record Launch Plan contains_secrets must be false",
				"canonicaljson|local|containsSecrets",
			},
			disposition: "the launch-plan secrecy flag: a plan declaring contains_secrets is refused, so a " +
				"secret-bearing plan cannot enter an immutable record",
			witnesses: []string{"canonicaljson.TestSessionRecordV1NestedTaggedShapesReachBothIdentityEntries"},
		},
		{
			sites: []string{
				"canonicaljson|lit|decode JSON token: %v",
				"canonicaljson|lit|unexpected trailing JSON token %v",
			},
			disposition: "constant decoder diagnostics naming JSON tokens (syntax elements), not secret values; " +
				"they carry the offending input shape, never a credential",
			witnesses: []string{"canonicaljson.TestCalculateObjectIdentityRefusesMalformedJSONInput"},
		},
		{
			sites: []string{
				"dirnode|lit|bootstrap process token is empty",
				"dirnode|param|token",
			},
			disposition: "the opaque bootstrap idempotency token: compared in-memory only, and refusals name " +
				"the member, never the value",
			witnesses: []string{"dirnode.TestProcessGuardRefusesReuse"},
		},
		{
			sites: []string{
				"config|field|TranscriptGrepEnabled",
				`config|lit|toml:"transcript_grep_enabled"`,
			},
			disposition: "the explicit transcript-grep opt-in switch: an admitted boolean member, defaulting " +
				"to false, gating source-local bounded grep per Section 16.7 — not a transcript value",
			witnesses: []string{"config.TestLoadAcceptsExactV2DirectoryAndV3TerminalShapes"},
		},
	}
}

// TestSecretSiteRosterIsComplete requires every derived secret site to
// carry exactly one row and every row site to resolve to a derived site.
// Removing a secret-bearing declaration, or adding one, reddens here in
// exactly one direction.
func TestSecretSiteRosterIsComplete(t *testing.T) {
	derived := deriveSecretSites(t)
	rows := secretSiteRows()
	seen := map[string]bool{}
	for _, row := range rows {
		for _, site := range row.sites {
			if !derived[site] {
				t.Fatalf("orphaned census row site %q: production derives no such site", site)
			}
			if seen[site] {
				t.Fatalf("duplicate census row site %q: one disposition per site", site)
			}
			seen[site] = true
		}
	}
	var missing []string
	for site := range derived {
		if !seen[site] {
			missing = append(missing, site)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("derived secret sites with no disposition row:\n  %s", strings.Join(missing, "\n  "))
	}
}

// TestSecretSiteWitnessesExist verifies every cited witness names a real
// test function in its package, parsed from that package's test sources:
// a renamed or deleted witness reddens here instead of citing a ghost.
func TestSecretSiteWitnessesExist(t *testing.T) {
	t.Parallel()
	needed := map[string]map[string]bool{}
	for _, row := range secretSiteRows() {
		for _, witness := range row.witnesses {
			parts := strings.Split(witness, ".")
			if len(parts) != 2 {
				t.Fatalf("witness %q is not package.Test shaped", witness)
			}
			if needed[parts[0]] == nil {
				needed[parts[0]] = map[string]bool{}
			}
			needed[parts[0]][parts[1]] = true
		}
	}
	for pkg, names := range needed {
		found := testFuncNames(t, pkg)
		for name := range names {
			if !found[name] {
				t.Errorf("witness %s.%s does not exist; the census cites a ghost", pkg, name)
			}
		}
	}
}

// testFuncNames parses a package's test files for test function names.
// The walk is over declarations, so a name mentioned in a comment or a
// string cannot satisfy it.
func testFuncNames(t *testing.T, pkg string) map[string]bool {
	t.Helper()
	directory := filepath.Join("..", pkg)
	if pkg == "secprim" {
		directory = "."
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read %s: %v", directory, err)
	}
	found := map[string]bool{}
	scanned := false
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		scanned = true
		syntax, err := parser.ParseFile(token.NewFileSet(), name, contents, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range syntax.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			if strings.HasPrefix(function.Name.Name, "Test") {
				found[function.Name.Name] = true
			}
		}
	}
	if !scanned {
		t.Fatalf("scanned no test files in %s; the witness check is blind", pkg)
	}
	return found
}

// TestSecretTokensAreProven requires every instrument token to fire on at
// least one derived site: a token that matches nothing is dead ceremony,
// and removing a live token orphans its sites in the roster above.
func TestSecretTokensAreProven(t *testing.T) {
	t.Parallel()
	derived := deriveSecretSites(t)
	for _, word := range secretTokens {
		fires := false
		for site := range derived {
			if strings.Contains(strings.ToLower(site), word) {
				fires = true
				break
			}
		}
		if !fires {
			t.Errorf("token %q fires on no derived site; the instrument is blind to its class", word)
		}
	}
}

// excludedAxerrorKeys parses the excludedDetailKeys table out of the
// axerror package source, so the re-drive below follows the table instead
// of retyping it: an added key is driven automatically, and a removed key
// cannot be driven against a stale copy.
func excludedAxerrorKeys(t *testing.T) []string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "axerror", "details.go"))
	if err != nil {
		t.Fatal(err)
	}
	syntax, err := parser.ParseFile(token.NewFileSet(), "details.go", contents, 0)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	ast.Inspect(syntax, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "excludedDetailKeys" {
			return true
		}
		if len(spec.Values) != 1 {
			t.Fatal("excludedDetailKeys has no single initializer; the derivation is blind")
		}
		table, ok := spec.Values[0].(*ast.CompositeLit)
		if !ok {
			t.Fatal("excludedDetailKeys is not a composite literal; the derivation is blind")
		}
		for _, element := range table.Elts {
			pair, ok := element.(*ast.KeyValueExpr)
			if !ok {
				t.Fatal("excludedDetailKeys carries a non-pair element; the derivation is blind")
			}
			literal, ok := pair.Key.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				t.Fatal("excludedDetailKeys carries a non-literal key; the derivation is blind")
			}
			key, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			keys = append(keys, key)
		}
		return false
	})
	if len(keys) == 0 {
		t.Fatal("derived no excluded detail keys; the scanner is broken, not the table")
	}
	return keys
}

// TestSecretAxerrorTableRefused drives every key of the axerror exclusion
// table through the production Structured Error constructor and requires
// a refusal: the gate is proven on the table it owns, including the keys
// (auth_json, bundle_*, transcript shapes) no census token fires on.
func TestSecretAxerrorTableRefused(t *testing.T) {
	t.Parallel()
	for _, key := range excludedAxerrorKeys(t) {
		_, err := axerror.New(axerror.Spec{
			Version: axerror.Version100,
			Code:    "provider_process_failed",
			Message: "probe",
			IDs:     axerror.NoIDs(),
			Details: axerror.Details{key: "value"},
		})
		if err == nil {
			t.Errorf("axerror.New admitted excluded detail key %q", key)
			continue
		}
		if !errors.Is(err, axerror.ErrInvalidDetails) {
			t.Errorf("axerror.New(%q) = %v, want ErrInvalidDetails", key, err)
		}
	}
}

// TestSecretScanBlindSpot pins the deliberate Authority blind spot: bare
// "auth"-substring declarations (authority identifiers, authorization
// enums) are protocol vocabulary, never credential values, and the
// instrument does not roster them. A secret named only with "auth" and
// none of the instrument tokens would be missed; that residue is stated
// here, not hidden.
func TestSecretScanBlindSpot(t *testing.T) {
	t.Parallel()
	sites := map[string]bool{}
	admitName(sites, "x", "decl", "ScanRootAuthorityIDs")
	admitName(sites, "x", "field", "AuthSubject")
	admitLocal(sites, "x", "auth")
	if len(sites) != 0 {
		t.Fatalf("blind-spot declarations rostered sites %v; the bound moved", sites)
	}
	// And the instrument still sees the class it must roster.
	prove := map[string]bool{}
	admitName(prove, "x", "decl", "DBPassword")
	if len(prove) != 1 {
		t.Fatalf("instrument missed a password declaration: %v", prove)
	}
}
