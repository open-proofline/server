# Codex Prompt: Create Draft Pull Request from Current Branch

Prepare a draft pull request for the current checked-out branch.

Do **not** change application code.
Do **not** create a new branch.
Do **not** merge anything.
Do **not** mark the PR ready for review unless explicitly requested.

## Inputs

Issue number: `<ISSUE_NUMBER>`

Repository:

```text
TheSilkky/safety-recorder
```

Target base branch:

```text
<TARGET_BASE_BRANCH>
```

Examples:

```text
main
develop
release/v0.5.0-prep
```

If no target base branch is supplied, infer it only from explicit maintainer wording or current release workflow. If the intended base branch is unclear, stop and ask before creating the PR.

## First steps

Check the current branch and repository state:

```bash
git status --short --branch --untracked-files=all
CURRENT_BRANCH="$(git branch --show-current)"
git rev-parse HEAD
git log --oneline -5
```

Check the target base branch:

```bash
git fetch origin "<TARGET_BASE_BRANCH>"
git rev-parse "origin/<TARGET_BASE_BRANCH>"
```

Do not create a PR if:

- the current branch is the same as `<TARGET_BASE_BRANCH>`
- the target base branch does not exist on `origin`
- the target base branch was inferred ambiguously
- the working tree contains unrelated changes

Review the issue:

```bash
gh issue view <ISSUE_NUMBER> --repo TheSilkky/safety-recorder
```

Review the diff against the target base branch:

```bash
git diff --stat "origin/<TARGET_BASE_BRANCH>..."
git diff "origin/<TARGET_BASE_BRANCH>..."
```

If the diff contains unrelated changes, stop and summarize the problem instead of creating the PR.

## Base branch policy

Use the supplied target base branch for both diff review and PR creation.

Do not assume `main` is the base branch.

Expected examples:

```text
feature issue work for next release -> base develop
release-prep fix -> base release/v0.5.0-prep
final release PR -> base main
hotfix branch -> base main
```

GitHub CLI supports specifying the PR base branch with `--base` / `-B`. If `--base` is omitted, GitHub CLI may use branch config or the repository default branch, which is not safe for release-prep or develop workflows.

## Validation before PR

If Go code changed, run:

```bash
cd server
gofmt -w .
go test ./...
go vet ./...
```

If only Markdown changed, inspect docs and links manually. Go tests are not required unless code changed.

If simulator behaviour is relevant:

```bash
cd server
go run ./cmd/simclient --chunks 5 --interval 1s --download-bundle
```

Do not claim validation passed unless it actually ran.

## PR creation

Push the current branch:

```bash
git push -u origin "$(git branch --show-current)"
```

Create a draft PR against the target base branch:

```bash
gh pr create \
  --repo TheSilkky/safety-recorder \
  --base "<TARGET_BASE_BRANCH>" \
  --head "$(git branch --show-current)" \
  --draft \
  --title "<short title>" \
  --body "Closes #<ISSUE_NUMBER>

## Target base branch
- Base: \`<TARGET_BASE_BRANCH>\`
- Head: \`$(git branch --show-current)\`

## Summary
- ...

## Validation
- [ ] cd server && go test ./...
- [ ] cd server && go vet ./...
"
```

If validation failed but a PR is still useful, keep it as draft and clearly state what failed in the PR body.

## PR body requirements

The PR body should include:

- linked issue using `Closes #<ISSUE_NUMBER>`, unless the issue should not close automatically
- target base branch
- current head branch
- concise summary
- validation commands run
- docs updated, if any
- follow-up work, if any
- tests skipped and why, if any
- whether the issue was generated from a different reviewed branch/ref and whether it was revalidated against this PR base
- whether key custody/decryption assumptions changed; if so, link the explicit design and docs updates
- note that the PR remains draft until maintainer review

## Constraints

- Do not claim production readiness.
- Do not add unrelated changes while creating the PR.
- Do not treat server-side decryption or server-side key storage as permanently forbidden, but confirm any key custody/decryption change is explicit, documented, reviewed, and in scope.
- Do not create public issue/PR content containing raw tokens, secrets, private deployment details, exploit details, or user safety data.
- Do not use `--base main` unless `main` is explicitly the intended target base branch.

## Output

Summarize:

1. current branch
2. target base branch
3. issue linked
4. commits included
5. validation commands run
6. PR URL, if created
7. any manual follow-up required
