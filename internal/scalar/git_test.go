package scalar

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGitOIDUsesExactObjectFormatGrammar(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"sha1:" + strings.Repeat("a", 40), "sha256:" + strings.Repeat("b", 64)} {
		parsed, err := ParseGitOID(value)
		if err != nil || parsed.String() != value {
			t.Fatalf("ParseGitOID(%q) = %#v, %v", value, parsed, err)
		}
		encoded, err := json.Marshal(parsed)
		if err != nil {
			t.Fatal(err)
		}
		var decoded GitOID
		if err := json.Unmarshal(encoded, &decoded); err != nil || decoded.String() != value {
			t.Fatalf("GitOID round trip = %q, %v; want %q", decoded, err, value)
		}
	}
	for _, value := range []string{"sha1:" + strings.Repeat("A", 40), "sha1:abc", "sha256:" + strings.Repeat("g", 64)} {
		if _, err := ParseGitOID(value); err == nil {
			t.Fatalf("ParseGitOID(%q) accepted malformed OID", value)
		}
	}
	if _, err := ParseGitOIDForObjectFormat("sha1:"+strings.Repeat("a", 40), "sha256"); err == nil {
		t.Fatal("ParseGitOIDForObjectFormat accepted mismatched prefix")
	}
}

func TestGitRefMatchesFullyQualifiedCheckRefFormatSubset(t *testing.T) {
	t.Parallel()
	for _, value := range []string{"refs/heads/main", "refs/remotes/origin/feature/данные"} {
		if parsed, err := ParseGitRef(value); err != nil || parsed.String() != value {
			t.Fatalf("ParseGitRef(%q) = %#v, %v", value, parsed, err)
		}
	}
	for _, value := range []string{"HEAD", "main", "refs/heads/.hidden", "refs/heads/a..b", "refs/heads/a.lock", "refs/heads/a b", "refs/heads/a@{b"} {
		if _, err := ParseGitRef(value); err == nil {
			t.Fatalf("ParseGitRef(%q) accepted malformed ref", value)
		}
	}
}

func TestSanitizedGitURLRefusesCredentialsAndLocalSchemes(t *testing.T) {
	t.Parallel()
	for _, value := range []string{
		"https://github.com/relux/repo.git",
		"https://github.com",
		"ssh://git@github.com/relux/repo.git",
		"ssh://git@github.com",
		"git://github.com/relux/repo.git",
		"git@github.com:relux/repo.git",
	} {
		if parsed, err := ParseSanitizedGitURL(value); err != nil || parsed.String() != value {
			t.Fatalf("ParseSanitizedGitURL(%q) = %#v, %v", value, parsed, err)
		}
	}
	for _, value := range []string{
		"https://token@github.com/relux/repo.git",
		"ssh://git:secret@github.com/relux/repo.git",
		"https://github.com/relux/repo.git?token=x",
		"https://github.com/relux/repo.git#fragment",
		"file:///tmp/repo",
		"/tmp/repo",
		`C:\repo`,
	} {
		if _, err := ParseSanitizedGitURL(value); err == nil {
			t.Fatalf("ParseSanitizedGitURL(%q) accepted unsafe URL", value)
		}
	}
}
