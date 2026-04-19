// Package skill contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal types, validation, and assembly behavior.
package skill

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"agent-skill-eval-go/tool"
)

// Skill describes the static skill definition used by the new architecture.
type Skill struct {
	Name         string
	Version      string
	Description  string
	Tools        []string
	SystemPrompt string
	Instructions string
	Metadata     map[string]string
	SourcePath   string
	Directory    string
	SourceFormat string
	BoundTools   []tool.Tool
}

// Loader describes a future skill loading abstraction.
type Loader interface {
	LoadSkill(path string) (*Skill, error)
}

// Parser describes a future skill parsing abstraction.
type Parser interface {
	ParseSkill(r io.Reader) (*Skill, error)
}

// LoadSkill loads a minimal SKILL.md from either a directory or a direct file path.
func LoadSkill(path string) (*Skill, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: path is required", ErrSkillLoadFailed)
	}

	resolvedPath := path
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrSkillLoadFailed, path, err)
	}
	if info.IsDir() {
		resolvedPath = filepath.Join(path, "SKILL.md")
		if _, err := os.Stat(resolvedPath); err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrSkillLoadFailed, resolvedPath, err)
		}
	} else if filepath.Base(path) != "SKILL.md" {
		return nil, fmt.Errorf("%w: %s is not a SKILL.md file", ErrSkillLoadFailed, path)
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrSkillLoadFailed, resolvedPath, err)
	}
	defer file.Close()

	skill, err := ParseSkill(file)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrSkillLoadFailed, resolvedPath, err)
	}
	skill.SourcePath = resolvedPath
	skill.Directory = filepath.Dir(resolvedPath)
	skill.SourceFormat = "skill_markdown"
	return skill, nil
}

// ParseSkill parses the minimal supported SKILL.md format.
func ParseSkill(r io.Reader) (*Skill, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: reader is nil", ErrSkillParseFailed)
	}

	lines, err := readLines(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSkillParseFailed, err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("%w: empty skill content", ErrSkillParseFailed)
	}

	skill := &Skill{
		Metadata: make(map[string]string),
	}
	section := ""
	var intro []string
	var instructions []string
	var tools []string

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			switch section {
			case "instructions":
				instructions = append(instructions, "")
			}
			continue
		}

		if strings.HasPrefix(line, "# ") && skill.Name == "" {
			skill.Name = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			continue
		}
		if heading, ok := parseSectionHeading(line); ok {
			section = heading
			continue
		}
		if key, value, ok := parseMetadataLine(line); ok && section != "instructions" {
			skill.Metadata[key] = value
			switch strings.ToLower(key) {
			case "name":
				if skill.Name == "" {
					skill.Name = value
				}
			case "version":
				skill.Version = value
			case "description":
				skill.Description = value
			}
			continue
		}

		switch section {
		case "instructions":
			instructions = append(instructions, raw)
		case "tools":
			if toolName, ok := parseToolItem(line); ok {
				tools = append(tools, toolName)
			}
		default:
			if skill.Description == "" {
				intro = append(intro, line)
			}
		}
	}

	if skill.Description == "" && len(intro) > 0 {
		skill.Description = strings.TrimSpace(strings.Join(intro, " "))
	}
	skill.Instructions = strings.TrimSpace(strings.Join(instructions, "\n"))
	skill.SystemPrompt = skill.Instructions
	skill.Tools = dedupeTools(tools)

	if err := skill.Validate(); err != nil {
		return nil, err
	}
	return skill, nil
}

// NewSkill constructs a normalized skill value and validates it.
func NewSkill(name, version, description string, toolNames []string, systemPrompt, instructions string) (Skill, error) {
	s := Skill{
		Name:         strings.TrimSpace(name),
		Version:      strings.TrimSpace(version),
		Description:  strings.TrimSpace(description),
		Tools:        normalizeToolNames(toolNames),
		SystemPrompt: strings.TrimSpace(systemPrompt),
		Instructions: strings.TrimSpace(instructions),
	}
	if err := s.Validate(); err != nil {
		return Skill{}, err
	}
	return s, nil
}

// Validate checks whether the skill itself is internally valid.
func (s Skill) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidSkill, ErrSkillNameRequired)
	}
	if strings.TrimSpace(s.SystemPrompt) == "" && strings.TrimSpace(s.Instructions) == "" {
		return fmt.Errorf("%w: %w", ErrInvalidSkill, ErrSkillInstructionsMissing)
	}

	seen := make(map[string]struct{}, len(s.Tools))
	for _, toolName := range s.Tools {
		normalized := strings.TrimSpace(toolName)
		if normalized == "" {
			return fmt.Errorf("%w: %w", ErrInvalidSkill, ErrDuplicateToolName)
		}
		if _, ok := seen[normalized]; ok {
			return fmt.Errorf("%w: %w %q", ErrInvalidSkill, ErrDuplicateToolName, normalized)
		}
		seen[normalized] = struct{}{}
	}

	return nil
}

// ResolveTools resolves tool names through the provided registry.
func (s Skill) ResolveTools(registry *tool.Registry) ([]tool.Tool, error) {
	if registry == nil {
		return nil, fmt.Errorf("%w: registry is nil", ErrToolResolutionFailed)
	}

	resolved := make([]tool.Tool, 0, len(s.Tools))
	seen := make(map[string]struct{}, len(s.Tools))
	for _, toolName := range s.Tools {
		normalized := strings.TrimSpace(toolName)
		if _, ok := seen[normalized]; ok {
			return nil, fmt.Errorf("%w: %w %q", ErrToolResolutionFailed, ErrDuplicateToolName, normalized)
		}
		found, err := registry.Lookup(normalized)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %w", ErrToolResolutionFailed, normalized, err)
		}
		resolved = append(resolved, found)
		seen[normalized] = struct{}{}
	}

	return resolved, nil
}

// AttachResolvedTools returns a copy of the skill with bound tool instances.
func (s Skill) AttachResolvedTools(tools []tool.Tool) (Skill, error) {
	seen := make(map[string]struct{}, len(tools))
	copied := make([]tool.Tool, 0, len(tools))
	for _, resolved := range tools {
		spec := resolved.Spec()
		if strings.TrimSpace(spec.Name) == "" {
			return Skill{}, fmt.Errorf("%w: resolved tool name is empty", ErrToolResolutionFailed)
		}
		if _, ok := seen[spec.Name]; ok {
			return Skill{}, fmt.Errorf("%w: %w %q", ErrToolResolutionFailed, ErrDuplicateToolName, spec.Name)
		}
		seen[spec.Name] = struct{}{}
		copied = append(copied, resolved)
	}

	s.BoundTools = copied
	return s, nil
}

func normalizeToolNames(names []string) []string {
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		normalized = append(normalized, strings.TrimSpace(name))
	}
	return normalized
}

func readLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func parseSectionHeading(line string) (string, bool) {
	if !strings.HasPrefix(line, "## ") {
		return "", false
	}
	title := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
	switch title {
	case "metadata", "instructions", "tools":
		return title, true
	default:
		return "", false
	}
}

func parseMetadataLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" || value == "" {
		return "", "", false
	}
	switch strings.ToLower(key) {
	case "name", "version", "description":
		return key, value, true
	default:
		return "", "", false
	}
}

func parseToolItem(line string) (string, bool) {
	if strings.HasPrefix(line, "- ") {
		name := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if name != "" {
			return name, true
		}
	}
	return "", false
}

func dedupeTools(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
