package clonesnap

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/relux-works/agent-session-manager/internal/clonebundle"
	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// This file owns the contained capture walk over a provider store
// directory. Enumeration observes names only: every payload open
// descends from the verified store-root handle through the landed
// secprim Guard, so a symlinked intermediate component or a trailing
// symlink refuses before any payload byte is read, and a FIFO,
// directory, device, or socket refuses on its opened shape. Every
// refusal names the member. The walk never escapes the store root:
// nothing enumerated is opened except through the Guard on a
// validated member name.
//
// The walk reads every included-class member twice — once for the
// capture bytes and the pre digest, once for the post digest — and
// enumerates the store twice. Any difference is a detected source
// race, decided before any manifest is sealed. Excluded-class
// members are never opened: their bytes cannot reach a blob, a
// manifest, a log line, or an error string, because no read ever
// observes them.

// PlanItem is one capture-plan candidate: the native key the walk
// must reconcile and the closed class that decides its disposition.
// Required candidates must be present when their class is included;
// optional included-class candidates absent from the store seal as
// excluded rows instead of refusing.
type PlanItem struct {
	NativeKey string
	Class     string
	Required  bool
}

// memberKind is the observed shape of one enumerated store member.
type memberKind string

const (
	memberFile  memberKind = "file"
	memberDir   memberKind = "dir"
	memberOther memberKind = "other"
)

// observedMember is one enumerated store member: its slash-separated
// key beneath the store root, its observed shape, and its observed
// size. Observation never opens a payload.
type observedMember struct {
	key  string
	kind memberKind
	size uint64
}

// capturedMember is one walked included-class member: the bytes read
// through the Guard and their sealed descriptor.
type capturedMember struct {
	key        string
	class      string
	payload    []byte
	descriptor BlobDescriptor
}

// enumerateStore lists every store member beneath root as
// slash-separated keys. It observes names, shapes, and sizes only;
// it opens no payload. A failed observation is a refusal, never an
// absence: a member that cannot be listed cannot be captured.
func enumerateStore(root string) ([]observedMember, error) {
	var members []observedMember
	var descend func(relative string) error
	descend = func(relative string) error {
		directory := root
		if relative != "" {
			directory = filepath.Join(root, filepath.FromSlash(relative))
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			return invalid("capture cannot enumerate store member %q: %v", relative, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if name == "" || name == "." || name == ".." || strings.Contains(name, "/") || strings.Contains(name, `\`) {
				return invalid("capture store member %q is not a plain file name", name)
			}
			key := name
			if relative != "" {
				key = relative + "/" + name
			}
			info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(key)))
			if err != nil {
				return invalid("capture cannot stat store member %q: %v", key, err)
			}
			mode := info.Mode()
			switch {
			case mode.IsRegular():
				if info.Size() < 0 {
					return invalid("capture store member %q has a negative size", key)
				}
				members = append(members, observedMember{key: key, kind: memberFile, size: uint64(info.Size())})
			case mode.IsDir():
				members = append(members, observedMember{key: key, kind: memberDir})
				if err := descend(key); err != nil {
					return err
				}
			default:
				members = append(members, observedMember{key: key, kind: memberOther})
			}
		}
		return nil
	}
	if err := descend(""); err != nil {
		return nil, err
	}
	sort.Slice(members, func(left, right int) bool { return members[left].key < members[right].key })
	return members, nil
}

// checkPlan validates the whole capture plan before anything is
// opened or installed: every key is sanitized, every class is in the
// closed nine-class vocabulary, keys are sorted unique, and every
// member name passes the containment grammar. A plan this gate
// refuses leaves the store and the object sink untouched.
func checkPlan(platform scalar.Platform, plan []PlanItem) (map[string]PlanItem, error) {
	byKey := make(map[string]PlanItem, len(plan))
	previous := ""
	for index, item := range plan {
		if index > 0 && item.NativeKey <= previous {
			return nil, invalid("capture plan keys are not sorted unique by native item key")
		}
		previous = item.NativeKey
		if err := clonebundle.SanitizeNativeKey(item.NativeKey); err != nil {
			return nil, invalid("capture plan key %q: %v", item.NativeKey, err)
		}
		if err := secprim.CheckMemberPath(platform, item.NativeKey); err != nil {
			return nil, invalid("capture plan key %q escapes the store root: %v", item.NativeKey, err)
		}
		if !clonebundle.ValidCaptureClass(item.Class) {
			return nil, invalid("capture plan key %q carries class %q outside the nine-class vocabulary", item.NativeKey, item.Class)
		}
		byKey[item.NativeKey] = item
	}
	return byKey, nil
}

// excludedClass reports whether the class is always excluded:
// credential, machine-auth, runtime-state, and transient-lock rows
// are never included and their bytes are never opened.
func excludedClass(class string) bool {
	switch class {
	case "credential", "machine_auth", "runtime_state", "transient_lock":
		return true
	default:
		return false
	}
}

// exclusionReason is the stable exclusion reason sealed for an
// always-excluded class.
func exclusionReason(class string) string {
	return class + "_excluded"
}

// openMember opens one planned member through the store Guard and
// returns its bytes. The Guard refuses symlink escapes before any
// payload byte is read and refuses non-regular shapes on the opened
// descriptor; this wrapper names the member on every failure. The
// limit refuses oversized members with capability_unavailable before
// their bytes are read, so no manifest is ever published over a
// member this package cannot carry.
func openMember(guard secprim.Guard, root *os.File, member string, limit uint64) ([]byte, error) {
	file, err := guard.Open(root, member)
	if err != nil {
		return nil, invalid("capture member %q: %v", member, err)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, invalid("capture member %q cannot be stated: %v", member, err)
	}
	if info.Size() < 0 {
		return nil, invalid("capture member %q has a negative size", member)
	}
	if uint64(info.Size()) > limit {
		return nil, invalid("capture member %q of %d bytes exceeds the %d byte bound: capability_unavailable", member, info.Size(), limit)
	}
	payload, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, invalid("capture member %q cannot be read: %v", member, err)
	}
	if uint64(len(payload)) != uint64(info.Size()) {
		return nil, invalid("capture member %q changed size during its read: source_not_quiescent", member)
	}
	return payload, nil
}

// planMemberClass resolves the planned class for one enumerated key.
// The lookup is exact: a key that merely shares a prefix with a
// planned key is unplanned, and an unplanned store member refuses
// capture before any manifest is sealed, because silently dropping
// source bytes would forge completeness.
func planMemberClass(plan map[string]PlanItem, key string) (string, error) {
	item, planned := plan[key]
	if !planned {
		return "", invalid("capture store member %q is not a plan candidate", key)
	}
	return item.Class, nil
}

// intermediatePrefix reports whether the observed other stands on
// the path of a planned key: a symlink or special that a nested
// member traverses. Its refusal belongs to the planned member's
// Guard open — which names the full member — not to the plan gate.
func intermediatePrefix(plan map[string]PlanItem, key string) bool {
	prefix := key + "/"
	for planned := range plan {
		if strings.HasPrefix(planned, prefix) {
			return true
		}
	}
	return false
}

// blockedAncestor reports whether a strict directory prefix of the
// key was observed as a non-directory: a symlink, special, or file
// standing where the Guard must descend. The planned member routes
// to its Guard open, which refuses with the full member named,
// instead of reporting the unobserved member absent.
func blockedAncestor(observed map[string]observedMember, key string) bool {
	for offset := 0; offset < len(key); offset++ {
		if key[offset] != '/' {
			continue
		}
		if member, ok := observed[key[:offset]]; ok && member.kind != memberDir {
			return true
		}
	}
	return false
}
