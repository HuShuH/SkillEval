package main

import (
	"fmt"

	"agent-skill-eval-go/internal/registry"
	"agent-skill-eval-go/internal/runner"
	"agent-skill-eval-go/internal/spec"
	"agent-skill-eval-go/internal/validate"
)

type loadedRunInputs struct {
	Registry  *registry.Registry
	TestCases []spec.TestCase
}

type loadedValidateInputs struct {
	Skills    []spec.SkillSpec
	Registry  *registry.Registry
	TestCases []spec.TestCase
}

func loadRunInputs(skillsDir, casesFile string) (*loadedRunInputs, error) {
	reg, err := registry.LoadSkills(skillsDir)
	if err != nil {
		return nil, fmt.Errorf("load skills: %w", err)
	}

	testCases, err := runner.LoadTestCases(casesFile)
	if err != nil {
		return nil, fmt.Errorf("load testcases: %w", err)
	}

	return &loadedRunInputs{
		Registry:  reg,
		TestCases: testCases,
	}, nil
}

func loadValidateInputs(skillsDir, casesFile string) (*loadedValidateInputs, []string) {
	skills, validationErrors := validate.SkillFiles(skillsDir)
	if len(validationErrors) > 0 {
		return nil, validationErrors
	}

	reg, err := registry.LoadSkills(skillsDir)
	if err != nil {
		return nil, []string{fmt.Sprintf("load skills: %v", err)}
	}

	testCases, err := runner.LoadTestCases(casesFile)
	if err != nil {
		return nil, []string{fmt.Sprintf("load testcases: %v", err)}
	}

	return &loadedValidateInputs{
		Skills:    skills,
		Registry:  reg,
		TestCases: testCases,
	}, nil
}
