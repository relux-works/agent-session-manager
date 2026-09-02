package config

import (
	"strconv"
	"strings"
	"unicode"
)

// Admission for the argv AX hands to ssh(1) for a mesh peer.
//
// SPEC.md Section 6.3 requires refusing StrictHostKeyChecking=no, an empty
// UserKnownHostsFile, "or an equivalent host-authentication bypass". That
// equivalence class has no enumeration: OpenSSH keeps adding option names, a
// name can be spelled through a grouped short flag, and options such as
// ProxyCommand replace the authenticated transport outright. Admission is
// therefore derived the other way round. The tables below declare the exact
// argv AX passes to ssh(1); every argument outside that declaration is
// refused, so an option this package has never heard of fails closed instead
// of being admitted.

// sshShortOptionsWithoutValue and sshShortOptionsWithValue transcribe the
// ssh(1) short-option arity from the OpenSSH 10.2p1 usage text:
//
//	usage: ssh [-46AaCfGgKkMNnqsTtVvXxYy] [-B bind_interface] [-b bind_address]
//	           [-c cipher_spec] [-D [bind_address:]port] [-E log_file]
//	           [-e escape_char] [-F configfile] [-I pkcs11] [-i identity_file]
//	           [-J destination] [-L address] [-l login_name] [-m mac_spec]
//	           [-O ctl_cmd] [-o option] [-P tag] [-p port] [-R address]
//	           [-S ctl_path] [-W host:port] [-w local_tun[:remote_tun]]
//	           destination [command [argument ...]]
//	       ssh [-Q query_option]
//
// Arity is what makes grouped short flags parse the way OpenSSH's getopt
// parses them: in "-vo StrictHostKeyChecking=no" the v carries no value, so o
// still consumes the following argument, and OpenSSH resolves that argv to
// stricthostkeychecking false.
const (
	sshShortOptionsWithoutValue = "46AaCfGgKkMNnqsTtVvXxYy"
	sshShortOptionsWithValue    = "BbcDEeFIiJLlmOoPpQRSWw"
)

// Refusal reasons. They are a closed vocabulary appended to the ssh_args
// clause, never derived from the rejected argument itself.
const (
	sshArgumentAdmitted              = ""
	sshRefusalHostAuthentication     = "host authentication bypass"
	sshRefusalUnpermittedArgument    = "unpermitted argument"
	sshRefusalUnpermittedFlag        = "unpermitted flag"
	sshRefusalUnpermittedFlagValue   = "unpermitted flag value"
	sshRefusalMissingFlagValue       = "flag without its value"
	sshRefusalUnpermittedOption      = "unpermitted option"
	sshRefusalUnpermittedOptionValue = "unpermitted option value"
)

// sshValueRule reports the refusal reason for a value-taking short option, or
// sshArgumentAdmitted when the value is admitted.
type sshValueRule func(string) string

// sshPermittedFlags declares the valueless ssh(1) short options AX admits.
// Every other letter is refused, including letters OpenSSH itself accepts.
var sshPermittedFlags = map[byte]struct{}{
	'4': {}, // IPv4 only
	'6': {}, // IPv6 only
	'C': {}, // request compression
	'T': {}, // no remote pseudo-terminal
	'a': {}, // disable agent forwarding
	'q': {}, // quiet
	'v': {}, // verbose
}

// sshPermittedValueFlags declares the value-taking ssh(1) short options AX
// admits together with the rule that admits their value.
var sshPermittedValueFlags = map[byte]sshValueRule{
	'i': sshFlagValueRule(sshWordValue),               // identity file
	'l': sshFlagValueRule(sshWordValue),               // login name
	'p': sshFlagValueRule(sshNumericValue(1, 65_535)), // port
	'o': admitSSHOption,                               // configuration option
}

// sshOptionRule declares how one ssh_config option name given through -o is
// admitted. A nil permits refuses every value of that name.
type sshOptionRule struct {
	permits            func(string) bool
	hostAuthentication bool
}

