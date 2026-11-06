# Agent Instructions

## Start Here
1. Read this file.
2. Read `project/constitution.md`.
3. Read `project/PROJECT-SETUP.md`.
4. For the current task, read its change directory and relevant Specs.
5. Inspect the actual code before proposing edits.

## Source of Truth
- Product baseline: `project/requirements.md`
- Architecture: `project/architecture.md`
- Technical decisions: `project/adr/`
- Current behavior: `project/specs/`
- Current change: `changes/<change-id>/`
- Commands: `project/PROJECT-SETUP.md`

If documents conflict, stop and report the conflict. Do not silently choose one.

## Scope
- Implement only approved tasks.
- Do not refactor unrelated code.
- Do not alter public API/data contracts without approval.
- Report out-of-scope necessities and wait for approval.

## Evidence
- Never claim a command ran unless it actually ran.
- Record exact commands and outcomes.
- Mark unrun tests as NOT RUN and blocked tests as BLOCKED.
- Do not fabricate logs, screenshots, metrics, or links.

## Safety
- Never expose secrets.
- Do not run destructive commands or production operations without explicit approval.
- Do not modify production data.
- Ask before adding dependencies or changing infrastructure.

## Completion
A task is complete only when its DoD is met, required checks pass, review findings are resolved/waived, and documentation is synchronized.
