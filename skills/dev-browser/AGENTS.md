# DEV-BROWSER KNOWLEDGE BASE

**Scope:** `skills/dev-browser/`
**Parent:** `skills/AGENTS.md`

## OVERVIEW
`dev-browser` is a standalone TypeScript package for browser automation with persistent page/session state and optional extension relay mode.

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Server startup and cleanup | `skills/dev-browser/src/index.ts` | launches browser context, HTTP API, signal cleanup |
| Client operations | `skills/dev-browser/src/client.ts` | page creation, load waits, snapshot refs |
| Extension relay mode | `skills/dev-browser/src/relay.ts` | websocket/CDP bridge |
| Snapshot generation | `skills/dev-browser/src/snapshot/` | injected browser script + tests |
| Entrypoint scripts | `skills/dev-browser/scripts/` | `start-server.ts`, `start-relay.ts` |

## CONVENTIONS
- Runtime is Node + `tsx`; scripts are executed through `npx tsx`.
- TypeScript is strict, `moduleResolution: bundler`, and `noEmit: true`.
- Test runner is Vitest with longer timeouts for Playwright-backed flows.

## COMMANDS
```bash
npm --prefix skills/dev-browser run start-server
npm --prefix skills/dev-browser run start-extension
npm --prefix skills/dev-browser test
```

## ANTI-PATTERNS (THIS SUBTREE)
- Do not run scripts from unrelated working directories when using `@/*` alias behavior.
- Do not commit temporary/debug output from `tmp/` or browser profile data.
- Do not introduce fixed waits when page-state checks already exist in client helpers.
