package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"agent-skill-eval-go/agent"
	"agent-skill-eval-go/eval"
	"agent-skill-eval-go/providers"
	"agent-skill-eval-go/server"
	"agent-skill-eval-go/skill"
	"agent-skill-eval-go/tool"
)

type cliOptions struct {
	ConfigPath           string
	PrintEffectiveConfig bool
	RebuildIndex         bool
	ListRuns             bool
	ArchiveRuns          string
	DeleteRuns           string
	PruneKeep            int
	PruneStatus          string
	DryRun               bool
	CasesPath            string
	Prompt               string
	SkillAPath           string
	SkillBPath           string
	OutputFormat         string
	Mode                 string
	MaxIters             int
	WorkspaceRoot        string
	OutputDir            string
	HTMLReport           bool
	RunID                string
	Serve                bool
	Stream               bool
	Listen               string
	ProviderName         string
	Model                string
	BaseURL              string
	APIKey               string
	APIKeyEnv            string
	RunTimeout           time.Duration
	ProviderTimeout      time.Duration
	MaxRetries           int
	RetryBackoffMS       int
}

func main() {
	if err := runCLI(context.Background(), os.Stdout, os.Stderr, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCLI(ctx context.Context, stdout io.Writer, stderr io.Writer, args []string) error {
	fs := flag.NewFlagSet("agent-skill-eval", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		casesPath       = fs.String("cases", "", "path to case file in JSON or JSONL")
		configPath      = fs.String("config", "", "path to run config JSON file")
		printEffective  = fs.Bool("print-effective-config", false, "print the effective config JSON and exit")
		rebuildIndex    = fs.Bool("rebuild-index", false, "rebuild run index under --output-dir and exit")
		listRuns        = fs.Bool("list-runs", false, "list indexed runs under --output-dir and exit")
		archiveRuns     = fs.String("archive-runs", "", "comma-separated run ids to archive under --output-dir")
		deleteRuns      = fs.String("delete-runs", "", "comma-separated run ids to delete under --output-dir")
		pruneKeep       = fs.Int("prune-keep", -1, "keep only the newest N runs under --output-dir")
		pruneStatus     = fs.String("prune-status", "all", "prune/list status filter: all|failed|errored|timed_out|passed")
		dryRun          = fs.Bool("dry-run", false, "preview management actions without changing files")
		prompt          = fs.String("prompt", "", "single quick-run prompt")
		skillAPath      = fs.String("skill-a", "", "path to skill A JSON stub")
		skillBPath      = fs.String("skill-b", "", "path to skill B JSON stub")
		outputFormat    = fs.String("output-format", "text", "output format: text or json")
		mode            = fs.String("mode", "single", "run mode: single or pair")
		maxIters        = fs.Int("max-iters", 3, "maximum agent iterations")
		workspaceRoot   = fs.String("workspace-root", "", "workspace root for tools")
		outputDir       = fs.String("output-dir", "", "optional output root for report.json and events.jsonl")
		htmlReport      = fs.Bool("html-report", false, "also export a static report.html into the output directory")
		runID           = fs.String("run-id", "", "optional run id for output directory")
		serve           = fs.Bool("serve", false, "serve read-only API from output directory")
		stream          = fs.Bool("stream", false, "start a temporary local API server while this run executes")
		listen          = fs.String("listen", ":8080", "HTTP listen address for --serve")
		providerName    = fs.String("provider", "stub", "provider: stub or openai")
		model           = fs.String("model", "", "provider model name placeholder")
		baseURL         = fs.String("base-url", "", "provider base URL placeholder")
		apiKey          = fs.String("api-key", "", "provider API key placeholder")
		runTimeout      = fs.Duration("timeout", 0, "optional overall run timeout")
		providerTimeout = fs.Duration("provider-timeout", 30*time.Second, "provider HTTP timeout")
		maxRetries      = fs.Int("max-retries", 0, "maximum provider retries for retryable errors")
		retryBackoffMS  = fs.Int("retry-backoff-ms", 200, "base retry backoff in milliseconds")
	)

	if err := fs.Parse(args); err != nil {
		return err
	}

	explicitFlags := collectExplicitFlags(fs)
	effective := defaultCLIOptions()

	if strings.TrimSpace(*configPath) != "" {
		cfg, err := eval.LoadRunConfig(*configPath)
		if err != nil {
			return err
		}
		applyRunConfigFile(&effective, *cfg)
	}
	applyFlagOverrides(&effective, explicitFlags, flagInputs{
		configPath:      *configPath,
		printEffective:  *printEffective,
		rebuildIndex:    *rebuildIndex,
		listRuns:        *listRuns,
		archiveRuns:     *archiveRuns,
		deleteRuns:      *deleteRuns,
		pruneKeep:       *pruneKeep,
		pruneStatus:     *pruneStatus,
		dryRun:          *dryRun,
		casesPath:       *casesPath,
		prompt:          *prompt,
		skillAPath:      *skillAPath,
		skillBPath:      *skillBPath,
		outputFormat:    *outputFormat,
		mode:            *mode,
		maxIters:        *maxIters,
		workspaceRoot:   *workspaceRoot,
		outputDir:       *outputDir,
		htmlReport:      *htmlReport,
		runID:           *runID,
		serve:           *serve,
		stream:          *stream,
		listen:          *listen,
		providerName:    *providerName,
		model:           *model,
		baseURL:         *baseURL,
		apiKey:          *apiKey,
		runTimeout:      *runTimeout,
		providerTimeout: *providerTimeout,
		maxRetries:      *maxRetries,
		retryBackoffMS:  *retryBackoffMS,
	})

	if effective.PrintEffectiveConfig {
		return printEffectiveConfig(stdout, effective)
	}

	if managementRequested(effective) {
		return runManagement(stdout, effective)
	}

	if effective.Serve {
		if strings.TrimSpace(effective.OutputDir) == "" {
			return errors.New("--output-dir is required in --serve mode")
		}
		return server.New(effective.OutputDir).ListenAndServe(effective.Listen)
	}

	if err := validateCLIOptions(effective); err != nil {
		return err
	}

	if effective.Mode != "single" && effective.Mode != "pair" {
		return fmt.Errorf("invalid mode %q", effective.Mode)
	}
	if effective.OutputFormat != "text" && effective.OutputFormat != "json" {
		return fmt.Errorf("invalid output format %q", effective.OutputFormat)
	}
	if effective.ProviderName != "stub" && effective.ProviderName != "openai" {
		return fmt.Errorf("invalid provider %q", effective.ProviderName)
	}
	if effective.HTMLReport && strings.TrimSpace(effective.OutputDir) == "" {
		return errors.New("--output-dir is required when --html-report is enabled")
	}
	if effective.MaxRetries < 0 {
		return errors.New("--max-retries must be >= 0")
	}
	if effective.RetryBackoffMS < 0 {
		return errors.New("--retry-backoff-ms must be >= 0")
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if effective.RunTimeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, effective.RunTimeout)
		defer cancel()
	}

	cases, err := loadInputCases(effective.CasesPath, effective.Prompt)
	if err != nil {
		return err
	}

	skillA, err := loadCLISkill(effective.SkillAPath, "skill-a")
	if err != nil {
		return err
	}
	skillA, err = attachDefaultTools(skillA, effective.WorkspaceRoot)
	if err != nil {
		return err
	}

	var liveServer *server.Server
	var httpServer *http.Server
	if effective.Stream {
		if strings.TrimSpace(effective.OutputDir) == "" {
			return errors.New("--output-dir is required in --stream mode")
		}
		liveServer = server.New(effective.OutputDir)
		httpServer = &http.Server{
			Addr:    effective.Listen,
			Handler: liveServer.Handler(),
		}
		go func() {
			_ = httpServer.ListenAndServe()
		}()
		time.Sleep(25 * time.Millisecond)
	}

	effectiveRunID := effective.RunID
	if strings.TrimSpace(effectiveRunID) == "" {
		effectiveRunID = "cli-run"
	}
	if liveServer != nil {
		liveServer.Hub.StartRun(effectiveRunID)
	}

	runner := eval.Runner{
		Config: eval.RunConfig{
			WorkspaceRoot: effective.WorkspaceRoot,
			EventSink: func(caseID string, side string, event agent.Event) {
				if liveServer == nil {
					return
				}
				liveServer.Hub.Publish(server.RunStreamEvent{
					RunID:     effectiveRunID,
					CaseID:    caseID,
					Side:      side,
					EventType: event.Type,
					Event:     event,
					Timestamp: time.Now().UTC(),
				})
			},
		},
	}
	defer func() {
		if liveServer != nil {
			liveServer.Hub.CompleteRun(effectiveRunID)
		}
		if httpServer != nil {
			_ = httpServer.Close()
		}
	}()

	agentA := agent.Agent{
		Name:     "agent-a",
		Provider: nil,
		ProviderConfig: providers.Config{
			Model:        effective.Model,
			BaseURL:      effective.BaseURL,
			APIKey:       effective.APIKey,
			Timeout:      effective.ProviderTimeout,
			MaxRetries:   effective.MaxRetries,
			RetryBackoff: time.Duration(effective.RetryBackoffMS) * time.Millisecond,
		},
		MaxIterations: effective.MaxIters,
		Instructions:  skillA.Instructions,
		SystemPrompt:  skillA.SystemPrompt,
	}
	agentA.Provider, err = buildProvider(effective.ProviderName, agentA.ProviderConfig, "A")
	if err != nil {
		return err
	}

	if effective.Mode == "single" {
		report, runErr := runner.RunCases(runCtx, agentA, skillA, cases)
		report.Metadata = map[string]string{
			"provider_mode": effective.ProviderName,
			"skill_loader":  "temporary_json_or_default",
			"model":         effective.Model,
			"skill_a":       effective.SkillAPath,
		}
		if effective.RunTimeout > 0 {
			report.Metadata["run_timeout"] = effective.RunTimeout.String()
		}
		if strings.TrimSpace(effective.OutputDir) != "" {
			store := eval.NewOutputStore(effective.OutputDir, effectiveRunID)
			if _, err := store.WriteRunReport(report); err != nil {
				return err
			}
			if effective.HTMLReport {
				if _, err := store.WriteRunReportHTML(report); err != nil {
					return err
				}
			}
			if err := rebuildRunIndex(effective.OutputDir); err != nil {
				return err
			}
		}
		return writeRunOutput(stdout, effective.OutputFormat, report, runErr)
	}

	skillB, err := loadCLISkill(effective.SkillBPath, "skill-b")
	if err != nil {
		return err
	}
	skillB, err = attachDefaultTools(skillB, effective.WorkspaceRoot)
	if err != nil {
		return err
	}

	agentB := agent.Agent{
		Name:     "agent-b",
		Provider: nil,
		ProviderConfig: providers.Config{
			Model:        effective.Model,
			BaseURL:      effective.BaseURL,
			APIKey:       effective.APIKey,
			Timeout:      effective.ProviderTimeout,
			MaxRetries:   effective.MaxRetries,
			RetryBackoff: time.Duration(effective.RetryBackoffMS) * time.Millisecond,
		},
		MaxIterations: effective.MaxIters,
		Instructions:  skillB.Instructions,
		SystemPrompt:  skillB.SystemPrompt,
	}
	agentB.Provider, err = buildProvider(effective.ProviderName, agentB.ProviderConfig, "B")
	if err != nil {
		return err
	}

	report, runErr := runner.RunCasePairs(runCtx, cases, agentA, skillA, agentB, skillB)
	report.Metadata = map[string]string{
		"provider_mode": effective.ProviderName,
		"skill_loader":  "temporary_json_or_default",
		"model":         effective.Model,
		"skill_a":       effective.SkillAPath,
		"skill_b":       effective.SkillBPath,
	}
	if effective.RunTimeout > 0 {
		report.Metadata["run_timeout"] = effective.RunTimeout.String()
	}
	if strings.TrimSpace(effective.OutputDir) != "" {
		store := eval.NewOutputStore(effective.OutputDir, effectiveRunID)
		if _, err := store.WritePairReport(report); err != nil {
			return err
		}
		if effective.HTMLReport {
			if _, err := store.WritePairReportHTML(report); err != nil {
				return err
			}
		}
		if err := rebuildRunIndex(effective.OutputDir); err != nil {
			return err
		}
	}
	return writePairOutput(stdout, effective.OutputFormat, report, runErr)
}

type flagInputs struct {
	configPath      string
	printEffective  bool
	rebuildIndex    bool
	listRuns        bool
	archiveRuns     string
	deleteRuns      string
	pruneKeep       int
	pruneStatus     string
	dryRun          bool
	casesPath       string
	prompt          string
	skillAPath      string
	skillBPath      string
	outputFormat    string
	mode            string
	maxIters        int
	workspaceRoot   string
	outputDir       string
	htmlReport      bool
	runID           string
	serve           bool
	stream          bool
	listen          string
	providerName    string
	model           string
	baseURL         string
	apiKey          string
	runTimeout      time.Duration
	providerTimeout time.Duration
	maxRetries      int
	retryBackoffMS  int
}

func defaultCLIOptions() cliOptions {
	return cliOptions{
		OutputFormat:    "text",
		Mode:            "single",
		PruneKeep:       -1,
		PruneStatus:     "all",
		MaxIters:        3,
		Listen:          ":8080",
		ProviderName:    "stub",
		ProviderTimeout: 30 * time.Second,
		RetryBackoffMS:  200,
	}
}

func collectExplicitFlags(fs *flag.FlagSet) map[string]bool {
	explicit := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		explicit[f.Name] = true
	})
	return explicit
}

