package eval

import "testing"

func TestCheckerExactMatchPassAndFail(t *testing.T) {
	pass := EvaluateChecks(Case{
		Checkers: []CheckerConfig{{Type: CheckerExactMatch, Config: map[string]any{"value": "done"}}},
	}, "done")
	if !pass.Passed {
		t.Fatalf("expected exact match pass")
	}

	fail := EvaluateChecks(Case{
		Checkers: []CheckerConfig{{Type: CheckerExactMatch, Config: map[string]any{"value": "done"}}},
	}, "nope")
	if fail.Passed {
		t.Fatalf("expected exact match fail")
	}
}

func TestCheckerContainsPassAndFail(t *testing.T) {
	pass := EvaluateChecks(Case{
		Checkers: []CheckerConfig{{Type: CheckerContains, Config: map[string]any{"value": "report"}}},
	}, "report generated")
	if !pass.Passed {
		t.Fatalf("expected contains pass")
	}

	fail := EvaluateChecks(Case{
		Checkers: []CheckerConfig{{Type: CheckerContains, Config: map[string]any{"value": "report"}}},
	}, "summary generated")
	if fail.Passed {
		t.Fatalf("expected contains fail")
	}
}

func TestCheckerNonEmptyPassAndFail(t *testing.T) {
	pass := EvaluateChecks(Case{
		Checkers: []CheckerConfig{{Type: CheckerNonEmpty}},
	}, "hello")
	if !pass.Passed {
		t.Fatalf("expected non-empty pass")
	}

	fail := EvaluateChecks(Case{
		Checkers: []CheckerConfig{{Type: CheckerNonEmpty}},
	}, "   ")
	if fail.Passed {
		t.Fatalf("expected non-empty fail")
	}
}

func TestCheckerUncheckedDefaultBehavior(t *testing.T) {
	got := EvaluateChecks(Case{}, "anything")
	if !got.Passed || got.Checked {
		t.Fatalf("expected unchecked pass result, got %+v", got)
	}
}
