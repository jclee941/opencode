# Monorepo Naming and Structure Rules

Apply these rules for all changes in this repository.

## Structure

1. Keep top-level domains stable and purpose-driven:
   - `skills/`: reusable skill packages
   - `scripts/`: operational tooling
   - `rules/`: instruction files
   - `docs/`: standards and reference docs
2. Keep root config naming explicit: repository-level config should remain root `*.jsonc` (for example `opencode.jsonc`, `oh-my-opencode.jsonc`).
3. Do not place runtime artifacts in source domains (`logs/`, `log/`, `data/`, `tmp/`, `profiles/`).
4. Prefer non-breaking evolution: introduce structure conventions before moving large directories.

## Naming

1. Use `kebab-case` for directory names by default (`^[a-z0-9][a-z0-9-]*$`).
2. Use lowercase file names with dots/hyphens allowed by default (`^[a-z0-9][a-z0-9.-]*$`).
3. Keep language/tooling exceptions only when conventional (`__tests__`, `__snapshots__`, `__fixtures__`).
4. Keep contract file names uppercase only when standard (`AGENTS.md`, `SKILL.md`, `README.md`, `CHANGELOG.md`, `LICENSE`).
5. Avoid ambiguous names (single-symbol files, ad-hoc temp names) in source directories.
6. Use standardized backup naming:
   - `<filename>.backup-YYYYMMDD-HHMMSS[-mmmZ]`

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

## Enforcement Source of Truth

1. Naming validation behavior is defined by `scripts/validate-monorepo-naming.mjs`.
2. If you change naming exceptions in docs, update this file and `scripts/validate-monorepo-naming.mjs` together in the same change.
3. Ignore/runtime directories are excluded from naming checks by policy and validator:
   - `.git/`, `node_modules/`, `data/`, `log/`, `logs/`, `tmp/`, `profiles/`, `.sisyphus/`, `.cache/`, `dist/`, `coverage/`, `.next/`, `.venv/`

## Refactoring policy

1. Rename safely: update all direct references in docs/config/scripts.
2. Keep behavior unchanged unless the request explicitly asks for functional changes.
3. Run verification after rename/refactor:
   - `npm run lint:naming`
   - format check and targeted diagnostics

## Script migration policy

1. For this repository, operational script files must be implemented in Go (`*.go`), not shell script files (`*.sh`).
2. When touching an existing `*.sh` operational script, migrate it to a Go entrypoint in the same change.
3. After migration, update all direct references in docs/config/skills and remove the superseded `*.sh` file.

## Existing document normalization policy

Use this when reorganizing existing documents to comply with repository structure and naming rules.

Execution checklist reference: `rules/document-normalization-runbook.md`

1. Move standards/guides/runbooks to `docs/` unless they are runtime instructions (`rules/`) or skill contracts (`skills/**/AGENTS.md`, `skills/**/SKILL.md`).
2. Keep repository-wide operational rules in `rules/`; do not duplicate the same policy text in `docs/`.
3. Keep root-level markdown files only for established contract files (`README.md`, `AGENTS.md`, `CHANGELOG.md`, `LICENSE`) and root configs already defined by policy.
4. Apply naming rules during migration:
   - directories: `kebab-case`
   - regular files: lowercase with dots/hyphens
   - contract files: uppercase only when standard (`AGENTS.md`, `SKILL.md`, `README.md`, `CHANGELOG.md`, `LICENSE`)
5. For document renames, preserve meaning while normalizing names (for example `Troubleshooting Guide.md` -> `troubleshooting-guide.md`).
6. For document moves/renames, update all direct references in markdown, configs, and scripts in the same change.
7. If replacing a document, keep one canonical target file and remove or redirect duplicates to avoid policy drift.
8. Before finishing document normalization, run `npm run lint:naming` and confirm no new naming violations.
