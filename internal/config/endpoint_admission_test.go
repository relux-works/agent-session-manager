package config

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// peerDocumentWithEndpoint builds a document whose only interesting content is
// one mesh peer carrying the given endpoint, so every case below is decided by
// the production loadConfigDocument entry and its mesh.peers[0].endpoint gate.
// ssh_args is left empty so no refusal can come from the neighbouring gate.
func peerDocumentWithEndpoint(endpoint string) []byte {
	return append(minimalValidConfigVersion(scalar.PlatformMacOS, CurrentVersion), []byte(`
[[mesh.peers]]
host_id = "`+testPeerID+`"
name = "peer"
endpoint = `+strconv.Quote(endpoint)+`
platform = "linux"
ssh_args = []
workspace_roots = []
`)...)
}

func requireEndpointRefusal(t *testing.T, reason, endpoint string) {
	t.Helper()
	_, err := loadConfigDocument(peerDocumentWithEndpoint(endpoint), scalar.PlatformMacOS, nil)
	requireConfigClause(t, err, "mesh.peers[0].endpoint "+reason)
}

func requireEndpointAdmitted(t *testing.T, endpoint string) {
	t.Helper()
	if _, err := loadConfigDocument(peerDocumentWithEndpoint(endpoint), scalar.PlatformMacOS, nil); err != nil {
		t.Fatalf("endpoint %q refused: %v", endpoint, err)
	}
}

// requireEndpointRefusedByItsOwnClause is the isolation rule every negative case
// below obeys: the refused endpoint and an otherwise identical endpoint with
// only the named violation removed. Admitting the neighbour proves the rest of
// the string is grammar-clean, so the named clause — not some incidental second
// defect in the same string — is what decided the refusal.
func requireEndpointRefusedByItsOwnClause(t *testing.T, reason, endpoint, neighbour string) {
	t.Helper()
	requireEndpointRefusal(t, reason, endpoint)
	requireEndpointAdmitted(t, neighbour)
}

// TestLoadRefusesOptionLikeMeshEndpoints covers the audit probe that loaded
// every one of these endpoints with err=nil. SPEC.md Section 6.2 passes the
// endpoint to ssh(1) as an atomic argv value, and ssh(1) reads its destination
// positionally through getopt: an argv value beginning with "-" is an option,
// not a destination. Without this gate the endpoint field reaches exactly the
// Section 6.3 host-authentication bypass that ssh_args refuses, through a
// field the ssh_args gate never inspects.
//
// These are the reported shapes and they are kept as regression coverage, but
// none of them isolates the clause: strip the leading "-" from any of them and
// the [user@]host[:port] grammar refuses the remainder anyway, so they pin the
// clause's reason string and its ordering ahead of the grammar rather than its
// coverage class. The class is pinned by the test below.
func TestLoadRefusesOptionLikeMeshEndpoints(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-F/attacker/config",
		"-J",
		"-4",
		"-",
		"--",
		"--proxy-command=id",
	} {
		endpoint := endpoint
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()
			requireEndpointRefusal(t, endpointRefusalOptionLike, endpoint)
		})
	}
}

// optionShapedEndpointsTheGrammarWouldAdmit carries, for each option-shaped
// endpoint, the same endpoint with only its leading "-" removed. Every one of
// those neighbours is admitted by the [user@]host[:port] grammar, because "-"
// is a legal login-name byte — so for these shapes the leading-hyphen clause is
// the only thing standing between the field and ssh(1), and a narrowing of that
// clause admits them.
//
// ssh(1) parses "-ivan@peer.example" as -i with identity file "van@peer.example":
// an option, not a destination, which is the exact failure mode this bug closes.
var optionShapedEndpointsTheGrammarWouldAdmit = map[string]string{
	"identity file letter": "-ivan@peer.example",
	"address family digit": "-4@peer.example",
	"login name letter":    "-l@peer.example",
	"port letter":          "-p2222@peer.example",
	"dotted login name":    "-i.ssh.key@peer.example:22",
	"bracketed IPv6 host":  "-ivan@[2001:db8::1]:22",
	"user and port":        "-oivan@peer.example:2222",
}

