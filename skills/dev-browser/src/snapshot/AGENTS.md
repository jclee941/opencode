# SNAPSHOT SUBSYSTEM KNOWLEDGE BASE

**Scope:** `skills/dev-browser/src/snapshot/`
**Parent:** `skills/dev-browser/src/AGENTS.md`

## OVERVIEW
This subtree owns browser-side snapshot extraction: script generation/injection and ref-based element lookup used by automation clients.

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Snapshot script source | `skills/dev-browser/src/snapshot/browser-script.ts` | generated browser-executed logic |
| Script injection/cache | `skills/dev-browser/src/snapshot/inject.ts` | inject + cache lifecycle |
| Public snapshot API | `skills/dev-browser/src/snapshot/index.ts` | exports for consumers |
| Behavioral tests | `skills/dev-browser/src/snapshot/__tests__/snapshot.test.ts` | Playwright + Vitest verification |

## CONVENTIONS
- Keep browser-executed code JavaScript-compatible (no TS-only syntax in evaluated blocks).
- Preserve stable ref semantics because client selection depends on ref identity.
- Update tests with snapshot behavior changes in the same edit.

## ANTI-PATTERNS (THIS SUBTREE)
- Do not move snapshot responsibilities into `client.ts`; keep source-of-truth in this module.
- Do not skip cache reset/test updates when changing injected script behavior.
