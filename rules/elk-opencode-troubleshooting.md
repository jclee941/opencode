# ELK Troubleshooting For OpenCode Runtime

Apply this rule when the affected target is OpenCode runtime behavior, agent execution,
plugin orchestration, or background task lifecycle in this workspace.

Shared ELK behavior comes from `rules/elk-troubleshooting-global.md` (source of truth).
This file defines only OpenCode-specific deltas.

## Scope translation

1. `session scope`: single OpenCode session id (`sessionID`) and its child task ids.
2. `agent scope`: one or more agents (`sisyphus`, `hephaestus`, `atlas`, `explore`, `librarian`, `oracle`, `metis`, `momus`).
3. `runtime scope`: opencode core plus plugin/runtime layer (`oh-my-opencode`, `opencode-supermemory`, `opencode-antigravity-auth`).

## OpenCode-first evidence dimensions

Correlate with at least two dimensions, preferring these keys when present:

1. `@timestamp`
2. `sessionID` or `session_id`
3. `taskId` or `task_id`
4. `agent` / `subagent`
5. `event` / `status` / `level`
6. `message`

If key names differ by index template, map them via `mcphub_elk-get_mappings` before query tuning.

## OpenCode ELK integration baseline

1. Confirm Elasticsearch reachability and health (`mcphub_elk-elasticsearch_health`).
2. Discover candidate indices via `mcphub_elk-list_indices` and choose active OpenCode-related indices by recency.
3. Verify mappings for selected indices (`mcphub_elk-get_mappings`) and lock query fields to mapped types.
4. Validate fresh ingestion using bounded windows (`now-10m` to `now`) and small result sizes (`20-100`).
5. Confirm at least one event contains correlation keys for session + agent + severity.
6. If no fresh event exists, treat as observability incident and perform log-path enrollment before deeper root-cause work.

## OpenCode-specific blind spots

1. Session cancellation may appear as normal lifecycle (`cancel` + `idle`) and can be duplicated by design.
2. Background cancellation can happen with limited surface logs; absence of one log line is not proof of absence.
3. Time-window mismatch (timezone drift, too-wide/too-narrow window) can produce false no-hit results.
4. Index rollover can move recent events to a different backing index or data stream.
5. Partial auth can allow index listing but fail search-level reads.

## Integration completion criteria (OpenCode)

1. At least one fresh OpenCode event is queryable in the selected time window.
2. Event includes usable correlation keys for `session`, `agent`, and severity/error classification.
3. A bounded query reproduces the target symptom pattern (for example cancel/idle transition or stale-timeout interruption).
4. Post-change recheck after 10-30 minutes shows ingestion continuity.

## Response contract delta

In addition to global response contract, include:

1. Scope line: `session` | `agent` | `runtime`.
2. Selected index names and mapped field keys used for correlation.
3. Whether duplicate cancel/idle lines are interpreted as expected lifecycle or anomaly, with evidence.
