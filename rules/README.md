# Ruleset Architecture

This file defines how rule files in `rules/` are composed, resolved, and loaded.

## Priority order

Apply rules in this order when overlap exists:

1. `hard-autonomy-no-questions.md` (execution posture, question policy, blocked-step handling)
2. `requirements-verification.md` (always-check requirements spec before/after implementation)
3. Domain rules (`elk-troubleshooting-global.md`, `elk-opencode.md`, `elk-proxmox.md`)
4. `session-init.md` (session checklist and verification baseline)
5. `deployment-automation.md` (CI/CD policy, manual deploy prohibition)
6. `monorepo-standards.md` (structure, naming, document normalization)
7. `requirements-modularization-checklist.md` (requirement/spec modularization and checklist quality)

## Tier model

Rules are grouped into tiers by loading scope:

### Tier 1 — Always loaded (via `opencode.jsonc` instructions)

These are loaded into every session as baseline rules:

- `hard-autonomy-no-questions.md` — execution posture, zero-question policy
- `session-init.md` — session startup checklist
- `requirements-verification.md` — requirements check and verification gate
- `deployment-automation.md` — CI/CD policy
- `monorepo-standards.md` — structure and naming

### Tier 2 — Domain rules (loaded when domain is in scope)

Loaded only when the task touches the relevant domain:

- ELK global: `elk-troubleshooting-global.md`
- ELK OpenCode: `elk-opencode.md`
- ELK Proxmox: `elk-proxmox.md`

### Tier 3 — Process rules (loaded when specific process is in scope)

- `requirements-modularization-checklist.md` — requirement authoring and checklist quality
- `document-normalization-runbook.md` — document move/rename/dedup tasks

## Inheritance model

1. `elk-troubleshooting-global.md` is the ELK troubleshooting source of truth.
2. `elk-opencode.md` adds only OpenCode-specific deltas (scope translation, evidence dimensions, blind spots).
3. `elk-proxmox.md` adds only Proxmox-specific deltas (environment baseline, infrastructure remediation, data source prioritization).

## Conflict resolution rules

1. Prefer stricter safety constraints when two rules differ.
2. Do not duplicate the same normative rule text across multiple files; keep one canonical source and link to it.
3. If a domain rule needs an exception, state the exception explicitly and bound it to that domain only.

When conflicts appear, resolve and document in this order:

1. `hard-autonomy-no-questions.md`
2. domain rule
3. `session-init.md`
4. repository standards

Write conflict decisions in one line:

`applied: <base rule>, overridden by: <exception rule>, scope: <bounded scope>`

## High-risk operation policy

Canonical source: `rules/hard-autonomy-no-questions.md`.

All high-risk and blocked-operation handling must reference that file instead of repeating the same normative text.

## Change hygiene for rules

1. Keep each rule file focused on one responsibility.
2. When moving normative content, update inbound references in the same change.
3. Validate with `npm run lint:naming` after changes.
4. For reversible in-scope recommendations, prefer immediate application over optional suggestion text.

## Instruction loading model

1. `opencode.jsonc` loads Tier 1 rules via explicit file paths (not glob).
2. `AGENTS.md` is loaded natively by OpenCode — it must not appear in the instructions array.
3. Explicit listing is preferred over `rules/*.md` glob to prevent Tier 2/3 domain rules from consuming context tokens in unrelated sessions.
4. File-level ownership must stay single-source (no duplicated normative text).
5. Task prompts still reference only the files needed for that task.

## Task-time composition order

1. Base execution rules (Tier 1 — always loaded):
   - `rules/hard-autonomy-no-questions.md`
   - `rules/session-init.md`
   - `rules/requirements-verification.md`
2. Domain rules (Tier 2 — only when relevant):
   - global ELK: `rules/elk-troubleshooting-global.md`
   - OpenCode ELK: `rules/elk-opencode.md`
   - Proxmox ELK: `rules/elk-proxmox.md`
3. Repository governance (Tier 1 baseline + Tier 3 process):
   - `rules/deployment-automation.md` (Tier 1)
   - `rules/monorepo-standards.md` (Tier 1)
   - `rules/requirements-modularization-checklist.md` (Tier 3 — requirement/checklist tasks)
   - `rules/document-normalization-runbook.md` (Tier 3 — document move/rename tasks)

## Reference hygiene

1. Use exact relative paths under `rules/`.
2. Avoid duplicate references to equivalent rules.
3. Keep the reference set minimal and task-specific.
4. If a referenced file is canonical SSoT, point secondary rules to it instead of copying text.
