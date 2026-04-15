package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/spec"
)

// SkillFiles validates raw skill JSON files in a directory and returns parsed skills.
func SkillFiles(dir string) ([]spec.SkillSpec, []string) {
	errors := make([]string, 0)

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, []string{fmt.Sprintf("skills directory does not exist: %s", dir)}
		}
		return nil, []string{fmt.Sprintf("stat skills directory %s: %v", dir, err)}
	}
	if !info.IsDir() {
		return nil, []string{fmt.Sprintf("skills path is not a directory: %s", dir)}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, []string{fmt.Sprintf("read skills directory %s: %v", dir, err)}
	}

	jsonFiles := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			jsonFiles = append(jsonFiles, entry.Name())
		}
	}
	sort.Strings(jsonFiles)

	skills := make([]spec.SkillSpec, 0, len(jsonFiles))
	seenNames := make(map[string]string)

	for _, fileName := range jsonFiles {
		path := filepath.Join(dir, fileName)
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("read skill file %s: %v", path, err))
			continue
		}

		var skill spec.SkillSpec
		if err := json.Unmarshal(data, &skill); err != nil {
			errors = append(errors, fmt.Sprintf("parse skill file %s: %v", path, err))
			continue
		}

		if skill.Name == "" {
			errors = append(errors, fmt.Sprintf("skill file %s: skill name must not be empty", path))
			continue
		}

		if firstPath, exists := seenNames[skill.Name]; exists {
			errors = append(errors, fmt.Sprintf("duplicate skill name %q in files %s and %s", skill.Name, firstPath, path))
			continue
		}

		seenNames[skill.Name] = path
		skills = append(skills, skill)
	}

	return skills, errors
}

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
