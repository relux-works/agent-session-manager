// AC-HOST-001 real-carrier lane: the responder runs as a real child process
// behind a real OpenSSH loopback carrier (unprivileged sshd on 127.0.0.1 with
// a temp host key, temp authorized_keys and a forced command), and the
// initiator drives the production Dial/Call over the ssh stdio stream.
// Asserts Section 11.10.1 end to end: the client requests exactly the
// production ServeArgv words, the server observes exactly that request, TLS
// records are the first bytes, no PTY, no banner, stderr bounded.
// No ax command exists, so the forced command execs the production Serve
// entry through this test binary; the symbolic argv pin stays covered by
// TestServeArgvPinsChannelVersion and TestLaunchForConfig.
// Native Tailscale SSH is a stated bound (HC-PARITY by construction).
package hostchannel_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/relux-works/agent-session-manager/internal/hostchannel"
	"github.com/relux-works/agent-session-manager/internal/hosttrust"
	"github.com/relux-works/agent-session-manager/internal/localstore"
	"github.com/relux-works/agent-session-manager/internal/rpcwire"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

// sshHelperInput carries everything the responder child needs to reopen the
// fixture store and serve one channel over its stdio.
type sshHelperInput struct {
	ConfigPath   string   `json:"config_path"`
	DataDir      string   `json:"data_dir"`
	StateDir     string   `json:"state_dir"`
	CacheDir     string   `json:"cache_dir"`
	RuntimeDir   string   `json:"runtime_dir"`
	HostID       string   `json:"host_id"`
	CredentialID string   `json:"credential_id"`
	Allowlisted  []string `json:"allowlisted"`
	LeafDER      string   `json:"leaf_der_b64"`
	LeafKeyDER   string   `json:"leaf_key_der_b64"`
	Platform     string   `json:"platform"`
	AXVersion    string   `json:"ax_version"`
}

type sshStdio struct {
	stdin  io.WriteCloser
	stdout io.Reader
	once   sync.Once
}

func (s *sshStdio) Read(data []byte) (int, error)  { return s.stdout.Read(data) }
func (s *sshStdio) Write(data []byte) (int, error) { return s.stdin.Write(data) }
func (s *sshStdio) Close() error {
	s.once.Do(func() { _ = s.stdin.Close() })
	return nil
}

type prefixRecorder struct {
	r      io.Reader
	prefix []byte
}

func (p *prefixRecorder) Read(data []byte) (int, error) {
	n, err := p.r.Read(data)
	if len(p.prefix) < 8 && n > 0 {
		p.prefix = append(p.prefix, data[:n]...)
		if len(p.prefix) > 8 {
			p.prefix = p.prefix[:8]
		}
	}
	return n, err
}

