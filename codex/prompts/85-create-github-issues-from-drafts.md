# Codex Prompt: Prepare GitHub Issue Creation Commands from Reviewed Backlog Drafts

Read reviewed backlog drafts and generate a script that can create GitHub issues with GitHub CLI.

Do **not** modify application code.
Do **not** create GitHub issues directly unless explicitly told to run commands.
Do **not** execute the generated script.
Do **not** include sensitive vulnerability details in public issues.
Do **not** modify workflows, Dockerfiles, SQL migrations, Go code, generated files, binaries, database files, or uploaded blob data.

## Goal

Generate a shell script that creates GitHub issues from maintainer-reviewed backlog draft files.

Create:

```text
scripts/create-backlog-issues.sh
```

## Repository

```text
TheSilkky/safety-recorder
```

## Draft directory

Backlog drafts are normally created by `80-backlog-scan-issue-drafts.md`.

They should live under a timestamped directory such as:

```text
.backlog-drafts/YYYY-MM-DD/
```

or:

```text
.backlog-drafts/current/
```

Before generating the script:

1. Prefer an explicitly provided draft directory, if the maintainer gives one.
2. Otherwise, inspect `.backlog-drafts/` and choose the newest timestamped directory.
3. If multiple directories exist and the newest is unclear, stop and ask which draft directory to use.
4. Do not use `.backlog-drafts/private-notes/` or any `private-notes/` directory for public issue creation.

## Requirements

- Use `gh issue create`.
- For each `NNN-*.md` file in the selected draft directory, derive:
  - title from the first Markdown heading
  - body from the entire file
  - labels from the `## Labels` section
- Do not execute the script.
- Make the script readable and reviewable.
- Add comments explaining that labels must exist or GitHub CLI may fail.
- Exclude drafts marked sensitive/private or not for public creation.
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

3. Refuse to run if GitHub CLI is not installed or not authenticated.

4. Print which issue draft is being created.

5. Use `--body-file`.

6. Use labels from the draft where practical.

## Review requirements

Before generating the script, inspect every draft in the selected draft directory.

Exclude any draft that:

- is marked sensitive
- says not to create publicly
- lives under `private-notes/`
- contains raw tokens
- contains secrets
- contains private deployment details
- contains user safety data
- contains exploit details that should go through `SECURITY.md`

If a draft is excluded, list it in the output summary and explain why.

## Optional helper output

Also create, if useful:

```text
.backlog-drafts/<selected-directory>/create-issues-review.md
```

This file should summarize:

- selected draft directory
- issue drafts included in the script
- issue drafts excluded
- labels used
- command to run the script
- warning that running twice may create duplicates

## Constraints

Allowed files:

- `scripts/create-backlog-issues.sh`
- `.backlog-drafts/<selected-directory>/create-issues-review.md`

Do not change anything else unless needed only for a tiny documentation link.

## Validation

After generating the script:

```bash
git diff --stat
git diff -- scripts/create-backlog-issues.sh .backlog-drafts
```

Do not run Go tests unless code was changed.

Do not execute the script.

## Output

Summarize:

1. selected draft directory
2. issue drafts included
3. issue drafts excluded
4. labels used
5. script path
6. command to run after maintainer review
7. reminder that running twice may create duplicate issues
