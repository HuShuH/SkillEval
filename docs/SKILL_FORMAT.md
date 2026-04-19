# SKILL Format

The new skill loader reads a minimal `SKILL.md` file from a skill directory.

## Smallest Supported Shape

```md
# simple_writer

Description: Minimal example skill
Version: 0.1.0

## Instructions
Do the task.
Return a concise answer.

## Tools
- filesystem
- finish
```

## Parsed Fields

- Title (`# ...`) → `name`
- `Description: ...` → `description`
- `Version: ...` → `version`
- `## Instructions` body → `instructions` and `system_prompt`
- `## Tools` bullet list → `tools`

## Notes

- `Instructions` is required
- Tool names are deduplicated
- `SKILL.md` can be loaded by passing either:
  - the skill directory path
  - the direct `SKILL.md` file path

See:

- `examples/skills/simple-writer/SKILL.md`
- `examples/skills/cautious-writer/SKILL.md`
