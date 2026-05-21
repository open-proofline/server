# Task: Add GitHub Actions builds for Go binary and Docker image

Add automated GitHub Actions CI/CD for this private repository.

Do not change application behaviour.

## Goals

Implement a GitHub Actions workflow that:

1. Runs Go tests on pull requests and pushes.
2. Builds Linux binaries as downloadable workflow artifacts.
3. Builds the Docker image from `server/Dockerfile`.
4. Publishes the Docker image to GitHub Container Registry on pushes to `main` and version tags.
5. Does not require local Docker to work.

The target image should be:

```text
ghcr.io/${{ github.repository }}