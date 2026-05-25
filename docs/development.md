# Development

Safety Recorder is a Go backend only. Keep changes small, boring, and testable.

## AI Assistance

This repository uses OpenAI Codex as an AI-assisted development tool. Codex may generate or modify code and documentation, but changes are accepted only after maintainer review and testing.

The maintainer remains responsible for correctness, security, licensing, releases, deployment decisions, and real-world use. Use of Codex does not imply endorsement, audit, certification, or maintenance by OpenAI.

For rollback points, scoped prompts, review steps, and backlog handling, see [codex-change-control.md](codex-change-control.md).

## Repository Layout

```text
server/
  cmd/api          API server entry point
  cmd/simclient    simulator CLI
  internal/config  environment configuration and HTTP timeout parsing
  internal/db      SQLite setup, schema_migrations, and compatibility migrations
  internal/envelope client-side chunk encryption envelope helpers
  internal/httpapi HTTP handlers, muxes, middleware, bundles, web assets
  internal/incidents incident, stream, chunk, checkin, and token repository code
  internal/storage local immutable blob storage
  migrations       embedded SQLite schema
docs/              project documentation
docs/reports/      public technical review reports and report prompts
```

See [code-map.md](code-map.md) for a package-level walkthrough.

## Technical Review Reports

Public technical review reports live in [reports/](reports/). They are
AI-assisted engineering review artifacts reviewed by the maintainer, not formal
security audits, penetration tests, compliance certifications, or production
readiness endorsements.

Use the Phase 1 prompt in
[reports/prompts/phase-1-deep-research-technical-review.md](reports/prompts/phase-1-deep-research-technical-review.md)
to create a source-cited draft outside Codex. Use
[../codex/prompts/95-validate-deep-research-report.md](../codex/prompts/95-validate-deep-research-report.md)
for the Phase 2 Codex cleanup and public-safety validation pass.

## Commands

From `server/`:

```bash
gofmt -w .
go test ./...
```

Use `go vet ./...` when reviewing larger changes:

```bash
go vet ./...
```

## Documentation Checks

When editing docs, keep these claims aligned:

- current version and scope in [../README.md](../README.md)
- route details in [api.md](api.md)
- security assumptions in [security-model.md](security-model.md) and [threat-model.md](threat-model.md)
- future key custody/decryption designs in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md)
- package layout in [code-map.md](code-map.md)
- release notes in [../CHANGELOG.md](../CHANGELOG.md)

Do not claim production readiness unless deployment hardening has actually been implemented.

## Backlog Discipline

New ideas discovered during unrelated work should become issues or backlog items unless they are required to finish the current task. Capture the context, acceptance criteria, tests, docs impact, and out-of-scope items instead of expanding the active diff.

## Branch Protection And Required Checks

The default branch is protected by an active GitHub repository ruleset named
`Protect main`. It targets `~DEFAULT_BRANCH`, currently `main`.

Current ruleset requirements:

- block branch deletion
- block non-fast-forward pushes
- require pull requests before merge
- require one approving review
- dismiss stale approvals when new commits are pushed
- require the latest `main` to pass before merge
- require the `Go tests`, `Build Linux binary`, and `Build Docker image` CI jobs
- allow merge, squash, and rebase merge methods
- allow repository admins to bypass only through pull requests

These settings are implemented as a repository ruleset, not classic branch
protection. If the ruleset changes, update this section to match the exported
ruleset.

The admin bypass is for maintainer-authored changes when no independent
write-access reviewer is available. Use it only after required checks pass and
the maintainer has reviewed the diff. Routine collaborator changes should still
receive a qualifying approval.

The CI workflow runs on pull requests, all branch pushes, and `v*` tags. Pushes
to `main` and `v*` tags also publish Docker image tags to GHCR when package
publishing is available. Workflow-level token permissions stay read-only. A
tag-only binary attestation job can mint release attestations for `v*` tag
pushes, and the trusted Docker publish job can mint and publish GHCR image
attestations.
`packages: write` is granted only to the trusted Docker publish job.

## Pinned GitHub Actions

External GitHub Actions in [../.github/workflows/ci.yml](../.github/workflows/ci.yml)
are pinned to full 40-character commit SHAs. Keep the intended upstream version
tag in a same-line comment, for example `owner/action@<sha> # v1`, so reviewers
can see the expected release line and Dependabot can update that version
documentation.

