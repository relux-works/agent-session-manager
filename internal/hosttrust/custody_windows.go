//go:build windows

package hosttrust

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os/user"
	"unsafe"

	"golang.org/x/sys/windows"
)

// platformModeOK always holds on Windows: Go synthesizes mode bits from the
// read-only attribute, so an exact comparison would refuse every path. The
// owner-only DACL governs instead and is enforced by verifyOwner below.
func platformModeOK(_ fs.FileInfo, _ fs.FileMode) bool { return true }

// secureStaged installs the owner-only DACL on a newly created path. Staging
// files and directories inherit the parent ACL by default, which typically
// grants SYSTEM and Administrators: without this step every file the store
// creates would fail its own verification below.
func secureStaged(path string, dir bool) error {
	if err := installOwnerOnlyACL(path, dir); err != nil {
		return trustError(TrustError{Operation: "secure custody", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	return nil
}

// verifyOwner enforces owner-only custody on Windows: reparse-point escapes
// refuse, the file owner SID must equal the process user SID, and the DACL
// must grant access exclusively to that owner. Ownership alone does not
// establish grants, so every allow entry is audited, including inherited
// ones: a parent that grants SYSTEM, Administrators or any other principal
// refuses until the operator removes the grant. A null DACL (full access
// for everyone) and an empty or deny-only DACL (unusable) refuse as well.
func verifyOwner(path string, info fs.FileInfo) error {
	if info.Mode()&fs.ModeSymlink != 0 {
		return trustError(TrustError{Operation: "inspect ownership", Err: ErrTrustUnsafeCustody})
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	attributes, err := windows.GetFileAttributes(name)
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return trustError(TrustError{Operation: "inspect ownership", Err: ErrTrustUnsafeCustody})
	}
	current, err := user.Current()
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	userSID, err := windows.StringToSid(current.Uid)
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	owner, _, err := descriptor.Owner()
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	if owner == nil || !owner.Equals(userSID) {
		return trustError(TrustError{Operation: "inspect ownership", Err: ErrTrustUnsafeCustody})
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	if dacl == nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: ErrTrustUnsafeCustody})
	}
	aces, err := extractAllowedACEs(dacl)
	if err != nil {
		return trustError(TrustError{Operation: "inspect ownership", Err: errors.Join(ErrTrustUnsafeCustody, err)})
	}
	if !ownerOnlyGrants(owner.String(), aces) {
		return trustError(TrustError{Operation: "inspect ownership", Err: ErrTrustUnsafeCustody})
	}
	return nil
}

// extractAllowedACEs translates the native DACL into the platform-independent
// audit shape. Unknown entry types fail closed through aceOther.
func extractAllowedACEs(dacl *windows.ACL) ([]allowedACE, error) {
	var aces []allowedACE
	for index := uint32(0); index < uint32(dacl.AceCount); index++ {
		var raw *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, index, &raw); err != nil {
			return nil, err
		}
		if raw == nil {
			return nil, errors.New("absent access entry")
		}
		switch raw.Header.AceType {
		case windows.ACCESS_DENIED_ACE_TYPE:
			aces = append(aces, allowedACE{kind: aceDeny})
			continue
		case windows.ACCESS_ALLOWED_ACE_TYPE:
		default:
			aces = append(aces, allowedACE{kind: aceOther})
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&raw.SidStart))
		aces = append(aces, allowedACE{kind: aceAllow, sid: sid.String()})
	}
	return aces, nil
}

// installOwnerOnlyACL replaces the DACL with one protected owner-only grant:
// full access for the process user SID and nothing for anyone else. The
// protected flag blocks inherited grants from the parent, which otherwise
// reintroduce SYSTEM and Administrators. Directory grants carry container
// and object inheritance so paths the store creates later start owner-only
// too; every creation is still verified explicitly after install.
func installOwnerOnlyACL(path string, dir bool) error {
	current, err := user.Current()
	if err != nil {
		return err
	}
	sid, err := windows.StringToSid(current.Uid)
	if err != nil {
		return err
	}
	sidLen := windows.GetLengthSid(sid)
	aceSize := uint16(8 + sidLen)
	buffer := make([]byte, 8+int(aceSize))
	buffer[0] = 2 // ACL_REVISION
	binary.LittleEndian.PutUint16(buffer[2:], 8+aceSize)
	binary.LittleEndian.PutUint16(buffer[4:], 1)
	buffer[8] = windows.ACCESS_ALLOWED_ACE_TYPE
	if dir {
		buffer[9] = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
	}
	binary.LittleEndian.PutUint16(buffer[10:], aceSize)
	binary.LittleEndian.PutUint32(buffer[12:], uint32(windows.GENERIC_ALL))
	destination := (*windows.SID)(unsafe.Pointer(&buffer[16]))
	if err := windows.CopySid(sidLen, destination, sid); err != nil {
		return err
	}
	acl := (*windows.ACL)(unsafe.Pointer(&buffer[0]))
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil)
}
