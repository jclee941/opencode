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

- Active plugins: 2
- Configured MCP servers: 1
- Capability surfaces tracked: 10
- OMO conflict watchlist entries: 18

## Active Plugins

| Order | Plugin | Requested entry | Dependency version | Config surface | Notes |
|---|---|---|---|---|---|
| 1 | `opencode-claude-auth` | `opencode-claude-auth` | `latest` | `-` | - |
| 2 | `oh-my-openagent` | `oh-my-openagent` | `latest` | `-` | - |

## Configured MCP Servers

| Name | Type | Enabled | Target |
|---|---|---|---|
| `mcphub` | `remote` | `true` | `http://192.168.50.112:3000/mcp` |

## Capability Surfaces

| Path | Kind | Status | Backing | Schema | Notes |
|---|---|---|---|---|---|
| `oh-my-opencode.jsonc` | `plugin-config` | `present, inactive` | `oh-my-opencode` | `-` | Primary orchestration, routing, fallback, browser, and search settings. |
| `pilot/config.yaml` | `plugin-config` | `present, inactive` | `opencode-pilot` | `-` | GitHub work polling and session defaults. |
| `skills/agent-browser/SKILL.md` | `skill` | `available` | `agent-browser` | `-` | Policy or capability skill surface. |
| `skills/debugging-expert/SKILL.md` | `skill` | `available` | `debugging-expert` | `-` | Policy or capability skill surface. |
| `skills/dev-browser/SKILL.md` | `skill` | `available` | `dev-browser` | `-` | Standalone executable browser automation skill package. |
| `skills/frontend-ui-ux/SKILL.md` | `skill` | `available` | `frontend-ui-ux` | `-` | Policy or capability skill surface. |
| `skills/git-master/SKILL.md` | `skill` | `available` | `git-master` | `-` | Policy or capability skill surface. |
| `smart-title.jsonc` | `plugin-config` | `present, inactive` | `opencode-smart-title` | `-` | Config file exists locally even though the plugin is not active. |
| `snippet/config.jsonc` | `supporting-config` | `supporting` | `-` | `-` | Snippet output and skill rendering settings. |
| `subtask2.jsonc` | `plugin-config` | `present, inactive` | `@openspoon/subtask2` | `-` | Config file exists locally even though the plugin is not active. |

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

