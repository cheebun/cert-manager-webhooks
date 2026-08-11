# Remote Repository Setup

Step-by-step procedure for creating (or recreating) the GitHub remote repository
with all required permissions and branch protection.

## Prerequisites

- [GitHub CLI (`gh`)](https://cli.github.com/) authenticated
- Local repository with `main` and `gh-pages` branches ready

## 1. Delete existing repo (if recreating)

```bash
gh repo delete cheebun/cert-manager-webhooks --yes
```

## 2. Create repository

```bash
gh repo create cheebun/cert-manager-webhooks \
  --public \
  --description "A monorepo of cert-manager DNS01 webhook solvers for multiple DNS providers"
```

## 3. Configure merge strategy

Only allow **Squash and Merge**; auto-delete branches after merge:

```bash
gh api repos/cheebun/cert-manager-webhooks --method PATCH \
  --field allow_squash_merge=true \
  --field allow_merge_commit=false \
  --field allow_rebase_merge=false \
  --field squash_merge_commit_title=PR_TITLE \
  --field squash_merge_commit_message=PR_BODY \
  --field delete_branch_on_merge=true
```

## 4. Configure GitHub Actions permissions

Grant Actions write access and allow Release Please to create PRs:

```bash
gh api repos/cheebun/cert-manager-webhooks/actions/permissions/workflow \
  --method PUT \
  --field default_workflow_permissions=write \
  --field can_approve_pull_request_reviews=true
```

## 5. Wait for GitHub to initialize

```bash
sleep 30
```

> GitHub needs time to provision the repository infrastructure (webhooks,
> Actions runners, etc.). Pushing too early may cause workflow triggers
> to be missed.

## 6. Push branches

```bash
git push -u origin main --force
git push origin gh-pages --force
```

## 7. Verify

```bash
# Check repo settings
gh api repos/cheebun/cert-manager-webhooks \
  --jq '{allow_squash_merge, allow_merge_commit, allow_rebase_merge, delete_branch_on_merge}'

# Check Actions permissions
gh api repos/cheebun/cert-manager-webhooks/actions/permissions/workflow \
  --jq '{default_workflow_permissions, can_approve_pull_request_reviews}'

# Check CI runs
gh run list --repo cheebun/cert-manager-webhooks --limit 5
```

## One-liner

For convenience, the full flow as a single script:

```bash
#!/usr/bin/env bash
set -euo pipefail

REPO="cheebun/cert-manager-webhooks"
DESC="A monorepo of cert-manager DNS01 webhook solvers for multiple DNS providers"

# Delete & create
gh repo delete "$REPO" --yes 2>/dev/null || true
gh repo create "$REPO" --public --description "$DESC"

# Merge settings
gh api "repos/$REPO" --method PATCH \
  -F allow_squash_merge=true -F allow_merge_commit=false -F allow_rebase_merge=false \
  -F squash_merge_commit_title=PR_TITLE -F squash_merge_commit_message=PR_BODY \
  -F delete_branch_on_merge=true --silent

# Actions permissions
gh api "repos/$REPO/actions/permissions/workflow" --method PUT \
  -f default_workflow_permissions=write -F can_approve_pull_request_reviews=true --silent

# Wait & push
sleep 30
git push -u origin main --force
git push origin gh-pages --force
```

## Notes

- **`gh-pages` branch** is required by `helm/chart-releaser-action` for GitHub
  Pages-based Helm chart repository hosting.
- **Squash merge** produces a clean linear history where each commit corresponds
  to one PR — works well with Release Please changelog generation.
- **`GITHUB_TOKEN` scope**: the default token has write access to packages under
  the repository's own namespace (`ghcr.io/cheebun/cert-manager-webhooks/*`).
  Pushing to a different namespace (e.g., `ghcr.io/cheebun/charts/*`) requires
  a Personal Access Token with `write:packages` scope.
