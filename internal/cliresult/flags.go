package cliresult

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/relux-works/agent-session-manager/internal/axerror"
)

// CommonFlag is one of the flags Section 14.2 requires of every user command.
type CommonFlag string

// The ten flags of the Section 14.2 list, in the order it states them.
const (
	FlagConfig         CommonFlag = "--config"
	FlagDataDir        CommonFlag = "--data-dir"
	FlagStateDir       CommonFlag = "--state-dir"
	FlagCacheDir       CommonFlag = "--cache-dir"
	FlagRuntimeDir     CommonFlag = "--runtime-dir"
	FlagJSON           CommonFlag = "--json"
	FlagNoColor        CommonFlag = "--no-color"
	FlagNonInteractive CommonFlag = "--non-interactive"
	FlagTimeout        CommonFlag = "--timeout"
	FlagVerbose        CommonFlag = "--verbose"
)

// FlagYes is the conditional confirmation flag. It is deliberately not a member
// of CommonFlags: Section 14.2 states that only "commands with a documented
// confirmation additionally accept --yes; commands without such a confirmation
// MUST reject it".
const FlagYes CommonFlag = "--yes"

var commonFlagOrder = []CommonFlag{
	FlagConfig, FlagDataDir, FlagStateDir, FlagCacheDir, FlagRuntimeDir,
	FlagJSON, FlagNoColor, FlagNonInteractive, FlagTimeout, FlagVerbose,
}

// valuedFlags are the common flags that take an argument.
var valuedFlags = map[CommonFlag]struct{}{
	FlagConfig: {}, FlagDataDir: {}, FlagStateDir: {},
	FlagCacheDir: {}, FlagRuntimeDir: {}, FlagTimeout: {},
}

// CommonFlags returns the exact ten flags of the Section 14.2 list, in order.
func CommonFlags() []CommonFlag { return append([]CommonFlag(nil), commonFlagOrder...) }

// SurfaceCommand is one command of the Section 14.1 surface. It is a different
// vocabulary from Command: a Command is the tag inside a CLI Result, while a
// SurfaceCommand is what an operator types, and Section 14.1 has surfaces that
// produce no CLI Result at all.
type SurfaceCommand string

// The Section 14.1 v0.3.0 command surface, plus the v0.5.0 TerminalBackend
// inspection commands the same section adds.
const (
	SurfaceUmbrella       SurfaceCommand = "NAME"
	SurfaceStart          SurfaceCommand = "start"
	SurfaceList           SurfaceCommand = "list"
	SurfaceStatus         SurfaceCommand = "status"
	SurfaceAttach         SurfaceCommand = "attach"
	SurfaceTakeover       SurfaceCommand = "takeover"
	SurfaceFork           SurfaceCommand = "fork"
	SurfaceStop           SurfaceCommand = "stop"
	SurfaceResume         SurfaceCommand = "resume"
	SurfaceSync           SurfaceCommand = "sync"
	SurfaceDiff           SurfaceCommand = "diff"
	SurfaceMaterialize    SurfaceCommand = "materialize"
	SurfaceDoctor         SurfaceCommand = "doctor"
	SurfaceLogs           SurfaceCommand = "logs"
	SurfacePeerList       SurfaceCommand = "peer list"
	SurfacePeerProbe      SurfaceCommand = "peer probe"
	SurfaceSetProfile     SurfaceCommand = "session set-profile"
	SurfaceCloneAdapters  SurfaceCommand = "session clone adapters"
	SurfaceCloneDoctor    SurfaceCommand = "session clone doctor"
	SurfaceCloneList      SurfaceCommand = "session clone list"
	SurfaceCloneInspect   SurfaceCommand = "session clone inspect"
	SurfaceClonePlan      SurfaceCommand = "session clone plan"
	SurfaceCloneRun       SurfaceCommand = "session clone run"
	SurfaceCloneVerify    SurfaceCommand = "session clone verify"
	SurfaceCloneOpen      SurfaceCommand = "session clone open"
	SurfacePane           SurfaceCommand = "pane"
	SurfaceRPCServe       SurfaceCommand = "rpc serve"
	SurfaceBackendsList   SurfaceCommand = "terminal backends list"
	SurfaceBackendsShow   SurfaceCommand = "terminal backends show"
	SurfaceBackendsProbe  SurfaceCommand = "terminal backends probe"
	SurfaceBackendsDoctor SurfaceCommand = "terminal backends doctor"
)

