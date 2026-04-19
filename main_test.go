package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-skill-eval-go/eval"
)

func TestRunCLIInvalidModeFails(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runCLI(context.Background(), &stdout, &stderr, []string{"--mode", "bad", "--prompt", "hello"})
	if err == nil {
		t.Fatalf("expected invalid mode error")
	}
}

func TestRunCLISingleJSONOutput(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--cases", casesPath,
		"--output-format", "json",
		"--workspace-root", root,
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	output := stdout.String()
	for _, fragment := range []string{`"results"`, `"summary"`, `"provider_mode": "stub"`} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output:\n%s", fragment, output)
		}
	}
}

func TestRunCLIConfigLoadsSuccessfully(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	configPath := filepath.Join(root, "run.json")
	config := `{
	  "mode": "single",
	  "cases": ` + jsonString(casesPath) + `,
	  "provider": { "name": "stub" },
	  "execution": { "max_iters": 2, "workspace_root": ` + jsonString(root) + ` },
	  "output": { "output_dir": ` + jsonString(filepath.Join(root, "out")) + `, "run_id": "cfg-run" },
	  "skills": {}
	}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("unexpected config write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--config", configPath,
		"--output-format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"provider_mode": "stub"`) {
		t.Fatalf("expected config-backed output, got %s", stdout.String())
	}
}

func TestRunCLIFlagsOverrideConfig(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	configPath := filepath.Join(root, "run.json")
	config := `{
	  "mode": "single",
	  "cases": ` + jsonString(casesPath) + `,
	  "provider": { "name": "stub" },
	  "execution": { "max_iters": 1 },
	  "output": {},
	  "skills": {}
	}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("unexpected config write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--config", configPath,
		"--max-iters", "4",
		"--print-effective-config",
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"max_iters": 4`) {
		t.Fatalf("expected overridden max_iters, got %s", stdout.String())
	}
}

func TestRunCLIPrintEffectiveConfigRedactsSensitiveValues(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	configPath := filepath.Join(root, "run.json")
	config := `{
	  "mode": "single",
	  "cases": ` + jsonString(casesPath) + `,
	  "provider": {
	    "name": "openai",
	    "model": "gpt-test",
	    "base_url": "https://example.invalid/v1",
	    "api_key": "super-secret"
	  },
	  "execution": {},
	  "output": {},
	  "skills": {}
	}`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("unexpected config write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--config", configPath,
		"--print-effective-config",
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"api_key": "REDACTED"`) {
		t.Fatalf("expected redacted api key, got %s", output)
	}
	if strings.Contains(output, "super-secret") {
		t.Fatalf("effective config leaked secret: %s", output)
	}
}

func TestRunCLIListRunsManagementMode(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "out")
	store := eval.NewOutputStore(outputRoot, "run-1")
	report := eval.BuildRunReport([]eval.CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: eval.CheckResult{Checked: true, Passed: true}},
	})
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := eval.RebuildIndex(outputRoot); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--list-runs",
		"--output-dir", outputRoot,
		"--output-format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected management error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"run_id": "run-1"`) {
		t.Fatalf("expected listed run, got %s", stdout.String())
	}
}

func TestRunCLIArchiveRunsDryRun(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "out")
	store := eval.NewOutputStore(outputRoot, "run-1")
	report := eval.BuildRunReport([]eval.CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: eval.CheckResult{Checked: true, Passed: true}},
	})
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("write report: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--archive-runs", "run-1",
		"--dry-run",
		"--output-dir", outputRoot,
	})
	if err != nil {
		t.Fatalf("unexpected management error: %v", err)
	}
	if !strings.Contains(stdout.String(), "archive dry_run=true") {
		t.Fatalf("expected dry-run output, got %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outputRoot, "run-1")); err != nil {
		t.Fatalf("run should still exist after dry-run: %v", err)
	}
}

func TestRunCLIRebuildIndexManagementMode(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "out")
	store := eval.NewOutputStore(outputRoot, "run-1")
	report := eval.BuildRunReport([]eval.CaseResult{
		{CaseID: "case-1", Passed: true, StopReason: "finished", Iterations: 1, Check: eval.CheckResult{Checked: true, Passed: true}},
	})
	if _, err := store.WriteRunReport(report); err != nil {
		t.Fatalf("write report: %v", err)
	}

	if err := os.Remove(filepath.Join(outputRoot, eval.IndexFileName)); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove index: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--rebuild-index",
		"--output-dir", outputRoot,
		"--output-format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected management error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputRoot, eval.IndexFileName)); err != nil {
		t.Fatalf("expected rebuilt index: %v", err)
	}
}

func TestRunCLILoadsRealSkillDirectory(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill-a")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("unexpected mkdir error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# writer\n\nDescription: writes files\n\n## Instructions\nFollow the task.\n\n## Tools\n- finish\n"), 0o644); err != nil {
		t.Fatalf("unexpected skill write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--prompt", "hello",
		"--skill-a", skillDir,
		"--output-format", "json",
		"--workspace-root", root,
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"provider_mode": "stub"`) {
		t.Fatalf("expected normal output, got %s", stdout.String())
	}
}

func TestRunCLIFallbackSkillStillWorks(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--prompt", "hello",
	})
	if err != nil {
		t.Fatalf("unexpected fallback cli error: %v", err)
	}
}

func TestRunCLIPairSkillPathError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "pair",
		"--prompt", "hello",
		"--skill-a", "/missing/skill-a",
		"--skill-b", "/missing/skill-b",
	})
	if err == nil {
		t.Fatalf("expected pair skill path error")
	}
}

func TestRunCLIExplicitStubProvider(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--provider", "stub",
		"--prompt", "hello",
		"--output-format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"provider_mode": "stub"`) {
		t.Fatalf("expected stub provider metadata, got %s", stdout.String())
	}
}

