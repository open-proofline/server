# Add go vet To CI

## Priority

P2

## Type

maintenance

## Labels

- `backlog`
- `ci`
- `testing`
- `maintenance`

## Summary

Local review guidance recommends `go vet ./...` for larger changes, but CI currently runs only `go test ./...`. Add `go vet` to CI so vet regressions are caught consistently.

## Context

`docs/development.md` and `codex/prompts/90-release-check.md` mention `go vet ./...`. `.github/workflows/ci.yml` currently has a `Go tests` job that runs `go test ./...` but no vet step.

## Proposed change

Add a CI step or job that runs `go vet ./...` from `server/`. Decide whether it should run before tests, after tests, or as a separate required check.

## Acceptance criteria

- [ ] CI runs `go vet ./...` from `server/`.
- [ ] CI output names the vet step clearly.
- [ ] Docs/release checklist stay consistent with CI.
- [ ] The workflow still builds the binary and Docker image only after tests pass.

## Tests / validation

- [ ] `cd server && go test ./...`
- [ ] `cd server && go vet ./...`
- [ ] workflow syntax reviewed

## Out of scope

Do not add unrelated linters, formatters, third-party CI services, or broad workflow refactors.

## Notes

Related file: `.github/workflows/ci.yml`.
