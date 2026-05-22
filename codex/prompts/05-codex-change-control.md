# Codex Prompt: Change-Control Check

Review the requested Codex task before making changes.

Do not change code.
Do not change documentation unless the user explicitly asks this prompt to update process docs.
Do not add features.

## Check

Assess whether the task has:

- a clear goal
- files or areas likely affected
- files or areas that must not change
- validation commands
- explicit out-of-scope items
- a rollback/checkpoint point, or a clean enough working tree for a small change

## Backlog Gate

If the request introduces future work that is not necessary for the current task, recommend creating an issue or backlog item instead of implementing it now.

Security vulnerabilities should follow `SECURITY.md`, not a public issue template. Non-sensitive security hardening can become a normal backlog item.

## Output

Return a short readiness assessment:

- `Ready`: the task is scoped enough to start.
- `Needs clarification`: one or two specific details are missing.
- `Create backlog item`: the request is future work or too broad for the current task.

Include the likely validation commands. Do not make file changes.
