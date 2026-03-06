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
│   ├── bmad-integration.md             # Tier 1: BMAD artifacts (on-demand)
│   ├── auto-build-pipeline.md          # Tier 1: autonomous pipeline (on-demand)
│   ├── mcp-schema-hygiene.md          # Tier 1: MCP schema validation (on-demand)
│   └── elk-troubleshooting.md          # Tier 2: ELK troubleshooting (on-demand)
├── docs/                       # Reference documents (never auto-loaded)
├── skills/                     # OpenCode skills (auto-discovered)
├── scripts/
│   ├── gen-opencode-config.go          # Merges config/*.jsonc → opencode.jsonc
│   ├── kratos-project-sync.go          # Kratos project sync/install/uninstall
│   ├── validate-monorepo-naming.mjs    # Naming convention validator
│   ├── lint-assistant-phrasing.mjs     # Assistant phrasing linter
│   ├── claude-hook-autonomy-guard.mjs  # Autonomy guard git hook
│   └── validate-config-refs.go         # Config cross-reference validator
│   ├── antigravity-telegram-bot.go     # Telegram bot for antigravity account management
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
| Plugin configurations       | Root `*.jsonc` files (antigravity, dcp, etc.)  |
| GitHub Actions workflows    | `.github/workflows/`                           |
| Naming convention validator | `scripts/validate-monorepo-naming.mjs`         |
| ELK integration tests       | `tests/`                                       |
| Slash commands              | `command/`                                     |
| Config cross-ref validator  | `scripts/validate-config-refs.go`              |
| Antigravity Telegram bot  | `scripts/antigravity-telegram-bot.go`              |

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
npm run telegram:bot             # run antigravity Telegram auth bot
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
