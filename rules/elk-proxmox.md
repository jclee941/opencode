# ELK Troubleshooting — Personal Proxmox

Apply when the request involves troubleshooting services, VMs, containers, networking,
storage, or automation running on personal Proxmox infrastructure.

Shared ELK behavior comes from `rules/elk-troubleshooting-global.md` (source of truth).
This file defines only Proxmox-specific deltas and execution flow.

For destructive or irreversible operations, follow `rules/hard-autonomy-no-questions.md`.

## Environment baseline

If any item is unknown, treat it as a hypothesis and verify via ELK/Proxmox evidence before acting.

1. MCP endpoint is configured through `mcphub` in `opencode.jsonc`.
2. ELK cluster may run as single-node homelab deployment.
3. `yellow` cluster/index health can be expected in single-node mode when replicas are configured.
4. Treat missing Proxmox-native fields (`vmid`, `ctid`, `node`) as a logging coverage gap, not immediate infra failure.
5. Current logs indicate mixed platform/application ingestion:
   - infra-style stream in `logs-*` (for example Ceph/system logs)
   - app stream in `safework2-logs-*` (HTTP/request-centric fields)
6. Host identifiers observed in logs can differ by source format:
   - `host.hostname` (for example `pve3`)
   - `host_name` / `host.name` aliases

## Default usage

1. Apply to any task that can affect service health, VM/CT behavior, node resources, storage, or network paths.
2. During non-incident work, use lightweight pre-check and post-check verification with ELK evidence.
3. During incidents, run the full triage sequence.

## Non-incident flow

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

## Baseline capture checklist

1. Record cluster health summary (status, active primaries, unassigned shards).
2. Record top candidate indices for the incident window.
3. Record one representative host identifier and one representative service/module key.
4. Record one representative failing log line/event and timestamp.
5. Only then propose remediation sequence.

## Data source prioritization

1. Discover cluster/index health first (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
   - Include ingest sanity checks (write blocks, rejection/backpressure signals, disk watermark symptoms).
2. Prioritize indices by intent:
   - platform/runtime logs: `logs-*`
   - app/service logs: `<service>-logs-*` (for example `safework2-logs-*`)
   - alert pipeline diagnostics: `elastalert_status*`
3. Confirm mappings for selected index before querying (`mcphub_elk-get_mappings`).
4. Use existing mapped fields first; do not assume field names between indices are identical.

## ELK correlation keys

Prefer these mapped keys for queries:

- time: `@timestamp` or `timestamp`
- host: `host.name`, `host.hostname`, `host`, `fields.host_name`
- service/module: `service`, `module`, `container_name`
- severity/error: `level`, `error_severity`, `error.*`

## Query templates

Adapt fields per mapping. If index uses `timestamp` instead of `@timestamp`, adapt sort/range accordingly.

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

## Proxmox infrastructure remediation

1. Translate log evidence into Proxmox scope explicitly:
   - node-level symptom (host saturation, storage, networking)
   - VM/CT-level symptom (restart loop, OOM, service crash)
   - shared-service symptom (DNS, reverse proxy, logging pipeline)
2. Apply remediation in this order:
   - single service in one VM/CT
   - single VM/CT
   - single node
   - cluster-wide change (last)
3. For personal homelab reliability, prefer low-blast-radius actions:
   - restart one workload before rebooting a node
   - scale down noisy ingestion before changing storage settings
   - isolate one index pattern before template-wide changes
4. Use host identifiers from logs to map symptoms back to Proxmox node/guest scope before proposing fixes.

## Proxmox-focused priorities

1. Check these domains in order:
   - cluster/node health
   - storage latency/capacity
   - VM/CT restart loops and OOM signals
   - network reachability and DNS failures
   - backup/snapshot failures
2. Flag high-risk operations before execution:
   - mass restart, storage repair, index deletion, forced reindex
   - stop-condition: if rollback is not defined and validated, do not execute the high-risk step.
3. Prefer staged fixes (single node or single workload first), then expand if validated.
4. If issue is likely replica-related in single-node ELK (`yellow` only, primaries active), avoid unnecessary recovery actions.
5. If restart loops are detected, classify as workload-level first unless node-wide symptoms are also present.

## Completion criteria

1. Fresh events are queryable for the affected target.
2. Events contain correlation keys needed for triage.
3. Bounded error query reproduces symptom signal.
4. Post-change recheck confirms continued ingestion.

## Response contract

In addition to `rules/elk-troubleshooting-global.md` response format:

1. Infrastructure scope line: `service` | `VM/CT` | `node` | `cluster`.
2. Evidence snapshot with index, host key used, service/module key used, time window.
3. Step-by-step remediation must stay low blast radius first.

## Guardrails

1. Inherit shared ELK guardrails from `rules/elk-troubleshooting-global.md`.
2. Do not escalate from workload-level to node/cluster-level without new evidence.
3. Treat `yellow` as potentially normal in single-node setups when primaries are healthy.
