# Codex Code Review Prompt

Review the current changes for correctness, security, and scope control.

Do not add features unless needed to fix a bug.

Focus on:

- upload overwrite risks
- temp file cleanup
- SHA-256 verification
- duplicate chunk rejection
- upload size limits
- SQLite constraints
- token leakage
- raw token logging
- path traversal
- template escaping
- tests that do not assert important behaviour

Output:

1. Critical issues
2. Important issues
3. Minor issues
4. Suggested minimal fixes

If you make changes, keep them small and run:

```bash
go test ./...