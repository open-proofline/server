# Codex Prompt: Prepare GitHub Issue Creation Commands from Reviewed Backlog Drafts

Read `.backlog-drafts/`.

Do **not** modify application code.
Do **not** create GitHub issues directly unless explicitly told to run commands.
Do **not** include sensitive vulnerability details in public issues.
Do **not** modify workflows, Dockerfiles, SQL migrations, Go code, generated files, binaries, database files, or uploaded blob data.

## Goal

Generate a shell script that creates GitHub issues from reviewed backlog draft files using GitHub CLI.

Create:

```text
scripts/create-backlog-issues.sh
```

## Repository

Use repo:

```text
TheSilkky/safety-recorder
```

## Requirements

- Use `gh issue create`.
- For each `.backlog-drafts/NNN-*.md` file, derive:
  - title from the first Markdown heading
  - body from the entire file
  - labels from the `## Labels` section
- Do not execute the script.
- Make the script readable and reviewable.
- Add comments explaining that labels must exist or GitHub CLI may fail.
- Do not create issues marked sensitive/private.
- Do not include drafts that say they should not be public.
- Keep the idempotence warning clear: running the script twice may create duplicate issues.
- Keep output readable.
- Quote file paths and arguments safely.
- Put the script under `scripts/`.
- Create `scripts/` if it does not exist.

## Expected script behaviour

The script should:

1. Stop on errors.

   ```bash
   set -euo pipefail
   ```

2. Define the repository once.

   ```bash
   REPO="TheSilkky/safety-recorder"
   ```

3. Create issues one by one.

4. Use `--body-file` rather than embedding large Markdown bodies inline.

5. Use labels from the draft if practical.

6. Print which issue draft is being created.

7. Refuse to run if GitHub CLI is not installed or not authenticated.

Example shape:

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="TheSilkky/safety-recorder"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required" >&2
  exit 1
fi

gh auth status >/dev/null

echo "Creating issue from .backlog-drafts/001-example.md"
gh issue create \
  --repo "$REPO" \
  --title "Example issue" \
  --body-file ".backlog-drafts/001-example.md" \
  --label "backlog,maintenance"
```

## Review requirements

Before generating the script, inspect every draft under `.backlog-drafts/`.

Exclude any draft that:

- is marked sensitive
- says not to create publicly
- contains raw tokens
- contains secrets
- contains private deployment details
- contains user safety data
- contains exploit details that should go through `SECURITY.md`

If a draft is excluded, list it in the output summary and explain why.

## Optional helper output

Also create, if useful:

```text
.backlog-drafts/create-issues-review.md
```

This file should summarize:

- issue drafts included in the script
- issue drafts excluded
- labels used
- command to run the script
- warning that running twice may create duplicates

## Constraints

Allowed files:

- `scripts/create-backlog-issues.sh`
- `.backlog-drafts/create-issues-review.md`

Do not change anything else unless needed only for a tiny documentation link.

## Validation

After generating the script:

```bash
git diff --stat
git diff -- scripts/create-backlog-issues.sh .backlog-drafts/create-issues-review.md
```

Do not run Go tests unless code was changed.

Do not execute the script.

## Output

Summarize:

1. issue drafts included
2. issue drafts excluded
3. labels used
4. script path
5. command to run after maintainer review
6. reminder that running twice may create duplicate issues
