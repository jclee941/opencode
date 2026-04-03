# CONFIG KNOWLEDGE BASE

**Scope:** `config/`
**Parent:** `AGENTS.md`

## OVERVIEW

Three JSONC source files define the merged OpenCode config. `config/` is the only writable source for `opencode.jsonc`.

## STRUCTURE

```text
config/
├── base.jsonc       # instructions, plugins, MCP, permissions, formatters, watcher
├── lsp.jsonc        # language servers, schema associations, editor behavior
└── providers.jsonc  # provider/model registry; currently minimal by design
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add/remove always-loaded rules | `config/base.jsonc` | Tier 0 only; explicit paths, not globs |
| Change plugin order or permissions | `config/base.jsonc` | `oh-my-opencode` must stay last |
| Add formatter/watcher ignore | `config/base.jsonc` | shared repo-wide defaults |
| Tune language servers | `config/lsp.jsonc` | commands, extensions, initialization blocks |
| Add JSON schema mapping | `config/lsp.jsonc` | `json.schemas.fileMatch` entries |
| Register internal provider models | `config/providers.jsonc` | cross-checked by `lint:config-refs` |

## CONVENTIONS

- JSONC comments are allowed; generator strips them before merge.
- Merge order is deterministic from sorted `config/*.jsonc` filenames.
- `providers.jsonc` can stay sparse; some external prefixes are whitelisted by validator logic.
- `config/lsp.jsonc` is the canonical place for schema-to-file matching.

## ANTI-PATTERNS (THIS SUBTREE)

- Do not edit `opencode.jsonc` instead of these source files.
- Do not put Tier 1/2 rule paths into `instructions`.
- Do not move `oh-my-opencode` ahead of other plugins.
- Do not add config surfaces here without updating generator and ref-validation expectations.
