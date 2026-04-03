# TESTS KNOWLEDGE BASE

**Scope:** `tests/`
**Parent:** `AGENTS.md`

## OVERVIEW

Root tests are Vitest ELK integration checks. They target live Elasticsearch-style endpoints, not isolated unit stubs.

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Shared ES request logic | `tests/elk/helpers.ts` | auth headers, JSON parsing, KST index naming |
| Cluster health smoke test | `tests/elk/elk-health.test.ts` | basic connectivity gate |
| Ingestion expectations | `tests/elk/elk-ingestion.test.ts` | document write/read flow |
| Index naming behavior | `tests/elk/elk-index.test.ts` | `logs-opencode-YYYY.MM.DD` contract |
| Mapping assertions | `tests/elk/elk-mapping.test.ts` | field schema expectations |
| Query behavior | `tests/elk/elk-query.test.ts` | search/result assumptions |

## CONVENTIONS

- `vitest.config.ts` limits discovery to `tests/**/*.test.ts`.
- `tests/elk/helpers.ts` is the source of truth for ES auth and index-name helpers.
- Date-sensitive index helpers use `Asia/Seoul`, not UTC.
- Environment overrides are `ES_URL`, `ES_USER`, and `ES_PASSWORD`.

## ANTI-PATTERNS (THIS SUBTREE)

- Do not duplicate raw fetch/auth logic across ELK tests.
- Do not hardcode UTC-derived index names.
- Do not convert these checks into mock-only tests without changing the suite's purpose.
