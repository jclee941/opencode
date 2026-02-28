# Autonomous Build Pipeline

Orchestrates spec-to-PR automation by composing existing OpenCode primitives
(Archon tasks, git worktrees, ralph-loop, Momus review, BMAD artifacts) into
a self-validating pipeline.

Priority: Tier 1 baseline. Loaded globally via `opencode.jsonc` instructions.

## Scope

**In scope:** Automated implementation pipelines triggered by explicit user command
(`/start-work`, `auto-build`, or direct request to "build from spec/tasks").
**Out of scope:** Manual coding sessions, exploratory research, single-file fixes.

## Activation

This rule activates when ANY of these conditions are met:

1. User explicitly requests autonomous build (`"auto-build"`, `"build all tasks"`,
   `"implement the spec"`, `"start the pipeline"`).
2. `/start-work` command is invoked with a Prometheus plan or Archon project.
3. BMAD artifacts are detected AND user requests full sprint execution.

When none of these conditions are met, skip all pipeline rules silently.

## Pipeline phases

```
┌─────────┐    ┌─────────┐    ┌──────────┐    ┌───────┐    ┌────────┐    ┌───────┐
│  INGEST │───▶│  PLAN   │───▶│  BUILD   │───▶│  QA   │───▶│ MERGE  │───▶│  PR   │
└─────────┘    └─────────┘    └──────────┘    └───────┘    └────────┘    └───────┘
```

### Phase 1 — Ingest (spec → task graph)

Load requirements from available sources in priority order:

| Priority | Source                      | Tool                                               |
| -------- | --------------------------- | -------------------------------------------------- |
| 1        | BMAD artifacts              | Glob `_bmad-output/`, follow `bmad-integration.md` |
| 2        | Archon project tasks        | `find_tasks(filter_by="project", ...)`             |
| 3        | GitHub Issues               | `gh issue list` or `mcphub_github-list_issues`     |
| 4        | Inline spec (user-provided) | Parse from user message                            |

Rules:

1. Normalize all sources into Archon tasks if not already tracked.
2. Establish dependency order from task descriptions and feature groupings.
3. Identify parallelizable task clusters (no shared files, independent features).
4. Record the task graph in Kratos memory for session continuity.

### Phase 2 — Plan (task graph → execution plan)

1. Group tasks into parallelizable clusters based on file overlap analysis.
2. For each cluster, determine: branch name, worktree path, estimated scope.
3. Consult Metis for complex multi-task plans (3+ tasks with dependencies).
4. Submit plan to Momus for review before execution.
5. Create todo list reflecting the execution plan.

#### Branch naming convention

```
auto/{feature-slug}        — feature branches
auto/{feature-slug}-{n}    — parallel sub-branches within a feature
```

#### Parallelism rules

- Maximum 3 parallel worktree builds per pipeline run (context budget).
- Independent features with no shared files → parallel.
- Tasks sharing source files → sequential within same worktree.
- Infrastructure/config tasks → sequential, run first (base layer).

### Phase 3 — Build (execution plan → code)

For each task cluster:

1. **Create worktree**: `worktree_make(action="create", name="{slug}", branch="auto/{slug}")`.
2. **Set Archon status**: `manage_task("update", task_id="...", status="doing")`.
3. **Delegate implementation**: Use `task()` with appropriate category and skills.
   - Frontend work → `category="visual-engineering"`, `load_skills=["frontend-ui-ux"]`
   - Backend/logic → `category="deep"` or `category="unspecified-high"`
   - Simple changes → `category="quick"`
4. **Track progress**: Update todos as each task completes.
5. **Verify per-task**: Diagnostics, tests, build on changed files.

#### Delegation prompt template

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

#### Build failure recovery

1. First failure: Analyze error, apply targeted fix, re-verify.
2. Second failure: Consult Oracle with full error context and prior attempts.
3. Third failure: Mark task as `blocked` in Archon, skip to next task, report.

### Phase 4 — QA (self-validating verification loop)

Run after each task cluster completes, before merge:

1. **Diagnostics**: `lsp_diagnostics` on every changed file — zero errors required.
2. **Tests**: Run project test suite (`npm test`, `pytest`, `go test`, etc.).
3. **Build**: Full project build must succeed (exit code 0).
4. **Lint**: Run project linter if configured.
5. **Type check**: Run type checker if configured (`tsc --noEmit`, `mypy`, etc.).
6. **Review**: Submit to Momus for automated code review on non-trivial changes.

#### QA failure handling

