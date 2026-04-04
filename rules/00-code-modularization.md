# Code Modularization Policy

Apply this policy when creating, reviewing, or modifying code files.
This is the canonical source for file size governance and modularization verification.

Priority: Tier 0 baseline. Globally loaded — applies to all file operations.

For detailed split strategies, language-specific guidance, and examples, see `docs/code-modularization-reference.md`.

## Scope

**In scope:** Source code (`.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.go`, `.rs`, `.java`, `.cs`), configuration-as-code (`.tf`, `.yml`, Kubernetes manifests), test files, shell/scripting files.

**Out of scope:** Generated files, data files, markdown docs, vendored code, migration files.

## Core principles

1. **Single Responsibility**: Every file should have one clear reason to change.
2. **Cognitive Load**: Understand a file's purpose within 30 seconds of opening it.
3. **Navigability**: File names and directory structure must be grep-friendly.
4. **Minimal Coupling**: Splits must reduce inter-file dependencies, not increase them.
5. **Incremental Delivery**: Modularization is refactoring — behavior must not change.

## Thresholds

### Hard (mandatory action)

| Metric                       | Threshold | Action                                      |
| ---------------------------- | --------- | ------------------------------------------- |
| Lines of code (LOC)          | > 500     | Must split before merging new functionality |
| Lines of code (LOC)          | > 300     | Must split if adding new logic              |
| Exported symbols             | > 15      | Must split by export grouping               |
| Cyclomatic complexity (file) | > 20      | Must extract complex branches               |
| Classes/components           | > 3       | Must split to one-per-file                  |
| Distinct concerns            | > 2       | Must split by concern boundary              |

### Soft (review and justify)

| Metric         | Threshold | Action                                       |
| -------------- | --------- | -------------------------------------------- |
| LOC            | > 200     | Assess if natural split points exist         |
| Function count | > 10      | Assess grouping by responsibility            |
| Import count   | > 15      | Assess if file is a hub that should delegate |

### When NOT to split

- Single cohesive unit (one algorithm, one state machine)
- Splitting would create circular dependencies
- Barrel/index files (exempt from LOC limits)
- Test files mirroring a single source file
- Generated code

## Execution rules

1. When a file exceeds a hard threshold during implementation, split immediately.
2. Choose the most obvious split strategy; document the choice in the commit message.
3. Modularization commits must not mix with functional changes.
4. Commit message format: `refactor({module}): split {original} into {n} files`
5. Verify no circular dependencies introduced after any split.

## Dependency direction

Handlers → Services → Repositories → Clients (downward only).
Types and Utils may be imported by any layer (sideways/upward).
Never reverse the direction.

## Verification checklist

| #   | Check                               | Evidence                           |
| --- | ----------------------------------- | ---------------------------------- |
| 1   | All output files below 500 LOC      | `wc -l` on each file               |
| 2   | No circular dependencies            | Import trace or `madge --circular` |
| 3   | All existing tests pass             | Test runner output                 |
| 4   | Build/typecheck succeeds            | Exit code 0                        |
| 5   | No new lint errors                  | `lsp_diagnostics` on changed files |
| 6   | All external imports updated        | Grep for old paths returns 0       |
| 7   | File names follow naming convention | `npm run lint:naming`              |

- **Catch-all names**: `helpers.ts`, `utils.ts`, `misc.ts` as dumps.

## Rollback procedure

If a split makes code worse (increased coupling, harder navigation, test failures):

1. **Revert via git**: `git revert <split-commit-hash>` or `git reset --hard` if not pushed.
2. **Alternative split**: Try a different boundary using `docs/code-modularization-reference.md` strategies.
3. **Accept temporary large file**: Some units are inherently cohesive; document the exception in a file-level comment.
4. **Merge criteria**: Re-merge if split files have >50% shared imports or developers frequently open both to understand one.

## Barrel file governance

Barrel files (index.ts, index.js) are exempt from LOC limits but require:

1. **No side effects**: Only re-exports; no initialization, config loading, or module augmentation.
2. **Re-export limit**: Maximum 25 explicit exports per barrel. Split domain if exceeded.
3. **No nested barrels**: Do not import from barrel files in sibling barrels.
4. **Naming**: Use `index.{ext}` for barrels; avoid `mod.ts`, `main.ts`, `all.ts`.
5. **Prefer explicit imports**: Only use barrel imports for public API; internal code should import from source files.

> **Anti-pattern**: A barrel file that imports from another barrel, creating deep resolution chains.

## Reference composition

1. Loaded as Tier 0 baseline rule (globally loaded via `opencode.jsonc`).
2. Defers to `00-hard-autonomy-no-questions.md` on execution posture.
3. Defers to `00-monorepo-standards.md` on naming conventions.
4. Detailed strategies and language guidance: `docs/code-modularization-reference.md`.
