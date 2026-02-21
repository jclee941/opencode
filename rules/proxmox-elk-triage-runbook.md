# Proxmox ELK Work Runbook

Use this runbook for all tasks touching personal Proxmox infrastructure, not only incident response.
This includes routine changes, feature rollouts, maintenance, and troubleshooting.

Shared ELK policy is inherited from `rules/elk-troubleshooting-global.md`.
This runbook focuses on Proxmox execution flow and templates.

## Default usage scope

1. Apply to any task that can affect service health, VM/CT behavior, node resources, storage, or network paths.
2. During non-incident work, use lightweight pre-check and post-check verification with ELK evidence.
3. During incidents, run the full triage sequence below.

## Non-incident default flow

1. Pre-check:
   - confirm target scope (`service` | `VM/CT` | `node` | `cluster`)
   - capture a 10-30m baseline from relevant indices
2. Change execution:
   - prefer smallest reversible change first
   - keep blast radius to one workload when possible
3. Post-check:
   - verify error trend did not regress
   - confirm ingestion/query path is healthy

## Incident brief format

1. Symptom: one-line user-visible impact.
2. Scope: `service` | `VM/CT` | `node` | `cluster`.
3. Time window: explicit timezone + range (for example `KST`, `now-30m` to `now`).
4. Blast radius: single workload, single host, or multi-host.

## Incident triage sequence

1. ELK platform check:
   - `mcphub_elk-elasticsearch_health`
   - `mcphub_elk-list_indices`
2. Data availability check:
   - pick candidate indices (`logs-*`, `<service>-logs-*`, `elastalert_status*`)
   - verify mappings with `mcphub_elk-get_mappings`
3. Incident query:
   - run bounded `mcphub_elk-search` with `queryBody`
   - correlate by host + service/module + severity + time
4. Classification:
   - workload issue (restart loop, app 4xx/5xx spike)
   - node issue (resource pressure, storage/network symptoms)
   - cluster issue (broad ingestion/query failures)
5. Remediation proposal:
   - smallest reversible change first
   - include rollback note and stop-condition

## Query templates

Use these as base payloads and adapt fields per mapping.

```json
{
  "size": 50,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-30m" } } },
        { "term": { "level": "error" } }
      ]
    }
  }
}
```

```json
{
  "size": 25,
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-15m" } } },
        { "term": { "service": "safework2-api" } }
      ]
    }
  }
}
```

## Guardrails

1. Inherit shared ELK guardrails from `rules/elk-troubleshooting-global.md`.
2. Do not escalate from workload-level to node/cluster-level without new evidence.
3. Treat `yellow` as potentially normal in single-node setups when primaries are healthy.

## Response contract

1. Inherit shared response contract from `rules/elk-troubleshooting-global.md`.
2. Add Proxmox scope line: `service` | `VM/CT` | `node` | `cluster`.
3. Step-by-step remediation must stay low blast radius first.
