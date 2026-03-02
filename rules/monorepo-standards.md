# Monorepo Naming and Structure Rules

Apply these rules as default monorepo standards across projects unless a
project-level policy explicitly overrides them.

For detailed structure guidance, Bazel compatibility, and document normalization procedures,
see `docs/monorepo-structure-and-naming.md`.

## Structure

1. Keep top-level domains stable and purpose-driven:
   - `skills/`: reusable skill packages
   - `scripts/`: operational tooling
   - `rules/`: instruction files
   - `docs/`: standards and reference docs
2. Keep root config naming explicit: repository-level config should remain
   root `*.jsonc` with clear, purpose-driven names.
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

## Enforcement source of truth

1. Naming validation behavior is defined by `scripts/validate-monorepo-naming.mjs`.
2. If you change naming exceptions in docs, update this file and `scripts/validate-monorepo-naming.mjs` together in the same change.
3. Ignore/runtime directories are excluded from naming checks by policy and validator:
   - `.git/`, `node_modules/`, `data/`, `log/`, `logs/`, `tmp/`, `profiles/`, `.sisyphus/`, `.cache/`, `dist/`, `coverage/`, `.next/`, `.venv/`, `.ruff_cache/`

## Refactoring policy

1. Rename safely: update all direct references in docs/config/scripts.
2. Keep behavior unchanged unless the request explicitly asks for functional changes.
3. Run verification after rename/refactor:
   - `npm run lint:naming`
   - format check and targeted diagnostics

## Script migration policy

1. Operational script files are enforced as Go (`*.go`) by default.
2. When touching an existing shell operational script (`*.sh`), migrate it to
   a Go entrypoint in the same change.
3. After migration, update direct references in docs/config/scripts and remove
   the superseded shell script.
4. Exception: Node.js scripts (`*.mjs`) used as git hooks, linters, or validators
   that depend on the Node.js ecosystem (e.g., `commitlint`, AST parsing) are
   exempt from Go migration.

### Shell-to-Go migration checklist

When migrating a `.sh` script to Go, follow this sequence in a single change:

1. **Create Go entrypoint** in the same directory as the original script.
   - File name: match the original minus extension (`deploy.sh` → `deploy.go`).
   - Package: `main` with `func main()`.
   - Use `os/exec` for subprocess calls, `flag` for CLI arguments.
2. **Preserve behavior exactly**:
   - Same CLI flags and positional arguments.
   - Same exit codes (0 = success, non-zero = failure).
   - Same stdout/stderr output format.
   - Same environment variable consumption (`os.Getenv`).
3. **Prefer stdlib over dependencies**:
   - `os/exec` over third-party command runners.
   - `path/filepath` over string manipulation for paths.
   - `encoding/json` for JSON parsing (replace `jq`).
   - `regexp` for pattern matching (replace `grep`/`sed`/`awk`).
4. **Update all references** in the same commit:
   - `package.json` scripts.
   - `config/*.jsonc` (formatters, watchers).
   - `.githooks/*` hook scripts.
   - `.github/workflows/*.yml` CI steps.
   - `docs/` and `rules/` prose references.
   - `AGENTS.md` structure tree.
5. **Delete the original `.sh` file** in the same commit.
6. **Verify**:
   - `go build scripts/<name>.go` succeeds.
   - `go vet scripts/<name>.go` clean.
   - `npm run lint:naming` passes.
   - Manual smoke test: run the new Go script with the same args.

## Reference composition

1. Loaded as Tier 0 rule via `opencode.jsonc`.
2. Defers to `hard-autonomy-no-questions.md` on execution posture.
3. Detailed structure guidance and Bazel profile: `docs/monorepo-structure-and-naming.md`.
