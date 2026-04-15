package validate

import (
	"fmt"

	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/spec"
)

// Skills validates loaded skill specs.
func Skills(skills []spec.SkillSpec) []string {
	errors := make([]string, 0)
	for _, skill := range skills {
		if skill.Name == "" {
			errors = append(errors, "skill name must not be empty")
		}
	}
	return errors
}

// TestCases validates testcase definitions against registry state.
func TestCases(testCases []spec.TestCase, reg *registry.Registry) []string {
	errors := make([]string, 0)
	seenCaseIDs := make(map[string]struct{})

	for index, testCase := range testCases {
		label := fmt.Sprintf("testcase[%d]", index)
		if testCase.CaseID == "" {
			errors = append(errors, fmt.Sprintf("%s: case_id must not be empty", label))
		} else {
			if _, exists := seenCaseIDs[testCase.CaseID]; exists {
				errors = append(errors, fmt.Sprintf("%s: duplicate case_id %q", label, testCase.CaseID))
			} else {
				seenCaseIDs[testCase.CaseID] = struct{}{}
			}
			label = fmt.Sprintf("case_id=%q", testCase.CaseID)
		}

		if testCase.Skill.Name == "" {
			errors = append(errors, fmt.Sprintf("%s: skill.name must not be empty", label))
		} else if _, ok := reg.Get(testCase.Skill.Name); !ok {
			errors = append(errors, fmt.Sprintf("%s: referenced skill %q not found", label, testCase.Skill.Name))
		}

		if testCase.HardChecks.ExpectedArgs != nil && testCase.HardChecks.ExpectedToolName == "" {
			errors = append(errors, fmt.Sprintf("%s: hard_checks.expected_tool_name must be set when hard_checks.expected_args is provided", label))
		}
	}

	return errors
}

// All validates both skills and testcases.
func All(skills []spec.SkillSpec, testCases []spec.TestCase, reg *registry.Registry) []string {
	errors := Skills(skills)
	errors = append(errors, TestCases(testCases, reg)...)
	return errors
}
