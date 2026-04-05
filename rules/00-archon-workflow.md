# Archon Integration & Workflow

Apply this policy to all task management and project knowledge operations.
Archon MCP server is the primary system for task tracking, project organization,
and knowledge base access.

Priority: Tier 0 baseline. Loaded globally via `opencode.jsonc` instructions.

## Core rule

Archon MCP server is the **sole** task management system. Do not use IDE-native
task tracking, local todo files, or ad-hoc tracking mechanisms. All task state
lives in Archon.

## Task-driven development cycle

Follow this cycle for every coding task:

1. **Get task** — `find_tasks(task_id="...")` or `find_tasks(filter_by="status", filter_value="todo")`.
2. **Start work** — `manage_task("update", task_id="...", status="doing")`.
3. **Research** — Search knowledge base (see RAG workflow below) before implementation.
4. **Implement** — Write code based on research and task requirements.
5. **Review** — `manage_task("update", task_id="...", status="review")`.
6. **Complete** — `manage_task("update", task_id="...", status="done")` after verification.
7. **Next task** — `find_tasks(filter_by="status", filter_value="todo")`.

Status flow: `todo` → `doing` → `review` → `done`.

## Task granularity

1. Each task represents 30 minutes to 4 hours of work.
2. For feature-specific projects: create detailed implementation tasks (setup, implement, test, document).
3. For codebase-wide projects: create feature-level tasks.
4. Higher `task_order` = higher priority (0–100).

## RAG workflow (research before implementation)

### Searching specific documentation

1. Get sources — `rag_get_available_sources()` returns list with id, title, url.
2. Find source ID — match to documentation (e.g., "Supabase docs" → `src_abc123`).
3. Search — `rag_search_knowledge_base(query="vector functions", source_id="src_abc123")`.
4. Read full page — `rag_read_full_page(page_id="...")` for complete content.

### General research

1. `rag_search_knowledge_base(query="authentication JWT", match_count=5)` — 2–5 keywords only.
2. `rag_search_code_examples(query="React hooks", match_count=3)` — find code examples.

### Query quality

1. Keep queries SHORT: 2–5 keywords for best results.
2. Good: `"vector search"`, `"authentication JWT"`, `"React hooks"`.
3. Bad: `"how to implement user authentication with JWT tokens in React"`.

## RAG source management

### Adding documentation sources

Sources are managed via the Archon UI (`archon.jclee.me` / `192.168.50.108:3737`).

1. Navigate to **Knowledge Base** → **Sources** in the Archon UI.
2. Add a URL (documentation site root, e.g., `https://docs.example.com`).
3. The crawl pipeline processes: URL → crawl pages → chunk text → embed via Ollama (`nomic-embed-text`, 768-dim) → store vectors in Supabase.

### Crawl pipeline details

- Embedding model: `nomic-embed-text:latest` on Ollama at `192.168.50.109:11434`.
- Vector dimensions: 768.
- Storage: Supabase tables `archon_crawled_pages`, `archon_page_metadata`, `archon_code_examples`.
- Crawl status is visible in the Archon UI under each source.

### Linking sources to projects

1. Sources exist independently of projects.
2. Link a source to a project via `archon_project_sources` table (managed through UI).
3. When searching with `source_id`, filter is applied at query time.

### Troubleshooting crawl failures

1. Check Archon server logs: `docker logs archon-server` on LXC 108.
2. Zombie crawl processes can stall the pipeline — restart with `docker compose restart archon-server`.
3. Verify Ollama connectivity: `curl http://192.168.50.109:11434/api/embeddings -d '{"model":"nomic-embed-text","prompt":"test"}'`.
4. Stale crawl data can be cleaned via Supabase SQL against `archon_crawled_pages`.

## Project workflows

### New project

1. `manage_project("create", title="...", description="...")`.
2. `manage_task("create", project_id="...", title="...", task_order=N)` for each task.

### Existing project

1. `find_projects(query="...")` or `find_projects()` to list all.
2. `find_tasks(filter_by="project", filter_value="proj-123")` to get tasks.
3. Continue work or create new tasks.

### Document management

1. `find_documents(project_id="...")` — list project documents.
2. `manage_document("create", project_id="...", title="...", document_type="spec|design|note|prp|api|guide")`.
3. Documents support structured JSON content and tags for categorization.

## Tool reference

### Projects

- `find_projects(query="...")` — search projects.
- `find_projects(project_id="...")` — get specific project with full details.
- `manage_project("create"/"update"/"delete", ...)` — manage projects.
- `get_project_features(project_id="...")` — get project feature tracking.

### Tasks

- `find_tasks(query="...")` — search tasks by keyword.
- `find_tasks(task_id="...")` — get specific task with full details.
- `find_tasks(filter_by="status"/"project"/"assignee", filter_value="...")` — filter tasks.
- `manage_task("create"/"update"/"delete", ...)` — manage tasks.

### Documents

- `find_documents(project_id="...", document_type="spec|design|note|prp|api|guide")` — list/filter project documents.
- `manage_document("create"/"update"/"delete", ...)` — manage documents.

### Knowledge base

- `rag_get_available_sources()` — list all crawled sources.
- `rag_list_pages_for_source(source_id="...")` — browse source page structure.
- `rag_search_knowledge_base(query="...", source_id="...")` — search docs.
- `rag_search_code_examples(query="...", source_id="...")` — find code.
- `rag_read_full_page(page_id="...")` — read full page content.

### Versions

- `find_versions(project_id="...")` — list version history.
- `manage_version("create"/"restore", project_id="...", field_name="docs|features|data|prd")` — manage versions.

## Infrastructure reference

| Component | Address | Role |
|-----------|---------|------|
| Archon UI | `192.168.50.108:3737` | Web frontend |
| Archon Server | `192.168.50.108:8181` | FastAPI + Socket.IO |
| Archon MCP | `192.168.50.108:8051` | MCP tool server |
| Archon Agents | `192.168.50.108:8052` | Agent execution |
| Ollama | `192.168.50.109:11434` | LLM + embedding inference |
| Supabase | `supabase.jclee.me` | Vector storage + settings |
| MCPhub | See `config/base.jsonc` | MCP server aggregation |

### Known limitations

- Archon supports only: openai, google, openrouter, ollama provider types.
- `archon-agent-work-orders` container requires Docker profile `work-orders` (not running by default).
- `GITHUB_PAT_TOKEN` env var is optional — warning on restart if unset, does not affect core functionality.

## Enforcement

1. Never skip task status updates during implementation.
2. Never start coding without checking current tasks first.
3. Never use alternative task tracking when Archon is available.
4. If Archon MCP is unreachable, report the skip and proceed — do not block work.

## Reference composition

1. Loaded as Tier 0 baseline rule via `opencode.jsonc`.
2. Defers to `00-hard-autonomy-no-questions.md` on execution posture.
3. Defers to `00-session-init.md` for MCP bootstrap sequence.
