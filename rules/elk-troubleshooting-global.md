# Global ELK Troubleshooting Policy

Apply this policy to all troubleshooting tasks by default, regardless of project or stack.

This file is the canonical source for shared ELK troubleshooting rules.

## Specialization model

1. Domain-specific ELK rule files must reference this policy for shared behavior.
2. Domain-specific files should define only deltas (environment, field mapping variants, scope translation, escalation specifics).
3. Do not duplicate shared query guardrails/output contracts in specialization files unless the domain requires a documented exception.

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
