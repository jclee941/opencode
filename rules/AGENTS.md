# RULES KNOWLEDGE BASE

**Scope:** `rules/`
**Parent:** `AGENTS.md`

## OVERVIEW

9 rule files organized in a 3-tier architecture. Tier 1 rules load in every session via `opencode.jsonc`; Tier 2/3 rules load only when their domain or process is in scope. Architecture is documented in `rules/README.md`.

## STRUCTURE

```text
rules/
├── README.md                              # tier model, conflict resolution, loading scope (108 lines)
├── hard-autonomy-no-questions.md           # Tier 1: zero-question execution policy (77 lines)
├── session-init.md                        # Tier 1: session startup checklist (99 lines)
├── requirements-verification.md           # Tier 1: requirements check gate (78 lines)
├── deployment-automation.md               # Tier 1: CI/CD policy, manual deploy prohibition (140 lines)
├── monorepo-standards.md                  # Tier 1: structure, naming, script migration (98 lines)
├── mcp-schema-hygiene.md                  # Tier 1: MCP tool call schema validation (85 lines)
├── code-modularization.md                 # Tier 1: file size governance, split strategies (591 lines)
└── elk-troubleshooting.md                 # Tier 2: ELK troubleshooting, all domains (329 lines)
```

## WHERE TO LOOK

| Task                                  | Location                        | Notes                                                      |
| ------------------------------------- | ------------------------------- | ---------------------------------------------------------- |
| Priority and conflict resolution      | `README.md`                     | canonical ordering and tier model                          |
| Execution posture and question policy | `hard-autonomy-no-questions.md` | highest-priority rule, denylist regex, blocked-step format |
| Session startup checklist             | `session-init.md`               | 6-step init, guardrails                                    |
| Requirements check before/after impl  | `requirements-verification.md`  | pre/during/post verification gates                         |
| CI/CD and deploy rules                | `deployment-automation.md`      | platform strategies, secrets, environment protection       |
| Naming and structure enforcement      | `monorepo-standards.md`         | kebab-case, Bazel compat, script migration policy          |
| MCP tool call validation              | `mcp-schema-hygiene.md`         | schema compliance, -32602 prevention, name mapping rules   |
| File size governance and splits       | `code-modularization.md`        | 500 LOC threshold, split strategies, circular dep prevention|
| ELK troubleshooting (all domains)     | `elk-troubleshooting.md`        | unified triage, OpenCode + Proxmox domain sections         |

## CONVENTIONS

- Each rule file has one responsibility; do not merge unrelated policies.
- Tier 1 rules are explicitly listed in `opencode.jsonc` `instructions` array — never use glob.
- ELK troubleshooting is a single file with embedded domain sections (OpenCode, Proxmox).
- `hard-autonomy-no-questions.md` overrides all other rules when conflict exists.
- `README.md` is the architecture document, not a rule itself.

## ANTI-PATTERNS (THIS SUBTREE)

- Do not duplicate normative text across rule files; keep one canonical source and reference it.
- Do not add Tier 2/3 rules to `opencode.jsonc` instructions — they consume context in unrelated sessions.
- Do not create new rule files without updating `README.md` tier classification and priority order.
- Do not treat `README.md` as a loadable instruction; it defines architecture, not execution policy.
