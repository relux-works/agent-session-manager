package clonereadback

import "fmt"

// invalid builds one refusal. Every gate refuses with a literal
// detail at the deciding site so each rule stays a distinct,
// censusable gate.
func invalid(format string, arguments ...any) error {
	return fmt.Errorf("clonereadback: "+format, arguments...)
}