func applyRunConfigFile(target *cliOptions, cfg eval.RunConfigFile) {
	if value := strings.TrimSpace(cfg.Mode); value != "" {
		target.Mode = value
	}
	if value := strings.TrimSpace(cfg.Cases); value != "" {
		target.CasesPath = value
	}
	if value := strings.TrimSpace(cfg.Prompt); value != "" {
		target.Prompt = value
	}
	if value := strings.TrimSpace(cfg.Provider.Name); value != "" {
		target.ProviderName = value
	}
	if value := strings.TrimSpace(cfg.Provider.Model); value != "" {
		target.Model = value
	}
	if value := strings.TrimSpace(cfg.Provider.BaseURL); value != "" {
		target.BaseURL = value
	}
	if value := strings.TrimSpace(cfg.Provider.APIKey); value != "" {
		target.APIKey = value
	}
	if value := strings.TrimSpace(cfg.Provider.APIKeyEnv); value != "" {
		target.APIKeyEnv = value
		target.APIKey = strings.TrimSpace(os.Getenv(value))
	}
	if duration, err := cfg.RunTimeout(); err == nil && duration > 0 {
		target.RunTimeout = duration
	}
	if duration, err := cfg.ProviderTimeout(); err == nil && duration > 0 {
		target.ProviderTimeout = duration
	}
	if cfg.Execution.MaxRetries != 0 {
		target.MaxRetries = cfg.Execution.MaxRetries
	}
	if cfg.Execution.RetryBackoffMS != 0 {
		target.RetryBackoffMS = cfg.Execution.RetryBackoffMS
	}
	if cfg.Execution.MaxIters != 0 {
		target.MaxIters = cfg.Execution.MaxIters
	}
	if value := strings.TrimSpace(cfg.Execution.WorkspaceRoot); value != "" {
		target.WorkspaceRoot = value
	}
	if value := strings.TrimSpace(cfg.Output.OutputDir); value != "" {
		target.OutputDir = value
	}
	if value := strings.TrimSpace(cfg.Output.RunID); value != "" {
		target.RunID = value
	}
	target.HTMLReport = cfg.Output.HTMLReport
	if value := strings.TrimSpace(cfg.Skills.SkillA); value != "" {
		target.SkillAPath = value
	}
	if value := strings.TrimSpace(cfg.Skills.SkillB); value != "" {
		target.SkillBPath = value
	}
}

