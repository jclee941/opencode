# Deployment Platform Strategies

Detailed platform-specific patterns, release strategies, and CI/CD templates.
Executive rules and core policy are in `rules/deployment-automation.md`.

## GitHub Actions

### Standard deploy workflow structure

```yaml
name: Deploy
on:
  workflow_run:
    workflows: ["CI"]
    types: [completed]
    branches: [main]

concurrency:
  group: deploy-production
  cancel-in-progress: false

jobs:
  deploy:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    runs-on: ubuntu-latest
    timeout-minutes: 15
    environment: production
    permissions:
      contents: read
      deployments: write
    steps:
      - uses: actions/checkout@v4
      - name: Validate secrets
        run: |
          if [ -z "${{ secrets.DEPLOY_TOKEN }}" ]; then
            echo "::error::DEPLOY_TOKEN not configured"
            exit 1
          fi
      - name: Deploy
        run: # platform-specific deploy command
        env:
          DEPLOY_TOKEN: ${{ secrets.DEPLOY_TOKEN }}

  verify:
    needs: deploy
    runs-on: ubuntu-latest
    timeout-minutes: 5
    steps:
      - name: Health check
        run: curl -f https://app.example.com/health

  notify:
    needs: [deploy, verify]
    if: always() && !cancelled()
    runs-on: ubuntu-latest
    steps:
      - name: Notify
        run: # Slack webhook or similar
```

### 1Password secrets integration

```yaml
steps:
  - uses: 1password/load-secrets-action@v2
    with:
      export-env: true
    env:
      OP_SERVICE_ACCOUNT_TOKEN: ${{ secrets.OP_SERVICE_ACCOUNT_TOKEN }}
      API_KEY: op://Infrastructure/api-credentials/api-key
      DB_PASSWORD: op://Infrastructure/database/password
```

## Cloudflare Workers

```yaml
jobs:
  deploy:
    steps:
      - uses: actions/checkout@v4
      - uses: cloudflare/wrangler-action@v3
        with:
          apiToken: ${{ secrets.CLOUDFLARE_API_TOKEN }}
          command: deploy --env production
```

## Docker / Container Registry

```yaml
jobs:
  build-and-push:
    steps:
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v5
        with:
          push: true
          tags: ghcr.io/${{ github.repository }}:${{ github.sha }}
```

## Terraform

```yaml
jobs:
  plan:
    steps:
      - uses: hashicorp/setup-terraform@v3
      - run: terraform init
      - run: terraform plan -out=tfplan
      - uses: actions/upload-artifact@v4
        with:
          name: tfplan
          path: tfplan

  apply:
    needs: plan
    environment: production
    steps:
      - uses: actions/download-artifact@v4
        with:
          name: tfplan
      - run: terraform apply tfplan
```

## GitLab CI

```yaml
deploy:
  stage: deploy
  environment:
    name: production
    url: https://app.example.com
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
      when: on_success
  script:
    -  # platform-specific deploy
```

## Release strategies

### Tag-based releases

```yaml
on:
  push:
    tags: ["v*.*.*"]

jobs:
  release:
    steps:
      - uses: actions/checkout@v4
      - name: Create Release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
```

### Conventional commit releases

Use `semantic-release` or `release-please` for automated versioning:

```yaml
jobs:
  release:
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          release-type: node
```

### Dry-run support

All deploy scripts must support `--dry-run` for local testing:

```bash
#!/usr/bin/env bash
set -euo pipefail

DRY_RUN=${DRY_RUN:-false}

if [ "$DRY_RUN" = "true" ]; then
  echo "[DRY RUN] Would deploy to production"
  # show what would happen without executing
else
  # actual deploy
fi
```

## Rollback patterns

### Instant rollback (container-based)

```bash
# Roll back to previous image tag
docker service update --image ghcr.io/org/app:previous-sha service-name
```

### Terraform rollback

```bash
# Revert to previous state
git revert HEAD
git push  # triggers CI/CD pipeline with reverted config
```

### Feature flag rollback

For feature-flag-gated deployments, disable the flag instead of redeploying.
This is the fastest rollback path when available.
