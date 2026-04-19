package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRunConfigSuccess(t *testing.T) {
	root := t.TempDir()
	casesPath := filepath.Join(root, "cases.json")
	if err := os.WriteFile(casesPath, []byte(`[{"id":"case-1","prompt":"hello"}]`), 0o644); err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	t.Setenv("OPENAI_API_KEY", "secret")

	configPath := filepath.Join(root, "run.json")
	content := `{
	  "mode": "single",
	  "cases": ` + quoteJSON(casesPath) + `,
	  "provider": {
	    "name": "openai",
	    "model": "gpt-test",
	    "base_url": "https://example.invalid/v1",
	    "api_key_env": "OPENAI_API_KEY",
	    "provider_timeout": "3s"
	  },
	  "execution": {
	    "timeout": "10s",
	    "max_retries": 2,
	    "retry_backoff_ms": 200,
	    "max_iters": 4,
	    "workspace_root": ` + quoteJSON(root) + `
	  },
	  "output": {
	    "output_dir": ` + quoteJSON(filepath.Join(root, "out")) + `,
	    "run_id": "run-1",
	    "html_report": true
	  },
	  "skills": {
	    "skill_a": ` + quoteJSON(filepath.Join(root, "skill-a")) + `
	  }
	}`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("unexpected config write error: %v", err)
	}

	cfg, err := LoadRunConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}
	if cfg.Mode != "single" || cfg.Provider.Name != "openai" || cfg.Execution.MaxIters != 4 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.EffectiveAPIKey() != "secret" {
		t.Fatalf("unexpected effective api key")
	}
}

func TestRunConfigInvalidModeProvider(t *testing.T) {
	cfg := RunConfigFile{Mode: "bad", Prompt: "hello", Provider: ProviderConfigFile{Name: "stub"}}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected invalid mode error")
	}

	cfg = RunConfigFile{Mode: "single", Prompt: "hello", Provider: ProviderConfigFile{Name: "bad"}}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected invalid provider error")
	}
}

func TestRunConfigPairRequiresSkillB(t *testing.T) {
	cfg := RunConfigFile{
		Mode:     "pair",
		Prompt:   "hello",
		Provider: ProviderConfigFile{Name: "stub"},
		Skills:   SkillConfig{SkillA: "/tmp/skill-a"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected missing skill_b error")
	}
}

func TestRunConfigHTMLReportRequiresOutputDir(t *testing.T) {
	cfg := RunConfigFile{
		Mode:     "single",
		Prompt:   "hello",
		Provider: ProviderConfigFile{Name: "stub"},
		Output:   OutputConfig{HTMLReport: true},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected output_dir error")
	}
}

func TestRunConfigOpenAIRequiresFields(t *testing.T) {
	cfg := RunConfigFile{
		Mode:     "single",
		Prompt:   "hello",
		Provider: ProviderConfigFile{Name: "openai"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected openai config error")
	}
}

func TestRunConfigNegativeExecutionValuesFail(t *testing.T) {
	cfg := RunConfigFile{
		Mode:     "single",
		Prompt:   "hello",
		Provider: ProviderConfigFile{Name: "stub"},
		Execution: ExecutionConfig{
			MaxRetries:     -1,
			RetryBackoffMS: 0,
			MaxIters:       1,
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected max_retries error")
	}
}

func TestExampleConfigsLoad(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(".."); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	t.Setenv("OPENAI_API_KEY", "example-secret")
	for _, path := range []string{
		"configs/single.stub.json",
		"configs/single.openai.json",
		"configs/pair.stub.json",
		"configs/pair.openai.json",
	} {
		cfg, err := LoadRunConfig(path)
		if err != nil {
			t.Fatalf("load example config %s: %v", path, err)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("validate example config %s: %v", path, err)
		}
	}
}

func quoteJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
