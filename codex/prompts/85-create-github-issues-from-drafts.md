Core rule: public issue drafts must include `## Priority`, `## Type`, `## Labels`,
and `## Branch scope`. Issue-creation scripts must fail closed when those sections
are missing and must pass labels to `gh issue create`.

# Codex Prompt Patch: 85 Create GitHub Issues From Drafts

Apply this patch to `codex/prompts/85-create-github-issues-from-drafts.md`.

## Required draft metadata

Verify every included public draft contains:

```md
## Priority
## Type
## Labels
## Branch scope
```

If any required section is missing, exclude the draft and list it in the output summary.

The `## Labels` section must contain backtick-wrapped GitHub label names as bullet items, for example:

```md
## Labels

Suggested GitHub labels:

- `backlog`
- `security`
- `ci`
```

Every public draft must include:

```md
- `backlog`
```

If a draft is missing `backlog`, exclude it unless the maintainer explicitly approves creating the issue without the backlog label.

## Label handling

The generated script must pass labels to `gh issue create`.

Use one `--label` argument per label.

Example:

```bash
gh issue create \
  --repo "$REPO" \
  --title "$title" \
  --body-file "$draft" \
  --label "backlog" \
  --label "security" \
  --label "ci"
```

Before generating the script, list repository labels if GitHub CLI is available:

```bash
gh label list --repo TheSilkky/safety-recorder --limit 200
```

If a draft contains a label that does not exist in the repository, exclude that draft and list the missing label in the output summary.

Do not create labels unless the maintainer explicitly asks.

Do not silently create unlabeled issues.

Do not silently drop labels.

## Priority handling

Priority remains in the issue body under `## Priority`.

Do not convert priority to a GitHub label unless the repository has matching priority labels and the maintainer explicitly asks to use them.

If priority is missing, exclude the draft and list it in the output summary.

## Review exclusions

Exclude any draft that:

- is marked sensitive
- says not to create publicly
- lives under `private-notes/`
- is missing `## Priority`
- is missing `## Type`
- is missing `## Labels`
- is missing `## Branch scope`
- is missing the `backlog` label
- references a label that does not exist in the repository
- says `sensitive-do-not-publicly-file`
- says `revalidate-on-main-or-develop` without a maintainer note that revalidation is complete
- contains raw tokens
- contains secrets
- contains private deployment details
- contains user safety data
- contains exploit details that should go through `SECURITY.md`

## Generated script requirements

The script should:

```bash
set -euo pipefail
```

It should define:

```bash
REPO="TheSilkky/safety-recorder"
DRAFT_DIR="<selected branch-scoped draft dir>"
```

For each included draft, it should call:

```bash
gh issue create \
  --repo "$REPO" \
  --title "$title" \
  --body-file "$draft" \
  --label "backlog" \
  --label "<other-label>"
```

Use safe quoting for file paths, titles, and labels.

## Optional helper output

Create or update:

```text
.backlog-drafts/<selected-directory>/create-issues-review.md
```

Include:

- selected draft directory
- branch scope for that directory
- issue drafts included
- issue drafts excluded
- priorities found
- labels used
- missing/non-existent labels found
- command to run the script
- warning that running twice may create duplicates
- warning that branch-scoped drafts should be revalidated if the source branch moved

## Output summary

Summarize:

1. selected draft directory
2. selected draft branch scope
3. issue drafts included
4. issue drafts excluded
5. priorities found
6. labels used
7. labels missing from repository, if any
8. script path
9. command to run after maintainer review
10. branch-scope/revalidation warnings
11. reminder that running twice may create duplicate issues
