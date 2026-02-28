# Code Modularization Policy

Apply this policy when creating, reviewing, or modifying code files.
This is the canonical source for file size governance and modularization verification.

Priority: Tier 1 baseline. Loaded globally via `opencode.jsonc` instructions.

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

## Anti-patterns

- **Premature splitting**: Splitting <200 LOC files or during prototyping.
- **Shotgun splitting**: Every function in its own file regardless of cohesion.
- **False modularity**: Split files still share state via globals or circular deps.
- **Over-abstraction**: Abstract base classes or DI frameworks just to justify a split.
- **Catch-all names**: `helpers.ts`, `utils.ts`, `misc.ts` as dumps.

## Reference composition

1. Loaded as Tier 1 baseline rule via `opencode.jsonc`.
2. Defers to `hard-autonomy-no-questions.md` on execution posture.
3. Defers to `monorepo-standards.md` on naming conventions.
4. Detailed strategies and language guidance: `docs/code-modularization-reference.md`.
