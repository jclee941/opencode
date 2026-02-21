# Global ELK Troubleshooting Policy

Apply this policy to all troubleshooting tasks by default, regardless of project or stack.

This file is the canonical source for shared ELK troubleshooting rules.

## Specialization model

1. Domain-specific ELK rule files must reference this policy for shared behavior.
2. Domain-specific files should define only deltas (environment, field mapping variants, scope translation, escalation specifics).
3. Do not duplicate shared query guardrails/output contracts in specialization files unless the domain requires a documented exception.

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

## ELK unlinked fallback (mandatory)

When troubleshooting evidence is missing from ELK, execute this sequence before deep root-cause work:

1. Confirm the target emits logs locally (service log file, journald, container logs, or app logger).
2. Confirm ELK path health:
   - Elasticsearch reachable
   - index write path available
   - ingestion agent/forwarder status healthy
3. Verify index and mapping existence for the target workload.
4. If ingestion is missing, perform minimum integration work:
   - add or fix log shipper input for target logs
   - ensure required metadata fields (`host`, `service/module`, timestamp, severity)
   - route to a bounded index pattern
5. Validate integration success by ingesting fresh events and querying them with bounded `queryBody`.
6. Resume normal troubleshooting only after ELK visibility is confirmed.

## Integration completion criteria

1. At least one fresh event for the affected target is queryable in ELK.
2. Events include usable correlation keys (time + host + service/module + severity).
3. Troubleshooting response includes pre/post integration evidence snapshot.

## Missing prerequisite handling

1. If integration needs unavailable credentials/endpoints, complete all safe non-secret steps first.
2. Then report exactly one missing value required to finish integration.

## Query guardrails

1. Always use explicit time bounds.
2. Keep query size small by default (20-100).
3. Do not run unfiltered wildcard scans first.
4. Tighten dimensions before widening time range or size.

## Required output format for troubleshooting responses

1. Root-cause hypothesis tied to concrete ELK evidence.
2. Evidence snapshot:
   - index
   - key fields used
   - time window
3. Fix plan with rollback notes.
4. Verification checklist for the next 10-30 minutes.
