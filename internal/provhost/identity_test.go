package provhost

import (
	"fmt"
	"github.com/relux-works/agent-session-manager/internal/invcore"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/canonicaljson"
)

// specIdentityExample is the Section 5.5 Provider Identity Record
// example, copied verbatim from the pinned document: an Antigravity
// backend_conversation_uuid with a non-null realm fingerprint.
const specIdentityExample = `{
  "schema": "urn:ax:schema:provider-identity",
  "schema_version": "1.0.0",
  "record_id": "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2",
  "subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab",
  "provider_id": "antigravity",
  "provider_version": "1.1.14",
  "provider_version_range": ">=1.1.14 <1.2.0",
  "native_session_id": "11111111-2222-4333-8444-555555555555",
  "identity_kind": "backend_conversation_uuid",
  "logical_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab",
  "backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad",
  "opaque_identity": {},
  "created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab",
  "created_at": "2026-08-19T04:09:45.000Z",
  "extensions": {}
}`

// TestSpecIdentityExampleDecodes proves the Section 5.5 example
// passes the production entry point for its provider.
func TestSpecIdentityExampleDecodes(t *testing.T) {
	if err := CheckIdentity([]byte(specIdentityExample), "antigravity"); err != nil {
		t.Fatalf("CheckIdentity(spec example): %v", err)
	}
}

// TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest pins the true
// fact the identity stated bound rests on: VerifyObjectIdentity over the
// pinned Section 5.5 example returns the example's own record_id with nil
// error, so the example IS the true omit-self digest. A bound claiming
// the example is merely illustrative — and justifying skipped attestation
// by it — fails here instead of shipping as a considered decision. The
// assertion is exact (field and digest), never a nil-error-only check
// that would also pass on a vacuous verifier.
func TestSpecIdentityExampleVerifiesAgainstItsClaimedDigest(t *testing.T) {
	verified, field, err := canonicaljson.VerifyObjectIdentity([]byte(specIdentityExample))
	if err != nil {
		t.Fatalf("VerifyObjectIdentity(spec example): %v", err)
	}
	if field != canonicaljson.SelfRecordID {
		t.Fatalf("VerifyObjectIdentity(spec example) field = %q, want record_id", field)
	}
	const want = "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2"
	if verified.String() != want {
		t.Fatalf("VerifyObjectIdentity(spec example) = %q, want the claimed %q", verified, want)
	}
}

// TestNoProductionPathAttestsProviderIdentityBinding retains its historical
// reviewer-test name and pins two facts: CheckIdentity accepts the Section 5.5
// example by its shape even though the generic verifier rejects the mismatched
// self-digest, and a production AST census names every direct
// VerifyObjectIdentity call site. The census permits the five sessrepo
// durable-record sites, merkleinventory.ClassifyJSON, and the four Git
// assembly identity checks. Its syntactic bound does not include indirect
// function-value calls.
func TestNoProductionPathAttestsProviderIdentityBinding(t *testing.T) {
	raw := identityVariant(t, "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2", "sha256:"+strings.Repeat("0", 64))
	if _, _, err := canonicaljson.VerifyObjectIdentity(raw); err == nil {
		t.Fatal("negative control is not a mismatched self-identity")
	}
	if err := CheckIdentity(raw, "antigravity"); err != nil {
		t.Fatalf("shape-only provider identity admission changed: %v", err)
	}
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller cannot locate the test file")
	}
	root := filepath.Dir(self)
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("no module root above the test file; the scan is blind")
		}
		root = parent
	}
	allowed, callers, scanned, parsed := scanAttestationSites(t, root)
	if scanned == 0 {
		t.Fatal("scanned no production sources; the check is blind")
	}
	if len(allowed) != 10 {
		t.Fatalf("identity attestation sites = %d, want exactly 10 (five session-leaf sites, one inventory classifier, and four Git assembly checks): a moved call site must rewrite the bound, a new one must justify itself, a stale exception must go", len(allowed))
	}
	if len(callers) != 0 {
		t.Fatalf("unowned production call sites attest an identity binding, contradicting the stated bound:\n  %s", strings.Join(callers, "\n  "))
	}
	sessionSites, inventorySites, gitAssemblySites := 0, 0, 0
	for _, site := range allowed {
		switch {
		case strings.Contains(site, filepath.Join("internal", "sessrepo")+string(filepath.Separator)):
			sessionSites++
		case strings.Contains(site, filepath.Join("internal", "merkleinventory", "index.go")):
			inventorySites++
		case strings.Contains(site, filepath.Join("internal", "gitsnap", "assembly.go")):
			gitAssemblySites++
		}
	}
	if sessionSites != 5 || inventorySites != 1 || gitAssemblySites != 4 {
		t.Fatalf("identity attestation ownership = session %d, inventory %d, Git assembly %d; want 5, 1, and 4: %q", sessionSites, inventorySites, gitAssemblySites, allowed)
	}
	t.Logf("attestation scan: %d production files, %d parsed, five session-leaf sites, one inventory classifier site, four Git assembly sites, 0 elsewhere", scanned, parsed)
}

