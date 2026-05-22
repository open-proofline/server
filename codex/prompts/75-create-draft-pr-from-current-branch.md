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

## First steps

Check the current branch and repository state:

```bash
git status
git branch --show-current
git log --oneline -5
```

Review the issue:

```bash
gh issue view <ISSUE_NUMBER> --repo TheSilkky/safety-recorder
```

Review the diff:

```bash
git diff --stat main...
git diff main...
```

If the diff contains unrelated changes, stop and summarize the problem instead of creating the PR.

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

Create a draft PR:

```bash
gh pr create \
  --repo TheSilkky/safety-recorder \
  --base main \
  --head "$(git branch --show-current)" \
  --draft \
  --title "<short title>" \
  --body "Closes #<ISSUE_NUMBER>

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

- linked issue using `Closes #<ISSUE_NUMBER>`
- concise summary
- validation commands run
- docs updated, if any
- follow-up work, if any
- tests skipped and why, if any
- note that the PR remains draft until maintainer review

## Constraints

- Do not claim production readiness.
- Do not add unrelated changes while creating the PR.
- Do not create public issue/PR content containing raw tokens, secrets, private deployment details, exploit details, or user safety data.

## Output

Summarize:

1. current branch
2. issue linked
3. commits included
4. validation commands run
5. PR URL, if created
6. any manual follow-up required
