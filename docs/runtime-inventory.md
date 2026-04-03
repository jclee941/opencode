<!-- AUTO-GENERATED - do not edit directly.
Source of truth: config/base.jsonc, package.json, config/lsp.jsonc, scripts/omo-conflicts.json, and explicit local surface discovery in scripts/gen-runtime-inventory.go
Regenerate: npm run gen:runtime-inventory
-->
# Runtime Inventory

This report stays intentionally narrow: it shows load-time plugin and MCP visibility from `config/base.jsonc`, the OMO conflict watchlist from `scripts/omo-conflicts.json`, and explicit local capability surfaces that commonly change decision-making in this repo. It does not attempt to expand every nested setting inside files like `oh-my-opencode.jsonc`.

## Scope

- Show active plugins requested in `config/base.jsonc`.
- Show configured MCP servers from `config/base.jsonc`.
- Show explicit local capability surfaces that affect operator decisions in this repo.
- Show the OMO conflict watchlist derived from `scripts/omo-conflicts.json`.

## Inputs/constraints

- Source-of-truth plugin and MCP declarations come from `config/base.jsonc`.
- Package install versions come from `package.json`.
- Schema coverage comes from `config/lsp.jsonc`.
- OMO conflict truth comes from `scripts/omo-conflicts.json`.
- This report is static visibility only; it does not perform live MCP health checks.

## Decision/rules

- Keep the report deterministic and generated.
- Do not duplicate OMO conflict data outside the generator source.
- Prefer explicit file-path visibility over interpreting every nested runtime setting.

## Verification

- Regenerate with `npm run gen:runtime-inventory`.
- Check freshness with `npm run gen:runtime-inventory:check`.
- `npm run prepush:check` also enforces this report stays current.

## Rollback/safety

- This artifact is additive and does not change runtime plugin or MCP behavior.
- Remove the generator, generated doc, and npm script wiring together to revert this feature.

- Active plugins: 5
- Configured MCP servers: 1
- Capability surfaces tracked: 11
- OMO conflict watchlist entries: 18

## Active Plugins

| Order | Plugin | Requested entry | Dependency version | Config surface | Notes |
|---|---|---|---|---|---|
| 1 | `opencode-pty` | `opencode-pty@latest` | `-` | `-` | Interactive PTY support for long-running shell sessions. |
| 2 | `@nick-vi/opencode-type-inject` | `@nick-vi/opencode-type-inject@latest` | `-` | `-` | Type/tool augmentation plugin loaded from npm dependency entry. |
| 3 | `opencode-pilot` | `opencode-pilot@latest` | `-` | `pilot/config.yaml` | Active queue/polling plugin with pilot/config.yaml. |
| 4 | `@tarquinen/opencode-dcp` | `@tarquinen/opencode-dcp@latest` | `-` | `dcp.jsonc` | Active dynamic context pruning plugin with dedicated root config. |
| 5 | `oh-my-opencode` | `oh-my-opencode@latest` | `-` | `oh-my-opencode.jsonc` | Must remain last in config/base.jsonc; activates OMO conflict filtering. |

## Configured MCP Servers

| Name | Type | Enabled | Target |
|---|---|---|---|
| `mcphub` | `remote` | `true` | `http://192.168.50.112:3000/mcp` |

## Capability Surfaces

| Path | Kind | Status | Backing | Schema | Notes |
|---|---|---|---|---|---|
| `dcp.jsonc` | `plugin-config` | `active` | `@tarquinen/opencode-dcp` | `https://raw.githubusercontent.com/Opencode-DCP/opencode-dynamic-context-pruning/master/dcp.schema.json` | Dynamic context pruning policy and protected tool settings. |
| `oh-my-opencode.jsonc` | `plugin-config` | `active` | `oh-my-opencode` | `https://raw.githubusercontent.com/code-yeongyu/oh-my-openagent/master/assets/oh-my-opencode.schema.json` | Primary orchestration, routing, fallback, browser, and search settings. |
| `pilot/config.yaml` | `plugin-config` | `active` | `opencode-pilot` | `-` | GitHub work polling and session defaults. |
| `skills/agent-browser/SKILL.md` | `skill` | `available` | `agent-browser` | `-` | Policy or capability skill surface. |
| `skills/debugging-expert/SKILL.md` | `skill` | `available` | `debugging-expert` | `-` | Policy or capability skill surface. |
| `skills/dev-browser/SKILL.md` | `skill` | `available` | `dev-browser` | `-` | Standalone executable browser automation skill package. |
| `skills/frontend-ui-ux/SKILL.md` | `skill` | `available` | `frontend-ui-ux` | `-` | Policy or capability skill surface. |
| `skills/git-master/SKILL.md` | `skill` | `available` | `git-master` | `-` | Policy or capability skill surface. |
| `smart-title.jsonc` | `plugin-config` | `present, blocked with OMO` | `opencode-smart-title` | `-` | Config file exists locally even though the plugin is not active. |
| `snippet/config.jsonc` | `supporting-config` | `supporting` | `-` | `https://raw.githubusercontent.com/JosXa/opencode-snippets/main/schema/config.schema.json` | Snippet output and skill rendering settings. |
| `subtask2.jsonc` | `plugin-config` | `present, blocked with OMO` | `@openspoon/subtask2` | `-` | Config file exists locally even though the plugin is not active. |

## OMO Conflict Watchlist

These entries are auto-removed when `oh-my-opencode` is present, based on the shared conflict list in `scripts/omo-conflicts.json` and the resolver in `scripts/gen-opencode-config.go`.

- `@openspoon/subtask2`
- `@plannotator/opencode`
- `micode`
- `octto`
- `opencode-convodump`
- `opencode-daytona`
- `opencode-devcontainers`
- `opencode-helicone-session`
- `opencode-morph-fast-apply`
- `opencode-notify`
- `opencode-openai-codex-auth`
- `opencode-scheduler`
- `opencode-smart-title`
- `opencode-supermemory`
- `opencode-wakatime`
- `opencode-websearch-cited`
- `opencode-worktree`
- `opencode-zellij-namer`

