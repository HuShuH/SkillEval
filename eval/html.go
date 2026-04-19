// Package eval contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal static HTML report rendering for offline review.
package eval

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"agent-skill-eval-go/agent"
)

type htmlPageData struct {
	Title       string
	ReportID    string
	CreatedAt   string
	TotalCases  int
	IsPair      bool
	RunSummary  *RunSummary
	PairSummary *PairSummary
	RunResults  []runCaseView
	PairResults []pairCaseView
}

type runCaseView struct {
	CaseID             string
	AgentName          string
	StopReason         string
	Passed             bool
	FinalAnswer        string
	Error              string
	ErrorClass         string
	FailedIteration    int
	Iterations         int
	ToolExecutionCount int
	EventSummary       []eventSummaryView
	StatusClass        string
}

type pairCaseView struct {
	CaseID      string
	Error       string
	ScoreReason string
	SideA       runCaseView
	SideB       runCaseView
	StatusClass string
}

type eventSummaryView struct {
	Type        string
	Message     string
	Iteration   int
	StatusClass string
}

// RenderRunReportHTML renders a complete offline HTML report for a single-mode run.
func RenderRunReportHTML(report RunReport) ([]byte, error) {
	data := htmlPageData{
		Title:      "Run Report",
		ReportID:   report.ReportID,
		CreatedAt:  report.CreatedAt,
		TotalCases: report.TotalCases,
		RunSummary: &report.Summary,
		RunResults: make([]runCaseView, 0, len(report.Results)),
	}
	for _, result := range report.Results {
		data.RunResults = append(data.RunResults, buildRunCaseView(result))
	}
	return renderHTML(data)
}

// RenderPairReportHTML renders a complete offline HTML report for a pair-mode run.
func RenderPairReportHTML(report PairReport) ([]byte, error) {
	data := htmlPageData{
		Title:       "Pair Report",
		ReportID:    report.ReportID,
		CreatedAt:   report.CreatedAt,
		TotalCases:  report.TotalCases,
		IsPair:      true,
		PairSummary: &report.Summary,
		PairResults: make([]pairCaseView, 0, len(report.Results)),
	}
	for _, result := range report.Results {
		data.PairResults = append(data.PairResults, pairCaseView{
			CaseID:      result.CaseID,
			Error:       result.Error,
			ScoreReason: result.Score.Reason,
			SideA:       buildRunCaseView(result.A.CaseResult),
			SideB:       buildRunCaseView(result.B.CaseResult),
			StatusClass: pairStatusClass(result),
		})
	}
	return renderHTML(data)
}

func buildRunCaseView(result CaseResult) runCaseView {
	return runCaseView{
		CaseID:             result.CaseID,
		AgentName:          result.AgentName,
		StopReason:         result.StopReason,
		Passed:             result.Passed,
		FinalAnswer:        truncateText(result.FinalAnswer, 240),
		Error:              result.Error,
		ErrorClass:         result.ErrorClass,
		FailedIteration:    result.FailedIteration,
		Iterations:         result.Iterations,
		ToolExecutionCount: len(result.ToolExecutions),
		EventSummary:       summarizeEvents(result.Events),
		StatusClass:        caseStatusClass(result),
	}
}

func summarizeEvents(events []agent.Event) []eventSummaryView {
	important := make([]eventSummaryView, 0, 4)
	for _, event := range events {
		if !isImportantEvent(event.Type) {
			continue
		}
		important = append(important, eventSummaryView{
			Type:        event.Type,
			Message:     truncateText(event.Message, 120),
			Iteration:   event.Iteration,
			StatusClass: eventStatusClass(event.Type),
		})
	}
	return important
}

func isImportantEvent(eventType string) bool {
	switch eventType {
	case "provider.request.failed", "provider.request.retried", "tool.validation.failed", "run.timed_out", "run.canceled":
		return true
	default:
		return false
	}
}

func caseStatusClass(result CaseResult) string {
	switch result.StopReason {
	case "timed_out", "canceled":
		return "warn"
	}
	if result.Error != "" || result.ErrorClass != "" || !result.Passed {
		return "error"
	}
	return "ok"
}

