# Code Modularization Reference

Detailed strategies, language-specific guidance, and examples for code modularization.
Executive rules and thresholds are in `rules/code-modularization.md`.

## Split strategies

### 1. Extract by concern

Split when a file handles multiple distinct responsibilities.

```
// Before: user-controller.ts (handles auth + CRUD + validation)
// After:
//   user-auth.ts        — login, logout, token refresh
//   user-crud.ts        — create, read, update, delete
//   user-validation.ts  — input schemas, validators
```

### 2. Extract by layer

Split when business logic mixes with infrastructure.

```
// Before: order-service.ts (business logic + DB queries + HTTP calls)
// After:
//   order-service.ts     — business orchestration only
//   order-repository.ts  — database access
//   order-client.ts      — external API calls
```

### 3. Extract types/interfaces

Split when type definitions grow beyond 50 lines or are shared across files.

```
// Before: api-handler.ts (types + handler logic mixed)
// After:
//   api-handler.ts  — handler implementation
//   api-types.ts    — request/response types, DTOs
```

### 4. Extract constants/config

Split when magic values or config objects clutter business logic.

```
// Before: payment.ts (constants + logic)
// After:
//   payment.ts           — processing logic
//   payment-constants.ts — fee rates, limits, status enums
```

### 5. Extract utilities

Split when helper functions are reusable across modules.

```
// Before: report-generator.ts (formatting + generation + utils)
// After:
//   report-generator.ts  — report assembly
//   format-utils.ts      — date formatting, number formatting (shared)
```

### 6. Component decomposition (frontend)

Split when a React/Vue/Svelte component exceeds 200 lines or renders 3+ distinct UI sections.

```
// Before: Dashboard.tsx (sidebar + chart + table + filters)
// After:
//   Dashboard.tsx        — layout composition
//   DashboardSidebar.tsx — navigation
//   DashboardChart.tsx   — chart rendering
//   DashboardTable.tsx   — data table
//   DashboardFilters.tsx — filter controls
```

## Directory organization patterns

### Feature-based (preferred for applications)

```
src/
  features/
    auth/
      auth-service.ts
      auth-controller.ts
      auth-types.ts
      auth.test.ts
    orders/
      order-service.ts
      order-repository.ts
      order-types.ts
      order.test.ts
  shared/
    utils/
    types/
```

### Layer-based (acceptable for small projects)

```
src/
  controllers/
  services/
  repositories/
  types/
  utils/
```

## Language-specific guidance

### TypeScript / JavaScript

- One exported class per file (exception: closely related small classes).
- Barrel files (`index.ts`) exempt from LOC limits but should only re-export.
- Prefer named exports over default exports for grep-ability.
- Keep React hooks in separate files when they exceed 30 lines.
- Colocate test files next to source files (`foo.ts` → `foo.test.ts`).

### Python

- One class per module when class exceeds 200 lines.
- Use `__init__.py` as barrel files only (re-exports).
- Keep Django views/viewsets in separate files per resource.
- Separate models, serializers, and views into distinct files.

### Go

- Group by package responsibility, not by type.
- One file per major type and its methods.
- Keep `_test.go` files alongside source files.
- Avoid `utils.go` catch-all; name by specific utility purpose.

### Rust

- One module per major type or trait implementation.
- Use `mod.rs` or file-based modules for sub-grouping.
- Keep trait definitions separate from implementations when shared.

### Terraform / IaC

- Split by resource group: `network.tf`, `compute.tf`, `storage.tf`.
- Keep variables in `variables.tf`, outputs in `outputs.tf`.
- One module per logical infrastructure component.
- Keep environment-specific values in `.tfvars` files.

### YAML / Kubernetes

- One resource per file for complex resources.
- Group related simple resources (ConfigMap + Secret) in one file.
- Use kustomize overlays for environment splits.

## Circular dependency prevention

### Detection

```
# JavaScript/TypeScript
npx madge --circular src/

# Python
pylint --disable=all --enable=cyclic-import src/

# Manual: trace imports and look for A → B → A patterns
```

### Resolution patterns

1. **Extract shared types**: If A imports types from B and B imports types from A, extract shared types to C.
2. **Dependency inversion**: Have both A and B depend on an interface defined in a third module.
3. **Event-based decoupling**: Replace direct imports with event emitters or message passing.

### Dependency direction rule

```
Handlers → Services → Repositories → Clients
    ↑           ↑           ↑
  Types ← ← ← ← ← ← ← ← ←
  Utils ← ← ← ← ← ← ← ← ←
```

Downward-only flow. Types and Utils may be imported from any layer.

## Refactoring procedure

1. Identify the split boundary (concern, layer, type, or component).
2. Create target files with clear names reflecting their responsibility.
3. Move code in atomic commits (one logical move per commit).
4. Update all imports/references in the same commit.
5. Run full test suite after each move.
6. Verify no circular dependencies introduced.
7. Commit message format: `refactor({module}): split {original} into {n} files`

## Metrics and monitoring

### File health indicators

| Metric              | Healthy | Warning | Critical |
| ------------------- | ------- | ------- | -------- |
| LOC                 | < 200   | 200-500 | > 500    |
| Exported symbols    | < 10    | 10-15   | > 15     |
| Cyclomatic complex. | < 10    | 10-20   | > 20     |
| Import count        | < 10    | 10-15   | > 15     |
| Function count      | < 8     | 8-10    | > 10     |

### Automated checks

```bash
# LOC check
find src/ -name '*.ts' -exec wc -l {} + | sort -rn | head -20

# Circular dependency check
npx madge --circular --extensions ts src/

# Export count
grep -c '^export ' src/**/*.ts | sort -t: -k2 -rn | head -20
```
