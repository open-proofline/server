Add Docker support for the Go backend.

Requirements:
- Add `server/Dockerfile`.
- Add `server/.dockerignore`.
- Use a multi-stage Alpine-based build.
- Build the `./cmd/api` binary.
- Run as a non-root user.
- Use `/data` as the container data directory.
- Preserve these environment variables:
  - SAFE_PRIVATE_BIND_ADDR
  - SAFE_PUBLIC_BIND_ADDR
  - SAFE_DATA_DIR
  - SAFE_DB_PATH
  - SAFE_MAX_UPLOAD_BYTES
- Default private bind to `0.0.0.0:8080`.
- Default public bind to `0.0.0.0:8081`.
- Expose ports 8080 and 8081.
- Do not add Kubernetes, Compose, reverse proxy config, cloud deployment files, or unrelated features.
- Update README with Docker build/run instructions.
- Run `go test ./...` after changes.