# Autonomous Build Pipeline — Completion Phases

Merge and PR phase implementation for the autonomous pipeline.
Part of: `auto-build-pipeline.md` (split for modularity)

## Phase 5 — Merge (worktree → main branch)

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

### Merge ordering

When multiple parallel branches are ready:

1. Merge infrastructure/config changes first.
2. Merge independent features in task-order priority.
3. Re-verify after each merge before proceeding to next.

## Phase 6 — PR (merged code → pull request)

After all task clusters are merged:

1. **Create PR**: Use `gh pr create` or `mcphub_github-create_pull_request`.
2. **PR body**: Include:
   - Summary of all implemented tasks with Archon task IDs.
   - Files changed per task cluster.
   - QA verification results (tests, build, diagnostics).
   - Any blocked tasks with reasons.
3. **Labels**: Add relevant labels based on task types.
4. **Update Archon**: Set all completed tasks to `done`.

### PR skip conditions

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

## Deployment note

Pipeline defers to `deployment-automation.md` — pipeline produces PRs, not deployments.
Deployment happens separately after PR review and merge.
