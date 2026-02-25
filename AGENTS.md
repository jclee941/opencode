# PROJECT KNOWLEDGE BASE

**Generated:** 2026-02-22 23:35:00 Asia/Seoul
**Commit:** e69b9a8
**Branch:** main

## OVERVIEW

OpenCode home workspace for agent runtime configuration and skill execution. The only executable code surface is `skills/dev-browser`; other skill folders are policy contracts. Root behavior is config-driven via `*.jsonc` files and plugin declarations.

## STRUCTURE

```text
./
├── config/                     # modular config partials (source of truth)
│   ├── base.jsonc              # core settings: plugins, mcp, formatter, watcher, tui, keybinds
│   ├── providers.jsonc         # google + minimax providers, 7 model definitions
│   └── lsp.jsonc               # 12 language server configurations
├── skills/                     # mixed registry: executable + policy skills
├── rules/                      # session/global operating policies (11 files, tiered)
├── scripts/                    # repo utilities (.mjs enforcers + Go generators)
│   ├── gen-opencode-config.go  # merges config/*.jsonc → opencode.jsonc
│   ├── validate-monorepo-naming.mjs
│   ├── lint-assistant-phrasing.mjs
│   ├── claude-hook-autonomy-guard.mjs
│   └── kratos-project-sync.go
├── pilot/                      # pilot service config/templates
├── plugins/                    # local plugin directory (currently empty)
├── snippet/                    # snippet config (config.jsonc)
├── docs/                       # monorepo standards docs
├── opencode.jsonc              # GENERATED — do not edit directly, run gen:config
├── oh-my-opencode.jsonc        # agent/category model mapping
├── dcp.jsonc                   # DCP plugin schema defaults
├── antigravity.jsonc           # Antigravity provider config
└── supermemory.jsonc            # Supermemory plugin config
```

## WHERE TO LOOK

| Task                           | Location                                 | Notes                                                               |
| ------------------------------ | ---------------------------------------- | ------------------------------------------------------------------- |
| Runtime and provider tuning    | `config/*.jsonc`                         | source of truth; `opencode.jsonc` is generated output               |
| Agent/model routing            | `oh-my-opencode.jsonc`                   | role-to-model and category mapping                                  |
| Rate-limit fallback config     | `oh-my-opencode.jsonc`                   | native `runtime_fallback` + per-agent/category `fallback_models`    |
| Plugin management              | `config/base.jsonc`                      | 9 plugins declared in source; reflected in generated opencode.jsonc |
| Rule architecture and priority | `rules/README.md`                        | tier model, conflict resolution, loading scope                      |
| Autonomy and question policy   | `rules/hard-autonomy-no-questions.md`    | highest-priority rule, zero-question enforcement                    |
| ELK troubleshooting            | `rules/elk-*.md`                         | global + OpenCode + Proxmox domain rules                            |
| Naming validation              | `scripts/validate-monorepo-naming.mjs`   | enforces kebab-case dirs, lowercase files                           |
| Autonomy guard hook            | `scripts/claude-hook-autonomy-guard.mjs` | pre-commit hook for forbidden phrasing                              |
| Config generation              | `scripts/gen-opencode-config.go`         | merges config/\*.jsonc → opencode.jsonc; supports --check           |
| Browser automation server      | `skills/dev-browser/src/index.ts`        | HTTP API + page registry lifecycle                                  |
| Browser relay bridge           | `skills/dev-browser/src/relay.ts`        | extension/CDP relay mode                                            |
| Snapshot extraction and tests  | `skills/dev-browser/src/snapshot/`       | browser script injection + Vitest tests                             |

## CODE MAP

| Symbol       | Type     | Location                           | Refs | Role                                  |
| ------------ | -------- | ---------------------------------- | ---- | ------------------------------------- |
| `serve`      | function | `skills/dev-browser/src/index.ts`  | high | launches browser context + API server |
| `connect`    | function | `skills/dev-browser/src/client.ts` | high | persistent client/session attach      |
| `serveRelay` | function | `skills/dev-browser/src/relay.ts`  | high | extension relay server entry          |

## CONVENTIONS

- Config-first root: runtime behavior is configured in `*.jsonc`, not root application code.
- `skills/dev-browser` is an independent package; do not assume root npm workspace wiring.
- TypeScript runtime is interpreter-driven (`tsx`) with strict `noEmit` config.
- Tests are local-only under dev-browser (`vitest` + Playwright); no CI workflow exists.
- Rules follow a 3-tier model: Tier 1 always-loaded (6 rules), Tier 2 domain-scoped, Tier 3 process-scoped.
- Plugin config files (`dcp.jsonc`, `supermemory.jsonc`) are root-level siblings to `opencode.jsonc`.
- Operational scripts must be Go (`*.go`), not shell. Migrate `*.sh` on contact.

## ANTI-PATTERNS (THIS PROJECT)

- Never commit secret/account cache files (`antigravity-accounts.json`, signature caches, env secrets).
- Do not treat `logs/`, `log/`, `data/`, `profiles/`, `.sisyphus/` as source-of-truth code.
- Do not edit `skills/*/SKILL.md` when changing runtime behavior unless policy update is intentional.
- Do not assume remote CI checks; run local validation commands explicitly.
- Do not load Tier 2/3 rules via glob in `opencode.jsonc`; use explicit paths to avoid context bloat.
- Do not edit `opencode.jsonc` directly; edit `config/*.jsonc` and run `gen:config`.

## UNIQUE STYLES

- Skill docs are operational contracts; `git-master` and `frontend-ui-ux` constrain agent behavior.
- Repo mixes config/runtime/policy artifacts in one root; boundaries are directory-based.
- `dev-browser` keeps a nested AGENTS chain (`skills/` -> `dev-browser/` -> `src/` -> `snapshot/`).
- Rules have their own README-driven architecture with priority ordering and inheritance model.

## COMMANDS

```bash
# root sanity
git status
npm run lint:naming
npm run lint:assistant-phrasing

# config generation
npm run gen:config          # merge config/*.jsonc → opencode.jsonc
npm run gen:config:check    # verify opencode.jsonc is up-to-date (exit 1 if stale)

# dev browser
npm --prefix skills/dev-browser run start-server
npm --prefix skills/dev-browser run start-extension
npm --prefix skills/dev-browser test
```

## NOTES

- No `.github/workflows/` in repo; verification is local execution.
- Child AGENTS files carry only module deltas; avoid repeating root sections verbatim.
- `skills/dev-browser/profiles/` and `skills/dev-browser/tmp/` are runtime artifacts, not code.
- `plugins/` directory exists but is empty; plugins are installed via npm and declared in `opencode.jsonc`.
- `pilot/` contains pilot service config (`config.yaml`) and prompt templates (`templates/`).
- Rate-limit fallback is configured natively in `oh-my-opencode.jsonc` via `runtime_fallback` (global) and per-agent/category `fallback_models`. No standalone fallback JSON file needed.