// TestLoadRefusesOptionShapedEndpointsTheGrammarWouldAdmit is the load-bearing
// pin for the leading-hyphen clause. Each case is refused as option-like, and
// its de-hyphenated neighbour is admitted — so narrowing the clause (carving
// out "@"-bearing values, matching only "--", or moving the check after the
// user/host split) admits a live ssh(1) option injection and reddens here.
func TestLoadRefusesOptionShapedEndpointsTheGrammarWouldAdmit(t *testing.T) {
	t.Parallel()

	for _, name := range sortedEndpointCaseNames(optionShapedEndpointsTheGrammarWouldAdmit) {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			endpoint := optionShapedEndpointsTheGrammarWouldAdmit[name]
			requireEndpointRefusedByItsOwnClause(t, endpointRefusalOptionLike, endpoint, strings.TrimPrefix(endpoint, "-"))
		})
	}
}

// TestLoadRefusesMeshEndpointsCarryingWhitespace covers the third probe shape.
// A whitespace-carrying endpoint cannot name one host, and it is the same
// option injection one word-split away from any consumer that does not keep
// the value atomic.
//
// No whitespace-carrying endpoint could be admitted with this clause removed —
// neither a login name nor a DNS label spells a space — so unlike the
// leading-hyphen clause this one cannot be pinned by admission. What it does
// own is the reason and the ordering: with the clause deleted or narrowed these
// endpoints report the host or user clause instead, which reddens every case
// here. The neighbours below carry that as far as it goes, by proving the
// remainder of each endpoint is grammar-valid once the space is removed.
func TestLoadRefusesMeshEndpointsCarryingWhitespace(t *testing.T) {
	t.Parallel()

	for _, name := range sortedEndpointCaseNames(whitespaceEndpointCases) {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testCase := whitespaceEndpointCases[name]
			if testCase.neighbour == "" {
				requireEndpointRefusal(t, endpointRefusalWhitespace, testCase.endpoint)
				return
			}
			requireEndpointRefusedByItsOwnClause(t, endpointRefusalWhitespace, testCase.endpoint, testCase.neighbour)
		})
	}
}

// whitespaceEndpointCases pairs each whitespace shape with the same endpoint
// minus the space where that remainder is grammar-valid. The reported probe
// shapes carry an injected option after the space, so no such neighbour exists
// for them and they are asserted on the refusal alone.
var whitespaceEndpointCases = map[string]struct{ endpoint, neighbour string }{
	"trailing option":   {endpoint: "peer.example -oStrictHostKeyChecking=no"},
	"trailing flag":     {endpoint: "peer.example -F /attacker/config"},
	"double space":      {endpoint: "peer.example  -oProxyCommand=id"},
	"leading space":     {endpoint: " peer.example", neighbour: "peer.example"},
	"trailing space":    {endpoint: "peer.example ", neighbour: "peer.example"},
	"two words":         {endpoint: "peer example", neighbour: "peerexample"},
	"remote command":    {endpoint: "peer.example id", neighbour: "peer.exampleid"},
	"space before at":   {endpoint: "ivan @peer.example", neighbour: "ivan@peer.example"},
	"space after at":    {endpoint: "ivan@ peer.example", neighbour: "ivan@peer.example"},
	"space before port": {endpoint: "peer.example :22", neighbour: "peer.example:22"},
}

// TestLoadRefusesMeshEndpointWhitespaceOutsideTheASCIISpace records which gate
// owns the remaining whitespace. Only the ASCII space survives the endpoint's
// printable-character bound, so tab, newline and the non-breaking space are
// refused one clause earlier, by that bound rather than by the grammar.
func TestLoadRefusesMeshEndpointWhitespaceOutsideTheASCIISpace(t *testing.T) {
	t.Parallel()

	for name, endpoint := range map[string]string{
		"tab":                "peer.example\t-oProxyCommand=id",
		"newline":            "peer.example\n-oProxyCommand=id",
		"carriage return":    "peer.example\r-oProxyCommand=id",
		"non-breaking space": "peer.example\u00a0-oProxyCommand=id",
	} {
		name, endpoint := name, endpoint
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := loadConfigDocument(peerDocumentWithEndpoint(endpoint), scalar.PlatformMacOS, nil)
			requireConfigClause(t, err, "mesh.peers[0].endpoint")
		})
	}
}

