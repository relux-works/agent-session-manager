// tracecheck is the headless CI entry point for AX specification-to-code
// ownership verification. It reports inventory coverage, not runtime support.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/relux-works/agent-session-manager/internal/traceability"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("tracecheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "repository root")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	report, err := traceability.VerifyRepository(os.DirFS(*root))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(
		stdout,
		"traceability ok: contracts=%d normative_sections=%d acceptance_cases=%d fixtures=%d compatibility_contracts=%d\n",
		report.Contracts,
		report.NormativeSections,
		report.AcceptanceCases,
		report.Fixtures,
		report.CompatibilityContracts,
	)
	return err
}
