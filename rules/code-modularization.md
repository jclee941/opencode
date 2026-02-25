# Code Modularization Policy

Apply this policy when creating, reviewing, or modifying code files.
This is the canonical source for file size governance, split strategies,
and modularization verification.

Priority: Tier 1 baseline. Loaded globally via `opencode.jsonc` instructions.

## Scope

**In scope:**

- Source code files (`.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.go`, `.rs`, `.java`, `.cs`, etc.)
- Configuration-as-code files (Terraform `.tf`, Ansible `.yml`, Kubernetes manifests)
- Test files that mirror source structure
- Shell/scripting files (`.sh`, `.bash`, `.zsh`, `.mjs`)

**Out of scope:**

- Generated files (bundler output, compiled assets, lock files)
- Data files (JSON fixtures, CSV, SQL dumps)
- Markdown documentation (governed by `monorepo-standards.md`)
- Vendored/third-party code
- Migration files (database migrations have their own sequencing rules)

---

## Core principles

1. **Single Responsibility**: Every file should have one clear reason to change.
2. **Cognitive Load**: A developer should understand a file's purpose within 30 seconds of opening it.
3. **Navigability**: File names and directory structure should make the codebase grep-friendly.
4. **Minimal Coupling**: Splits should reduce inter-file dependencies, not increase them.
5. **Incremental Delivery**: Modularization is a refactoring activity — behavior must not change.

---

## Thresholds and triggers

### Hard thresholds (mandatory action)

| Metric                       | Threshold | Action Required                             |
| ---------------------------- | --------- | ------------------------------------------- |
| Total lines of code (LOC)    | > 500     | Must split before merging new functionality |
| Total lines of code (LOC)    | > 300     | Should split; mandatory if adding new logic |
| Exported symbols             | > 15      | Must split by export grouping               |
| Cyclomatic complexity (file) | > 20      | Must extract complex branches to modules    |
| Number of classes/components | > 3       | Must split to one-class/component-per-file  |
| Number of distinct concerns  | > 2       | Must split by concern boundary              |

### Soft thresholds (review and justify)

| Metric                    | Threshold | Review Action                                |
| ------------------------- | --------- | -------------------------------------------- |
| Total lines of code (LOC) | > 200     | Assess if natural split points exist         |
| Function/method count     | > 10      | Assess grouping by responsibility            |
| Import statements         | > 15      | Assess if file is a hub that should delegate |
| Scroll depth              | > 5 pages | Assess developer navigation cost             |

### When NOT to split

Do not split when:

- The file is a single cohesive unit (e.g., one complex algorithm, one state machine)
- Splitting would create circular dependencies
- Splitting would scatter closely related logic across 3+ files with no clear boundary
- The file is a barrel/index that re-exports from submodules (barrel files are exempt from LOC limits)
- The file is a test file mirroring a single source file (test files follow source structure)
- The file is generated code that will be overwritten

### Pre-split validation

Before splitting any file:

1. Verify the file actually exceeds a threshold (count LOC, exports, or complexity).
2. Identify natural seam lines — boundaries where concerns change.
3. Confirm the split will not introduce circular imports.
4. Confirm each resulting file will have a clear, single responsibility.
5. If no natural seams exist, the file may be legitimately large — document why and proceed.

---

## Split strategies

### Strategy 1: Split by responsibility

The most common and preferred strategy. Each output file owns one responsibility.

**When to use:** File contains multiple distinct concerns (e.g., data fetching + rendering + validation).

**Procedure:**

1. Identify distinct responsibilities by scanning top-level declarations.
2. Group related functions, types, and constants by responsibility.
3. Create one file per responsibility group.
4. Move shared types/constants to a `types.ts` or `constants.ts` file.
5. Update imports in all consumers.

**Example:**

```
# Before
user-service.ts (600 LOC)
  - User CRUD operations
  - Password hashing utilities
  - Email notification logic
  - Input validation schemas

# After
user-service.ts (200 LOC)        — CRUD orchestration only
user-password.ts (80 LOC)        — password hashing, comparison, policy
user-notifications.ts (120 LOC)  — email templates, send logic
user-validation.ts (100 LOC)     — input schemas, sanitization
user-types.ts (60 LOC)           — shared types, interfaces, constants
```

