# PROJECT KNOWLEDGE BASE

OpenCode configuration repository. AI agent instruction files, LSP/formatter configs,
MCP server integration, and plugin management for all `qws941` projects.

## STRUCTURE

```text
./
├── config/
│   ├── base.jsonc              # Core config (instructions, plugins, MCP, watcher)
│   ├── providers.jsonc         # LLM provider definitions
│   └── lsp.jsonc               # Language server configurations
├── rules/
│   ├── README.md               # Tier model, priority order, conflict resolution
│   ├── hard-autonomy-no-questions.md   # Tier 0: zero-question execution policy
│   ├── archon-workflow.md              # Tier 0: Archon task management
│   ├── session-init.md                 # Tier 0: session bootstrap + MCP schema hygiene
│   ├── requirements-verification.md    # Tier 0: requirements check gate
│   ├── monorepo-standards.md           # Tier 0: structure and naming
│   ├── deployment-automation.md        # Tier 1: CI/CD policy (on-demand)
│   ├── code-modularization.md          # Tier 1: file size governance (on-demand)
│   ├── auto-build-pipeline.md          # Tier 1: autonomous pipeline (on-demand)
│   ├── mcp-schema-hygiene.md          # Tier 1: MCP schema validation (on-demand)
│   ├── onepassword-integration.md      # Tier 1: 1Password integration (on-demand)
│   ├── onepassword-secrets-naming.md   # Tier 1: 1Password naming spec (on-demand)
│   ├── msa-refactoring.md              # Tier 1: MSA refactoring guidance (on-demand)
│   └── elk-troubleshooting.md          # Tier 2: ELK troubleshooting (on-demand)
├── docs/                       # Reference documents (never auto-loaded)
├── skills/                     # OpenCode skills (auto-discovered)
├── scripts/
│   ├── gen-opencode-config.go          # Merges config/*.jsonc → opencode.jsonc
│   ├── kratos-project-sync.go          # Kratos project sync/install/uninstall
│   ├── validate-monorepo-naming.mjs    # Naming convention validator
│   ├── lint-assistant-phrasing.mjs     # Assistant phrasing linter
│   ├── claude-hook-autonomy-guard.mjs  # Autonomy guard git hook
│   ├── validate-config-refs.go         # Config cross-reference validator
│   └── omo-auto-update.go              # OMO auto-update cron (git pull + npm install)
│   └── omo-auto-update.go              # OMO auto-update cron (git pull + npm install)
├── .githooks/                  # Git hook scripts (commitlint, pre-push)
├── .github/                    # GitHub Actions workflows
├── tests/                      # Vitest test suites (ELK integration)
├── plugin/                     # Plugin development (subtask2)
├── command/                    # Slash command definitions (plannotator)
├── pilot/                      # Pilot plugin config + templates
├── snippet/                    # User snippet config
├── thoughts/                   # Thought ledgers
├── *.jsonc                     # Plugin configs (auto-managed by plugins)
├── package.json                # Dependencies and npm scripts
└── opencode.jsonc              # AUTO-GENERATED — edit config/*.jsonc instead
```

## WHERE TO LOOK

| Task                        | Location                                       |
| --------------------------- | ---------------------------------------------- |
| Add/modify loaded rules     | `config/base.jsonc` → `instructions` array     |
| Rule priority and tiers     | `rules/README.md`                              |
| LLM providers and models    | `config/providers.jsonc`                        |
| LSP server configs          | `config/lsp.jsonc`                              |
| Regenerate opencode.jsonc   | `go run scripts/gen-opencode-config.go`         |
| Plugin configurations       | Root `*.jsonc` files (dcp, etc.)               |
| GitHub Actions workflows    | `.github/workflows/`                           |
| Naming convention validator | `scripts/validate-monorepo-naming.mjs`         |
| ELK integration tests       | `tests/`                                       |
| Slash commands              | `command/`                                     |
| Config cross-ref validator  | `scripts/validate-config-refs.go`              |
| OMO auto-update (cron)    | `scripts/omo-auto-update.go`                       |

## COMMANDS

```bash
npm test                        # vitest run
npm run gen:config              # generate opencode.jsonc from config/*.jsonc
npm run gen:config:check        # verify config freshness (CI gate)
npm run lint:naming             # validate monorepo naming conventions
npm run lint:assistant-phrasing # lint denylist phrases in rule files
npm run hooks:install           # install git hooks from .githooks/
npm run prepush:check           # pre-push gate (config + naming + config-refs)
npm run lint:config-refs        # validate config cross-references
npm run kratos:sync             # sync Kratos project memory
npm run omo:update               # pull latest + npm install (manual)
npm run omo:update:dry          # preview what omo:update would do
npm run lint:modularization     # validate LOC/complexity thresholds
```

## CONVENTIONS

- `opencode.jsonc` is auto-generated. Edit `config/*.jsonc` sources instead.
- Tier 0 rules load every session (~1,500 tokens). Tier 1+ load on-demand.
- `hard-autonomy-no-questions.md` overrides all other rules on conflict.
- Each rule file has single responsibility. No duplicate normative text.
- kebab-case for filenames. Conventional commits for all changes.
- Operational scripts must be Go (`.go`). Node (`.mjs`) only for hooks/linters/validators.
- Formatters: oxfmt (TS/JS/CSS/JSON/YAML/MD), ruff (Python), gofmt (Go).

## ANTI-PATTERNS

- Never edit `opencode.jsonc` directly — changes are overwritten by generator.
- Never add Tier 1/2 rules to the instructions array — they waste context tokens.
- Never suppress type errors (`as any`, `@ts-ignore`).
- Never duplicate normative text across rule files.
- Never batch MCP tool calls (`mcphub_*`) — call each directly.
## CONFIGURATION WORKFLOW

opencode.jsonc is **AUTO-GENERATED** from source files in `config/`. Direct edits will be **LOST**.

### Source Files (Edit These)

| File | Purpose |
|------|---------|
| `config/base.jsonc` | Core config: instructions, plugins, MCP, permissions, formatters, watcher |
| `config/providers.jsonc` | LLM provider and model definitions |
| `config/lsp.jsonc` | Language server configurations |

### Workflow

1. **Edit source files** in `config/*.jsonc`
2. **Regenerate** with: `npm run gen:config`
3. **Stage changes**: `git add config/ opencode.jsonc`
4. **Commit**: `git commit -m "feat: update config"`

### Recovery (If You Edited opencode.jsonc Directly)

If you accidentally edited `opencode.jsonc` directly:

```bash
# Check what changes you made
go run scripts/config-recover.go

# Migrate changes to source files (manual review required)
go run scripts/config-recover.go --apply

# Or discard changes and regenerate
go run scripts/config-recover.go --reset
```

### Protection

- **Pre-commit hook**: Blocks commits with direct edits to `opencode.jsonc`
- **Pre-push hook**: Validates config is up-to-date before pushing
- **Git attributes**: Marks `opencode.jsonc` as generated (suppresses diff noise)

Install hooks: `npm run hooks:install`
