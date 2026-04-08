# Agent Workspace

`.agent` contains reusable assets for agent prompt development in this repository.

## Layout

- `project/`: stable repository context that most tasks should reuse
- `tasks/`: task-specific prompts grouped by work type
- `templates/`: prompt skeletons for new tasks, implementation asks, and reviews
- `examples/`: request, response, and config examples that reduce guesswork
- `evals/`: lightweight acceptance criteria for prompt runs
- `checklists/`: done/review/release checklists for consistent delivery
- `changelog/`: notes about prompt structure changes

## Working Rules

1. Reuse `project/*.md` before writing a new task prompt.
2. Keep each task prompt focused on one outcome.
3. Put durable repo knowledge in `project/`, not in task files.
4. Prefer examples and explicit verification steps over long prose.
5. Update the matching `evals/` file when a task prompt meaningfully changes.

## Suggested Flow

1. Read `project/overview.md` and `project/conventions.md`.
2. Start from `templates/task-prompt.md`.
3. Add references to concrete files and commands.
4. Add or update examples in `examples/`.
5. Add pass/fail checks in `evals/`.
