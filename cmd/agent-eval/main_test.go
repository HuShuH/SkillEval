package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainNoSubcommand(t *testing.T) {
	output, err := runCLI(t)
	if err == nil {
		t.Fatal("expected failure when no subcommand is provided")
	}
	if !strings.Contains(output, "usage:") {
		t.Fatalf("expected usage output, got: %s", output)
	}
}

func TestMainUnknownSubcommand(t *testing.T) {
	output, err := runCLI(t, "badcmd")
	if err == nil {
		t.Fatal("expected failure for unknown subcommand")
	}
	if !strings.Contains(output, "unknown subcommand") {
		t.Fatalf("expected unknown subcommand message, got: %s", output)
	}
}

func TestValidateSuccess(t *testing.T) {
	output, err := runCLI(t, "validate")
	if err != nil {
		t.Fatalf("expected validate to succeed, got error: %v\noutput: %s", err, output)
	}
	for _, want := range []string{"skills loaded:", "testcases loaded:", "validation: ok"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got: %s", want, output)
		}
	}
}

func TestValidateDuplicateCaseID(t *testing.T) {
	casesFile := writeTempCasesFile(t,
		`{"case_id":"dup_case","prompt":"hello","allowed_tools":[],"skill":{"name":"hello_world"},"hard_checks":{"expected_output":"hello world"},"timeout_seconds":3}`,
		`{"case_id":"dup_case","prompt":"hello again","allowed_tools":[],"skill":{"name":"hello_world"},"hard_checks":{"expected_output":"hello world"},"timeout_seconds":3}`,
	)

	output, err := runCLI(t, "validate", "--cases-file", casesFile)
	if err == nil {
		t.Fatal("expected validate to fail for duplicate case_id")
	}
	if !strings.Contains(output, "duplicate case_id") {
		t.Fatalf("expected duplicate case_id message, got: %s", output)
	}
}

func TestValidateReferencedSkillNotFound(t *testing.T) {
	casesFile := writeTempCasesFile(t,
		`{"case_id":"missing_skill_case","prompt":"hello","allowed_tools":[],"skill":{"name":"missing_skill"},"hard_checks":{"expected_output":"hello"},"timeout_seconds":3}`,
	)

	output, err := runCLI(t, "validate", "--cases-file", casesFile)
	if err == nil {
		t.Fatal("expected validate to fail for missing skill")
	}
	if !strings.Contains(output, "referenced skill") && !strings.Contains(output, "not found") {
		t.Fatalf("expected referenced skill error, got: %s", output)
	}
}

func TestValidateExpectedArgsWithoutToolName(t *testing.T) {
	casesFile := writeTempCasesFile(t,
		`{"case_id":"bad_args_case","prompt":"hello","allowed_tools":[],"skill":{"name":"hello_world"},"hard_checks":{"expected_args":{"value":"ok"}},"timeout_seconds":3}`,
	)

	output, err := runCLI(t, "validate", "--cases-file", casesFile)
	if err == nil {
		t.Fatal("expected validate to fail when expected_args is set without expected_tool_name")
	}
	if !strings.Contains(output, "expected_tool_name must be set") {
		t.Fatalf("expected expected_tool_name validation error, got: %s", output)
	}
}

func TestRunSuccess(t *testing.T) {
	reportPath := filepath.Join(t.TempDir(), "run.json")
	output, err := runCLI(t, "run", "--out", reportPath)
	if err != nil {
		t.Fatalf("expected run to succeed, got error: %v\noutput: %s", err, output)
	}
	for _, want := range []string{"total:", "passed:", "failed:"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got: %s", want, output)
		}
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report file to be written at %s: %v", reportPath, err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := helperArgs(os.Args)
	os.Args = append([]string{"agent-eval"}, args...)
	main()
	os.Exit(0)
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	cmdArgs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func writeTempCasesFile(t *testing.T, lines ...string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "cases.jsonl")
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp cases file failed: %v", err)
	}
	return path
}

func helperArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}
	return nil
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
