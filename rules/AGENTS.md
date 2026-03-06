# RULES KNOWLEDGE BASE

**Scope:** `rules/`
**Parent:** `AGENTS.md`

## OVERVIEW

11 rule files organized in a 3-tier architecture. Tier 0 rules load in every session via `opencode.jsonc`; Tier 1 rules load on-demand when their domain matches; Tier 2 rules load only when their specific domain is in scope. Architecture is documented in `rules/README.md`.

## STRUCTURE

```text
rules/
├── README.md                              # tier model, conflict resolution, loading scope
├── hard-autonomy-no-questions.md           # Tier 0: zero-question execution policy (77 lines)
├── archon-workflow.md                      # Tier 0: Archon task management (140 lines)
├── session-init.md                        # Tier 0: session startup checklist (99 lines)
├── requirements-verification.md           # Tier 0: requirements check gate (78 lines)
├── monorepo-standards.md                  # Tier 0: structure and naming (98 lines)
├── deployment-automation.md               # Tier 1: CI/CD policy (on-demand, 140 lines)
├── code-modularization.md                 # Tier 1: file size governance (on-demand)
├── bmad-integration.md                    # Tier 1: BMAD artifacts (on-demand)
├── auto-build-pipeline.md                 # Tier 1: autonomous pipeline (on-demand)
├── mcp-schema-hygiene.md                  # Tier 1: MCP tool call schema validation (on-demand, 85 lines)
└── elk-troubleshooting.md                 # Tier 2: ELK troubleshooting (on-demand)
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
| Archon task management and RAG      | `archon-workflow.md`            | task cycle, RAG search, project/document tools             |
| BMAD artifact consumption           | `bmad-integration.md`           | `_bmad-output/` detection, artifact mapping                |
| Spec-to-PR autonomous pipeline      | `auto-build-pipeline.md`        | `/start-work` trigger, build loop, verification gates      |

## CONVENTIONS

- Each rule file has one responsibility; do not merge unrelated policies.
- Tier 0 rules are explicitly listed in `opencode.jsonc` `instructions` array — never use glob.
- ELK troubleshooting is a single file with embedded domain sections (OpenCode, Proxmox).
- `hard-autonomy-no-questions.md` overrides all other rules when conflict exists.
- `README.md` is the architecture document, not a rule itself.

## ANTI-PATTERNS (THIS SUBTREE)

- Do not duplicate normative text across rule files; keep one canonical source and reference it.
- Do not add Tier 1/2 rules to `opencode.jsonc` instructions — they consume context in unrelated sessions.
- Do not create new rule files without updating `README.md` tier classification and priority order.
- Do not treat `README.md` as a loadable instruction; it defines architecture, not execution policy.
