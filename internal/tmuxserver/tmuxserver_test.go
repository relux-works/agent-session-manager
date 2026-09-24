package tmuxserver

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/relux-works/agent-session-manager/internal/axerror"
	"github.com/relux-works/agent-session-manager/internal/catalog"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/terminalbackend"
)

const (
	fixtureSession       = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
	fixtureGeneration    = "generation-7"
	fixtureStaleGen      = "generation-6"
	fixtureBroker        = "absent"
	fixtureRemedy        = "sign in to the GUI realm and retry"
	fixtureDecoyCapacity = "headless_creation"
)

// realmAdmitted models a probe-reported landed admission holding the
// credential realm row: in production the probe reports the
// Reconcile/ResolveEvidence verdict, and tests model the report.
func realmAdmitted() terminalbackend.Admitted {
	return terminalbackend.Admitted{Capabilities: []string{"credential_capable_execution_realm"}}
}

// boundAdmission models a live admission bound to the request's typed
// generation.
func boundAdmission() RealmAdmission {
	return RealmAdmission{Admitted: realmAdmitted(), RawGeneration: fixtureGeneration}
}

// staleAdmission models yesterday's live proof replayed today: the
// row is admitted but bound to the previous generation.
func staleAdmission() RealmAdmission {
	return RealmAdmission{Admitted: realmAdmitted(), RawGeneration: fixtureStaleGen}
}

// brokerReport models a live broker report: a same-user principal
// bound to the typed generation plus the bound server admission.
func brokerReport() BrokerReport {
	return BrokerReport{
		Principal: BrokerPrincipal{UID: currentUID(), Generation: fixtureGeneration},
		Server:    boundAdmission(),
	}
}

// foreignUID stages a broker principal the same-user arm must refuse
// without privilege: the test identity shifted out of range.
func foreignUID() uint32 { return currentUID() + 100000 }

// catalogDecoyCapabilities derives every non-realm member of the
// sixteen-value Section 4.D capability table from the pinned catalog
// (internal/catalog, generated from the adopted spec lock): the realm
// row credential_capable_execution_realm is the only admission that
// may authorize, so every other member is a decoy the membership arm
// must refuse. The derivation — not a fixture constant — is what
// makes the completeness claim: a new Section 4.D member is tested
// the moment the catalog carries it, and a drifted table fails here
// instead of silently narrowing the set. The realm row is matched by
// its spec literal, never the production constant, and its presence
// is asserted so the exclusion cannot go vacuous.
func catalogDecoyCapabilities(t *testing.T) []string {
	t.Helper()
	var decoys []string
	section := 0
	seenRealm := false
	for _, capability := range catalog.Current().Capabilities {
		if capability.NormativeSection != "4.D" {
			continue
		}
		section++
		name := string(capability.Name)
		if name == "credential_capable_execution_realm" {
			seenRealm = true
			continue
		}
		decoys = append(decoys, name)
	}
	if section != 16 {
		t.Fatalf("Section 4.D capabilities = %d, want the sixteen-value table", section)
	}
	if !seenRealm {
		t.Fatal("Section 4.D table carries no credential_capable_execution_realm row")
	}
	return decoys
}

func validRequest(t *testing.T, root string) Request {
	t.Helper()
	return Request{
		Caller:      CallerForeground,
		Platform:    scalar.PlatformMacOS,
		Root:        root,
		SessionID:   fixtureSession,
		Generation:  fixtureGeneration,
		BrokerState: fixtureBroker,
		Remediation: fixtureRemedy,
	}
}

func hostileAmbient() Ambient {
	return Ambient{
		TMUXEnv:          "/tmp/tmux-501/default,12345,0",
		TMUXTmpDir:       "/tmp/operator-tmux",
		InheritedSocket:  "/tmp/inherited.sock",
		DefaultPath:      "/tmp/tmux-501/default",
		ConventionalName: "default",
	}
}

