# DEV-BROWSER SRC KNOWLEDGE BASE

**Scope:** `skills/dev-browser/src/`
**Parent:** `skills/dev-browser/AGENTS.md`

## OVERVIEW
Source code is layered into service (`index.ts`), client API (`client.ts`), relay bridge (`relay.ts`), and snapshot internals (`snapshot/`).

## CODE MAP
| Symbol | Type | Location | Role |
|--------|------|----------|------|
| `serve` | function | `skills/dev-browser/src/index.ts` | starts automation server and browser context |
| `connect` | function | `skills/dev-browser/src/client.ts` | attaches to server and returns client operations |
| `waitForPageLoad` | function | `skills/dev-browser/src/client.ts` | unified readiness checks (doc + network idle) |
| `getAISnapshot` | method | `skills/dev-browser/src/client.ts` | returns snapshot for agent-driven interactions |

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Add API endpoint | `skills/dev-browser/src/index.ts` | express route handlers and registry mutations |
| Improve navigation stability | `skills/dev-browser/src/client.ts` | load-state polling and timeout behavior |
| Extension transport changes | `skills/dev-browser/src/relay.ts` | relay server and websocket lifecycle |
| Snapshot behavior | `skills/dev-browser/src/snapshot/` | browser-side extraction logic |

## CONVENTIONS
- Keep exported types centralized in `skills/dev-browser/src/types.ts` for shared contracts.
- Prefer existing client wait/snapshot helpers over one-off call sequences.
- Keep relay/server/client responsibilities separated; avoid cross-layer coupling.

## ANTI-PATTERNS (THIS SUBTREE)
- Do not bypass cleanup paths for browser context or sockets.
- Do not duplicate snapshot extraction logic outside `snapshot/`.