type surfaceEntry struct {
	// userCommand is false for the two surfaces Section 14.1 calls internal:
	// "internal commands pane and rpc serve MAY be hidden from short help but
	// MUST have documented --help".
	userCommand bool
	// documentedConfirmation is true for the three surfaces whose Section 14.1
	// grammar carries [--yes]: takeover, stop, and materialize.
	documentedConfirmation bool
	// acceptsJSON is false only for rpc serve, which Section 14.2 calls "an RPC
	// protocol endpoint, not a CLI Result producer" that "MUST reject --json".
	acceptsJSON bool
}

// surfaceRegistry is the reviewed Section 14.1 surface table.
var surfaceRegistry = map[SurfaceCommand]surfaceEntry{
	SurfaceUmbrella:       {userCommand: true, acceptsJSON: true},
	SurfaceStart:          {userCommand: true, acceptsJSON: true},
	SurfaceList:           {userCommand: true, acceptsJSON: true},
	SurfaceStatus:         {userCommand: true, acceptsJSON: true},
	SurfaceAttach:         {userCommand: true, acceptsJSON: true},
	SurfaceTakeover:       {userCommand: true, acceptsJSON: true, documentedConfirmation: true},
	SurfaceFork:           {userCommand: true, acceptsJSON: true},
	SurfaceStop:           {userCommand: true, acceptsJSON: true, documentedConfirmation: true},
	SurfaceResume:         {userCommand: true, acceptsJSON: true},
	SurfaceSync:           {userCommand: true, acceptsJSON: true},
	SurfaceDiff:           {userCommand: true, acceptsJSON: true},
	SurfaceMaterialize:    {userCommand: true, acceptsJSON: true, documentedConfirmation: true},
	SurfaceDoctor:         {userCommand: true, acceptsJSON: true},
	SurfaceLogs:           {userCommand: true, acceptsJSON: true},
	SurfacePeerList:       {userCommand: true, acceptsJSON: true},
	SurfacePeerProbe:      {userCommand: true, acceptsJSON: true},
	SurfaceSetProfile:     {userCommand: true, acceptsJSON: true},
	SurfaceCloneAdapters:  {userCommand: true, acceptsJSON: true},
	SurfaceCloneDoctor:    {userCommand: true, acceptsJSON: true},
	SurfaceCloneList:      {userCommand: true, acceptsJSON: true},
	SurfaceCloneInspect:   {userCommand: true, acceptsJSON: true},
	SurfaceClonePlan:      {userCommand: true, acceptsJSON: true},
	SurfaceCloneRun:       {userCommand: true, acceptsJSON: true},
	SurfaceCloneVerify:    {userCommand: true, acceptsJSON: true},
	SurfaceCloneOpen:      {userCommand: true, acceptsJSON: true},
	SurfacePane:           {acceptsJSON: true},
	SurfaceRPCServe:       {},
	SurfaceBackendsList:   {userCommand: true, acceptsJSON: true},
	SurfaceBackendsShow:   {userCommand: true, acceptsJSON: true},
	SurfaceBackendsProbe:  {userCommand: true, acceptsJSON: true},
	SurfaceBackendsDoctor: {userCommand: true, acceptsJSON: true},
}

// ErrUnknownSurface reports a surface outside the Section 14.1 command table.
var ErrUnknownSurface = errors.New("unregistered ax command surface")

// Surfaces returns every registered command surface in sorted order.
func Surfaces() []SurfaceCommand {
	result := make([]SurfaceCommand, 0, len(surfaceRegistry))
	for surface := range surfaceRegistry {
		result = append(result, surface)
	}
	sort.Slice(result, func(left, right int) bool { return result[left] < result[right] })
	return result
}

