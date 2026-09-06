package secprim

import (
	"strings"
	"testing"
)

func TestRedactCorpusValues(t *testing.T) {
	t.Parallel()
	line := "failed with hunter2-hunter for provider pi"
	if got := Redact(line, []string{"hunter2-hunter"}); got != "failed with [redacted] for provider pi" {
		t.Fatalf("Redact = %q", got)
	}
	// Short values are skipped: matching bigrams would redact ordinary
	// text. The bound is minSecretRunes (8).
	if got := Redact("abcdefg stays", []string{"abcdefg"}); got != "abcdefg stays" {
		t.Fatalf("short secret redacted: %q", got)
	}
	if got := Redact("abcdefgh goes", []string{"abcdefgh"}); got != "[redacted] goes" {
		t.Fatalf("boundary secret kept: %q", got)
	}
}

func TestRedactPrivateKeyBlocks(t *testing.T) {
	t.Parallel()
	block := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaAo=\n-----END OPENSSH PRIVATE KEY-----"
	if got := Redact("key "+block+" here", nil); got != "key [redacted private key block] here" {
		t.Fatalf("Redact = %q", got)
	}
	rsa := "-----BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----"
	if got := Redact(rsa, nil); got != "[redacted private key block]" {
		t.Fatalf("Redact = %q", got)
	}
	// Certificates are routinely logged for pinning and keep passing.
	cert := "-----BEGIN CERTIFICATE-----\nabc\n-----END CERTIFICATE-----"
	if got := Redact(cert, nil); got != cert {
		t.Fatalf("certificate redacted: %q", got)
	}
	// An unterminated block fails closed to end of input.
	if got := Redact("key -----BEGIN EC PRIVATE KEY-----\nabc", nil); got != "key [redacted private key block]" {
		t.Fatalf("Redact = %q", got)
	}
}

func TestRedactSensitivePairs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"equals", "password=hunter2-hunter", "password=[redacted]"},
		{"colon", "Authorization: Bearer abc12345", "Authorization: [redacted]"},
		{"compact colon", "token:abc12345", "token:[redacted]"},
		{"spaced equals", "api_key = hunter2-hunter", "api_key = [redacted]"},
		{"json pair", `{"password": "hunter2-hunter"}, other`, `{"password": "[redacted]"}, other`},
		{"single quoted", "token='abc12345', next", "token='[redacted]', next"},
		{"json spaced", `{"password": "hunter2-hunter"}`, `{"password": "[redacted]"}`},
		{"case folded", "PASSWORD=hunter2-hunter", "PASSWORD=[redacted]"},
		{"trailing comma", "secret=hunter2-hunter, ok", "secret=[redacted], ok"},
		{"innocuous key kept", "token_count=5", "token_count=5"},
		{"bare key kept", "key=sortable-id", "key=sortable-id"},
		{"bare auth kept", "auth=true", "auth=true"},
		{"pwd kept", "PWD=/home/ax", "PWD=/home/ax"},
		{"compound key unlisted", "db_password=hunter2", "db_password=hunter2"},
		{"empty value kept", "password=", "password="},
	}
	for _, kase := range cases {
		t.Run(kase.name, func(t *testing.T) {
			t.Parallel()
			if got := Redact(kase.input, nil); got != kase.want {
				t.Fatalf("Redact(%q) = %q, want %q", kase.input, got, kase.want)
			}
		})
	}
}

func TestRedactURLUserinfo(t *testing.T) {
	t.Parallel()
	if got := Redact("fetch https://user:hunter2@example.com/x", nil); got != "fetch https://[redacted]@example.com/x" {
		t.Fatalf("Redact = %q", got)
	}
	if got := Redact("plain https://example.com/x", nil); got != "plain https://example.com/x" {
		t.Fatalf("bare URL rewritten: %q", got)
	}
}

func TestRedactIsIdempotent(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"password=hunter2-hunter",
		"-----BEGIN EC PRIVATE KEY-----\nabc\n-----END EC PRIVATE KEY-----",
		"https://user:hunter2@example.com/x",
		"plain text",
		"[redacted] and [redacted private key block]",
	}
	secrets := []string{"hunter2-hunter"}
	for _, input := range inputs {
		once := Redact(input, secrets)
		if twice := Redact(once, secrets); twice != once {
			t.Errorf("Redact is not idempotent on %q: %q then %q", input, once, twice)
		}
	}
}

// TestSensitiveKeyListIsReviewed pins the exact scrubber list: every
// entry is asserted individually, so an added key reddens here until its
// row is reviewed, and a removed key reddens until its absence is
// justified. The list is intentionally not shared with the axerror
// detail-key gate (free-text k=v shapes versus JSON detail keys), and
// the divergence is documented on Redact.
func TestSensitiveKeyListIsReviewed(t *testing.T) {
	t.Parallel()
	want := []string{
		"access_token", "api_key", "apikey",
		"auth_json", "authorization", "auth_token",
		"bearer_token", "client_secret", "cookie",
		"cookies", "credential", "credentials",
		"dotenv", "env_secret", "environment_secret",
		"oauth_token", "passphrase", "password",
		"passwd", "private_key", "refresh_token",
		"secret", "secrets", "session_token",
		"ssh_private_key", "subscription_token", "token",
	}
	if len(sensitiveKeyNames) != len(want) {
		t.Fatalf("sensitive keys = %d, want %d reviewed entries", len(sensitiveKeyNames), len(want))
	}
	for _, key := range want {
		if !sensitiveKeyNames[key] {
			t.Errorf("reviewed key %q missing from the scrubber list", key)
		}
	}
	for key := range sensitiveKeyNames {
		found := false
		for _, reviewed := range want {
			if key == reviewed {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unreviewed key %q in the scrubber list", key)
		}
	}
	// The deliberate absences are load-bearing: bare "key", "auth", and
	// "pwd" must stay out, or ordinary diagnostics get scrubbed.
	for _, absent := range []string{"key", "auth", "pwd"} {
		if sensitiveKeyNames[absent] {
			t.Errorf("generic key %q must not join the scrubber list", absent)
		}
		if got := Redact(absent+"=ordinary-value", nil); strings.Contains(got, "[redacted]") {
			t.Errorf("Redact(%q=...) scrubbed an ordinary diagnostic: %q", absent, got)
		}
	}
}
