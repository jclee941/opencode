# ELK Troubleshooting For Personal Proxmox

Apply this guidance when the request is about troubleshooting services, VMs, containers, networking, storage, or automation running on personal Proxmox infrastructure.

For all Proxmox-related work (routine or incident), also apply `rules/proxmox-elk-triage-runbook.md`.

## Rule composition

1. Shared ELK behavior comes from `rules/elk-troubleshooting-global.md` (source of truth).
2. This file adds only Proxmox-specific constraints and field/scope translation.
3. For execution steps and query payload skeletons, use `rules/proxmox-elk-triage-runbook.md`.

## Scope and intent

1. Default to ELK-first troubleshooting: inspect evidence in Elasticsearch/Kibana before proposing fixes.
2. For destructive or irreversible operations and question policy, follow `rules/hard-autonomy-no-questions.md`.
3. Keep actions reversible and evidence-based.

## Current environment baseline (known/likely)

If any baseline item is unknown, treat it as a hypothesis and verify via ELK/Proxmox evidence before acting.

1. MCP endpoint is configured through `mcphub` in `opencode.jsonc`.
2. ELK cluster may run as single-node homelab deployment.
3. `yellow` cluster/index health can be expected in single-node mode when replicas are configured.
4. Treat missing Proxmox-native fields (`vmid`, `ctid`, `node`) as a logging coverage gap, not immediate infra failure.
5. Current logs indicate mixed platform/application ingestion:
   - infra-style stream in `logs-*` (for example Ceph/system logs)
   - app stream in `safework2-logs-*` (HTTP/request-centric fields)
6. Example host identifiers observed in logs can differ by source format:
   - `host.hostname` (for example `pve3`)
   - `host_name` / `host.name` aliases (for example short host labels)

## Required troubleshooting workflow

1. Confirm symptom window and impacted target (node, VM, CT, service).
   - Always state timezone and an explicit initial window (for example `now-15m` to `now`).
2. Query Elasticsearch for matching error events before making configuration changes.
3. Correlate at least two dimensions when possible:
   - `host.name` or node identifier
   - service/process name
   - timestamp range
   - severity (`error`, `warning`)
4. Identify probable root cause from logs, then propose the minimum safe remediation.
5. After remediation, re-query logs to verify error-rate reduction.

## Baseline capture checklist (before deep fixes)

1. Record cluster health summary and include:
   - cluster status (`green`/`yellow`/`red`)
   - active primaries and unassigned shards
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
5. If `logs-*` does not match, use the discovered index names/data-stream backing from `mcphub_elk-list_indices`.

## ELK usage rules

1. Inherit shared query guardrails/output contract from `rules/elk-troubleshooting-global.md`.
2. Use index discovery first (`mcphub_elk-list_indices`) and mapping checks (`mcphub_elk-get_mappings`) before deep queries.
3. Correlate using actual mapped keys, preferring:
    - time: `@timestamp` or `timestamp`
    - host: `host.name`, `host.hostname`, `host`, `fields.host_name`
    - service/module: `service`, `module`, `container_name`
    - severity/error: `level`, `error_severity`, `error.*`
4. When search payload shape is uncertain, fall back to index health + mappings + bounded manual evidence, then report the exact missing query contract.

## Search query contract

1. `mcphub_elk-search` requires `queryBody`.
2. Global guardrails still apply (`rules/elk-troubleshooting-global.md`), including explicit time bounds and bounded query size.
3. Minimum safe query pattern:

```json
{
  "size": 50,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-30m" } } }
      ]
    }
  }
}
```

4. If index uses `timestamp` instead of `@timestamp`, adapt sort/range accordingly.
5. For quick incident triage, start with `size <= 50`, then tighten by `host`/`service`/`level`.

## Proxmox infrastructure utilization

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
4. Use host identifiers from logs (`host.name`, `host.hostname`, `host_id`) to map symptoms back to Proxmox node/guest scope before proposing fixes.

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
4. If issue is likely replica-related in single-node ELK (`yellow` only, primaries active), avoid unnecessary recovery actions; focus on primary ingestion and query correctness.
5. If restart loops are detected in app/system logs, classify as workload-level first unless node-wide symptoms are also present.

## Output expectations

1. Provide concise root-cause statement tied to log evidence.
2. Provide a fix plan with rollback notes.
3. Provide a post-check list (what to monitor in ELK for 10-30 minutes).
4. Include an "Infrastructure scope" line: `service` | `VM/CT` | `node` | `cluster`.
5. Include a short "Evidence snapshot" line with:
   - index
   - host key used
   - service/module key used
   - time window
