# SKILLS KNOWLEDGE BASE

**Scope:** `skills/`
**Parent:** `AGENTS.md`

## OVERVIEW
`skills/` is a mixed registry: one executable skill (`dev-browser`) plus policy-style skill docs (`agent-browser`, `frontend-ui-ux`, `git-master`).

## STRUCTURE
```text
skills/
├── dev-browser/      # runnable browser automation service + client
├── agent-browser/    # CLI usage policy docs
├── frontend-ui-ux/   # UI/UX guidance docs
└── git-master/       # git workflow policy docs
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Run browser service | `skills/dev-browser/package.json` | `start-server`, `start-extension`, `test` scripts |
| Browser server internals | `skills/dev-browser/src/index.ts` | page registry and lifecycle |
| Persistent browser client | `skills/dev-browser/src/client.ts` | connect/list/page/snapshot operations |
| Skill policy contract | `skills/*/SKILL.md` | behavior constraints and anti-patterns |

## CONVENTIONS
- Treat `SKILL.md` files as policy surfaces; change only when policy update is intentional.
- `dev-browser` is an independent Node/npm package; do not assume root workspace wiring.
- Browser profile/runtime artifacts under `skills/dev-browser/profiles/` and `skills/dev-browser/tmp/` are operational data.

## ANTI-PATTERNS (THIS SUBTREE)
- Do not implement runtime changes in policy-only skill folders.
- Do not treat cached profile files as source code.
- Do not copy root conventions verbatim into child docs; keep only skill-specific deltas.