// TestHostileSSHHelperResponder is the responder child behind sshd. Under a
// normal suite run the gate env is unset and it skips.
func TestHostileSSHHelperResponder(t *testing.T) {
	inputPath := os.Getenv("AX_HC_SSH_HELPER_INPUT")
	if inputPath == "" {
		t.Skip("responder child only runs behind the loopback carrier test")
	}
	// From here on this process is the responder: report on stderr and exit
	// with a code, never through the testing framework.
	fail := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "responder fatal: "+format+"\n", args...)
		os.Exit(1)
	}
	stdinStat, err := os.Stdin.Stat()
	if err != nil {
		fail("stat stdin: %v", err)
	}
	stdoutStat, err := os.Stdout.Stat()
	if err != nil {
		fail("stat stdout: %v", err)
	}
	if stdinStat.Mode()&os.ModeCharDevice != 0 || stdoutStat.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(os.Stderr, "responder pty=unexpected-terminal")
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "responder pty=none")
	fmt.Fprintf(os.Stderr, "responder original_command=%s\n", os.Getenv("SSH_ORIGINAL_COMMAND"))
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		fail("read input: %v", err)
	}
	var input sshHelperInput
	if err := json.Unmarshal(raw, &input); err != nil {
		fail("decode input: %v", err)
	}
	leafDER, err := base64.StdEncoding.DecodeString(input.LeafDER)
	if err != nil {
		fail("decode leaf: %v", err)
	}
	leafKeyDER, err := base64.StdEncoding.DecodeString(input.LeafKeyDER)
	if err != nil {
		fail("decode key: %v", err)
	}
	paths, err := localstore.ResolvePaths(localstore.ResolveRequest{
		Platform: scalar.PlatformLinux,
		Flags: map[string]string{
			"--config": input.ConfigPath, "--data-dir": input.DataDir,
			"--state-dir": input.StateDir, "--cache-dir": input.CacheDir,
			"--runtime-dir": input.RuntimeDir,
		},
	})
	if err != nil {
		fail("ResolvePaths: %v", err)
	}
	store, err := hosttrust.Open(paths)
	if err != nil {
		fail("Open: %v", err)
	}
	cert, err := hostchannel.CredentialFromIssued(hosttrust.IssuedCredential{LeafDER: leafDER, LeafKeyDER: leafKeyDER})
	if err != nil {
		fail("CredentialFromIssued: %v", err)
	}
	var calls atomic.Int64
	handler := func(_ context.Context, _ rpcwire.Request, _ hostchannel.Binding, _ func(func() error) error) (json.RawMessage, error) {
		calls.Add(1)
		return json.RawMessage(`{}`), nil
	}
	stream := &helperStdio{}
	serveErr := hostchannel.Serve(stream, hostchannel.ServerConfig{
		Store: store, Credential: cert,
		LocalHostID: input.HostID, LocalCredentialID: input.CredentialID,
		Allowlisted: input.Allowlisted, Platform: input.Platform, AXVersion: input.AXVersion,
		Handler:          handler,
		HandshakeTimeout: 15 * time.Second, HelloTimeout: 10 * time.Second, RequestTimeout: 10 * time.Second,
	})
	fmt.Fprintf(os.Stderr, "responder calls=%d\n", calls.Load())
	fmt.Fprintf(os.Stderr, "responder serve_err=%v\n", serveErr)
	if calls.Load() != 1 {
		os.Exit(1)
	}
	os.Exit(0)
}

type helperStdio struct{ once sync.Once }

func (s *helperStdio) Read(data []byte) (int, error)  { return os.Stdin.Read(data) }
func (s *helperStdio) Write(data []byte) (int, error) { return os.Stdout.Write(data) }
func (s *helperStdio) Close() error {
	s.once.Do(func() { _ = os.Stdin.Close() })
	return nil
}

func lookPath(t *testing.T, names ...string) (string, bool) {
	t.Helper()
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path, true
		}
		if filepath.IsAbs(name) {
			if info, err := os.Stat(name); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
				return name, true
			}
		}
	}
	return "", false
}

