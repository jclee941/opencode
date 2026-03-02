# Ruleset Architecture

How rule files in `rules/` are composed, resolved, and loaded.

## Priority order

Conflict resolution priority (1 = highest). Priority is independent of tier:

| # | File | Tier | Domain |
|---|------|------|--------|
| 1 | `hard-autonomy-no-questions.md` | 0 | Execution posture, blocked-step handling |
| 2 | `archon-workflow.md` | 0 | Archon task management, RAG workflow |
| 3 | `requirements-verification.md` | 0 | Requirements check before/after implementation |
| 4 | Domain rules (`elk-troubleshooting.md`) | 2 | Domain-specific |
| 5 | `session-init.md` | 0 | Session bootstrap, MCP schema hygiene |
| 6 | `deployment-automation.md` | 1 | CI/CD policy |
| 7 | `monorepo-standards.md` | 0 | Structure, naming |
| 8 | `code-modularization.md` | 1 | File size governance |
| 9 | `bmad-integration.md` | 1 | BMAD artifact consumption |
| 10 | `auto-build-pipeline.md` | 1 | Spec-to-PR pipeline |

## Tier model

### Tier 0 — Always loaded (via `config/base.jsonc` instructions)

Minimal set loaded every session (~1,500 tokens total):

- `hard-autonomy-no-questions.md` — execution posture, zero-question policy
- `archon-workflow.md` — Archon task management, RAG workflow
- `session-init.md` — session bootstrap + MCP schema hygiene (merged)
- `requirements-verification.md` — requirements check gate
- `monorepo-standards.md` — structure and naming

### Tier 1 — On-demand (agent reads when task domain matches)

Files remain in `rules/` but are NOT in the instructions array:

- `deployment-automation.md` — read when: deploy/CI task
- `code-modularization.md` — read when: refactor/split, or file >300 LOC touched
- `bmad-integration.md` — read when: `_bmad-output/` detected
- `auto-build-pipeline.md` — read when: `/start-work` or auto-build triggered

### Tier 2 — Domain-specific (loaded when domain in scope)

- `elk-troubleshooting.md` — ELK troubleshooting (all domains)

## Deleted / merged files

- `mcp-schema-hygiene.md` → merged into `session-init.md`
- `AGENTS.md` (rules/) → deleted (duplicate of root AGENTS.md)

## Conflict resolution

1. `hard-autonomy-no-questions.md` overrides all.
2. Prefer stricter safety constraints.
3. One canonical source per rule — no duplication.
4. Format: `applied: <base>, overridden by: <exception>, scope: <bounded>`

## Instruction loading

1. `config/base.jsonc` loads Tier 0 via explicit paths (not glob).
2. Root `AGENTS.md` loaded natively by OpenCode — never in instructions array.
3. Tier 1 rules read on-demand, not pre-loaded.
4. Reduces baseline from ~35K to ~2K tokens.
