package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"agent-skill-eval-go/internal/adapters"
	"agent-skill-eval-go/internal/report"
	"agent-skill-eval-go/internal/runner"
	"agent-skill-eval-go/internal/validate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		if err := runCommand(os.Args[2:]); err != nil {
			if errors.Is(err, errAlreadyReported) {
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := validateCommand(os.Args[2:]); err != nil {
			if errors.Is(err, errAlreadyReported) {
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func runCommand(args []string) error {
	runFlags := flag.NewFlagSet("run", flag.ContinueOnError)
	runFlags.SetOutput(os.Stderr)

	skillsDir := runFlags.String("skills-dir", "./testdata/skills", "directory containing skill JSON files")
	casesFile := runFlags.String("cases-file", "./testdata/cases/mvp.jsonl", "path to testcase JSONL file")
	outPath := runFlags.String("out", "./reports/run.json", "path to output report JSON")
	jsonOutput := runFlags.Bool("json", false, "emit machine-readable JSON output")

	if err := runFlags.Parse(args); err != nil {
		return err
	}

	inputs, err := loadRunInputs(*skillsDir, *casesFile)
	if err != nil {
		return reportRunFailure(*jsonOutput, *outPath, err.Error())
	}

	adapter := adapters.MockAdapter{}
	results := runner.RunCases(context.Background(), inputs.Registry, adapter, inputs.TestCases)
	summary := report.Summarize(results)

	if err := report.WriteJSON(*outPath, summary); err != nil {
		return reportRunFailure(*jsonOutput, *outPath, fmt.Sprintf("write report: %v", err))
	}

	if *jsonOutput {
		return writeRunJSON(runResult{
			OK:         true,
			Total:      summary.Total,
			Passed:     summary.Passed,
			Failed:     summary.Failed,
			ReportPath: *outPath,
		})
	}

	fmt.Printf("total: %d\n", summary.Total)
	fmt.Printf("passed: %d\n", summary.Passed)
	fmt.Printf("failed: %d\n", summary.Failed)
	fmt.Printf("report: %s\n", *outPath)
	return nil
}

func validateCommand(args []string) error {
	validateFlags := flag.NewFlagSet("validate", flag.ContinueOnError)
	validateFlags.SetOutput(os.Stderr)

	skillsDir := validateFlags.String("skills-dir", "./testdata/skills", "directory containing skill JSON files")
	casesFile := validateFlags.String("cases-file", "./testdata/cases/mvp.jsonl", "path to testcase JSONL file")
	jsonOutput := validateFlags.Bool("json", false, "emit machine-readable JSON output")

	if err := validateFlags.Parse(args); err != nil {
		return err
	}

	inputs, validationErrors := loadValidateInputs(*skillsDir, *casesFile)
	if len(validationErrors) > 0 {
		return reportValidateFailure(*jsonOutput, 0, 0, validationErrors)
	}

	validationErrors = validate.All(inputs.Skills, inputs.TestCases, inputs.Registry)
	if len(validationErrors) > 0 {
		return reportValidateFailure(*jsonOutput, len(inputs.Skills), len(inputs.TestCases), validationErrors)
	}

	if *jsonOutput {
		return writeValidateJSON(validateResult{
			OK:              true,
			SkillsLoaded:    len(inputs.Skills),
			TestCasesLoaded: len(inputs.TestCases),
			Errors:          []string{},
		})
	}

	fmt.Printf("skills loaded: %d\n", len(inputs.Skills))
	fmt.Printf("testcases loaded: %d\n", len(inputs.TestCases))
	fmt.Println("validation: ok")
	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agent-eval run [--skills-dir PATH] [--cases-file PATH] [--out PATH] [--json]")
	fmt.Fprintln(os.Stderr, "  agent-eval validate [--skills-dir PATH] [--cases-file PATH] [--json]")
}
