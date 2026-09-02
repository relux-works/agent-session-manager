package config

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// installationDocumentWithExtensions builds a document whose only interesting
// content is one directory installation carrying the given inline extensions
// table, so every case below is decided by the production loadConfigDocument
// entry and its directory_installations[0].extensions gate.
func installationDocumentWithExtensions(inlineTable string) []byte {
	return append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[directory_installations]]
installation_id = "`+testDigest+`"
environment_id = "local"
provider_id = "codex"
adapter_id = "codex-local"
scan_root_authority_ids = ["`+testDigest+`"]
enabled = true
extensions = `+inlineTable+`
`)...)
}

func requireExtensionsAdmitted(t *testing.T, inlineTable string) Snapshot {
	t.Helper()
	snapshot, err := loadConfigDocument(installationDocumentWithExtensions(inlineTable), scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatalf("extensions %s refused: %v", inlineTable, err)
	}
	return snapshot
}

func requireExtensionsRefused(t *testing.T, inlineTable string) {
	t.Helper()
	_, err := loadConfigDocument(installationDocumentWithExtensions(inlineTable), scalar.PlatformMacOS, nil)
	requireConfigClause(t, err, "directory_installations[0].extensions")
}

// requireExtensionsRoundTripTo loads the inline table through the production
// entry and asserts the loaded extensions map re-encodes to exactly want.
// encoding/json sorts object keys, so want is a byte-identity assertion over
// the whole nested structure: a dropped key, an added key, a renamed key, a
// changed value or a changed scalar type all fail here. Admission alone cannot
// establish that, which is why this is separate from the admission tests.
func requireExtensionsRoundTripTo(t *testing.T, inlineTable, want string) {
	t.Helper()
	snapshot := requireExtensionsAdmitted(t, inlineTable)
	loaded, ok := snapshot.Configuration()
	if !ok {
		t.Fatalf("Snapshot.Configuration() absent for admitted extensions %s", inlineTable)
	}
	encoded, err := json.Marshal(loaded.Value.DirectoryInstallations[0].Extensions)
	if err != nil {
		t.Fatalf("re-encoding loaded extensions %s failed: %v", inlineTable, err)
	}
	if string(encoded) != want {
		t.Fatalf("extensions %s round-tripped as %s, want byte-identical %s", inlineTable, encoded, want)
	}
}

// TestExtensionKeyAdmissionIsDecidedByTheReverseDNSGrammarAlone pins the
// admission rule to the only key rule the pinned specification states.
//
// SPEC.md:345-347 (v0.5.0, SPEC.md SHA-256
// 562546d240f0fa3e71b47e6359a002f9892c0efd97e19eb55917527552ac484a):
//
//	"A reverse-DNS key is 3-253 lowercase ASCII characters, contains at least
//	one dot, and has dot-separated labels matching [a-z][a-z0-9-]{0,62}."
//
// The specification reserves no label, so a key is admitted or refused only by
// that grammar. A previous implementation refused any key whose dot- or
// dash-separated part was one of secret, token, password, credential, auth,
// environment, env or endpoint. That rule appears nowhere in the pinned
// specification and made reverse-DNS namespaces this organisation legitimately
// owns unusable. Narrowing the gate back to any name blacklist fails here.
func TestExtensionKeyAdmissionIsDecidedByTheReverseDNSGrammarAlone(t *testing.T) {
	t.Parallel()

	admitted := []string{
		// Every label the removed blacklist refused, in a key the grammar accepts.
		"works.relux.env-tools",
		"works.relux.env",
		"works.relux.environment",
		"com.example.auth-manager",
		"com.example.auth",
		"io.example.endpoint-list",
		"io.example.endpoint",
		"works.relux.secret-rotation-policy",
		"works.relux.secrets",
		"com.example.token-budget",
		"com.example.tokens",
		"net.example.password-policy-docs",
		"net.example.passwords",
		"org.example.credential-realm-name",
		"org.example.credentials",
		// The blacklist normalised dots too, so a whole namespace was unusable.
		"env.example.tool",
		"auth.example.tool",
		// Grammar edges that must stay admitted.
		"a.b",
		"a1.b-c",
		strings.Repeat("z", 63) + ".b",
	}
	for _, key := range admitted {
		key := key
		t.Run("admits "+key, func(t *testing.T) {
			t.Parallel()
			snapshot := requireExtensionsAdmitted(t, `{ "`+key+`" = true }`)
			loaded, ok := snapshot.Configuration()
			if !ok {
				t.Fatalf("Snapshot.Configuration() absent for admitted key %q", key)
			}
			if value, present := loaded.Value.DirectoryInstallations[0].Extensions[key]; !present || value != true {
				t.Fatalf("extension %q loaded as %#v, present=%v", key, value, present)
			}
		})
	}
}

// TestExtensionKeyRefusalStillEnforcesTheReverseDNSGrammar is the negative half
// of the admission rule above: each key here violates SPEC.md:345-347 and must
// be refused through the production entry. Widening the gate to accept any of
// them fails this test.
func TestExtensionKeyRefusalStillEnforcesTheReverseDNSGrammar(t *testing.T) {
	t.Parallel()

	refused := map[string]string{
		"single label has no dot":                 "singlelabel",
		"uppercase is not lowercase":              "works.Relux.fixture",
		"underscore is not in the label alphabet": "works.relux_fixture",
		"label may not start with a digit":        "works.1relux.fixture",
		"label may not start with a dash":         "works.-relux.fixture",
		"empty trailing label":                    "works.relux.",
		"empty leading label":                     ".relux.fixture",
		"non-ASCII is outside the alphabet":       "works.relüx.fixture",
		"label longer than 63 characters":         "works." + strings.Repeat("b", 64),
	}
	for name, key := range refused {
		name, key := name, key
		t.Run("refuses "+name, func(t *testing.T) {
			t.Parallel()
			requireExtensionsRefused(t, `{ "`+key+`" = true }`)
		})
	}

	t.Run("refuses a key longer than 253 bytes", func(t *testing.T) {
		t.Parallel()
		// extensionKey(61) is 253 bytes and admitted; one byte more is refused.
		requireExtensionsAdmitted(t, `{ "`+extensionKey(61)+`" = true }`)
		if len(extensionKey(61)) != 253 {
			t.Fatalf("fixture key length = %d, want the SPEC bound 253", len(extensionKey(61)))
		}
		requireExtensionsRefused(t, `{ "`+extensionKey(62)+`" = true }`)
	})
}

// TestExtensionValueObjectKeysAreAdmittedAsData pins that the pinned
// specification imposes no naming rule inside an extension value.
//
// SPEC.md:347-349: "ExtensionValue is JSON null, boolean, a common-model
// integer, string, array, or string-keyed object with maximum nesting depth 4".
//
// "string-keyed" is the whole constraint on a nested key. The removed
// implementation additionally refused nested keys named secret, token,
// password, credential, auth, environment, env or endpoint at any depth, which
// the specification never states. Only the depth bound is a real gate here, and
// the second half of this test proves it still holds.
func TestExtensionValueObjectKeysAreAdmittedAsData(t *testing.T) {
	t.Parallel()

	admitted := []string{
		`{ "works.relux.fixture" = { endpoint = "https://example.invalid" } }`,
		`{ "works.relux.fixture" = { token = "fixture" } }`,
		`{ "works.relux.fixture" = { password = "fixture" } }`,
		`{ "works.relux.fixture" = { credential = "fixture" } }`,
		`{ "works.relux.fixture" = { auth = "fixture" } }`,
		`{ "works.relux.fixture" = { env = "fixture" } }`,
		`{ "works.relux.fixture" = { environment = "fixture" } }`,
		`{ "works.relux.fixture" = { secret = "fixture" } }`,
		// Nested at every depth the specification still allows.
		`{ "works.relux.fixture" = { a = { endpoint = "fixture" } } }`,
		`{ "works.relux.fixture" = { a = { b = { token = "fixture" } } } }`,
		// A key shape that is not reverse-DNS at all is still a legal nested key.
		`{ "works.relux.fixture" = { "Mixed Case Label" = "fixture" } }`,
	}
	for _, table := range admitted {
		table := table
		t.Run("admits "+table, func(t *testing.T) {
			t.Parallel()
			requireExtensionsAdmitted(t, table)
		})
	}

	t.Run("still refuses object nesting past depth 4", func(t *testing.T) {
		t.Parallel()
		requireExtensionsAdmitted(t, `{ "works.relux.fixture" = { a = { b = { c = { d = "leaf" } } } } }`)
		requireExtensionsRefused(t, `{ "works.relux.fixture" = { a = { b = { c = { d = { e = "leaf" } } } } } }`)
	})
}

// TestExtensionValueObjectKeysArePreservedAsData pins the preservation half of
// the claim the admission tests above cannot make.
//
// README and this change's commit message both state that a nested key named
// endpoint or token is "preserved as data". Admission does not establish that:
// a mutant that admits the document and then silently deletes those nested keys
// from the loaded map passes every admission assertion, because nothing there
// reads the value back. That is a claim with nothing pinning it.
//
// Each case below therefore drives the production loadConfigDocument entry and
// asserts the loaded extensions re-encode byte-identically, so a dropped,
// renamed or rewritten nested key fails. The values are distinct per label so a
// mutant that drops one key cannot be masked by another that survives.
func TestExtensionValueObjectKeysArePreservedAsData(t *testing.T) {
	t.Parallel()

	// Every label the removed blacklist refused inside an extension value.
	for _, label := range []string{
		"secret", "token", "password", "credential",
		"auth", "env", "environment", "endpoint",
	} {
		label := label
		t.Run("preserves nested "+label, func(t *testing.T) {
			t.Parallel()
			value := "value-of-" + label
			requireExtensionsRoundTripTo(t,
				`{ "works.relux.fixture" = { `+label+` = "`+value+`" } }`,
				`{"works.relux.fixture":{"`+label+`":"`+value+`"}}`)
		})
	}

	t.Run("preserves a blacklisted label at every admitted depth", func(t *testing.T) {
		t.Parallel()
		requireExtensionsRoundTripTo(t,
			`{ "works.relux.fixture" = { endpoint = "d1", a = { token = "d2", b = { password = "d3", c = { credential = "d4" } } } } }`,
			`{"works.relux.fixture":{"a":{"b":{"c":{"credential":"d4"},"password":"d3"},"token":"d2"},"endpoint":"d1"}}`)
	})

	t.Run("preserves non-string nested values carrying blacklisted labels", func(t *testing.T) {
		t.Parallel()
		requireExtensionsRoundTripTo(t,
			`{ "works.relux.fixture" = { token = 7, secret = true, auth = ["endpoint", "token"], env = { environment = "leaf" } } }`,
			`{"works.relux.fixture":{"auth":["endpoint","token"],"env":{"environment":"leaf"},"secret":true,"token":7}}`)
	})

	t.Run("preserves a whole extension key that carries a blacklisted label", func(t *testing.T) {
		t.Parallel()
		// The root key and the nested keys are both previously refused names,
		// so this one case fails if either arm of the removed rule returns.
		requireExtensionsRoundTripTo(t,
			`{ "works.relux.env-tools" = { endpoint = "https://example.invalid", credential = "profile-name" } }`,
			`{"works.relux.env-tools":{"credential":"profile-name","endpoint":"https://example.invalid"}}`)
	})
}

// TestExtensionAdmissionDoesNotClaimSecretDetection records the fact that
// removing the name blacklist removes no secret gate, because the blacklist was
// never one: it inspected zero values.
//
// The only "secret, token, endpoint credential" clause in the pinned
// specification is SPEC.md:2596-2597, and it is scoped to a terminal
// backend-config settings object, not to extension keys:
//
//	"An arbitrary blob, raw command/argv, secret, token, endpoint credential,
//	unrestricted environment, or environment passthrough is forbidden."
//
// That clause is enforced where it applies, by the closed registered settings
// schema at validation.go validateTerminal, which refuses any settings object a
// backend implementation version did not register. This test pins both halves:
// an extension value carrying credential-shaped text loads as opaque data, and
// an unregistered backend settings object is still refused.
func TestExtensionAdmissionDoesNotClaimSecretDetection(t *testing.T) {
	t.Parallel()

	t.Run("extension values are opaque data, not scanned", func(t *testing.T) {
		t.Parallel()
		// The removed blacklist admitted exactly this key while refusing
		// works.relux.env-tools, which is why it was never secret detection.
		requireExtensionsAdmitted(t, `{ "com.example.deploy" = "AKIAIOSFODNN7EXAMPLE" }`)
	})

	t.Run("SPEC.md:2562-2563 stays enforced by the closed v2 table shape", func(t *testing.T) {
		t.Parallel()
		// The §6.4 clause an earlier review cited when it demanded coverage for
		// the removed name blacklist reads, verbatim:
		//
		//	"No v2 table accepts a secret, endpoint credential, model token,
		//	auth root, or arbitrary environment passthrough."
		//
		// It governs which fields a v2 table declares, not how an extension key
		// is spelled, and SPEC.md:2344-2345 makes the same distinction for
		// values while explicitly permitting a field to *name* a credential
		// profile: "Secret values MUST NOT be accepted in config fields; a
		// provider MAY name a machine-local environment variable or credential
		// profile." Both are enforced structurally, by the closed table shape
		// schema.go decodes with DisallowUnknownFields: an undeclared member of
		// directory_installations cannot carry a credential at all.
		document := append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[directory_installations]]
installation_id = "`+testDigest+`"
environment_id = "local"
provider_id = "codex"
adapter_id = "codex-local"
scan_root_authority_ids = ["`+testDigest+`"]
enabled = true
extensions = {}
api_token = "AKIAIOSFODNN7EXAMPLE"
`)...)
		// The same document without api_token loads, so the closed shape is the
		// only thing left to refuse it and the refusal cannot be attributed to
		// an unrelated missing member.
		requireExtensionsAdmitted(t, `{}`)
		_, err := loadConfigDocument(document, scalar.PlatformMacOS, nil)
		if err == nil {
			t.Fatal("undeclared directory_installations member api_token loaded, want refusal")
		}
		if !errors.Is(err, ErrConfigDecode) {
			t.Fatalf("api_token refused with %v, want the closed-shape ErrConfigDecode clause", err)
		}
	})

	t.Run("SPEC.md:2596-2597 stays enforced by the registered settings schema", func(t *testing.T) {
		t.Parallel()
		configuration := validCurrentConfiguration()
		configuration.Terminal.BackendConfig = []BackendConfig{{
			BackendID: "ax.tmux", ConfigVersion: "1.0.0",
			Settings: map[string]any{"endpoint_credential": "AKIAIOSFODNN7EXAMPLE"},
		}}
		_, err := EncodeCurrent(configuration, DecodeContext{
			RuntimePlatform: scalar.PlatformMacOS, BackendSettings: rejectBackendSettings{},
		})
		requireConfigClause(t, err, "terminal.backend_config[0].settings")
	})
}