// isSessrepoLeafFile reports whether a scan path sits inside the owning
// session-persistence directory: <root>/internal/sessrepo/. The match
// anchors on the full module-relative owning path, never on a bare
// segment, so internal/provhost/sessrepo/plant.go (same leaf name,
// foreign owner) and internal/xsessrepo/plant.go (near-miss name) both
// land outside the allowlist while the leaf's own files stay admitted.
func isSessrepoLeafFile(root, path string) bool {
	leaf := filepath.Join(root, "internal", "sessrepo") + string(filepath.Separator)
	return strings.HasPrefix(path, leaf)
}

// isInventoryIdentityAttestationFile admits only the inventory classifier
// file, where a verified immutable ID is required before Merkle membership.
// Anchoring the full path rejects similarly named foreign or sibling trees.
func isInventoryIdentityAttestationFile(root, path string) bool {
	return filepath.Clean(path) == filepath.Join(root, "internal", "merkleinventory", "index.go")
}

// isGitAssemblyIdentityFile admits only the capture assembly's four immutable
// identity checks. A same-named file in another package is not an owner.
func isGitAssemblyIdentityFile(root, path string) bool {
	return filepath.Clean(path) == filepath.Join(root, "internal", "gitsnap", "assembly.go")
}

// scanAttestationSites walks the production trees beneath root and
// classifies every VerifyObjectIdentity call: inside one of the three
// exact owners it is allowlisted, anywhere else it contradicts the bound. It returns
// the allowlisted entries, the contradicting entries, and the scanned
// and parsed file counts.
func scanAttestationSites(t *testing.T, root string) (allowed, callers []string, scanned, parsed int) {
	t.Helper()
	for _, tree := range []string{"internal", "cmd"} {
		base := filepath.Join(root, tree)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		// Recursion stays local: the core selects single-package
		// production files, while this attestation spans trees.
		// Fail-closed parsing is the core's.
		if err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			scanned++
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			if !strings.Contains(string(source), "VerifyObjectIdentity") {
				return nil
			}
			parsed++
			syntax, fileSet, failure := invcore.ParseBytes(path, source, 0)
			if failure != "" {
				t.Fatalf("parse %s: %s", path, failure)
			}
			ast.Inspect(syntax, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				record := func(position token.Position) {
					entry := fmt.Sprintf("%s:%d", position.Filename, position.Line)
					if isSessrepoLeafFile(root, path) || isInventoryIdentityAttestationFile(root, path) || isGitAssemblyIdentityFile(root, path) {
						allowed = append(allowed, entry)
						return
					}
					callers = append(callers, entry)
				}
				switch fun := call.Fun.(type) {
				case *ast.SelectorExpr:
					if fun.Sel.Name == "VerifyObjectIdentity" {
						record(fileSet.Position(call.Pos()))
					}
				case *ast.Ident:
					if fun.Name == "VerifyObjectIdentity" {
						// The canonicaljson definition itself is a
						// FuncDecl, never a call; a bare-identifier call
						// anywhere is a production call site.
						record(fileSet.Position(call.Pos()))
					}
				}
				return true
			})
			return nil
		}); err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
	return allowed, callers, scanned, parsed
}