// endpointGrammarCases pins the declared [user@]host[:port] grammar in both
// directions. Every refusal names the clause it violates and carries a
// neighbour that differs only by that violation and is admitted, so a case
// cannot silently come to rest on a clause other than the one it names, and
// weakening one clause reddens only its own cases.
var endpointGrammarCases = map[string]struct {
	endpoint  string
	reason    string
	neighbour string
}{
	// Admitted, including the endpoint SPEC.md Section 6.2 shows.
	"spec example":            {endpoint: "ivan@workstation.tailnet.ts.net", reason: endpointAdmitted},
	"bare host":               {endpoint: "peer.example", reason: endpointAdmitted},
	"single label":            {endpoint: "x", reason: endpointAdmitted},
	"hyphen inside label":     {endpoint: "ax-host.example", reason: endpointAdmitted},
	"host and port":           {endpoint: "peer.example:2222", reason: endpointAdmitted},
	"user host and port":      {endpoint: "ivan@peer.example:22", reason: endpointAdmitted},
	"underscore in user":      {endpoint: "ax_user@peer.example", reason: endpointAdmitted},
	"IPv4 literal":            {endpoint: "192.0.2.10", reason: endpointAdmitted},
	"IPv4 literal and port":   {endpoint: "192.0.2.10:2222", reason: endpointAdmitted},
	"bracketed IPv6":          {endpoint: "[2001:db8::1]", reason: endpointAdmitted},
	"bracketed IPv6 and port": {endpoint: "[2001:db8::1]:2222", reason: endpointAdmitted},
	"bracketed IPv6 loopback": {endpoint: "[::1]:22", reason: endpointAdmitted},
	"user and IPv6":           {endpoint: "ivan@[2001:db8::1]:2222", reason: endpointAdmitted},
	"longest label":           {endpoint: "a" + strings.Repeat("b", 62), reason: endpointAdmitted},
	"longest user":            {endpoint: strings.Repeat("u", 64) + "@peer.example", reason: endpointAdmitted},
	"lowest port":             {endpoint: "peer.example:1", reason: endpointAdmitted},
	"highest port":            {endpoint: "peer.example:65535", reason: endpointAdmitted},
	// A leading-zero port is admitted and reads as decimal: ssh(1) resolves
	// "022" to 22. Pinned as reviewed behaviour rather than left unstated,
	// because the grammar declares no canonical-spelling rule for the port.
	"leading zero port": {endpoint: "peer.example:022", reason: endpointAdmitted},

	// Refused by the user clause.
	"empty user":        {endpoint: "@peer.example", reason: endpointRefusalUser, neighbour: "u@peer.example"},
	"user over bound":   {endpoint: strings.Repeat("u", 65) + "@peer.example", reason: endpointRefusalUser, neighbour: strings.Repeat("u", 64) + "@peer.example"},
	"slash in user":     {endpoint: "a/b@peer.example", reason: endpointRefusalUser, neighbour: "a.b@peer.example"},
	"second at sign":    {endpoint: "ivan@ivan@peer.example", reason: endpointRefusalUser, neighbour: "ivan.ivan@peer.example"},
	"non-ASCII user":    {endpoint: "иван@peer.example", reason: endpointRefusalUser, neighbour: "ivan@peer.example"},
	"percent in user":   {endpoint: "ivan%40root@peer.example", reason: endpointRefusalUser, neighbour: "ivan.40root@peer.example"},
	"semicolon in user": {endpoint: "ivan;id@peer.example", reason: endpointRefusalUser, neighbour: "ivan.id@peer.example"},

	// Refused by the host clause.
	"empty label":            {endpoint: "peer..example", reason: endpointRefusalHost, neighbour: "peer.example"},
	"trailing dot":           {endpoint: "peer.example.", reason: endpointRefusalHost, neighbour: "peer.example"},
	"label starts with dash": {endpoint: "ivan@-peer.example", reason: endpointRefusalHost, neighbour: "ivan@peer.example"},
	"label ends with dash":   {endpoint: "peer-.example", reason: endpointRefusalHost, neighbour: "peer.example"},
	"underscore in host":     {endpoint: "peer_example", reason: endpointRefusalHost, neighbour: "peer-example"},
	"slash in host":          {endpoint: "peer.example/path", reason: endpointRefusalHost, neighbour: "peer.example.path"},
	"semicolon in host":      {endpoint: "peer.example;id", reason: endpointRefusalHost, neighbour: "peer.exampleid"},
	"non-ASCII host":         {endpoint: "界.example", reason: endpointRefusalHost, neighbour: "jie.example"},
	"label over bound":       {endpoint: "a" + strings.Repeat("b", 63), reason: endpointRefusalHost, neighbour: "a" + strings.Repeat("b", 62)},
	"bare IPv6":              {endpoint: "2001:db8::1", reason: endpointRefusalHost, neighbour: "[2001:db8::1]"},
	// A bare IPv6 literal that ends in a port-shaped group is the shape a
	// "accept anything net.ParseIP accepts" widening of admitEndpointHost lets
	// through: the host part left after the port split is still a valid IP.
	"bare IPv6 and port": {endpoint: "2001:db8::1:2222", reason: endpointRefusalHost, neighbour: "[2001:db8::1]:2222"},
	"bare IPv6 loopback": {endpoint: "::1:22", reason: endpointRefusalHost, neighbour: "[::1]:22"},
	"unbalanced bracket": {endpoint: "[2001:db8::1", reason: endpointRefusalHost, neighbour: "[2001:db8::1]"},
	"empty brackets":     {endpoint: "[]", reason: endpointRefusalHost, neighbour: "[::1]"},
	"bracketed IPv4":     {endpoint: "[192.0.2.10]", reason: endpointRefusalHost, neighbour: "192.0.2.10"},
	"bracketed name":     {endpoint: "[peer.example]", reason: endpointRefusalHost, neighbour: "peer.example"},
	"empty host":         {endpoint: "ivan@:22", reason: endpointRefusalHost, neighbour: "ivan@h:22"},
	"colon inside host":  {endpoint: "peer.example::22", reason: endpointRefusalHost, neighbour: "peer.example:22"},

	// Refused by the port clause.
	"zero port":        {endpoint: "peer.example:0", reason: endpointRefusalPort, neighbour: "peer.example:1"},
	"port over bound":  {endpoint: "peer.example:65536", reason: endpointRefusalPort, neighbour: "peer.example:65535"},
	"empty port":       {endpoint: "peer.example:", reason: endpointRefusalPort, neighbour: "peer.example:22"},
	"non-numeric port": {endpoint: "peer.example:22a", reason: endpointRefusalPort, neighbour: "peer.example:22"},
	"signed port":      {endpoint: "peer.example:+22", reason: endpointRefusalPort, neighbour: "peer.example:22"},
	// Refused by the 1..5-byte port length, which leading zeros are what makes
	// reachable: the value is decimal 22, inside the numeric bound, so these
	// cases rest on the length clause alone rather than on "port over bound".
	// The six-byte one sits one step past the limit; the seven-byte one is
	// depth behind it. Review found the seven-byte case alone pins the bound to
	// somewhere in [5,6] rather than to 5, so narrowing len(port) > 5 to > 6
	// survived while admitting "peer.example:000022".
	"port at length bound":   {endpoint: "peer.example:000022", reason: endpointRefusalPort, neighbour: "peer.example:22"},
	"port over length bound": {endpoint: "peer.example:0000022", reason: endpointRefusalPort, neighbour: "peer.example:22"},
	"IPv6 without colon":     {endpoint: "[2001:db8::1]2222", reason: endpointRefusalPort, neighbour: "[2001:db8::1]:2222"},
}

