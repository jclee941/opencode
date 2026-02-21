# Ruleset Architecture

This file defines how rule files in `rules/` are composed and resolved.

## Priority order

Apply rules in this order when overlap exists:

1. `hard-autonomy-no-questions.md` (execution posture, question policy, blocked-step handling)
2. `requirements-verification.md` (always-check requirements spec before/after implementation)
3. Domain rules (`elk-troubleshooting-global.md`, `elk-error-tracking-runbook.md`, `elk-opencode-troubleshooting.md`, `opencode-elk-triage-runbook.md`, `elk-proxmox-troubleshooting.md`, `proxmox-elk-triage-runbook.md`)
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
- ELK error tracking: `elk-error-tracking-runbook.md`
- ELK OpenCode delta: `elk-opencode-troubleshooting.md`
- ELK OpenCode runbook: `opencode-elk-triage-runbook.md`
- ELK Proxmox delta: `elk-proxmox-troubleshooting.md`
- ELK Proxmox runbook: `proxmox-elk-triage-runbook.md`

### Tier 3 — Process rules (loaded when specific process is in scope)

- `requirements-modularization-checklist.md` — requirement authoring and checklist quality
- `document-normalization-runbook.md` — document move/rename/dedup tasks
- `instructions-reference-composition.md` — instruction composition guide

## Inheritance model

1. `elk-troubleshooting-global.md` is the ELK troubleshooting source of truth.
2. `elk-error-tracking-runbook.md` is the default execution checklist for cross-domain error tracking in ELK.
3. `elk-opencode-troubleshooting.md` only adds OpenCode-specific constraints, fields, and scope translation.
4. `opencode-elk-triage-runbook.md` is the execution checklist for OpenCode work; it must reference global policy for shared ELK rules instead of duplicating them.
5. `elk-proxmox-troubleshooting.md` only adds Proxmox-specific constraints, fields, and scope translation.
6. `proxmox-elk-triage-runbook.md` is the execution checklist for Proxmox work; it must reference global policy for shared ELK rules instead of duplicating them.

## Conflict resolution rules

1. Prefer stricter safety constraints when two rules differ.
2. Do not duplicate the same normative rule text across multiple files; keep one canonical source and link to it.
3. If a domain rule needs an exception, state the exception explicitly and bound it to that domain only.

## High-risk operation policy

Canonical source: `rules/hard-autonomy-no-questions.md`.

All high-risk and blocked-operation handling must reference that file instead of repeating the same normative text.

## Change hygiene for rules

1. Keep each rule file focused on one responsibility.
2. When moving normative content, update inbound references in the same change.
3. Validate with `npm run lint:naming` after changes.
4. For reversible in-scope recommendations, prefer immediate application over optional suggestion text.

## Instruction reference composition

Canonical composition guide: `rules/instructions-reference-composition.md`.

Runtime loading model in this repository:

1. `opencode.jsonc` loads Tier 1 rules via explicit file paths (not glob).
2. `AGENTS.md` is loaded natively by OpenCode — it must not appear in the instructions array.
3. Explicit listing is preferred over `rules/*.md` glob to prevent Tier 2/3 domain rules from consuming context tokens in unrelated sessions.
4. File-level ownership must stay single-source (no duplicated normative text).
5. Task prompts still reference only the files needed for that task.