// TestAttestationAllowlistAnchorsOwningPath drives all three owners through a
// synthetic module tree. A plant under internal/provhost/sessrepo/, a
// near-miss under internal/xsessrepo/, a foreign merkleinventory tree,
// any inventory file other than index.go, and any Git assembly sibling or
// near-match land outside the allowlist. The production suite executes behavioral tests alongside
// token-preserving mutants in the inventory task's battery.
func TestAttestationAllowlistAnchorsOwningPath(t *testing.T) {
	root := t.TempDir()
	leafCall := "package sessrepo\nfunc load(raw []byte) {\n\t_, _, _ = canonicaljson.VerifyObjectIdentity(raw)\n}\n"
	inventoryCall := "package merkleinventory\nfunc classify(raw []byte) {\n\t_, _, _ = canonicaljson.VerifyObjectIdentity(raw)\n}\n"
	foreignCall := "package plant\nfunc plant(raw []byte) {\n\t_, _, _ = canonicaljson.VerifyObjectIdentity(raw)\n}\n"
	for _, file := range []struct{ rel, body string }{
		{"internal/sessrepo/chain.go", leafCall},
		{"internal/merkleinventory/index.go", inventoryCall},
		{"internal/gitsnap/assembly.go", inventoryCall},
		{"internal/provhost/sessrepo/plant.go", foreignCall},
		{"internal/xsessrepo/plant.go", foreignCall},
		{"internal/provhost/merkleinventory/index.go", foreignCall},
		{"internal/xmerkleinventory/index.go", foreignCall},
		{"internal/merkleinventory/other.go", foreignCall},
		{"internal/gitsnap/other.go", foreignCall},
		{"internal/xgitsnap/assembly.go", foreignCall},
	} {
		path := filepath.Join(root, filepath.FromSlash(file.rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(file.body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	allowed, callers, scanned, parsed := scanAttestationSites(t, root)
	if scanned != 10 || parsed != 10 {
		t.Fatalf("synthetic scan files = %d scanned %d parsed, want 10 and 10", scanned, parsed)
	}
	if len(allowed) != 3 {
		t.Fatalf("synthetic allowlisted sites = %q, want the session leaf, inventory classifier, and Git assembly", allowed)
	}
	joined := strings.Join(allowed, "\n")
	if !strings.Contains(joined, filepath.Join("internal", "sessrepo", "chain.go")) || !strings.Contains(joined, filepath.Join("internal", "merkleinventory", "index.go")) || !strings.Contains(joined, filepath.Join("internal", "gitsnap", "assembly.go")) {
		t.Fatalf("synthetic allowlisted sites = %q, want the session leaf, inventory index, and Git assembly", allowed)
	}
	if len(callers) != 7 {
		t.Fatalf("synthetic contradicting sites = %q, want foreign and sibling plants", callers)
	}
}

// TestCheckIdentityDoesNotVerifyBinding measures the stated bound at its real
// entry. A generic verifier call elsewhere can validate another schema and says
// nothing about provider identity. No repository-wide absence claim is made.
func TestCheckIdentityDoesNotVerifyBinding(t *testing.T) {
	for _, claimed := range []string{"sha256:" + strings.Repeat("0", 64), "sha256:" + strings.Repeat("f", 64)} {
		raw := identityVariant(t, "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2", claimed)
		if _, _, err := canonicaljson.VerifyObjectIdentity(raw); err == nil {
			t.Fatal("negative control is not a mismatched self-identity")
		}
		if err := CheckIdentity(raw, "antigravity"); err != nil {
			t.Fatalf("documented shape-only provider identity admission changed: %v", err)
		}
	}
}

// identityVariant rewrites one unique substring of the example.
func identityVariant(t *testing.T, old, new string) []byte {
	t.Helper()
	if strings.Count(specIdentityExample, old) != 1 {
		t.Fatalf("identity variant anchor %q is not unique", old)
	}
	return []byte(strings.Replace(specIdentityExample, old, new, 1))
}

// TestCheckIdentityRefusals drives every identity rule, including
// the five normative Section 5.5 negatives: unknown identity_kind,
// absent opaque_identity, an object opaque value, an absolute source
// path in that map, and an Antigravity backend_conversation_uuid
// with a null realm fingerprint.
func TestCheckIdentityRefusals(t *testing.T) {
	long128 := strings.Repeat("v", 128)
	long129 := strings.Repeat("v", 129)
	long1024 := strings.Repeat("v", 1024)
	long1025 := strings.Repeat("v", 1025)
	long256 := strings.Repeat("r", 256)
	long257 := strings.Repeat("r", 257)
	long512 := strings.Repeat("s", 512)
	long513 := strings.Repeat("s", 513)
	key64 := "k" + strings.Repeat("e", 63)
	key65 := "k" + strings.Repeat("e", 64)
	rows := []struct {
		name   string
		body   []byte
		member string
		detail string
	}{
		{"unknown member", identityVariant(t, `"extensions": {}`, `"extensions": {}, "score": 1`), "score", "unknown member"},
		{"absent opaque_identity", []byte(strings.Replace(specIdentityExample, "  \"opaque_identity\": {},\n", "", 1)), "opaque_identity", "misses a required member"},
		{"wrong schema", identityVariant(t, `"urn:ax:schema:provider-identity"`, `"urn:ax:schema:provider-manifest"`), "schema", "not the provider identity"},
		{"wrong schema version", identityVariant(t, `"schema_version": "1.0.0"`, `"schema_version": "1.1.0"`), "schema_version", "not 1.0.0"},
		{"bad record digest", identityVariant(t, `"record_id": "sha256:c879d766da67a8cfb3a3f6eae2234faa5d52d8df987496eae2218f40e5e220c2"`, `"record_id": "sha256:zzzz"`), "record_id", "not a digest"},
		{"bad subject uuid", identityVariant(t, `"subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"subject_id": "not-a-uuid"`), "subject_id", "not a UUIDv7"},
		{"bad session uuid", identityVariant(t, `"session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"session_id": "0198f4c8-3e70-7a11-8a2b-1234567890ax"`), "session_id", "not a UUIDv7"},
		{"subject differs", identityVariant(t, `"subject_id": "0198f4c8-3e70-7a11-8a2b-1234567890ab"`, `"subject_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab"`), "subject_id", "does not equal session_id"},
		{"bad provider grammar", identityVariant(t, `"provider_id": "antigravity"`, `"provider_id": "Antigravity"`), "provider_id", "not a provider id"},
		{"empty provider version", identityVariant(t, `"provider_version": "1.1.14"`, `"provider_version": ""`), "provider_version", "not 1..128 characters"},
		{"provider version 129", identityVariant(t, `"provider_version": "1.1.14"`, `"provider_version": "`+long129+`"`), "provider_version", "not 1..128 characters"},
		{"empty version range", identityVariant(t, `"provider_version_range": ">=1.1.14 <1.2.0"`, `"provider_version_range": ""`), "provider_version_range", "not 1..256 characters"},
		{"version range 257", identityVariant(t, `"provider_version_range": ">=1.1.14 <1.2.0"`, `"provider_version_range": "`+long257+`"`), "provider_version_range", "not 1..256 characters"},
		{"empty native session", identityVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": ""`), "native_session_id", "not 1..512 characters"},
		{"native session 513", identityVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": "`+long513+`"`), "native_session_id", "not 1..512 characters"},
		{"unknown identity_kind", identityVariant(t, `"identity_kind": "backend_conversation_uuid"`, `"identity_kind": "window_handle"`), "identity_kind", "not a registry member"},
		{"bad workspace uuid", identityVariant(t, `"logical_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890ab"`, `"logical_workspace_id": "0198f4c8-6c30-7d44-8d5e-1234567890aX"`), "logical_workspace_id", "not a UUIDv7"},
		{"bad realm digest", identityVariant(t, `"backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"`, `"backend_realm_fingerprint": "nope"`), "backend_realm_fingerprint", "not a digest or null"},
		{"antigravity null realm", identityVariant(t, `"backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"`, `"backend_realm_fingerprint": null`), "backend_realm_fingerprint", "required for this backend kind"},
		{"opaque object value", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"adapter": {"nested": true}}`), "opaque_identity", "not 1..1024 characters"},
		{"opaque absolute path", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"workdir": "/Users/iv/work"}`), "opaque_identity", "begins with an absolute path"},
		{"opaque windows path", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"workdir": "C:\\work"}`), "opaque_identity", "begins with an absolute path"},
		{"opaque bad key", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"Bad Key!": "x"}`), "opaque_identity", "not a provider key"},
		{"opaque digit-first key", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"0adapter": "x"}`), "opaque_identity", "not a provider key"},
		{"opaque 65-character key", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"`+key65+`": "x"}`), "opaque_identity", "not a provider key"},
		{"opaque empty value", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"adapter": ""}`), "opaque_identity", "not 1..1024 characters"},
		{"opaque long value", identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"adapter": "`+long1025+`"}`), "opaque_identity", "not 1..1024 characters"},
		{"bad host uuid", identityVariant(t, `"created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890ab"`, `"created_by_host_id": "0198f4c8-4a10-7b22-8b3c-1234567890aG"`), "created_by_host_id", "not a UUIDv7"},
		{"bad timestamp", identityVariant(t, `"created_at": "2026-08-19T04:09:45.000Z"`, `"created_at": "yesterday"`), "created_at", "not a timestamp"},
		{"extensions array", identityVariant(t, `"extensions": {}`, `"extensions": []`), "extensions", "not an object"},
		{"raw WTF-8 surrogate", identityVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, "\"native_session_id\": \"11111\xed\xa0\x801111-2222-4333-8444-555555555555\""), "", "not valid UTF-8"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireFrameRefusal(t, CheckIdentity(row.body, "antigravity"), row.member, row.detail)
		})
	}
	if err := CheckIdentity(identityVariant(t, `"provider_version": "1.1.14"`, `"provider_version": "`+long128+`"`), "antigravity"); err != nil {
		t.Fatalf("128-character provider version refused: %v", err)
	}
	if err := CheckIdentity(identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"adapter": "`+long1024+`"}`), "antigravity"); err != nil {
		t.Fatalf("1024-character opaque value refused: %v", err)
	}
	if err := CheckIdentity(identityVariant(t, `"provider_version_range": ">=1.1.14 <1.2.0"`, `"provider_version_range": "`+long256+`"`), "antigravity"); err != nil {
		t.Fatalf("256-character version range refused: %v", err)
	}
	if err := CheckIdentity(identityVariant(t, `"native_session_id": "11111111-2222-4333-8444-555555555555"`, `"native_session_id": "`+long512+`"`), "antigravity"); err != nil {
		t.Fatalf("512-character native session refused: %v", err)
	}
	if err := CheckIdentity(identityVariant(t, `"opaque_identity": {}`, `"opaque_identity": {"`+key64+`": "x"}`), "antigravity"); err != nil {
		t.Fatalf("64-character opaque key refused: %v", err)
	}
}

// TestCheckIdentityNamesAnotherProvider proves the caller seat: a
// well-formed record for another provider is a caller error, not a
// frame error, and a null realm is honest for non-Antigravity kinds.
func TestCheckIdentityNamesAnotherProvider(t *testing.T) {
	requireLocalRefusal(t, CheckIdentity([]byte(specIdentityExample), "codex"), "invalid_config", "names another provider")
	// Equal-length pair: claude and gemini are both six characters,
	// so a length-based correlation passes this fixture and fails in
	// production. The gate must compare the provider itself.
	claude := identityVariant(t, `"identity_kind": "backend_conversation_uuid"`, `"identity_kind": "session_uuid"`)
	claude = []byte(strings.Replace(string(claude), `"provider_id": "antigravity"`, `"provider_id": "claude"`, 1))
	claude = []byte(strings.Replace(string(claude), `"backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"`, `"backend_realm_fingerprint": null`, 1))
	if len("claude") != len("gemini") {
		t.Fatal("claude and gemini are no longer the same length; the fixture is blind")
	}
	requireLocalRefusal(t, CheckIdentity(claude, "gemini"), "invalid_config", "names another provider")
	otherKind := identityVariant(t, `"identity_kind": "backend_conversation_uuid"`, `"identity_kind": "session_uuid"`)
	otherKind = []byte(strings.Replace(string(otherKind), `"provider_id": "antigravity"`, `"provider_id": "codex"`, 1))
	otherKind = []byte(strings.Replace(string(otherKind), `"backend_realm_fingerprint": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaad"`, `"backend_realm_fingerprint": null`, 1))
	if err := CheckIdentity(otherKind, "codex"); err != nil {
		t.Fatalf("codex session_uuid with null realm refused: %v", err)
	}
}

