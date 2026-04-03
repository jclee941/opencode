# Autonomous Build Pipeline

Orchestrates spec-to-PR automation by composing existing OpenCode primitives
(Archon tasks, git worktrees, ralph-loop, Momus review, BMAD artifacts) into
a self-validating pipeline.

Priority: Tier 1 on-demand. Read when autonomous pipeline triggers are in scope.

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

## Pipeline overview

```
┌─────────┐    ┌─────────┐    ┌──────────┐    ┌───────┐    ┌────────┐    ┌───────┐
│  INGEST │───▶│  PLAN   │───▶│  BUILD   │───▶│  QA   │───▶│ MERGE  │───▶│  PR   │
└─────────┘    └─────────┘    └──────────┘    └───────┘    └────────┘    └───────┘
```

**Execution rules:** See `02-auto-build-pipeline-execution.md` (Build + QA phases)
**Completion rules:** See `03-auto-build-pipeline-completion.md` (Merge + PR phases)

## Phase 1 — Ingest (spec → task graph)

Load requirements from available sources in priority order:

| Priority | Source                      | Tool                                               |
| -------- | --------------------------- | -------------------------------------------------- |
| 1        | BMAD artifacts              | Glob `_bmad-output/`, normalize directly into task graph |
| 2        | Archon project tasks        | `find_tasks(filter_by="project", ...)`             |
| 3        | GitHub Issues               | `gh issue list` or `mcphub_github-list_issues`     |
| 4        | Inline spec (user-provided) | Parse from user message                            |

Rules:

1. Normalize all sources into Archon tasks if not already tracked.
2. Establish dependency order from task descriptions and feature groupings.
3. Identify parallelizable task clusters (no shared files, independent features).
4. Record the task graph in Kratos memory for session continuity.

## Phase 2 — Plan (task graph → execution plan)

1. Group tasks into parallelizable clusters based on file overlap analysis.
2. For each cluster, determine: branch name, worktree path, estimated scope.
3. Consult Metis for complex multi-task plans (3+ tasks with dependencies).
4. Submit plan to Momus for review before execution.
5. Create todo list reflecting the execution plan.

### Branch naming convention

```
auto/{feature-slug}        — feature branches
auto/{feature-slug}-{n}    — parallel sub-branches within a feature
```

### Parallelism rules

- Maximum 3 parallel worktree builds per pipeline run (context budget).
- Independent features with no shared files → parallel.
- Tasks sharing source files → sequential within same worktree.
- Infrastructure/config tasks → sequential, run first (base layer).

## Session limits

1. Maximum tasks per pipeline run: 10 (prevents context exhaustion).
2. Maximum parallel worktrees: 3.
3. Maximum QA retry cycles per task: 3.
4. Maximum total pipeline duration: defer to ralph-loop iteration limits.
5. If session context is degrading, complete current task, merge, and stop.

## Reference composition

1. Loaded as Tier 1 on-demand rule (not pre-loaded via `opencode.jsonc`).
2. Defers to `00-hard-autonomy-no-questions.md` on execution posture.
3. BMAD artifact ingestion is handled inline by this rule's ingest phase.
4. Defers to `00-archon-workflow.md` for task management.
5. See `02-auto-build-pipeline-execution.md` for Build + QA phase details.
6. See `03-auto-build-pipeline-completion.md` for Merge + PR phase details.
