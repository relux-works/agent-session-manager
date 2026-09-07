package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/relux-works/agent-session-manager/internal/invcore"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// liftVector is one production-derived discovery failure and the secrets
// its local rendering carries: the machine-local paths, provider IDs,
// and owner identities the wire message must not reproduce.
type liftVector struct {
	name string
	// make drives the real production entry (Discover or Verify) to a
	// refusal.
	make func(t *testing.T) error
	// secrets are concrete local strings the failure's Error() carries.
	secrets []string
	// code is the stable provider code the failure carries.
	code string
	// message is the exact static wire message Lift builds for it.
	message string
	// exit is the Section 15.2 status of the lifted code.
	exit int
}

func liftVectors() []liftVector {
	return []liftVector{
		{
			name: "duplicate invalid_config",
			make: func(t *testing.T) error {
				t.Helper()
				fake := newFakeSystem()
				fake.addFile("/plugins/a", "ax-provider-dup", []byte("one"), fakeUID)
				fake.addFile("/plugins/b", "ax-provider-dup", []byte("two"), fakeUID)
				_, err := Discover(fakeConfig("/plugins/a", "/plugins/b"), fakeOwner(), fake)
				if err == nil {
					t.Fatal("Discover admitted a duplicate provider ID")
				}
				return err
			},
			secrets: []string{"dup"},
			code:    codeInvalidConfig,
			message: "provider discovery refused the request: invalid configuration",
			exit:    3,
		},
		{
			name: "unreadable directory local_precondition_failed",
			make: func(t *testing.T) error {
				t.Helper()
				fake := newFakeSystem()
				fake.entries["/plugins"] = []string{}
				fake.readDirErr["/plugins"] = errors.New("permission denied at /plugins")
				_, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
				if err == nil {
					t.Fatal("Discover admitted an unreadable plugin directory")
				}
				return err
			},
			secrets: []string{"/plugins", "permission denied at /plugins"},
			code:    codeLocalPrecondition,
			message: "provider discovery could not establish its filesystem preconditions",
			exit:    3,
		},
		{
			name: "retargeted executable integrity_failure",
			make: func(t *testing.T) error {
				t.Helper()
				fake := newFakeSystem()
				fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
				record := trustOneFoo(t, discoverOneFoo(t, fake))
				fake.canon["/plugins/ax-provider-foo"] = "/opt/real/evil"
				fake.files["/opt/real/evil"] = fakeFile{content: []byte("bytes"), uid: fakeUID, regular: true}
				err := Verify(record, fakeOwner(), fake)
				if err == nil {
					t.Fatal("Verify accepted a retargeted symlink")
				}
				return err
			},
			secrets: []string{"foo", "/opt/real/foo", "/opt/real/evil"},
			code:    codeIntegrityFailure,
			message: "provider trust receipt no longer matches the filesystem",
			exit:    9,
		},
		{
			name: "changed owner integrity_failure",
			make: func(t *testing.T) error {
				t.Helper()
				fake := newFakeSystem()
				fake.addLinkedFile("/plugins", "ax-provider-foo", "/opt/real/foo", []byte("bytes"), fakeUID, true)
				record := trustOneFoo(t, discoverOneFoo(t, fake))
				fake.files["/opt/real/foo"] = fakeFile{content: []byte("bytes"), uid: foreignUID, regular: true}
				err := Verify(record, fakeOwner(), fake)
				if err == nil {
					t.Fatal("Verify accepted a changed owner")
				}
				return err
			},
			secrets: []string{"foo", "uid:2000"},
			code:    codeIntegrityFailure,
			message: "provider trust receipt no longer matches the filesystem",
			exit:    9,
		},
	}
}

