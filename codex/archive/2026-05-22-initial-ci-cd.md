# Historical Prompt: Initial CI/CD

This prompt is archived.

Current CI/CD should:

- run Go tests from `server/`
- build a Linux amd64 binary artifact
- build the Docker image from `server/Dockerfile`
- publish `ghcr.io/thesilkky/safety-recorder` on pushes to `main` and version tags
- avoid requiring local Docker to work

Use current documentation and workflow files as source of truth.
