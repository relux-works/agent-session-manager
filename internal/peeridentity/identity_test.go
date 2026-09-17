package peeridentity_test

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/relux-works/agent-session-manager/internal/config"
	"github.com/relux-works/agent-session-manager/internal/peeridentity"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/specdoc"
)

const localID = "0198f4c8-4a10-7b22-8b3c-1234567890ab"
const peerID = "0198f4c8-7d40-7e55-8e6f-1234567890ab"
const otherID = "0198f4c8-a070-7188-9172-1234567890ab"

func document(version string) string {
	return fmt.Sprintf(`schema = "urn:ax:schema:config"
schema_version = %q
host_id = %q
host_name = "local"
platform = "macos"
`, version, localID) + peer(peerID, "workstation", "ivan@workstation.example")
}
func peer(id, name, endpoint string) string {
	return fmt.Sprintf(`
[[mesh.peers]]
host_id = %q
name = %q
endpoint = %q
platform = "linux"
`, id, name, endpoint)
}
func inputs(doc string) config.Inputs {
	files := fstest.MapFS{"config.toml": &fstest.MapFile{Data: []byte(doc), Mode: 0600}}
	return config.Inputs{Platform: scalar.PlatformMacOS, HomeDir: "/home", TempDir: "/tmp", WorkingDir: "/",
		LookupEnv: func(string) (string, bool) { return "", false },
		Stat:      func(path string) (fs.FileInfo, error) { return fs.Stat(files, strings.TrimPrefix(path, "/")) },
		ReadFile:  func(path string) ([]byte, error) { return fs.ReadFile(files, strings.TrimPrefix(path, "/")) },
	}
}
func load(doc string) (peeridentity.Directory, error) {
	return peeridentity.Load(inputs(doc), config.Overrides{config.ConfigFile: "/config.toml"})
}
func requireLoad(t *testing.T, doc string) peeridentity.Directory {
	t.Helper()
	d, err := load(doc)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func requireTarget(t *testing.T, d peeridentity.Directory, selector string) peeridentity.Target {
	t.Helper()
	target, err := d.Resolve(selector)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func TestLoadPeerIdentityVersions(t *testing.T) {
	for _, version := range []string{config.Version1, config.Version2, config.CurrentVersion} {
		t.Run(version, func(t *testing.T) {
			d := requireLoad(t, document(version))
			if got := d.Local(); got.ID != localID || got.Name != "local" || got.Platform != scalar.PlatformMacOS {
				t.Fatalf("local = %+v", got)
			}
			for _, selector := range []string{peerID, "workstation"} {
				target := requireTarget(t, d, selector)
				if got := target.Host(); got.ID != peerID || got.Name != "workstation" || got.Platform != scalar.PlatformLinux {
					t.Fatalf("host = %+v", got)
				}
				if target.KeyProvenance() != "external_ssh" {
					t.Fatal("key authority must remain external SSH")
				}
				argv, err := target.RPCArgv()
				want := []string{"-T", "ivan@workstation.example", "ax", "rpc", "serve", "--stdio"}
				if err != nil || !reflect.DeepEqual(argv, want) {
					t.Fatalf("argv = %q, %v", argv, err)
				}
				if err := target.CheckProtocolHost(peerID); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestLoadIdentityRefusals(t *testing.T) {
	base := document(config.CurrentVersion)
	for _, tc := range []struct{ name, doc, clause string }{
		{"malformed local UUID", strings.Replace(base, localID, "bad-local-secret", 1), "host_id"},
		{"wrong UUID version", strings.Replace(base, peerID, strings.Replace(peerID, "7e55", "4e55", 1), 1), "mesh.peers[0].host_id"},
		{"uppercase UUID", strings.Replace(base, peerID, strings.ToUpper(peerID), 1), "mesh.peers[0].host_id"},
		{"empty peer ID", strings.Replace(base, peerID, "", 1), "mesh.peers[0].host_id"},
		{"missing peer ID", strings.Replace(base, fmt.Sprintf("host_id = %q", peerID), "", 1), "mesh.peers[0] required member"},
		{"local identity reused", strings.Replace(base, peerID, localID, 1), "mesh.peers duplicate host_id"},
		{"duplicate identity", base + peer(peerID, "different alias", "elsewhere.example"), "mesh.peers duplicate host_id"},
		{"duplicate alias", base + peer(otherID, "workstation", "elsewhere.example"), "mesh.peers duplicate name"},
		{"empty alias", strings.Replace(base, `name = "workstation"`, `name = ""`, 1), "mesh.peers[0].name"},
		{"long alias", strings.Replace(base, `name = "workstation"`, fmt.Sprintf("name = %q", strings.Repeat("界", 65)), 1), "mesh.peers[0].name"},
		{"control alias", strings.Replace(base, `name = "workstation"`, `name = "bad\u001bname"`, 1), "mesh.peers[0].name"},
		{"bad peer platform", strings.Replace(base, `platform = "linux"`, `platform = "ios"`, 1), "mesh.peers[0].platform"},
		{"empty endpoint", strings.Replace(base, "ivan@workstation.example", "", 1), "mesh.peers[0].endpoint"},
		{"missing endpoint", strings.Replace(base, `endpoint = "ivan@workstation.example"`, "", 1), "mesh.peers[0] required member"},
		{"option endpoint", strings.Replace(base, "ivan@workstation.example", "-oStrictHostKeyChecking=no", 1), "mesh.peers[0].endpoint option-like value"},
		{"credential endpoint", strings.Replace(base, "ivan@workstation.example", "ivan:private-secret@workstation.example", 1), "mesh.peers[0].endpoint user"},
		{"invalid port", strings.Replace(base, "ivan@workstation.example", "host.example:65536", 1), "mesh.peers[0].endpoint port"},
		{"host key bypass", base + `ssh_args = ["-o", "StrictHostKeyChecking=no"]`, "mesh.peers[0].ssh_args host authentication bypass"},
		{"grouped host key bypass", base + `ssh_args = ["-voStrictHostKeyChecking=no"]`, "mesh.peers[0].ssh_args host authentication bypass"},
		{"known hosts bypass", base + `ssh_args = ["-o", "UserKnownHostsFile=/dev/null"]`, "mesh.peers[0].ssh_args host authentication bypass"},
		{"proxy bypass", base + `ssh_args = ["-o", "ProxyCommand=private-secret"]`, "mesh.peers[0].ssh_args unpermitted option"},
		{"malformed optional args", base + `ssh_args = ""`, "closed TOML shape"},
		{"invented key provenance", base + `key_provenance = "verified"`, "closed TOML shape"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, err := load(tc.doc)
			if err == nil {
				t.Fatal("invalid identity admitted")
			}
			var de *config.DocumentError
			if !errors.As(err, &de) || !strings.Contains(de.Clause, tc.clause) {
				t.Fatalf("wrong refusal: %v (%+v)", err, de)
			}
			if _, err := d.Resolve("workstation"); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
				t.Fatal("failed load returned usable peer")
			}
			if strings.Contains(fmt.Sprint(err), "private-secret") {
				t.Fatal("error leaked secret")
			}
		})
	}
}

func TestResolveAllowlistAndAliasAmbiguity(t *testing.T) {
	d := requireLoad(t, document(config.CurrentVersion)+peer(otherID, peerID, "other.example"))
	if _, err := d.Resolve(peerID); !errors.Is(err, peeridentity.ErrAmbiguousAlias) {
		t.Fatalf("ambiguous alias accepted: %v", err)
	}
	if requireTarget(t, d, "workstation").Host().ID != peerID {
		t.Fatal("unambiguous alias lost identity")
	}
	if requireTarget(t, d, otherID).Host().ID != otherID {
		t.Fatal("unambiguous stable identity lost")
	}
	for _, selector := range []string{"", localID, "local", "Workstation", "workstation.example", "discovered.example", "workstation "} {
		if _, err := d.Resolve(selector); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
			t.Fatalf("unlisted selector admitted: %v", err)
		}
	}
	for _, name := range []string{"界", strings.Repeat("界", 64), "Workstation"} {
		d := requireLoad(t, document(config.CurrentVersion)+peer(otherID, name, "other.example"))
		if requireTarget(t, d, name).Host().Name != name {
			t.Fatal("alias changed")
		}
	}
}

func TestProtocolIdentityRefusal(t *testing.T) {
	d := requireLoad(t, document(config.CurrentVersion)+peer(otherID, "other", "other.example"))
	target := requireTarget(t, d, peerID)
	for _, id := range []string{"", localID, otherID, "workstation", "ivan@workstation.example", "private-secret", strings.ToUpper(peerID)} {
		err := target.CheckProtocolHost(id)
		if !errors.Is(err, peeridentity.ErrHostIdentityMismatch) {
			t.Fatal("protocol identity not bound to target")
		}
		if strings.Contains(err.Error(), id) && id != "" {
			t.Fatal("identity leaked in refusal")
		}
	}
	var zero peeridentity.Target
	if err := zero.CheckProtocolHost(peerID); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
		t.Fatal("zero target admitted")
	}
	if _, err := zero.RPCArgv(); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
		t.Fatal("zero target produced argv")
	}
	if zero.KeyProvenance() != "" {
		t.Fatal("zero target claims provenance")
	}
}

func TestSSHTargetAtomicArgv(t *testing.T) {
	for _, tc := range []struct {
		endpoint    string
		destination string
		port        string
	}{
		{"host.example", "host.example", ""}, {"u@host.example:1", "u@host.example", "1"},
		{"host.example:65535", "host.example", "65535"}, {"[2001:db8::1]", "2001:db8::1", ""},
		{"u@[2001:db8::1]:2222", "u@2001:db8::1", "2222"},
	} {
		t.Run(tc.endpoint, func(t *testing.T) {
			doc := strings.Replace(document(config.CurrentVersion), "ivan@workstation.example", tc.endpoint, 1) + `ssh_args = ["-o", "BatchMode=yes", "-i", "/fixture/private-selector", "-p", "22"]`
			target := requireTarget(t, requireLoad(t, doc), peerID)
			want := []string{"-T"}
			if tc.port != "" {
				want = append(want, "-p", tc.port)
			}
			want = append(want, "-o", "BatchMode=yes", "-i", "/fixture/private-selector", "-p", "22", tc.destination, "ax", "rpc", "serve", "--stdio")
			got, err := target.RPCArgv()
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("argv %q, %v", got, err)
			}
			got[0] = "mutated"
			again, _ := target.RPCArgv()
			if !reflect.DeepEqual(again, want) {
				t.Fatal("argv exposed shared state")
			}
			for _, format := range []string{"%v", "%+v", "%#v", "%s", "%q"} {
				rendered := fmt.Sprintf(format, target)
				if strings.Contains(rendered, "private-selector") || strings.Contains(rendered, tc.endpoint) {
					t.Fatal("target formatting leaked execution inputs")
				}
			}
		})
	}
}

func disclosure(id, policy string) string {
	return fmt.Sprintf(`
[[directory_peer_disclosure]]
host_id = %q
environment_observations = %q
native_observations = %q
manual_metadata = %q
generated_metadata = %q
job_operation_status = %q
extensions = {}
`, id, policy, policy, policy, policy, policy)
}

var classes = []string{"environment_observations", "native_observations", "manual_metadata", "generated_metadata", "job_operation_status"}

func TestDisclosurePolicyRefusals(t *testing.T) {
	base := document(config.CurrentVersion)
	for _, policy := range []string{"mesh_sanitized", "reference_only"} {
		d := requireLoad(t, base+"\n[directory]\ngenerated_summary_upgrade_choice = \"local_only\"\n"+disclosure(peerID, policy))
		for _, class := range classes {
			got, err := d.DisclosurePolicy(peerID, class)
			if err != nil || got != policy {
				t.Fatalf("%s policy=%s error=%v", class, got, err)
			}
		}
	}
	for _, tc := range []struct {
		name, doc, selector, class string
		want                       error
	}{
		{"default local only", base, peerID, "manual_metadata", peeridentity.ErrDisclosureDenied},
		{"explicit local only", base + "\n[directory]\ndefault_metadata_policy = \"mesh_sanitized\"\n" + disclosure(peerID, "local_only"), peerID, "manual_metadata", peeridentity.ErrDisclosureDenied},
		{"unset summary choice", base + disclosure(peerID, "mesh_sanitized"), peerID, "generated_metadata", peeridentity.ErrDisclosureDenied},
		{"legacy version", document(config.Version1), peerID, "manual_metadata", peeridentity.ErrDisclosureDenied},
		{"disclosure does not authorize", base + disclosure(otherID, "mesh_sanitized"), otherID, "manual_metadata", peeridentity.ErrNotAllowlisted},
		{"raw excerpts", base + disclosure(peerID, "mesh_sanitized"), peerID, "raw_excerpts", peeridentity.ErrDisclosureDenied},
		{"embeddings", base + disclosure(peerID, "mesh_sanitized"), peerID, "embeddings", peeridentity.ErrDisclosureDenied},
		{"model payloads", base + disclosure(peerID, "mesh_sanitized"), peerID, "model_payloads", peeridentity.ErrDisclosureDenied},
		{"runtime paths", base + disclosure(peerID, "mesh_sanitized"), peerID, "runtime_paths", peeridentity.ErrDisclosureDenied},
		{"auth detail", base + disclosure(peerID, "mesh_sanitized"), peerID, "auth_status_details", peeridentity.ErrDisclosureDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := requireLoad(t, tc.doc)
			got, err := d.DisclosurePolicy(tc.selector, tc.class)
			if !errors.Is(err, tc.want) || got != "" {
				t.Fatalf("disclosure admitted: %q %v", got, err)
			}
		})
	}
	d := requireLoad(t, base+"\n[directory]\ndefault_metadata_policy = \"reference_only\"\n")
	if p, err := d.DisclosurePolicy("workstation", "manual_metadata"); p != "reference_only" || err != nil {
		t.Fatal("default lost")
	}
	// The class gate must refuse raw material even with a permissive default;
	// local_only must not accidentally be the reason a raw-class test passes.
	for _, class := range []string{"raw_excerpts", "embeddings", "model_payloads", "auth_status_details", "runtime_paths", ""} {
		if _, err := d.DisclosurePolicy(peerID, class); !errors.Is(err, peeridentity.ErrDisclosureDenied) {
			t.Fatal("unknown class escaped through permissive default")
		}
	}
	for _, entry := range []string{
		disclosure(peerID, "raw"), disclosure(peerID, "mesh_sanitized") + disclosure(peerID, "reference_only"),
		strings.Replace(disclosure(peerID, "mesh_sanitized"), `manual_metadata = "mesh_sanitized"`, "", 1),
		disclosure(peerID, "mesh_sanitized") + "raw_excerpts = true\n",
	} {
		if _, err := load(base + entry); err == nil {
			t.Fatal("malformed disclosure accepted")
		}
	}
}