func applyFlagOverrides(target *cliOptions, explicit map[string]bool, values flagInputs) {
	if explicit["config"] {
		target.ConfigPath = values.configPath
	}
	if explicit["print-effective-config"] {
		target.PrintEffectiveConfig = values.printEffective
	}
	if explicit["rebuild-index"] {
		target.RebuildIndex = values.rebuildIndex
	}
	if explicit["list-runs"] {
		target.ListRuns = values.listRuns
	}
	if explicit["archive-runs"] {
		target.ArchiveRuns = values.archiveRuns
	}
	if explicit["delete-runs"] {
		target.DeleteRuns = values.deleteRuns
	}
	if explicit["prune-keep"] {
		target.PruneKeep = values.pruneKeep
	}
	if explicit["prune-status"] {
		target.PruneStatus = values.pruneStatus
	}
	if explicit["dry-run"] {
		target.DryRun = values.dryRun
	}
	if explicit["cases"] {
		target.CasesPath = values.casesPath
	}
	if explicit["prompt"] {
		target.Prompt = values.prompt
	}
	if explicit["skill-a"] {
		target.SkillAPath = values.skillAPath
	}
	if explicit["skill-b"] {
		target.SkillBPath = values.skillBPath
	}
	if explicit["output-format"] {
		target.OutputFormat = values.outputFormat
	}
	if explicit["mode"] {
		target.Mode = values.mode
	}
	if explicit["max-iters"] {
		target.MaxIters = values.maxIters
	}
	if explicit["workspace-root"] {
		target.WorkspaceRoot = values.workspaceRoot
	}
	if explicit["output-dir"] {
		target.OutputDir = values.outputDir
	}
	if explicit["html-report"] {
		target.HTMLReport = values.htmlReport
	}
	if explicit["run-id"] {
		target.RunID = values.runID
	}
	if explicit["serve"] {
		target.Serve = values.serve
	}
	if explicit["stream"] {
		target.Stream = values.stream
	}
	if explicit["listen"] {
		target.Listen = values.listen
	}
	if explicit["provider"] {
		target.ProviderName = values.providerName
	}
	if explicit["model"] {
		target.Model = values.model
	}
	if explicit["base-url"] {
		target.BaseURL = values.baseURL
	}
	if explicit["api-key"] {
		target.APIKey = values.apiKey
		target.APIKeyEnv = ""
	}
	if explicit["timeout"] {
		target.RunTimeout = values.runTimeout
	}
	if explicit["provider-timeout"] {
		target.ProviderTimeout = values.providerTimeout
	}
	if explicit["max-retries"] {
		target.MaxRetries = values.maxRetries
	}
	if explicit["retry-backoff-ms"] {
		target.RetryBackoffMS = values.retryBackoffMS
	}
}

