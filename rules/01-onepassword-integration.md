# 1Password Integration

All variables, secrets, and credentials are managed through 1Password.
This rule defines the canonical patterns for secret resolution, template
injection, vault organization, and CI/CD integration.

Priority: Tier 1 on-demand. Read when: secrets, credentials, `.env`,
`op://` references, or 1Password integration is in scope.

## Scope

This rule applies to all secret and credential management across the repository:
Go scripts, `.env` files, MCP server configuration, CI/CD pipelines, and
Docker/container environments. It covers the `homelab` vault and any future
vaults used by this project.

## Inputs/constraints

- All secrets live in the `homelab` 1Password vault, organized by service item and section.
- The 1Password CLI (`op`) must be available on the host for `op read`, `op run`, and `op inject`.
- CI/CD uses a 1Password Service Account (`OP_SERVICE_ACCOUNT_TOKEN`) — never a personal session.
- The `op://` URI format is: `op://<vault>/<item>/[<section>/]<field>`.
- Literal secrets must never appear in tracked files; `.env` and `.env.*` are gitignored.

## Decision/rules

### Secret reference format

Use the `op://` URI scheme for all secret references:

```
op://<vault>/<item>/<field>                    # field without section
op://<vault>/<item>/<section>/<field>           # field within a named section
```

Examples from the homelab vault:

```
op://homelab/github/credential                           # GitHub PAT (top-level)
op://homelab/supabase/Keys/service_key                   # Supabase JWT (section: Keys)
op://homelab/slack/MCP Tokens/xoxp_token                 # Slack XOXP (section: MCP Tokens)
op://homelab/elk/Passwords/elastic_password              # ELK password (section: Passwords)
op://homelab/proxmox/Credentials/api_token_value         # Proxmox API token (section: Credentials)
op://homelab/cloudflare/API Keys/api_key                 # Cloudflare API key (section: API Keys)
```

### Vault organization — homelab item/section conventions

Items are organized by service name. Sections within each item follow these conventions:

| Section Name     | Content                                           | Example Fields                      |
|------------------|---------------------------------------------------|-------------------------------------|
| `credential`     | Primary credential (top-level field, no section)  | API key, PAT, bot token             |
| `Credentials`    | Login credentials                                 | `api_token_value`, `endpoint`       |
| `Keys`           | Cryptographic keys and JWT secrets                | `service_key`, `anon_key`, `jwt_secret` |
| `Passwords`      | Service passwords                                 | `elastic_password`, `kibana_password` |
| `API Keys`       | Third-party API keys                              | `api_key`, `personal_access_token`  |
| `Connection`     | URLs and endpoints                                | `url`, `dashboard_url`, `rest_url`  |
| `Dashboard`      | Web UI credentials                                | `username`, `password`              |
| `Database`       | Database-specific credentials                     | `db_password`                       |
| `Account`        | Account-level identifiers                         | `email`, `account_id`, `zone_id`    |
| `MCP Tokens`     | MCP-specific tokens                               | `xoxp_token`, `xapp_token`          |
| `OpenCode Tokens`| OpenCode-specific tokens                          | `app_token`, `bot_token`            |
| `CF Access`      | Cloudflare Access credentials                     | `client_id`, `client_secret`        |

When creating new 1Password items, follow these conventions:
1. Item name = service name in lowercase (e.g., `github`, `supabase`, `elk`).
2. Use the most specific 1P template: `API_CREDENTIAL` for API keys, `PASSWORD` for multi-credential services, `DATABASE` for DB connections, `SERVER` for infrastructure, `SSH_KEY` for SSH keys.
3. Group related fields into named sections matching the table above.
4. Tag items by domain: `homelab`, `infra`, `database`, `llm`, `mcp`, `logging`, `network`, `notification`, `search`, `ai`.
5. Put the primary authentication credential in the top-level `credential` field when possible.

## Implementation patterns

See `02-onepassword-integration-patterns.md` for:
- Go script `resolveValue` pattern
- `.env.tpl` template pattern
- MCP server configuration
- CI/CD GitHub Actions integration
- Docker/container integration

## Secret reference

See `03-onepassword-integration-reference.md` for:
- Complete homelab service secret reference map
- Secret rotation procedures
- Verification checklist