### Strategy 2: Split by abstraction layer

Separate interface from implementation, or separate layers of a pipeline.

**When to use:** File mixes abstraction levels (e.g., HTTP handler + database query + business logic).

**Procedure:**

1. Identify abstraction layers: interface → business logic → data access → infrastructure.
2. Create one file per layer.
3. Dependencies flow downward only (handler → service → repository).
4. Never allow lower layers to import from upper layers.

**Example:**

```
# Before
orders.ts (500 LOC)
  - Express route handlers
  - Order business rules
  - Database queries
  - Payment gateway calls

# After
orders-handler.ts (100 LOC)      — HTTP request/response only
orders-service.ts (150 LOC)      — business rules, orchestration
orders-repository.ts (120 LOC)   — database access
orders-payment.ts (80 LOC)       — payment gateway integration
orders-types.ts (50 LOC)         — shared types
```

### Strategy 3: Split by entity/domain

Separate code by the domain entity it operates on.

**When to use:** File handles multiple entities (e.g., users + teams + invitations in one file).

**Procedure:**

1. Identify distinct entities or domain objects.
2. Create one file per entity.
3. Extract shared utilities to a common file.
4. Maintain consistent naming: `{entity}-{concern}.ts`.

### Strategy 4: Split by lifecycle phase

Separate code by when it runs in the application lifecycle.

**When to use:** File mixes initialization, runtime, and cleanup code.

**Procedure:**

1. Identify lifecycle phases: setup → configuration → runtime → teardown.
2. Create files per phase: `{module}-setup.ts`, `{module}-runtime.ts`, `{module}-cleanup.ts`.
3. Entry point file orchestrates lifecycle by importing phases.

### Strategy 5: Extract utilities and helpers

Move generic, reusable functions to utility modules.

**When to use:** File contains helper functions that are not specific to its primary concern.

**Procedure:**

1. Identify functions that operate on generic inputs (strings, arrays, dates) rather than domain objects.
2. Move to `utils/` or `helpers/` directory with descriptive file names.
3. Do NOT create a single `utils.ts` catch-all — split utilities by domain:
   - `utils/string.ts`, `utils/date.ts`, `utils/validation.ts`
4. Utility files must be pure (no side effects, no state, no I/O).

### Strategy 6: Extract types and constants

Move type definitions and constants to dedicated files.

**When to use:** Types or constants are shared across 2+ files, or a file's type definitions exceed 100 LOC.

**Procedure:**

1. Create `{module}-types.ts` for interfaces, type aliases, and enums.
2. Create `{module}-constants.ts` for configuration values, magic numbers, and enum-like objects.
3. Co-locate types with their module — do NOT create a global `types/` directory unless types are truly cross-cutting.

---

## Directory organization after splits

### Flat structure (preferred for ≤ 5 files)

When a module splits into 5 or fewer files, keep them flat in the parent directory:

```
services/
├── user-service.ts
├── user-repository.ts
├── user-validation.ts
├── user-types.ts
└── order-service.ts
```

### Directory structure (required for > 5 files)

When a module splits into more than 5 files, create a subdirectory with an index:

```
services/
├── user/
│   ├── index.ts              — public API (re-exports)
│   ├── user-service.ts       — business logic
│   ├── user-repository.ts    — data access
│   ├── user-validation.ts    — input validation
│   ├── user-password.ts      — password utilities
│   ├── user-notifications.ts — email logic
│   └── user-types.ts         — types and constants
└── order/
    ├── index.ts
    └── ...
```

### Index/barrel file rules

1. Barrel files (`index.ts`) ONLY re-export — no logic, no side effects.
2. Barrel files are exempt from LOC thresholds.
3. Export only the public API — internal implementation stays non-exported.
4. Prefer named exports over default exports in barrel files.
5. Do not nest barrels more than 2 levels deep.

---

## Naming conventions after splits

### File naming

