// Package skill contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal types, validation, and assembly behavior.
package skill

import "errors"

var (
	ErrNotImplemented           = errors.New("skill loading not implemented")
	ErrInvalidSkill             = errors.New("invalid skill")
	ErrSkillNameRequired        = errors.New("skill name is required")
	ErrSkillInstructionsMissing = errors.New("skill instructions or system prompt is required")
	ErrDuplicateToolName        = errors.New("duplicate tool name in skill")
	ErrDuplicateSkill           = errors.New("skill already registered")
	ErrSkillNotFound            = errors.New("skill not found")
	ErrToolResolutionFailed     = errors.New("skill tool resolution failed")
	ErrSkillLoadFailed          = errors.New("skill load failed")
	ErrSkillParseFailed         = errors.New("skill parse failed")
)
