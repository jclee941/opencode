# PROJECT KNOWLEDGE BASE

**Generated:** 2026-02-20 13:08:00 Asia/Seoul
**Commit:** 0aa70d9
**Branch:** main

## OVERVIEW

OpenCode home workspace for agent runtime configuration and skill execution. The only executable code surface is `skills/dev-browser`; other skill folders are policy contracts.

## STRUCTURE

```text
./
├── skills/                 # mixed registry: executable + policy skills
├── rules/                  # session/global operating policies
├── scripts/                # repo validation utilities
├── pilot/                  # pilot service config/templates
├── docs/                   # monorepo standards docs
├── opencode.jsonc          # core runtime/provider/plugin config
└── oh-my-opencode.jsonc    # agent/category model mapping
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Runtime and provider tuning | `opencode.jsonc` | MCP endpoints, plugins, formatter, instructions |
| Agent/model routing | `oh-my-opencode.jsonc` | role-to-model and category mapping |
| Browser automation server | `skills/dev-browser/src/index.ts` | HTTP API + page registry lifecycle |
| Browser relay bridge | `skills/dev-browser/src/relay.ts` | extension/CDP relay mode |
| Snapshot extraction and tests | `skills/dev-browser/src/snapshot/` | browser script injection + Vitest tests |
| Repository policy constraints | `rules/` | autonomy, session init, naming, ELK troubleshooting |

## CODE MAP

| Symbol | Type | Location | Refs | Role |
|--------|------|----------|------|------|
| `serve` | function | `skills/dev-browser/src/index.ts` | high | launches browser context + API server |
| `connect` | function | `skills/dev-browser/src/client.ts` | high | persistent client/session attach |
| `serveRelay` | function | `skills/dev-browser/src/relay.ts` | high | extension relay server entry |

## CONVENTIONS

- Config-first root: runtime behavior is configured in `*.jsonc`, not root application code.
- `skills/dev-browser` is an independent package; do not assume root npm workspace wiring.
- TypeScript runtime is interpreter-driven (`tsx`) with strict `noEmit` config.
- Tests are local-only under dev-browser (`vitest` + Playwright); no CI workflow exists.

## ANTI-PATTERNS (THIS PROJECT)

- Never commit secret/account cache files (`antigravity-accounts.json`, signature caches, env secrets).
- Do not treat `logs/`, `log/`, `data/`, `profiles/`, `.sisyphus/` as source-of-truth code.
- Do not edit `skills/*/SKILL.md` when changing runtime behavior unless policy update is intentional.
- Do not assume remote CI checks; run local validation commands explicitly.

## UNIQUE STYLES

- Skill docs are operational contracts; `git-master` and `frontend-ui-ux` constrain agent behavior.
- Repo mixes config/runtime/policy artifacts in one root; boundaries are directory-based.
- `dev-browser` keeps a nested AGENTS chain (`skills/` -> `dev-browser/` -> `src/` -> `snapshot/`).

## COMMANDS

```bash
# root sanity
git status
npm run lint:naming

# dev browser
npm --prefix skills/dev-browser run start-server
npm --prefix skills/dev-browser run start-extension
npm --prefix skills/dev-browser test
```

## NOTES

- No `.github/workflows/` in repo; verification is local execution.
- Child AGENTS files carry only module deltas; avoid repeating root sections verbatim.
- `skills/dev-browser/profiles/` and `skills/dev-browser/tmp/` are runtime artifacts, not code.
