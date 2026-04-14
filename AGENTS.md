 # Agent Instructions for Go Agent Skill Evaluation Framework

  ## Project Goal

  Build a minimal, reliable MVP for a Go-based agent skill evaluation framework.

  This repository is currently focused on the smallest runnable version only.
  Do not design for advanced capabilities yet.

  ## Current Scope

  Only build the following MVP components:

  - skill registry
  - testcase loader
  - mock adapter
  - hard checker
  - runner
  - report
  - CLI run

  ## Out of Scope

  Do not add these capabilities in the current stage:

  - multi-agent orchestration
  - LLM judge
  - web search
  - memory systems
  - sandboxing
  - distributed execution
  - cloud deployment
  - GUI or web frontend

  ## Engineering Principles

  - Use Go as the primary language for all core logic.
  - Keep the implementation minimal and runnable.
  - Prefer simple, explicit designs over abstraction-heavy architecture.
  - Build only what is needed for the current step.
  - Avoid speculative extension points unless they clearly support the MVP.
  - Keep modules small, testable, and easy to replace later.

  ## Development Workflow

  - Develop incrementally.
  - Each task should change only one small piece of the system at a time.
  - After every change, the project must still compile locally.
  - Prefer small PR-sized changes over large refactors.
  - Do not mix unrelated changes in one step.
  - If a design is uncertain, choose the simplest version that keeps forward progress.

  ## Build Discipline

  - Every change must preserve local buildability.
  - Before finishing a task, verify the code can compile.
  - If tests exist for the touched area, run them.
  - Do not leave the repository in a partially broken state.

  ## Git Checkpoints

  - Create a git checkpoint before major changes.
  - Create another git checkpoint after completing a stable step.
  - Keep commit scope small and meaningful.
  - Use commit messages that describe the concrete engineering change.

  ## Recommended Go-Oriented Layout

  This structure is a guideline for the MVP:

  - cmd/
      - CLI entrypoints
  - internal/
      - application internals
      - registry/
      - testcase/
      - adapter/
      - checker/
      - runner/
      - report/
  - pkg/
      - only for reusable public packages if truly needed
  - configs/
      - static config samples if needed
  - testdata/
      - testcase fixtures and mock inputs
  - docs/
      - short design notes and MVP docs

  Prefer internal/ for most implementation during the MVP stage.

  ## Current Phase File Boundaries

  In the current first phase, only prioritize creating or modifying these files and paths:

  - cmd/agent-eval/main.go
  - internal/spec/types.go
  - internal/registry/registry.go
  - internal/adapters/adapter.go
  - internal/adapters/mock.go
  - internal/checker/checker.go
  - internal/runner/runner.go
  - internal/report/report.go
  - testdata/skills/
  - testdata/cases/

  During this phase, do not add:

  - OpenAI adapter
  - network calls
  - databases
  - concurrent execution
  - extra CLI subcommands

  If a proposed change requires files outside this boundary, stop and confirm the need before expanding scope.

  ## MVP Component Expectations

  - skill registry: register and resolve available skills
  - testcase loader: load testcase definitions from local files
  - mock adapter: provide deterministic mock execution behavior
  - hard checker: evaluate outputs with explicit rule-based checks
  - runner: connect registry, loader, adapter, checker, and execution flow
  - report: emit simple structured evaluation results
  - CLI run: provide a minimal command to execute evaluations locally

  ## Coding Guidance

  - Favor standard library solutions first.
  - Keep interfaces small and only introduce them when they help the current MVP.
  - Avoid premature generic frameworks.
  - Prefer deterministic behavior for runner, adapter, and checker.
  - Keep output formats simple and machine-readable.
  - Write code that is easy to debug from the CLI.

  ## Task Boundaries

  When implementing new work:

  - choose one small target
  - implement it fully
  - verify it compiles
  - checkpoint with git
  - then move to the next step

  Do not jump ahead to advanced architecture.

  ## Definition of Done for Each Step

  A step is complete only if:

  - the targeted change is implemented
  - the code compiles locally
  - related tests pass if present
  - no unrelated files were changed unnecessarily
  - the change is ready for a git checkpoint

  ## Communication Style

  When proposing or making changes:

  - be concise
  - be concrete
  - state assumptions clearly
  - focus on the immediate next engineering step
  - avoid broad redesign unless explicitly requested