func printEffectiveConfig(w io.Writer, options cliOptions) error {
	safe := map[string]any{
		"mode": options.Mode,
		"management": map[string]any{
			"rebuild_index": options.RebuildIndex,
			"list_runs":     options.ListRuns,
			"archive_runs":  options.ArchiveRuns,
			"delete_runs":   options.DeleteRuns,
			"prune_keep":    options.PruneKeep,
			"prune_status":  options.PruneStatus,
			"dry_run":       options.DryRun,
		},
		"cases":  options.CasesPath,
		"prompt": options.Prompt,
		"provider": map[string]any{
			"name":             options.ProviderName,
			"model":            options.Model,
			"base_url":         options.BaseURL,
			"api_key_env":      options.APIKeyEnv,
			"api_key":          redactSecret(options.APIKey),
			"provider_timeout": options.ProviderTimeout.String(),
		},
		"execution": map[string]any{
			"timeout":          options.RunTimeout.String(),
			"max_retries":      options.MaxRetries,
			"retry_backoff_ms": options.RetryBackoffMS,
			"max_iters":        options.MaxIters,
			"workspace_root":   options.WorkspaceRoot,
		},
		"output": map[string]any{
			"output_dir":  options.OutputDir,
			"run_id":      options.RunID,
			"html_report": options.HTMLReport,
			"format":      options.OutputFormat,
		},
		"skills": map[string]any{
			"skill_a": options.SkillAPath,
			"skill_b": options.SkillBPath,
		},
	}
	encoded, err := json.MarshalIndent(safe, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(encoded))
	return err
}

func redactSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "REDACTED"
}

func validateCLIOptions(options cliOptions) error {
	if options.MaxRetries < 0 {
		return errors.New("--max-retries must be >= 0")
	}
	if options.RetryBackoffMS < 0 {
		return errors.New("--retry-backoff-ms must be >= 0")
	}
	if options.HTMLReport && strings.TrimSpace(options.OutputDir) == "" {
		return errors.New("--output-dir is required when --html-report is enabled")
	}
	if options.ProviderName == "openai" {
		if strings.TrimSpace(options.Model) == "" {
			return errors.New("--model is required when --provider=openai")
		}
		if strings.TrimSpace(options.BaseURL) == "" {
			return errors.New("--base-url is required when --provider=openai")
		}
		if strings.TrimSpace(options.APIKey) == "" && strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
			return errors.New("--api-key or OPENAI_API_KEY is required when --provider=openai")
		}
	}

	cfg := eval.RunConfigFile{
		Mode:   options.Mode,
		Cases:  options.CasesPath,
		Prompt: options.Prompt,
		Provider: eval.ProviderConfigFile{
			Name:            options.ProviderName,
			Model:           options.Model,
			BaseURL:         options.BaseURL,
			APIKeyEnv:       options.APIKeyEnv,
			APIKey:          options.APIKey,
			ProviderTimeout: durationString(options.ProviderTimeout),
		},
		Execution: eval.ExecutionConfig{
			Timeout:        durationString(options.RunTimeout),
			MaxRetries:     options.MaxRetries,
			RetryBackoffMS: options.RetryBackoffMS,
			MaxIters:       options.MaxIters,
			WorkspaceRoot:  options.WorkspaceRoot,
		},
		Output: eval.OutputConfig{
			OutputDir:  options.OutputDir,
			RunID:      options.RunID,
			HTMLReport: options.HTMLReport,
		},
		Skills: eval.SkillConfig{
			SkillA: options.SkillAPath,
			SkillB: options.SkillBPath,
		},
	}
	return cfg.Validate()
}