// UserSurfaces returns the surfaces Section 14.2's "all user commands" ranges
// over, in sorted order. It excludes exactly the two Section 14.1 calls
// internal.
func UserSurfaces() []SurfaceCommand {
	var result []SurfaceCommand
	for _, surface := range Surfaces() {
		if surfaceRegistry[surface].userCommand {
			result = append(result, surface)
		}
	}
	return result
}

// AcceptsYes reports whether a surface has a documented confirmation, and is
// therefore permitted to accept --yes.
func AcceptsYes(surface SurfaceCommand) (bool, error) {
	entry, known := surfaceRegistry[surface]
	if !known {
		return false, fmt.Errorf("%w: %q", ErrUnknownSurface, surface)
	}
	return entry.documentedConfirmation, nil
}

// AcceptsJSON reports whether a surface accepts --json.
func AcceptsJSON(surface SurfaceCommand) (bool, error) {
	entry, known := surfaceRegistry[surface]
	if !known {
		return false, fmt.Errorf("%w: %q", ErrUnknownSurface, surface)
	}
	return entry.acceptsJSON, nil
}

// Invocation is the parsed common-flag state of one command invocation.
type Invocation struct {
	surface        SurfaceCommand
	values         map[CommonFlag]string
	json           bool
	noColor        bool
	nonInteractive bool
	verbose        bool
	yes            bool
	timeout        time.Duration
	timeoutSet     bool
	operands       []string
}

// Surface reports the command surface this invocation targets.
func (invocation *Invocation) Surface() SurfaceCommand { return invocation.surface }

// JSON reports whether --json selected structured output.
func (invocation *Invocation) JSON() bool { return invocation.json }

// NoColor reports whether --no-color was supplied.
func (invocation *Invocation) NoColor() bool { return invocation.noColor }

// NonInteractive reports whether --non-interactive forbids prompts.
func (invocation *Invocation) NonInteractive() bool { return invocation.nonInteractive }

// Verbose reports whether --verbose was supplied.
func (invocation *Invocation) Verbose() bool { return invocation.verbose }

// Yes reports whether --yes was supplied. It is false for every surface without
// a documented confirmation, because ParseCommonFlags refuses the flag there.
func (invocation *Invocation) Yes() bool { return invocation.yes }

// Timeout reports the --timeout value and whether one was supplied.
func (invocation *Invocation) Timeout() (time.Duration, bool) {
	return invocation.timeout, invocation.timeoutSet
}

// Path reports the value of one path-valued common flag.
func (invocation *Invocation) Path(flag CommonFlag) (string, bool) {
	value, ok := invocation.values[flag]
	return value, ok
}

// Operands returns the arguments that are not common flags. Section 14.1's
// command-specific flags and positional arguments are parsed by the command
// surface that owns them, not here; this parser implements the Section 14.2
// common set and hands everything else back untouched.
func (invocation *Invocation) Operands() []string {
	return append([]string(nil), invocation.operands...)
}

// Mode reports the output mode this invocation selected.
func (invocation *Invocation) Mode() Mode {
	if invocation.json {
		return ModeJSON
	}
	return ModeText
}

// ParseCommonFlags parses the Section 14.2 common flags for one surface.
//
// It implements exactly the two refusals that section states, and returns each
// as a Structured Error so that the process exit status follows from the code
// rather than from a second, parallel decision:
//
//   - "--yes; commands without such a confirmation MUST reject it";
//   - "the internal streaming command ax rpc serve --stdio is an RPC protocol
//     endpoint, not a CLI Result producer, and MUST reject --json".
//
// Everything that is not a common flag is returned as an operand. This parser
// does not implement the Section 14.1 command-specific grammar, and does not
// claim to: a command's own flags are refused or accepted by the surface that
// declares them.
func ParseCommonFlags(surface SurfaceCommand, argv []string) (*Invocation, *axerror.Error) {
	entry, known := surfaceRegistry[surface]
	if !known {
		return nil, mustInvalidArguments(fmt.Sprintf("%q is not an ax command surface", surface))
	}
	invocation := &Invocation{surface: surface, values: map[CommonFlag]string{}}
	for index := 0; index < len(argv); index++ {
		argument := argv[index]
		name, inlineValue, inline := splitFlag(argument)
		flag := CommonFlag(name)
		if flag != FlagYes && !isCommonFlag(flag) {
			invocation.operands = append(invocation.operands, argument)
			continue
		}
		value := inlineValue
		if _, needsValue := valuedFlags[flag]; needsValue && !inline {
			index++
			if index >= len(argv) {
				return nil, mustInvalidArguments(fmt.Sprintf("%s requires a value", flag))
			}
			value = argv[index]
		} else if _, needsValue := valuedFlags[flag]; !needsValue && inline {
			return nil, mustInvalidArguments(fmt.Sprintf("%s takes no value", flag))
		}
		if failure := invocation.apply(entry, flag, value); failure != nil {
			return nil, failure
		}
	}
	return invocation, nil
}

