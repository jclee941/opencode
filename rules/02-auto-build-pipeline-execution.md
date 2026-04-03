# Autonomous Build Pipeline — Execution Phases

Build and QA phase implementation for the autonomous pipeline.
Part of: `auto-build-pipeline.md` (split for modularity)

## Phase 3 — Build (execution plan → code)

For each task cluster:

1. **Create worktree**: `worktree_make(action="create", name="{slug}", branch="auto/{slug}")`.
2. **Set Archon status**: `manage_task("update", task_id="...", status="doing")`.
3. **Delegate implementation**: Use `task()` with appropriate category and skills.
   - Frontend work → `category="visual-engineering"`, `load_skills=["frontend-ui-ux"]`
   - Backend/logic → `category="deep"` or `category="unspecified-high"`
   - Simple changes → `category="quick"`
4. **Track progress**: Update todos as each task completes.
5. **Verify per-task**: Diagnostics, tests, build on changed files.

### Delegation prompt template

```
1. TASK: Implement {task_title} in worktree at {worktree_path}
2. EXPECTED OUTCOME: {acceptance_criteria} — all tests pass, diagnostics clean
3. REQUIRED TOOLS: read, write, edit, bash, grep, glob, lsp_diagnostics
4. MUST DO: Follow existing codebase patterns. Run diagnostics on all changed files.
   Match conventions in {reference_files}. Update imports if moving/adding files.
5. MUST NOT DO: Do not modify files outside {scope_boundary}.
   Do not suppress type errors. Do not create new dependencies without justification.
6. CONTEXT: Project: {project_name}. Architecture: {arch_notes}.
   Related files: {file_list}. Branch: auto/{slug}.
```

### Build failure recovery

1. First failure: Analyze error, apply targeted fix, re-verify.
2. Second failure: Consult Oracle with full error context and prior attempts.
3. Third failure: Mark task as `blocked` in Archon, skip to next task, report.

## Phase 4 — QA (self-validating verification loop)

Run after each task cluster completes, before merge:

1. **Diagnostics**: `lsp_diagnostics` on every changed file — zero errors required.
2. **Tests**: Run project test suite (`npm test`, `pytest`, `go test`, etc.).
3. **Build**: Full project build must succeed (exit code 0).
4. **Lint**: Run project linter if configured.
5. **Type check**: Run type checker if configured (`tsc --noEmit`, `mypy`, etc.).
6. **Review**: Submit to Momus for automated code review on non-trivial changes.

### QA failure handling

| Failure type    | Action                                     |
| --------------- | ------------------------------------------ |
| Diagnostics     | Fix immediately, re-run QA                 |
| Test failure    | Analyze, fix if caused by changes, re-run  |
| Build failure   | Fix, re-run full QA                        |
| Lint            | Auto-fix if possible, manual fix otherwise |
| Momus rejection | Address feedback, re-submit                |
| Pre-existing    | Document and exclude from QA gate          |

Maximum QA retry cycles: 3. After 3 failures, mark task blocked and continue.

## Interaction with other rules

1. BMAD integration: Pipeline consumes BMAD stories as task source (Phase 1).
2. Archon workflow: All tasks tracked in Archon throughout pipeline.
3. Code modularization: All generated code must pass modularization thresholds.
4. Requirements verification: Each task verified against its acceptance criteria.
5. Hard autonomy: Pipeline runs without questions; blocked steps use standard format.