1. Use the module name as prefix: `{module}-{concern}.{ext}`.
2. Concern suffixes must be descriptive:
   - `-service`, `-handler`, `-controller` — orchestration layer
   - `-repository`, `-store`, `-dao` — data access layer
   - `-types`, `-interfaces` — type definitions
   - `-constants`, `-config` — static values
   - `-utils`, `-helpers` — pure utility functions
   - `-validation`, `-schema` — input validation
   - `-middleware` — request/response interceptors
   - `-factory`, `-builder` — object construction
   - `-adapter`, `-client` — external service integration
3. Follow project naming convention (kebab-case default per `monorepo-standards.md`).
4. Test files mirror source: `{module}-{concern}.test.{ext}` or `{module}-{concern}.spec.{ext}`.

### Symbol naming

1. After splitting, do NOT rename public symbols unless the rename improves clarity.
2. Internal symbols may be renamed to match their new file context.
3. Re-exported symbols keep their original names.

---

## Language-specific guidance

### TypeScript / JavaScript

1. One component per file for React/Vue/Svelte components.
2. Custom hooks in separate files: `use-{name}.ts`.
3. Context providers in separate files: `{name}-context.tsx`.
4. Avoid circular imports — use dependency injection or event-based communication.
5. Prefer named exports; use default exports only for components and pages.
6. Type-only imports (`import type { ... }`) to prevent circular dependency issues.
7. Co-locate component styles: `{component}.module.css` next to `{component}.tsx`.

```typescript
// GOOD: Clean module boundary
// user-service.ts
import type { User, CreateUserInput } from "./user-types.ts";
import { hashPassword } from "./user-password.ts";
import { UserRepository } from "./user-repository.ts";

export class UserService {
  constructor(private repo: UserRepository) {}
  async create(input: CreateUserInput): Promise<User> {
    /* ... */
  }
}

// BAD: God file with mixed concerns
// user.ts — 700 LOC with CRUD + auth + email + validation
```

### Python

1. One class per file when classes exceed 100 LOC.
2. `__init__.py` serves as barrel — re-export public API only.
3. Private modules prefixed with underscore: `_internal_helpers.py`.
4. Type stubs in separate `.pyi` files only when needed for third-party compatibility.
5. Use relative imports within a package; absolute imports across packages.
6. Dataclasses and Pydantic models in `models.py` or `schemas.py`.

```python
# GOOD: Package structure
# users/
#   __init__.py        — from .service import UserService
#   service.py         — business logic
#   repository.py      — database access
#   schemas.py         — Pydantic models
#   exceptions.py      — custom exceptions

# BAD: Single file
# users.py — 600 LOC with everything
```

### Go

1. One file per major type or interface.
2. Package-level organization — Go packages are already modular boundaries.
3. `_test.go` files mirror source files.
4. Internal packages (`internal/`) for implementation details.
5. Interface files: `{name}.go` with interface + constructor.
6. Implementation files: `{name}_{impl}.go` or grouped by backend.
7. Avoid `utils.go` — distribute helpers to the package that uses them.

```go
// GOOD: One responsibility per file
// user.go         — User type, interface
// user_service.go — UserService implementation
// user_store.go   — database operations
// user_http.go    — HTTP handlers

// BAD: Single file
// user.go — 800 LOC with types + handlers + DB + validation
```

### Rust

1. One module per file; `mod.rs` or `{name}.rs` at directory level.
2. Use `pub(crate)` for internal APIs, `pub` only for true public surface.
3. Trait definitions in separate files from implementations.
4. Error types in `error.rs` per module.
5. Builder patterns in `{type}_builder.rs`.

### Terraform / HCL

1. Split by resource group, not by resource type:
   - `networking.tf` — VPC, subnets, security groups
   - `compute.tf` — instances, autoscaling
   - `database.tf` — RDS, ElastiCache
   - `monitoring.tf` — CloudWatch, alerts
2. Variables in `variables.tf`, outputs in `outputs.tf`.
3. Provider configuration in `providers.tf` or `versions.tf`.
4. Backend configuration in `backend.tf`.
5. Modules for reusable infrastructure patterns.