func durationString(value time.Duration) string {
	if value <= 0 {
		return ""
	}
	return value.String()
}

func rebuildRunIndex(outputRoot string) error {
	if strings.TrimSpace(outputRoot) == "" {
		return nil
	}
	_, err := eval.RebuildAndWriteRunIndex(outputRoot)
	return err
}

func managementRequested(options cliOptions) bool {
	return options.RebuildIndex || options.ListRuns || strings.TrimSpace(options.ArchiveRuns) != "" || strings.TrimSpace(options.DeleteRuns) != "" || options.PruneKeep >= 0
}

func runManagement(stdout io.Writer, options cliOptions) error {
	if strings.TrimSpace(options.OutputDir) == "" {
		return errors.New("--output-dir is required for management commands")
	}
	switch {
	case options.RebuildIndex:
		if err := eval.RebuildIndex(options.OutputDir); err != nil {
			return err
		}
		return writeManagementOutput(stdout, options.OutputFormat, map[string]any{
			"action": "rebuild_index",
			"ok":     true,
		})
	case options.ListRuns:
		runs, err := eval.ListRuns(options.OutputDir, eval.ListFilter{Status: options.PruneStatus})
		if err != nil {
			return err
		}
		return writeManagementOutput(stdout, options.OutputFormat, map[string]any{
			"action": "list_runs",
			"runs":   runs,
		})
	case strings.TrimSpace(options.ArchiveRuns) != "":
		result, err := eval.ArchiveRuns(options.OutputDir, splitCSV(options.ArchiveRuns), options.DryRun)
		if err != nil {
			return err
		}
		return writeManagementOutput(stdout, options.OutputFormat, result)
	case strings.TrimSpace(options.DeleteRuns) != "":
		result, err := eval.DeleteRuns(options.OutputDir, splitCSV(options.DeleteRuns), options.DryRun)
		if err != nil {
			return err
		}
		return writeManagementOutput(stdout, options.OutputFormat, result)
	case options.PruneKeep >= 0:
		result, err := eval.PruneRuns(options.OutputDir, options.PruneKeep, options.PruneStatus, options.DryRun)
		if err != nil {
			return err
		}
		return writeManagementOutput(stdout, options.OutputFormat, result)
	default:
		return nil
	}
}