func TestLoadAbsenceReadFailureAndRecovery(t *testing.T) {
	doc := document(config.CurrentVersion)
	good := inputs(doc)
	for _, kind := range []string{"missing", "stat failure", "partial read", "empty", "malformed"} {
		t.Run(kind, func(t *testing.T) {
			in := inputs(doc)
			switch kind {
			case "missing":
				in.Stat = func(path string) (fs.FileInfo, error) { return fs.Stat(fstest.MapFS{}, strings.TrimPrefix(path, "/")) }
				// The selected parent exists, while config and infrastructure roots do not.
				in.Stat = func(path string) (fs.FileInfo, error) {
					if path == "/" {
						return fs.Stat(fstest.MapFS{}, ".")
					}
					return nil, fs.ErrNotExist
				}
			case "stat failure":
				in.Stat = func(string) (fs.FileInfo, error) { return nil, errors.New("private-secret") }
			case "partial read":
				in.ReadFile = func(string) ([]byte, error) { return []byte(doc), errors.New("private-secret") }
			case "empty":
				in.ReadFile = func(string) ([]byte, error) { return []byte{}, nil }
			case "malformed":
				in.ReadFile = func(string) ([]byte, error) { return []byte("host_id ="), nil }
			}
			d, err := peeridentity.Load(in, config.Overrides{config.ConfigFile: "/config.toml"})
			if err == nil {
				t.Fatal("missing/failed configuration admitted")
			}
			if kind == "missing" && !errors.Is(err, peeridentity.ErrConfigurationRequired) {
				t.Fatalf("absence: %v", err)
			}
			if kind != "missing" && errors.Is(err, peeridentity.ErrConfigurationRequired) {
				t.Fatal("read failure treated as absence")
			}
			if strings.Contains(fmt.Sprintf("%+v", err), "private-secret") {
				t.Fatal("read error leaked secret")
			}
			if _, err := d.Resolve(peerID); !errors.Is(err, peeridentity.ErrNotAllowlisted) {
				t.Fatal("failed load authorized")
			}
			recovered, err := peeridentity.Load(good, config.Overrides{config.ConfigFile: "/config.toml"})
			if err != nil {
				t.Fatal(err)
			}
			requireTarget(t, recovered, peerID)
		})
	}
	if _, err := peeridentity.FromSnapshot(config.Snapshot{}); !errors.Is(err, peeridentity.ErrConfigurationRequired) {
		t.Fatal("zero snapshot admitted")
	}
	// Key provenance never inspects a private key: the only read is selected TOML.
	reads := 0
	in := inputs(doc + `ssh_args = ["-i", "/fixture/unreadable-key"]`)
	reader := in.ReadFile
	in.ReadFile = func(path string) ([]byte, error) {
		reads++
		if path != "/config.toml" {
			t.Fatal("unexpected credential read")
		}
		return reader(path)
	}
	d, err := peeridentity.Load(in, config.Overrides{config.ConfigFile: "/config.toml"})
	if err != nil {
		t.Fatal(err)
	}
	target := requireTarget(t, d, peerID)
	if target.KeyProvenance() != "external_ssh" || reads != 1 {
		t.Fatal("incorrect key provenance")
	}
}