func pairStatusClass(result PairResult) string {
	switch {
	case result.A.CaseResult.StopReason == "timed_out" || result.B.CaseResult.StopReason == "timed_out" ||
		result.A.CaseResult.StopReason == "canceled" || result.B.CaseResult.StopReason == "canceled":
		return "warn"
	case result.Error != "" || result.A.CaseResult.Error != "" || result.B.CaseResult.Error != "":
		return "error"
	case result.A.CaseResult.Passed && result.B.CaseResult.Passed:
		return "ok"
	default:
		return "error"
	}
}

func eventStatusClass(eventType string) string {
	switch eventType {
	case "provider.request.retried":
		return "warn"
	case "provider.request.failed", "tool.validation.failed":
		return "error"
	case "run.timed_out", "run.canceled":
		return "warn"
	default:
		return "neutral"
	}
}

func truncateText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

func renderHTML(data htmlPageData) ([]byte, error) {
	tmpl, err := template.New("report").Parse(reportHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse html template: %w", err)
	}
	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return nil, fmt.Errorf("render html template: %w", err)
	}
	return buffer.Bytes(), nil
}

const reportHTMLTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>{{.Title}}</title>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; margin: 0; background: #f7f8fa; color: #101828; }
    main { max-width: 1200px; margin: 0 auto; padding: 24px; }
    .panel { background: #fff; border: 1px solid #d8dee9; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    .grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
    .kv { border: 1px solid #d8dee9; border-radius: 8px; padding: 10px; }
    .kv strong { display: block; margin-bottom: 4px; }
    .case { border-left: 5px solid #2563eb; }
    .case.ok { border-left-color: #027a48; }
    .case.error { border-left-color: #b42318; }
    .case.warn { border-left-color: #b54708; }
    .badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 12px; font-weight: 600; }
    .badge.ok { background: #ecfdf3; color: #027a48; }
    .badge.error { background: #fef3f2; color: #b42318; }
    .badge.warn { background: #fffaeb; color: #b54708; }
    .muted { color: #667085; }
    pre { white-space: pre-wrap; word-break: break-word; background: #f8fafc; border: 1px solid #d8dee9; padding: 10px; border-radius: 8px; }
    ul { margin: 8px 0 0 18px; }
    .pair { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
    @media (max-width: 900px) { .grid, .pair { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <main>
    <section class="panel">
      <h1>{{.Title}}</h1>
      <p class="muted">Report ID: {{.ReportID}} · Created At: {{.CreatedAt}} · Total Cases: {{.TotalCases}}</p>
    </section>

    {{if .RunSummary}}
    <section class="panel">
      <h2>Summary</h2>
      <div class="grid">
        <div class="kv"><strong>Total Cases</strong>{{.RunSummary.TotalCases}}</div>
        <div class="kv"><strong>Passed</strong>{{.RunSummary.Passed}}</div>
        <div class="kv"><strong>Failed</strong>{{.RunSummary.Failed}}</div>
        <div class="kv"><strong>Unchecked</strong>{{.RunSummary.Unchecked}}</div>
        <div class="kv"><strong>Errored</strong>{{.RunSummary.Errored}}</div>
        <div class="kv"><strong>Timed Out</strong>{{.RunSummary.TimedOutCount}}</div>
        <div class="kv"><strong>Canceled</strong>{{.RunSummary.CanceledCount}}</div>
        <div class="kv"><strong>Average Iterations</strong>{{printf "%.2f" .RunSummary.AverageIterations}}</div>
        <div class="kv"><strong>Total Tool Calls</strong>{{.RunSummary.TotalToolCalls}}</div>
      </div>
      <h3>Stop Reasons</h3>
      <pre>{{printf "%v" .RunSummary.StopReasons}}</pre>
      <h3>Error Classes</h3>
      <pre>{{printf "%v" .RunSummary.ErrorClasses}}</pre>
    </section>

    <section class="panel">
      <h2>Cases</h2>
      {{range .RunResults}}
      <article class="panel case {{.StatusClass}}">
        <div><strong>{{.CaseID}}</strong> {{if .Passed}}<span class="badge ok">passed</span>{{else if eq .StatusClass "warn"}}<span class="badge warn">{{.StopReason}}</span>{{else}}<span class="badge error">failed</span>{{end}}</div>
        <p class="muted">agent={{.AgentName}} · stop={{.StopReason}} · iterations={{.Iterations}} · failed_iteration={{.FailedIteration}} · tool_calls={{.ToolExecutionCount}}</p>
        {{if .Error}}<p><strong>Error:</strong> {{.Error}}</p>{{end}}
        {{if .ErrorClass}}<p><strong>Error Class:</strong> {{.ErrorClass}}</p>{{end}}
        {{if .FinalAnswer}}<p><strong>Final Answer:</strong></p><pre>{{.FinalAnswer}}</pre>{{end}}
        {{if .EventSummary}}
        <p><strong>Key Events</strong></p>
        <ul>
          {{range .EventSummary}}
          <li><span class="badge {{.StatusClass}}">{{.Type}}</span> iteration={{.Iteration}} {{if .Message}}· {{.Message}}{{end}}</li>
          {{end}}
        </ul>
        {{end}}
      </article>
      {{end}}
    </section>
    {{end}}

    {{if .PairSummary}}
    <section class="panel">
      <h2>Pair Summary</h2>
      <div class="grid">
        <div class="kv"><strong>Total Pairs</strong>{{.PairSummary.TotalPairs}}</div>
        <div class="kv"><strong>Both Passed</strong>{{.PairSummary.BothPassed}}</div>
        <div class="kv"><strong>Only A Passed</strong>{{.PairSummary.OnlyAPassed}}</div>
        <div class="kv"><strong>Only B Passed</strong>{{.PairSummary.OnlyBPassed}}</div>
        <div class="kv"><strong>Both Failed</strong>{{.PairSummary.BothFailed}}</div>
        <div class="kv"><strong>Errored Pairs</strong>{{.PairSummary.ErroredPairs}}</div>
        <div class="kv"><strong>A Avg Iterations</strong>{{printf "%.2f" .PairSummary.A.AverageIterations}}</div>
        <div class="kv"><strong>B Avg Iterations</strong>{{printf "%.2f" .PairSummary.B.AverageIterations}}</div>
        <div class="kv"><strong>A Tool Calls</strong>{{.PairSummary.A.TotalToolCalls}}</div>
      </div>
    </section>

    <section class="panel">
      <h2>Pair Cases</h2>
      {{range .PairResults}}
      <article class="panel case {{.StatusClass}}">
        <div><strong>{{.CaseID}}</strong> {{if .Error}}<span class="badge error">pair error</span>{{end}}</div>
        {{if .Error}}<p><strong>Pair Error:</strong> {{.Error}}</p>{{end}}
        {{if .ScoreReason}}<p class="muted">score={{.ScoreReason}}</p>{{end}}
        <div class="pair">
          <section>
            <h3>Side A</h3>
            <p class="muted">passed={{.SideA.Passed}} · stop={{.SideA.StopReason}} · iterations={{.SideA.Iterations}}</p>
            {{if .SideA.Error}}<p><strong>Error:</strong> {{.SideA.Error}}</p>{{end}}
            {{if .SideA.ErrorClass}}<p><strong>Error Class:</strong> {{.SideA.ErrorClass}}</p>{{end}}
            {{if .SideA.FinalAnswer}}<pre>{{.SideA.FinalAnswer}}</pre>{{end}}
            {{if .SideA.EventSummary}}<ul>{{range .SideA.EventSummary}}<li><span class="badge {{.StatusClass}}">{{.Type}}</span> iteration={{.Iteration}}</li>{{end}}</ul>{{end}}
          </section>
          <section>
            <h3>Side B</h3>
            <p class="muted">passed={{.SideB.Passed}} · stop={{.SideB.StopReason}} · iterations={{.SideB.Iterations}}</p>
            {{if .SideB.Error}}<p><strong>Error:</strong> {{.SideB.Error}}</p>{{end}}
            {{if .SideB.ErrorClass}}<p><strong>Error Class:</strong> {{.SideB.ErrorClass}}</p>{{end}}
            {{if .SideB.FinalAnswer}}<pre>{{.SideB.FinalAnswer}}</pre>{{end}}
            {{if .SideB.EventSummary}}<ul>{{range .SideB.EventSummary}}<li><span class="badge {{.StatusClass}}">{{.Type}}</span> iteration={{.Iteration}}</li>{{end}}</ul>{{end}}
          </section>
        </div>
      </article>
      {{end}}
    </section>
    {{end}}
  </main>
</body>
</html>`
