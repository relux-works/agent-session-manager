//go:build windows

package hosttrust

import (
	"encoding/binary"
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

// windowsOwnerSID resolves the process user SID the custody checks compare
// against.
func windowsOwnerSID(t *testing.T) *windows.SID {
	t.Helper()
	current, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	sid, err := windows.StringToSid(current.Uid)
	if err != nil {
		t.Fatal(err)
	}
	return sid
}

// windowsSetProtectedDACL replaces the DACL with exactly the given allow
// entries behind the protected flag, so no parent grant can leak in. Each
// entry carries the given access mask; directory entries carry container
// and object inheritance when inherit is set. It mirrors the production
// installOwnerOnlyACL buffer layout for N entries.
func windowsSetProtectedDACL(t *testing.T, path string, dir bool, grants []windowsAllowGrant) {
	t.Helper()
	type aceLayout struct {
		size uint16
		mask uint32
		sid  *windows.SID
	}
	layouts := make([]aceLayout, 0, len(grants))
	total := 8
	for _, grant := range grants {
		size := uint16(8 + windows.GetLengthSid(grant.sid))
		layouts = append(layouts, aceLayout{size: size, mask: grant.mask, sid: grant.sid})
		total += int(size)
	}
	buffer := make([]byte, total)
	buffer[0] = 2 // ACL_REVISION
	binary.LittleEndian.PutUint16(buffer[2:], uint16(total))
	binary.LittleEndian.PutUint16(buffer[4:], uint16(len(layouts)))
	offset := 8
	for _, layout := range layouts {
		buffer[offset] = windows.ACCESS_ALLOWED_ACE_TYPE
		if dir {
			buffer[offset+1] = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
		}
		binary.LittleEndian.PutUint16(buffer[offset+2:], layout.size)
		binary.LittleEndian.PutUint32(buffer[offset+4:], layout.mask)
		destination := (*windows.SID)(unsafe.Pointer(&buffer[offset+8]))
		if err := windows.CopySid(windows.GetLengthSid(layout.sid), destination, layout.sid); err != nil {
			t.Fatal(err)
		}
		offset += int(layout.size)
	}
	acl := (*windows.ACL)(unsafe.Pointer(&buffer[0]))
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil); err != nil {
		t.Fatalf("SetNamedSecurityInfo(%s) error = %v", path, err)
	}
}

type windowsAllowGrant struct {
	sid  *windows.SID
	mask uint32
}

func windowsVerifyFile(t *testing.T, path string) error {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return verifyOwnerFile(path, info, 0o600)
}