func TestSnapshotIsolationAndDisclosureBinding(t *testing.T) {
	doc := document(config.CurrentVersion) + peer(otherID, "other", "other.example") + `ssh_args = ["-i", "/fixture/private-selector"]` + "\n[directory]\ngenerated_summary_upgrade_choice = \"mesh_sanitized\"\n" + disclosure(otherID, "local_only") + disclosure(peerID, "mesh_sanitized")
	snapshot, err := config.Load(inputs(doc), config.Overrides{config.ConfigFile: "/config.toml"})
	if err != nil {
		t.Fatal(err)
	}
	loaded, _ := snapshot.Configuration()
	loaded.Value.Mesh.Peers[0].HostID = otherID
	bytes := snapshot.Document()
	bytes[0] = '!'
	d, err := peeridentity.FromSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if requireTarget(t, d, "workstation").Host().ID != peerID {
		t.Fatal("snapshot mutated")
	}
	for _, class := range classes {
		if p, err := d.DisclosurePolicy(peerID, class); err != nil || p != "mesh_sanitized" {
			t.Fatal("policy bound to wrong peer")
		}
		if _, err := d.DisclosurePolicy(otherID, class); !errors.Is(err, peeridentity.ErrDisclosureDenied) {
			t.Fatal("other peer policy escaped")
		}
	}
	if strings.Contains(fmt.Sprintf("%#v", d), "private-selector") {
		t.Fatal("directory formatting leaked key selector")
	}
	// One peer matching by both its name and ID is still one identity.
	same := requireLoad(t, strings.Replace(document(config.Version1), `name = "workstation"`, fmt.Sprintf("name = %q", peerID), 1))
	requireTarget(t, same, peerID)
}