func TestHostileRealCarrierOpenSSH(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("loopback sshd carrier requires darwin or linux, have %s", runtime.GOOS)
	}
	sshdPath, ok := lookPath(t, "/usr/sbin/sshd", "sshd")
	if !ok {
		t.Skip("loopback sshd carrier requires an sshd binary")
	}
	sshPath, ok := lookPath(t, "ssh")
	if !ok {
		t.Skip("loopback sshd carrier requires an ssh binary")
	}
	keygenPath, ok := lookPath(t, "ssh-keygen")
	if !ok {
		t.Skip("loopback sshd carrier requires ssh-keygen")
	}
	user := os.Getenv("USER")
	if user == "" {
		t.Skip("loopback sshd carrier requires $USER")
	}
	f := newFixture(t)
	census := censusHostile(t, f)
	dir := t.TempDir()
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	hostKey := filepath.Join(dir, "ssh_host_ed25519_key")
	clientKey := filepath.Join(dir, "id_ed25519")
	run(keygenPath, "-t", "ed25519", "-N", "", "-f", hostKey, "-q")
	run(keygenPath, "-t", "ed25519", "-N", "", "-f", clientKey, "-q")
	clientPub, err := os.ReadFile(clientKey + ".pub")
	if err != nil {
		t.Fatalf("ReadFile(pub): %v", err)
	}
	// The responder child input: store paths plus credential bytes.
	input := sshHelperInput{
		ConfigPath: filepath.Join(f.dirB, "config.toml"), DataDir: filepath.Join(f.dirB, "data"),
		StateDir: f.dirB, CacheDir: filepath.Join(f.dirB, "cache"), RuntimeDir: filepath.Join(f.dirB, "runtime"),
		HostID: hostB, CredentialID: f.credBID, Allowlisted: []string{hostA},
		LeafDER:    base64.StdEncoding.EncodeToString(f.issuedB.LeafDER),
		LeafKeyDER: base64.StdEncoding.EncodeToString(f.issuedB.LeafKeyDER),
		Platform:   "linux", AXVersion: "0.6.0",
	}
	inputRaw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	inputPath := filepath.Join(dir, "helper-input.json")
	if err := os.WriteFile(inputPath, inputRaw, 0o600); err != nil {
		t.Fatalf("WriteFile(input): %v", err)
	}
	testBin, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatalf("Abs(testbin): %v", err)
	}
	for _, quoted := range []string{testBin, inputPath} {
		if strings.ContainsAny(quoted, "'\\") {
			t.Fatalf("unsafe helper path %q", quoted)
		}
	}
	forced := fmt.Sprintf("env AX_HC_SSH_HELPER_INPUT='%s' '%s' -test.run '^TestHostileSSHHelperResponder$'", inputPath, testBin)
	authorized := fmt.Sprintf("command=\"%s\",no-pty,no-agent-forwarding,no-X11-forwarding,no-port-forwarding,no-user-rc %s",
		strings.ReplaceAll(forced, "\"", "\\\""), strings.TrimSpace(string(clientPub)))
	if err := os.WriteFile(filepath.Join(dir, "authorized_keys"), []byte(authorized+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(authorized_keys): %v", err)
	}
	// The client requests exactly the production responder words; sshd's
	// forced command substitutes the in-process Serve entry and reports the
	// original request for the end-to-end argv assertion.
	remoteWords := append([]string(nil), hostchannel.ServeArgv...)
	if len(remoteWords) != 6 || remoteWords[0] != "ax" {
		t.Fatalf("ServeArgv = %q", remoteWords)
	}
	snapshot := loadSnapshot(t, v4Document)
	if argv, err := hostchannel.LaunchForConfig(snapshot, hostB); err != nil {
		t.Fatalf("LaunchForConfig: %v", err)
	} else if argv[0] != "-T" {
		t.Fatalf("production transport prefix = %q, want no-PTY -T first", argv)
	}
	freePort := func() int {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Listen: %v", err)
		}
		defer listener.Close()
		return listener.Addr().(*net.TCPAddr).Port
	}
	var sshd *exec.Cmd
	var sshdErr bytes.Buffer
	var port int
	started := false
	var lastReason string
	var sshdWaited chan error
	for range 3 {
		port = freePort()
		config := "Port " + fmt.Sprint(port) + "\n" +
			"ListenAddress 127.0.0.1\n" +
			"HostKey " + hostKey + "\n" +
			"PidFile " + filepath.Join(dir, "sshd.pid") + "\n" +
			"AuthorizedKeysFile " + filepath.Join(dir, "authorized_keys") + "\n" +
			"PasswordAuthentication no\nPubkeyAuthentication yes\nKbdInteractiveAuthentication no\n" +
			"StrictModes no\nLogLevel ERROR\nPermitTunnel no\nAllowAgentForwarding no\nX11Forwarding no\n"
		configPath := filepath.Join(dir, "sshd_config")
		if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
			t.Fatalf("WriteFile(sshd_config): %v", err)
		}
		sshdErr.Reset()
		sshd = exec.Command(sshdPath, "-D", "-e", "-f", configPath)
		sshd.Stderr = &sshdErr
		if err := sshd.Start(); err != nil {
			lastReason = err.Error()
			continue
		}
		waited := make(chan error, 1)
		go func() { waited <- sshd.Wait() }()
		deadline := time.Now().Add(10 * time.Second)
	poll:
		for {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				started = true
				break
			}
			select {
			case <-waited:
				break poll
			default:
			}
			if time.Now().After(deadline) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if started {
			// One Wait owns this process; the deferred cleanup consumes
			// its result instead of calling Wait a second time (a second
			// Wait blocks forever).
			sshdWaited = waited
			break
		}
		_ = sshd.Process.Kill()
		<-waited
		lastReason = strings.TrimSpace(sshdErr.String())
		if lastReason == "" {
			lastReason = "no listener and no diagnostics"
		}
	}
	if !started {
		t.Skipf("sshd cannot bind on 127.0.0.1 in this environment: %s", lastReason)
	}
	defer func() {
		_ = sshd.Process.Kill()
		<-sshdWaited
	}()
	sshArgs := []string{"-T", "-p", fmt.Sprint(port), "-i", clientKey,
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null", "-o", "IdentitiesOnly=yes",
		"-o", "LogLevel=ERROR", "-o", "ConnectTimeout=10",
		user + "@127.0.0.1"}
	sshArgs = append(sshArgs, remoteWords...)
	sshCmd := exec.Command(sshPath, sshArgs...)
	sshStdin, err := sshCmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	sshStdout, err := sshCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	var sshStderr bytes.Buffer
	sshCmd.Stderr = &sshStderr
	if err := sshCmd.Start(); err != nil {
		t.Fatalf("ssh start: %v", err)
	}
	recorder := &prefixRecorder{r: sshStdout}
	stream := &sshStdio{stdin: sshStdin, stdout: recorder}
	clientConfig := f.clientConfig()
	clientConfig.HandshakeTimeout = 15 * time.Second
	clientConfig.HelloTimeout = 10 * time.Second
	clientConfig.RequestTimeout = 10 * time.Second
	client, err := hostchannel.Dial(stream, clientConfig)
	if err != nil {
		_ = sshCmd.Process.Kill()
		_ = sshCmd.Wait()
		t.Fatalf("Dial over OpenSSH: %v\nssh stderr:\n%s", err, sshStderr.String())
	}
	response, err := client.Call("health.get", json.RawMessage(`{}`))
	if err != nil || !response.OK() {
		_ = sshCmd.Process.Kill()
		_ = sshCmd.Wait()
		t.Fatalf("Call over OpenSSH = %+v, %v\nssh stderr:\n%s", response, err, sshStderr.String())
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	waitErr := sshCmd.Wait()
	report := sshStderr.String()
	if waitErr != nil {
		t.Fatalf("ssh exit: %v\nssh stderr:\n%s", waitErr, report)
	}
	// Section 11.10.1 over the real carrier: TLS records first (0x16
	// handshake, 0x03 legacy version), no PTY, the exact responder words on
	// the wire, exactly one served call, bounded stderr.
	if len(recorder.prefix) < 2 || recorder.prefix[0] != 0x16 || recorder.prefix[1] != 0x03 {
		t.Fatalf("first stdout bytes = %x, want TLS records", recorder.prefix)
	}
	original := ""
	calls := ""
	pty := ""
	for _, line := range strings.Split(report, "\n") {
		if rest, ok := strings.CutPrefix(line, "responder original_command="); ok {
			original = rest
		}
		if rest, ok := strings.CutPrefix(line, "responder calls="); ok {
			calls = rest
		}
		if rest, ok := strings.CutPrefix(line, "responder pty="); ok {
			pty = rest
		}
	}
	if original != strings.Join(remoteWords, " ") {
		t.Fatalf("wire command = %q, want %q\nssh stderr:\n%s", original, strings.Join(remoteWords, " "), report)
	}
	if pty != "none" {
		t.Fatalf("pty = %q\nssh stderr:\n%s", pty, report)
	}
	if calls != "1" {
		t.Fatalf("served calls = %q\nssh stderr:\n%s", calls, report)
	}
	if len(report) >= 64*1024 {
		t.Fatalf("ssh stderr = %d bytes, want bounded diagnostics", len(report))
	}
	census.assertUnchanged(t, f)
}
