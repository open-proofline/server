# Codex Prompt: Readability Maintenance

Review the Go backend for readability and maintainability.

Do not add features.
Do not change endpoint behaviour.
Do not add a web framework unless explicitly requested.

## Goal

Make the code easier for a human to understand and debug while preserving behaviour.

## Focus on

- splitting overly large files
- clearer handler names
- clearer route registration
- clearer package responsibilities
- reducing duplicated request/response helpers
- comments around non-obvious logic
- private API and emergency viewer separation
- stream/bundle logic readability
- simulator readability
- tests that clearly describe behaviour

## Do not

- add React
- add Node
- add npm
- add Gin, Echo, Fiber, or chi
- add OAuth
- add JWT
- add user accounts
- add Docker Compose
- add Kubernetes
- add new features
- change public JSON field names
- change database schema unless required for a bug
- change token/security model
- reformat unrelated docs unless necessary

## Validation

After changes:

```bash
gofmt -w .
go test ./...
```

Summarize what changed and why.