// A file secured through the production install path verifies owner-only.
func TestWindowsOwnerOnlyFileAccepted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custody.dat")
	if err := os.WriteFile(path, []byte("custody"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := secureStaged(path, false); err != nil {
		t.Fatalf("secureStaged error = %v", err)
	}
	if err := windowsVerifyFile(t, path); err != nil {
		t.Fatalf("verifyOwnerFile(secured) error = %v", err)
	}
}

// An explicit allow entry for another principal refuses, even when the
// owner grant is intact and the entry grants only read access.
func TestWindowsOtherPrincipalRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custody.dat")
	if err := os.WriteFile(path, []byte("custody"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := secureStaged(path, false); err != nil {
		t.Fatal(err)
	}
	if err := windowsVerifyFile(t, path); err != nil {
		t.Fatalf("verifyOwnerFile(secured) error = %v", err)
	}
	everyone, err := windows.StringToSid("S-1-1-0")
	if err != nil {
		t.Fatal(err)
	}
	windowsSetProtectedDACL(t, path, false, []windowsAllowGrant{
		{sid: windowsOwnerSID(t), mask: uint32(windows.GENERIC_ALL)},
		{sid: everyone, mask: uint32(windows.GENERIC_READ)},
	})
	if err := windowsVerifyFile(t, path); err == nil {
		t.Fatal("verifyOwnerFile(other-principal grant) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustUnsafeCustody) {
		t.Fatalf("verifyOwnerFile error = %v, want ErrTrustUnsafeCustody", err)
	}
}

// Inherited grants are audited, not waived: a file inheriting an allow
// entry for another principal from its parent refuses.
func TestWindowsInheritedGrantRefused(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent")
	if err := os.Mkdir(parent, 0o700); err != nil {
		t.Fatal(err)
	}
	everyone, err := windows.StringToSid("S-1-1-0")
	if err != nil {
		t.Fatal(err)
	}
	windowsSetProtectedDACL(t, parent, true, []windowsAllowGrant{
		{sid: windowsOwnerSID(t), mask: uint32(windows.GENERIC_ALL)},
		{sid: everyone, mask: uint32(windows.GENERIC_READ)},
	})
	path := filepath.Join(parent, "child.dat")
	if err := os.WriteFile(path, []byte("custody"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := windowsVerifyFile(t, path); err == nil {
		t.Fatal("verifyOwnerFile(inherited other-principal grant) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustUnsafeCustody) {
		t.Fatalf("verifyOwnerFile error = %v, want ErrTrustUnsafeCustody", err)
	}
}

// A null DACL grants everyone full access and must refuse.
func TestWindowsNullDACLRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custody.dat")
	if err := os.WriteFile(path, []byte("custody"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := secureStaged(path, false); err != nil {
		t.Fatal(err)
	}
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION, nil, nil, nil, nil); err != nil {
		t.Fatalf("SetNamedSecurityInfo(null DACL) error = %v", err)
	}
	if err := windowsVerifyFile(t, path); err == nil {
		t.Fatal("verifyOwnerFile(null DACL) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustUnsafeCustody) {
		t.Fatalf("verifyOwnerFile error = %v, want ErrTrustUnsafeCustody", err)
	}
}

// Reparse points refuse: a symlink at a custody path never verifies,
// without following the link.
func TestWindowsReparseRefused(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.dat")
	if err := os.WriteFile(target, []byte("custody"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.dat")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink creation needs privilege or developer mode: %v", err)
	}
	if err := windowsVerifyFile(t, link); err == nil {
		t.Fatal("verifyOwnerFile(symlink) succeeded, want refusal")
	} else if !errors.Is(err, ErrTrustUnsafeCustody) {
		t.Fatalf("verifyOwnerFile error = %v, want ErrTrustUnsafeCustody", err)
	}
}

// A secured directory verifies, and files created inside it inherit the
// owner-only DACL through the production inheritance flags.
func TestWindowsSecureStagedDirectoryInherits(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "custody-dir")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := secureStaged(dir, true); err != nil {
		t.Fatalf("secureStaged(dir) error = %v", err)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyOwnerDir(dir, info, 0o700); err != nil {
		t.Fatalf("verifyOwnerDir(secured) error = %v", err)
	}
	path := filepath.Join(dir, "child.dat")
	if err := os.WriteFile(path, []byte("custody"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := windowsVerifyFile(t, path); err != nil {
		t.Fatalf("verifyOwnerFile(inherited owner-only) error = %v", err)
	}
}

// The production store entry points open, initialize and read on Windows
// through the owner-only DACL install and audit.
func TestWindowsOpenInitializesOwnerOnlyStore(t *testing.T) {
	stateDir := t.TempDir()
	store, err := Open(resolvedPathsForTest(t, filepath.Join(stateDir, "config.toml"), stateDir))
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	snapshot, err := store.Initialize()
	if err != nil {
		t.Fatalf("Initialize error = %v", err)
	}
	if snapshot.Generation != 1 {
		t.Fatalf("Initialize generation = %d, want 1", snapshot.Generation)
	}
	fresh, err := store.ReadSnapshot()
	if err != nil {
		t.Fatalf("ReadSnapshot error = %v", err)
	}
	if fresh.Generation != 1 {
		t.Fatalf("ReadSnapshot generation = %d, want 1", fresh.Generation)
	}
}
