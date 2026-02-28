# Deployment Automation Policy

Apply this policy to all deployment and release tasks across all projects.
This is the canonical source for deployment automation rules.

For platform-specific strategies and detailed patterns, see `docs/deployment-platform-strategies.md`.

## Core rules

1. All deployments must be automated via CI/CD. Manual deployment is forbidden.
2. Never run deploy commands locally (`wrangler deploy`, `scp`, `rsync`, `kubectl apply`, `terraform apply` against production).
3. Never SSH into production to deploy code.
4. If a project lacks automated deployment, set it up before deploying.
5. Disable manual deploy scripts in `package.json` when CI/CD exists.

## Workflow structure requirements

### Pre-deploy gates

1. CI must pass before deploy (`workflow_run` with `conclusion == 'success'`).
2. Pre-deploy verification step when health checks are available.
3. Secrets validation step before deploy.

### Deploy concurrency

1. Use `concurrency` group for deploy workflows.
2. Set `cancel-in-progress: false` for production deployments.
3. Use unique group names per environment (`deploy-production`, `release-${{ github.ref }}`).

### Post-deploy verification

1. Run health checks or smoke tests after deployment.
2. Chain verification as a separate job with `needs:` dependency.

### Failure notification

1. Notify on deployment failure (Slack webhook, GitHub Issue, or email).
2. Use `if: always() && !cancelled()` for notification jobs.
3. Include: status, affected service, run URL, and individual job results.

## Secrets management

1. Use GitHub Secrets for all credentials — never hardcode in workflow files.
2. Use environment-scoped secrets for production-only credentials.
3. Validate secret availability before deploy steps (graceful skip if missing).
4. Use 1Password service accounts via `OP_SERVICE_ACCOUNT_TOKEN` for infrastructure secrets.

## Environment protection

1. Use GitHub environment protection rules for production deployments when supported.
2. Assign `environment: production` to deploy jobs.
3. Use `timeout-minutes` on all deploy jobs (10-15 min default).
4. Use minimum required `permissions` per job — never `permissions: write-all`.

## Exceptions and blocked operations

1. If automated deployment cannot be set up (missing secrets, no CI access), complete all safe preparatory work first.
2. Report exactly: what is blocked, what secret or access is missing, and the exact next step to unblock.
3. Emergency hotfixes that bypass CI must be documented and followed by a proper CI deployment immediately after.
4. Local `deploy.sh` scripts are acceptable only as development-time helpers with `--dry-run` support, never for production.

## Verification checklist

1. Trigger a test deployment to confirm the pipeline works end-to-end.
2. Verify deployment success via health checks, smoke tests, or log evidence.
3. Confirm rollback mechanism exists.
4. Verify concurrency settings prevent parallel production deploys.
5. Verify failure notifications reach the intended channel.
6. Verify secrets are not exposed in workflow logs (use masking).

## Reference composition

1. Loaded as Tier 1 baseline rule via `opencode.jsonc`.
2. Platform strategies and release patterns: `docs/deployment-platform-strategies.md`.
