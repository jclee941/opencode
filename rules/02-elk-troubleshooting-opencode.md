# ELK Troubleshooting — OpenCode Runtime

OpenCode-specific ELK troubleshooting procedures.
Part of: `01-elk-troubleshooting.md` (split for modularity)

## Scope

Apply this section when the affected target is OpenCode runtime behavior, agent execution,
plugin orchestration, or background task lifecycle.

## OpenCode scope translation

1. `session scope`: single OpenCode session id (`sessionID`) and its child task ids.
2. `agent scope`: one or more agents (`sisyphus`, `hephaestus`, `explore`, `librarian`, `oracle`, `metis`, `momus`).
3. `runtime scope`: opencode core plus plugin/runtime layer (`oh-my-opencode`, `opencode-supermemory`).

## OpenCode evidence dimensions

Correlate with at least two dimensions, preferring these keys when present:

1. `@timestamp`
2. `sessionID` or `session_id`
3. `taskId` or `task_id`
4. `agent` / `subagent`
5. `event` / `status` / `level`
6. `message`

If key names differ by index template, map them via `mcphub_elk-get_mappings` before query tuning.

## OpenCode index pattern

- Active index: `logs-opencode-YYYY.MM.DD`
- ~370 docs/day typical volume

## OpenCode non-incident flow

1. Pre-check: define scope (`session` | `agent` | `runtime`), capture baseline from `logs-opencode-YYYY.MM.DD` for `now-30m` to `now`.
2. Change execution: prefer smallest reversible change first, keep blast radius to one scope when possible.
3. Post-check: verify fresh events still ingest, verify symptom-related fields remain queryable.

## OpenCode incident triage

1. ELK platform check (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
2. Data availability: identify active OpenCode index by date, verify mappings.
3. Incident query: bounded search with explicit time range, correlate by session + agent + severity/status.
4. Classification: session-level lifecycle pattern, agent-level execution/staleness, runtime/plugin-level gap.
5. Remediation: smallest reversible change first, include rollback note and stop-condition.

## OpenCode query template

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

## OpenCode blind spots

1. Session cancellation may appear as normal lifecycle (`cancel` + `idle`) and can be duplicated by design.
2. Background cancellation can happen with limited surface logs; absence of one log line is not proof of absence.
3. Time-window mismatch (timezone drift, too-wide/too-narrow window) can produce false no-hit results.

## OpenCode response contract

In addition to the base output format:

1. Scope line: `session` | `agent` | `runtime`.
2. Selected index names and mapped field keys used for correlation.
3. Whether duplicate cancel/idle lines are interpreted as expected lifecycle or anomaly, with evidence.

## OpenCode guardrails

1. Do not escalate from session-level to runtime-level without new evidence.
2. Treat duplicate cancel/idle lines as hypothesis until correlated with surrounding events.