// sshOptionRegistry is the declared source for -o admission. An absent name is
// refused; a present name is admitted only for the values its rule permits.
//
// The second group names the options that select or relax how ssh(1)
// authenticates the peer host key. Their presence does not widen admission —
// they are refused exactly like an undeclared name — but it lets the refusal
// report the Section 6.3 host-authentication clause instead of an unknown-name
// clause. StrictHostKeyChecking is declared with the single enforcing spelling
// so its aliases (no, off, false, accept-new, ask, and anything OpenSSH adds
// later) fall outside the permitted set without being listed.
var sshOptionRegistry = map[string]sshOptionRule{
	"addressfamily":       {permits: sshEnumeratedValue("any", "inet", "inet6")},
	"batchmode":           {permits: sshEnumeratedValue("yes", "no")},
	"compression":         {permits: sshEnumeratedValue("yes", "no")},
	"connectionattempts":  {permits: sshNumericValue(1, 16)},
	"connecttimeout":      {permits: sshNumericValue(1, 300)},
	"identitiesonly":      {permits: sshEnumeratedValue("yes", "no")},
	"identityfile":        {permits: sshWordValue},
	"loglevel":            {permits: sshEnumeratedValue("quiet", "fatal", "error", "info", "verbose", "debug", "debug1", "debug2", "debug3")},
	"port":                {permits: sshNumericValue(1, 65_535)},
	"serveralivecountmax": {permits: sshNumericValue(1, 100)},
	"serveraliveinterval": {permits: sshNumericValue(1, 86_400)},
	"tcpkeepalive":        {permits: sshEnumeratedValue("yes", "no")},
	"user":                {permits: sshWordValue},

	"stricthostkeychecking":            {permits: sshEnumeratedValue("yes"), hostAuthentication: true},
	"checkhostip":                      {hostAuthentication: true},
	"globalknownhostsfile":             {hostAuthentication: true},
	"hostkeyalgorithms":                {hostAuthentication: true},
	"hostkeyalias":                     {hostAuthentication: true},
	"knownhostscommand":                {hostAuthentication: true},
	"nohostauthenticationforlocalhost": {hostAuthentication: true},
	"updatehostkeys":                   {hostAuthentication: true},
	"userknownhostsfile":               {hostAuthentication: true},
	"verifyhostkeydns":                 {hostAuthentication: true},
}

// admitSSHArguments returns sshArgumentAdmitted when every argument is
// declared as permitted, or the refusal reason for the first argument that is
// not. Arguments are walked with ssh(1) getopt semantics: short options group,
// a value-taking letter ends its group and takes the rest of the argument or
// the whole next argument as its value, and a non-option argument is ssh(1)'s
// destination or remote command rather than an option.
func admitSSHArguments(arguments []string) string {
	for index := 0; index < len(arguments); index++ {
		argument := arguments[index]
		if len(argument) < 2 || argument[0] != '-' || argument == "--" {
			return sshRefusalUnpermittedArgument
		}
		group := argument[1:]
		for position := 0; position < len(group); position++ {
			letter := group[position]
			if _, valueless := sshPermittedFlags[letter]; valueless {
				continue
			}
			rule, permitted := sshPermittedValueFlags[letter]
			if !permitted {
				return sshRefusalUnpermittedFlag
			}
			value := group[position+1:]
			if value == "" {
				if index+1 >= len(arguments) {
					return sshRefusalMissingFlagValue
				}
				index++
				value = arguments[index]
			}
			if reason := rule(value); reason != sshArgumentAdmitted {
				return reason
			}
			break
		}
	}
	return sshArgumentAdmitted
}

// admitSSHOption admits one -o argument against sshOptionRegistry.
func admitSSHOption(option string) string {
	name, value := parseSSHConfigOption(option)
	rule, declared := sshOptionRegistry[name]
	if !declared {
		return sshRefusalUnpermittedOption
	}
	if rule.permits != nil && rule.permits(value) {
		return sshArgumentAdmitted
	}
	if rule.hostAuthentication {
		return sshRefusalHostAuthentication
	}
	return sshRefusalUnpermittedOptionValue
}

// parseSSHConfigOption splits an ssh_config option into its case-folded name
// and its raw value, accepting the separators OpenSSH accepts: "=", any run of
// whitespace, or both. A quoted value is unquoted.
func parseSSHConfigOption(option string) (string, string) {
	option = strings.TrimSpace(option)
	separator := strings.IndexFunc(option, func(character rune) bool { return character == '=' || unicode.IsSpace(character) })
	if separator < 0 {
		return strings.ToLower(option), ""
	}
	name := strings.ToLower(strings.TrimSpace(option[:separator]))
	remainder := strings.TrimLeftFunc(option[separator:], func(character rune) bool { return character == '=' || unicode.IsSpace(character) })
	value := strings.TrimSpace(remainder)
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		value = strings.TrimSpace(value[1 : len(value)-1])
	}
	return name, value
}

// sshFlagValueRule adapts a value predicate to the refusal-reason contract of
// a value-taking short option.
func sshFlagValueRule(permits func(string) bool) sshValueRule {
	return func(value string) string {
		if permits(value) {
			return sshArgumentAdmitted
		}
		return sshRefusalUnpermittedFlagValue
	}
}

// sshEnumeratedValue permits exactly the listed spellings, case-folded the way
// OpenSSH folds option values.
func sshEnumeratedValue(permitted ...string) func(string) bool {
	return func(value string) bool { return oneOf(strings.ToLower(value), permitted...) }
}

// sshNumericValue permits a plain decimal integer inside the closed range.
func sshNumericValue(min, max uint64) func(string) bool {
	return func(value string) bool {
		number, err := strconv.ParseUint(value, 10, 64)
		return err == nil && between(number, min, max)
	}
}

// sshWordValue permits a single printable whitespace-free word. It belongs to
// the options whose value is a name or path that cannot itself relax host
// authentication. Refusing embedded whitespace keeps the admitted value a
// single ssh_config token, which is what OpenSSH accepts for these options.
func sshWordValue(value string) bool {
	if strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return false
	}
	return validatePrintableCharacters(value, 1, 4_096) == nil
}
