# Document Branch Protection And Required Checks

## Priority

P3

## Type

documentation

## Labels

- `backlog`
- `docs`
- `ci`

## Summary

The repository has CI, Docker/GHCR publishing, and release guidance, but no documented branch-protection or required-check expectations. Add a short maintainer-facing policy.

## Context

`.github/workflows/ci.yml` runs tests, binary build, and Docker build/publish conditions. `docs/development.md` has a release checklist, and `README.md` shows CI/GHCR badges. There is no short statement of which checks should be required before merging or tagging.

## Proposed change

Document recommended branch protection for `main`, required CI checks, tag/release expectations, and how to handle emergency fixes without bypassing review discipline.

## Acceptance criteria

- [ ] Docs list recommended required checks for pull requests.
- [ ] Docs mention tag/release expectations for `v*` publishing.
- [ ] Docs explain how Codex-generated changes should be reviewed before merge.
- [ ] Docs do not imply GitHub settings are already configured unless verified.

## Tests / validation

- [ ] docs updated, if relevant
- [ ] no Go tests required unless code changes

## Out of scope

Do not modify repository settings through GitHub, add new GitHub Apps, or change workflow behavior in this issue.

## Notes

Related docs: `docs/development.md`, `docs/codex-change-control.md`, `README.md`; related workflow: `.github/workflows/ci.yml`.
