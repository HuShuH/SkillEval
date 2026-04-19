// Package skill contains Phase 1 migration skeleton types for the new architecture.
// It currently provides only minimal types, validation, and assembly behavior.
package skill

import (
	"fmt"
	"sort"
	"strings"
)

// Registry stores skills by name for the new architecture.
type Registry struct {
	skills map[string]Skill
	order  []string
}

// NewRegistry creates an empty skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register adds one validated skill.
func (r *Registry) Register(s Skill) error {
	if r == nil {
		return fmt.Errorf("%w: registry is nil", ErrInvalidSkill)
	}
	if err := s.Validate(); err != nil {
		return err
	}

	name := strings.TrimSpace(s.Name)
	if _, ok := r.skills[name]; ok {
		return fmt.Errorf("%w: %q", ErrDuplicateSkill, name)
	}

	r.skills[name] = s
	r.order = append(r.order, name)
	return nil
}

// Lookup returns one skill by name.
func (r *Registry) Lookup(name string) (Skill, error) {
	if r == nil {
		return Skill{}, fmt.Errorf("%w: %q", ErrSkillNotFound, name)
	}
	found, ok := r.skills[strings.TrimSpace(name)]
	if !ok {
		return Skill{}, fmt.Errorf("%w: %q", ErrSkillNotFound, name)
	}
	return found, nil
}

// List returns all registered skills in registration order.
func (r *Registry) List() []Skill {
	if r == nil {
		return nil
	}
	skills := make([]Skill, 0, len(r.order))
	for _, name := range r.order {
		skills = append(skills, r.skills[name])
	}
	return skills
}

// Names returns sorted skill names for inspection.
func (r *Registry) Names() []string {
	if r == nil {
		return nil
	}
	names := append([]string(nil), r.order...)
	sort.Strings(names)
	return names
}
