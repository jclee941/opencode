# 1Password Secrets Naming Specification

Canonical naming rules for 1Password vault items, sections, fields, types,
tags, and templates. Location-independent — referenceable from any local project.

Priority: Tier 1 on-demand. Read when: creating or auditing 1Password items,
defining `op://` references, or structuring new service credentials.

## Scope

This spec applies to all 1Password items in the `homelab` vault and any future
vaults. It defines the schema for item titles, section names, field names,
field types, tags, and template selection. All projects referencing `op://` URIs
follow this spec.

## Inputs/constraints

- 1Password `op://` URI format: `op://vault/item/[section/]field`.
- Spaces in section/field names require quoting in CLI: `op read "op://homelab/slack/MCP Tokens/bot_token"`.
- Periods, equals signs, and backslashes in names require escaping with `\`.
- Names with unsupported characters must use UUIDs instead of labels.
- Field types are enforced by 1Password's schema: `CONCEALED`, `STRING`, `URL`, `EMAIL`, `OTP`, `DATE`.

## Decision/rules

### Item title naming

| Rule | Convention | Example |
|------|-----------|---------|
| Case | **lowercase** | `github`, `supabase`, `elk` |
| Format | Service name only, no prefixes | `cloudflare` not `cf-api` |
| Multi-word | Hyphenated kebab-case | `open-webui`, `auth-bridge` |
| Environment scoping | Suffix with env when needed | `supabase-staging`, `elk-prod` |
| Per-access-level split | Suffix with access scope | `github-readonly`, `github-admin` |

### Section naming

Sections use **Title Case**. The canonical section names are:

| Section Name | Purpose | Typical Fields |
|-------------|---------|----------------|
| *(no section)* | Primary credential (top-level) | `credential`, `password`, `username` |
| `Connection` | URLs and endpoints | `url`, `dashboard_url`, `rest_url`, `*_url` |
| `Credentials` | Composite auth credentials | `api_token_value`, `endpoint`, `admin_password` |
| `API Keys` | Third-party API keys | `api_key`, `personal_access_token` |
| `Keys` | Cryptographic / JWT keys | `service_key`, `anon_key`, `jwt_secret`, `private_key` |
| `Passwords` | Service passwords | `elastic_password`, `kibana_password` |
| `Dashboard` | Web UI credentials | `username`, `password` |
| `Database` | Database-specific credentials | `db_password`, `connection_string` |
| `Account` | Account-level identifiers | `email`, `account_id`, `zone_id` |
| `CF Access` | Cloudflare Access credentials | `client_id`, `client_secret`, `token_id` |
| `<App> Tokens` | App-specific token groups | `bot_token`, `xoxp_token`, `app_token` |

Rules for section naming:

1. Always Title Case — `Passwords` not `passwords` or `PASSWORDS`.
2. Use the canonical name from the table above when the purpose matches.
3. New sections follow `<Noun>` or `<Qualifier> <Noun>` pattern.
4. App-specific token sections use `<AppName> Tokens` pattern (e.g., `MCP Tokens`, `OpenCode Tokens`).
5. Never create a section named `secrets` (lowercase) — use `Passwords`, `Keys`, or `Credentials` instead.
6. Avoid single-field sections — merge into the closest canonical section.

### Field naming

Fields use **snake_case**. Grammar patterns:

| Pattern | Usage | Examples |
|---------|-------|----------|
| `<service>_password` | Service-specific passwords | `elastic_password`, `kibana_password`, `admin_password` |
| `<purpose>_key` | Keys (crypto, API, service) | `service_key`, `anon_key`, `api_key` |
| `<type>_token` | Tokens (auth, bot, app) | `bot_token`, `xoxp_token`, `app_token`, `refresh_token` |
| `<thing>_id` | Identifiers | `account_id`, `zone_id`, `chat_id`, `client_id` |
| `<qualifier>_url` | URLs and endpoints | `dashboard_url`, `rest_url`, `elasticsearch_url` |
| `<thing>_secret` | Shared secrets | `jwt_secret`, `client_secret` |
| `<bare_noun>` | Simple primary fields | `credential`, `password`, `username`, `email`, `endpoint`, `url` |

Rules for field naming:

1. Always snake_case — `api_token_value` not `apiTokenValue` or `API_TOKEN_VALUE`.
2. Prefer descriptive suffixes over generic names: `elastic_password` not `password` (within a multi-field section).
3. Top-level `credential` field holds the single primary secret when the item has one main key.
4. URL fields end in `_url` (except bare `url` for the primary endpoint).
5. Never prefix fields with the service name when the item title already identifies the service.
6. Boolean-like fields are not used — 1P is for secrets, not configuration.

### Field type selection

| Type | 1P CLI | Use When |
|------|--------|----------|
| `CONCEALED` | `password` | Passwords, tokens, API keys, JWT secrets, private keys — anything that must not be displayed |
| `STRING` | `text` | Non-secret identifiers: account IDs, zone IDs, usernames, chat IDs |
| `URL` | `url` | Endpoints and dashboard URLs — enables "open in browser" in 1P UI |
| `EMAIL` | `email` | Email addresses used for authentication |
| `OTP` | `otp` | TOTP seeds (accepts `otpauth://` URIs) |
| `DATE` | `date` | Expiration dates, rotation timestamps (`YYYY-MM-DD`) |

