# Monorepo Structure and Naming Reference

This document provides detailed structure guidance, compatibility profiles, and
procedures that supplement the executive rules in `rules/monorepo-standards.md`.

## Top-level structure

- `skills/`: Skill packages and skill-specific references.
- `scripts/`: Operational scripts only.
- `pilot/`: Pilot configs and templates.
- `rules/`: Session and workflow instruction files.
- `docs/`: Repository-wide standards and reference docs.
- Root `*.jsonc`: OpenCode-level global configuration files.

Do not place runtime artifacts (logs, caches, temp files) inside source directories.

## Google-style monorepo layout principles

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

## Google3/Bazel style compatibility profile

Use this profile when a subtree is managed with Bazel/google3-style conventions.

1. Keep Bazel package paths stable and reviewable; prefer small package directories with explicit ownership.
2. Allow `lower_snake_case` for Bazel-oriented directories in addition to default `kebab-case`.
3. Allow canonical Bazel control files:
   - `BUILD`, `BUILD.bazel`, `WORKSPACE`, `WORKSPACE.bazel`, `MODULE.bazel`
4. Allow ownership contract file used in google3-style workflows:
   - `OWNERS`
5. Keep Starlark file names lowercase and descriptive (`*.bzl`, prefer `lower_snake_case`).
6. Do not mix unrelated package concerns in one Bazel package; split by clear build/runtime boundaries.

## Naming details

### Directory naming

- Use `kebab-case` for directories (`^[a-z0-9][a-z0-9-]*$`).
- Keep domain-first grouping (`dev-browser`, `agent-browser`).

### File naming

- Default file style: lowercase with dots/hyphens (`^[a-z0-9][a-z0-9.-]*$`).
- Keep extension-specific conventions:
  - Config files: `*.jsonc`, `*.json`, `*.yaml`, `*.yml`
  - Operational scripts: `*.go`
  - Markdown docs: `*.md`
- Allowed uppercase exceptions for contract files:
  - `AGENTS.md`, `SKILL.md`, `README.md`, `CHANGELOG.md`, `LICENSE`
- Avoid ambiguous/temporary names at root (for example single-symbol filenames).

### Backup naming (standard)

Use one backup format consistently:

`<original-filename>.backup-YYYYMMDD-HHMMSS[-mmmZ]`

Examples:

- `opencode.jsonc.backup-20260217-111649`
- `oh-my-opencode.jsonc.backup-20260219-111047-824Z`

## Document normalization procedure

Use this when reorganizing existing documents to comply with repository structure and naming rules.

### Execution checklist

1. Inventory current documents and classify each as:
   - runtime instruction (`rules/`)
   - standards/reference (`docs/`)
   - skill contract (`skills/**/AGENTS.md`, `skills/**/SKILL.md`)
   - root contract (`README.md`, `AGENTS.md`, `CHANGELOG.md`, `LICENSE`)
2. Choose canonical targets:
   - one canonical file per policy/topic
   - mark duplicates for deletion or redirect
3. Apply structure moves:
   - standards/guides/runbooks → `docs/`
   - operational instructions → `rules/`
4. Apply naming normalization:
   - directories: `kebab-case`
   - regular files: lowercase with dots/hyphens
   - contract files: uppercase only when standard
5. Update references in the same change (markdown links, config/script references, rule cross-references).
6. Verify: run `npm run lint:naming` and spot-check moved/renamed links.

### Safety rules

1. Preserve document meaning during rename/move.
2. Remove stale duplicate documents to avoid policy drift.
3. Do not place runtime artifacts in source domains.

### Output contract

1. List files moved/renamed/deleted.
2. List references updated.
3. Report naming lint result.

## Refactoring policy for renames

When renaming files/directories:

1. Rename to rule-compliant names.
2. Update all references in configs/scripts/docs.
3. Run validation (`grep`, diagnostics, formatter checks).
4. Keep behavior unchanged; rename only unless explicitly requested.

## Scope guardrails

- Do not rename files under `node_modules/`, `data/`, `logs/`, `log/`, `tmp/`, `profiles/`.
- Do not rename generated lock files unless toolchain explicitly requires it.
- Prioritize low-risk, high-value normalization first (typos, backup formats, ambiguous filenames).

## Enforcement source of truth

- Naming validation behavior is implemented in `scripts/validate-monorepo-naming.mjs`.
- Keep docs policy and validator exceptions synchronized in the same change.
- Ignore/runtime directories excluded from checks:
  - `.git/`, `node_modules/`, `data/`, `log/`, `logs/`, `tmp/`, `profiles/`, `.sisyphus/`, `.cache/`, `dist/`, `coverage/`, `.next/`, `.venv/`, `.ruff_cache/`