// TestLiftCarriesDiscoveryFailuresToTheWire drives the real lift: every
// production-derived failure above lifts through Lift into a Structured
// Error axerror accepts, carrying the stable code and exit with the
// exact static message, and no machine-local path, provider ID, or
// owner identity reaches the wire message. The secrets are asserted
// present in the local rendering first, so the absence below is a
// measured removal rather than a vacuous non-mention.
func TestLiftCarriesDiscoveryFailuresToTheWire(t *testing.T) {
	t.Parallel()

	for _, vector := range liftVectors() {
		t.Run(vector.name, func(t *testing.T) {
			t.Parallel()
			failure := vector.make(t)
			var local Error
			if !errors.As(failure, &local) {
				t.Fatalf("failure %v is not a provider Error", failure)
			}
			for _, secret := range vector.secrets {
				if !strings.Contains(local.Error(), secret) {
					t.Fatalf("local rendering %q does not carry %q; the wire absence below would prove nothing", local.Error(), secret)
				}
			}
			lifted, err := Lift(local)
			if err != nil {
				t.Fatalf("Lift: %v", err)
			}
			if lifted.Version() != axerror.Version100 {
				t.Fatalf("lifted version = %q, want Structured Error 1.0.0", lifted.Version())
			}
			if string(lifted.Code()) != vector.code {
				t.Fatalf("lifted code = %q, want %q", lifted.Code(), vector.code)
			}
			if lifted.Message() != vector.message {
				t.Fatalf("lifted message = %q, want %q", lifted.Message(), vector.message)
			}
			if lifted.ExitCode() != vector.exit {
				t.Fatalf("lifted exit = %d, want %d", lifted.ExitCode(), vector.exit)
			}
			if lifted.Retryable() {
				t.Fatalf("lifted failure claims retryable: %v", lifted)
			}
			if len(lifted.DetailKeys()) != 0 {
				t.Fatalf("lifted details = %v, want no diagnostic values", lifted.DetailKeys())
			}
			wire, err := json.Marshal(lifted)
			if err != nil {
				t.Fatalf("marshal lifted failure: %v", err)
			}
			for _, secret := range vector.secrets {
				if strings.Contains(string(wire), secret) {
					t.Fatalf("wire message %s reproduces local secret %q", wire, secret)
				}
			}
			// The cause stays local and usable: the lifted failure
			// unwraps to the provider refusal it was built from.
			var back Error
			if !errors.As(lifted, &back) {
				t.Fatalf("lifted failure does not unwrap to a provider Error")
			}
			if back.Code() != vector.code {
				t.Fatalf("unwrapped code = %q, want %q", back.Code(), vector.code)
			}
		})
	}
}

// TestLiftRefusesUnregisteredCode fails closed: an Error carrying a code
// outside the closed set lifts to no wire object, and the refusal is the
// registry's unregistered-code refusal rather than a minted default.
func TestLiftRefusesUnregisteredCode(t *testing.T) {
	t.Parallel()

	lifted, err := Lift(Error{code: "bogus_code", detail: "planted control"})
	if err == nil {
		t.Fatalf("Lift admitted an unregistered code: %v", lifted)
	}
	if lifted != nil {
		t.Fatalf("Lift returned a wire object alongside its refusal: %v", lifted)
	}
	if !errors.Is(err, axerror.ErrUnregisteredCode) {
		t.Fatalf("Lift refusal = %v, want the unregistered-code refusal", err)
	}
}

// TestLiftRefusesRegisteredCodeWithoutLiftArm drives the case the
// closed-set negative misses: a REGISTERED code with no lift arm.
// not_found is registered by Structured Error 1.0.0 (exit 4), so the
// registry admits it and only Lift's default arm decides. Fail-open
// would hand back a wire object carrying exit 4 under a generic message
// the lift author never reviewed for this code; fail-closed returns no
// wire object and hands the local failure back so the caller keeps the
// diagnostic without minting anything. The errors.Is refusal pins which
// sub-arm fired: the unregistered-code refusal must NOT fire here, which
// is also what pins the premise that not_found is registered.
func TestLiftRefusesRegisteredCodeWithoutLiftArm(t *testing.T) {
	t.Parallel()

	lifted, err := Lift(Error{code: "not_found", detail: "planted control"})
	if err == nil {
		t.Fatalf("Lift admitted a registered code with no lift arm: %v", lifted)
	}
	if lifted != nil {
		t.Fatalf("Lift returned a wire object alongside its refusal: %v", lifted)
	}
	if errors.Is(err, axerror.ErrUnregisteredCode) {
		t.Fatalf("Lift refusal took the unregistered-code arm for registered code not_found: %v", err)
	}
	var back Error
	if !errors.As(err, &back) {
		t.Fatalf("Lift refusal %v does not hand back the local failure", err)
	}
	if back.Code() != "not_found" {
		t.Fatalf("handed-back code = %q, want not_found", back.Code())
	}
}

