# Requirements Modularization and Checklist Rules

Apply this policy when writing or updating requirement/specification content in this repository.

## Core principles

1. Keep requirement content modular by responsibility; one file should own one topic.
2. Keep normative rules in `rules/`; keep explanatory standards/examples in `docs/`.
3. Avoid duplicate normative text across files; use one canonical file and cross-reference from others.
4. Prefer small, reversible edits to requirements so diffs stay reviewable.

## Requirement structure contract

Every non-trivial requirement document should include these sections in order:

1. Scope: what is in/out.
2. Inputs/constraints: assumptions, dependencies, and hard boundaries.
3. Decision/rules: mandatory behavior.
4. Verification: how completion is proven.
5. Rollback/safety notes: how to revert risky changes.

If a section is not applicable, state `N/A` explicitly instead of omitting it.

## Modularization rules

1. Split by domain first, then by lifecycle phase (for example: policy vs runbook vs troubleshooting).
2. Keep each file single-responsibility and reference canonical siblings instead of copying content.
3. When introducing a new rule file, update inbound references in the same change.
4. Use kebab-case filenames for regular docs/rules; reserve uppercase names for standard contract files only.

## Checklist authoring rules

1. Checklists must be executable and testable: each item is an action with observable evidence.
2. Order items as pre-check -> execution -> post-check.
3. Use concise imperative wording (for example: `Run`, `Verify`, `Record`, `Update`).
4. Include explicit stop-conditions for high-risk steps.
5. Include expected evidence shape (for example: command output, log query, lint/test result).

## Checklist completion criteria

A checklist is complete only when:

1. Every item has evidence or a documented `blocked` reason.
2. Verification items (diagnostics/tests/build where applicable) have pass/fail outcomes recorded.
3. Cross-references and impacted file paths are updated and valid.

## Reference composition

1. Use `rules/instructions-reference-composition.md` as the canonical composition guide.
2. Keep task-time references minimal: include this file only when requirement/checklist work is in scope.
3. Resolve conflicts with this priority: `hard-autonomy-no-questions.md` -> domain rule -> `session-init.md` -> repository standards.
