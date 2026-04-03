# ELK Troubleshooting — Proxmox Infrastructure

Proxmox-specific ELK troubleshooting procedures.
Part of: `01-elk-troubleshooting.md` (split for modularity)

## Scope

Apply this section when the request involves troubleshooting services, VMs, containers,
networking, storage, or automation running on personal Proxmox infrastructure.

For destructive or irreversible operations, follow `rules/00-hard-autonomy-no-questions.md`.

## Proxmox environment baseline

If any item is unknown, treat it as a hypothesis and verify via ELK/Proxmox evidence before acting.

1. MCP endpoint is configured through `mcphub` in `opencode.jsonc`.
2. ELK cluster may run as single-node homelab deployment.
3. `yellow` cluster/index health can be expected in single-node mode when replicas are configured.
4. Treat missing Proxmox-native fields (`vmid`, `ctid`, `node`) as a logging coverage gap, not immediate infra failure.
5. Current logs use service-scoped index naming:
   - service-scoped indices: `logs-{service}-YYYY.MM.DD` (e.g. `logs-system-2026.02.22`, `logs-docker-2026.02.22`)
   - legacy flat index: `logs-YYYY.MM.DD` (being phased out, some services still write here)
   - app-specific: `safework2-logs-*` (HTTP/request-centric fields, may not exist daily)
6. Host identifiers observed in logs can differ by source format:
   - `host.hostname` (for example `pve3`)
   - `host_name` / `host.name` aliases

## Proxmox non-incident flow

1. Pre-check: confirm target scope (`service` | `VM/CT` | `node` | `cluster`), capture a 10-30m baseline from relevant indices.
2. Change execution: prefer smallest reversible change first, keep blast radius to one workload when possible.
3. Post-check: verify error trend did not regress, confirm ingestion/query path is healthy.

## Proxmox incident triage

1. ELK platform check (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
2. Data availability: pick candidate indices (`logs-{service}-YYYY.MM.DD` preferred, `logs-YYYY.MM.DD` legacy fallback, `elastalert_status*`), verify mappings.
3. Incident query: bounded search, correlate by host + service/module + severity + time.
4. Classification: workload issue (restart loop, app 4xx/5xx spike), node issue (resource pressure, storage/network), cluster issue (broad ingestion/query failures).
5. Remediation: smallest reversible change first, include rollback note and stop-condition.

## Proxmox baseline capture checklist

1. Record cluster health summary (status, active primaries, unassigned shards).
2. Record top candidate indices for the incident window.
3. Record one representative host identifier and one representative service/module key.
4. Record one representative failing log line/event and timestamp.
5. Only then propose remediation sequence.

## Proxmox data source prioritization

1. Discover cluster/index health first (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
   - Include ingest sanity checks (write blocks, rejection/backpressure signals, disk watermark symptoms).
2. Prioritize indices by intent:
   - service-scoped logs: `logs-{service}-YYYY.MM.DD` (preferred, 30+ services)
   - legacy flat logs: `logs-YYYY.MM.DD` (fallback, being phased out)
   - app logs: `safework2-logs-*` (if present)
   - alert pipeline diagnostics: `elastalert_status*`
3. Confirm mappings for selected index before querying (`mcphub_elk-get_mappings`).
4. Use existing mapped fields first; do not assume field names between indices are identical.

## Proxmox ELK correlation keys

- time: `@timestamp` or `timestamp`
- host: `host.name`, `host.hostname`, `host`, `fields.host_name`
- service/module: `service`, `module`, `container_name`
- severity/error: `level`, `error_severity`, `error.*`

## Proxmox query template

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

1. Check these domains in order: cluster/node health → storage latency/capacity → VM/CT restart loops and OOM → network reachability and DNS → backup/snapshot failures.
2. Flag high-risk operations before execution (mass restart, storage repair, index deletion, forced reindex). Stop-condition: if rollback is not defined and validated, do not execute.
3. Prefer staged fixes (single node or single workload first), then expand if validated.
4. If issue is likely replica-related in single-node ELK (`yellow` only, primaries active), avoid unnecessary recovery actions.
5. If restart loops are detected, classify as workload-level first unless node-wide symptoms are also present.

## Proxmox response contract

In addition to the base output format:

1. Infrastructure scope line: `service` | `VM/CT` | `node` | `cluster`.
2. Evidence snapshot with index, host key used, service/module key used, time window.
3. Step-by-step remediation must stay low blast radius first.

## Proxmox guardrails

1. Do not escalate from workload-level to node/cluster-level without new evidence.
2. Treat `yellow` as potentially normal in single-node setups when primaries are healthy.
