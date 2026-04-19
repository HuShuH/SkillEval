// Package tool contains Phase 1 migration skeleton types for the new architecture.
// It currently provides minimal tool validation and execution contracts.
package tool

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateCall applies minimal schema checks before tool execution.
func ValidateCall(spec Spec, call Call) error {
	required := requiredSet(spec)
	for field := range required {
		value, ok := valueForField(call, field)
		if !ok {
			return fmt.Errorf("%w: missing required field %q", ErrInvalidTool, field)
		}
		if field == "operation" || field == "path" || field == "command" || field == "final_answer" {
			text, ok := value.(string)
			if !ok || strings.TrimSpace(text) == "" {
				return fmt.Errorf("%w: field %q must be a non-empty string", ErrInvalidTool, field)
			}
		}
	}

	for field, parameter := range spec.Parameters {
		value, ok := valueForField(call, field)
		if !ok {
			continue
		}
		if err := validateType(field, parameter, value); err != nil {
			return err
		}
		if len(parameter.Enum) > 0 {
			text, _ := value.(string)
			if !contains(parameter.Enum, text) {
				return fmt.Errorf("%w: field %q must be one of %v", ErrInvalidTool, field, parameter.Enum)
			}
		}
	}

	switch spec.Name {
	case "filesystem":
		switch call.Operation {
		case OpWriteFile:
			if _, ok := call.Input["content"]; !ok {
				return fmt.Errorf("%w: missing required field %q", ErrInvalidTool, "content")
			}
		case OpReadFile, OpListDir:
		default:
			return fmt.Errorf("%w: field %q must be one of %v", ErrInvalidTool, "operation", []string{OpReadFile, OpWriteFile, OpListDir})
		}
	case "finish":
		if strings.TrimSpace(stringValue(call.Input["final_answer"])) == "" {
			return fmt.Errorf("%w: field %q must be a non-empty string", ErrInvalidTool, "final_answer")
		}
	}

	return nil
}

func requiredSet(spec Spec) map[string]struct{} {
	required := make(map[string]struct{}, len(spec.Required))
	for _, field := range spec.Required {
		required[field] = struct{}{}
	}
	if len(required) == 0 {
		for name, parameter := range spec.Parameters {
			if parameter.Required {
				required[name] = struct{}{}
			}
		}
	}
	return required
}

func RequiredFields(spec Spec) []string {
	required := requiredSet(spec)
	fields := make([]string, 0, len(required))
	for field := range required {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func valueForField(call Call, field string) (any, bool) {
	if field == "operation" {
		if call.Operation == "" {
			return nil, false
		}
		return call.Operation, true
	}
	value, ok := call.Input[field]
	return value, ok
}

func validateType(field string, parameter ParameterSpec, value any) error {
	switch parameter.Type {
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%w: field %q must be a string", ErrInvalidTool, field)
		}
	case TypeInteger:
		switch value.(type) {
		case int, int32, int64, float64:
		default:
			return fmt.Errorf("%w: field %q must be an integer", ErrInvalidTool, field)
		}
	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%w: field %q must be a boolean", ErrInvalidTool, field)
		}
	case TypeObject:
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("%w: field %q must be an object", ErrInvalidTool, field)
		}
	case TypeArray:
		switch value.(type) {
		case []any, []string:
		default:
			return fmt.Errorf("%w: field %q must be an array", ErrInvalidTool, field)
		}
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
