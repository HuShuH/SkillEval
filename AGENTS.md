# Agent Instructions for Universal AI Agent Skill Evaluation System

## Project Goal
Build a robust, automated evaluation system for testing general-purpose AI Agent Skills (tool use, reasoning, safety, etc.).

## Directory Structure (to be created)
- schemas/          # JSON schemas for skills, test cases, results
- skills/           # Example skills (hello_world, etc.)
- runners/          # Core evaluation runners (single, batch, with timeout)
- tests/            # pytest-based automated tests
- reports/          # Generated evaluation reports (gitignore)
- config/           # Config files, env settings

## Key Requirements
- Use Python 3.12+
- Strict isolation with Conda (or uv if we switch later)
- Timeout and resource control for every skill execution
- Support multi-agent orchestration in the eval system itself later
- All code must be testable with pytest
- Git checkpoints before/after major changes

## Workflow Rules
- Always create git checkpoint before and after tasks
- Prefer modular, reusable components
- Output clear JSON results for every evaluation
