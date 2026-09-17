package sshtransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/peeridentity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
)

const peerID = "0198f4c8-7d40-7e55-8e6f-1234567890ab"

// The fixture is an executable process, not a fake Runner. It never connects to
// SSH, reads credentials, or claims to emulate cryptographic authentication.
func TestMain(m *testing.M) {
	mode := os.Getenv("AX_SSH_FIXTURE")
	if mode == "" {
		os.Exit(m.Run())
	}
	switch mode {
	case "argv":
		b, _ := json.Marshal(os.Args[1:])
		fmt.Println(string(b))
	case "echo":
		_, _ = io.Copy(os.Stdout, os.Stdin)
	case "sink":
		_, _ = io.Copy(io.Discard, os.Stdin)
	case "exit255":
		fmt.Fprint(os.Stderr, "credential-canary")
		os.Exit(255)
	case "exit7":
		fmt.Fprintln(os.Stdout, `{"ok":true}`)
		os.Exit(7)
	case "empty":
	case "partial":
		fmt.Fprint(os.Stdout, `{"ok":true}`)
	case "blank":
		fmt.Fprintln(os.Stdout)
	case "oversize":
		_, _ = os.Stdout.Write(bytes.Repeat([]byte{'x'}, MaxLineBytes+1))
		fmt.Println()
	case "max":
		_, _ = os.Stdout.Write(bytes.Repeat([]byte{'x'}, MaxLineBytes))
		fmt.Println()
	case "stderr-max":
		_, _ = os.Stderr.Write(bytes.Repeat([]byte{'s'}, StderrLimit))
	case "stderr-over":
		_, _ = os.Stderr.Write(bytes.Repeat([]byte{'s'}, StderrLimit+1))
		time.Sleep(10 * time.Second)
	case "stall":
		fmt.Fprintln(os.Stdout, `{"ready":true}`)
		time.Sleep(30 * time.Second)
	case "flood":
		for {
			fmt.Fprintln(os.Stdout, `{"flood":true}`)
		}
	case "close-input":
		_ = os.Stdin.Close()
		fmt.Fprintln(os.Stdout, `{"ready":true}`)
		time.Sleep(30 * time.Second)
	case "descendant", "detached":
		fixtureDescendant(mode)
	case "hold":
		time.Sleep(30 * time.Second)
	case "config":
		// Native OpenSSH config evaluation only. Explicit file suppresses ambient
		// config. -G exits without a connection or private key access.
		args := append([]string{"-G", "-F", os.Getenv("AX_SSH_CONFIG")}, os.Args[1:]...)
		cmd := exec.Command(os.Getenv("AX_SSH_BINARY"), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
	default:
		os.Exit(99)
	}
	os.Exit(0)
}

func document(extra string) string {
	return `schema = "urn:ax:schema:config"
schema_version = "3.0.0"
host_id = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
host_name = "local"
platform = "macos"
[mesh]
connect_timeout_seconds = 7
rpc_timeout_seconds = 10
[[mesh.peers]]
host_id = "` + peerID + `"
name = "peer"
endpoint = "alice@[::1]:2222"
platform = "linux"
` + extra
}
func snapshot(t *testing.T, doc string) config.Snapshot {
	t.Helper()
	files := fstest.MapFS{"config.toml": &fstest.MapFile{Data: []byte(doc), Mode: 0600}}
	in := config.Inputs{Platform: scalar.PlatformMacOS, HomeDir: "/home", TempDir: "/tmp", WorkingDir: "/",
		LookupEnv: func(string) (string, bool) { return "", false },
		Stat:      func(p string) (fs.FileInfo, error) { return fs.Stat(files, strings.TrimPrefix(p, "/")) },
		ReadFile:  func(p string) ([]byte, error) { return fs.ReadFile(files, strings.TrimPrefix(p, "/")) }}
	s, err := config.Load(in, config.Overrides{config.ConfigFile: "/config.toml"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func client(t *testing.T, mode, extra string) *Client {
	t.Helper()
	c, err := New(snapshot(t, document(extra)))
	if err != nil {
		t.Fatal(err)
	}
	c.executable, err = os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	c.env = []string{"AX_SSH_FIXTURE=" + mode, "GOCOVERDIR=" + t.TempDir(), "GORACE=atexit_sleep_ms=0"}
	return c
}
func open(t *testing.T, c *Client) *Session {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	s, err := c.Open(ctx, "peer")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
func drain(s *Session) ([][]byte, error) {
	var lines [][]byte
	for {
		line, err := s.Receive()
		if err != nil {
			return lines, err
		}
		lines = append(lines, line)
	}
}

func TestOpenStructuredCommand(t *testing.T) {
	c := client(t, "argv", `ssh_args = ["-i", "/fixture/key;$(do-not-run)", "-o", "ConnectTimeout=99"]`)
	s := open(t, c)
	lines, err := drain(s)
	if err != io.EOF || len(lines) != 1 {
		t.Fatalf("lines=%d err=%v", len(lines), err)
	}
	var args []string
	if err := json.Unmarshal(lines[0], &args); err != nil {
		t.Fatal(err)
	}
	want := []string{"-T", "-p", "2222", "-i", "/fixture/key;$(do-not-run)", "-o", "ConnectTimeout=99", "alice@::1", "ax", "rpc", "serve", "--stdio"}
	if !reflect.DeepEqual(args[len(args)-len(want):], want) {
		t.Fatalf("atomic tail=%q", args)
	}
	if s.Target().Host().ID != peerID || s.Target().CheckProtocolHost("peer") == nil {
		t.Fatal("protocol identity obligation lost")
	}
	if c.connectSeconds != 7 || c.rpcTimeout != 10*time.Second {
		t.Fatal("config timeout not composed")
	}
}

func TestOpenRefusesBeforeStart(t *testing.T) {
	c := client(t, "argv", "")
	for _, selector := range []string{"", "alice@[::1]:2222", "discovered"} {
		s, err := c.Open(context.Background(), selector)
		if s != nil || !errors.Is(err, peeridentity.ErrNotAllowlisted) {
			t.Fatalf("%q: %v", selector, err)
		}
	}
	if _, err := New(config.Snapshot{}); !errors.Is(err, peeridentity.ErrConfigurationRequired) {
		t.Fatal(err)
	}
	var zero Client
	if _, err := zero.Open(context.Background(), "peer"); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Open(ctx, "peer"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	c.executable = filepath.Join(t.TempDir(), "missing-sensitive-executable")
	if s, err := c.Open(context.Background(), "peer"); s != nil || !errors.Is(err, ErrStart) || strings.Contains(err.Error(), "sensitive") {
		t.Fatalf("start=%v", err)
	}
}

func TestDuplexAndSendBoundaries(t *testing.T) {
	s := open(t, client(t, "echo", ""))
	want := bytes.Repeat([]byte{'x'}, MaxLineBytes)
	sent := make(chan error, 1)
	go func() { sent <- s.Send(want) }()
	got, err := s.Receive()
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("max receive=%d %v", len(got), err)
	}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	if err := s.Send([]byte(`{"second":true}`)); err != nil {
		t.Fatal(err)
	}
	got, err = s.Receive()
	if err != nil || string(got) != `{"second":true}` {
		t.Fatal(err)
	}
	if err := s.CloseWrite(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Receive(); err != io.EOF {
		t.Fatal(err)
	}
	if err := s.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.Send([]byte("late")); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	for name, line := range map[string][]byte{"empty": {}, "oversize": bytes.Repeat([]byte{'x'}, MaxLineBytes+1), "newline": []byte("{}\n{}")} {
		t.Run(name, func(t *testing.T) {
			// A sink cannot mask a missing send gate with a receive refusal.
			s := open(t, client(t, "sink", ""))
			if err := s.Send(line); !errors.Is(err, ErrLine) {
				t.Fatal(err)
			}
			if !errors.Is(s.Wait(), ErrLine) {
				t.Fatal("session was not terminated")
			}
		})
	}
}

func TestReceiveFailureAndLimits(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want error
		code int
	}{
		{"empty", io.EOF, 0}, {"max", io.EOF, 0}, {"stderr-max", io.EOF, 0},
		{"partial", ErrLine, 0}, {"blank", ErrLine, 0}, {"oversize", ErrLine, 0}, {"stderr-over", ErrStderrLimit, 0},
		{"exit255", ErrExit, 255}, {"exit7", ErrExit, 7},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			s := open(t, client(t, tc.mode, ""))
			lines, err := drain(s)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
			if tc.mode == "max" && (len(lines) != 1 || len(lines[0]) != MaxLineBytes) {
				t.Fatal("max line lost")
			}
			if tc.code != 0 {
				var failure *Failure
				if !errors.As(err, &failure) || failure.ExitCode != tc.code {
					t.Fatalf("exit=%#v", err)
				}
			}
			if strings.Contains(fmt.Sprint(err), "credential-canary") {
				t.Fatal("diagnostic disclosure")
			}
		})
	}
}

func TestCancellationAndCleanup(t *testing.T) {
	for _, mode := range []string{"stall", "flood"} {
		t.Run(mode, func(t *testing.T) {
			c := client(t, mode, "")
			ctx, cancel := context.WithCancel(context.Background())
			s, err := c.Open(ctx, "peer")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = s.Close() })
			if _, err = s.Receive(); err != nil {
				t.Fatal(err)
			}
			cancel()
			if err = s.Wait(); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if s.cmd.ProcessState == nil {
				t.Fatal("direct child not reaped")
			}
			if _, err = s.Receive(); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if err = s.Close(); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	t.Run("blocked-write", func(t *testing.T) {
		c := client(t, "stall", "")
		ctx, cancel := context.WithCancel(context.Background())
		s, err := c.Open(ctx, "peer")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		if _, err = s.Receive(); err != nil {
			t.Fatal(err)
		}
		sent := make(chan error, 1)
		go func() { sent <- s.Send(bytes.Repeat([]byte{'x'}, MaxLineBytes)) }()
		cancel()
		if err = <-sent; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if s.cmd.ProcessState == nil {
			t.Fatal("not reaped")
		}
	})
	t.Run("deadline", func(t *testing.T) {
		c := client(t, "stall", "")
		c.rpcTimeout = 50 * time.Millisecond
		before := time.Now()
		s := open(t, c)
		after := time.Now()
		deadline, ok := s.ctx.Deadline()
		if !ok || deadline.Before(before.Add(c.rpcTimeout)) || deadline.After(after.Add(c.rpcTimeout)) {
			t.Fatal("configured RPC lifetime not applied to the process context")
		}
		if !errors.Is(s.Wait(), context.DeadlineExceeded) {
			t.Fatal("timeout lost")
		}
	})
	t.Run("closed-input", func(t *testing.T) {
		s := open(t, client(t, "close-input", ""))
		if _, err := s.Receive(); err != nil {
			t.Fatal(err)
		}
		if err := s.Send(bytes.Repeat([]byte{'x'}, MaxLineBytes)); !errors.Is(err, ErrStream) {
			t.Fatal(err)
		}
	})
	t.Run("explicit-close", func(t *testing.T) {
		s := open(t, client(t, "stall", ""))
		if err := s.Close(); !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
		if err := s.Close(); !errors.Is(err, ErrClosed) {
			t.Fatal(err)
		}
	})
}

func TestOpenNativeSSHPolicy(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH unavailable; native effective-policy evaluation unverified")
	}
	c := client(t, "config", `ssh_args = ["-o", "ConnectTimeout=99"]`)
	file := filepath.Join(t.TempDir(), "ssh_config")
	hostile := `Host *
 StrictHostKeyChecking no
 NoHostAuthenticationForLocalhost yes
 VerifyHostKeyDNS yes
 UpdateHostKeys yes
 ControlMaster auto
 ControlPath /fixture/control
 ControlPersist yes
 ForwardAgent yes
 ForwardX11 yes
 Tunnel point-to-point
 LocalForward 127.0.0.1:9999 127.0.0.1:9998
 RemoteForward 127.0.0.1:9997 127.0.0.1:9996
 PermitLocalCommand yes
 LocalCommand false
 RemoteCommand false
 RequestTTY force
 SessionType none
 ForkAfterAuthentication yes
 StdinNull yes
 BatchMode no
 NumberOfPasswordPrompts 4
 ConnectionAttempts 5
 IdentityFile none
 UserKnownHostsFile /fixture/known_hosts
 GlobalKnownHostsFile /fixture/global_known_hosts
`
	if err = os.WriteFile(file, []byte(hostile), 0600); err != nil {
		t.Fatal(err)
	}
	c.env = append(c.env, "AX_SSH_CONFIG="+file, "AX_SSH_BINARY="+ssh)
	s := open(t, c)
	lines, err := drain(s)
	if err != io.EOF {
		t.Fatal(err)
	}
	effective := map[string]string{}
	for _, line := range lines {
		key, value, ok := strings.Cut(string(line), " ")
		if ok {
			effective[key] = value
		}
	}
	want := map[string]string{"stricthostkeychecking": "true", "nohostauthenticationforlocalhost": "no", "verifyhostkeydns": "false", "updatehostkeys": "false", "controlmaster": "false", "controlpersist": "no", "clearallforwardings": "yes", "forwardagent": "no", "forwardx11": "no", "tunnel": "false", "permitlocalcommand": "no", "requesttty": "false", "sessiontype": "default", "forkafterauthentication": "no", "stdinnull": "no", "batchmode": "yes", "numberofpasswordprompts": "0", "connectionattempts": "1", "connecttimeout": "7", "hostname": "::1", "user": "alice", "port": "2222"}
	for key, value := range want {
		if effective[key] != value {
			t.Errorf("%s=%q want=%q", key, effective[key], value)
		}
	}
	for _, key := range []string{"localforward", "remoteforward", "controlpath", "remotecommand"} {
		if _, ok := effective[key]; ok {
			t.Errorf("unexpected active %s=%q", key, effective[key])
		}
	}
}

func TestOpenComposedArgvBound(t *testing.T) {
	// Vary one legal IdentityFile value. The additional fixed policy must be
	// counted, even though the accepted peer target itself still fits the bound.
	base := make([]string, 0)
	for i := 0; i < 15; i++ {
		base = append(base, "-i", strings.Repeat("a", 4096))
	}
	for _, size := range []int{65536, 65537} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			probe := append(append([]string{}, base...), "-i", "a")
			b, _ := json.Marshal(probe)
			c := client(t, "argv", "ssh_args = "+string(b))
			target, _ := c.directory.Resolve("peer")
			argv, err := c.argv(target)
			if err != nil {
				t.Fatal(err)
			}
			n := 0
			for _, a := range argv {
				n += len(a)
			}
			probe[len(probe)-1] = strings.Repeat("a", 1+size-n)
			b, _ = json.Marshal(probe)
			c = client(t, "empty", "ssh_args = "+string(b))
			s, err := c.Open(context.Background(), "peer")
			if size == 65537 {
				if s != nil || !errors.Is(err, peeridentity.ErrSSHArgvTooLarge) {
					t.Fatalf("past bound: %v", err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = s.Close() })
				if _, err = drain(s); err != io.EOF {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestOpenConfigurationCompatibility(t *testing.T) {
	for _, version := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		t.Run(version, func(t *testing.T) {
			c, err := New(snapshot(t, strings.Replace(document(""), `schema_version = "3.0.0"`, `schema_version = "`+version+`"`, 1)))
			if err != nil {
				t.Fatal(err)
			}
			fixture := client(t, "argv", "")
			c.executable, c.env = fixture.executable, fixture.env
			s := open(t, c)
			lines, err := drain(s)
			if err != io.EOF || len(lines) != 1 {
				t.Fatalf("%s: %v", version, err)
			}
			var args []string
			if json.Unmarshal(lines[0], &args) != nil {
				t.Fatal("argv unavailable")
			}
			if !reflect.DeepEqual(args[len(args)-4:], []string{"ax", "rpc", "serve", "--stdio"}) {
				t.Fatal("fixed command drift")
			}
			if c.rpcTimeout != 10*time.Second || c.connectSeconds != 7 {
				t.Fatal("legacy timeout drift")
			}
		})
	}
}
