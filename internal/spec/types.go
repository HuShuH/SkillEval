package spec

// TestCase describes one evaluation case loaded from JSON or JSONL.
type TestCase struct {
	CaseID         string     `json:"case_id"`
	Prompt         string     `json:"prompt"`
	AllowedTools   []string   `json:"allowed_tools"`
	Skill          SkillRef   `json:"skill"`
	ExpectedOutput string     `json:"expected_output"`
	HardChecks     HardChecks `json:"hard_checks"`
	TimeoutSeconds int        `json:"timeout_seconds"`
}

// SkillRef identifies which skill should be used for a testcase.
type SkillRef struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// SkillSpec describes skill metadata for registry use in the MVP.
type SkillSpec struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	AllowedTools   []string `json:"allowed_tools,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
}

// HardChecks defines deterministic rule-based expectations for a testcase.
type HardChecks struct {
	ExpectedOutput   string                 `json:"expected_output,omitempty"`
	ExpectedToolName string                 `json:"expected_tool_name,omitempty"`
	ExpectedArgs     map[string]interface{} `json:"expected_args,omitempty"`
}

// ToolCall represents one tool invocation emitted by the adapter or runner.
type ToolCall struct {
	ToolName string                 `json:"tool_name"`
	Args     map[string]interface{} `json:"args,omitempty"`
}

// AgentOutput represents the mock adapter result for a single run.
type AgentOutput struct {
	FinalOutput string     `json:"final_output"`
	ToolCalls   []ToolCall `json:"tool_calls,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// RunResult represents the checked result of executing one testcase.
type RunResult struct {
	CaseID       string      `json:"case_id"`
	Skill        SkillRef    `json:"skill"`
	FinalOutput  string      `json:"final_output,omitempty"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	Passed       bool        `json:"passed"`
	Reasons      []string    `json:"reasons,omitempty"`
	Error        string      `json:"error,omitempty"`
	AgentOutput  AgentOutput `json:"agent_output"`
	TimeoutSeconds int       `json:"timeout_seconds,omitempty"`
}

// ReportSummary aggregates run results for final reporting.
type ReportSummary struct {
	Total   int         `json:"total"`
	Passed  int         `json:"passed"`
	Failed  int         `json:"failed"`
	Results []RunResult `json:"results,omitempty"`
	Error   string      `json:"error,omitempty"`
}
