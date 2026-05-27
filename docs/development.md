# Development

Proofline currently contains a Go backend only. Keep changes small, boring, and testable. Repository, module, Docker, and GHCR artifact names may still use `safety-recorder` until an explicit migration is performed.

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

- current name, version, and scope in [../README.md](../README.md)
- planned incident modes in [incident-modes.md](incident-modes.md)
- route details in [api.md](api.md)
- security assumptions in [security-model.md](security-model.md) and [threat-model.md](threat-model.md)
- future key custody/decryption designs in [key-custody.md](key-custody.md), [browser-decryption.md](browser-decryption.md), and [break-glass-key-access.md](break-glass-key-access.md)
- package layout in [code-map.md](code-map.md)
- release notes in [../CHANGELOG.md](../CHANGELOG.md)

Do not claim production readiness unless deployment hardening has actually been implemented. Do not treat the docs-only Proofline rename as a repository, module, Docker image, or GHCR namespace migration.

## Backlog Discipline

New ideas discovered during unrelated work should become issues or backlog items unless they are required to finish the current task. Capture the context, acceptance criteria, tests, docs impact, and out-of-scope items instead of expanding the active diff.

Local backlog drafts are review artifacts. Keep them branch-scoped under `.backlog-drafts/YYYY-MM-DD/<branch-slug>/` or `.backlog-drafts/current/<branch-slug>/`, then treat GitHub Issues as the source of truth after reviewed drafts are created publicly.

## Branch Protection And Required Checks

This repository uses GitHub repository rulesets rather than classic branch
protection.

Current branch rulesets:

| Ruleset | Target | Purpose |
|---|---|---|
| `Protect main` | `~DEFAULT_BRANCH`, currently `main` | Stable release line. Final release PRs and hotfixes merge here. |
| `Protect develop` | `refs/heads/develop` | Next-release integration branch. Normal issue PRs merge here after `v0.5.0`. |
| `Protect release/v*` | `refs/heads/release/v*` | Short-lived release-prep branches such as `release/v0.6.0-prep`. |

All three rulesets are active and block branch deletion and non-fast-forward
updates. They require pull requests before merge, one approving review, stale
approval dismissal on new pushes, and strict required status checks.

Required checks:

- `Go tests`
- `Build Linux binary`
- `Build Docker image`

The rulesets allow merge, squash, and rebase merge methods. `Protect develop`
and `Protect release/v*` require review thread resolution. `Protect main`
currently does not require review thread resolution.

The exported rulesets include a repository-role bypass actor with bypass mode
limited to pull requests. Use bypass only for maintainer-authored changes when
no independent write-access reviewer is available, after required checks pass
and the maintainer has reviewed the diff. Routine collaborator changes should
still receive a qualifying approval.

Do not require tag-only jobs such as `Attest Linux binary`, `Upload release
binary`, or `Publish Docker image` as pull request status checks. Those jobs run
only for trusted release/tag contexts and would make normal PRs unmergeable if
required on PRs.

If the rulesets change, update this section to match the exported rulesets.

## Branch Model

After `v0.5.0`, use this branch flow:

```text
issue branches -> develop
develop -> release/vX.Y.Z-prep
release/vX.Y.Z-prep -> main
tag vX.Y.Z from main
main -> develop sync after release
```

Branch purposes:

- `main` is the stable release line.
- `develop` is the next-release integration branch.
- `release/v*` branches are short-lived release-candidate branches.
- Final `v*` tags are created from `main`.
- Release-candidate tags may be created from release-prep branches when
  validating release automation.

When creating PRs, set the intended base branch explicitly:

- issue work for the next release: base `develop`
- release-prep fixes: base `release/vX.Y.Z-prep`
- final release PR: base `main`
- hotfixes: base `main`, then sync back to `develop`

## CI And Release Automation

The CI workflow runs on pull requests, all branch pushes, and `v*` tags. Pushes
to `main` and `v*` tags also publish Docker image tags to GHCR when package
publishing is available. Workflow-level token permissions stay read-only. A
tag-only binary attestation job can mint release attestations for `v*` tag
pushes, and the trusted Docker publish job can mint and publish GHCR image
attestations.
For `v*` tags, CI also uploads the Linux amd64 binary as a GitHub Release asset.
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
- `Upload release binary` uploads `safety-recorder-linux-amd64` to the matching
  GitHub Release after the binary attestation job passes.
- `Publish Docker image` attaches an attestation to published GHCR images on
  trusted `main` and `v*` tag pushes.

The workflow keeps top-level permissions read-only. `id-token: write` and
`attestations: write` are granted only to jobs that create attestations. The
release binary upload job gets `contents: write` only so it can create a
minimal Release when needed and upload the binary asset. The Docker publish job
also keeps `packages: write` because it pushes images to GHCR. The image
attestation is pushed to the registry without creating a linked artifact storage
record.

For `v*` tag workflows, CI creates a minimal GitHub Release if one does not
already exist, verifies that the tag exists remotely with
`gh release create --verify-tag`, and uploads `safety-recorder-linux-amd64` as
a Release asset.
Release-candidate, alpha, and beta tags are marked as prereleases and are not
promoted as the latest release. The workflow does not overwrite existing
Release assets by default; if the binary asset already exists, the upload step
fails so the maintainer can review the existing asset before retrying.

After a release workflow run, verify the Release and asset with:

```bash
gh release view <tag> --repo TheSilkky/safety-recorder
```

After a release workflow run, verify the downloaded binary artifact with:

```bash
gh attestation verify ./safety-recorder-linux-amd64 \
  -R TheSilkky/safety-recorder \
  --signer-workflow TheSilkky/safety-recorder/.github/workflows/ci.yml \
  --source-ref refs/tags/<tag>
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
- verify the GitHub Release binary asset plus release binary and GHCR image attestations
- verify Docker base-image digest pins still match the intended tag families
- verify private/public route separation is documented
- verify raw tokens, request bodies, uploaded bytes, and Authorization headers are not logged
- verify no local DBs, uploaded blobs, generated binaries, `.env` files, or temporary files are committed