### Configuration files (YAML, JSONC)

1. Split large config files by concern when they exceed 300 lines.
2. Use config composition (imports, includes, extends) where the format supports it.
3. Reference: this project's `config/` directory splits `base.jsonc`, `providers.jsonc`, `lsp.jsonc`.

---

## Test file modularization

### Mirroring rule

Test files follow the same modularization as source files:

1. When a source file splits, split its test file to match.
2. Each test file covers exactly one source file.
3. Test file naming mirrors source: `{source-name}.test.{ext}` or `{source-name}.spec.{ext}`.

### Test helper extraction

1. Shared test utilities go in `__tests__/helpers/` or `test/helpers/`.
2. Fixtures go in `__tests__/fixtures/` or `test/fixtures/`.
3. Factory functions go in `__tests__/factories/` or `test/factories/`.
4. Mock implementations go alongside their test files or in `__mocks__/`.

### When test files exceed thresholds

1. If a test file exceeds 500 LOC, split by test category:
   - `{module}.unit.test.ts` — unit tests
   - `{module}.integration.test.ts` — integration tests
   - `{module}.e2e.test.ts` — end-to-end tests
2. Alternatively, split by feature within the module:
   - `{module}-create.test.ts`
   - `{module}-update.test.ts`
   - `{module}-delete.test.ts`

---

## Circular dependency prevention

### Detection

1. Before committing a split, verify no circular imports exist.
2. Use tooling when available:
   - TypeScript: `madge --circular`
   - Python: `pydeps --no-show --no-config`
   - Go: compiler catches cycles natively
3. Manual check: trace import chains — if A → B → C → A, there is a cycle.

### Resolution strategies

1. **Extract shared types**: Move the shared dependency to a third file that both sides import.
2. **Dependency inversion**: Depend on abstractions (interfaces) rather than implementations.
3. **Event-based decoupling**: Use events or callbacks instead of direct imports.
4. **Merge back**: If two files are so tightly coupled that separation creates cycles, they belong together.

### Dependency direction rules

1. Handlers/Controllers → Services → Repositories → Database clients
2. Never reverse: Repositories must NOT import from Handlers.
3. Types flow sideways: any layer may import from `*-types` files.
4. Utils flow upward: any layer may import from `utils/` files.
5. Configuration flows downward: entry points configure, lower layers consume.

```
┌─────────────┐
│  Handlers    │  ← HTTP/CLI/Event entry points
└──────┬───────┘
       │ imports
┌──────▼───────┐
│  Services    │  ← Business logic, orchestration
└──────┬───────┘
       │ imports
┌──────▼───────┐
│ Repositories │  ← Data access, external APIs
└──────┬───────┘
       │ imports
┌──────▼───────┐
│   Clients    │  ← Database, HTTP, message queue
└──────────────┘

Types ← imported by any layer (sideways)
Utils ← imported by any layer (upward)
```

---

## Refactoring procedure

### Step-by-step execution

1. **Identify target**: File exceeding a hard threshold or flagged in review.
2. **Analyze seams**: Read the file and identify natural responsibility boundaries.
3. **Choose strategy**: Select the most appropriate split strategy from this document.
4. **Plan the split**: List output files with their responsibilities before writing code.
5. **Execute atomically**:
   - Create new files with extracted code.
   - Update the original file to import from new files.
   - Update all external consumers to import from new locations.
   - If barrel file needed, create it.
6. **Verify behavior preservation**:
   - Run `lsp_diagnostics` on all changed files.
   - Run existing tests — all must pass.
   - Run build/typecheck — must succeed.
   - Verify no circular dependencies introduced.
7. **Verify threshold compliance**: Confirm all output files are below thresholds.

### Atomic change rule

1. A modularization change must be a single commit.
2. The commit must not mix modularization with functional changes.
3. Commit message format: `refactor({module}): split {original} into {n} files`
4. If functional changes are also needed, commit modularization first, then functional changes.

### Rollback safety