Rules:

1. Default to `CONCEALED` for any value that grants access.
2. Use `STRING` only for values that are safe to display (IDs, usernames, non-secret config).
3. Use `URL` for all endpoint fields — improves 1P UI usability.
4. Never use `STRING` for passwords, tokens, or keys.

### Template (category) selection

| Template | Use When | Default Fields |
|----------|----------|---------------|
| `API_CREDENTIAL` | Token/key-based API access (most services) | `username`, `credential`, `type`, `filename` |
| `PASSWORD` | Multi-credential services with username+password | `username`, `password` |
| `LOGIN` | Web login with URL | `username`, `password`, `url` |
| `DATABASE` | Database connections | `hostname`, `port`, `database`, `username`, `password` |
| `SERVER` | Infrastructure nodes (Proxmox, Synology) | `URL`, `username`, `password`, `admin_console` |
| `SSH_KEY` | SSH key pairs | `private_key`, `public_key`, `passphrase`, `fingerprint` |
| `SECURE_NOTE` | Documentation, notes, non-credential data | `notesPlain` |

Selection decision tree:

1. Service accessed via API key/token only → `API_CREDENTIAL`.
2. Service has username + password login → `PASSWORD` or `LOGIN` (use `LOGIN` if URL matters).
3. Database with host/port/db/user/pass → `DATABASE`.
4. Infrastructure node with SSH/console → `SERVER`.
5. SSH key pair → `SSH_KEY`.
6. None of the above → `SECURE_NOTE` with custom sections.

### Tag taxonomy

Tags use **lowercase**, single-word or hyphenated:

| Tag | Applied To |
|-----|-----------|
| `homelab` | All items in the homelab vault |
| `infra` | Infrastructure services (Proxmox, Synology, Traefik) |
| `database` | Database services (Supabase, PostgreSQL) |
| `llm` | LLM/AI providers (OpenRouter, Exa) |
| `mcp` | MCP server integrations (MCPhub, Slack MCP) |
| `logging` | Logging and monitoring (ELK, Grafana) |
| `network` | Network and DNS (Cloudflare, Traefik) |
| `notification` | Notification services (Telegram, Slack) |
| `search` | Search services (Exa, Elasticsearch) |
| `ai` | AI/ML services (OpenRouter, Exa) |
| `ci` | CI/CD related credentials (GitHub Actions) |
| `automation` | Automation services (n8n, Archon) |

Rules:

1. Every item gets the `homelab` tag plus at least one domain tag.
2. Items can have multiple domain tags (e.g., Slack: `notification`, `mcp`).
3. Never use tags for environment scoping — use item title suffixes instead.

### op:// URI construction rules

Given the naming conventions above, URIs follow this pattern:

```
op://homelab/<service-name>/<field>                    # top-level field
op://homelab/<service-name>/<Section Name>/<field>     # sectioned field
```

Examples:

```
op://homelab/github/credential                         # top-level credential
op://homelab/supabase/Keys/service_key                 # sectioned field
op://homelab/slack/MCP Tokens/xoxp_token               # space in section (quote in CLI)
op://homelab/cloudflare/API Keys/api_key               # space in section
op://homelab/elk/Passwords/elastic_password            # standard section
```

Rules:

1. Vault name is always `homelab` (or the specific vault for non-homelab projects).
2. Item name matches the item title (lowercase service name).
3. Section name matches the section label exactly (Title Case, spaces preserved).
4. Field name matches the field label exactly (snake_case).
5. Omit section for top-level fields (e.g., `credential`, `password`).
6. Quote the entire URI in CLI when sections contain spaces.

## Item structure archetypes

See `02-onepassword-secrets-naming-examples.md` for complete examples:
- API-key service (e.g., OpenRouter, Exa)
- Multi-credential service (e.g., Supabase)
- Infrastructure node (e.g., Proxmox)
- Logging/monitoring (e.g., ELK)
- Messaging/notification (e.g., Slack)

## Migration and verification

See `03-onepassword-secrets-naming-operations.md` for:
- Known inconsistencies and migration
- Verification checklist
- Rollback/safety procedures
