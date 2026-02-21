# Monorepo Structure and Naming Rules

This document defines the default structure and naming conventions for this repository.

## 1) Top-level structure

- `skills/`: Skill packages and skill-specific references.
- `scripts/`: Operational scripts only.
- `pilot/`: Pilot configs and templates.
- `rules/`: Session and workflow instruction files.
- `docs/`: Repository-wide standards and reference docs.
- Root `*.jsonc`: OpenCode-level global configuration files.

Do not place runtime artifacts (logs, caches, temp files) inside source directories.

## 1.1) Google-style monorepo layout principles

Use these principles when adding new packages or restructuring modules:

- Keep a small number of stable root domains (`apps/`, `packages/`, `tools/`, `docs/`, `infra/`) for long-term discoverability.
- Place runnable products under `apps/`, reusable libraries under `packages/`, and operational tooling under `tools/`.
- Treat documentation as first-class code (`docs/` with architecture, standards, runbooks).
- Keep ownership boundaries explicit: one package, one clear public API surface.
- Enforce naming/linting rules centrally so all subprojects follow the same conventions.

For this repository, existing domains map as follows:

- `skills/` behaves as `packages/skills`.
- `scripts/` behaves as `tools/scripts`.
- `docs/` remains repository-wide documentation.

This mapping is intentionally non-breaking; no forced directory moves are required immediately.

## 2) Directory naming

- Use `kebab-case` for directories (`^[a-z0-9][a-z0-9-]*$`).
- Keep domain-first grouping (`dev-browser`, `agent-browser`).

## 3) File naming

- Default file style: lowercase with dots/hyphens (`^[a-z0-9][a-z0-9.-]*$`).
- Keep extension-specific conventions:
  - Config files: `*.jsonc`, `*.json`, `*.yaml`, `*.yml`
  - Operational scripts: `*.go`
  - Markdown docs: `*.md`
- Allowed uppercase exceptions for contract files:
  - `AGENTS.md`, `SKILL.md`, `README.md`, `CHANGELOG.md`, `LICENSE`
- Avoid ambiguous/temporary names at root (for example single-symbol filenames).

### 3.1 Backup naming (standard)

Use one backup format consistently:

`<original-filename>.backup-YYYYMMDD-HHMMSS[-mmmZ]`

Examples:

- `opencode.jsonc.backup-20260217-111649`
- `oh-my-opencode.jsonc.backup-20260219-111047-824Z`

## 4) Refactoring policy for renames

When renaming files/directories:

1. Rename to rule-compliant names.
2. Update all references in configs/scripts/docs.
3. Run validation (`grep`, diagnostics, formatter checks).
4. Keep behavior unchanged; rename only unless explicitly requested.

## 5) Scope guardrails

- Do not rename files under `node_modules/`, `data/`, `logs/`, `log/`, `tmp/`, `profiles/`.
- Do not rename generated lock files unless toolchain explicitly requires it.
- Prioritize low-risk, high-value normalization first (typos, backup formats, ambiguous filenames).

## 6) Phased refactor plan

Phase 1 (now):

- Normalize ambiguous and typo-prone filenames.
- Standardize backup filename format.
- Add automated naming validation.

Phase 2 (planned):

- Introduce virtual domain docs (`apps/`, `packages/`, `tools/`) and migration map.
- Add package-level ownership metadata and README index.

Phase 3 (optional):

- Physically migrate directories to `apps/`, `packages/`, `tools/` if operationally beneficial.
- Add CI checks for naming and structure policy.

## 7) Enforcement source of truth

- Naming validation behavior is implemented in `scripts/validate-monorepo-naming.mjs`.
- Keep docs policy and validator exceptions synchronized in the same change.
- Ignore/runtime directories excluded from checks:
  - `.git/`, `node_modules/`, `data/`, `log/`, `logs/`, `tmp/`, `profiles/`, `.sisyphus/`, `.cache/`, `dist/`, `coverage/`, `.next/`, `.venv/`