// TestLoadAdmitsExactlyTheDeclaredMeshEndpointGrammar drives every case above
// through the production Load entry. The admitted half proves the gate is not
// vacuously strict; the refused half names the clause and admits its neighbour,
// so a weakened clause reddens its own cases and leaves the others green.
func TestLoadAdmitsExactlyTheDeclaredMeshEndpointGrammar(t *testing.T) {
	t.Parallel()

	names := make([]string, 0, len(endpointGrammarCases))
	for name := range endpointGrammarCases {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			testCase := endpointGrammarCases[name]
			if testCase.reason == endpointAdmitted {
				requireEndpointAdmitted(t, testCase.endpoint)
				return
			}
			requireEndpointRefusedByItsOwnClause(t, testCase.reason, testCase.endpoint, testCase.neighbour)
		})
	}
}

// TestEveryRefusedMeshEndpointCaseDeclaresAnIsolatingNeighbour keeps the
// isolation rule structural rather than a convention a later case can quietly
// drop: a refusal case with no neighbour proves nothing about the clause it
// names, because a second defect in the same string could be what refuses it.
func TestEveryRefusedMeshEndpointCaseDeclaresAnIsolatingNeighbour(t *testing.T) {
	t.Parallel()

	for name, testCase := range endpointGrammarCases {
		if testCase.reason == endpointAdmitted {
			if testCase.neighbour != "" {
				t.Errorf("admitted case %q declares a neighbour %q", name, testCase.neighbour)
			}
			continue
		}
		if testCase.neighbour == "" {
			t.Errorf("refused case %q declares no isolating neighbour", name)
		}
	}
}

