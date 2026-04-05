# Ruleset Architecture

How rule files in `rules/` are composed, resolved, and loaded.

## Priority order

Conflict resolution priority (1 = highest). Priority is independent of tier.
Files not listed in this table are treated as domain add-ons and resolved by
specificity after Tier 0 baseline rules.

| # | File | Tier | Domain |
|---|------|------|--------|
| 1 | `00-hard-autonomy-no-questions.md` | 0 | Execution posture, blocked-step handling |
| 2 | `00-archon-workflow.md` | 0 | Archon task management, RAG workflow |
| 3 | `00-requirements-verification.md` | 0 | Requirements check before/after implementation |
| 4 | `00-session-init.md` | 0 | Session bootstrap, MCP schema hygiene |
| 5 | `00-monorepo-standards.md` | 0 | Structure, naming |
| 6 | **`00-code-modularization.md`** | **0** | **File size governance (200 LOC)** |
| 7 | `deployment-automation.md` | 1 | CI/CD policy |
| 8 | `01-auto-build-pipeline.md` | 1 | Spec-to-PR pipeline (overview) |
| 9 | `02-auto-build-pipeline-execution.md` | 1 | Spec-to-PR pipeline (build + QA phases) |
| 11 | `03-auto-build-pipeline-completion.md` | 1 | Spec-to-PR pipeline (merge + PR phases) |
| 12 | `01-mcp-schema-hygiene.md` | 1 | MCP tool call schema validation |
| 13 | `01-onepassword-integration.md` | 1 | 1Password integration policy |
| 14 | `02-onepassword-integration-patterns.md` | 1 | 1Password implementation patterns |
| 15 | `03-onepassword-integration-reference.md` | 1 | 1Password reference map |
| 16 | `01-onepassword-secrets-naming.md` | 1 | 1Password naming specification |
| 17 | `02-onepassword-secrets-naming-examples.md` | 1 | 1Password naming examples |
| 18 | `03-onepassword-secrets-naming-operations.md` | 1 | 1Password naming operations |
| 19 | `01-msa-refactoring.md` | 1 | Monolith → MSA migration guidance |
| 20 | `01-elk-troubleshooting.md` | 2 | ELK troubleshooting (overview) |
| 21 | `02-elk-troubleshooting-opencode.md` | 2 | ELK OpenCode domain |
| 22 | `03-elk-troubleshooting-proxmox.md` | 2 | ELK Proxmox domain |

## Tier model

### Tier 0 — Always loaded (via `config/base.jsonc` instructions)

Minimal set loaded every session (~1,500 tokens total):

- `00-hard-autonomy-no-questions.md` — execution posture, zero-question policy
- `00-archon-workflow.md` — Archon task management, RAG workflow
- `00-session-init.md` — session bootstrap + MCP schema hygiene (merged)
- `00-requirements-verification.md` — requirements check gate
- `00-monorepo-standards.md` — structure and naming
- **`00-code-modularization.md` — file size governance (200 LOC limit)**

### Tier 1 — On-demand (agent reads when task domain matches)

Files remain in `rules/` but are NOT in the instructions array:

- `deployment-automation.md` — read when: deploy/CI task
- `01-auto-build-pipeline.md` — read when: `/start-work` or auto-build triggered
  - `02-auto-build-pipeline-execution.md` — execution phase
  - `03-auto-build-pipeline-completion.md` — completion phase
- `01-mcp-schema-hygiene.md` — read when: MCP -32602 errors or new MCP tool integration
- `01-onepassword-integration.md` — read when: secrets/credentials/`op://` handling
  - `02-onepassword-integration-patterns.md` — implementation patterns
  - `03-onepassword-integration-reference.md` — reference map
- `01-onepassword-secrets-naming.md` — read when: 1Password schema/name audits
  - `02-onepassword-secrets-naming-examples.md` — examples
  - `03-onepassword-secrets-naming-operations.md` — operations
- `01-msa-refactoring.md` — read when: monolith decomposition/service boundary work

### Tier 2 — Domain-specific (loaded when domain in scope)

- `01-elk-troubleshooting.md` — ELK troubleshooting (overview)
  - `02-elk-troubleshooting-opencode.md` — OpenCode domain
  - `03-elk-troubleshooting-proxmox.md` — Proxmox domain

## File numbering convention (200 LOC compliance)

All rule files follow the 200 LOC modularization limit:

- **00-***: Main/overview files (entry point, <200 LOC)
- **02-***: Execution/pattern files (implementation details, <200 LOC)
- **03-***: Completion/reference files (reference material, <200 LOC)

Example: `auto-build-pipeline` is split into:
- `01-auto-build-pipeline.md` (92 lines) — overview
- `02-auto-build-pipeline-execution.md` (69 lines) — build + QA phases
- `03-auto-build-pipeline-completion.md` (83 lines) — merge + PR phases

### Tier 2 domain numbering

Domain-specific rule series (e.g., `elk-troubleshooting`) use their own `01-`/`02-`/`03-` numbering within the domain. The prefix indicates position within the series, not the tier:
- `01-elk-troubleshooting.md` — ELK overview (entry point)
- `02-elk-troubleshooting-opencode.md` — OpenCode domain specifics
- `03-elk-troubleshooting-proxmox.md` — Proxmox domain specifics

## Metadata files in `rules/`

- `AGENTS.md` in this directory is a local knowledge index for the `rules/`
  subtree. It is documentation metadata, not a loadable runtime rule.

## Conflict resolution

1. `00-hard-autonomy-no-questions.md` overrides all.
2. Prefer stricter safety constraints.
3. One canonical source per rule — no duplication.
4. Format: `applied: <base>, overridden by: <exception>, scope: <bounded>`

## Instruction loading

1. `config/base.jsonc` loads Tier 0 via explicit paths (not glob).
2. Root `AGENTS.md` loaded natively by OpenCode — never in instructions array.
3. Tier 1 rules read on-demand, not pre-loaded.
4. Reduces baseline from ~35K to ~2K tokens.

## Consistency invariants

To prevent rule drift and contradictory guidance:

1. Any file declaring `Priority` or `Tier` must match this README.
2. `config/base.jsonc` instructions must contain only Tier 0 files.
3. If a rule file is added/removed/renamed, update this README and
   `rules/AGENTS.md` in the same change.
