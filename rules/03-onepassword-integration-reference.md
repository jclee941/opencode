# 1Password Integration — Secret Reference

Complete secret reference and operational procedures for 1Password integration.
Part of: `01-onepassword-integration.md` (split for modularity)

## Homelab service secret reference map

Complete `op://` reference map for services used in this repository:

| Service     | Env Var / Config Key         | op:// Reference                                        |
|-------------|------------------------------|--------------------------------------------------------|
| GitHub      | `GITHUB_TOKEN`               | `op://homelab/github/credential`                       |
| GitHub      | `GITHUB_PAT`                 | `op://homelab/github/API Keys/personal_access_token`   |
| Proxmox     | `PROXMOX_TOKEN_VALUE`        | `op://homelab/proxmox/Credentials/api_token_value`     |
| Proxmox     | `PROXMOX_ENDPOINT`           | `op://homelab/proxmox/Credentials/endpoint`            |
| Proxmox     | `PROXMOX_SSH_KEY`            | `op://homelab/proxmox/Keys/private_key`                |
| Supabase    | `SUPABASE_SERVICE_KEY`       | `op://homelab/supabase/Keys/service_key`               |
| Supabase    | `SUPABASE_ANON_KEY`          | `op://homelab/supabase/Keys/anon_key`                  |
| Supabase    | `SUPABASE_JWT_SECRET`        | `op://homelab/supabase/Keys/jwt_secret`                |
| Supabase    | `SUPABASE_DB_PASSWORD`       | `op://homelab/supabase/Database/db_password`            |
| Supabase    | `SUPABASE_URL`               | `op://homelab/supabase/Connection/url`                 |
| OpenRouter  | `OPENROUTER_API_KEY`         | `op://homelab/openrouter/credential`                   |
| Slack       | `SLACK_XOXP_TOKEN`           | `op://homelab/slack/MCP Tokens/xoxp_token`             |
| Slack       | `SLACK_BOT_TOKEN`            | `op://homelab/slack/OpenCode Tokens/bot_token`         |
| Slack       | `SLACK_APP_TOKEN`            | `op://homelab/slack/OpenCode Tokens/app_token`         |
| ELK         | `ES_PASSWORD`                | `op://homelab/elk/Passwords/elastic_password`           |
| ELK         | `KIBANA_PASSWORD`            | `op://homelab/elk/Passwords/kibana_password`           |
| ELK         | `ES_URL`                     | `op://homelab/elk/Connection/elasticsearch_url`        |
| MCPhub      | `MCPHUB_ADMIN_PASSWORD`      | `op://homelab/mcphub/Credentials/admin_password`       |
| MCPhub      | `OP_SERVICE_ACCOUNT_TOKEN`   | `op://homelab/mcphub/Credentials/op_service_account_token` |
| Archon      | `ARCHON_URL`                 | `op://homelab/archon/Connection/url`                   |
| Cloudflare  | `CF_API_KEY`                 | `op://homelab/cloudflare/API Keys/api_key`             |
| Cloudflare  | `CF_ACCOUNT_ID`              | `op://homelab/cloudflare/Account/account_id`           |
| Cloudflare  | `CF_ZONE_ID`                 | `op://homelab/cloudflare/Account/zone_id`              |
| Telegram    | `TELEGRAM_BOT_TOKEN`         | `op://homelab/telegram/credential`                     |
| Telegram    | `TELEGRAM_CHAT_ID`           | `op://homelab/telegram/chat_id`                        |
| Telegram    | `TELEGRAM_API_ID`            | `op://homelab/telegram/App/api_id`                     |
| Telegram    | `TELEGRAM_API_HASH`          | `op://homelab/telegram/App/api_hash`                   |
| Exa         | `EXA_API_KEY`                | `op://homelab/exa/API Keys/api_key`                    |

## Secret rotation

1Password does not rotate external API keys natively. Use this pattern:

1. Generate new key via service API or dashboard.
2. Add new key to 1Password alongside old key (e.g., `api_key_v2`).
3. Update all `op://` references to point to new key.
4. Deploy/restart services to pick up new key.
5. Verify services work with new key.
6. Remove old key from service dashboard.
7. Remove old key from 1Password after 7-day grace period.

## Verification

| Check | Method |
|-------|--------|
| `op` CLI available | `which op` returns path |
| Vault accessible | `op vault list` shows `homelab` |
| Item exists | `op item get <item>` succeeds |
| Reference valid | `op read op://homelab/github/credential` returns value |
| Template injection works | `op inject -i .env.tpl -o .env` produces valid `.env` |
| CI/CD secret loads | GitHub Action step succeeds with `load-secrets-action` |

## Rollback/safety

- If `op read` fails, check: CLI auth (`op signin`), vault access, item/field path spelling.
- If CI/CD fails, check: `OP_SERVICE_ACCOUNT_TOKEN` secret configured, service account has vault access.
- If injection produces empty values, check: template syntax (no extra spaces in `op://` path), item exists, field exists.
- Emergency: Plain values can be used temporarily if 1Password is unavailable — commit only to local working copy, never push.

## Reference composition

1. Loaded as Tier 1 on-demand rule.
2. Defers to `00-hard-autonomy-no-questions.md` on execution posture.
3. Defers to `01-onepassword-secrets-naming.md` for field naming conventions.
4. Naming conventions are enforced by `scripts/validate-monorepo-naming.mjs` for code and by this rule for 1Password items.
