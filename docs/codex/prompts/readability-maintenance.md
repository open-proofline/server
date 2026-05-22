Review the Go backend for readability and maintainability.

Do not add features.
Do not change behaviour.
Do not add a web framework yet.

Goal:
Make the code easier for a human to understand and debug.

Focus on:
- splitting overly large files
- clearer handler names
- clearer route registration
- reducing duplicated request/response helpers
- adding comments only where logic is non-obvious
- keeping private API and emergency viewer servers clearly separated
- preserving tests and existing endpoint behaviour

Do not:
- add React, Node, npm, Gin, Echo, Fiber, chi, OAuth, JWT, Docker Compose, Kubernetes, or new features
- change public JSON field names
- change database schema unless required for a bug
- change the token/security model

After changes:
- run gofmt
- run go test ./...
- summarize what changed and why