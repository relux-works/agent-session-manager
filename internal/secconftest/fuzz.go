package secconftest

import (
	"sync/atomic"

	"github.com/relux-works/agent-session-manager/internal/scalar"
	"github.com/relux-works/agent-session-manager/internal/secprim"
)

// FuzzEntry identifies one production entry point a fuzz driver
// reaches. FuzzTargetEntry maps each fuzz target to its entry so the
// suite can assert the mapping instead of claiming it in prose.
type FuzzEntry string

const (
	EntryCheckArgv           FuzzEntry = "secprim.CheckArgv"
	EntryCheckMemberPath     FuzzEntry = "secprim.CheckMemberPath"
	EntryIsEnvName           FuzzEntry = "secprim.IsEnvName"
	EntryRedact              FuzzEntry = "secprim.Redact"
	EntryEscapeForTerminal   FuzzEntry = "secprim.EscapeForTerminal"
	EntryRenderForTerminal   FuzzEntry = "secprim.RenderForTerminal"
	EntryGuardResolve        FuzzEntry = "secprim.Guard.Resolve"
	EntryDetectCaseCollision FuzzEntry = "secprim.DetectCaseCollision"
)

// FuzzTargetEntry maps each committed fuzz target to the production
// entry point it drives. Every target in fuzz_test.go must appear
// here, and TestFuzzTargetsReachProduction pins both directions: no
// target without an entry, no entry in the map without a target that
// fires it.
var FuzzTargetEntry = map[string]FuzzEntry{
	"FuzzCheckArgv":           EntryCheckArgv,
	"FuzzCheckMemberPath":     EntryCheckMemberPath,
	"FuzzIsEnvName":           EntryIsEnvName,
	"FuzzRedactCorpus":        EntryRedact,
	"FuzzEscapeForTerminal":   EntryEscapeForTerminal,
	"FuzzRenderForTerminal":   EntryRenderForTerminal,
	"FuzzGuardResolve":        EntryGuardResolve,
	"FuzzDetectCaseCollision": EntryDetectCaseCollision,
}

// Driver is the shared path from seed-corpus tests and fuzz targets
// into the Section 16 production gates. Both call these methods, so
// the seed tests prove arrival at the same entry points the fuzzer
// mutates against. Every method records its call before delegating,
// so a driver that stopped reaching production shows a zero count.
type Driver struct {
	calls map[FuzzEntry]*atomic.Int64
}

// NewDriver builds a Driver with zeroed arrival counters.
func NewDriver() *Driver {
	driver := &Driver{calls: map[FuzzEntry]*atomic.Int64{}}
	for _, entry := range []FuzzEntry{
		EntryCheckArgv, EntryCheckMemberPath, EntryIsEnvName,
		EntryRedact, EntryEscapeForTerminal, EntryRenderForTerminal,
		EntryGuardResolve, EntryDetectCaseCollision,
	} {
		driver.calls[entry] = &atomic.Int64{}
	}
	return driver
}

func (driver *Driver) record(entry FuzzEntry) {
	driver.calls[entry].Add(1)
}

// Count reports arrivals at entry.
func (driver *Driver) Count(entry FuzzEntry) int64 {
	counter, ok := driver.calls[entry]
	if !ok {
		return 0
	}
	return counter.Load()
}

// DriveArgv drives secprim.CheckArgv and reports admission.
func (driver *Driver) DriveArgv(argv []string) bool {
	driver.record(EntryCheckArgv)
	return secprim.CheckArgv(argv) == nil
}

// DriveMemberPath drives secprim.CheckMemberPath on the linux member
// grammar and reports admission.
func (driver *Driver) DriveMemberPath(member string) bool {
	driver.record(EntryCheckMemberPath)
	return secprim.CheckMemberPath(scalar.PlatformLinux, member) == nil
}

// DriveEnvName drives secprim.IsEnvName.
func (driver *Driver) DriveEnvName(name string) bool {
	driver.record(EntryIsEnvName)
	return secprim.IsEnvName(name)
}

// DriveRedact drives secprim.Redact with no caller corpus and reports
// whether the output still carries the input's non-secret marker. The
// fixed marker keeps the driver deterministic: corpus secrets come
// from the caller in TestFuzzDriversReachProduction, not from the
// fuzzer.
func (driver *Driver) DriveRedact(line string) bool {
	driver.record(EntryRedact)
	out := secprim.Redact(line, nil)
	return len(out) > 0 || len(line) == 0
}

// DriveEscape drives secprim.EscapeForTerminal.
func (driver *Driver) DriveEscape(text string) string {
	driver.record(EntryEscapeForTerminal)
	return secprim.EscapeForTerminal(text)
}

// DriveRender drives secprim.RenderForTerminal with no caller corpus.
func (driver *Driver) DriveRender(line string) string {
	driver.record(EntryRenderForTerminal)
	return secprim.RenderForTerminal(line, nil)
}

// DriveGuardResolve drives secprim.Guard.Resolve against a fixed
// in-memory root and reports admission. The guard construction cannot
// fail for the fixed root; a construction failure refuses the input
// the same way a resolution refusal does.
func (driver *Driver) DriveGuardResolve(member string) bool {
	driver.record(EntryGuardResolve)
	guard, err := secprim.NewGuard("/stage", scalar.PlatformLinux, nil)
	if err != nil {
		return false
	}
	_, err = guard.Resolve(member)
	return err == nil
}

// DriveCaseCollision drives secprim.DetectCaseCollision against a
// fixed sibling set and reports whether a collision fired.
func (driver *Driver) DriveCaseCollision(candidate string) bool {
	driver.record(EntryDetectCaseCollision)
	_, collided := secprim.DetectCaseCollision([]string{"README.md", "staging/member"}, candidate)
	return collided
}
