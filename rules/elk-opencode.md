# ELK Troubleshooting — OpenCode Runtime

Apply when the affected target is OpenCode runtime behavior, agent execution,
plugin orchestration, or background task lifecycle in this workspace.

Shared ELK behavior comes from `rules/elk-troubleshooting-global.md` (source of truth).
This file defines only OpenCode-specific deltas and execution flow.

## Scope translation

1. `session scope`: single OpenCode session id (`sessionID`) and its child task ids.
2. `agent scope`: one or more agents (`sisyphus`, `hephaestus`, `atlas`, `explore`, `librarian`, `oracle`, `metis`, `momus`).
3. `runtime scope`: opencode core plus plugin/runtime layer (`oh-my-opencode`, `opencode-supermemory`, `opencode-antigravity-auth`).

## Evidence dimensions

Correlate with at least two dimensions, preferring these keys when present:

1. `@timestamp`
2. `sessionID` or `session_id`
3. `taskId` or `task_id`
4. `agent` / `subagent`
5. `event` / `status` / `level`
6. `message`

If key names differ by index template, map them via `mcphub_elk-get_mappings` before query tuning.

## Default usage

1. Apply to incident and non-incident work when OpenCode logs are required for verification.
2. During non-incident work, run lightweight pre-check and post-check evidence capture.
3. During incidents, run the full triage sequence.

## Non-incident flow

1. Pre-check:
   - define scope (`session` | `agent` | `runtime`)
   - capture baseline from OpenCode-related index for `now-30m` to `now`
2. Change execution:
   - prefer smallest reversible change first
   - keep blast radius to one scope when possible
3. Post-check:
   - verify fresh events still ingest
   - verify symptom-related fields remain queryable

## Incident brief format

1. Symptom: one-line user-visible impact.
2. Scope: `session` | `agent` | `runtime`.
3. Time window: explicit timezone + range (for example `KST`, `now-30m` to `now`).
4. Blast radius: single session, single agent, or multi-agent/runtime.

## Incident triage sequence

1. ELK platform check:
   - `mcphub_elk-elasticsearch_health`
   - `mcphub_elk-list_indices`
2. Data availability check:
   - identify active OpenCode candidate indices by recency
   - verify mappings with `mcphub_elk-get_mappings`
3. Incident query:
   - run bounded `mcphub_elk-search` with explicit time range
   - correlate by session + agent + severity/status
4. Classification:
   - session-level lifecycle pattern
   - agent-level execution/staleness pattern
   - runtime/plugin-level observability gap
5. Remediation proposal:
   - smallest reversible change first
   - include rollback note and stop-condition

## Query templates

Adapt fields to actual mappings discovered via `mcphub_elk-get_mappings`.

```json
{
  "size": 50,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-30m", "lte": "now" } } },
        { "terms": { "event.keyword": ["cancel", "idle", "stale timeout"] } }
      ]
    }
  }
}
```

```json
{
  "size": 25,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-15m", "lte": "now" } } },
        { "term": { "agent.keyword": "hephaestus" } }
      ]
    }
  }
}
```

## Blind spots

1. Session cancellation may appear as normal lifecycle (`cancel` + `idle`) and can be duplicated by design.
2. Background cancellation can happen with limited surface logs; absence of one log line is not proof of absence.
3. Time-window mismatch (timezone drift, too-wide/too-narrow window) can produce false no-hit results.
4. Index rollover can move recent events to a different backing index or data stream.
5. Partial auth can allow index listing but fail search-level reads.

## Completion criteria

1. At least one fresh OpenCode event is queryable in the selected time window.
2. Event includes usable correlation keys for `session`, `agent`, and severity/error classification.
3. A bounded query reproduces the target symptom pattern.
4. Post-change recheck after 10-30 minutes shows ingestion continuity.

## Response contract

In addition to `rules/elk-troubleshooting-global.md` response format:

1. Scope line: `session` | `agent` | `runtime`.
2. Selected index names and mapped field keys used for correlation.
3. Whether duplicate cancel/idle lines are interpreted as expected lifecycle or anomaly, with evidence.

## Guardrails

1. Inherit shared ELK guardrails from `rules/elk-troubleshooting-global.md`.
2. Do not escalate from session-level to runtime-level without new evidence.
3. Treat duplicate cancel/idle lines as hypothesis until correlated with surrounding events.