func (invocation *Invocation) apply(entry surfaceEntry, flag CommonFlag, value string) *axerror.Error {
	switch flag {
	case FlagJSON:
		if !entry.acceptsJSON {
			return mustInvalidArguments(fmt.Sprintf(
				"%s rejects --json: it is an RPC protocol endpoint, not a CLI Result producer",
				invocation.surface))
		}
		invocation.json = true
	case FlagYes:
		if !entry.documentedConfirmation {
			return mustInvalidArguments(fmt.Sprintf(
				"%s has no documented confirmation and rejects --yes", invocation.surface))
		}
		invocation.yes = true
	case FlagNoColor:
		invocation.noColor = true
	case FlagNonInteractive:
		invocation.nonInteractive = true
	case FlagVerbose:
		invocation.verbose = true
	case FlagTimeout:
		duration, err := parseTimeout(value)
		if err != nil {
			return mustInvalidArguments(err.Error())
		}
		invocation.timeout = duration
		invocation.timeoutSet = true
	default:
		if value == "" {
			return mustInvalidArguments(fmt.Sprintf("%s requires a non-empty value", flag))
		}
		invocation.values[flag] = value
	}
	return nil
}

// parseTimeout reads a --timeout DURATION value. Section 1.6 says "durations
// MUST be integer milliseconds", so a bare integer is milliseconds; a Go
// duration string is also accepted and must resolve to a whole number of
// milliseconds, because a sub-millisecond timeout is not representable in the
// common data model.
func parseTimeout(value string) (time.Duration, error) {
	if value == "" {
		return 0, errors.New("--timeout requires a value")
	}
	if milliseconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if milliseconds < 0 {
			return 0, errors.New("--timeout must not be negative")
		}
		return time.Duration(milliseconds) * time.Millisecond, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("--timeout %q is neither integer milliseconds nor a duration", value)
	}
	if duration < 0 {
		return 0, errors.New("--timeout must not be negative")
	}
	if duration%time.Millisecond != 0 {
		return 0, fmt.Errorf("--timeout %q is not a whole number of milliseconds", value)
	}
	return duration, nil
}

func splitFlag(argument string) (string, string, bool) {
	if !strings.HasPrefix(argument, "--") {
		return argument, "", false
	}
	name, value, found := strings.Cut(argument, "=")
	return name, value, found
}

func isCommonFlag(flag CommonFlag) bool {
	for _, candidate := range commonFlagOrder {
		if candidate == flag {
			return true
		}
	}
	return false
}

// mustInvalidArguments builds the Section 15.3 exit-2 usage failure. The code
// is fixed and the message is written here, so the construction below cannot
// fail for any input this package produces; a failure would mean the message
// bound was violated, and panicking is preferable to returning a nil failure
// that a caller would read as success.
func mustInvalidArguments(message string) *axerror.Error {
	failure, err := axerror.New(axerror.Spec{
		Version: axerror.Version100,
		Code:    "invalid_arguments",
		Message: message,
		IDs:     axerror.NoIDs(),
		Details: axerror.Details{},
	})
	if err != nil {
		panic(fmt.Sprintf("cli result usage failure is unconstructible: %v", err))
	}
	return failure
}
