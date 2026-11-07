# Agent Instructions

## Start Here
1. Read this file.
2. Read `project/constitution.md` and `project/PROJECT-SETUP.md`.
3. For the current task, read its change directory and relevant Specs.
4. Read relevant UX/UI files (`project/design-system.md`, `project/user-flow.md`, `project/page-specs/`) when changing UI.
5. Inspect the actual code and Git status before proposing edits.

## Source of Truth
- Product direction / MVP: `project/product-brief.md`, `project/mvp-scope.md` (human-approved)
- Requirements: `project/requirements.md`, `project/user-stories.md`
- UX/UI baseline: `project/user-flow.md`, `project/design-system.md`, `project/page-specs/`
- Architecture: `project/architecture.md`, `project/tech-stack.md`
- Technical decisions: `project/adr/`
- Behavior specs: `project/specs/`（仅 `implemented` / `verified` 表示当前能力；`planned` 是待实现目标）
- Current change: `changes/<change-id>/`
- Commands and environment: `project/PROJECT-SETUP.md`

If documents conflict, stop and report the conflict. Do not silently choose one. After a human decision, update the authoritative source and all affected downstream documents, including change history.

## Scope
- Implement only approved tasks.
- Do not refactor unrelated code.
- Do not alter product scope, public API/data contracts, architecture, or approved UI baselines without approval.
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
A task is complete only when its DoD is met, required checks pass, review findings are resolved/waived, acceptance is recorded, and documentation is synchronized.
