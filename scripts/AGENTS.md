# SCRIPTS KNOWLEDGE BASE

**Scope:** `scripts/`
**Parent:** `AGENTS.md`

## OVERVIEW

Operational automation lives here: config generation, git automation, Kratos sync, hook installation, and repository validators. Go is the default runtime; `.mjs` exists only where Node tooling is the point.

## STRUCTURE

```text
scripts/
├── gen-opencode-config.go               # merge config/*.jsonc -> opencode.jsonc
├── gen-runtime-inventory.go             # derive docs/runtime-inventory.md from runtime sources
├── omo-conflicts.json                   # canonical OMO conflict watchlist shared by generators
├── git-ship.go                          # checks -> stage -> commit -> optional push
├── install-git-hooks.go                 # copies .githooks/* into .git/hooks
├── config-recover.go                    # recover from direct edits to opencode.jsonc
├── pre-commit.go                        # git pre-commit hook (blocks direct opencode.jsonc edits)
├── pre-push.go                          # git pre-push hook (runs validation checks)
├── kratos-project-sync.go               # project discovery + systemd sync helpers
├── opencode-tmux-server.go              # per-tmux-window OpenCode server lifecycle manager
├── switch-model.go                      # primary model preset switcher
├── validate-config-refs.go              # models/plugins/config cross-checks
├── validate-requirements-verification.go# rules/docs structure validator
├── omo-auto-update.go                   # cron: git pull + npm install for Dependabot merges
├── validate-monorepo-naming.mjs         # kebab-case + no-.sh enforcement
├── lint-assistant-phrasing.mjs          # denylist phrasing scan for markdown/session exports
├── claude-hook-autonomy-guard.mjs       # runtime guard for assistant output
└── validate-modularization.go           # LOC/complexity thresholds per 00-code-modularization.md
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Regenerate config | `gen-opencode-config.go` | comment stripping, deep merge, OMO conflict removal |
| Regenerate runtime visibility report | `gen-runtime-inventory.go` | active plugins, MCP, local surfaces, OMO watchlist |
| Change OMO conflict source | `omo-conflicts.json` | shared by config generation and runtime inventory |
| Change automated commit flow | `git-ship.go` | secret path blocks, `--paths`, upstream fallback |
| Add config validation | `validate-config-refs.go` | checks root JSONC surfaces together |
| Change requirements doc lint | `validate-requirements-verification.go` | section contract + imperative verbs |
| Enforce naming policy | `validate-monorepo-naming.mjs` | also blocks tracked `.sh` files |
| Install/update git hooks | `install-git-hooks.go` | syncs `.githooks/` into live hooks |
| Adjust autonomy wording guard | `claude-hook-autonomy-guard.mjs` | denylist regex for assistant output |
| Manage per-tmux-window servers | `opencode-tmux-server.go` | start/stop/attach/status/cleanup lifecycle |
| Switch primary model preset | `switch-model.go` | updates base.jsonc + oh-my-opencode.jsonc |
| Auto-update OMO from Dependabot | `omo-auto-update.go` | cron: git pull --ff-only + npm install + config regen |
| Validate modularization limits | `validate-modularization.go` | LOC/complexity per 00-code-modularization.md |

## CONVENTIONS

- Run scripts from repo root; many resolve paths relative to `os.Getwd()`.
- Keep operational behavior in Go even when parsing JSONC or shelling out.
- Keep validator output CI-friendly: deterministic ordering, plain text, non-zero exit on failure.
- `git-ship.go` is the sanctioned automation path for staged commits in this repo.

## ANTI-PATTERNS (THIS SUBTREE)

- Do not add new operational `.sh` scripts.
- Do not bypass `npm run prepush:check` inside commit automation.
- Do not weaken blocked secret-path checks in `git-ship.go`.
- Do not add nondeterministic merge or validation ordering.
