# ELK Troubleshooting Policy

Apply this policy to all ELK-based troubleshooting tasks regardless of project or stack.
Domain-specific sections (OpenCode, Proxmox) activate only when the affected target
matches that domain.

Priority: Tier 2 domain rule. Loaded when ELK work is in scope.

## Scope classification

Error tracking spans three scopes:

1. Application/runtime errors (service logs, agent/runtime logs, worker logs).
2. Platform errors (node/container/system-level failures affecting workloads).
3. Ingestion-path errors (shipper, pipeline, mapping, index routing failures).

## Core rule

0. All related error-tracking signals must be enrolled and verifiable in ELK before deep troubleshooting.
1. Troubleshooting must be evidence-first: collect ELK signals before proposing a fix.
2. If ELK is unavailable, explicitly report the gap and proceed with bounded local evidence.
3. If logs are not visible in ELK for the affected system, treat it as an observability incident and perform ELK integration/enrollment work first.

## Standard workflow

1. Define incident scope and time window (timezone + explicit range).
2. Check ELK platform baseline:
   - `mcphub_elk-elasticsearch_health`
   - `mcphub_elk-list_indices`
3. Select candidate indices and verify field mappings:
   - `mcphub_elk-get_mappings`
4. Run bounded queries with `mcphub_elk-search` using `queryBody`.
5. Correlate by at least two dimensions:
   - time
   - host/node
   - service/module
   - severity/error class
6. Propose minimum reversible remediation.
7. Re-check ELK evidence after change (10-30m window).

## Enrollment and verification sequence

1. Confirm local error emission exists at source (app log, journald, container log, runtime logger).
2. Confirm ingestion path health (shipper/forwarder status and destination reachability).
3. Ensure target index pattern is explicit and bounded.
4. Ensure required fields are queryable:
   - `@timestamp`
   - `message`
   - `level` or severity equivalent
   - `service/module` or workload identifier
   - `host/node` when applicable
5. Query `now-10m` to `now`; require at least one fresh event.
6. Query focused error window (`level:error` or equivalent) with small size (`20-100`).
7. Re-check after 10-30 minutes to confirm ingestion continuity.

## Query guardrails

1. Always use explicit time bounds.
2. Keep query size small by default (20-100).
3. Do not run unfiltered wildcard scans first.
4. Tighten dimensions before widening time range or size.

## Query templates

Adapt fields to actual mappings discovered via `mcphub_elk-get_mappings`.
If index uses `timestamp` instead of `@timestamp`, adapt sort/range accordingly.

```json
{
  "size": 50,
  "sort": [{ "@timestamp": { "order": "desc" } }],
  "query": {
    "bool": {
      "filter": [
        { "range": { "@timestamp": { "gte": "now-30m", "lte": "now" } } },
        { "term": { "level.keyword": "error" } }
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
        { "term": { "service.keyword": "<target-service>" } }
      ]
    }
  }
}
```

## Common false-negative traps

1. Green/yellow cluster status without fresh ingest events.
2. Index exists but mapping mismatch blocks useful filters.
3. Timezone/time-window mismatch causing no hits.
4. Index rollover moving recent data to a different backing index.
5. Partial auth: list/health allowed but search denied.

## Missing prerequisite handling

1. If integration needs unavailable credentials/endpoints, complete all safe non-secret steps first.
2. Then report exactly one missing value required to finish integration.

## Completion criteria

1. Fresh error-tracking events are queryable for the affected target.
2. Events contain correlation keys needed for triage.
3. Bounded error query reproduces symptom signal.
4. Post-change recheck confirms continued ingestion.

## Required output format

1. Scope (`application` | `platform` | `ingestion-path`).
2. Root-cause hypothesis tied to concrete ELK evidence.
3. Evidence snapshot:
   - index
   - key fields used
   - time window
4. Fix plan with rollback notes.
5. Verification checklist for the next 10-30 minutes.

---

## Domain: OpenCode Runtime

Apply this section when the affected target is OpenCode runtime behavior, agent execution,
plugin orchestration, or background task lifecycle.

### OpenCode scope translation

1. `session scope`: single OpenCode session id (`sessionID`) and its child task ids.
2. `agent scope`: one or more agents (`sisyphus`, `hephaestus`, `explore`, `librarian`, `oracle`, `metis`, `momus`).
3. `runtime scope`: opencode core plus plugin/runtime layer (`oh-my-opencode`, `opencode-supermemory`, `opencode-antigravity-auth`).

### OpenCode evidence dimensions

Correlate with at least two dimensions, preferring these keys when present:

1. `@timestamp`
2. `sessionID` or `session_id`
3. `taskId` or `task_id`
4. `agent` / `subagent`
5. `event` / `status` / `level`
6. `message`

If key names differ by index template, map them via `mcphub_elk-get_mappings` before query tuning.

### OpenCode index pattern

- Active index: `logs-opencode-YYYY.MM.DD`
- ~370 docs/day typical volume

### OpenCode non-incident flow