| Failure type    | Action                                     |
| --------------- | ------------------------------------------ |
| Diagnostics     | Fix immediately, re-run QA                 |
| Test failure    | Analyze, fix if caused by changes, re-run  |
| Build failure   | Fix, re-run full QA                        |
| Lint            | Auto-fix if possible, manual fix otherwise |
| Momus rejection | Address feedback, re-submit                |
| Pre-existing    | Document and exclude from QA gate          |

Maximum QA retry cycles: 3. After 3 failures, mark task blocked and continue.

### Phase 5 — Merge (worktree → main branch)

After QA passes for a task cluster:

1. **Rebase**: Rebase worktree branch onto latest main/target branch.
2. **Conflict resolution**: If conflicts arise during rebase:
   - Auto-resolve trivial conflicts (whitespace, import ordering).
   - For non-trivial conflicts: consult Oracle with both versions, apply resolution.
   - If Oracle cannot resolve: mark as blocked, report to user.
3. **Re-verify**: Run QA loop again after rebase to catch integration issues.
4. **Merge**: Fast-forward merge into target branch (no merge commits).
5. **Cleanup**: Remove worktree after successful merge.
6. **Update Archon**: Set task status to `review` or `done`.

#### Merge ordering

When multiple parallel branches are ready:

1. Merge infrastructure/config changes first.
2. Merge independent features in task-order priority.
3. Re-verify after each merge before proceeding to next.

### Phase 6 — PR (merged code → pull request)

After all task clusters are merged:

1. **Create PR**: Use `gh pr create` or `mcphub_github-create_pull_request`.
2. **PR body**: Include:
   - Summary of all implemented tasks with Archon task IDs.
   - Files changed per task cluster.
   - QA verification results (tests, build, diagnostics).
   - Any blocked tasks with reasons.
3. **Labels**: Add relevant labels based on task types.
4. **Update Archon**: Set all completed tasks to `done`.

#### PR skip conditions

- No remote configured → skip PR, report local completion only.
- User explicitly requested no PR → skip.
- All tasks blocked → no PR, report blockers only.

## Pipeline state tracking

Track pipeline state in Kratos memory for crash recovery:

```yaml
pipeline:
  id: "auto-{timestamp}"
  status: "running|completed|failed|partial"
  phase: "ingest|plan|build|qa|merge|pr"
  tasks:
    - id: "task-xxx"
      status: "pending|building|qa|merged|blocked"
      worktree: "auto/feature-slug"
      branch: "auto/feature-slug"
  blocked: []
  completed: []
```

Save state after each phase transition:
`mcphub_kratos-memory_save(summary="Pipeline state: {phase}", text="{yaml}", tags=["auto-pipeline"])`

## Session limits

1. Maximum tasks per pipeline run: 10 (prevents context exhaustion).
2. Maximum parallel worktrees: 3.
3. Maximum QA retry cycles per task: 3.
4. Maximum total pipeline duration: defer to ralph-loop iteration limits.
5. If session context is degrading, complete current task, merge, and stop.

## Interaction with other rules

1. BMAD integration: Pipeline consumes BMAD stories as task source (Phase 1).
2. Archon workflow: All tasks tracked in Archon throughout pipeline.
3. Code modularization: All generated code must pass modularization thresholds.
4. Deployment automation: Pipeline does not deploy — it produces a PR.
5. Requirements verification: Each task verified against its acceptance criteria.
6. Hard autonomy: Pipeline runs without questions; blocked steps use standard format.

## Verification checklist

| #   | Check                      | Evidence                                    |
| --- | -------------------------- | ------------------------------------------- |
| 1   | All tasks attempted        | Archon task status audit                    |
| 2   | QA passed for merged tasks | Test/build/diagnostic output per task       |
| 3   | No regressions introduced  | Full test suite passes on final branch      |
| 4   | Blocked tasks documented   | Blocker reason + next step per blocked task |
| 5   | Pipeline state persisted   | Kratos memory entry                         |
| 6   | PR created (if applicable) | PR URL or skip reason                       |
| 7   | Worktrees cleaned up       | `worktree_overview` shows no stale entries  |

## Reference composition

1. Loaded as Tier 1 baseline rule via `opencode.jsonc`.
2. Defers to `hard-autonomy-no-questions.md` on execution posture.
3. Defers to `bmad-integration.md` for BMAD artifact consumption.
4. Defers to `archon-workflow.md` for task management.
5. Defers to `requirements-verification.md` for acceptance criteria verification.
6. Defers to `code-modularization.md` for file size governance.
7. Defers to `deployment-automation.md` — pipeline produces PRs, not deployments.