// endpointBoundEdges names every numeric bound the endpoint grammar declares
// and pins it at its edge: the last value the clause admits and the first value
// past it, which must be refused by that clause. A bound proven at a distant
// value pins a range rather than a limit — "peer.example:0000022" is seven
// bytes, so it stayed refused when len(port) > 5 was narrowed to > 6 and that
// mutant survived the whole suite. This table is the structural form of the
// rule, so a bound that loses its adjacent case reddens here.
//
// Each refused spelling is chosen so only the named bound decides it: the
// six-byte port is decimal 22, inside 1..65535, and the 65536 port is five
// bytes, inside the length bound. The endpoint's own 1..1024 character bound is
// owned by schema_test.go, which proves it at 1024 accepted / 1025 refused.
var endpointBoundEdges = map[string]struct {
	admitted string
	refused  string
	reason   string
}{
	"user length lower":       {admitted: "u@peer.example", refused: "@peer.example", reason: endpointRefusalUser},
	"user length upper":       {admitted: strings.Repeat("u", 64) + "@peer.example", refused: strings.Repeat("u", 65) + "@peer.example", reason: endpointRefusalUser},
	"host label length lower": {admitted: "peer.a.example", refused: "peer..example", reason: endpointRefusalHost},
	"host label length upper": {admitted: "a" + strings.Repeat("b", 62), refused: "a" + strings.Repeat("b", 63), reason: endpointRefusalHost},
	"port length lower":       {admitted: "peer.example:1", refused: "peer.example:", reason: endpointRefusalPort},
	"port length upper":       {admitted: "peer.example:65535", refused: "peer.example:000022", reason: endpointRefusalPort},
	"port value lower":        {admitted: "peer.example:1", refused: "peer.example:0", reason: endpointRefusalPort},
	"port value upper":        {admitted: "peer.example:65535", refused: "peer.example:65536", reason: endpointRefusalPort},
}

// TestEveryDeclaredMeshEndpointBoundIsPinnedAtItsEdge drives both sides of every
// bound through the production Load entry. Moving a bound by one in either
// direction reddens the pair that owns it and leaves the others green.
func TestEveryDeclaredMeshEndpointBoundIsPinnedAtItsEdge(t *testing.T) {
	t.Parallel()

	for _, name := range sortedEndpointCaseNames(endpointBoundEdges) {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			edge := endpointBoundEdges[name]
			requireEndpointAdmitted(t, edge.admitted)
			requireEndpointRefusal(t, edge.reason, edge.refused)
		})
	}
}

// TestMeshEndpointAdmissionRefusesThroughTheMeshPeerClause guards the
// production wiring: the refusal must come from validateMesh's peer prefix,
// not from an unrelated bound that happens to reject the same string.
func TestMeshEndpointAdmissionRefusesThroughTheMeshPeerClause(t *testing.T) {
	t.Parallel()

	_, err := loadConfigDocument(peerDocumentWithEndpoint("-oStrictHostKeyChecking=no"), scalar.PlatformMacOS, nil)
	if !errors.Is(err, ErrConfigValidation) {
		t.Fatalf("error = %v, want ErrConfigValidation", err)
	}
	var documentError *DocumentError
	if !errors.As(err, &documentError) {
		t.Fatalf("error = %T, want *DocumentError", err)
	}
	if !strings.HasPrefix(documentError.Clause, "mesh.peers[0].endpoint ") {
		t.Fatalf("clause = %q, want the mesh peer endpoint prefix", documentError.Clause)
	}
}

func sortedEndpointCaseNames[Case any](cases map[string]Case) []string {
	names := make([]string, 0, len(cases))
	for name := range cases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