GitHub documents full-length commit SHA pinning as the immutable release model
for actions. When updating an action, verify that the SHA belongs to the
upstream action repository and corresponds to the intended tag or release, not a
fork. Review the action release notes, changed inputs, changed permissions, and
the workflow diff before merging.

Dependabot remains enabled for the `github-actions` ecosystem in
[../.github/dependabot.yml](../.github/dependabot.yml). Prefer reviewing those
pull requests rather than bypassing maintainer review. For manual updates, use
`git ls-remote --tags https://github.com/<owner>/<repo>.git 'refs/tags/<tag>' 'refs/tags/<tag>^{}'`
and pin the peeled commit SHA when the tag is annotated. After any action
update, confirm the required PR checks still pass and that trusted GHCR
publication remains limited to `main` and `v*` tag pushes.

## Artifact Attestations

Release provenance attestations are generated by the CI workflow:

- `Attest Linux binary` attests `safety-recorder-linux-amd64` on `v*` tag
  pushes.
- `Publish Docker image` attaches an attestation to published GHCR images on
  trusted `main` and `v*` tag pushes.

The workflow keeps top-level permissions read-only. `id-token: write` and
`attestations: write` are granted only to jobs that create attestations. The
Docker publish job also keeps `packages: write` because it pushes images to
GHCR. The image attestation is pushed to the registry without creating a linked
artifact storage record.

After a release workflow run, verify the downloaded binary artifact with:

```bash
gh attestation verify ./safety-recorder-linux-amd64 \
  -R TheSilkky/safety-recorder \
  --signer-workflow TheSilkky/safety-recorder/.github/workflows/ci.yml
```

Verify a published container image with:

```bash
docker login ghcr.io
gh attestation verify oci://ghcr.io/thesilkky/safety-recorder:<tag> \
  -R TheSilkky/safety-recorder \
  --signer-workflow TheSilkky/safety-recorder/.github/workflows/ci.yml
```

Use `--source-ref refs/tags/<tag>` for a release tag or
`--source-ref refs/heads/main` for a main-branch image when you want the CLI to
enforce the expected source ref. Attestation verification confirms provenance
from the expected workflow; it does not prove that an artifact is vulnerability
free or production-ready.

## Pinned Docker Base Images

Base images in [../server/Dockerfile](../server/Dockerfile) are pinned with
`tag@sha256:<digest>` references. Keep the human-readable tag before the digest
so reviewers can see the intended release family while builds stay tied to an
immutable manifest.

When refreshing a Docker base-image digest, inspect the current manifest for the
intended tag:

```bash
docker buildx imagetools inspect docker.io/library/golang:1.26-alpine
docker buildx imagetools inspect docker.io/library/alpine:3.23
```

Dependabot is enabled for the `docker` ecosystem in
[../.github/dependabot.yml](../.github/dependabot.yml) and monitors the
Dockerfile under `server/`. Prefer reviewing Dependabot pull requests for
routine base-image refreshes.

Update only the digest for the same intended tag unless the issue, Dependabot
pull request, or release explicitly calls for a version-family change. Review
the upstream image tag, version, architecture coverage, release notes or
changelog where available, and the Dockerfile diff before merging. After a
digest update, run the Docker build locally when practical and confirm the
required CI Docker build still passes.

Only create `v*` tags after the release checklist is complete and the tagged
commit has passed CI. If an emergency fix is needed, keep the change narrow,
preserve review discipline, and document any skipped validation in the release
or follow-up notes.

Codex-generated changes should be reviewed like any other contribution: inspect
the diff, confirm the scope matches the issue, run the relevant validation
commands, and make sure security warnings and private/public listener boundaries
were not weakened.

## Release Checklist

Before tagging:

- run `gofmt -w .`
- run `go test ./...`
- run `go vet ./...` when practical
- verify README badges and links
- verify `LICENSE` and `SECURITY.md`
- verify Docker/GHCR notes
- verify release binary and GHCR image attestations
- verify Docker base-image digest pins still match the intended tag families
- verify private/public route separation is documented
- verify raw tokens, request bodies, uploaded bytes, and Authorization headers are not logged
- verify no local DBs, uploaded blobs, generated binaries, `.env` files, or temporary files are committed