1. Pre-check: define scope (`session` | `agent` | `runtime`), capture baseline from `logs-opencode-YYYY.MM.DD` for `now-30m` to `now`.
2. Change execution: prefer smallest reversible change first, keep blast radius to one scope when possible.
3. Post-check: verify fresh events still ingest, verify symptom-related fields remain queryable.

### OpenCode incident triage

1. ELK platform check (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
2. Data availability: identify active OpenCode index by date, verify mappings.
3. Incident query: bounded search with explicit time range, correlate by session + agent + severity/status.
4. Classification: session-level lifecycle pattern, agent-level execution/staleness, runtime/plugin-level gap.
5. Remediation: smallest reversible change first, include rollback note and stop-condition.

### OpenCode query template

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

### OpenCode blind spots

1. Session cancellation may appear as normal lifecycle (`cancel` + `idle`) and can be duplicated by design.
2. Background cancellation can happen with limited surface logs; absence of one log line is not proof of absence.
3. Time-window mismatch (timezone drift, too-wide/too-narrow window) can produce false no-hit results.

### OpenCode response contract

In addition to the base output format:

1. Scope line: `session` | `agent` | `runtime`.
2. Selected index names and mapped field keys used for correlation.
3. Whether duplicate cancel/idle lines are interpreted as expected lifecycle or anomaly, with evidence.

### OpenCode guardrails

1. Do not escalate from session-level to runtime-level without new evidence.
2. Treat duplicate cancel/idle lines as hypothesis until correlated with surrounding events.

---

## Domain: Proxmox Infrastructure

Apply this section when the request involves troubleshooting services, VMs, containers,
networking, storage, or automation running on personal Proxmox infrastructure.

For destructive or irreversible operations, follow `rules/hard-autonomy-no-questions.md`.

### Proxmox environment baseline

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

### Proxmox non-incident flow

1. Pre-check: confirm target scope (`service` | `VM/CT` | `node` | `cluster`), capture a 10-30m baseline from relevant indices.
2. Change execution: prefer smallest reversible change first, keep blast radius to one workload when possible.
3. Post-check: verify error trend did not regress, confirm ingestion/query path is healthy.

### Proxmox incident triage

1. ELK platform check (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
2. Data availability: pick candidate indices (`logs-{service}-YYYY.MM.DD` preferred, `logs-YYYY.MM.DD` legacy fallback, `elastalert_status*`), verify mappings.
3. Incident query: bounded search, correlate by host + service/module + severity + time.
4. Classification: workload issue (restart loop, app 4xx/5xx spike), node issue (resource pressure, storage/network), cluster issue (broad ingestion/query failures).
5. Remediation: smallest reversible change first, include rollback note and stop-condition.

### Proxmox baseline capture checklist

1. Record cluster health summary (status, active primaries, unassigned shards).
2. Record top candidate indices for the incident window.
3. Record one representative host identifier and one representative service/module key.
4. Record one representative failing log line/event and timestamp.
5. Only then propose remediation sequence.

### Proxmox data source prioritization

1. Discover cluster/index health first (`mcphub_elk-elasticsearch_health`, `mcphub_elk-list_indices`).
   - Include ingest sanity checks (write blocks, rejection/backpressure signals, disk watermark symptoms).
2. Prioritize indices by intent:
   - service-scoped logs: `logs-{service}-YYYY.MM.DD` (preferred, 30+ services)
   - legacy flat logs: `logs-YYYY.MM.DD` (fallback, being phased out)
   - app logs: `safework2-logs-*` (if present)
   - alert pipeline diagnostics: `elastalert_status*`
3. Confirm mappings for selected index before querying (`mcphub_elk-get_mappings`).
4. Use existing mapped fields first; do not assume field names between indices are identical.

### Proxmox ELK correlation keys

- time: `@timestamp` or `timestamp`
- host: `host.name`, `host.hostname`, `host`, `fields.host_name`
- service/module: `service`, `module`, `container_name`
- severity/error: `level`, `error_severity`, `error.*`

### Proxmox query template

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

### Proxmox infrastructure remediation

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

### Proxmox-focused priorities

1. Check these domains in order: cluster/node health → storage latency/capacity → VM/CT restart loops and OOM → network reachability and DNS → backup/snapshot failures.
2. Flag high-risk operations before execution (mass restart, storage repair, index deletion, forced reindex). Stop-condition: if rollback is not defined and validated, do not execute.
3. Prefer staged fixes (single node or single workload first), then expand if validated.
4. If issue is likely replica-related in single-node ELK (`yellow` only, primaries active), avoid unnecessary recovery actions.
5. If restart loops are detected, classify as workload-level first unless node-wide symptoms are also present.

### Proxmox response contract

In addition to the base output format:

1. Infrastructure scope line: `service` | `VM/CT` | `node` | `cluster`.
2. Evidence snapshot with index, host key used, service/module key used, time window.
3. Step-by-step remediation must stay low blast radius first.

### Proxmox guardrails

1. Do not escalate from workload-level to node/cluster-level without new evidence.
2. Treat `yellow` as potentially normal in single-node setups when primaries are healthy.
