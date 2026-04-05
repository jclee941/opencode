# RULES KNOWLEDGE BASE

**Scope:** `rules/`
**Parent:** `AGENTS.md`

## OVERVIEW

`rules/` contains Tier 0 baseline rules (always loaded), Tier 1 on-demand rules,
and Tier 2 domain rules. Canonical tier/priority mapping lives in
`rules/README.md`.

## STRUCTURE

```text
rules/
├── README.md                         # canonical tier model + conflict resolution
├── AGENTS.md                         # this index (metadata, not runtime rule)
├── 00-hard-autonomy-no-questions.md     # Tier 0: zero-question execution posture
├── 00-archon-workflow.md                # Tier 0: Archon task and RAG workflow
├── 00-session-init.md                   # Tier 0: session bootstrap and MCP init
├── 00-requirements-verification.md      # Tier 0: requirement extraction/verification gate
├── 00-monorepo-standards.md             # Tier 0: naming/structure governance
├── 00-code-modularization.md            # Tier 0: file split/size governance (200 LOC)
├── deployment-automation.md          # Tier 1 on-demand: CI/CD and deployment safety
├── 01-auto-build-pipeline.md            # Tier 1 on-demand: spec-to-PR pipeline (overview)
├── 02-auto-build-pipeline-execution.md  # Tier 1 on-demand: build pipeline (execution)
├── 03-auto-build-pipeline-completion.md # Tier 1 on-demand: build pipeline (completion)
├── 01-mcp-schema-hygiene.md             # Tier 1 on-demand: MCP schema correctness
├── 01-onepassword-integration.md        # Tier 1 on-demand: 1Password integration policy
├── 02-onepassword-integration-patterns.md   # Tier 1 on-demand: 1Password patterns
├── 03-onepassword-integration-reference.md  # Tier 1 on-demand: 1Password reference
├── 01-onepassword-secrets-naming.md     # Tier 1 on-demand: 1Password naming specification
├── 02-onepassword-secrets-naming-examples.md  # Tier 1 on-demand: naming examples
├── 03-onepassword-secrets-naming-operations.md # Tier 1 on-demand: naming operations
├── 01-msa-refactoring.md                # Tier 1 on-demand: monolith → MSA migration guidance
├── 01-elk-troubleshooting.md            # Tier 2: ELK troubleshooting (overview)
├── 02-elk-troubleshooting-opencode.md   # Tier 2: ELK OpenCode domain
└── 03-elk-troubleshooting-proxmox.md    # Tier 2: ELK Proxmox domain
```

## WHERE TO LOOK

| Need | File |
|------|------|
| Canonical priority/tier map | `README.md` |
| Highest-priority execution posture | `00-hard-autonomy-no-questions.md` |
| Task lifecycle / Archon usage | `00-archon-workflow.md` |
| Session bootstrap and MCP init order | `00-session-init.md` |
| Requirement extraction + completion proof | `00-requirements-verification.md` |
| Naming conventions and script migration | `00-monorepo-standards.md` |
| **File modularization limits (200 LOC)** | **`00-code-modularization.md`** |
| CI/CD deploy controls | `deployment-automation.md` |
| Autonomous spec-to-PR pipeline | `01-auto-build-pipeline.md` + `02-*` + `03-*` |
| MCP -32602 prevention | `01-mcp-schema-hygiene.md` |
| Secret integration and `op://` usage | `01-onepassword-integration.md` |
| 1Password item/field naming schema | `01-onepassword-secrets-naming.md` |
| Microservice decomposition guidance | `01-msa-refactoring.md` |
| ELK troubleshooting runbook | `01-elk-troubleshooting.md` |

## 200 LOC MODULARIZATION RULE

All rule files follow the 200 LOC limit:
- **00-***: Main/overview files (<200 LOC)
- **02-***: Execution/pattern files (<200 LOC)
- **03-***: Completion/reference files (<200 LOC)

## CONVENTIONS

- Keep one canonical owner per rule topic; avoid duplicate normative text.
- Update this index and `rules/README.md` together when rule inventory changes.
- Treat this file as metadata/documentation, not as an executable instruction file.

## ANTI-PATTERNS

- Do not reference deleted/nonexistent rule files.
- Do not mark Tier 1 rules as globally pre-loaded when they are on-demand.
- Do not change tier/priority statements in individual rule files without updating
  `rules/README.md`.