// Drive the actual pinned TOML fixture; this must not become a hand-maintained
// approximation of the configuration example.
func TestLoadPinnedPeerConfigurationExample(t *testing.T) {
	spec, err := specdoc.Load()
	if err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	inExample, inTOML := false, false
	for line := 1; line <= spec.LineCount(); line++ {
		text, _ := spec.Line(line)
		if text == "### 6.2 Normative example" {
			inExample = true
			continue
		}
		if !inExample {
			continue
		}
		if text == "~~~toml" {
			inTOML = true
			continue
		}
		if inTOML && text == "~~~" {
			break
		}
		if inTOML {
			body.WriteString(text)
			body.WriteByte('\n')
		}
	}
	if body.Len() == 0 {
		t.Fatal("pinned TOML example absent")
	}
	d := requireLoad(t, body.String())
	target := requireTarget(t, d, "workstation")
	argv, err := target.RPCArgv()
	want := []string{"-T", "-o", "BatchMode=yes", "ivan@workstation.tailnet.ts.net", "ax", "rpc", "serve", "--stdio"}
	if err != nil || !reflect.DeepEqual(argv, want) {
		t.Fatalf("pinned SSH target: %q, %v", argv, err)
	}
	if err := target.CheckProtocolHost(peerID); err != nil {
		t.Fatal(err)
	}
}