// requireLocalError pins the refusal the caller renders: the TOP-LEVEL
// dynamic type must be *tmuxserver.Error, and no *axerror.Error wrapper
// may be reachable. errors.As alone looks through the
// capability_unavailable wrapper (axerror.Error.Unwrap returns the
// cause, which is redacted off the wire by design), so a wrapped
// custody refusal would satisfy a code/detail assertion while the
// caller sees the typed unavailable. Every custody assertion goes
// through this helper, so the wrapper direction of the absence split
// is measured here, not in prose.
func requireLocalError(t *testing.T, err error, code, detail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("err = nil, want %s at %s", code, detail)
	}
	local, ok := err.(*Error)
	if !ok {
		t.Fatalf("err top-level type = %T (%v), want *tmuxserver.Error", err, err)
	}
	var wrapped *axerror.Error
	if errors.As(err, &wrapped) {
		t.Fatalf("err wraps *axerror.Error (%v), want a bare *tmuxserver.Error", err)
	}
	if local.Code != code || local.Detail != detail {
		t.Fatalf("err = %s at %s, want %s at %s", local.Code, local.Detail, code, detail)
	}
}

func requireCapabilityUnavailable(t *testing.T, err error, caller, broker, generation, remedy string) {
	t.Helper()
	if err == nil {
		t.Fatal("err = nil, want capability_unavailable")
	}
	var failure *axerror.Error
	if !errors.As(err, &failure) {
		t.Fatalf("err type = %T (%v), want *axerror.Error", err, err)
	}
	if string(failure.Code()) != "capability_unavailable" {
		t.Fatalf("code = %q, want %q", string(failure.Code()), "capability_unavailable")
	}
	if failure.ExitCode() != 6 {
		t.Fatalf("exit = %d, want 6", failure.ExitCode())
	}
	want := map[string]string{
		"capability":             "credential_capable_execution_realm",
		"caller_realm":           caller,
		"broker_state":           broker,
		"tmux_server_generation": generation,
		"remediation":            remedy,
	}
	for key, value := range want {
		got, ok := failure.Detail(key)
		if !ok {
			t.Fatalf("missing typed detail %q", key)
		}
		if got != value {
			t.Fatalf("detail %q = %v, want %q", key, got, value)
		}
	}
}

type spawnSpy struct {
	calls  int
	socket string
	argv   []string
	err    error
}

func (spy *spawnSpy) spawn(socket string, argv []string) error {
	spy.calls++
	spy.socket = socket
	spy.argv = append([]string(nil), argv...)
	return spy.err
}

// probeServerSpy is the background-path census for the foreground
// probe dependency: the background path never probes the dedicated
// server directly, so every background-path Acquire test arms this
// spy and asserts zero calls. A fallback plant that re-enters the
// foreground path trips this spy before it reaches the spawner.
type probeServerSpy struct {
	calls int
}

func (spy *probeServerSpy) probe(socket string) (ServerReport, error) {
	spy.calls++
	return ServerReport{}, nil
}

func (spy *probeServerSpy) requireSilent(t *testing.T) {
	t.Helper()
	if spy.calls != 0 {
		t.Fatalf("server probe calls = %d, want 0 on the background path", spy.calls)
	}
}

func runtimeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func runtimeDir(t *testing.T, root string) string {
	t.Helper()
	dir, err := EnsureRuntimeDir(root, RuntimeDirName, scalar.PlatformMacOS, nil)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// The derived path is the path the specification mandates: pinned
// SPEC 806-807 selects the dedicated server by
// `tmux -S <runtime>/tmux/ax.sock`. Both leaves are asserted as spec
// literals plus the joined form, so a renamed constant reddens here.
func TestSocketPathDerivesOnlyFromRuntimeDir(t *testing.T) {
	if RuntimeDirName != "tmux" {
		t.Fatalf("RuntimeDirName = %q, want the SPEC 806-808 leaf %q", RuntimeDirName, "tmux")
	}
	if SocketName != "ax.sock" {
		t.Fatalf("SocketName = %q, want the SPEC 806-807 leaf %q", SocketName, "ax.sock")
	}
	root := t.TempDir()
	dir := filepath.Join(root, RuntimeDirName)
	if got := SocketPath(dir); got != filepath.Join(root, "tmux", "ax.sock") {
		t.Fatalf("SocketPath = %q, want <root>/tmux/ax.sock", got)
	}
}
