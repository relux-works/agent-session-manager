package secprim

import (
	"errors"
	"fmt"
)

// Refusal sentinels. Every refusal this package produces wraps exactly one
// of these, so callers match on the kind without parsing human text.
var (
	ErrUnsafePath = errors.New("secprim unsafe path")
	ErrUnsafeArgv = errors.New("secprim unsafe argv")
	ErrUnsafeEnv  = errors.New("secprim unsafe environment")
)

// Error is the only refusal this package produces. Kind is one of the
// sentinels above; Rule names the specification rule that fired
// (for example "member traversal" or "argv NUL"); Detail names the
// offending member, argument position, or variable name without echoing
// foreign bytes beyond what names the site.
type Error struct {
	Kind   error
	Rule   string
	Detail string
	cause  error
}

func (err *Error) Error() string {
	if err.cause == nil {
		return fmt.Sprintf("%v: %s: %s", err.Kind, err.Rule, err.Detail)
	}
	return fmt.Sprintf("%v: %s: %s: %v", err.Kind, err.Rule, err.Detail, err.cause)
}

// WithCause records the underlying filesystem failure for errors.Is and
// errors.As without changing the refusal rendering the witnesses pin:
// the cause appends after the pinned prefix.
func (err *Error) WithCause(cause error) *Error {
	err.cause = cause
	return err
}

func (err *Error) Unwrap() []error {
	if err.cause == nil {
		return []error{err.Kind}
	}
	return []error{err.Kind, err.cause}
}

// The three refusal constructors below are declared as variables so the
// refusal-inventory gate can observe every exercised refusal site. Each is
// the single construction site for its failure class; production code must
// not build Error values any other way: no aliasing a constructor into a
// second name, no &Error{} or Error{} literal, no new(Error), and no
// wrapper function or method that manufactures an *Error. The inventory
// test enforces every member of that shape space, not just the bare
// constructor spelling.
//
// Stated bound. The gate matches the Error identifier spelling: a
// refusal type spelled through a type alias (type X = Error), a
// defined type (type X Error, even when converted back to *Error), or
// a factory whose result spells the type through such an alias is
// outside the witness. The derivation is syntactic on purpose — it
// runs on parsed source with no type information — so it closes the
// enumerated spellings, not the class. See
// TestRefusalInventorySpellingBoundIsStated, which pins this bound by
// feeding the alias spellings through the derivation and requiring
// the gate to stay silent about them.
//
// failPath reports a Section 16.3 path or filesystem refusal: a member
// outside the contained grammar, a symlink or special file where a
// regular file is required, a case-fold collision, or an unmanaged
// replacement.
//
// failArgv reports a Section 16.4 argv refusal: an empty vector, an empty
// executable or argument element, a NUL byte, or invalid UTF-8.
//
// failEnv reports a Section 16.7 environment refusal: a name outside the
// environment-name grammar, a literal colliding with an inherited name, or
// a NUL byte or invalid UTF-8 in a value.
var failPath = func(rule, detail string) *Error {
	return &Error{Kind: ErrUnsafePath, Rule: rule, Detail: detail}
}

var failArgv = func(rule, detail string) *Error {
	return &Error{Kind: ErrUnsafeArgv, Rule: rule, Detail: detail}
}

var failEnv = func(rule, detail string) *Error {
	return &Error{Kind: ErrUnsafeEnv, Rule: rule, Detail: detail}
}