func TestSSHTotalArgvByteBound(t *testing.T) {
	baseline := requireTarget(t, requireLoad(t, document(config.CurrentVersion)), peerID)
	argv, _ := baseline.RPCArgv()
	fixed := 0
	for _, arg := range argv {
		fixed += len(arg)
	}
	for _, total := range []int{65_536, 65_537} {
		remaining := total - fixed
		var quoted []string
		for remaining > 0 {
			size := remaining - 2
			if size > 4096 {
				size = 4096
			}
			if remaining-size-2 > 0 && remaining-size-2 < 3 {
				size -= 3
			}
			quoted = append(quoted, fmt.Sprintf("%q", "-i"), fmt.Sprintf("%q", strings.Repeat("a", size)))
			remaining -= size + 2
		}
		doc := document(config.CurrentVersion) + "ssh_args = [" + strings.Join(quoted, ",") + "]"
		target := requireTarget(t, requireLoad(t, doc), peerID)
		got, err := target.RPCArgv()
		if total == 65_536 {
			if err != nil {
				t.Fatal(err)
			}
			size := 0
			for _, arg := range got {
				size += len(arg)
			}
			if size != total {
				t.Fatalf("argv bytes %d", size)
			}
		} else if !errors.Is(err, peeridentity.ErrSSHArgvTooLarge) || got != nil {
			t.Fatal("oversized composed argv admitted")
		}
	}
}

func TestDisclosureClassPolicyBinding(t *testing.T) {
	for _, allowed := range classes {
		t.Run(allowed, func(t *testing.T) {
			entry := disclosure(peerID, "local_only")
			entry = strings.Replace(entry, allowed+` = "local_only"`, allowed+` = "mesh_sanitized"`, 1)
			d := requireLoad(t, document(config.CurrentVersion)+"\n[directory]\ngenerated_summary_upgrade_choice = \"mesh_sanitized\"\n"+entry)
			for _, class := range classes {
				policy, err := d.DisclosurePolicy(peerID, class)
				if class == allowed {
					if err != nil || policy != "mesh_sanitized" {
						t.Fatal("class lost its own policy")
					}
				} else if !errors.Is(err, peeridentity.ErrDisclosureDenied) || policy != "" {
					t.Fatal("class borrowed another class policy")
				}
			}
		})
	}
}