1. Before starting, verify git working tree is clean.
2. If any verification step fails, revert all changes and reassess the split plan.
3. Never leave a partially-split file in the codebase.

---

## Integration with other rules

### Interaction with `monorepo-standards.md`

1. New files created by splits must follow kebab-case naming.
2. New directories created by splits must follow structure conventions.
3. Barrel files use `index.ts` (not `index.js` unless the project is JS-only).

### Interaction with `requirements-verification.md`

1. Modularization is a refactoring task — verify behavior preservation (no functional change).
2. Post-split verification follows the standard evidence checklist: diagnostics, tests, build.

### Interaction with `deployment-automation.md`

1. Modularization changes must pass CI before merge.
2. No special deployment consideration — modularization is code-only.

### Interaction with `hard-autonomy-no-questions.md`

1. When a file exceeds a hard threshold during implementation, split it immediately.
2. Do not ask whether to split — apply the policy and report what was done.
3. Choose the most obvious split strategy; document the choice in the commit message.

---

## Anti-patterns

### Premature splitting

- Splitting a 100-LOC file into 5 files of 20 LOC each.
- Creating a directory structure for a single small module.
- Splitting before the code has stabilized (initial prototyping phase).

### Shotgun splitting

- Moving every function to its own file regardless of cohesion.
- Creating files with only 1-2 exports that are always used together.
- Splitting by syntactic element (all interfaces in one file, all functions in another).

### False modularity

- Splitting a file but keeping all internal state shared via global/module-level variables.
- Creating circular dependencies between the split files.
- Barrel files that re-export everything, defeating the purpose of encapsulation.

### Over-abstraction

- Creating abstract base classes just to justify a split.
- Introducing unnecessary interfaces between tightly-coupled components.
- Adding dependency injection framework complexity for a 2-file module.

### Naming anti-patterns

- Generic names: `helpers.ts`, `utils.ts`, `misc.ts`, `common.ts` as catch-alls.
- Numbered files: `service1.ts`, `service2.ts` instead of descriptive names.
- Acronym-only names: `ush.ts` instead of `user-service-handler.ts`.

---

## Metrics and monitoring

### Codebase health indicators

Track these metrics at the project level to identify modularization debt:

1. **Files over 500 LOC**: Count should be 0 (excluding generated/vendored).
2. **Files over 300 LOC**: Count should be ≤ 5% of source files.
3. **Average file LOC**: Target 100-200 LOC for source files.
4. **Maximum file LOC**: Hard ceiling at 500 LOC for any source file.
5. **Circular dependency count**: Must be 0.

### When to schedule modularization work

1. **Proactive**: During feature work, if the file you're editing exceeds 300 LOC.
2. **Reactive**: During code review, if a reviewer flags file size.
3. **Systematic**: Monthly codebase health review for files approaching thresholds.
4. **Gate**: Before adding new functionality to a file already at 400+ LOC.

---

## Verification checklist

After any modularization change, confirm:

| #   | Check                                       | Evidence                                    |
| --- | ------------------------------------------- | ------------------------------------------- |
| 1   | All output files below 500 LOC              | `wc -l` on each file                        |
| 2   | No circular dependencies introduced         | Import trace or `madge --circular`          |
| 3   | All existing tests pass                     | Test runner output                          |
| 4   | Build/typecheck succeeds                    | Build command exit code 0                   |
| 5   | No new lint errors                          | `lsp_diagnostics` on changed files          |
| 6   | All external imports updated                | Grep for old import paths returns 0 results |
| 7   | Barrel file exports public API only         | Manual review of index file                 |
| 8   | File names follow naming convention         | `npm run lint:naming` or manual check       |
| 9   | Commit contains only modularization changes | `git diff --stat` review                    |
| 10  | Each output file has single responsibility  | File header or first docstring describes it |

---

## Reference composition

1. This rule is loaded as a Tier 1 baseline rule via `opencode.jsonc`.
2. For conflict resolution, follow priority order in `rules/README.md`.
3. This rule defers to `hard-autonomy-no-questions.md` on execution posture.
4. This rule defers to `monorepo-standards.md` on naming conventions.
