Add useful documentation comments to the Go backend.

Do not change behaviour.

Focus on helping a human understand the codebase quickly.

Requirements:

- Add package comments for each internal package explaining its responsibility.
- Add comments for exported types, exported functions, and exported methods in normal Go doc style.
- Add short comments around important safety/security logic, especially:
  - upload size limiting
  - temporary file handling
  - SHA-256 verification
  - duplicate chunk rejection
  - immutable storage / no overwrite behaviour
  - SQLite schema constraints
  - request logging choices
  - why v0.1 has no public authentication
- Add comments explaining the main request flow:
  1. create incident
  2. upload chunk
  3. stream to temp file
  4. hash while reading
  5. verify hash
  6. move into final immutable path
  7. insert metadata
- Do not add noisy comments that merely repeat the code.
- Do not rename public API fields.
- Do not add features.
- Do not change endpoint behaviour.
- Do not add dependencies.
- After commenting, run `gofmt` and `go test ./...`.