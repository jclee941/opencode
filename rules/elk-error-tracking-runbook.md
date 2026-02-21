# ELK Error Tracking Runbook (All Domains)

Use this runbook when the goal is end-to-end error tracking visibility in ELK,
regardless of project, service, runtime, or infrastructure layer.

Shared policy is inherited from `rules/elk-troubleshooting-global.md`.
Domain files may add deltas only.

## Scope

1. Application/runtime errors (service logs, agent/runtime logs, worker logs).
2. Platform errors (node/container/system-level failures affecting workloads).
3. Ingestion-path errors (shipper, pipeline, mapping, index routing failures).

## Mandatory baseline (all scopes)

1. Cluster health check: `mcphub_elk-elasticsearch_health`.
2. Index discovery check: `mcphub_elk-list_indices`.
3. Mapping check for selected index: `mcphub_elk-get_mappings`.
4. Fresh-ingest check with explicit time range and bounded size: `mcphub_elk-search`.
5. Correlation check using at least two keys: time + host/service/module + severity/error class.

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

## Query templates

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

## Completion criteria

1. Fresh error-tracking events are queryable for the affected target.
2. Events contain correlation keys needed for triage.
3. Bounded error query reproduces symptom signal.
4. Post-change recheck confirms continued ingestion.

## Response contract

1. Scope (`application` | `platform` | `ingestion-path`).
2. Evidence snapshot (index, key fields, time window).
3. Minimal reversible fix and rollback note.
4. Verification checklist for next 10-30 minutes.
