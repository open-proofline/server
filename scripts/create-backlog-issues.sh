#!/usr/bin/env bash
set -euo pipefail

REPO="TheSilkky/safety-recorder"

# Review .backlog-drafts/create-issues-review.md before running this script.
# Labels must already exist in the repository or gh issue create may fail.
# Running this script more than once may create duplicate GitHub issues.

if ! command -v gh >/dev/null 2>&1; then
  echo "error: GitHub CLI (gh) is required" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "error: gh auth status failed; authenticate with gh before running" >&2
  exit 1
fi

cat <<'WARNING'
About to create backlog issues in TheSilkky/safety-recorder.

Review each draft and .backlog-drafts/create-issues-review.md before continuing.
Labels must already exist in the repository or issue creation may fail.
Running this script more than once may create duplicate issues.
WARNING

read -r -p "Create these GitHub issues now? [y/N] " confirm
case "$confirm" in
  y|Y|yes|YES)
    ;;
  *)
    echo "Canceled."
    exit 0
    ;;
esac

create_issue() {
  local file="$1"
  local title="$2"
  local labels="$3"

  echo
  echo "Creating issue from ${file}"
  gh issue create \
    --repo "$REPO" \
    --title "$title" \
    --body-file "$file" \
    --label "$labels"
}

create_issue ".backlog-drafts/001-add-default-emergency-token-expiry-policy.md" \
  "Add Default Emergency-Token Expiry Policy" \
  "backlog,security,maintenance"

create_issue ".backlog-drafts/002-add-reverse-proxy-and-wireguard-deployment-examples.md" \
  "Add Reverse Proxy And WireGuard Deployment Examples" \
  "backlog,deployment,docs"

create_issue ".backlog-drafts/003-add-rate-limiting-guidance-and-proxy-examples.md" \
  "Add Rate Limiting Guidance And Proxy Examples" \
  "backlog,security,deployment,docs"

create_issue ".backlog-drafts/004-update-emergency-viewer-dom-during-polling.md" \
  "Update Emergency Viewer DOM During Polling" \
  "backlog,bug,testing"

create_issue ".backlog-drafts/005-define-retention-backup-and-secure-deletion-policy.md" \
  "Define Retention, Backup, And Secure Deletion Policy" \
  "backlog,security,deployment,docs"

create_issue ".backlog-drafts/006-add-go-vet-to-ci.md" \
  "Add go vet To CI" \
  "backlog,ci,testing,maintenance"

create_issue ".backlog-drafts/007-document-branch-protection-and-required-checks.md" \
  "Document Branch Protection And Required Checks" \
  "backlog,docs,ci"

create_issue ".backlog-drafts/008-design-production-key-sharing-and-emergency-access.md" \
  "Design Production Key Sharing And Emergency Access" \
  "backlog,security,docs,ios"

create_issue ".backlog-drafts/009-plan-ios-local-recorder-prototype.md" \
  "Plan iOS Local Recorder Prototype" \
  "backlog,ios,docs"

echo
echo "Done."