// TestNaiveLiftShapeIsRefusedByTheWireGate proves the seam Lift exists
// to close: a message built from Error() with the cause attached — the
// ordinary fmt.Errorf("...: %v", err) lift — is refused by the
// production axerror gate with the causal-leak refusal, in both the
// message and the details direction.
func TestNaiveLiftShapeIsRefusedByTheWireGate(t *testing.T) {
	t.Parallel()

	fake := newFakeSystem()
	fake.entries["/plugins"] = []string{}
	cause := errors.New("local disk permission denied for the lift control")
	fake.readDirErr["/plugins"] = cause
	_, err := Discover(fakeConfig("/plugins"), fakeOwner(), fake)
	if err == nil {
		t.Fatal("Discover admitted an unreadable plugin directory")
	}
	var local Error
	if !errors.As(err, &local) {
		t.Fatalf("failure %v is not a provider Error", err)
	}
	if !strings.Contains(local.Error(), cause.Error()) {
		t.Fatalf("local rendering %q does not reproduce its cause; the control is misbuilt", local.Error())
	}
	if _, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    axerror.Code(local.Code()),
		Message: local.Error(),
		Details: axerror.Details{},
		Cause:   cause,
	}); !errors.Is(err, axerror.ErrCausalLeak) {
		t.Fatalf("naive message lift error = %v, want the causal-leak refusal", err)
	}
	if _, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    axerror.Code(local.Code()),
		Message: "provider discovery could not establish its filesystem preconditions",
		Details: axerror.Details{"failure": "wrapping " + cause.Error()},
		Cause:   cause,
	}); !errors.Is(err, axerror.ErrCausalLeak) {
		t.Fatalf("naive details lift error = %v, want the causal-leak refusal", err)
	}
}

// loadProviderCodes derives the closed refusal-code set from every
// production source file of the package rooted at dir, never from
// memory: a code added to production without a lift vector reddens the
// totality test below instead of shipping unwitnessed. Both const and
// var bindings count, in any file, grouped or alone, with or without an
// explicit type — the value must be a string literal in each case. Any
// code-prefixed declaration the scan cannot classify to a literal value
// (an identifier or call value, an iota-inherited empty value list, or
// a code-prefixed type declaration such as an alias) is an error, not a
// skip: an unclassifiable binding must fail the census loudly rather
// than pass the totality test silently. Test files never count.
func loadProviderCodes(dir string) (map[string]bool, error) {
	scannedFiles, fileSet, failures := invcore.ScanProduction(dir)
	if len(failures) != 0 {
		return nil, errors.New(strings.Join(failures, "; "))
	}
	codes := map[string]bool{}
	scanned := 0
	for _, production := range scannedFiles {
		name, syntax := production.Name, production.Syntax
		scanned++
		for _, decl := range syntax.Decls {
			general, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if general.Tok == token.TYPE {
				for _, spec := range general.Specs {
					if alias, ok := spec.(*ast.TypeSpec); ok && strings.HasPrefix(alias.Name.Name, "code") {
						position := fileSet.Position(alias.Pos())
						return nil, fmt.Errorf("%s:%d: code-prefixed type declaration %q has no classifiable literal value", name, position.Line, alias.Name.Name)
					}
				}
				continue
			}
			if general.Tok != token.CONST && general.Tok != token.VAR {
				continue
			}
			for _, spec := range general.Specs {
				values, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, ident := range values.Names {
					if !strings.HasPrefix(ident.Name, "code") {
						continue
					}
					if i >= len(values.Values) {
						// Grouped specs inherit the previous value
						// list when Values is empty; resolving that
						// needs constant evaluation, so fail closed.
						position := fileSet.Position(ident.Pos())
						return nil, fmt.Errorf("%s:%d: code binding %q carries no value expression", name, position.Line, ident.Name)
					}
					value := values.Values[i]
					if len(values.Values) == 1 && len(values.Names) > 1 {
						// `var a, b = f()` shares one value across
						// names; same constant-evaluation problem.
						position := fileSet.Position(ident.Pos())
						return nil, fmt.Errorf("%s:%d: code binding %q shares its value expression", name, position.Line, ident.Name)
					}
					literal, ok := value.(*ast.BasicLit)
					if !ok || literal.Kind != token.STRING {
						position := fileSet.Position(value.Pos())
						return nil, fmt.Errorf("%s:%d: code binding %q is not a string literal", name, position.Line, ident.Name)
					}
					codes[strings.Trim(literal.Value, `"`)] = true
				}
			}
		}
	}
	if scanned == 0 {
		return nil, fmt.Errorf("scanned no production sources in %s; the check is blind", dir)
	}
	return codes, nil
}

