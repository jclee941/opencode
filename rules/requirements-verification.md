# Requirements Verification Policy
specifications are always checked before work starts and verified after work completes.
## Core rule
1. Before starting implementation, search for relevant requirement/specification documents.
2. If a requirements spec exists, validate every change against it.
3. If no requirements spec exists, state the assumption explicitly and proceed.
4. After implementation, run a completion check against the original requirements.
## Pre-implementation checklist
1. Search for requirement sources in this order:
   - `docs/` directory for specification or design documents.
   - `rules/` directory for applicable policy constraints.
   - Archon project management (`mcphub_archon-find_tasks`, `mcphub_archon-find_documents`).
   - Supermemory (`supermemory` search for prior decisions and context).
   - Issue tracker (GitHub Issues / PR descriptions).
2. If found, extract acceptance criteria and hard constraints before writing code.
3. If not found, document the inferred requirements as assumptions in the task output.
## During implementation
1. Cross-check each significant change against extracted requirements.
2. Flag any deviation from spec immediately with rationale.
3. Do not silently drop or skip a requirement; report it as blocked with reason.
## Post-implementation verification
1. Run a line-by-line completion check: every requirement item must have one of:
   - `done` — with evidence (file path, test result, command output).
   - `partial` — with explanation of what remains.
   - `blocked` — with reason and next step.
   - `N/A` — with justification.
2. Verification evidence must include all applicable items:
   - Diagnostics (`lsp_diagnostics` on changed files).
   - Tests (run if test suite exists).
   - Build/typecheck (run if build command exists).
3. Report spec compliance summary in final output.
## Verification output format

```
## Requirements Verification
| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | ...         | done   | ...      |
| 2 | ...         | blocked| ...      |
```

## Scope boundaries
1. This policy applies to implementation tasks, not pure research or exploration.
2. For trivial single-file changes with no existing spec, inline the inferred requirement
   in the final output instead of searching exhaustively.
3. Do not create new requirement documents unless explicitly requested;
   this rule governs checking and verification, not authoring.
## Requirement authoring standards

When writing or updating requirement/specification content:

1. Keep requirement content modular by responsibility; one file owns one topic.
2. Keep normative rules in `rules/`; explanatory standards in `docs/`.
3. Avoid duplicate normative text across files; use one canonical file and cross-reference.

### Structure contract

Every non-trivial requirement document includes these sections:

1. Scope — what is in/out.
2. Inputs/constraints — assumptions, dependencies, hard boundaries.
3. Decision/rules — mandatory behavior.
4. Verification — how completion is proven.
5. Rollback/safety notes — how to revert risky changes.

If a section is not applicable, state `N/A` explicitly.

### Checklist quality rules

1. Each item must be executable and testable with observable evidence.
2. Order: pre-check → execution → post-check.
3. Use imperative wording (`Run`, `Verify`, `Record`, `Update`).
4. Include explicit stop-conditions for high-risk steps.
5. Include expected evidence shape (command output, log query, lint/test result).

## Reference composition
1. This rule is loaded as a Tier 1 baseline rule via `opencode.jsonc`.
2. For conflict resolution, follow priority order in `rules/README.md`.
