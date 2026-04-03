# 1Password Secrets Naming — Examples

Complete item structure archetypes for 1Password secrets naming.
Part of: `01-onepassword-secrets-naming.md` (split for modularity)

## API-key service (e.g., OpenRouter, Exa)

```
Title: openrouter
Category: API_CREDENTIAL
Tags: homelab, llm, ai

Fields:
  credential [CONCEALED]     → primary API key

Sections: (none needed — single credential)
```

## Multi-credential service (e.g., Supabase)

```
Title: supabase
Category: PASSWORD
Tags: homelab, database

Fields:
  credential [CONCEALED]     → primary credential (if applicable)

Section "Connection":
  url [URL]                  → API endpoint
  dashboard_url [URL]        → web dashboard
  rest_url [URL]             → REST API URL

Section "Keys":
  service_key [CONCEALED]    → service role key
  anon_key [CONCEALED]       → anon/public key
  jwt_secret [CONCEALED]     → JWT signing secret

Section "Database":
  db_password [CONCEALED]    → Postgres password
```

## Infrastructure node (e.g., Proxmox)

```
Title: proxmox
Category: SERVER
Tags: homelab, infra

Section "Credentials":
  api_token_value [CONCEALED] → API token
  endpoint [URL]              → API endpoint

Section "Keys":
  private_key [CONCEALED]     → SSH private key

Section "Dashboard":
  username [STRING]           → web UI username
  password [CONCEALED]        → web UI password
```

## Logging/monitoring (e.g., ELK)

```
Title: elk
Category: PASSWORD
Tags: homelab, logging, search

Section "Connection":
  elasticsearch_url [URL]    → ES endpoint
  kibana_url [URL]           → Kibana endpoint

Section "Passwords":
  elastic_password [CONCEALED] → elastic superuser
  kibana_password [CONCEALED]  → kibana system user
```

## Messaging/notification (e.g., Slack)

```
Title: slack
Category: API_CREDENTIAL
Tags: homelab, notification, mcp

Fields:
  credential [CONCEALED]     → primary bot token

Section "MCP Tokens":
  xoxp_token [CONCEALED]    → user-level token
  xapp_token [CONCEALED]    → app-level token

Section "OpenCode Tokens":
  bot_token [CONCEALED]     → OpenCode bot token
  app_token [CONCEALED]     → OpenCode app token
  refresh_token [CONCEALED] → OAuth refresh token
```