// derivedProviderCodes derives the closed refusal-code set from the
// production package, failing the test on any blind or unclassifiable
// scan.
func derivedProviderCodes(t *testing.T) map[string]bool {
	t.Helper()
	codes, err := loadProviderCodes(mustProviderDir(t))
	if err != nil {
		t.Fatalf("derive provider codes: %v", err)
	}
	if len(codes) == 0 {
		t.Fatal("derived zero provider codes; the scan is blind, not the package")
	}
	return codes
}

// writeFixtureFile stages one package source file under dir for the
// derivation control plants below. Fixtures live in a temporary
// directory, never in the production package.
func writeFixtureFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

// TestDerivedCodesIncludesVarBinding plants the first control the old
// single-file const-only scan missed: a var-bound code counts toward
// the closed set, so a production var code without a lift vector
// reddens the totality test instead of passing silently.
func TestDerivedCodesIncludesVarBinding(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "codes.go", "package provider\n\nconst codeHeld = \"held_code\"\n")
	writeFixtureFile(t, dir, "extra.go", "package provider\n\nvar codePlanted = \"planted_var_code\"\n")
	codes, err := loadProviderCodes(dir)
	if err != nil {
		t.Fatalf("loadProviderCodes: %v", err)
	}
	for _, want := range []string{"held_code", "planted_var_code"} {
		if !codes[want] {
			t.Fatalf("derived codes = %v, want %q", codes, want)
		}
	}
}

// TestDerivedCodesScansEveryPackageFile plants the second control the
// old scan missed: a const declared in another file of the same package
// counts. A scan reading only provider.go derives one code here and the
// assertion on the second reddens it.
func TestDerivedCodesScansEveryPackageFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "a.go", "package provider\n\nconst codeFirst = \"first_code\"\n")
	writeFixtureFile(t, dir, "b.go", "package provider\n\nconst codeSecond = \"second_file_code\"\n")
	codes, err := loadProviderCodes(dir)
	if err != nil {
		t.Fatalf("loadProviderCodes: %v", err)
	}
	for _, want := range []string{"first_code", "second_file_code"} {
		if !codes[want] {
			t.Fatalf("derived codes = %v, want %q", codes, want)
		}
	}
}

// TestDerivedCodesIncludesGroupedConsts pins the grouped form: every
// member of a parenthesized const block counts individually.
func TestDerivedCodesIncludesGroupedConsts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixtureFile(t, dir, "codes.go", "package provider\n\nconst (\n\tcodeGroupedA = \"grouped_a\"\n\tcodeGroupedB string = \"grouped_b\"\n)\n")
	codes, err := loadProviderCodes(dir)
	if err != nil {
		t.Fatalf("loadProviderCodes: %v", err)
	}
	for _, want := range []string{"grouped_a", "grouped_b"} {
		if !codes[want] {
			t.Fatalf("derived codes = %v, want %q", codes, want)
		}
	}
}

// TestDerivedCodesFailsClosedOnUnclassifiableBinding plants the
// fail-closed controls: a code-prefixed type alias and a code binding
// whose value is an identifier rather than a literal cannot be
// classified to a code value, so the derivation errors instead of
// silently omitting them from the denominator.
func TestDerivedCodesFailsClosedOnUnclassifiableBinding(t *testing.T) {
	t.Parallel()

	aliased := t.TempDir()
	writeFixtureFile(t, aliased, "codes.go", "package provider\n\ntype codeHandle = string\n\nconst codeHeld codeHandle = \"held_code\"\n")
	if codes, err := loadProviderCodes(aliased); err == nil {
		t.Fatalf("aliased-type derivation = %v, want a fail-closed error", codes)
	}

	indirect := t.TempDir()
	writeFixtureFile(t, indirect, "codes.go", "package provider\n\nconst codeBase = \"base_code\"\n\nconst codeAlias = codeBase\n")
	if codes, err := loadProviderCodes(indirect); err == nil {
		t.Fatalf("identifier-valued derivation = %v, want a fail-closed error", codes)
	}
}

// TestLiftCoversTheClosedCodeSet is the totality direction: every
// production-derived code has a real-failure lift vector above, and
// every vector names a derived code, so neither an unwitnessed code
// nor an orphan row survives.
func TestLiftCoversTheClosedCodeSet(t *testing.T) {
	t.Parallel()

	derived := derivedProviderCodes(t)
	covered := map[string]bool{}
	for _, vector := range liftVectors() {
		if !derived[vector.code] {
			t.Errorf("lift vector %q names underived code %q", vector.name, vector.code)
		}
		covered[vector.code] = true
	}
	for code := range derived {
		if !covered[code] {
			t.Errorf("production code %q has no lift vector; the lift is partial", code)
		}
	}
}
