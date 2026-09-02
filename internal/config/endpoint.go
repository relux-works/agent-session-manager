package config

import (
	"net"
	"strings"
	"unicode"
)

// Admission for the mesh peer endpoint AX hands to ssh(1) as its destination.
//
// SPEC.md Section 6.2 states that the endpoint and every SSH argument "are
// passed as atomic argv values, never through a shell", and Section 6.3
// authorizes a peer only when "the SSH endpoint resolves to the authenticated
// host expected by SSH host-key policy". ssh(1) reads that destination
// positionally through getopt, so an atomic argv value is not by itself a
// destination: an endpoint beginning with "-" is parsed as an option instead,
// and "-oStrictHostKeyChecking=no" written into this field is the same host-
// authentication bypass Section 6.3 requires ssh_args to refuse, reached
// through a field the ssh_args gate never inspects. An endpoint carrying
// whitespace is the same injection one word-split away, and it cannot name a
// host either way.
//
// The field is therefore admitted against the closed [user@]host[:port]
// grammar below rather than filtered for known-bad shapes: a spelling this
// package has never considered is refused instead of admitted. The grammar is
// deliberately narrower than every destination ssh(1) would accept; widening
// it is a reviewed change, not an accident.
const (
	endpointAdmitted          = ""
	endpointRefusalOptionLike = "option-like value"
	endpointRefusalWhitespace = "embedded whitespace"
	endpointRefusalUser       = "user"
	endpointRefusalHost       = "host"
	endpointRefusalPort       = "port"
)

// admitMeshEndpoint returns endpointAdmitted when the endpoint matches the
// declared [user@]host[:port] grammar, or the refusal reason for the first
// clause it violates. The overall length is not checked here; the peer's
// 1..1024 printable-character bound owns it.
func admitMeshEndpoint(endpoint string) string {
	// Checked ahead of the grammar so the refusal names the option-injection
	// shape rather than reporting a malformed user or host.
	if strings.HasPrefix(endpoint, "-") {
		return endpointRefusalOptionLike
	}
	if strings.IndexFunc(endpoint, unicode.IsSpace) >= 0 {
		return endpointRefusalWhitespace
	}
	remainder := endpoint
	// The last "@" separates user from host: OpenSSH splits there too, so a
	// host part can never smuggle an "@" past this split.
	if separator := strings.LastIndexByte(remainder, '@'); separator >= 0 {
		if !admitEndpointUser(remainder[:separator]) {
			return endpointRefusalUser
		}
		remainder = remainder[separator+1:]
	}
	host := remainder
	switch {
	case strings.HasPrefix(remainder, "["):
		closing := strings.IndexByte(remainder, ']')
		if closing < 0 {
			return endpointRefusalHost
		}
		host, remainder = remainder[:closing+1], remainder[closing+1:]
		if remainder != "" {
			if remainder[0] != ':' || !admitEndpointPort(remainder[1:]) {
				return endpointRefusalPort
			}
		}
	default:
		if separator := strings.LastIndexByte(remainder, ':'); separator >= 0 {
			host = remainder[:separator]
			if !admitEndpointPort(remainder[separator+1:]) {
				return endpointRefusalPort
			}
		}
	}
	if !admitEndpointHost(host) {
		return endpointRefusalHost
	}
	return endpointAdmitted
}

// admitEndpointUser permits a 1..64 byte login name spelled from the portable
// POSIX name characters plus "." and "-". A leading "-" is already refused for
// the whole endpoint, which is where a login name could otherwise reintroduce
// an option.
func admitEndpointUser(user string) bool {
	if len(user) < 1 || len(user) > 64 {
		return false
	}
	for index := 0; index < len(user); index++ {
		if !isEndpointUserByte(user[index]) {
			return false
		}
	}
	return true
}

func isEndpointUserByte(character byte) bool {
	switch {
	case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z':
		return true
	case character >= '0' && character <= '9':
		return true
	case character == '.' || character == '_' || character == '-':
		return true
	}
	return false
}

// admitEndpointHost permits a bracketed IP literal or a dot-separated name
// whose labels are the LDH characters of a DNS label. An IPv4 literal is
// already an LDH name. A bare IPv6 literal is refused: its colons are
// indistinguishable from a port separator, so it must be bracketed.
//
// The per-label bound is the DNS 63-byte label; the whole-name bound is left
// to the endpoint's 1..1024 printable-character bound so that bound stays
// reachable and provable through this grammar.
func admitEndpointHost(host string) bool {
	if strings.HasPrefix(host, "[") {
		if !strings.HasSuffix(host, "]") || len(host) < 3 {
			return false
		}
		address := net.ParseIP(host[1 : len(host)-1])
		return address != nil && address.To4() == nil
	}
	if host == "" {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if !admitEndpointHostLabel(label) {
			return false
		}
	}
	return true
}

func admitEndpointHostLabel(label string) bool {
	if len(label) < 1 || len(label) > 63 {
		return false
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for index := 0; index < len(label); index++ {
		character := label[index]
		switch {
		case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-':
		default:
			return false
		}
	}
	return true
}

// admitEndpointPort permits a plain decimal TCP port in 1..65535. The digits
// are checked directly rather than through strconv, which also accepts a sign
// prefix that is not a port.
func admitEndpointPort(port string) bool {
	if len(port) < 1 || len(port) > 5 {
		return false
	}
	number := 0
	for index := 0; index < len(port); index++ {
		if port[index] < '0' || port[index] > '9' {
			return false
		}
		number = number*10 + int(port[index]-'0')
	}
	return number >= 1 && number <= 65_535
}
