package config

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// peerDocumentWithSSHArgs builds a document whose only interesting content is
// one mesh peer carrying the given argv, so every case below is decided by the
// production loadConfigDocument entry and its mesh.peers[0].ssh_args gate.
func peerDocumentWithSSHArgs(arguments ...string) []byte {
	quoted := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		quoted = append(quoted, strconv.Quote(argument))
	}
	return append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[mesh.peers]]
host_id = "`+testPeerID+`"
name = "peer"
endpoint = "peer.example"
platform = "linux"
ssh_args = [`+strings.Join(quoted, ", ")+`]
workspace_roots = []
`)...)
}

func requireSSHArgsRefusal(t *testing.T, reason string, arguments ...string) {
	t.Helper()
	_, err := loadConfigDocument(peerDocumentWithSSHArgs(arguments...), scalar.PlatformMacOS, nil)
	requireConfigClause(t, err, "mesh.peers[0].ssh_args "+reason)
}

func requireSSHArgsAdmitted(t *testing.T, arguments ...string) {
	t.Helper()
	if _, err := loadConfigDocument(peerDocumentWithSSHArgs(arguments...), scalar.PlatformMacOS, nil); err != nil {
		t.Fatalf("ssh_args %q refused: %v", arguments, err)
	}
}

// TestSSHShortOptionTablesAreDerivedFromTheOpenSSHGrammar pins the arity of
// every admitted short option to the transcribed ssh(1) usage grammar. A
// letter admitted with the wrong arity would make the grouped-flag walk
// diverge from OpenSSH, which is exactly how -vo StrictHostKeyChecking=no
// slipped past the previous name-matching gate.
func TestSSHShortOptionTablesAreDerivedFromTheOpenSSHGrammar(t *testing.T) {
	t.Parallel()

	seen := map[byte]string{}
	for _, letter := range []byte(sshShortOptionsWithoutValue) {
		if where, duplicate := seen[letter]; duplicate {
			t.Fatalf("letter %q declared twice in the ssh(1) grammar (%s)", letter, where)
		}
		seen[letter] = "valueless"
	}
	for _, letter := range []byte(sshShortOptionsWithValue) {
		if where, duplicate := seen[letter]; duplicate {
			t.Fatalf("letter %q declared twice in the ssh(1) grammar (%s)", letter, where)
		}
		seen[letter] = "value-taking"
	}
	for letter := range sshPermittedFlags {
		if seen[letter] != "valueless" {
			t.Fatalf("permitted flag -%c has arity %q in the ssh(1) grammar, want valueless", letter, seen[letter])
		}
	}
	for letter := range sshPermittedValueFlags {
		if seen[letter] != "value-taking" {
			t.Fatalf("permitted value flag -%c has arity %q in the ssh(1) grammar, want value-taking", letter, seen[letter])
		}
		if _, both := sshPermittedFlags[letter]; both {
			t.Fatalf("letter -%c is declared in both permitted tables", letter)
		}
	}
}

// TestLoadRefusesEveryHostAuthenticationOptionDeclaredInTheRegistry derives
// its cases from sshOptionRegistry rather than listing spellings, so a newly
// declared host-authentication option is covered the moment it is added.
func TestLoadRefusesEveryHostAuthenticationOptionDeclaredInTheRegistry(t *testing.T) {
	t.Parallel()

	// Candidate values a caller would reach for to relax host authentication.
	// The first one the declared rule does not permit is the case value.
	candidates := []string{"no", "off", "false", "accept-new", "ask", "/dev/null", "none", "", "/tmp/attacker-writable"}
	names := make([]string, 0, len(sshOptionRegistry))
	for name, rule := range sshOptionRegistry {
		if rule.hostAuthentication {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatal("sshOptionRegistry declares no host-authentication option")
	}
	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rule := sshOptionRegistry[name]
			bypass, found := "", false
			for _, candidate := range candidates {
				if rule.permits == nil || !rule.permits(candidate) {
					bypass, found = candidate, true
					break
				}
			}
			if !found {
				t.Fatalf("option %q permits every bypass candidate", name)
			}
			requireSSHArgsRefusal(t, sshRefusalHostAuthentication, "-o", name+"="+bypass)
			requireSSHArgsRefusal(t, sshRefusalHostAuthentication, "-o"+name+"="+bypass)
		})
	}
}

// permittedSSHOptionSamples carries, for every option the registry permits, a
// value the rule admits and a value it refuses. The key set is asserted to
// equal the permitted registry exactly, so widening the registry without
// proving both directions of the new rule reddens this test.
var permittedSSHOptionSamples = map[string]struct{ admitted, refused string }{
	"addressfamily":       {admitted: "inet", refused: "unix"},
	"batchmode":           {admitted: "yes", refused: "maybe"},
	"compression":         {admitted: "no", refused: "delayed"},
	"connectionattempts":  {admitted: "3", refused: "0"},
	"connecttimeout":      {admitted: "30", refused: "301"},
	"identitiesonly":      {admitted: "yes", refused: "1"},
	"identityfile":        {admitted: "/home/ax/.ssh/id_ed25519", refused: "/home/ax/.ssh/id ed25519"},
	"loglevel":            {admitted: "ERROR", refused: "trace"},
	"port":                {admitted: "2222", refused: "65536"},
	"serveralivecountmax": {admitted: "3", refused: "101"},
	"serveraliveinterval": {admitted: "30", refused: "86401"},
	"tcpkeepalive":        {admitted: "yes", refused: "always"},
	"user":                {admitted: "ax", refused: "ax user"},
}

// TestLoadAdmitsExactlyTheDeclaredPermittedSSHOptions proves admission is
// closed in both directions: every permitted option loads with a value its
// declared rule admits, and refuses a value the rule does not.
func TestLoadAdmitsExactlyTheDeclaredPermittedSSHOptions(t *testing.T) {
	t.Parallel()

	permitted := map[string]struct{}{}
	for name, rule := range sshOptionRegistry {
		if rule.permits != nil && !rule.hostAuthentication {
			permitted[name] = struct{}{}
		}
	}
	for name := range permitted {
		if _, ok := permittedSSHOptionSamples[name]; !ok {
			t.Fatalf("permitted option %q has no admitted/refused sample", name)
		}
	}
	for name := range permittedSSHOptionSamples {
		if _, ok := permitted[name]; !ok {
			t.Fatalf("sample %q is not a permitted option in sshOptionRegistry", name)
		}
	}
	names := make([]string, 0, len(permittedSSHOptionSamples))
	for name := range permittedSSHOptionSamples {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sample := permittedSSHOptionSamples[name]
			requireSSHArgsAdmitted(t, "-o", name+"="+sample.admitted)
			requireSSHArgsRefusal(t, sshRefusalUnpermittedOptionValue, "-o", name+"="+sample.refused)
		})
	}
}

// TestLoadRefusesEverySSHShortOptionOutsideThePermittedTables walks the whole
// declared ssh(1) short-option grammar and requires every letter AX does not
// declare as permitted to be refused at the production entry. -F, the config
// file redirection cited in the report, is one derived case among them.
func TestLoadRefusesEverySSHShortOptionOutsideThePermittedTables(t *testing.T) {
	t.Parallel()

	for _, letter := range []byte(sshShortOptionsWithoutValue + sshShortOptionsWithValue) {
		if _, permitted := sshPermittedFlags[letter]; permitted {
			continue
		}
		if _, permitted := sshPermittedValueFlags[letter]; permitted {
			continue
		}
		letter := letter
		t.Run(string(letter), func(t *testing.T) {
			t.Parallel()
			requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-"+string(letter), "value")
			requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-q"+string(letter), "value")
		})
	}
	// The cited -F bypass in both spellings, named explicitly rather than only
	// derived, because an attacker-writable ssh_config redirects every option
	// this gate protects.
	requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-F", "/tmp/attacker/ssh_config")
	requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-F/tmp/attacker/ssh_config")
	// A letter ssh(1) itself does not define is refused rather than skipped.
	requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-z")
}

// TestLoadParsesGroupedShortFlagsTheWayOpenSSHParsesThem covers the grouping
// bypasses reproduced against OpenSSH 10.2p1: ssh -G -vo
// StrictHostKeyChecking=no prints "stricthostkeychecking false", and ssh -G
// -4o UserKnownHostsFile=/dev/null prints "userknownhostsfile /dev/null".
func TestLoadParsesGroupedShortFlagsTheWayOpenSSHParsesThem(t *testing.T) {
	t.Parallel()

	t.Run("grouped valueless flag still lets -o take the next argument", func(t *testing.T) {
		t.Parallel()
		requireSSHArgsRefusal(t, sshRefusalHostAuthentication, "-vo", "StrictHostKeyChecking=no")
		requireSSHArgsRefusal(t, sshRefusalHostAuthentication, "-4o", "UserKnownHostsFile=/dev/null")
		requireSSHArgsRefusal(t, sshRefusalUnpermittedOption, "-6o", "ProxyCommand=/bin/sh")
	})
	t.Run("grouped valueless flag still lets -o take the attached value", func(t *testing.T) {
		t.Parallel()
		requireSSHArgsRefusal(t, sshRefusalHostAuthentication, "-voStrictHostKeyChecking=no")
		requireSSHArgsRefusal(t, sshRefusalHostAuthentication, "-qvoUserKnownHostsFile=/dev/null")
	})
	t.Run("a value-taking letter ends its group", func(t *testing.T) {
		t.Parallel()
		// -qp2222 is q plus p=2222, matching ssh -G -qp2222 (port 2222). The
		// digits are the port value, not further flag letters.
		requireSSHArgsAdmitted(t, "-qp2222")
		requireSSHArgsAdmitted(t, "-4v")
		requireSSHArgsAdmitted(t, "-vqTCa")
		requireSSHArgsRefusal(t, sshRefusalUnpermittedFlagValue, "-qp65536")
		// -i consumes the whole next argument as its identity file exactly as
		// OpenSSH does: ssh -G -i -oProxyCommand=id host sets no proxycommand.
		requireSSHArgsAdmitted(t, "-i", "-oProxyCommand=id")
	})
	t.Run("an unpermitted letter inside a group is refused", func(t *testing.T) {
		t.Parallel()
		requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-vF", "/tmp/attacker/ssh_config")
		requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "-4J", "jump.example")
	})
}

// TestLoadRefusesUndeclaredSSHOptionNamesAtTheProductionEntry covers the
// option names the previous three-name filter did not know at all. None of
// them is listed in production: they are refused because the registry does not
// declare them.
func TestLoadRefusesUndeclaredSSHOptionNamesAtTheProductionEntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		reason    string
		arguments []string
	}{
		{name: "ProxyCommand", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "ProxyCommand=/bin/sh -c id"}},
		{name: "ProxyCommand combined", reason: sshRefusalUnpermittedOption, arguments: []string{"-oProxyCommand=/bin/sh"}},
		{name: "ProxyUseFdpass", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "ProxyUseFdpass=yes"}},
		{name: "Include", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "Include=/tmp/attacker/ssh_config"}},
		{name: "PermitLocalCommand and LocalCommand", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "PermitLocalCommand=yes", "-o", "LocalCommand=id"}},
		{name: "LocalCommand alone", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "LocalCommand=id"}},
		{name: "KnownHostsCommand", reason: sshRefusalHostAuthentication, arguments: []string{"-o", "KnownHostsCommand=/bin/true"}},
		{name: "Match exec", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "Match=exec /bin/true"}},
		{name: "unknown to this parser", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "ThisOptionDoesNotExistYet=1"}},
		{name: "option name only", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "ProxyCommand"}},
		{name: "separator only", reason: sshRefusalUnpermittedOption, arguments: []string{"-o", "="}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			requireSSHArgsRefusal(t, test.reason, test.arguments...)
		})
	}
}

// TestLoadRefusesSSHArgumentsThatAreNotOptions keeps ssh_args to options only.
// A bare word is ssh(1)'s destination or remote command, which would run on
// the peer.
func TestLoadRefusesSSHArgumentsThatAreNotOptions(t *testing.T) {
	t.Parallel()

	requireSSHArgsRefusal(t, sshRefusalUnpermittedArgument, "peer.example")
	requireSSHArgsRefusal(t, sshRefusalUnpermittedArgument, "-q", "id")
	requireSSHArgsRefusal(t, sshRefusalUnpermittedArgument, "--")
	requireSSHArgsRefusal(t, sshRefusalUnpermittedArgument, "-")
	// A GNU-style long option is not ssh(1) grammar; getopt reads its leading
	// dash as an undefined option letter, and so does this walk.
	requireSSHArgsRefusal(t, sshRefusalUnpermittedFlag, "--proxy-command=id")
	requireSSHArgsRefusal(t, sshRefusalMissingFlagValue, "-q", "-o")
	requireSSHArgsRefusal(t, sshRefusalMissingFlagValue, "-p")
}

// TestLoadAdmitsTheSpecifiedPeerArgumentExample keeps the Section 6.3 example
// argv loadable, so the closed admission above is not vacuously strict.
func TestLoadAdmitsTheSpecifiedPeerArgumentExample(t *testing.T) {
	t.Parallel()

	requireSSHArgsAdmitted(t, "-o", "BatchMode=yes")
	requireSSHArgsAdmitted(t)
	requireSSHArgsAdmitted(t, "-4", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-p", "2222", "-i", "/home/ax/.ssh/id_ed25519")
	requireSSHArgsAdmitted(t, "-o", "StrictHostKeyChecking=yes")
}

// TestSSHArgumentAdmissionIsTheOnlyGateBetweenArgvAndTheLoader guards the
// production wiring: the refusal clause must come from validateMesh's peer
// prefix, not from an unrelated bound.
func TestSSHArgumentAdmissionRefusesThroughTheMeshPeerClause(t *testing.T) {
	t.Parallel()

	_, err := loadConfigDocument(peerDocumentWithSSHArgs("-o", "ProxyCommand=id"), scalar.PlatformMacOS, nil)
	if !errors.Is(err, ErrConfigValidation) {
		t.Fatalf("error = %v, want ErrConfigValidation", err)
	}
	var documentError *DocumentError
	if !errors.As(err, &documentError) {
		t.Fatalf("error = %T, want *DocumentError", err)
	}
	if !strings.HasPrefix(documentError.Clause, "mesh.peers[0].ssh_args ") {
		t.Fatalf("clause = %q, want the mesh peer ssh_args prefix", documentError.Clause)
	}
}
