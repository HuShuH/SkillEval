package spec

// TestCase describes one evaluation case loaded from JSON or JSONL.
type TestCase struct {
	CaseID         string     `json:"case_id"`
	Prompt         string     `json:"prompt"`
	AllowedTools   []string   `json:"allowed_tools"`
	Skill          SkillRef   `json:"skill"`
	HardChecks     HardChecks `json:"hard_checks"`
	TimeoutSeconds int        `json:"timeout_seconds"`
}

// SkillRef identifies which skill should be used for a testcase.
type SkillRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// SkillSpec describes minimal skill metadata for the MVP registry.
type SkillSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HardChecks defines deterministic rule-based expectations for a testcase.
type HardChecks struct {
	ExpectedOutput   string                 `json:"expected_output,omitempty"`
	ExpectedToolName string                 `json:"expected_tool_name,omitempty"`
	ExpectedArgs     map[string]interface{} `json:"expected_args,omitempty"`
}

// ToolCall represents one tool invocation emitted by the adapter.
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Args     map[string]interface{} `json:"args,omitempty"`
}

// AgentOutput represents the execution output returned by the adapter.
type AgentOutput struct {
	FinalOutput string     `json:"final_output"`
	ToolCalls   []ToolCall `json:"tool_calls,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// RunResult represents the checked result of executing one testcase.
type RunResult struct {
	CaseID      string      `json:"case_id"`
	Skill       SkillRef    `json:"skill"`
	AgentOutput AgentOutput `json:"agent_output"`
	Passed      bool        `json:"passed"`
	Reasons     []string    `json:"reasons,omitempty"`
	Error       string      `json:"error,omitempty"`
	DurationMS  int64       `json:"duration_ms"`
}

// ReportSummary aggregates run results for final reporting.
type ReportSummary struct {
	Total       int         `json:"total"`
	Passed      int         `json:"passed"`
	Failed      int         `json:"failed"`
	PassRate    float64     `json:"pass_rate"`
	GeneratedAt string      `json:"generated_at"`
	Results     []RunResult `json:"results,omitempty"`
	Error       string      `json:"error,omitempty"`
}
