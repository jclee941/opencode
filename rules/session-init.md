# Session Initialization

Use this checklist at the start of every coding task.

## Scope

This rule applies to coding-session startup, repository context bootstrapping,
and initial verification before implementation begins.

## Inputs/constraints

- The active working directory determines the project context and must be resolved before project-scoped memory queries.
- MCP bootstrap steps are independent and should not block safe local work when a server is unavailable.
- Relevant files should be inspected before editing, and changes should stay scoped to the user request.

## Decision/rules

The mandatory startup behavior is defined by the checklist, MCP session bootstrap, failure policy,
and context persistence sections below.

1. Infer the request and expected output from available context, then execute.
2. Inspect relevant files before editing.
3. Follow existing project patterns and naming conventions.
4. Make the smallest safe change that solves the task.
5. Run targeted verification after changes (diagnostics/tests/build as applicable).
6. Report what changed, where it changed, and verification results.

## MCP session bootstrap

Run these steps at session start to establish project context before any task work.

### Tool call constraints

- MCP server tools (`mcphub_*`) are **external tools** and cannot be called via the `batch` tool.
- Call each MCP tool directly as a separate tool invocation.
- Multiple MCP calls that have no dependency on each other can be issued in the same response as parallel direct calls — but never inside a `batch` block.
- Non-MCP tools (`supermemory`, `bash`, `read`, `grep`, etc.) can be batched normally.

### MCP server inventory

All MCP servers are accessed via `mcphub` (remote at `192.168.50.112:3000`).

| Server       | Init Required     | Init Tool                        | Purpose                              |
| ------------ | ----------------- | -------------------------------- | ------------------------------------ |
| `git`        | Yes (per-session) | `mcphub_git-git_set_working_dir` | Git operations                       |
| `kratos`     | Yes (per-session) | `mcphub_kratos-project_switch`   | Project-scoped persistent memory     |
| `github`     | No                | —                                | GitHub API (repos, issues, PRs)      |
| `elk`        | No                | —                                | Elasticsearch queries and management |
| `proxmox`    | No                | —                                | VM/LXC/cluster management            |
| `supabase`   | No                | —                                | Database, edge functions, branches   |
| `terraform`  | No                | —                                | Provider/resource/module registry    |
| `slack`      | No                | —                                | Channel messaging, user search       |
| `playwright` | No                | —                                | Browser automation, screenshots      |

Non-MCP memory tools (built-in plugins, batchable):

| Tool          | Init Required | Purpose                                   |
| ------------- | ------------- | ----------------------------------------- |
| `supermemory` | No            | Cross-session memory (user/project scope) |

### CWD resolution

The placeholder `<cwd>` always refers to the **project directory the user opened OpenCode in** — the working directory shown in the environment context (e.g. `Working directory: /home/jclee/dev/myproject`).

- `<cwd>` is almost never `~/.config/opencode`. That path is the OpenCode config home, not a project.
- If the environment shows `Working directory: ~/.config/opencode`, that IS the correct CWD only when the user is explicitly working on OpenCode configuration itself.
- Derive `<project_name>` from the last path segment of `<cwd>` (e.g. `/home/jclee/dev/safework2` → `safework2`).

### Step 1: Project context (sequential — must complete before memory queries)

1. Switch Kratos to current working directory.
   - `mcphub_kratos-project_switch(project_path=<cwd>)`
   - Activates project-scoped memory for all subsequent `memory_search`/`memory_save` calls.
   - On failure (unknown project): **auto-create** the project, then retry:
     1. Run `mcphub_kratos-memory_save(summary="Project initialized: <project_name>", text="Auto-created project entry for <cwd>. CWD: <cwd>", tags=["project-init"])`
     2. Retry `mcphub_kratos-project_switch(project_path=<cwd>)`
     3. If retry also fails: proceed without Kratos — do not block session.

2. Set Git working directory for the session.
   - `mcphub_git-git_set_working_dir(path=<cwd>, validateGitRepo=true)`
   - Enables all subsequent `mcphub_git-*` calls without explicit `path` parameter.
   - Do not set `initializeIfNotPresent=true` — never auto-init a git repo.
   - If CWD is not a git repo: skip silently and note that GitHub Actions check (step 3) is also skipped.

3. Check latest GitHub Actions status for the tracked remote repository.
   - Run only when all conditions are met:
     - current directory is a Git repository,
     - `origin` remote points to GitHub,
     - `gh` CLI is available and authenticated.
   - Recommended command sequence:
     - `git remote get-url origin`
     - `gh run list --limit 5 --json status,conclusion,workflowName,headBranch,createdAt,url`
   - Record the latest run status in startup notes before proceeding.
   - If any condition is not met: skip this step and continue initialization.

### Step 2: Context retrieval (parallel direct calls — NOT batch)

Fire these as parallel direct tool calls in a single response. Do not use the `batch` tool.

4. Search Supermemory for prior decisions and project knowledge.
   - `supermemory(mode="search", query=<project_name or task keywords>, scope="project")`
   - Retrieves past architectural decisions, resolved issues, and learned patterns.

5. Search Kratos memory for project-specific context.
   - `mcphub_kratos-memory_search(q=<task keywords or module name>)`
   - Retrieves project-scoped memories (architecture notes, error solutions, conventions).

### Failure policy

- Each MCP step is independent. If one MCP server is unreachable, skip that step and proceed.
- Do not retry failed MCP connections — report the skip in task output and continue.
- Never block session initialization waiting for MCP responses.

### Context persistence

After completing significant work, persist key findings back to MCP:

- `mcphub_kratos-memory_save(summary=..., text=..., tags=[...])` — project-scoped knowledge.
- `supermemory(mode="add", content=..., type="learned-pattern", scope="project")` — cross-session patterns.
- Do not persist trivial or ephemeral information (single typo fixes, temp file paths).

## Verification

1. Verify project context is resolved before project-scoped memory queries run.
2. Verify relevant files are inspected before editing.
3. Verify targeted diagnostics, tests, or build checks run after significant changes.

## Rollback/safety

1. If an MCP bootstrap step fails, skip that step and continue without blocking safe local work.
2. Do not expose secrets or credentials in session output.
3. Keep edits focused on the requested scope so unexpected changes remain easy to revert.

General guardrails:
Do not expose secrets or credentials in output.
Keep edits focused on the requested scope.
