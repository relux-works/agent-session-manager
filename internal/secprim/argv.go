package secprim

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// CheckArgv validates one process argv vector under Section 16.4: every
// process invocation uses an argv array and native process APIs, and shell
// command construction from names, paths, provider output, or peer data is
// forbidden. This package offers no string-to-argv constructor at all —
// there is no shell word-splitting entry point to misuse — and CheckArgv
// decides what a validated vector is:
//
//   - the vector is non-empty and names a non-empty executable;
//   - no element is empty: an empty argv element is legal to exec but never
//     meaningful to an AX or provider CLI, so it fails closed;
//   - no element carries a NUL byte, which cannot cross execve;
//   - every element is valid UTF-8, because argv elements recur in receipts,
//     logs, and terminal rendering, all of which are UTF-8.
//
// No length bound is enforced here: the Section 5.1 byte bounds apply to
// the Provider SpawnPlan wire object and stay in internal/provhost, which
// admits the plan. A plan admitted there still crosses this gate at
// process start, so a caller that bypasses plan admission cannot bypass
// argv validation.
func CheckArgv(argv []string) error {
	if len(argv) == 0 {
		return failArgv("argv empty", "no executable")
	}
	for index, element := range argv {
		position := "argv[" + strconv.Itoa(index) + "]"
		switch {
		case element == "":
			if index == 0 {
				return failArgv("argv executable empty", position)
			}
			return failArgv("argv element empty", position)
		case strings.ContainsRune(element, 0):
			return failArgv("argv NUL", position)
		case !utf8.ValidString(element):
			return failArgv("argv encoding", position)
		}
	}
	return nil
}

// Command is a validated argv vector. The only constructor takes an array,
// so a command cannot be built from a shell string: word-splitting hostile
// input never happens because no entry point performs it. The zero value
// is unusable; build one with NewCommand.
type Command struct {
	argv []string
}

// NewCommand validates the vector with CheckArgv and keeps a copy: the
// caller's slice cannot be rewritten after admission to smuggle in an
// unchecked element.
func NewCommand(argv []string) (Command, error) {
	if err := CheckArgv(argv); err != nil {
		return Command{}, err
	}
	return Command{argv: append([]string(nil), argv...)}, nil
}

// Argv returns the validated vector. The result is a copy for the same
// reason the constructor copies: a retained alias would let the holder
// rewrite an admitted command.
func (command Command) Argv() []string {
	return append([]string(nil), command.argv...)
}

// Executable reports argv[0], the process image the host will start. It is
// never empty: NewCommand refused that.
func (command Command) Executable() string {
	return command.argv[0]
}
