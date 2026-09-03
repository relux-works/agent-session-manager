// tracecheck is the headless CI entry point for AX specification-to-code
// ownership verification. It reports inventory coverage, not runtime support.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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
	var sections sectionFlags
	flags.Var(&sections, "section", "assigned normative section or same-section range; repeat for multiple scopes")
	if err := flags.Parse(arguments); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}

	var report traceability.Report
	var err error
	if len(sections) == 0 {
		report, err = traceability.VerifyRepository(os.DirFS(*root))
	} else {
		report, err = traceability.VerifyAssignedSections(os.DirFS(*root), sections)
	}
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(
		stdout,
		"traceability ok: contracts=%d normative_sections=%d acceptance_cases=%d fixtures=%d compatibility_contracts=%d assigned_scopes=%d\n",
		report.Contracts,
		report.NormativeSections,
		report.AcceptanceCases,
		report.Fixtures,
		report.CompatibilityContracts,
		report.AssignedScopes,
	); err != nil {
		return err
	}
	// Coverage is printed as the measured ratio it was computed from. A gate
	// that reported "sections owned" would repeat the claim this line exists to
	// contradict.
	_, err = fmt.Fprintf(
		stdout,
		"section coverage: bindings=%d full=%d partial=%d sliver=%d unevidenced=%d unmeasured=%d unowned=%d clauses_discharged=%d/%d\n",
		report.SectionBindings,
		report.FullCoverage,
		report.PartialCoverage,
		report.SliverCoverage,
		report.UnevidencedCoverage,
		report.UnmeasuredCoverage,
		report.UnownedSections,
		report.DischargedClauses,
		report.NormativeClauses,
	)
	return err
}

type sectionFlags []string

func (sections *sectionFlags) String() string {
	return strings.Join(*sections, ",")
}

func (sections *sectionFlags) Set(value string) error {
	*sections = append(*sections, value)
	return nil
}