func writeManagementOutput(w io.Writer, format string, value any) error {
	if format == "json" {
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(encoded))
		return err
	}
	switch current := value.(type) {
	case eval.ManageResult:
		_, err := fmt.Fprintf(w, "%s dry_run=%v affected=%v skipped=%v errors=%v\n", current.Action, current.DryRun, current.Affected, current.Skipped, current.Errors)
		return err
	default:
		_, err := fmt.Fprintln(w, value)
		return err
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func buildProvider(providerName string, config providers.Config, label string) (providers.ChatClient, error) {
	switch providerName {
	case "stub":
		return cliStubProvider{Label: label}, nil
	case "openai":
		if strings.TrimSpace(config.Model) == "" {
			return nil, errors.New("--model is required when --provider=openai")
		}
		if strings.TrimSpace(config.BaseURL) == "" {
			return nil, errors.New("--base-url is required when --provider=openai")
		}
		if strings.TrimSpace(config.APIKey) == "" && strings.TrimSpace(os.Getenv("OPENAI_API_KEY")) == "" {
			return nil, errors.New("--api-key or OPENAI_API_KEY is required when --provider=openai")
		}
		return providers.OpenAIClient{Config: config}, nil
	default:
		return nil, fmt.Errorf("invalid provider %q", providerName)
	}
}

func loadInputCases(path string, prompt string) ([]eval.Case, error) {
	if strings.TrimSpace(path) != "" {
		return eval.LoadCases(path)
	}
	if strings.TrimSpace(prompt) != "" {
		c, err := eval.NewCase("quick-run", prompt, "")
		if err != nil {
			return nil, err
		}
		return []eval.Case{c}, nil
	}
	return nil, errors.New("either --cases or --prompt is required")
}

type cliSkillFile struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Tools        []string `json:"tools"`
	SystemPrompt string   `json:"system_prompt"`
	Instructions string   `json:"instructions"`
}

func loadCLISkill(path string, fallbackName string) (skill.Skill, error) {
	if strings.TrimSpace(path) == "" {
		return skill.NewSkill(fallbackName, "", "temporary CLI skill", []string{"finish"}, "", "temporary cli instructions")
	}

	loaded, err := skill.LoadSkill(path)
	if err == nil {
		return *loaded, nil
	}

	data, fileErr := os.ReadFile(path)
	if fileErr != nil {
		return skill.Skill{}, err
	}

	var input cliSkillFile
	if err := json.Unmarshal(data, &input); err != nil {
		return skill.Skill{}, err
	}

	return skill.NewSkill(input.Name, input.Version, input.Description, input.Tools, input.SystemPrompt, input.Instructions)
}

func attachDefaultTools(s skill.Skill, workspaceRoot string) (skill.Skill, error) {
	resolved := []tool.Tool{
		tool.FinishTool{},
		tool.FilesystemTool{Config: tool.FilesystemConfig{WorkspaceRoot: workspaceRoot}},
	}

	if len(s.Tools) == 0 {
		s.Tools = []string{"finish"}
	}
	return s.AttachResolvedTools(resolved)
}

func writeRunOutput(w io.Writer, format string, report eval.RunReport, runErr error) error {
	switch format {
	case "json":
		encoded, err := eval.EncodeRunReport(report)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, string(encoded)); err != nil {
			return err
		}
	default:
		if _, err := fmt.Fprintln(w, eval.FormatRunSummary(report)); err != nil {
			return err
		}
	}
	return runErr
}

func writePairOutput(w io.Writer, format string, report eval.PairReport, runErr error) error {
	switch format {
	case "json":
		encoded, err := eval.EncodePairReport(report)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w, string(encoded)); err != nil {
			return err
		}
	default:
		if _, err := fmt.Fprintln(w, eval.FormatPairSummary(report)); err != nil {
			return err
		}
	}
	return runErr
}

type cliStubProvider struct {
	Label string
}

func (p cliStubProvider) ChatCompletion(ctx context.Context, req providers.ChatRequest) (providers.ChatResponse, error) {
	prompt := ""
	for index := len(req.Messages) - 1; index >= 0; index-- {
		if req.Messages[index].Role == "user" {
			prompt = req.Messages[index].Content
			break
		}
	}

	answer := fmt.Sprintf("[%s stub provider] %s", p.Label, prompt)
	return providers.ChatResponse{
		Finish: &providers.FinishSignal{
			FinalAnswer: answer,
			Reason:      "stub_provider",
		},
	}, nil
}
