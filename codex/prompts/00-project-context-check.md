# Codex Prompt: Project Context Check

Read project context before making changes.

Required sources:

- `README.md`
- `AGENTS.md`
- `docs/README.md`
- relevant files in `docs/`
- relevant prompt in `codex/prompts/`

Summarize:

- current project scope
- current backend surfaces
- private/public listener split
- current security boundaries
- current known exclusions
- files likely affected by the requested task
- files or areas that must not change
- likely validation commands

Do not change files yet.
Do not add features.
