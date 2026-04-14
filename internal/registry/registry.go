package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"agent-skill-eval-go/internal/spec"
)

// Registry stores loaded skill specs keyed by skill name.
type Registry struct {
	skills map[string]spec.SkillSpec
}

// LoadSkills loads all .json skill specs from the given directory.
func LoadSkills(dir string) (*Registry, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("skills directory does not exist: %s", dir)
		}
		return nil, fmt.Errorf("stat skills directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skills path is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read skills directory %s: %w", dir, err)
	}

	registry := &Registry{
		skills: make(map[string]spec.SkillSpec),
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read skill file %s: %w", path, err)
		}

		var skill spec.SkillSpec
		if err := json.Unmarshal(data, &skill); err != nil {
			return nil, fmt.Errorf("parse skill file %s: %w", path, err)
		}

		if skill.Name == "" {
			return nil, fmt.Errorf("skill file %s: missing skill name", path)
		}

		if _, exists := registry.skills[skill.Name]; exists {
			return nil, fmt.Errorf("duplicate skill name %q in file %s", skill.Name, path)
		}

		registry.skills[skill.Name] = skill
	}

	return registry, nil
}

// Get returns one skill spec by name.
func (r *Registry) Get(name string) (spec.SkillSpec, bool) {
	if r == nil {
		return spec.SkillSpec{}, false
	}

	skill, ok := r.skills[name]
	return skill, ok
}

// List returns all skill specs sorted by name for stable output.
func (r *Registry) List() []spec.SkillSpec {
	if r == nil || len(r.skills) == 0 {
		return []spec.SkillSpec{}
	}

	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	sort.Strings(names)

	skills := make([]spec.SkillSpec, 0, len(names))
	for _, name := range names {
		skills = append(skills, r.skills[name])
	}

	return skills
}