func TestRunCLIStubProviderStillWorksWithTimeoutAndRetryFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--provider", "stub",
		"--prompt", "hello",
		"--timeout", "1s",
		"--max-retries", "2",
		"--retry-backoff-ms", "5",
		"--output-format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"provider_mode": "stub"`) {
		t.Fatalf("expected stub output, got %s", stdout.String())
	}
}

func TestRunCLIOpenAIRequiresConfig(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--provider", "openai",
		"--prompt", "hello",
		"--api-key", "secret-key",
	})
	if err == nil || !strings.Contains(err.Error(), "--model is required") {
		t.Fatalf("expected model error, got %v", err)
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestRunCLIOpenAIProviderWithFakeHTTPServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected provider path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"real answer"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--provider", "openai",
		"--prompt", "hello",
		"--model", "gpt-test",
		"--base-url", server.URL,
		"--api-key", "secret-key",
		"--output-format", "json",
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}
	output := stdout.String()
	for _, fragment := range []string{`"provider_mode": "openai"`, `"final_answer": "real answer"`} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected fragment %q in output:\n%s", fragment, output)
		}
	}
}

func TestRunCLIOpenAIErrorDoesNotLeakKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad auth"}}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--provider", "openai",
		"--prompt", "hello",
		"--model", "gpt-test",
		"--base-url", server.URL,
		"--api-key", "super-secret-key",
	})
	if err == nil {
		t.Fatalf("expected provider error")
	}
	if strings.Contains(err.Error(), "super-secret-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestRunCLIOpenAITimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"late"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--provider", "openai",
		"--prompt", "hello",
		"--model", "gpt-test",
		"--base-url", server.URL,
		"--api-key", "secret-key",
		"--provider-timeout", "10ms",
		"--output-format", "json",
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error leaked api key: %v", err)
	}
}

func TestRunCLIRejectsNegativeRetryFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--prompt", "hello",
		"--max-retries", "-1",
	})
	if err == nil || !strings.Contains(err.Error(), "--max-retries must be >= 0") {
		t.Fatalf("expected retry flag error, got %v", err)
	}
}

func TestRunCLIWritesOutputDir(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "out")
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--cases", casesPath,
		"--output-dir", outputRoot,
		"--run-id", "run-1",
		"--workspace-root", root,
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}

	reportPath := filepath.Join(outputRoot, "run-1", "report.json")
	eventPath := filepath.Join(outputRoot, "run-1", "cases", "case-1", "events.jsonl")
	indexPath := filepath.Join(outputRoot, "index.json")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report file: %v", err)
	}
	if _, err := os.Stat(eventPath); err != nil {
		t.Fatalf("expected event file: %v", err)
	}
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("expected index file: %v", err)
	}
}

func TestRunCLIWritesHTMLReport(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "out")
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--cases", casesPath,
		"--output-dir", outputRoot,
		"--run-id", "run-1",
		"--html-report",
		"--workspace-root", root,
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}

	reportPath := filepath.Join(outputRoot, "run-1", "report.html")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("expected html report file: %v", err)
	}
	if !strings.Contains(string(data), "<!doctype html>") {
		t.Fatalf("expected html document, got %s", string(data))
	}
}

func TestRunCLIHTMLReportRequiresOutputDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--prompt", "hello",
		"--html-report",
	})
	if err == nil || !strings.Contains(err.Error(), "--output-dir is required") {
		t.Fatalf("expected output-dir error, got %v", err)
	}
}

func TestRunCLINoOutputDirDoesNotWriteFiles(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--cases", casesPath,
		"--workspace-root", root,
	})
	if err != nil {
		t.Fatalf("unexpected cli error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "run-1")); !os.IsNotExist(err) {
		t.Fatalf("expected no generated run directory, got err=%v", err)
	}
}

func TestRunCLIOutputDirFileFails(t *testing.T) {
	root := t.TempDir()
	outputRoot := filepath.Join(root, "out-file")
	if err := os.WriteFile(outputRoot, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("unexpected output root file write error: %v", err)
	}
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected cases write error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := runCLI(context.Background(), &stdout, &stderr, []string{
		"--mode", "single",
		"--cases", casesPath,
		"--output-dir", outputRoot,
		"--run-id", "run-1",
		"--workspace-root", root,
	})
	if err == nil {
		t.Fatalf("expected output directory error")
	}
	if !strings.Contains(err.Error(), "create run output dir") {
		t.Fatalf("expected clear output dir error, got %v", err)
	}
}

func TestRunCLIServeRequiresOutputDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runCLI(context.Background(), &stdout, &stderr, []string{"--serve"})
	if err == nil || !strings.Contains(err.Error(), "--output-dir is required") {
		t.Fatalf("expected output-dir error, got %v", err)
	}
}

func TestRunCLIServeModeStartsAPI(t *testing.T) {
	root := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected listen error: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("unexpected listener close error: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		done <- runCLI(context.Background(), &stdout, &stderr, []string{
			"--serve",
			"--output-dir", root,
			"--listen", addr,
		})
	}()

	var response *http.Response
	for i := 0; i < 50; i++ {
		response, err = http.Get("http://" + addr + "/healthz")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("server did not start: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected health status: %d", response.StatusCode)
	}

	select {
	case err := <-done:
		if err == nil || !errors.Is(err, http.ErrServerClosed) {
			t.Fatalf("serve returned unexpectedly: %v", err)
		}
	default:
	}
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