// specIdentifyResult wraps the identity example in the
// identify-session success shape.
const specIdentifyResultPrefix = `{
  "identity": `

// TestDecodeIdentifyResultFixtures proves the wrapper: the example
// record with exact confidence and one evidence member validates,
// and every wrapper rule refuses with its own arm.
func TestDecodeIdentifyResultFixtures(t *testing.T) {
	indented := strings.ReplaceAll(specIdentityExample, "\n", "\n  ")
	body := []byte(specIdentifyResultPrefix + "  " + strings.TrimPrefix(indented, "  ") + `,
  "confidence": "exact",
  "matched_evidence": ["native_id"]
}`)
	if err := DecodeIdentifyResult(body, "antigravity"); err != nil {
		t.Fatalf("DecodeIdentifyResult(spec): %v", err)
	}
	with := func(old, new string) []byte {
		t.Helper()
		full := string(body)
		if strings.Count(full, old) != 1 {
			t.Fatalf("identify variant anchor %q is not unique", old)
		}
		return []byte(strings.Replace(full, old, new, 1))
	}
	rows := []struct {
		name   string
		body   []byte
		member string
		detail string
	}{
		{"unknown member", with(`"matched_evidence": ["native_id"]`, `"matched_evidence": ["native_id"], "score": 1`), "score", "unknown member"},
		{"missing member", []byte(strings.Replace(string(body), "  \"confidence\": \"exact\",\n", "", 1)), "confidence", "misses a required member"},
		{"bad confidence", with(`"confidence": "exact"`, `"confidence": "certain"`), "confidence", "not exact strong or weak"},
		{"evidence empty", with(`"matched_evidence": ["native_id"]`, `"matched_evidence": []`), "matched_evidence", "not 1..4 members"},
		{"evidence five", with(`"matched_evidence": ["native_id"]`, `"matched_evidence": ["native_id", "store_path", "provider_event", "backend_lookup", "native_id"]`), "matched_evidence", "not 1..4 members"},
		{"evidence unknown", with(`"matched_evidence": ["native_id"]`, `"matched_evidence": ["window_title"]`), "matched_evidence", "unknown member"},
		{"evidence unsorted", with(`"matched_evidence": ["native_id"]`, `"matched_evidence": ["store_path", "native_id"]`), "matched_evidence", "not sorted unique"},
		{"evidence duplicated", with(`"matched_evidence": ["native_id"]`, `"matched_evidence": ["native_id", "native_id"]`), "matched_evidence", "not sorted unique"},
		{"identity not object", []byte(`{"identity": [], "confidence": "exact", "matched_evidence": ["native_id"]}`), "identity", "not an object"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			requireFrameRefusal(t, DecodeIdentifyResult(row.body, "antigravity"), row.member, row.detail)
		})
	}
	// A well-formed record for another provider is a caller error,
	// not a frame error.
	requireLocalRefusal(t, DecodeIdentifyResult(body, "codex"), "invalid_config", "names another provider")
}
