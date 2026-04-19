// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal JSON-based run configuration loading and validation.
package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// RunConfigFile describes one reproducible single/pair run configuration.
type RunConfigFile struct {
	Mode      string             `json:"mode"`
	Cases     string             `json:"cases,omitempty"`
	Prompt    string             `json:"prompt,omitempty"`
	Provider  ProviderConfigFile `json:"provider"`
	Execution ExecutionConfig    `json:"execution"`
	Output    OutputConfig       `json:"output"`
	Skills    SkillConfig        `json:"skills"`
}

// ProviderConfigFile contains provider-specific settings.
type ProviderConfigFile struct {
	Name            string `json:"name"`
	Model           string `json:"model,omitempty"`
	BaseURL         string `json:"base_url,omitempty"`
	APIKeyEnv       string `json:"api_key_env,omitempty"`
	APIKey          string `json:"api_key,omitempty"`
	ProviderTimeout string `json:"provider_timeout,omitempty"`
}

// ExecutionConfig contains execution and retry settings.
type ExecutionConfig struct {
	Timeout        string `json:"timeout,omitempty"`
	MaxRetries     int    `json:"max_retries,omitempty"`
	RetryBackoffMS int    `json:"retry_backoff_ms,omitempty"`
	MaxIters       int    `json:"max_iters,omitempty"`
	WorkspaceRoot  string `json:"workspace_root,omitempty"`
}

// OutputConfig contains report output settings.
type OutputConfig struct {
	OutputDir  string `json:"output_dir,omitempty"`
	RunID      string `json:"run_id,omitempty"`
	HTMLReport bool   `json:"html_report,omitempty"`
}

// SkillConfig contains skill input paths.
type SkillConfig struct {
	SkillA string `json:"skill_a,omitempty"`
	SkillB string `json:"skill_b,omitempty"`
}

// LoadRunConfig loads and validates one JSON run configuration file.
func LoadRunConfig(path string) (*RunConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read run config %q: %w", path, err)
	}

	var cfg RunConfigFile
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode run config %q: %w", path, err)
	}
	return &cfg, nil
}

// Validate checks the minimum required run configuration fields.
func (c RunConfigFile) Validate() error {
	if c.Mode != "" && c.Mode != "single" && c.Mode != "pair" {
		return fmt.Errorf("invalid mode %q", c.Mode)
	}
	if c.Provider.Name != "" && c.Provider.Name != "stub" && c.Provider.Name != "openai" {
		return fmt.Errorf("invalid provider %q", c.Provider.Name)
	}
	if c.Mode == "pair" && strings.TrimSpace(c.Skills.SkillB) == "" {
		return fmt.Errorf("skill_b is required in pair mode")
	}
	if strings.TrimSpace(c.Cases) == "" && strings.TrimSpace(c.Prompt) == "" {
		return fmt.Errorf("either cases or prompt is required")
	}
	if strings.TrimSpace(c.Cases) != "" {
		if _, err := os.Stat(c.Cases); err != nil {
			return fmt.Errorf("stat cases path %q: %w", c.Cases, err)
		}
	}
	if c.Output.HTMLReport && strings.TrimSpace(c.Output.OutputDir) == "" {
		return fmt.Errorf("output_dir is required when html_report is enabled")
	}
	if c.Execution.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0")
	}
	if c.Execution.RetryBackoffMS < 0 {
		return fmt.Errorf("retry_backoff_ms must be >= 0")
	}
	if c.Execution.MaxIters < 0 {
		return fmt.Errorf("max_iters must be >= 0")
	}
	if _, err := parseOptionalDuration(c.Execution.Timeout, "timeout"); err != nil {
		return err
	}
	if _, err := parseOptionalDuration(c.Provider.ProviderTimeout, "provider_timeout"); err != nil {
		return err
	}
	if c.Provider.Name == "openai" {
		if strings.TrimSpace(c.Provider.Model) == "" {
			return fmt.Errorf("model is required when provider is openai")
		}
		if strings.TrimSpace(c.Provider.BaseURL) == "" {
			return fmt.Errorf("base_url is required when provider is openai")
		}
		if strings.TrimSpace(c.Provider.APIKey) == "" && strings.TrimSpace(c.Provider.APIKeyEnv) == "" {
			return fmt.Errorf("api_key or api_key_env is required when provider is openai")
		}
		if envName := strings.TrimSpace(c.Provider.APIKeyEnv); envName != "" && strings.TrimSpace(os.Getenv(envName)) == "" {
			return fmt.Errorf("api_key_env %q is not set", envName)
		}
	}
	return nil
}

// RunTimeout returns the parsed run timeout.
func (c RunConfigFile) RunTimeout() (time.Duration, error) {
	return parseOptionalDuration(c.Execution.Timeout, "timeout")
}

// ProviderTimeout returns the parsed provider timeout.
func (c RunConfigFile) ProviderTimeout() (time.Duration, error) {
	return parseOptionalDuration(c.Provider.ProviderTimeout, "provider_timeout")
}

// EffectiveAPIKey returns the inline key or env-resolved key without exposing it in errors.
func (c RunConfigFile) EffectiveAPIKey() string {
	if key := strings.TrimSpace(c.Provider.APIKey); key != "" {
		return key
	}
	if envName := strings.TrimSpace(c.Provider.APIKeyEnv); envName != "" {
		return strings.TrimSpace(os.Getenv(envName))
	}
	return ""
}

func parseOptionalDuration(value string, field string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: %w", field, value, err)
	}
	if duration < 0 {
		return 0, fmt.Errorf("%s must be >= 0", field)
	}
	return duration, nil
}
