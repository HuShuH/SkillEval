package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var errAlreadyReported = errors.New("already reported")

type validateResult struct {
	OK              bool     `json:"ok"`
	SkillsLoaded    int      `json:"skills_loaded"`
	TestCasesLoaded int      `json:"testcases_loaded"`
	Errors          []string `json:"errors"`
}

type runResult struct {
	OK         bool   `json:"ok"`
	Total      int    `json:"total,omitempty"`
	Passed     int    `json:"passed,omitempty"`
	Failed     int    `json:"failed,omitempty"`
	ReportPath string `json:"report_path"`
	Error      string `json:"error,omitempty"`
}

func reportRunFailure(jsonOutput bool, reportPath string, errorMessage string) error {
	if jsonOutput {
		if err := writeRunJSON(runResult{
			OK:         false,
			ReportPath: reportPath,
			Error:      errorMessage,
		}); err != nil {
			return err
		}
		return errAlreadyReported
	}
	return errors.New(errorMessage)
}

func reportValidateFailure(jsonOutput bool, skillsLoaded, testCasesLoaded int, validationErrors []string) error {
	if jsonOutput {
		if err := writeValidateJSON(validateResult{
			OK:              false,
			SkillsLoaded:    skillsLoaded,
			TestCasesLoaded: testCasesLoaded,
			Errors:          validationErrors,
		}); err != nil {
			return err
		}
		return errAlreadyReported
	}

	for _, validationError := range validationErrors {
		fmt.Fprintf(os.Stderr, "- %s\n", validationError)
	}
	return fmt.Errorf("validation failed with %d error(s)", len(validationErrors))
}

func writeRunJSON(result runResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run result: %w", err)
	}
	data = append(data, '\n')
	if _, err := os.Stdout.Write(data); err != nil {
		return fmt.Errorf("write run result: %w", err)
	}
	return nil
}

func writeValidateJSON(result validateResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal validate result: %w", err)
	}
	data = append(data, '\n')
	if _, err := os.Stdout.Write(data); err != nil {
		return fmt.Errorf("write validate result: %w", err)
	}
	return nil
}
