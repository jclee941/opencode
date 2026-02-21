# OpenCode ELK Work Runbook

Use this runbook for tasks affecting OpenCode runtime logs, agent lifecycle behavior,
or plugin-driven execution flow in this workspace.

Shared ELK policy is inherited from `rules/elk-troubleshooting-global.md`.
OpenCode-specific deltas are defined in `rules/elk-opencode-troubleshooting.md`.

## Default usage scope

1. Apply to incident and non-incident work when OpenCode logs are required for verification.
2. During non-incident work, run lightweight pre-check and post-check evidence capture.
3. During incidents, run the full triage sequence.

## Non-incident default flow

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

Use these as base payloads and adapt fields to actual mappings.

```json
{
  "size": 50,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-30m", "lte": "now" } } },
        {
          "terms": {
            "event.keyword": ["cancel", "idle", "stale timeout"]
          }
        }
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

## Guardrails

1. Inherit shared ELK guardrails from `rules/elk-troubleshooting-global.md`.
2. Inherit OpenCode-specific deltas from `rules/elk-opencode-troubleshooting.md`.
3. Do not escalate from session-level to runtime-level without new evidence.
4. Treat duplicate cancel/idle lines as hypothesis until correlated with surrounding events.

## Response contract

1. Inherit shared response contract from `rules/elk-troubleshooting-global.md`.
2. Add OpenCode scope line: `session` | `agent` | `runtime`.
3. Include index + field mapping evidence used for each claim.
