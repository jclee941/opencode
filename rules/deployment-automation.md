# Deployment Automation Policy

Apply this policy to all deployment and release tasks across all projects.

This file is the canonical source for deployment automation rules.

## Core rule

1. All deployments must be automated via CI/CD. Manual deployment is forbidden.
2. Never run deploy commands locally (`wrangler deploy`, `scp`, `rsync`, `kubectl apply`, `terraform apply` against production).
3. Never SSH into production to deploy code.
4. If a project lacks automated deployment, set it up before deploying.
5. Disable manual deploy scripts in `package.json` when CI/CD exists (reference pattern: safewallet `"deploy": "node -e \"console.error('Manual deploy is disabled.')\""`).

## Deployment strategies by platform

### GitHub Actions (default CI/CD)

1. Use GitHub Actions as the default CI/CD platform for all GitHub-hosted repos.
2. Workflows must live in `.github/workflows/` and be committed to the repository.
3. Trigger deployments via:
   - `workflow_run` chaining: CI success on protected branch triggers deploy (preferred for multi-stage).
   - Push to protected branches (`main`, `master`, `production`).
   - Tag events (`v*`) for versioned releases.
   - `workflow_dispatch` for manual re-runs only (not primary deploy trigger).
4. Reusable workflows (`workflow_call`) for shared logic across services (reference: `_terraform-plan.yml`, `_terraform-apply.yml`).
5. Pin action versions to commit SHAs, not mutable tags (reference: `actions/checkout@de0fac2e...`).

### Cloudflare Workers / Pages

1. Deploy via GitHub Actions using `cloudflare/wrangler-action` or equivalent CI step.
2. Alternatively, connect repo to Cloudflare for git-ref-based push-to-deploy.
3. Never run `wrangler deploy` or `wrangler pages deploy` manually.
4. Branch-based preview deployments are encouraged for staging/review.
5. Production deploys must trigger only from the configured production branch.

### Docker / Container images

1. Build images in CI (GitHub Actions or equivalent).
2. Push to GHCR (`ghcr.io`) or configured registry via CI — never push manually.
3. Use matrix strategy for multi-service builds (reference: blacklist 5-service matrix).
4. Tag images with both version and `latest`.
5. Include checksums (`sha256sum`) for release bundles.

### Terraform / Infrastructure

1. Infrastructure changes must go through CI/CD.
2. Never run `terraform apply` locally against production state.
3. Use plan-and-approve workflow:
   - PR triggers `terraform plan` with output as PR comment.
   - Merge triggers `terraform apply`.
4. Store Terraform state remotely with locking enabled.
5. Use reusable workflows for multi-workspace plans (reference: `_terraform-plan.yml`).
6. Run on self-hosted runners when accessing homelab infrastructure.

### GitLab CI (exception)

1. `hycu_fsds` uses GitLab CI (`.gitlab-ci.yml`) — this is the only approved GitLab CI project.
2. Same automation-first rules apply: no manual deploys.

## Workflow structure requirements

### Pre-deploy gates

1. CI must pass before deploy (`workflow_run` with `conclusion == 'success'`).
2. Pre-deploy verification step when health checks are available.
3. Secrets validation step before deploy (reference: worker-deploy secrets_check pattern).

### Deploy concurrency

1. Use `concurrency` group for deploy workflows.
2. Set `cancel-in-progress: false` for production deployments — never cancel an in-progress deploy.
3. Use unique group names per environment (`deploy-production`, `release-${{ github.ref }}`).

### Post-deploy verification

1. Run health checks or smoke tests after deployment.
2. Chain verification as a separate job with `needs:` dependency.
3. Reference pattern: safewallet `verify-before-deploy` -> `deploy` -> `verify-production`.

### Failure notification

1. Notify on deployment failure (Slack webhook, GitHub Issue, or email).
2. Use `if: always() && !cancelled()` for notification jobs.
3. Include: status, affected service, run URL, and individual job results.
4. Reference patterns: safewallet Slack notify, terraform notify-failure issue creation.

## Release strategies

### Tag-based releases

1. Use semantic versioning tags (`v*`) to trigger release workflows.
2. Validate VERSION file matches tag before proceeding.
3. Check CHANGELOG.md has entry for the release version.
4. Create GitHub Release with release notes and artifacts.

### Conventional commit auto-release

1. Use `workflow_run` to trigger release after CI succeeds on main branch.
2. Analyze conventional commits since last tag for semver bump.
3. Auto-generate changelog from commit history.
4. Reference: resume `mathieudutour/github-tag-action` pattern.

### Dry-run support

1. Release workflows should support `dry_run` input via `workflow_dispatch`.
2. Dry run builds and packages without publishing or pushing.

## Secrets management

1. Use GitHub Secrets for all credentials — never hardcode in workflow files.
2. Use environment-scoped secrets for production-only credentials.
3. Validate secret availability before deploy steps (graceful skip if missing).
4. Use 1Password service accounts via `OP_SERVICE_ACCOUNT_TOKEN` for infrastructure secrets.
5. Reference `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID` pattern for CF deployments.

## Environment protection

1. Use GitHub environment protection rules for production deployments when supported.
2. Assign `environment: production` to deploy jobs.
3. Use `timeout-minutes` on all deploy jobs (10-15 min default).
4. Use minimum required `permissions` per job — never `permissions: write-all`.

## Anti-patterns (NEVER)

1. Never run `wrangler deploy` from local machine.
2. Never run `terraform apply` from local machine against production.
3. Never push Docker images from local machine.
4. Never create GitHub releases manually when automation exists.
5. Never bypass CI by deploying from a non-protected branch.
6. Never store deploy credentials in `.env` files committed to the repo.
7. Never use `pull_request_target` for deploy triggers (security risk).

## Known violations to remediate

These existing manual deploy patterns should be migrated to CI/CD:

| Project | File | Current Pattern | Target |
|---------|------|-----------------|--------|
| propose | `package.json` | `wrangler deploy --env production` | GitHub Actions or CF git-ref |
| resume | `package.json` | CLI-based worker deploy | GitHub Actions |
| resume | `tools/scripts/deployment/deploy.sh` | Shell script deploy | GitHub Actions |
| youtube | `deploy/ffmpeg-worker/deploy.sh` | Shell script deploy | GitHub Actions |
| splunk | `scripts/deploy/deploy.sh` | Shell script deploy | GitHub Actions |
| terraform | `200-oc/opencode/deploy.sh` | Shell script deploy | GitHub Actions or Makefile CI |

## Exceptions and blocked operations

1. If automated deployment cannot be set up (missing secrets, no CI access), complete all safe preparatory work first.
2. Report exactly: what is blocked, what secret or access is missing, and the exact next step to unblock.
3. Emergency hotfixes that bypass CI must be documented and followed by a proper CI deployment immediately after.
4. Local `deploy.sh` scripts are acceptable only as development-time helpers with `--dry-run` support, never for production.

## Verification checklist

After setting up or modifying deployment automation:

1. Trigger a test deployment to confirm the pipeline works end-to-end.
2. Verify deployment success via health checks, smoke tests, or log evidence.
3. Confirm rollback mechanism exists (revert commit, re-deploy previous tag, or platform rollback).
4. Verify concurrency settings prevent parallel production deploys.
5. Verify failure notifications reach the intended channel.
6. Verify secrets are not exposed in workflow logs (use masking).
