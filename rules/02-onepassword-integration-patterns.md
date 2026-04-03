# 1Password Integration — Implementation Patterns

Implementation patterns for 1Password secret integration.
Part of: `01-onepassword-integration.md` (split for modularity)

## Go script pattern — `resolveValue`

All Go scripts that consume secrets use the established `resolveValue` pattern.
This is the canonical implementation (from `scripts/opencode-auth-bridge/main.go`):

```go
func resolveValue(value string, fallbackRef string) (string, error) {
    candidate := strings.TrimSpace(value)
    if candidate == "" {
        candidate = strings.TrimSpace(fallbackRef)
    }
    if candidate == "" {
        return "", nil
    }
    if !strings.HasPrefix(candidate, "op://") {
        return candidate, nil
    }
    cmd := exec.Command("op", "read", candidate)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("resolve %s: %s", candidate, strings.TrimSpace(string(output)))
    }
    return strings.TrimSpace(string(output)), nil
}

func envOrDefault(key string, fallback string) string {
    value := strings.TrimSpace(os.Getenv(key))
    if value != "" {
        return value
    }
    return fallback
}
```

**Resolution order**: env var → CLI flag → `op://` fallback reference.

When writing new Go scripts that consume secrets:
1. Copy `resolveValue` and `envOrDefault` from the auth-bridge.
2. Define `op://` default constants at file scope.
3. Accept both plain values and `op://` references in CLI flags and env vars.
4. Log resolution source (env/flag/1P) at debug level, never log the resolved value.

## `.env.tpl` template pattern

For environments that use `.env` files, maintain a `.env.tpl` template with `op://` references.
The template is tracked in git; the resolved `.env` is gitignored.

```env
# .env.tpl — tracked in git, contains op:// references only
ES_URL=http://192.168.50.105:9200
ES_USER=elastic
ES_PASSWORD=op://homelab/elk/Passwords/elastic_password
TELEGRAM_BOT_TOKEN=op://homelab/telegram/credential
```

Generate the live `.env` from the template:

```bash
op inject -i .env.tpl -o .env
```

Rules for `.env.tpl` files:
1. Non-secret values (URLs, usernames) can remain as literals.
2. All passwords, tokens, and keys use `op://` references.
3. Comment each `op://` reference with the service and purpose if not obvious.
4. `.env` and `.env.*` must be in `.gitignore` (already enforced in this repo).

## MCP server configuration

For MCP servers that require credentials (mcphub, slack, elk, etc.):

**Option A — `op run` wrapper** (preferred for simple env injection):
```bash
op run -- docker-compose up -d
```

**Option B — Template injection** for JSON/JSONC config:
```jsonc
// mcphub.json.tpl
{
  "servers": {
    "slack": {
      "env": {
        "SLACK_BOT_TOKEN": "op://homelab/slack/OpenCode Tokens/bot_token"
      }
    }
  }
}
```
Apply: `op inject -i mcphub.json.tpl -o mcphub.json`

**Option C — Go resolveValue** (for Go-based MCP tooling):
Use the `resolveValue` pattern documented above.

## CI/CD — GitHub Actions

Use the `1password/load-secrets-action@v2` with a Service Account:

```yaml
jobs:
  build:
    steps:
      - name: Load secrets from 1Password
        uses: 1password/load-secrets-action@v2
        with:
          export-env: true
        env:
          OP_SERVICE_ACCOUNT_TOKEN: ${{ secrets.OP_SERVICE_ACCOUNT_TOKEN }}
          # Map op:// references to env vars
          GITHUB_TOKEN: op://homelab/github/credential
          ES_PASSWORD: op://homelab/elk/Passwords/elastic_password
          SUPABASE_SERVICE_KEY: op://homelab/supabase/Keys/service_key
```

CI/CD rules:
1. Store `OP_SERVICE_ACCOUNT_TOKEN` as a GitHub repository secret.
2. Grant the Service Account access only to the `homelab` vault.
3. Never use personal `op` sessions in CI — only Service Accounts.
4. Use `export-env: true` to inject secrets as environment variables.

## Docker / container integration

Never bake secrets into container images. Use one of:

1. **`op run` wrapper** — injects env vars into the subprocess:
   ```bash
   op run -- docker-compose up -d
   ```

2. **Template injection before launch** — generates `.env` at deploy time:
   ```bash
   op inject -i .env.tpl -o .env
   docker-compose --env-file .env up -d
   ```

3. **Go entrypoint with resolveValue** — for Go services, resolve at startup.
