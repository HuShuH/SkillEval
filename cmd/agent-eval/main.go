package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"agent-skill-eval-go/internal/adapters"
	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/report"
	"agent-skill-eval-go/internal/runner"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		if err := runCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := validateCommand(os.Args[2:]); err != nil {
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

	if err := runFlags.Parse(args); err != nil {
		return err
	}

	reg, err := registry.LoadSkills(*skillsDir)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	testCases, err := runner.LoadTestCases(*casesFile)
	if err != nil {
		return fmt.Errorf("load testcases: %w", err)
	}

	adapter := adapters.MockAdapter{}
	results := runner.RunCases(context.Background(), reg, adapter, testCases)
	summary := report.Summarize(results)

	if err := report.WriteJSON(*outPath, summary); err != nil {
		return fmt.Errorf("write report: %w", err)
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

	if err := validateFlags.Parse(args); err != nil {
		return err
	}

	reg, err := registry.LoadSkills(*skillsDir)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}

	testCases, err := runner.LoadTestCases(*casesFile)
	if err != nil {
		return fmt.Errorf("load testcases: %w", err)
	}

	fmt.Printf("skills loaded: %d\n", len(reg.List()))
	fmt.Printf("testcases loaded: %d\n", len(testCases))
	fmt.Println("validation: ok")
	return nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  agent-eval run [--skills-dir PATH] [--cases-file PATH] [--out PATH]")
	fmt.Fprintln(os.Stderr, "  agent-eval validate [--skills-dir PATH] [--cases-file PATH]")
}
